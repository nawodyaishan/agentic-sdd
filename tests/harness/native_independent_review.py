"""Credential-free native API probes for the independent-review example.

Run ONLY with native-independent-review.sh. It evaluates permissions and config
inventory through the same endpoint allowlist as native_opencode.py. It submits
no prompt, starts no model, reads no credentials and does not dispatch a child
agent or any tool. Permission evaluation is narrower evidence than a real
reviewer invocation, and a permission result is not an OS sandbox.
"""
import base64
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import time
import unittest
from urllib.parse import urlencode
from urllib.request import Request, urlopen

EXPECTED_VERSION = "opencode v2.2.5"
BINARY_SHA256 = "8d060e1d2fbb9bb44cf09ae6ed1f3f6a7cff070b5227e2f90970132b23a92050"
COORDINATOR = "sdd-review-coordinator"
REVIEWER = "sdd-independent-reviewer"
DENIED_FOR_BOTH = [
    ("edit", ["specs/demo/tasks.md"]), ("shell", ["touch changed"]), ("execute", ["*"]),
    ("grep", [".*"]), ("glob", ["**/*"]), ("webfetch", ["https://example.test"]),
    ("websearch", ["query"]), ("question", ["ask"]),
    ("opencode_list_mcp_resources", ["synthetic"]),
    ("opencode_read_mcp_resource", ["synthetic:file:///private"]),
    ("browser", ["*"]), ("chrome_devtools_evaluate_script", ["*"]),
    ("external_directory", ["/private/*"]),
    ("read", ["app.env"]), ("read", ["opencode.db-wal"]), ("read", [".git/config"]),
    ("skill", ["agentic-sdd-implement"]),
]


class IndependentReviewProbe(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if os.environ.get("SDD_NATIVE_FIXTURE") != "1" or os.getuid() == 0:
            raise RuntimeError("use the unprivileged network-disabled container wrapper")
        binary = Path("/usr/local/bin/opencode")
        if hashlib.sha256(binary.read_bytes()).hexdigest() != BINARY_SHA256:
            raise RuntimeError("binary is not the pinned OpenCode 2.2.5 build")
        result = subprocess.run(["opencode", "--version"], capture_output=True, text=True, timeout=30, check=True)
        if result.stdout.strip() != EXPECTED_VERSION:
            raise RuntimeError("version mismatch: " + result.stdout.strip())
        cls.root = Path("/tmp/fixture")
        cls.root.mkdir(mode=0o700)
        cls.project = cls.root / "project"
        (cls.project / ".opencode").mkdir(parents=True)
        (cls.project / "specs/demo").mkdir(parents=True)
        (cls.project / "src").mkdir()
        cls.global_dir = Path(os.environ["XDG_CONFIG_HOME"]) / "opencode"
        cls.global_dir.mkdir(parents=True, exist_ok=True)
        Path(os.environ["HOME"]).mkdir(mode=0o700, exist_ok=True)
        cls.example = json.loads(Path("/templates/independent-review.jsonc").read_text())
        cls.lockdown = json.loads(Path("/templates/policy-lockdown.jsonc").read_text())
        cls.log = (cls.root / "server.log").open("w")
        os.environ["OPENCODE_PASSWORD"] = "credential-free-local-fixture"
        cls.auth = "Basic " + base64.b64encode(b"opencode:credential-free-local-fixture").decode()
        cls.server = subprocess.Popen(["opencode", "serve", "--hostname", "127.0.0.1", "--port", "4096"],
                                      cwd=cls.project, stdout=cls.log, stderr=cls.log)
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
        # Same deliberate allowlist as native_opencode.py: no inference or command routes.
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
        # Initialization time after reload, not a provider retry or a passing result.
        time.sleep(1)

    def setUp(self):
        self.write_config(self.project / ".opencode/opencode.jsonc", self.example)
        self.write_config(self.global_dir / "opencode.jsonc", {})
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

    def test_01_profiles_visible_as_primary_and_subagent(self):
        agents = {a["id"]: a for a in self.api("GET", "/api/agent", scoped=True)["data"]}
        self.assertEqual(agents[COORDINATOR]["mode"], "primary")
        self.assertEqual(agents[REVIEWER]["mode"], "subagent")
        for name in (COORDINATOR, REVIEWER):
            provider, _, model_id = self.example["agents"][name]["model"].partition("/")
            if "model" in agents[name]:
                self.assertEqual(agents[name]["model"], {"id": model_id, "providerID": provider}, name)
            if "variant" in agents[name]:
                self.assertEqual(agents[name]["variant"], self.example["agents"][name]["variant"], name)

    def test_02_no_commands_providers_or_integrations_exported(self):
        commands = {c["name"] for c in self.api("GET", "/api/command", scoped=True)["data"]}
        self.assertNotIn(COORDINATOR, commands)
        self.assertNotIn(REVIEWER, commands)
        self.assertEqual(self.api("GET", "/api/model", scoped=True)["data"], [])
        self.assertEqual(self.api("GET", "/api/mcp", scoped=True)["data"], [])

    def test_03_coordinator_allows_only_named_reads_skill_and_reviewer_launch(self):
        self.effect(COORDINATOR, "read", ["specs/demo/spec.md"], "allow")
        self.effect(COORDINATOR, "skill", ["agentic-sdd-verification-review"], "allow")
        self.effect(COORDINATOR, "subagent", [REVIEWER], "allow")
        self.effect(COORDINATOR, "subagent", ["general"], "deny")
        self.effect(COORDINATOR, "subagent", [COORDINATOR], "deny")
        for action, resources in DENIED_FOR_BOTH:
            with self.subTest(agent=COORDINATOR, action=action):
                self.effect(COORDINATOR, action, resources, "deny")

    def test_04_reviewer_allows_reads_and_skill_but_never_delegation(self):
        self.effect(REVIEWER, "read", ["src/main.go"], "allow")
        self.effect(REVIEWER, "read", ["specs/demo/tasks.md"], "allow")
        self.effect(REVIEWER, "skill", ["agentic-sdd-verification-review"], "allow")
        self.effect(REVIEWER, "subagent", [REVIEWER], "deny")
        self.effect(REVIEWER, "subagent", [COORDINATOR], "deny")
        self.effect(REVIEWER, "subagent", ["general"], "deny")
        for action, resources in DENIED_FOR_BOTH:
            with self.subTest(agent=REVIEWER, action=action):
                self.effect(REVIEWER, action, resources, "deny")

    def test_05_global_child_denial_blocks_the_route_without_weakening_it(self):
        self.write_config(self.global_dir / "opencode.jsonc", self.lockdown)
        # Documented collision: a higher-authority subagent denial stays authoritative.
        self.effect(COORDINATOR, "subagent", [REVIEWER], "deny")
        self.effect(REVIEWER, "read", ["src/main.go"], "allow")
        self.effect(REVIEWER, "subagent", [REVIEWER], "deny")


if __name__ == "__main__":
    unittest.main()
