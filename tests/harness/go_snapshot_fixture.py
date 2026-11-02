"""Independent golden fixture for the Go scoped-identity snapshot (T09).

Builds the exact fixture workspace the Go test builds and prints the reference
consumer's snapshot result, so the Go implementation can be checked against
values derived from the Python reference instead of from itself.

    TMPDIR=/tmp/opencode python3 -B tests/harness/go_snapshot_fixture.py

Development reference only; no provider, task writer or agent loop.
"""
import os
import subprocess
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import receipt_tools as rt  # noqa: E402

CONTENTS = {
    ".gitignore": "ignored.txt\n",
    "code.txt": "original\n",
    "tasks.md": "B1 awaiting human review\n",
    "sub/dep.txt": "dependency\n",
    "gone.txt": "gone\n",
}


def git(root, *args):
    env = {"PATH": os.environ.get("PATH", "/usr/bin:/bin"), "HOME": str(root),
           "GIT_CONFIG_GLOBAL": os.devnull, "GIT_CONFIG_NOSYSTEM": "1",
           "GIT_CONFIG_COUNT": "3",
           "GIT_CONFIG_KEY_0": "init.templateDir", "GIT_CONFIG_VALUE_0": "",
           "GIT_CONFIG_KEY_1": "core.hooksPath", "GIT_CONFIG_VALUE_1": os.devnull,
           "GIT_CONFIG_KEY_2": "commit.gpgsign", "GIT_CONFIG_VALUE_2": "false",
           "GIT_TERMINAL_PROMPT": "0", "GIT_NO_REPLACE_OBJECTS": "1"}
    return subprocess.check_output(["git", "--no-optional-locks", "-C", str(root), *args],
                                   stderr=subprocess.PIPE, timeout=60, env=env)


def build(root):
    repo = root / "repo"
    repo.mkdir()
    git(repo, "init", "-q")
    git(repo, "config", "user.name", "Synthetic Fixture")
    git(repo, "config", "user.email", "fixture@example.invalid")
    for name, text in CONTENTS.items():
        target = repo / name
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(text)
        os.chmod(target, 0o644)
    git(repo, "add", ".")
    git(repo, "commit", "-qm", "synthetic fixture")
    (repo / "staged.txt").write_text("staged-new\n")
    os.chmod(repo / "staged.txt", 0o644)
    git(repo, "add", "staged.txt")
    (repo / "code.txt").write_text("unstaged\n")
    (repo / "new.txt").write_text("untracked\n")
    os.chmod(repo / "new.txt", 0o644)
    (repo / "ignored.txt").write_text("ignored-content\n")
    os.chmod(repo / "ignored.txt", 0o644)
    (repo / "gone.txt").unlink()
    os.chmod(repo / "tasks.md", 0o755)
    return repo


def main():
    with tempfile.TemporaryDirectory(prefix="sdd-golden-", dir=os.environ.get("TMPDIR")) as tmp:
        repo = build(Path(tmp))
        full = rt.snapshot(repo, ["."])
        focused = rt.snapshot(repo, ["code.txt"])
        print("full_diff =", repr(full["diff"]))
        print("focused_diff =", repr(focused["diff"]))
        print("base_revision_length =", len(full["base_revision"]))


if __name__ == "__main__":
    main()
