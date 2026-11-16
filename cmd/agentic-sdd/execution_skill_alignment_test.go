package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestExecutionContractSkillAlignment is G14: a structural alignment check that
// the orchestration skill and its references reflect the verified execution CLI
// contract. It reads the named ground-truth files and asserts presence/absence
// of canonical tokens only; it performs no writes and no model/MCP calls.
func TestExecutionContractSkillAlignment(t *testing.T) {
	readFile := func(rel string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		return string(b)
	}

	skill := readFile("opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md")
	dispatch := readFile("opencode-pack/skills/agentic-sdd-orchestrate/references/dispatch-contract.md")
	failure := readFile("opencode-pack/skills/agentic-sdd-orchestrate/references/failure-and-reconciliation.md")
	context := readFile("opencode-pack/skills/agentic-sdd-orchestrate/references/context-and-cost.md")
	routing := readFile("opencode-pack/routing.md")
	docs := readFile("docs/execution-runs.md")

	hasToken := func(text, token string) bool {
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(token) + `\b`)
		return re.MatchString(text)
	}
	lineContaining := func(text, needle string) string {
		for _, line := range strings.Split(text, "\n") {
			if strings.Contains(line, needle) {
				return line
			}
		}
		return ""
	}

	// Presence: all seven canonical command families in the skill table.
	for _, cmd := range []string{"run init", "run status", "run check", "run record", "run clean", "task brief", "review package"} {
		if !strings.Contains(skill, cmd) {
			t.Errorf("SKILL.md missing command family %q", cmd)
		}
	}

	// Presence: run record row keeps the claim distinction.
	recordRow := lineContaining(skill, "| `run record` |")
	if recordRow == "" {
		t.Fatal("SKILL.md missing the `run record` table row")
	}
	if !strings.Contains(recordRow, "claim") {
		t.Error("`run record` row does not say `claim`")
	}

	// Presence: required denial wording.
	if !strings.Contains(skill, "not a sandbox") {
		t.Error("SKILL.md missing required denial `not a sandbox`")
	}

	// Presence: exit tokens and status semantics in the exit bullet, not headings.
	exitBullet := lineContaining(skill, "`run check` requires `--`")
	if exitBullet == "" {
		t.Fatal("SKILL.md missing the run check exit bullet")
	}
	for _, tok := range []string{"0", "1", "2", "3", "124"} {
		if !hasToken(exitBullet, tok) {
			t.Errorf("exit bullet missing exit token %q", tok)
		}
	}
	if !strings.Contains(exitBullet, "run status") || !strings.Contains(exitBullet, "stale") || !strings.Contains(exitBullet, "blocked") {
		t.Error("exit bullet does not bind run status to stale/blocked and exit 0")
	}

	// Presence: 017 recovery rules must be in failure-and-reconciliation.md,
	// not merely in docs/execution-runs.md.
	for _, reason := range []string{"missing_artifact", "brief_digest_mismatch", "unregistered_brief", "brief_inspection_failed"} {
		if !strings.Contains(failure, reason) {
			t.Errorf("failure-and-reconciliation.md missing brief-drift reason %q", reason)
		}
	}
	if !strings.Contains(failure, "reconcile_source_change") {
		t.Error("failure-and-reconciliation.md missing `reconcile_source_change`")
	}
	if !strings.Contains(failure, "--force") {
		t.Error("failure-and-reconciliation.md missing `--force`")
	}
	if !strings.Contains(failure, "Foreign checkpoint") {
		t.Error("failure-and-reconciliation.md missing the foreign-checkpoint rule")
	}
	if !strings.Contains(failure, "Record, clean and brief refusals") || !strings.Contains(failure, "nothing is deleted") {
		t.Error("failure-and-reconciliation.md missing the record/clean/brief refusal rule")
	}

	// Presence: dispatch handoff fields remain documented.
	for _, field := range []string{"run ID", "task ID", "brief path and digest", "check log paths", "review package manifest"} {
		if !strings.Contains(dispatch, field) {
			t.Errorf("dispatch-contract.md missing handoff field %q", field)
		}
	}

	// Presence: docs/execution-runs.md referenced from SKILL.md and failure-and-reconciliation.md.
	if !strings.Contains(skill, "docs/execution-runs.md") {
		t.Error("SKILL.md does not reference `docs/execution-runs.md`")
	}
	if !strings.Contains(failure, "docs/execution-runs.md") {
		t.Error("failure-and-reconciliation.md does not reference `docs/execution-runs.md`")
	}

	// Absence: active legacy execution command examples.
	agentFacing := skill + "\n" + dispatch + "\n" + failure + "\n" + context + "\n" + routing + "\n" + docs
	for _, legacy := range []string{"agentic-sdd run", "agentic-sdd task", "agentic-sdd review"} {
		if strings.Contains(agentFacing, legacy) {
			t.Errorf("active legacy execution command example found: %q", legacy)
		}
	}

	// Absence: affirmative sandbox forms.
	for _, form := range []string{"is a sandbox", "acts as a sandbox"} {
		if strings.Contains(agentFacing, form) {
			t.Errorf("affirmative sandbox form found: %q", form)
		}
	}

	// Absence: affirmative approval/resume phrases.
	for _, phrase := range []string{"run init is permission", "resume the batch", "run status approves"} {
		if strings.Contains(agentFacing, phrase) {
			t.Errorf("affirmative approval/resume phrase found: %q", phrase)
		}
	}
}
