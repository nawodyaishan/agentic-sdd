"""Explicit S0 container driver; no real endpoint, installs, credentials or inference."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile


ROOT = Path(__file__).resolve().parents[2]
IMAGE = "sha256:57b29a7c050c4e679e4bea572e14b3965cf4b5839de2da49d55e49ffc4024f39"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output")
    parser.add_argument("--case", choices=("budget-http", "native-shape", "native-compat", "separate-worker"))
    args = parser.parse_args()
    os.umask(0o077)
    output = Path(args.output).absolute() if args.output else Path(tempfile.mkdtemp(prefix="sdd-s0.", dir="/tmp/opencode"))
    if (".." in output.parts or output.resolve().is_relative_to(ROOT) or
            any(p.is_symlink() for p in [output, *output.parents])):
        raise ValueError("private external non-symlink output required")
    output.mkdir(mode=0o700, parents=True, exist_ok=True)
    if list(output.iterdir()) or output.stat().st_mode & 0o077:
        raise ValueError("new or empty private output required")
    binary = Path(os.environ.get("OPENCODE_TEST_BINARY") or shutil.which("opencode")).resolve(strict=True)
    env = {"PATH": "/usr/local/bin:/usr/bin:/bin", "HOME": str(output), "LC_ALL": "C"}
    subprocess.run(["docker", "image", "inspect", IMAGE], env=env, check=True, capture_output=True, timeout=15)
    base = ["docker", "run", "--rm", "--pull=never", "--network=none", "--read-only", "--cap-drop=ALL",
            "--security-opt=no-new-privileges", "--pids-limit=128", "--memory=1g", "--cpus=1", "--user=65534:65534",
            "--tmpfs", "/tmp:rw,exec,nosuid,nodev,mode=1777,size=512m"]
    runtime = ["-e", "SDD_S0_CONTAINER=1", "-e", "SDD_NATIVE_FIXTURE=1", "-e", "HOME=/tmp/home",
               "-e", "XDG_CONFIG_HOME=/tmp/config", "-e", "XDG_DATA_HOME=/tmp/data", "-e", "XDG_CACHE_HOME=/tmp/cache",
               "-e", "XDG_STATE_HOME=/tmp/state", "-e", "SDD_NATIVE_EXPECTED_VERSION=opencode v2.2.5", "--workdir", "/fixture",
               "--mount", f"type=bind,src={binary},dst=/usr/local/bin/opencode,readonly",
               "--mount", f"type=bind,src={ROOT}/integrations/opencode,dst=/templates,readonly"]
    names = ["offline_transport.py", "test_offline_transport.py", "test_s0_native.py", "native_opencode.py"]
    for name in names:
        runtime += ["--mount", f"type=bind,src={ROOT}/tests/harness/{name},dst=/fixture/{name},readonly"]
    inputs = output / "worker-input"; inputs.mkdir(mode=0o700)
    inputs.chmod(0o755)  # Synthetic mount only; external enclosing root stays 0700.
    (inputs / "sentinel.txt").write_text("synthetic worker input\n")
    (inputs / "sentinel.txt").chmod(0o644)
    # A synthetic host canary, never real user/home/auth data.
    sentinel = output / "host-only.txt"; sentinel.write_text("synthetic host only")
    worker = ["--mount", f"type=bind,src={inputs},dst=/input,readonly", "--mount",
              f"type=bind,src={ROOT}/tests/harness/s0_worker.py,dst=/worker.py,readonly"]
    cases = [("budget-http", runtime, ["python3", "-B", "-m", "unittest", "test_offline_transport"]),
             ("native-shape", runtime, ["python3", "-B", "-m", "unittest", "test_s0_native"]),
             ("native-compat", runtime, ["python3", "-B", "/fixture/native_opencode.py"]),
             ("separate-worker", worker, ["python3", "-B", "/worker.py", str(sentinel)])]
    if args.case:
        cases = [case for case in cases if case[0] == args.case]
    result = dict(kind="S0 offline fixtures, not native-model acceptance", image=IMAGE,
                  expected_native_version="opencode v2.2.5", native_binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),
                  model_requests=0, selection=[case[0] for case in cases], cases=[])
    print("Private S0 evidence:", output, flush=True)
    for name, mounts, command in cases:
        cidfile = output / (name + ".cid")
        log = output / (name + ".log")
        try:
            with log.open("xb") as stream:
                process = subprocess.run([*base, "--cidfile", str(cidfile), *mounts, IMAGE, *command],
                                         env=env, stdout=stream, stderr=subprocess.STDOUT, timeout=120)
            result["cases"].append(dict(id=name, exit_code=process.returncode, log=log.name,
                                        log_sha256=hashlib.sha256(log.read_bytes()).hexdigest()))
            print(name, "exit", process.returncode, flush=True)
            if process.returncode:
                break
        except (OSError, subprocess.TimeoutExpired):
            result["cases"].append(dict(id=name, exit_code=None, reason="fixture unavailable/interrupted; not a pass"))
            break
        finally:
            if cidfile.exists():
                cid = cidfile.read_text().strip()
                if not re.fullmatch(r"[0-9a-f]{64}", cid):
                    raise ValueError("owned container identity invalid")
                cleaned = subprocess.run(["docker", "rm", "--force", cid], env=env, capture_output=True, timeout=10)
                if cleaned.returncode and b"No such container" not in cleaned.stderr:
                    raise ValueError("owned cleanup uncertain; stop")
    result["completed"] = len(result["cases"]) == len(cases) and all(c["exit_code"] == 0 for c in result["cases"])
    (output / "observation.json").write_text(json.dumps(result, indent=2) + "\n")
    return 0 if result["completed"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
