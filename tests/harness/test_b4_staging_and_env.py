"""B4/T08 regressions for AC-17: Taskfile verify/tidy env and trusted staging."""

import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

import trusted_ci as gate


ROOT = Path(__file__).resolve().parents[2]


class TaskfileVerifyTidyEnvTests(unittest.TestCase):
    @unittest.skipUnless(shutil.which("task"), "Go Task is unavailable")
    def test_verify_and_tidy_check_strip_hostile_go_controls(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            bin_dir = root / "bin"
            bin_dir.mkdir()
            capture = root / "capture.jsonl"
            fake_go = bin_dir / "go"
            fake_go.write_text(
                "#!/bin/sh\n"
                'printf "%s|%s|%s|%s|%s|%s|%s\\n" '
                '"$GOTOOLCHAIN" "$GOPROXY" "$GOSUMDB" "$GOENV" "$GOWORK" "$GOFLAGS" "$*" >> "$CAPTURE"\n'
            )
            fake_go.chmod(0o755)
            env = {
                "PATH": str(bin_dir) + os.pathsep + os.environ.get("PATH", "/usr/bin:/bin"),
                "HOME": str(root / "home"),
                "TMPDIR": str(root / "tmp"),
                "CAPTURE": str(capture),
                "GOTOOLCHAIN": "auto",
                "GOPROXY": "https://invalid.example",
                "GOSUMDB": "example",
                "GOENV": "/unreviewed/env",
                "GOWORK": "/unreviewed/go.work",
                "GOFLAGS": "-tags=agentic_sdd_experimental",
            }
            Path(env["HOME"]).mkdir()
            Path(env["TMPDIR"]).mkdir()

            for task_name in ("verify", "tidy-check"):
                with self.subTest(task=task_name):
                    capture.unlink(missing_ok=True)
                    out_dir = root / f"out-{task_name}"
                    res = subprocess.run(
                        ["task", task_name, f"SDD_OUTPUT={out_dir}"],
                        cwd=ROOT,
                        env=env,
                        text=True,
                        capture_output=True,
                    )
                    self.assertEqual(res.returncode, 0, res.stderr)
                    lines = capture.read_text().splitlines()
                    self.assertEqual(len(lines), 1)
                    parts = lines[0].split("|")
                    self.assertEqual(parts[:6], ["local", "off", "off", "off", "off", ""])

    def test_taskfile_go_entrypoint_inventory_is_explicit(self):
        text = (ROOT / "Taskfile.yml").read_text()
        for snippet in (
            'GOCACHE="$2" GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS= go run ./cmd/sdd-check --scope focused --check tidy-check',
            'GOCACHE="$2" GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS= go run ./cmd/sdd-check --scope required',
        ):
            self.assertIn(snippet, text)


class TrustedStagingGeneratedOutputTests(unittest.TestCase):
    def _seed_clean_repo(self, root: Path):
        (root / "go.mod").write_text("module example.com/test\n\ngo 1.23\n")
        (root / "main.go").write_text("package main\nfunc main() {}\n")
        (root / "website/src").mkdir(parents=True)
        (root / "website/src/index.md").write_text("# Docs\n")
        (root / ".gitignore").write_text("dist/\nwebsite/node_modules/\nignored_by_candidate/\n")

    def test_fresh_export_and_generated_output_trees_stage_identically(self):
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            fresh = base / "fresh"
            dirty = base / "dirty"
            fresh.mkdir()
            dirty.mkdir()
            self._seed_clean_repo(fresh)
            self._seed_clean_repo(dirty)

            # Reviewed generated-output directories in dirty checkout, including
            # internal symlinks inside website/node_modules/.bin.
            (dirty / "dist").mkdir()
            (dirty / "dist/artifact.tar.gz").write_bytes(b"fake-archive")
            for rel in (
                "website/node_modules/pkg",
                "website/node_modules/.bin",
                "website/.astro",
                "website/.cache",
                "website/dist",
                "website/.export-check",
                "website/coverage",
            ):
                (dirty / rel).mkdir(parents=True, exist_ok=True)
                (dirty / rel / "generated.txt").write_text("generated\n")
            (dirty / "website/node_modules/.bin/tool").symlink_to("../pkg/cli.js")

            fresh_manifest = gate.stage_tree(fresh, base / "staged-fresh")
            dirty_manifest = gate.stage_tree(dirty, base / "staged-dirty")
            self.assertEqual(fresh_manifest, dirty_manifest)
            self.assertEqual(
                hashlib.sha256(json.dumps(fresh_manifest, sort_keys=True).encode()).hexdigest(),
                hashlib.sha256(json.dumps(dirty_manifest, sort_keys=True).encode()).hexdigest(),
            )
            self.assertFalse((base / "staged-dirty/dist").exists())
            self.assertFalse((base / "staged-dirty/website/node_modules").exists())
            self.assertFalse((base / "staged-dirty/website/dist").exists())

    def test_arbitrary_candidate_gitignore_cannot_hide_source_or_unsafe_entries(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            src = root / "candidate"
            src.mkdir()
            self._seed_clean_repo(src)
            ignored_dir = src / "ignored_by_candidate"
            ignored_dir.mkdir()
            (ignored_dir / "hidden.go").write_text("package main\n")

            manifest = gate.stage_tree(src, root / "staged")
            self.assertIn("ignored_by_candidate/hidden.go", manifest)

            # Unsafe entry inside an arbitrary .gitignore'd directory must still fail.
            (ignored_dir / ".env").write_text("SYNTHETIC=1\n")
            with self.assertRaises(ValueError):
                gate.stage_tree(src, root / "staged-env")

            (ignored_dir / ".env").unlink()
            (ignored_dir / "escape").symlink_to(root / "outside")
            with self.assertRaises(ValueError):
                gate.stage_tree(src, root / "staged-link")

    def test_symlink_at_excluded_generated_directory_path_is_refused(self):
        for rel in ("dist", "website/node_modules", "website/dist"):
            with self.subTest(rel=rel), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                src = root / "candidate"
                src.mkdir()
                self._seed_clean_repo(src)
                target = src / rel
                target.parent.mkdir(parents=True, exist_ok=True)
                target.symlink_to(root / "outside")
                with self.assertRaises(ValueError):
                    gate.stage_tree(src, root / "staged")

    def test_review_paths_flags_qualification_contract_changes(self):
        self.assertEqual(
            gate.review_paths({}, {"harness/qualification.json": {"sha256": "x"}}),
            ["harness/qualification.json"],
        )


if __name__ == "__main__":
    unittest.main()
