package harnessevidence

// Safe external receipt export: scrub known literal redactions and common
// bearer/key-value credential forms in memory BEFORE any bytes are written,
// re-validate the sanitized receipt, then write once, exclusively, 0600, to a
// new file in an existing private parent outside the target checkout. This is
// not universal secret detection: encoded, split or unidentified secrets and
// sensitive prose still require human curation. Raw check logs are never
// sanitized here. No existing evidence is overwritten.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const maxExportBytes = 1 << 20

var (
	bearerPattern = regexp.MustCompile(`(?i)(bearer\s+)[^\s"'<>]+`)
	secretPattern = regexp.MustCompile(`(?i)((?:api[_-]?key|token|password|secret)\s*[:=]\s*)[^\s,;"'<>]+`)
)

// Export validates, redacts and writes the receipt bytes to output.
// Redactions are nonempty literal strings supplied from a private source,
// never from CLI arguments or receipt content.
func Export(data []byte, output, workspace string, redactions []string) error {
	receipt, err := ParseReceipt(data)
	if err != nil {
		return err
	}
	for _, s := range redactions {
		if s == "" {
			return fmt.Errorf("redactions must be nonempty literal strings")
		}
	}
	sanitized, err := scrubReceipt(receipt, redactions)
	if err != nil {
		return err
	}
	sanitizedData, err := marshalReceipt(sanitized)
	if err != nil {
		return err
	}
	if _, err := ParseReceipt(sanitizedData); err != nil {
		return fmt.Errorf("redaction produced an invalid receipt")
	}
	if len(sanitizedData) > maxExportBytes {
		return fmt.Errorf("receipt export exceeds bound")
	}
	if err := checkExportTarget(output, workspace); err != nil {
		return err
	}
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("receipt destination unavailable; no overwrite")
	}
	_, writeErr := file.Write(sanitizedData)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	if closeErr := file.Close(); writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		return fmt.Errorf("receipt export incomplete; destination must be discarded")
	}
	return nil
}

func scrubReceipt(r Receipt, redactions []string) (Receipt, error) {
	scrub := func(s string) string {
		for _, secret := range sortedRedactions(redactions) {
			s = strings.ReplaceAll(s, secret, "[REDACTED]")
		}
		s = bearerPattern.ReplaceAllString(s, "${1}[REDACTED]")
		return secretPattern.ReplaceAllString(s, "${1}[REDACTED]")
	}
	apply := func(value *string) {
		if value != nil {
			*value = scrub(*value)
		}
	}
	apply(&r.RunID)
	apply(&r.Workspace)
	apply(&r.RequestRef)
	apply(r.AuthorizationRef)
	apply(r.Feature)
	apply(r.Batch)
	for i := range r.TaskIDs {
		r.TaskIDs[i] = scrub(r.TaskIDs[i])
	}
	for i := range r.AllowedEffects {
		r.AllowedEffects[i] = scrub(r.AllowedEffects[i])
	}
	apply(r.Profile)
	for i := range r.Tools {
		r.Tools[i] = scrub(r.Tools[i])
	}
	for _, value := range r.Identity.fields() {
		apply(value)
	}
	apply(r.Model.Provider)
	apply(r.Model.ID)
	apply(r.Model.Variant)
	apply(r.SessionID)
	for i := range r.Checks {
		apply(&r.Checks[i].ID)
		apply(r.Checks[i].Evidence)
		apply(&r.Checks[i].Reason)
	}
	apply(r.Review)
	apply(r.WorkflowState)
	for i := range r.Findings {
		r.Findings[i].ID = scrub(r.Findings[i].ID)
		r.Findings[i].Location = scrub(r.Findings[i].Location)
		r.Findings[i].Impact = scrub(r.Findings[i].Impact)
		r.Findings[i].Evidence = scrub(r.Findings[i].Evidence)
	}
	for i := range r.AffectedFiles {
		r.AffectedFiles[i] = scrub(r.AffectedFiles[i])
	}
	apply(&r.NextAction)
	unknowns := make(map[string]string, len(r.Unknowns))
	for key, reason := range r.Unknowns {
		if scrub(key) != key {
			return r, fmt.Errorf("sensitive field name refused")
		}
		unknowns[key] = scrub(reason)
	}
	r.Unknowns = unknowns
	return r, nil
}

func sortedRedactions(redactions []string) []string {
	ordered := append([]string(nil), redactions...)
	sort.SliceStable(ordered, func(i, j int) bool { return len(ordered[i]) > len(ordered[j]) })
	return ordered
}

func marshalReceipt(r Receipt) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return nil, fmt.Errorf("receipt encoding unavailable")
	}
	return buf.Bytes(), nil
}

// checkExportTarget enforces the export boundary: an absolute new file in an
// existing private directory outside the target checkout, with no symlinked
// components anywhere from the file to the filesystem root.
func checkExportTarget(output, workspace string) error {
	if output == "" || !filepath.IsAbs(output) {
		return fmt.Errorf("receipt must be outside target checkout")
	}
	for _, part := range strings.Split(filepath.ToSlash(output), "/") {
		if part == ".." {
			return fmt.Errorf("receipt must be outside target checkout")
		}
	}
	resolvedWorkspace, err := resolveExisting(workspace)
	if err != nil {
		return fmt.Errorf("workspace unavailable")
	}
	if insidePath(resolvedWorkspace, output) {
		return fmt.Errorf("receipt must be outside target checkout")
	}
	for _, ancestor := range ancestors(output) {
		if info, err := os.Lstat(ancestor); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("receipt output symlink refused")
		}
	}
	parent := filepath.Dir(output)
	info, err := os.Stat(parent)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("existing private external parent required")
	}
	if _, err := os.Lstat(output); err == nil {
		return fmt.Errorf("receipt destination unavailable; no overwrite")
	}
	return nil
}
