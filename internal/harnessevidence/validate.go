package harnessevidence

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"agentic-sdd/internal/harnesscheck"
)

// Observations maps a JSON pointer to the independently observed value for
// that receipt claim. Only the permittered native pointers are consulted:
// /session_id, /cost_usd, /model/provider, /model/id and /model/variant.
// Observation values are caller-supplied and are never certified provenance.
type Observations map[string]any

// ValidateInput carries the independently supplied observations a receipt is
// validated against. None of it is derived from the receipt itself.
type ValidateInput struct {
	Workspace        string // absolute workspace path
	Identity         Identity
	AuthorizationRef string // independently reconciled current authorization; "" when there is none
	Contract         []byte // trusted check contract bytes
	Result           []byte // independently observed harnesscheck result.json bytes
	EvidenceRoot     string // private external evidence root (absolute)
	LogDigests       map[string]string
	Observations     Observations
	PaidAuthorized   bool
}

// Validate parses and semantically validates receipt bytes against
// independently supplied current observations. It checks consistency and
// freshness only: it never supplies consent, certifies the observer, resolves
// findings or evaluates review reasoning, and it modifies no state.
func Validate(data []byte, in ValidateInput) error {
	receipt, err := ParseReceipt(data)
	if err != nil {
		return err
	}
	return validateReceipt(receipt, in)
}

func validateReceipt(r Receipt, in ValidateInput) error {
	if r.Mode == "direct-fix" && r.WorkflowState != nil {
		return fmt.Errorf("direct fix has no canonical task state")
	}
	if err := checkObservations(r, in.Observations); err != nil {
		return err
	}
	if r.Limits.PaidAuthorized != in.PaidAuthorized {
		return fmt.Errorf("receipt cannot create paid authorization")
	}
	contract, err := harnesscheck.Parse(in.Contract)
	if err != nil {
		return fmt.Errorf("invalid trusted contract")
	}
	contractDigest := sha256Hex(in.Contract)
	result, err := parseResult(in.Result)
	if err != nil {
		return err
	}
	workspace, err := resolveExisting(in.Workspace)
	if err != nil {
		return fmt.Errorf("workspace unavailable")
	}
	if r.Workspace != workspace || result.Workspace != r.Workspace {
		return fmt.Errorf("workspace changed; reconcile location before reuse")
	}
	if err := checkEvidenceRoot(in.EvidenceRoot, workspace); err != nil {
		return err
	}
	if !r.Identity.equal(in.Identity) {
		return fmt.Errorf("identity changed; evidence is not fresh")
	}
	if result.ContractSHA256 != contractDigest || !sameOptionalText(r.Identity.Contract, &contractDigest) {
		return fmt.Errorf("contract identity mismatch")
	}
	if err := checkUnknowns(r); err != nil {
		return err
	}
	if r.Execution != "blocked" && (in.AuthorizationRef == "" || !sameOptionalText(r.AuthorizationRef, &in.AuthorizationRef)) {
		return fmt.Errorf("independently reconciled authorization required")
	}
	if r.AuthorizationRef != nil && !sameOptionalText(r.AuthorizationRef, &in.AuthorizationRef) {
		return fmt.Errorf("receipt cannot create authorization")
	}
	if err := checkSelectionCoverage(r, contract); err != nil {
		return err
	}
	if err := checkResultAgreement(r, result); err != nil {
		return err
	}
	ids := make([]string, 0, len(r.Checks))
	for _, c := range r.Checks {
		ids = append(ids, c.ID)
	}
	if !reflect.DeepEqual(ids, r.Selection.CheckIDs) {
		return fmt.Errorf("check coverage missing or duplicated")
	}
	seen := map[string]bool{}
	for _, f := range r.Findings {
		if seen[f.ID] {
			return fmt.Errorf("duplicate finding IDs")
		}
		seen[f.ID] = true
	}
	for _, c := range r.Checks {
		if err := checkEntry(c, r, in); err != nil {
			return fmt.Errorf("%s: %w", c.ID, err)
		}
	}
	if err := checkExecutionConsistency(r); err != nil {
		return err
	}
	if sameOptionalText(r.Review, strp("Ready")) {
		allPassed := len(r.Checks) > 0
		for _, c := range r.Checks {
			allPassed = allPassed && c.Status == "passed"
		}
		if !allPassed || r.Selection.Scope != "required" || r.AuthorizationRef == nil {
			return fmt.Errorf("Ready lacks required coverage or resolved findings")
		}
		for _, f := range r.Findings {
			if f.Disposition != "disputed" {
				return fmt.Errorf("Ready lacks required coverage or resolved findings")
			}
		}
	}
	return nil
}

// checkObservations compares every native/model/billing claim in the receipt
// with an independently supplied observation. A null claim is consistent with
// a missing or null observation; a non-null claim must match exactly.
func checkObservations(r Receipt, observations Observations) error {
	measured := map[string]any{}
	if r.SessionID != nil {
		measured["/session_id"] = *r.SessionID
	} else {
		measured["/session_id"] = nil
	}
	if r.CostUSD != nil {
		measured["/cost_usd"] = *r.CostUSD
	} else {
		measured["/cost_usd"] = nil
	}
	for ptr, value := range map[string]*string{
		"/model/provider": r.Model.Provider,
		"/model/id":       r.Model.ID,
		"/model/variant":  r.Model.Variant,
	} {
		if value != nil {
			measured[ptr] = *value
		} else {
			measured[ptr] = nil
		}
	}
	for ptr, value := range measured {
		observed, ok := observations[ptr]
		if !ok {
			observed = nil
		}
		if !equalObserved(value, observed) {
			return fmt.Errorf("native/model/billing claim lacks independent observation")
		}
	}
	return nil
}

func equalObserved(claimed, observed any) bool {
	if claimed == nil {
		return observed == nil
	}
	switch value := claimed.(type) {
	case string:
		text, ok := observed.(string)
		return ok && text == value
	case float64:
		number, ok := toFloat(observed)
		return ok && number == value
	}
	return false
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

// checkUnknowns requires a nonempty reason for every null observation outside
// the structural nulls, and rejects reasons that describe nothing null.
func checkUnknowns(r Receipt) error {
	nulls := receiptNulls(r)
	structural := map[string]bool{"/review": true, "/workflow_state": true}
	if r.Mode == "direct-fix" {
		structural["/feature"] = true
		structural["/batch"] = true
	}
	for i, c := range r.Checks {
		if c.Status != "passed" {
			structural[fmt.Sprintf("/checks/%d/exit_code", i)] = true
			structural[fmt.Sprintf("/checks/%d/started_at", i)] = true
			structural[fmt.Sprintf("/checks/%d/ended_at", i)] = true
			structural[fmt.Sprintf("/checks/%d/evidence", i)] = true
		}
	}
	for ptr := range nulls {
		if !structural[ptr] && strings.TrimSpace(r.Unknowns[ptr]) == "" {
			return fmt.Errorf("missing unknown-observation reason")
		}
	}
	for ptr, reason := range r.Unknowns {
		if !nulls[ptr] || strings.TrimSpace(reason) == "" {
			return fmt.Errorf("unknown reason does not describe a null observation")
		}
	}
	return nil
}

// receiptNulls lists the JSON pointers of every null value in a schema-valid
// receipt. Only typed fields can be null, so this walk is exhaustive.
func receiptNulls(r Receipt) map[string]bool {
	nulls := map[string]bool{}
	if r.SessionID == nil {
		nulls["/session_id"] = true
	}
	if r.AuthorizationRef == nil {
		nulls["/authorization_ref"] = true
	}
	if r.Feature == nil {
		nulls["/feature"] = true
	}
	if r.Batch == nil {
		nulls["/batch"] = true
	}
	if r.Profile == nil {
		nulls["/profile"] = true
	}
	if r.Review == nil {
		nulls["/review"] = true
	}
	if r.WorkflowState == nil {
		nulls["/workflow_state"] = true
	}
	if r.CostUSD == nil {
		nulls["/cost_usd"] = true
	}
	if r.Limits.SpendUSD == nil {
		nulls["/limits/spend_usd"] = true
	}
	for name, value := range map[string]*string{
		"base_revision": r.Identity.BaseRevision, "diff": r.Identity.Diff, "config": r.Identity.Config,
		"skills": r.Identity.Skills, "environment": r.Identity.Environment, "runtime": r.Identity.Runtime,
		"harness": r.Identity.Harness, "contract": r.Identity.Contract,
	} {
		if value == nil {
			nulls["/identity/"+name] = true
		}
	}
	for name, value := range map[string]*string{
		"provider": r.Model.Provider, "id": r.Model.ID, "variant": r.Model.Variant,
	} {
		if value == nil {
			nulls["/model/"+name] = true
		}
	}
	for i, c := range r.Checks {
		if c.ExitCode == nil {
			nulls[fmt.Sprintf("/checks/%d/exit_code", i)] = true
		}
		if c.StartedAt == nil {
			nulls[fmt.Sprintf("/checks/%d/started_at", i)] = true
		}
		if c.EndedAt == nil {
			nulls[fmt.Sprintf("/checks/%d/ended_at", i)] = true
		}
		if c.Evidence == nil {
			nulls[fmt.Sprintf("/checks/%d/evidence", i)] = true
		}
	}
	return nulls
}

func checkSelectionCoverage(r Receipt, contract harnesscheck.Contract) error {
	ids := map[string]bool{}
	for _, c := range contract.Checks {
		if ids[c.ID] {
			return fmt.Errorf("duplicate trusted contract IDs")
		}
		ids[c.ID] = true
	}
	selected := map[string]bool{}
	for _, id := range r.Selection.CheckIDs {
		selected[id] = true
	}
	var expected []string
	for _, c := range contract.Checks {
		if r.Selection.Scope == "focused" {
			if selected[c.ID] {
				expected = append(expected, c.ID)
			}
		} else if containsScope(c.Scopes, r.Selection.Scope) {
			expected = append(expected, c.ID)
		}
	}
	if len(r.Selection.CheckIDs) == 0 || !reflect.DeepEqual(r.Selection.CheckIDs, expected) {
		return fmt.Errorf("selection coverage mismatch")
	}
	return nil
}

func containsScope(scopes []string, scope string) bool {
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

func checkResultAgreement(r Receipt, result harnesscheck.Result) error {
	if r.Selection.Scope != result.Selection.Scope || !reflect.DeepEqual(r.Selection.CheckIDs, result.Selection.CheckIDs) {
		return fmt.Errorf("receipt disagrees with independent check result")
	}
	converted, err := ChecksFromResult(result)
	if err != nil || !reflect.DeepEqual(r.Checks, converted) {
		return fmt.Errorf("receipt disagrees with independent check result")
	}
	if r.Execution != result.Execution {
		return fmt.Errorf("receipt disagrees with independent check result")
	}
	return nil
}

func parseResult(data []byte) (harnesscheck.Result, error) {
	return ParseResult(data)
}

// ParseResult strictly decodes independently observed runner result bytes.
func ParseResult(data []byte) (harnesscheck.Result, error) {
	var result harnesscheck.Result
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&result); err != nil {
		return result, fmt.Errorf("independent check result required")
	}
	if err := dec.Decode(new(any)); err == nil {
		return result, fmt.Errorf("independent check result required")
	}
	if result.Version != 1 {
		return result, fmt.Errorf("independent check result required")
	}
	return result, nil
}

// ChecksFromResult converts runner check results into the receipt's exact
// string representation so timestamp formatting differences are visible.
func ChecksFromResult(result harnesscheck.Result) ([]CheckEntry, error) {
	data, err := json.Marshal(result.Checks)
	if err != nil {
		return nil, fmt.Errorf("check result conversion unavailable")
	}
	var out []CheckEntry
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("check result conversion unavailable")
	}
	return out, nil
}

func checkEntry(c CheckEntry, r Receipt, in ValidateInput) error {
	if c.StartedAt != nil && c.EndedAt != nil {
		start, startErr := time.Parse(time.RFC3339, *c.StartedAt)
		end, endErr := time.Parse(time.RFC3339, *c.EndedAt)
		if startErr != nil || endErr != nil || end.Before(start) {
			return fmt.Errorf("check time order invalid")
		}
	}
	if c.Status == "passed" {
		if c.ExitCode == nil || *c.ExitCode != 0 || c.StartedAt == nil || c.EndedAt == nil || c.Evidence == nil || c.Truncated {
			return fmt.Errorf("passed check lacks complete evidence")
		}
		for _, value := range r.Identity.fields() {
			if value == nil {
				return fmt.Errorf("unknown identity prevents pass reuse")
			}
		}
	}
	if c.Status == "blocked" || c.Status == "not_run" {
		if c.ExitCode != nil || c.StartedAt != nil {
			return fmt.Errorf("nonexecuted check has execution observations")
		}
	}
	if c.Evidence != nil {
		if err := privateRegular(*c.Evidence, in.EvidenceRoot); err != nil {
			return err
		}
		data, err := os.ReadFile(*c.Evidence)
		if err != nil || sha256Hex(data) != in.LogDigests[*c.Evidence] {
			return fmt.Errorf("evidence digest changed or unavailable")
		}
	}
	return nil
}

func checkExecutionConsistency(r Receipt) error {
	allPassed := true
	firstNonPass := len(r.Checks)
	for i, c := range r.Checks {
		if c.Status != "passed" {
			allPassed = false
			if i < firstNonPass {
				firstNonPass = i
			}
		}
	}
	expected := "completed"
	if !allPassed {
		expected = r.Checks[firstNonPass].Status
	}
	if expected == "not_run" || r.Execution != expected {
		return fmt.Errorf("execution disagrees with check outcomes")
	}
	if firstNonPass < len(r.Checks) {
		for _, c := range r.Checks[firstNonPass+1:] {
			if c.Status != "not_run" {
				return fmt.Errorf("execution disagrees with check outcomes")
			}
		}
	}
	return nil
}

// privateRegular enforces the evidence boundary: an absolute path inside the
// authorized private root, no traversal or symlinked components, a bounded
// regular file and a private root. It assumes a quiescent trusted filesystem.
func privateRegular(path, root string) error {
	if path == "" || root == "" || !filepath.IsAbs(path) || !filepath.IsAbs(root) {
		return fmt.Errorf("evidence outside authorized root")
	}
	var parts []string
	for _, p := range []string{path, root} {
		parts = strings.Split(filepath.ToSlash(p), "/")
		for _, part := range parts {
			if part == ".." {
				return fmt.Errorf("evidence outside authorized root")
			}
		}
	}
	if !insidePath(root, path) {
		return fmt.Errorf("evidence outside authorized root")
	}
	for _, ancestor := range ancestors(path) {
		if info, err := os.Lstat(ancestor); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("evidence symlink refused")
		}
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("evidence missing")
	}
	rootInfo, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("evidence missing")
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || rootInfo.Mode().Perm()&0o077 != 0 || info.Size() > 10<<20 {
		return fmt.Errorf("evidence is not bounded and private")
	}
	return nil
}

func ancestors(path string) []string {
	out := []string{path}
	for current := filepath.Dir(path); ; current = filepath.Dir(current) {
		out = append(out, current)
		if current == string(filepath.Separator) || current == "." {
			break
		}
		if len(out) > 256 {
			break
		}
	}
	return out
}

func insidePath(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func checkEvidenceRoot(root, workspace string) error {
	if root == "" || !filepath.IsAbs(root) {
		return fmt.Errorf("evidence root must be outside target workspace")
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("evidence root must be outside target workspace")
	}
	if insidePath(workspace, resolvedRoot) {
		return fmt.Errorf("evidence root must be outside target workspace")
	}
	return nil
}

func resolveExisting(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func strp(s string) *string { return &s }
