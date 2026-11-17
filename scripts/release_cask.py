#!/usr/bin/env python3
"""Homebrew cask renderer and downgrade guard for the agentic-sdd tap (B7/T14, FR-06).

Subcommands
  render           --dir DIR --manifest FILE --tag vX.Y.Z --commit SHA --out FILE
  check-downgrade  --tag vX.Y.Z --cask FILE

render re-verifies the upload set against its manifest (the same check the
publisher runs), bound to the tag's commit, before it writes anything, so the cask
digest is always the verified one for that commit. check-downgrade refuses when the
tap's cask version is already at or above the tag. The tap migration is complete, so
it also refuses a missing cask (an unexpected tap state, not a first publication) and
refuses while a Formula/agentic-sdd.rb is still present (the migration is incomplete).
The tap root is the cask's grandparent when the cask sits in a Casks/ directory,
otherwise the cask's own directory. Both lstat the cask: a symlink (even a dangling
one) or other non-regular entry is refused, never read or written through, and an
existing cask must contain exactly one version line.

Exit codes: 0 pass; 1 render refusal; 2 invalid input; 3 downgrade refused.
"""

import argparse
import os
from pathlib import Path
import re
import stat
import sys


sys.path.insert(0, str(Path(__file__).resolve().parent))

import release_manifest as rm  # noqa: E402  (same directory; isolated mode skips cwd)


DESCRIPTION = (
    "Spec-driven development workflow skills, kept in sync across Codex, "
    "Claude Code, and Antigravity CLI"
)
CASK_VERSION_RE = re.compile(r'(?m)^\s*version\s+"(\d+\.\d+\.\d+)"\s*$')
FORMULA_NAME = "agentic-sdd.rb"

CASK_TEMPLATE = """cask "agentic-sdd" do
  version "{version}"
  sha256 "{sha256}"

  url "https://github.com/nawodyaishan/agentic-sdd/releases/download/{tag}/{archive}"
  name "agentic-sdd"
  desc "{desc}"
  homepage "https://github.com/nawodyaishan/agentic-sdd"
  license "MIT"

  binary "asdx"
  binary "asdx-core"
  binary "agentic-sdd"
  binary "agentic-sdd-tui"

  postflight do
    if OS.mac?
      system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{{staged_path}}/asdx"]
      system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{{staged_path}}/asdx-core"]
      system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{{staged_path}}/agentic-sdd"]
      system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{{staged_path}}/agentic-sdd-tui"]
    end
  end
end
"""


class Guard(Exception):
    """The downgrade guard refused the cask update."""


def lstat_or_none(path):
    """lstat never follows a link, so a dangling symlink is seen as an entry, not as missing."""
    try:
        return os.lstat(path)
    except FileNotFoundError:
        return None


def cmd_render(args):
    version = rm.check_tag(args.tag)
    # verify_release refuses a manifest whose commit differs from --commit.
    data, archive = rm.verify_release(args.dir, args.manifest, args.tag, commit=args.commit)
    digest = next(entry["sha256"] for entry in data["files"] if entry["name"] == archive)
    text = CASK_TEMPLATE.format(
        version=version,
        sha256=digest,
        tag=args.tag,
        archive=archive,
        desc=DESCRIPTION,
    )
    out = Path(args.out)
    info = lstat_or_none(out)
    if info is not None and not stat.S_ISREG(info.st_mode):
        raise rm.Refusal(f"{out} is a symlink or not a regular file; refusing to write through it")
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(text, encoding="utf-8")
    print(f"release_cask: rendered {out} for {args.tag} with verified digest {digest}")
    return 0


def tap_formula_path(cask):
    """The formula that must no longer exist once the tap migrated to a cask."""
    tap = cask.parent.parent if cask.parent.name == "Casks" else cask.parent
    return tap / "Formula" / FORMULA_NAME


def cmd_check_downgrade(args):
    version = rm.check_tag(args.tag)
    cask = Path(args.cask)
    formula = tap_formula_path(cask)
    # lstat, not exists(): a dangling formula symlink is still a leftover formula.
    if lstat_or_none(formula) is not None:
        raise rm.Refusal(
            f"tap still has a formula at {formula}; the migration to the cask is incomplete; "
            "refusing the cask update"
        )
    info = lstat_or_none(cask)
    if info is None:
        raise rm.Refusal(
            f"tap has no cask at {cask}; a missing cask is an unexpected tap state, not a "
            "first publication; refusing the cask update"
        )
    if not stat.S_ISREG(info.st_mode):
        raise Guard(f"existing cask {cask} is a symlink or not a regular file; refusing the update")
    matches = CASK_VERSION_RE.findall(cask.read_text(encoding="utf-8"))
    if len(matches) != 1:
        raise Guard(f"cannot read exactly one version line from existing cask {cask} "
                    f"(found {len(matches)}); refusing the update")
    current = tuple(int(part) for part in matches[0].split("."))
    new = tuple(int(part) for part in version.split("."))
    if current >= new:
        raise Guard(
            f"tap cask is at {matches[0]}, which is not older than {version}; "
            "refusing the cask update"
        )
    print(f"release_cask: tap cask {matches[0]} is older than {version}; update allowed")
    return 0


def build_parser():
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = parser.add_subparsers(dest="command", required=True)

    render = sub.add_parser("render", help="render the cask from a verified manifest")
    render.add_argument("--dir", required=True)
    render.add_argument("--manifest", required=True)
    render.add_argument("--tag", required=True)
    render.add_argument("--commit", required=True)
    render.add_argument("--out", required=True)
    render.set_defaults(func=cmd_render)

    guard = sub.add_parser("check-downgrade", help="refuse a cask at or above the tag")
    guard.add_argument("--tag", required=True)
    guard.add_argument("--cask", required=True)
    guard.set_defaults(func=cmd_check_downgrade)
    return parser


def main(argv=None):
    args = build_parser().parse_args(argv)
    try:
        return args.func(args)
    except rm.Refusal as exc:
        print(f"release_cask: refused: {exc}", file=sys.stderr)
        return 1
    except rm.UsageError as exc:
        print(f"release_cask: invalid input: {exc}", file=sys.stderr)
        return 2
    except Guard as exc:
        print(f"release_cask: downgrade guard: {exc}", file=sys.stderr)
        return 3


if __name__ == "__main__":
    sys.exit(main())
