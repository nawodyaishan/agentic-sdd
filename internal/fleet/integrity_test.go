package fleet

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"agentic-sdd/internal/skillsync"
)

func TestFleetApplyIntegrity(t *testing.T) {
	t.Run("resealed plan cannot force a drifted overwrite", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		applyPlan(t, buildPlanFor(t, configPath, home, false), home)
		target := filepath.Join(TargetRootForHome(home), "sdd-lightweight.md")
		drifted := "manual drift after apply\n"
		if err := os.WriteFile(target, []byte(drifted), 0644); err != nil {
			t.Fatal(err)
		}

		plan := buildPlanFor(t, configPath, home, false)
		if !plan.HasConflicts() {
			t.Fatalf("expected a drift conflict, got %+v", plan.ProposedOps)
		}

		// Rewrite every plan field that used to authorize the write, then
		// re-seal: the plan is a consistency token, not authorization.
		mutated := *plan
		mutated.ProposedOps = append([]ProposedOp(nil), plan.ProposedOps...)
		mutated.LastKnownInstalled = map[string]string{}
		for k, v := range plan.LastKnownInstalled {
			mutated.LastKnownInstalled[k] = v
		}
		for i := range mutated.ProposedOps {
			op := &mutated.ProposedOps[i]
			op.Op = "update"
			op.Adopt = true
			op.Reason = ""
			op.CurrentDigest = digestOf([]byte(drifted))
			mutated.LastKnownInstalled[op.File] = digestOf([]byte(drifted))
		}
		if err := mutated.Seal(); err != nil {
			t.Fatal(err)
		}

		var out bytes.Buffer
		if _, err := ApplyPlan(&mutated, home, &out); !errors.Is(err, ErrUnsafeTarget) {
			t.Fatalf("expected the re-derived drift state to refuse the write, got %v", err)
		}
		if got := readString(t, target); got != drifted {
			t.Fatalf("resealed plan overwrote manually drifted content: %q", got)
		}
	})

	t.Run("resealed plan cannot break FileList, source or model symmetry", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		plan := buildPlanFor(t, configPath, home, false)

		clone := func() *Plan {
			cp := *plan
			cp.ProposedOps = append([]ProposedOp(nil), plan.ProposedOps...)
			cp.FileList = append([]string(nil), plan.FileList...)
			cp.LastKnownInstalled = map[string]string{}
			for k, v := range plan.LastKnownInstalled {
				cp.LastKnownInstalled[k] = v
			}
			return &cp
		}
		apply := func(p *Plan) error {
			t.Helper()
			if err := p.Seal(); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			_, err := ApplyPlan(p, home, &out)
			return err
		}

		sourceDigest := clone()
		sourceDigest.ProposedOps[0].SourceDigest = digestOf([]byte("not the canonical source"))
		if err := apply(sourceDigest); !errors.Is(err, ErrStalePlan) {
			t.Fatalf("expected ErrStalePlan for a changed source digest, got %v", err)
		}

		sourceRef := clone()
		sourceRef.ProposedOps[0].SourceRef = "preset:sdd-lightweight"
		if err := apply(sourceRef); !errors.Is(err, ErrStalePlan) {
			t.Fatalf("expected ErrStalePlan for a changed source reference, got %v", err)
		}

		model := clone()
		model.ProposedOps[0].Model = "9router/ag/claude-sonnet-5.5-high"
		if err := apply(model); !errors.Is(err, ErrStalePlan) {
			t.Fatalf("expected ErrStalePlan for a changed model, got %v", err)
		}

		duplicateFile := clone()
		duplicateFile.FileList = append(duplicateFile.FileList, plan.FileList[0])
		if err := apply(duplicateFile); !errors.Is(err, ErrPlanConflict) {
			t.Fatalf("expected ErrPlanConflict for a duplicate file list entry, got %v", err)
		}

		missingOperation := clone()
		missingOperation.ProposedOps[0].File = "other-agent.md"
		if err := apply(missingOperation); !errors.Is(err, ErrPlanConflict) {
			t.Fatalf("expected ErrPlanConflict for an operation outside the file list, got %v", err)
		}
		if _, err := os.Lstat(filepath.Join(TargetRootForHome(home), "sdd-lightweight.md")); !os.IsNotExist(err) {
			t.Fatalf("a rejected plan wrote a target: %v", err)
		}
	})

	t.Run("drifted managed removal requires explicit adoption", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		applyPlan(t, buildPlanFor(t, configPath, home, false), home)
		target := filepath.Join(TargetRootForHome(home), "sdd-lightweight.md")
		drifted := "edited before disable\n"
		if err := os.WriteFile(target, []byte(drifted), 0644); err != nil {
			t.Fatal(err)
		}
		disabledConfig := writeFleetConfig(t, t.TempDir(), `"sdd-lightweight":{"source":"agent:sdd-lightweight","enabled":false}`)

		plan := buildPlanFor(t, disabledConfig, home, false)
		if plan.ProposedOps[0].Op != "conflict" {
			t.Fatalf("expected the drifted removal to conflict, got %+v", plan.ProposedOps[0])
		}
		var out bytes.Buffer
		if _, err := ApplyPlan(plan, home, &out); !errors.Is(err, ErrPlanConflict) {
			t.Fatalf("expected ErrPlanConflict, got %v", err)
		}
		if got := readString(t, target); got != drifted {
			t.Fatalf("conflicting removal deleted or rewrote drifted content: %q", got)
		}

		adopt := buildPlanFor(t, disabledConfig, home, true)
		if adopt.ProposedOps[0].Op != "remove" || !adopt.ProposedOps[0].Adopt {
			t.Fatalf("expected an explicit adopted removal, got %+v", adopt.ProposedOps[0])
		}
		result := applyPlan(t, adopt, home)
		if result.Removed != 1 {
			t.Fatalf("expected one removal, got %+v", result)
		}
		if _, err := os.Lstat(target); !os.IsNotExist(err) {
			t.Fatalf("adopted removal did not remove the profile: %v", err)
		}
	})

	t.Run("lost live pointer is rebuilt from completed backups", func(t *testing.T) {
		home := newFleetHome(t)
		firstConfig := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		applyPlan(t, buildPlanFor(t, firstConfig, home, false), home)

		if err := os.Remove(LivePointerPath(home)); err != nil {
			t.Fatal(err)
		}
		if live, err := ReadLiveManifest(home); err != nil || live != nil {
			t.Fatalf("expected the live pointer to be absent, got %+v (%v)", live, err)
		}

		secondConfig := writeFleetConfig(t, t.TempDir(),
			singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high")+","+
				`"sdd-debug-diagnose":{"source":"agent:sdd-debug-diagnose","model":"9router/ag/gemini-3.8-flash-high"}`)
		plan := buildPlanFor(t, secondConfig, home, false)
		if plan.HasConflicts() {
			t.Fatalf("a lost live pointer must not turn a managed file into a conflict: %+v", plan.ProposedOps)
		}
		for _, op := range plan.ProposedOps {
			if op.InstanceID == "sdd-lightweight" && op.Op != "noop" {
				t.Fatalf("rebuilt ownership did not preserve the managed no-op: %+v", op)
			}
		}
		applyPlan(t, plan, home)

		live, err := ReadLiveManifest(home)
		if err != nil || live == nil {
			t.Fatalf("read live manifest: %v", err)
		}
		for _, name := range []string{"sdd-lightweight.md", "sdd-debug-diagnose.md"} {
			target := filepath.Join(TargetRootForHome(home), name)
			rec, ok := live.RecordFor(target)
			if !ok || !rec.IsManaged() {
				t.Fatalf("rebuilt live manifest lost ownership of %s: %+v", name, live)
			}
		}
	})

	t.Run("concurrent applies serialize and never double-write", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		plan := buildPlanFor(t, configPath, home, false)

		const workers = 8
		var wg sync.WaitGroup
		start := make(chan struct{})
		errs := make([]error, workers)
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				var out bytes.Buffer
				_, errs[i] = ApplyPlan(plan, home, &out)
			}(i)
		}
		close(start)
		wg.Wait()

		success := 0
		for i, err := range errs {
			if err == nil {
				success++
				continue
			}
			if !errors.Is(err, ErrLockHeld) && !errors.Is(err, ErrStalePlan) {
				t.Fatalf("worker %d failed with an unexpected error: %v", i, err)
			}
		}
		if success != 1 {
			t.Fatalf("expected exactly one successful apply, got %d (errors: %v)", success, errs)
		}

		target := filepath.Join(TargetRootForHome(home), "sdd-lightweight.md")
		installed := readString(t, target)
		if !strings.Contains(installed, "model: 9router/ag/gemini-3.8-flash-high\n") {
			t.Fatalf("concurrent apply produced incomplete content:\n%s", installed)
		}
		live, err := ReadLiveManifest(home)
		if err != nil || live == nil {
			t.Fatalf("read live manifest: %v", err)
		}
		if rec, ok := live.RecordFor(target); !ok || rec.InstalledDigest != digestOf([]byte(installed)) {
			t.Fatalf("live manifest does not match the written bytes: %+v", live)
		}
	})
}

func TestFleetBackupDigestVerification(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "opencode-agents"), 0700); err != nil {
		t.Fatal(err)
	}
	saved := []byte("previous profile\n")
	if err := os.WriteFile(filepath.Join(dir, "opencode-agents", "sdd-lightweight.md"), saved, 0600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "sdd-lightweight.md")
	record := FileRecord{
		Target:       target,
		BackupRef:    "opencode-agents/sdd-lightweight.md",
		BackupDigest: digestOf(saved),
	}
	if err := verifyBackupCopies(dir, []FileRecord{record}); err != nil {
		t.Fatalf("expected the matching backup copy to verify: %v", err)
	}

	mismatched := record
	mismatched.BackupDigest = digestOf([]byte("another file"))
	if err := verifyBackupCopies(dir, []FileRecord{mismatched}); !errors.Is(err, ErrUnsafeTarget) {
		t.Fatalf("expected a digest mismatch to be refused, got %v", err)
	}

	missing := record
	missing.BackupDigest = ""
	if err := verifyBackupCopies(dir, []FileRecord{missing}); !errors.Is(err, ErrUnsafeTarget) {
		t.Fatalf("expected unrecorded content to be refused, got %v", err)
	}

	traversal := record
	traversal.BackupRef = "../sdd-lightweight.md"
	if err := verifyBackupCopies(dir, []FileRecord{traversal}); !errors.Is(err, ErrUnsafeTarget) {
		t.Fatalf("expected an unexpected backup reference to be refused, got %v", err)
	}

	symlinked := filepath.Join(dir, "opencode-agents", "linked.md")
	if err := os.Symlink(filepath.Join(dir, "opencode-agents", "sdd-lightweight.md"), symlinked); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	linkRecord := FileRecord{
		Target:       filepath.Join(t.TempDir(), "linked.md"),
		BackupRef:    "opencode-agents/linked.md",
		BackupDigest: digestOf(saved),
	}
	if err := verifyBackupCopies(dir, []FileRecord{linkRecord}); !errors.Is(err, ErrUnsafeTarget) {
		t.Fatalf("expected a symlinked backup copy to be refused, got %v", err)
	}
}

func TestFleetLockAtomic(t *testing.T) {
	root := t.TempDir()
	const workers = 12
	var wg sync.WaitGroup
	start := make(chan struct{})
	locks := make([]*Lock, workers)
	errs := make([]error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			locks[i], errs[i] = AcquireLock(root)
		}(i)
	}
	close(start)
	wg.Wait()

	var winners []*Lock
	for i, err := range errs {
		if err == nil {
			winners = append(winners, locks[i])
			continue
		}
		if !errors.Is(err, ErrLockHeld) {
			t.Fatalf("worker %d failed with an unexpected error: %v", i, err)
		}
		if locks[i] != nil {
			_ = locks[i].Release()
			t.Fatalf("worker %d reported both a lock and an error", i)
		}
	}
	if len(winners) != 1 {
		for _, l := range winners {
			_ = l.Release()
		}
		t.Fatalf("expected exactly one lock winner, got %d", len(winners))
	}
	if err := winners[0].Release(); err != nil {
		t.Fatalf("release winner: %v", err)
	}

	reacquired, err := AcquireLock(root)
	if err != nil {
		t.Fatalf("reacquire after release: %v", err)
	}
	if err := reacquired.Release(); err != nil {
		t.Fatalf("release reacquired lock: %v", err)
	}
}

func TestFleetLockOwnership(t *testing.T) {
	root := t.TempDir()
	lock, err := AcquireLock(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, LockFileName)

	// Replace the path with another owner's lock file, as an external actor
	// could. Releasing ours must never delete or rewrite that file.
	replacement := fmt.Sprintf("4321\n%s\nsomeone-else\n", time.Now().UTC().Format(time.RFC3339Nano))
	if err := os.WriteFile(path, []byte(replacement), 0600); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if got := readString(t, path); got != replacement {
		t.Fatalf("Release modified a lock it no longer owns: %q", got)
	}
}

func TestFleetLockOwnerRecordIsDiagnostic(t *testing.T) {
	root := t.TempDir()
	lock, err := AcquireLock(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, LockFileName)
	// An alive PID and a stale timestamp are only diagnostics: the held
	// operating-system lock remains authoritative.
	staleRecord := fmt.Sprintf("%d\n%s\nrecorded\n", os.Getpid(), time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano))
	if err := os.WriteFile(path, []byte(staleRecord), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := AcquireLock(root); !errors.Is(err, ErrLockHeld) {
		t.Fatalf("expected the held lock to stay authoritative, got %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}

	// An abandoned record with no live descriptor is reclaimable regardless
	// of its recorded age or PID.
	abandoned := fmt.Sprintf("999999\n%s\nabandoned\n", time.Now().UTC().Format(time.RFC3339Nano))
	if err := os.WriteFile(path, []byte(abandoned), 0600); err != nil {
		t.Fatal(err)
	}
	reclaimed, err := AcquireLock(root)
	if err != nil {
		t.Fatalf("expected an unlocked record to be reclaimed: %v", err)
	}
	lines := strings.Split(readString(t, path), "\n")
	if len(lines) == 0 || lines[0] != strconv.Itoa(os.Getpid()) {
		t.Fatalf("reclaimed lock does not record this process: %q", lines)
	}
	if err := reclaimed.Release(); err != nil {
		t.Fatalf("release reclaimed lock: %v", err)
	}
}

func TestFleetRestoreIntegrity(t *testing.T) {
	t.Run("managed update restores and replans without adoption", func(t *testing.T) {
		home := newFleetHome(t)
		firstConfig := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		applyPlan(t, buildPlanFor(t, firstConfig, home, false), home)
		target := filepath.Join(TargetRootForHome(home), "sdd-lightweight.md")
		firstRender := readString(t, target)

		secondConfig := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/claude-sonnet-5.5-high"))
		second := applyPlan(t, buildPlanFor(t, secondConfig, home, false), home)
		if second.Updated != 1 || second.BackupID == "" {
			t.Fatalf("unexpected update result: %+v", second)
		}

		var out bytes.Buffer
		restored, err := RestoreBackup(home, second.BackupID, &out)
		if err != nil {
			t.Fatalf("RestoreBackup: %v\noutput: %s", err, out.String())
		}
		if got := readString(t, target); got != firstRender {
			t.Fatalf("restore did not return the previous managed render: %q", got)
		}
		sidecar, err := ReadManifestFile(filepath.Join(restored.BackupPath, ManifestFileName))
		if err != nil || sidecar == nil {
			t.Fatalf("read restore sidecar: %v", err)
		}
		if len(sidecar.Files) != 1 || sidecar.Files[0].Provenance != ProvenanceRestored {
			t.Fatalf("restored managed content must keep managed-style provenance: %+v", sidecar.Files)
		}
		if sidecar.Files[0].DesiredDigest != digestOf([]byte(firstRender)) {
			t.Fatalf("restored record does not bind the restored bytes: %+v", sidecar.Files[0])
		}

		replan := buildPlanFor(t, secondConfig, home, false)
		if replan.HasConflicts() {
			t.Fatalf("replan after a managed restore must not conflict: %+v", replan.ProposedOps)
		}
		if replan.ProposedOps[0].Op != "update" {
			t.Fatalf("expected a normal update after restore, got %+v", replan.ProposedOps[0])
		}
		applyPlan(t, replan, home)
		if got := readString(t, target); !strings.Contains(got, "model: 9router/ag/claude-sonnet-5.5-high\n") {
			t.Fatalf("replan after restore did not reapply the render: %q", got)
		}
	})

	t.Run("adopted content stays unmanaged after restore", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", "9router/ag/gemini-3.8-flash-high"))
		targetRoot := TargetRootForHome(home)
		if err := os.MkdirAll(targetRoot, 0755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetRoot, "sdd-lightweight.md")
		original := "user-owned before adoption\n"
		if err := os.WriteFile(target, []byte(original), 0644); err != nil {
			t.Fatal(err)
		}
		result := applyPlan(t, buildPlanFor(t, configPath, home, true), home)

		var out bytes.Buffer
		restored, err := RestoreBackup(home, result.BackupID, &out)
		if err != nil {
			t.Fatalf("RestoreBackup: %v\noutput: %s", err, out.String())
		}
		if got := readString(t, target); got != original {
			t.Fatalf("adoption reversal did not restore the user content: %q", got)
		}
		sidecar, err := ReadManifestFile(filepath.Join(restored.BackupPath, ManifestFileName))
		if err != nil || sidecar == nil {
			t.Fatalf("read restore sidecar: %v", err)
		}
		if len(sidecar.Files) != 1 || sidecar.Files[0].Provenance != ProvenanceUnknown {
			t.Fatalf("restored unmanaged content must not be claimed: %+v", sidecar.Files)
		}
		live, err := ReadLiveManifest(home)
		if err != nil || live == nil {
			t.Fatalf("read live manifest: %v", err)
		}
		if rec, ok := live.RecordFor(target); ok && rec.IsManaged() {
			t.Fatalf("restored unmanaged content is still claimed as managed: %+v", rec)
		}

		replan := buildPlanFor(t, configPath, home, false)
		if !replan.HasConflicts() {
			t.Fatalf("replan over restored user content must conflict without adoption: %+v", replan.ProposedOps)
		}
	})

	t.Run("legacy listing and restore accept fleet backups with non-sdd profiles", func(t *testing.T) {
		home := newFleetHome(t)
		configPath := writeFleetConfig(t, t.TempDir(), `"my-agent":{"source":"agent:sdd-lightweight","model":"9router/ag/gemini-3.8-flash-high"}`)
		targetRoot := TargetRootForHome(home)
		if err := os.MkdirAll(targetRoot, 0755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(targetRoot, "my-agent.md")
		original := "user original profile\n"
		if err := os.WriteFile(target, []byte(original), 0644); err != nil {
			t.Fatal(err)
		}
		result := applyPlan(t, buildPlanFor(t, configPath, home, true), home)
		if got := readString(t, target); got == original {
			t.Fatal("adoption did not install the rendered profile")
		}

		backups, err := skillsync.ListBackups("", home)
		if err != nil {
			t.Fatalf("ListBackups: %v", err)
		}
		var found bool
		for _, b := range backups {
			if b.ID != result.BackupID {
				continue
			}
			found = true
			if b.Unusable != "" {
				t.Fatalf("a fleet backup with a fleet-manifest.json sidecar is unusable: %s", b.Unusable)
			}
			if !strings.Contains(b.Summary, "opencode-agents") {
				t.Fatalf("fleet backup summary does not describe the agent profile: %q", b.Summary)
			}
		}
		if !found {
			t.Fatalf("fleet backup %s is not listed", result.BackupID)
		}

		var out bytes.Buffer
		if err := skillsync.Restore("", home, result.BackupID, true, &out); err != nil {
			t.Fatalf("legacy restore of a fleet backup: %v\noutput: %s", err, out.String())
		}
		if got := readString(t, target); got != original {
			t.Fatalf("legacy restore did not return the non-sdd profile content: %q", got)
		}
	})
}
