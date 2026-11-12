package appops

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentic-sdd/internal/fleet"
	"agentic-sdd/internal/skillsync"
)

// fleetTestConfig is one valid personal fleet document used by most fixtures.
// Every fixture is isolated under t.TempDir; no test reads or writes a real
// home, Git checkout, provider credential or network resource.
const fleetTestConfig = `{
  "schemaVersion": 1,
  "instances": {
    "sdd-implement-1": {
      "source": "preset:sdd-implement-1",
      "model": "9router/ag/gemini-3.8-flash-high"
    }
  }
}
`

const fleetTestPackAgent = `---
description: fleet appops test profile
mode: subagent
model: opencode/test-base
---
Body content is never modified by the fleet renderer.
`

type fleetFixture struct {
	root string
	home string
	cfg  string
}

func newFleetFixture(t *testing.T, config string) fleetFixture {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(root, "fleet.json")
	if err := os.WriteFile(cfg, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	return fleetFixture{root: root, home: home, cfg: cfg}
}

func (f fleetFixture) targetRoot() string {
	return filepath.Join(f.home, ".config", "opencode", "agents")
}

func (f fleetFixture) target(id string) string {
	return filepath.Join(f.targetRoot(), id+".md")
}

func writeFleetTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func digestString(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// fleetApplyForTest runs the shared preview/apply path exactly as a caller
// would. It exists to build fixture homes, not to bypass the public seam.
func fleetApplyForTest(t *testing.T, configPath, repo, home string) fleet.ApplyResult {
	t.Helper()
	previewReq, err := NewFleetPreviewRequest(configPath, repo, home)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := FleetPreview(previewReq)
	if err != nil {
		t.Fatalf("fleet preview: %v", err)
	}
	applyReq, err := NewFleetApplyRequest(preview.Plan, home)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ApplyFleet(applyReq, preview.PreviewToken, io.Discard)
	if err != nil {
		t.Fatalf("fleet apply: %v", err)
	}
	if result.FleetApply == nil {
		t.Fatal("fleet apply result has no FleetApply detail")
	}
	return *result.FleetApply
}

func TestFleetStatus(t *testing.T) {
	t.Run("zero one eighteen and one hundred instances", func(t *testing.T) {
		cases := []struct {
			name string
			ids  []string
		}{
			{"zero", nil},
			{"one", []string{"sdd-implement-1"}},
			{"eighteen", fleetInstanceIDs(18)},
			{"one hundred", fleetInstanceIDs(100)},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				f := newFleetFixture(t, fleetConfigForIDs(tc.ids))
				req, err := NewFleetStatusRequest(f.cfg, "", f.home)
				if err != nil {
					t.Fatal(err)
				}
				res, err := FleetStatus(req)
				if err != nil {
					t.Fatalf("FleetStatus: %v", err)
				}
				if res.Status == nil {
					t.Fatal("nil status")
				}
				if got := res.Status.Counts[fleet.StatusCountInstances]; got != len(tc.ids) {
					t.Fatalf("instance count = %d, want %d", got, len(tc.ids))
				}
				if len(res.Status.Instances) != len(tc.ids) {
					t.Fatalf("instances = %d, want %d", len(res.Status.Instances), len(tc.ids))
				}
				if res.Status.ConfigDigest == "" {
					t.Fatal("empty config digest")
				}
				for _, inst := range res.Status.Instances {
					if inst.Installed != fleet.InstalledAbsent {
						t.Fatalf("%s installed state = %q, want absent", inst.InstanceID, inst.Installed)
					}
					if inst.ConfiguredModel != "9router/ag/gemini-3.8-flash-high" {
						t.Fatalf("%s configured model = %q", inst.InstanceID, inst.ConfiguredModel)
					}
				}
				if _, err := os.Lstat(f.targetRoot()); !os.IsNotExist(err) {
					t.Fatalf("status created or touched the target root: %v", err)
				}
			})
		}
	})

	t.Run("validation", func(t *testing.T) {
		f := newFleetFixture(t, fleetTestConfig)
		if _, err := NewFleetStatusRequest("", "", f.home); err == nil {
			t.Fatal("empty config path accepted")
		}
		if _, err := NewFleetStatusRequest(f.cfg, "", ""); err == nil {
			t.Fatal("empty home accepted")
		}
		req, err := NewFleetStatusRequest(f.cfg, "", f.home)
		if err != nil {
			t.Fatal(err)
		}
		if !filepath.IsAbs(req.ConfigPath()) {
			t.Fatalf("config path %q is not absolute", req.ConfigPath())
		}
		res, err := FleetStatus(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Status.Instances = nil
		again, err := FleetStatus(req)
		if err != nil {
			t.Fatal(err)
		}
		if len(again.Status.Instances) != 1 {
			t.Fatalf("independent status call returned %d instances", len(again.Status.Instances))
		}
	})

	t.Run("invalid config is the typed invalid config error", func(t *testing.T) {
		f := newFleetFixture(t, `{"schemaVersion":1,"instances":{"a":{"source":"bogus"}}}`)
		req, err := NewFleetStatusRequest(f.cfg, "", f.home)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := FleetStatus(req); !errors.Is(err, fleet.ErrInvalidConfig) {
			t.Fatalf("error = %v, want ErrInvalidConfig", err)
		}
	})
}

func TestFleetSourcePreview(t *testing.T) {
	f := newFleetFixture(t, fleetTestConfig)
	original := mustReadFile(t, f.cfg)
	changed := strings.Replace(fleetTestConfig, "gemini-3.8-flash-high", "gemini-3.8-flash-medium", 1)
	req, err := NewFleetSourcePreviewRequest(f.cfg, "", f.home, []byte(changed))
	if err != nil {
		t.Fatal(err)
	}

	res, err := FleetSourcePreview(req)
	if err != nil {
		t.Fatalf("FleetSourcePreview: %v", err)
	}
	if res.Preview.Op != fleet.FileOpUpdate {
		t.Fatalf("op = %q, want update", res.Preview.Op)
	}
	if res.BaseDigest != digestString(original) {
		t.Fatalf("base digest = %q, want sha256 of current bytes", res.BaseDigest)
	}
	if res.Preview.NewDigest != digestString(res.Normalized) {
		t.Fatalf("new digest does not match normalized bytes")
	}
	if !strings.Contains(string(res.Normalized), "gemini-3.8-flash-medium") {
		t.Fatalf("normalized bytes missing staged model: %s", res.Normalized)
	}
	if !strings.HasSuffix(string(res.Normalized), "\n") {
		t.Fatal("normalized bytes do not end with exactly one trailing newline")
	}
	if got := mustReadFile(t, f.cfg); string(got) != string(original) {
		t.Fatal("source preview modified the personal source")
	}

	// The draft is copied at construction and never retained between calls.
	draft := req.DraftBytes()
	draft[0] = 'X'
	if got := req.DraftBytes(); string(got) != changed {
		t.Fatal("mutating DraftBytes changed the request")
	}
	second := strings.Replace(fleetTestConfig, "gemini-3.8-flash-high", "opencode/test-second", 1)
	secondReq, err := NewFleetSourcePreviewRequest(f.cfg, "", f.home, []byte(second))
	if err != nil {
		t.Fatal(err)
	}
	secondRes, err := FleetSourcePreview(secondReq)
	if err != nil {
		t.Fatal(err)
	}
	if secondRes.BaseDigest != res.BaseDigest {
		t.Fatalf("base digest changed between independent reads: %q vs %q", secondRes.BaseDigest, res.BaseDigest)
	}
	if string(secondRes.Normalized) == string(res.Normalized) {
		t.Fatal("independent preview calls returned the same normalized draft")
	}

	t.Run("invalid draft", func(t *testing.T) {
		bad, err := NewFleetSourcePreviewRequest(f.cfg, "", f.home, []byte(`{"schemaVersion": 1,`))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := FleetSourcePreview(bad); !errors.Is(err, fleet.ErrInvalidConfig) {
			t.Fatalf("error = %v, want ErrInvalidConfig", err)
		}
		if got := mustReadFile(t, f.cfg); string(got) != string(original) {
			t.Fatal("invalid draft modified the personal source")
		}
	})

	t.Run("external edit changes the base digest", func(t *testing.T) {
		externallyEdited := strings.Replace(fleetTestConfig, "gemini-3.8-flash-high", "opencode/external", 1)
		writeFleetTestFile(t, f.cfg, externallyEdited)
		again, err := FleetSourcePreview(req)
		if err != nil {
			t.Fatal(err)
		}
		if again.BaseDigest == res.BaseDigest {
			t.Fatal("base digest did not change after an external edit")
		}
		writeFleetTestFile(t, f.cfg, fleetTestConfig)
	})

	t.Run("non-regular source refused", func(t *testing.T) {
		dirCfg := filepath.Join(f.root, "config-dir")
		if err := os.MkdirAll(dirCfg, 0700); err != nil {
			t.Fatal(err)
		}
		req, err := NewFleetSourcePreviewRequest(dirCfg, "", f.home, []byte(fleetTestConfig))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := FleetSourcePreview(req); !errors.Is(err, fleet.ErrUnsafeTarget) {
			t.Fatalf("error = %v, want ErrUnsafeTarget", err)
		}
	})
}

func TestFleetPreview(t *testing.T) {
	t.Run("create update remove noop and conflict", func(t *testing.T) {
		f := newFleetFixture(t, `{"schemaVersion":1,"instances":{
  "alpha":{"source":"preset:sdd-implement-1","model":"9router/ag/gemini-3.8-flash-high"},
  "beta":{"source":"preset:sdd-implement-1","model":"9router/ag/gemini-3.8-flash-high"},
  "gamma":{"source":"preset:sdd-implement-1","model":"9router/ag/gemini-3.8-flash-high"}
}}`)
		fleetApplyForTest(t, f.cfg, "", f.home)

		// An unmanaged file at a new instance's target is a conflict, and a
		// disabled managed instance is a removal.
		writeFleetTestFile(t, f.target("echo"), "unmanaged content\n")
		backupRoot := filepath.Join(f.home, ".agentic-sdd", "backups")
		backupsBefore, err := os.ReadDir(backupRoot)
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		next := `{"schemaVersion":1,"instances":{
  "alpha":{"source":"preset:sdd-implement-1","model":"9router/ag/gemini-3.8-flash-high","enabled":false},
  "beta":{"source":"preset:sdd-implement-1","model":"9router/ag/gemini-3.8-flash-medium"},
  "gamma":{"source":"preset:sdd-implement-1","model":"9router/ag/gemini-3.8-flash-high"},
  "delta":{"source":"preset:sdd-implement-1","model":"9router/ag/gemini-3.8-flash-high"},
  "echo":{"source":"preset:sdd-implement-1","model":"9router/ag/gemini-3.8-flash-high"}
}}`
		writeFleetTestFile(t, f.cfg, next)

		req, err := NewFleetPreviewRequest(f.cfg, "", f.home)
		if err != nil {
			t.Fatal(err)
		}
		res, err := FleetPreview(req)
		if err != nil {
			t.Fatalf("FleetPreview: %v", err)
		}
		if res.Plan == nil || res.Plan.PlanID == "" {
			t.Fatal("preview returned no sealed plan")
		}
		if !res.Plan.VerifyPlanID() {
			t.Fatal("plan ID does not verify")
		}
		if res.PreviewToken == "" {
			t.Fatal("empty preview token")
		}
		ops := map[string]string{}
		for _, op := range res.Plan.ProposedOps {
			if op.Adopt {
				t.Fatalf("plan op for %s unexpectedly adopts", op.InstanceID)
			}
			ops[op.InstanceID] = op.Op
		}
		wantOps := map[string]string{
			"alpha": fleet.FileOpRemove,
			"beta":  fleet.FileOpUpdate,
			"gamma": fleet.FileOpNoOp,
			"delta": fleet.FileOpCreate,
			"echo":  fleet.FileOpConflict,
		}
		for id, want := range wantOps {
			if got := ops[id]; got != want {
				t.Fatalf("%s op = %q, want %q", id, got, want)
			}
		}
		if len(res.Files) != len(res.Plan.ProposedOps) {
			t.Fatalf("files = %d, ops = %d", len(res.Files), len(res.Plan.ProposedOps))
		}
		if res.Files[0].Op != fleet.FileOpRemove || res.Files[1].Op != fleet.FileOpConflict {
			t.Fatalf("file order = %q,%q; removals and conflicts must come first", res.Files[0].Op, res.Files[1].Op)
		}
		for _, file := range res.Files {
			switch file.Op {
			case fleet.FileOpCreate:
				if len(file.NewBytes) == 0 || file.OldBytes != nil {
					t.Fatalf("create preview for %s has wrong old/new bytes", file.Path)
				}
			case fleet.FileOpUpdate:
				if len(file.OldBytes) == 0 || len(file.NewBytes) == 0 {
					t.Fatalf("update preview for %s is incomplete", file.Path)
				}
			case fleet.FileOpRemove:
				if !file.Removed || len(file.NewBytes) != 0 {
					t.Fatalf("remove preview for %s is not a removal", file.Path)
				}
			case fleet.FileOpConflict:
				if !file.Conflict || len(file.OldBytes) == 0 {
					t.Fatalf("conflict preview for %s is not a conflict", file.Path)
				}
			case fleet.FileOpNoOp:
			default:
				t.Fatalf("unexpected preview op %q", file.Op)
			}
			if file.Op != fleet.FileOpCreate && file.Op != fleet.FileOpRemove && file.OldDigest != digestString(file.OldBytes) {
				t.Fatalf("old digest for %s does not match old bytes", file.Path)
			}
		}
		if res.ConfigDigest == "" || res.SourceRev == "" || res.RendererVersion == "" {
			t.Fatal("preview is missing plan identity fields")
		}

		// A preview is read-only: the unmanaged file is untouched and no plan
		// or backup was written.
		if got := string(mustReadFile(t, f.target("echo"))); got != "unmanaged content\n" {
			t.Fatalf("preview changed an unmanaged target: %q", got)
		}
		backupsAfter, err := os.ReadDir(backupRoot)
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if len(backupsAfter) != len(backupsBefore) {
			t.Fatalf("preview created a backup: %d -> %d entries", len(backupsBefore), len(backupsAfter))
		}
	})

	t.Run("plan copy cannot mutate the request", func(t *testing.T) {
		f := newFleetFixture(t, fleetTestConfig)
		previewReq, err := NewFleetPreviewRequest(f.cfg, "", f.home)
		if err != nil {
			t.Fatal(err)
		}
		preview, err := FleetPreview(previewReq)
		if err != nil {
			t.Fatal(err)
		}
		applyReq, err := NewFleetApplyRequest(preview.Plan, f.home)
		if err != nil {
			t.Fatal(err)
		}
		copyPlan := applyReq.Plan()
		copyPlan.ProposedOps[0].Op = fleet.FileOpRemove
		if got := applyReq.Plan().ProposedOps[0].Op; got != preview.Plan.ProposedOps[0].Op {
			t.Fatalf("mutating a returned plan changed the request: %q", got)
		}
	})
}

func TestFleetApply(t *testing.T) {
	t.Run("token match applies with counted backup and no lock leak", func(t *testing.T) {
		f := newFleetFixture(t, fleetTestConfig)
		previewReq, err := NewFleetPreviewRequest(f.cfg, "", f.home)
		if err != nil {
			t.Fatal(err)
		}
		preview, err := FleetPreview(previewReq)
		if err != nil {
			t.Fatal(err)
		}
		applyReq, err := NewFleetApplyRequest(preview.Plan, f.home)
		if err != nil {
			t.Fatal(err)
		}
		result, err := ApplyFleet(applyReq, preview.PreviewToken, io.Discard)
		if err != nil {
			t.Fatalf("ApplyFleet: %v", err)
		}
		if result.Outcome != skillsync.OutcomeApplied {
			t.Fatalf("outcome = %q, want applied", result.Outcome)
		}
		if result.FleetApply == nil {
			t.Fatal("missing FleetApply detail")
		}
		if result.FleetApply.Created != 1 || result.FleetApply.Updated != 0 || result.FleetApply.Removed != 0 {
			t.Fatalf("counts = created %d updated %d removed %d, want 1/0/0",
				result.FleetApply.Created, result.FleetApply.Updated, result.FleetApply.Removed)
		}
		if result.FleetApply.BackupID == "" || result.BackupID != result.FleetApply.BackupID {
			t.Fatalf("backup identity missing: result %q detail %q", result.BackupID, result.FleetApply.BackupID)
		}
		if result.PreviewToken != preview.PreviewToken {
			t.Fatal("result did not carry the consumed token")
		}
		if _, err := os.Stat(f.target("sdd-implement-1")); err != nil {
			t.Fatalf("target was not installed: %v", err)
		}
		// The engine lock must be released after the write.
		lock, err := fleet.AcquireLock(skillsync.BackupRootFor("", f.home))
		if err != nil {
			t.Fatalf("lock leaked after apply: %v", err)
		}
		if err := lock.Release(); err != nil {
			t.Fatal(err)
		}
		// A fresh preview is a no-op and a fresh apply reports no change.
		secondPreview, err := FleetPreview(previewReq)
		if err != nil {
			t.Fatal(err)
		}
		secondReq, err := NewFleetApplyRequest(secondPreview.Plan, f.home)
		if err != nil {
			t.Fatal(err)
		}
		secondResult, err := ApplyFleet(secondReq, secondPreview.PreviewToken, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		if secondResult.Outcome != skillsync.OutcomeNoChange || secondResult.FleetApply == nil || !secondResult.FleetApply.NoOp {
			t.Fatalf("second apply = %q, want no-change", secondResult.Outcome)
		}
	})

	t.Run("update and remove counts", func(t *testing.T) {
		f := newFleetFixture(t, `{"schemaVersion":1,"instances":{
  "alpha":{"source":"preset:sdd-implement-1","model":"9router/ag/gemini-3.8-flash-high"},
  "beta":{"source":"preset:sdd-implement-1","model":"9router/ag/gemini-3.8-flash-high"}
}}`)
		fleetApplyForTest(t, f.cfg, "", f.home)
		writeFleetTestFile(t, f.cfg, `{"schemaVersion":1,"instances":{
  "alpha":{"source":"preset:sdd-implement-1","model":"9router/ag/gemini-3.8-flash-medium"},
  "beta":{"source":"preset:sdd-implement-1","model":"9router/ag/gemini-3.8-flash-high","enabled":false}
}}`)
		preview, err := FleetPreview(mustFleetPreviewRequest(t, f.cfg, "", f.home))
		if err != nil {
			t.Fatal(err)
		}
		applyReq, err := NewFleetApplyRequest(preview.Plan, f.home)
		if err != nil {
			t.Fatal(err)
		}
		result, err := ApplyFleet(applyReq, preview.PreviewToken, io.Discard)
		if err != nil {
			t.Fatalf("ApplyFleet: %v", err)
		}
		if result.FleetApply == nil {
			t.Fatal("missing FleetApply detail")
		}
		if result.FleetApply.Updated != 1 || result.FleetApply.Removed != 1 || result.FleetApply.Created != 0 {
			t.Fatalf("counts = created %d updated %d removed %d, want 0/1/1",
				result.FleetApply.Created, result.FleetApply.Updated, result.FleetApply.Removed)
		}
		if result.FleetApply.BackupID == "" {
			t.Fatal("update/remove apply did not record a backup")
		}
		if _, err := os.Stat(f.target("beta")); !os.IsNotExist(err) {
			t.Fatalf("disabled managed instance was not removed: %v", err)
		}
		if got := string(mustReadFile(t, f.target("alpha"))); !strings.Contains(got, "gemini-3.8-flash-medium") {
			t.Fatalf("updated target does not contain the new model: %s", got)
		}
	})

	t.Run("token mismatch refuses without writing", func(t *testing.T) {
		f := newFleetFixture(t, fleetTestConfig)
		previewReq, err := NewFleetPreviewRequest(f.cfg, "", f.home)
		if err != nil {
			t.Fatal(err)
		}
		preview, err := FleetPreview(previewReq)
		if err != nil {
			t.Fatal(err)
		}
		applyReq, err := NewFleetApplyRequest(preview.Plan, f.home)
		if err != nil {
			t.Fatal(err)
		}
		result, err := ApplyFleet(applyReq, "not-the-preview-token", io.Discard)
		if !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("error = %v, want ErrNeedsPreview", err)
		}
		if result.Outcome != skillsync.OutcomeFailed {
			t.Fatalf("outcome = %q, want failed", result.Outcome)
		}
		if _, err := os.Stat(f.target("sdd-implement-1")); !os.IsNotExist(err) {
			t.Fatalf("refused apply wrote a target: %v", err)
		}
		if _, err := os.Stat(filepath.Join(f.home, ".agentic-sdd", "backups")); !os.IsNotExist(err) {
			t.Fatalf("refused apply created a backup: %v", err)
		}
	})

	t.Run("empty token refuses", func(t *testing.T) {
		f := newFleetFixture(t, fleetTestConfig)
		preview, err := FleetPreview(mustFleetPreviewRequest(t, f.cfg, "", f.home))
		if err != nil {
			t.Fatal(err)
		}
		applyReq, err := NewFleetApplyRequest(preview.Plan, f.home)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ApplyFleet(applyReq, "", io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("error = %v, want ErrNeedsPreview", err)
		}
		if _, err := os.Stat(f.target("sdd-implement-1")); !os.IsNotExist(err) {
			t.Fatal("empty-token apply wrote a target")
		}
	})

	t.Run("stale config refuses", func(t *testing.T) {
		f := newFleetFixture(t, fleetTestConfig)
		preview, err := FleetPreview(mustFleetPreviewRequest(t, f.cfg, "", f.home))
		if err != nil {
			t.Fatal(err)
		}
		applyReq, err := NewFleetApplyRequest(preview.Plan, f.home)
		if err != nil {
			t.Fatal(err)
		}
		writeFleetTestFile(t, f.cfg, strings.Replace(fleetTestConfig, "gemini-3.8-flash-high", "opencode/other", 1))
		if _, err := ApplyFleet(applyReq, preview.PreviewToken, io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("error = %v, want ErrNeedsPreview", err)
		}
		if _, err := os.Stat(f.target("sdd-implement-1")); !os.IsNotExist(err) {
			t.Fatal("stale-config apply wrote a target")
		}
	})

	t.Run("stale target refuses", func(t *testing.T) {
		f := newFleetFixture(t, fleetTestConfig)
		preview, err := FleetPreview(mustFleetPreviewRequest(t, f.cfg, "", f.home))
		if err != nil {
			t.Fatal(err)
		}
		applyReq, err := NewFleetApplyRequest(preview.Plan, f.home)
		if err != nil {
			t.Fatal(err)
		}
		writeFleetTestFile(t, f.target("sdd-implement-1"), "appeared after preview\n")
		if _, err := ApplyFleet(applyReq, preview.PreviewToken, io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("error = %v, want ErrNeedsPreview", err)
		}
		if got := string(mustReadFile(t, f.target("sdd-implement-1"))); got != "appeared after preview\n" {
			t.Fatalf("stale-target apply changed the file: %q", got)
		}
	})

	t.Run("stale source refuses", func(t *testing.T) {
		root := t.TempDir()
		repo := filepath.Join(root, "repo")
		writeFleetTestFile(t, filepath.Join(repo, "opencode-pack", "agents", "sdd-implement-1.md"), fleetTestPackAgent)
		writeFleetTestFile(t, filepath.Join(repo, "opencode-pack", "adapters", "adapter-test.md"), fleetTestPackAgent)
		writeFleetTestFile(t, filepath.Join(repo, "opencode-pack", "presets", "free-first", "free-test.md"), fleetTestPackAgent)
		home := filepath.Join(root, "home")
		if err := os.MkdirAll(home, 0700); err != nil {
			t.Fatal(err)
		}
		cfg := filepath.Join(root, "fleet.json")
		writeFleetTestFile(t, cfg, `{"schemaVersion":1,"instances":{"sdd-implement-1":{"source":"agent:sdd-implement-1"}}}`)

		preview, err := FleetPreview(mustFleetPreviewRequest(t, cfg, repo, home))
		if err != nil {
			t.Fatal(err)
		}
		applyReq, err := NewFleetApplyRequest(preview.Plan, home)
		if err != nil {
			t.Fatal(err)
		}
		writeFleetTestFile(t, filepath.Join(repo, "opencode-pack", "agents", "sdd-implement-1.md"),
			strings.Replace(fleetTestPackAgent, "test profile", "changed profile", 1))
		if _, err := ApplyFleet(applyReq, preview.PreviewToken, io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("error = %v, want ErrNeedsPreview", err)
		}
		if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "agents", "sdd-implement-1.md")); !os.IsNotExist(err) {
			t.Fatal("stale-source apply wrote a target")
		}
	})
}

func TestFleetRestore(t *testing.T) {
	t.Run("preview and apply token match with unrelated files preserved", func(t *testing.T) {
		f := newFleetFixture(t, fleetTestConfig)
		fleetApplyForTest(t, f.cfg, "", f.home)

		secondConfig := strings.Replace(fleetTestConfig, "gemini-3.8-flash-high", "gemini-3.8-flash-medium", 1)
		writeFleetTestFile(t, f.cfg, secondConfig)
		fleetApplyForTest(t, f.cfg, "", f.home)

		backups, err := fleet.ListFleetBackups(f.home)
		if err != nil {
			t.Fatal(err)
		}
		if len(backups) < 2 {
			t.Fatalf("fleet backups = %d, want at least 2", len(backups))
		}
		backupID := backups[0].ID

		unrelated := filepath.Join(f.targetRoot(), "unrelated.md")
		writeFleetTestFile(t, unrelated, "keep me\n")

		previewReq, err := NewFleetRestorePreviewRequest(f.home, backupID)
		if err != nil {
			t.Fatal(err)
		}
		preview, err := FleetRestorePreview(previewReq)
		if err != nil {
			t.Fatalf("FleetRestorePreview: %v", err)
		}
		if preview.PreviewToken == "" || preview.Preview == nil {
			t.Fatal("restore preview is incomplete")
		}
		if preview.BackupID != backupID || preview.Preview.BackupID != backupID {
			t.Fatalf("backup identity = %q/%q, want %q", preview.BackupID, preview.Preview.BackupID, backupID)
		}
		if len(preview.Preview.Files) == 0 {
			t.Fatal("restore preview lists no files")
		}

		applyReq, err := NewFleetRestoreApplyRequest(f.home, backupID)
		if err != nil {
			t.Fatal(err)
		}
		result, err := ApplyFleetRestore(applyReq, preview.PreviewToken, io.Discard)
		if err != nil {
			t.Fatalf("ApplyFleetRestore: %v", err)
		}
		if result.Outcome != skillsync.OutcomeApplied {
			t.Fatalf("outcome = %q, want applied", result.Outcome)
		}
		if result.FleetRestore == nil || result.FleetRestore.Restored == 0 {
			t.Fatalf("restore detail = %+v", result.FleetRestore)
		}
		if result.PreviewToken != preview.PreviewToken {
			t.Fatal("restore result did not carry the consumed token")
		}
		if got := string(mustReadFile(t, unrelated)); got != "keep me\n" {
			t.Fatalf("restore changed an unrelated file: %q", got)
		}
	})

	t.Run("token mismatch refuses without writing", func(t *testing.T) {
		f := newFleetFixture(t, fleetTestConfig)
		fleetApplyForTest(t, f.cfg, "", f.home)
		writeFleetTestFile(t, f.cfg, strings.Replace(fleetTestConfig, "gemini-3.8-flash-high", "gemini-3.8-flash-medium", 1))
		fleetApplyForTest(t, f.cfg, "", f.home)

		backups, err := fleet.ListFleetBackups(f.home)
		if err != nil || len(backups) == 0 {
			t.Fatalf("list backups: %v (%d)", err, len(backups))
		}
		backupID := backups[0].ID
		before := mustReadFile(t, f.target("sdd-implement-1"))

		applyReq, err := NewFleetRestoreApplyRequest(f.home, backupID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ApplyFleetRestore(applyReq, "wrong-token", io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("error = %v, want ErrNeedsPreview", err)
		}
		if got := mustReadFile(t, f.target("sdd-implement-1")); string(got) != string(before) {
			t.Fatal("refused restore changed the target")
		}
	})

	t.Run("modified target refuses at preview and apply", func(t *testing.T) {
		f := newFleetFixture(t, fleetTestConfig)
		fleetApplyForTest(t, f.cfg, "", f.home)
		writeFleetTestFile(t, f.cfg, strings.Replace(fleetTestConfig, "gemini-3.8-flash-high", "gemini-3.8-flash-medium", 1))
		fleetApplyForTest(t, f.cfg, "", f.home)

		backups, err := fleet.ListFleetBackups(f.home)
		if err != nil || len(backups) == 0 {
			t.Fatalf("list backups: %v (%d)", err, len(backups))
		}
		backupID := backups[0].ID
		previewReq, err := NewFleetRestorePreviewRequest(f.home, backupID)
		if err != nil {
			t.Fatal(err)
		}
		preview, err := FleetRestorePreview(previewReq)
		if err != nil {
			t.Fatal(err)
		}
		applyReq, err := NewFleetRestoreApplyRequest(f.home, backupID)
		if err != nil {
			t.Fatal(err)
		}
		writeFleetTestFile(t, f.target("sdd-implement-1"), "modified after preview\n")
		if _, err := ApplyFleetRestore(applyReq, preview.PreviewToken, io.Discard); !errors.Is(err, ErrNeedsPreview) {
			t.Fatalf("apply error = %v, want ErrNeedsPreview", err)
		}
		if _, err := FleetRestorePreview(previewReq); !errors.Is(err, fleet.ErrUnsafeTarget) {
			t.Fatalf("preview error = %v, want ErrUnsafeTarget", err)
		}
		if got := string(mustReadFile(t, f.target("sdd-implement-1"))); got != "modified after preview\n" {
			t.Fatalf("refused restore changed the modified target: %q", got)
		}
	})

	t.Run("legacy backup refuses", func(t *testing.T) {
		f := newFleetFixture(t, fleetTestConfig)
		backupID := "20260101T000000.000000000Z"
		legacyDir := filepath.Join(f.home, ".agentic-sdd", "backups", backupID)
		if err := os.MkdirAll(legacyDir, 0700); err != nil {
			t.Fatal(err)
		}
		previewReq, err := NewFleetRestorePreviewRequest(f.home, backupID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := FleetRestorePreview(previewReq); !errors.Is(err, fleet.ErrLegacyProvenance) {
			t.Fatalf("error = %v, want ErrLegacyProvenance", err)
		}
	})

	t.Run("validation", func(t *testing.T) {
		home := t.TempDir()
		if _, err := NewFleetRestorePreviewRequest("", "20260101T000000.000000000Z"); err == nil {
			t.Fatal("empty home accepted")
		}
		if _, err := NewFleetRestorePreviewRequest(home, "../escape"); err == nil {
			t.Fatal("invalid backup ID accepted")
		}
		if _, err := NewFleetRestoreApplyRequest(home, ""); err == nil {
			t.Fatal("empty backup ID accepted")
		}
	})
}

func TestFleetDoctor(t *testing.T) {
	t.Run("offline default consults nothing", func(t *testing.T) {
		req, err := NewFleetDoctorRequest([]string{"9router/ag/gemini-3.8-flash-high"}, "", "")
		if err != nil {
			t.Fatal(err)
		}
		res := FleetDoctor(req)
		if !res.Report.Offline {
			t.Fatal("default doctor is not offline")
		}
		if len(res.Report.ToolsInspected) != 0 || len(res.Report.Failures) != 0 {
			t.Fatalf("offline doctor inspected or failed: %+v", res.Report)
		}
		if len(res.Report.Entries) != 1 || res.Report.Entries[0].Label != fleet.DoctorNotChecked {
			t.Fatalf("offline labels = %+v", res.Report.Entries)
		}
	})

	t.Run("explicit local catalog dump labels from local evidence only", func(t *testing.T) {
		dump := filepath.Join(t.TempDir(), "catalog.json")
		writeFleetTestFile(t, dump, `{"providers":[{"id":"9router","auth":"authenticated"}],"models":["9router/ag/gemini-3.8-flash-high"]}`)
		req, err := NewFleetDoctorRequest([]string{"9router/ag/gemini-3.8-flash-high"}, "", dump)
		if err != nil {
			t.Fatal(err)
		}
		res := FleetDoctor(req)
		if res.Report.Offline {
			t.Fatal("explicit catalog dump did not enable local inspection")
		}
		if len(res.Report.Entries) != 1 {
			t.Fatalf("entries = %+v", res.Report.Entries)
		}
		if got := res.Report.Entries[0].Label; got != fleet.DoctorAuthenticated {
			t.Fatalf("label = %q, want authenticated from local marker", got)
		}
	})

	t.Run("validation and immutability", func(t *testing.T) {
		models := []string{"a/b", "c/d"}
		req, err := NewFleetDoctorRequest(models, "", "")
		if err != nil {
			t.Fatal(err)
		}
		models[0] = "changed/x"
		if got := req.Models()[0]; got != "a/b" {
			t.Fatalf("request models changed to %q", got)
		}
		returned := req.Models()
		returned[1] = "changed/y"
		if got := req.Models()[1]; got != "c/d" {
			t.Fatalf("returned models mutated the request: %q", got)
		}
	})
}

func TestFleetApplyRequestValidation(t *testing.T) {
	f := newFleetFixture(t, fleetTestConfig)
	preview, err := FleetPreview(mustFleetPreviewRequest(t, f.cfg, "", f.home))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewFleetApplyRequest(nil, f.home); err == nil {
		t.Fatal("nil plan accepted")
	}
	if _, err := NewFleetApplyRequest(preview.Plan, ""); err == nil {
		t.Fatal("empty home accepted")
	}
	unsealed := *preview.Plan
	unsealed.PlanID = ""
	if _, err := NewFleetApplyRequest(&unsealed, f.home); err == nil {
		t.Fatal("plan without identity accepted")
	}
	if _, err := NewFleetPreviewRequest("", "", f.home); err == nil {
		t.Fatal("preview without config accepted")
	}
	if _, err := NewFleetPreviewRequest(f.cfg, "", ""); err == nil {
		t.Fatal("preview without home accepted")
	}
}

func mustFleetPreviewRequest(t *testing.T, configPath, repo, home string) FleetPreviewRequest {
	t.Helper()
	req, err := NewFleetPreviewRequest(configPath, repo, home)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func fleetInstanceIDs(n int) []string {
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		ids = append(ids, "agent-"+pad3(i))
	}
	return ids
}

func pad3(n int) string {
	digits := []byte{byte('0' + n/100%10), byte('0' + n/10%10), byte('0' + n%10)}
	if n >= 1000 {
		return "999"
	}
	return string(digits)
}

func fleetConfigForIDs(ids []string) string {
	var b strings.Builder
	b.WriteString(`{"schemaVersion":1,"instances":{`)
	for i, id := range ids {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`"` + id + `":{"source":"preset:sdd-implement-1","model":"9router/ag/gemini-3.8-flash-high"}`)
	}
	b.WriteString("}}")
	return b.String()
}
