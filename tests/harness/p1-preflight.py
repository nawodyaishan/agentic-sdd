"""P1 host driver. Default fake; --live is the single allocation-bound observation."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import subprocess
import tempfile
import time


ROOT = Path(__file__).resolve().parents[2]
IMAGE = "sha256:57b29a7c050c4e679e4bea572e14b3965cf4b5839de2da49d55e49ffc4024f39"
BINARY_SHA256 = "8d060e1d2fbb9bb44cf09ae6ed1f3f6a7cff070b5227e2f90970132b23a92050"
REAL_CONTROL = Path("/tmp/opencode/agentic-sdd-p1-20261101-control")
OFFLINE_PROOF = Path("/tmp/opencode/agentic-sdd-p1-20261101-offline-proof.json")


def source_identity():
    names = ["p1_transport.py", "p1_native.py", "p1-preflight.py", "test_p1_transport.py", "test_p1_network.py", "test_p1_driver.py"]
    return hashlib.sha256(b"".join(name.encode() + b"\0" + hashlib.sha256((ROOT / "tests/harness" / name).read_bytes()).hexdigest().encode() + b"\n"
                                  for name in sorted(names))).hexdigest()


def remove_container(docker, cidfile):
    if not cidfile.exists():
        return True
    cid = cidfile.read_text().strip()
    if not re.fullmatch(r"[0-9a-f]{64}", cid):
        return False
    try:
        result = docker("rm", "--force", cid, capture_output=True)
        return result.returncode == 0 or b"No such container" in result.stderr
    except (OSError, subprocess.SubprocessError):
        return False


def run_policy(docker, tests, cidfile, output):
    try:
        with (output / "policy-tests.log").open("xb") as log:
            return docker(*tests, IMAGE, "python3", "-B", "-m", "unittest", "test_p1_transport", "test_p1_network", "test_p1_driver",
                          stdout=log, stderr=subprocess.STDOUT, timeout=45)
    finally:
        if not remove_container(docker, cidfile):
            raise ValueError("policy fixture cleanup uncertain; stop")


def write_observation(output, result, current_source):
    if current_source != result["source_identity"]:
        result.update(completed=False, source_changed=True)
    with (output / "observation.json").open("x") as stream:
        json.dump(result, stream, indent=2); stream.write("\n")
        stream.flush(); os.fsync(stream.fileno())


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--live", action="store_true")
    mode.add_argument("--tests-only", action="store_true")
    args = parser.parse_args()
    os.umask(0o077)
    output = Path(tempfile.mkdtemp(prefix="sdd-p1-live." if args.live else "sdd-p1-fake.", dir="/tmp/opencode"))
    control = REAL_CONTROL if args.live else output / "control"
    binary = Path(shutil.which("opencode")).resolve(strict=True)
    if hashlib.sha256(binary.read_bytes()).hexdigest() != BINARY_SHA256:
        raise ValueError("reviewed exact 2.2.5 binary changed; no execution")
    env = {"PATH": "/usr/local/bin:/usr/bin:/bin", "HOME": str(output), "LC_ALL": "C"}
    def docker(*parts, **options):
        return subprocess.run(["docker", *parts], env=env, timeout=options.pop("timeout", 20), **options)
    docker("image", "inspect", IMAGE, check=True, capture_output=True)
    name = "sdd-p1-" + secrets.token_hex(6)
    token = secrets.token_hex(32)
    deadline = time.monotonic() + 600
    envfile = output / "local-capability.env"
    envfile.write_text("P1_GATE_TOKEN=" + token + "\nP1_MODE=" + ("live" if args.live else "fake") +
                       "\nP1_DEADLINE=" + str(deadline) + "\nP1_REAL_ALLOCATION=p1-20261101-one-of-six\n")
    base = ["run", "--rm", "--pull=never", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges",
            "--pids-limit=128", "--memory=1g", "--cpus=1", "--tmpfs", "/tmp:rw,exec,nosuid,nodev,mode=1777,size=512m"]
    source = source_identity()
    if args.live:
        if OFFLINE_PROOF.is_symlink() or OFFLINE_PROOF.stat().st_mode & 0o077:
            raise ValueError("private offline proof required")
        proof = json.loads(OFFLINE_PROOF.read_text())
        if not proof.get("completed") or proof.get("source_identity") != source or proof.get("native_binary_sha256") != BINARY_SHA256:
            raise ValueError("fresh matching offline controls required before live execution")
        # A dated, independently inspected public price excerpt is required.
        price = json.loads(Path("/tmp/opencode/agentic-sdd-p1-20261101-price-proof.json").read_text())
        if (price.get("model") != "opencode/deepseek-v4.1-flash-free" or price.get("date") != "2026-11-01" or
                price.get("input_usd") != 0 or price.get("output_usd") != 0 or price.get("cache_read_usd") != 0):
            raise ValueError("exact current zero-price route not verified")
    policy_cid = output / "policy.cid"
    if not args.live:
        tests = [*base, "--cidfile", str(policy_cid), "--network=none", "--user", "65534:65534", "-e", "P1_NETWORK_FIXTURE=1", "--workdir", "/fixture/tests/harness"]
        for file in ("p1_transport.py", "test_p1_transport.py", "test_p1_network.py", "test_p1_driver.py", "p1-preflight.py"):
            tests += ["--mount", f"type=bind,src={ROOT}/tests/harness/{file},dst=/fixture/tests/harness/{file},readonly"]
        checked = run_policy(docker, tests, policy_cid, output)
        if checked.returncode:
            print("Offline policy/courier tests failed; no topology or live request", flush=True)
            return 1
        if args.tests_only:
            print("Offline policy/courier tests passed (partial scope)", flush=True)
            return 0
    # Exclusive creation after proof checks; never adopt/reset an existing live allocation.
    control.mkdir(mode=0o700)
    cids = []
    created_network = False
    result = dict(kind="P1 real observation" if args.live else "P1 offline topology proof", output=str(output),
                  model="opencode/deepseek-v4.1-flash-free", max_upstream_attempts=1, real_requested=args.live,
                  source_identity=source, native_binary_sha256=BINARY_SHA256, phase="setup", completed=False)
    print("Private P1 evidence:", output, flush=True)
    try:
        docker("network", "create", "--internal", "--opt", "com.docker.network.bridge.gateway_mode_ipv4=isolated",
               name, check=True, capture_output=True)
        created_network = True
        inspected = json.loads(docker("network", "inspect", name, check=True, capture_output=True).stdout)[0]
        if not inspected["Internal"] or inspected["Options"].get("com.docker.network.bridge.gateway_mode_ipv4") != "isolated":
            raise ValueError("internal isolated topology not established")
        result["network_internal_isolated"] = True
        gate_cid = output / "gate.cid"; cids.append(gate_cid)
        gate = docker(*base, "-d", "--cidfile", str(gate_cid), "--network", name, "--network-alias", "p1-gate",
            "--user", f"{os.getuid()}:{os.getgid()}", "--env-file", str(envfile), "--mount",
            f"type=bind,src={control},dst=/control", "--mount", f"type=bind,src={ROOT}/tests/harness/p1_transport.py,dst=/p1_transport.py,readonly",
            IMAGE, "python3", "-I", "-B", "/p1_transport.py", check=True, capture_output=True)
        gate_id = gate.stdout.decode().strip()
        if not re.fullmatch(r"[0-9a-f]{64}", gate_id):
            raise ValueError("gate identity unavailable")
        # Also in fake mode: prove runtime isolation with a dual-homed gate,
        # while fake code can only dispatch to its own loopback fake upstream.
        docker("network", "connect", "--gw-priority", "1", "bridge", gate_id, check=True, capture_output=True)
        runtime_cid = output / "runtime.cid"; cids.append(runtime_cid)
        result["phase"] = "native-client"
        with (output / "native-observation.log").open("xb") as stream:
            native = docker(*base, "--cidfile", str(runtime_cid), "--network", name, "--user", "65534:65534",
                "--env-file", str(envfile), "--mount", f"type=bind,src={binary},dst=/usr/local/bin/opencode,readonly",
                "--mount", f"type=bind,src={ROOT}/tests/harness/p1_native.py,dst=/p1_native.py,readonly",
                IMAGE, "python3", "-I", "-B", "/p1_native.py", stdout=stream, stderr=subprocess.STDOUT, timeout=150)
        result["native_container_exit"] = native.returncode
        result["phase"] = "stopped-after-at-most-one"
        result["completed"] = native.returncode == 0
    except (OSError, ValueError, subprocess.SubprocessError):
        result["reason"] = "setup/client/control failure; reconcile stable allocation, never retry live automatically"
    finally:
        for path in reversed(cids):
            if not remove_container(docker, path):
                result["completed"] = False; result["cleanup_uncertain"] = True
        if created_network:
            try:
                removed = docker("network", "rm", name, capture_output=True)
                if removed.returncode:
                    result["completed"] = False; result["cleanup_uncertain"] = True
            except (OSError, subprocess.SubprocessError):
                result["completed"] = False; result["cleanup_uncertain"] = True
        journal = control / "allocation.jsonl"
        if journal.exists():
            records = [json.loads(line) for line in journal.read_text().splitlines()]
            result["gate_observation"] = records[-1]
            result["journal_sha256"] = hashlib.sha256(journal.read_bytes()).hexdigest()
        for key in ("native-observation.log", "gate.cid", "runtime.cid"):
            path = output / key
            if path.exists():
                result[key + "_sha256"] = hashlib.sha256(path.read_bytes()).hexdigest()
        write_observation(output, result, source_identity())
        if result["completed"] and not args.live:
            gate = result.get("gate_observation", {})
            if gate.get("attempts") != 1 or gate.get("real_upstream") is not False or gate.get("result", {}).get("status") != "observed":
                raise ValueError("native fake completion and single dispatch were not observed")
            # This is check evidence, never human consent or a workflow ledger.
            if OFFLINE_PROOF.exists():
                raise ValueError("existing proof retained; reconcile rather than overwrite")
            with OFFLINE_PROOF.open("x") as stream:
                json.dump(result, stream, indent=2); stream.write("\n")
    print(json.dumps(result), flush=True)
    return 0 if result["completed"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
