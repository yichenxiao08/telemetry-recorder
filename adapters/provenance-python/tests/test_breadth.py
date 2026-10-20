"""Synthetic fixture behavior and optional call safety; no new engine behavior."""
import importlib
import sys
import unittest
from pathlib import Path
from unittest.mock import patch

import scout_probe
from recorder import Recorder


class Breadth(unittest.TestCase):
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
            self.assertTrue(scout_probe.record_decision("hidden-rule", output={"ref":"synthetic:x"}, inputs={}, chosen={"option":"first"}, alternatives=[], rule={"id":"synthetic-first"}))
        self.assertEqual(recorder.snapshot()["accepted"], 2)
