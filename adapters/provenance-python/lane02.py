"""Synthetic lane-02 driver: ordinary OTLP, native CLI, no database access."""
import argparse
import datetime
import hashlib
import importlib
import json
import math
import subprocess
import sys
import time
import urllib.request
from collections import Counter
from pathlib import Path

from count_instrumentation import count
from lane02_generator import DEVELOPMENT, RESERVED, VARIANTS, ORIGIN_NS, guard, rows
from recorder import Recorder
from scout_probe import Injector, load_config
from spike_support import Serving

CONTRACT_SHA = "6a3716b8b1c21b3167331d0c0e7421ffc529fdfb"


class HarnessClock:
    """Only the harness chooses synthetic wall time; the target has no clock."""
    def __init__(self, case_index):
        self.base = ORIGIN_NS + case_index*60_000_000_000

    def start(self, index):
        return self.base + index*16_666_667


def utc(ns):
    return datetime.datetime.fromtimestamp(ns/1e9, datetime.timezone.utc).isoformat().replace("+00:00", "Z")


def post(endpoint, path, payload):
    request = urllib.request.Request(endpoint+path, json.dumps(payload).encode(), {"Content-Type": "application/json"})
    with urllib.request.urlopen(request, timeout=30) as response:
        return json.loads(response.read())


def quantile(values, q):
    ordered = sorted(values)
    position = q*(len(ordered)-1)
    lo = math.floor(position)
    hi = min(lo+1, len(ordered)-1)
    return ordered[lo]+(position-lo)*(ordered[hi]-ordered[lo])


def rankings(result, variant):
    presentation = result["presentation"]
    mechanisms = ["previous_age"] if variant == VARIANTS[0] else ["previous_age", "current_value"]
    out = {}
    for name, findings in (("legacy_median", result["differences"]),
                           ("median", presentation["candidate_median"]),
                           ("tail", presentation["candidate_tail"])):
        positive = [f for f in findings if f["status"] == "ok" and f["score"] > 0]
        ranks = {f["feature"]: i+1 for i, f in enumerate(positive)}
        out[name] = {"ranked": [{"feature": f["feature"], "score": f["score"]} for f in positive],
                     "positive_candidates": len(positive), "top_1_feature": positive[0]["feature"] if positive else None,
                     "mechanism_ranks": {f: ranks.get(f) for f in mechanisms},
                     "recall": {str(k): sum(ranks.get(f, 999) <= k for f in mechanisms)/len(mechanisms) for k in (1, 3, 5)},
                     "nuisance_above_mechanism": [{"nuisance": f, "mechanism": m} for f in ranks if f.startswith("noise_")
                                                 for m in mechanisms if ranks[f] < ranks.get(m, 999)]}
        groups = {}
        for f in findings:
            groups.setdefault(f["score"], []).append(f["feature"])
        out[name]["exact_score_ties"] = [{"score": score, "features": fields} for score, fields in groups.items()
                                           if len(fields) > 1]
    return out


def raw_numeric(manifest, profile):
    def select(record, path):
        value = record
        for key in path.split("."):
            value = value[int(key)] if isinstance(value, list) else value[key]
        return value
    terms = {}
    for feature in profile["numeric"]:
        arms = []
        for arm in ("problem", "reference"):
            values = []
            for entry in manifest:
                if entry["label"] == arm:
                    value = select(entry, feature["selector"])
                    if feature.get("subtract"):
                        value -= select(entry, feature["subtract"])
                    values.append(value)
            arms.append(values)
        p, r = arms
        x, m = quantile(p, .5), quantile(r, .5)
        mad = quantile([abs(v-m) for v in r], .5)
        scale = max(1.4826*mad, feature["epsilon"])
        e = sum(v >= x if x >= m else v <= x for v in r)
        terms[feature["name"]] = {"problem_median": x, "reference_median": m, "mad": mad,
                                   "scale": scale, "z": abs(x-m)/scale, "e": e,
                                   "epsilon": feature["epsilon"], "floor": feature["floor"],
                                   "below_floor": abs(x-m) < feature["floor"]}
    return terms


def analytic_checks(result, manifest, variant):
    failures = []
    by_feature = {f["feature"]: f for f in result["presentation"]["candidate_tail"]}
    age_p = 400 if variant == VARIANTS[0] else 0 if variant == "within-range-subgroup" else 80
    current_p = 0 if variant == VARIANTS[0] else 320
    for feature, expected in (("previous_age", age_p), ("current_value", current_p)):
        f = by_feature[feature]
        t = f["terms"]
        if (f["problem_observed"], f["reference_observed"], t["problem_outside"], t["reference_outside"]) != (400, 400, expected, 0):
            failures.append(f"deterministic {feature} counts mismatch")
        if abs(t["score"]-expected/402) > 1e-12:
            failures.append(f"deterministic {feature} score mismatch")
    arms = {x["run_id"]: x["label"] for x in manifest}
    invariants = {arm: Counter() for arm in ("problem", "reference")}
    for item in result["invariants"]:
        invariants[arms[item["run_id"]]][item["status"]] += 1
    expected = ({"violated": 400}, {"satisfied": 400}) if variant == VARIANTS[0] else (
        {"violated": 80, "not_applicable": 320}, {"satisfied": 80, "not_applicable": 320})
    for index, arm in enumerate(("problem", "reference")):
        if dict(invariants[arm]) != expected[index]:
            failures.append(f"deterministic {arm} invariant mismatch")
    for name, f in by_feature.items():
        if name.startswith("noise_") and not (variant == "moderate-decoy" and name == "noise_0") and f["score"] != 0:
            failures.append(f"nuisance tail expectation mismatch: {name}")
    return failures, invariants


def verify_freeze(root, path):
    if path is None:
        raise ValueError("evaluation runner requires committed source-freeze manifest")
    manifest = json.loads(path.read_text())
    required = {"adapters/provenance-python/fixtures/lane02_target.py", "adapters/provenance-python/lane02_generator.py",
                "adapters/provenance-python/lane02.py", "benchmark/problems/lane-02/profile.json", "internal/cohort/scoring.go",
                "internal/cohort/tail.go", "adapters/provenance-python/recorder.py", "adapters/provenance-python/scout_probe.py"}
    if manifest.get("contract_sha") != CONTRACT_SHA or not required.issubset(manifest.get("files", {})):
        raise ValueError("wrong or incomplete source freeze")
    committed = subprocess.check_output(["git", "show", "HEAD:"+path.resolve().relative_to(root).as_posix()], cwd=root)
    if committed != path.read_bytes().replace(b"\r\n", b"\n"):
        raise ValueError("source-freeze manifest is not committed unchanged")
    for name, expected in manifest["files"].items():
        actual = hashlib.sha256((root/name).read_bytes().replace(b"\r\n", b"\n")).hexdigest()
        if actual != expected:
            raise ValueError(f"source freeze mismatch: {name}")
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path, required=True)
    parser.add_argument("--seeds", type=int, nargs="+", default=list(DEVELOPMENT))
    parser.add_argument("--evaluation-runner", action="store_true")
    parser.add_argument("--source-freeze", type=Path)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[2]
    for seed in args.seeds:
        guard(seed, args.evaluation_runner)
    if args.evaluation_runner and tuple(args.seeds) != RESERVED:
        raise ValueError("evaluation must attempt the exact full reserved seed order")
    freeze_hash = verify_freeze(root, args.source_freeze) if args.evaluation_runner else None
    if args.evaluation_runner:
        # A second command cannot silently replay holdouts into another output
        # directory. Never remove this ledger to retry a failed evaluation.
        with (root/".cache/lane02-evaluation.lock").open("x", encoding="utf-8") as ledger:
            json.dump({"contract_sha": CONTRACT_SHA, "source_freeze_sha256": freeze_hash,
                       "seeds": args.seeds, "output": str(args.out)}, ledger)
    # Fresh directories preserve every failed attempt; never overwrite a prior run.
    args.out.mkdir(parents=True, exist_ok=False)
    target = Path(__file__).parent/"fixtures/lane02_target.py"
    baseline = args.out/"target-before.py"
    baseline.write_bytes(target.read_bytes())
    cfg_path = root/"benchmark/problems/lane-02/signalscout.yaml"
    cfg = load_config(cfg_path)
    profile = root/"benchmark/problems/lane-02/profile.json"
    report = {"synthetic_only": True, "contract_sha": CONTRACT_SHA, "source_freeze_sha256": freeze_hash,
              "evaluation": args.evaluation_runner, "cases": [], "status": "attempted",
              "planned_cases": [{"seed": seed, "variant": variant} for seed in args.seeds for variant in VARIANTS]}
    def save():
        (args.out/"report.json").write_text(json.dumps(report, indent=2)+"\n", encoding="utf-8")
    save()
    try:
        with Serving(args.out/"fresh.db") as server:
            recorder = Recorder(server.endpoint, service=cfg["service"],
                                capture_fields={"gap", "current_value", "history", "previous_frame", "frame", "noise", "output"})
            injector = Injector(recorder, cfg["probes"])
            injector.install()
            try:
                sys.path.insert(0, str(target.parent))
                app = importlib.import_module("lane02_target")
                for seed_index, seed in enumerate(args.seeds):
                    for variant_index, variant in enumerate(VARIANTS):
                        clock = HarnessClock(seed_index*len(VARIANTS)+variant_index)
                        name = f"lane-02-{seed}-{variant}"
                        case = {"seed": seed, "variant": variant, "status": "attempted"}
                        report["cases"].append(case)
                        save()
                        before = recorder.snapshot()
                        manifest, overhead, enter_cost, exit_cost = [], [], [], []
                        scheduled = time.perf_counter()
                        for index, row in enumerate(rows(seed, variant, evaluation_runner=args.evaluation_runner)):
                            time.sleep(max(0, scheduled+index/60-time.perf_counter()))
                            manager = recorder.run("lane02_target.decide", run_id=row["run_id"], start_ns=clock.start(index),
                                                   attributes={"signalscout.session.id": row["session"]})
                            begin = time.perf_counter_ns()
                            manager.__enter__()
                            enter_cost.append(time.perf_counter_ns()-begin)
                            try:
                                returned = app.decide(**row["inputs"])
                                overhead.append(app.decide.__signalscout_probe__.last["wrapper_ns"])
                            finally:
                                begin = time.perf_counter_ns()
                                manager.__exit__(*sys.exc_info())
                                exit_cost.append(time.perf_counter_ns()-begin)
                            label = "problem" if returned["lane"] != "right" else "reference"
                            manifest.append({"run_id": row["run_id"], "label": label, "inputs": row["inputs"],
                                             "output": returned, "start_ns": clock.start(index), "session": row["session"]})
                        begin = time.perf_counter_ns()
                        drained = recorder.flush(10)
                        case["flush_ns"] = time.perf_counter_ns()-begin
                        after = recorder.snapshot()
                        case["capture"] = {k: v-before.get(k, 0) for k, v in after.items()
                                           if isinstance(v, int) and not isinstance(v, bool) and k not in ("capacity", "queue_size")}
                        case["manifest"] = manifest
                        case["overhead_ns"] = {"wrapper_p50": quantile(overhead, .5), "wrapper_p99": quantile(overhead, .99),
                                               "run_enter_p50": quantile(enter_cost, .5), "run_exit_p50": quantile(exit_cost, .5),
                                               "run_enter_p99": quantile(enter_cost, .99), "run_exit_p99": quantile(exit_cost, .99)}
                        case["status"] = "incomplete"
                        save()
                        if not drained or any(v for k, v in case["capture"].items() if k.startswith("drop") or k.endswith("failed")):
                            raise RuntimeError("transport/capture incomplete; retain case, no zero association")
                        if case["capture"].get("serialized") != 800 or case["capture"].get("captured_args") != 4800:
                            raise RuntimeError("capture completeness mismatch")
                        for entry in manifest:
                            post(server.endpoint, "/api/annotations", {"id": "annotation:"+entry["run_id"]+"/oracle-2",
                                 "run_id": entry["run_id"], "dimension": name, "value": entry["label"], "source": "synthetic-oracle-v2",
                                 "reviewer": "synthetic-reviewer-v2", "synthetic": True, "rubric_id": "lane-02-expected-right",
                                 "rubric_version": "1", "created_at": clock.base+60_000_000_000, "evidence_ids": []})
                        selections = {}
                        for arm in ("problem", "reference"):
                            command = [str(server.binary), "cohort", "create", name+"-"+arm, "--endpoint", server.endpoint,
                                       "--profile", str(profile), "--from", utc(clock.base), "--to", utc(clock.base+60_000_000_000),
                                       "--limit", "800", "--dimension", name, "--value", arm]
                            selections[arm] = json.loads(subprocess.check_output(command))
                        command = [str(server.binary), "--db", str(server.database), "explain-cohort", selections["problem"]["id"], selections["reference"]["id"]]
                        result = json.loads(subprocess.check_output(command+["--json"]))
                        case["result"] = result
                        case["rankings"] = rankings(result, variant)
                        case["raw_numeric"] = raw_numeric(manifest, json.loads(profile.read_text()))
                        case["contract_premise_misses"] = [{"feature": f["feature"], "problem_outside": f["terms"]["problem_outside"],
                                                             "reference_outside": f["terms"]["reference_outside"]}
                                                            for f in result["presentation"]["candidate_tail"]
                                                            if variant in ("benign-minority-tail", "unrelated-nuisance")
                                                            and f["feature"].startswith("noise_")
                                                            and (f["terms"]["problem_outside"] or f["terms"]["reference_outside"])]
                        failures, inv = analytic_checks(result, manifest, variant)
                        case["expectation_mismatches"], case["invariants_by_arm"] = failures, inv
                        links = {u for f in result["differences"]+result["presentation"]["candidate_tail"] for u in f["evidence_urls"]}
                        for link in links:
                            path, anchor = link.split("#")
                            with urllib.request.urlopen(server.endpoint+path, timeout=20) as response:
                                if 'id="'+anchor+'"' not in response.read().decode():
                                    raise RuntimeError("native evidence anchor missing")
                        case["verified_evidence_links"] = len(links)
                        (args.out/(name+".txt")).write_bytes(subprocess.check_output(command))
                        case["status"] = "completed" if not failures else "failed"
                        save()
                        print(seed, variant, case["rankings"]["median"]["mechanism_ranks"],
                              case["rankings"]["tail"]["mechanism_ranks"], failures, flush=True)
                        if failures:
                            raise RuntimeError("analytic expectation mismatch; stopped, retain results")
                report["doctor"] = injector.doctor()
            finally:
                injector.uninstall()
                recorder.close(10)
        report["instrumentation"] = count(target, baseline, cfg_path)
        report["status"] = "completed"
    except BaseException as error:
        report["status"], report["error"] = "failed", type(error).__name__+": "+str(error)
        if report["cases"] and report["cases"][-1]["status"] in ("attempted", "incomplete"):
            case = report["cases"][-1]
            if case["status"] == "attempted":
                case["status"] = "failed"
            case["error"] = report["error"]
        raise
    finally:
        report["case_counts"] = dict(Counter(case["status"] for case in report["cases"]))
        report["case_counts"]["unattempted"] = len(report["planned_cases"])-len(report["cases"])
        save()


if __name__ == "__main__":
    main()
