"""Publish every diagnostic attempt; do not rerun targets or alter raw timings."""
import argparse
from collections import defaultdict
import gzip
import hashlib
import json
from pathlib import Path
import shutil
from statistics import median

from profile_wrapper import PHASES, percentile, stats

ROOT = Path(__file__).resolve().parents[2]


def portable(value):
    if isinstance(value, dict):
        return {k: portable(v) for k, v in value.items()}
    if isinstance(value, list):
        return [portable(v) for v in value]
    if isinstance(value, str):
        for root in (str(ROOT), str(ROOT).replace("\\", "/"), str(ROOT).replace("\\", "\\\\")):
            value = value.replace(root, "<repo>")
    return value


def dump(path, value):
    path.write_text(json.dumps(portable(value), indent=2)+"\n", encoding="utf-8")


def group_key(s):
    c = s["configuration"]
    return tuple(c[k] for k in ("payload", "mode", "observer", "feature", "factor"))


def f(value):
    return f"{value:.2f}"


def tables(summaries):
    groups = defaultdict(list)
    for s in summaries:
        groups[group_key(s)].append(s)
    lines = ["# Wrapper Stage 1: every trial group", "",
             "Each cell is the median of trial p50/p99s, followed by the min–max across trials.",
             "Instrumented, calibration-subtracted estimates are separate from plain controls.",
             "Only plain controls measure total cost without inserted phase scopes; both observers still record worker intervals.",
             "cProfile is a separate run and is not a budget validation.", "",
             "| Payload / mode / observer / capture / factor | Trials | p50 us (range) | p99 us (range) | Passes | Drops / failed / unknown |",
             "|---|---:|---:|---:|---:|---:|"]
    for key, trials in sorted(groups.items()):
        p50 = [s["adjusted"]["p50_us"] for s in trials]
        p99 = [s["adjusted"]["p99_us"] for s in trials]
        counts = [sum(s.get("counts", {}).get(k, 0) for s in trials) for k in ("dropped", "failed", "unknown")]
        lines.append(f"| {' / '.join(key)} | {len(trials)} | {f(median(p50))} ({f(min(p50))}–{f(max(p50))}) | "
                     f"{f(median(p99))} ({f(min(p99))}–{f(max(p99))}) | {sum(s['budget_met'] for s in trials)}/{len(trials)} | {' / '.join(map(str, counts))} |")
    for payload in ("fixed", "lane-inputs", "lane-driver"):
        for mode in ("60hz", "tight"):
            trials = groups.get((payload, mode, "phase", "full", "default"), [])
            if not trials:
                continue
            lines += ["", f"## Exclusive phases: {payload}, {mode}, full, default", "",
                      "Quantiles are medians of trial quantiles. Fractions divide summed exclusive times by summed adjusted call time; they need not add to 100% because residual and negatives are retained.", "",
                      "| Phase | p50 us | p99 us | Fraction |", "|---|---:|---:|---:|"]
            denominator = sum(t["adjusted"]["sum_ns"] for t in trials)
            for phase in PHASES:
                p50 = median(t["phases"][phase]["p50_us"] for t in trials)
                p99 = median(t["phases"][phase]["p99_us"] for t in trials)
                fraction = sum(t["phases"][phase]["sum_ns"] for t in trials)/denominator
                lines.append(f"| {phase} | {f(p50)} | {f(p99)} | {fraction:.1%} |")
            lines += [f"| Unattributed residual | {f(median(t['residual']['p50_us'] for t in trials))} | {f(median(t['residual']['p99_us'] for t in trials))} | "
                      f"{sum(t['residual']['sum_ns'] for t in trials)/denominator:.1%} |"]
    lines += ["", "## Individual trials (including every budget miss)", "",
              "| Trial | Calls | Raw p50 / p99 us | Adjusted p50 / p99 us | Budget | Ack / failed / drops / unknown |",
              "|---|---:|---:|---:|---|---:|"]
    for s in summaries:
        c = s["configuration"]
        name = "/".join(str(c[k]) for k in ("payload", "mode", "observer", "feature", "factor", "trial"))
        counts = [s.get("counts", {}).get(k, 0) for k in ("acknowledged", "failed", "dropped", "unknown")]
        lines.append(f"| {name} | {s['calls']} | {f(s['raw']['p50_us'])} / {f(s['raw']['p99_us'])} | "
                     f"{f(s['adjusted']['p50_us'])} / {f(s['adjusted']['p99_us'])} | {'PASS' if s['budget_met'] else 'MISS'} | {' / '.join(map(str, counts))} |")
    return "\n".join(lines)+"\n"


def raw_analysis(raw):
    samples = raw["samples"]
    threshold = percentile([s["raw_total_ns"] for s in samples], .99)
    tails = [s for s in samples if s["raw_total_ns"] >= threshold]
    fractions = {}
    for kind in ("gc", "_encode", "_send", "gc_or_encode"):
        intervals = sorted((a, b) for k, a, b, _ in raw["intervals"] if k == kind or kind == "gc_or_encode" and k in ("gc", "_encode"))
        # Overlapping intervals can end out of order: retain the maximum end.
        merged = []
        for a, b in intervals:
            if merged and a <= merged[-1][1]:
                merged[-1] = (merged[-1][0], max(merged[-1][1], b))
            else:
                merged.append((a, b))
        def count(items):
            index, matched = 0, 0
            for s in items:
                while index < len(merged) and merged[index][1] < s["started_ns"]:
                    index += 1
                if index < len(merged) and merged[index][0] <= s["stopped_ns"]:
                    matched += 1
            return matched
        fractions[kind] = {"all": count(samples), "tail": count(tails)}
    queue_groups = {}
    for name, subset in (("nonfull_before", [s for s in samples if s.get("queue_depth_before", 0) < 256]),
                         ("full_before", [s for s in samples if s.get("queue_depth_before", 0) == 256])):
        queue_groups[name] = {"calls": len(subset), **stats([s["adjusted_total_ns"] for s in subset])}
    return {"calls": len(samples), "tail_calls": len(tails), "overlap": fractions,
            "tail_off_cpu": stats([max(0, s["stopped_ns"]-s["started_ns"]-s["caller_thread_cpu_ns"]) for s in tails]),
            "queue_groups": queue_groups,
            "clipped_phase_sum_ns": {p: sum(max(0, s["adjusted_exclusive_ns"].get(p, 0)) for s in samples) for p in PHASES},
            "clipped_adjusted_total": stats([max(0, s["adjusted_total_ns"]) for s in samples])}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--out", type=Path, required=True)
    parser.add_argument("--allow-interrupted", action="store_true")
    args = parser.parse_args()
    args.out.mkdir(parents=True, exist_ok=False)
    manifest, summaries, analyses, cases = [], [], {}, []
    progress = json.loads((args.source/"progress.json").read_text())
    plan = json.loads((args.source/"plan.json").read_text())
    if len(progress) != len(plan) and not args.allow_interrupted:
        raise ValueError("suite not finished: do not publish partial data as complete")
    for path in sorted(args.source.rglob("*")):
        if not path.is_file() or path.suffix in (".db", ".db-wal", ".db-shm") or ".db" in path.name:
            continue
        relative = path.relative_to(args.source)
        data = path.read_bytes()
        manifest.append({"path": str(relative).replace("\\", "/"), "original_sha256": hashlib.sha256(data).hexdigest(), "original_bytes": len(data)})
        target = args.out/relative
        target.parent.mkdir(parents=True, exist_ok=True)
        if path.name == "summary.json":
            summary = json.loads(data)
            summaries.append(summary)
            dump(target, summary)
        elif path.name == "raw.json.gz":
            shutil.copyfile(path, target)  # Exact compressed raw timing bytes.
            with gzip.open(path, "rt", encoding="utf-8") as stream:
                analyses[str(relative.parent)] = raw_analysis(json.load(stream))
        elif path.suffix == ".json" and len(data) > 100000:
            with gzip.open(str(target)+".gz", "wt", encoding="utf-8") as stream:
                json.dump(portable(json.loads(data)), stream)
            if path.name == "report.json":
                report = json.loads(data)
                cases.append({"path": str(relative), "status": report["status"], "counts": report["case_counts"],
                              "cases": [{k: c.get(k) for k in ("seed", "variant", "status", "error", "capture", "overhead_ns", "expectation_mismatches", "verified_evidence_links")}
                                        for c in report["cases"]]})
        elif path.suffix == ".json":
            dump(target, json.loads(data))
        else:
            shutil.copyfile(path, target)  # Keep failures, logs and raw pstats.
    dump(args.out/"raw-manifest.json", manifest)
    dump(args.out/"all-summaries.json", summaries)
    dump(args.out/"tail-analysis.json", analyses)
    dump(args.out/"driver-case-status.json", cases)
    dump(args.out/"publication.json", {"planned": len(plan), "recorded_exits": len(progress),
         "summaries": len(summaries), "interrupted": len(progress) != len(plan),
         "original_timing_gzip_copied_byte_identically": True,
         "json_metadata_paths_normalized": True, "databases_remain_local": True,
         "raw_pstats_paths_unmodified": True})
    (args.out/"report.md").write_text(tables(summaries), encoding="utf-8")
    print(len(summaries), "summaries;", len(cases), "driver attempts; no target rerun")


if __name__ == "__main__":
    main()
