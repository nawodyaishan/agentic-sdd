#!/usr/bin/env python3
"""Release eligibility binding for the 023 publisher preflight (B2/T4).

The token-bearing publisher must not run until a read-only preflight has proved
two things from GitHub records and git objects, not from candidate prose:

  tag-run    The completed "Release build" run was caused by the original tag
             push of this exact version at this exact commit, by the owner, with
             the reviewed build workflow at that commit. GitHub's rendered
             run-name carries ref, creation flag, sender id and SHA, so the
             original event stays bound even after a rerun.

  promotion  The tagged commit is the merge of one same-repository `dev` ->
             `main` promotion PR merged by the owner. The merge object itself
             supplies the immutable first parent (prior `main`) and second
             parent (tested `dev` head). The version is read from the tracked
             `release-version` blob in the merged tree and in trusted `main`,
             never from a PR title or body, and must strictly increase over the
             first parent and over every earlier stable release tag.

Inputs are JSON files captured by the workflow's `gh api` calls and files read
from git object storage. This program performs no network access, runs no
candidate code and writes nothing outside stdout/stderr. Missing, ambiguous or
contradictory evidence refuses.

Interpretation notes:
  * `display_title` is GitHub's rendered run-name. The reviewed build workflow
    renders `refs/tags/<tag>|<created>|<sender.id>|<sha>`. A rerun keeps the
    original run's titles; `triggering_actor` is never used as identity.
  * PR `base.sha` in the REST response can be a later `main` tip, so it is not
    used. Parent order comes from the merge commit's own parents.
  * Prose cannot authorize a version. If a PR title or body names a different
    version, the tracked blob still decides; the prose is not evidence.
  * Prior tags that are not strict stable `vMAJOR.MINOR.PATCH` (prerelease,
    leading zero, other `v*` names) are not release records and do not count.

Subcommands
  tag-run    --run FILE --run-id ID --workflow-id ID
             --tagged-workflow-blob SHA --reviewed-workflow-blob SHA
             --tag-ref FILE --tag-object FILE
             --tag vX.Y.Z --tag-commit SHA --repository-id ID
  promotion  --pulls FILE --merge-sha SHA --parent1 SHA --parent2 SHA
             --version vX.Y.Z --merge-version-file FILE
             --parent-version-file FILE --trusted-version-file FILE
             --prior-tags FILE --repository-id ID

Exit codes: 0 verified; 1 refusal; 2 invalid usage or input.
"""

import argparse
import json
from pathlib import Path
import re
import sys


# The sole repository owner (spec AC1/AC4, B2 amendment: no App, no bot proxy).
OWNER_ID = 50957846
OWNER_TYPE = "User"
BUILD_WORKFLOW_NAME = "Release build"
BUILD_WORKFLOW_PATH = ".github/workflows/release.yml"

VERSION_RE = re.compile(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)")
COMMIT_RE = re.compile(r"[0-9a-f]{40}")
POSITIVE_INT_RE = re.compile(r"[1-9][0-9]*")
MAX_JSON_BYTES = 4 << 20


class Refusal(Exception):
    """Eligibility evidence was present but does not authorize a release."""


class UsageError(Exception):
    """Invalid arguments or inputs; nothing was verified."""


def deny(message):
    raise Refusal(message)


def load_json(path, label):
    try:
        raw = Path(path).read_bytes()
    except OSError as exc:
        raise UsageError(f"cannot read {label} {path}: {exc}")
    if len(raw) > MAX_JSON_BYTES:
        raise UsageError(f"{label} {path} exceeds the size bound")
    try:
        return json.loads(raw.decode("utf-8"))
    except (UnicodeDecodeError, ValueError) as exc:
        raise UsageError(f"{label} {path} is not valid JSON: {exc}")


def load_object(path, label):
    value = load_json(path, label)
    if not isinstance(value, dict):
        raise UsageError(f"{label} {path} must be a JSON object")
    return value


def load_array(path, label):
    value = load_json(path, label)
    if not isinstance(value, list):
        raise UsageError(f"{label} {path} must be a JSON array")
    return value


def positive_int(value, label):
    if isinstance(value, bool):
        raise UsageError(f"{label} must be a positive integer")
    if isinstance(value, int):
        number = value
    elif isinstance(value, str) and POSITIVE_INT_RE.fullmatch(value):
        number = int(value)
    else:
        raise UsageError(f"{label} must be a positive integer")
    if number <= 0:
        raise UsageError(f"{label} must be a positive integer")
    return number


def commit_sha(value, label):
    if not isinstance(value, str) or not COMMIT_RE.fullmatch(value):
        raise UsageError(f"{label} must be a full lowercase 40-character SHA")
    return value


def tag_version(value):
    if not isinstance(value, str) or not VERSION_RE.fullmatch(value):
        raise UsageError(f"version must be a stable vMAJOR.MINOR.PATCH tag, got {value!r}")
    return value


def version_tuple(text, label):
    tag = tag_version(text.strip())
    return tuple(int(part) for part in tag[1:].split("."))


def read_version_file(path, label):
    try:
        raw = Path(path).read_text(encoding="utf-8")
    except (OSError, UnicodeDecodeError) as exc:
        raise UsageError(f"cannot read {label} {path}: {exc}")
    lines = [line.strip() for line in raw.splitlines() if line.strip()]
    if len(lines) != 1:
        deny(f"{label} must contain exactly one release-version line, got {lines!r}")
    return tag_version(lines[0])


def read_prior_tags(path):
    try:
        raw = Path(path).read_text(encoding="utf-8")
    except (OSError, UnicodeDecodeError) as exc:
        raise UsageError(f"cannot read prior tags {path}: {exc}")
    names = []
    for line in raw.splitlines():
        name = line.strip()
        if not name:
            continue
        if name.startswith("refs/tags/"):
            name = name[len("refs/tags/"):]
        if VERSION_RE.fullmatch(name):
            names.append(name)
    return names


def repo_id_of(value, label, expected):
    if not isinstance(value, dict):
        deny(f"{label} is missing from the GitHub record")
    number = value.get("id")
    if isinstance(number, bool) or not isinstance(number, int) or number != expected:
        deny(f"{label} id {number!r} is not this repository ({expected})")
    return number


def actor_is_owner(value, label):
    if not isinstance(value, dict):
        deny(f"{label} is missing from the GitHub record")
    if value.get("id") != OWNER_ID:
        deny(f"{label} id {value.get('id')!r} is not the repository owner ({OWNER_ID})")
    if value.get("type") != OWNER_TYPE:
        deny(f"{label} type {value.get('type')!r} is not a human account")
    return value


def cmd_tag_run(args):
    tag = tag_version(args.tag)
    commit = commit_sha(args.tag_commit, "tag commit")
    repository_id = positive_int(args.repository_id, "repository id")
    workflow_id = positive_int(args.workflow_id, "build workflow id")
    run_id = positive_int(args.run_id, "build run id")

    run = load_object(args.run, "build run")
    if positive_int(run.get("id"), "build run id in the record") != run_id:
        deny("the run record is not the build run named by the event payload")
    if run.get("status") != "completed":
        deny(f"build run status is {run.get('status')!r}, not completed")
    if run.get("conclusion") != "success":
        deny(f"build run conclusion is {run.get('conclusion')!r}, not success")
    if run.get("event") != "push":
        deny(f"build run event is {run.get('event')!r}, not a tag push")
    if run.get("path") != BUILD_WORKFLOW_PATH:
        deny(f"build run path is {run.get('path')!r}, not {BUILD_WORKFLOW_PATH}")
    if run.get("name") != BUILD_WORKFLOW_NAME:
        deny(f"build run workflow name is {run.get('name')!r}, not {BUILD_WORKFLOW_NAME!r}")
    if positive_int(run.get("workflow_id"), "build workflow id in the record") != workflow_id:
        deny("the run record's workflow id is not the reviewed build workflow")

    repo_id_of(run.get("repository"), "repository", repository_id)
    repo_id_of(run.get("head_repository"), "head repository", repository_id)
    actor_is_owner(run.get("actor"), "build run actor")

    head_sha = commit_sha(run.get("head_sha"), "build run head SHA")
    if head_sha != commit:
        deny(f"build run head {head_sha} is not the tag commit {commit}")
    if run.get("head_branch") != tag:
        deny(f"build run head branch {run.get('head_branch')!r} is not {tag!r}")

    display = run.get("display_title")
    if not isinstance(display, str):
        deny("build run has no rendered run-name")
    fields = display.split("|")
    if len(fields) != 4:
        deny(f"run-name {display!r} does not carry ref|created|sender|sha")
    ref, created, sender, rendered = fields
    if ref != f"refs/tags/{tag}":
        deny(f"run-name ref {ref!r} is not refs/tags/{tag}")
    if created != "true":
        deny(f"run-name creation flag is {created!r}; only a new tag may release")
    if sender != str(OWNER_ID):
        deny(f"run-name sender {sender!r} is not the repository owner ({OWNER_ID})")
    if rendered != commit:
        deny(f"run-name SHA {rendered!r} is not the tag commit {commit}")

    tagged_blob = commit_sha(args.tagged_workflow_blob, "tagged build workflow blob")
    reviewed_blob = commit_sha(args.reviewed_workflow_blob, "reviewed build workflow blob")
    if tagged_blob != reviewed_blob:
        deny("the build workflow at the tag commit is not the reviewed workflow on the default branch")

    ref_object = load_object(args.tag_ref, "tag ref")
    if ref_object.get("ref") != f"refs/tags/{tag}":
        deny(f"tag ref {ref_object.get('ref')!r} is not refs/tags/{tag}")
    target = ref_object.get("object")
    if not isinstance(target, dict):
        deny("tag ref has no target object")
    target_type = target.get("type")
    target_sha = commit_sha(target.get("sha"), "tag target SHA")
    tag_object = load_json(args.tag_object, "tag object")
    if target_type == "commit":
        if tag_object is not None:
            deny("a lightweight tag cannot carry an annotated tag object")
        peeled = target_sha
    elif target_type == "tag":
        if not isinstance(tag_object, dict):
            deny("an annotated tag must supply its tag object")
        if tag_object.get("tag") != tag:
            deny("the annotated tag object names a different tag")
        if tag_object.get("sha") not in (None, target_sha):
            deny("the annotated tag object is not the ref's target")
        inner = tag_object.get("object")
        if not isinstance(inner, dict) or inner.get("type") != "commit":
            deny("the annotated tag does not point at a commit")
        peeled = commit_sha(inner.get("sha"), "annotated tag peeled commit")
    else:
        deny(f"tag target type {target_type!r} is not a commit or annotated tag")
    if peeled != commit:
        deny(f"tag {tag} peels to {peeled}, not the build commit {commit}")

    print(
        f"release_eligibility: tag-run verified {display} "
        f"run {run_id} workflow {workflow_id} path {BUILD_WORKFLOW_PATH} "
        f"repo {repository_id} tag {tag}"
    )


def cmd_promotion(args):
    merge = commit_sha(args.merge_sha, "merge commit")
    parent1 = commit_sha(args.parent1, "first parent")
    parent2 = commit_sha(args.parent2, "second parent")
    repository_id = positive_int(args.repository_id, "repository id")
    tag = tag_version(args.version)
    if parent1 == parent2 or merge in (parent1, parent2):
        deny("the tagged commit is not a two-parent merge")

    merge_version = read_version_file(args.merge_version_file, "merged tree release-version")
    parent_version = read_version_file(args.parent_version_file, "first-parent release-version")
    trusted_version = read_version_file(args.trusted_version_file, "trusted main release-version")
    if merge_version != tag:
        deny(f"the merged tree's release-version {merge_version} is not the tag {tag}")
    if trusted_version != tag:
        deny(f"trusted main's release-version {trusted_version} is not the tag {tag}; "
             "a changed base invalidates this promotion evidence")
    if parent_version == merge_version:
        deny("the promotion does not change release-version relative to its first parent")
    merge_tuple = version_tuple(merge_version, "merged tree version")
    parent_tuple = version_tuple(parent_version, "first-parent version")
    if not merge_tuple > parent_tuple:
        deny(f"version {merge_version} does not strictly increase over {parent_version}")

    earlier = [name for name in read_prior_tags(args.prior_tags) if name != tag]
    newer = [name for name in earlier if not merge_tuple > version_tuple(name, "prior tag")]
    if newer:
        deny(f"version {merge_version} is not greater than prior release tag(s) "
             f"{', '.join(sorted(newer))}")

    pulls = load_array(args.pulls, "associated pull requests")
    same_merge = [
        item for item in pulls
        if isinstance(item, dict)
        and item.get("merged") is True
        and item.get("merge_commit_sha") == merge
    ]
    if not same_merge:
        deny(f"no merged promotion PR has merge commit {merge}")
    if len(same_merge) > 1:
        deny(f"{len(same_merge)} merged PRs claim merge commit {merge}; refusing ambiguity")
    pull = same_merge[0]
    number = positive_int(pull.get("number"), "promotion PR number")

    base = pull.get("base")
    head = pull.get("head")
    if not isinstance(base, dict) or not isinstance(head, dict):
        deny("promotion PR has no base/head record")
    if base.get("ref") != "main":
        deny(f"promotion PR base {base.get('ref')!r} is not main")
    if head.get("ref") != "dev":
        deny(f"promotion PR head {head.get('ref')!r} is not dev")
    repo_id_of(base.get("repo"), "promotion PR base repository", repository_id)
    repo_id_of(head.get("repo"), "promotion PR head repository", repository_id)
    actor_is_owner(pull.get("merged_by"), "promotion PR merger")

    head_sha = commit_sha(head.get("sha"), "promotion PR head SHA")
    if head_sha != parent2:
        deny(f"promotion PR head {head_sha} is not the merge's second parent {parent2}")

    print(
        f"release_eligibility: promotion verified PR #{number} {head_sha} -> {merge} "
        f"first-parent {parent1} version {tag} repo {repository_id} "
        f"({parent_version} -> {merge_version})"
    )
    return 0


def build_parser():
    parser = argparse.ArgumentParser(description=__doc__,
                                     formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="command", required=True)

    run = sub.add_parser("tag-run", help="verify the original, owner-created tag push run")
    run.add_argument("--run", required=True)
    run.add_argument("--run-id", required=True)
    run.add_argument("--workflow-id", required=True)
    run.add_argument("--tagged-workflow-blob", required=True)
    run.add_argument("--reviewed-workflow-blob", required=True)
    run.add_argument("--tag-ref", required=True)
    run.add_argument("--tag-object", required=True)
    run.add_argument("--tag", required=True)
    run.add_argument("--tag-commit", required=True)
    run.add_argument("--repository-id", required=True)
    run.set_defaults(func=cmd_tag_run)

    promotion = sub.add_parser("promotion", help="verify the merged dev -> main promotion")
    promotion.add_argument("--pulls", required=True)
    promotion.add_argument("--merge-sha", required=True)
    promotion.add_argument("--parent1", required=True)
    promotion.add_argument("--parent2", required=True)
    promotion.add_argument("--version", required=True)
    promotion.add_argument("--merge-version-file", required=True)
    promotion.add_argument("--parent-version-file", required=True)
    promotion.add_argument("--trusted-version-file", required=True)
    promotion.add_argument("--prior-tags", required=True)
    promotion.add_argument("--repository-id", required=True)
    promotion.set_defaults(func=cmd_promotion)
    return parser


def main(argv=None):
    args = build_parser().parse_args(argv)
    try:
        return args.func(args) or 0
    except Refusal as exc:
        print(f"release_eligibility: refused: {exc}", file=sys.stderr)
        return 1
    except UsageError as exc:
        print(f"release_eligibility: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
