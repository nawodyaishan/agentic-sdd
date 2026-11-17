"""B7/T15 regressions for FR-07/AC-07: reproducible build flags and consolidated snapshot policy."""

import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]


def _extract_yaml_block(text: str, id_name: str, section_header: str) -> str:
    section_idx = text.find(section_header)
    if section_idx == -1:
        raise AssertionError(f"missing section {section_header}")
    section = text[section_idx:]
    next_top = re.search(r"\n[a-z_]+:\s*\n", section[len(section_header):])
    if next_top:
        section = section[: len(section_header) + next_top.start()]
    pattern = rf"(?ms)^  - id: {re.escape(id_name)}\b.*?(?=^  - id: |\Z)"
    match = re.search(pattern, section)
    if not match:
        raise AssertionError(f"missing block id {id_name} in {section_header}")
    return match.group(0)


class GoReleaserReproducibilityTests(unittest.TestCase):
    def test_goreleaser_replaces_wall_clock_date_with_commit_date(self):
        text = (ROOT / ".goreleaser.yml").read_text()

        # Wall-clock .Date must not remain anywhere in .goreleaser.yml (especially ldflags).
        self.assertIsNone(
            re.search(r"\{\{[^}]*\.Date\b[^}]*\}\}", text),
            "wall-clock {{ .Date }} must not remain in .goreleaser.yml",
        )
        self.assertNotIn("agentic_sdd_experimental", text)

        for build_id in ("cli-core", "cli-legacy", "tui-canonical", "tui-legacy"):
            with self.subTest(build=build_id):
                block = _extract_yaml_block(text, build_id, "builds:")
                self.assertIn("- -trimpath", block)
                self.assertIn('mod_timestamp: "{{ .CommitTimestamp }}"', block)
                if build_id.startswith("cli-"):
                    self.assertIn(
                        "-X agentic-sdd/internal/version.Date={{ .CommitDate }}",
                        block,
                    )

        for uni_id in (
            "cli-core-universal",
            "cli-legacy-universal",
            "tui-canonical-universal",
            "tui-legacy-universal",
        ):
            with self.subTest(universal=uni_id):
                block = _extract_yaml_block(text, uni_id, "universal_binaries:")
                self.assertIn('mod_timestamp: "{{ .CommitTimestamp }}"', block)


@unittest.skipUnless(shutil.which("task") and shutil.which("git"), "Go Task and git are required")
class SnapshotPolicyTests(unittest.TestCase):
    def _init_clean_repo(self, base: Path) -> tuple[Path, Path, Path]:
        repo = base / "repo"
        repo.mkdir()
        (repo / "cmd/agentic-sdd").mkdir(parents=True)
        (repo / "experiments/tui/cmd/tui-companion").mkdir(parents=True)
        (repo / "go.mod").write_text("module agentic-sdd\n\ngo 1.23\n")
        (repo / "experiments/tui/go.mod").write_text("module agentic-sdd/experiments/tui\n\ngo 1.26\n")
        (repo / "cmd/agentic-sdd/main.go").write_text("package main\nfunc main() {}\n")
        (repo / "experiments/tui/cmd/tui-companion/main.go").write_text("package main\nfunc main() {}\n")
        shutil.copyfile(ROOT / "Taskfile.yml", repo / "Taskfile.yml")
        shutil.copyfile(ROOT / "Taskfile.experimental.yml", repo / "Taskfile.experimental.yml")
        shutil.copyfile(ROOT / ".goreleaser.yml", repo / ".goreleaser.yml")

        git_env = self._git_env(base / "git-home")
        self._git(repo, git_env, "init", "-q")
        self._git(repo, git_env, "config", "user.name", "Snapshot Test")
        self._git(repo, git_env, "config", "user.email", "snapshot@example.invalid")
        self._git(repo, git_env, "add", ".")
        self._git(repo, git_env, "commit", "-qm", "initial clean commit")

        bin_dir = base / "bin"
        bin_dir.mkdir()
        log_dir = base / "logs"
        log_dir.mkdir()
        return repo, bin_dir, log_dir

    def _git_env(self, home: Path) -> dict[str, str]:
        home.mkdir(parents=True, exist_ok=True)
        return {
            "PATH": os.environ.get("PATH", "/usr/bin:/bin"),
            "HOME": str(home),
            "GIT_CONFIG_GLOBAL": os.devnull,
            "GIT_CONFIG_NOSYSTEM": "1",
            "GIT_CONFIG_COUNT": "3",
            "GIT_CONFIG_KEY_0": "init.templateDir",
            "GIT_CONFIG_VALUE_0": "",
            "GIT_CONFIG_KEY_1": "core.hooksPath",
            "GIT_CONFIG_VALUE_1": os.devnull,
            "GIT_CONFIG_KEY_2": "commit.gpgsign",
            "GIT_CONFIG_VALUE_2": "false",
            "GIT_TERMINAL_PROMPT": "0",
        }

    def _git(self, repo: Path, env: dict[str, str], *args: str) -> str:
        res = subprocess.run(
            ["git", "--no-optional-locks", "-C", str(repo), *args],
            env=env,
            text=True,
            capture_output=True,
            check=True,
        )
        return res.stdout

    def _install_fake_tools(
        self,
        bin_dir: Path,
        log_dir: Path,
        *,
        goreleaser_ver="v2.14.0",
        go_ver="go1.27.1",
        include_goreleaser=True,
        goreleaser_check_ok=True,
        tui_deps_ok=True,
    ):
        fake_go = bin_dir / "go"
        list_exit = 0 if tui_deps_ok else 1
        fake_go.write_text(
            "#!/bin/sh\n"
            'if [ "$1" = "env" ] && [ "$2" = "GOVERSION" ]; then\n'
            f'  printf "%s\\n" "{go_ver}"\n'
            "  exit 0\n"
            "fi\n"
            'if [ "$1" = "env" ] && [ "$2" = "GOMODCACHE" ]; then\n'
            f'  printf "%s\\n" "{log_dir / "gomodcache"}"\n'
            "  exit 0\n"
            "fi\n"
            'if [ "$1" = "list" ]; then\n'
            f"  exit {list_exit}\n"
            "fi\n"
            'if [ "$1" = "version" ]; then\n'
            f'  printf "go version %s linux/amd64\\n" "{go_ver}"\n'
            "  exit 0\n"
            "fi\n"
            "exit 0\n"
        )
        fake_go.chmod(0o755)

        if include_goreleaser:
            check_exit = 0 if goreleaser_check_ok else 1
            fake_gr = bin_dir / "goreleaser"
            fake_gr.write_text(
                "#!/bin/sh\n"
                'if [ "$1" = "--version" ]; then\n'
                f'  printf "GitVersion: %s\\n" "{goreleaser_ver}"\n'
                "  exit 0\n"
                "fi\n"
                'if [ "$1" = "check" ]; then\n'
                f"  exit {check_exit}\n"
                "fi\n"
                'printf "PWD=%s\\nARGS=%s\\nGOCACHE=%s\\nXDG_CACHE_HOME=%s\\nGOTOOLCHAIN=%s\\nGOPROXY=%s\\nGOSUMDB=%s\\nGOENV=%s\\nGOWORK=%s\\nGOFLAGS=%s\\nTOKENS=%s|%s|%s|%s|%s|%s\\n" '
                '"$PWD" "$*" "$GOCACHE" "$XDG_CACHE_HOME" "$GOTOOLCHAIN" "$GOPROXY" "$GOSUMDB" "$GOENV" "$GOWORK" "$GOFLAGS" '
                '"$HOMEBREW_TAP_TOKEN" "$GITHUB_TOKEN" "$GH_TOKEN" "$GITLAB_TOKEN" "$GITEA_TOKEN" "$SDD_SOURCE_DIGEST" '
                f'> "{log_dir}/goreleaser.log"\n'
                'mkdir -p "$PWD/dist"\n'
                'printf "fake-checksum\\n" > "$PWD/dist/checksums.txt"\n'
                "exit 0\n"
            )
            fake_gr.chmod(0o755)

    def _base_env(self, bin_dir: Path, tmp_dir: Path, extra_env=None) -> dict[str, str]:
        sys_path = os.pathsep.join(
            p for p in os.environ.get("PATH", "/usr/bin:/bin").split(os.pathsep)
            if p and not (Path(p) / "goreleaser").exists() and not (Path(p) / "go").exists()
        )
        task_dir = str(Path(shutil.which("task")).parent)
        git_dir = str(Path(shutil.which("git")).parent)
        merged_path = os.pathsep.join(dict.fromkeys([str(bin_dir), task_dir, git_dir, sys_path, "/usr/bin", "/bin"]))
        env = {
            "PATH": merged_path,
            "HOME": str(tmp_dir / "home"),
            "TMPDIR": str(tmp_dir / "tmp"),
            "HOMEBREW_TAP_TOKEN": "secret-tap",
            "GITHUB_TOKEN": "secret-gh",
            "GH_TOKEN": "secret-gh-cli",
            "GITLAB_TOKEN": "secret-gl",
            "GITEA_TOKEN": "secret-gt",
            "SDD_SOURCE_DIGEST": "dirty-override",
            "GOTOOLCHAIN": "auto",
            "GOPROXY": "https://invalid.example",
            "GOSUMDB": "example",
            "GOENV": "/unreviewed/env",
            "GOWORK": "/unreviewed/work",
            "GOFLAGS": "-tags=agentic_sdd_experimental",
        }
        if extra_env:
            env.update(extra_env)
        Path(env["HOME"]).mkdir(parents=True, exist_ok=True)
        Path(env["TMPDIR"]).mkdir(parents=True, exist_ok=True)
        return env

    def _run_entrypoints(self, repo: Path, bin_dir: Path, tmp_dir: Path, extra_env=None, cli_args=()):
        env = self._base_env(bin_dir, tmp_dir, extra_env=extra_env)
        pub = subprocess.run(
            ["task", "release", *cli_args],
            cwd=repo,
            env=env,
            text=True,
            capture_output=True,
        )
        exp = subprocess.run(
            ["task", "-t", str(repo / "Taskfile.experimental.yml"), "release:snapshot", *cli_args],
            cwd=repo,
            env=env,
            text=True,
            capture_output=True,
        )
        return pub, exp

    def _assert_same_blocked(self, pub: subprocess.CompletedProcess, exp: subprocess.CompletedProcess, expected_substr: str):
        self.assertNotEqual(pub.returncode, 0, pub.stderr + pub.stdout)
        self.assertNotEqual(exp.returncode, 0, exp.stderr + exp.stdout)
        self.assertIn("exit status 2", pub.stderr)
        pub_line = pub.stderr.splitlines()[0]
        exp_line = exp.stderr.splitlines()[0]
        self.assertTrue(pub_line.startswith("snapshot: BLOCKED - "), pub.stderr)
        self.assertEqual(pub_line, exp_line)
        self.assertIn(expected_substr.lower(), pub_line.lower())

    def test_prerequisites_block_both_entrypoints_identically(self):
        cases = [
            ("missing-goreleaser", dict(include_goreleaser=False), "goreleaser"),
            ("wrong-goreleaser-version", dict(goreleaser_ver="v2.13.0"), "v2.14.0"),
            ("goreleaser-check-fails", dict(goreleaser_check_ok=False), "goreleaser check"),
            ("wrong-go-version", dict(go_ver="go1.25.0"), "go1.27.1"),
            ("missing-offline-tui-deps", dict(tui_deps_ok=False), "offline"),
        ]
        for name, kwargs, expected_substr in cases:
            with self.subTest(case=name), tempfile.TemporaryDirectory() as tmp:
                base = Path(tmp)
                repo, bin_dir, log_dir = self._init_clean_repo(base)
                self._install_fake_tools(bin_dir, log_dir, **kwargs)
                pub, exp = self._run_entrypoints(repo, bin_dir, base)
                self._assert_same_blocked(pub, exp, expected_substr)
                self.assertFalse((log_dir / "goreleaser.log").exists())

    def test_dirty_tree_refused_by_both_entrypoints(self):
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            repo, bin_dir, log_dir = self._init_clean_repo(base)
            self._install_fake_tools(bin_dir, log_dir)
            (repo / "untracked.txt").write_text("dirty\n")
            pub, exp = self._run_entrypoints(repo, bin_dir, base)
            self._assert_same_blocked(pub, exp, "dirty")
            self.assertFalse((log_dir / "goreleaser.log").exists())

    def test_output_inside_source_tree_is_refused(self):
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            repo, bin_dir, log_dir = self._init_clean_repo(base)
            self._install_fake_tools(bin_dir, log_dir)
            pub, exp = self._run_entrypoints(repo, bin_dir, base, cli_args=[f"SDD_OUTPUT={repo / 'dist'}"])
            self._assert_same_blocked(pub, exp, "outside")
            self.assertFalse((repo / "dist").exists())
            self.assertFalse((log_dir / "goreleaser.log").exists())

    def test_clean_snapshot_runs_externally_without_tokens_or_publishing(self):
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            repo, bin_dir, log_dir = self._init_clean_repo(base)
            self._install_fake_tools(bin_dir, log_dir)
            out_pub = base / "out-pub"
            out_exp = base / "out-exp"
            env = self._base_env(bin_dir, base)

            pub = subprocess.run(
                ["task", "release", f"SDD_OUTPUT={out_pub}"],
                cwd=repo,
                env=env,
                text=True,
                capture_output=True,
            )
            self.assertEqual(pub.returncode, 0, pub.stderr + pub.stdout)
            self.assertTrue((out_pub / "checksums.txt").is_file())
            self.assertFalse((repo / "dist").exists())
            self.assertFalse((repo / ".cache").exists())

            log_pub = (log_dir / "goreleaser.log").read_text()
            self.assertIn("ARGS=release --snapshot --clean", log_pub)
            self.assertNotIn(f"PWD={repo.resolve()}", log_pub)
            self.assertNotIn(f"GOCACHE={repo.resolve()}", log_pub)
            self.assertNotIn(f"XDG_CACHE_HOME={repo.resolve()}", log_pub)
            self.assertIn("GOTOOLCHAIN=local", log_pub)
            self.assertIn("GOPROXY=off", log_pub)
            self.assertIn("GOSUMDB=off", log_pub)
            self.assertIn("GOENV=off", log_pub)
            self.assertIn("GOWORK=off", log_pub)
            self.assertIn("GOFLAGS=\n", log_pub)
            self.assertIn("TOKENS=|||||\n", log_pub)

            (log_dir / "goreleaser.log").unlink()
            exp = subprocess.run(
                ["task", "-t", str(repo / "Taskfile.experimental.yml"), "release:snapshot", f"SDD_OUTPUT={out_exp}"],
                cwd=repo,
                env=env,
                text=True,
                capture_output=True,
            )
            self.assertEqual(exp.returncode, 0, exp.stderr + exp.stdout)
            self.assertTrue((out_exp / "checksums.txt").is_file())
            self.assertFalse((repo / "dist").exists())
            self.assertFalse((repo / ".cache").exists())
            log_exp = (log_dir / "goreleaser.log").read_text()
            self.assertIn("ARGS=release --snapshot --clean", log_exp)
            self.assertIn("TOKENS=|||||\n", log_exp)


class TwoBuildDeterminismTests(unittest.TestCase):
    def test_two_clean_exports_produce_identical_four_binaries(self):
        if not (shutil.which("go") and shutil.which("git")):
            self.skipTest("go and git are required")
        tui_check = subprocess.run(
            ["go", "list", "-deps", "./cmd/tui-companion"],
            cwd=ROOT / "experiments/tui",
            env={
                **os.environ,
                "GOTOOLCHAIN": "local",
                "GOPROXY": "off",
                "GOSUMDB": "off",
                "GOENV": "off",
                "GOWORK": "off",
                "GOFLAGS": "",
            },
            capture_output=True,
            text=True,
        )
        if tui_check.returncode != 0:
            self.skipTest("experiments/tui module dependencies are not cached offline")

        script = ROOT / "tests/reproducible-build-check.sh"
        self.assertTrue(script.is_file())
        res = subprocess.run(
            ["sh", str(script), "HEAD"],
            cwd=ROOT,
            env={**os.environ, "SDD_REPRO_ARCHES": "amd64"},
            text=True,
            capture_output=True,
        )
        self.assertEqual(res.returncode, 0, res.stderr + res.stdout)
        for name in ("asdx-core", "agentic-sdd", "asdx", "agentic-sdd-tui"):
            self.assertIn(name, res.stdout)


if __name__ == "__main__":
    unittest.main()
