"""Offline unit checks; these do not enable native or model execution."""
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from native_independent_review_dispatch import native_environment


class DispatchEnvironmentTests(unittest.TestCase):
    def test_pwd_matches_project_instead_of_inherited_ssh_home(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with patch.dict("os.environ", {"PWD": "/home/stale-ssh-location",
                                           "OPENAI_API_KEY": "must-not-be-inherited"}):
                env = native_environment(root, root / "fixture")
            self.assertEqual(str(root / "fixture"), env["PWD"])
            self.assertEqual(str(root / "home"), env["HOME"])
            self.assertNotIn("OPENAI_API_KEY", env)
            for key in ("XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME"):
                self.assertTrue(Path(env[key]).is_relative_to(root))


if __name__ == "__main__":
    unittest.main()
