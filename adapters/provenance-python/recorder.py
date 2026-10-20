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
                    self.end_ns or time.time_ns(), self.attributes, [], typ is not None))
        except Exception:
            self.recorder._count("drop_invalid")
        finally:
            if self.token is not None:
                try:
                    self.recorder._active.reset(self.token)
                except Exception:
                    self.recorder._count("drop_invalid")
        return False  # Never suppress or replace the host application's exception.


class Recorder:
    def __init__(self, endpoint="http://127.0.0.1:4318", *, service="synthetic.provenance",
                 capture_fields=(), capacity=256, timeout=.5, cooldown=1,
                 resource_attributes=None, _start_worker=True):
        if not 1 <= capacity <= 4096 or not 0 < timeout <= 5 or not 0 < cooldown <= 60:
            raise ValueError("invalid recorder bounds")
        self.resource_attributes = bounded({} if resource_attributes is None else resource_attributes)
        self.endpoint, self.service = endpoint.rstrip("/"), service
        self.capture_fields = frozenset(capture_fields)
        self.timeout, self.cooldown = timeout, cooldown
        self._q = queue.Queue(capacity)
        self._counts, self._lock = {}, threading.Lock()
        self._active = contextvars.ContextVar("signalscout_run", default=None)
        self._stop, self._closed = threading.Event(), False
        self._cooldown_until = 0.0
        # Only this daemon performs network I/O. It never holds the stats lock
        # while exporting. Queue waits occur only here or in explicit flush.
        self._worker = threading.Thread(target=self._work, name="signalscout-export", daemon=True)
        if _start_worker:
            self._worker.start()

    def _count(self, name, n=1):
        with self._lock:
            self._counts[name] = self._counts.get(name, 0) + n

    def snapshot(self):
        with self._lock:
            result = dict(self._counts)
        result.update(queue_size=self._q.qsize(), capacity=self._q.maxsize,
                      cooldown_active=time.monotonic() < self._cooldown_until)
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
        if time.monotonic() < self._cooldown_until:
            self._count("drop_cooldown")
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
            # Circuit fast path is reported separately from full serialization.
            if self._closed:
                self._count("attempted")
                self._count("drop_closed")
                return False
            if time.monotonic() < self._cooldown_until:
                self._count("attempted")
                self._count("drop_cooldown")
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
        request = urllib.request.Request(self.endpoint + "/v1/traces", self._encode(items),
                                         {"Content-Type": "application/x-protobuf"})
        self._count("network_requests")
        with urllib.request.urlopen(request, timeout=self.timeout) as response:
            if response.status != 200:
                raise OSError("OTLP export failed")

    def _work(self):
        while not self._stop.is_set() or not self._q.empty():
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
                if time.monotonic() < self._cooldown_until:
                    raise OSError("cooldown")
                self._send(batch)
                self._count("exported", len(batch))
            except Exception:
                self._cooldown_until = time.monotonic() + self.cooldown
                self._count("export_failed", len(batch))
            finally:
                for _ in batch:
                    self._q.task_done()

    def flush(self, timeout=2):
        deadline = time.monotonic() + max(0, min(timeout, 10))
        with self._q.all_tasks_done:
            while self._q.unfinished_tasks:
                left = deadline - time.monotonic()
                if left <= 0:
                    return False
                self._q.all_tasks_done.wait(left)
        return True

    def close(self, timeout=2):
        result = self.flush(timeout)
        self._closed = True
        self._stop.set()
        return result
