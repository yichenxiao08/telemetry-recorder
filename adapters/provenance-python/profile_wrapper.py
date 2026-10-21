"""Measurement-only fresh-process runner. No reserved-seed argument exists."""
import argparse
from collections import deque
import cProfile
import gc
import gzip
import hashlib
import json
import os
from pathlib import Path
import platform
import pstats
import subprocess
import sys
import threading
import time
import types

import recorder
import scout_probe
from profile_phases import Meter, PHASES, copies
from spike_support import Serving

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(Path(__file__).parent/"fixtures"))
sys.path.insert(0, str(ROOT/"benchmark/problems/lane-02/plain"))
from benchmark_target import calculate


def save(path, value):
    path.write_text(json.dumps(value, indent=2)+"\n", encoding="utf-8")


def percentile(values, q):
    values = sorted(values)
    if not values:
        return None
    position = q*(len(values)-1)
    lo = int(position)
    return values[lo]+(values[min(lo+1, len(values)-1)]-values[lo])*(position-lo)


def stats(values):
    return {"p50_us": percentile(values, .5)/1000 if values else None,
            "p99_us": percentile(values, .99)/1000 if values else None,
            "sum_ns": sum(values)}


def summarize(samples):
    total = sum(x["adjusted_total_ns"] for x in samples)
    phases = {}
    for phase in PHASES:
        values = [x["adjusted_exclusive_ns"].get(phase, 0) for x in samples]
        phases[phase] = {**stats(values), "fraction": sum(values)/total if total else None}
    result = {"calls": len(samples), "raw": stats([s["raw_total_ns"] for s in samples]),
              "adjusted": stats([s["adjusted_total_ns"] for s in samples]), "phases": phases,
              "residual": stats([s["residual_ns"] for s in samples]),
              "negative_adjustments": sum(s["negative_adjustments"] for s in samples)}
    result["budget_met"] = result["adjusted"]["p50_us"] <= 100 and result["adjusted"]["p99_us"] <= 250
    return result


class Factors:
    """Bounded overlap evidence, not a causal GIL/scheduler classifier."""
    def __init__(self):
        self.intervals = deque(maxlen=100000)
        self.gc_start = {}
        self.overflow = 0

    def append(self, kind, start, end, thread):
        if len(self.intervals) == self.intervals.maxlen:
            self.overflow += 1
        self.intervals.append((kind, start, end, thread))

    def callback(self, phase, info):
        identity = (threading.get_ident(), info["generation"])
        if phase == "start":
            self.gc_start[identity] = time.perf_counter_ns()
        else:
            self.append("gc", self.gc_start.pop(identity, time.perf_counter_ns()),
                        time.perf_counter_ns(), identity[0])

    def observe(self, cls):
        for name in ("_encode", "_send"):
            original = getattr(cls, name)
            def wrapped(rec, *a, _original=original, _name=name, **kw):
                begin = time.perf_counter_ns()
                try:
                    return _original(rec, *a, **kw)
                finally:
                    self.append(_name, begin, time.perf_counter_ns(), threading.get_ident())
            setattr(cls, name, wrapped)


def inputs(payload):
    if payload == "fixed":
        fields = ["gap", "previous_lane", "previous_frame", "frame"]
        return calculate, fields, [((.51, 0, 1, i), {}, .51+i) for i in range(10000)]
    import lane02_generator
    # Development guard invoked with its default False; never an evaluation flag.
    from lane02_target import decide
    rows = [r for seed in (101, 103) for variant in lane02_generator.VARIANTS
            for r in lane02_generator.rows(seed, variant)]
    return decide, ["gap", "current_value", "history", "previous_frame", "frame", "noise"], [
        ((), r["inputs"], decide(**r["inputs"])) for r in rows]


def child(args):
    args.out.mkdir(parents=True, exist_ok=False)
    meter = Meter()
    calibration = meter.calibrate()
    api = copies(meter) if args.observer == "phase" else types.SimpleNamespace(
        Recorder=recorder.Recorder, Probe=scout_probe.Probe, Injector=scout_probe.Injector)
    # A private subclass keeps process controls from replacing original methods.
    class Observed(api.Recorder):
        pass
    factors = Factors()
    factors.observe(Observed)
    gc.callbacks.append(factors.callback)
    if args.factor == "gc-disabled":
        gc.disable()
    try:
        if args.payload == "lane-driver":
            lane_driver(args, meter, api, Observed, factors, calibration)
        else:
            with Serving(args.out/"store.db") as server:
                micro(args, meter, api, Observed, factors, calibration, server.endpoint)
    finally:
        gc.callbacks.remove(factors.callback)
        gc.enable()


def micro(args, meter, api, cls, factors, calibration, endpoint):
    original, fields, rows = inputs(args.payload)
    spec = {"module": original.__module__, "function": original.__name__}
    if args.feature != "off":
        spec["capture"] = {"args": fields if args.feature == "full" else [], "return": args.feature == "full"}
    r = cls(endpoint, service="synthetic.wrapper.profile",
            capture_fields=fields+["output"] if args.feature != "off" else (),
            _start_worker=args.factor != "worker-idle")
    owner = api.Injector(r, [spec])
    function = original if args.feature == "bare" else api.Probe(owner, original, spec).wrap()
    probe = getattr(function, "__signalscout_probe__", None)
    calls = 1800 if args.mode == "60hz" else 10000
    profiler = cProfile.Profile() if args.observer == "cprofile" else None
    samples = []
    started = time.perf_counter()
    if profiler:
        profiler.enable()
    for index in range(calls):
        if args.mode == "60hz":
            time.sleep(max(0, started+index/60-time.perf_counter()))
        a, kw, expected = rows[index % len(rows)]
        depth = r._q.qsize()
        meter.begin()
        returned = function(*a, **kw)
        item = meter.finish(index)
        if (returned["value"] != expected if args.payload == "fixed" else returned != expected):
            raise AssertionError("changed target return")
        if args.observer != "phase" and probe:
            body = probe.last["body_ns"]
            item["raw_total_ns"] -= body
            item["adjusted_total_ns"] -= body
            item["target_body_ns"] = body
            item["residual_ns"] = item["adjusted_total_ns"]
        item["queue_depth_before"] = depth
        samples.append(item)
    if profiler:
        profiler.disable()
        profiler.dump_stats(str(args.out/"profile.pstats"))
        ps = pstats.Stats(profiler)
        save(args.out/"profile-functions.json", [
            {"file": Path(k[0]).name, "line": k[1], "function": k[2],
             "primitive_calls": v[0], "calls": v[1], "self_seconds": v[2], "cumulative_seconds": v[3]}
            for k, v in sorted(ps.stats.items(), key=lambda kv: -kv[1][3])])
    if args.mode == "60hz":
        time.sleep(max(0, started+30-time.perf_counter()))
    elapsed = time.perf_counter()-started
    flush_start = time.perf_counter()
    drained = r.flush(.01 if args.factor == "worker-idle" else 10)
    flush_seconds = time.perf_counter()-flush_start
    counts = r.snapshot()
    r.close(0)
    report(args, samples, calibration, factors,
           {"elapsed_seconds": elapsed, "counts": counts, "flush_seconds": flush_seconds,
            "flush": dict(drained), "mean_envelope_bytes": counts.get("envelope_bytes", 0)/max(1, counts.get("serialized", 0))})


def lane_driver(args, meter, api, cls, factors, calibration):
    import lane02
    samples = []
    observed = []
    class LaneRecorder(cls):
        def __init__(self, *a, **kw):
            super().__init__(*a, **kw)
            observed.append(self)

        def run(self, *a, **kw):
            manager = super().run(*a, **kw)
            class Boundary:
                def __enter__(self):
                    meter.begin()
                    return manager.__enter__()
                def __exit__(self, *exc):
                    try:
                        return manager.__exit__(*exc)
                    finally:
                        samples.append(meter.finish(len(samples)))
            return Boundary()
    lane02.Recorder, lane02.Injector = LaneRecorder, api.Injector
    if args.mode == "tight":
        class TimeFacade:
            def __getattr__(self, name):
                return getattr(time, name)
            def sleep(self, seconds):
                pass
        lane02.time = TimeFacade()
    sys.argv = ["lane02.py", "--out", str(args.out/"driver"), "--seeds", "101", "103"]
    status, error = "completed", None
    try:
        lane02.main()
    except BaseException as exc:
        status, error = "failed", type(exc).__name__+": "+str(exc)
    report(args, samples, calibration, factors,
           {"status": status, "error": error, "counts": observed[0].snapshot() if observed else {}})


def report(args, samples, calibration, factors, extra):
    # Snapshot once: GC callbacks can append while report allocations collect.
    # Iterating a live deque through a Python generator is not safe here.
    intervals_snapshot = list(factors.intervals)
    with gzip.open(args.out/"raw.json.gz", "wt", encoding="utf-8") as stream:
        json.dump({"samples": samples, "calibration": calibration, "intervals": intervals_snapshot}, stream)
    summary = summarize(samples)
    threshold = percentile([s["raw_total_ns"] for s in samples], .99)
    tails = [s for s in samples if s["raw_total_ns"] >= threshold]
    overlap = {}
    for kind in ("gc", "_encode", "_send"):
        intervals = sorted((a, b) for k, a, b, _ in intervals_snapshot if k == kind)
        def count_overlap(items):
            # Linear interval merge; do not add observer work to the measured path.
            index, count = 0, 0
            for sample in items:
                while index < len(intervals) and intervals[index][1] < sample["started_ns"]:
                    index += 1
                if index < len(intervals) and intervals[index][0] <= sample["stopped_ns"]:
                    count += 1
            return count
        overlap[kind] = {"all_calls": count_overlap(samples), "top_1_percent_calls": count_overlap(tails)}
    off_cpu = [max(0, s["stopped_ns"]-s["started_ns"]-s["caller_thread_cpu_ns"]) for s in tails]
    save(args.out/"summary.json", {"synthetic_only": True, "configuration": {
        k: str(v) if isinstance(v, Path) else v for k, v in vars(args).items()},
        "python": sys.version, "platform": platform.platform(), "cpu": os.environ.get("PROCESSOR_IDENTIFIER"),
        "gc_enabled_during_calls": args.factor != "gc-disabled", "gc_threshold": gc.get_threshold(),
        "source_sha256": {name: hashlib.sha256((Path(__file__).parent/name).read_bytes()).hexdigest()
                          for name in ("recorder.py", "scout_probe.py", "profile_phases.py", "profile_wrapper.py")},
        "calibration": {k: stats(v) if isinstance(v, list) else v for k, v in calibration.items()},
        **summary, "overlap": overlap, "tail_off_cpu": stats(off_cpu),
        "observer_overflow": factors.overflow+max(0, len(samples)-20000), **extra})


def suite(args):
    args.out.mkdir(parents=True, exist_ok=False)
    jobs = []
    for payload in ("fixed", "lane-inputs"):
        for mode in ("60hz", "tight"):
            for observer, feature, factor in [
                *[("phase", f, "default") for f in ("bare", "off", "empty", "full")],
                ("plain", "full", "default"), ("phase", "full", "gc-disabled"),
                ("phase", "full", "worker-idle")]:
                for trial in range(1, 6):
                    jobs.append((payload, mode, observer, feature, factor, trial))
            jobs.append((payload, mode, "cprofile", "full", "default", 1))
    for mode in ("60hz", "tight"):
        for trial in range(1, 6):
            jobs.append(("lane-driver", mode, "phase", "full", "default", trial))
    save(args.out/"plan.json", jobs)
    progress = []
    for payload, mode, observer, feature, factor, trial in jobs:
        name = f"{payload}-{mode}-{observer}-{feature}-{factor}-{trial}"
        command = [sys.executable, __file__, "--child", "--out", str(args.out/name),
                   "--payload", payload, "--mode", mode, "--observer", observer,
                   "--feature", feature, "--factor", factor, "--trial", str(trial)]
        begin = time.time()
        with (args.out/(name+".log")).open("w", encoding="utf-8") as log:
            completed = subprocess.run(command, stdout=log, stderr=subprocess.STDOUT, check=False)
        progress.append({"name": name, "exit_code": completed.returncode, "elapsed_seconds": time.time()-begin})
        save(args.out/"progress.json", progress)
        print(name, completed.returncode, flush=True)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path, required=True)
    parser.add_argument("--child", action="store_true")
    parser.add_argument("--payload", choices=("fixed", "lane-inputs", "lane-driver"), default="fixed")
    parser.add_argument("--mode", choices=("60hz", "tight"), default="tight")
    parser.add_argument("--observer", choices=("phase", "plain", "cprofile"), default="phase")
    parser.add_argument("--feature", choices=("bare", "off", "empty", "full"), default="full")
    parser.add_argument("--factor", choices=("default", "gc-disabled", "worker-idle"), default="default")
    parser.add_argument("--trial", type=int, default=1)
    args = parser.parse_args()
    (child if args.child else suite)(args)


if __name__ == "__main__":
    main()
