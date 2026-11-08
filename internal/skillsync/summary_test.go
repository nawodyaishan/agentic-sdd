package skillsync

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Feature 010 / Batch B3 / T05 — Check C3 (parser drift pin).
//
// The parser is only safe if it stays pinned to real producer output: every
// non-empty line of a real preview, apply and restore transcript must be
// classified or match one of the known trailer/notice prefixes. A reworded
// producer line fails here instead of silently producing wrong counts.
func TestSummarizePlanDrift(t *testing.T) {
	home := t.TempDir()
	opts := Options{WithOpenCode: true, OpenCodeFreeFirst: true}

	// Seed one skill that differs (so it is replaced), one that matches the
	// embedded source (so it is up to date), and an existing AGENTS.md (so the
	// Free-first notice line appears).
	source, err := sourceFS("")
	if err != nil {
		t.Fatalf("source: %v", err)
	}
	specContent, err := fs.ReadFile(source, "agentic-sdd-spec/SKILL.md")
	if err != nil {
		t.Fatalf("read embedded spec skill: %v", err)
	}
	claudeDir := filepath.Join(home, ".claude", "skills")
	if err := os.MkdirAll(filepath.Join(claudeDir, "agentic-sdd-plan"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "agentic-sdd-plan", "SKILL.md"), []byte("# locally modified plan skill\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(claudeDir, "agentic-sdd-spec"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "agentic-sdd-spec", "SKILL.md"), specContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "opencode", "AGENTS.md"), []byte("# existing agents file\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Preview: every line must be classified or allowlisted.
	var preview bytes.Buffer
	if err := SyncWithOptions("", home, false, opts, &preview); err != nil {
		t.Fatalf("preview: %v", err)
	}
	summary := SummarizePlan(preview.String())
	if len(summary.Unclassified) != 0 {
		t.Fatalf("preview produced unclassified lines %q\nfull transcript:\n%s", summary.Unclassified, preview.String())
	}
	if summary.Install <= 0 {
		t.Fatalf("preview install count = %d, want > 0\n%s", summary.Install, preview.String())
	}
	if summary.Replace != 1 {
		t.Fatalf("preview replace count = %d, want 1\n%s", summary.Replace, preview.String())
	}
	if summary.UpToDate != 1 {
		t.Fatalf("preview up-to-date count = %d, want 1\n%s", summary.UpToDate, preview.String())
	}
	wantClients := []string{"agents", "codex", "claude", "agy", "opencode-agents"}
	gotClients := make([]string, 0, len(summary.Clients))
	for _, c := range summary.Clients {
		gotClients = append(gotClients, c.Client)
	}
	if !slices.Equal(gotClients, wantClients) {
		t.Fatalf("preview client order = %v, want %v", gotClients, wantClients)
	}
	for _, c := range summary.Clients {
		if c.Client != "claude" {
			continue
		}
		if c.Replace != 1 || c.UpToDate != 1 {
			t.Fatalf("claude counts = %+v, want 1 replace and 1 up to date", c)
		}
	}
	if got, want := summary.TotalChanges(), summary.Install+summary.Replace; got != want {
		t.Fatalf("TotalChanges() = %d, want install+replace = %d", got, want)
	}

	// Apply: the same plan lines plus the "Installed ..." trailer.
	var applyBuf bytes.Buffer
	if err := SyncWithOptions("", home, true, opts, &applyBuf); err != nil {
		t.Fatalf("apply: %v", err)
	}
	applySummary := SummarizePlan(applyBuf.String())
	if len(applySummary.Unclassified) != 0 {
		t.Fatalf("apply produced unclassified lines %q\nfull transcript:\n%s", applySummary.Unclassified, applyBuf.String())
	}
	if applySummary.Install <= 0 || applySummary.Replace != 1 {
		t.Fatalf("apply counts = install %d, replace %d, want install > 0 and replace 1\n%s", applySummary.Install, applySummary.Replace, applyBuf.String())
	}

	// Restore preview for the backup the apply created: after changing the
	// home, it must report replacements or restores plus removals, and every
	// line must still be recognized.
	backups, err := ListBackups("", home)
	if err != nil {
		t.Fatalf("list backups: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("backups = %d, want 1", len(backups))
	}
	if err := os.RemoveAll(filepath.Join(home, ".agents", "skills", "agentic-sdd-router")); err != nil {
		t.Fatal(err)
	}
	var restoreBuf bytes.Buffer
	if err := Restore("", home, backups[0].ID, false, &restoreBuf); err != nil {
		t.Fatalf("restore preview: %v", err)
	}
	restoreSummary := SummarizePlan(restoreBuf.String())
	if len(restoreSummary.Unclassified) != 0 {
		t.Fatalf("restore produced unclassified lines %q\nfull transcript:\n%s", restoreSummary.Unclassified, restoreBuf.String())
	}
	if (restoreSummary.Restore == 0 && restoreSummary.Replace == 0) || restoreSummary.Remove == 0 {
		t.Fatalf("restore counts = restore %d, replace %d, remove %d, want a replacement/restore and a removal\n%s",
			restoreSummary.Restore, restoreSummary.Replace, restoreSummary.Remove, restoreBuf.String())
	}

	// An unknown line must be reported, not silently dropped or counted.
	unknown := "bogus verb agents/foo"
	polluted := SummarizePlan(preview.String() + unknown + "\n")
	if !slices.Contains(polluted.Unclassified, unknown) {
		t.Fatalf("unclassified lines = %q, want %q", polluted.Unclassified, unknown)
	}
	if polluted.Install != summary.Install || polluted.Replace != summary.Replace || polluted.UpToDate != summary.UpToDate {
		t.Fatalf("unknown line changed counts: got install %d replace %d up-to-date %d, want %d/%d/%d",
			polluted.Install, polluted.Replace, polluted.UpToDate, summary.Install, summary.Replace, summary.UpToDate)
	}
}
