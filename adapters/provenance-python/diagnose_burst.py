"""Stage 1 observation only: private AST timing copies, never a recorder fix.

No application payload is logged. Instrumentation perturbs scheduling and cost;
its result must stay separate from the unchanged-engine burst reproduction.
"""
import ast
from collections import defaultdict
from contextlib import contextmanager
import importlib
from pathlib import Path
import threading
import time


class Meter:
    def __init__(self):
        self.local = threading.local()
        self.lock = threading.Lock()
        self.values = defaultdict(lambda: {"count": 0, "total_ns": 0, "min_ns": None, "max_ns": 0})

    def add(self, probe, phase, elapsed):
        with self.lock:
            value = self.values[probe + ":" + phase]
            value["count"] += 1
            value["total_ns"] += elapsed
            value["min_ns"] = elapsed if value["min_ns"] is None else min(value["min_ns"], elapsed)
            value["max_ns"] = max(value["max_ns"], elapsed)

    @contextmanager
    def call(self, probe):
        key = probe.spec["module"] + "." + probe.spec["function"]
        calls = getattr(self.local, "calls", [])
        if not calls:
            self.local.calls = calls
        frame = {"key": key, "body_ns": 0}
        before = probe.owner.recorder._counts.get("sampled_out:" + key, 0)
        calls.append(frame)
        started = time.perf_counter_ns()
        try:
            yield
        finally:
            elapsed = time.perf_counter_ns() - started
            calls.pop()
            after = probe.owner.recorder._counts.get("sampled_out:" + key, 0)
            phase = "unsampled_wrapper" if after > before else "selected_wrapper"
            self.add(key, phase, elapsed - frame["body_ns"])

    def target(self, probe, original, args, kwargs):
        started = time.perf_counter_ns()
        try:
            return original(*args, **kwargs)
        finally:
            calls = getattr(self.local, "calls", [])
            if calls:
                calls[-1]["body_ns"] += time.perf_counter_ns() - started

    @contextmanager
    def phase(self, name):
        stack = getattr(self.local, "phases", [])
        if not stack:
            self.local.phases = stack
        frame = [0]
        stack.append(frame)
        started = time.perf_counter_ns()
        try:
            yield
        finally:
            elapsed = time.perf_counter_ns() - started
            stack.pop()
            if stack:
                stack[-1][0] += elapsed
            calls = getattr(self.local, "calls", [])
            key = calls[-1]["key"] if calls else "outside_probe"
            self.add(key, name + "_inclusive", elapsed)
            self.add(key, name + "_exclusive", elapsed - frame[0])

    def json_encode(self, function, *args, **kwargs):
        with self.phase("envelope_json"):
            return function(*args, **kwargs)

    def report(self):
        return {key: {**value, "mean_us": value["total_ns"] / value["count"] / 1000}
                for key, value in sorted(self.values.items())}


class Instrument(ast.NodeTransformer):
    def __init__(self):
        self.cls, self.function = "", ""
        self.insertions = defaultdict(int)

    def scope(self, node, phase, call=False):
        self.insertions[phase] += 1
        expression = ast.Call(ast.Attribute(ast.Name("_burst_meter", ast.Load()), "call" if call else "phase", ast.Load()),
                             [ast.Name("self", ast.Load())] if call else [ast.Constant(phase)], [])
        return ast.copy_location(ast.With([ast.withitem(expression)], [node]), node)

    def visit_ClassDef(self, node):
        previous, self.cls = self.cls, node.name
        node = self.generic_visit(node)
        self.cls = previous
        return node

    def visit_FunctionDef(self, node):
        previous, self.function = self.function, node.name
        node = self.generic_visit(node)
        phase = None
        call = self.cls == "Probe" and node.name == "wrapped"
        if call:
            phase = "wrapper_call"
        elif not self.cls and node.name == "bounded":
            phase = "value_snapshot"
        elif self.cls == "Recorder":
            phase = {"record_decision": "record_prepare", "_enqueue": "queue_admission", "_encode": "worker_otlp_encode"}.get(node.name)
        if phase:
            node.body = [self.scope(ast.If(ast.Constant(True), node.body, []), phase, call)]
        self.function = previous
        return node

    def visit_If(self, node):
        sampling = self.cls == "Probe" and self.function == "wrapped" and ast.unparse(node.test) == "self.sampling"
        node = self.generic_visit(node)
        return self.scope(node, "sampling_decision") if sampling else node

    def visit_Try(self, node):
        snapshot = self.cls == "Probe" and self.function == "wrapped" and node.body and ast.unparse(node.body[0]).startswith("bound = self.signature.bind(")
        node = self.generic_visit(node)
        return self.scope(node, "argument_snapshot") if snapshot else node

    def visit_Call(self, node):
        node = self.generic_visit(node)
        if self.cls == "Probe" and self.function in ("wrapped", "unsampled", "method_wrapped") and isinstance(node.func, ast.Name) and node.func.id == "original":
            self.insertions["target_body"] += 1
            return ast.copy_location(ast.Call(ast.Attribute(ast.Name("_burst_meter", ast.Load()), "target", ast.Load()),
                [ast.Name("self", ast.Load()), node.func, ast.Name("args", ast.Load()), ast.Name("kwargs", ast.Load())], []), node)
        if self.cls == "Recorder" and self.function == "record_decision" and isinstance(node.func, ast.Attribute) and isinstance(node.func.value, ast.Name) and node.func.value.id == "json" and node.func.attr == "dumps":
            self.insertions["envelope_json"] += 1
            return ast.copy_location(ast.Call(ast.Attribute(ast.Name("_burst_meter", ast.Load()), "json_encode", ast.Load()), [node.func, *node.args], node.keywords), node)
        return node


def copies(meter):
    namespaces, insertions = {}, {}
    for name in ("recorder", "scout_probe"):
        original = importlib.import_module(name)
        path = Path(original.__file__)
        transform = Instrument()
        tree = ast.fix_missing_locations(transform.visit(ast.parse(path.read_text(encoding="utf-8"))))
        namespace = {**vars(original), "_burst_meter": meter}
        exec(compile(tree, "stage1-diagnostic-copy:" + path.name, "exec"), namespace)
        namespaces[name], insertions[name] = namespace, dict(transform.insertions)
    namespaces["scout_probe"]["Recorder"] = namespaces["recorder"]["Recorder"]
    namespaces["scout_probe"]["bounded"] = namespaces["recorder"]["bounded"]
    return namespaces, insertions


def main():
    import argparse
    import json
    import sys
    parser = argparse.ArgumentParser()
    parser.add_argument("--engine", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("runner", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    sys.path.insert(0, str(Path(args.engine).resolve() / "adapters/provenance-python"))
    meter = Meter()
    namespaces, insertions = copies(meter)
    if "sampling_decision" not in insertions["scout_probe"]:
        raise RuntimeError("measured engine has no sampling branch")
    status = None
    try:
        status = namespaces["scout_probe"]["main"](args.runner[1:] if args.runner[:1] == ["--"] else args.runner)
        return status
    finally:
        Path(args.out).write_text(json.dumps({"runner_exit": status, "insertions": insertions, "timings": meter.report(),
            "limits": "Instrumented diagnostic; bounded aggregate counters, no cost qualification. Wrapper timing excludes target-body time but omits the outer method guard. Phase inclusive/exclusive totals must not be summed together."}, indent=2), encoding="utf-8")


if __name__ == "__main__":
    raise SystemExit(main())
