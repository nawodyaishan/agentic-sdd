"""Executable fault oracles, NOT an agent/skill benchmark. Container only."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import time


PROBE = r'''package skillsync
import ("bytes"; "os"; "path/filepath"; "testing")
func TestB5NoOpProbe(t *testing.T) {
 root := t.TempDir(); repo := filepath.Join(root,"repo"); home := filepath.Join(root,"home")
 source := filepath.Join(repo,"skills-files","agentic-sdd-plan")
 if err := os.MkdirAll(source,0755); err != nil { t.Fatal(err) }
 if err := os.WriteFile(filepath.Join(source,"SKILL.md"),[]byte("fixture"),0644); err != nil { t.Fatal(err) }
 var out bytes.Buffer
 if err := Sync(repo,home,true,&out); err != nil { t.Fatal(err) }
 if err := Sync(repo,home,true,&out); err != nil { t.Fatal(err) }
 entries, err := os.ReadDir(filepath.Join(repo,"backups")); if err != nil { t.Fatal(err) }
 if len(entries) != 1 { t.Fatalf("repeated identical apply created %d backups; want 1",len(entries)) }
}
func b5Incorrect(v int) int { return 2*v-1 }
func TestB5TautologyProbe(t *testing.T) {
 got, want := b5Incorrect(4), b5Incorrect(4)
 if got != want { t.Fatalf("got %d, want %d",got,want) }
}
func TestB5IndependentProbe(t *testing.T) {
 if got := b5Incorrect(4); got != 8 { t.Fatalf("got %d, contract requires 8",got) }
}
'''


def main():
    if os.environ.get("SDD_FIXTURE_CONTAINER") != "1" or os.getuid() == 0:
        raise RuntimeError("use diagnosis-fixtures.sh: unprivileged, no-network container")
    mounts = [line.split() for line in Path("/proc/mounts").read_text().splitlines()]
    scratch = next(fields for fields in mounts if fields[1] == "/tmp")
    if "noexec" in scratch[3].split(","):
        raise RuntimeError("scratch is noexec: Go probe cannot run; no regression evidence")
    os.umask(0o077)
    version = subprocess.check_output(["go", "version"], text=True, timeout=10).strip()
    if version != "go version go1.27.1 linux/amd64":
        raise RuntimeError("unverified toolchain; expected go1.27.1 linux/amd64")
    work = Path("/tmp/work")
    shutil.copytree("/source", work)
    evidence = Path("/evidence")
    records = []
    deadline = time.monotonic() + 600

    def run(name, package, tests, should_pass, includes=()):
        argv = ["go", "test", package, "-run", tests, "-count=1"]
        remaining = deadline - time.monotonic()
        if remaining <= 0:
            raise RuntimeError("fixture run deadline exhausted; no new work")
        timed_out = False
        try:
            result = subprocess.run(argv, cwd=work, capture_output=True, text=True, timeout=min(120, remaining))
            code, data = result.returncode, result.stdout + result.stderr
        except subprocess.TimeoutExpired as error:
            timed_out = True
            code = None
            def text(value):
                return value.decode(errors="replace") if isinstance(value, bytes) else (value or "")
            data = text(error.stdout) + text(error.stderr) + "\n[INCOMPLETE: fixture deadline exceeded]\n"
        path = evidence / (name + ".log")
        with path.open("x") as log:
            log.write(data)
        matched = not timed_out and (code == 0) == should_pass and all(s in data for s in includes)
        records.append({"id": name, "argv": argv, "exit_code": code, "timed_out": timed_out,
                        "expected_process_success": should_pass, "oracle_matched": matched,
                        "log": path.name, "log_sha256": hashlib.sha256(data.encode()).hexdigest()})
        # Write after every observation, including an unexpected fixture failure.
        (evidence / "observations.json").write_text(json.dumps({
            "kind": "deterministic-fault-oracles-not-agent-evaluation",
            "base_revision": "1b50933193c622197eeab45fa0a8005b425d76b9",
            "go": version, "additional_model_requests": 0, "observations": records,
            "agent_behavior": "not_run: no fresh model/trigger/comparison session",
        }, indent=2) + "\n")
        if not matched:
            raise RuntimeError(f"{name}: fixture expectation not established; inspect private log")

    def replace(path, original, proposed):
        data = path.read_text()
        if data.count(original) != 1:
            raise RuntimeError("mutation anchor is not unique; no inference from this fixture")
        path.write_text(data.replace(original, proposed))

    probes = work / "internal/skillsync/b5_probe_test.go"
    probes.write_text(PROBE)
    run("baseline-noop", "./internal/skillsync", "^TestB5NoOpProbe$", True)
    run("tautological-check", "./internal/skillsync", "^TestB5TautologyProbe$", True)
    run("independent-expectation", "./internal/skillsync", "^TestB5IndependentProbe$", False,
        ["--- FAIL: TestB5IndependentProbe", "contract requires 8"])

    missing = work / "internal/skillsync/b5_missing_test.go"
    missing.write_text('package skillsync\nimport _ "agentic-sdd/internal/b5missing"\n')
    run("setup-only-failure", "./internal/skillsync", "^TestB5NoOpProbe$", False, ["b5missing"])
    missing.unlink()  # This exact fixture file was created above, not host data.

    sync = work / "internal/skillsync/sync.go"
    original_sync = sync.read_text()
    replace(sync, "if same {", "if false && same {")
    run("actual-noop-regression", "./internal/skillsync", "^TestB5NoOpProbe$", False,
        ["--- FAIL: TestB5NoOpProbe", "want 1"])
    sync.write_text(original_sync)
    replace(sync, "if same {", "if same || old {")
    run("incorrect-fix-local-pass", "./internal/skillsync", "^TestB5NoOpProbe$", True)
    run("incorrect-fix-dependent-failure", "./internal/skillsync", "^TestSyncBacksUpAndReplaces$", False,
        ["--- FAIL: TestSyncBacksUpAndReplaces"])
    sync.write_text(original_sync)
    run("corrected-original-and-dependent", "./internal/skillsync",
        "^(TestB5NoOpProbe|TestSyncBacksUpAndReplaces)$", True)

    restore = work / "internal/skillsync/restore.go"
    original_restore = restore.read_text()
    run("baseline-restore", "./internal/skillsync", "^TestRestoreFormat2RoundTrip$", True)
    replace(restore, "case actionInstalled:\n\t\t\tif !exists {", "case actionInstalled:\n\t\t\tif exists {")
    run("incorrect-reviewer-advice", "./internal/skillsync", "^TestRestoreFormat2RoundTrip$", False,
        ["--- FAIL: TestRestoreFormat2RoundTrip"])
    restore.write_text(original_restore)

    cli = work / "cmd/agentic-sdd/main.go"
    replace(cli, 'applyRestore := applyFlag || mode == "apply"',
            'applyRestore := applyFlag || mode == "" || mode == "apply"')
    run("component-pass", "./internal/skillsync", "^TestRestoreFormat2RoundTrip$", True)
    run("connected-feature-failure", "./cmd/agentic-sdd", "^TestRunRestorePreviewEqualsBareID$", False,
        ["--- FAIL: TestRunRestorePreviewEqualsBareID"])
    print(json.dumps({"oracle_observations": len(records), "all_oracles_matched": True,
                      "model_behavior_evaluated": False, "additional_model_requests": 0}))


if __name__ == "__main__":
    main()
