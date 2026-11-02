"""Pinned local tokenizer data in a network-none lab; NOT hosted-token proof.

Run only with the explicitly acquired wheel/data mounts. No Hub, Transformers,
remote model code, inference endpoint or automatic package resolution.
"""
import hashlib
import json
import os
from pathlib import Path
import sys
import unittest


@unittest.skipUnless(os.environ.get("SDD_TOKENIZER_SOURCE_FIXTURE") == "1", "requires approved isolated tokenizer mounts")
class TokenizerSourceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if os.getuid() == 0:
            raise RuntimeError("unprivileged network-none lab required")
        import tokenizers
        if tokenizers.__version__ != "0.22.1":
            raise RuntimeError("exact reviewed tokenizer wheel required")
        cls.data = Path("/token-source")
        manifest = json.loads((cls.data / "manifest.json").read_text())
        expected = {"tokenizer.json": "89085f12ef79460ac5f66d1119325ddfc694b4ab209d80bbd81d35f081dc9614",
                    "tokenizer_config.json": "841f8cf146e3f0ad1082594a31f68ecf7608c20467ef355081333d85bbaeb1cb"}
        if manifest != expected:
            raise RuntimeError("source manifest is not the reviewed public archive")
        cls.config = json.loads((cls.data / "tokenizer_config.json").read_text())
        cls.raw = json.loads((cls.data / "tokenizer.json").read_text())
        for name in ("tokenizer.json", "tokenizer_config.json"):
            if hashlib.sha256((cls.data / name).read_bytes()).hexdigest() != expected[name]:
                raise RuntimeError("public tokenizer data identity changed")
        cls.tokenizer = tokenizers.Tokenizer.from_file(str(cls.data / "tokenizer.json"))
        cls.bos = cls.config["bos_token"]["content"]
        cls.eos = cls.config["eos_token"]["content"]

    def test_no_remote_code_or_silent_truncation_is_loaded(self):
        self.assertNotIn("transformers", sys.modules)
        self.assertNotIn("huggingface_hub", sys.modules)
        self.assertIsNone(self.tokenizer.truncation)
        self.assertIsNone(self.tokenizer.padding)
        self.assertEqual([], self.tokenizer.encode("", add_special_tokens=False).ids)

    def test_special_token_ids_match_independent_public_data(self):
        # Derive expectations from the immutable source table, not encode().
        expected = {row["content"]: row["id"] for row in self.raw["added_tokens"]}
        for text in (self.bos, self.eos, "<｜User｜>", "<｜Assistant｜>"):
            with self.subTest(text=text):
                self.assertIn(text, expected)
                self.assertEqual([expected[text]], self.tokenizer.encode(text, add_special_tokens=False).ids)

    def test_known_text_and_long_input_do_not_use_character_estimates(self):
        vocab = self.raw["model"]["vocab"]
        self.assertEqual([vocab["Hello"], vocab["!"]], self.tokenizer.encode("Hello!", add_special_tokens=False).ids)
        # A source config's old max_length must not silently truncate/count low.
        encoded = self.tokenizer.encode(self.bos * 40000, add_special_tokens=False)
        self.assertEqual(40000, len(encoded.ids))
        self.assertGreater(len(encoded.ids), 32768)

    def test_plain_chat_example_has_explicit_local_not_hosted_identity(self):
        # Literal plain system/user projection inspected in the public template.
        # This never establishes how an external gateway expands tools/messages.
        plain = self.bos + "Synthetic system." + "<｜User｜>Hello!<｜Assistant｜>"
        count = len(self.tokenizer.encode(plain, add_special_tokens=False).ids)
        self.assertGreater(count, 0)
        print(json.dumps(dict(kind="public local-tokenizer source audit, not hosted input accounting",
                              local_plain_chat_tokens=count, source_model_max_length=self.config["model_max_length"],
                              hosted_projection_verified=False, model_requests=0,
                              dependency_version="tokenizers 0.22.1", transforms_or_hub_imported=False)))


if __name__ == "__main__":
    unittest.main()
