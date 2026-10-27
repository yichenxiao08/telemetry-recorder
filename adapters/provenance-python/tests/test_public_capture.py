import hashlib
import json
import subprocess
import sys
import tempfile
import threading
import unittest
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import yaml
from capture_policy import Policy
from public_capture import prepare
from scout_probe import load_config
from recorder import Recorder
from spike_support import Serving


RAW = b'{"version":1,"profiles":{"public":{"fields":{"inputs.speed":{"kind":"number"}}}}}'


class PublicCaptureTests(unittest.TestCase):
    def test_explicit_config_and_default_gate(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "config.yaml"
            cfg = {"version": 1, "synthetic_only": False, "public_data_experiment": True,
                   "capture_policy": "policy.json", "capture_profile": "public"}
            path.write_text(yaml.safe_dump(cfg))
            self.assertTrue(load_config(path)["public_data_experiment"])
            for changes in ({"synthetic_only": True}, {"capture_policy": ""},
                            {"capture_profile": ""}, {"auto_otel": True},
                            {"public_data_experiment": False}):
                path.write_text(yaml.safe_dump({**cfg, **changes}))
                with self.assertRaises(ValueError):
                    load_config(path)
            path.write_text("version: 1\nsynthetic_only: true\n")
            self.assertTrue(load_config(path)["synthetic_only"])

    def test_policy_endpoint_and_mismatch(self):
        expected = {"mode": "public-data-experiment", "profile": "public", "sha256": hashlib.sha256(RAW).hexdigest()}
        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(200)
                self.end_headers()
                self.wfile.write(json.dumps(expected).encode())
            def log_message(self, *args):
                pass
        server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        self.addCleanup(server.server_close)
        self.addCleanup(server.shutdown)
        endpoint = "http://127.0.0.1:" + str(server.server_port)
        policy = Policy(json.loads(RAW))
        digest, _ = prepare(endpoint, policy, "public", RAW)
        self.assertEqual(digest, expected["sha256"])
        expected["sha256"] = "wrong"
        with self.assertRaisesRegex(ValueError, "mismatch"):
            prepare(endpoint, policy, "public", RAW)
        for target in ("http://example.com:1234", "http://localhost:1234", "http://127.0.0.1:1234/path", "http://user@127.0.0.1:1234"):
            with self.assertRaises(ValueError):
                prepare(target, policy, "public", RAW, check_server=False)
        invalid = Policy({"version": 1, "profiles": {"public": {"fields": {"inputs.x": {"kind": "boolean"}}}}})
        with self.assertRaisesRegex(ValueError, "numeric"):
            prepare(endpoint, invalid, "public", RAW, check_server=False)

    def test_strict_intake_with_fresh_store_and_numeric_capture(self):
        with tempfile.TemporaryDirectory() as tmp:
            tmp = Path(tmp)
            policy_file = tmp / "policy.json"
            policy_file.write_bytes(RAW)
            database = tmp / "fresh.db"
            with Serving(database, ["--capture-policy", str(policy_file), "--capture-profile", "public", "--public-data-experiment"]) as server:
                policy = Policy(json.loads(RAW))
                digest, opener = prepare(server.endpoint, policy, "public", RAW)
                recorder = Recorder(server.endpoint, capture_fields={"speed", "secret"},
                                    capture_policy=policy, capture_profile="public",
                                    public_digest=digest, public_opener=opener)
                run_id = "ab" * 16
                with recorder.run("private synthetic text", run_id=run_id):
                    self.assertTrue(recorder.record_decision("step", inputs={
                        "speed": {"value": 12}, "secret": {"value": "synthetic-secret"}},
                        output={"ref": "value:test"}, chosen={"option": "synthetic"}, alternatives=[], rule={}))
                self.assertTrue(recorder.close(5))
                detail = json.dumps(server.show(run_id))
                self.assertIn("speed", detail)
                self.assertNotIn("synthetic-secret", detail)
                self.assertNotIn("private synthetic text", detail)
                request = urllib.request.Request(server.endpoint + "/v1/traces", b"", method="POST")
                with self.assertRaises(urllib.error.HTTPError) as caught:
                    urllib.request.urlopen(request)
                self.assertEqual(caught.exception.code, 403)
                with self.assertRaisesRegex(ValueError, "mismatch"):
                    prepare(server.endpoint, policy, "public", RAW + b" ")

    def test_real_probe_public_mode(self):
        with tempfile.TemporaryDirectory() as tmp:
            tmp = Path(tmp)
            policy_file = tmp / "policy.json"
            policy_file.write_bytes(RAW)
            (tmp / "target.py").write_text("def step(speed):\n    return speed + 1\n\ndef main():\n    print(step(12))\n")
            config = tmp / "config.yaml"
            doctor = tmp / "doctor.json"
            with Serving(tmp / "fresh.db", ["--capture-policy", str(policy_file), "--capture-profile", "public", "--public-data-experiment"]) as server:
                config.write_text(yaml.safe_dump({"version": 1, "synthetic_only": False,
                    "public_data_experiment": True, "endpoint": server.endpoint,
                    "capture_policy": str(policy_file), "capture_profile": "public",
                    "probes": [{"module": "target", "function": "step", "capture": {"args": ["speed"]}}]}))
                result = subprocess.run([sys.executable, "-m", "scout_probe", "--config", str(config),
                    "--doctor", str(doctor), "--path", str(tmp), "--drain-timeout", "5", "target:main"],
                    capture_output=True, text=True, timeout=15)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(result.stdout.strip(), "13")
                report = json.loads(doctor.read_text())
                self.assertIn("public-data experiment", report["privacy"])
                self.assertEqual(report["records_captured"], 1)
                self.assertEqual(report["counters"]["failed"], 0)
                self.assertEqual(report["counters"]["unknown"], 0)


if __name__ == "__main__":
    unittest.main()
