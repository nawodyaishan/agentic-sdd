"""Offline development receipt reference; no provider, task writer or agent loop.

Callers supply independently observed authority, identities and result/log digests.
Accepting a receipt never authorizes work or certifies the observer/CI trust chain.
Requires the same optional jsonschema development dependency as validate_contracts.
"""
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import stat
import subprocess
from datetime import datetime

from jsonschema import Draft202012Validator
from validate_contracts import FORMATS, ROOT


def sha256(data):
    return hashlib.sha256(data).hexdigest()


def schema_check(data, relative):
    schema = json.loads((ROOT / relative).read_text())
    validator = Draft202012Validator(schema, format_checker=FORMATS)
    if next(validator.iter_errors(data), None) is not None:
        # Never echo a ValidationError: its rendering includes rejected values.
        raise ValueError("invalid schema; no input values exported")


def snapshot(workspace, paths):
    """Hash explicit scopes, HEAD/index entries, and working bytes/kinds/modes.

    Symlinks/special files/submodules and concurrent writers are unsupported;
    missing files are recorded, not silently omitted. Ignored files are excluded
    unless explicitly named. Never select auth stores or a real user's home.
    """
    workspace = Path(workspace).resolve(strict=True)
    if not isinstance(paths, (list, tuple)) or not paths or len(paths) != len(set(paths)):
        raise ValueError("explicit unique identity scope required")
    for path in paths:
        if not isinstance(path, str) or not path or PurePosixPath(path).is_absolute() or any(
            part in ("..", ".git") for part in PurePosixPath(path).parts
        ) or any(ch in path for ch in "\\*?[]:\x00\n\r") or str(PurePosixPath(path)) != path:
            raise ValueError("unsafe identity scope")
        current = workspace
        for part in PurePosixPath(path).parts:
            current = current / part
            if current.is_symlink():
                raise ValueError("identity scope symlink refused")

    git_env = {"PATH": os.environ.get("PATH", "/usr/bin:/bin"), "HOME": os.devnull,
               "GIT_CONFIG_GLOBAL": os.devnull, "GIT_CONFIG_NOSYSTEM": "1",
               "GIT_OPTIONAL_LOCKS": "0", "GIT_TERMINAL_PROMPT": "0", "GIT_NO_REPLACE_OBJECTS": "1"}

    def git(*args):
        try:
            result = subprocess.check_output(
                ["git", "--no-optional-locks", "--literal-pathspecs", "-C", str(workspace), *args],
                stderr=subprocess.PIPE, timeout=15,
                env=git_env)
        except (subprocess.SubprocessError, OSError):
            raise ValueError("identity Git observation unavailable") from None
        if len(result) > 10 << 20:
            raise ValueError("identity Git observation exceeds bound")
        return result

    if Path(os.fsdecode(git("rev-parse", "--show-toplevel")).strip()).resolve() != workspace:
        raise ValueError("workspace must be the repository root")
    base = git("rev-parse", "--verify", "HEAD").decode().strip()
    head = git("ls-tree", "-r", "-z", "HEAD", "--", *paths)
    index = git("ls-files", "--stage", "-z", "--", *paths)
    others = git("ls-files", "--others", "--exclude-standard", "-z", "--", *paths)
    names = set()
    records = [["scope", path] for path in sorted(paths)]
    for kind, raw in (("HEAD", head), ("index", index)):
        for entry in raw.split(b"\0"):
            if not entry:
                continue
            meta, name = entry.split(b"\t", 1)
            path = os.fsdecode(name)
            names.add(path)
            if meta.split()[0] not in (b"100644", b"100755"):
                raise ValueError("identity contains unsupported Git file kind")
            if kind == "index" and meta.split()[-1] != b"0":
                raise ValueError("unmerged index cannot establish identity")
            records.append([kind, path, meta.decode("ascii")])
    names.update(os.fsdecode(p) for p in others.split(b"\0") if p)
    # Git does not report untracked FIFOs/devices. Inspect kinds without reading
    # them; prune ignored directories and .git so this never opens auth/cache data.
    def ignored(name):
        probe = subprocess.run(["git", "--no-optional-locks", "--literal-pathspecs", "-C", str(workspace),
                                "check-ignore", "-q", "--", name],
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=15, env=git_env)
        if probe.returncode not in (0, 1):
            raise ValueError("ignore observation unavailable")
        return probe.returncode == 0

    seen = 0
    for scope in paths:
        target = workspace / scope
        if target.is_symlink():
            raise ValueError("identity symlink refused")
        if not target.is_dir():
            continue
        for directory, dirs, files in os.walk(target, followlinks=False):
            for name in dirs[:]:
                child = Path(directory) / name
                rel = child.relative_to(workspace).as_posix()
                if name == ".git" or ignored(rel):
                    dirs.remove(name)
                elif child.is_symlink():
                    raise ValueError("identity symlink refused")
            for name in files:
                child = Path(directory) / name
                seen += 1
                if seen > 10000:
                    raise ValueError("identity entry count exceeds bound")
                if not stat.S_ISREG(child.lstat().st_mode) and not ignored(child.relative_to(workspace).as_posix()):
                    raise ValueError("identity special file refused")
    # Explicit untracked/ignored/deleted file paths also form part of the scope.
    for path in paths:
        target = workspace / path
        if path != "." and (not target.exists() or not target.is_dir()):
            names.add(path)
    if len(names) > 10000:
        raise ValueError("identity file count exceeds bound")
    total = 0
    for name in sorted(names):
        target = workspace / name
        for ancestor in [target, *target.parents]:
            if ancestor == workspace:
                break
            if ancestor.is_symlink():
                raise ValueError("identity symlink refused")
        if not target.exists():
            records.append(["worktree", name, "missing"])
            continue
        info = target.lstat()
        if not stat.S_ISREG(info.st_mode) or info.st_size > 32 << 20:
            raise ValueError("identity requires bounded regular files")
        data = target.read_bytes()
        total += len(data)
        if total > 128 << 20:
            raise ValueError("identity bytes exceed bound")
        records.append(["worktree", name, "file", stat.S_IMODE(info.st_mode), sha256(data)])
    payload = json.dumps(sorted(records), ensure_ascii=True, separators=(",", ":")).encode()
    return {"base_revision": base, "diff": "sha256-v1:" + sha256(payload)}


def nulls(value, pointer=""):
    if value is None:
        yield pointer
    elif isinstance(value, dict):
        for key, child in value.items():
            yield from nulls(child, pointer + "/" + key.replace("~", "~0").replace("/", "~1"))
    elif isinstance(value, list):
        for i, child in enumerate(value):
            yield from nulls(child, pointer + "/" + str(i))


def private_regular(path, root):
    path, root = Path(path), Path(root)
    if (not path.is_absolute() or not root.is_absolute() or ".." in path.parts
            or ".." in root.parts or not path.is_relative_to(root)):
        raise ValueError("evidence outside authorized root")
    for part in [path, *path.parents]:
        if part.is_symlink():
            raise ValueError("evidence symlink refused")
    try:
        info = path.lstat()
        root_info = root.stat()
    except OSError:
        raise ValueError("evidence missing") from None
    if (not stat.S_ISREG(info.st_mode) or stat.S_IMODE(info.st_mode) & 0o077
            or stat.S_IMODE(root_info.st_mode) & 0o077 or info.st_size > 10 << 20):
        raise ValueError("evidence is not bounded and private")
    return path


def validate_receipt(receipt, *, workspace, identity, authorization_ref, contract, result, evidence_root, log_digests,
                     observations=None, paid_authorized=False):
    """Validate against independently supplied current observations, never itself.

    This checks consistency/freshness, not whether the authorization reference is
    genuine or a reviewer performed sound engineering review. The caller owns
    those observations. No source, task, approval or evidence is modified.
    """
    schema_check(receipt, "harness/evidence/receipt.schema.json")
    if receipt["mode"] == "direct-fix" and receipt["workflow_state"] is not None:
        raise ValueError("direct fix has no canonical task state")
    observations = observations or {}
    measured = {"/session_id": receipt["session_id"], "/cost_usd": receipt["cost_usd"],
                **{"/model/" + key: value for key, value in receipt["model"].items()}}
    if any(value != observations.get(ptr) for ptr, value in measured.items()):
        raise ValueError("native/model/billing claim lacks independent observation")
    if receipt["limits"]["paid_authorized"] != paid_authorized:
        raise ValueError("receipt cannot create paid authorization")
    checked = json.loads(contract)
    schema_check(checked, "harness/checks.schema.json")
    if not isinstance(result, dict) or result.get("version") != 1:
        raise ValueError("independent check result required")
    if receipt["workspace"] != str(Path(workspace).resolve(strict=True)) or result.get("workspace") != receipt["workspace"]:
        raise ValueError("workspace changed; reconcile location before reuse")
    evidence_root = Path(evidence_root)
    if not evidence_root.is_absolute() or evidence_root.resolve().is_relative_to(Path(workspace).resolve()):
        raise ValueError("evidence root must be outside target workspace")
    if receipt["identity"] != identity:
        raise ValueError("identity changed; evidence is not fresh")
    if result.get("contract_sha256") != sha256(contract) or identity["contract"] != sha256(contract):
        raise ValueError("contract identity mismatch")
    observed_nulls = set(nulls(receipt))
    structural = {"/review", "/workflow_state"}
    if receipt["mode"] == "direct-fix":
        structural.update(("/feature", "/batch"))
    for i, ch in enumerate(receipt["checks"]):
        if ch["status"] != "passed":
            structural.update("/checks/" + str(i) + "/" + k for k in ("exit_code", "started_at", "ended_at", "evidence"))
    for ptr in observed_nulls - structural:
        if not receipt["unknowns"].get(ptr, "").strip():
            raise ValueError("missing unknown-observation reason")
    if any(ptr not in observed_nulls or not reason.strip() for ptr, reason in receipt["unknowns"].items()):
        raise ValueError("unknown reason does not describe a null observation")
    if receipt["execution"] != "blocked" and (not authorization_ref or receipt["authorization_ref"] != authorization_ref):
        raise ValueError("independently reconciled authorization required")
    if receipt["authorization_ref"] is not None and receipt["authorization_ref"] != authorization_ref:
        raise ValueError("receipt cannot create authorization")
    definitions = checked["checks"]
    ids = [c["id"] for c in definitions]
    if len(ids) != len(set(ids)):
        raise ValueError("duplicate trusted contract IDs")
    selection = receipt["selection"]
    selected = selection["check_ids"]
    expected = [c["id"] for c in definitions if
                (c["id"] in selected if selection["scope"] == "focused" else selection["scope"] in c["scopes"])]
    if not selected or selected != expected:
        raise ValueError("selection coverage mismatch")
    if selection != result.get("selection") or receipt["checks"] != result.get("checks") or receipt["execution"] != result.get("execution"):
        raise ValueError("receipt disagrees with independent check result")
    checks = receipt["checks"]
    if [ch["id"] for ch in checks] != selected:
        raise ValueError("check coverage missing or duplicated")
    findings = receipt["findings"]
    if len({f["id"] for f in findings}) != len(findings):
        raise ValueError("duplicate finding IDs")
    for ch in checks:
        if ch["started_at"] and ch["ended_at"]:
            start = datetime.fromisoformat(ch["started_at"].upper().replace("Z", "+00:00"))
            end = datetime.fromisoformat(ch["ended_at"].upper().replace("Z", "+00:00"))
            if end < start:
                raise ValueError("check time order invalid")
        if ch["status"] == "passed":
            if ch["exit_code"] != 0 or not all(ch[k] for k in ("started_at", "ended_at", "evidence")) or ch["truncated"]:
                raise ValueError("passed check lacks complete evidence")
            if any(v is None for v in identity.values()):
                raise ValueError("unknown identity prevents pass reuse")
        if ch["status"] in ("blocked", "not_run") and (ch["exit_code"] is not None or ch["started_at"] is not None):
            raise ValueError("nonexecuted check has execution observations")
        if ch["evidence"] is not None:
            path = private_regular(ch["evidence"], evidence_root)
            if sha256(path.read_bytes()) != log_digests.get(str(path)):
                raise ValueError("evidence digest changed or unavailable")
    all_passed = all(ch["status"] == "passed" for ch in checks)
    first_nonpass = next((i for i, ch in enumerate(checks) if ch["status"] != "passed"), len(checks))
    expected_execution = "completed" if all_passed else checks[first_nonpass]["status"]
    if (receipt["execution"] != expected_execution or expected_execution == "not_run"
            or any(ch["status"] != "not_run" for ch in checks[first_nonpass + 1:])):
        raise ValueError("execution disagrees with check outcomes")
    if receipt["review"] == "Ready" and (not all_passed or selection["scope"] != "required"
            or not authorization_ref or any(f["disposition"] != "disputed" for f in findings)):
        raise ValueError("Ready lacks required coverage or resolved findings")


def export_receipt(receipt, output, *, workspace, redactions=()):
    """Export only a curated schema allowlist, scrub known secrets BEFORE writing.

    Not a native-log converter or universal secret detector. Callers must omit
    sensitive free text and supply literal synthetic/known-secret redactions from
    a private source (never CLI args). No auth store is read. No concurrent writer.
    """
    schema_check(receipt, "harness/evidence/receipt.schema.json")
    if any(not isinstance(s, str) or not s for s in redactions):
        raise ValueError("redactions must be nonempty literal strings")

    def scrub(value):
        if isinstance(value, str):
            for secret in sorted(redactions, key=len, reverse=True):
                value = value.replace(secret, "[REDACTED]")
            value = re.sub(r"(?i)(bearer\s+)[^\s\"'<>]+", r"\1[REDACTED]", value)
            return re.sub(r"(?i)((?:api[_-]?key|token|password|secret)\s*[:=]\s*)[^\s,;\"'<>]+", r"\1[REDACTED]", value)
        if isinstance(value, list):
            return [scrub(v) for v in value]
        if isinstance(value, dict):
            if any(scrub(key) != key for key in value):
                raise ValueError("sensitive field name refused")
            return {key: scrub(v) for key, v in value.items()}
        return value

    sanitized = scrub(receipt)
    schema_check(sanitized, "harness/evidence/receipt.schema.json")
    data = (json.dumps(sanitized, ensure_ascii=True, indent=2, allow_nan=False) + "\n").encode()
    if len(data) > 1 << 20:
        raise ValueError("receipt export exceeds bound")
    output, workspace = Path(output).absolute(), Path(workspace).resolve(strict=True)
    if ".." in output.parts or output.is_relative_to(workspace):
        raise ValueError("receipt must be outside target checkout")
    for part in [output, *output.parents]:
        if part.is_symlink():
            raise ValueError("receipt output symlink refused")
    parent = output.parent
    if not parent.is_dir() or stat.S_IMODE(parent.stat().st_mode) & 0o077:
        raise ValueError("existing private external parent required")
    try:
        fd = os.open(output, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    except OSError:
        raise ValueError("receipt destination unavailable; no overwrite") from None
    with os.fdopen(fd, "wb") as stream:
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())
