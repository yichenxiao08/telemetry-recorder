"""Development/toy tests for opt-in completeness manifests and the drain timeout.
No reserved seeds, lane-02 artifacts or evaluation flags."""
import io
import json
import subprocess
import sys
import tempfile
import threading
import time
import unittest
from contextlib import redirect_stderr
from pathlib import Path
from unittest.mock import patch

import yaml

import capture_manifest as cm
import recorder as recorder_module
import scout_probe
from recorder import MAX_DRAIN_SECONDS, Recorder, check_drain_seconds
from scout_probe import Injector, Probe
from spike_support import Serving
from test_export_diagnostics import ITEM
from test_queue_bound import identity as accounting_identity, span


def identity_function(a):
    return a


def boom(a):
    raise KeyError(a)


def probed(recorder, sampling=None):
    spec = {"module": __name__, "function": "identity_function", "capture": {"args": ["a"], "return": True}}
    if sampling:
        spec["sampling"] = sampling
    return Probe(Injector(recorder, [spec]), identity_function, spec).wrap()


def manifests_of(batches):
    found = {}
    for batch in batches:
        for item in batch:
            if item["name"] == cm.MANIFEST_NAME:
                found[item["trace"]] = json.loads(item["attrs"]["signalscout.manifest.json"])
    return found


def scope_of(batches):
    return [json.loads(item["attrs"]["signalscout.manifest.json"])
            for batch in batches for item in batch if item["name"] == cm.SCOPE_NAME]


class Sink:
    """Stands in for the network: records every batch handed to _send."""
    def __init__(self, fail=None):
        self.batches, self.lock, self.fail = [], threading.Lock(), fail

    def __call__(self, batch):
        if self.fail is not None:
            self.fail(batch)
        with self.lock:
            self.batches.append(list(batch))


class ManifestCase(unittest.TestCase):
    def make(self, **kw):
        r = Recorder(manifest=True, **kw)
        self.addCleanup(lambda: r.close(0))
        return r


class Conservation(ManifestCase):
    def check_buckets(self, manifest):
        for key, b in manifest["buckets"].items():
            with self.subTest(bucket=key):
                self.assertEqual(b["seen"], b["sampled_out"] + b["selected"])
                self.assertEqual(b["serialized"], b["admitted"] + b["rejected"])
                self.assertLessEqual(b["serialized"], b["selected"])
                self.assertEqual(b["admitted"], b["ack"] + b["failed"] + b["unknown"])
                self.assertEqual(b["dropped"], b["rejected"])

    def run_workload(self, r, runs=30, calls=6, threads=4):
        function = probed(r, {"every_nth": 2})
        def produce(offset):
            for n in range(runs // threads):
                with r.run("episode", run_id=f"{offset * 1000 + n + 1:032x}"):
                    for _ in range(calls):
                        function(1)
        workers = [threading.Thread(target=produce, args=(i,)) for i in range(threads)]
        for t in workers:
            t.start()
        for t in workers:
            t.join(10)

    def test_state_conservation_with_drops_and_concurrency(self):
        gate = threading.Event()
        sink = Sink(lambda batch: gate.wait(.6))  # hold the first send so the bound is certain to bite
        r = self.make(queue_limits=(40, 10**7))
        with patch.object(r, "_send", side_effect=sink):
            self.run_workload(r)
            gate.set()
            report = r.close(10)
        found = manifests_of(sink.batches)
        self.assertEqual(len(found), 28)  # 7 runs/thread x 4 threads
        total = {k: 0 for k in cm.FIELDS}
        for manifest in found.values():
            self.assertTrue(manifest["final"])
            self.check_buckets(manifest)
            for b in manifest["buckets"].values():
                for name in cm.FIELDS:
                    total[name] += b[name]
        s = r.snapshot()
        accounting_identity(self, s)
        self.assertGreater(s["dropped"], 0)
        self.assertEqual(total["ack"], s["acknowledged"])
        self.assertEqual(total["rejected"], s["dropped"])
        self.assertEqual(total["admitted"], s["accepted"])
        self.assertEqual(total["failed"] + total["unknown"], 0)
        self.assertEqual(s["unknown"], 0)
        self.assertTrue(report["drained"])
        self.assertEqual(s["manifest"]["records_failed"] + s["manifest"]["records_unknown"], 0)

    def test_failed_and_unknown_terminal_states_are_credited(self):
        for phase, field in (("connect", "failed"), ("wait-for-ack", "unknown")):
            with self.subTest(phase=phase):
                def fail(batch, phase=phase):
                    if any(item["name"] not in (cm.MANIFEST_NAME, cm.SCOPE_NAME) for item in batch):
                        r.diagnostics.phase = phase
                        raise OSError("synthetic")
                sink = Sink(fail)
                r = self.make(cooldown=.01)
                with patch.object(r, "_send", side_effect=sink):
                    with r.run("episode", run_id="a" * 32):
                        probed(r)(1)
                    r.close(10)
                manifest = manifests_of(sink.batches)["a" * 32]
                self.check_buckets(manifest)
                self.assertEqual(sum(b[field] for b in manifest["buckets"].values()), 2)
                s = r.snapshot()
                accounting_identity(self, s)
                self.assertEqual(s[field], 2)

    def test_marker_follows_settlement_of_every_referenced_item(self):
        sink = Sink()
        r = self.make()
        with patch.object(r, "_send", side_effect=sink):
            for n in range(5):
                with r.run("episode", run_id=f"{n + 1:032x}"):
                    probed(r)(n)
            r.close(10)
        for index, batch in enumerate(sink.batches):
            for item in batch:
                if item["name"] == cm.MANIFEST_NAME:
                    earlier = {i["span"] for b in sink.batches[:index] for i in b if i["trace"] == item["trace"]}
                    self.assertIn("0000000000000001", earlier)  # root delivered strictly before the marker
                    self.assertGreaterEqual(len(earlier), 2)

    def test_marker_is_not_a_span_attempt(self):
        sink = Sink()
        r = self.make()
        with patch.object(r, "_send", side_effect=sink):
            with r.run("episode", run_id="b" * 32):
                probed(r)(1)
            r.close(10)
        s = r.snapshot()
        self.assertEqual((s["attempted"], s["acknowledged"]), (2, 2))  # decision + root only
        self.assertEqual(s["manifest"]["manifests_built"], 1)
        self.assertGreaterEqual(s["manifest"]["records_acked"], 2)  # manifest + scope tail
        accounting_identity(self, s)

    def test_lost_marker_leaves_spans_accounted_and_visible(self):
        def fail(batch):
            if any(item["name"] in (cm.MANIFEST_NAME, cm.SCOPE_NAME) for item in batch):
                r.diagnostics.phase = "connect"
                raise OSError("synthetic")
        sink = Sink(fail)
        r = self.make(cooldown=.01)
        with patch.object(r, "_send", side_effect=sink):
            with r.run("episode", run_id="c" * 32):
                probed(r)(1)
            r.close(10)
        self.assertEqual(manifests_of(sink.batches), {})
        s = r.snapshot()
        self.assertGreaterEqual(s["manifest"]["records_failed"], 1)
        self.assertEqual((s["attempted"], s["acknowledged"], s["failed"], s["unknown"]), (2, 2, 0, 0))

    def test_duplicate_delivery_after_lost_ack_is_credited_once(self):
        calls = {"n": 0}
        def flaky(batch):
            if any(item["name"] != cm.SCOPE_NAME for item in batch):
                calls["n"] += 1
                if calls["n"] == 1:  # the first attempt is delivered but its ack is lost
                    sink.batches.append(list(batch))
                    r.diagnostics.phase = "wait-for-ack"
                    raise TimeoutError("synthetic")
        sink = Sink(flaky)
        r = self.make()
        with patch.object(r, "_send", side_effect=sink):
            with r.run("episode", run_id="d" * 32):
                probed(r)(1)
            r.close(10)
        s = r.snapshot()
        accounting_identity(self, s)
        self.assertEqual((s["acknowledged"], s["unknown"], s["retry_requests"] >= 1), (2, 0, True))
        manifest = manifests_of(sink.batches)["d" * 32]
        self.assertEqual(sum(b["ack"] for b in manifest["buckets"].values()), 2)
        self.check_buckets(manifest)


class Bounds(ManifestCase):
    def test_pending_ledgers_scope_ids_and_outbox_are_bounded_and_overflow_is_visible(self):
        r = self.make(_start_worker=False, capacity=1)
        for n in range(cm.MAX_PENDING_RUNS + 76):
            with r.run("episode", run_id=f"{n + 1:032x}"):
                pass
        state = r._manifest
        self.assertEqual(len(state.ledgers), cm.MAX_PENDING_RUNS)
        self.assertEqual(state.ledger_overflow, 76)
        self.assertEqual(len(state.scope_pending), cm.MAX_SCOPE_PENDING)
        self.assertEqual(state.population_overflow, 76)
        self.assertEqual(state.expected, cm.MAX_PENDING_RUNS + 76)
        for _ in range(8):
            state.collect(False, 1)
            self.assertLessEqual(len(state.outbox), cm.MAX_OUTBOX)
        self.assertEqual(len(state.outbox), cm.MAX_OUTBOX)
        shown = r.snapshot()["manifest"]
        self.assertEqual((shown["ledger_overflow"], shown["population_overflow"]), (76, 76))

    def test_unemitted_scope_tail_reports_overflow_never_a_clean_population(self):
        r = self.make(_start_worker=False, capacity=1)
        for n in range(cm.MAX_SCOPE_PENDING + 3):
            with r.run("episode", run_id=f"{n + 1:032x}"):
                pass
        state = r._manifest
        for _ in range(100):  # the worker drains 32 records per pass
            state.collect(True, 1)
            state.take(32)
        self.assertTrue(state.tail_sent)
        self.assertEqual(state.population_overflow, 3)
        self.assertEqual(state.expected, cm.MAX_SCOPE_PENDING + 3)  # the true count stays visible

    def test_bucket_cap_collapses_into_a_visible_overflow_bucket(self):
        ledger = cm.RunLedger("1" * 32)
        for n in range(cm.MAX_BUCKETS + 5):
            ledger.bucket(f"probe{n}")
        self.assertEqual(len(ledger.buckets), cm.MAX_BUCKETS + 1)
        self.assertIn(cm.OVERFLOW, ledger.buckets)

    def test_reused_run_id_is_counted_not_silently_merged(self):
        r = self.make(_start_worker=False)
        outer = r.run("episode", run_id="e" * 32).__enter__()
        with r.run("episode", run_id="e" * 32) as inner:
            self.assertIsNone(inner.ledger)
        self.assertEqual(r._manifest.ledger_overflow, 1)
        self.assertEqual(r._manifest.expected, 1)
        outer.__exit__(None, None, None)

    def test_scope_chunks_are_emitted_when_full(self):
        sink = Sink()
        r = self.make()
        with patch.object(r, "_send", side_effect=sink):
            for n in range(cm.SCOPE_CHUNK * 2 + 5):
                with r.run("episode", run_id=f"{n + 1:032x}"):
                    pass
            r.close(10)
        records = scope_of(sink.batches)
        self.assertEqual(sorted(x["chunk"] for x in records), [0, 1, 2])
        tail = next(x for x in records if x["final"])
        self.assertEqual((tail["chunks"], tail["expected"], tail["population_overflow"]), (3, 261, 0))
        listed = [i for x in records for i in x["ids"]]
        self.assertEqual(len(listed), 261)
        self.assertEqual(len(set(listed)), 261)


class WirePaths(unittest.TestCase):
    def test_manifest_off_installs_nothing_and_bytes_are_unchanged(self):
        plain = Recorder(_start_worker=False)
        off = Recorder(_start_worker=False, manifest=False)
        on = Recorder(_start_worker=False, manifest=True)
        self.addCleanup(lambda: (plain.close(0), off.close(0), on.close(0)))
        self.assertIsNone(off._manifest)
        for name in ("_enqueue", "_deliver", "_settle"):
            self.assertNotIn(name, vars(off))
            self.assertIn(name, vars(on))
        item = dict(ITEM, attrs={"k": "v"}, events=[("e", 5, {"a": 1, "b": "text", "c": 1.5, "d": True})])
        tagged = dict(item, ledger=cm.new_bucket())  # the hidden ledger reference is never encoded
        self.assertEqual(on._encode([tagged]), plain._encode([item]))
        self.assertEqual(off._encode([item]), plain._encode([item]))
        self.assertEqual(recorder_module.estimate_span_bytes(tagged), recorder_module.estimate_span_bytes(item))

    def test_ordinary_run_items_carry_no_ledger_and_no_extra_spans_when_off(self):
        sink = Sink()
        r = Recorder()
        self.addCleanup(lambda: r.close(0))
        with patch.object(r, "_send", side_effect=sink):
            with r.run("episode", run_id="f" * 32):
                probed(r)(1)
            self.assertTrue(r.flush(5))
        names = [i["name"] for b in sink.batches for i in b]
        self.assertEqual(len(names), 2)
        self.assertNotIn(cm.MANIFEST_NAME, names)
        self.assertTrue(all("ledger" not in i for b in sink.batches for i in b))
        self.assertNotIn("manifest", r.snapshot())

    def test_manifest_span_is_a_non_root_child_inside_the_run_interval(self):
        sink = Sink()
        r = Recorder(manifest=True)
        self.addCleanup(lambda: r.close(0))
        with patch.object(r, "_send", side_effect=sink):
            with r.run("episode", run_id="9" * 32, start_ns=1000, end_ns=5000):
                pass
            r.close(10)
        item = next(i for b in sink.batches for i in b if i["name"] == cm.MANIFEST_NAME)
        self.assertEqual((item["parent"], item["start"], item["end"]), (cm.ROOT_SPAN_ID, 5000, 5000))
        self.assertEqual(item["attrs"]["signalscout.record.kind"], cm.KIND_MANIFEST)
        self.assertEqual(r._encode([item]), r._encode([item]))  # encodes with the unchanged encoder
        scope = next(i for b in sink.batches for i in b if i["name"] == cm.SCOPE_NAME)
        self.assertEqual(scope["parent"], "")
        self.assertEqual(scope["trace"], r._manifest.epoch)


class RealIntake(unittest.TestCase):
    """Producer -> unchanged real OTLP intake -> Go classification, on a fresh toy database."""
    def classify(self, server, epoch):
        raw = subprocess.check_output([str(server.binary), "--db", str(server.database), "cohort",
                                       "completeness", "--scope", epoch, "--runs"], timeout=30)
        return json.loads(raw)

    def test_complete_partial_and_absent_survive_the_wire(self):
        with tempfile.TemporaryDirectory() as directory, Serving(Path(directory) / "manifest.db") as server:
            r = Recorder(server.endpoint, manifest=True, capacity=3, _start_worker=False)
            self.addCleanup(lambda: r.close(0))
            function = probed(r)
            with r.run("episode", run_id="a" * 32):      # decision + root fit the bounded queue
                function(1)
            with r.run("episode", run_id="b" * 32):      # one decision fits; the second and the root are rejected
                function(1)
                function(2)
            ghost = "c" * 32
            r._manifest.expected += 1                    # a run the producer expected whose spans were all lost
            r._manifest.scope_pending.append(ghost)
            r._worker.start()
            report = r.close(10)
            self.assertTrue(report["drained"])
            s = r.snapshot()
            accounting_identity(self, s)
            self.assertEqual((s["accepted"], s["dropped"]), (3, 2))
            result = self.classify(server, r._manifest.epoch)
            self.assertEqual(result["runs"], {"a" * 32: "complete", "b" * 32: "partial", ghost: "absent"})
            self.assertEqual((result["scope_expected"], result["listed_runs"], result["population_known"]), (3, 3, True))
            self.assertEqual(result["counts"]["lost_items"], 2)  # B's second decision and B's root


class Caller(ManifestCase):
    def test_host_exit_never_flushes_joins_or_touches_the_network_with_manifest_on(self):
        r = self.make(_start_worker=False, capacity=1)
        with patch.object(r, "flush", side_effect=AssertionError("flush on host")), \
             patch.object(r, "close", side_effect=AssertionError("close on host")), \
             patch.object(r, "_send", side_effect=AssertionError("network on host")), \
             patch.object(r._worker, "join", side_effect=AssertionError("join on host")):
            start = time.monotonic()
            for _ in range(50):
                probed(r)(1)
            self.assertLess(time.monotonic() - start, 1)
        s = r.snapshot()
        accounting_identity(self, s)
        self.assertEqual(s["attempted"], 100)

    def test_host_exception_and_return_value_are_unchanged(self):
        r = self.make(_start_worker=False)
        self.assertEqual(probed(r)(7), 7)
        spec = {"module": __name__, "function": "boom", "capture": {"args": ["a"]}}
        with self.assertRaises(KeyError) as raised:
            Probe(Injector(r, [spec]), boom, spec).wrap()("x")
        self.assertEqual(raised.exception.args, ("x",))
        self.assertEqual(r.snapshot()["manifest"]["pending_runs"], 2)  # both runs closed, awaiting a worker


class DrainTimeout(unittest.TestCase):
    def test_validated_ceiling_is_explicit_and_documented(self):
        self.assertEqual(MAX_DRAIN_SECONDS, 60)
        for good in (0, .001, 2, 30, 60, 60.0):
            self.assertEqual(check_drain_seconds(good), float(good))
        for bad in (-1, 60.001, 61, 1e9, float("inf"), float("nan"), True, "30", None):
            with self.subTest(bad=bad), self.assertRaises(ValueError):
                check_drain_seconds(bad)

    def test_flush_and_close_reject_instead_of_silently_clamping(self):
        r = Recorder(_start_worker=False)
        self.addCleanup(lambda: r.close(0))
        for call in (r.flush, r.close):
            with self.assertRaises(ValueError):
                call(61)
        self.assertFalse(r._closed)  # a rejected close changed nothing
        begin = time.monotonic()
        self.assertTrue(r.flush(0))
        self.assertLess(time.monotonic() - begin, 1)

    def test_a_timeout_above_the_old_ceiling_is_honoured(self):
        r = Recorder(_start_worker=False)
        self.addCleanup(lambda: r.close(0))
        r._enqueue(span(0))  # never drained: flush waits on the queue condition
        waits = []
        def wait(left):
            waits.append(left)
            r._q.unfinished_tasks = 0  # end the wait without sleeping
        with patch.object(r._q.all_tasks_done, "wait", side_effect=wait):
            r.flush(30)
        self.assertGreater(waits[0], 20)  # the old code would have capped this at 10

    def test_cli_default_is_two_seconds_and_bad_values_are_usage_errors(self):
        for flag in ("nan", "-1", "61", "inf", "abc"):
            with self.subTest(flag=flag), redirect_stderr(io.StringIO()), self.assertRaises(SystemExit) as raised:
                scout_probe.main(["--config", "unused.yaml", "--doctor", "unused.json",
                                  "--drain-timeout", flag, "m:f"])
            self.assertEqual(raised.exception.code, 2)
        fixtures = Path(__file__).resolve().parents[1] / "fixtures"
        for flags, expected in (([], 2.0), (["--drain-timeout", "30"], 30.0), (["--drain-timeout", "60"], 60.0)):
            with self.subTest(flags=flags), tempfile.TemporaryDirectory() as directory:
                cfg = Path(directory) / "config.yaml"
                cfg.write_text(yaml.safe_dump({"version": 1, "synthetic_only": True,
                    "probes": [{"module": "tuple_environment", "function": "loop"}]}))
                recorder = Recorder(_start_worker=False)
                timeouts = []
                real_close = recorder.close
                recorder.close = lambda timeout=2: (timeouts.append(timeout), real_close(0))[1]
                sys.path.insert(0, str(fixtures))
                try:
                    with patch.object(scout_probe, "Recorder", return_value=recorder), redirect_stderr(io.StringIO()):
                        scout_probe.main(["--config", str(cfg), "--doctor", str(Path(directory) / "d.json"),
                                          "--path", str(fixtures), "--preflight", *flags, "tuple_environment:main"])
                finally:
                    sys.path.remove(str(fixtures))
                    sys.modules.pop("tuple_environment", None)
                self.assertEqual(timeouts, [expected])

    def test_drain_applies_at_close_only(self):
        r = Recorder()
        self.addCleanup(lambda: r.close(0))
        release = threading.Event()
        with patch.object(r, "_send", side_effect=lambda batch: release.wait(5)):
            r._enqueue(span(0))
            begin = time.monotonic()
            report = r.close(.3)  # the stalled worker is abandoned at the drain deadline
            elapsed = time.monotonic() - begin
            release.set()
        self.assertFalse(report["drained"])
        self.assertGreaterEqual(elapsed, .25)
        self.assertLess(elapsed, 2)


if __name__ == "__main__":
    unittest.main()
