"""Diagnostic observers must preserve the original behavior and wire format."""
import contextvars
import json
import threading
import types
import unittest
from unittest.mock import patch

import recorder
import scout_probe
from profile_phases import Meter, PHASES, copies


def target(value):
    return {"values": value}


def throwing(value):
    raise value


class Clock:
    def __init__(self):
        self.tick = 1000000

    def perf_counter_ns(self):
        self.tick += 100
        return self.tick

    def time_ns(self):
        return 1767225600000000000


class ProfilingTests(unittest.TestCase):
    def record(self, instrumented, capture=True):
        meter = Meter()
        api = copies(meter) if instrumented else types.SimpleNamespace(
            Recorder=recorder.Recorder, Probe=scout_probe.Probe)
        r = api.Recorder(_start_worker=False, capture_fields=("value", "output") if capture else ())
        owner = types.SimpleNamespace(recorder=r, parent=contextvars.ContextVar("test", default=None))
        spec = {"module": __name__, "function": "target"}
        if capture:
            spec["capture"] = {"args": ["value"], "return": True}
        probe = api.Probe(owner, target, spec)
        values = [1, 2]
        clock = Clock()
        meter.clock_source = clock
        meter.begin()
        with patch.object(recorder, "time", clock), patch.object(scout_probe, "time", clock):
            with r.run("synthetic", run_id="1"*32, start_ns=1767225600000000000,
                       end_ns=1767225600000001000):
                returned = probe.wrap()(values)
                self.assertIs(returned["values"], values)
                values.append(3)
        sample = meter.finish()
        items = []
        while not r._q.empty():
            items.append(r._q.get_nowait())
        body = json.loads(items[0]["events"][0][2]["signalscout.record.json"])
        if capture:
            self.assertEqual(body["inputs"][0]["inline"], [1, 2])
            self.assertEqual(body["output"]["inline"], {"values": [1, 2]})
        else:
            self.assertNotIn("inline", body["inputs"][0])
            self.assertNotIn("inline", body["output"])
        return r._encode(items), sample

    def test_wire_snapshot_and_privacy_equivalence(self):
        for capture in (False, True):
            with self.subTest(capture=capture):
                original, _ = self.record(False, capture)
                diagnostic, sample = self.record(True, capture)
                self.assertEqual(original, diagnostic)
                self.assertEqual(set(sample["adjusted_exclusive_ns"]), set(PHASES))
                self.assertGreater(sample["scope_counts"]["target_body"], 0)

    def test_exception_identity_and_cleanup(self):
        meter = Meter()
        api = copies(meter)
        r = api.Recorder(_start_worker=False)
        owner = types.SimpleNamespace(recorder=r, parent=contextvars.ContextVar("test", default=None))
        f = api.Probe(owner, throwing, {"module": __name__, "function": "throwing"}).wrap()
        error = RuntimeError("synthetic")
        meter.begin()
        try:
            f(error)
        except RuntimeError as caught:
            self.assertIs(caught, error)
        else:
            self.fail("host exception suppressed")
        meter.finish()
        self.assertIsNone(r._active.get())
        self.assertIsNone(owner.parent.get())
        self.assertEqual(meter.stack, [])

    def test_calibration_bounded_samples_and_thread_exclusion(self):
        meter = Meter(capacity=1)
        result = meter.calibrate(50)
        self.assertEqual(len(result["consecutive_timer_ns"]), 50)
        self.assertGreater(result["subtracted_whole_ns"], 0)
        for _ in range(2):
            meter.begin()
            def other():
                with meter.phase("queue_put"):
                    pass
            worker = threading.Thread(target=other)
            worker.start()
            worker.join()
            sample = meter.finish()
            self.assertNotIn("queue_put", sample["scope_counts"])
        self.assertEqual(len(meter.samples), 1)
        self.assertEqual(meter.overflow, 1)

    def test_original_classes_untouched(self):
        original = (recorder.Recorder, scout_probe.Probe)
        diagnostic = copies(Meter())
        self.assertIsNot(original[0], diagnostic.Recorder)
        self.assertIsNot(original[1], diagnostic.Probe)
        self.assertEqual(original, (recorder.Recorder, scout_probe.Probe))

    def test_harness_development_only_and_summary(self):
        import profile_wrapper
        function, fields, data = profile_wrapper.inputs("lane-inputs")
        self.assertEqual(len(data), 12*800)
        self.assertEqual(len(fields), 6)
        self.assertEqual(function(**data[0][1]), data[0][2])
        meter = Meter()
        meter.begin()
        with meter.phase("queue_put"):
            pass
        sample = meter.finish()
        summary = profile_wrapper.summarize([sample])
        self.assertEqual(summary["calls"], 1)
        self.assertEqual(summary["raw"]["p50_us"], sample["raw_total_ns"]/1000)

    def test_reporting_uses_one_observation_snapshot(self):
        import argparse
        import gzip
        from pathlib import Path
        import tempfile
        import profile_wrapper as harness
        meter = Meter()
        calibration = meter.calibrate(10)
        meter.begin()
        sample = meter.finish()
        factors = harness.Factors()
        factors.append("gc", sample["started_ns"], sample["stopped_ns"], 0)
        original = harness.summarize
        def mutate(samples):
            factors.append("_encode", sample["started_ns"], sample["stopped_ns"], 1)
            return original(samples)
        with tempfile.TemporaryDirectory() as directory:
            args = argparse.Namespace(out=Path(directory), factor="default")
            with patch.object(harness, "summarize", side_effect=mutate):
                harness.report(args, [sample], calibration, factors, {})
            saved = json.loads((Path(directory)/"summary.json").read_text())
            with gzip.open(Path(directory)/"raw.json.gz", "rt") as stream:
                raw = json.load(stream)
            self.assertEqual(len(raw["intervals"]), 1)
            self.assertEqual(saved["overlap"]["gc"]["all_calls"], 1)
            self.assertEqual(saved["overlap"]["_encode"]["all_calls"], 0)

    def test_raw_analysis_retains_negative_corrections_and_merges_intervals(self):
        from report_wrapper_profile import raw_analysis
        samples = [{"raw_total_ns": 10, "adjusted_total_ns": -2,
                    "started_ns": 100, "stopped_ns": 110, "caller_thread_cpu_ns": 0,
                    "adjusted_exclusive_ns": {"clock": -1}, "queue_depth_before": 256}]
        result = raw_analysis({"samples": samples, "intervals": [
            ["gc", 0, 105, 0], ["_encode", 2, 3, 1], ["_encode", 104, 120, 1]]})
        self.assertEqual(result["overlap"]["gc_or_encode"]["tail"], 1)
        self.assertEqual(result["clipped_adjusted_total"]["p50_us"], 0)
        self.assertEqual(samples[0]["adjusted_total_ns"], -2)
        self.assertEqual(result["queue_groups"]["full_before"]["calls"], 1)
        self.assertEqual(result["zero_thread_cpu_calls"], 1)
