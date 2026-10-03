"""One fresh subprocess per sample; explicit finish and no retry/resume."""
import asyncio
from contextlib import asynccontextmanager, contextmanager
from importlib.metadata import version
import json
import os
from pathlib import Path
import secrets
import socket
import subprocess
import sys
import tempfile
import time
import threading

from inspect_ai.tool import mcp_connection, mcp_server_stdio


class PluginError(RuntimeError):
    """Finite errors intentionally exclude input, credentials and child output."""


_active_session = threading.Lock()


@contextmanager
def _serial_session():
    if not _active_session.acquire(blocking=False):
        raise PluginError("PLUGIN_CONCURRENCY_UNSUPPORTED")
    try:
        yield
    finally:
        _active_session.release()


def _versions():
    if (sys.version_info[:2] != (3, 12) or version("inspect-ai") != "0.3.275"
            or version("mcp") != "1.26.0"):
        raise PluginError("PLUGIN_HOST_UNSUPPORTED")


def _cli(binary, root, plan, command):
    try:
        # Both output streams are bounded on disk, not accumulated by communicate.
        with tempfile.TemporaryFile() as out, tempfile.TemporaryFile() as err:
            child = subprocess.Popen([str(Path(binary).resolve()), "plugin", command,
                                      "--root", str(Path(root).resolve()),
                                      "--session-plan", plan], stdout=out, stderr=err)
            try:
                child.wait(timeout=120)
            except BaseException:
                child.kill()
                child.wait()
                raise
            if out.tell() > 1 << 20 or err.tell() > 256 << 10:
                raise PluginError("PLUGIN_RESOURCE_LIMIT")
            out.seek(0)
            data = json.load(out)
            if child.returncode or not isinstance(data, dict):
                raise PluginError("PLUGIN_EVIDENCE_INVALID")
            return data
    except Exception:
        raise PluginError("PLUGIN_EVIDENCE_INVALID") from None


def projection(binary, root, plan):
    _versions()
    data = _cli(binary, root, plan, "projection")
    host = data["plan"]["host"]
    if host != {"name": "inspect", "version": "0.3.275", "model": "mock/statetwin",
                "freshSession": True, "toolsOnly": True, "additionalTools": 0}:
        raise PluginError("PLUGIN_HOST_UNSUPPORTED")
    if set(data["task"]) != {"taskId", "objective", "context", "tools"}:
        raise PluginError("PLUGIN_PROJECTION_INVALID")
    return data


def verified_report(binary, root, plan):
    report = _cli(binary, root, plan, "inspect")
    if report["evidence"] != "world-replay-verified" or report["cleanup"] != "complete":
        raise PluginError("PLUGIN_EVIDENCE_INVALID")
    return report


class _Control:
    def __init__(self, address, token, session_id):
        self.address, self.token, self.session_id = address, token, session_id
        self.stream = None
        self.sock = None
        self.sequence = 0

    def connect(self):
        deadline = time.monotonic() + 10
        while True:
            try:
                if os.name == "nt":
                    self.stream = open(self.address, "r+b", buffering=0)
                else:
                    self.sock = socket.socket(socket.AF_UNIX)
                    self.sock.settimeout(10)
                    self.sock.connect(self.address)
                    self.stream = self.sock.makefile("rwb", buffering=0)
                break
            except OSError:
                if self.sock:
                    self.sock.close()
                if time.monotonic() >= deadline:
                    raise PluginError("PLUGIN_STARTUP_TIMEOUT") from None
                time.sleep(0.02)
        ready = self.exchange({"version": "v1", "sessionId": self.session_id, "token": self.token})
        self.token = None
        if ready.get("status") != "ready":
            raise PluginError("PLUGIN_STARTUP_FAILED")
        if self.sock:
            self.sock.settimeout(30)

    def exchange(self, request):
        raw = json.dumps(request, separators=(",", ":")).encode() + b"\n"
        if len(raw) > 64 << 10:
            raise PluginError("PLUGIN_RESOURCE_LIMIT")
        try:
            # Unbuffered streams may short-write, so explicitly complete the frame.
            view = memoryview(raw)
            while view:
                n = self.stream.write(view)
                if not n:
                    raise OSError()
                view = view[n:]
            response = self.stream.readline((64 << 10) + 1)
            if not response.endswith(b"\n") or len(response) > 64 << 10:
                raise PluginError("PLUGIN_CONTROL_INVALID")
            result = json.loads(response)
            if (result.get("version") != "v1" or result.get("sessionId") != self.session_id
                    or result.get("sequence") != request.get("sequence", 0)):
                raise PluginError("PLUGIN_CONTROL_INVALID")
            return result
        except Exception:
            raise PluginError("PLUGIN_CONTROL_INVALID") from None

    def request(self, operation, outcome, answer=None):
        self.sequence += 1
        req = {"version": "v1", "sessionId": self.session_id, "sequence": self.sequence,
               "operation": operation, "hostOutcome": outcome}
        if answer is not None:
            req["answer"] = answer
        return self.exchange(req)

    def close(self):
        if self.stream:
            self.stream.close()
        if self.sock:
            self.sock.close()


class Session:
    def __init__(self, binary, root, plan):
        self.binary, self.root, self.plan = binary, root, plan
        self.public = projection(binary, root, plan)
        self.tools = []
        self._used = False
        self._sealed = False
        self._control = None

    @asynccontextmanager
    async def connect(self):
        if self._used:
            raise PluginError("PLUGIN_RESUME_UNSUPPORTED")
        self._used = True
        # macOS's default TMPDIR can exceed Unix-domain socket path limits.
        with _serial_session(), tempfile.TemporaryDirectory(prefix="stp-", dir=None if os.name == "nt" else "/tmp") as directory:
            name = "statetwin-" + secrets.token_hex(16)
            address = "\\\\.\\pipe\\" + name if os.name == "nt" else str(Path(directory) / (name + ".sock"))
            token = secrets.token_hex(32)
            control = _Control(address, token, self.public["plan"]["id"])
            self._control = control
            server = mcp_server_stdio(name="statetwin", command=str(Path(self.binary).resolve()),
                args=["plugin", "serve", "--root", str(Path(self.root).resolve()), "--session-plan", self.plan],
                env={"STATETWIN_PLUGIN_CONTROL": address, "STATETWIN_PLUGIN_CONTROL_TOKEN": token})
            # Inspect stores MCP connections per asyncio task: never enter this
            # context with wait_for/create_task (which would create a second world).
            async with mcp_connection(server):
                try:
                    await asyncio.to_thread(control.connect)
                    self.tools = await server.tools()
                    async with asyncio.timeout(self.public["plan"]["deadlineSeconds"]):
                        yield self
                finally:
                    try:
                        if not self._sealed:
                            try:
                                await asyncio.to_thread(control.request, "abort", "cancelled")
                            except Exception:
                                pass  # Server EOF also seals an interrupted terminal.
                    finally:
                        control.close()
                        self.tools = []

    async def finish(self, answer=None):
        if self._sealed or self._control is None:
            raise PluginError("PLUGIN_STATE_CONFLICT")
        if len(json.dumps(answer).encode()) > 32 << 10:
            raise PluginError("PLUGIN_RESOURCE_LIMIT")
        response = await asyncio.to_thread(self._control.request, "finish", "completed", answer)
        self._sealed = True
        if response.get("status") != "sealed":
            raise PluginError("PLUGIN_EVIDENCE_INVALID")
        return response["report"]
