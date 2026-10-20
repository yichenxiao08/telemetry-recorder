"""Injected/toy failures only; no reserved evaluation or generator calls."""
import http.client
import json
import socket
import time
import unittest
import urllib.error
from unittest.mock import patch

from export_diagnostics import Diagnostics, observe
from recorder import Recorder


ITEM = {"trace": "1"*32, "span": "2"*16, "parent": "", "name": "synthetic",
        "start": 1, "end": 2, "attrs": {}, "events": [], "error": False}


class ExportDiagnostics(unittest.TestCase):
    def make(self, **kwargs):
        recorder = Recorder(_start_worker=False, **kwargs)
        self.addCleanup(lambda: recorder.close(0))
        return recorder

    def failed(self, phase, error):
        r = self.make()
        def send(items):
            r.diagnostics.phase = phase
            raise error
        r._enqueue(ITEM)
        r._stop.set()
        with patch.object(r, "_send", side_effect=send):
            r._work()
        self.assertEqual(r.snapshot()["export_failed"], 1)
        return r.snapshot()["diagnostics"]

    def test_failure_categories_and_safe_samples(self):
        cases = [
            ("encode", ValueError("SYNTHETIC_PAYLOAD_SENTINEL"), "encode_failure"),
            ("wait-for-ack", TimeoutError("SYNTHETIC_PAYLOAD_SENTINEL"), "timeout"),
            ("connect", urllib.error.URLError(ConnectionRefusedError(10061, "private")), "connection_error"),
            ("response", urllib.error.HTTPError("http://private", 503, "private", {}, None), "http_non_2xx"),
            ("send", RuntimeError("SYNTHETIC_PAYLOAD_SENTINEL"), "other_failure"),
        ]
        for phase, error, kind in cases:
            with self.subTest(kind=kind):
                result = self.failed(phase, error)
                self.assertEqual(result["counters_spans_except_flush_timeout"][kind], 1)
                sample = result["last_failures"][0]
                self.assertEqual(sample["phase"], phase)
                self.assertEqual(sample["exception_class"], type(error).__name__)
                self.assertGreater(sample["attempt_unix_ns"], 0)
                self.assertLessEqual(len(sample["message"]), 160)
                self.assertNotIn("private", json.dumps(result))
                self.assertNotIn("SYNTHETIC_PAYLOAD_SENTINEL", json.dumps(result))
                if kind == "http_non_2xx":
                    self.assertEqual(sample["http_status"], 503)
                    self.assertEqual(result["counters_spans_except_flush_timeout"]["http_status_503"], 1)

    def test_real_stdlib_phase_hooks_preserve_exception_identity(self):
        d = Diagnostics()
        error = ConnectionRefusedError("synthetic")
        connection = http.client.HTTPConnection("127.0.0.1")
        with patch.object(connection, "_create_connection", side_effect=error), observe(d):
            try:
                connection.request("POST", "/", b"synthetic")
            except ConnectionRefusedError as actual:
                self.assertIs(actual, error)
            else:
                self.fail("exception altered")
        self.assertEqual(d.phase, "connect")
        connection = http.client.HTTPConnection("127.0.0.1")
        connection._HTTPConnection__state = http.client._CS_REQ_SENT
        connection.sock = socket.socket()
        self.addCleanup(connection.sock.close)
        with patch.object(connection, "response_class", side_effect=TimeoutError("synthetic")), observe(d):
            with self.assertRaises(TimeoutError):
                connection.getresponse()
        self.assertEqual(d.phase, "wait-for-ack")

    def test_bounded_counts_samples_and_batches(self):
        d = Diagnostics()
        for i in range(1000):
            d.failure(type("Injected"+str(i), (Exception,), {})(), 1, i)
            d.batch(1+i%32, i, 2, "test", 0)
        result = d.snapshot()
        self.assertEqual(len(result["last_failures"]), 64)
        self.assertEqual(len(result["last_worker_batches"]), 256)
        self.assertLessEqual(len(result["failures_by_class_spans"]), 33)
        self.assertEqual(len(result["worker_batch_sizes"]), 32)
        self.assertEqual(sum(result["failures_by_class_spans"].values()), 1000)

    def test_drop_flush_shutdown_and_high_water_counters(self):
        r = self.make(capacity=1)
        self.assertTrue(r._enqueue(ITEM))
        self.assertFalse(r._enqueue(ITEM))
        r._cooldown_until = time.monotonic()+1
        self.assertFalse(r._enqueue(ITEM))
        self.assertFalse(r.flush(.001))
        r.close(0)
        self.assertFalse(r._enqueue(ITEM))
        s = r.snapshot()
        c = s["diagnostics"]["counters_spans_except_flush_timeout"]
        self.assertEqual(s["queue_high_water"], 1)
        self.assertEqual(c["queue_full_drop"], s["drop_full"])
        self.assertEqual(c["cooldown_drop"], s["drop_cooldown"])
        self.assertEqual(c["shutdown_drop"], s["drop_closed"])
        self.assertGreater(c["flush_timeout"], 0)
        self.assertEqual(c["flush_drop"], 0)  # Timeout does not discard pending data.
        self.assertEqual(s["queue_size"], 1)

    def test_cooldown_suppressed_batch_keeps_legacy_meaning(self):
        r = self.make()
        r._enqueue(ITEM)
        r._cooldown_until = time.monotonic()+1
        r._stop.set()
        with patch.object(r, "_send", side_effect=AssertionError("must not send")):
            r._work()
        s = r.snapshot()
        self.assertEqual(s["export_failed"], 1)
        self.assertEqual(s["diagnostics"]["counters_spans_except_flush_timeout"]["cooldown_suppressed"], 1)

    def test_diagnostics_do_not_change_encoded_bytes(self):
        r = self.make()
        before = r._encode([ITEM])
        r.diagnostics.failure(ValueError("private"), 1, 1)
        r.snapshot()
        self.assertEqual(before, r._encode([ITEM]))
        # Independently construct the expected OTLP payload, including ordering.
        from opentelemetry.proto.collector.trace.v1.trace_service_pb2 import ExportTraceServiceRequest
        req = ExportTraceServiceRequest()
        resource = req.resource_spans.add()
        resource.resource.attributes.add(key="service.name").value.string_value = r.service
        resource.resource.attributes.add(key="signalscout.synthetic").value.bool_value = True
        scope = resource.scope_spans.add()
        scope.scope.name = "signalscout.synthetic.spike"
        span = scope.spans.add(trace_id=bytes.fromhex(ITEM["trace"]), span_id=bytes.fromhex(ITEM["span"]),
                              name="synthetic", start_time_unix_nano=1, end_time_unix_nano=2)
        span.status.code = 1
        self.assertEqual(before, req.SerializeToString())

    def test_doctor_exposes_bounded_diagnostics_and_batch_sizes(self):
        from scout_probe import Injector
        r = self.make()
        for _ in range(3):
            r._enqueue(ITEM)
        r._stop.set()
        with patch.object(r, "_send", return_value=None):
            r._work()
        doctor = Injector(r, []).doctor()
        self.assertEqual(doctor["counters"]["exported"], 3)
        self.assertEqual(doctor["counters"]["diagnostics"]["worker_batch_sizes"], {3: 1})
        self.assertEqual(doctor["counters"]["queue_high_water"], 3)
        self.assertIn("synthetic-only", doctor["privacy"])
