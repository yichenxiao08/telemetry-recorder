"""Publish saved development/toy measurements; never generate data or run engines."""
import argparse
import copy
import gzip
import hashlib
import json
from collections import Counter
from pathlib import Path
from statistics import median


def read(path):
    return json.loads(path.read_text(encoding="utf-8-sig"))


def portable_metadata(data, root):
    # JSON backslashes are escaped, unlike terminal/listener logs.
    escaped = json.dumps(root.decode())[1:-1].encode()
    return data.replace(escaped, b"<repo>").replace(root, b"<repo>").replace(root.replace(b"\\", b"/"), b"<repo>")


def canonical(case):
    # Initial strict projection, deliberately retained after discovering that
    # fresh event timestamps also change evidence hashes. Never call it a pass
    # by stripping additional fields; semantic comparison is separate below.
    value = {k: copy.deepcopy(case[k]) for k in ("result", "rankings", "raw_numeric",
             "manifest", "expectation_mismatches", "invariants_by_arm", "verified_evidence_links")}
    for arm in ("problem", "reference"):
        del value["result"][arm]["created_at"]
    return json.dumps(value, sort_keys=True, separators=(",", ":")).encode()


def analytic(case):
    """Compare all analytic content with verified evidence mapped to its run."""
    value = json.loads(canonical(case))
    result = value["result"]
    evidence_runs = {}
    for arm in ("problem", "reference"):
        for run, evidence in result[arm]["decision_ids"].items():
            if evidence in evidence_runs and evidence_runs[evidence] != run:
                raise ValueError("evidence belongs to multiple runs")
            evidence_runs[evidence] = run
    def evidence_id(evidence):
        if evidence not in evidence_runs:
            raise ValueError("unmapped evidence, not an ignorable replay difference")
        return "decision-for-run:"+evidence_runs[evidence]
    urls = {"/run?id="+run+"#e-"+hashlib.sha256(evidence.encode()).hexdigest(): evidence_id(evidence)
            for evidence, run in evidence_runs.items()}
    for arm in ("problem", "reference"):
        result[arm]["decision_ids"] = {run: evidence_id(evidence) for run, evidence in result[arm]["decision_ids"].items()}
    for item in result.get("invariants", []):
        item["evidence_id"] = evidence_id(item["evidence_id"])
    findings = list(result.get("differences", []))
    for key in ("the_problem", "candidate_median", "candidate_tail"):
        findings += result.get("presentation", {}).get(key, [])
    for item in findings:
        item["evidence_ids"] = [evidence_id(e) for e in item["evidence_ids"]]
        # Exact original URL, including anchor hash and run, must be valid.
        item["evidence_urls"] = [urls[url] for url in item["evidence_urls"]]
    return json.dumps(value, sort_keys=True, separators=(",", ":")).encode()


def accounting(counts):
    value = {k: counts.get(k, 0) for k in ("attempted", "acknowledged", "failed", "dropped", "unknown")}
    value["identity_pass"] = value["attempted"] == sum(value[k] for k in ("acknowledged", "failed", "dropped", "unknown"))
    value["nonnegative"] = all(value[k] >= 0 for k in ("attempted", "acknowledged", "failed", "dropped", "unknown"))
    value["legacy_export_failed"] = counts.get("export_failed", 0)
    value["drop_invalid"] = counts.get("drop_invalid", 0)
    return value


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=Path, required=True)
    parser.add_argument("--reference", type=Path, required=True)
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    args.out.mkdir(parents=True, exist_ok=False)
    baseline_cases = read(args.reference)["cases"]
    baseline = {(c["seed"], c["variant"]): canonical(c) for c in baseline_cases}
    baseline_analytic = {(c["seed"], c["variant"]): analytic(c) for c in baseline_cases}
    acceptance_path = args.input/"acceptance-final/acceptance.json"
    if not acceptance_path.exists():
        acceptance_path = args.input/"acceptance/acceptance.json"
    summary = {"synthetic_only": True, "runs": [], "trials": [], "controls": [], "files": [],
               "acceptance": read(acceptance_path),
               "projection": "Strict projection retained: full result, rankings, raw_numeric, manifest, expectation_mismatches, invariants_by_arm, verified_evidence_links; exclude ONLY two cohort creation clocks. Separate analytic projection maps evidence IDs and exact validated URL hashes to their cohort run, preserving all remaining fields, counts, order, scores, terms and values. Original strict publication is retained in ../recorder-capture-loss-fix."}
    cases = ["# All development cases", "", "No reserved seeds; no failed case is a zero association.", "",
             "| Repetition | Seed / variant | Status | Attempted | Ack | Failed | Dropped | Unknown | Identity | Strict / analytic bytes match | Calls/sec | HWM (cumulative) | Wrapper p50 / p99 us | Budget |",
             "|---|---|---|---:|---:|---:|---:|---:|---|---|---:|---:|---|---|"]
    for directory in sorted((args.input/"development").glob("default-*")):
        if not directory.is_dir():
            continue
        report_path, diagnostics_path = directory/"driver/report.json", directory/"diagnostics.json"
        if not report_path.exists():
            summary["runs"].append({"name": directory.name, "status": "failed before report"})
            continue
        report, diagnostics = read(report_path), read(diagnostics_path)
        run = {"name": directory.name, "status": report["status"], "case_counts": report["case_counts"],
               "accounting": accounting(diagnostics["final"]), "high_water": diagnostics["final"]["queue_high_water"],
               "diagnostics": diagnostics["final"]["diagnostics"], "per_case": []}
        for index, case in enumerate(report["cases"]):
            phase = next((p for p in diagnostics["cases"] if p["index"] == index), None)
            counts = accounting(case.get("capture", {}))
            costs = case.get("overhead_ns", {})
            p50, p99 = (costs.get(k, 0)/1000 for k in ("wrapper_p50", "wrapper_p99"))
            projected = canonical(case) if "result" in case else None
            reference = baseline[(case["seed"], case["variant"])]
            analytic_bytes = analytic(case) if "result" in case else None
            analytic_reference = baseline_analytic[(case["seed"], case["variant"])]
            item = {"seed": case["seed"], "variant": case["variant"], "status": case["status"],
                    "accounting": counts, "overhead_ns": costs, "budget_met": bool(costs) and p50 <= 100 and p99 <= 250,
                    "score_count_bytes_match": projected == reference, "reference_sha256": hashlib.sha256(reference).hexdigest(),
                    "replay_sha256": hashlib.sha256(projected).hexdigest() if projected else None,
                    "analytic_bytes_match": analytic_bytes == analytic_reference,
                    "analytic_reference_sha256": hashlib.sha256(analytic_reference).hexdigest(),
                    "analytic_replay_sha256": hashlib.sha256(analytic_bytes).hexdigest() if analytic_bytes else None,
                    "phase": {k: phase[k] for k in ("calls", "wall_ns", "achieved_calls_per_second", "flush_ns", "drained")} if phase else None,
                    "queue_high_water": phase["after"]["queue_high_water"] if phase else None,
                    "expectation_mismatches": case.get("expectation_mismatches")}
            run["per_case"].append(item)
            rate = phase["achieved_calls_per_second"] if phase else 0
            hwm = phase["after"]["queue_high_water"] if phase else "NA"
            cases.append(f'| {directory.name} | {case["seed"]} / {case["variant"]} | {case["status"]} | '
                         f'{counts["attempted"]} | {counts["acknowledged"]} | {counts["failed"]} | {counts["dropped"]} | '
                         f'{counts["unknown"]} | {counts["identity_pass"]} | {item["score_count_bytes_match"]} / {item["analytic_bytes_match"]} | '
                         f'{rate:.3f} | {hwm} | {p50:.3f} / {p99:.3f} | {item["budget_met"]} |')
        summary["runs"].append(run)
    costs = ["# Every overhead trial and miss", "",
             "Same S0a payload/method. Alternating Checkpoint A/fixed order; fresh process/store per trial.", "",
             "| Condition / mode | Trial / source | p50 / p99 us | 100/250 pass | Context enter p50 / p99 us | Context exit p50 / p99 us | Flush s | Attempted / ack / failed / dropped / unknown |",
             "|---|---|---|---|---|---|---:|---|"]
    for trial in read(args.input/"costs/trials.json"):
        summary["trials"].append(trial)
        if "result" not in trial:
            costs.append(f'| {trial["name"]} | FAILED process {trial["exit_code"]} | NA | False | NA | NA | NA | NA |')
            continue
        r = trial["result"]
        if r["recorder_source"] == "fixed":
            a = accounting(r["drained"])
            trial["accounting"] = a
            trial["snapshot_accounting"] = accounting(r["counts"])
            values = " / ".join(str(a[k]) for k in ("attempted", "acknowledged", "failed", "dropped", "unknown"))
        else:
            c = r["counts"]
            values = f'{c.get("attempted", 0)} / {c.get("exported", 0)} / legacy unack={c.get("export_failed", 0)} / drops={sum(c.get(k, 0) for k in ("drop_full", "drop_closed", "drop_cooldown"))}; old unknown not separately classified'
        costs.append(f'| {r["condition"]} / {r["mode"]} | {r["trial"]} / {r["recorder_source"]} | '
                     f'{r["p50_us"]:.3f} / {r["p99_us"]:.3f} | {r["budget_met"]} | '
                     f'{r["context_enter_p50_us"]:.3f} / {r["context_enter_p99_us"]:.3f} | '
                     f'{r["context_exit_p50_us"]:.3f} / {r["context_exit_p99_us"]:.3f} | {r["flush_seconds"]:.6f} | {values} |')
    aggregates = []
    for condition in ("reachable", "unreachable", "saturated"):
        for mode in ("60hz", "tight"):
            for source in ("checkpoint-a", "fixed"):
                rows = [t["result"] for t in summary["trials"] if "result" in t and t["result"]["condition"] == condition
                        and t["result"]["mode"] == mode and t["result"]["recorder_source"] == source]
                aggregates.append({"condition": condition, "mode": mode, "source": source, "trials": len(rows),
                    "passed": sum(r["budget_met"] for r in rows),
                    **{key+"_median": median(r[key] for r in rows) if rows else None for key in
                       ("p50_us", "p99_us", "context_enter_p50_us", "context_enter_p99_us",
                        "context_exit_p50_us", "context_exit_p99_us", "flush_seconds", "mean_envelope_bytes")}})
    summary["cost_aggregates"] = aggregates
    for path in sorted((args.input/"controls").glob("*/report.json")):
        value = read(path)
        summary["controls"].append({"name": value["condition"], "accounting": accounting(value["snapshot"]),
            "flush": value["drained"], "queue_high_water": value["snapshot"]["queue_high_water"],
            "achieved_calls_per_second": value["achieved_calls_per_second"], "wall_seconds": value["wall_seconds"],
            "wrapper_p50_us": value["wrapper_p50_us"], "wrapper_p99_us": value["wrapper_p99_us"],
            "flush_ns": value["flush_ns"], "diagnostics": value["snapshot"]["diagnostics"]})
    # Keep every raw report/sidecar/timeline/log, including initial failing tests.
    # SQLite files stay local; no payloads are introduced into diagnostics.
    root = str(Path(__file__).resolve().parents[2]).encode()
    for path in sorted(args.input.rglob("*")):
        if not path.is_file() or path.name.startswith("publication") or path.suffix not in (".json", ".jsonl", ".txt", ".log", ".py"):
            continue
        data = original = path.read_bytes()
        if path.suffix in (".log", ".txt", ".json") and path.name not in ("report.json", "diagnostics.json", "trial.json", "trials.json", "acceptance.json"):
            data = portable_metadata(data, root)
        relative = path.relative_to(args.input)
        destination = args.out/"raw"/relative.parent/(relative.name+".gz")
        destination.parent.mkdir(parents=True, exist_ok=True)
        with destination.open("wb") as stream:
            with gzip.GzipFile(filename="", fileobj=stream, mode="wb", mtime=0) as compressed:
                compressed.write(data)
        assert gzip.decompress(destination.read_bytes()) == data
        summary["files"].append({"path": relative.as_posix(), "sha256_original": hashlib.sha256(original).hexdigest(),
                                 "sha256_published": hashlib.sha256(data).hexdigest(), "paths_redacted": data != original})
    (args.out/"summary.json").write_text(json.dumps(summary, indent=2)+"\n", encoding="utf-8")
    (args.out/"cases.md").write_text("\n".join(cases)+"\n", encoding="utf-8")
    (args.out/"overhead.md").write_text("\n".join(costs)+"\n", encoding="utf-8")
    lines = ["# Recorder capture-loss fix: measured results", "", "Synthetic development/toy data only; no reserved evaluation rerun.", "",
             "## Separate acceptance results", ""]
    for test in summary["acceptance"]["tests"]:
        lines.append(f'- **{test["acceptance"]}: {test["status"]}**. Flush: `{json.dumps(test["flush"], sort_keys=True)}`.')
        if "committed_requests" in test:
            lines.append(f'  {test["committed_requests"]} real committed requests; {len(test["stored_signal_ids"])} stored signals (one span, one event); identical request SHA-256s.')
    lines += ["", "## Every development repetition", "",
              "| Run | Case statuses | Attempted | Ack | Failed | Dropped | Unknown | Legacy export_failed | HWM / 256 | Identity / nonnegative |",
              "|---|---|---:|---:|---:|---:|---:|---:|---:|---|"]
    for run in summary["runs"]:
        if "accounting" not in run:
            lines.append(f'| {run["name"]} | {run["status"]} | NA | NA | NA | NA | NA | NA | NA | False |')
            continue
        a = run["accounting"]
        lines.append(f'| {run["name"]} | {run["case_counts"]} | {a["attempted"]} | {a["acknowledged"]} | '
                     f'{a["failed"]} | {a["dropped"]} | {a["unknown"]} | {a["legacy_export_failed"]} | {run["high_water"]} | '
                     f'{a["identity_pass"]} / {a["nonnegative"]} |')
    lines += ["", "## Paired five-trial cost aggregates", "",
              "Medians of the five trial quantiles, not pooled call quantiles. Every trial/miss and four-state count is in [overhead.md](overhead.md).", "",
              "| Condition / mode | Source | Trials | p50 / p99 us | Pass | Context enter p50 / p99 us | Context exit p50 / p99 us | Flush s |",
              "|---|---|---:|---|---:|---|---|---:|"]
    for a in aggregates:
        if not a["trials"]:
            continue
        lines.append(f'| {a["condition"]} / {a["mode"]} | {a["source"]} | {a["trials"]} | '
                     f'{a["p50_us_median"]:.3f} / {a["p99_us_median"]:.3f} | {a["passed"]} | '
                     f'{a["context_enter_p50_us_median"]:.3f} / {a["context_enter_p99_us_median"]:.3f} | '
                     f'{a["context_exit_p50_us_median"]:.3f} / {a["context_exit_p99_us_median"]:.3f} | {a["flush_seconds_median"]:.6f} |')
    lines += ["", "## Unchanged toy controls", "",
              "| Control | Attempted | Ack | Failed | Dropped | Unknown | HWM | Retry spans / attempts |",
              "|---|---:|---:|---:|---:|---:|---:|---|"]
    for c in summary["controls"]:
        a = c["accounting"]
        lines.append(f'| {c["name"]} | {a["attempted"]} | {a["acknowledged"]} | {a["failed"]} | '
                     f'{a["dropped"]} | {a["unknown"]} | {c["queue_high_water"]} | see raw attempt diagnostics |')
    lines += ["", "All [120 per-case results](cases.md) retain the failed strict fresh-byte comparison beside the separate analytic comparison, rate, high-water mark and every cost-budget miss. Event timestamps include measured caller offsets, so event hashes/anchors change across fresh captures. Analytic comparison validates each evidence-to-run association and exact URL hash before mapping it to a stable run identity; no score/count/value/coverage is dropped. The original strict publication is preserved in ../recorder-capture-loss-fix. Raw JSON reports/sidecars/timelines are byte-preserved gzip archives; only incidental local paths in metadata/logs are redacted with both hashes in summary.json. SQLite stores stay local. The separate existing-store regression includes all fields and creation clocks and compares original text bytes. See the main document for the proposed gate and limitations."]
    (args.out/"summary.md").write_text("\n".join(lines)+"\n", encoding="utf-8")
    print(json.dumps({"development_statuses": dict(Counter(r["status"] for r in summary["runs"])),
                      "cost_aggregates": aggregates}, indent=2))


if __name__ == "__main__":
    main()
