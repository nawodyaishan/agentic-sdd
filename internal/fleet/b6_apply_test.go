package fleet

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
)

// B6 (T13) fleet consumer tests: lock order and exclusion, before-state under
// the guard, metadata finalization outcomes, reconciliation and read-only status.

const (
	b6Model1 = "9router/ag/gemini-3.8-flash-high"
	b6Model2 = "9router/ag/gemini-3.8-flash-medium"
	b6Target = "sdd-lightweight.md"
)

func b6CanonicalHome(t *testing.T, home string) string {
	t.Helper()
	abs, err := filepath.Abs(home)
	if err != nil {
		t.Fatal(err)
	}
	canon, err := filepath.EvalSymlinks(abs)
	if err != nil {
		t.Fatal(err)
	}
	return canon
}

// b6SetSeams installs test seams for one test and restores the previous ones.
func b6SetSeams(t *testing.T, s b6Seams) {
	t.Helper()
	prev := b6Test
	b6Test = s
	t.Cleanup(func() { b6Test = prev })
}

func b6ConfigFor(t *testing.T, model string) string {
	t.Helper()
	return writeFleetConfig(t, t.TempDir(), singleInstance("agent:sdd-lightweight", model))
}

// b6Create applies a fresh create of the managed profile and returns its backup.
func b6Create(t *testing.T, home string) {
	t.Helper()
	applyPlan(t, buildPlanFor(t, b6ConfigFor(t, b6Model1), home, false), home)
}

// b6Update returns a plan that changes the installed profile to b6Model2.
func b6Update(t *testing.T, home string) *Plan {
	t.Helper()
	plan := buildPlanFor(t, b6ConfigFor(t, b6Model2), home, false)
	if len(plan.ProposedOps) != 1 || plan.ProposedOps[0].Op != "update" {
		t.Fatalf("expected one update op, got %+v", plan.ProposedOps)
	}
	return plan
}

func b6Sidecar(t *testing.T, backupDir string) *Manifest {
	t.Helper()
	m, err := ReadManifestFile(filepath.Join(backupDir, ManifestFileName))
	if err != nil || m == nil {
		t.Fatalf("read sidecar in %s: %v (nil=%t)", backupDir, err, m == nil)
	}
	return m
}

func b6LiveExists(t *testing.T, home string) bool {
	t.Helper()
	_, err := os.Stat(LivePointerPath(home))
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	t.Fatal(err)
	return false
}

func TestB6LockOrderIsLegacyThenAgentGuard(t *testing.T) {
	home := newFleetHome(t)
	plan := buildPlanFor(t, b6ConfigFor(t, b6Model1), home, false)
	var order []string
	b6SetSeams(t, b6Seams{acquired: func(step string) { order = append(order, step) }})
	applyPlan(t, plan, home)
	if got := strings.Join(order, ","); got != "legacy,agent" {
		t.Fatalf("lock acquisition order = %q, want legacy,agent", got)
	}
}

func TestB6RestoreLockOrderIsLegacyThenAgentGuard(t *testing.T) {
	home := newFleetHome(t)
	b6Create(t, home)
	plan := b6Update(t, home)
	applyPlan(t, plan, home)
	backup := filepath.Base(latestBackup(t, home))
	var order []string
	b6SetSeams(t, b6Seams{acquired: func(step string) { order = append(order, step) }})
	if _, err := RestoreBackup(home, backup, io.Discard); err != nil {
		t.Fatalf("RestoreBackup: %v", err)
	}
	if got := strings.Join(order, ","); got != "legacy,agent" {
		t.Fatalf("restore lock order = %q, want legacy,agent", got)
	}
}

func TestB6AgentGuardHeldElsewhereRefusesAfterLegacyOnly(t *testing.T) {
	home := newFleetHome(t)
	plan := buildPlanFor(t, b6ConfigFor(t, b6Model1), home, false)
	guard, err := skillsync.AcquireTargetMutation(b6CanonicalHome(t, home), skillsync.DomainAgentTarget)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = guard.Close() }()

	var order []string
	b6SetSeams(t, b6Seams{acquired: func(step string) { order = append(order, step) }})
	_, err = ApplyPlan(plan, home, io.Discard)
	if !errors.Is(err, skillsync.ErrMutationConflict) {
		t.Fatalf("ApplyPlan error = %v, want ErrMutationConflict", err)
	}
	if got := strings.Join(order, ","); got != "legacy" {
		t.Fatalf("acquisitions before refusal = %q, want legacy only", got)
	}
	if n := len(backupDirs(t, home)); n != 0 {
		t.Fatalf("refused apply created %d backup(s)", n)
	}
	if _, err := os.Stat(filepath.Join(TargetRootForHome(home), b6Target)); !os.IsNotExist(err) {
		t.Fatalf("refused apply wrote the target: %v", err)
	}
	// The legacy lock must have been released on the refusal path.
	lock, err := AcquireLock(skillsync.BackupRootFor("", home))
	if err != nil {
		t.Fatalf("legacy lock leaked after refusal: %v", err)
	}
	_ = lock.Release()
}

func TestB6LegacyLockHeldRefusesBeforeAgentGuard(t *testing.T) {
	home := newFleetHome(t)
	plan := buildPlanFor(t, b6ConfigFor(t, b6Model1), home, false)
	lock, err := AcquireLock(skillsync.BackupRootFor("", home))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.Release() }()

	var order []string
	b6SetSeams(t, b6Seams{acquired: func(step string) { order = append(order, step) }})
	if _, err := ApplyPlan(plan, home, io.Discard); !errors.Is(err, ErrLockHeld) {
		t.Fatalf("ApplyPlan error = %v, want ErrLockHeld", err)
	}
	if len(order) != 0 {
		t.Fatalf("agent guard was reached while the legacy lock was held: %v", order)
	}
	guard, err := skillsync.AcquireTargetMutation(b6CanonicalHome(t, home), skillsync.DomainAgentTarget)
	if err != nil {
		t.Fatalf("agent guard unexpectedly held: %v", err)
	}
	_ = guard.Close()
}

func TestB6ExpectedDigestIsCheckedUnderTheGuard(t *testing.T) {
	home := newFleetHome(t)
	b6Create(t, home)
	plan := b6Update(t, home)
	target := filepath.Join(TargetRootForHome(home), b6Target)
	// The race happens after plan revalidation and before the guarded engine.
	// Only the planned expected digest can catch it.
	b6SetSeams(t, b6Seams{beforeEngine: func() {
		if err := os.WriteFile(target, []byte("racer\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}})
	if _, err := ApplyPlan(plan, home, io.Discard); !errors.Is(err, skillsync.ErrBeforeStateChanged) {
		t.Fatalf("ApplyPlan error = %v, want ErrBeforeStateChanged", err)
	}
	if got := readString(t, target); got != "racer\n" {
		t.Fatalf("racing bytes were overwritten: %q", got)
	}
	if n := len(backupDirs(t, home)); n != 1 {
		t.Fatalf("expected only the first apply's backup, got %d", n)
	}
}

func TestB6RestoreExpectedDigestIsCheckedUnderTheGuard(t *testing.T) {
	home := newFleetHome(t)
	b6Create(t, home)
	applyPlan(t, b6Update(t, home), home)
	backup := filepath.Base(latestBackup(t, home))
	target := filepath.Join(TargetRootForHome(home), b6Target)
	b6SetSeams(t, b6Seams{beforeEngine: func() {
		if err := os.WriteFile(target, []byte("racer\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}})
	if _, err := RestoreBackup(home, backup, io.Discard); !errors.Is(err, skillsync.ErrBeforeStateChanged) {
		t.Fatalf("RestoreBackup error = %v, want ErrBeforeStateChanged", err)
	}
	if got := readString(t, target); got != "racer\n" {
		t.Fatalf("racing bytes were overwritten by restore: %q", got)
	}
}

func TestB6NewerTargetBetweenPlanAndApplyIsRefused(t *testing.T) {
	home := newFleetHome(t)
	plan := buildPlanFor(t, b6ConfigFor(t, b6Model1), home, false)
	target := filepath.Join(TargetRootForHome(home), b6Target)
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("newer user bytes\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyPlan(plan, home, io.Discard); !errors.Is(err, ErrStalePlan) {
		t.Fatalf("ApplyPlan error = %v, want ErrStalePlan", err)
	}
	if got := readString(t, target); got != "newer user bytes\n" {
		t.Fatalf("newer target was overwritten: %q", got)
	}
	if n := len(backupDirs(t, home)); n != 0 {
		t.Fatalf("refused apply created %d backup(s)", n)
	}
}

func TestB6MetadataFailureAfterSidecarIsIncompleteNotSuccess(t *testing.T) {
	home := newFleetHome(t)
	plan := buildPlanFor(t, b6ConfigFor(t, b6Model1), home, false)
	b6SetSeams(t, b6Seams{metadata: func(step string) error {
		if step == "sidecar" {
			return errors.New("injected sidecar write failure")
		}
		return nil
	}})
	result, err := ApplyPlan(plan, home, io.Discard)
	if !errors.Is(err, ErrMetadataIncomplete) {
		t.Fatalf("ApplyPlan error = %v, want ErrMetadataIncomplete", err)
	}
	if result.BackupPath == "" {
		t.Fatal("incomplete outcome did not report the backup path")
	}
	// Files are committed: the new bytes are installed and nothing was rolled back.
	target := filepath.Join(TargetRootForHome(home), b6Target)
	if got := readString(t, target); !strings.Contains(got, "model: "+b6Model1+"\n") {
		t.Fatalf("committed file missing desired bytes:\n%s", got)
	}
	// The live pointer is written before the sidecar becomes applied, so a
	// sidecar failure leaves a pending transaction that the next mutation finishes.
	if got := b6Sidecar(t, result.BackupPath).State; got != ManifestStateApplying {
		t.Fatalf("sidecar state = %q, want applying", got)
	}
	if !b6LiveExists(t, home) {
		t.Fatal("live pointer must be written before the sidecar is finalized")
	}
}

func TestB6MetadataFailureAfterLiveIsIncomplete(t *testing.T) {
	home := newFleetHome(t)
	plan := buildPlanFor(t, b6ConfigFor(t, b6Model1), home, false)
	b6SetSeams(t, b6Seams{metadata: func(step string) error {
		if step == "live" {
			return errors.New("injected live pointer failure")
		}
		return nil
	}})
	result, err := ApplyPlan(plan, home, io.Discard)
	if !errors.Is(err, ErrMetadataIncomplete) {
		t.Fatalf("ApplyPlan error = %v, want ErrMetadataIncomplete", err)
	}
	// The sidecar stays applying, so the transaction remains pending.
	if got := b6Sidecar(t, result.BackupPath).State; got != ManifestStateApplying {
		t.Fatalf("sidecar state = %q, want applying while the live pointer is missing", got)
	}
	if b6LiveExists(t, home) {
		t.Fatal("live pointer exists after its write failed")
	}
}

func TestB6NextMutationFinishesMissingLivePointer(t *testing.T) {
	home := newFleetHome(t)
	stale := buildPlanFor(t, b6ConfigFor(t, b6Model1), home, false)
	b6SetSeams(t, b6Seams{metadata: func(step string) error {
		if step == "live" {
			return errors.New("injected live pointer failure")
		}
		return nil
	}})
	result, err := ApplyPlan(stale, home, io.Discard)
	if !errors.Is(err, ErrMetadataIncomplete) {
		t.Fatalf("setup: ApplyPlan error = %v, want ErrMetadataIncomplete", err)
	}
	b6SetSeams(t, b6Seams{})
	if _, err := ApplyPlan(stale, home, io.Discard); !errors.Is(err, ErrStalePlan) {
		t.Fatalf("stale plan after reconciliation = %v, want ErrStalePlan", err)
	}
	if got := b6Sidecar(t, result.BackupPath).State; got != ManifestStateApplied {
		t.Fatalf("reconciled sidecar state = %q, want applied", got)
	}
	if !b6LiveExists(t, home) {
		t.Fatal("reconciliation did not write the missing live pointer")
	}
}

func TestB6NextMutationReconcilesCommittedTransaction(t *testing.T) {
	home := newFleetHome(t)
	stale := buildPlanFor(t, b6ConfigFor(t, b6Model1), home, false)
	b6SetSeams(t, b6Seams{metadata: func(step string) error {
		if step == "sidecar" {
			return errors.New("injected sidecar write failure")
		}
		return nil
	}})
	result, err := ApplyPlan(stale, home, io.Discard)
	if !errors.Is(err, ErrMetadataIncomplete) {
		t.Fatalf("setup: ApplyPlan error = %v, want ErrMetadataIncomplete", err)
	}
	b6SetSeams(t, b6Seams{})

	// Any authorized mutation reconciles first. The stale plan is then refused
	// by its own compare-and-swap, not by the interrupted state.
	if _, err := ApplyPlan(stale, home, io.Discard); !errors.Is(err, ErrStalePlan) {
		t.Fatalf("stale plan after reconciliation = %v, want ErrStalePlan", err)
	}
	m := b6Sidecar(t, result.BackupPath)
	if m.State != ManifestStateApplied {
		t.Fatalf("reconciled sidecar state = %q, want applied", m.State)
	}
	target := filepath.Join(TargetRootForHome(home), b6Target)
	digest, _, err := onDiskDigest(target)
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := m.RecordFor(target); !ok || r.InstalledDigest != digest {
		t.Fatalf("reconciled record does not describe the installed bytes: %+v", r)
	}
	if !b6LiveExists(t, home) {
		t.Fatal("reconciliation did not finalize the live pointer")
	}
	fresh := buildPlanFor(t, b6ConfigFor(t, b6Model1), home, false)
	if len(fresh.ProposedOps) != 1 || fresh.ProposedOps[0].Op != "noop" {
		t.Fatalf("fresh plan after reconciliation = %+v, want noop", fresh.ProposedOps)
	}
}

func TestB6NewerBytesBlockReconciliationWithoutOverwrite(t *testing.T) {
	home := newFleetHome(t)
	stale := buildPlanFor(t, b6ConfigFor(t, b6Model1), home, false)
	b6SetSeams(t, b6Seams{metadata: func(step string) error {
		if step == "sidecar" {
			return errors.New("injected sidecar write failure")
		}
		return nil
	}})
	result, err := ApplyPlan(stale, home, io.Discard)
	if !errors.Is(err, ErrMetadataIncomplete) {
		t.Fatalf("setup: ApplyPlan error = %v", err)
	}
	b6SetSeams(t, b6Seams{})
	liveBefore := readString(t, LivePointerPath(home))
	target := filepath.Join(TargetRootForHome(home), b6Target)
	newer := []byte("newer user bytes\n")
	if err := os.WriteFile(target, newer, 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := ApplyPlan(stale, home, io.Discard); !errors.Is(err, ErrPendingReconciliation) {
		t.Fatalf("ApplyPlan error = %v, want ErrPendingReconciliation", err)
	} else if !strings.Contains(err.Error(), ConflictRecoveryHint) {
		t.Fatalf("ApplyPlan error lacks the recovery hint: %v", err)
	}
	if _, err := RestoreBackup(home, filepath.Base(result.BackupPath), io.Discard); !errors.Is(err, ErrPendingReconciliation) {
		t.Fatalf("RestoreBackup error = %v, want ErrPendingReconciliation", err)
	} else if !strings.Contains(err.Error(), ConflictRecoveryHint) {
		t.Fatalf("RestoreBackup error lacks the recovery hint: %v", err)
	}
	if got := readString(t, target); got != string(newer) {
		t.Fatalf("newer bytes were overwritten: %q", got)
	}
	if got := b6Sidecar(t, result.BackupPath).State; got != ManifestStateApplying {
		t.Fatalf("conflicting sidecar was finalized: %q", got)
	}
	if got := readString(t, LivePointerPath(home)); got != liveBefore {
		t.Fatal("live pointer changed while the conflict was unreconciled")
	}
}

func TestB6StatusReportsPendingWithoutRepair(t *testing.T) {
	t.Run("committed transaction is reported and not finalized", func(t *testing.T) {
		home := newFleetHome(t)
		config := b6ConfigFor(t, b6Model1)
		b6SetSeams(t, b6Seams{metadata: func(step string) error {
			if step == "sidecar" {
				return errors.New("injected sidecar write failure")
			}
			return nil
		}})
		result, err := ApplyPlan(buildPlanFor(t, config, home, false), home, io.Discard)
		if !errors.Is(err, ErrMetadataIncomplete) {
			t.Fatalf("setup: %v", err)
		}
		b6SetSeams(t, b6Seams{})
		liveBefore := readString(t, LivePointerPath(home))
		status, err := BuildStatus(config, "", home)
		if err != nil {
			t.Fatal(err)
		}
		id := filepath.Base(result.BackupPath)
		if len(status.PendingTransactions) != 1 || status.PendingTransactions[0] != (PendingTransaction{BackupID: id, Outcome: "committed"}) {
			t.Fatalf("pending transactions = %+v, want one committed entry for %s", status.PendingTransactions, id)
		}
		if got := b6Sidecar(t, result.BackupPath).State; got != ManifestStateApplying {
			t.Fatalf("status repaired the sidecar to %q", got)
		}
		if got := readString(t, LivePointerPath(home)); got != liveBefore {
			t.Fatal("status rewrote the live pointer")
		}
	})

	t.Run("newer bytes are reported as a conflict and not finalized", func(t *testing.T) {
		home := newFleetHome(t)
		config := b6ConfigFor(t, b6Model1)
		b6SetSeams(t, b6Seams{metadata: func(step string) error {
			if step == "sidecar" {
				return errors.New("injected sidecar write failure")
			}
			return nil
		}})
		result, err := ApplyPlan(buildPlanFor(t, config, home, false), home, io.Discard)
		if !errors.Is(err, ErrMetadataIncomplete) {
			t.Fatalf("setup: %v", err)
		}
		b6SetSeams(t, b6Seams{})
		target := filepath.Join(TargetRootForHome(home), b6Target)
		if err := os.WriteFile(target, []byte("newer\n"), 0644); err != nil {
			t.Fatal(err)
		}
		status, err := BuildStatus(config, "", home)
		if err != nil {
			t.Fatal(err)
		}
		if len(status.PendingTransactions) != 1 || status.PendingTransactions[0].Outcome != "conflict" {
			t.Fatalf("pending transactions = %+v, want one conflict", status.PendingTransactions)
		}
		if got := b6Sidecar(t, result.BackupPath).State; got != ManifestStateApplying {
			t.Fatalf("status changed the sidecar to %q", got)
		}
		if got := readString(t, target); got != "newer\n" {
			t.Fatalf("status modified the target: %q", got)
		}
	})
}

func TestB6LockDirectorySurvivesRestoreAndCleanup(t *testing.T) {
	home := newFleetHome(t)
	b6Create(t, home)
	applyPlan(t, b6Update(t, home), home)
	if _, err := RestoreBackup(home, filepath.Base(latestBackup(t, home)), io.Discard); err != nil {
		t.Fatal(err)
	}
	// Restore removes or rewrites only managed profile files; the guard
	// directory under <home>/.agentic-sdd/locks must survive with its leaves.
	for _, leaf := range []string{"agent-target.lock"} {
		if _, err := os.Lstat(filepath.Join(home, ".agentic-sdd", "locks", leaf)); err != nil {
			t.Fatalf("lock leaf %s was removed: %v", leaf, err)
		}
	}
}
