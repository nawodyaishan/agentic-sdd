package fleet

// Feature 014 B5 / T09 — focused tests for the opt-in offline fleet doctor.
// Seam: RunDoctor(DoctorOptions) DoctorReport and DoctorModelsFromConfig(path).
// All fixtures are isolated (t.TempDir); no network, no model calls, no real
// user home. Acceptance refs: spec AC-19..AC-22, R6, NFR-05, §8 test 14.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// recordingProbe is the only inspection source the pure-labeling tests use,
// so zero tool calls beyond the counted probe can happen.
type recordingProbe struct {
	ev       DoctorEvidence
	err      error
	calls    int
	lastCall DoctorEvidence
}

func (p *recordingProbe) Evidence() (DoctorEvidence, error) {
	p.calls++
	return p.ev, p.err
}

func wantEntries(t *testing.T, rep DoctorReport, want map[string]DoctorLabel) {
	t.Helper()
	if len(rep.Entries) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(rep.Entries), len(want), rep.Entries)
	}
	for _, e := range rep.Entries {
		wl, ok := want[e.Model]
		if !ok {
			t.Fatalf("unexpected entry %q", e.Model)
		}
		if e.Label != wl {
			t.Errorf("model %q label = %q, want %q (reason %q)", e.Model, e.Label, wl, e.Reason)
		}
	}
}

func TestFleetDoctorOfflineByDefault(t *testing.T) {
	models := []string{
		"opencode-go/deepseek-v4.1-flash#high",
		"9router/ag/gemini-3.8-flash-high",
		"openai/gpt-5",
		"not a valid route",
	}
	probe := &recordingProbe{}
	// Default mode: no Probe, no OpenCodeBin, no CatalogDump — nothing may be consulted.
	rep := RunDoctor(DoctorOptions{Models: models})
	if !rep.Offline {
		t.Fatalf("default RunDoctor must report offline mode, got %+v", rep)
	}
	if len(rep.ToolsInspected) != 0 {
		t.Errorf("offline mode must inspect no tools, got %v", rep.ToolsInspected)
	}
	if probe.calls != 0 {
		t.Fatalf("offline mode must not consult probes (%d calls)", probe.calls)
	}
	for _, e := range rep.Entries {
		if e.Label != DoctorNotChecked {
			t.Errorf("model %q label = %q, want %q in offline default", e.Model, e.Label, DoctorNotChecked)
		}
	}
	// Entries mirror the caller-provided order, including malformed routes.
	for i, e := range rep.Entries {
		if e.Model != models[i] {
			t.Fatalf("entry %d = %q, want %q (order must mirror input)", i, e.Model, models[i])
		}
	}
	// The malformed route is still "not checked" offline: nothing was checked.
}

func TestFleetDoctorLabelsFromLocalEvidence(t *testing.T) {
	probe := &recordingProbe{ev: DoctorEvidence{
		CatalogObtained: true,
		CatalogModels: []string{
			"opencode-go/deepseek-v4.1-flash",
			"9router/ag/gemini-3.8-flash-high",
			"openai/gpt-5",
		},
		ConfiguredProviders:    []string{"opencode-go", "9router", "openai"},
		AuthenticatedProviders: []string{"opencode-go"},
	}}
	rep := RunDoctor(DoctorOptions{Probe: probe, Models: []string{
		"opencode-go/deepseek-v4.1-flash#high", // in catalog + explicit local auth marker
		"9router/ag/gemini-3.8-flash-high",     // in catalog, no auth evidence
		"opencode-go/absent-model",             // provider configured, model not listed
		"unseenprovider/some-model",            // provider unknown, fresh catalog says no
		"OPENED GO/bad id",                     // malformed route
	}})
	wantEntries(t, rep, map[string]DoctorLabel{
		"opencode-go/deepseek-v4.1-flash#high": DoctorAuthenticated,
		"9router/ag/gemini-3.8-flash-high":     DoctorCatalogVisible,
		"opencode-go/absent-model":             DoctorConfigured,
		"unseenprovider/some-model":            DoctorUnknown,
		"OPENED GO/bad id":                     DoctorUnknown,
	})
	if probe.calls != 1 {
		t.Errorf("evidence must be gathered once, got %d calls", probe.calls)
	}
	// Authenticated reason must cite the local marker, not the route string.
	for _, e := range rep.Entries {
		if e.Model == "opencode-go/deepseek-v4.1-flash#high" && !strings.Contains(strings.ToLower(e.Reason), "marker") {
			t.Errorf("authenticated reason must cite local evidence: %q", e.Reason)
		}
	}
}

// TestFleetDoctorOpenAIRouteOnly covers AC-20 and spec §8 test 14:
// 9Router and OpenAI routes present with catalog visibility but no auth
// markers must never be reported as authenticated, and nothing may call a
// network or model.
func TestFleetDoctorOpenAIRouteOnly(t *testing.T) {
	dump := filepath.Join(t.TempDir(), "catalog.json")
	contents := `{"providers":[{"id":"9router"},{"id":"openai"}],
		"models":["9router/ag/gemini-3.8-flash-high","openai/gpt-5"]}`
	if err := os.WriteFile(dump, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	rep := RunDoctor(DoctorOptions{
		CatalogDump: dump,
		Models:      []string{"9router/ag/gemini-3.8-flash-high", "openai/gpt-5"},
	})
	for _, e := range rep.Entries {
		if e.Label == DoctorAuthenticated {
			t.Fatalf("route strings and catalog presence alone must never yield authenticated: %+v", e)
		}
		if e.Label != DoctorCatalogVisible {
			t.Errorf("model %q label = %q, want %q", e.Model, e.Label, DoctorCatalogVisible)
		}
	}
	if !strings.Contains(rep.Entries[1].Reason, "not proof") {
		t.Errorf("openai entry must warn that presence is not entitlement proof: %q", rep.Entries[1].Reason)
	}
	tools := strings.Join(rep.ToolsInspected, " ")
	if !strings.Contains(tools, dump) {
		t.Errorf("report must name the exact inspected file, got %q", tools)
	}
	// Offline default over the same routes: still no inference at all.
	offline := RunDoctor(DoctorOptions{Models: []string{"9router/ag/gemini-3.8-flash-high", "openai/gpt-5"}})
	for _, e := range offline.Entries {
		if e.Label != DoctorNotChecked {
			t.Fatalf("offline default inferred %q for %q; nothing may be claimed", e.Label, e.Model)
		}
	}
}

func TestFleetDoctorAuthMarkerRequiredForAuthenticated(t *testing.T) {
	dump := filepath.Join(t.TempDir(), "catalog.json")
	contents := `{"providers":[{"id":"openai","auth":"oauth"}],
		"models":["openai/gpt-5"],
		"note":"local export, redacted"}`
	if err := os.WriteFile(dump, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	rep := RunDoctor(DoctorOptions{CatalogDump: dump, Models: []string{"openai/gpt-5"}})
	if len(rep.Entries) != 1 || rep.Entries[0].Label != DoctorAuthenticated {
		t.Fatalf("explicit local oauth marker is independent evidence; got %+v", rep.Entries)
	}
	if !strings.Contains(rep.Entries[0].Reason, "marker") {
		t.Errorf("reason must attribute the label to the local marker: %q", rep.Entries[0].Reason)
	}
	// The dump's unrelated content must not leak into the report (NFR-08).
	for _, e := range rep.Entries {
		if strings.Contains(e.Reason, "redacted") || strings.Contains(e.Model, "redacted") {
			t.Fatalf("report echoed private dump content: %+v", e)
		}
	}
}

func TestFleetDoctorUnknownMarkerIsNotAuthentication(t *testing.T) {
	dump := filepath.Join(t.TempDir(), "catalog.json")
	contents := `{"providers":[{"id":"opencode-go","auth":"maybe-some-day"}],
		"models":["opencode-go/deepseek-v4.1-flash"]}`
	if err := os.WriteFile(dump, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	rep := RunDoctor(DoctorOptions{CatalogDump: dump, Models: []string{"opencode-go/deepseek-v4.1-flash"}})
	if rep.Entries[0].Label != DoctorCatalogVisible {
		t.Fatalf("unrecognized auth marker must not yield authenticated, got %+v", rep.Entries[0])
	}
}

func TestFleetDoctorUnavailableWhenInspectionFails(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-opencode-here")
	rep := RunDoctor(DoctorOptions{OpenCodeBin: missing, Models: []string{"opencode-go/deepseek-v4.1-flash"}})
	if rep.Entries[0].Label != DoctorUnavailable {
		t.Fatalf("missing requested binary must label unavailable, got %+v", rep.Entries[0])
	}
	if len(rep.Failures) == 0 || !strings.Contains(strings.Join(rep.Failures, " "), missing) {
		t.Errorf("failures must name the attempted tool path, got %v", rep.Failures)
	}
	if rep.Offline {
		t.Error("a requested-but-failed inspection is not offline default mode")
	}
}

func TestFleetDoctorProbeErrorYieldsUnavailable(t *testing.T) {
	probe := &recordingProbe{err: fmt.Errorf("simulated local tool failure")}
	rep := RunDoctor(DoctorOptions{Probe: probe, Models: []string{"opencode-go/x1", "9router/ag/y1"}})
	for _, e := range rep.Entries {
		if e.Label != DoctorUnavailable {
			t.Fatalf("failed inspection must label unavailable, got %+v", e)
		}
	}
}

func TestFleetDoctorCatalogProviderOnlyDump(t *testing.T) {
	dump := filepath.Join(t.TempDir(), "providers.json")
	if err := os.WriteFile(dump, []byte(`{"providers":[{"id":"9router","auth":"authenticated"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	rep := RunDoctor(DoctorOptions{CatalogDump: dump, Models: []string{
		"9router/ag/gemini-3.8-flash-high", // provider configured, no catalog listing produced
		"ghostprovider/z",                  // no evidence at all
	}})
	wantEntries(t, rep, map[string]DoctorLabel{
		"9router/ag/gemini-3.8-flash-high": DoctorConfigured,
		"ghostprovider/z":                  DoctorUnknown,
	})
}

func TestFleetDoctorMalformedDumpFailsClosed(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(broken, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	rep := RunDoctor(DoctorOptions{CatalogDump: broken, Models: []string{"opencode-go/deepseek-v4.1-flash"}})
	if rep.Entries[0].Label != DoctorUnavailable {
		t.Fatalf("unparseable inspection source must yield unavailable, got %+v", rep.Entries[0])
	}
	if !strings.Contains(strings.Join(rep.Failures, " "), broken) {
		t.Errorf("failure must name the dump path, got %v", rep.Failures)
	}
	// Errors stay sanitized: the report never echoes raw content.
	joined := fmt.Sprint(rep.Failures) + fmt.Sprint(rep.Entries)
	if strings.Contains(joined, "not json") {
		t.Errorf("report leaked dump content: %s", joined)
	}
}

func TestFleetDoctorHelpResolvedListingOnlyUsesListingSubcommands(t *testing.T) {
	if isWindows() {
		t.Skip("POSIX shell fixture")
	}
	dir := t.TempDir()
	argvLog := filepath.Join(dir, "argv.log")
	bin := filepath.Join(dir, "opencode")
	script := `#!/bin/sh
printf '%s\n' "$*" >> ` + argvLog + `
case "$1" in
  --help) printf 'usage: opencode [command]\n  models   list locally known models\n  run      start a session (must never be invoked)\n' ;;
  models) printf 'opencode-go/deepseek-v4.1-flash\n9router/ag/gemini-3.8-flash-high\n\n' ;;
  *) exit 1 ;;
esac
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	rep := RunDoctor(DoctorOptions{OpenCodeBin: bin, Models: []string{
		"opencode-go/deepseek-v4.1-flash",
		"9router/ag/gemini-3.8-flash-high",
		"opencode-go/unlisted-but-provider-visible",
	}})
	if rep.Entries[0].Label != DoctorCatalogVisible || rep.Entries[1].Label != DoctorCatalogVisible {
		t.Fatalf("catalog listing should make both models visible: %+v", rep.Entries)
	}
	if rep.Entries[2].Label != DoctorConfigured {
		t.Fatalf("provider prefix seen in catalog means configured at best: %+v", rep.Entries[2])
	}
	// The bin never yields authenticated: no local auth markers come from it.
	for _, e := range rep.Entries {
		if e.Label == DoctorAuthenticated {
			t.Fatalf("bin probe must not claim authentication: %+v", e)
		}
	}
	logged, err := os.ReadFile(argvLog)
	if err != nil {
		t.Fatal(err)
	}
	var argvCalls []string
	for _, line := range strings.Split(string(logged), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			argvCalls = append(argvCalls, line)
		}
	}
	if got := strings.Join(argvCalls, "|"); got != "--help|models" {
		t.Fatalf("doctor must run only help + the advertised listing command, got argv %q", got)
	}
	// A binary whose help advertises no listing keeps evidence honest.
	noListing := filepath.Join(dir, "opencode-nomodels")
	if err := os.WriteFile(noListing, []byte("#!/bin/sh\ncase \"$1\" in --help) printf 'opencode — run only\\n';; *) exit 1;; esac\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	rep2 := RunDoctor(DoctorOptions{OpenCodeBin: noListing, Models: []string{"opencode-go/deepseek-v4.1-flash"}})
	if rep2.Entries[0].Label != DoctorUnavailable {
		t.Fatalf("no advertised listing command must leave entries unavailable, got %+v", rep2.Entries[0])
	}
}

func TestFleetDoctorModelRouteGrammar(t *testing.T) {
	probe := &recordingProbe{ev: DoctorEvidence{CatalogObtained: true}}
	cases := map[string]DoctorLabel{
		"opencode/deepseek-v4.1-flash-free": DoctorUnknown, // valid grammar, absent from empty catalog
		"opencode-go/mimo-v2.6-flash#high":  DoctorUnknown, // valid with variant
		"9router/ag/gemini-3.8-flash-high":  DoctorUnknown, // valid lane string
		"openai/gpt-5":                      DoctorUnknown, // valid route, never entitlement
		"UPPERCASE/model":                   DoctorUnknown, // malformed provider case
		"provider/":                         DoctorUnknown, // empty model
		"/model":                            DoctorUnknown, // empty provider
		"provider/model with space":         DoctorUnknown, // whitespace
		"":                                  DoctorUnknown, // empty route
		"provider/model#":                   DoctorUnknown, // empty variant
	}
	models := make([]string, 0, len(cases))
	for m := range cases {
		models = append(models, m)
	}
	rep := RunDoctor(DoctorOptions{Probe: probe, Models: models})
	wantEntries(t, rep, cases)
	// Malformed routes must be caught offline by grammar without any network
	// dependency: the single probe call is the only "inspection" ever made.
	if probe.calls != 1 {
		t.Errorf("one evidence gather expected, got %d", probe.calls)
	}
	for _, e := range rep.Entries {
		if e.Model != "opencode/deepseek-v4.1-flash-free" && e.Model != "opencode-go/mimo-v2.6-flash#high" &&
			e.Model != "9router/ag/gemini-3.8-flash-high" && e.Model != "openai/gpt-5" {
			if !strings.Contains(e.Reason, "grammar") {
				t.Errorf("malformed %q reason must cite the offline grammar check: %q", e.Model, e.Reason)
			}
		}
	}
}

func TestFleetDoctorMergesDumpAndBinEvidence(t *testing.T) {
	if isWindows() {
		t.Skip("POSIX shell fixture")
	}
	dir := t.TempDir()
	dump := filepath.Join(dir, "catalog.json")
	if err := os.WriteFile(dump, []byte(`{"providers":[{"id":"opencode-go","auth":"api-key"}],"models":["opencode-go/mimo-v2.6-flash"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "opencode")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\ncase \"$1\" in --help) echo 'models  list';; models) echo '9router/ag/gemini-3.8-flash-high';; *) exit 1;; esac\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	rep := RunDoctor(DoctorOptions{CatalogDump: dump, OpenCodeBin: bin, Models: []string{
		"opencode-go/mimo-v2.6-flash",      // dump catalog + dump auth marker
		"9router/ag/gemini-3.8-flash-high", // bin catalog only
	}})
	if rep.Entries[0].Label != DoctorAuthenticated {
		t.Fatalf("dump evidence lost in merge: %+v", rep.Entries[0])
	}
	if rep.Entries[1].Label != DoctorCatalogVisible {
		t.Fatalf("bin evidence lost in merge: %+v", rep.Entries[1])
	}
	if len(rep.ToolsInspected) != 2 {
		t.Errorf("both sources must be listed, got %v", rep.ToolsInspected)
	}
}

func TestFleetDoctorEmptyModels(t *testing.T) {
	rep := RunDoctor(DoctorOptions{})
	if len(rep.Entries) != 0 || !rep.Offline {
		t.Fatalf("empty default run must be an inert offline report, got %+v", rep)
	}
}

func TestFleetDoctorModelsFromConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "fleet.json")
	contents := `{
  "schemaVersion": 1,
  "fleet": "personal",
  "preset": "opencode-go",
  "instances": {
    "sdd-implement-9r-1": {"source": "preset:sdd-implement-1", "model": "9router/ag/gemini-3.8-flash-high", "enabled": true},
    "sdd-implement-1": {"source": "preset:sdd-implement-1", "model": "opencode-go/deepseek-v4.1-flash#high", "enabled": true},
    "sdd-openai-architect": {"source": "agent:sdd-openai-architect", "enabled": false},
    "vision": {"source": "agent:sdd-visual-inspector", "model": "opencode-go/mimo-v2.6-flash", "enabled": false}
  }
}`
	if err := os.WriteFile(cfg, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := DoctorModelsFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Sorted by instance id: sdd-implement-1, sdd-implement-9r-1, vision.
	want := []string{
		"opencode-go/deepseek-v4.1-flash#high",
		"9router/ag/gemini-3.8-flash-high",
		"opencode-go/mimo-v2.6-flash",
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("models = %v, want %v (deterministic instance-id order, disabled instances still configured)", got, want)
	}

	// Oversize input is refused before any parse (NFR-06 / R8 cap).
	oversize := filepath.Join(dir, "oversize.json")
	f, err := os.Create(oversize)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"instances":{"a":{"model":"`); err != nil {
		t.Fatal(err)
	}
	for written := 25; written < 1024*1024+64; written += 1024 {
		if _, err := f.WriteString(strings.Repeat("x", 1024)); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()
	if _, err := DoctorModelsFromConfig(oversize); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversize config must be refused with the B1 cap error, got %v", err)
	}

	invalidUTF8 := filepath.Join(dir, "invalid-utf8.json")
	if err := os.WriteFile(invalidUTF8, []byte("{\"instances\":{\"a\":{\"model\":\"x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Append raw bytes that are invalid UTF-8 (a lone continuation byte and
	// an overlong marker) so the pre-read must reject encoding, not JSON.
	f2, err := os.OpenFile(invalidUTF8, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f2.Write([]byte{0x80, 0xC0, 0xAF, '"', '}', '}'}); err != nil {
		t.Fatal(err)
	}
	f2.Close()

	for name, body := range map[string]string{
		"invalid-json": "{not json",
		"wrong-shape":  `[1,2,3]`,
	} {
		bad := filepath.Join(dir, name+".json")
		if err := os.WriteFile(bad, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := DoctorModelsFromConfig(bad); err == nil {
			t.Errorf("%s: malformed config must fail the pre-read", name)
		}
	}
	if _, err := DoctorModelsFromConfig(invalidUTF8); err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Errorf("invalid UTF-8 config must be refused with an encoding error, got %v", err)
	}

	if _, err := DoctorModelsFromConfig(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("missing config file must error")
	}
}
