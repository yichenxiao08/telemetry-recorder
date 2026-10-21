"""Diagnostic AST copies only: no edits to the probe, recorder or target."""
import ast
from collections import Counter, deque
from pathlib import Path
import threading
import time
import types

import recorder
import scout_probe


PHASES = ("entry_argument_access", "allowlist_privacy", "value_capture",
          "envelope_json", "span_lineage", "clock", "queue_put", "exit_path",
          "run_context_enter", "run_context_exit", "counter_bookkeeping")


class Scope:
    __slots__ = ("meter", "name", "frame")

    def __init__(self, meter, name):
        self.meter, self.name, self.frame = meter, name, None

    def __enter__(self):
        if self.meter.active:
            self.frame = [self.name, time.perf_counter_ns(), 0, 0]
            self.meter.stack.append(self.frame)
        return self

    def __exit__(self, *exc):
        if self.frame is not None:
            stopped = time.perf_counter_ns()
            name, started, children, child_count = self.meter.stack.pop()
            elapsed = stopped-started
            self.meter.raw[name] += elapsed-children
            self.meter.inclusive[name] += elapsed
            self.meter.counts[name] += 1
            self.meter.child_counts[name] += child_count
            if self.meter.stack:
                self.meter.stack[-1][2] += elapsed
                self.meter.stack[-1][3] += 1
        return False  # Preserve the exact exception object.


class Meter:
    def __init__(self, capacity=20000):
        self.thread = threading.get_ident()
        self.enabled = False
        self.stack = []
        self.samples = deque(maxlen=capacity)
        self.overflow = 0
        self.timer_internal_ns = 0
        self.timer_whole_ns = 0
        self.clock_source = time

    @property
    def active(self):
        return self.enabled and threading.get_ident() == self.thread

    def phase(self, name):
        return Scope(self, name)

    def clock(self, name, *args, **kwargs):
        with self.phase("clock"):
            return getattr(self.clock_source, name)(*args, **kwargs)

    def json_encode(self, function, *args, **kwargs):
        with self.phase("envelope_json"):
            return function(*args, **kwargs)

    def begin(self):
        if self.enabled:
            raise RuntimeError("nested measurement boundary")
        self.raw, self.inclusive, self.counts, self.child_counts = (Counter() for _ in range(4))
        self.started = time.perf_counter_ns()
        self.cpu_started = time.thread_time_ns()
        self.enabled = True

    def finish(self, label=None):
        stopped = time.perf_counter_ns()
        cpu_stopped = time.thread_time_ns()
        self.enabled = False
        if self.stack:
            raise RuntimeError("unbalanced timing scopes")
        body = self.inclusive.get("target_body", 0)
        whole = stopped-self.started-body
        counts = sum(v for k, v in self.counts.items() if k != "target_body")
        # Empty-scope costs are measured in this same process before calls.
        adjusted = {name: self.raw[name]-self.counts[name]*self.timer_internal_ns-
                    self.child_counts[name]*(self.timer_whole_ns-self.timer_internal_ns)
                    for name in PHASES}
        total = whole-counts*self.timer_whole_ns
        item = {"label": label, "started_ns": self.started, "stopped_ns": stopped,
                "raw_total_ns": whole, "adjusted_total_ns": total,
                "caller_thread_cpu_ns": cpu_stopped-self.cpu_started,
                "target_body_ns": body, "raw_exclusive_ns": dict(self.raw),
                "inclusive_ns": dict(self.inclusive), "scope_counts": dict(self.counts),
                "adjusted_exclusive_ns": adjusted,
                "negative_adjustments": sum(v < 0 for v in adjusted.values()),
                "residual_ns": total-sum(adjusted.values())}
        if len(self.samples) == self.samples.maxlen:
            self.overflow += 1
        self.samples.append(item)
        return item

    def calibrate(self, count=10000):
        # Sample the same bookkeeping path, excluding calibration from the
        # published call buffer. Retain all timer and empty-scope observations.
        internal, whole, pairs = [], [], []
        for _ in range(count):
            self.begin()
            start = time.perf_counter_ns()
            with self.phase("empty"):
                pass
            whole.append(time.perf_counter_ns()-start)
            internal.append(self.raw["empty"])
            self.enabled = False
            begin = time.perf_counter_ns()
            pairs.append(time.perf_counter_ns()-begin)
        from statistics import median
        self.timer_internal_ns = median(internal)
        self.timer_whole_ns = median(whole)
        return {"empty_scope_internal_ns": internal, "empty_scope_whole_ns": whole,
                "consecutive_timer_ns": pairs, "subtracted_internal_ns": self.timer_internal_ns,
                "subtracted_whole_ns": self.timer_whole_ns}


class Instrument(ast.NodeTransformer):
    def __init__(self):
        self.cls, self.function = "", ""
        self.inserted = Counter()

    def wrap(self, node, phase):
        self.inserted[phase] += 1
        scope = ast.Call(ast.Attribute(ast.Name("_meter", ast.Load()), "phase", ast.Load()),
                         [ast.Constant(phase)], [])
        return ast.copy_location(ast.With([ast.withitem(scope)], [node]), node)

    def visit_ClassDef(self, node):
        previous, self.cls = self.cls, node.name
        node = self.generic_visit(node)
        self.cls = previous
        return node

    def visit_FunctionDef(self, node):
        previous, self.function = self.function, node.name
        node = self.generic_visit(node)
        phase = {("Run", "__enter__"): "run_context_enter", ("Run", "__exit__"): "run_context_exit",
                 ("Recorder", "_enqueue"): "queue_put", ("Recorder", "_span"): "span_lineage",
                 ("Recorder", "_count"): "counter_bookkeeping"}.get((self.cls, node.name))
        if not self.cls and node.name == "bounded":
            phase = "value_capture"
        if phase:
            node.body = [self.wrap(ast.If(ast.Constant(True), node.body, []), phase)]
        self.function = previous
        return node

    def visit_Call(self, node):
        node = self.generic_visit(node)
        if self.function not in ("wrapped", "record_decision", "__enter__", "__exit__"):
            return node
        if isinstance(node.func, ast.Attribute) and isinstance(node.func.value, ast.Name):
            if node.func.value.id == "time" and node.func.attr in ("perf_counter_ns", "time_ns", "monotonic"):
                return ast.copy_location(ast.Call(ast.Attribute(ast.Name("_meter", ast.Load()), "clock", ast.Load()),
                    [ast.Constant(node.func.attr), *node.args], node.keywords), node)
            if node.func.value.id == "json" and node.func.attr == "dumps":
                return ast.copy_location(ast.Call(ast.Attribute(ast.Name("_meter", ast.Load()), "json_encode", ast.Load()),
                    [node.func, *node.args], node.keywords), node)
        return node

    def visit_Assign(self, node):
        text = ast.unparse(node)
        node = self.generic_visit(node)
        phase = None
        if self.function == "wrapped":
            name = text.split(" = ", 1)[0]
            if name == "result" and "original(" in text:
                phase = "target_body"
            elif name in ("begin", "run, manager, parent_token, otel_token", "entry_ns", "run", "sid", "parent",
                          "parent_token", "span_context", "otel_token", "fields", "bound", "item"):
                phase = "entry_argument_access"
            elif name in ("output", "body_start", "wall_start", "body_ns", "wall_end", "self.last"):
                phase = "exit_path"
        elif self.function == "record_decision":
            name = text.split(" = ", 1)[0]
            if name in ("envelope", "body"):
                phase = "envelope_json"
            elif name in ("meta", "events", "edge", "ts"):
                phase = "span_lineage"
            elif name in ("ref", "entry", "out"):
                phase = "value_capture"
        return self.wrap(node, phase) if phase else node

    def visit_If(self, node):
        text = ast.unparse(node.test)
        node = self.generic_visit(node)
        if self.function == "wrapped" and text in ("name in self.capture", "self.capture_return and error is None"):
            return self.wrap(node, "allowlist_privacy")
        if self.function == "record_decision":
            return self.wrap(node, "envelope_json" if "body.encode(" in text else "allowlist_privacy")
        return node

    def visit_For(self, node):
        target = ast.unparse(node.target)
        node = self.generic_visit(node)
        if self.function == "record_decision":
            return self.wrap(node, "span_lineage" if target == "(i, entry)" else "value_capture")
        return node

    def visit_Expr(self, node):
        text = ast.unparse(node)
        node = self.generic_visit(node)
        if self.function == "wrapped" and ("context.detach(" in text or ".parent.reset(" in text):
            return self.wrap(node, "exit_path")
        if self.function == "wrapped" and "bound.apply_defaults(" in text:
            return self.wrap(node, "entry_argument_access")
        if self.function == "record_decision" and "_count('envelope_bytes'" in text:
            return self.wrap(node, "envelope_json")
        return node


def copies(meter):
    """Compile private copies; no sys.modules replacement or source writes."""
    namespaces, insertions = {}, {}
    for module in (recorder, scout_probe):
        path = Path(module.__file__)
        transform = Instrument()
        tree = ast.fix_missing_locations(transform.visit(ast.parse(path.read_text(encoding="utf-8"))))
        namespace = {"__name__": "diagnostic_"+module.__name__, "__file__": str(path)}
        exec(compile(tree, "diagnostic-copy:"+path.name, "exec"), namespace)
        namespace["_meter"] = meter
        namespaces[module.__name__] = namespace
        insertions[module.__name__] = dict(transform.inserted)
    namespaces["scout_probe"]["bounded"] = namespaces["recorder"]["bounded"]
    namespaces["scout_probe"]["Recorder"] = namespaces["recorder"]["Recorder"]
    required = set(PHASES)-{"clock"}
    present = set().union(*(set(value) for value in insertions.values()))
    if not required <= present:
        raise RuntimeError("missing phase instrumentation: "+str(sorted(required-present)))
    return types.SimpleNamespace(Recorder=namespaces["recorder"]["Recorder"],
                                 Probe=namespaces["scout_probe"]["Probe"],
                                 Injector=namespaces["scout_probe"]["Injector"], insertions=insertions)
