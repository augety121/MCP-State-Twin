"""Bounded capture for the trusted CLI, with owned-child cleanup on every exit."""
import os
import subprocess
import time


class ProcessError(RuntimeError):
    pass


def capture(argv, timeout=120, stdout_limit=1 << 20, stderr_limit=256 << 10):
    child = subprocess.Popen(argv, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    output = [bytearray(), bytearray()]
    streams = (child.stdout, child.stderr)
    limits = (stdout_limit, stderr_limit)
    deadline = time.monotonic() + timeout
    try:
        # Python 3.12 supports nonblocking pipes on Windows as well as POSIX.
        # Never wait for EOF in a reader thread: inherited write handles can
        # outlive the direct child. The same deadline covers exit and draining.
        active = set(range(len(streams)))
        for stream in streams:
            os.set_blocking(stream.fileno(), False)
        while active or child.poll() is None:
            if time.monotonic() >= deadline:
                raise ProcessError("PLUGIN_OPERATION_TIMEOUT")
            progressed = False
            for index in tuple(active):
                try:
                    data = os.read(streams[index].fileno(),
                                   min(65536, limits[index] - len(output[index]) + 1))
                except BlockingIOError:
                    continue
                if not data:
                    active.remove(index)
                    continue
                if len(output[index]) + len(data) > limits[index]:
                    raise ProcessError("PLUGIN_RESOURCE_LIMIT")
                output[index].extend(data)
                progressed = True
            if not progressed:
                time.sleep(min(0.005, max(0, deadline - time.monotonic())))
    except OSError:
        raise ProcessError("PLUGIN_EVIDENCE_INVALID") from None
    finally:
        if child.poll() is None:
            child.kill()
        child.wait()
        for stream in streams:
            stream.close()
    if child.returncode:
        raise ProcessError("PLUGIN_EVIDENCE_INVALID")
    return bytes(output[0])
