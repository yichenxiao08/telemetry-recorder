"""Development/toy validation only. Separate processes prevent exporter overlap."""
import argparse
import hashlib
import importlib.util
import json
import socket
import subprocess
import sys
import time
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
REPO = HERE.parents[1]


def save(path, value):
    path.write_text(json.dumps(value, indent=2)+"\n", encoding="utf-8")


def acceptance(out):
    sys.path.insert(0, str(HERE/"tests"))
    from test_recorder_retry import ExplicitAcceptance
    evidence = []
    class Result(unittest.TextTestResult):
        def addSuccess(self, test):
            super().addSuccess(test)
            evidence.append({"test": test.id(), "status": "passed", **test.evidence})
    suite = unittest.defaultTestLoader.loadTestsFromTestCase(ExplicitAcceptance)
    result = unittest.TextTestRunner(verbosity=2, resultclass=Result).run(suite)
    save(out/"acceptance.json", {"passed": result.wasSuccessful(), "tests": evidence,
                               "failures": [(str(t), e) for t, e in result.failures+result.errors]})
    return 0 if result.wasSuccessful() else 1


def development(out):
    runs = []
    for repetition in range(1, 11):
        name = f"default-{repetition:02}"
        print("START", name, flush=True)
        begin = time.monotonic()
        with (out/(name+".log")).open("w", encoding="utf-8") as log:
            status = subprocess.run([sys.executable, str(HERE/"capture_loss.py"),
                                     "--out", str(out/name)], cwd=HERE, stdout=log,
                                    stderr=subprocess.STDOUT, check=False).returncode
        runs.append({"name": name, "exit_code": status, "seconds": time.monotonic()-begin})
        save(out/"runs.json", runs)
        print("END", runs[-1], flush=True)
        # Development repetitions are predeclared distinct runs, never a retry
        # of a failed evaluation. Every failure and unattempted case is retained.
    return int(any(r["exit_code"] for r in runs))


def cost_child(args):
    import measure
    from spike_support import Serving
    if args.reference:
        spec = importlib.util.spec_from_file_location("reference_recorder", args.reference)
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        measure.Recorder = module.Recorder
    with Serving(args.out/"fresh.db") as server, socket.socket() as down:
        down.bind(("127.0.0.1", 0))
        endpoint = f"http://127.0.0.1:{down.getsockname()[1]}" if args.condition == "unreachable" else server.endpoint
        result = measure.trial(endpoint, args.condition, args.mode, args.trial)
    result["recorder_source"] = "checkpoint-a" if args.reference else "fixed"
    result["payload_sha256"] = hashlib.sha256((HERE/"fixtures/benchmark_target.py").read_bytes()).hexdigest()
    save(args.out/"trial.json", result)
    return 0


def costs(out, reference):
    results = []
    # Predeclare alternating order, identical payload/method and fresh stores.
    # Each process exits before the next, including all outage daemon workers.
    for condition in ("reachable", "unreachable", "saturated"):
        for mode in ("60hz", "tight"):
            for trial in range(1, 6):
                versions = ("reference", "fixed") if trial % 2 else ("fixed", "reference")
                for version in versions:
                    name = f"{condition}-{mode}-{trial}-{version}"
                    print("START", name, flush=True)
                    command = [sys.executable, str(Path(__file__).resolve()), "cost-child", "--out", str(out/name),
                               "--condition", condition, "--mode", mode, "--trial", str(trial)]
                    if version == "reference":
                        command += ["--reference", str(reference)]
                    with (out/(name+".log")).open("w", encoding="utf-8") as log:
                        status = subprocess.run(command, cwd=HERE, stdout=log, stderr=subprocess.STDOUT, check=False).returncode
                    record = {"name": name, "exit_code": status}
                    if (out/name/"trial.json").exists():
                        record["result"] = json.loads((out/name/"trial.json").read_text())
                    results.append(record)
                    save(out/"trials.json", results)
                    print("END", name, status, flush=True)
    return int(any(r["exit_code"] for r in results))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("action", choices=("acceptance", "development", "costs", "cost-child"))
    parser.add_argument("--out", type=Path, required=True)
    parser.add_argument("--reference", type=Path)
    parser.add_argument("--condition", choices=("reachable", "unreachable", "saturated"))
    parser.add_argument("--mode", choices=("60hz", "tight"))
    parser.add_argument("--trial", type=int)
    args = parser.parse_args()
    args.out = args.out.resolve()
    if args.reference:
        args.reference = args.reference.resolve()
    if args.action == "costs" and not args.reference:
        parser.error("paired comparison requires the immutable Checkpoint A recorder source")
    args.out.mkdir(parents=True, exist_ok=False)
    if args.action == "acceptance":
        return acceptance(args.out)
    if args.action == "development":
        return development(args.out)
    if args.action == "costs":
        return costs(args.out, args.reference)
    return cost_child(args)


if __name__ == "__main__":
    raise SystemExit(main())
