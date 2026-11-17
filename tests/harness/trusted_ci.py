"""Trusted deterministic CI adapter. No model runtime or approval ledger."""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import pty
import re
import select
import shutil
import signal
import stat
import struct
import subprocess
import sys
import tarfile
import tempfile
import time


IMAGE = "golang@sha256:1ae0735f00daffa3aaf1363a5184c0d2dc55c78e3db4ec70241cdac97bf84b59"
REQUIRED = ["mod-verify", "tidy-check", "vet", "test", "build-darwin", "diff-check"]
EXCLUDED = {".cache", ".codegraph", "backups", "bin", "dist", "__pycache__"}
EXCLUDED_PATHS = {
    "website/node_modules",
    "website/.astro",
    "website/.cache",
    "website/dist",
    "website/.export-check",
    "website/coverage",
}
QUALIFICATION_CONTRACT_PATH = "harness/qualification.json"
AUDITED_PYTHON_SUITES = (
    "test_core_workflow.py",
    "test_research_workflow.py",
    "test_supported_core_go.py",
    "test_appops_core_selection.py",
    "test_trusted_ci.py",
    "test_trusted_ci_container.py",
)
ALLOWED_QUALIFICATION_PREREQS = {
    "python3",
    "task",
    "docker",
    "running Docker daemon",
    "preloaded_go_image",
    "git",
}
SHIPPED_APPLICATIONS = ("asdx", "asdx-core", "agentic-sdd", "agentic-sdd-tui")
DARWIN_ARCHES = ("amd64", "arm64")
DARWIN_BUILD_IDS = ("cli-core", "cli-legacy", "tui-canonical", "tui-legacy")
NESTED_TUI_CHECKS = (
    ("tui-vendor-check", ["go", "list", "-mod=vendor", "./..."]),
    ("tui-vet", ["go", "vet", "-mod=vendor", "./..."]),
    ("tui-test", ["go", "test", "-mod=vendor", "-count=1", "-timeout=15m", "./..."]),
)
MACHO_CPUTYPES = {0x01000007: "amd64", 0x0100000C: "arm64"}
SENSITIVE_DIRS = {".local", ".config", ".opencode", ".claude", ".codex", ".agents", ".ssh", ".aws", ".azure", ".kube", ".gnupg", ".secrets"}
SENSITIVE_FILES = {"auth.json", "credentials.json", "tokens.json", ".netrc", ".git-credentials", ".npmrc", ".pypirc", ".envrc"}
PROBES = (
    ("tests/harness/trusted-noop-probe.go.txt", "internal/skillsync/zz_sdd_trusted_noop_test.go"),
    ("tests/harness/trusted-boundary-probe.go.txt", "cmd/agentic-sdd/zz_sdd_trusted_boundary_test.go"),
)
PROBE = PROBES[0][1]


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
        path in ("Makefile", "Taskfile.yml", "Taskfile.yaml") or path.startswith((".github/workflows/", "harness/checks", "harness/qualification", "cmd/sdd-check/", "internal/harnesscheck/", "tests/"))
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
            if path.is_symlink():
                raise ValueError("source directory symlink refused")
            rel = path.relative_to(root).as_posix()
            if (base == root and name in EXCLUDED) or name == "__pycache__" or rel in EXCLUDED_PATHS:
                dirs.remove(name)
                continue
            if name == ".git":
                if not input_tree:
                    raise ValueError("candidate introduced Git metadata")
                dirs.remove(name)
                continue
            if name in SENSITIVE_DIRS:
                raise ValueError("credential/runtime source directory refused before reading")
        for name in files:
            path = base / name
            if path.is_symlink():
                raise ValueError("source requires bounded regular files")
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
                     "tests/harness/trusted-noop-probe.go.txt", "tests/harness/trusted-boundary-probe.go.txt"):
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
        for _, probe_target in PROBES:
            if (work / probe_target).exists():
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
        probe_digests = {}
        for probe_src, probe_target in PROBES:
            probe_bytes = (base / probe_src).read_bytes()
            target_path = work / probe_target
            target_path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
            target_path.write_bytes(probe_bytes)
            probe_digests[probe_target] = digest(probe_bytes)
        summary["probe_sha256"] = probe_digests
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


def qualification_contract(trusted, candidate):
    for contract in (trusted, candidate):
        if not isinstance(contract, dict) or contract.get("version") != 1 or contract.get("kind") != "qualification-contract":
            raise ValueError("missing versioned qualification contract")
        if contract.get("core_contract") != "harness/checks.json":
            raise ValueError("qualification contract must reference harness/checks.json")
    if trusted != candidate:
        raise ValueError("qualification contract modified; reviewed adoption required")

    lanes = trusted.get("lanes")
    if not isinstance(lanes, dict) or set(lanes) != {"core", "nested_tui", "python_suites", "archive_distribution"}:
        raise ValueError("invalid qualification lane inventory")

    core = lanes["core"]
    if core.get("status") != "active" or core.get("contract") != "harness/checks.json" or core.get("scope") != "required":
        raise ValueError("invalid core lane reference in qualification contract")

    nested_tui = lanes["nested_tui"]
    if not isinstance(nested_tui, dict):
        raise ValueError("invalid nested_tui lane")
    if nested_tui.get("cwd") != "experiments/tui":
        raise ValueError("nested_tui cwd must remain experiments/tui")
    tui_status = nested_tui.get("status")
    if tui_status == "blocked":
        if not isinstance(nested_tui.get("blocker"), str) or not nested_tui["blocker"].strip():
            raise ValueError("nested_tui blocked lane requires non-empty blocker")
    elif tui_status == "active":
        if "blocker" in nested_tui:
            raise ValueError("active nested_tui lane must not declare a blocker")
        tui_checks = nested_tui.get("checks")
        if not isinstance(tui_checks, list) or len(tui_checks) != len(NESTED_TUI_CHECKS):
            raise ValueError("active nested_tui lane requires reviewed -mod=vendor checks")
        for ch, (exp_id, exp_argv) in zip(tui_checks, NESTED_TUI_CHECKS):
            if not isinstance(ch, dict) or set(ch) != {"id", "argv"}:
                raise ValueError("invalid nested_tui check entry")
            if ch.get("id") != exp_id or ch.get("argv") != exp_argv:
                raise ValueError(f"nested_tui check {exp_id} must match reviewed -mod=vendor command")
    else:
        raise ValueError("nested_tui lane status must be active or blocked")

    py_lane = lanes["python_suites"]
    if py_lane.get("status") != "active" or py_lane.get("discovery_policy") != "individually-named-audited-suites-only":
        raise ValueError("python_suites lane must forbid broad discovery")
    suites = py_lane.get("suites")
    if not isinstance(suites, list) or not suites:
        raise ValueError("python_suites requires explicit suites")
    seen_ids = set()
    seen_files = []
    for suite in suites:
        if not isinstance(suite, dict):
            raise ValueError("invalid python suite entry")
        sid = suite.get("id", "")
        if not re.fullmatch(r"[a-z][a-z0-9-]+", sid) or sid in seen_ids:
            raise ValueError("invalid or duplicate python suite ID")
        seen_ids.add(sid)
        suite_path = suite.get("path", "")
        suite_file = Path(suite_path).name
        if suite_file not in AUDITED_PYTHON_SUITES or suite_path != f"tests/harness/{suite_file}":
            raise ValueError(f"unaudited or invalid python suite path: {suite_path}")
        if suite_file in seen_files:
            raise ValueError("duplicate python suite file")
        seen_files.append(suite_file)
        if suite.get("cwd") != ".":
            raise ValueError("python suite cwd must be repository root")
        expected_argv = ["python3", "-B", "-m", "unittest", "discover", "-s", "tests/harness", "-p", suite_file]
        if suite.get("argv") != expected_argv:
            raise ValueError(f"python suite {sid} argv must select only {suite_file} without broad discovery")
        timeout = suite.get("timeout_seconds")
        if not isinstance(timeout, int) or isinstance(timeout, bool) or not 1 <= timeout <= 3600:
            raise ValueError(f"invalid timeout for suite {sid}")
        prereqs = suite.get("prerequisites")
        if not isinstance(prereqs, list) or not prereqs or any(p not in ALLOWED_QUALIFICATION_PREREQS for p in prereqs):
            raise ValueError(f"invalid prerequisites for suite {sid}")
        env = suite.get("env")
        if not isinstance(env, dict) or any(not isinstance(k, str) or not isinstance(v, str) for k, v in env.items()):
            raise ValueError(f"invalid env for suite {sid}")
        allowed_env = {"SDD_TRUSTED_CI_CONTAINERS": "1"} if suite_file == "test_trusted_ci_container.py" else {}
        if env != allowed_env:
            raise ValueError(f"unreviewed env override in python suite {sid}")
    if set(seen_files) != set(AUDITED_PYTHON_SUITES):
        raise ValueError("python_suites must select every audited supported-core suite")

    arch = lanes["archive_distribution"]
    if arch.get("status") != "active":
        raise ValueError("archive_distribution lane must be active")
    if arch.get("shipped_applications") != list(SHIPPED_APPLICATIONS):
        raise ValueError("shipped_applications must match the four shipped command names")
    if arch.get("sdd_check_counted_as_shipped") is not False:
        raise ValueError("sdd-check must not be counted as a shipped application")
    if arch.get("darwin_architectures") != list(DARWIN_ARCHES):
        raise ValueError("darwin_architectures must cover amd64 and arm64")
    if arch.get("checksum_file") != "checksums.txt" or arch.get("protocol_revision") != 2 or arch.get("distribution_marker") != "generic":
        raise ValueError("invalid archive_distribution contract constants")
    builds = arch.get("darwin_builds")
    if not isinstance(builds, list) or [b.get("id") for b in builds] != list(DARWIN_BUILD_IDS):
        raise ValueError("darwin_builds must cover all four build IDs")
    for b in builds:
        if b.get("binary") not in SHIPPED_APPLICATIONS or b.get("arches") != list(DARWIN_ARCHES):
            raise ValueError("invalid darwin_builds entry")
    return copy.deepcopy(trusted)


def inspect_macho_arches(data):
    if len(data) < 28:
        raise ValueError("binary too small for Mach-O header")
    magic_be = struct.unpack(">I", data[:4])[0]
    if magic_be in (0xCAFEBABE, 0xCAFEBABF):
        is_64 = magic_be == 0xCAFEBABF
        nfat = struct.unpack(">I", data[4:8])[0]
        if not 1 <= nfat <= 16:
            raise ValueError("invalid fat Mach-O slice count")
        entry_size = 32 if is_64 else 20
        header_end = 8 + nfat * entry_size
        if header_end > len(data):
            raise ValueError("truncated fat Mach-O header")
        arches = set()
        for idx in range(nfat):
            pos = 8 + idx * entry_size
            if is_64:
                cputype, _sub, offset, size, _align, _res = struct.unpack(">IIQQII", data[pos:pos + 32])
            else:
                cputype, _sub, offset, size, _align = struct.unpack(">IIIII", data[pos:pos + 20])
            if offset < header_end or size < 28 or offset + size > len(data):
                raise ValueError("invalid fat Mach-O slice bounds")
            slice_magic, slice_cpu = struct.unpack("<II", data[offset:offset + 8])
            if slice_magic != 0xFEEDFACF or slice_cpu != cputype:
                raise ValueError("fat Mach-O slice header mismatch")
            if cputype not in MACHO_CPUTYPES:
                raise ValueError(f"unsupported Mach-O cputype {cputype:#x}")
            arches.add(MACHO_CPUTYPES[cputype])
        return sorted(arches)
    magic_le, cputype = struct.unpack("<II", data[:8])
    if magic_le == 0xFEEDFACF:
        if cputype not in MACHO_CPUTYPES:
            raise ValueError(f"unsupported Mach-O cputype {cputype:#x}")
        return [MACHO_CPUTYPES[cputype]]
    raise ValueError("not a supported 64-bit or universal Mach-O binary")


def verify_paired_archive(archive_path, checksums_path=None, expected_members=SHIPPED_APPLICATIONS, expected_arches=DARWIN_ARCHES):
    archive_path = Path(archive_path)
    if archive_path.is_symlink() or not archive_path.is_file():
        raise ValueError("archive must be a regular non-symlink file")
    archive_bytes = archive_path.read_bytes()
    if len(archive_bytes) > (512 << 20):
        raise ValueError("archive exceeds 512 MiB bound")
    archive_sha256 = digest(archive_bytes)

    if checksums_path is not None:
        checksums_path = Path(checksums_path)
        if checksums_path.is_symlink() or not checksums_path.is_file() or checksums_path.stat().st_size > (1 << 20):
            raise ValueError("checksums file must be a bounded regular non-symlink file")
        matched = False
        for line in checksums_path.read_text(encoding="utf-8").splitlines():
            stripped = line.strip()
            if not stripped:
                continue
            parts = stripped.split()
            if len(parts) < 2 or not re.fullmatch(r"[0-9a-fA-F]{64}", parts[0]):
                raise ValueError("invalid checksums.txt line")
            filename = parts[1].lstrip("./")
            if filename == archive_path.name:
                if parts[0].lower() != archive_sha256:
                    raise ValueError("archive checksum mismatch")
                matched = True
                break
        if not matched:
            raise ValueError("archive missing from checksums.txt")

    members = {}
    total_extracted = 0
    with tarfile.open(archive_path, "r:gz") as tf:
        entries = tf.getmembers()
        if len(entries) > 64:
            raise ValueError("archive exceeds member count bound")
        for member in entries:
            if not member.isreg():
                raise ValueError(f"archive member {member.name!r} is not a regular file")
            clean = member.name[2:] if member.name.startswith("./") else member.name
            if not clean or "/" in clean or clean in (".", "..") or "\\" in clean:
                raise ValueError(f"invalid archive member path {member.name!r}")
            if clean in members:
                raise ValueError(f"duplicate archive member {clean!r}")
            if member.mode & 0o111 == 0:
                raise ValueError(f"archive member {clean!r} is not executable")
            extracted = tf.extractfile(member)
            if extracted is None:
                raise ValueError(f"unreadable archive member {clean!r}")
            data = extracted.read((512 << 20) + 1)
            total_extracted += len(data)
            if total_extracted > (512 << 20):
                raise ValueError("archive extracted size exceeds bound")
            arches = inspect_macho_arches(data)
            if set(arches) != set(expected_arches):
                raise ValueError(f"archive member {clean!r} architectures {arches} != {sorted(expected_arches)}")
            members[clean] = {"sha256": digest(data), "size": len(data), "arches": arches}

    if set(members) != set(expected_members):
        raise ValueError(f"archive member mismatch: got {sorted(members)}, want {sorted(expected_members)}")
    return {"archive": archive_path.name, "sha256": archive_sha256, "members": members}


def _run_pty_smoke(argv, env, send_q_on_alt=False, timeout=15.0):
    master, slave = pty.openpty()
    output = bytearray()
    sent_q = False
    try:
        proc = subprocess.Popen(argv, stdin=slave, stdout=slave, stderr=slave, env=env, close_fds=True)
    except BaseException:
        os.close(master)
        raise
    finally:
        os.close(slave)
    try:
        deadline = time.monotonic() + timeout
        while True:
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                proc.kill()
                proc.wait(timeout=5)
                raise ValueError("PTY pair smoke command timed out")
            ready, _, _ = select.select([master], [], [], min(0.05, remaining))
            if ready:
                try:
                    chunk = os.read(master, 65536)
                except OSError:
                    break
                if not chunk:
                    break
                output.extend(chunk)
                if send_q_on_alt and not sent_q and b"\x1b[?1049h" in output:
                    os.write(master, b"q")
                    sent_q = True
            elif proc.poll() is not None:
                break
        code = proc.wait(timeout=5)
    finally:
        os.close(master)
    return code, bytes(output)


def verify_native_pair_smoke(pair_dir, expected_identity=None):
    pair_dir = Path(pair_dir)
    if pair_dir.is_symlink() or not pair_dir.is_dir():
        raise ValueError("pair_dir must be a regular directory")
    bins = {}
    for name in SHIPPED_APPLICATIONS:
        path = pair_dir / name
        info = path.lstat()
        if path.is_symlink() or not stat.S_ISREG(info.st_mode) or info.st_mode & 0o111 == 0:
            raise ValueError(f"shipped pair binary missing or unsafe: {name}")
        bins[name] = path

    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        home = root / "home"
        tmpdir = root / "tmp"
        home.mkdir(mode=0o700)
        tmpdir.mkdir(mode=0o700)
        env = {
            "PATH": "/usr/local/bin:/usr/bin:/bin",
            "HOME": str(home),
            "TMPDIR": str(tmpdir),
            "LC_ALL": "C.UTF-8",
            "TERM": "xterm-256color",
            "AGENTIC_SDD_NO_UPDATE_CHECK": "1",
        }
        proto_res = subprocess.run([str(bins["asdx"]), "--protocol-info"], env=env, capture_output=True, text=True, timeout=15)
        if proto_res.returncode != 0:
            raise ValueError("asdx --protocol-info failed")
        proto = json.loads(proto_res.stdout)
        if proto.get("product") != "asdx" or proto.get("protocol_revision") != 2 or proto.get("distribution") != "generic":
            raise ValueError("unexpected asdx protocol info")
        for field in ("revision", "commit", "source_digest"):
            if not proto.get(field) or proto[field] in ("dev", "none", "unknown"):
                raise ValueError(f"uninjected pair identity field {field}")
        if expected_identity is not None:
            for key, value in expected_identity.items():
                if proto.get(key) != value:
                    raise ValueError(f"identity mismatch for {key}")

        legacy_proto = subprocess.run([str(bins["agentic-sdd-tui"]), "--protocol-info"], env=env, capture_output=True, text=True, timeout=15)
        if legacy_proto.returncode != 0 or legacy_proto.stderr != "" or json.loads(legacy_proto.stdout) != proto:
            raise ValueError("agentic-sdd-tui --protocol-info identity or stream mismatch")

        cli_ver = subprocess.run([str(bins["asdx-core"]), "--version"], env=env, capture_output=True, text=True, timeout=15)
        legacy_cli_ver = subprocess.run([str(bins["agentic-sdd"]), "--version"], env=env, capture_output=True, text=True, timeout=15)
        if cli_ver.returncode != 0 or "asdx-core" not in cli_ver.stdout or legacy_cli_ver.returncode != 0 or "agentic-sdd" not in legacy_cli_ver.stdout:
            raise ValueError("CLI --version check failed")

        canon_prev = subprocess.run([str(bins["asdx-core"]), "preview", "--home", str(home)], env=env, capture_output=True, timeout=15)
        legacy_prev = subprocess.run([str(bins["agentic-sdd"]), "preview", "--home", str(home)], env=env, capture_output=True, timeout=15)
        if canon_prev.returncode != 0 or legacy_prev.returncode != 0 or canon_prev.stdout != legacy_prev.stdout:
            raise ValueError("legacy CLI preview stdout diverged from canonical asdx-core")
        if b"agentic-sdd: deprecated compatibility name; running asdx-core" not in legacy_prev.stderr:
            raise ValueError("legacy CLI preview missing deprecation notice on stderr")

        pos_code, pos_out = _run_pty_smoke([str(bins["asdx-core"]), "tui", "--home", str(home)], env, send_q_on_alt=True)
        if pos_code != 0:
            raise ValueError(f"matched pair identity handshake failed (exit {pos_code}): {pos_out[-400:]!r}")

        alone = root / "alone"
        alone.mkdir(mode=0o700)
        shutil.copy2(bins["asdx-core"], alone / "asdx-core")
        (alone / "asdx-core").chmod(0o755)
        miss_code, miss_out = _run_pty_smoke([str(alone / "asdx-core"), "tui", "--home", str(home)], env)
        if miss_code != 1 or b"is not installed beside" not in miss_out or b"Preview only." in miss_out:
            raise ValueError("missing sibling check did not fail closed cleanly")

        mismatch_dir = root / "mismatch"
        mismatch_dir.mkdir(mode=0o700)
        shutil.copy2(bins["asdx-core"], mismatch_dir / "asdx-core")
        (mismatch_dir / "asdx-core").chmod(0o755)
        fake_asdx = mismatch_dir / "asdx"
        fake_asdx.write_text(
            "#!/bin/sh\n"
            'printf \'{"product":"asdx","protocol_revision":2,"distribution":"generic",'
            '"revision":"v0.0.0-mismatch","commit":"0000000000000000000000000000000000000000",'
            '"source_digest":"local-sha256:mismatch"}\\n\'\n'
        )
        fake_asdx.chmod(0o755)
        mis_code, mis_out = _run_pty_smoke([str(mismatch_dir / "asdx-core"), "tui", "--home", str(home)], env)
        if mis_code != 1 or b"does not match this CLI" not in mis_out or b"Preview only." in mis_out:
            raise ValueError("mismatched identity check did not fail closed cleanly")

    return {
        "identity": {"revision": proto["revision"], "commit": proto["commit"], "source_digest": proto["source_digest"]},
        "positive_handshake_exit": pos_code,
        "missing_sibling_exit": miss_code,
        "mismatched_identity_exit": mis_code,
    }


def _check_prereq(name, env, overrides=None):
    if overrides and name in overrides:
        return bool(overrides[name])
    if name == "python3":
        return bool(sys.executable)
    if name in ("task", "docker", "git"):
        return shutil.which(name, path=env.get("PATH")) is not None
    if name == "running Docker daemon":
        try:
            return subprocess.run(["docker", "info", "--format", "{{.ServerVersion}}"], env=env, capture_output=True, timeout=10).returncode == 0
        except (OSError, subprocess.SubprocessError):
            return False
    if name == "preloaded_go_image":
        try:
            return subprocess.run(["docker", "image", "inspect", IMAGE], env=env, capture_output=True, timeout=10).returncode == 0
        except (OSError, subprocess.SubprocessError):
            return False
    return False


def run_qualification(trusted, candidate, output, *, lane="all", suite_ids=None, archive_path=None, checksums_path=None, pair_dir=None, prereq_overrides=None):
    if lane not in ("all", "python", "archive"):
        raise ValueError("lane must be all, python, or archive")
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
    private.mkdir(mode=0o700)
    evidence.mkdir(mode=0o700)
    q_logs = evidence / "qualification"
    q_logs.mkdir(mode=0o700)
    env = safe_environment(private)
    env["PATH"] = os.environ.get("PATH", env["PATH"])

    report = {
        "version": 1,
        "execution": "blocked",
        "reason": "qualification not started",
        "lanes": {
            "core": {"status": "not_run", "note": "run separately via harness/checks.json"},
            "nested_tui": {"status": "not_run", "note": "qualification not started"},
            "python_suites": {"status": "not_run", "suites": []},
            "archive_distribution": {"status": "not_run"},
        },
        "trust": "caller-selected reviewed base; candidate cannot designate itself trusted",
    }

    try:
        for req_file in ("go.mod", "harness/checks.json", QUALIFICATION_CONTRACT_PATH, "tests/harness/trusted_ci.py"):
            if not (trusted / req_file).is_file() or (trusted / req_file).is_symlink():
                raise ValueError(f"trusted base missing {req_file}")
        base, work = private / "trusted", private / "candidate"
        stage_tree(trusted, base)
        stage_tree(candidate, work)
        required_contract(load_json(base / "harness/checks.json"), load_json(work / "harness/checks.json"))
        q_contract = qualification_contract(load_json(base / QUALIFICATION_CONTRACT_PATH), load_json(work / QUALIFICATION_CONTRACT_PATH))
        tui_contract = q_contract["lanes"]["nested_tui"]
        if tui_contract["status"] == "blocked":
            report["lanes"]["nested_tui"] = {"status": "blocked", "blocker": tui_contract["blocker"]}
        else:
            report["lanes"]["nested_tui"] = {
                "status": "not_run",
                "note": "offline vendored lane active in contract (-mod=vendor); run separately",
            }

        overall_blocked = False
        overall_failed = False

        if lane in ("all", "python"):
            suites = q_contract["lanes"]["python_suites"]["suites"]
            if suite_ids is not None:
                wanted = set(suite_ids)
                suites = [s for s in suites if s["id"] in wanted]
                if len(suites) != len(wanted):
                    raise ValueError("unknown python suite ID requested")
            suite_results = []
            for suite in suites:
                missing = [p for p in suite["prerequisites"] if not _check_prereq(p, env, prereq_overrides)]
                if missing:
                    overall_blocked = True
                    suite_results.append({
                        "id": suite["id"],
                        "argv": suite["argv"],
                        "status": "blocked",
                        "missing_prerequisites": missing,
                    })
                    continue
                suite_home = private / f"home-{suite['id']}"
                suite_tmp = private / f"tmp-{suite['id']}"
                suite_home.mkdir(mode=0o700, exist_ok=True)
                suite_tmp.mkdir(mode=0o700, exist_ok=True)
                suite_env = {
                    "PATH": env["PATH"],
                    "HOME": str(suite_home),
                    "TMPDIR": str(suite_tmp),
                    "LC_ALL": "C.UTF-8",
                    "GOTOOLCHAIN": "local",
                    "GOPROXY": "off",
                    "GOSUMDB": "off",
                    "GOENV": "off",
                    "GOWORK": "off",
                    "GOFLAGS": "",
                    "GIT_CONFIG_GLOBAL": os.devnull,
                    "GIT_CONFIG_NOSYSTEM": "1",
                    "GIT_OPTIONAL_LOCKS": "0",
                    "GIT_TERMINAL_PROMPT": "0",
                }
                suite_env.update(suite["env"])
                argv = [sys.executable, *suite["argv"][1:]]
                proc = subprocess.run(argv, cwd=work, env=suite_env, capture_output=True, timeout=suite["timeout_seconds"])
                log_file = q_logs / f"{suite['id']}.log"
                log_file.write_bytes(proc.stdout + proc.stderr)
                status = "passed" if proc.returncode == 0 else "failed"
                if status != "passed":
                    overall_failed = True
                suite_results.append({
                    "id": suite["id"],
                    "argv": suite["argv"],
                    "status": status,
                    "exit_code": proc.returncode,
                    "log": str(log_file),
                })
            py_status = "failed" if overall_failed else "blocked" if overall_blocked else "passed"
            report["lanes"]["python_suites"] = {"status": py_status, "suites": suite_results}

        if lane in ("all", "archive"):
            arch_report = {}
            if archive_path is None and pair_dir is None:
                overall_blocked = True
                arch_report = {"status": "blocked", "reason": "archive or pair-dir prerequisite not supplied"}
            else:
                try:
                    if archive_path is not None:
                        arch_report["paired_archive"] = verify_paired_archive(archive_path, checksums_path)
                    if pair_dir is not None:
                        arch_report["native_pair_smoke"] = verify_native_pair_smoke(pair_dir)
                    arch_report["status"] = "passed"
                except (ValueError, OSError, tarfile.TarError, json.JSONDecodeError) as exc:
                    overall_failed = True
                    arch_report["status"] = "failed"
                    arch_report["reason"] = str(exc)
            report["lanes"]["archive_distribution"] = arch_report

        if overall_failed:
            report.update(execution="failed", reason="one or more selected qualification checks failed")
        elif overall_blocked:
            report.update(execution="blocked", reason="selected qualification lane has blocked prerequisites")
        else:
            if q_contract["lanes"]["nested_tui"]["status"] == "blocked":
                report.update(execution="completed", reason="selected qualification lanes passed; nested_tui stays blocked until §12.5 owner decision")
            else:
                report.update(execution="completed", reason="selected qualification lanes passed")
    except (ValueError, OSError, subprocess.SubprocessError) as exc:
        report.update(execution="blocked", reason=str(exc))

    write_json(evidence / "qualification.json", report)
    return 0 if report["execution"] == "completed" else 2 if report["execution"] == "blocked" else 1


def main():
    def cancel(_number, _frame):
        raise KeyboardInterrupt
    signal.signal(signal.SIGTERM, cancel)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("run", "whitespace", "qualify", "verify-archive"))
    parser.add_argument("--trusted")
    parser.add_argument("--candidate")
    parser.add_argument("--output")
    parser.add_argument("--lane", choices=("all", "python", "archive"), default="all")
    parser.add_argument("--archive")
    parser.add_argument("--checksums")
    parser.add_argument("--pair-dir")
    args = parser.parse_args()
    if args.action == "verify-archive":
        if not args.archive:
            parser.error("verify-archive requires --archive")
        try:
            summary = verify_paired_archive(args.archive, args.checksums)
        except (ValueError, OSError, tarfile.TarError) as exc:
            print(json.dumps({"execution": "failed", "reason": str(exc)}))
            return 1
        print(json.dumps({"execution": "completed", "archive": summary}))
        return 0
    if not args.trusted or not args.candidate:
        parser.error(f"{args.action} requires --trusted and --candidate")
    if args.action == "whitespace":
        return whitespace(Path(args.trusted), Path(args.candidate))
    if not args.output:
        parser.error(f"{args.action} requires external --output")
    if args.action == "qualify":
        if Path(__file__).resolve() != Path(args.trusted).resolve() / "tests/harness/trusted_ci.py":
            print("qualification launcher must be loaded from --trusted", file=sys.stderr)
            return 2
        try:
            return run_qualification(
                args.trusted,
                args.candidate,
                args.output,
                lane=args.lane,
                archive_path=args.archive,
                checksums_path=args.checksums,
                pair_dir=args.pair_dir,
            )
        except (ValueError, OSError):
            print("invalid qualification input; no work authorized", file=sys.stderr)
            return 2
    try:
        return run_gate(args.trusted, args.candidate, args.output)
    except (ValueError, OSError):
        print("invalid gate input; no work authorized", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
