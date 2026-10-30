"""Trusted deterministic CI adapter. No model runtime or approval ledger."""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import re
import signal
import stat
import subprocess
import sys
import tempfile


IMAGE = "golang@sha256:1ae0735f00daffa3aaf1363a5184c0d2dc55c78e3db4ec70241cdac97bf84b59"
REQUIRED = ["mod-verify", "tidy-check", "vet", "test", "build-darwin", "diff-check"]
EXCLUDED = {".cache", ".codegraph", "backups", "bin", "__pycache__"}
SENSITIVE_DIRS = {".local", ".config", ".opencode", ".claude", ".codex", ".agents", ".ssh", ".aws", ".azure", ".kube", ".gnupg", ".secrets"}
SENSITIVE_FILES = {"auth.json", "credentials.json", "tokens.json", ".netrc", ".git-credentials", ".npmrc", ".pypirc", ".envrc"}
PROBE = "internal/skillsync/zz_sdd_trusted_noop_test.go"


def digest(data):
    return hashlib.sha256(data).hexdigest()


def load_json(path):
    def unique(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError("duplicate JSON key")
            result[key] = value
        return result
    if path.stat().st_size > 1 << 20:
        raise ValueError("JSON exceeds bound")
    return json.loads(path.read_text(), object_pairs_hook=unique)


def required_contract(trusted, candidate):
    for contract in (trusted, candidate):
        if contract.get("version") != 1 or not isinstance(contract.get("checks"), list):
            raise ValueError("missing versioned check contract")
        ids = [ch["id"] for ch in contract["checks"]]
        if len(ids) != len(set(ids)):
            raise ValueError("duplicate contract ID")
    selected = [ch for ch in trusted["checks"] if "required" in ch["scopes"]]
    proposed = [ch for ch in candidate["checks"] if "required" in ch["scopes"]]
    if [ch["id"] for ch in selected] != REQUIRED or selected != proposed:
        raise ValueError("required contract removed/replaced; reviewed adoption required")
    if any(ch.get("cwd") != "." for ch in selected):
        raise ValueError("adapter supports reviewed repository-root checks only")
    if selected[-1]["steps"] != [{"argv": ["git", "diff", "--check"], "env": {}}]:
        raise ValueError("unsupported diff-check recipe; reviewed adapter update required")
    return copy.deepcopy(selected)


def review_paths(before, after):
    return sorted(path for path in before.keys() | after.keys() if before.get(path) != after.get(path) and (
        path == "Makefile" or path.startswith((".github/workflows/", "harness/checks", "cmd/sdd-check/", "internal/harnesscheck/", "tests/"))
        or path.endswith("_test.go")))


def execution_outcome(returncode, result, stdout_result):
    if stdout_result != result or [ch.get("id") for ch in result.get("checks", [])] != REQUIRED:
        return "incomplete"
    execution = result.get("execution")
    if execution in ("failed", "blocked", "cancelled", "incomplete"):
        return execution if returncode != 0 else "incomplete"
    if returncode != 0 or execution != "completed":
        return "incomplete"
    if any(ch.get("status") != "passed" or ch.get("exit_code") != 0 or ch.get("truncated") is not False for ch in result["checks"]):
        return "incomplete"
    return "completed"


def observed_revision(root, env):
    try:
        top = subprocess.check_output(["git", "--no-pager", "-c", "core.fsmonitor=false", "-C", str(root),
                                       "rev-parse", "--show-toplevel"], env=env, stderr=subprocess.DEVNULL, timeout=10)
        if Path(os.fsdecode(top).strip()).resolve() != root.resolve():
            return None
        head = subprocess.check_output(["git", "--no-pager", "-c", "core.fsmonitor=false", "-C", str(root),
                                        "rev-parse", "--verify", "HEAD"], env=env, stderr=subprocess.DEVNULL, timeout=10).decode().strip()
        return head if re.fullmatch(r"[0-9a-f]{40}|[0-9a-f]{64}", head) else None
    except (OSError, ValueError, subprocess.SubprocessError):
        return None


def inventory(root, input_tree=False):
    if root.name in SENSITIVE_DIRS:
        raise ValueError("credential/runtime directory cannot be a source root")
    records = {}
    total = 0
    for directory, dirs, files in os.walk(root, followlinks=False):
        base = Path(directory)
        for name in dirs[:]:
            path = base / name
            if (base == root and name in EXCLUDED) or name == "__pycache__":
                dirs.remove(name)
                continue
            if name == ".git":
                if not input_tree:
                    raise ValueError("candidate introduced Git metadata")
                dirs.remove(name)
                continue
            if name in SENSITIVE_DIRS:
                raise ValueError("credential/runtime source directory refused before reading")
            if path.is_symlink():
                raise ValueError("source directory symlink refused")
        for name in files:
            if base == root and (name in EXCLUDED or name == ".git" and input_tree):
                continue
            if (name in SENSITIVE_FILES or name == ".env" or name.startswith(".env.")
                    or Path(name).suffix.lower() in (".db", ".sqlite", ".sqlite3")):
                raise ValueError("credential/runtime source file refused before reading")
            path = base / name
            info = path.lstat()
            if not stat.S_ISREG(info.st_mode) or info.st_size > 32 << 20:
                raise ValueError("source requires bounded regular files")
            data = path.read_bytes()
            total += len(data)
            if total > 128 << 20 or len(records) >= 10000:
                raise ValueError("source tree exceeds bound")
            records[path.relative_to(root).as_posix()] = {"sha256": digest(data), "executable": bool(info.st_mode & 0o111)}
    return records


def stage_tree(source, destination):
    source, destination = Path(source), Path(destination)
    records = inventory(source, input_tree=True)
    destination.mkdir(mode=0o700)
    for name, observed in records.items():
        path = destination / name
        path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        data = (source / name).read_bytes()
        if digest(data) != observed["sha256"]:
            raise ValueError("source changed while staging")
        path.write_bytes(data)
        path.chmod(0o755 if observed["executable"] else 0o644)
    return records


def container_argv(workspace, cache, cidfile, argv, env, readonly=False):
    if not argv or argv[0] != "go" or set(env) - {"GOOS", "GOARCH"}:
        raise ValueError("only trusted Go steps and explicit target overrides supported")
    if any("," in str(path) or not Path(path).is_absolute() for path in (workspace, cache, cidfile)):
        raise ValueError("unsafe mount/cid path")
    result = ["docker", "run", "--rm", "--pull=never", "--network=none", "--read-only", "--cap-drop=ALL",
              "--security-opt=no-new-privileges", "--pids-limit=256", "--memory=2g", "--cpus=2",
              f"--user={os.getuid()}:{os.getgid()}", "--cidfile", str(cidfile),
              "--tmpfs", "/tmp:rw,exec,nosuid,nodev,mode=1777,size=1g", "--workdir", "/workspace",
              "--mount", f"type=bind,src={workspace},dst=/workspace" + (",readonly" if readonly else ""),
              "--mount", f"type=bind,src={cache},dst=/cache",
              "-e", "HOME=/tmp/home", "-e", "TMPDIR=/tmp", "-e", "GOCACHE=/cache/go-build",
              "-e", "GOMODCACHE=/tmp/go-mod-cache", "-e", "GOPROXY=off", "-e", "GOSUMDB=off",
              "-e", "GOTOOLCHAIN=local", "-e", "GOENV=off", "-e", "CGO_ENABLED=0",
              "-e", "GOMAXPROCS=2", "-e", "GOFLAGS=-buildvcs=false"]
    for key, value in sorted(env.items()):
        result += ["-e", key + "=" + value]
    return result + [IMAGE, *argv]


def safe_environment(home):
    return {"PATH": "/usr/local/bin:/usr/bin:/bin", "HOME": str(home), "LC_ALL": "C",
            "GIT_CONFIG_GLOBAL": os.devnull, "GIT_CONFIG_NOSYSTEM": "1", "GIT_OPTIONAL_LOCKS": "0",
            "GIT_TERMINAL_PROMPT": "0"}


def write_json(path, data):
    with path.open("x") as stream:
        json.dump(data, stream, indent=2, sort_keys=True)
        stream.write("\n")


def cleanup_containers(cidfiles, env):
    uncertain = False
    for path in cidfiles:
        if not path.exists():
            continue
        cid = path.read_text().strip()
        if not re.fullmatch(r"[0-9a-f]{64}", cid):
            uncertain = True
            continue
        try:
            stopped = subprocess.run(["docker", "rm", "--force", cid], env=env, capture_output=True, timeout=10)
            if stopped.returncode and b"No such container" not in stopped.stderr:
                uncertain = True
        except (OSError, subprocess.TimeoutExpired):
            uncertain = True
    return uncertain


def whitespace(trusted, candidate):
    # Data-only Git diff from a trusted cwd; never read candidate Git config,
    # execute external diff/textconv, or follow a program-created symlink.
    inventory(trusted)
    inventory(candidate)
    with tempfile.TemporaryDirectory(dir=trusted.parent) as tmp:
        root = Path(tmp)
        before, after = root / "before", root / "after"
        stage_tree(trusted, before); stage_tree(candidate, after)
        result = subprocess.run(["git", "--no-pager", "-c", "core.fsmonitor=false", "diff", "--no-index",
                                 "--no-ext-diff", "--no-textconv", "--check", str(before), str(after)],
                                cwd=root, env=safe_environment(root), stdout=subprocess.DEVNULL,
                                stderr=subprocess.DEVNULL, timeout=30)
    # --no-index returns 1 for a clean but nonempty patch; --check adds bit 2
    # for whitespace errors. No raw diff is captured or emitted by this helper.
    if result.returncode not in (0, 1):
        print("whitespace observation failed; inspect changed text in the candidate")
        return 1
    return 0


def run_gate(trusted, candidate, output):
    os.umask(0o077)
    trusted, candidate, output = Path(trusted).resolve(strict=True), Path(candidate).resolve(strict=True), Path(output).absolute()
    if trusted == candidate or trusted.is_relative_to(candidate) or candidate.is_relative_to(trusted):
        raise ValueError("separate trusted and candidate directories required")
    if any(output.is_relative_to(root) or root.is_relative_to(output) for root in (trusted, candidate)) or ".." in output.parts:
        raise ValueError("external output must not overlap inputs")
    for path in [output, *output.parents]:
        if path.is_symlink():
            raise ValueError("output symlink refused")
    if output.exists():
        if list(output.iterdir()) or output.stat().st_mode & 0o077:
            raise ValueError("new or empty private output required")
    else:
        output.mkdir(mode=0o700, parents=True)
    private, evidence = output / "private", output / "evidence"
    private.mkdir(mode=0o700); evidence.mkdir(mode=0o700)
    env = safe_environment(private)
    summary = {"version": 1, "execution": "blocked", "reason": "bootstrap not established",
               "phase": "bootstrap-validation",
               "image": IMAGE, "review_required_paths": [], "unknowns": ["native session/model/billing correlation not observed"],
               "trust": "caller-selected reviewed base; no candidate fallback or remote enforcement claim"}
    cidfiles, process = [], None
    try:
        if os.getuid() == 0 or sys.platform != "linux":
            raise ValueError("non-root Linux host required")
        if Path(__file__).resolve() != trusted / "tests/harness/trusted_ci.py":
            raise ValueError("bootstrap launcher must be loaded from the supplied trusted base")
        for project in (trusted, candidate):
            if project == Path.home().resolve() or not (project / "go.mod").is_file() or (project / "go.mod").is_symlink():
                raise ValueError("credential-free project source required, never a home directory")
        for path in ("cmd/sdd-check/main.go", "internal/harnesscheck/runner.go", "harness/checks.json",
                     "tests/harness/trusted-noop-probe.go.txt"):
            if not (trusted / path).is_file() or (trusted / path).is_symlink():
                raise ValueError("trusted base lacks bootstrap/runner/probe; adopt reviewed B7 first")
        base, work = private / "trusted", private / "candidate"
        summary["trusted_revision"] = observed_revision(trusted, env)
        summary["candidate_revision"] = observed_revision(candidate, env)
        if summary["trusted_revision"] is None or summary["candidate_revision"] is None:
            summary["unknowns"].append("one or both inputs are exports without an independently observed Git-root HEAD; manifests identify their bytes")
        base_manifest = stage_tree(trusted, base)
        candidate_manifest = stage_tree(candidate, work)
        summary.update(trusted_input_sha256=digest(json.dumps(base_manifest, sort_keys=True).encode()),
                       candidate_input_sha256=digest(json.dumps(candidate_manifest, sort_keys=True).encode()),
                       review_required_paths=review_paths(base_manifest, candidate_manifest))
        summary["phase"] = "contract-comparison"
        selected = required_contract(load_json(base / "harness/checks.json"), load_json(work / "harness/checks.json"))
        if (work / PROBE).exists():
            raise ValueError("candidate collides with reserved independent probe")
        summary["phase"] = "image-preflight"
        subprocess.run(["docker", "image", "inspect", IMAGE], env=env, check=True, capture_output=True, timeout=15)
        cache = private / "bootstrap-cache"; cache.mkdir(mode=0o700)
        cid = private / "bootstrap.cid"; cidfiles.append(cid)
        summary["phase"] = "runner-build"
        bootstrap = subprocess.run(container_argv(base, cache, cid,
            ["go", "build", "-trimpath", "-o", "/cache/sdd-check", "./cmd/sdd-check"], {}, readonly=True),
            env=env, capture_output=True, timeout=120)
        (evidence / "bootstrap.log").write_bytes(bootstrap.stdout + bootstrap.stderr)
        binary = cache / "sdd-check"
        if bootstrap.returncode or not binary.is_file() or binary.is_symlink():
            raise ValueError("trusted runner bootstrap did not complete")
        summary["runner_sha256"] = digest(binary.read_bytes())
        probe = (base / "tests/harness/trusted-noop-probe.go.txt").read_bytes()
        (work / PROBE).write_bytes(probe)
        summary["probe_sha256"] = digest(probe)
        initial = inventory(work)
        for ch in selected:
            ch["cwd"] = "."
            if ch["id"] == "diff-check":
                ch["prerequisites"] = ["git", "python3"]
                ch["steps"] = [{"argv": [sys.executable, str(base / "tests/harness/trusted_ci.py"), "whitespace",
                                         "--trusted", str(base), "--candidate", str(work)], "env": {}}]
                continue
            ch["prerequisites"] = ["docker"]
            cache = private / (ch["id"] + "-cache"); cache.mkdir(mode=0o700)
            for i, step in enumerate(ch["steps"]):
                cid = private / f"{ch['id']}-{i}.cid"; cidfiles.append(cid)
                step["argv"] = container_argv(work, cache, cid, step["argv"], step["env"])
                step["env"] = {}
        translated = private / "isolated-checks.json"
        write_json(translated, {"version": 1, "checks": selected})
        result_dir = evidence / "checks"
        summary["phase"] = "check-execution"
        process = subprocess.Popen([str(binary), "--scope", "required", "--workspace", str(work),
                                    "--contract", str(translated), "--output", str(result_dir), "--deadline", "15m"],
                                   cwd=private, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        stdout, stderr = process.communicate(timeout=930)
        result = load_json(result_dir / "result.json")
        summary["check_result"] = result
        summary["canonical_contract_sha256"] = digest((base / "harness/checks.json").read_bytes())
        summary["isolated_contract_sha256"] = digest(translated.read_bytes())
        summary["phase"] = "freshness"
        if inventory(work) != initial:
            raise ValueError("candidate inputs changed during checks; results not fresh")
        try:
            stdout_result = json.loads(stdout)
        except ValueError:
            stdout_result = None
        observed = execution_outcome(process.returncode, result, stdout_result)
        if observed != "completed":
            summary.update(execution=observed, reason="independent required checks did not all pass")
        elif any(ch["status"] != "passed" or ch["truncated"] for ch in result["checks"]):
            raise ValueError("required evidence is incomplete")
        else:
            summary.update(execution="completed", reason="checks passed; gate/test changes still require human review, not Ready or approval")
        summary["log_digests"] = {ch["id"]: digest(Path(ch["evidence"]).read_bytes()) for ch in result["checks"] if ch["evidence"]}
        summary["phase"] = "complete"
    except KeyboardInterrupt:
        summary.update(execution="cancelled", reason="owned gate interrupted; partial evidence retained")
    except subprocess.TimeoutExpired:
        summary.update(execution="incomplete", reason="bounded bootstrap/check operation timed out")
    except (ValueError, OSError, subprocess.SubprocessError, KeyError, TypeError):
        summary.update(execution="incomplete" if summary["phase"] in ("check-execution", "freshness") else "blocked",
                       reason="invalid/unavailable bootstrap, input, contract or fresh evidence; no fallback")
    finally:
        if process is not None and process.poll() is None:
            process.terminate()
            try:
                process.communicate(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill(); process.communicate(timeout=5)
        if cleanup_containers(cidfiles, env):
            summary.update(execution="incomplete", reason="owned-container cleanup uncertain; reconcile before retry")
        write_json(evidence / "gate.json", summary)
    print(json.dumps({"execution": summary["execution"], "evidence": str(evidence / "gate.json")}))
    return 0 if summary["execution"] == "completed" else 2 if summary["execution"] == "blocked" else 1


def main():
    def cancel(_number, _frame):
        raise KeyboardInterrupt
    signal.signal(signal.SIGTERM, cancel)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("run", "whitespace"))
    parser.add_argument("--trusted", required=True)
    parser.add_argument("--candidate", required=True)
    parser.add_argument("--output")
    args = parser.parse_args()
    if args.action == "whitespace":
        return whitespace(Path(args.trusted), Path(args.candidate))
    if not args.output:
        parser.error("run requires external --output")
    try:
        return run_gate(args.trusted, args.candidate, args.output)
    except (ValueError, OSError):
        print("invalid gate input; no work authorized", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
