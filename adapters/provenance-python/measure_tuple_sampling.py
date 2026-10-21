"""One small synthetic local cost observation, without an exporter or optimization."""
import json
import statistics
import sys
import time
from pathlib import Path

from recorder import Recorder
from scout_probe import Injector, Probe

sys.path.insert(0, str(Path(__file__).resolve().parent / "fixtures"))
from tuple_environment import Environment


def measure():
    observations = {}
    for label, sampling in (("unsampled", {}), ("sampled", {"every_nth": 10})):
        recorder = Recorder(_start_worker=False, capture_fields={"output"}, capacity=4096)
        owner = Injector(recorder, [])
        spec = {"module": "tuple_environment", "function": "Environment.step",
                "capture": {"return": True}, "sampling": sampling}
        call = Probe(owner, Environment.step, spec, Environment, "self").wrap()
        env = Environment()
        costs = []
        with recorder.run("synthetic.cost"):
            for _ in range(300):
                start = time.perf_counter_ns()
                call(env, 1)
                costs.append((time.perf_counter_ns() - start) / 1000)
        counts = recorder.snapshot()
        observations[label] = {"calls": 300, "p50_us": statistics.median(costs),
            "p99_us": sorted(costs)[296], "mean_us": statistics.mean(costs),
            "captured": counts["probe_decisions_captured"],
            "envelope_bytes": counts["envelope_bytes"],
            "queue_high_water": counts["queue_high_water"],
            "drops": {k: v for k, v in counts.items() if k.startswith("drop") and v}}
        while not recorder._q.empty():
            recorder._q.get_nowait()
            recorder._q.task_done()
        recorder.close(0)
    return {"observations": observations,
            "limits": "One fixed-order pair, 300 calls per arm; caller timings include target; no export, pacing, calibration, baseline revision, confidence interval or budget qualification."}


if __name__ == "__main__":
    print(json.dumps(measure(), indent=2))
