#!/usr/bin/env python3
"""Exercise bounded resources and owned cleanup of the canonical verifier."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]


class VerifyResourcesTest(unittest.TestCase):
    def run_verifier(self, memory=None):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            binary = directory / "docker"
            calls = directory / "calls.jsonl"
            binary.write_text(
                "#!/usr/bin/env python3\n"
                "import json, os, pathlib, sys\n"
                "args = sys.argv[1:]\n"
                "with open(os.environ['VERIFY_TEST_CALLS'], 'a') as f:\n"
                "    f.write(json.dumps(args) + '\\n')\n"
                "if args and args[0] == 'run' and '--cidfile' in args:\n"
                "    pathlib.Path(args[args.index('--cidfile') + 1]).write_text('a' * 64)\n"
            )
            binary.chmod(0o755)
            environment = os.environ.copy()
            environment["PATH"] = str(directory) + os.pathsep + environment["PATH"]
            environment["VERIFY_TEST_CALLS"] = str(calls)
            environment.pop("MSGVAULT_VERIFY_MEMORY_MB", None)
            if memory is not None:
                environment["MSGVAULT_VERIFY_MEMORY_MB"] = memory
            result = subprocess.run(
                ["bash", str(ROOT / "scripts/verify-downstream.sh")],
                cwd=ROOT, env=environment, capture_output=True, text=True, check=False,
            )
            observed = [json.loads(line) for line in calls.read_text().splitlines()] if calls.exists() else []
            return result, observed

    def assert_memory(self, expected, calls):
        run = next(call for call in calls if call[0] == "run")
        for flag in ("--memory", "--memory-swap"):
            value = next((item[len(flag) + 1:] for item in run if item.startswith(flag + "=")), None)
            if value is None:
                value = run[run.index(flag) + 1]
            amount = int(value[:-1]) * (1024 if value.endswith("g") else 1)
            self.assertEqual(expected, amount)

    def test_default_keeps_two_gibibyte_cap(self):
        result, calls = self.run_verifier()
        self.assertEqual(0, result.returncode, result.stderr)
        self.assert_memory(2048, calls)

    def test_lower_bounded_cap_and_owned_cleanup(self):
        result, calls = self.run_verifier("1536")
        self.assertEqual(0, result.returncode, result.stderr)
        self.assert_memory(1536, calls)
        self.assertIn(["rm", "-f", "a" * 64], calls)

    def test_invalid_caps_fail_before_any_docker_call(self):
        for value in ("0", "511", "2049", "4096", "1536m", "1536;echo", "999999999999999999999999"):
            with self.subTest(value=value):
                result, calls = self.run_verifier(value)
                self.assertNotEqual(0, result.returncode)
                self.assertEqual([], calls)


if __name__ == "__main__":
    unittest.main()
