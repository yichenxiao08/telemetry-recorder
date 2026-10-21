"""Tests authored before tuple and deterministic sampling implementation."""
import importlib
import json
import subprocess
import sys
import tempfile
import unittest
from contextlib import redirect_stderr
from unittest.mock import patch
import io
from pathlib import Path

import yaml
from recorder import Recorder, bounded
from scout_probe import Injector, load_config
import scout_probe
from spike_support import Serving
from test_probe_usability import drain

ROOT = Path(__file__).resolve().parents[1]
FIXTURE = ROOT / "fixtures"


class TupleSampling(unittest.TestCase):
    def setUp(self):
        sys.path.insert(0, str(FIXTURE))
        sys.modules.pop("tuple_environment", None)
        self.recorder = Recorder(_start_worker=False, capture_fields={"output"}, capacity=4096)
        self.owner = None

    def tearDown(self):
        if self.owner:
            self.owner.uninstall()
        self.recorder.close(0)
        sys.path.remove(str(FIXTURE))
        sys.modules.pop("tuple_environment", None)

    def attach(self, capture=None, sampling=None, function="Environment.step"):
        spec = {"module": "tuple_environment", "function": function,
                "capture": capture or {"return": True}}
        if sampling is not None:
            spec["sampling"] = sampling
        self.owner = Injector(self.recorder, [spec])
        self.owner.install()
        return importlib.import_module("tuple_environment")

    def test_tuple_namedtuple_snapshot_and_identity(self):
        module = self.attach(function="Environment.echo")
        env = module.Environment()
        for value in (([1], 2.0, False), module.Step([1], 2.0, False)):
            self.assertIs(env.echo(value), value)
            value[0].append(99)
        _, records = drain(self.recorder)
        self.assertEqual([r["output"]["inline"] for r in records], [[[1], 2.0, False]] * 2)
        self.assertEqual(bounded((1, 2)), [1, 2])

    def test_index_and_field_allowlists_default_off(self):
        module = self.attach({"return_items": [1, "done"]}, function="Environment.echo")
        env = module.Environment()
        env.echo(module.Step([7], 2.0, False))
        _, records = drain(self.recorder)
        self.assertEqual(records[0]["output"]["inline"], [{"unknown": "not_allowlisted"}, 2.0, False])
        self.assertFalse(records[0]["complete"])
        self.assertEqual(self.recorder.snapshot().get("drop_element", 0), 0)

    def test_default_off_empty_allowlist_and_nonsequence_cannot_leak(self):
        module = self.attach({"return_items": []}, function="Environment.echo")
        env = module.Environment()
        env.echo(("SYNTHETIC_SECRET", 2))
        env.echo("SYNTHETIC_SECRET")
        _, records = drain(self.recorder)
        self.assertEqual(records[0]["output"]["inline"], [{"unknown": "not_allowlisted"}] * 2)
        self.assertEqual(len(records), 1)
        self.assertNotIn("SYNTHETIC_SECRET", json.dumps(records))
        self.assertEqual(self.recorder.snapshot()["drop_invalid"], 1)

    def test_default_capture_off(self):
        module = self.attach({"args": []}, function="Environment.echo")
        module.Environment().echo(("SYNTHETIC_SECRET",))
        _, records = drain(self.recorder)
        self.assertEqual(records[0]["output"]["capture"], "off")
        self.assertNotIn("SYNTHETIC_SECRET", json.dumps(records))

    def test_index_allowlist_on_plain_tuple_and_depth_unknown(self):
        module = self.attach({"return_items": [1]}, function="Environment.echo")
        module.Environment().echo(("SYNTHETIC_SECRET", 7, 9))
        _, records = drain(self.recorder)
        self.assertEqual(records[0]["output"]["inline"], [{"unknown": "not_allowlisted"}, 7, {"unknown": "not_allowlisted"}])
        value = (1,)
        for _ in range(8):
            value = (value,)
        reasons = []
        self.assertIn("unknown", str(bounded(value, on_unknown=reasons.append)))
        self.assertEqual(reasons, ["too deep"])

    def test_element_failures_are_visible_and_do_not_discard_record(self):
        module = self.attach(function="Environment.echo")
        numpy_scalar = type("float64", (), {"__module__": "numpy"})()
        module.Environment().echo((object(), "x" * 257, numpy_scalar, [1]))
        _, records = drain(self.recorder)
        self.assertEqual(len(records), 1)
        self.assertEqual(records[0]["output"]["inline"][2], {"unknown": "numpy_unsupported"})
        self.assertEqual(records[0]["output"]["element_drops"], 3)
        self.assertFalse(records[0]["complete"])
        self.assertEqual(self.recorder.snapshot()["drop_element"], 3)
        self.assertIn("numpy_unsupported", str(self.owner.doctor()))
        self.assertEqual(self.recorder.snapshot().get("drop_invalid", 0), 0)

    def test_sampling_is_before_serialization_and_resets_per_run(self):
        module = self.attach(sampling={"every_nth": 3, "max_per_run": 2}, function="Environment.echo")
        env = module.Environment()
        for _ in range(2):
            with self.recorder.run("episode"):
                for index in range(1, 11):
                    env.echo((index,) if index in (1, 4) else object())
        items, records = drain(self.recorder)
        self.assertEqual([r["sampling"]["call_index"] for r in records], [1, 4, 1, 4])
        self.assertTrue(all(r["sampling"]["rule"] == {"every_nth": 3, "max_per_run": 2} for r in records))
        self.assertTrue(all(not r["complete"] for r in records))
        self.assertEqual(self.recorder.snapshot().get("drop_element", 0), 0)
        probe = self.owner.doctor()["probes"][0]
        self.assertEqual((probe["calls"], probe["records_captured"], probe["fraction"]), (20, 4, .2))
        roots = [i for i in items if not i["parent"]]
        self.assertEqual([i["attrs"]["signalscout.sampling.0.seen"] for i in roots], [10, 10])

    def test_bounds_and_exception_identity(self):
        with self.assertRaises(ValueError):
            bounded(tuple(range(33)))
        with self.assertRaises(ValueError):
            bounded((((((([1],),),),),),))
        module = self.attach(sampling={"max_per_run": 1}, function="Environment.raising")
        error = ValueError("synthetic")
        env = module.Environment()
        with self.recorder.run("episode"):
            for _ in range(2):
                with self.assertRaises(ValueError) as caught:
                    env.raising(error)
                self.assertIs(caught.exception, error)
        self.assertEqual(env.invocations, 2)
        _, records = drain(self.recorder)
        self.assertEqual(len(records), 1)
        self.assertEqual(records[0]["verdict"], "raised")

    def test_config_validates_sampling_and_allowlists(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "config.yaml"
            spec = {"module": "tuple_environment", "function": "Environment.step"}
            for bad in ({"every_nth": 0}, {"max_per_run": True}, {"every_nth": 1.5}, {"unknown": 1}):
                spec["sampling"] = bad
                path.write_text(yaml.safe_dump({"version": 1, "synthetic_only": True, "probes": [spec]}))
                with self.assertRaises(ValueError):
                    load_config(path)
            spec.pop("sampling")
            for bad in ([True], [-1], [32], ["bad.field"], [1, 1], "reward"):
                spec["capture"] = {"return_items": bad}
                path.write_text(yaml.safe_dump({"version": 1, "synthetic_only": True, "probes": [spec]}))
                with self.assertRaises(ValueError):
                    load_config(path)

    def test_overflow_remains_a_drop_with_sampling(self):
        self.recorder._q.maxsize = 1
        module = self.attach(sampling={"every_nth": 2})
        with self.recorder.run("episode"):
            for _ in range(20):
                module.Environment().step(1)
        self.assertEqual(self.recorder.snapshot()["drop_full"], 10)  # nine selected + root
        self.assertEqual(self.recorder.snapshot()["sampled_out:tuple_environment.Environment.step"], 10)

    def test_queue_drop_exit_five(self):
        self.recorder._q.maxsize = 1
        with tempfile.TemporaryDirectory() as directory:
            cfg = Path(directory) / "config.yaml"
            cfg.write_text(yaml.safe_dump({"version": 1, "synthetic_only": True,
                "probes": [{"module": "tuple_environment", "function": "loop"},
                           {"module": "tuple_environment", "function": "Environment.step",
                            "sampling": {"every_nth": 10}}]}))
            with patch.object(scout_probe, "Recorder", return_value=self.recorder), redirect_stderr(io.StringIO()):
                code = scout_probe.main(["--config", str(cfg), "--doctor", str(Path(directory) / "doctor.json"),
                                         "--path", str(FIXTURE), "tuple_environment:main"])
            self.assertEqual(code, 5)
            self.assertGreater(self.recorder.snapshot()["drop_full"], 0)
            self.assertEqual(self.recorder.snapshot()["sampled_out:tuple_environment.Environment.step"], 90)

    def test_no_user_hooks_and_numpy_array_reason(self):
        class TrapTuple(tuple):
            _fields = ("state", "reward")

            def __iter__(self):
                raise AssertionError("caller iterator")

            def __getitem__(self, index):
                raise AssertionError("caller indexing")

        self.assertEqual(bounded(TrapTuple((1, 2))), [1, 2])
        with self.assertRaisesRegex(ValueError, "numpy_unsupported"):
            bounded(type("ndarray", (), {"__module__": "numpy"})())

    def test_every_nth_only_and_bounded_per_run_reporting(self):
        module = self.attach(sampling={"every_nth": 2})
        for _ in range(34):
            with self.recorder.run("episode"):
                env = module.Environment()
                for _ in range(5):
                    env.step(1)
        _, records = drain(self.recorder)
        self.assertEqual([r["sampling"]["call_index"] for r in records], [1, 3, 5] * 34)
        report = self.owner.doctor()
        self.assertEqual(len(report["sampling_runs"]), 32)
        self.assertEqual(report["sampling_runs_omitted"], 2)

    def test_element_drop_exit_five_real_serve(self):
        with tempfile.TemporaryDirectory() as directory, Serving(Path(directory) / "invalid.db") as server:
            cfg = Path(directory) / "config.yaml"
            cfg.write_text(yaml.safe_dump({"version": 1, "synthetic_only": True, "endpoint": server.endpoint,
                "probes": [{"module": "tuple_environment", "function": "Environment.echo", "capture": {"return": True}}]}))
            doctor = Path(directory) / "doctor.json"
            result = subprocess.run([sys.executable, "-m", "scout_probe", "--config", str(cfg),
                "--doctor", str(doctor), "--path", str(FIXTURE), "tuple_environment:invalid_main"], capture_output=True, text=True, timeout=20)
            self.assertEqual(result.returncode, 5, result.stderr)
            report = json.loads(doctor.read_text())
            self.assertEqual(report["records_captured"], 1)
            self.assertEqual(report["counters"]["drop_element"], 1)

    def test_real_serve_sampled_steps_runs_and_trace(self):
        with tempfile.TemporaryDirectory() as directory, Serving(Path(directory) / "tuple.db") as server:
            cfg = Path(directory) / "config.yaml"
            cfg.write_text(yaml.safe_dump({"version": 1, "synthetic_only": True, "endpoint": server.endpoint,
                "probes": [{"module": "tuple_environment", "function": "loop"},
                           {"module": "tuple_environment", "function": "Environment.step",
                            "capture": {"return": True}, "sampling": {"every_nth": 10, "max_per_run": 4}}]}))
            doctor = Path(directory) / "doctor.json"
            result = subprocess.run([sys.executable, "-m", "scout_probe", "--config", str(cfg),
                "--doctor", str(doctor), "--path", str(FIXTURE), "tuple_environment:main"], capture_output=True, text=True, timeout=20)
            self.assertEqual(result.returncode, 0, result.stderr)
            report = json.loads(doctor.read_text())
            self.assertEqual(report["probes"][1]["records_captured"], 4)
            self.assertIn("fraction=4/100", result.stderr)
            runs = json.loads(subprocess.check_output([str(server.binary), "--db", str(server.database), "runs", "list"]))
            self.assertEqual(len(runs), 1)
            detail = server.show(runs[0]["run_id"])
            steps = [json.loads(s["attrs"]["signalscout.record.json"]) for s in detail["signals"]
                     if s.get("attrs", {}).get("signalscout.record.kind") == "decision"
                     and s["attrs"].get("signalscout.step.id") == "tuple_environment.Environment.step"]
            self.assertEqual(sorted(s["sampling"]["call_index"] for s in steps), [1, 11, 21, 31])
            for step in steps:
                trace = subprocess.check_output([str(server.binary), "--db", str(server.database), "trace",
                    step["output"]["ref"], "--json", "--from", "2026-01-01T00:00:00Z", "--to", "2027-01-01T00:00:00Z",
                    "--limit", "200"], text=True)
                self.assertIn("tuple_environment.Environment.step", trace)
                self.assertIn('"visited": 1', trace)
