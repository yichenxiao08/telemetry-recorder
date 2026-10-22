"""Synthetic rule fixtures have independent, analytic counts."""
import importlib.util
from pathlib import Path
import unittest
import sys


class RelationFixtureTests(unittest.TestCase):
    def test_admission_fixture_has_independent_boolean_and_bounds_cases(self):
        path = Path(__file__).resolve().parents[1] / "fixtures/relations/range_target.py"
        spec = importlib.util.spec_from_file_location("ranges_plain", path)
        target = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(target)
        mismatches, inverted = [], []
        for frame in range(16):
            value, low, high, enabled = target.arguments(frame)
            result = target.admit(value, low, high, enabled)
            if result["accepted"] != (enabled and low <= value <= high):
                mismatches.append(frame)
            if low > high:
                inverted.append(frame)
            else:
                self.assertTrue(low <= result["clipped"] <= high)
            self.assertEqual(result["limits"], (low, high))
        self.assertEqual(mismatches, [5])
        self.assertEqual(inverted, [0, 8])

    def test_minority_violation_is_independent_of_flag(self):
        path = Path(__file__).resolve().parents[1] / "fixtures/relations/relation_target.py"
        spec = importlib.util.spec_from_file_location("relations_plain", path)
        target = importlib.util.module_from_spec(spec)
        sys.path.insert(0, str(path.parent))
        try:
            spec.loader.exec_module(target)
        finally:
            sys.path.pop(0)
        table = {(violation, flag): 0 for violation in (False, True) for flag in (False, True)}
        for frame in range(40):
            state, reported, flag, ordered = target.run_case(frame)
            violation = reported != abs(state[3] - state[0])
            table[violation, flag] += 1
            self.assertTrue(ordered)
        self.assertEqual(table, {(True, True): 2, (True, False): 2,
                                 (False, True): 18, (False, False): 18})
