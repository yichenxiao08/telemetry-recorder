"""Development/toy recorder delivery tests. No reserved data or evaluation flags."""
import copy
import hashlib
import http.server
import threading
import time
import unittest
import urllib.error
import urllib.request
import tempfile
from pathlib import Path
from unittest.mock import patch

from recorder import Recorder
from spike_support import Serving
from test_export_diagnostics import ITEM


class RetryLifecycle(unittest.TestCase):
    def make(self, **kw):
        r = Recorder(_start_worker=False, **kw)
        self.addCleanup(lambda: r.close(0))
        return r

    def settle(self, r):
        r._stop.set()
        r._work()
        report = r.flush(.1)
        self.assertEqual(report["attempted"], sum(report[k] for k in ("acknowledged", "failed", "dropped", "unknown")))
        return report

    def test_transient_failure_recovers_once_with_same_batch(self):
        r = self.make()
        r._enqueue(ITEM)
        sent = []
        def send(batch):
            sent.append(r._encode(batch))
            if len(sent) == 1:
                r.diagnostics.phase = "wait-for-ack"
                raise TimeoutError("synthetic")
        with patch.object(r, "_send", side_effect=send):
            result = self.settle(r)
        self.assertTrue(result)
        self.assertEqual(result["acknowledged"], 1)
        self.assertEqual(result["unknown"], 0)
        self.assertEqual(sent[0], sent[1])
        self.assertEqual(r.snapshot().get("export_failed", 0), 0)
        self.assertEqual(r.snapshot()["retry_requests"], 1)

    def test_sustained_outage_retry_bound_and_unknown(self):
        r = self.make(cooldown=.01)
        r._enqueue(ITEM)
        def send(batch):
            r.diagnostics.phase = "wait-for-ack"
            raise TimeoutError("synthetic")
        with patch.object(r, "_send", side_effect=send) as calls:
            result = self.settle(r)
        self.assertEqual(calls.call_count, 2)
        self.assertFalse(result)
        self.assertEqual(result["unknown"], 1)
        self.assertEqual(result["failed"], 0)
        self.assertTrue(result["drained"])

    def test_encode_and_terminal_rejection_are_not_retried(self):
        for phase, error in (("encode", ValueError("synthetic")),
                             ("response", urllib.error.HTTPError("http://synthetic", 400, "synthetic", {}, None))):
            with self.subTest(phase=phase):
                r = self.make()
                r._enqueue(ITEM)
                def send(batch):
                    r.diagnostics.phase = phase
                    raise error
                with patch.object(r, "_send", side_effect=send) as calls:
                    result = self.settle(r)
                self.assertEqual(calls.call_count, 1)
                self.assertEqual(result["failed"], 1)
                self.assertEqual(result["unknown"], 0)
                self.assertFalse(result)

    def test_saturation_and_flush_unknown_are_exact(self):
        r = self.make(capacity=2)
        for _ in range(7):
            r._enqueue(ITEM)
        begin = time.monotonic()
        result = r.flush(.01)
        self.assertLess(time.monotonic()-begin, .1)
        self.assertFalse(result)
        self.assertEqual((result["attempted"], result["dropped"], result["unknown"]), (7, 5, 2))
        self.assertFalse(result["drained"])

    def test_stuck_network_has_total_deadline_and_bounded_threads(self):
        r = self.make()
        release, entered = threading.Event(), threading.Event()
        self.addCleanup(release.set)
        def send(batch):
            r.diagnostics.phase = "wait-for-ack"
            entered.set()
            release.wait(5)
        r._enqueue(ITEM)
        with patch.object(r, "_send", side_effect=send):
            worker = threading.Thread(target=r._work, daemon=True)
            worker.start()
            self.assertTrue(entered.wait(1))
            time.sleep(1.35)
            report = r.flush(0)
            self.assertEqual(report["unknown"], 1)
            self.assertTrue(report["drained"])
            self.assertEqual(r.snapshot()["deadline_expired_spans"], 1)
            for _ in range(1000):
                r._enqueue(ITEM)
            self.assertLessEqual(r.snapshot()["queue_size"], 256)
            self.assertEqual(r.snapshot()["network_operation_slots"], 1)
            self.assertEqual(r.snapshot().get("retry_requests", 0), 0)
            release.set()
            r._stop.set()
            worker.join(3)
            self.assertFalse(worker.is_alive())
            final = r.flush(0)
            self.assertFalse(final)
            self.assertEqual(final["unknown"], 1)  # A late reply never credits the expired batch.
            self.assertEqual(final["acknowledged"], 256)
            self.assertEqual(final["dropped"], 744)
            self.assertEqual(final["attempted"], 1001)


class ExplicitAcceptance(unittest.TestCase):
    def test_cooldown_does_not_discard_queued_batches(self):
        r = Recorder(_start_worker=False, cooldown=.15)
        self.addCleanup(lambda: r.close(0))
        for _ in range(40):
            r._enqueue(ITEM)
        r._cooldown_until = time.monotonic()+10
        sent = []
        with patch.object(r, "_send", side_effect=lambda batch: sent.extend(batch)):
            worker = threading.Thread(target=r._work, daemon=True)
            worker.start()
            time.sleep(.04)
            self.assertEqual(len(sent), 0)
            self.assertEqual(r.snapshot()["queue_size"], 40)
            self.assertTrue(r._enqueue(ITEM))
            self.assertTrue(r.snapshot()["cooldown_active"])
            r._cooldown_until = 0  # Release the controlled cooldown after the assertions.
            report = r.flush(2)
            self.assertTrue(report)
            self.assertEqual(len(sent), 41)
            self.assertEqual(report["acknowledged"], 41)
            self.assertEqual(report["dropped"], 0)
            self.assertEqual(r.snapshot().get("drop_cooldown", 0), 0)
            self.assertEqual(r.snapshot()["diagnostics"]["counters_spans_except_flush_timeout"]["cooldown_suppressed"], 0)
            self.evidence = {"acceptance": "cooldown preserves queued batches", "queued_during_cooldown": 40,
                             "admitted_during_cooldown": 1, "flush": dict(report)}
            r.close(0)
            worker.join(1)

    def test_duplicate_delivery_is_idempotent_after_lost_ack(self):
        with tempfile.TemporaryDirectory() as temp, Serving(Path(temp)/"fresh.db") as server:
            requests, commits = [], []
            class Relay(http.server.BaseHTTPRequestHandler):
                def log_message(self, *a):
                    pass
                def do_POST(self):
                    body = self.rfile.read(int(self.headers["Content-Length"]))
                    requests.append(hashlib.sha256(body).hexdigest())
                    request = urllib.request.Request(server.endpoint+"/v1/traces", body,
                                                     {"Content-Type": "application/x-protobuf"})
                    with urllib.request.urlopen(request, timeout=3) as response:
                        payload = response.read()
                    commits.append(time.monotonic())
                    if len(requests) == 1:
                        time.sleep(.65)
                    try:
                        self.send_response(200)
                        self.end_headers()
                        self.wfile.write(payload)
                    except (BrokenPipeError, ConnectionError):
                        pass
            relay = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Relay)
            thread = threading.Thread(target=relay.serve_forever, daemon=True)
            thread.start()
            r = Recorder(f"http://127.0.0.1:{relay.server_port}")
            try:
                item = copy.deepcopy(ITEM)
                item["events"] = [("synthetic.event", 1, {"synthetic": True})]
                r._enqueue(item)
                report = r.flush(3)
                self.assertTrue(report)
                self.assertEqual(report["acknowledged"], 1)
                self.assertEqual(len(requests), 2)
                self.assertEqual(requests[0], requests[1])
                self.assertEqual(len(commits), 2)
                detail = server.show(item["trace"])
                signals = detail["signals"]
                self.assertEqual(len(signals), 2)  # One span plus one event, never four.
                self.assertEqual(len({s["id"] for s in signals}), 2)
                self.evidence = {"acceptance": "duplicate delivery idempotency", "requests": requests,
                                 "committed_requests": len(commits), "stored_signal_ids": [s["id"] for s in signals],
                                 "flush": dict(report)}
            finally:
                r.close(0)
                relay.shutdown()
                relay.server_close()
                thread.join(2)
