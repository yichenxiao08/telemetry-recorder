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


def profile_readout(source):
    lines = ["# Separate cProfile readout", "",
             "One separate process per payload/mode; these are not five-trial budget measurements.",
             "Cumulative children overlap their parents. Do not sum the whole table.",
             "Raw profiles also contain background network activity; the caller-only wrapper is the denominator here.",
             "Function fractions are diagnostic proxies, not promised optimization savings.", "",
             "| Profile | Caller-only function | Calls | Self ms | Cumulative ms | Fraction of probe wrapper cumulative |",
             "|---|---|---:|---:|---:|---:|"]
    for path in sorted(source.glob("*-cprofile-*/profile-functions.json")):
        functions = json.loads(path.read_text())
        root = next(v["cumulative_seconds"] for v in functions if v["file"] == "scout_probe.py" and v["function"] == "wrapped")
        for item in functions:
            if (item["file"], item["function"]) not in {
                ("scout_probe.py", "wrapped"), ("recorder.py", "record_decision"),
                ("recorder.py", "bounded"), ("__init__.py", "dumps"),
                ("encoder.py", "iterencode"), ("inspect.py", "bind"),
                ("inspect.py", "apply_defaults"), ("recorder.py", "__enter__"),
                ("recorder.py", "__exit__"), ("recorder.py", "_enqueue"),
                ("recorder.py", "_span") }:
                continue
            lines.append(f"| {path.parent.name} | {item['file']}:{item['function']} | {item['calls']} | "
                         f"{item['self_seconds']*1000:.2f} | {item['cumulative_seconds']*1000:.2f} | {item['cumulative_seconds']/root:.1%} |")
    return "\n".join(lines)+"\n"


def context_tail_readout(summaries, analyses):
    groups = defaultdict(list)
    for s in summaries:
        key = Path(s["configuration"]["out"]).name
        if key in analyses:
            groups[group_key(s)].append(analyses[key])
    lines = ["# Context and tail observations", "",
             "Context values below are raw inclusive scope timings: they include child timing scopes.",
             "Corrected exclusive context values are in report.md; neither is an unobserved production cost.",
             "Tail means calls at or above each trial's raw p99 (ties retained). Overlap is not causal attribution.",
             "Thread CPU accounting can be too coarse for one call; zero CPU is not proof of idle time.", "",
             "| Group | Enter p50/p99 us | Exit p50/p99 us | Tail calls | Tail GC / encode / either / send overlap | Zero thread CPU calls |",
             "|---|---:|---:|---:|---:|---:|"]
    for key, values in sorted(groups.items()):
        contexts = []
        for p in ("run_context_enter", "run_context_exit"):
            contexts.append("not measured" if key[2] != "phase" or key[3] == "bare" else
                            " / ".join(f(median(v["context_inclusive"][p][q] for v in values)) for q in ("p50_us", "p99_us")))
        tails = sum(v["tail_calls"] for v in values)
        overlaps = [sum(v["overlap"][k]["tail"] for v in values) for k in ("gc", "_encode", "gc_or_encode", "_send")]
        lines.append(f"| {' / '.join(key)} | {contexts[0]} | {contexts[1]} | {tails} | "
                     f"{' / '.join(map(str, overlaps))} | {sum(v['zero_thread_cpu_calls'] for v in values)}/{sum(v['calls'] for v in values)} |")
    lines += ["", "## No-worker queue stratification", "",
              "Nonfull before a call does not ensure two free slots. These are experimental saturation costs, not successful recording costs.", "",
              "| Trial | Queue group | Calls | Adjusted p50 / p99 us |", "|---|---|---:|---:|"]
    for name, a in sorted(analyses.items()):
        if "worker-idle" not in name:
            continue
        for group, s in a["queue_groups"].items():
            if s["calls"]:
                lines.append(f"| {name} | {group} | {s['calls']} | {f(s['p50_us'])} / {f(s['p99_us'])} |")
    return "\n".join(lines)+"\n"


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
            "context_inclusive": {p: stats([s.get("inclusive_ns", {}).get(p, 0) for s in samples])
                                  for p in ("run_context_enter", "run_context_exit")},
            "zero_thread_cpu_calls": sum(s["caller_thread_cpu_ns"] == 0 for s in samples),
            "thread_cpu_exceeds_wall_calls": sum(s["caller_thread_cpu_ns"] > s["stopped_ns"]-s["started_ns"] for s in samples),
            "tail_off_cpu": stats([max(0, s["stopped_ns"]-s["started_ns"]-s["caller_thread_cpu_ns"]) for s in tails]),
            "queue_groups": queue_groups,
            "clipped_phase_sum_ns": {p: sum(max(0, s["adjusted_exclusive_ns"].get(p, 0)) for s in samples) for p in PHASES},
            "clipped_adjusted_total": stats([max(0, s["adjusted_total_ns"]) for s in samples])}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--out", type=Path, required=True)
    parser.add_argument("--allow-interrupted", action="store_true")
    parser.add_argument("--single-child", action="store_true")
    args = parser.parse_args()
    args.out.mkdir(parents=True, exist_ok=False)
    manifest, summaries, analyses, cases = [], [], {}, []
    if args.single_child:
        plan = [{"single_child_smoke": True}]
        progress = [{"exit_code": 0 if (args.source/"summary.json").exists() else 1}]
    else:
        progress = json.loads((args.source/"progress.json").read_text())
        plan = json.loads((args.source/"plan.json").read_text())
    if len(progress) != len(plan) and not args.allow_interrupted:
        raise ValueError("suite not finished: do not publish partial data as complete")
    for path in sorted(args.source.rglob("*")):
        if not path.is_file() or path.name.endswith((".db", ".db-wal", ".db-shm")):
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
                raw = json.load(stream)
            analysis = raw_analysis(raw)
            if relative.parent.name.startswith("lane-inputs-"):
                from lane02_generator import VARIANTS
                per_case = []
                for index in range(12):
                    subset = [s for s in raw["samples"] if (s["label"] % 9600)//800 == index]
                    per_case.append({"seed": (101, 103)[index//6], "variant": VARIANTS[index % 6],
                                     "calls": len(subset), "raw": stats([s["raw_total_ns"] for s in subset]),
                                     "adjusted": stats([s["adjusted_total_ns"] for s in subset])})
                analysis["development_input_cases"] = per_case
            analyses[str(relative.parent)] = analysis
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
         "single_child_smoke": args.single_child,
         "summaries": len(summaries), "interrupted": len(progress) != len(plan),
         "original_timing_gzip_copied_byte_identically": True,
         "json_metadata_paths_normalized": True, "databases_remain_local": True,
         "raw_pstats_paths_unmodified": True})
    (args.out/"report.md").write_text(tables(summaries), encoding="utf-8")
    (args.out/"cprofile-readout.md").write_text(profile_readout(args.source), encoding="utf-8")
    (args.out/"context-and-tail.md").write_text(context_tail_readout(summaries, analyses), encoding="utf-8")
    print(len(summaries), "summaries;", len(cases), "driver attempts; no target rerun")


if __name__ == "__main__":
    main()
