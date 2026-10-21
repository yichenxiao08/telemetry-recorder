"""Fresh synthetic accepted fixtures through the real binary and read-only CLI."""
import tempfile
import unittest
from pathlib import Path

from trace_walkthrough import exercise


class TraceCLI(unittest.TestCase):
    def test_real_intake_fixture_a_b_and_all_evidence_anchors(self):
        with tempfile.TemporaryDirectory() as directory:
            result = exercise(Path(directory))
        self.assertEqual(result["stale_invariant"], "violated")
        self.assertEqual(result["fresh_invariant"], "satisfied")
        self.assertEqual(result["target_source_lines_changed"], 0)
        self.assertGreater(result["anchors_checked"], 10)
        self.assertGreater(result["pipeline_decisions"], 3)
        self.assertEqual(result["drops"], {})


if __name__ == "__main__":
    unittest.main()
