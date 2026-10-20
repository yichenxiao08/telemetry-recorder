"""Only development seeds. No evaluation flag in tests, imports or CI."""
import ast
import importlib
import json
import sys
import unittest
import tempfile
import urllib.request
import hashlib
import subprocess
from pathlib import Path
from unittest.mock import patch

import lane02_generator as gen

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(Path(__file__).resolve().parents[1]/"fixtures"))
import lane02_target as target
from lane02 import HarnessClock, post, utc
from recorder import Recorder
from scout_probe import Injector, load_config
from spike_support import Serving


class Lane02Development(unittest.TestCase):
    def test_golden_independent_vectors(self):
        vectors = json.loads((ROOT/"benchmark/problems/lane-02/golden.json").read_text())
        for v in vectors:
            self.assertIn(v["seed"], gen.DEVELOPMENT)
            for value in v["values"]:
                args = (v["seed"], v["variant"], v["session"], value["pair"], value["field"], value["arm"])
                self.assertEqual(gen.uniform(*args), value["u"])
                self.assertEqual(gen.digest(*args).hex(), value["digest"])
            order = gen.ordering(v["seed"], v["variant"], v["session"])
            self.assertEqual(order, [(x["arm"], x["pair"]) for x in v["ordering"]])
            frames = [x["inputs"]["frame"] for x in gen.rows(v["seed"], v["variant"])
                      if x["session_index"] == v["session"]]
            self.assertEqual(frames, [x["frame"] for x in v["ordering"]])

    def test_holdouts_rejected_before_hash_or_generation(self):
        with patch.object(gen.hashlib, "sha256", side_effect=AssertionError("hash before guard")):
            for seed in gen.RESERVED:
                with self.assertRaises(ValueError):
                    gen.rows(seed, gen.VARIANTS[0])
                with self.assertRaises(ValueError):
                    gen.uniform(seed, gen.VARIANTS[0], 0, 0, "gap", "shared")
                with self.assertRaises(ValueError):
                    gen.ordering(seed, gen.VARIANTS[0], 0)
            importlib.reload(gen)  # Import itself has no generation or hashing.
        for path in Path(__file__).parent.glob("test_*.py"):
            for node in ast.walk(ast.parse(path.read_text(encoding="utf-8-sig"))):
                if isinstance(node, ast.keyword) and node.arg == "evaluation_runner":
                    self.fail("tests cannot set the evaluation flag")
        self.assertNotIn("--"+"evaluation-runner", (ROOT/".github/workflows/ci.yml").read_text())

    def test_counterfactuals_and_noise(self):
        for seed in gen.DEVELOPMENT:
            for variant in gen.VARIANTS:
                data = gen.rows(seed, variant)
                self.assertEqual(data, gen.rows(seed, variant))
                labels = {"problem": 0, "reference": 0}
                stale = current = 0
                for row in data:
                    fields = row["inputs"]
                    result = target.decide(**fields)
                    label = "problem" if result["lane"] != "right" else "reference"
                    labels[label] += 1
                    self.assertEqual(label == "problem", row["arm"] == "P")
                    self.assertEqual(result, target.decide(**{**fields, "noise": [999]*7}))
                    if label != "problem":
                        continue
                    if result["previous_used"]:
                        stale += 1
                        self.assertEqual(fields["history"][str(fields["previous_frame"])], -1)
                        fresh = {**fields, "previous_frame": fields["frame"]-1}
                        self.assertEqual(target.decide(**fresh)["lane"], "right")
                        self.assertTrue(target.decide(**fresh)["previous_used"])
                        self.assertEqual(fields["history"][str(fields["frame"]-1)], 1)
                    else:
                        current += 1
                        self.assertEqual(target.decide(**{**fields, "current_value": 1})["lane"], "right")
                self.assertEqual(labels, {"problem": 400, "reference": 400})
                self.assertEqual((stale, current), (400, 0) if variant == gen.VARIANTS[0] else (80, 320))

    def test_plain_target_and_harness_timestamp_origin(self):
        text = Path(target.__file__).read_text()
        self.assertFalse(any(isinstance(n, (ast.Import, ast.ImportFrom)) for n in ast.walk(ast.parse(text))))
        self.assertNotIn("clock", text)
        self.assertEqual(gen.ORIGIN_NS, 1767225600000000000)

    def test_real_probe_intake_clock_and_evidence(self):
        cfg = load_config(ROOT/"benchmark/problems/lane-02/signalscout.yaml")
        original = target.decide
        with tempfile.TemporaryDirectory() as directory, Serving(Path(directory)/"test.db") as server:
            recorder = Recorder(server.endpoint, capture_fields={"gap", "current_value", "history", "previous_frame", "frame", "noise", "output"})
            injector = Injector(recorder, cfg["probes"])
            try:
                injector.install()
                row = gen.rows(101, gen.VARIANTS[0])[0]
                clock = HarnessClock(0)
                with recorder.run("lane02_target.decide", run_id=row["run_id"], start_ns=clock.start(0)):
                    result = target.decide(**row["inputs"])
                self.assertTrue(recorder.flush(3))
                detail = server.show(row["run_id"])
                self.assertEqual(detail["run"]["started_at"], gen.ORIGIN_NS)
                events = [x for x in detail["signals"] if x["name"] == "signalscout.decision"]
                self.assertEqual(len(events), 1)
                event = events[0]
                self.assertGreaterEqual(event["ts_ns"], clock.start(0))
                self.assertLess(event["ts_ns"], clock.start(0)+1_000_000_000)
                envelope = json.loads(event["attrs"]["signalscout.record.json"])
                self.assertEqual(envelope["output"]["inline"], result)
                captured = {x["name"]: x["inline"] for x in envelope["inputs"]}
                self.assertEqual(captured, row["inputs"])
                anchor = "e-"+hashlib.sha256(event["id"].encode()).hexdigest()
                with urllib.request.urlopen(server.endpoint+"/run?id="+row["run_id"], timeout=10) as response:
                    self.assertIn('id="'+anchor+'"', response.read().decode())
                other = next(x for x in gen.rows(101, gen.VARIANTS[0]) if x["arm"] != row["arm"])
                with recorder.run("lane02_target.decide", run_id=other["run_id"], start_ns=clock.start(1)):
                    other_result = target.decide(**other["inputs"])
                self.assertTrue(recorder.flush(3))
                for entry, output in ((row, result), (other, other_result)):
                    post(server.endpoint, "/api/annotations", {"id": "annotation:"+entry["run_id"]+"/unit", "run_id": entry["run_id"],
                         "dimension": "lane02-unit", "value": "problem" if output["lane"] == "left" else "reference",
                         "source": "synthetic-unit", "reviewer": "synthetic-unit", "synthetic": True, "rubric_id": "unit",
                         "rubric_version": "1", "created_at": clock.base, "evidence_ids": []})
                for arm in ("problem", "reference"):
                    subprocess.check_output([str(server.binary), "cohort", "create", arm, "--endpoint", server.endpoint,
                        "--profile", str(ROOT/"benchmark/problems/lane-02/profile.json"), "--from", utc(clock.base),
                        "--to", utc(clock.base+60_000_000_000), "--limit", "2", "--dimension", "lane02-unit", "--value", arm])
                command = [str(server.binary), "--db", str(server.database), "explain-cohort", "problem", "reference"]
                comparison = json.loads(subprocess.check_output(command+["--json"]))
                self.assertEqual(comparison["presentation"]["the_problem"][0]["feature"], "output.lane")
                for finding in comparison["presentation"]["candidate_tail"]:
                    self.assertIsNone(finding["score"])
                    self.assertEqual(finding["status"], "insufficient_tail_evidence")
                text = subprocess.check_output(command).decode()
                self.assertIn("The problem", text)
                self.assertIn("Candidate median", text)
                self.assertIn("Separate reference-tail", text)
                self.assertNotIn("output.lane", text.split("Candidate median")[1])
                self.assertFalse(any(v for k, v in recorder.snapshot().items() if k.startswith("drop") or k.endswith("failed")))
            finally:
                injector.uninstall()
                target.decide = original
                recorder.close(2)
