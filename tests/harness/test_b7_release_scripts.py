"""B7/T14 offline regressions for FR-06/AC-06: release artifact manifest, digest
re-verification, sanitized records, cask rendering and the downgrade guard.

The scripts are exercised as subprocesses in isolated mode inside temporary
directories. No network, no GitHub API, no tap, no publish.
"""

import hashlib
import io
import json
from pathlib import Path
import re
import subprocess
import sys
import tarfile
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
MANIFEST_SCRIPT = ROOT / "scripts" / "release_manifest.py"
CASK_SCRIPT = ROOT / "scripts" / "release_cask.py"
GORELEASER = ROOT / ".goreleaser.yml"

COMMIT = "1234567890abcdef1234567890abcdef12345678"
OTHER_COMMIT = "fedcba0987654321fedcba0987654321fedcba09"
TAG = "v0.13.0"
VERSION = "0.13.0"
BINARIES = ("asdx", "asdx-core", "agentic-sdd", "agentic-sdd-tui")


def _run(script, *args):
    return subprocess.run(
        [sys.executable, "-I", "-B", str(script), *args],
        capture_output=True,
        text=True,
        cwd="/",
        timeout=60,
    )


def _sha256(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


# The real layout: this repo's .goreleaser.yml sets no wrap_in_directory, so every member
# sits at the archive root (no top-level directory). Its default archive `files` glob also
# adds LICENSE and README.md at the root. A local snapshot in dist/ (an older config)
# confirms the flat layout offline; the pinned goreleaser v2.14.0 is not checked offline.
ARCHIVE_EXTRAS = ("LICENSE", "README.md")


def build_archive(path, names):
    with tarfile.open(path, "w:gz") as tar:
        for name in names:
            data = f"fake {name}\n".encode()
            info = tarfile.TarInfo(name=name)
            info.size = len(data)
            info.mode = 0o755
            tar.addfile(info, io.BytesIO(data))


def _write_checksums(dist):
    lines = [f"{_sha256(path)}  {path.name}" for path in sorted(Path(dist).glob("*.tar.gz"))]
    (Path(dist) / "checksums.txt").write_text("\n".join(lines) + "\n")


def _refresh_upload_set(out, archive_name):
    """Re-list checksums and manifest digests after the archive was rewritten, so that only
    the archive layout differs from what the manifest was written against."""
    (out / "checksums.txt").write_text(f"{_sha256(out / archive_name)}  {archive_name}\n")
    manifest_path = out / "manifest.json"
    manifest = json.loads(manifest_path.read_text())
    for entry in manifest["files"]:
        path = out / entry["name"]
        entry["sha256"] = _sha256(path)
        entry["size"] = path.stat().st_size
    manifest_path.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n")


def make_dist(root, tag=TAG, members=BINARIES, archives=1):
    version = tag[1:]
    dist = Path(root) / "dist"
    dist.mkdir(parents=True)
    for index in range(archives):
        arch = "all" if index == 0 else f"x{index}"
        build_archive(dist / f"agentic-sdd_{version}_darwin_{arch}.tar.gz",
                      [*members, *ARCHIVE_EXTRAS])
    _write_checksums(dist)
    return dist


def write_release(root, **kwargs):
    dist = make_dist(root, **kwargs)
    out = Path(root) / "release-artifacts"
    result = _run(MANIFEST_SCRIPT, "write", "--dist", str(dist), "--out", str(out),
                  "--tag", TAG, "--commit", COMMIT)
    return dist, out, result


class ManifestTests(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)

    def tearDown(self):
        self._tmp.cleanup()

    def _verify(self, out, tag=TAG, commit=COMMIT):
        return _run(MANIFEST_SCRIPT, "verify", "--dir", str(out),
                    "--manifest", str(out / "manifest.json"),
                    "--tag", tag, "--commit", commit)

    def test_write_then_verify_passes_and_copies_only_the_upload_set(self):
        _, out, result = write_release(self.root)
        self.assertEqual(result.returncode, 0, result.stderr)
        names = sorted(p.name for p in out.iterdir())
        self.assertEqual(names, sorted([
            f"agentic-sdd_{VERSION}_darwin_all.tar.gz", "checksums.txt", "manifest.json",
        ]))
        verify = self._verify(out)
        self.assertEqual(verify.returncode, 0, verify.stderr)

    def test_manifest_records_tag_commit_and_digests(self):
        _, out, _ = write_release(self.root)
        import json
        data = json.loads((out / "manifest.json").read_text())
        self.assertEqual(data["format"], 1)
        self.assertEqual(data["tag"], TAG)
        self.assertEqual(data["version"], VERSION)
        self.assertEqual(data["commit"], COMMIT)
        by_name = {f["name"]: f for f in data["files"]}
        archive = f"agentic-sdd_{VERSION}_darwin_all.tar.gz"
        self.assertEqual(by_name[archive]["sha256"], _sha256(out / archive))
        self.assertEqual(by_name["checksums.txt"]["sha256"], _sha256(out / "checksums.txt"))
        self.assertEqual(sorted(data["archive_members"]), sorted((*BINARIES, *ARCHIVE_EXTRAS)))

    def test_write_refuses_an_archive_wrapped_in_a_top_level_directory(self):
        dist = make_dist(self.root)
        wrapped = f"agentic-sdd_{VERSION}_darwin_all"
        build_archive(dist / f"agentic-sdd_{VERSION}_darwin_all.tar.gz",
                      [f"{wrapped}/{name}" for name in (*BINARIES, *ARCHIVE_EXTRAS)])
        _write_checksums(dist)
        out = self.root / "release-artifacts"
        result = _run(MANIFEST_SCRIPT, "write", "--dist", str(dist), "--out", str(out),
                      "--tag", TAG, "--commit", COMMIT)
        self.assertEqual(result.returncode, 1, result.stderr)
        self.assertIn("not at the archive root", result.stderr)
        self.assertFalse(out.exists())

    def test_write_refuses_a_shipped_binary_nested_under_a_directory(self):
        dist = make_dist(self.root)
        build_archive(dist / f"agentic-sdd_{VERSION}_darwin_all.tar.gz",
                      [*BINARIES[:-1], f"bin/{BINARIES[-1]}", *ARCHIVE_EXTRAS])
        _write_checksums(dist)
        out = self.root / "release-artifacts"
        result = _run(MANIFEST_SCRIPT, "write", "--dist", str(dist), "--out", str(out),
                      "--tag", TAG, "--commit", COMMIT)
        self.assertEqual(result.returncode, 1, result.stderr)
        self.assertIn("not at the archive root", result.stderr)
        self.assertFalse(out.exists())

    def test_write_refuses_an_archive_that_lists_a_binary_twice(self):
        dist = make_dist(self.root)
        build_archive(dist / f"agentic-sdd_{VERSION}_darwin_all.tar.gz",
                      [*BINARIES, *ARCHIVE_EXTRAS, BINARIES[0]])
        _write_checksums(dist)
        out = self.root / "release-artifacts"
        result = _run(MANIFEST_SCRIPT, "write", "--dist", str(dist), "--out", str(out),
                      "--tag", TAG, "--commit", COMMIT)
        self.assertEqual(result.returncode, 1, result.stderr)
        self.assertIn("more than once", result.stderr)
        self.assertFalse(out.exists())

    def test_verify_refuses_a_wrapped_archive_even_when_every_digest_is_consistent(self):
        _, out, _ = write_release(self.root)
        archive_name = f"agentic-sdd_{VERSION}_darwin_all.tar.gz"
        wrapped = f"agentic-sdd_{VERSION}_darwin_all"
        build_archive(out / archive_name,
                      [f"{wrapped}/{name}" for name in (*BINARIES, *ARCHIVE_EXTRAS)])
        _refresh_upload_set(out, archive_name)
        result = self._verify(out)
        self.assertEqual(result.returncode, 1, result.stderr)
        self.assertIn("not at the archive root", result.stderr)

    def test_verify_refuses_tampered_archive_bytes(self):
        _, out, _ = write_release(self.root)
        archive = out / f"agentic-sdd_{VERSION}_darwin_all.tar.gz"
        archive.write_bytes(archive.read_bytes() + b"tamper")
        result = self._verify(out)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("sha256", result.stderr.lower())

    def test_verify_refuses_tampered_checksums_file(self):
        _, out, _ = write_release(self.root)
        (out / "checksums.txt").write_text("0" * 64 + "  x\n")
        result = self._verify(out)
        self.assertNotEqual(result.returncode, 0)

    def test_verify_refuses_extra_file_in_upload_set(self):
        _, out, _ = write_release(self.root)
        (out / "extra.txt").write_text("unexpected")
        result = self._verify(out)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("extra.txt", result.stderr)

    def _rewrite_manifest(self, out, mutate):
        manifest_path = out / "manifest.json"
        manifest = json.loads(manifest_path.read_text())
        mutate(manifest)
        manifest_path.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n")

    def _add_entry(self, out, name, data):
        """Place a file in the upload set and list it with a correct digest and size."""
        (out / name).write_bytes(data)

        def mutate(manifest):
            manifest["files"].append({"name": name, "sha256": hashlib.sha256(data).hexdigest(),
                                      "size": len(data)})
        self._rewrite_manifest(out, mutate)

    def test_verify_refuses_extra_manifest_entry_even_with_a_matching_digest(self):
        _, out, _ = write_release(self.root)
        self._add_entry(out, "evil.txt", b"not a release asset")
        result = self._verify(out)
        self.assertEqual(result.returncode, 1, result.stderr)
        self.assertIn("evil.txt", result.stderr)

    def test_verify_refuses_a_second_archive_outside_the_release_pattern(self):
        _, out, _ = write_release(self.root)
        data = (out / f"agentic-sdd_{VERSION}_darwin_all.tar.gz").read_bytes()
        self._add_entry(out, "evil.tar.gz", data)
        result = self._verify(out)
        self.assertEqual(result.returncode, 1, result.stderr)
        self.assertIn("evil.tar.gz", result.stderr)

    def test_verify_refuses_a_second_archive_matching_the_release_pattern(self):
        _, out, _ = write_release(self.root)
        data = (out / f"agentic-sdd_{VERSION}_darwin_all.tar.gz").read_bytes()
        self._add_entry(out, f"agentic-sdd_{VERSION}_darwin_x1.tar.gz", data)
        result = self._verify(out)
        self.assertEqual(result.returncode, 1, result.stderr)

    def test_verify_refuses_a_manifest_that_omits_a_present_archive(self):
        _, out, _ = write_release(self.root)
        archive = f"agentic-sdd_{VERSION}_darwin_all.tar.gz"
        self._rewrite_manifest(out, lambda m: m.update(
            files=[f for f in m["files"] if f["name"] != archive]))
        result = self._verify(out)
        self.assertEqual(result.returncode, 1, result.stderr)

    def test_verify_refuses_a_manifest_that_omits_checksums(self):
        _, out, _ = write_release(self.root)
        self._rewrite_manifest(out, lambda m: m.update(
            files=[f for f in m["files"] if f["name"] != "checksums.txt"]))
        result = self._verify(out)
        self.assertEqual(result.returncode, 1, result.stderr)

    def test_verify_refuses_missing_file(self):
        _, out, _ = write_release(self.root)
        (out / f"agentic-sdd_{VERSION}_darwin_all.tar.gz").unlink()
        result = self._verify(out)
        self.assertNotEqual(result.returncode, 0)

    def test_verify_refuses_commit_mismatch(self):
        _, out, _ = write_release(self.root)
        result = self._verify(out, commit=OTHER_COMMIT)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("commit", result.stderr.lower())

    def test_verify_refuses_tag_mismatch(self):
        _, out, _ = write_release(self.root)
        result = self._verify(out, tag="v0.13.1")
        self.assertNotEqual(result.returncode, 0)

    def test_write_refuses_archive_missing_a_shipped_binary(self):
        _, out, result = write_release(self.root, members=BINARIES[:3])
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("agentic-sdd-tui", result.stderr)
        self.assertFalse(out.exists())

    def test_write_refuses_ambiguous_archives(self):
        _, _, result = write_release(self.root, archives=2)
        self.assertNotEqual(result.returncode, 0)

    def test_write_refuses_non_empty_output_directory(self):
        dist = make_dist(self.root)
        out = self.root / "release-artifacts"
        out.mkdir()
        (out / "stale.txt").write_text("old")
        result = _run(MANIFEST_SCRIPT, "write", "--dist", str(dist), "--out", str(out),
                      "--tag", TAG, "--commit", COMMIT)
        self.assertNotEqual(result.returncode, 0)

    def test_write_refuses_invalid_tag_and_commit(self):
        dist = make_dist(self.root)
        out = self.root / "o"
        bad_tag = _run(MANIFEST_SCRIPT, "write", "--dist", str(dist), "--out", str(out),
                       "--tag", "0.13.0", "--commit", COMMIT)
        self.assertEqual(bad_tag.returncode, 2)
        bad_commit = _run(MANIFEST_SCRIPT, "write", "--dist", str(dist), "--out", str(out),
                          "--tag", TAG, "--commit", "main")
        self.assertEqual(bad_commit.returncode, 2)

    def test_record_writes_sanitized_status_and_rejects_unknown_status(self):
        out = self.root / "diag" / "verification-record.json"
        ok = _run(MANIFEST_SCRIPT, "record", "--out", str(out), "--tag", TAG,
                  "--commit", COMMIT, "--status", "failure")
        self.assertEqual(ok.returncode, 0, ok.stderr)
        import json
        data = json.loads(out.read_text())
        self.assertEqual(data["job_status"], "failure")
        self.assertIsNone(data["manifest_sha256"])
        self.assertEqual(set(data), {"format", "tag", "commit", "job_status",
                                     "manifest_sha256", "recorded_at"})
        bad = _run(MANIFEST_SCRIPT, "record", "--out", str(self.root / "x.json"),
                   "--tag", TAG, "--commit", COMMIT, "--status", "maybe")
        self.assertEqual(bad.returncode, 2)


class CaskTests(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)
        _, self.out, result = write_release(self.root)
        assert result.returncode == 0, result.stderr
        self.archive = f"agentic-sdd_{VERSION}_darwin_all.tar.gz"
        self.digest = _sha256(self.out / self.archive)

    def tearDown(self):
        self._tmp.cleanup()

    def _render(self, out_path, tag=TAG, commit=COMMIT):
        return _run(CASK_SCRIPT, "render", "--dir", str(self.out),
                    "--manifest", str(self.out / "manifest.json"),
                    "--tag", tag, "--commit", commit, "--out", str(out_path))

    def test_render_requires_the_tag_commit(self):
        cask = self.root / "c.rb"
        result = _run(CASK_SCRIPT, "render", "--dir", str(self.out),
                      "--manifest", str(self.out / "manifest.json"),
                      "--tag", TAG, "--out", str(cask))
        self.assertEqual(result.returncode, 2, result.stderr)
        self.assertFalse(cask.exists())

    def test_render_refuses_a_manifest_whose_commit_differs_from_the_tag_commit(self):
        cask = self.root / "c.rb"
        result = self._render(cask, commit=OTHER_COMMIT)
        self.assertEqual(result.returncode, 1, result.stderr)
        self.assertIn("commit", result.stderr.lower())
        self.assertFalse(cask.exists())

    def test_render_refuses_an_invalid_commit(self):
        result = self._render(self.root / "c.rb", commit="main")
        self.assertEqual(result.returncode, 2, result.stderr)

    def test_render_uses_verified_digest_and_release_url(self):
        cask = self.root / "tap" / "Casks" / "agentic-sdd.rb"
        result = self._render(cask)
        self.assertEqual(result.returncode, 0, result.stderr)
        text = cask.read_text()
        self.assertIn(f'version "{VERSION}"', text)
        self.assertIn(f'sha256 "{self.digest}"', text)
        self.assertIn(
            f'url "https://github.com/nawodyaishan/agentic-sdd/releases/download/{TAG}/{self.archive}"',
            text,
        )
        self.assertIn('homepage "https://github.com/nawodyaishan/agentic-sdd"', text)
        for binary in BINARIES:
            self.assertIn(f'binary "{binary}"', text)
        for binary in BINARIES:
            self.assertIn(f'#{{staged_path}}/{binary}"', text)
        self.assertIn('"/usr/bin/xattr"', text)

    def test_render_matches_goreleaser_description_and_binaries(self):
        goreleaser = GORELEASER.read_text()
        desc = re.search(r'(?m)^    description: "(.+)"$', goreleaser).group(1)
        cask = self.root / "c.rb"
        self.assertEqual(self._render(cask).returncode, 0)
        self.assertIn(f'desc "{desc}"', cask.read_text())
        self.assertIn('license "MIT"', cask.read_text())

    def test_render_refuses_tampered_archive_and_writes_nothing(self):
        (self.out / self.archive).write_bytes(b"tampered")
        cask = self.root / "c.rb"
        result = self._render(cask)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(cask.exists())

    def test_render_refuses_tag_that_does_not_match_manifest(self):
        cask = self.root / "c.rb"
        result = self._render(cask, tag="v0.13.1")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(cask.exists())

    def test_render_refuses_invalid_tag(self):
        result = self._render(self.root / "c.rb", tag="v0.13")
        self.assertEqual(result.returncode, 2)

    def _guard(self, cask_version, tag=TAG, write=True):
        cask = self.root / "tap-cask.rb"
        if write and cask_version is not None:
            cask.write_text(f'cask "agentic-sdd" do\n  version "{cask_version}"\nend\n')
        return _run(CASK_SCRIPT, "check-downgrade", "--tag", tag, "--cask", str(cask))

    def test_downgrade_guard_refuses_equal_version(self):
        self.assertEqual(self._guard("0.13.0").returncode, 3)

    def test_downgrade_guard_refuses_newer_tap_version(self):
        self.assertEqual(self._guard("0.14.0").returncode, 3)

    def test_downgrade_guard_compares_numerically_not_lexically(self):
        # 0.13.10 is newer than v0.13.9 even though "0.13.10" < "0.13.9" as text.
        self.assertEqual(self._guard("0.13.10", tag="v0.13.9").returncode, 3)

    def test_downgrade_guard_allows_older_tap_version(self):
        self.assertEqual(self._guard("0.12.9").returncode, 0)

    def test_downgrade_guard_refuses_a_missing_cask_instead_of_treating_it_as_first_publication(self):
        # The tap migration is complete (Casks/agentic-sdd.rb exists), so a missing cask is an
        # unexpected tap state, not a first publication.
        result = self._guard(None)
        self.assertEqual(result.returncode, 1, result.stderr)
        self.assertIn("unexpected tap state", result.stderr)
        self.assertNotIn("first publication allowed", result.stdout)

    def _tap(self, cask_version="0.12.9", formula=False, cask=True):
        tap = self.root / "tap-layout"
        (tap / "Casks").mkdir(parents=True, exist_ok=True)
        cask_path = tap / "Casks" / "agentic-sdd.rb"
        if cask and cask_version is not None:
            cask_path.write_text(f'cask "agentic-sdd" do\n  version "{cask_version}"\nend\n')
        if formula:
            (tap / "Formula").mkdir(exist_ok=True)
            (tap / "Formula" / "agentic-sdd.rb").write_text('class AgenticSdd < Formula\nend\n')
        return cask_path

    def test_downgrade_guard_refuses_a_missing_cask_in_a_tap_layout(self):
        cask_path = self._tap(cask=False)
        result = _run(CASK_SCRIPT, "check-downgrade", "--tag", TAG, "--cask", str(cask_path))
        self.assertEqual(result.returncode, 1, result.stderr)
        self.assertIn("unexpected tap state", result.stderr)

    def test_downgrade_guard_refuses_while_a_formula_is_still_present(self):
        cask_path = self._tap(cask_version="0.12.9", formula=True)
        result = _run(CASK_SCRIPT, "check-downgrade", "--tag", TAG, "--cask", str(cask_path))
        self.assertEqual(result.returncode, 1, result.stderr)
        self.assertIn("migration", result.stderr)
        self.assertIn("Formula", result.stderr)

    def test_downgrade_guard_refuses_a_dangling_formula_symlink_too(self):
        cask_path = self._tap(cask_version="0.12.9")
        (cask_path.parent.parent / "Formula").mkdir()
        (cask_path.parent.parent / "Formula" / "agentic-sdd.rb").symlink_to(self.root / "gone.rb")
        result = _run(CASK_SCRIPT, "check-downgrade", "--tag", TAG, "--cask", str(cask_path))
        self.assertEqual(result.returncode, 1, result.stderr)

    def test_downgrade_guard_still_updates_an_existing_cask_when_no_formula_exists(self):
        cask_path = self._tap(cask_version="0.12.9")
        result = _run(CASK_SCRIPT, "check-downgrade", "--tag", TAG, "--cask", str(cask_path))
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_downgrade_guard_refuses_unparseable_existing_cask(self):
        cask = self.root / "tap-cask.rb"
        cask.write_text('cask "agentic-sdd" do\nend\n')
        result = _run(CASK_SCRIPT, "check-downgrade", "--tag", TAG, "--cask", str(cask))
        self.assertEqual(result.returncode, 3)

    def test_downgrade_guard_refuses_a_symlinked_cask_even_when_its_target_is_older(self):
        target = self.root / "real-cask.rb"
        target.write_text('cask "agentic-sdd" do\n  version "0.12.9"\nend\n')
        link = self.root / "linked-cask.rb"
        link.symlink_to(target)
        result = _run(CASK_SCRIPT, "check-downgrade", "--tag", TAG, "--cask", str(link))
        self.assertEqual(result.returncode, 3, result.stderr)
        self.assertIn("symlink", result.stderr)

    def test_downgrade_guard_refuses_a_dangling_symlink_instead_of_allowing_first_publication(self):
        link = self.root / "dangling-cask.rb"
        link.symlink_to(self.root / "missing.rb")
        result = _run(CASK_SCRIPT, "check-downgrade", "--tag", TAG, "--cask", str(link))
        self.assertEqual(result.returncode, 3, result.stderr)

    def test_downgrade_guard_refuses_a_cask_with_more_than_one_version_line(self):
        cask = self.root / "tap-cask.rb"
        cask.write_text('cask "agentic-sdd" do\n  version "0.12.9"\n  version "0.14.0"\nend\n')
        result = _run(CASK_SCRIPT, "check-downgrade", "--tag", TAG, "--cask", str(cask))
        self.assertEqual(result.returncode, 3, result.stderr)
        self.assertIn("exactly one", result.stderr)

    def test_render_refuses_to_write_through_a_symlinked_cask(self):
        target = self.root / "victim.rb"
        target.write_text("untouched\n")
        for label, link_target in (("existing", target), ("dangling", self.root / "gone.rb")):
            with self.subTest(link=label):
                link = self.root / f"tap-cask-{label}.rb"
                link.symlink_to(link_target)
                result = self._render(link)
                self.assertEqual(result.returncode, 1, result.stderr)
                self.assertIn("symlink", result.stderr)
                self.assertEqual(target.read_text(), "untouched\n")
                self.assertFalse((self.root / "gone.rb").exists())

    def test_downgrade_guard_rejects_non_release_tags(self):
        for tag in ("v0.13", "v0.13.0-rc1", "0.13.0", "v0.13.0 "):
            with self.subTest(tag=tag):
                self.assertEqual(self._guard(None, tag=tag).returncode, 2)


PUBLISHER_SUITE = 901
CI_SUITE = 900


def _check_run(name, status="completed", conclusion="success", suite=CI_SUITE):
    """A check run as the workflow projects it: the suite is check_suite.id, never a URL."""
    return {
        "name": name,
        "status": status,
        "conclusion": conclusion,
        "check_suite": {"id": suite},
    }


class CiGateTests(unittest.TestCase):
    """Decision 1 (plan 12.8): refuse to publish unless the tagged commit's CI passed."""

    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)

    def tearDown(self):
        self._tmp.cleanup()

    def _gate(self, runs=(), raw=None, suite_id=PUBLISHER_SUITE, commit=COMMIT, command="check-ci"):
        path = self.root / "check-runs.jsonl"
        if raw is None:
            raw = "".join(json.dumps(run) + "\n" for run in runs)
        path.write_text(raw)
        return _run(MANIFEST_SCRIPT, command, "--runs", str(path), "--commit", commit,
                    "--suite-id", str(suite_id))

    def _wait(self, runs=(), raw=None, suite_id=PUBLISHER_SUITE, commit=COMMIT):
        return self._gate(runs, raw=raw, suite_id=suite_id, commit=commit, command="wait-ci")

    def test_passes_when_every_ci_run_completed_successfully(self):
        runs = [
            _check_run("test"),
            _check_run("docs", conclusion="success"),
            _check_run("verify", suite=PUBLISHER_SUITE),
            _check_run("publish", status="in_progress", conclusion=None, suite=PUBLISHER_SUITE),
        ]
        self.assertEqual(self._gate(runs).returncode, 0)
        self.assertEqual(self._wait(runs).returncode, 0)

    def test_in_progress_ci_run_blocks_publishing_until_it_completes(self):
        still_running = [_check_run("test"), _check_run("slow", status="in_progress",
                                                        conclusion=None)]
        final = self._gate(still_running)
        self.assertEqual(final.returncode, 1, final.stderr)
        self.assertIn("slow (in_progress)", final.stderr)
        # wait-ci reports pending (4), not a refusal and not a pass, so the step polls again.
        self.assertEqual(self._wait(still_running).returncode, 4)
        done = [_check_run("test"), _check_run("slow")]
        self.assertEqual(self._gate(done).returncode, 0)
        self.assertEqual(self._wait(done).returncode, 0)

    def test_queued_ci_run_blocks_publishing_until_it_completes(self):
        queued = [_check_run("test", status="queued", conclusion=None)]
        self.assertEqual(self._gate(queued).returncode, 1)
        self.assertIn("have not completed", self._gate(queued).stderr)
        self.assertEqual(self._wait(queued).returncode, 4)

    def test_publisher_own_runs_never_block_even_when_incomplete(self):
        runs = [_check_run("test"),
                _check_run("publish", status="in_progress", conclusion=None, suite=PUBLISHER_SUITE)]
        self.assertEqual(self._gate(runs).returncode, 0)
        self.assertEqual(self._wait(runs).returncode, 0)

    def test_refuses_when_the_api_returns_no_runs(self):
        result = self._gate(raw="")
        self.assertEqual(result.returncode, 1)
        self.assertIn("no check runs", result.stderr)
        self.assertEqual(self._wait(raw="").returncode, 1)

    def test_refuses_when_only_this_publisher_run_is_present(self):
        runs = [
            _check_run("verify", suite=PUBLISHER_SUITE),
            _check_run("publish", status="in_progress", conclusion=None, suite=PUBLISHER_SUITE),
        ]
        self.assertEqual(self._gate(runs).returncode, 1)
        self.assertIn("no check runs", self._gate(runs).stderr)
        self.assertEqual(self._wait(runs).returncode, 1)

    def test_refuses_a_failed_completed_check_run_and_names_it(self):
        runs = [_check_run("test"), _check_run("lint", conclusion="failure")]
        result = self._gate(runs)
        self.assertEqual(result.returncode, 1)
        self.assertIn("lint (failure)", result.stderr)
        self.assertEqual(self._wait(runs).returncode, 1)

    def test_completed_failure_fails_fast_while_another_run_is_still_pending(self):
        runs = [_check_run("lint", conclusion="failure"),
                _check_run("slow", status="in_progress", conclusion=None)]
        result = self._wait(runs)
        self.assertEqual(result.returncode, 1, "a failure must not wait for pending runs")
        self.assertIn("lint (failure)", result.stderr)

    def test_refuses_every_completed_conclusion_other_than_success(self):
        for conclusion in ("skipped", "cancelled", "neutral", "timed_out", "action_required",
                           "stale"):
            with self.subTest(conclusion=conclusion):
                runs = [_check_run("test", conclusion=conclusion)]
                self.assertEqual(self._gate(runs).returncode, 1)
                self.assertEqual(self._wait(runs).returncode, 1)

    def test_blank_lines_are_tolerated(self):
        raw = "\n" + json.dumps(_check_run("test")) + "\n\n"
        self.assertEqual(self._gate(raw=raw).returncode, 0)
        self.assertEqual(self._wait(raw=raw).returncode, 0)

    def test_malformed_input_is_invalid_not_a_pass(self):
        for command in ("check-ci", "wait-ci"):
            with self.subTest(command=command):
                self.assertEqual(self._gate(raw="not json\n", command=command).returncode, 2)
                self.assertEqual(
                    self._gate(raw=json.dumps({"status": "completed"}) + "\n", command=command).returncode,
                    2)

    def test_own_suite_is_excluded_by_check_suite_id_even_without_a_details_url(self):
        runs = [_check_run("test"),
                {"name": "publish", "status": "in_progress", "conclusion": None,
                 "check_suite": {"id": PUBLISHER_SUITE}}]
        self.assertEqual(self._gate(runs).returncode, 0)
        self.assertEqual(self._wait(runs).returncode, 0)

    def test_foreign_suite_is_not_excluded_even_if_its_url_names_the_publisher_run(self):
        runs = [{"name": "sneaky", "status": "in_progress", "conclusion": None,
                 "check_suite": {"id": CI_SUITE},
                 "details_url": "https://github.com/nawodyaishan/agentic-sdd/actions/runs/222/job/1"}]
        self.assertEqual(self._wait(runs).returncode, 4)
        self.assertEqual(self._gate(runs).returncode, 1)
        self.assertIn("sneaky (in_progress)", self._gate(runs).stderr)

    def test_run_without_a_check_suite_is_not_own_and_still_counts(self):
        pending = [{"name": "legacy", "status": "in_progress", "conclusion": None}]
        self.assertEqual(self._wait(pending).returncode, 4)
        failed = [{"name": "legacy", "status": "completed", "conclusion": "failure"}]
        self.assertEqual(self._wait(failed).returncode, 1)

    def test_non_integer_check_suite_ids_are_not_own(self):
        for value in ("901", 901.0, None, [901], {"id": 901}):
            with self.subTest(value=value):
                runs = [{"name": "odd", "status": "in_progress", "conclusion": None,
                         "check_suite": {"id": value}}]
                self.assertEqual(self._wait(runs).returncode, 4)

    def test_boolean_check_suite_id_never_matches_suite_one(self):
        runs = [{"name": "odd", "status": "in_progress", "conclusion": None,
                 "check_suite": {"id": True}}]
        self.assertEqual(self._wait(runs, suite_id=1).returncode, 4)

    def test_unknown_check_run_status_is_refused_at_once_and_never_waited_on(self):
        # Only queued, in_progress and completed are understood; anything else (stale,
        # requested, waiting, pending, ...) must not consume the 40-minute wait budget.
        for status in ("stale", "requested", "waiting", "pending", "bogus"):
            with self.subTest(status=status):
                runs = [_check_run("test"), _check_run("odd", status=status, conclusion=None)]
                wait = self._wait(runs)
                self.assertEqual(wait.returncode, 1, wait.stderr)
                self.assertIn(f"odd", wait.stderr)
                self.assertIn(status, wait.stderr)
                self.assertIn("refusing to publish", wait.stderr)
                self.assertEqual(self._gate(runs).returncode, 1)

    def test_rejects_invalid_suite_id_and_commit(self):
        for command in ("check-ci", "wait-ci"):
            with self.subTest(command=command):
                runs = [_check_run("test")]
                for bad in ("abc", "0", "-5", "", "1.5", "01"):
                    with self.subTest(suite_id=bad):
                        self.assertEqual(self._gate(runs, suite_id=bad, command=command).returncode, 2)
                self.assertEqual(self._gate(runs, commit="main", command=command).returncode, 2)


if __name__ == "__main__":
    unittest.main()
