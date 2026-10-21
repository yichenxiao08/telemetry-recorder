"""Synthetic-only runner/method acceptance. No unseen project or test report."""
import importlib
import io
import json
import subprocess
import sys
import tempfile
import unittest
from contextlib import redirect_stderr, redirect_stdout
from pathlib import Path
from unittest.mock import patch

import yaml
import scout_probe
from recorder import Recorder
from scout_probe import Injector
from spike_support import Serving
from opentelemetry.proto.collector.trace.v1.trace_service_pb2 import ExportTraceServiceRequest

ROOT = Path(__file__).resolve().parents[1]
FIXTURE = ROOT / "fixtures/usability"


def drain(recorder):
    items, decisions = [], []
    while not recorder._q.empty():
        item = recorder._q.get_nowait()
        recorder._q.task_done()
        items.append(item)
        for name, _, fields in item["events"]:
            if name == "signalscout.decision":
                decisions.append(json.loads(fields["signalscout.record.json"]))
    return items, decisions


class Methods(unittest.TestCase):
    def setUp(self):
        sys.path.insert(0, str(FIXTURE))
        sys.modules.pop("class_environment", None)
        self.recorder = Recorder(_start_worker=False, capture_fields={"value", "action", "seed", "self.score", "self.steps", "cls.family", "output"})

    def tearDown(self):
        self.recorder.close(0)
        sys.path.remove(str(FIXTURE))
        sys.modules.pop("class_environment", None)

    def attach(self, functions):
        specs = [{"module": "class_environment", "function": name, "capture": capture} for name, capture in functions]
        self.owner = Injector(self.recorder, specs)
        self.owner.install()
        self.addCleanup(self.owner.uninstall)
        return importlib.import_module("class_environment")

    def test_methods_identity_snapshot_and_default_receiver_off(self):
        module = self.attach([("Environment.reset", {"args": ["seed"], "return": True}), ("Environment.step", {"args": ["action"], "self": ["score", "steps"], "return": True}), ("Environment.identity", {"args": ["value"], "cls": ["family"], "return": True}), ("Environment.policy", {"args": ["value"], "return": True}), ("Environment.raising", {})])
        self.assertTrue(all(p["status"] == "attached" for p in self.owner.doctor()["probes"]), self.owner.doctor())
        env = module.Environment()
        state = env.reset(3)
        self.assertIs(state, env.state)
        self.assertIs(env.step(2), state)
        value = [1]
        self.assertIs(module.Environment.identity(value), value)
        self.assertIs(module.Environment.policy(value), value)
        error = ValueError("synthetic error")
        with self.assertRaises(ValueError) as caught:
            env.raising(error)
        self.assertIs(caught.exception, error)
        state.append(99)
        value.append(99)
        _, decisions = drain(self.recorder)
        step = next(d for d in decisions if d["step"]["id"].endswith("Environment.step"))
        inputs = {i["name"]: i for i in step["inputs"]}
        self.assertEqual(inputs["self.score"]["inline"], 3)
        self.assertEqual(inputs["self.steps"]["inline"], 0)
        self.assertEqual(step["output"]["inline"], [2])
        identity = next(d for d in decisions if d["step"]["id"].endswith("Environment.identity"))
        self.assertEqual(identity["output"]["inline"], [1])
        self.assertEqual(next(i["inline"] for i in identity["inputs"] if i["name"] == "value"), [1])
        self.assertFalse(any(i["name"] in ("self", "cls") for d in decisions for i in d["inputs"]))
        self.assertIsNone(self.recorder._active.get())

    def test_super_is_one_outer_decision(self):
        module = self.attach([("Base.step", {"args": ["value"], "return": True}), ("Child.step", {"args": ["value"], "return": True})])
        self.assertEqual(module.Child().step(5), 7)
        _, decisions = drain(self.recorder)
        self.assertEqual([d["step"]["id"] for d in decisions], ["class_environment.Child.step"])
        self.assertEqual(self.recorder.snapshot()["probe_super_suppressed"], 1)

    def test_super_exception_is_not_caught_or_retried_by_probe(self):
        module = self.attach([("Base.raising", {}), ("Child.raising", {})])
        env = module.Child()
        error = ValueError("synthetic super exception")
        with self.assertRaises(ValueError) as caught:
            env.raising(error)
        self.assertIs(caught.exception, error)
        self.assertEqual(env.invocations, 1)
        self.assertEqual(self.owner.methods.get(), ())

    def test_function_static_and_class_method_wire_values_match(self):
        module = self.attach([("decision", {"args": ["value"], "return": True}), ("Environment.policy", {"args": ["value"], "return": True}), ("Environment.identity", {"args": ["value"], "return": True})])
        expected = {"synthetic": [1, 2]}
        module.decision(expected)
        module.Environment.policy(expected)
        module.Environment.identity(expected)
        items, _ = drain(self.recorder)
        request = ExportTraceServiceRequest.FromString(self.recorder._encode(items))
        signatures = []
        for span in request.resource_spans[0].scope_spans[0].spans:
            for event in span.events:
                if event.name == "signalscout.decision":
                    envelope = json.loads(next(a.value.string_value for a in event.attributes if a.key == "signalscout.record.json"))
                    signatures.append(([(v["name"], v["capture"], v.get("inline")) for v in envelope["inputs"]], envelope["output"]["inline"], envelope["chosen"], envelope["alternatives"], envelope["rule"]))
        self.assertEqual(len(signatures), 3)
        self.assertEqual(signatures, [signatures[0]] * 3)

    def test_unsupported_specific_reasons_without_property_execution(self):
        self.attach([("Environment.unsafe", {}), ("Environment.generating", {}), ("Environment.asynchronous", {}), ("Environment.step", {"self": ["unsafe"]}), ("Child.missing", {}), ("Unsupported.opaque", {}), ("CustomAccess.step", {}), ("Environment.Inner.step", {})])
        report = self.owner.doctor()
        self.assertTrue(all(p["status"] == "unsupported" for p in report["probes"]))
        self.assertIn("property", report["probes"][0]["reason"])
        self.assertIn("generator/async", report["probes"][1]["reason"])
        self.assertIn("nested", report["probes"][-1]["reason"])
        self.assertIn("descriptor", report["probes"][5]["reason"])
        self.assertIn("dunder", report["probes"][6]["reason"])


class Runner(unittest.TestCase):
    def config(self, directory, function="Environment.step", endpoint="http://127.0.0.1:1"):
        path = Path(directory) / "signalscout.yaml"
        path.write_text(yaml.safe_dump({"version": 1, "synthetic_only": True, "endpoint": endpoint,
            "probes": [{"module": "class_environment", "function": function, "capture": {"args": ["action"] if function == "Environment.step" else [], "return": True}}]}))
        return path

    def command(self, path, doctor, entry, *target_args):
        return [sys.executable, "-m", "scout_probe", "--config", str(path), "--doctor", str(doctor), "--path", str(FIXTURE), entry, *target_args]

    def test_real_binary_strict_argparse_and_runs_list(self):
        with tempfile.TemporaryDirectory() as directory, Serving(Path(directory) / "fresh.db") as server:
            cfg = self.config(directory, endpoint=server.endpoint)
            doctor = Path(directory) / "doctor.json"
            result = subprocess.run(self.command(cfg, doctor, "class_environment:main", "--", "run", "--episodes", "2", "--seed", "7"), capture_output=True, text=True, timeout=20)
            self.assertEqual(result.returncode, 0, result.stderr)
            payload = json.loads(result.stdout)
            self.assertEqual(payload["argv"], [str(FIXTURE / "class_environment.py"), "run", "--episodes", "2", "--seed", "7"])
            self.assertEqual(payload["rewards"], [13, 13])
            report = json.loads(doctor.read_text())
            self.assertEqual(report["records_captured"], 6)
            self.assertEqual(report["runs_created"], 6)
            runs = json.loads(subprocess.check_output([str(server.binary), "--db", str(server.database), "runs", "list"]))
            self.assertEqual(len(runs), 6)
            self.assertIn("records captured: 6", result.stderr)

    def test_direct_argv_zero_capture_preflight_and_restoration(self):
        with tempfile.TemporaryDirectory() as directory, Serving(Path(directory) / "argv.db") as server:
            cfg = self.config(directory, "snapshot_argv", server.endpoint)
            doctor = Path(directory) / "doctor.json"
            preflight = subprocess.run(self.command(cfg, doctor, "class_environment:main", "--preflight", "--", "run", "--episodes", "2"), capture_output=True, text=True, timeout=10)
            self.assertEqual(preflight.returncode, 0, preflight.stderr)
            self.assertEqual(preflight.stdout, "")
            self.assertEqual(json.loads(doctor.read_text())["runs_created"], 0)
            result = subprocess.run(self.command(cfg, doctor, "class_environment:unused_main"), capture_output=True, text=True, timeout=10)
            self.assertEqual(result.returncode, 4, result.stderr)
            self.assertIn("zero", result.stderr.lower())
            result = subprocess.run(self.command(cfg, doctor, "class_environment:zero_exit_main"), capture_output=True, text=True, timeout=10)
            self.assertEqual(result.returncode, 4, result.stderr)
            previous_argv, previous_path = sys.argv, list(sys.path)
            sys.modules.pop("class_environment", None)
            stderr, stdout = io.StringIO(), io.StringIO()
            with patch.object(sys, "argv", self.command(cfg, doctor, "class_environment:argv_main", "--", "--target", "synthetic")[2:]), redirect_stdout(stdout), redirect_stderr(stderr):
                runner_argv = sys.argv
                code = scout_probe.main()
                self.assertIs(sys.argv, runner_argv)
            self.assertEqual(code, 0)
            self.assertEqual(json.loads(stdout.getvalue()), [str(FIXTURE / "class_environment.py"), "--target", "synthetic"])
            self.assertIs(sys.argv, previous_argv)
            self.assertEqual(sys.path, previous_path)
            sys.modules.pop("class_environment", None)
            with patch.object(sys, "argv", self.command(cfg, doctor, "class_environment:main", "--", "run")[2:]), redirect_stdout(io.StringIO()), redirect_stderr(io.StringIO()):
                runner_argv = sys.argv
                with self.assertRaises(SystemExit) as failure:
                    scout_probe.main()
                self.assertEqual(failure.exception.code, 2)
                self.assertIs(sys.argv, runner_argv)
            self.assertEqual(sys.path, previous_path)
            sys.modules.pop("class_environment", None)

    def test_preflight_reports_unsupported_without_target_execution(self):
        with tempfile.TemporaryDirectory() as directory:
            cfg = self.config(directory, "Environment.generating")
            doctor = Path(directory) / "doctor.json"
            result = subprocess.run(self.command(cfg, doctor, "class_environment:main", "--preflight"), capture_output=True, text=True, timeout=10)
            self.assertEqual(result.returncode, 3, result.stderr)
            self.assertEqual(result.stdout, "")
            self.assertIn("generator/async", result.stderr)
            report = json.loads(doctor.read_text())
            self.assertFalse(report["target_ran"])
            self.assertEqual(report["runs_created"], 0)
