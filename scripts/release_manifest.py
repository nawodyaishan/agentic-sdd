#!/usr/bin/env python3
"""Release artifact manifest, digest re-verification and sanitized records (B7/T14, FR-06).

The read-only verify job writes the upload set (the darwin archive, checksums.txt)
and a manifest that binds each file to its SHA-256 digest, size, the release tag
and the source commit. The publisher re-hashes that set against the manifest and
never rebuilds the bytes it uploads.

Subcommands
  write     --dist DIR --out DIR --tag vX.Y.Z --commit SHA
  verify    --dir DIR --manifest FILE --tag vX.Y.Z --commit SHA
  record    --out FILE --tag vX.Y.Z --commit SHA --status STATUS [--manifest FILE]
  wait-ci   --runs FILE --commit SHA --suite-id ID
  check-ci  --runs FILE --commit SHA --suite-id ID

Both CI subcommands read JSON Lines (one check run per line, as the workflow's
`gh api --jq` projects them: name, status, conclusion and check_suite.id). The
publisher's own check suite is excluded by exact integer match of check_suite.id,
never by URL, so the release's own jobs cannot satisfy the gate. A run with a
missing or non-integer check_suite.id is not the publisher's and always counts.
Among the other runs, refuse when there are none, or when any completed run did
not conclude success.

  wait-ci   judges one snapshot for polling: exit 0 when every other run has
            completed and succeeded; exit 4 (pending) while some are still
            queued or in progress; refuse (1) on no runs, any completed failure,
            or a status other than queued, in_progress or completed (such as
            stale or requested), without waiting for pending runs.
  check-ci  is the final decision on a snapshot: pass only when every other run
            has completed and succeeded. A run that has not completed is refused.

The publish workflow polls wait-ci on a bounded budget, then runs check-ci on the
final snapshot. Exit codes: 0 pass; 1 verification refusal; 2 invalid usage or
input; 4 pending (wait-ci only).
Records hold no environment, token or runner detail; they are safe to retain.
"""

import argparse
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import re
import shutil
import sys
import tarfile


FORMAT = 1
SHIPPED_BINARIES = ("asdx", "asdx-core", "agentic-sdd", "agentic-sdd-tui")
MANIFEST_NAME = "manifest.json"
CHECKSUMS_NAME = "checksums.txt"
STATUSES = ("success", "failure", "cancelled", "skipped")

TAG_RE = re.compile(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)")
COMMIT_RE = re.compile(r"[0-9a-f]{40}")
SHA256_RE = re.compile(r"[0-9a-f]{64}")
SAFE_NAME_RE = re.compile(r"[A-Za-z0-9][A-Za-z0-9._+-]*")
CHECKSUM_LINE_RE = re.compile(r"([0-9a-f]{64})  (\S+)")


class Refusal(Exception):
    """A verification failure: the release must not proceed."""


class UsageError(Exception):
    """Invalid arguments or inputs; nothing was verified."""


def check_tag(tag):
    if not isinstance(tag, str) or not TAG_RE.fullmatch(tag):
        raise UsageError(f"tag must be a plain release tag vX.Y.Z, got {tag!r}")
    return tag[1:]


def check_commit(commit):
    if not isinstance(commit, str) or not COMMIT_RE.fullmatch(commit):
        raise UsageError("commit must be a 40-character lowercase hex SHA")


def archive_pattern(version):
    return re.compile(rf"agentic-sdd_{re.escape(version)}_darwin_[A-Za-z0-9]+\.tar\.gz")


def sha256_file(path):
    digest = hashlib.sha256()
    with open(path, "rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def parse_checksums(path):
    if not path.is_file() or path.is_symlink():
        raise Refusal(f"{path.name} is missing or not a regular file")
    listed = {}
    for line in path.read_text(encoding="utf-8").splitlines():
        if not line.strip():
            continue
        match = CHECKSUM_LINE_RE.fullmatch(line)
        if not match:
            raise Refusal(f"unrecognised line in {path.name}")
        digest, name = match.groups()
        if name in listed:
            raise Refusal(f"duplicate entry for {name} in {path.name}")
        listed[name] = digest
    return listed


def archive_members(path):
    """Return the exact member names, each a file at the archive root.

    The layout is flat: .goreleaser.yml sets no wrap_in_directory, so no member sits under a
    directory and a wrapper directory is refused as well. Names are exact, not basenames, so
    a nested binary cannot pass as a root one. Duplicates, unsafe paths, non-regular entries
    and a missing shipped binary are refused.
    """
    names = []
    with tarfile.open(path, "r:gz") as tar:
        for member in tar.getmembers():
            if "/" in member.name:
                raise Refusal(f"archive member is not at the archive root: {member.name}")
            if member.name.startswith("/") or ".." in Path(member.name).parts:
                raise Refusal(f"archive member path is unsafe: {member.name}")
            if not member.isfile():
                raise Refusal(f"archive member is not a regular file: {member.name}")
            if member.name in names:
                raise Refusal(f"archive lists a member more than once: {member.name}")
            names.append(member.name)
    missing = sorted(set(SHIPPED_BINARIES) - set(names))
    if missing:
        raise Refusal(f"archive is missing shipped binaries: {', '.join(missing)}")
    return sorted(names)


def load_manifest(path):
    try:
        data = json.loads(Path(path).read_text(encoding="utf-8"))
    except (OSError, ValueError) as exc:
        raise Refusal(f"cannot read manifest {path}: {exc}")
    if not isinstance(data, dict) or data.get("format") != FORMAT:
        raise Refusal("unsupported manifest format")
    for key in ("tag", "version", "commit", "files", "archive_members"):
        if key not in data:
            raise Refusal(f"manifest is missing {key}")
    return data


def verify_release(directory, manifest_path, tag, commit=None):
    """Re-hash the upload set against the manifest. Returns (manifest, archive name)."""
    version = check_tag(tag)
    if commit is not None:
        check_commit(commit)
    directory = Path(directory)
    data = load_manifest(manifest_path)
    if data["tag"] != tag or data["version"] != version:
        raise Refusal(f"manifest tag {data['tag']!r} does not match {tag!r}")
    if commit is not None and data["commit"] != commit:
        raise Refusal(f"manifest commit {data['commit']} does not match {commit}")
    if not isinstance(data["files"], list) or not data["files"]:
        raise Refusal("manifest lists no files")

    entries = []
    for entry in data["files"]:
        if not isinstance(entry, dict):
            raise Refusal("manifest file entry is malformed")
        name, digest, size = entry.get("name"), entry.get("sha256"), entry.get("size")
        if not isinstance(name, str) or not SAFE_NAME_RE.fullmatch(name) or name == MANIFEST_NAME:
            raise Refusal(f"manifest file name is unsafe: {name!r}")
        if not isinstance(digest, str) or not SHA256_RE.fullmatch(digest):
            raise Refusal(f"manifest sha256 is malformed for {name}")
        if not isinstance(size, int) or isinstance(size, bool) or size < 0:
            raise Refusal(f"manifest size is malformed for {name}")
        entries.append(entry)
    names = [entry["name"] for entry in entries]
    if len(set(names)) != len(names):
        raise Refusal("manifest lists a file more than once")
    if CHECKSUMS_NAME not in names:
        raise Refusal(f"manifest does not bind {CHECKSUMS_NAME}")
    archives = [name for name in names if archive_pattern(version).fullmatch(name)]
    if len(archives) != 1:
        raise Refusal(f"expected exactly one darwin archive for {version}, found {len(archives)}")
    archive = archives[0]
    # The upload set is exactly checksums.txt and the one version archive: any other
    # entry, even one with a correct digest, would be published as a release asset.
    expected = {CHECKSUMS_NAME, archive}
    if set(names) != expected:
        extra = sorted(set(names) - expected)
        missing = sorted(expected - set(names))
        raise Refusal(f"manifest must list exactly {CHECKSUMS_NAME} and {archive}; "
                      f"extra: {', '.join(extra) or 'none'}; missing: {', '.join(missing) or 'none'}")

    present = set()
    for path in directory.iterdir():
        if path.name == MANIFEST_NAME:
            continue
        if path.is_symlink() or not path.is_file():
            raise Refusal(f"unexpected non-regular entry {path.name}")
        present.add(path.name)
    missing = sorted(set(names) - present)
    extra = sorted(present - set(names))
    if missing:
        raise Refusal(f"missing artifacts: {', '.join(missing)}")
    if extra:
        raise Refusal(f"unexpected files not in manifest: {', '.join(extra)}")

    for entry in entries:
        path = directory / entry["name"]
        if sha256_file(path) != entry["sha256"]:
            raise Refusal(f"sha256 mismatch for {entry['name']}")
        if path.stat().st_size != entry["size"]:
            raise Refusal(f"size mismatch for {entry['name']}")

    listed = parse_checksums(directory / CHECKSUMS_NAME)
    if listed != {archive: sha256_file(directory / archive)}:
        raise Refusal(f"{CHECKSUMS_NAME} does not match the archive digest")
    if sorted(archive_members(directory / archive)) != sorted(data["archive_members"]):
        raise Refusal("archive members differ from the manifest")
    return data, archive


def cmd_write(args):
    version = check_tag(args.tag)
    check_commit(args.commit)
    dist = Path(args.dist)
    out = Path(args.out)
    if not dist.is_dir():
        raise UsageError(f"dist directory not found: {dist}")
    matches = [p for p in sorted(dist.iterdir()) if archive_pattern(version).fullmatch(p.name)]
    if len(matches) != 1:
        raise Refusal(f"expected exactly one darwin archive for {version}, found {len(matches)}")
    archive = matches[0]
    if archive.is_symlink() or not archive.is_file():
        raise Refusal(f"archive is not a regular file: {archive.name}")
    checksums = parse_checksums(dist / CHECKSUMS_NAME)
    if checksums != {archive.name: sha256_file(archive)}:
        raise Refusal(f"{CHECKSUMS_NAME} does not match the archive digest")
    members = archive_members(archive)
    if out.exists() and any(out.iterdir()):
        raise UsageError(f"output directory must be empty: {out}")
    out.mkdir(parents=True, exist_ok=True)
    files = []
    for source in (archive, dist / CHECKSUMS_NAME):
        target = out / source.name
        shutil.copyfile(source, target)
        files.append({
            "name": target.name,
            "sha256": sha256_file(target),
            "size": target.stat().st_size,
        })
    manifest = {
        "format": FORMAT,
        "tag": args.tag,
        "version": version,
        "commit": args.commit,
        "archive_members": members,
        "files": sorted(files, key=lambda item: item["name"]),
    }
    (out / MANIFEST_NAME).write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n",
                                     encoding="utf-8")
    print(f"release_manifest: wrote {len(files)} artifacts and {MANIFEST_NAME} to {out}")
    return 0


def cmd_verify(args):
    check_tag(args.tag)
    check_commit(args.commit)
    verify_release(args.dir, args.manifest, args.tag, commit=args.commit)
    print(f"release_manifest: verified artifacts for {args.tag} at {args.commit}")
    return 0


def cmd_record(args):
    check_tag(args.tag)
    check_commit(args.commit)
    if args.status not in STATUSES:
        raise UsageError(f"status must be one of {', '.join(STATUSES)}")
    manifest_sha = None
    if args.manifest and Path(args.manifest).is_file():
        manifest_sha = sha256_file(args.manifest)
    record = {
        "format": FORMAT,
        "tag": args.tag,
        "commit": args.commit,
        "job_status": args.status,
        "manifest_sha256": manifest_sha,
        "recorded_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    }
    out = Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps(record, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print(f"release_manifest: recorded {args.status} for {args.tag} at {out}")
    return 0


def load_check_runs(path):
    try:
        text = Path(path).read_text(encoding="utf-8")
    except (OSError, ValueError) as exc:
        raise UsageError(f"cannot read check runs {path}: {exc}")
    runs = []
    for number, line in enumerate(text.splitlines(), 1):
        if not line.strip():
            continue
        try:
            item = json.loads(line)
        except ValueError:
            raise UsageError(f"check run line {number} is not JSON")
        if not isinstance(item, dict) or not isinstance(item.get("name"), str) \
                or not isinstance(item.get("status"), str):
            raise UsageError(f"check run line {number} is malformed")
        runs.append(item)
    return runs


PENDING_EXIT = 4
KNOWN_CHECK_STATUSES = ("queued", "in_progress", "completed")


def is_own_suite(run, suite_id):
    """True only when check_suite.id is exactly this publisher run's integer suite id."""
    suite = run.get("check_suite")
    if not isinstance(suite, dict):
        return False
    value = suite.get("id")
    return type(value) is int and value == suite_id


def judge_check_runs(args):
    """Return the check runs still pending. Refuse on no runs or any completed failure."""
    check_commit(args.commit)
    if not args.suite_id or not re.fullmatch(r"[1-9][0-9]*", args.suite_id):
        raise UsageError("suite id must be a positive integer")
    suite_id = int(args.suite_id)
    runs = load_check_runs(args.runs)
    others = [run for run in runs if not is_own_suite(run, suite_id)]
    if not others:
        raise Refusal(f"no check runs on {args.commit} outside this publisher run; refusing to publish")
    # Only queued, in_progress and completed are understood. Any other status (stale,
    # requested, waiting, pending, ...) is refused at once: treating it as pending would
    # burn the whole wait budget on a state that may never change.
    unknown = [run for run in others if run["status"] not in KNOWN_CHECK_STATUSES]
    if unknown:
        names = ", ".join(f"{run['name']} ({run['status']})" for run in unknown)
        raise Refusal(f"check runs on {args.commit} report a status other than queued, "
                      f"in_progress or completed: {names}; refusing to publish")
    failed = [run for run in others
              if run["status"] == "completed" and run.get("conclusion") != "success"]
    if failed:
        names = ", ".join(f"{run['name']} ({run.get('conclusion')})" for run in failed)
        raise Refusal(f"check runs did not succeed on {args.commit}: {names}; refusing to publish")
    return others, [run for run in others if run["status"] != "completed"]


def cmd_wait_ci(args):
    others, pending = judge_check_runs(args)
    if pending:
        names = ", ".join(f"{run['name']} ({run['status']})" for run in pending)
        print(f"release_manifest: {len(pending)} check runs on {args.commit} not completed yet: {names}")
        return PENDING_EXIT
    print(f"release_manifest: {len(others)} check runs on {args.commit} completed successfully")
    return 0


def cmd_check_ci(args):
    others, pending = judge_check_runs(args)
    if pending:
        names = ", ".join(f"{run['name']} ({run['status']})" for run in pending)
        raise Refusal(f"{len(pending)} check runs on {args.commit} have not completed ({names}); "
                      "refusing to publish")
    print(f"release_manifest: {len(others)} check runs on {args.commit} completed successfully")
    return 0


def build_parser():
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = parser.add_subparsers(dest="command", required=True)

    write = sub.add_parser("write", help="copy the upload set and write the manifest")
    write.add_argument("--dist", required=True)
    write.add_argument("--out", required=True)
    write.add_argument("--tag", required=True)
    write.add_argument("--commit", required=True)
    write.set_defaults(func=cmd_write)

    verify = sub.add_parser("verify", help="re-hash the upload set against the manifest")
    verify.add_argument("--dir", required=True)
    verify.add_argument("--manifest", required=True)
    verify.add_argument("--tag", required=True)
    verify.add_argument("--commit", required=True)
    verify.set_defaults(func=cmd_verify)

    record = sub.add_parser("record", help="write a sanitized verification record")
    record.add_argument("--out", required=True)
    record.add_argument("--tag", required=True)
    record.add_argument("--commit", required=True)
    record.add_argument("--status", required=True)
    record.add_argument("--manifest")
    record.set_defaults(func=cmd_record)

    wait = sub.add_parser("wait-ci", help="poll step: exit 4 while the tagged commit's CI is pending")
    wait.add_argument("--runs", required=True)
    wait.add_argument("--commit", required=True)
    wait.add_argument("--suite-id", required=True)
    wait.set_defaults(func=cmd_wait_ci)

    check = sub.add_parser("check-ci", help="refuse unless every CI run on the tagged commit completed and passed")
    check.add_argument("--runs", required=True)
    check.add_argument("--commit", required=True)
    check.add_argument("--suite-id", required=True)
    check.set_defaults(func=cmd_check_ci)
    return parser


def main(argv=None):
    args = build_parser().parse_args(argv)
    try:
        return args.func(args)
    except Refusal as exc:
        print(f"release_manifest: refused: {exc}", file=sys.stderr)
        return 1
    except UsageError as exc:
        print(f"release_manifest: invalid input: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
