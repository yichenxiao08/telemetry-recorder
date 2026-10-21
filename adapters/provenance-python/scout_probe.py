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
        output = fields.get("output")
        run = _optional.recorder._active.get()
        if type(output) is dict and "ref" not in output and run and run.valid:
            fields["output"] = {"ref": f"value:{run.trace_id}/{run.next_id}.return@1", **output}
        accepted = _optional.recorder.record_decision(
            step_id, _parent_id=_optional.parent.get(), **fields)
        if accepted:
            _optional.recorder._count("probe_decisions_captured")
        return accepted
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
        if type(capture) is not dict or set(capture) - {"args", "return", "self", "cls"}:
            raise ValueError("unknown capture key")
        names = capture.get("args", [])
        if type(names) is not list or len(names) > 32 or any(type(v) is not str for v in names):
            raise ValueError("invalid field allowlist")
        if type(capture.get("return", False)) is not bool:
            raise ValueError("return capture must be boolean")
        for receiver in ("self", "cls"):
            fields = capture.get(receiver, [])
            if type(fields) is not list or len(fields) > 16 or any(type(n) is not str or not n.isidentifier() or n.startswith("__") for n in fields) or len(set(fields)) != len(fields):
                raise ValueError("receiver capture requires <=16 explicit attribute names")
    return cfg


class Probe:
    def __init__(self, owner, function, spec, declaring_class=None, receiver_kind=None):
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
        self.declaring_class, self.receiver_kind = declaring_class, receiver_kind
        self.receiver_parameter = next(iter(self.signature.parameters), None) if receiver_kind else None
        self.capture = set(spec.get("capture", {}).get("args", []))
        if self.capture - set(self.signature.parameters):
            raise ValueError("allowlist names not in signature")
        if self.receiver_parameter in self.capture:
            raise ValueError("whole self/cls capture forbidden; allowlist attributes instead")
        self.receiver_fields = spec.get("capture", {}).get(receiver_kind, []) if receiver_kind else []
        if len(self.signature.parameters) - bool(receiver_kind) + len(self.receiver_fields) > 32:
            raise ValueError("combined argument/receiver field budget exceeds 32")
        for name in ("self", "cls"):
            if spec.get("capture", {}).get(name) and name != receiver_kind:
                raise ValueError("receiver allowlist does not match callable kind")
        for name in self.receiver_fields:
            for cls in declaring_class.__mro__:
                value = vars(cls).get(name)
                if isinstance(value, (property, staticmethod, classmethod)) or inspect.getattr_static(type(value), "__get__", None) is not None:
                    raise ValueError("receiver property/descriptor capture unsupported: " + name)
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
                recorder._count("probe_calls:" + self.spec["module"] + "." + self.spec["function"])
                run = recorder._active.get()
                if run is None:
                    enter = time.perf_counter_ns()
                    manager = recorder.run(self.spec["module"] + "." + self.spec["function"])
                    run = manager.__enter__()
                    if run.valid:
                        recorder._count("probe_runs_created")
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
                    if name == self.receiver_parameter:
                        continue
                    item = {"ref": f"value:{run.trace_id}/{sid}.arg.{name}@1"}
                    if name in self.capture:
                        try:
                            item["value"] = bounded(value)
                            recorder._count("captured_args")
                        except Exception:
                            recorder._count("capture_invalid")
                    fields[name] = item
                if self.receiver_kind and self.receiver_fields:
                    receiver = bound.arguments[self.receiver_parameter]
                    if self.receiver_kind == "self":
                        if len(type(receiver).__mro__) > 64:
                            raise ValueError("receiver MRO exceeds capture budget")
                        if any("__dict__" in vars(cls) and type(vars(cls)["__dict__"]) is not types.GetSetDescriptorType for cls in type(receiver).__mro__):
                            raise ValueError("custom receiver dictionary unsupported")
                        values = object.__getattribute__(receiver, "__dict__")
                        if type(values) is not dict:
                            raise ValueError("custom receiver dictionary unsupported")
                    else:
                        if type(receiver) is not type or len(receiver.__mro__) > 64:
                            raise ValueError("custom/oversized receiver MRO unsupported")
                        values = {}
                        for attribute in self.receiver_fields:
                            for cls in receiver.__mro__:
                                if attribute in vars(cls):
                                    values[attribute] = vars(cls)[attribute]
                                    break
                    for attribute in self.receiver_fields:
                        name = self.receiver_kind + "." + attribute
                        item = {"ref": f"value:{run.trace_id}/{sid}.arg.{name}@1"}
                        if attribute in values:
                            try:
                                item["value"] = bounded(values[attribute])
                            except Exception:
                                recorder._count("capture_invalid")
                        else:
                            recorder._count("capture_missing")
                        fields[name] = item
            except Exception:
                recorder._count("probe_bind_failed")
                fields = {"unavailable": {"ref": "value:unknown/unavailable@1"}}
            body_start = time.perf_counter_ns()
            # One UTC anchor per run; elapsed monotonic time retains sub-tick
            # Windows call durations and nested ordering even on a coarse wall clock.
            wall_start = run.start_ns + body_start - run.monotonic_start if run and run.valid else time.time_ns()
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
                wall_end = wall_start + body_ns
                # Capture failures must not change the host's result or exception.
                try:
                    output = {"ref": f"value:{run.trace_id}/{sid}.return@1"}
                    if self.capture_return and error is None:
                        output["value"] = result
                    accepted = recorder.record_decision(self.spec["module"] + "." + self.spec["function"],
                        output=output, inputs=fields,
                        chosen={"option": "raised" if error is not None else "returned"},
                        alternatives=[], rule={"id": "unobserved"},
                        verdict="raised" if error is not None else "returned",
                        reason_code=type(error).__name__ if error is not None else "",
                        record_id=sid, at_ns=wall_start, end_ns=wall_end,
                        _span_id=sid, _parent_id=parent, _error=error is not None)
                    if accepted:
                        recorder._count("probe_decisions_captured")
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
        if self.declaring_class is not None:
            @functools.wraps(original)
            def method_wrapped(*args, **kwargs):
                token = None
                suppressed = False
                try:
                    receiver = (args[0] if args else kwargs.get(self.receiver_parameter)) if self.receiver_kind else None
                    stack = self.owner.methods.get()
                    for obj, name, cls in stack:
                        if obj is receiver and name == original.__name__ and cls is not self.declaring_class and issubclass(cls, self.declaring_class):
                            recorder._count("probe_calls:" + self.spec["module"] + "." + self.spec["function"])
                            recorder._count("probe_super_suppressed")
                            suppressed = True
                            break
                    if len(stack) >= 64:
                        recorder._count("drop_method_depth")
                        suppressed = True
                    if not suppressed:
                        token = self.owner.methods.set(stack + ((receiver, original.__name__, self.declaring_class),))
                except Exception:
                    recorder._count("probe_method_setup_failed")
                if suppressed:
                    return original(*args, **kwargs)
                try:
                    return wrapped(*args, **kwargs)
                finally:
                    if token is not None:
                        try:
                            self.owner.methods.reset(token)
                        except Exception:
                            recorder._count("probe_cleanup_failed")
            method_wrapped.__signalscout_probe__ = self
            return method_wrapped
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
        self.methods = contextvars.ContextVar("scout_probe_methods", default=())

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
                if type(module) is not types.ModuleType:
                    raise ValueError("custom module objects unsupported")
                parts = spec["function"].split(".")
                container, name, cls, receiver, descriptor = module, parts[-1], None, None, None
                if len(parts) > 2:
                    raise ValueError("nested classes/paths unsupported")
                if len(parts) == 2:
                    cls = vars(module).get(parts[0])
                    if type(cls) is not type or cls.__module__ != module.__name__ or cls.__qualname__ != parts[0]:
                        raise ValueError("only directly defined top-level classes with plain metaclasses supported")
                    if len(cls.__mro__) > 64:
                        raise ValueError("class MRO exceeds 64-entry budget")
                    if name.startswith("__") or any(any(field in vars(base) for field in ("__getattribute__", "__getattr__", "__setattr__", "__slots__", "__new__", "__call__", "__get__", "__del__")) for base in cls.__mro__ if base is not object):
                        raise ValueError("dunder/custom-access/slots classes unsupported")
                    if name not in vars(cls):
                        raise ValueError("missing/inherited method; configure its declaring base class")
                    container = cls
                function = vars(container)[name]
                if isinstance(function, property):
                    raise ValueError("property probes unsupported")
                if type(function) is classmethod:
                    descriptor, receiver, function = classmethod, "cls", function.__func__
                elif type(function) is staticmethod:
                    descriptor, function = staticmethod, function.__func__
                elif cls is not None:
                    receiver = "self"
                if not inspect.isfunction(function):
                    raise ValueError("custom descriptors/native callables unsupported")
                if hasattr(function, "__signalscout_probe__"):
                    raise ValueError("already wrapped")
                wrapped = Probe(self, function, spec, cls, receiver).wrap()
                setattr(container, name, descriptor(wrapped) if descriptor else wrapped)
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
        counters = self.recorder.snapshot()
        for item in result:
            item["calls"] = counters.get("probe_calls:" + item["module"] + "." + item["function"], 0)
        return {"privacy": "synthetic-only; per-field capture default OFF; NOT P0",
                "probes": result, "counters": counters}


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


def summary(report):
    print("Signal Scout probe summary (synthetic only; capture default OFF)", file=sys.stderr)
    print(f"configured probes: {len(report['probes'])}; attached: {sum(p['status']=='attached' for p in report['probes'])}", file=sys.stderr)
    for probe in report["probes"]:
        reason = probe.get("reason", "")
        if probe["status"] == "attached" and probe["calls"] == 0:
            reason = "not invoked" if not report["preflight"] else "dry import; target not executed"
        print(f"  {probe['module']}:{probe['function']}: {probe['status']}; calls={probe['calls']}; {reason}", file=sys.stderr)
    print(f"records captured: {report['records_captured']}; run contexts created: {report['runs_created']}; acknowledged spans: {report['counters'].get('acknowledged', 0)}; flush complete: {bool(report['drained'])}", file=sys.stderr)
    if not report["preflight"] and (report["records_captured"] == 0 or report["runs_created"] == 0):
        print("ZERO CAPTURE: configured target produced no probed decisions/run contexts", file=sys.stderr)


def main(argv=None):
    global _optional
    import argparse
    parser = argparse.ArgumentParser(allow_abbrev=False)
    parser.add_argument("--config", required=True)
    parser.add_argument("--doctor", required=True)
    parser.add_argument("--path", default=".")
    parser.add_argument("--preflight", action="store_true", help="Dry import/attachment and isolated argv report; never call the entry point")
    parser.add_argument("entry", help="module:function; the target source remains unmodified")
    arguments = list(sys.argv[1:] if argv is None else argv)
    target_args = []
    if "--" in arguments:
        split = arguments.index("--")
        arguments, target_args = arguments[:split], arguments[split+1:]
    args = parser.parse_args(arguments)
    cfg = load_config(args.config)
    names = {n for p in cfg["probes"] for n in p.get("capture", {}).get("args", [])}
    names.update(receiver + "." + field for p in cfg["probes"] for receiver in ("self", "cls") for field in p.get("capture", {}).get(receiver, []))
    if any(p.get("capture", {}).get("return", False) for p in cfg["probes"]):
        names.add("output")
    recorder = Recorder(cfg.get("endpoint", "http://127.0.0.1:4318"),
                        service=cfg.get("service", "synthetic.probe"), capture_fields=names,
                        _start_worker=not args.preflight)
    injector = Injector(recorder, cfg["probes"])
    previous_optional, _optional = _optional, injector
    # python -m names this module __main__; optional target imports must share it.
    sys.modules.setdefault("scout_probe", sys.modules[__name__])
    auto = standard_otel(recorder) if cfg.get("auto_otel", False) and not args.preflight else {"status": "disabled"}
    previous_path, previous_argv = list(sys.path), sys.argv
    sys.path.insert(0, str(Path(args.path).resolve()))
    injector.install()
    started = time.perf_counter()
    target_argv = []
    target_ran = False
    try:
        import importlib
        module, function = args.entry.split(":")
        if not function.isidentifier():
            raise ValueError("entry point must be a top-level module:function")
        spec = importlib.util.find_spec(module)
        if spec is None:
            raise ValueError("target module not found")
        origin = spec.origin
        target_argv = [str(Path(origin).resolve()) if origin and origin not in ("built-in", "frozen") else module, *target_args]
        sys.argv = target_argv
        loaded = importlib.import_module(module)
        entry = vars(loaded).get(function)
        if not inspect.isfunction(entry) or inspect.isgeneratorfunction(entry) or inspect.iscoroutinefunction(entry) or inspect.isasyncgenfunction(entry):
            raise ValueError("entry point must be a plain synchronous function")
        if not args.preflight:
            target_ran = True
            try:
                entry()
            except SystemExit as exc:
                # Normal target completion still passes through capture checks.
                # Nonzero target exits/errors retain their original status.
                if exc.code is not None and exc.code != 0:
                    raise
    finally:
        _optional = previous_optional
        injector.uninstall()
        sys.argv = previous_argv
        sys.path[:] = previous_path
        flush_start = time.perf_counter()
        drained = recorder.close(2)
        report = injector.doctor()
        report.update(auto_otel=auto, elapsed_seconds=time.perf_counter()-started,
                      flush_seconds=time.perf_counter()-flush_start, drained=drained,
                      preflight=args.preflight, target_argv=target_argv, target_ran=target_ran,
                      records_captured=report["counters"].get("probe_decisions_captured",0),
                      runs_created=report["counters"].get("probe_runs_created",0))
        Path(args.doctor).write_text(json.dumps(report, indent=2)+"\n", encoding="utf-8")
        summary(report)
        print(json.dumps(report), file=sys.stderr)
    if not args.preflight and (report["records_captured"] == 0 or report["runs_created"] == 0):
        return 4
    if any(p["status"] != "attached" or not args.preflight and p["calls"] == 0 for p in report["probes"]):
        return 3
    if not drained or any(v for k,v in report["counters"].items() if k.startswith("drop") or k in ("failed", "unknown")):
        return 5
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
