"""Pure saved-report checks; no generator, reserved data or engine invocation."""
import copy
import hashlib
import json
import unittest

from report_capture_fix import accounting, analytic, canonical, portable_metadata
from recorder import FlushReport


class Reporting(unittest.TestCase):
    def test_analytic_comparison_validates_evidence_without_erasing_values(self):
        def case(evidence):
            url = "/run?id=synthetic-run#e-"+hashlib.sha256(evidence.encode()).hexdigest()
            finding = {"score": .7, "count": 4, "evidence_ids": [evidence], "evidence_urls": [url]}
            return {"result": {"problem": {"created_at": 1, "decision_ids": {"synthetic-run": evidence}},
                               "reference": {"created_at": 2, "decision_ids": {}}, "differences": [finding],
                               "invariants": [{"evidence_id": evidence, "observed": 3}]},
                    "rankings": [1], "raw_numeric": [2], "manifest": ["event:literal-input"],
                    "expectation_mismatches": [], "invariants_by_arm": [4], "verified_evidence_links": 1}
        before, after = case("event:old"), case("event:new")
        self.assertNotEqual(canonical(before), canonical(after))  # Keep the strict failure.
        self.assertEqual(analytic(before), analytic(after))
        for key, value in (("score", .8), ("count", 5)):
            changed = copy.deepcopy(after)
            changed["result"]["differences"][0][key] = value
            self.assertNotEqual(analytic(before), analytic(changed))
        changed = copy.deepcopy(after)
        changed["manifest"] = ["event:different-input"]
        self.assertNotEqual(analytic(before), analytic(changed))
        changed = copy.deepcopy(after)
        changed["result"]["differences"][0]["evidence_urls"] = ["/run?id=wrong#e-invalid"]
        with self.assertRaises(KeyError):
            analytic(changed)
        changed = copy.deepcopy(after)
        changed["result"]["invariants"][0]["evidence_id"] = "event:unknown"
        with self.assertRaises(ValueError):
            analytic(changed)

    def test_incomplete_flush_counts_survive_explicit_json_conversion(self):
        for complete in (False, True):
            value = FlushReport(attempted=1, acknowledged=int(complete), failed=0,
                                dropped=0, unknown=int(not complete), complete=complete, drained=complete)
            self.assertEqual(bool(value), complete)
            # Python's indented dict encoder checks truthiness. Convert at
            # each report boundary so false completeness cannot erase counts.
            decoded = json.loads(json.dumps(dict(value), indent=2))
            self.assertEqual(decoded, dict(value))
            self.assertEqual(decoded["unknown"], int(not complete))

    def test_metadata_paths_are_portable_with_json_escaping(self):
        root = b"C:\\synthetic\\repo"
        for value in (b"C:\\synthetic\\repo/file", b'C:\\\\synthetic\\\\repo/file', b"C:/synthetic/repo/file"):
            self.assertEqual(portable_metadata(value, root), b"<repo>/file")

    def test_four_state_identity_does_not_hide_unknown(self):
        for counts, valid in (({"attempted": 9, "acknowledged": 3, "failed": 1, "dropped": 2, "unknown": 3}, True),
                              ({"attempted": 9, "acknowledged": 3, "failed": 1, "dropped": 2, "unknown": 0}, False)):
            with self.subTest(counts=counts):
                result = accounting(counts)
                self.assertEqual(result["identity_pass"], valid)
                self.assertEqual(result["unknown"], counts["unknown"])
                self.assertTrue(result["nonnegative"])
        # A derived remainder must not hide double acknowledgement as a
        # negative unknown, even though its algebraic identity still balances.
        result = accounting({"attempted": 1, "acknowledged": 2, "unknown": -1})
        self.assertTrue(result["identity_pass"])
        self.assertFalse(result["nonnegative"])

    def test_regression_projection_ignores_only_two_creation_clocks(self):
        case = {"result": {"problem": {"created_at": 1}, "reference": {"created_at": 2},
                           "score": .7, "count": 4, "evidence_ids": ["synthetic"]},
                "rankings": [1], "raw_numeric": [2], "manifest": [3],
                "expectation_mismatches": [], "invariants_by_arm": [4], "verified_evidence_links": 1}
        original = copy.deepcopy(case)
        changed = copy.deepcopy(case)
        changed["result"]["problem"]["created_at"] = 9
        changed["result"]["reference"]["created_at"] = 8
        self.assertEqual(canonical(case), canonical(changed))
        for key, value in (("score", .8), ("count", 5), ("evidence_ids", ["different"])):
            candidate = copy.deepcopy(changed)
            candidate["result"][key] = value
            self.assertNotEqual(canonical(case), canonical(candidate))
        self.assertEqual(case, original)


if __name__ == "__main__":
    unittest.main()
