"""Development-only diagnostics around the unchanged lane-02 driver.

No reserved flag, cohorts or equations are added. Files are fresh per invocation;
the original report is retained even if its completeness check aborts a run.
"""
import argparse
from collections import deque
import json
import subprocess
import sys
import threading
import time
from pathlib import Path

import lane02
from recorder import Recorder


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    if args.out.exists():
        raise ValueError("fresh diagnostics directory required")
    # The driver itself creates this child. The wrapper only creates its parent.
    args.out.mkdir(parents=True)
    path = args.out/"driver"
    events = deque(maxlen=2048)
    stopped = threading.Event()
    result = {"synthetic_only": True, "seeds": [101, 103], "cases": [],
              "method": "Unmodified development driver; unchanged transport defaults. Observer thread drains bounded telemetry rings to disk. Caller samples contain timings only, no captured fields.",
              "observation_overflow": 0}

    def save():
        (args.out/"diagnostics.json").write_text(json.dumps(result, indent=2)+"\n")

    class ObservedRecorder(Recorder):
        def __init__(self, *a, **kw):
            super().__init__(*a, **kw)
            self.calls, self.active_phase = 0, None
            result["defaults"] = {"capacity": self._q.maxsize, "timeout": self.timeout, "cooldown": self.cooldown}

        def run(self, *a, **kw):
            if self.calls % 800 == 0:
                self.active_phase = {"index": self.calls//800, "started_unix_ns": time.time_ns(),
                                     "started_perf_ns": time.perf_counter_ns(), "calls": 0,
                                     "before": self.snapshot()}
            self.calls += 1
            self.active_phase["calls"] += 1
            manager = super().run(*a, **kw)
            observer = self
            class ObservedRun:
                def __enter__(self):
                    return manager.__enter__()
                def __exit__(self, *exc):
                    returned = manager.__exit__(*exc)
                    try:
                        probe = sys.modules["lane02_target"].decide.__signalscout_probe__
                        event = {"kind": "caller", "unix_ns": time.time_ns(),
                                 "case_index": observer.active_phase["index"],
                                 "wrapper_ns": probe.last["wrapper_ns"]}
                        if len(events) == events.maxlen:
                            result["observation_overflow"] += 1
                        events.append(event)
                    except Exception:
                        result["observation_overflow"] += 1
                    return returned
            return ObservedRun()

        def flush(self, timeout=2):
            phase = self.active_phase
            if phase is not None:
                wall = time.perf_counter_ns()-phase.pop("started_perf_ns")
                phase.update(wall_ns=wall, achieved_calls_per_second=phase["calls"]/(wall/1e9),
                             finished_unix_ns=time.time_ns())
                self.active_phase = None
            begin = time.perf_counter_ns()
            drained = super().flush(timeout)
            if phase is not None:
                phase.update(flush_ns=time.perf_counter_ns()-begin, drained=drained, after=self.snapshot())
                result["cases"].append(phase)
                save()
            return drained

    original_recorder = lane02.Recorder
    lane02.Recorder = ObservedRecorder
    old_argv = sys.argv
    # Only development seeds, without any opt-in evaluation argument.
    sys.argv = ["lane02.py", "--out", str(path), "--seeds", "101", "103"]
    recorder_seen = []
    original_init = ObservedRecorder.__init__
    def register(self, *a, **kw):
        original_init(self, *a, **kw)
        recorder_seen.append(self)
    ObservedRecorder.__init__ = register

    def sample():
        seen_batches, seen_failures = set(), set()
        with (args.out/"timeline.jsonl").open("w") as stream:
            while not stopped.is_set() or events:
                while events:
                    stream.write(json.dumps(events.popleft())+"\n")
                if recorder_seen:
                    recorder = recorder_seen[0]
                    snapshot = recorder.snapshot()
                    stream.write(json.dumps({"kind": "queue", "unix_ns": time.time_ns(),
                                              "depth": snapshot["queue_size"], "high_water": snapshot["queue_high_water"]})+"\n")
                    for key, seen, kind in (("last_worker_batches", seen_batches, "batch"),
                                            ("last_failures", seen_failures, "failure")):
                        for event in snapshot["diagnostics"][key]:
                            identity = event["attempt_unix_ns"]
                            if identity not in seen:
                                stream.write(json.dumps({"kind": kind, **event})+"\n")
                                seen.add(identity)
                        # Keep only IDs still present in the bounded source ring.
                        seen.intersection_update(e["attempt_unix_ns"] for e in snapshot["diagnostics"][key])
                stream.flush()
                stopped.wait(.02)
    worker = threading.Thread(target=sample, daemon=True)
    worker.start()
    try:
        lane02.main()
        result["status"] = "completed"
    except BaseException as error:
        result.update(status="failed", error=type(error).__name__+": "+str(error))
        raise
    finally:
        if recorder_seen:
            result["final"] = recorder_seen[0].snapshot()
        stopped.set()
        worker.join(5)
        result["sampler_stopped"] = not worker.is_alive()
        save()
        lane02.Recorder, sys.argv = original_recorder, old_argv


if __name__ == "__main__":
    main()
