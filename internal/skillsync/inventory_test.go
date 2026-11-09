package skillsync

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Feature 011 / Batch B2 / T09 — read-only selection inventory.
//
// These tests pin the source distributions as an explicit drift contract:
// 18 embedded core skills, one optional orchestration skill, 14 base agent
// profiles and four optional Free-first profile templates.

func TestInventoryContractValues(t *testing.T) {
	statuses := map[InventoryStatus]string{
		InventoryStatusMatchesSource:     "matches source",
		InventoryStatusDiffersFromSource: "differs from source",
		InventoryStatusMissing:           "missing",
		InventoryStatusUnavailable:       "unavailable/refused",
	}
	for status, want := range statuses {
		if string(status) != want {
			t.Fatalf("status %q = %q, want %q", want, status, want)
		}
	}
	kinds := map[string]string{
		InventoryKindSkill:   "skill",
		InventoryKindProfile: "profile",
	}
	for kind, want := range kinds {
		if kind != want {
			t.Fatalf("kind %q != %q", kind, want)
		}
	}
	origins := map[string]string{
		InventoryOriginCore:   "core",
		InventoryOriginPack:   "pack",
		InventoryOriginPreset: "preset",
	}
	for origin, want := range origins {
		if origin != want {
			t.Fatalf("origin %q != %q", origin, want)
		}
	}
}

func TestInventoryEmbeddedCoreOnly(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")

	inv, err := InspectSelection("", home, Options{})
	if err != nil {
		t.Fatalf("InspectSelection core: %v", err)
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		t.Fatal(err)
	}
	if inv.Home != absHome {
		t.Fatalf("Home = %q, want %q", inv.Home, absHome)
	}
	if inv.Repo != "" {
		t.Fatalf("Repo = %q, want empty for embedded source", inv.Repo)
	}
	if want := filepath.Join(absHome, ".agentic-sdd", "backups"); inv.BackupRoot != want {
		t.Fatalf("BackupRoot = %q, want %q", inv.BackupRoot, want)
	}

	wantSkills := inventoryEmbeddedSkillIDs(t)
	if len(wantSkills) != 18 {
		t.Fatalf("embedded core distribution changed: %d skills, want 18", len(wantSkills))
	}
	if !reflect.DeepEqual(inv.SkillIDs, wantSkills) {
		t.Fatalf("SkillIDs = %v, want %v", inv.SkillIDs, wantSkills)
	}
	if len(inv.ProfileIDs) != 0 {
		t.Fatalf("ProfileIDs = %v, want none for core-only inspection", inv.ProfileIDs)
	}

	targets := clientTargets(absHome)
	if len(inv.Entries) != len(wantSkills)*len(targets) {
		t.Fatalf("Entries = %d, want %d", len(inv.Entries), len(wantSkills)*len(targets))
	}
	for _, target := range targets {
		for _, id := range wantSkills {
			entry := inventoryFindEntry(t, inv, id, InventoryKindSkill, target.name)
			if entry.Origin != InventoryOriginCore {
				t.Fatalf("%s/%s Origin = %q, want %q", target.name, id, entry.Origin, InventoryOriginCore)
			}
			if entry.Status != InventoryStatusMissing {
				t.Fatalf("%s/%s Status = %q, want %q", target.name, id, entry.Status, InventoryStatusMissing)
			}
			if want := filepath.Join(target.path, id); entry.Path != want {
				t.Fatalf("%s/%s Path = %q, want %q", target.name, id, entry.Path, want)
			}
		}
	}

	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("inventory created the home directory: %v", err)
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatalf("inventory wrote into the fixture root: %v, %d entries", err, len(entries))
	}

	again, err := InspectSelection("", home, Options{})
	if err != nil {
		t.Fatalf("InspectSelection repeat: %v", err)
	}
	if !reflect.DeepEqual(inv, again) {
		t.Fatalf("inventory is not deterministic:\nfirst=%+v\nsecond=%+v", inv, again)
	}
}

func TestInventoryEmbeddedPackAndPresets(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	coreIDs := inventoryEmbeddedSkillIDs(t)
	baseProfiles := inventoryEmbeddedProfileIDs(t, "agents")
	if len(baseProfiles) != 14 {
		t.Fatalf("embedded base profile distribution changed: %d, want 14", len(baseProfiles))
	}
	wantSkills := append([]string(nil), coreIDs...)
	wantSkills = append(wantSkills, "agentic-sdd-orchestrate")
	sort.Strings(wantSkills)

	inv, err := InspectSelection("", home, Options{WithOpenCode: true})
	if err != nil {
		t.Fatalf("InspectSelection pack: %v", err)
	}
	if len(inv.SkillIDs) != 19 {
		t.Fatalf("SkillIDs = %d, want 19 with the optional pack", len(inv.SkillIDs))
	}
	if !reflect.DeepEqual(inv.SkillIDs, wantSkills) {
		t.Fatalf("SkillIDs = %v, want %v", inv.SkillIDs, wantSkills)
	}
	if !reflect.DeepEqual(inv.ProfileIDs, baseProfiles) {
		t.Fatalf("ProfileIDs = %v, want %v", inv.ProfileIDs, baseProfiles)
	}

	targets := clientTargets(inv.Home)
	if want := len(targets)*len(wantSkills) + len(baseProfiles); len(inv.Entries) != want {
		t.Fatalf("Entries = %d, want %d", len(inv.Entries), want)
	}
	for _, target := range targets {
		entry := inventoryFindEntry(t, inv, "agentic-sdd-orchestrate", InventoryKindSkill, target.name)
		if entry.Origin != InventoryOriginPack {
			t.Fatalf("%s orchestrate Origin = %q, want %q", target.name, entry.Origin, InventoryOriginPack)
		}
		if entry.Status != InventoryStatusMissing {
			t.Fatalf("%s orchestrate Status = %q, want missing", target.name, entry.Status)
		}
	}
	agentTarget := openCodeAgentsTarget(inv.Home)
	for _, id := range baseProfiles {
		entry := inventoryFindEntry(t, inv, id, InventoryKindProfile, agentTarget.name)
		if entry.Origin != InventoryOriginPack {
			t.Fatalf("%s Origin = %q, want %q", id, entry.Origin, InventoryOriginPack)
		}
		if entry.Status != InventoryStatusMissing {
			t.Fatalf("%s Status = %q, want missing", id, entry.Status)
		}
		if want := filepath.Join(agentTarget.path, id+".md"); entry.Path != want {
			t.Fatalf("%s Path = %q, want %q", id, entry.Path, want)
		}
	}

	presetProfiles := inventoryEmbeddedProfileIDs(t, "presets/free-first")
	if len(presetProfiles) != 4 {
		t.Fatalf("embedded preset distribution changed: %d, want 4", len(presetProfiles))
	}
	wantAllProfiles := append(append([]string(nil), baseProfiles...), presetProfiles...)
	sort.Strings(wantAllProfiles)

	freeFirst, err := InspectSelection("", home, Options{WithOpenCode: true, OpenCodeFreeFirst: true})
	if err != nil {
		t.Fatalf("InspectSelection pack+presets: %v", err)
	}
	if !reflect.DeepEqual(freeFirst.ProfileIDs, wantAllProfiles) {
		t.Fatalf("ProfileIDs = %v, want %v", freeFirst.ProfileIDs, wantAllProfiles)
	}
	if want := len(targets)*len(wantSkills) + len(wantAllProfiles); len(freeFirst.Entries) != want {
		t.Fatalf("Entries = %d, want %d", len(freeFirst.Entries), want)
	}
	for _, id := range presetProfiles {
		entry := inventoryFindEntry(t, freeFirst, id, InventoryKindProfile, agentTarget.name)
		if entry.Origin != InventoryOriginPreset {
			t.Fatalf("preset %s Origin = %q, want %q", id, entry.Origin, InventoryOriginPreset)
		}
	}
	for _, id := range baseProfiles {
		entry := inventoryFindEntry(t, freeFirst, id, InventoryKindProfile, agentTarget.name)
		if entry.Origin != InventoryOriginPack {
			t.Fatalf("base %s Origin = %q, want %q", id, entry.Origin, InventoryOriginPack)
		}
	}
}

func TestInventoryCustomSourceWithoutPack(t *testing.T) {
	repo := inventoryWriteCoreRepo(t, map[string]string{"agentic-sdd-solo": "solo body"})
	home := filepath.Join(t.TempDir(), "home")
	absRepo, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		t.Fatal(err)
	}

	inv, err := InspectSelection(repo, home, Options{})
	if err != nil {
		t.Fatalf("InspectSelection core: %v", err)
	}
	if inv.Home != absHome {
		t.Fatalf("Home = %q, want %q", inv.Home, absHome)
	}
	if inv.Repo != absRepo {
		t.Fatalf("Repo = %q, want %q", inv.Repo, absRepo)
	}
	if want := filepath.Join(absRepo, "backups"); inv.BackupRoot != want {
		t.Fatalf("BackupRoot = %q, want %q", inv.BackupRoot, want)
	}
	if !reflect.DeepEqual(inv.SkillIDs, []string{"agentic-sdd-solo"}) {
		t.Fatalf("SkillIDs = %v, want the one custom skill", inv.SkillIDs)
	}
	if len(inv.Entries) != len(clientTargets(absHome)) {
		t.Fatalf("Entries = %d, want one per client", len(inv.Entries))
	}
	for _, entry := range inv.Entries {
		if entry.Kind != InventoryKindSkill || entry.Origin != InventoryOriginCore || entry.Status != InventoryStatusMissing {
			t.Fatalf("unexpected custom entry: %+v", entry)
		}
	}

	partial, err := InspectSelection(repo, home, Options{WithOpenCode: true})
	if !errors.Is(err, ErrPackMissing) {
		t.Fatalf("WithOpenCode error = %v, want ErrPackMissing", err)
	}
	if partial.Home != absHome || partial.Repo != absRepo {
		t.Fatalf("partial context = %+v", partial)
	}
	if !reflect.DeepEqual(partial.SkillIDs, []string{"agentic-sdd-solo"}) {
		t.Fatalf("partial SkillIDs = %v, want core only", partial.SkillIDs)
	}
	if len(partial.ProfileIDs) != 0 {
		t.Fatalf("partial ProfileIDs = %v, want none", partial.ProfileIDs)
	}
	if len(partial.Entries) != len(clientTargets(absHome)) {
		t.Fatalf("partial Entries = %d, want core inventory only", len(partial.Entries))
	}
	for _, entry := range partial.Entries {
		if entry.Origin != InventoryOriginCore || entry.Status != InventoryStatusMissing {
			t.Fatalf("partial entry = %+v", entry)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, "backups")); !os.IsNotExist(err) {
		t.Fatalf("inventory created the backup root: %v", err)
	}
}

func TestInventoryPackSourceErrorsDistinguishMissingFromInvalid(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")

	repoMissing := inventoryWriteCoreRepo(t, map[string]string{"agentic-sdd-solo": "solo"})
	partial, err := InspectSelection(repoMissing, home, Options{WithOpenCode: true})
	if !errors.Is(err, ErrPackMissing) {
		t.Fatalf("absent pack error = %v, want ErrPackMissing", err)
	}
	if len(partial.Entries) != len(clientTargets(home)) {
		t.Fatalf("absent pack returned %d entries, want the core inventory", len(partial.Entries))
	}

	repoFile := inventoryWriteCoreRepo(t, map[string]string{"agentic-sdd-solo": "solo"})
	if err := os.WriteFile(filepath.Join(repoFile, "opencode-pack"), []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectSelection(repoFile, home, Options{WithOpenCode: true}); err == nil || errors.Is(err, ErrPackMissing) {
		t.Fatalf("non-directory pack error = %v, want a distinct non-missing error", err)
	}

	repoNoAgents := inventoryWriteCoreRepo(t, map[string]string{"agentic-sdd-solo": "solo"})
	inventoryWriteOrchestrate(t, repoNoAgents, "orchestrate")
	if _, err := InspectSelection(repoNoAgents, home, Options{WithOpenCode: true}); err == nil || errors.Is(err, ErrPackMissing) {
		t.Fatalf("missing agents source error = %v, want a distinct non-missing error", err)
	}

	repoSymlinkProfile := inventoryWriteCoreRepo(t, map[string]string{"agentic-sdd-solo": "solo"})
	inventoryWriteOrchestrate(t, repoSymlinkProfile, "orchestrate")
	outsideProfile := filepath.Join(t.TempDir(), "outside-profile.md")
	if err := os.WriteFile(outsideProfile, []byte("outside profile"), 0644); err != nil {
		t.Fatal(err)
	}
	agentsSource := filepath.Join(repoSymlinkProfile, "opencode-pack", "agents")
	if err := os.MkdirAll(agentsSource, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideProfile, filepath.Join(agentsSource, "sdd-outside.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectSelection(repoSymlinkProfile, home, Options{WithOpenCode: true}); err == nil || errors.Is(err, ErrPackMissing) {
		t.Fatalf("symlinked pack profile error = %v, want a distinct non-missing error", err)
	}

	repoSymlinkSkill := inventoryWriteCoreRepo(t, map[string]string{"agentic-sdd-solo": "solo"})
	orchSource := filepath.Join(repoSymlinkSkill, "opencode-pack", "skills", "agentic-sdd-orchestrate")
	if err := os.MkdirAll(orchSource, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideProfile, filepath.Join(orchSource, "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectSelection(repoSymlinkSkill, home, Options{WithOpenCode: true}); err == nil || errors.Is(err, ErrPackMissing) {
		t.Fatalf("symlinked pack skill error = %v, want a distinct non-missing error", err)
	}
}

func TestInventoryCoreTargetStatuses(t *testing.T) {
	const body = "solo body"
	repo := inventoryWriteCoreRepo(t, map[string]string{"agentic-sdd-solo": body})
	home := filepath.Join(t.TempDir(), "home")
	targets := clientTargets(home)

	inventoryWriteSkill(t, filepath.Join(targets[0].path, "agentic-sdd-solo"), body)
	inventoryWriteSkill(t, filepath.Join(targets[1].path, "agentic-sdd-solo"), "different body")
	// targets[2] intentionally absent: missing.
	targetSymlink := filepath.Join(targets[3].path, "agentic-sdd-solo")
	if err := os.MkdirAll(filepath.Dir(targetSymlink), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(targets[0].path, "agentic-sdd-solo"), targetSymlink); err != nil {
		t.Fatal(err)
	}
	// An unrelated managed-looking skill must not enter this selection's inventory.
	inventoryWriteSkill(t, filepath.Join(targets[0].path, "agentic-sdd-unrelated"), "unrelated")

	before := inventorySnapshot(t, home)
	inv, err := InspectSelection(repo, home, Options{})
	if err != nil {
		t.Fatalf("InspectSelection: %v", err)
	}
	after := inventorySnapshot(t, home)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("inventory changed target content:\nbefore=%v\nafter=%v", before, after)
	}

	if len(inv.Entries) != len(targets) {
		t.Fatalf("Entries = %d, want only the selected skill per client", len(inv.Entries))
	}
	want := map[string]InventoryStatus{
		"agents": InventoryStatusMatchesSource,
		"codex":  InventoryStatusDiffersFromSource,
		"claude": InventoryStatusMissing,
		"agy":    InventoryStatusUnavailable,
	}
	for client, status := range want {
		entry := inventoryFindEntry(t, inv, "agentic-sdd-solo", InventoryKindSkill, client)
		if entry.Status != status {
			t.Fatalf("%s Status = %q, want %q", client, entry.Status, status)
		}
	}
	if _, ok := inventoryLookupEntry(inv, "agentic-sdd-unrelated", InventoryKindSkill, "agents"); ok {
		t.Fatal("unrelated installed skill was enumerated")
	}
}

func TestInventoryUnsafeTargetParents(t *testing.T) {
	repo := inventoryWriteCoreRepo(t, map[string]string{"agentic-sdd-solo": "solo body"})
	home := filepath.Join(t.TempDir(), "home")
	targets := clientTargets(home)

	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".agents"), []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}
	outsideSkills := filepath.Join(t.TempDir(), "outside-skills")
	inventoryWriteSkill(t, filepath.Join(outsideSkills, "agentic-sdd-solo"), "solo body")
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideSkills, filepath.Join(home, ".codex", "skills")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude", "skills"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".gemini"), []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}

	before := inventorySnapshot(t, home)
	inv, err := InspectSelection(repo, home, Options{})
	if err != nil {
		t.Fatalf("InspectSelection: %v", err)
	}
	after := inventorySnapshot(t, home)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("inventory changed unsafe target fixtures")
	}

	want := map[string]InventoryStatus{
		"agents": InventoryStatusUnavailable,
		"codex":  InventoryStatusUnavailable,
		"claude": InventoryStatusMissing,
		"agy":    InventoryStatusUnavailable,
	}
	for client, status := range want {
		entry := inventoryFindEntry(t, inv, "agentic-sdd-solo", InventoryKindSkill, client)
		if entry.Status != status {
			t.Fatalf("%s Status = %q, want %q", client, entry.Status, status)
		}
	}
	for _, target := range targets {
		if target.name == "claude" {
			continue
		}
		if entry := inventoryFindEntry(t, inv, "agentic-sdd-solo", InventoryKindSkill, target.name); entry.Status != InventoryStatusUnavailable {
			t.Fatalf("%s unsafe parent was treated as %q", target.name, entry.Status)
		}
	}
}

func TestInventoryProfileStatuses(t *testing.T) {
	repo := inventoryWriteCoreRepo(t, map[string]string{"agentic-sdd-solo": "solo"})
	inventoryWriteOrchestrate(t, repo, "orchestrate")
	inventoryWritePackProfile(t, repo, "agents", "sdd-alpha", "alpha body")
	inventoryWritePackProfile(t, repo, "agents", "sdd-beta", "beta body")
	inventoryWritePackProfile(t, repo, "agents", "sdd-gamma", "gamma body")
	inventoryWritePackProfile(t, repo, "agents", "sdd-delta", "delta body")
	inventoryWritePackProfile(t, repo, "presets/free-first", "sdd-preset", "preset body")

	home := filepath.Join(t.TempDir(), "home")
	agentTarget := openCodeAgentsTarget(home)
	if err := os.MkdirAll(agentTarget.path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentTarget.path, "sdd-alpha.md"), []byte("alpha body"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentTarget.path, "sdd-beta.md"), []byte("different body"), 0644); err != nil {
		t.Fatal(err)
	}
	// sdd-gamma intentionally absent: missing.
	outsideProfile := filepath.Join(t.TempDir(), "outside-profile.md")
	if err := os.WriteFile(outsideProfile, []byte("delta body"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideProfile, filepath.Join(agentTarget.path, "sdd-delta.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentTarget.path, "sdd-preset.md"), []byte("preset body"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentTarget.path, "vision.md"), []byte("unrelated user profile"), 0644); err != nil {
		t.Fatal(err)
	}

	// A symlinked file inside a managed skill tree is refused, not followed.
	targetSkills := clientTargets(home)
	agentsSkill := filepath.Join(targetSkills[0].path, "agentic-sdd-orchestrate")
	if err := os.MkdirAll(agentsSkill, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideProfile, filepath.Join(agentsSkill, "SKILL.md")); err != nil {
		t.Fatal(err)
	}

	before := inventorySnapshot(t, home)
	inv, err := InspectSelection(repo, home, Options{WithOpenCode: true, OpenCodeFreeFirst: true})
	if err != nil {
		t.Fatalf("InspectSelection: %v", err)
	}
	after := inventorySnapshot(t, home)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("inventory changed profile fixtures")
	}

	wantProfiles := []string{"sdd-alpha", "sdd-beta", "sdd-delta", "sdd-gamma", "sdd-preset"}
	if !reflect.DeepEqual(inv.ProfileIDs, wantProfiles) {
		t.Fatalf("ProfileIDs = %v, want %v", inv.ProfileIDs, wantProfiles)
	}
	wantStatus := map[string]InventoryStatus{
		"sdd-alpha":  InventoryStatusMatchesSource,
		"sdd-beta":   InventoryStatusDiffersFromSource,
		"sdd-gamma":  InventoryStatusMissing,
		"sdd-delta":  InventoryStatusUnavailable,
		"sdd-preset": InventoryStatusMatchesSource,
	}
	for id, status := range wantStatus {
		entry := inventoryFindEntry(t, inv, id, InventoryKindProfile, agentTarget.name)
		if entry.Status != status {
			t.Fatalf("%s Status = %q, want %q", id, entry.Status, status)
		}
		if want := filepath.Join(agentTarget.path, id+".md"); entry.Path != want {
			t.Fatalf("%s Path = %q, want %q", id, entry.Path, want)
		}
	}
	if got := inventoryFindEntry(t, inv, "sdd-preset", InventoryKindProfile, agentTarget.name).Origin; got != InventoryOriginPreset {
		t.Fatalf("sdd-preset Origin = %q, want %q", got, InventoryOriginPreset)
	}
	if got := inventoryFindEntry(t, inv, "sdd-alpha", InventoryKindProfile, agentTarget.name).Origin; got != InventoryOriginPack {
		t.Fatalf("sdd-alpha Origin = %q, want %q", got, InventoryOriginPack)
	}
	if _, ok := inventoryLookupEntry(inv, "vision", InventoryKindProfile, agentTarget.name); ok {
		t.Fatal("unrelated user profile was enumerated")
	}
	entry := inventoryFindEntry(t, inv, "agentic-sdd-orchestrate", InventoryKindSkill, targetSkills[0].name)
	if entry.Status != InventoryStatusUnavailable {
		t.Fatalf("symlinked skill content Status = %q, want %q", entry.Status, InventoryStatusUnavailable)
	}

	// A symlinked profile parent refuses every profile instead of following it.
	home2 := filepath.Join(t.TempDir(), "home2")
	outsideAgents := filepath.Join(t.TempDir(), "outside-agents")
	if err := os.MkdirAll(outsideAgents, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home2, ".config", "opencode"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideAgents, filepath.Join(home2, ".config", "opencode", "agents")); err != nil {
		t.Fatal(err)
	}
	inv2, err := InspectSelection(repo, home2, Options{WithOpenCode: true})
	if err != nil {
		t.Fatalf("InspectSelection symlinked agents parent: %v", err)
	}
	for _, id := range inv2.ProfileIDs {
		entry := inventoryFindEntry(t, inv2, id, InventoryKindProfile, agentTarget.name)
		if entry.Status != InventoryStatusUnavailable {
			t.Fatalf("%s with unsafe parent Status = %q, want %q", id, entry.Status, InventoryStatusUnavailable)
		}
	}
}

func TestInventoryUnreadablePackSourceIsNotMissing(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")

	repoProfile := inventoryWriteCoreRepo(t, map[string]string{"agentic-sdd-solo": "solo"})
	inventoryWriteOrchestrate(t, repoProfile, "orchestrate")
	inventoryWritePackProfile(t, repoProfile, "agents", "sdd-alpha", "alpha body")
	profilePath := filepath.Join(repoProfile, "opencode-pack", "agents", "sdd-alpha.md")
	if err := os.Chmod(profilePath, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(profilePath, 0644) })
	if probe, err := os.Open(profilePath); err == nil {
		_ = probe.Close()
		t.Skip("file permissions are not enforced in this environment")
	}
	if partial, err := InspectSelection(repoProfile, home, Options{WithOpenCode: true}); err == nil || errors.Is(err, ErrPackMissing) {
		t.Fatalf("unreadable profile source error = %v, want a distinct non-missing error (partial=%+v)", err, partial)
	}

	repoSkill := inventoryWriteCoreRepo(t, map[string]string{"agentic-sdd-solo": "solo"})
	inventoryWriteOrchestrate(t, repoSkill, "orchestrate")
	inventoryWritePackProfile(t, repoSkill, "agents", "sdd-alpha", "alpha body")
	skillPath := filepath.Join(repoSkill, "opencode-pack", "skills", "agentic-sdd-orchestrate", "SKILL.md")
	if err := os.Chmod(skillPath, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(skillPath, 0644) })
	if _, err := InspectSelection(repoSkill, home, Options{WithOpenCode: true}); err == nil || errors.Is(err, ErrPackMissing) {
		t.Fatalf("unreadable skill source error = %v, want a distinct non-missing error", err)
	}
}

func TestInventoryRejectsFreeFirstWithoutOpenCode(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	_, err := InspectSelection("", home, Options{OpenCodeFreeFirst: true})
	if err == nil || !strings.Contains(err.Error(), "--opencode-free-first requires --with-opencode") {
		t.Fatalf("error = %v, want the existing option constraint", err)
	}
	if _, statErr := os.Stat(home); !os.IsNotExist(statErr) {
		t.Fatalf("invalid options created the home directory: %v", statErr)
	}
}

func inventoryEmbeddedSkillIDs(t *testing.T) []string {
	t.Helper()
	source, err := sourceFS("")
	if err != nil {
		t.Fatal(err)
	}
	ids, err := discoverSkills(source)
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

func inventoryEmbeddedProfileIDs(t *testing.T, dir string) []string {
	t.Helper()
	pack, err := openCodePackFS("")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := fs.ReadDir(pack, dir)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(entry.Name(), ".md"))
	}
	sort.Strings(ids)
	return ids
}

func inventoryWriteCoreRepo(t *testing.T, skills map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	for id, body := range skills {
		inventoryWriteSkill(t, filepath.Join(repo, "skills-files", id), body)
	}
	return repo
}

func inventoryWriteSkill(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func inventoryWriteOrchestrate(t *testing.T, repo, body string) {
	t.Helper()
	inventoryWriteSkill(t, filepath.Join(repo, "opencode-pack", "skills", "agentic-sdd-orchestrate"), body)
}

func inventoryWritePackProfile(t *testing.T, repo, dir, id, body string) {
	t.Helper()
	target := filepath.Join(repo, "opencode-pack", filepath.FromSlash(dir))
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, id+".md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func inventoryFindEntry(t *testing.T, inv SelectionInventory, id, kind, client string) InventoryEntry {
	t.Helper()
	entry, ok := inventoryLookupEntry(inv, id, kind, client)
	if !ok {
		t.Fatalf("missing inventory entry id=%q kind=%q client=%q in %v", id, kind, client, inv.Entries)
	}
	return entry
}

func inventoryLookupEntry(inv SelectionInventory, id, kind, client string) (InventoryEntry, bool) {
	for _, entry := range inv.Entries {
		if entry.ID == id && entry.Kind == kind && entry.Client == client {
			return entry, true
		}
	}
	return InventoryEntry{}, false
}

func inventorySnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			snapshot[rel] = "symlink:" + target
		case info.IsDir():
			snapshot[rel] = "dir:" + info.Mode().String()
		case info.Mode().IsRegular():
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			snapshot[rel] = "file:" + string(data)
		default:
			snapshot[rel] = "special:" + info.Mode().String()
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
