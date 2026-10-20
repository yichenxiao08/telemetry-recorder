import importlib
import inspect
import json
import socket
import subprocess
import sys
import tempfile
import threading
import time
import unittest
import urllib.request
from pathlib import Path
from unittest.mock import patch

from recorder import Recorder
from scout_probe import Injector, Probe, load_config


def identity(a):
    return a


def mutate(a):
    a.append(9)
    return a


def raising(a):
    raise a


def generator(a):
    yield a


async def coroutine(a):
    return a


def decision(recorder, value=1):
    return recorder.record_decision("synthetic.step", output={"ref": "value:out", "value": value},
        inputs={"a": {"value": value}}, chosen={"option": "returned"}, alternatives=[],
        rule={"id": "unobserved"})


def wrapped(recorder, function, capture=True):
    spec = {"module": __name__, "function": function.__name__,
            "capture": {"args": ["a"] if capture else [], "return": capture}}
    owner = Injector(recorder, [spec])
    return Probe(owner, function, spec).wrap()


class Safety(unittest.TestCase):
    def make(self, **kwargs):
        r = Recorder(_start_worker=False, **kwargs)
        self.addCleanup(lambda: r.close(0))
        return r

    def test_overflow_and_bounded_memory(self):
        r = self.make(capacity=2)
        with r.run("synthetic"):
            for _ in range(1000):
                decision(r)
        counts = r.snapshot()
        self.assertEqual(counts["accepted"], 2)
        self.assertEqual(counts["drop_full"], 999)  # 1000 decisions + root
        self.assertEqual(counts["queue_size"], 2)
        self.assertFalse(r.flush(.001))

    def test_default_capture_off_and_snapshot_before_mutation(self):
        r = self.make(capture_fields=("a", "output"))
        f = wrapped(r, mutate)
        original = [1]
        result = f(original)
        self.assertIs(result, original)
        self.assertEqual(original, [1, 9])
        item = r._q.get_nowait()
        envelope = json.loads(item["events"][0][2]["signalscout.record.json"])
        self.assertEqual(envelope["inputs"][0]["inline"], [1])
        self.assertEqual(envelope["output"]["inline"], [1, 9])
        default = self.make()
        wrapped(default, identity)("SYNTHETIC_PRIVATE_SENTINEL")
        payload = json.dumps(default._q.get_nowait())
        self.assertNotIn("SYNTHETIC_PRIVATE_SENTINEL", payload)

    def test_invalid_values_do_not_raise_or_call_user_hooks(self):
        class Hostile:
            def __str__(self):
                raise AssertionError("must not stringify")
            def __iter__(self):
                raise AssertionError("must not iterate")
        r = self.make(capture_fields=("a", "output"))
        for value in [Hostile(), float("nan"), "x"*1000, [[[list(range(32))]*32]*32]]:
            with r.run("synthetic"):
                self.assertFalse(decision(r, value))
        with r.run("synthetic"):
            self.assertFalse(r.record_decision("x", output={"ref":"x"}, inputs={},
                chosen={"option":"x"}, alternatives=[], rule={}, config_refs=Hostile()))
        self.assertGreaterEqual(r.snapshot()["drop_invalid"], 5)

    def test_original_exception_and_context_cleanup(self):
        r = self.make(capture_fields=("a",))
        error = RuntimeError("synthetic host error")
        try:
            wrapped(r, raising)(error)
        except RuntimeError as actual:
            self.assertIs(actual, error)
        else:
            self.fail("host exception suppressed")
        self.assertIsNone(r._active.get())
        self.assertFalse(decision(r))
        with r.run("outer") as outer:
            with r.run("inner"):
                pass
            self.assertIs(r._active.get(), outer)
        self.assertIsNone(r._active.get())

    def test_exit_has_no_flush_join_or_network_and_full_queue_no_wait(self):
        r = self.make(capacity=1)
        with patch.object(r, "flush", side_effect=AssertionError("flush on host")), \
             patch.object(r, "_send", side_effect=AssertionError("network on host")), \
             patch.object(r._worker, "join", side_effect=AssertionError("join on host")):
            start = time.monotonic()
            wrapped(r, identity)(1)
            self.assertLess(time.monotonic()-start, .1)
        self.assertEqual(r.snapshot()["accepted"], 1)
        self.assertEqual(r.snapshot()["drop_full"], 1)

    def test_server_down_cooldown_fast_path(self):
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", 0))
            port = listener.getsockname()[1]
        r = Recorder(f"http://127.0.0.1:{port}", timeout=.1, cooldown=5)
        self.addCleanup(lambda: r.close(.2))
        wrapped(r, identity)(1)
        self.assertTrue(r.flush(1))
        self.assertGreater(r.snapshot().get("export_failed", 0), 0)
        requests = r.snapshot()["network_requests"]
        for _ in range(100):
            wrapped(r, identity)(1)
        self.assertTrue(r.flush(1))
        self.assertEqual(r.snapshot()["network_requests"], requests)
        self.assertGreater(r.snapshot()["drop_cooldown"], 0)

    def test_unsupported_and_import_hook_doctor(self):
        r = self.make()
        for f in (generator, coroutine, len):
            with self.assertRaises(ValueError):
                wrapped(r, f)
        with tempfile.TemporaryDirectory() as root:
            Path(root, "synthetic_hook_target.py").write_text(
                "def target(a):\n    return a + 1\n\ndef lazy(a):\n    yield a\n")
            specs = [{"module":"synthetic_hook_target","function":"target"},
                     {"module":"synthetic_hook_target","function":"lazy"},
                     {"module":"never_imported","function":"x"}]
            owner = Injector(r, specs)
            sys.path.insert(0, root)
            owner.install()
            try:
                module = importlib.import_module("synthetic_hook_target")
                self.assertEqual(module.target(3), 4)
                self.assertEqual(str(inspect.signature(module.target)), "(a)")
                statuses = [v["status"] for v in owner.doctor()["probes"]]
                self.assertEqual(statuses, ["attached", "unsupported", "failed"])
            finally:
                owner.uninstall()
                sys.path.remove(root)
                sys.modules.pop("synthetic_hook_target", None)

    def test_config_capture_off_and_unknown_keys_rejected(self):
        with tempfile.TemporaryDirectory() as root:
            path = Path(root, "signalscout.yaml")
            path.write_text("version: 1\nsynthetic_only: true\nprobes:\n - module: x\n   function: y\n")
            cfg = load_config(path)
            self.assertNotIn("capture", cfg["probes"][0])
            path.write_text("version: 1\nsynthetic_only: true\nunknown: 3\n")
            with self.assertRaises(ValueError):
                load_config(path)


class Intake(unittest.TestCase):
    def test_real_intake_decodes_probed_otlp_and_lineage(self):
        repo = Path(__file__).resolve().parents[3]
        binary = repo/"bin"/"signalscout.exe"
        self.assertTrue(binary.exists(), "run task build before Python tests")
        with tempfile.TemporaryDirectory() as root:
            db = Path(root, "synthetic.db")
            with socket.socket() as listener:
                listener.bind(("127.0.0.1", 0))
                port = listener.getsockname()[1]
            endpoint = f"http://127.0.0.1:{port}"
            process = subprocess.Popen([str(binary), "--db", str(db), "serve", "--listen",
                f"127.0.0.1:{port}", "--retention", "0"], stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL)
            try:
                deadline = time.monotonic()+10
                while True:
                    try:
                        with urllib.request.urlopen(endpoint+"/runs", timeout=.1):
                            break
                    except OSError:
                        if time.monotonic() > deadline or process.poll() is not None:
                            self.fail("real intake did not start")
                        time.sleep(.02)
                r = Recorder(endpoint, capture_fields=("a","output"), service="synthetic.safety")
                try:
                    trace_id = "0000000000000000000000000000aa01"
                    with r.run("synthetic.safe", run_id=trace_id):
                        self.assertEqual(wrapped(r, identity)(7), 7)
                    self.assertTrue(r.flush(2))
                    self.assertEqual(r.snapshot().get("export_failed", 0), 0)
                    raw = subprocess.check_output([str(binary), "--db", str(db),
                                                   "runs", "show", trace_id])
                    detail = json.loads(raw)
                    events = [v for v in detail["signals"] if v["name"] == "signalscout.decision"]
                    self.assertEqual(len(events), 1)
                    envelope = json.loads(events[0]["attrs"]["signalscout.record.json"])
                    self.assertEqual(envelope["inputs"][0]["inline"], 7)
                    self.assertEqual(envelope["output"]["inline"], 7)
                    edges = [v for v in detail["signals"] if v["name"] == "signalscout.lineage"]
                    self.assertEqual(len(edges), 1)
                    self.assertEqual(json.loads(edges[0]["attrs"]["signalscout.record.json"])["output_ref"],
                                     envelope["output"]["ref"])
                finally:
                    r.close(1)
            finally:
                process.terminate()
                process.wait(timeout=5)


if __name__ == "__main__":
    unittest.main()

class BlockedExporter(unittest.TestCase):
    def test_blocked_worker_does_not_block_host_or_context_exit(self):
        entered, release = threading.Event(), threading.Event()
        r = Recorder(capacity=1, capture_fields=("a","output"), _start_worker=False)
        def send(batch):
            self.assertNotEqual(threading.get_ident(), threading.main_thread().ident)
            entered.set()
            release.wait(2)
        r._send = send
        r._worker.start()
        try:
            with r.run("synthetic"):
                decision(r)
                self.assertTrue(entered.wait(1))
                start = time.monotonic()
                for _ in range(50):
                    wrapped(r, identity)(1)
                self.assertLess(time.monotonic()-start, .1)
                self.assertFalse(r.flush(.01))
                self.assertLessEqual(r.snapshot()["queue_size"], 1)
                self.assertGreater(r.snapshot()["drop_full"], 0)
        finally:
            release.set()
            r.close(2)

class FrameReference(unittest.TestCase):
    def test_zero_source_changes_and_all_envelope_fields_match_go(self):
        import hashlib
        fixture = Path(__file__).resolve().parents[1]/"fixtures"
        repo = Path(__file__).resolve().parents[3]
        source = fixture/"frame_target.py"
        before = source.read_bytes()
        expected = json.loads((repo/".cache/provenance/frame-go/facts.json").read_text())
        from google.protobuf.json_format import Parse
        from opentelemetry.proto.collector.trace.v1.trace_service_pb2 import ExportTraceServiceRequest
        req=Parse((repo/".cache/provenance/frame-go/traces.json").read_text(),
                  ExportTraceServiceRequest())
        go_decisions=[]
        go_edges=[]
        for span in req.resource_spans[0].scope_spans[0].spans:
            for e in span.events:
                body=next(a.value.string_value for a in e.attributes if a.key=="signalscout.record.json")
                (go_decisions if e.name=="signalscout.decision" else go_edges).append(json.loads(body))
        r=Recorder(_start_worker=False,capacity=256,
                   capture_fields=("gap","previous_lane","previous_frame","frame","output"))
        owner=Injector(r,[{"module":"frame_target","function":"choose_lane",
            "capture":{"args":["gap","previous_lane","previous_frame","frame"],"return":True}}])
        sys.path.insert(0,str(fixture))
        owner.install()
        try:
            module=importlib.import_module("frame_target")
            results=module.frames()
            self.assertEqual(results,[f["output"] for f in expected["facts"]])
            helper_decisions,helper_edges=[],[]
            while not r._q.empty():
                item=r._q.get_nowait()
                for name,_,values in item["events"]:
                    (helper_decisions if name=="signalscout.decision" else helper_edges).append(
                        json.loads(values["signalscout.record.json"]))
                r._q.task_done()
            self.assertEqual(len(helper_decisions),60)
            self.assertEqual(len(helper_edges),240)
            def normalize(value, run_id):
                return json.loads(json.dumps(value).replace(run_id,"RUN"))
            for a,b in zip(helper_decisions,go_decisions):
                self.assertEqual(normalize(a,a["run_id"]),normalize(b,b["run_id"]))
            for a,b in zip(helper_edges,go_edges):
                self.assertEqual(normalize(a,a["run_id"]),normalize(b,b["run_id"]))
            self.assertEqual(source.read_bytes(),before)
            self.assertEqual(hashlib.sha256(source.read_bytes()).hexdigest(),
                             hashlib.sha256(before).hexdigest())
        finally:
            owner.uninstall()
            sys.path.remove(str(fixture))
            sys.modules.pop("frame_target",None)
            r.close(0)

class ExtraSafety(unittest.TestCase):
    def test_capture_failure_never_changes_host_behavior(self):
        r=Recorder(_start_worker=False)
        try:
            value=object()
            with patch.object(r,"record_decision",side_effect=ValueError("synthetic recorder fault")):
                self.assertIs(wrapped(r,identity)(value),value)
                error=RuntimeError("synthetic")
                with self.assertRaises(RuntimeError) as actual:
                    wrapped(r,raising)(error)
                self.assertIs(actual.exception,error)
            self.assertIsNone(r._active.get())
        finally:
            r.close(0)

    def test_fixed_cost_payload(self):
        fixture=Path(__file__).resolve().parents[1]/"fixtures"
        sys.path.insert(0,str(fixture))
        try:
            from benchmark_target import calculate
            r=Recorder(_start_worker=False,
                capture_fields=("gap","previous_lane","previous_frame","frame","output"))
            spec={"module":"benchmark_target","function":"calculate",
                  "capture":{"args":["gap","previous_lane","previous_frame","frame"],"return":True}}
            f=Probe(Injector(r,[spec]),calculate,spec).wrap()
            f(.51,0,1,2)
            item=r._q.get_nowait()
            raw=item["events"][0][2]["signalscout.record.json"]
            d=json.loads(raw)
            self.assertEqual(len(d["inputs"]),4)
            self.assertEqual(len(d["output"]["inline"]["alternatives"]),2)
            self.assertLessEqual(len(raw.encode()),2048)
            r.close(0)
        finally:
            sys.path.remove(str(fixture))


def http_and_log(a):
    import logging
    import requests
    status=requests.get(a,timeout=1).status_code
    record=logging.makeLogRecord({"msg":"synthetic","levelno":20})
    return {"status":status,"trace":record.otelTraceID}


class StandardOtel(unittest.TestCase):
    def test_installed_http_and_logging_integrations(self):
        from http.server import BaseHTTPRequestHandler,ThreadingHTTPServer
        from scout_probe import standard_otel
        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(200)
                self.end_headers()
            def log_message(self,*args):
                pass
        server=ThreadingHTTPServer(("127.0.0.1",0),Handler)
        thread=threading.Thread(target=server.serve_forever,daemon=True)
        thread.start()
        r=Recorder(_start_worker=False)
        try:
            report=standard_otel(r)
            self.assertIn("standard requests",report["http"])
            self.assertIn("standard context correlation",report["logging"])
            result=wrapped(r,http_and_log,False)(f"http://127.0.0.1:{server.server_port}")
            self.assertEqual(result["status"],200)
            self.assertNotEqual(int(result["trace"],16),0)
            items=list(r._q.queue)
            self.assertTrue(any(i["attrs"].get("http.status_code")==200 for i in items))
        finally:
            from opentelemetry.instrumentation.requests import RequestsInstrumentor
            from opentelemetry.instrumentation.logging import LoggingInstrumentor
            RequestsInstrumentor().uninstrument()
            LoggingInstrumentor().uninstrument()
            server.shutdown()
            server.server_close()
            r.close(0)

class ReloadBound(unittest.TestCase):
    def test_reload_is_reported_and_doctor_memory_stays_bounded(self):
        r=Recorder(_start_worker=False)
        with tempfile.TemporaryDirectory() as root:
            Path(root,"synthetic_reload_target.py").write_text("def f(a):\n    return a\n")
            sys.path.insert(0,root)
            owner=Injector(r,[{"module":"synthetic_reload_target","function":"f"}])
            owner.install()
            try:
                module=importlib.import_module("synthetic_reload_target")
                self.assertTrue(hasattr(module.f,"__signalscout_probe__"))
                for _ in range(100):
                    importlib.reload(module)
                self.assertEqual(len(owner.results),1)
                self.assertEqual(owner.doctor()["probes"][0]["status"],"unsupported")
                self.assertFalse(hasattr(module.f,"__signalscout_probe__"))
                self.assertEqual(module.f(5),5)
            finally:
                owner.uninstall()
                sys.path.remove(root)
                sys.modules.pop("synthetic_reload_target",None)
                r.close(0)
