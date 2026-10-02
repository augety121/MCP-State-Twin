"""Inspect task/solver/scorer: mock-only until a separate live profile is accepted."""
import asyncio
import json

from inspect_ai import Task, task
from inspect_ai.dataset import Sample
from inspect_ai.scorer import Score, accuracy, scorer
from inspect_ai.solver import solver

from .session import PluginError, Session, projection, verified_report


@solver
def plugin_solver(binary: str, root: str, plan: str):
    async def solve(state, generate):
        if str(state.model) != "mockllm/model" or state.tools or len(state.messages) != 1:
            raise PluginError("PLUGIN_HOST_UNSUPPORTED")
        session = Session(binary, root, plan)
        async with session.connect():
            state.tools = session.tools
            try:
                state = await generate(state, cache=False, max_retries=0)
                answer = state.output.completion
                try:
                    answer = json.loads(answer)
                except (ValueError, TypeError):
                    pass
                await session.finish(answer)
            finally:
                state.tools = []
        return state
    return solve


@scorer(metrics=[accuracy()])
def plugin_scorer(binary: str, root: str, plan: str):
    async def score(state, target):
        # Ignores self-reported grades and framework metadata; independently
        # checks the raw terminal against the frozen Task and world.
        report = await asyncio.to_thread(verified_report, binary, root, plan)
        return Score(value=1 if report["decision"] == "passed" else 0,
                     explanation="State Twin terminal: " + report["decision"],
                     metadata={"sourceTrust": report["sourceTrust"],
                               "hostObservation": report["hostObservation"]})
    return score


@task
def plugin_task(binary: str, root: str, plan: str):
    public = projection(binary, root, plan)["task"]
    return Task(name="statetwin-offline-contract",
                dataset=[Sample(id=public["taskId"], input=public["objective"] + "\n" + public["context"])],
                solver=plugin_solver(binary, root, plan), scorer=plugin_scorer(binary, root, plan),
                epochs=1, message_limit=64)
