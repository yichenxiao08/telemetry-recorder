"""Development/toy tests for the optional item+byte retained-work bound. No reserved data."""
import io
import threading
import time
import unittest
from contextlib import redirect_stderr
from unittest.mock import patch

import scout_probe
from recorder import Recorder, estimate_span_bytes, SPAN_FIXED_BYTES
from test_export_diagnostics import ITEM


def span(n=0, pad=0):
    return {"trace": "1" * 32, "span": f"{n + 1:016x}", "parent": "", "name": "synthetic",
            "start": 1, "end": 2, "attrs": {"pad": "x" * pad} if pad else {}, "events": [], "error": False}


def wait_for(predicate, seconds=2):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        if predicate():
            return True
        time.sleep(.005)
    return False


def identity(testcase, snapshot):
    testcase.assertEqual(snapshot["attempted"], snapshot["acknowledged"] + snapshot["failed"]
                         + snapshot["dropped"] + snapshot["unknown"])


class QueueBound(unittest.TestCase):
    def make(self, **kw):
        r = Recorder(**kw)
        self.addCleanup(lambda: r.close(0))
        return r

    def test_default_behavior_unchanged(self):
        r = self.make(_start_worker=False)
        self.assertIsNone(r._limits)
        self.assertEqual(r._q.maxsize, 256)
        for n in range(257):
            r._enqueue(span(n))
        s = r.snapshot()
        self.assertEqual((s["accepted"], s["drop_full"], s["dropped"], s["capacity"]), (256, 1, 1, 256))
        self.assertNotIn("drop_bytes", s)
        self.assertFalse([k for k in s if k.startswith(("retained_", "limit_"))])
        self.assertNotIn("byte_limit_drop", s["diagnostics"]["counters_spans_except_flush_timeout"])
        self.assertIsInstance(r._q.get_nowait(), dict)  # queue elements are still raw spans
        for bad in (0, 4097):
            with self.assertRaises(ValueError):
                Recorder(_start_worker=False, capacity=bad)
        Recorder(_start_worker=False, capacity=4096).close(0)

    def test_limits_replace_capacity_and_validate(self):
        with self.assertRaises(ValueError):
            Recorder(_start_worker=False, capacity=100, queue_limits=(10, 10**6))
        for bad in ((0, 10**6), (10, 0), (True, 10**6), (10, True), (10.0, 10**6), (65537, 10**6),
                    (10, 256 * 1024 * 1024 + 1), (10,)):
            with self.subTest(bad=bad), self.assertRaises((ValueError, TypeError)):
                Recorder(_start_worker=False, queue_limits=bad)
        r = self.make(_start_worker=False, queue_limits=(6560, 53739520))  # above the legacy 4096 cap
        self.assertEqual(r.snapshot()["capacity"], 6560)

    def test_bound_by_items(self):
        r = self.make(_start_worker=False, queue_limits=(5, 10**7))
        results = [r._enqueue(span(n)) for n in range(8)]
        s = r.snapshot()
        self.assertEqual(results, [True] * 5 + [False] * 3)
        self.assertEqual((s["accepted"], s["drop_full"], s["drop_bytes"], s["dropped"]), (5, 3, 0, 3))
        self.assertEqual((s["retained_items"], s["retained_items_peak"]), (5, 5))
        identity(self, s)

    def test_bound_by_bytes(self):
        size = estimate_span_bytes(span(0, 1000))
        r = self.make(_start_worker=False, queue_limits=(100, 3 * size + size // 2))
        results = [r._enqueue(span(n, 1000)) for n in range(6)]
        s = r.snapshot()
        self.assertEqual(results, [True] * 3 + [False] * 3)
        self.assertEqual((s["accepted"], s["drop_full"], s["drop_bytes"]), (3, 0, 3))
        self.assertEqual(s["retained_bytes_estimate"], 3 * size)
        self.assertLessEqual(s["retained_bytes_estimate_peak"], 3 * size + size // 2)
        self.assertEqual(s["diagnostics"]["counters_spans_except_flush_timeout"]["byte_limit_drop"], 3)
        identity(self, s)

    def test_single_oversized_span_is_a_visible_byte_drop(self):
        r = self.make(_start_worker=False, queue_limits=(10, 600))
        self.assertGreater(estimate_span_bytes(span(0, 500)), 600)
        self.assertFalse(r._enqueue(span(0, 500)))
        self.assertTrue(r._enqueue(span(1)))
        s = r.snapshot()
        self.assertEqual((s["drop_bytes"], s["accepted"], s["retained_items"]), (1, 1, 1))
        identity(self, s)

    def test_estimate_is_deterministic_and_counts_payload(self):
        self.assertEqual(estimate_span_bytes(span(0)), SPAN_FIXED_BYTES + len("synthetic"))
        self.assertEqual(estimate_span_bytes(span(0, 100)) - estimate_span_bytes(span(0)), len("pad") + 100)
        self.assertEqual(estimate_span_bytes(span(3, 7)), estimate_span_bytes(span(3, 7)))

    def test_real_decision_span_is_sized_and_released(self):
        r = self.make(_start_worker=False, queue_limits=(10, 10**6))
        with r.run("synthetic", run_id="1" * 32):
            self.assertTrue(r.record_decision("step", output={"ref": "o"}, inputs={"a": {"ref": "i"}},
                                              chosen={"option": "a"}, alternatives=[], rule={"id": "r"}))
        s = r.snapshot()
        self.assertEqual(s["retained_items"], 2)  # decision span + root span
        self.assertGreater(s["retained_bytes_estimate"], 2 * SPAN_FIXED_BYTES)

    def test_in_flight_batch_counts_toward_the_bound(self):
        r = self.make(queue_limits=(2, 10**6))
        started, release = threading.Event(), threading.Event()
        def send(batch):
            started.set()
            release.wait(5)
        with patch.object(r, "_send", side_effect=send):
            self.assertTrue(r._enqueue(span(0)))
            self.assertTrue(started.wait(2))
            self.assertTrue(wait_for(lambda: r._q.qsize() == 0))  # the worker holds it in flight
            self.assertEqual(r.snapshot()["retained_items"], 1)
            self.assertTrue(r._enqueue(span(1)))   # queued + in flight == 2
            self.assertFalse(r._enqueue(span(2)))  # would pass if in-flight were not counted
            s = r.snapshot()
            self.assertEqual((s["retained_items"], s["drop_full"], s["queue_size"]), (2, 1, 1))
            release.set()
            report = r.flush(2)
            self.assertTrue(report["drained"])
            self.assertFalse(report["complete"])  # one visible drop: drained is not complete
        s = r.snapshot()
        self.assertEqual((s["retained_items"], s["retained_bytes_estimate"], s["acknowledged"]), (0, 0, 2))
        self.assertEqual(s["retained_items_peak"], 2)
        identity(self, s)

    def test_accounting_identity_with_concurrent_producers(self):
        for limits in ((50, 10**7), (10**4, 40 * 1200)):
            with self.subTest(limits=limits):
                r = self.make(queue_limits=limits)
                sent = []
                lock = threading.Lock()
                def send(batch):
                    with lock:
                        sent.append(len(batch))
                    time.sleep(.001)
                def produce(base):
                    for n in range(200):
                        r._enqueue(span(base * 1000 + n, 100 * (n % 7)))
                with patch.object(r, "_send", side_effect=send):
                    threads = [threading.Thread(target=produce, args=(i,)) for i in range(8)]
                    for t in threads:
                        t.start()
                    for t in threads:
                        t.join(10)
                    report = r.flush(5)
                s = r.snapshot()
                self.assertTrue(report["drained"])
                self.assertEqual(s["attempted"], 1600)
                identity(self, s)
                self.assertEqual(s["accepted"] + s["drop_full"] + s["drop_bytes"], 1600)
                self.assertEqual(s["unknown"], 0)
                self.assertEqual(s["acknowledged"], s["accepted"])
                self.assertEqual(sum(sent), s["acknowledged"])
                self.assertEqual((s["retained_items"], s["retained_bytes_estimate"]), (0, 0))
                self.assertLessEqual(s["retained_items_peak"], limits[0])
                self.assertLessEqual(s["retained_bytes_estimate_peak"], limits[1])
                self.assertGreater(s["dropped"], 0)

    def test_caller_never_blocks_when_full_and_worker_is_stalled(self):
        r = self.make(queue_limits=(4, 10**6))
        started, release = threading.Event(), threading.Event()
        def send(batch):
            started.set()
            release.wait(5)
        durations = []
        def produce():
            for n in range(500):
                begin = time.perf_counter()
                r._enqueue(span(n))
                durations.append(time.perf_counter() - begin)
        with patch.object(r, "_send", side_effect=send):
            r._enqueue(span(9999))
            self.assertTrue(started.wait(2))
            caller = threading.Thread(target=produce)
            caller.start()
            caller.join(1)
            self.assertFalse(caller.is_alive(), "caller blocked on a full bounded queue")
            self.assertEqual(len(durations), 500)
            self.assertLess(max(durations), .25)
            s = r.snapshot()
            self.assertEqual(s["retained_items"], 4)
            self.assertEqual(s["drop_full"], 501 - 4)
            identity(self, s)
            release.set()
            r.flush(2)
        identity(self, r.snapshot())

    def test_otlp_bytes_unchanged_by_the_bound(self):
        plain = Recorder(_start_worker=False)
        bounded = Recorder(_start_worker=False, queue_limits=(10, 10**6))
        self.addCleanup(lambda: (plain.close(0), bounded.close(0)))
        item = dict(ITEM, attrs={"k": "v"}, events=[("e", 5, {"a": 1, "b": "text", "c": 1.5, "d": True})])
        bounded._enqueue(item)
        queued, size = bounded._q.get_nowait()
        self.assertIs(queued, item)
        self.assertEqual(size, estimate_span_bytes(item))
        self.assertEqual(bounded._encode([queued]), plain._encode([item]))

    def test_cli_requires_both_bounds(self):
        for flags in (["--queue-max-items", "10"], ["--queue-max-bytes", "10"]):
            with self.subTest(flags=flags), redirect_stderr(io.StringIO()), self.assertRaises(SystemExit) as raised:
                scout_probe.main(["--config", "unused.yaml", "--doctor", "unused.json", *flags, "m:f"])
            self.assertEqual(raised.exception.code, 2)


if __name__ == "__main__":
    unittest.main()
