"""Synthetic fixture behavior and optional call safety; no new engine behavior."""
import importlib
import sys
import time
import unittest
from pathlib import Path
from unittest.mock import patch

import scout_probe
from recorder import Recorder


def short_step(value):
    time.sleep(.002)
    return value


class Breadth(unittest.TestCase):
    def test_duration_survives_coarse_wall_clock(self):
        recorder = Recorder(capture_fields={"value", "output"}, _start_worker=False)
        owner = scout_probe.Injector(recorder, [])
        probe = scout_probe.Probe(owner, short_step, {"module":__name__,"function":"short_step","capture":{"args":["value"],"return":True}})
        with patch("recorder.time.time_ns", return_value=1792478045000000000):
            self.assertEqual(probe.wrap()(3), 3)
        child, root = recorder._q.get_nowait(), recorder._q.get_nowait()
        self.assertGreater(child["end"]-child["start"], 0)
        self.assertGreater(root["end"]-root["start"], 0)
        self.assertLessEqual(root["start"], child["start"])
        self.assertGreaterEqual(root["end"], child["end"])

    def test_failed_run_setup_preserves_host(self):
        recorder = Recorder(_start_worker=False)
        owner = scout_probe.Injector(recorder, [])
        probe = scout_probe.Probe(owner, short_step, {"module":__name__,"function":"short_step"})
        invalid = recorder.run("x"*300)
        with patch.object(recorder, "run", return_value=invalid):
            self.assertEqual(probe.wrap()(3), 3)
        self.assertIsNone(recorder._active.get())

    def test_plain_and_optional_preserve_replies(self):
        sys.path.insert(0, str(Path(__file__).resolve().parents[1]/"fixtures"))
        plain = importlib.import_module("pipeline_target")
        enriched = importlib.import_module("pipeline_enriched")
        for index in range(12):
            self.assertEqual(plain.turn(index), enriched.turn(index))
        self.assertEqual(plain.turn(0)["reply"], "decline")
        self.assertEqual(plain.turn(1)["reply"], "synthetic repaired answer")
        self.assertEqual(plain.context_select(["a", "b", "c"], 1), ["c"])

    def test_optional_fault_never_escapes(self):
        self.assertFalse(scout_probe.record_decision("unconfigured"))
        with patch.object(scout_probe, "_optional", object()):
            self.assertFalse(scout_probe.record_decision("invalid"))
        recorder = Recorder(_start_worker=False)
        owner = scout_probe.Injector(recorder, [])
        with patch.object(scout_probe, "_optional", owner), recorder.run("synthetic"):
            output={"value": "synthetic"}
            self.assertTrue(scout_probe.record_decision("hidden-rule", output=output, inputs={}, chosen={"option":"first"}, alternatives=[], rule={"id":"synthetic-first"}))
            self.assertNotIn("ref", output)
            item=recorder._q.queue[0]
            self.assertTrue(item["events"][0][2]["signalscout.output.ref"].startswith("value:"+recorder._active.get().trace_id+"/"))
        self.assertEqual(recorder.snapshot()["accepted"], 2)
