"""External, config-driven synchronous Python call probes. Synthetic S0 only."""
from __future__ import annotations

import contextvars
import functools
import importlib.abc
import importlib.machinery
import inspect
import json
import runpy
import sys
import time
import types
from pathlib import Path

import yaml
from opentelemetry import context, trace
from opentelemetry.trace import NonRecordingSpan, SpanContext, TraceFlags

from recorder import Recorder, bounded


# Optional semantic calls share the process's single recorder, not a new exporter.
_optional = None


def record_decision(step_id, **fields):
    try:
        if _optional is None:
            return False
        return _optional.recorder.record_decision(
            step_id, _parent_id=_optional.parent.get(), **fields)
    except Exception:
        return False


def load_config(path):
    # No arbitrary tags or executable config. Capture is OFF unless allowlisted.
    text = Path(path).read_text(encoding="utf-8-sig")
    if len(text.encode("utf-8")) > 32768:
        raise ValueError("config too large")
    cfg = yaml.safe_load(text)
    if type(cfg) is not dict or cfg.get("version") != 1 or cfg.get("synthetic_only") is not True:
        raise ValueError("only version 1 synthetic configs supported")
    if set(cfg) - {"version", "synthetic_only", "endpoint", "service", "probes", "auto_otel"}:
        raise ValueError("unknown config key")
    probes = cfg.get("probes", [])
    if type(probes) is not list or len(probes) > 32:
        raise ValueError("invalid probes")
    seen = set()
    for p in probes:
        if type(p) is not dict or set(p) - {"module", "function", "capture"}:
            raise ValueError("unknown probe key")
        for key in ("module", "function"):
            if type(p.get(key)) is not str or not all(v.isidentifier() for v in p[key].split(".")):
                raise ValueError("invalid target")
        identity = (p["module"], p["function"])
        if identity in seen:
            raise ValueError("duplicate probe")
        seen.add(identity)
        capture = p.get("capture", {})
        if type(capture) is not dict or set(capture) - {"args", "return"}:
            raise ValueError("unknown capture key")
        names = capture.get("args", [])
        if type(names) is not list or len(names) > 32 or any(type(v) is not str for v in names):
            raise ValueError("invalid field allowlist")
        if type(capture.get("return", False)) is not bool:
            raise ValueError("return capture must be boolean")
    return cfg


class Probe:
    def __init__(self, owner, function, spec):
        if not inspect.isfunction(function) or function.__module__ != spec["module"]:
            raise ValueError("only directly defined module functions supported")
        if inspect.isgeneratorfunction(function) or inspect.iscoroutinefunction(function) or inspect.isasyncgenfunction(function):
            raise ValueError("generator/async functions unsupported")
        if function.__qualname__ != spec["function"] or "__signature__" in function.__dict__:
            raise ValueError("custom signature/alias/nested definitions unsupported")
        self.signature = inspect.signature(function, eval_str=False)
        if len(self.signature.parameters) > 32 or any(
            p.kind in (p.VAR_POSITIONAL, p.VAR_KEYWORD) for p in self.signature.parameters.values()
        ):
            raise ValueError("variadic/oversized signatures unsupported")
        self.owner, self.original, self.spec = owner, function, spec
        self.capture = set(spec.get("capture", {}).get("args", []))
        if self.capture - set(self.signature.parameters):
            raise ValueError("allowlist names not in signature")
        self.capture_return = spec.get("capture", {}).get("return", False)
        self.last = {}

    def wrap(self):
        original, recorder = self.original, self.owner.recorder

        @functools.wraps(original)
        def wrapped(*args, **kwargs):
            begin = time.perf_counter_ns()
            run, manager, parent_token, otel_token = None, None, None, None
            entry_ns = exit_ns = 0
            try:
                run = recorder._active.get()
                if run is None:
                    enter = time.perf_counter_ns()
                    manager = recorder.run(self.spec["module"] + "." + original.__name__)
                    run = manager.__enter__()
                    entry_ns = time.perf_counter_ns() - enter
                sid = f"{run.next_id:016x}"
                run.next_id += 1
                parent = self.owner.parent.get() or run.span_id
                parent_token = self.owner.parent.set(sid)
                span_context = SpanContext(int(run.trace_id, 16), int(sid, 16), False, TraceFlags(1))
                otel_token = context.attach(trace.set_span_in_context(NonRecordingSpan(span_context)))
            except Exception:
                recorder._count("probe_setup_failed")
            fields = {}
            try:
                bound = self.signature.bind(*args, **kwargs)
                bound.apply_defaults()
                for name, value in bound.arguments.items():
                    item = {"ref": f"value:{run.trace_id}/{sid}.arg.{name}@1"}
                    if name in self.capture:
                        try:
                            item["value"] = bounded(value)
                            recorder._count("captured_args")
                        except Exception:
                            recorder._count("capture_invalid")
                    fields[name] = item
            except Exception:
                recorder._count("probe_bind_failed")
                fields = {"unavailable": {"ref": "value:unknown/unavailable@1"}}
            wall_start = time.time_ns()
            body_start = time.perf_counter_ns()
            error = None
            try:
                try:
                    result = original(*args, **kwargs)
                except BaseException as exc:
                    error = exc
                    raise
                return result
            finally:
                body_ns = time.perf_counter_ns() - body_start
                wall_end = time.time_ns()
                # Capture failures must not change the host's result or exception.
                try:
                    output = {"ref": f"value:{run.trace_id}/{sid}.return@1"}
                    if self.capture_return and error is None:
                        output["value"] = result
                    recorder.record_decision(self.spec["module"] + "." + original.__name__,
                        output=output, inputs=fields,
                        chosen={"option": "raised" if error is not None else "returned"},
                        alternatives=[], rule={"id": "unobserved"},
                        verdict="raised" if error is not None else "returned",
                        reason_code=type(error).__name__ if error is not None else "",
                        record_id=sid, at_ns=wall_start, end_ns=wall_end,
                        _span_id=sid, _parent_id=parent, _error=error is not None)
                except Exception:
                    recorder._count("probe_capture_failed")
                finally:
                    try:
                        if otel_token is not None:
                            context.detach(otel_token)
                        if parent_token is not None:
                            self.owner.parent.reset(parent_token)
                        if manager is not None:
                            ending = time.perf_counter_ns()
                            manager.__exit__(type(error) if error is not None else None, error, None)
                            exit_ns = time.perf_counter_ns() - ending
                    except Exception:
                        recorder._count("probe_cleanup_failed")
                self.last = {"body_ns": body_ns, "context_enter_ns": entry_ns,
                             "context_exit_ns": exit_ns,
                             "wrapper_ns": time.perf_counter_ns() - begin - body_ns}
        wrapped.__signalscout_probe__ = self
        return wrapped


class Loader(importlib.abc.Loader):
    def __init__(self, owner, original):
        self.owner, self.original = owner, original

    def create_module(self, spec):
        method = getattr(self.original, "create_module", None)
        return method(spec) if method else None

    def exec_module(self, module):
        self.original.exec_module(module)
        self.owner.attach(module)


class Injector(importlib.abc.MetaPathFinder):
    def __init__(self, recorder, specs):
        self.recorder, self.specs, self.results = recorder, specs, []
        self.parent = contextvars.ContextVar("scout_probe_parent", default=None)

    def find_spec(self, fullname, path=None, target=None):
        if fullname not in {p["module"] for p in self.specs}:
            return None
        if target is not None:
            for p in self.specs:
                if p["module"] == fullname:
                    self._save_result({"module": fullname, "function": p["function"],
                        "status": "unsupported", "reason": "module reload unsupported; new definitions unwrapped"})
            return None
        spec = importlib.machinery.PathFinder.find_spec(fullname, path, target)
        if spec and spec.loader and hasattr(spec.loader, "exec_module"):
            spec.loader = Loader(self, spec.loader)
        return spec

    def _save_result(self, result):
        # One current result per configured probe; reload cannot grow history.
        for i, previous in enumerate(self.results):
            if (previous["module"], previous["function"]) == (result["module"], result["function"]):
                self.results[i] = result
                return
        if len(self.results) < len(self.specs):
            self.results.append(result)

    def attach(self, module):
        for spec in self.specs:
            if spec["module"] != module.__name__:
                continue
            result = {"module": module.__name__, "function": spec["function"]}
            try:
                if "." in spec["function"]:
                    raise ValueError("class/descriptor/nested paths unsupported")
                if type(module) is not types.ModuleType:
                    raise ValueError("custom module objects unsupported")
                function = vars(module)[spec["function"]]
                if hasattr(function, "__signalscout_probe__"):
                    raise ValueError("already wrapped")
                wrapped = Probe(self, function, spec).wrap()
                setattr(module, spec["function"], wrapped)
                result.update(status="attached", capture=spec.get("capture", {}))
            except Exception as exc:
                result.update(status="unsupported", reason=type(exc).__name__ + ": " + str(exc))
            self._save_result(result)

    def install(self):
        sys.meta_path.insert(0, self)
        for name in {p["module"] for p in self.specs}:
            if name in sys.modules:
                self.attach(sys.modules[name])

    def uninstall(self):
        if self in sys.meta_path:
            sys.meta_path.remove(self)
        # Wrappers are intentionally process-scoped. No mutation of source files.

    def doctor(self):
        result = list(self.results)
        for p in self.specs:
            if not any(r["module"] == p["module"] and r["function"] == p["function"] for r in result):
                result.append({"module": p["module"], "function": p["function"],
                               "status": "failed", "reason": "module never imported or unsupported loader"})
        return {"privacy": "synthetic-only; per-field capture default OFF; NOT P0",
                "probes": result, "counters": self.recorder.snapshot()}


def standard_otel(recorder):
    """Installed standard HTTP requests tracing + logging context correlation."""
    result = {}
    try:
        from opentelemetry.sdk.trace import SpanProcessor, TracerProvider

        class QueueProcessor(SpanProcessor):
            def on_end(self, span):
                try:
                    run = type("IDs", (), {"trace_id": f"{span.context.trace_id:032x}"})()
                    item = recorder._span(run, f"{span.context.span_id:016x}",
                        f"{span.parent.span_id:016x}" if span.parent else "",
                        span.name, span.start_time, span.end_time,
                        {k:v for k,v in span.attributes.items() if k in ("http.method","http.status_code")}, [],
                        span.status.status_code.name == "ERROR")
                    recorder._enqueue(item)
                except Exception:
                    recorder._count("auto_otel_drop")

        if type(trace.get_tracer_provider()).__name__ != "ProxyTracerProvider":
            result["provider"] = "existing provider preserved; queue bridge not attached"
        else:
            provider = TracerProvider()
            provider.add_span_processor(QueueProcessor())
            trace.set_tracer_provider(provider)
            result["provider"] = "SDK -> same bounded OTLP queue"
        from opentelemetry.instrumentation.requests import RequestsInstrumentor
        RequestsInstrumentor().instrument()
        result["http"] = "standard requests instrumentation; other HTTP libraries unsupported"
    except Exception as exc:
        result["http"] = "unavailable: " + type(exc).__name__
    try:
        from opentelemetry.instrumentation.logging import LoggingInstrumentor
        LoggingInstrumentor().instrument(set_logging_format=False)
        result["logging"] = "standard context correlation; log body export OFF"
    except Exception as exc:
        result["logging"] = "unavailable: " + type(exc).__name__
    return result


def main():
    global _optional
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", required=True)
    parser.add_argument("--doctor", required=True)
    parser.add_argument("--path", default=".")
    parser.add_argument("entry", help="module:function; the target source remains unmodified")
    args = parser.parse_args()
    cfg = load_config(args.config)
    names = {n for p in cfg["probes"] for n in p.get("capture", {}).get("args", [])}
    if any(p.get("capture", {}).get("return", False) for p in cfg["probes"]):
        names.add("output")
    recorder = Recorder(cfg.get("endpoint", "http://127.0.0.1:4318"),
                        service=cfg.get("service", "synthetic.probe"), capture_fields=names)
    injector = Injector(recorder, cfg["probes"])
    _optional = injector
    # python -m names this module __main__; optional target imports must share it.
    sys.modules.setdefault("scout_probe", sys.modules[__name__])
    auto = standard_otel(recorder) if cfg.get("auto_otel", False) else {"status": "disabled"}
    sys.path.insert(0, str(Path(args.path).resolve()))
    injector.install()
    started = time.perf_counter()
    try:
        import importlib
        module, function = args.entry.split(":")
        getattr(importlib.import_module(module), function)()
    finally:
        _optional = None
        injector.uninstall()
        flush_start = time.perf_counter()
        drained = recorder.close(2)
        report = injector.doctor()
        report.update(auto_otel=auto, elapsed_seconds=time.perf_counter()-started,
                      flush_seconds=time.perf_counter()-flush_start, drained=drained)
        Path(args.doctor).write_text(json.dumps(report, indent=2)+"\n", encoding="utf-8")
        print(json.dumps(report), file=sys.stderr)


if __name__ == "__main__":
    main()
