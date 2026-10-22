"""Synthetic diagnostic correctness; no scenario or reserved-seed execution."""
import unittest

from diagnose_burst import Meter, copies
from recorder import Recorder


class Diagnostic(unittest.TestCase):
    def test_private_copy_is_wire_identical_on_fixed_record(self):
        meter = Meter()
        namespaces, _ = copies(meter)
        encoded = []
        for cls in (Recorder, namespaces["recorder"]["Recorder"]):
            recorder = cls(_start_worker=False, capture_fields={"frame", "output"})
            try:
                with recorder.run("synthetic.diagnostic", run_id="ab" * 16, start_ns=10, end_ns=20):
                    self.assertTrue(recorder.record_decision("synthetic.step", output={"ref": "value:o@1", "value": 1},
                        inputs={"frame": {"ref": "value:f@1", "value": 10}}, chosen={"option": "returned"}, alternatives=[], rule={"id": "unobserved"},
                        record_id="fixed", at_ns=11, end_ns=12))
                items = []
                while not recorder._q.empty():
                    items.append(recorder._q.get_nowait())
                    recorder._q.task_done()
                encoded.append(recorder._encode(items))
            finally:
                recorder.close(.01)
        self.assertEqual(encoded[0], encoded[1])
        self.assertTrue(meter.report())

    def test_scope_preserves_exception_and_bounded_aggregate(self):
        meter = Meter()
        error = ValueError("synthetic")
        with self.assertRaises(ValueError) as caught:
            with meter.phase("fixed"):
                raise error
        self.assertIs(caught.exception, error)
        for _ in range(10000):
            with meter.phase("fixed"):
                pass
        report = meter.report()
        self.assertEqual(len(report), 2)
        self.assertEqual(report["outside_probe:fixed_inclusive"]["count"], 10001)
