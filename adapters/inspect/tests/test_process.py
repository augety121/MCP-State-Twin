"""Exercise real child pipes without an MCP server or model dependency."""
import importlib.util
from pathlib import Path
import sys
import tempfile
import time
import unittest
from unittest.mock import patch

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

    def test_inherited_pipe_does_not_extend_deadline(self):
        with tempfile.TemporaryDirectory() as directory:
            stop, done, ready = (Path(directory) / name for name in ('stop', 'done', 'ready'))
            descendant = (
                "import pathlib,time; "
                f"stop=pathlib.Path({str(stop)!r}); "
                f"pathlib.Path({str(ready)!r}).touch(); end=time.monotonic()+10\n"
                "while not stop.exists() and time.monotonic()<end: time.sleep(.01)\n"
                f"pathlib.Path({str(done)!r}).touch()"
            )
            launcher = (
                "import subprocess,sys; "
                f"subprocess.Popen([sys.executable,'-c',{descendant!r}],close_fds=False)"
            )
            started = time.monotonic()
            try:
                with self.assertRaisesRegex(process.ProcessError, '^PLUGIN_OPERATION_TIMEOUT$'):
                    self.run_child(launcher, timeout=1)
                self.assertLess(time.monotonic() - started, 3)
                self.assertTrue(ready.exists(), 'descendant must hold inherited pipes')
            finally:
                stop.touch()
                deadline = time.monotonic() + 12
                while not done.exists() and time.monotonic() < deadline:
                    time.sleep(.01)
                self.assertTrue(done.exists(), 'test descendant did not exit')

    def test_interruption_waits_and_closes_direct_child(self):
        original = process.subprocess.Popen
        children = []

        def spawn(*args, **kwargs):
            child = original(*args, **kwargs)
            children.append(child)
            return child

        with patch.object(process.subprocess, 'Popen', side_effect=spawn), \
                patch.object(process.os, 'read', side_effect=KeyboardInterrupt):
            with self.assertRaises(KeyboardInterrupt):
                self.run_child('import time; time.sleep(30)')
        self.assertIsNotNone(children[0].returncode)
        self.assertTrue(children[0].stdout.closed and children[0].stderr.closed)


if __name__ == '__main__':
    unittest.main()
