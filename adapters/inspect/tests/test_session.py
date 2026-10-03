import asyncio
import json
import os
from pathlib import Path
import shutil
import subprocess

import pytest
from inspect_ai.tool import ToolDef

from statetwin_inspect import PluginError, Session, verified_report


@pytest.fixture(scope="session")
def assets(tmp_path_factory):
    repo = Path(__file__).resolve().parents[3]
    binary = Path(os.environ["STATETWIN_TEST_BINARY"]).resolve()
    root = tmp_path_factory.mktemp("plugin") / "examples"
    shutil.copytree(repo / "examples", root,
                    ignore=shutil.ignore_patterns(".statetwin", "*.stb"))
    for domain in ("issue-tracker", "package-registry"):
        base = root / domain
        (base / ".statetwin").mkdir()
        for world in (("agent",) if domain == "issue-tracker" else ("agent", "project")):
            for reviewed in (False, True):
                src = base / "reviewed" / "world" / world if reviewed else base
                name = ("reviewed-" if reviewed else "") + world + "-world.stb"
                subprocess.run([str(binary), "bundle", "build", "--manifest", str(src / f"bundle-{world}.yaml"),
                                "--out", str(base / ".statetwin" / name)], check=True, capture_output=True)
    return binary, root


def plan_for(root, entry, suffix=""):
    plan = json.loads((root / "plugin-session.json").read_text())
    plan["host"]["name"], plan["host"]["version"] = "inspect", "0.3.275"
    plan["entryId"] = entry
    pack = json.loads((root / "baseline-pack.json").read_text())
    ref = next(e for e in pack["entries"] if e["id"] == entry)
    task = json.loads((root / ref["root"] / ref["task"]).read_text())
    plan["deadlineSeconds"] = task["budgets"]["episodeSeconds"]
    plan["id"] = "inspect-" + entry + suffix
    plan["output"] = plan["id"]
    name = plan["id"] + ".json"
    (root / name).write_text(json.dumps(plan))
    return name


def test_inspect_session_lifecycle_all_24(assets):
    binary, root = assets
    pack = json.loads((root / "baseline-pack.json").read_text())
    async def run():
        for entry in pack["entries"]:
            cases = json.loads((root / entry["root"] / entry["cases"]).read_text())
            task_id = json.loads((root / entry["root"] / entry["task"]).read_text())["id"]
            case = next(c for c in cases["cases"] if c["taskId"] == task_id and c["role"] == "positive")
            witness = json.loads((root / entry["root"] / case["witness"]).read_text())
            name = plan_for(root, entry["id"])
            session = Session(binary, root, name)
            public_text = json.dumps(session.public["task"])
            assert "oracle" not in public_text and "expectedOutcome" not in public_text
            async with session.connect():
                tools = {ToolDef(t).name: t for t in session.tools}
                assert set(tools) == set(session.public["task"]["tools"])
                for call in witness["calls"]:
                    try:
                        await tools[call["tool"]](**call["input"])
                    except Exception as error:
                        # Expected tool errors are part of refusal/fault witnesses;
                        # the trusted terminal below must still match the oracle.
                        from inspect_ai.tool import ToolError
                        if not isinstance(error, ToolError):
                            raise
                report = await session.finish(witness.get("answer"))
                assert report["decision"] == "passed", (entry["id"], report)
            checked = verified_report(binary, root, name)
            assert checked == report
            with pytest.raises(PluginError, match="RESUME_UNSUPPORTED"):
                async with session.connect():
                    pass
    asyncio.run(run())


def test_inspect_scorer_ignores_self_report_and_cleans_up(assets):
    binary, root = assets
    async def run():
        name = plan_for(root, "read-issue", "-self-report")
        session = Session(binary, root, name)
        async with session.connect():
            report = await session.finish({"success": True, "score": 1})
        assert report["decision"] == "failed"
        assert verified_report(binary, root, name)["decision"] == "failed"
        name = plan_for(root, "read-issue", "-exception")
        session = Session(binary, root, name)
        with pytest.raises(RuntimeError, match="test cancellation"):
            async with session.connect():
                raise RuntimeError("test cancellation")
        assert session.tools == []
        raw = json.loads((root / "inspect-read-issue-exception" / "session-report.json").read_text())
        assert raw["execution"] == "interrupted" and raw["decision"] == "invalid"
        with pytest.raises(PluginError):
            verified_report(binary, root, name)
    asyncio.run(run())


def test_inspect_rejects_additional_tools_before_start(assets):
    binary, root = assets
    name = plan_for(root, "read-issue", "-extra")
    path = root / name
    plan = json.loads(path.read_text())
    plan["host"]["additionalTools"] = 1
    path.write_text(json.dumps(plan))
    with pytest.raises(PluginError):
        Session(binary, root, name)
    assert not (root / plan["output"]).exists()


def _run_framework(assets, tmp_path, suffix):
    from inspect_ai import eval as inspect_eval
    from inspect_ai.model import ModelOutput, ChatCompletionChoice, ChatMessageAssistant, get_model
    from inspect_ai.tool import ToolCall
    from statetwin_inspect.task import plugin_task
    binary, root = assets
    name = plan_for(root, "read-issue", suffix)
    calls = ModelOutput(model="mockllm/model", choices=[ChatCompletionChoice(
        message=ChatMessageAssistant(content="", tool_calls=[ToolCall(
            id="read-1", function="get_issue", arguments={"owner": "octo", "repository": "demo", "number": 1})]),
        stop_reason="tool_calls")])
    answer = ModelOutput.from_content(model="mockllm/model", content=json.dumps(
        {"repository": "octo/demo", "number": 1, "state": "open", "title": "Target issue"}))
    model = get_model("mockllm/model", custom_outputs=[calls, answer])
    logs = inspect_eval(plugin_task(str(binary), str(root), name), model=model,
                        log_dir=str(tmp_path), log_format="json", retry_on_error=0,
                        fail_on_error=True, max_samples=1, display="none")
    assert len(logs) == 1 and logs[0].status == "success", logs[0].error
    assert next(iter(logs[0].samples[0].scores.values())).value == 1
    # The framework may log public tool results, never the control capability
    # or private grading/world definitions.
    text = logs[0].model_dump_json()
    for forbidden in ("STATETWIN_PLUGIN_CONTROL_TOKEN", "statetwin.dev/plugin-world-evidence", "allowed-business-changes", "worldReplayVerified"):
        assert forbidden not in text


def test_inspect_task_solver_scorer_and_log_boundary(assets, tmp_path):
    _run_framework(assets, tmp_path, "-framework")


@pytest.mark.skipif(os.environ.get("STATETWIN_INSPECT_PERFORMANCE") != "1",
                    reason="opt-in release performance qualification")
def test_inspect_framework_performance(assets, tmp_path):
    import platform
    import time
    from importlib.metadata import version
    binary, original = assets
    root = tmp_path / "inputs"
    shutil.copytree(original, root, ignore=shutil.ignore_patterns("inspect-*"))
    pack_path = root / "baseline-pack.json"
    pack = json.loads(pack_path.read_text())
    pack["entries"] = [e for e in pack["entries"] if e["id"] == "read-issue"]
    pack_path.write_text(json.dumps(pack))
    samples = []
    for index in range(30):
        started = time.perf_counter()
        _run_framework((binary, root), tmp_path / f"logs-{index}", f"-performance-{index}")
        samples.append(time.perf_counter() - started)
    ordered = sorted(samples)
    report = {"format": "statetwin.dev/inspect-performance/v1alpha1",
              "os": platform.system(), "arch": platform.machine(),
              "python": platform.python_version(), "inspect": version("inspect_ai"),
              "samples": samples, "p50Seconds": ordered[14], "p95Seconds": ordered[28],
              "maxSeconds": ordered[29], "source": "mock-framework-contract",
              "scope": "warm Python imports; plan/projection + Inspect eval + independent MCP child + original grading + scorer replay + logs + cleanup; asset build excluded",
              "rssStatus": "unavailable", "startupStatus": "not-separately-measured"}
    destination = os.environ.get("STATETWIN_INSPECT_PERFORMANCE_OUT")
    if destination:
        Path(destination).write_text(json.dumps(report, indent=2), encoding="utf-8")
    print(json.dumps(report))


def test_inspect_actual_cancellation_and_concurrency(assets):
    binary, root = assets
    async def run():
        first = Session(binary, root, plan_for(root, "read-issue", "-cancel"))
        second = Session(binary, root, plan_for(root, "read-issue", "-parallel"))
        ready = asyncio.Event()
        async def work():
            async with first.connect():
                ready.set()
                await asyncio.Event().wait()
        worker = asyncio.create_task(work())
        await asyncio.wait_for(ready.wait(), 15)
        with pytest.raises(PluginError, match="CONCURRENCY_UNSUPPORTED"):
            async with second.connect():
                pass
        worker.cancel()
        with pytest.raises(asyncio.CancelledError):
            await worker
        assert first.tools == []
        assert (root / "inspect-read-issue-cancel" / "claim.json").is_file()
        with pytest.raises(PluginError):
            verified_report(binary, root, first.plan)
    asyncio.run(run())
