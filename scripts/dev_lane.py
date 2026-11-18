#!/usr/bin/env python3
"""Dev-lane decisions: wake identity/scope, one head result, promotion admission.

Workflows loaded from the default branch fetch API documents, run git and write
what these pure functions return. No network, container, remote write, approval
or release verdict happens here.
"""
import argparse
import json
import re
import sys
from pathlib import Path

CHECK = "dev-gate"
PROMOTION = "promotion-admission"
NON_PRODUCT = ("docs/", "specs/")  # everything else is product work
SHA = re.compile(r"^[0-9a-f]{40}$")
NUM = re.compile(r"^[0-9]{1,20}$")
VER = re.compile(r"^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$")
# Only a positively recognised path may be non-product: a traversal, an absolute
# path or an odd character widens the gate instead of narrowing it.
SAFE = re.compile(r"^[A-Za-z0-9._+@/!~-]+$")


class Refused(ValueError):
    """An input this module will not interpret as evidence."""


def read(path):
    return json.loads(Path(path).read_text(encoding="utf-8"))


def write(path, document):
    Path(path).write_text(json.dumps(document, indent=2, sort_keys=True) + "\n",
                          encoding="utf-8")


def outputs(path, values):
    """Append `key=value` lines for a step's `$GITHUB_OUTPUT`."""
    if not path:
        return
    with open(path, "a", encoding="utf-8") as handle:
        for key in sorted(values):
            value = str(values[key])
            if "\n" in value or "\r" in value:
                raise Refused("line break in output %s" % key)
            handle.write("%s=%s\n" % (key, value))


def sha(value):
    text = str(value or "").lower()
    return text if SHA.match(text) else ""


def num(value):
    text = str(value or "")
    return text if NUM.match(text) else ""


def repo_name(record):
    if not isinstance(record, dict):
        return ""
    owner = record.get("repository")
    return str((owner if isinstance(owner, dict) else record).get("full_name", ""))


def version(text):
    found = VER.match(str(text or "").strip())
    return tuple(int(part) for part in found.groups()) if found else None


def _decision():
    return dict(decision="refused", reason="unresolved", problems=[], event="",
                run_id="", run_attempt="", head_sha="", head_repo="", head_branch="",
                base_sha="", compare_base="")


def _identity(run, d, *, repository_id, repository, run_id, name, path, event):
    """Identity from the API's run document, never from the event payload."""
    if not isinstance(run, dict):
        return ["run-document-invalid"]
    problems = []
    for actual, expected, label in (
            (str(run.get("id", "")), str(run_id), "run-id-mismatch"),
            (run.get("event"), event, "event-mismatch"),
            (run.get("path"), path, "workflow-path-mismatch"),
            (run.get("name"), name, "workflow-name-mismatch"),
            (str((run.get("repository") or {}).get("id", "")), str(repository_id),
             "repository-id-mismatch"),
            (repo_name(run), repository, "repository-name-mismatch")):
        if actual != expected:
            problems.append(label)
    d.update(run_id=num(run.get("id")), run_attempt=num(run.get("run_attempt")) or "1",
             event=str(run.get("event", "")), head_sha=sha(run.get("head_sha")),
             head_branch=str(run.get("head_branch", "")),
             head_repo=repo_name(run.get("head_repository")))
    if not d["head_sha"]:
        problems.append("head-sha-invalid")
    return problems


def _refuse(d, problems, reason):
    d.update(decision="refused", reason=reason, problems=sorted(set(problems)))
    return d


def resolve_push(run, branch, *, repository_id, repository, run_id, name, path,
                 branch_name="dev"):
    """A push wake reports only for the current tip of `dev`."""
    d = _decision()
    d["compare_base"] = "main"
    problems = _identity(run, d, repository_id=repository_id, repository=repository,
                         run_id=run_id, name=name, path=path, event="push")
    if d["head_branch"] != branch_name:
        problems.append("head-branch-mismatch")
    if d["head_repo"] != repository:
        problems.append("head-repository-mismatch")
    if problems:
        return _refuse(d, problems, problems[0])
    if not isinstance(branch, dict):
        return _refuse(d, ["branch-unavailable"], "branch-unavailable")
    tip = sha((branch.get("commit") or {}).get("sha"))
    if branch.get("name") != branch_name or not tip:
        return _refuse(d, ["branch-unavailable"], "branch-unavailable")
    d["base_sha"] = tip
    if tip != d["head_sha"]:
        # A newer tip makes this run's evidence stale: report nothing at all.
        d.update(decision="superseded", reason="branch-advanced")
    else:
        d.update(decision="accepted", reason="push-tip")
    return d


def resolve_pr(run, pulls, *, repository_id, repository, run_id, name, path,
               base_branch="dev"):
    """A pull-request wake reports only for an open dev-based PR's current head."""
    d = _decision()
    d["compare_base"] = base_branch
    problems = _identity(run, d, repository_id=repository_id, repository=repository,
                         run_id=run_id, name=name, path=path, event="pull_request")
    if problems:
        return _refuse(d, problems, problems[0])
    if not isinstance(pulls, list):
        return _refuse(d, ["pull-document-invalid"], "pull-unavailable")
    candidates = [p for p in pulls if isinstance(p, dict)
                  and str((p.get("base") or {}).get("ref", "")) == base_branch]
    if not candidates:
        return _refuse(d, ["no-pull-against-base"], "wrong-base")
    if {str(((p.get("base") or {}).get("repo") or {}).get("full_name", ""))
            for p in candidates} != {repository}:
        return _refuse(d, ["base-repository-mismatch"], "wrong-base")
    same = [p for p in candidates
            if sha((p.get("head") or {}).get("sha")) == d["head_sha"]]
    open_same = [p for p in same if p.get("state") == "open" and not p.get("merged")]
    if len(open_same) > 1:
        return _refuse(d, ["ambiguous-pull"], "ambiguous-pull")
    if not open_same:
        d.update(decision="superseded",
                 reason="pull-closed" if same else "head-moved")
        return d
    pull = open_same[0]
    base_sha = sha((pull.get("base") or {}).get("sha"))
    head_repo = str(((pull.get("head") or {}).get("repo") or {}).get("full_name", ""))
    number = num(pull.get("number"))
    if not base_sha:
        return _refuse(d, ["base-sha-invalid"], "wrong-base")
    if not number:
        return _refuse(d, ["pull-number-invalid"], "pull-unavailable")
    if not re.match(r"^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$", head_repo):
        return _refuse(d, ["head-repository-invalid"], "head-unavailable")
    # A fork's commit is not served by the base repository; checkout uses this.
    d.update(decision="accepted", reason="pull-head", head_repo=head_repo,
             base_sha=base_sha, pr_number=number)
    return d


def classify(names, *, force_product=""):
    """Classify a change set. Unrecognised paths widen the gate to product."""
    unique = sorted({str(n).strip("\r\n") for n in names if str(n).strip()})
    product, docs, odd = [], [], []
    for path in unique:
        if path.startswith("/") or ".." in path.split("/") or not SAFE.match(path):
            odd.append(path)
            product.append(path)
        elif path.startswith(NON_PRODUCT):
            docs.append(path)
        else:
            product.append(path)
    if force_product:
        scope, reason = "product", str(force_product)
    elif not unique:
        scope, reason = "no-change", "empty-diff"
    elif product:
        scope, reason = "product", "product-path"
    else:
        scope, reason = "docs-only", "docs-or-specs-only"
    return dict(scope=scope, reason=reason, product=product[:40], docs=docs[:40],
                odd=odd[:40],
                counts=dict(product=len(product), docs=len(docs), odd=len(odd)))


def check_payload(decision, scope, *, exit_code, gate_revision, run_id, attempt,
                  details_url):
    """One check-run body for the exact candidate head; never invent a pass."""
    if decision.get("decision") != "accepted":
        raise Refused("only an accepted wake may be reported")
    head = sha(decision.get("head_sha"))
    if not head:
        raise Refused("the resolved head is not a full SHA")
    if not isinstance(scope, dict) or scope.get("scope") not in (
            "product", "docs-only", "no-change"):
        raise Refused("a scope decision is required")
    text = str(exit_code).strip()
    ran = text not in ("", "not-run", "none")
    code = int(text) if ran and NUM.match(text) else (1 if ran else None)
    kind = "product gate" if scope["scope"] == "product" else "docs launcher"
    if scope["scope"] == "no-change":
        conclusion, reason, title = "success", "no-change", "no change: nothing to validate"
    elif ran and code == 0:
        conclusion, reason = "success", "validated"
        title = ("product gate passed on this head" if scope["scope"] == "product"
                 else "docs/specs-only change: reviewed docs launcher passed")
    elif ran:
        conclusion, reason, title = "failure", "validation-failed", "%s failed" % kind
    else:
        conclusion, reason, title = "failure", "validation-missing", "%s has no evidence" % kind
    summary = "\n".join([
        "candidate head: %s" % head,
        "wake run: %s attempt %s" % (decision["run_id"], decision["run_attempt"]),
        "reviewed revision: %s" % gate_revision,
        "scope: %s (%s); %d product, %d docs/spec, %d odd path(s)" % (
            scope["scope"], scope["reason"], scope["counts"]["product"],
            scope["counts"]["docs"], scope["counts"]["odd"]),
        "%s: %s" % (kind, "ran, exit %s" % code if ran else "not run"),
        "",
        "From the default-branch workflow `.github/workflows/dev-gate.yml`, using",
        "reviewed tooling with the candidate as data. The Actions app identity is",
        "an accepted source, not exclusive proof: promotion re-runs the gate and",
        "the publisher re-runs quality. Not an approval or release evidence.",
    ])
    body = dict(name=CHECK, head_sha=head, status="completed", conclusion=conclusion,
                output=dict(title=title[:120], summary=summary[:3000]))
    if details_url:
        body["details_url"] = str(details_url)
    return dict(head_sha=head, conclusion=conclusion, reason=reason, body=body)


def _admission(conclusion, reason, title, summary, **facts):
    return dict(conclusion=conclusion, reason=reason, title=title, summary=summary, **facts)


def admit(*, pull, checks, repository, owner, dev_tip, main_tip, merged_tree,
          head_tree, version_head, version_base):
    """Decide a dev -> main promotion from reviewed inputs only.

    Pull and tips come from the API, trees and versions from the trusted
    checkout, evidence from the candidate commit's check runs. A different
    base, head, tree or producer cannot be reused as evidence.
    """
    if not isinstance(pull, dict):
        return _admission("failure", "pull-unavailable", "pull request unavailable", "")
    base, head = pull.get("base") or {}, pull.get("head") or {}
    facts = dict(pr_number=num(pull.get("number")),
                 base_sha=sha(base.get("sha")), head_sha=sha(head.get("sha")),
                 repository=repository, owner=owner)
    base_ref, head_ref = str(base.get("ref", "")), str(head.get("ref", ""))
    if not all(facts[k] for k in ("pr_number", "base_sha", "head_sha")) or not (
            base_ref and head_ref):
        return _admission("failure", "pull-invalid", "pull document incomplete", "", **facts)
    if (base_ref, head_ref) != ("main", "dev"):
        return _admission("success", "not-promotion", "not a dev -> main promotion",
                          "Scope: %s -> %s; admission applies only to dev -> main."
                          % (base_ref or "?", head_ref or "?"), **facts)
    base_repo = str((base.get("repo") or {}).get("full_name", ""))
    head_repo = str((head.get("repo") or {}).get("full_name", ""))
    if base_repo != repository or head_repo != repository:
        return _admission("failure", "fork-dev", "a fork's dev is not the dev branch",
                          "head %s, base %s; both must be %s."
                          % (head_repo or "?", base_repo or "?", repository), **facts)
    if pull.get("state") != "open" or pull.get("merged"):
        return _admission("failure", "pull-closed", "promotion is not open", "", **facts)
    if str((pull.get("user") or {}).get("login", "")) != owner:
        return _admission("failure", "not-owner", "promotion is not owner-authored",
                          "expected author %s" % owner, **facts)
    if facts["head_sha"] != sha(dev_tip):
        return _admission("failure", "head-not-dev-tip", "head is no longer the dev tip",
                          "%s != %s" % (facts["head_sha"], sha(dev_tip)), **facts)
    if facts["base_sha"] != sha(main_tip):
        return _admission("failure", "base-not-current", "base is no longer the main tip",
                          "%s != %s" % (facts["base_sha"], sha(main_tip)), **facts)
    if not sha(merged_tree) or not sha(head_tree):
        return _admission("failure", "merge-tree-unavailable",
                          "prospective merge tree unavailable", "", **facts)
    if sha(merged_tree) != sha(head_tree):
        return _admission("failure", "base-drift", "merge would produce an untested tree",
                          "prospective %s, tested %s; update dev onto main and re-run."
                          % (sha(merged_tree), sha(head_tree)), **facts)
    vh, vb = version(version_head), version(version_base)
    if vb is None or vh is None:
        return _admission("failure", "version-missing",
                          "a readable release-version is missing",
                          "`vX.Y.Z` is required in release-version on main and head.",
                          **facts)
    if vh <= vb:
        return _admission("failure", "version-not-incremented",
                          "release version did not increase",
                          "head v%s, base v%s" % (".".join(map(str, vh)),
                                                  ".".join(map(str, vb))), **facts)
    runs = checks.get("check_runs", []) if isinstance(checks, dict) else checks
    named = [r for r in runs or [] if isinstance(r, dict) and r.get("name") == CHECK]
    if not named:
        return _admission("failure", "evidence-missing", "no dev-gate result", "", **facts)
    accepted = [r for r in named if sha(r.get("head_sha")) == facts["head_sha"]]
    if not accepted:
        return _admission("failure", "evidence-wrong-sha", "dev-gate is for another head",
                          "", **facts)
    if any(r.get("status") != "completed" for r in accepted):
        return _admission("failure", "evidence-incomplete", "dev-gate has not completed",
                          "", **facts)
    if any(str((r.get("app") or {}).get("slug", "")) != "github-actions" for r in accepted):
        return _admission("failure", "evidence-wrong-producer", "unexpected check producer",
                          "", **facts)
    if any(r.get("conclusion") != "success" for r in accepted):
        return _admission("failure", "evidence-failed", "dev-gate did not pass", "", **facts)
    return _admission("success", "accepted", "promotion admission passed",
                      "head %s, base %s, tree %s, v%s -> v%s." % (
                          facts["head_sha"], facts["base_sha"], sha(merged_tree),
                          ".".join(map(str, vb)), ".".join(map(str, vh))), **facts)


# ---------------------------------------------------------------------------
# CLI: thin wrappers; workflows do all IO
# ---------------------------------------------------------------------------


def names_file(path):
    """NUL/newline-separated paths as `git diff -z` emits them."""
    names = []
    for name in re.split(r"[\n\x00]", Path(path).read_text(encoding="utf-8",
                                                           errors="replace")):
        if not name.strip():
            continue
        candidate = name.strip()
        if len(candidate) >= 2 and candidate[0] == candidate[-1] == '"':
            try:
                decoded = json.loads(candidate)
            except json.JSONDecodeError:
                decoded = None
            if isinstance(decoded, str) and decoded:
                names.append(decoded)
                continue
        names.append(name)
    return names


def _emit(d, args):
    write(args.out, d)
    outputs(args.github_output, {
        "accepted": "true" if d["decision"] == "accepted" else "false",
        "decision": d["decision"], "reason": d["reason"], "head_sha": d["head_sha"],
        "head_repo": d["head_repo"], "base_sha": d["base_sha"],
        "compare_base": d["compare_base"]})
    if d["decision"] == "refused":
        print("dev-lane: refused (%s)" % ", ".join(d["problems"]), file=sys.stderr)
        return 1
    print("dev-lane: %s (%s) for %s" % (d["decision"], d["reason"], d["head_sha"]))
    return 0


def _wake(args):
    return dict(repository_id=args.repository_id, repository=args.repository,
                run_id=args.run_id, name=args.workflow_name, path=args.workflow_path)


def cmd_push(args):
    return _emit(resolve_push(read(args.run), read(args.branch), **_wake(args)), args)


def cmd_pull(args):
    return _emit(resolve_pr(read(args.run), read(args.pulls), **_wake(args)), args)


def cmd_head(args):
    head = sha(read(args.run).get("head_sha"))
    if not head:
        print("dev-lane: no full head SHA", file=sys.stderr)
        return 1
    print(head)
    return 0


def cmd_scope(args):
    scope = classify(names_file(args.names) if args.names else [],
                     force_product=args.force_product)
    write(args.out, scope)
    outputs(args.github_output, {"scope": scope["scope"]})
    print("dev-lane: scope %s (%s)" % (scope["scope"], scope["reason"]))
    return 0


def cmd_check(args):
    record = check_payload(read(args.decision), read(args.scope),
                           exit_code=args.exit_code, gate_revision=args.gate_revision,
                           run_id=args.run_id, attempt=args.attempt,
                           details_url=args.details_url)
    write(args.out, record["body"])
    outputs(args.github_output, {"conclusion": record["conclusion"]})
    print("dev-lane: %s %s (%s)" % (CHECK, record["conclusion"], record["reason"]))
    return 0


def cmd_admit(args):
    d = admit(pull=read(args.pull), checks=read(args.checks), repository=args.repository,
              owner=args.owner, dev_tip=args.dev_tip, main_tip=args.main_tip,
              merged_tree=sha(args.merged_tree), head_tree=sha(args.head_tree),
              version_head=args.version_head, version_base=args.version_base)
    head = sha(d.get("head_sha"))
    if not head:
        print("dev-lane: no candidate head SHA to report", file=sys.stderr)
        return 1
    body = dict(name=PROMOTION, head_sha=head, status="completed",
                conclusion=d["conclusion"],
                output=dict(title=d["title"][:120], summary=d["summary"][:3000]))
    if args.details_url:
        body["details_url"] = str(args.details_url)
    write(args.out, body)
    outputs(args.github_output, {"conclusion": d["conclusion"], "reason": d["reason"]})
    print("dev-lane: %s %s (%s)" % (PROMOTION, d["conclusion"], d["reason"]))
    return 0


def build_parser():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    sub = parser.add_subparsers(dest="command", required=True)
    io = argparse.ArgumentParser(add_help=False)
    io.add_argument("--out", required=True)
    io.add_argument("--github-output", default="")
    wake = argparse.ArgumentParser(add_help=False)
    wake.add_argument("--run", required=True)
    for flag in ("--repository-id", "--repository", "--run-id",
                 "--workflow-name", "--workflow-path"):
        wake.add_argument(flag, required=True)
    p = sub.add_parser("resolve-push", parents=[wake, io])
    p.add_argument("--branch", required=True)
    p.set_defaults(handler=cmd_push)
    p = sub.add_parser("resolve-pr", parents=[wake, io])
    p.add_argument("--pulls", required=True)
    p.set_defaults(handler=cmd_pull)
    p = sub.add_parser("head-sha")
    p.add_argument("--run", required=True)
    p.set_defaults(handler=cmd_head)
    p = sub.add_parser("scope", parents=[io])
    p.add_argument("--names", default="")
    p.add_argument("--force-product", default="")
    p.set_defaults(handler=cmd_scope)
    p = sub.add_parser("check", parents=[io])
    for flag in ("--decision", "--scope", "--gate-revision", "--run-id", "--attempt"):
        p.add_argument(flag, required=True)
    p.add_argument("--exit-code", default="not-run")
    p.add_argument("--details-url", default="")
    p.set_defaults(handler=cmd_check)
    p = sub.add_parser("admit", parents=[io])
    for flag in ("--pull", "--checks", "--repository", "--owner",
                 "--dev-tip", "--main-tip"):
        p.add_argument(flag, required=True)
    for flag in ("--merged-tree", "--head-tree", "--version-head",
                 "--version-base", "--details-url"):
        p.add_argument(flag, default="")
    p.set_defaults(handler=cmd_admit)
    return parser


def main(argv=None):
    args = build_parser().parse_args(list(sys.argv[1:] if argv is None else argv))
    try:
        return args.handler(args)
    except (Refused, OSError, json.JSONDecodeError, KeyError) as error:
        print("dev-lane: %s" % error, file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
