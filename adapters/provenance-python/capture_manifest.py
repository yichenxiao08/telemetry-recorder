"""Opt-in run completeness manifests. Synthetic/development use only.

Nothing here runs unless Recorder(manifest=True). The caller thread only does
bounded integer arithmetic under one small lock (no I/O, no waiting). The
worker thread alone builds and delivers manifest spans, after every item a
manifest references has settled. Manifest delivery never touches the span
accounting identity (attempted = acknowledged + failed + dropped + unknown).
"""
from __future__ import annotations

import json
import secrets
import threading
from collections import deque

VERSION = 1
ROOT_SPAN_ID = "0000000000000001"
MANIFEST_SPAN_ID = "ffffffffffffff01"
MANIFEST_NAME = "signalscout.capture.manifest"
SCOPE_NAME = "signalscout.capture.scope"  # reserved workload; never a cohort candidate
KIND_MANIFEST = "capture-manifest"
KIND_SCOPE = "capture-scope"

MAX_PENDING_RUNS = 1024   # unfinished per-run ledgers
MAX_BUCKETS = 34          # distinct buckets per run (<=32 probes + root + direct); the rest share "@overflow"
MAX_OUTBOX = 256          # manifest/scope spans waiting for the worker
SCOPE_CHUNK = 128         # run IDs per scope record
MAX_SCOPE_PENDING = 1024  # run IDs not yet emitted in a scope record

FIELDS = ("seen", "sampled_out", "selected", "serialized", "admitted", "rejected",
          "ack", "failed", "unknown", "detail_loss")
ROOT, DIRECT, OVERFLOW = "@root", "@direct", "@overflow"


def new_bucket():
    bucket = dict.fromkeys(FIELDS, 0)
    bucket["pending"] = 0  # admissions in progress; a ledger is never final while nonzero
    return bucket


class RunLedger:
    def __init__(self, trace_id):
        self.trace_id = trace_id
        self.buckets = {}
        self.closed = False
        self.end_ns = 0
        self._lock = threading.Lock()

    def bucket(self, key):
        found = self.buckets.get(key)
        if found is not None:
            return found
        with self._lock:
            found = self.buckets.get(key)
            if found is None:
                if len(self.buckets) >= MAX_BUCKETS:
                    key = OVERFLOW
                    found = self.buckets.get(key)
                if found is None:
                    found = self.buckets[key] = new_bucket()
            return found


class ManifestState:
    def __init__(self, epoch=None):
        self.lock = threading.Lock()
        self.epoch = epoch or secrets.token_hex(16)
        self.ledgers = {}
        self.closed = {}
        self.scope_pending = []
        self.chunk = 0
        self.expected = 0
        self.ledger_overflow = 0
        self.population_overflow = 0
        self.tail_sent = False
        self.outbox = deque()
        self.counts = dict.fromkeys(("manifests_built", "scope_records_built", "records_acked",
                                     "records_failed", "records_unknown"), 0)

    # ---- caller side: integer arithmetic only -------------------------------
    def open_run(self, trace_id):
        with self.lock:
            if trace_id in self.ledgers:
                self.ledger_overflow += 1  # a reused run ID cannot be accounted separately
                return None
            self.expected += 1
            if len(self.scope_pending) < MAX_SCOPE_PENDING:
                self.scope_pending.append(trace_id)
            else:
                self.population_overflow += 1
            if len(self.ledgers) >= MAX_PENDING_RUNS:
                self.ledger_overflow += 1
                return None
            ledger = self.ledgers[trace_id] = RunLedger(trace_id)
            return ledger

    def call(self, bucket, selected):
        with self.lock:
            bucket["seen"] += 1
            bucket["selected" if selected else "sampled_out"] += 1

    def serialized(self, bucket):
        with self.lock:
            bucket["serialized"] += 1

    def detail_loss(self, bucket):
        with self.lock:
            bucket["detail_loss"] += 1

    def begin(self, bucket):
        with self.lock:
            bucket["pending"] += 1

    def finish(self, bucket, admitted):
        with self.lock:
            bucket["pending"] -= 1
            bucket["admitted" if admitted else "rejected"] += 1

    def close_run(self, ledger, end_ns):
        with self.lock:
            ledger.closed, ledger.end_ns = True, end_ns
            self.closed[ledger.trace_id] = ledger

    # ---- worker side ---------------------------------------------------------
    def settle(self, batch, state):
        key = {"acknowledged": "ack", "failed": "failed", "unknown": "unknown"}[state]
        with self.lock:
            for item in batch:
                bucket = item.get("ledger")
                if bucket is not None:
                    bucket[key] += 1

    def settle_records(self, count, state):
        key = {"acknowledged": "records_acked", "failed": "records_failed", "unknown": "records_unknown"}[state]
        with self.lock:
            self.counts[key] += count

    @staticmethod
    def _ready(ledger):
        return all(b["pending"] == 0 and b["admitted"] == b["ack"] + b["failed"] + b["unknown"]
                   for b in ledger.buckets.values())

    def _manifest_item(self, ledger):
        buckets = {}
        for key, b in sorted(ledger.buckets.items()):
            row = {name: b[name] for name in FIELDS}
            if key == ROOT:  # the root span is built unconditionally: derive its call counts
                row["seen"] = row["selected"] = row["serialized"] = b["admitted"] + b["rejected"]
            row["dropped"] = b["rejected"]  # confirmed local drop == admission rejection
            buckets[key] = row
        body = json.dumps({"v": VERSION, "epoch": self.epoch, "seq": 1, "final": True, "buckets": buckets},
                          sort_keys=True, separators=(",", ":"))
        return {"trace": ledger.trace_id, "span": MANIFEST_SPAN_ID, "parent": ROOT_SPAN_ID,
                "name": MANIFEST_NAME, "start": ledger.end_ns, "end": ledger.end_ns,
                "attrs": {"signalscout.record.kind": KIND_MANIFEST,
                          "signalscout.manifest.version": VERSION,
                          "signalscout.manifest.json": body},
                "events": [], "error": False}

    def _scope_item(self, ids, final, now_ns):
        body = {"v": VERSION, "epoch": self.epoch, "chunk": self.chunk, "ids": ids, "final": final}
        if final:
            body.update(chunks=self.chunk + 1, expected=self.expected,
                        ledger_overflow=self.ledger_overflow, population_overflow=self.population_overflow)
        item = {"trace": self.epoch, "span": f"{self.chunk + 1:016x}", "parent": "",
                "name": SCOPE_NAME, "start": now_ns, "end": now_ns,
                "attrs": {"signalscout.record.kind": KIND_SCOPE,
                          "signalscout.manifest.version": VERSION,
                          "signalscout.manifest.json": json.dumps(body, sort_keys=True, separators=(",", ":"))},
                "events": [], "error": False}
        self.chunk += 1
        return item

    def collect(self, finalizing, now_ns):
        """Move ready records to the bounded outbox. Worker thread only."""
        with self.lock:
            for trace_id in list(self.closed):
                if len(self.outbox) >= MAX_OUTBOX:
                    break
                ledger = self.closed[trace_id]
                if not self._ready(ledger):
                    continue
                self.outbox.append(self._manifest_item(ledger))
                self.counts["manifests_built"] += 1
                del self.closed[trace_id], self.ledgers[trace_id]
            while len(self.scope_pending) >= SCOPE_CHUNK and len(self.outbox) < MAX_OUTBOX:
                ids, self.scope_pending = self.scope_pending[:SCOPE_CHUNK], self.scope_pending[SCOPE_CHUNK:]
                self.outbox.append(self._scope_item(ids, False, now_ns))
                self.counts["scope_records_built"] += 1
            if (finalizing and not self.tail_sent and len(self.outbox) < MAX_OUTBOX
                    and len(self.scope_pending) < SCOPE_CHUNK
                    and (self.expected or self.ledger_overflow or self.population_overflow)):
                ids, self.scope_pending = self.scope_pending, []
                self.outbox.append(self._scope_item(ids, True, now_ns))
                self.counts["scope_records_built"] += 1
                self.tail_sent = True
            return len(self.outbox)

    def take(self, count):
        with self.lock:
            return [self.outbox.popleft() for _ in range(min(count, len(self.outbox)))]

    def snapshot(self):
        with self.lock:
            return dict(self.counts, epoch=self.epoch, expected_runs=self.expected,
                        pending_runs=len(self.ledgers), outbox=len(self.outbox),
                        ledger_overflow=self.ledger_overflow,
                        population_overflow=self.population_overflow, scope_tail_sent=self.tail_sent)
