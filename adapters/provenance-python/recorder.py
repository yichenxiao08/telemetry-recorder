"""Synthetic S0 spike only. Not a production privacy or instrumentation library."""
from __future__ import annotations

import contextvars
import json
import math
import queue
import secrets
import threading
import time
import urllib.request

from export_diagnostics import Diagnostics, observe

from opentelemetry.proto.collector.trace.v1.trace_service_pb2 import ExportTraceServiceRequest
from opentelemetry.proto.common.v1.common_pb2 import AnyValue


def bounded(value, depth=0, budget=None):
    """Only builtins: never invoke a caller's serializer, repr or property."""
    if budget is None:
        budget = [256]
    budget[0] -= 1
    if depth > 6 or budget[0] < 0:
        raise ValueError("too deep")
    if value is None or type(value) is bool:
        return value
    if type(value) is int and abs(value) <= 2**53:
        return value
    if type(value) is float and math.isfinite(value):
        return value
    if type(value) is str and len(value) <= 256:
        return value
    if type(value) is list and len(value) <= 32:
        return [bounded(v, depth + 1, budget) for v in value]
    if type(value) is dict and len(value) <= 32:
        if any(type(k) is not str or len(k) > 128 for k in value):
            raise ValueError("invalid key")
        return {k: bounded(v, depth + 1, budget) for k, v in value.items()}
    raise ValueError("unsupported or oversized value")


def attrs(values):
    out = []
    for key, value in values.items():
        a = AnyValue()
        if type(value) is bool:
            a.bool_value = value
        elif type(value) is int:
            a.int_value = value
        elif type(value) is float:
            a.double_value = value
        else:
            a.string_value = value
        out.append({"key": key, "value": a})
    return out


class Run:
    def __init__(self, recorder, name, run_id, start_ns, end_ns, attributes):
        self.recorder, self.name, self.trace_id = recorder, name, run_id
        self.start_ns, self.end_ns, self.attributes = start_ns, end_ns, attributes
        self.span_id, self.next_id = "0000000000000001", 2
        self.valid, self.token = False, None

    def __enter__(self):
        try:
            self.name = bounded(self.name)
            if self.trace_id is None:
                self.trace_id = secrets.token_hex(16)
            if type(self.trace_id) is not str or len(self.trace_id) != 32 or int(self.trace_id, 16) == 0:
                raise ValueError("invalid run ID")
            self.attributes = bounded(self.attributes)
            for stamp in (self.start_ns, self.end_ns):
                if stamp is not None and (type(stamp) is not int or not 0 < stamp < 2**63):
                    raise ValueError("invalid timestamp")
            self.start_ns = self.start_ns or time.time_ns()
            self.monotonic_start = time.perf_counter_ns()
            self.valid = True
            self.token = self.recorder._active.set(self)
        except Exception:
            self.recorder._count("drop_invalid")
        return self

    def __exit__(self, typ, value, traceback):
        try:
            if self.valid:
                self.recorder._enqueue(self.recorder._span(
                    self, self.span_id, "", self.name, self.start_ns,
                    self.end_ns or self.start_ns + time.perf_counter_ns() - self.monotonic_start,
                    self.attributes, [], typ is not None))
        except Exception:
            self.recorder._count("drop_invalid")
        finally:
            if self.token is not None:
                try:
                    self.recorder._active.reset(self.token)
                except Exception:
                    self.recorder._count("drop_invalid")
        return False  # Never suppress or replace the host application's exception.


class ObservedQueue(queue.Queue):
    def __init__(self, capacity):
        super().__init__(capacity)
        self.high_water = 0

    def _put(self, item):
        # Queue holds its existing mutex here: no new host-side wait or I/O.
        super()._put(item)
        self.high_water = max(self.high_water, self._qsize())


class FlushReport(dict):
    """JSON-friendly accounting; truth means complete delivery, not just drain."""
    def __bool__(self):
        return self["complete"]


class Recorder:
    def __init__(self, endpoint="http://127.0.0.1:4318", *, service="synthetic.provenance",
                 capture_fields=(), capacity=256, timeout=.5, cooldown=1,
                 resource_attributes=None, _start_worker=True, retry_limit=1):
        if not 1 <= capacity <= 4096 or not 0 < timeout <= 5 or not 0 < cooldown <= 60:
            raise ValueError("invalid recorder bounds")
        self.resource_attributes = bounded({} if resource_attributes is None else resource_attributes)
        self.endpoint, self.service = endpoint.rstrip("/"), service
        self.capture_fields = frozenset(capture_fields)
        self.timeout, self.cooldown = timeout, cooldown
        if retry_limit not in (0, 1):
            raise ValueError("at most one retry")
        self.retry_limit = retry_limit
        self.attempt_budget, self.retry_backoff = 1.25, .05
        self._q = ObservedQueue(capacity)
        self.diagnostics = Diagnostics()
        self._counts, self._lock = {}, threading.Lock()
        self._active = contextvars.ContextVar("signalscout_run", default=None)
        self._stop, self._closed = threading.Event(), False
        self._cooldown_until = 0.0
        self._network_jobs = queue.Queue(1)
        self._network_stop = threading.Event()
        self._network_thread = None
        self._flight = None
        # One delivery worker and at most one daemon I/O worker. An I/O call
        # stuck in DNS/headers is quarantined, never replaced by more threads.
        # Neither holds the statistics lock during I/O or worker-only waits.
        self._worker = threading.Thread(target=self._work, name="signalscout-export", daemon=True)
        if _start_worker:
            self._worker.start()

    def _count(self, name, n=1):
        with self._lock:
            self._counts[name] = self._counts.get(name, 0) + n

    def snapshot(self):
        with self._lock:
            result = dict(self._counts)
            result["acknowledged"] = result.get("exported", 0)
            result["failed"] = result.get("failed", 0)
            result["dropped"] = sum(result.get(k, 0) for k in ("drop_full", "drop_closed", "drop_cooldown"))
            result["unknown"] = result.get("attempted", 0)-result["acknowledged"]-result["failed"]-result["dropped"]
        result.update(queue_size=self._q.qsize(), capacity=self._q.maxsize,
                      cooldown_active=time.monotonic() < self._cooldown_until)
        details = self.diagnostics.snapshot()
        details["counters_spans_except_flush_timeout"].update(
            cooldown_drop=result.get("drop_cooldown", 0),
            queue_full_drop=result.get("drop_full", 0),
            shutdown_drop=result.get("drop_closed", 0), flush_drop=0)
        result.update(queue_high_water=self._q.high_water, diagnostics=details)
        result.update(network_operation_slots=1, retry_limit=self.retry_limit,
                      attempt_budget_seconds=self.attempt_budget)
        return result

    def run(self, name, *, run_id=None, start_ns=None, end_ns=None, attributes=None):
        return Run(self, name, run_id, start_ns, end_ns, {} if attributes is None else attributes)

    def _span(self, run, span_id, parent, name, start, end, attributes, events, error=False):
        return {"trace": run.trace_id, "span": span_id, "parent": parent,
                "name": name, "start": start, "end": end, "attrs": attributes,
                "events": events, "error": error}

    def _enqueue(self, item):
        self._count("attempted")
        if self._closed:
            self._count("drop_closed")
            return False
        try:
            self._q.put_nowait(item)
        except queue.Full:
            self._count("drop_full")
            return False
        self._count("accepted")
        return True

    def record_decision(self, step_id, *, output, inputs, chosen, alternatives,
                        rule, config_refs=(), verdict="selected", reason_code="",
                        record_id=None, at_ns=None, end_ns=None, _span_id=None, _parent_id=None,
                        _error=False):
        try:
            # Closing still rejects immediately. Cooldown is worker-only:
            # existing and newly admitted queue items must not be discarded.
            if self._closed:
                self._count("attempted")
                self._count("drop_closed")
                return False
            run = self._active.get()
            if run is None or not run.valid or type(inputs) is not dict or len(inputs) > 32:
                raise ValueError("invalid run or inputs")
            if type(config_refs) not in (list, tuple) or len(config_refs) > 32:
                raise ValueError("invalid config refs")
            for stamp in (at_ns, end_ns):
                if stamp is not None and (type(stamp) is not int or not 0 < stamp < 2**63):
                    raise ValueError("invalid timestamp")
            if record_id is not None and (type(record_id) is not str or len(record_id) > 128):
                raise ValueError("invalid record ID")
            sid = _span_id or f"{run.next_id:016x}"
            run.next_id += 1
            rid = record_id or sid
            if type(step_id) is not str or not step_id or type(verdict) is not str or type(reason_code) is not str:
                raise ValueError("invalid selector")
            records = []
            for name, item in inputs.items():
                name = bounded(name)
                if type(name) is not str:
                    raise ValueError("invalid field name")
                if type(item) is not dict:
                    raise ValueError("invalid input")
                ref = bounded(item.get("ref", f"value:{run.trace_id}/{name}@1"))
                entry = {"name": name, "ref": ref, "capture": "off"}
                if name in self.capture_fields and "value" in item:
                    entry.update(inline=bounded(item["value"]), capture="allowed")
                records.append(entry)
            out = {"ref": bounded(output["ref"]), "capture": "off"}
            if "output" in self.capture_fields and "value" in output:
                out.update(inline=bounded(output["value"]), capture="allowed")
            envelope = {"version": 1, "id": f"decision:{run.trace_id}/{rid}",
                        "run_id": run.trace_id, "profile": {"id": "synthetic", "version": "1"},
                        "step": {"id": bounded(step_id), "version": "1",
                                 "span_ref": f"span:{run.trace_id}/{sid}"},
                        "output": out, "inputs": records, "chosen": bounded(chosen),
                        "alternatives": bounded(alternatives), "rule": bounded(rule),
                        "config_refs": bounded(list(config_refs)),
                        "verdict": bounded(verdict), "reason_code": bounded(reason_code),
                        "complete": all(v["capture"] == "allowed" for v in records)}
            body = json.dumps(envelope, separators=(",", ":"), allow_nan=False)
            if len(body.encode("utf-8")) > 8192 or len(alternatives) > 8:
                raise ValueError("oversized envelope")
            if type(chosen.get("option")) is not str:
                raise ValueError("invalid chosen option")
            meta = {"signalscout.record.version": 1, "signalscout.record.kind": "decision",
                    "signalscout.record.id": envelope["id"], "signalscout.record.json": body,
                    "signalscout.step.id": step_id, "signalscout.output.ref": out["ref"],
                    "signalscout.chosen.option": chosen["option"],
                    "signalscout.verdict": verdict, "signalscout.reason.code": reason_code,
                    "signalscout.synthetic": True}
            ts = at_ns or time.time_ns()
            events = [("signalscout.decision", ts, meta)]
            for i, entry in enumerate(records):
                edge = {"version": 1, "id": f"lineage:{run.trace_id}/{rid}-{i}",
                        "run_id": run.trace_id, "input_ref": entry["ref"], "output_ref": out["ref"],
                        "step_id": step_id, "decision_ref": envelope["id"], "relation": "derived_from"}
                events.append(("signalscout.lineage", ts,
                               {"signalscout.record.version": 1, "signalscout.record.kind": "lineage",
                                "signalscout.record.id": edge["id"],
                                "signalscout.record.json": json.dumps(edge, separators=(",", ":")),
                                "signalscout.synthetic": True}))
            # Bounded strings, lists and at most 32 inputs; no external object
            # survives this snapshot to be mutated while the worker exports.
            self._count("serialized")
            self._count("envelope_bytes", len(body.encode("utf-8")))
            return self._enqueue(self._span(run, sid, _parent_id or run.span_id, step_id, ts, end_ns or ts + 1, {}, events, _error))
        except Exception:
            self._count("drop_invalid")
            return False

    def _encode(self, items):
        req = ExportTraceServiceRequest()
        resource = req.resource_spans.add()
        resource.resource.attributes.add(key="service.name", value=AnyValue(string_value=self.service))
        resource.resource.attributes.add(key="signalscout.synthetic", value=AnyValue(bool_value=True))
        for a in attrs(self.resource_attributes):
            resource.resource.attributes.add(**a)
        scope = resource.scope_spans.add()
        scope.scope.name = "signalscout.synthetic.spike"
        for item in items:
            span = scope.spans.add(trace_id=bytes.fromhex(item["trace"]),
                span_id=bytes.fromhex(item["span"]), parent_span_id=bytes.fromhex(item["parent"]),
                name=item["name"], start_time_unix_nano=item["start"],
                end_time_unix_nano=item["end"])
            span.status.code = 2 if item["error"] else 1
            for a in attrs(item["attrs"]):
                span.attributes.add(**a)
            for name, ts, values in item["events"]:
                event = span.events.add(name=name, time_unix_nano=ts)
                for a in attrs(values):
                    event.attributes.add(**a)
        return req.SerializeToString()

    def _send(self, items):
        self.diagnostics.phase = "encode"
        request = urllib.request.Request(self.endpoint + "/v1/traces", self._encode(items),
                                         {"Content-Type": "application/x-protobuf"})
        self._count("network_requests")
        self.diagnostics.phase = "connect"
        with observe(self.diagnostics):
            with urllib.request.urlopen(request, timeout=self.timeout) as response:
                self.diagnostics.phase = "response"
                if response.status != 200:
                    raise OSError("OTLP export failed")

    def _network_loop(self):
        while not self._network_stop.is_set() or not self._network_jobs.empty():
            try:
                operation = self._network_jobs.get(timeout=.05)
            except queue.Empty:
                continue
            try:
                self._send(operation["batch"])
            except BaseException as error:
                operation["error"] = error
                operation["phase"] = self.diagnostics.phase
            finally:
                operation["done"].set()
                self._network_jobs.task_done()

    def _settle(self, size, state):
        with self._lock:
            key = "exported" if state == "acknowledged" else "export_failed"
            self._counts[key] = self._counts.get(key, 0)+size
            if state != "acknowledged":
                key = "failed" if state == "failed" else "unknown_terminal"
                self._counts[key] = self._counts.get(key, 0)+size

    def _deliver(self, batch):
        deadline = time.monotonic()+self.attempt_budget
        ambiguous = False
        for attempt in range(self.retry_limit+1):
            if attempt:
                self._network_stop.wait(min(self.retry_backoff, max(0, deadline-time.monotonic())))
                if time.monotonic() >= deadline:
                    break
                self._count("retry_requests")
            operation = {"batch": batch, "done": threading.Event(), "error": None, "phase": "encode"}
            self._flight = operation
            started_ns, clock_ns = time.time_ns(), time.perf_counter_ns()
            self._network_jobs.put_nowait(operation)  # Only this worker submits; no concurrent I/O.
            completed = operation["done"].wait(max(0, deadline-time.monotonic()))
            if not completed:
                self._count("deadline_expired_spans", len(batch))
                self.diagnostics.failure(TimeoutError(), len(batch), started_ns)
                self.diagnostics.batch(len(batch), started_ns, time.perf_counter_ns()-clock_ns,
                                       "deadline_unknown", self._q.qsize())
                self._settle(len(batch), "unknown")
                self._cooldown_until = time.monotonic()+self.cooldown
                return
            error = operation["error"]
            self._flight = None
            if error is None:
                self.diagnostics.batch(len(batch), started_ns, time.perf_counter_ns()-clock_ns,
                                       "exported", self._q.qsize())
                self._settle(len(batch), "acknowledged")
                return
            self.diagnostics.phase = operation["phase"]
            outcome = self.diagnostics.failure(error, len(batch), started_ns)
            self.diagnostics.batch(len(batch), started_ns, time.perf_counter_ns()-clock_ns,
                                   outcome, self._q.qsize())
            status = getattr(error, "code", None)
            terminal = operation["phase"] == "encode" or (
                isinstance(status, int) and 400 <= status < 500 and status not in (408, 429))
            known = operation["phase"] in ("encode", "connect") or terminal
            ambiguous |= not known
            if terminal or attempt == self.retry_limit:
                self._settle(len(batch), "unknown" if ambiguous else "failed")
                self._cooldown_until = time.monotonic()+self.cooldown
                return
        self._settle(len(batch), "unknown" if ambiguous else "failed")
        self._cooldown_until = time.monotonic()+self.cooldown

    def _work(self):
        self._network_thread = threading.Thread(target=self._network_loop, name="signalscout-network", daemon=True)
        self._network_thread.start()
        try:
            self._drain()
        finally:
            self._network_stop.set()

    def _drain(self):
        while not self._stop.is_set() or not self._q.empty():
            # Never dequeue/discard during cooldown or while a timed-out I/O
            # operation still owns the single slot. A late response is not
            # retroactively promoted to delivery after the logical deadline.
            if self._flight is not None:
                if not self._flight["done"].wait(.05):
                    continue
                self._flight = None
            remaining = self._cooldown_until-time.monotonic()
            if remaining > 0:
                self._network_stop.wait(min(.05, remaining))
                continue
            try:
                batch = [self._q.get(timeout=.05)]
            except queue.Empty:
                continue
            while len(batch) < 32:
                try:
                    batch.append(self._q.get_nowait())
                except queue.Empty:
                    break
            try:
                self._deliver(batch)
            finally:
                for _ in batch:
                    self._q.task_done()

    def flush(self, timeout=2):
        deadline = time.monotonic() + max(0, min(timeout, 10))
        with self._q.all_tasks_done:
            while self._q.unfinished_tasks:
                left = deadline - time.monotonic()
                if left <= 0:
                    self.diagnostics.count("flush_timeout")
                    break
                self._q.all_tasks_done.wait(left)
            drained = self._q.unfinished_tasks == 0
        counts = self.snapshot()
        result = FlushReport({key: counts.get(key, 0) for key in
                              ("attempted", "acknowledged", "failed", "dropped", "unknown")})
        result.update(drained=drained, complete=drained and not any(
            result[k] for k in ("failed", "dropped", "unknown")))
        return result

    def close(self, timeout=2):
        result = self.flush(timeout)
        self._closed = True
        self._stop.set()
        return result
