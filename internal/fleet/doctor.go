package fleet

// Feature 014 B5 / T09 — opt-in offline fleet model-route diagnostics.
//
// fleet doctor never performs network requests, model inference or provider
// calls, and it never writes files. It labels model routes with exactly the
// evidence gathered from local sources the user passed explicitly:
//
//	configured          provider is present in a local dump/config listing
//	catalog-visible     model route appears in a local catalog listing
//	authenticated       provider carries an explicit local auth marker
//	unknown             inspected, but the local evidence cannot decide
//	unavailable         a requested inspection tool failed or produced nothing
//	not checked (offline)  default mode: no inspection tool was offered
//
// A route string alone is never proof of anything: notably `openai/*` is a
// provider route, not evidence of a ChatGPT subscription (spec §2.2, AC-20),
// and no `authenticated` label may be inferred from grammar or catalog
// presence — only from an explicit local marker or an equivalent probe error
// absence of one. This file intentionally depends only on the frozen B1
// surface (Config/ParseConfig/validateModel); renderer, plan and manifest
// integration stays outside the doctor seam.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// DoctorLabel is one outcome of a local-only model-route inspection.
type DoctorLabel string

// The label vocabulary fixed by spec R6 / §4.2. Strings are part of the CLI
// output contract; keep them stable.
const (
	DoctorConfigured     DoctorLabel = "configured"
	DoctorCatalogVisible DoctorLabel = "catalog-visible"
	DoctorAuthenticated  DoctorLabel = "authenticated"
	DoctorUnknown        DoctorLabel = "unknown"
	DoctorUnavailable    DoctorLabel = "unavailable"
	DoctorNotChecked     DoctorLabel = "not checked (offline)"
)

// doctorAuthMarkers are the only dump provider markers treated as local
// authentication evidence. Anything else (including unknown strings) proves
// at most that the provider is configured.
var doctorAuthMarkers = map[string]bool{
	"authenticated": true,
	"oauth":         true,
	"api-key":       true,
	"key":           true,
	"subscription":  true,
	"logged-in":     true,
}

// doctorListingWord finds the catalog-listing subcommand advertised by
// `opencode --help`. Only fixed, read-only listing argv is ever executed.
var doctorListingWord = regexp.MustCompile(`(^|\s)models(\s|$|,)`)

// DoctorEvidence is the merged result of every local source inspected.
// Implementations must fill it only from files or local commands the user
// named explicitly.
type DoctorEvidence struct {
	// CatalogObtained is true when at least one catalog listing succeeded
	// (even an empty one). Absence of a listing is not absence of models.
	CatalogObtained bool
	// CatalogModels holds model routes visible in local catalog listings,
	// in provider/model form (usually without #variant).
	CatalogModels []string
	// ConfiguredProviders holds provider IDs present in local configuration
	// or implied by catalog listings.
	ConfiguredProviders []string
	// AuthenticatedProviders holds provider IDs with an explicit local
	// authentication marker. It is the only path to DoctorAuthenticated.
	AuthenticatedProviders []string
}

// DoctorProbe is an injectable local-evidence source. Tests substitute it to
// prove zero process and file access; production callers pass nil and get the
// built-in dump/binary probes. It must never touch the network or models.
type DoctorProbe interface {
	Evidence() (DoctorEvidence, error)
}

// DoctorOptions selects the models and the (opt-in) local inspection sources.
// With none of Probe, OpenCodeBin and CatalogDump set, RunDoctor stays in the
// offline default mode and consults nothing at all.
type DoctorOptions struct {
	// Models are provider/model[#variant] routes to label, in report order.
	Models []string
	// OpenCodeBin is an explicit path to a local OpenCode executable. Doctor
	// runs only `<bin> --help` and, when help advertises it, `<bin> models`.
	OpenCodeBin string
	// CatalogDump is an explicit path to a local JSON dump shaped
	// {"providers":[{"id":"...","auth":"..."}],"models":["provider/id",...]}.
	CatalogDump string
	// Probe replaces the built-in sources entirely when non-nil.
	Probe DoctorProbe
}

// DoctorEntry is the labeled outcome for one model route.
type DoctorEntry struct {
	Model  string
	Label  DoctorLabel
	Reason string
}

// DoctorReport is the full diagnostic output. Failures and ToolsInspected
// satisfy NFR-05: the report states exactly which local tools were consulted
// and why any of them produced nothing.
type DoctorReport struct {
	Entries        []DoctorEntry
	ToolsInspected []string
	Failures       []string
	// Offline is true for the default no-inspection mode.
	Offline bool
}

// RunDoctor labels every route from local evidence only. It never returns an
// error: inspection problems surface as unavailable labels plus sanitized
// Failures notes, so a partial report is still produced.
func RunDoctor(opts DoctorOptions) DoctorReport {
	if opts.Probe == nil && opts.OpenCodeBin == "" && opts.CatalogDump == "" {
		rep := DoctorReport{Offline: true}
		for _, model := range opts.Models {
			rep.Entries = append(rep.Entries, DoctorEntry{
				Model:  model,
				Label:  DoctorNotChecked,
				Reason: "offline default: no local tool was offered, so nothing was checked",
			})
		}
		return rep
	}

	ev, tools, failures := gatherDoctorEvidence(opts)
	rep := DoctorReport{ToolsInspected: tools, Failures: failures}
	for _, model := range opts.Models {
		label, reason := labelDoctorModel(model, ev, len(failures) > 0)
		rep.Entries = append(rep.Entries, DoctorEntry{Model: model, Label: label, Reason: reason})
	}
	return rep
}

// gatherDoctorEvidence consults exactly the sources named in opts, merges
// their evidence, and records the inspected tools and sanitized failures.
func gatherDoctorEvidence(opts DoctorOptions) (DoctorEvidence, []string, []string) {
	var ev DoctorEvidence
	var tools, failures []string

	mergeLists := func(dst *[]string, items []string) {
		for _, item := range items {
			if !containsString(*dst, item) {
				*dst = append(*dst, item)
			}
		}
		*dst = sortedCopy(*dst)
	}

	if opts.Probe != nil {
		probeEv, err := opts.Probe.Evidence()
		if err != nil {
			failures = append(failures, fmt.Sprintf("local probe failed: %v", err))
		}
		ev.CatalogObtained = probeEv.CatalogObtained
		mergeLists(&ev.CatalogModels, probeEv.CatalogModels)
		mergeLists(&ev.ConfiguredProviders, probeEv.ConfiguredProviders)
		mergeLists(&ev.AuthenticatedProviders, probeEv.AuthenticatedProviders)
		tools = append(tools, "injected local probe")
		return ev, tools, failures
	}

	if opts.CatalogDump != "" {
		dumpEv, err := readDoctorDump(opts.CatalogDump)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%v", err))
		} else {
			tools = append(tools, opts.CatalogDump)
			ev.CatalogObtained = ev.CatalogObtained || dumpEv.CatalogObtained
			mergeLists(&ev.CatalogModels, dumpEv.CatalogModels)
			mergeLists(&ev.ConfiguredProviders, dumpEv.ConfiguredProviders)
			mergeLists(&ev.AuthenticatedProviders, dumpEv.AuthenticatedProviders)
		}
	}

	if opts.OpenCodeBin != "" {
		binEv, runTools, runFailures := probeDoctorBinary(opts.OpenCodeBin)
		tools = append(tools, runTools...)
		failures = append(failures, runFailures...)
		ev.CatalogObtained = ev.CatalogObtained || binEv.CatalogObtained
		mergeLists(&ev.CatalogModels, binEv.CatalogModels)
		mergeLists(&ev.ConfiguredProviders, binEv.ConfiguredProviders)
		mergeLists(&ev.AuthenticatedProviders, binEv.AuthenticatedProviders)
	}

	return ev, tools, failures
}

// labelDoctorModel applies the evidence ladder. Any failure short-circuits to
// unavailable when no catalog listing could be produced at all, so a broken
// tool never silently degrades into a confident "unknown".
func labelDoctorModel(model string, ev DoctorEvidence, anyFailure bool) (DoctorLabel, string) {
	if !doctorModelValid(model) {
		return DoctorUnknown, "rejected by the offline model-route grammar check (no inspection applied)"
	}
	base, _ := splitDoctorVariant(model)
	provider := doctorProviderOf(model)

	if ev.CatalogObtained && containsString(ev.CatalogModels, base) {
		if containsString(ev.AuthenticatedProviders, provider) {
			note := "listed in a local catalog and provider has an explicit local authentication marker"
			if provider == "openai" {
				note += "; marker evidence only — never restate this as ChatGPT-plan entitlement"
			}
			return DoctorAuthenticated, note
		}
		if provider == "openai" {
			return DoctorCatalogVisible, "listed in a local catalog; openai/* is a route only and catalog presence is not proof of subscription or entitlement"
		}
		return DoctorCatalogVisible, "listed in a local catalog; authentication not verified from this evidence"
	}
	if containsString(ev.ConfiguredProviders, provider) || containsString(ev.AuthenticatedProviders, provider) {
		if provider == "openai" {
			return DoctorConfigured, "provider appears in local configuration; openai/* is a route only and is never treated as subscription proof"
		}
		return DoctorConfigured, "provider appears in local configuration; model was not seen in a local catalog listing"
	}
	if ev.CatalogObtained {
		return DoctorUnknown, "a local catalog listing was produced but neither the model nor its provider appears in it"
	}
	if anyFailure {
		return DoctorUnavailable, "the requested local inspection tool failed or was unusable, so no evidence was produced"
	}
	return DoctorUnknown, "local sources succeeded but contain no evidence about this provider"
}

// doctorModelValid is the grammar gate. The normative validator is B1's
// validateModel; doctor adds the empty-route rejection on top.
func doctorModelValid(model string) bool {
	if model == "" {
		return false
	}
	return validateModel(model) == nil
}

func splitDoctorVariant(model string) (base, variant string) {
	if i := strings.IndexByte(model, '#'); i >= 0 {
		return model[:i], model[i+1:]
	}
	return model, ""
}

func doctorProviderOf(model string) string {
	base, _ := splitDoctorVariant(model)
	if i := strings.IndexByte(base, '/'); i > 0 {
		return base[:i]
	}
	return base
}

// readDoctorDump parses one local JSON dump strictly enough for diagnostics:
// same 1 MiB cap as fleet.json, UTF-8 validation, object shape, and
// grammar-valid route/provider strings only. Content is never echoed.
func readDoctorDump(path string) (DoctorEvidence, error) {
	f, err := os.Open(path)
	if err != nil {
		return DoctorEvidence{}, fmt.Errorf("catalog dump unreadable at %s", path)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxConfigBytes+1))
	if err != nil {
		return DoctorEvidence{}, fmt.Errorf("catalog dump unreadable at %s", path)
	}
	if len(data) > MaxConfigBytes {
		return DoctorEvidence{}, fmt.Errorf("catalog dump at %s exceeds the 1 MiB inspection cap", path)
	}
	if !utf8.Valid(data) {
		return DoctorEvidence{}, fmt.Errorf("catalog dump at %s is not valid UTF-8", path)
	}
	var top map[string]json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&top); err != nil {
		return DoctorEvidence{}, fmt.Errorf("catalog dump at %s is not a valid JSON object", path)
	}
	var ev DoctorEvidence
	if raw, ok := top["providers"]; ok {
		var providers []struct {
			ID   string `json:"id"`
			Auth string `json:"auth"`
		}
		if err := json.Unmarshal(raw, &providers); err != nil {
			return DoctorEvidence{}, fmt.Errorf("catalog dump at %s has an unreadable providers list", path)
		}
		for _, p := range providers {
			trimmed := strings.TrimSpace(p.ID)
			if trimmed == "" || !doctorProviderValid(trimmed) {
				continue
			}
			ev.ConfiguredProviders = append(ev.ConfiguredProviders, trimmed)
			if doctorAuthMarkers[strings.ToLower(strings.TrimSpace(p.Auth))] {
				ev.AuthenticatedProviders = append(ev.AuthenticatedProviders, trimmed)
			}
		}
	}
	if raw, ok := top["models"]; ok {
		var models []string
		if err := json.Unmarshal(raw, &models); err != nil {
			return DoctorEvidence{}, fmt.Errorf("catalog dump at %s has an unreadable models list", path)
		}
		ev.CatalogObtained = true
		for _, m := range models {
			trimmed := strings.TrimSpace(m)
			if doctorModelValid(trimmed) {
				ev.CatalogModels = append(ev.CatalogModels, trimmed)
			}
		}
	}
	return ev, nil
}

var doctorProviderRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

func doctorProviderValid(p string) bool { return doctorProviderRe.MatchString(p) }

// probeDoctorBinary inspects a user-named local executable with two fixed,
// read-only argv forms: `--help` and the advertised `models` listing. The
// binary's own behavior is outside this CLI's guarantees; its output text is
// never echoed into failures, and agentic-sdd itself performs no network call.
func probeDoctorBinary(path string) (DoctorEvidence, []string, []string) {
	var ev DoctorEvidence
	label := path + " (local exec: --help"
	if abs, err := filepath.Abs(path); err == nil {
		_ = abs // path is reported as given; abs only proves it is resolvable
	}
	if _, err := os.Stat(path); err != nil {
		return ev, nil, []string{fmt.Sprintf("opencode binary not found at %s", path)}
	}
	help, code, err := runDoctorListing(path, "--help")
	if err != nil {
		return ev, nil, []string{fmt.Sprintf("local opencode --help failed at %s (exit %d; output not shown)", path, code)}
	}
	if len(doctorListingWord.FindStringSubmatch(help)) == 0 {
		return ev, nil, []string{fmt.Sprintf("opencode at %s does not advertise a local models listing; catalog evidence unavailable", path)}
	}
	out, code, err := runDoctorListing(path, "models")
	if err != nil {
		return ev,
			[]string{label + ")"},
			[]string{fmt.Sprintf("local opencode models listing failed at %s (exit %d; output not shown)", path, code)}
	}
	ev.CatalogObtained = true
	for _, line := range strings.Split(out, "\n") {
		route := strings.TrimSpace(line)
		if !doctorModelValid(route) {
			continue
		}
		ev.CatalogModels = append(ev.CatalogModels, route)
		if p := doctorProviderOf(route); !containsString(ev.ConfiguredProviders, p) {
			ev.ConfiguredProviders = append(ev.ConfiguredProviders, p)
		}
	}
	ev.CatalogModels = sortedCopy(ev.CatalogModels)
	ev.ConfiguredProviders = sortedCopy(ev.ConfiguredProviders)
	return ev, []string{label + ", models)"}, nil
}

type doctorLimitBuffer struct {
	buf   bytes.Buffer
	limit int
}

func (b *doctorLimitBuffer) Write(p []byte) (int, error) {
	if b.buf.Len() >= b.limit {
		return len(p), nil
	}
	rem := b.limit - b.buf.Len()
	if len(p) > rem {
		b.buf.Write(p[:rem])
		return len(p), nil
	}
	return b.buf.Write(p)
}

// runDoctorListing executes one fixed argv pair against the local binary with
// a bounded timeout and capped stdout. Only stdout is captured (up to 1 MiB);
// stderr is discarded so binary chatter can never leak into reports or process errors.
func runDoctorListing(path string, arg string) (string, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, arg)
	cmd.Stdin = nil
	stdout := &doctorLimitBuffer{limit: MaxConfigBytes}
	cmd.Stdout = stdout
	err := cmd.Run()
	if err != nil {
		code := -1
		var ee *exec.ExitError
		if asExitError(err, &ee) {
			code = ee.ExitCode()
		}
		return stdout.buf.String(), code, err
	}
	return stdout.buf.String(), 0, nil
}

func asExitError(err error, target **exec.ExitError) bool {
	ee, ok := err.(*exec.ExitError)
	if ok {
		*target = ee
	}
	return ok
}

// DoctorModelsFromConfig is the read-only pre-read used by `fleet doctor`:
// it loads the strict personal fleet.json through LoadConfigFile (rejecting non-regular
// files and bounding reads) and returns every declared model route (enabled or not)
// sorted by instance ID. It reads no other file and writes nothing.
func DoctorModelsFromConfig(path string) ([]string, error) {
	cfg, _, _, err := LoadConfigFile(path)
	if err != nil {
		return nil, err
	}
	var out []string
	seen := map[string]bool{}
	for _, id := range sortedInstanceIDs(cfg.Instances) {
		model := cfg.Instances[id].Model
		if model == "" || seen[model] {
			continue
		}
		seen[model] = true
		out = append(out, model)
	}
	return out, nil
}

func containsString(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func isWindows() bool { return runtime.GOOS == "windows" }
