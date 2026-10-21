"""One small synthetic paired observation; not budget qualification or optimization."""
import argparse
import hashlib
import importlib
import json
import sys
import time
from pathlib import Path

from recorder import Recorder
from scout_probe import Injector
from spike_support import Serving


def percentile(values, quantile):
    ordered = sorted(values)
    position = quantile * (len(ordered) - 1)
    lower = int(position)
    upper = min(lower + 1, len(ordered) - 1)
    return ordered[lower] + (ordered[upper] - ordered[lower]) * (position - lower)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path, required=True, help="Fresh output directory")
    args = parser.parse_args()
    root = Path(__file__).resolve().parent
    fixture = root / "fixtures/usability/class_environment.py"
    args.out.mkdir(parents=True, exist_ok=True)
    report = {"synthetic_only": True, "trials": 1, "pairs": 300, "warmup_pairs": 20,
              "total_calls_per_second": 60, "per_kind_calls_per_second": 30,
              "payload": [.51, .2, .1, 7], "receiver_capture": "OFF",
              "method_added_cost_against_budgets": "unmeasured; this is one exploratory paired trial, not five-trial qualification",
              "source_sha256": hashlib.sha256(fixture.read_bytes()).hexdigest()}
    with Serving(args.out / "fresh.db") as server:
        recorder = Recorder(server.endpoint, capture_fields={"a", "b", "c", "d", "output"})
        owner = Injector(recorder, [{"module": "class_environment", "function": name,
            "capture": {"args": ["a", "b", "c", "d"], "return": True}} for name in ("payload", "Environment.payload")])
        sys.path.insert(0, str(fixture.parent))
        owner.install()
        try:
            module = importlib.import_module("class_environment")
            environment = module.Environment()
            calls = {"function": module.payload, "method": environment.payload}
            samples = {name: [] for name in calls}
            next_tick = time.perf_counter()
            for pair in range(320):
                order = ("function", "method") if pair % 2 == 0 else ("method", "function")
                for name in order:
                    time.sleep(max(0, next_tick - time.perf_counter()))
                    next_tick += 1 / 60
                    started = time.perf_counter_ns()
                    output = calls[name](*report["payload"])
                    elapsed = time.perf_counter_ns() - started
                    assert output["choice"] == sum(report["payload"])
                    body = calls[name].__signalscout_probe__.last["body_ns"]
                    if pair >= 20:
                        samples[name].append({"caller_us": (elapsed - body) / 1000, "body_us": body / 1000, "total_us": elapsed / 1000})
            report["flush_complete"] = bool(recorder.close(5))
            report["counters"] = recorder.snapshot()
            report["samples"] = samples
            report["summary_us"] = {name: {"p50": percentile([v["caller_us"] for v in rows], .5),
                                          "p99": percentile([v["caller_us"] for v in rows], .99)} for name, rows in samples.items()}
            report["drops"] = {k: v for k, v in report["counters"].items() if (k.startswith("drop") or k in ("failed", "unknown")) and v}
            report["complete"] = report["flush_complete"] and not report["drops"] and report["counters"].get("probe_decisions_captured") == 640
            report["target_source_unchanged"] = hashlib.sha256(fixture.read_bytes()).hexdigest() == report["source_sha256"]
        finally:
            owner.uninstall()
            recorder.close(1)
    (args.out / "paired.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({k: report[k] for k in ("summary_us", "complete", "drops", "target_source_unchanged")}, indent=2))


if __name__ == "__main__":
    main()
