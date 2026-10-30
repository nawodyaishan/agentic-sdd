"""Credential-free native API probes. Run ONLY with native-opencode.sh.

No inference, command submission, shell execution, credential API or run CLI.
Permission evaluation is narrower evidence than invoking an agent tool.
"""
import json
import base64
import os
from pathlib import Path
import re
import subprocess
import time
import unittest
from urllib.parse import urlencode
from urllib.request import Request, urlopen


class NativeTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if os.environ.get("SDD_NATIVE_FIXTURE") != "1" or os.getuid() == 0:
            raise RuntimeError("use the unprivileged network-disabled container wrapper")
        cls.root = Path("/tmp/fixture")
        cls.root.mkdir(mode=0o700)
        cls.project = cls.root / "project"
        (cls.project / ".opencode").mkdir(parents=True)
        cls.global_dir = Path(os.environ["XDG_CONFIG_HOME"]) / "opencode"
        cls.global_dir.mkdir(parents=True, exist_ok=True)
        Path(os.environ["HOME"]).mkdir(mode=0o700, exist_ok=True)
        cls.template = json.loads(Path("/templates/opencode.jsonc").read_text())
        cls.lockdown = json.loads(Path("/templates/policy-lockdown.jsonc").read_text())
        result = subprocess.run(["opencode", "--version"], capture_output=True, text=True, timeout=30, check=True)
        if result.stdout.strip() != "opencode v2.2.0":
            raise RuntimeError("fixtures require the pinned 2.2.0 binary")
        cls.log = (cls.root / "server.log").open("w")
        os.environ["OPENCODE_PASSWORD"] = "credential-free-local-fixture"
        cls.auth = "Basic " + base64.b64encode(b"opencode:credential-free-local-fixture").decode()
        cls.server = subprocess.Popen(["opencode", "serve", "--hostname", "127.0.0.1", "--port", "4096"], cwd=cls.project, stdout=cls.log, stderr=cls.log)
        for _ in range(100):
            try:
                with urlopen(Request("http://127.0.0.1:4096/api/info", headers={"Authorization": cls.auth}), timeout=1) as response:
                    json.load(response)
                break
            except OSError:
                if cls.server.poll() is not None:
                    raise RuntimeError((cls.root / "server.log").read_text())
                time.sleep(0.1)
        else:
            cls.server.terminate()
            raise RuntimeError("local fixture server unavailable")

    @classmethod
    def tearDownClass(cls):
        cls.server.terminate()
        try:
            cls.server.wait(timeout=10)
        except subprocess.TimeoutExpired:
            cls.server.kill()
            cls.server.wait(timeout=5)
        cls.log.close()

    def api(self, method, path, body=None, *, scoped=False):
        # Deliberate positive endpoint allowlist: adding an inference call needs
        # a different, separately approved program, not a runtime option here.
        safe_get = {"/api/info", "/api/agent", "/api/command", "/api/model", "/api/mcp", "/api/config", "/api/plugin", "/api/skill"}
        safe_post = path in {"/api/session", "/api/location/reload"} or re.fullmatch(r"/api/session/ses_[\w-]+/permission", path)
        self.assertTrue((method == "GET" and path in safe_get) or (method == "POST" and safe_post))
        if scoped:
            path += "?" + urlencode({"location[directory]": str(self.project)})
        request = Request("http://127.0.0.1:4096" + path, method=method,
                          data=json.dumps(body).encode() if body is not None else None,
                          headers={"Content-Type": "application/json", "Authorization": self.auth})
        with urlopen(request, timeout=30) as response:
            return None if response.status == 204 else json.load(response)

    def write_config(self, path, value):
        path.write_text(json.dumps(value))
        self.api("POST", "/api/location/reload")
        # 2.2.0 builds extension registries after the reload response. This is
        # initialization time, not a provider retry or a passing test result.
        time.sleep(1)

    def setUp(self):
        # All paths here are scratch created by this process inside tmpfs.
        self.config = json.loads(json.dumps(self.template).replace("__FEATURE__", "demo").replace("__BUILD_PATH__", "src"))
        self.write_config(self.project / ".opencode/opencode.jsonc", self.config)
        self.write_config(self.global_dir / "opencode.jsonc", self.lockdown)
        direct = self.project / "opencode.jsonc"
        if direct.exists():
            direct.unlink()

    def session(self, agent):
        data = self.api("POST", "/api/session", {"agent": agent, "location": {"directory": str(self.project)}})
        return data["data"]["id"]

    def effect(self, agent, action, resources, expected):
        sid = self.session(agent)
        result = self.api("POST", f"/api/session/{sid}/permission", {"agent": agent, "action": action, "resources": resources}, scoped=True)
        self.assertEqual(result["data"]["effect"], expected, f"{agent} {action} {resources}: {result}")

    def test_01_profiles_commands_and_no_provider(self):
        agents = {a["id"]: a for a in self.api("GET", "/api/agent", scoped=True)["data"]}
        for name in ("sdd-draft", "sdd-build", "sdd-review"):
            self.assertIn(name, agents)
            self.assertEqual(agents[name]["mode"], "primary")
        commands = {c["name"]: c for c in self.api("GET", "/api/command", scoped=True)["data"]}
        for name in self.config["commands"]:
            self.assertIn(name, commands)
            # The list API exports only name/description, not execution fields.
            self.assertEqual(commands[name]["description"], self.config["commands"][name]["description"])
        self.assertEqual(self.api("GET", "/api/model", scoped=True)["data"], [])
        self.assertEqual(self.api("GET", "/api/mcp", scoped=True)["data"], [])

    def test_02_draft_path_and_build_permissions(self):
        self.effect("sdd-draft", "edit", ["specs/demo/spec.md"], "allow")
        self.effect("sdd-draft", "edit", ["src/main.go"], "deny")
        self.effect("sdd-draft", "edit", ["specs/other/spec.md"], "deny")
        self.effect("sdd-draft", "edit", ["specs/demo/spec.md", "src/main.go"], "deny")
        self.effect("sdd-build", "edit", ["src/main.go"], "allow")
        self.effect("sdd-build", "edit", [".opencode/opencode.jsonc"], "deny")
        self.effect("sdd-build", "shell", ["go test ./..."], "ask")
        for agent in self.config["agents"]:
            self.effect(agent, "read", ["app.env"], "deny")
            self.effect(agent, "read", ["opencode.db-wal"], "deny")
            self.effect(agent, "external_directory", ["/private/*"], "deny")
            self.effect(agent, "subagent", ["general"], "deny")

    def test_03_review_routes_deny(self):
        self.effect("sdd-review", "read", ["src/main.go"], "allow")
        for action, resource in [
            ("edit", "specs/demo/tasks.md"), ("shell", "touch changed"),
            ("execute", "*"), ("webfetch", "https://example.test"),
            ("websearch", "query"), ("grep", ".*"), ("glob", "**/*"),
            ("opencode_list_mcp_resources", "synthetic"),
            ("opencode_read_mcp_resource", "synthetic:file:///private"),
            ("synthetic_write", "*"), ("chrome_devtools_evaluate_script", "*"),
            ("browser", "*"),
        ]:
            with self.subTest(action=action):
                self.effect("sdd-review", action, [resource], "deny")

    def test_04_global_denial_overrides_project_allow(self):
        self.config["policies"].extend([
            {"action": "provider.use", "resource": "*", "effect": "allow"},
            {"action": "tool.use", "resource": "execute:*", "effect": "allow"},
        ])
        self.config["agents"]["sdd-build"]["permissions"].append({"action": "execute", "resource": "*", "effect": "allow"})
        self.write_config(self.project / ".opencode/opencode.jsonc", self.config)
        self.assertEqual(self.api("GET", "/api/model", scoped=True)["data"], [])
        self.effect("sdd-build", "execute", ["*"], "deny")

    def test_05_config_and_command_collision(self):
        self.write_config(self.project / "opencode.jsonc", {
            "commands": {"sdd-review": {"template": "direct definition must lose"}},
            "agents": {"sdd-review": {"description": "direct definition must lose"}},
        })
        commands = {c["name"]: c for c in self.api("GET", "/api/command", scoped=True)["data"]}
        self.assertEqual(commands["sdd-review"]["description"], self.config["commands"]["sdd-review"]["description"])
        sources = self.api("GET", "/api/config", scoped=True)
        self.assertIn("opencode.jsonc", json.dumps(sources))

    def test_06_invalid_policy_warns_not_enforces(self):
        self.config["policies"].append({"action": "not-a-policy", "resource": "*", "effect": "deny"})
        self.write_config(self.project / ".opencode/opencode.jsonc", self.config)
        self.api("GET", "/api/agent", scoped=True)
        logs = "\n".join(p.read_text(errors="replace") for p in Path("/tmp").rglob("*.log"))
        warnings = [line for line in logs.lower().splitlines()
                    if "polic" in line and re.search(r"invalid|dropp|warn", line)]
        self.assertTrue(warnings, "no invalid-policy warning observed")

    def test_07_integration_denial_prevents_mcp_startup(self):
        sentinel = self.root / "mcp-started"
        self.config["mcp"] = {"servers": {"synthetic": {
            "type": "local", "command": ["/bin/sh", "-c", f"echo started > {sentinel}"]
        }}}
        # A lower-authority attempt to re-enable the connector must lose.
        self.config["policies"].append({"action": "integration.use", "resource": "mcp:synthetic", "effect": "allow"})
        self.write_config(self.project / ".opencode/opencode.jsonc", self.config)
        self.assertEqual(self.api("GET", "/api/mcp", scoped=True)["data"], [])
        self.assertFalse(sentinel.exists(), "denied connector started")
        # Positive control in the disposable container proves the fixture is
        # executable, rather than a silently ignored MCP definition.
        control = json.loads(json.dumps(self.lockdown))
        control["policies"].append({"action": "integration.use", "resource": "mcp:synthetic", "effect": "allow"})
        self.write_config(self.global_dir / "opencode.jsonc", control)
        self.api("GET", "/api/mcp", scoped=True)
        self.assertTrue(sentinel.exists(), "MCP positive control did not start; denial result inconclusive")

    def test_08_unapproved_plugin_does_not_execute(self):
        sentinel = self.root / "plugin-started"
        plugins = self.project / ".opencode/plugins"
        plugins.mkdir()
        (plugins / "synthetic.js").write_text(
            "import { writeFileSync } from 'node:fs';\n"
            f"writeFileSync({json.dumps(str(sentinel))}, 'started');\n"
            "export default async () => ({});\n"
        )
        self.api("POST", "/api/location/reload")
        time.sleep(1)
        self.api("GET", "/api/plugin", scoped=True)
        self.assertFalse(sentinel.exists(), "unapproved plugin executed")
        control = json.loads(json.dumps(self.lockdown))
        control["policies"].append({"action": "integration.use", "resource": "plugin:" + str(plugins / "synthetic.js"), "effect": "allow"})
        self.config["policies"].append(control["policies"][-1])
        self.write_config(self.project / ".opencode/opencode.jsonc", self.config)
        self.write_config(self.global_dir / "opencode.jsonc", control)
        self.api("GET", "/api/plugin", scoped=True)
        self.assertTrue(sentinel.exists(), "plugin positive control did not execute; denial result inconclusive")

    def test_09_skill_collision_has_project_winner(self):
        for base, marker in ((self.global_dir, "GLOBAL_MARKER"), (self.project / ".opencode", "PROJECT_MARKER")):
            skill = base / "skills/agentic-sdd-fixture"
            skill.mkdir(parents=True)
            (skill / "SKILL.md").write_text(
                "---\nname: agentic-sdd-fixture\ndescription: Synthetic duplicate fixture\n---\n" + marker + "\n"
            )
        self.api("POST", "/api/location/reload")
        time.sleep(1)
        skills = self.api("GET", "/api/skill", scoped=True)["data"]
        matches = [s for s in skills if s.get("id", s.get("name")) == "agentic-sdd-fixture"]
        self.assertEqual(len(matches), 1)
        exported = json.dumps(matches[0])
        self.assertIn("PROJECT_MARKER", exported)
        self.assertNotIn("GLOBAL_MARKER", exported)


if __name__ == "__main__":
    unittest.main(verbosity=2)
