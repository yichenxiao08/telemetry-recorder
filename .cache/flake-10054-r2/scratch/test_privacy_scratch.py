"""Synthetic P0 producer privacy; no real content."""
import json
import unittest
import tempfile
import urllib.request
import urllib.error
from pathlib import Path

from capture_policy import Policy
from recorder import Recorder
from scout_probe import Injector, Probe
from spike_support import Serving


def synthetic_step(review):
    return {"lane": "left", "review": review}


def synthetic_raising(error):
    raise error


class SyntheticPrivate:
    def __init__(self):
        self.score = 3
        self.review = "person@example.invalid"
        self.result = {"lane": "left", "review": self.review}

    def step(self, action):
        self.score += action
        return self.result


class Privacy(unittest.TestCase):
    def test_method_fields_obey_profile_policy_without_mutating_host(self):
        policy = Policy({"version": 1, "profiles": {"synthetic": {"fields": {
            "inputs.self.score": {"kind": "number"}, "output.lane": {"kind": "enum", "values": ["left"]}}}}})
        recorder = Recorder(_start_worker=False, capture_fields={"action", "self.score", "self.review", "output"},
                            capture_policy=policy, capture_profile="synthetic")
        spec = {"module": __name__, "function": "SyntheticPrivate.step", "capture": {"args": ["action"], "self": ["score", "review"], "return": True}}
        owner = Injector(recorder, [spec])
        owner.install()
        try:
            instance = SyntheticPrivate()
            self.assertIs(instance.step(2), instance.result)
            self.assertEqual(instance.review, "person@example.invalid")
            self.assertEqual(instance.score, 5)
            items = []
            while not recorder._q.empty():
                items.append(recorder._q.get_nowait())
            stored = json.dumps(items)
            self.assertNotIn("example.invalid", stored)
            self.assertIn("self.score", stored)
            self.assertEqual(owner.doctor()["probes"][0]["status"], "attached")
        finally:
            owner.uninstall()
            recorder.close(.01)

    def test_exception_identity_and_message_not_buffered(self):
        policy = Policy({"version": 1, "profiles": {"synthetic": {"fields": {}}}})
        recorder = Recorder(_start_worker=False, capture_policy=policy, capture_profile="synthetic")
        owner = Injector(recorder, [])
        wrapped = Probe(owner, synthetic_raising, {"module": __name__, "function": "synthetic_raising"}).wrap()
        error = ValueError("Bearer syntheticExceptionSecret")
        with self.assertRaises(ValueError) as caught:
            wrapped(error)
        self.assertIs(caught.exception, error)
        items = []
        while not recorder._q.empty():
            items.append(recorder._q.get_nowait())
        self.assertNotIn("syntheticExceptionSecret", json.dumps(items))
        self.assertIsNone(recorder._active.get())
        recorder.close(.01)

    def test_hmac_and_artifact_policy(self):
        policy = Policy({"version": 1, "key_version": "test-v1", "profiles": {"synthetic": {"fields": {
            "email": {"kind": "hmac", "value_class": "email", "equality": True},
            "other": {"kind": "hmac", "value_class": "email", "equality": True},
            "artifact": {"kind": "artifact-sha256", "values": ["synthetic artifact"], "nonpersonal": True, "hash_basis": "redacted"}}}}}, b"k" * 32)
        token, ok = policy.value("synthetic", "email", "person@example.invalid")
        self.assertTrue(ok)
        self.assertNotIn("example.invalid", token)
        self.assertNotEqual(token, policy.value("synthetic", "other", "person@example.invalid")[0])
        self.assertEqual((token, True), policy.value("synthetic", "email", token))
        self.assertFalse(policy.value("synthetic", "email", "opaque credential")[1])
        artifact, ok = policy.value("synthetic", "artifact", "synthetic artifact")
        self.assertTrue(ok)
        self.assertEqual("redacted", artifact["hash_basis"])
        self.assertEqual((artifact, True), policy.value("synthetic", "artifact", artifact))

    def test_real_probe_intake_and_log_raw_copies(self):
        secrets = ["sk-syntheticSECRET123456789", "person@example.invalid", "4111111111111111",
                   "Bearer syntheticAuth123", "synthetic_sql_literal", "https://artifact.invalid/?token=planted"]
        policy_data = {"version": 1, "profiles": {"synthetic": {"fields": {"output.lane": {"kind": "enum", "values": ["left", "right"]}}}}}
        with tempfile.TemporaryDirectory() as directory:
            directory = Path(directory)
            policy_file = directory / "policy.json"
            policy_file.write_text(json.dumps(policy_data), encoding="utf-8")
            with Serving(directory / "fresh.db", ["--capture-policy", str(policy_file), "--capture-profile", "synthetic"]) as server:
                recorder = Recorder(server.endpoint, capture_fields={"review", "output"},
                                    capture_policy=Policy(policy_data), capture_profile="synthetic",
                                    resource_attributes={"private": secrets})
                owner = Injector(recorder, [])
                wrapped = Probe(owner, synthetic_step, {"module": __name__, "function": "synthetic_step", "capture": {"args": ["review"], "return": True}}).wrap()
                run_id = "de" * 16
                with recorder.run("synthetic-test", run_id=run_id, attributes={"exception.message": " ".join(secrets)}):
                    self.assertEqual(wrapped(secrets)["review"], secrets)
                self.assertTrue(recorder.close(5))
                log = {"resourceLogs": [{"resource": {"attributes": [{"key": "Authorization", "value": {"stringValue": secrets[3]}}]},
                    "scopeLogs": [{"scope": {"name": secrets[1]}, "logRecords": [{"traceId": run_id, "timeUnixNano": "1767225600000000000",
                    "body": {"stringValue": json.dumps({"nested": secrets})}, "attributes": [{"key": "split.one", "value": {"stringValue": "splitperson@"}}, {"key": "split.two", "value": {"stringValue": "example.invalid"}}]}]}]}]}
                request = urllib.request.Request(server.endpoint + "/v1/logs", json.dumps(log).encode(), {"Content-Type": "application/json"})
                with urllib.request.urlopen(request, timeout=5) as response:
                    self.assertEqual(response.status, 200)
                detail = server.show(run_id)
                stored = json.dumps(detail)
                self.assertGreaterEqual(len(detail["signals"]), 4)
                for secret in [*secrets, "splitperson@example.invalid"]:
                    self.assertNotIn(secret, stored)
                self.assertIn('"lane"', stored.replace('\\"', '"'))
                self.assertGreater(recorder.snapshot()["privacy_checked_batches"], 0)
                blocked = urllib.request.Request(server.endpoint + "/api/annotations", b"", {"Content-Type": "application/json"})
                with self.assertRaises(urllib.error.HTTPError) as caught:
                    urllib.request.urlopen(blocked, timeout=5)
                self.assertEqual(caught.exception.code, 403)

    def test_before_queue_redaction_and_default_off(self):
        policy = Policy({"version": 1, "profiles": {"synthetic": {"fields": {"inputs.frame": {"kind": "number"}}}}})
        recorder = Recorder(_start_worker=False, capture_fields={"frame", "review", "output"},
                            capture_policy=policy, capture_profile="synthetic",
                            service="person@example.invalid",
                            resource_attributes={"authorization": "Bearer synthetic_secret"})
        with recorder.run("person@example.invalid"):
            self.assertTrue(recorder.record_decision("Bearer synthetic_secret", output={"ref": "https://x.invalid/?token=synthetic", "value": "4111111111111111"},
                inputs={"frame": {"value": 10}, "review": {"value": {"excerpt": "person@example.invalid"}}},
                chosen={"option": "sk-syntheticSECRET123456789"}, alternatives=[], rule={"sql": "SELECT 'synthetic_literal'"}))
        items = []
        while not recorder._q.empty():
            items.append(recorder._q.get_nowait())
        encoded = json.dumps(items) + json.dumps(recorder.resource_attributes) + recorder.service
        for secret in ["example.invalid", "Bearer", "4111111111111111", "syntheticSECRET", "synthetic_literal", "https:"]:
            self.assertNotIn(secret, encoded)
        self.assertIn('"inline":10', encoded.replace('\\"', '"').replace(' ', ''))
        recorder.close(.01)

    def test_unknown_versions_and_invalid_policy(self):
        policy = Policy({"version": 1, "profiles": {"synthetic": {"fields": {}}}})
        self.assertNotIn("secret", policy.envelope("synthetic", '{"version":2,"secret":"planted"}'))
        for raw in [{"version":2,"profiles":{}}, {"version":1,"profiles":{"x":{"fields":{"password":{"kind":"enum","values":["x"]}}}}}]:
            with self.assertRaises(ValueError):
                Policy(raw)
