"""Feature 023 B2/T3 - short offline fixtures for the development lane.

The `Dev wake` file is candidate content and may only complete. Two listeners
loaded from the default branch do the judging: `dev-gate.yml` resolves a wake,
compares exact commits with local git, runs reviewed tooling with the candidate
as data and reports one result for the candidate head; `dev-promotion.yml`
admits only an owner-authored, same-repository `dev` -> `main` promotion with
current tips, a prospective merge tree equal to the tested tree, an increased
`release-version` and a passing `dev-gate`.

These tests execute shipped step text with stub `gh` and a scratch git
repository. They do not prove hosted delivery, permissions, rules, check-run
producer identity, container execution or publication behaviour.
"""
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
HELPER = ROOT / "scripts" / "dev_lane.py"
WAKE = ROOT / ".github/workflows/dev-wake.yml"
GATE = ROOT / ".github/workflows/dev-gate.yml"
PROMOTION = ROOT / ".github/workflows/dev-promotion.yml"
DOCS = ROOT / ".github/workflows/docs.yml"
TRUSTED = ROOT / "tests/harness/trusted_ci.py"

REPO = "owner/repository"
OWNER = "owner"
REPO_ID = "987654"
RUN, HEAD, BASE, MAIN, TIP = "4242", "1" * 40, "2" * 40, "3" * 40, "4" * 40
TREE = "a" * 40
WAKE_NAME, WAKE_PATH = "Dev wake", ".github/workflows/dev-wake.yml"


def load():
    spec = importlib.util.spec_from_file_location("dev_lane", HELPER)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def yaml(path):
    import yaml as module
    return module.safe_load(Path(path).read_text())


def text(path):
    return Path(path).read_text(encoding="utf-8")


def job(document, name):
    return document["jobs"][name]


def step(document, job_name, step_name):
    return next(s for s in job(document, job_name)["steps"]
                if s.get("name") == step_name)["run"]


def run_document(**changes):
    document = dict(id=int(RUN), run_attempt=1, event="push", name=WAKE_NAME,
                    path=WAKE_PATH, status="completed", conclusion="success",
                    head_branch="dev", head_sha=HEAD,
                    repository=dict(id=int(REPO_ID), full_name=REPO),
                    head_repository=dict(id=int(REPO_ID), full_name=REPO))
    document.update(changes)
    return document


def pull_document(**changes):
    document = dict(number=7, state="open", merged=False, draft=False,
                    user=dict(login=OWNER),
                    base=dict(ref="dev", sha=BASE, repo=dict(full_name=REPO)),
                    head=dict(ref="dev", sha=HEAD, repo=dict(full_name=REPO)))
    document.update(changes)
    return document


class HelperCase(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.lane = load()

    def push(self, run=None, branch=None):
        return self.lane.resolve_push(
            run if run is not None else run_document(),
            branch if branch is not None else dict(name="dev", commit=dict(sha=HEAD)),
            repository_id=REPO_ID, repository=REPO, run_id=RUN,
            name=WAKE_NAME, path=WAKE_PATH)

    def pull(self, run=None, pulls=None):
        return self.lane.resolve_pr(
            run or run_document(event="pull_request", head_branch="feature"),
            pulls if pulls is not None else [pull_document()],
            repository_id=REPO_ID, repository=REPO, run_id=RUN,
            name=WAKE_NAME, path=WAKE_PATH)


class ShellQuotingTests(unittest.TestCase):
    def test_helper_arguments_are_a_bash_array_not_a_split_string(self):
        # Found on a hosted run: WAKE_NAME is "Dev wake", so an unquoted string
        # variable split it into two arguments and the helper refused.
        text = GATE.read_text(encoding="utf-8")
        self.assertIn("common=(", text)
        self.assertNotIn("$common", text)
        self.assertEqual(text.count('"${common[@]}"'), 2)


class WakeResolutionTests(HelperCase):
    def test_push_at_tip_is_accepted_and_a_moved_tip_is_superseded(self):
        accepted = self.push()
        self.assertEqual(("accepted", "push-tip", "main"),
                         (accepted["decision"], accepted["reason"],
                          accepted["compare_base"]))
        superseded = self.push(branch=dict(name="dev", commit=dict(sha=MAIN)))
        self.assertEqual(("superseded", "branch-advanced", HEAD),
                         (superseded["decision"], superseded["reason"],
                          superseded["head_sha"]))

    def test_identity_mismatches_are_named_refusals(self):
        cases = {
            "run-id-mismatch": dict(id=999),
            "event-mismatch": dict(event="pull_request"),
            "workflow-path-mismatch": dict(path=".github/workflows/ci.yml"),
            "workflow-name-mismatch": dict(name="CI"),
            "repository-id-mismatch": dict(repository=dict(id=1, full_name=REPO)),
            "repository-name-mismatch": dict(
                repository=dict(id=int(REPO_ID), full_name="other/repo")),
            "head-branch-mismatch": dict(head_branch="main"),
            "head-repository-mismatch": dict(
                head_repository=dict(id=1, full_name="fork/repo")),
            "head-sha-invalid": dict(head_sha="short"),
        }
        for problem, changes in cases.items():
            with self.subTest(problem=problem):
                decision = self.push(run=run_document(**changes))
                self.assertEqual("refused", decision["decision"])
                self.assertIn(problem, decision["problems"])

    def test_an_unreadable_branch_refuses_instead_of_assuming(self):
        for branch in ({}, {"name": "main", "commit": {"sha": HEAD}},
                       {"name": "dev", "commit": {"sha": "short"}}):
            with self.subTest(branch=branch):
                self.assertEqual("refused", self.push(branch=branch)["decision"])

    def test_pr_head_freshness_base_and_fork_cases(self):
        accepted = self.pull()
        self.assertEqual(("accepted", "7", REPO, BASE),
                         (accepted["decision"], accepted["pr_number"],
                          accepted["head_repo"], accepted["base_sha"]))
        fork = self.pull(pulls=[pull_document(
            head=dict(ref="feature", sha=HEAD,
                      repo=dict(full_name="contributor/repository")))])
        self.assertEqual("contributor/repository", fork["head_repo"])
        for pulls, reason in (([pull_document(head=dict(
                ref="feature", sha=MAIN, repo=dict(full_name=REPO)))], "head-moved"),
                              ([pull_document(state="closed")], "pull-closed")):
            with self.subTest(reason=reason):
                d = self.pull(pulls=pulls)
                self.assertEqual("superseded", d["decision"])
                self.assertEqual(reason, d["reason"])
        for pulls, problem in (([pull_document(base=dict(
                ref="main", sha=BASE, repo=dict(full_name=REPO)))], "no-pull-against-base"),
                               ([pull_document(number=7), pull_document(number=8)],
                                "ambiguous-pull"),
                               ({"not": "a list"}, "pull-document-invalid")):
            with self.subTest(problem=problem):
                d = self.pull(pulls=pulls)
                self.assertEqual("refused", d["decision"])
                self.assertIn(problem, d["problems"])


class ScopeTests(HelperCase):
    def test_classification_is_conservative(self):
        cases = {
            "docs-only": (["docs/a.md", "specs/023/tasks.md"], "docs-only"),
            "product": (["README.md", "internal/fleet/fleet.go"], "product"),
            "mixed": (["docs/a.md", "Taskfile.yml"], "product"),
            "empty": ([], "no-change"),
            "traversal": (["../escape"], "product"),
            "absolute": (["/etc/passwd"], "product"),
            "odd": (["docs/a\tb.md"], "product"),
        }
        for name, (paths, expected) in cases.items():
            with self.subTest(case=name):
                self.assertEqual(expected, self.lane.classify(paths)["scope"])
        forced = self.lane.classify([], force_product="change-set-unavailable")
        self.assertEqual(("product", "change-set-unavailable"),
                         (forced["scope"], forced["reason"]))

    def test_the_shipped_scope_step_uses_git_nul_paths_and_widens_on_failure(self):
        document = yaml(GATE)
        body = step(document, "gate", "Classify change set with local git (NUL paths, old and new)")
        with tempfile.TemporaryDirectory(prefix="dev-scope-") as temporary:
            root = Path(temporary)
            trusted = root / "trusted"
            (trusted / "scripts").mkdir(parents=True)
            (trusted / "scripts" / "dev_lane.py").write_bytes(HELPER.read_bytes())
            self.git(trusted, ["init", "-q", "."])
            self.git(trusted, ["config", "user.email", "a@b.c"])
            self.git(trusted, ["config", "user.name", "t"])
            (trusted / "docs").mkdir()
            (trusted / "docs" / "a.md").write_text("base")
            self.git(trusted, ["add", "."])
            self.git(trusted, ["commit", "-qm", "base"])
            base = self.git(trusted, ["rev-parse", "HEAD"]).strip()
            self.git(trusted, ["update-ref", "refs/remotes/origin/main", base])
            (trusted / "docs" / "a.md").write_text("head")
            self.git(trusted, ["commit", "-qam", "docs"])
            head = self.git(trusted, ["rev-parse", "HEAD"]).strip()

            state = root / "state"
            state.mkdir()
            environment = dict(os.environ, TRUSTED=str(trusted), STATE=str(state),
                               HEAD_SHA=head, HEAD_REPO=REPO, COMPARE_BASE="main",
                               SERVER="https://github.invalid",
                               GITHUB_OUTPUT=str(root / "out"))
            (root / "out").touch()
            result = self.shell(body, environment, root)
            self.assertEqual(0, result.returncode, result.stderr.decode())
            scope = json.loads((state / "scope.json").read_text())
            self.assertEqual("docs-only", scope["scope"])
            self.assertEqual(["docs/a.md"], scope["docs"])

            # A renamed product file contributes both the old and the new path.
            self.git(trusted, ["mv", "docs/a.md", "internal.go"])
            self.git(trusted, ["commit", "-qm", "rename"])
            renamed = self.git(trusted, ["rev-parse", "HEAD"]).strip()
            environment["HEAD_SHA"] = renamed
            result = self.shell(body, environment, root)
            self.assertEqual(0, result.returncode, result.stderr.decode())
            scope = json.loads((state / "scope.json").read_text())
            self.assertEqual("product", scope["scope"])
            self.assertIn("internal.go", scope["product"])
            self.assertIn("docs/a.md", scope["docs"])

            # An unavailable head widens the gate; no truncated/empty pass.
            environment["HEAD_SHA"] = "f" * 40
            result = self.shell(body, environment, root)
            self.assertEqual(0, result.returncode, result.stderr.decode())
            scope = json.loads((state / "scope.json").read_text())
            self.assertEqual(("product", "head-unavailable"),
                             (scope["scope"], scope["reason"]))

    @staticmethod
    def git(root, args):
        return subprocess.run(["git", "-C", str(root)] + args, check=True,
                              capture_output=True, text=True).stdout

    @staticmethod
    def shell(body, environment, cwd):
        script = cwd / "step.sh"
        script.write_text(body, encoding="utf-8")
        return subprocess.run(["bash", str(script)], cwd=str(cwd),
                              env=environment, capture_output=True, timeout=120)


class CheckTests(HelperCase):
    def payload(self, scope, exit_code):
        return self.lane.check_payload(
            dict(decision="accepted", head_sha=HEAD, run_id=RUN, run_attempt="1"),
            self.lane.classify(scope), exit_code=exit_code, gate_revision=MAIN,
            run_id="5000", attempt="1", details_url="https://example.invalid/5000")

    def test_conclusions_never_invent_a_product_pass(self):
        cases = {
            "product-pass": (["internal/fleet/fleet.go"], "0", "success", "validated"),
            "product-fail": (["internal/fleet/fleet.go"], "1", "failure", "validation-failed"),
            "product-none": (["internal/fleet/fleet.go"], "not-run", "failure", "validation-missing"),
            "docs-pass": (["docs/a.md"], "0", "success", "validated"),
            "docs-none": (["docs/a.md"], "not-run", "failure", "validation-missing"),
            "empty": ([], "not-run", "success", "no-change"),
        }
        for name, (paths, code, conclusion, reason) in cases.items():
            with self.subTest(case=name):
                record = self.payload(paths, code)
                expected = (conclusion, reason)
                self.assertEqual(expected, (record["conclusion"], record["reason"]))
                self.assertEqual(HEAD, record["body"]["head_sha"])
                self.assertEqual("dev-gate", record["body"]["name"])
                self.assertEqual("completed", record["body"]["status"])

    def test_only_an_accepted_head_may_be_reported(self):
        for decision in (dict(decision="superseded", head_sha=HEAD),
                         dict(decision="accepted", head_sha="bad")):
            with self.subTest(decision=decision):
                with self.assertRaises(self.lane.Refused):
                    self.lane.check_payload(decision, self.lane.classify([]),
                                            exit_code="0", gate_revision=MAIN,
                                            run_id="1", attempt="1", details_url="")


def promotion_pull(**changes):
    base = dict(ref="main", sha=BASE, repo=dict(full_name=REPO))
    base.update(changes.pop("base", {}))
    head = dict(ref="dev", sha=HEAD, repo=dict(full_name=REPO))
    head.update(changes.pop("head", {}))
    return pull_document(base=base, head=head, **changes)


class AdmissionTests(HelperCase):
    def admit(self, pull=None, checks=None, **facts):
        values = dict(pull=pull if pull is not None else promotion_pull(),
                      checks=checks if checks is not None else dict(check_runs=[
                          dict(name="dev-gate", head_sha=HEAD, status="completed",
                               conclusion="success", app=dict(slug="github-actions"))]),
                      repository=REPO, owner=OWNER, dev_tip=HEAD, main_tip=BASE,
                      merged_tree=TREE, head_tree=TREE,
                      version_head="v0.14.0", version_base="v0.13.0")
        values.update(facts)
        return self.lane.admit(**values)

    def test_only_a_complete_promotion_is_admitted(self):
        accepted = self.admit()
        self.assertEqual(("success", "accepted"),
                         (accepted["conclusion"], accepted["reason"]))
        self.assertEqual(HEAD, accepted["head_sha"])

    def test_scope_and_adversarial_facts(self):
        cases = [
            ("success", "not-promotion", dict(
                pull=promotion_pull(base=dict(ref="feature", sha=BASE,
                                             repo=dict(full_name=REPO))))),
            ("success", "not-promotion", dict(
                pull=promotion_pull(head=dict(ref="topic", sha=HEAD,
                                             repo=dict(full_name=REPO))))),
            ("failure", "fork-dev", dict(pull=promotion_pull(
                head=dict(ref="dev", sha=HEAD,
                          repo=dict(full_name="fork/repository"))))),
            ("failure", "not-owner", dict(pull=promotion_pull(
                user=dict(login="contributor")))),
            ("failure", "head-not-dev-tip", dict(dev_tip=MAIN)),
            ("failure", "base-not-current", dict(main_tip=MAIN)),
            ("failure", "base-drift", dict(merged_tree="b" * 40)),
            ("failure", "version-missing", dict(version_head="")),
            ("failure", "version-missing", dict(version_base="")),
            ("failure", "version-not-incremented", dict(version_head="v0.13.0")),
            ("failure", "evidence-missing", dict(checks=dict(check_runs=[]))),
            ("failure", "evidence-wrong-sha", dict(checks=dict(check_runs=[
                dict(name="dev-gate", head_sha=MAIN, status="completed",
                     conclusion="success", app=dict(slug="github-actions"))]))),
            ("failure", "evidence-wrong-producer", dict(checks=dict(check_runs=[
                dict(name="dev-gate", head_sha=HEAD, status="completed",
                     conclusion="success", app=dict(slug="other-app"))]))),
            ("failure", "evidence-failed", dict(checks=dict(check_runs=[
                dict(name="dev-gate", head_sha=HEAD, status="completed",
                     conclusion="failure", app=dict(slug="github-actions"))]))),
        ]
        for conclusion, reason, kwargs in cases:
            with self.subTest(reason=reason):
                decision = self.admit(**kwargs)
                self.assertEqual((conclusion, reason),
                                 (decision["conclusion"], decision["reason"]))

    def test_numeric_version_order_not_lexical(self):
        self.assertEqual("failure", self.admit(version_head="v0.9.0",
                                               version_base="v0.10.0")["conclusion"])
        self.assertEqual("success", self.admit(version_head="v0.10.1",
                                               version_base="v0.10.0")["conclusion"])


class WiringTests(unittest.TestCase):
    def test_wake_is_candidate_content_with_no_rights(self):
        document = yaml(WAKE)
        wake = job(document, "wake")
        self.assertEqual({}, document["permissions"])
        self.assertEqual({}, wake["permissions"])
        self.assertNotIn("uses", [" ".join(wake["steps"][0])])
        self.assertNotIn("uses:", text(WAKE))
        triggers = document[True]
        self.assertEqual(["dev"], triggers["push"]["branches"])
        self.assertEqual(["dev"], triggers["pull_request"]["branches"])

    def test_gate_permissions_include_the_two_new_api_read_scopes(self):
        document = yaml(GATE)
        gate = job(document, "gate")
        self.assertEqual({"actions": "read", "contents": "read",
                          "pull-requests": "read", "checks": "write"},
                         gate["permissions"])
        self.assertIn("Dev wake", document[True]["workflow_run"]["workflows"])

    def test_reviewed_sources_and_candidate_binding(self):
        document = yaml(GATE)
        name = lambda s: next(x for x in job(document, "gate")["steps"]
                              if x.get("name") == s)
        trusted = name("Checkout reviewed default-branch tooling")["with"]
        self.assertEqual("${{ github.sha }}", trusted["ref"])
        self.assertEqual("gate-trusted", trusted["path"])
        self.assertEqual(0, trusted["fetch-depth"])
        candidate = name("Checkout the candidate as data only")["with"]
        self.assertEqual("${{ steps.resolve.outputs.head_sha }}", candidate["ref"])
        self.assertEqual("gate-candidate", candidate["path"])
        binding = name("Bind the candidate tree to the resolved head")["run"]
        self.assertIn('git -C "$CANDIDATE" rev-parse HEAD', binding)
        self.assertIn('[ "$actual" = "$HEAD_SHA" ]', binding)
        self.assertNotIn("workflow_dispatch", str(document[True]))

    def test_scope_has_no_compare_api_and_docs_runner_is_reviewed(self):
        gate = text(GATE)
        self.assertNotIn("compare/", gate)
        self.assertIn("diff --name-only -z --no-renames", gate)
        docs = step(yaml(GATE), "gate",
                    "Docs/specs only - reviewed main launcher, candidate as data")
        self.assertIn('"$TRUSTED/scripts/docs-ci.py"', docs)
        self.assertIn("--mode pr", docs)
        self.assertIn('"$CANDIDATE"', docs)
        commands = [line for line in docs.splitlines() if not line.lstrip().startswith("#")]
        self.assertNotIn("--artifact", " ".join(commands))
        self.assertNotIn("--candidate-ref", " ".join(commands))

    def test_product_gate_pin_matches_the_trusted_gate(self):
        document = yaml(GATE)
        pin = document["env"]["GATE_IMAGE"]
        import re
        trusted_pin = re.search(r'(?m)^IMAGE\s*=\s*"([^"\n]+)"', text(TRUSTED))
        self.assertIsNotNone(trusted_pin)
        self.assertEqual(trusted_pin.group(1), pin)
        body = step(document, "gate", "Product work - run the reviewed trusted gate")
        self.assertIn('task --dir "$GITHUB_WORKSPACE/gate-trusted" verify-trusted', body)

    def test_promotion_file_is_reviewed_and_main_deploy_is_untouched(self):
        promotion = yaml(PROMOTION)
        self.assertEqual(["main"],
                         promotion[True]["pull_request_target"]["branches"])
        self.assertEqual({}, promotion["permissions"])
        jobs = job(promotion, "admission")
        self.assertEqual({"contents": "read", "pull-requests": "read", "checks": "write"},
                         jobs["permissions"])
        docs = yaml(DOCS)
        self.assertEqual(["main"], docs[True]["push"]["branches"])
        self.assertEqual(["main"], docs[True]["pull_request_target"]["branches"])
        self.assertNotIn("dev", str(docs[True]))
        for name in ("deploy", "rollback"):
            self.assertIn("refs/heads/main", job(docs, name)["if"])
        for name in ("release.yml", "release-publish.yml"):
            triggers = yaml(ROOT / ".github/workflows" / name)[True]
            self.assertNotIn("dev", str(triggers))


if __name__ == "__main__":
    unittest.main()
