"""Bounded, payload-free observation of the existing urllib export path.

Transport hooks delegate unchanged to stdlib methods. Only the exporter thread
sets an observer; other HTTP callers keep their original behavior. Hooks never
log arguments, headers, endpoints, exception reprs or request/response bodies.
"""
from collections import Counter, deque
from contextlib import contextmanager
import errno
import functools
import http.client
import socket
import threading
import time
import urllib.error


_active = threading.local()


def _phase(name):
    observer = getattr(_active, "observer", None)
    if observer is not None:
        observer.phase = name


def _hook(original, before, after):
    @functools.wraps(original)
    def call(*args, **kwargs):
        _phase(before)
        result = original(*args, **kwargs)
        _phase(after)
        return result
    return call


# Installed once per interpreter, including repeated imports. Phase changes are
# thread-local assignments only; transport exceptions and return values pass
# through untouched. They do not move I/O onto the host path.
if not getattr(http.client.HTTPConnection.send, "_scout_observer", False):
    for cls, name, before, after in (
        (http.client.HTTPConnection, "connect", "connect", "send"),
        (http.client.HTTPSConnection, "connect", "connect", "send"),
        (http.client.HTTPConnection, "send", "send", "wait-for-ack"),
        (http.client.HTTPConnection, "getresponse", "wait-for-ack", "response"),
    ):
        wrapped = _hook(getattr(cls, name), before, after)
        wrapped._scout_observer = True
        setattr(cls, name, wrapped)


@contextmanager
def observe(diagnostics):
    previous = getattr(_active, "observer", None)
    _active.observer = diagnostics
    try:
        yield
    finally:
        _active.observer = previous


class Diagnostics:
    def __init__(self):
        self.lock = threading.Lock()
        self.phase = "encode"  # Only the worker mutates the current phase.
        self.counts = Counter({name: 0 for name in (
            "encode_failure", "timeout", "connection_error", "http_non_2xx",
            "cooldown_suppressed", "other_failure", "flush_timeout")})
        self.classes = Counter()
        self.samples = deque(maxlen=64)
        self.batches = deque(maxlen=256)
        self.batch_sizes = Counter()  # Existing batches are always 1..32.

    def failure(self, error, spans, started_ns):
        status = error.code if isinstance(error, urllib.error.HTTPError) else None
        if status is not None and not (type(status) is int and 100 <= status <= 599):
            status = None
        reason = error.reason if isinstance(error, urllib.error.URLError) else error
        if status is not None:
            kind, phase = "http_non_2xx", "response"
            message = f"HTTP status {status}; response detail redacted"
        elif self.phase == "encode":
            kind, phase = "encode_failure", self.phase
            message = "OTLP encode failed; detail redacted (may contain captured values)"
        elif isinstance(reason, (TimeoutError, socket.timeout)):
            kind, phase, message = "timeout", self.phase, "timed out"
        elif isinstance(reason, (ConnectionError, OSError)):
            kind, phase = "connection_error", self.phase
            number = getattr(reason, "errno", None)
            message = "transport error " + errno.errorcode.get(number, "UNKNOWN")
        else:
            kind, phase = "other_failure", self.phase
            message = "export failed; untrusted exception detail redacted"
        sample = {"exception_class": type(error).__name__[:80],
                  "reason_class": type(reason).__name__[:80],
                  "message": message[:160], "http_status": status,
                  "phase": phase, "kind": kind, "spans": spans,
                  "attempt_unix_ns": started_ns, "observed_unix_ns": time.time_ns()}
        with self.lock:
            self.counts[kind] += spans
            if status is not None:
                self.counts[f"http_status_{status}"] += spans
            # Cap distinct class keys as well as the ring, even for injected types.
            key = sample["exception_class"]
            self.classes[key if key in self.classes or len(self.classes) < 32 else "other"] += spans
            self.samples.append(sample)
        return kind

    def batch(self, size, started_ns, elapsed_ns, outcome, queue_depth):
        with self.lock:
            self.batch_sizes[size] += 1
            self.batches.append({"attempt_unix_ns": started_ns, "elapsed_ns": elapsed_ns,
                                 "size": size, "outcome": outcome, "queue_depth": queue_depth})

    def count(self, name, n=1):
        with self.lock:
            self.counts[name] += n

    def snapshot(self):
        with self.lock:
            return {"counters_spans_except_flush_timeout": dict(self.counts),
                    "failures_by_class_spans": dict(self.classes),
                    "last_failures": list(self.samples), "failure_sample_limit": self.samples.maxlen,
                    "worker_batch_sizes": dict(self.batch_sizes),
                    "last_worker_batches": list(self.batches), "batch_sample_limit": self.batches.maxlen}
