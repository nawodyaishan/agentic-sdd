"""B4/T08 regressions for FR-04/AC-04: qualification contract, Python and archive lanes."""

import copy
import io
import json
import os
from pathlib import Path
import shutil
import struct
import subprocess
import tarfile
import tempfile
import unittest

import trusted_ci as gate


ROOT = Path(__file__).resolve().parents[2]
AUDITED_SUITES = [
    "test_core_workflow.py",
    "test_research_workflow.py",
    "test_supported_core_go.py",
    "test_appops_core_selection.py",
    "test_trusted_ci.py",
    "test_trusted_ci_container.py",
]


def _make_macho_64(cputype: int, payload: bytes = b"\x00" * 64) -> bytes:
    header = struct.pack("<IIIIIII", 0xFEEDFACF, cputype, 0, 2, 0, 0, 0)
    return header + payload


def _make_universal_macho(cputypes=(0x01000007, 0x0100000C)) -> bytes:
    slices = [_make_macho_64(ct) for ct in cputypes]
    header_size = 8 + 20 * len(slices)
    offset = (header_size + 15) & ~15
    fat_header = struct.pack(">II", 0xCAFEBABE, len(slices))
    arch_entries = bytearray()
    body = bytearray(b"\x00" * (offset - header_size))
    for ct, sl in zip(cputypes, slices):
        arch_entries += struct.pack(">IIIII", ct, 0, offset, len(sl), 4)
        body += sl
        offset += len(sl)
    return fat_header + bytes(arch_entries) + bytes(body)


def _write_paired_archive(directory: Path, members: dict[str, bytes], archive_name="agentic-sdd_0.1.0_darwin_all.tar.gz"):
    archive_path = directory / archive_name
    with tarfile.open(archive_path, "w:gz") as tf:
        for name, data in members.items():
            info = tarfile.TarInfo(name=name)
            info.size = len(data)
            info.mode = 0o755
            tf.addfile(info, io.BytesIO(data))
    import hashlib
    digest = hashlib.sha256(archive_path.read_bytes()).hexdigest()
    checksums = directory / "checksums.txt"
    checksums.write_text(f"{digest}  {archive_name}\n")
    return archive_path, checksums


class QualificationContractTests(unittest.TestCase):
    def setUp(self):
        self.q_path = ROOT / "harness/qualification.json"
        self.assertTrue(self.q_path.is_file(), "missing harness/qualification.json")
        self.contract = gate.load_json(self.q_path)
        self.checks_contract = gate.load_json(ROOT / "harness/checks.json")

    def test_existing_checks_contract_is_unchanged(self):
        selected = gate.required_contract(self.checks_contract, copy.deepcopy(self.checks_contract))
        self.assertEqual(
            [ch["id"] for ch in selected],
            ["mod-verify", "tidy-check", "vet", "test", "build-darwin", "diff-check"],
        )

    def test_qualification_contract_names_all_surfaces_and_audited_suites_only(self):
        validated = gate.qualification_contract(self.contract, copy.deepcopy(self.contract))
        lanes = validated["lanes"]
        self.assertEqual(set(lanes), {"core", "nested_tui", "python_suites", "archive_distribution"})
        tui = lanes["nested_tui"]
        self.assertEqual(tui["status"], "active")
        self.assertNotIn("blocker", tui)
        self.assertEqual(tui["cwd"], "experiments/tui")
        tui_checks = tui["checks"]
        self.assertEqual([c["id"] for c in tui_checks], ["tui-vendor-check", "tui-vet", "tui-test"])
        for ch in tui_checks:
            self.assertEqual(ch["argv"][0], "go")
            self.assertIn("-mod=vendor", ch["argv"])
            self.assertNotEqual(ch["argv"][:3], ["go", "mod", "verify"])

        suites = lanes["python_suites"]["suites"]
        self.assertEqual([Path(s["path"]).name for s in suites], AUDITED_SUITES)
        for s in suites:
            suite_file = Path(s["path"]).name
            self.assertEqual(
                s["argv"],
                ["python3", "-B", "-m", "unittest", "discover", "-s", "tests/harness", "-p", suite_file],
            )
            self.assertNotIn("test_b4_", " ".join(s["argv"]))

        arch = lanes["archive_distribution"]
        self.assertEqual(arch["shipped_applications"], ["asdx", "asdx-core", "agentic-sdd", "agentic-sdd-tui"])
        self.assertIs(arch["sdd_check_counted_as_shipped"], False)
        self.assertEqual(arch["darwin_architectures"], ["amd64", "arm64"])

    def test_blocked_nested_tui_contract_is_validated_as_blocked_not_active(self):
        blocked = copy.deepcopy(self.contract)
        blocked["lanes"]["nested_tui"] = {
            "status": "blocked",
            "cwd": "experiments/tui",
            "blocker": "owner module-supply decision pending (plan §12.5)",
        }
        validated = gate.qualification_contract(blocked, copy.deepcopy(blocked))
        self.assertEqual(validated["lanes"]["nested_tui"]["status"], "blocked")
        self.assertIn("12.5", validated["lanes"]["nested_tui"]["blocker"])
        with self.assertRaises(ValueError):
            gate.qualification_contract(self.contract, blocked)

    def test_negative_contract_mutations_are_rejected(self):
        mutations = []

        m_remove = copy.deepcopy(self.contract)
        m_remove["lanes"]["python_suites"]["suites"].pop()
        mutations.append(("remove-suite", m_remove))

        m_broad = copy.deepcopy(self.contract)
        m_broad["lanes"]["python_suites"]["suites"][0]["argv"] = [
            "python3", "-B", "-m", "unittest", "discover", "-s", "tests/harness", "-p", "test_*.py"
        ]
        mutations.append(("broad-discovery", m_broad))

        m_excluded = copy.deepcopy(self.contract)
        m_excluded["lanes"]["python_suites"]["suites"][0]["path"] = "tests/harness/test_opencode.py"
        m_excluded["lanes"]["python_suites"]["suites"][0]["argv"][-1] = "test_opencode.py"
        mutations.append(("excluded-suite", m_excluded))

        m_env = copy.deepcopy(self.contract)
        m_env["lanes"]["python_suites"]["suites"][0]["env"] = {"GOFLAGS": "-tags=agentic_sdd_experimental"}
        mutations.append(("hostile-env", m_env))

        m_tui_active_blocker = copy.deepcopy(self.contract)
        m_tui_active_blocker["lanes"]["nested_tui"]["blocker"] = "still blocked"
        mutations.append(("tui-active-with-blocker", m_tui_active_blocker))

        m_tui_no_blocker = copy.deepcopy(self.contract)
        m_tui_no_blocker["lanes"]["nested_tui"] = {"status": "blocked", "cwd": "experiments/tui"}
        mutations.append(("tui-blocked-without-blocker", m_tui_no_blocker))

        m_tui_no_vendor = copy.deepcopy(self.contract)
        m_tui_no_vendor["lanes"]["nested_tui"]["checks"][0]["argv"] = ["go", "mod", "verify"]
        mutations.append(("tui-drop-mod-vendor", m_tui_no_vendor))

        m_tui_drop_check = copy.deepcopy(self.contract)
        m_tui_drop_check["lanes"]["nested_tui"]["checks"].pop()
        mutations.append(("tui-drop-check", m_tui_drop_check))

        m_tui_cwd = copy.deepcopy(self.contract)
        m_tui_cwd["lanes"]["nested_tui"]["cwd"] = "."
        mutations.append(("tui-wrong-cwd", m_tui_cwd))

        m_sdd_check = copy.deepcopy(self.contract)
        m_sdd_check["lanes"]["archive_distribution"]["shipped_applications"] = [
            "asdx", "asdx-core", "agentic-sdd", "sdd-check"
        ]
        mutations.append(("count-sdd-check", m_sdd_check))

        for name, candidate in mutations:
            with self.subTest(mutation=name):
                with self.assertRaises(ValueError):
                    gate.qualification_contract(self.contract, candidate)
                with self.assertRaises(ValueError):
                    gate.qualification_contract(candidate, candidate)

    def test_candidate_cannot_designate_itself_trusted_in_qualify(self):
        with tempfile.TemporaryDirectory() as tmp:
            out = Path(tmp) / "out"
            with self.assertRaises(ValueError):
                gate.run_qualification(ROOT, ROOT, out)


class PythonQualificationRunnerTests(unittest.TestCase):
    def _make_repo_pair(self, tmp: Path):
        trusted = tmp / "trusted"
        candidate = tmp / "candidate"
        for root in (trusted, candidate):
            (root / "harness").mkdir(parents=True)
            (root / "tests/harness").mkdir(parents=True)
            (root / "go.mod").write_text("module agentic-sdd\n\ngo 1.23\n")
            shutil.copyfile(ROOT / "harness/checks.json", root / "harness/checks.json")
            shutil.copyfile(ROOT / "harness/qualification.json", root / "harness/qualification.json")
            shutil.copyfile(ROOT / "tests/harness/trusted_ci.py", root / "tests/harness/trusted_ci.py")
            for suite in AUDITED_SUITES:
                (root / "tests/harness" / suite).write_text(
                    "import unittest\nclass T(unittest.TestCase):\n    def test_ok(self): pass\n"
                )
        return trusted, candidate

    def test_missing_docker_blocks_container_suite_rather_than_passing(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            trusted, candidate = self._make_repo_pair(root)
            out = root / "out"
            code = gate.run_qualification(
                trusted,
                candidate,
                out,
                lane="python",
                suite_ids=["core-workflow", "trusted-ci-container"],
                prereq_overrides={"docker": False, "running Docker daemon": False, "preloaded_go_image": False},
            )
            self.assertEqual(code, 2)
            report = json.loads((out / "evidence/qualification.json").read_text())
            self.assertEqual(report["execution"], "blocked")
            self.assertEqual(report["lanes"]["nested_tui"]["status"], "not_run")
            self.assertNotIn("blocker", report["lanes"]["nested_tui"])
            by_id = {s["id"]: s for s in report["lanes"]["python_suites"]["suites"]}
            self.assertEqual(by_id["core-workflow"]["status"], "passed")
            self.assertEqual(by_id["trusted-ci-container"]["status"], "blocked")
            self.assertIn("docker", by_id["trusted-ci-container"]["missing_prerequisites"])

    def test_blocked_nested_tui_contract_reports_blocked_in_qualification_report(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            trusted, candidate = self._make_repo_pair(root)
            for repo in (trusted, candidate):
                q_path = repo / "harness/qualification.json"
                data = json.loads(q_path.read_text())
                data["lanes"]["nested_tui"] = {
                    "status": "blocked",
                    "cwd": "experiments/tui",
                    "blocker": "owner module-supply decision pending (plan §12.5)",
                }
                q_path.write_text(json.dumps(data))
            out = root / "out"
            code = gate.run_qualification(
                trusted,
                candidate,
                out,
                lane="python",
                suite_ids=["core-workflow"],
            )
            self.assertEqual(code, 0)
            report = json.loads((out / "evidence/qualification.json").read_text())
            self.assertEqual(report["execution"], "completed")
            self.assertEqual(report["lanes"]["nested_tui"]["status"], "blocked")
            self.assertIn("12.5", report["lanes"]["nested_tui"]["blocker"])

    def test_failing_python_suite_fails_qualification_lane(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            trusted, candidate = self._make_repo_pair(root)
            (candidate / "tests/harness/test_core_workflow.py").write_text(
                "import unittest\nclass T(unittest.TestCase):\n    def test_fail(self): self.fail('deliberate B4 negative')\n"
            )
            out = root / "out"
            code = gate.run_qualification(
                trusted,
                candidate,
                out,
                lane="python",
                suite_ids=["core-workflow"],
            )
            self.assertEqual(code, 1)
            report = json.loads((out / "evidence/qualification.json").read_text())
            self.assertEqual(report["execution"], "failed")
            self.assertEqual(report["lanes"]["python_suites"]["suites"][0]["status"], "failed")


class ArchiveAndPairSmokeTests(unittest.TestCase):
    def test_valid_paired_archive_passes_and_negative_fixtures_fail(self):
        universal = _make_universal_macho((0x01000007, 0x0100000C))
        valid_members = {name: universal for name in ("asdx", "asdx-core", "agentic-sdd", "agentic-sdd-tui")}

        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            good_dir = root / "good"
            good_dir.mkdir()
            archive, checksums = _write_paired_archive(good_dir, valid_members)
            summary = gate.verify_paired_archive(archive, checksums)
            self.assertEqual(set(summary["members"]), {"asdx", "asdx-core", "agentic-sdd", "agentic-sdd-tui"})
            for info in summary["members"].values():
                self.assertEqual(info["arches"], ["amd64", "arm64"])

            # Missing shipped name
            missing_dir = root / "missing"
            missing_dir.mkdir()
            m_missing = dict(valid_members)
            m_missing.pop("agentic-sdd-tui")
            a, c = _write_paired_archive(missing_dir, m_missing)
            with self.assertRaisesRegex(ValueError, "member"):
                gate.verify_paired_archive(a, c)

            # sdd-check cannot substitute or be counted as one of the four shipped names
            sdd_dir = root / "sdd"
            sdd_dir.mkdir()
            m_sdd = dict(valid_members)
            m_sdd["sdd-check"] = universal
            a, c = _write_paired_archive(sdd_dir, m_sdd)
            with self.assertRaisesRegex(ValueError, "member"):
                gate.verify_paired_archive(a, c)

            # Wrong slice (amd64 only, missing arm64)
            slice_dir = root / "slice"
            slice_dir.mkdir()
            m_slice = dict(valid_members)
            m_slice["asdx"] = _make_macho_64(0x01000007)
            a, c = _write_paired_archive(slice_dir, m_slice)
            with self.assertRaisesRegex(ValueError, "arm64|slice|arch"):
                gate.verify_paired_archive(a, c)

            # Non-Mach-O (ELF) disguised in archive
            elf_dir = root / "elf"
            elf_dir.mkdir()
            m_elf = dict(valid_members)
            m_elf["asdx-core"] = b"\x7fELF" + b"\x00" * 60
            a, c = _write_paired_archive(elf_dir, m_elf)
            with self.assertRaises(ValueError):
                gate.verify_paired_archive(a, c)

            # Checksum mismatch
            checksums.write_text("0" * 64 + f"  {archive.name}\n")
            with self.assertRaisesRegex(ValueError, "checksum"):
                gate.verify_paired_archive(archive, checksums)

    def test_native_pair_smoke_detects_mismatched_identity_and_missing_sibling(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            pair_dir = root / "pair"
            pair_dir.mkdir()
            stub_mod = root / "stubmod"
            stub_mod.mkdir()
            (stub_mod / "go.mod").write_text("module b4stub\n\ngo 1.23\n")
            (stub_mod / "main.go").write_text(
                'package main\n'
                'import ("encoding/json"; "fmt"; "os")\n'
                'var revision, commit, sourceDigest, distribution, legacyTarget, legacyName string\n'
                'func main() {\n'
                '  for _, a := range os.Args[1:] {\n'
                '    if a == "--protocol-info" {\n'
                '      b, _ := json.Marshal(map[string]any{\n'
                '        "product": "asdx",\n'
                '        "protocol_revision": 2,\n'
                '        "distribution": distribution,\n'
                '        "revision": revision,\n'
                '        "commit": commit,\n'
                '        "source_digest": sourceDigest,\n'
                '      })\n'
                '      os.Stdout.Write(append(b, \'\\n\'))\n'
                '      return\n'
                '    }\n'
                '  }\n'
                '  if legacyTarget != "" { fmt.Fprintf(os.Stderr, "%s: deprecated compatibility name; running %s\\n", legacyName, legacyTarget) }\n'
                '}\n'
            )
            cli_ldflags = (
                "-X agentic-sdd/internal/buildinfo.Revision=v0.0.0-b4test "
                "-X agentic-sdd/internal/buildinfo.Commit=1111111111111111111111111111111111111111 "
                "-X agentic-sdd/internal/buildinfo.SourceDigest=local-sha256:aaaa "
                "-X agentic-sdd/internal/buildinfo.DistributionMarker=generic"
            )
            stub_ldflags = (
                "-X main.revision=v0.0.0-b4test "
                "-X main.commit=1111111111111111111111111111111111111111 "
                "-X main.sourceDigest=local-sha256:aaaa "
                "-X main.distribution=generic"
            )
            env = {
                "PATH": os.environ.get("PATH", "/usr/bin:/bin"),
                "HOME": str(root / "home"),
                "TMPDIR": str(root / "tmp"),
                "GOTOOLCHAIN": "local",
                "GOPROXY": "off",
                "GOSUMDB": "off",
                "GOENV": "off",
                "GOWORK": "off",
                "GOFLAGS": "",
                "GOCACHE": os.environ.get("GOCACHE") or str(ROOT / ".cache/go-build"),
            }
            Path(env["HOME"]).mkdir()
            Path(env["TMPDIR"]).mkdir()

            subprocess.run(
                ["go", "build", "-trimpath", "-ldflags", cli_ldflags, "-o", str(pair_dir / "asdx-core"), "./cmd/agentic-sdd"],
                cwd=ROOT, env=env, check=True,
            )
            subprocess.run(
                ["go", "build", "-trimpath", "-ldflags", cli_ldflags + " -X main.legacyTarget=asdx-core -X main.legacyName=agentic-sdd",
                 "-o", str(pair_dir / "agentic-sdd"), "./cmd/agentic-sdd"],
                cwd=ROOT, env=env, check=True,
            )
            subprocess.run(
                ["go", "build", "-trimpath", "-ldflags", stub_ldflags, "-o", str(pair_dir / "asdx"), "."],
                cwd=stub_mod, env=env, check=True,
            )
            subprocess.run(
                ["go", "build", "-trimpath", "-ldflags", stub_ldflags + " -X main.legacyTarget=asdx -X main.legacyName=agentic-sdd-tui",
                 "-o", str(pair_dir / "agentic-sdd-tui"), "."],
                cwd=stub_mod, env=env, check=True,
            )

            smoke = gate.verify_native_pair_smoke(pair_dir)
            self.assertEqual(smoke["identity"]["commit"], "1111111111111111111111111111111111111111")
            self.assertEqual(smoke["positive_handshake_exit"], 0)
            self.assertEqual(smoke["missing_sibling_exit"], 1)
            self.assertEqual(smoke["mismatched_identity_exit"], 1)

            # Negative: replace asdx with mismatched identity in pair_dir and verify smoke fails
            bad_ldflags = (
                "-X main.revision=v0.0.0-other "
                "-X main.commit=2222222222222222222222222222222222222222 "
                "-X main.sourceDigest=local-sha256:bbbb "
                "-X main.distribution=generic"
            )
            subprocess.run(
                ["go", "build", "-trimpath", "-ldflags", bad_ldflags, "-o", str(pair_dir / "asdx"), "."],
                cwd=stub_mod, env=env, check=True,
            )
            with self.assertRaisesRegex(ValueError, "identity|mismatch|handshake"):
                gate.verify_native_pair_smoke(pair_dir)

    def test_qualification_cli_and_candidate_contract_weakening_block(self):
        universal = _make_universal_macho((0x01000007, 0x0100000C))
        valid_members = {name: universal for name in ("asdx", "asdx-core", "agentic-sdd", "agentic-sdd-tui")}
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            archive, checksums = _write_paired_archive(root, valid_members)
            ok = subprocess.run(
                ["python3", "-B", str(ROOT / "tests/harness/trusted_ci.py"), "verify-archive",
                 "--archive", str(archive), "--checksums", str(checksums)],
                capture_output=True, text=True,
            )
            self.assertEqual(ok.returncode, 0, ok.stdout + ok.stderr)

            self_trust = subprocess.run(
                ["python3", "-B", str(ROOT / "tests/harness/trusted_ci.py"), "qualify",
                 "--trusted", str(ROOT), "--candidate", str(ROOT), "--output", str(root / "self-out")],
                capture_output=True, text=True,
            )
            self.assertEqual(self_trust.returncode, 2)

            # Candidate modifying harness/qualification.json must block before execution
            trusted = root / "trusted"
            candidate = root / "candidate"
            for r in (trusted, candidate):
                (r / "harness").mkdir(parents=True)
                (r / "tests/harness").mkdir(parents=True)
                (r / "go.mod").write_text("module agentic-sdd\n\ngo 1.23\n")
                shutil.copyfile(ROOT / "harness/checks.json", r / "harness/checks.json")
                shutil.copyfile(ROOT / "harness/qualification.json", r / "harness/qualification.json")
                shutil.copyfile(ROOT / "tests/harness/trusted_ci.py", r / "tests/harness/trusted_ci.py")
            mutated = json.loads((candidate / "harness/qualification.json").read_text())
            mutated["lanes"]["python_suites"]["suites"].pop()
            (candidate / "harness/qualification.json").write_text(json.dumps(mutated))
            out = root / "blocked-out"
            code = gate.run_qualification(trusted, candidate, out, lane="python")
            self.assertEqual(code, 2)
            report = json.loads((out / "evidence/qualification.json").read_text())
            self.assertEqual(report["execution"], "blocked")

            # Archive lane without artifacts blocks rather than passing
            out_arch = root / "arch-blocked"
            shutil.copyfile(ROOT / "harness/qualification.json", candidate / "harness/qualification.json")
            code_arch = gate.run_qualification(trusted, candidate, out_arch, lane="archive")
            self.assertEqual(code_arch, 2)
            report_arch = json.loads((out_arch / "evidence/qualification.json").read_text())
            self.assertEqual(report_arch["execution"], "blocked")
            self.assertEqual(report_arch["lanes"]["archive_distribution"]["status"], "blocked")


if __name__ == "__main__":
    unittest.main()
