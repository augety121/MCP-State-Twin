"""Bounded capture for the trusted CLI, with owned-child cleanup on every exit."""
import subprocess
import threading


class ProcessError(RuntimeError):
    pass


def capture(argv, timeout=120, stdout_limit=1 << 20, stderr_limit=256 << 10):
    child = subprocess.Popen(argv, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    output = [bytearray(), bytearray()]
    failed = threading.Event()
    readers = []

    def read(stream, index, limit):
        try:
            while True:
                data = stream.read1(min(65536, limit - len(output[index]) + 1))
                if not data:
                    return
                if len(output[index]) + len(data) > limit:
                    failed.set()
                    child.kill()
                    return
                output[index].extend(data)
        except Exception:
            failed.set()
            try:
                child.kill()
            except OSError:
                pass

    try:
        for index, (stream, limit) in enumerate(
                ((child.stdout, stdout_limit), (child.stderr, stderr_limit))):
            reader = threading.Thread(target=read, args=(stream, index, limit))
            reader.start()
            readers.append(reader)
        try:
            child.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            raise ProcessError("PLUGIN_OPERATION_TIMEOUT") from None
    finally:
        if child.poll() is None:
            child.kill()
        child.wait()
        for reader in readers:
            reader.join()
        child.stdout.close()
        child.stderr.close()
    if failed.is_set():
        raise ProcessError("PLUGIN_RESOURCE_LIMIT")
    if child.returncode:
        raise ProcessError("PLUGIN_EVIDENCE_INVALID")
    return bytes(output[0])
