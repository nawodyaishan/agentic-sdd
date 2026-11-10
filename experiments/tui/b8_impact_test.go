package tui

// B8/T22 regression tests for the derived impact projection. The projection is
// presentation-only: it must classify every planned change exactly once, keep
// unknown actions explicit and never modify the authoritative plan or token.

import (
	"fmt"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
)

func b8MixedPlan() []skillsync.ResultChange {
	planned := []skillsync.ResultChange{
		{Client: "claude", ID: "router", Kind: skillsync.InventoryKindSkill, Path: "/tmp/b8-home/.claude/skills/router", Action: skillsync.ChangeInstall},
		{Client: "claude", ID: "planner", Kind: skillsync.InventoryKindSkill, Path: "/tmp/b8-home/.claude/skills/planner", Action: skillsync.ChangeReplace},
		{Client: "claude", ID: "legacy", Kind: skillsync.InventoryKindSkill, Path: "/tmp/b8-home/.claude/skills/legacy", Action: skillsync.ChangeRemove},
		{Client: "agents", ID: "router", Kind: skillsync.InventoryKindSkill, Path: "/tmp/b8-home/.agents/skills/router", Action: skillsync.ChangeUpToDate},
		{Client: "agents", ID: "brand-new", Kind: skillsync.InventoryKindSkill, Path: "/tmp/b8-home/.agents/skills/brand-new", Action: skillsync.ChangeInstall},
		{Client: "opencode-agents", ID: "sdd-guide", Kind: skillsync.InventoryKindProfile, Path: "/tmp/b8-home/.config/opencode/agents/sdd-guide.md", Action: skillsync.ChangeReplace},
		{Client: "codex", ID: "future", Kind: skillsync.InventoryKindSkill, Path: "/tmp/b8-home/.codex/skills/future", Action: skillsync.ChangeAction("reticulate")},
	}
	return planned
}

func TestB8ImpactProjectionReconcilesEveryItemOnce(t *testing.T) {
	planned := b8MixedPlan()
	original := append([]skillsync.ResultChange(nil), planned...)

	got := projectImpact(planned)

	if got.total != len(planned) {
		t.Fatalf("total = %d, want %d", got.total, len(planned))
	}
	want := map[impactClass]int{
		impactNew:       2,
		impactReplace:   2,
		impactRestore:   0,
		impactRemove:    1,
		impactUnchanged: 1,
		impactUnknown:   1,
	}
	var sum int
	for class, count := range want {
		if c := got.count(class); c != count {
			t.Fatalf("count(%v) = %d, want %d", class, c, count)
		}
		sum += count
	}
	if sum != got.total {
		t.Fatalf("action counts sum to %d, want total %d", sum, got.total)
	}
	if got.count(impactUnknown) != 1 {
		t.Fatal("unknown action was not kept explicit")
	}
	if got.count(impactNew) != 2 {
		t.Fatalf("unknown or replaced action was silently counted as an addition: new = %d", got.count(impactNew))
	}
	if !got.hasUnknown() {
		t.Fatal("hasUnknown() = false for an unrecognized action")
	}
	if got.mutable() != 5 {
		t.Fatalf("mutable() = %d, want 5 (new + replace + restore + remove)", got.mutable())
	}

	// Destination groups partition the same items: no duplicates, no losses.
	grouped := 0
	for _, group := range got.groups {
		grouped += len(group.items)
		var groupSum int
		for _, class := range []impactClass{impactNew, impactReplace, impactRestore, impactRemove, impactUnchanged, impactUnknown} {
			groupSum += group.count(class)
		}
		if groupSum != len(group.items) {
			t.Fatalf("group %q counts %d items but holds %d", group.client, groupSum, len(group.items))
		}
	}
	if grouped != got.total {
		t.Fatalf("grouped items = %d, want %d", grouped, got.total)
	}
	if len(got.groups) != 4 {
		t.Fatalf("groups = %d, want 4 distinct destinations", len(got.groups))
	}

	// The projection is immutable: the authoritative plan is unchanged.
	for i := range original {
		if planned[i] != original[i] {
			t.Fatalf("projection mutated planned[%d]: %+v != %+v", i, planned[i], original[i])
		}
	}
}

func TestB8ImpactGroupsUseTypedDestinationIdentity(t *testing.T) {
	home := "/tmp/b8-用户-home/résumé/.claude/skills/日本語-skill"
	planned := []skillsync.ResultChange{
		{Client: "claude", ID: "日本語-skill", Kind: skillsync.InventoryKindSkill, Path: home, Action: skillsync.ChangeInstall},
	}
	got := projectImpact(planned)
	if len(got.groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(got.groups))
	}
	group := got.groups[0]
	if group.client != "claude" {
		t.Fatalf("group client = %q, want the typed client identity", group.client)
	}
	if group.root != "/tmp/b8-用户-home/résumé/.claude/skills" {
		t.Fatalf("group root = %q, want the exact destination directory", group.root)
	}
	if len(group.items) != 1 || group.items[0].change.Path != home {
		t.Fatal("the original item path was not retained")
	}
}

func TestB8ImpactNoOpClassification(t *testing.T) {
	upToDate := []skillsync.ResultChange{
		{Client: "claude", ID: "a", Path: "/tmp/a", Action: skillsync.ChangeUpToDate},
		{Client: "codex", ID: "b", Path: "/tmp/b", Action: skillsync.ChangeUpToDate},
	}
	if p := projectImpact(upToDate); !p.noOp() {
		t.Fatal("an all-up-to-date plan was not classified as a no-op")
	}
	if p := projectImpact(nil); p.noOp() {
		t.Fatal("an empty plan was classified as a successful no-op")
	}
	unknown := append([]skillsync.ResultChange(nil), upToDate...)
	unknown = append(unknown, skillsync.ResultChange{Client: "claude", ID: "c", Path: "/tmp/c", Action: skillsync.ChangeAction("mystery")})
	if p := projectImpact(unknown); p.noOp() {
		t.Fatal("a plan with an unknown action was classified as a no-op")
	}
}

// TestB8ImpactStress94Entries covers the review's fixture shape: 94 planned
// items, long and Unicode paths, and exact, non-duplicated classification.
func TestB8ImpactStress94Entries(t *testing.T) {
	clients := []string{"agents", "codex", "claude", "agy"}
	var planned []skillsync.ResultChange
	for i := 0; i < 94; i++ {
		client := clients[i%len(clients)]
		action := skillsync.ChangeUpToDate
		switch i % 10 {
		case 0, 1, 2, 3, 4:
			action = skillsync.ChangeInstall
		case 5, 6:
			action = skillsync.ChangeReplace
		case 7:
			action = skillsync.ChangeRemove
		}
		planned = append(planned, skillsync.ResultChange{
			Client: client,
			ID:     fmt.Sprintf("skill-%02d", i),
			Kind:   skillsync.InventoryKindSkill,
			Path:   fmt.Sprintf("/tmp/b8-home/.%s/skills/very-long-directory-name-%02d-日本語", client, i),
			Action: action,
		})
	}
	projection := projectImpact(planned)
	if projection.total != 94 {
		t.Fatalf("total = %d, want 94", projection.total)
	}
	if projection.count(impactNew)+projection.count(impactReplace)+projection.count(impactRemove)+projection.count(impactUnchanged) != 94 {
		t.Fatalf("classified counts do not reconcile: %+v", projection.counts)
	}
	seen := make(map[string]int)
	for _, group := range projection.groups {
		for _, item := range group.items {
			seen[item.change.Path]++
		}
	}
	if len(seen) != 94 {
		t.Fatalf("unique paths = %d, want 94", len(seen))
	}
	for path, n := range seen {
		if n != 1 {
			t.Fatalf("path %q counted %d times", path, n)
		}
	}
	for _, group := range projection.groups {
		if group.root == "" {
			t.Fatalf("group %q lost its exact destination root", group.client)
		}
	}
}

func TestB8ImpactSummaryUsesExactCounts(t *testing.T) {
	line := impactSummary(projectImpact(b8MixedPlan()))
	for _, want := range []string{"7 planned", "2 new", "2 replace (backup first)", "1 remove", "1 unchanged", "1 unknown"} {
		if !strings.Contains(line, want) {
			t.Fatalf("summary %q missing %q", line, want)
		}
	}
}
