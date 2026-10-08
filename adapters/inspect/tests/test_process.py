"""Exercise real child pipes without an MCP server or model dependency."""
import importlib.util
from pathlib import Path
import sys
import unittest

source = Path(__file__).resolve().parents[1] / "src/statetwin_inspect/_process.py"
spec = importlib.util.spec_from_file_location("bounded_cli_process", source)
process = importlib.util.module_from_spec(spec)
spec.loader.exec_module(process)


class CaptureTests(unittest.TestCase):
    def run_child(self, code, **kwargs):
        return process.capture([sys.executable, "-c", code], **kwargs)

    def test_exact_bound_and_both_pipes(self):
        result = self.run_child("import os; os.write(2, b'e'*4096); os.write(1, b'x'*4096)",
                                stdout_limit=4096, stderr_limit=4096)
        self.assertEqual(result, b'x'*4096)

    def test_overflow_kills_child_before_its_timeout(self):
        for fd in (1, 2):
            with self.subTest(fd=fd), self.assertRaisesRegex(process.ProcessError, 'PLUGIN_RESOURCE_LIMIT'):
                self.run_child(f"import os,time; os.write({fd}, b'x'*4097); time.sleep(30)",
                               timeout=5, stdout_limit=4096, stderr_limit=4096)

    def test_failure_does_not_disclose_output(self):
        with self.assertRaisesRegex(process.ProcessError, '^PLUGIN_EVIDENCE_INVALID$'):
            self.run_child("import sys; print('synthetic private error', file=sys.stderr); sys.exit(3)")

    def test_timeout_reaps_owned_child(self):
        with self.assertRaisesRegex(process.ProcessError, '^PLUGIN_OPERATION_TIMEOUT$'):
            self.run_child("import time; time.sleep(30)", timeout=0.1)


if __name__ == '__main__':
    unittest.main()
