"""Publish saved diagnostics only. No generator, database, engine or network I/O."""
import argparse
from collections import Counter
import gzip
import hashlib
import json
from pathlib import Path


def quantile(values, q):
    if not values:
        return None
    values = sorted(values)
    index = (len(values)-1)*q
    low = int(index)
    return values[low]+(values[min(low+1, len(values)-1)]-values[low])*(index-low)


def cost(calls, batches):
    # Diagnostic convention, not a scoring threshold: >=100 ms worker waits.
    waits = [(b["attempt_unix_ns"], b["attempt_unix_ns"]+b["elapsed_ns"])
             for b in batches if b["elapsed_ns"] >= 100_000_000]
    inside, outside = [], []
    for call in calls:
        selected = inside if any(a <= call["unix_ns"] <= b for a, b in waits) else outside
        selected.append(call["wrapper_ns"]/1000)
    return {"wait_definition_ms": 100, "slow_batches": len(waits),
            "calls_inside_wait": len(inside), "calls_outside_wait": len(outside),
            "wrapper_p99_inside_us": quantile(inside, .99), "wrapper_p99_outside_us": quantile(outside, .99),
            "batch_p50_ms": quantile([b["elapsed_ns"]/1e6 for b in batches], .5),
            "batch_p99_ms": quantile([b["elapsed_ns"]/1e6 for b in batches], .99),
            "batch_max_ms": max((b["elapsed_ns"]/1e6 for b in batches), default=None)}


def accounting(snapshot):
    counts = {k: snapshot.get(k, 0) for k in ("attempted", "accepted", "exported", "export_failed",
                                               "drop_cooldown", "drop_full", "drop_closed", "drop_invalid")}
    counts["dropped"] = sum(counts[k] for k in ("drop_cooldown", "drop_full", "drop_closed"))
    counts["unsettled"] = counts["attempted"]-counts["exported"]-counts["export_failed"]-counts["dropped"]
    counts["identity"] = "attempted = exported + export_failed(unacknowledged) + dropped + unsettled"
    return counts


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=Path, required=True)
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    args.out.mkdir(parents=True, exist_ok=False)
    summary = {"synthetic_only": True, "checkpoint": "A; no fix", "development_runs": [], "controls": [],
               "accounting_note": "export_failed is the unchanged legacy unacknowledged count, not proof of database loss; unknown persistence is not delivered",
               "environment": json.loads((args.input/"environment.json").read_text(encoding="utf-8-sig")),
               "measured_binary": json.loads((args.input/"measured-binary.json").read_text(encoding="utf-8-sig")),
               "protected_files": json.loads((args.input/"protected-after.json").read_text(encoding="utf-8-sig")),
               "files": []}

    def archive(source):
        relative = source.relative_to(args.input)
        destination = args.out/"raw"/relative.parent/(relative.name+".gz")
        destination.parent.mkdir(parents=True, exist_ok=True)
        data = original = source.read_bytes()
        # Report/data bytes remain exact. Only incidental local paths in logs
        # are made portable, with both original and published hashes disclosed.
        if source.suffix == ".log" or source.name.endswith("-terminal.txt"):
            root = str(Path(__file__).resolve().parents[2]).encode()
            data = data.replace(root, b"<repo>").replace(root.replace(b"\\", b"/"), b"<repo>")
        with destination.open("wb") as target:
            with gzip.GzipFile(filename="", mode="wb", fileobj=target, mtime=0) as zipped:
                zipped.write(data)
        digest = hashlib.sha256(original).hexdigest()
        published_digest = hashlib.sha256(data).hexdigest()
        if hashlib.sha256(gzip.decompress(destination.read_bytes())).hexdigest() != published_digest:
            raise RuntimeError("raw archive mismatch")
        summary["files"].append({"source_relative": relative.as_posix(),
                                 "archive": destination.relative_to(args.out).as_posix(),
                                 "bytes": source.stat().st_size, "sha256_original": digest,
                                 "sha256_published": published_digest,
                                 "local_paths_redacted": data != original})

    cases_lines = ["# Capture-loss per-case development results", "",
                   "All are fresh development stores. No reserved seeds. Zero is measured only where a case completed.", "",
                   "| Run | Seed / variant | Status | Attempted | Exported | Failed export | Cooldown / full drops | Calls/sec | Queue HWM (run cumulative) | Wrapper p50 / p99 us | 100/250 pass |",
                   "|---|---|---|---:|---:|---:|---|---:|---:|---|---|"]
    for directory in sorted(args.input.glob("default-*")):
        if not directory.is_dir():
            continue
        report_path, diag_path = directory/"driver/report.json", directory/"diagnostics.json"
        report, diag = json.loads(report_path.read_text()), json.loads(diag_path.read_text())
        calls, batches = [], []
        with (directory/"timeline.jsonl").open() as stream:
            for line in stream:
                event = json.loads(line)
                if event["kind"] == "caller":
                    calls.append(event)
                elif event["kind"] == "batch":
                    batches.append(event)
        record = {"name": directory.name, "status": report["status"],
                  "cases": dict(Counter(c["status"] for c in report["cases"])),
                  "unattempted": len(report["planned_cases"])-len(report["cases"]),
                  "accounting": accounting(diag["final"]),
                  "queue_high_water": diag["final"]["queue_high_water"],
                  "diagnostics": diag["final"]["diagnostics"],
                  "observation_overflow": diag["observation_overflow"],
                  "timing_correlation": cost(calls, batches), "per_case": []}
        for index, case in enumerate(report["cases"]):
            phase = next((p for p in diag["cases"] if p["index"] == index), None)
            capture = case.get("capture", {})
            timing = case.get("overhead_ns", {})
            p50, p99 = timing.get("wrapper_p50"), timing.get("wrapper_p99")
            passed = p50 is not None and p99 is not None and p50 <= 100_000 and p99 <= 250_000
            item = {"seed": case["seed"], "variant": case["variant"], "status": case["status"],
                    "capture": capture, "timing_ns": timing, "budget_met": passed,
                    "call_phase_wall_ns": phase["wall_ns"] if phase else None,
                    "achieved_calls_per_second": phase["achieved_calls_per_second"] if phase else None,
                    "queue_high_water_run_cumulative": phase["after"]["queue_high_water"] if phase else None,
                    "expectation_mismatches": case.get("expectation_mismatches")}
            record["per_case"].append(item)
            rate = f'{item["achieved_calls_per_second"]:.3f}' if phase else "NA"
            times = f"{p50/1000:.3f} / {p99/1000:.3f}" if p50 is not None else "NA"
            cases_lines.append(f'| {directory.name} | {case["seed"]} / {case["variant"]} | {case["status"]} | '
                               f'{capture.get("attempted", "NA")} | {capture.get("exported", "NA")} | {capture.get("export_failed", 0)} | '
                               f'{capture.get("drop_cooldown", 0)} / {capture.get("drop_full", 0)} | {rate} | '
                               f'{item["queue_high_water_run_cumulative"]} | {times} | {passed} |')
        summary["development_runs"].append(record)
        for source in (report_path, diag_path, directory/"timeline.jsonl", directory/"driver/fresh.db.log"):
            archive(source)
    for source in sorted((args.input/"controls").glob("*/report.json")):
        report = json.loads(source.read_text())
        snapshot = report["snapshot"]
        waits = report["worker_batches"]
        successes = [(event["upstream_ack_unix_ns"]-event["started_unix_ns"])/1e6
                     for event in report["relay_events"] if "upstream_ack_unix_ns" in event]
        summary["controls"].append({"name": report["condition"],
            "accounting": accounting(snapshot), "queue_high_water": snapshot["queue_high_water"],
            "capacity": report["capacity"], "wall_seconds": report["wall_seconds"],
            "achieved_calls_per_second": report["achieved_calls_per_second"],
            "wrapper_p50_us": report["wrapper_p50_us"], "wrapper_p99_us": report["wrapper_p99_us"],
            "context_enter_p50_us": quantile([e["context_enter_ns"]/1000 for e in report["caller_events"]], .5),
            "context_enter_p99_us": quantile([e["context_enter_ns"]/1000 for e in report["caller_events"]], .99),
            "context_exit_p50_us": quantile([e["context_exit_ns"]/1000 for e in report["caller_events"]], .5),
            "context_exit_p99_us": quantile([e["context_exit_ns"]/1000 for e in report["caller_events"]], .99),
            "flush_ms": report["flush_ns"]/1e6, "drained": report["drained"],
            "budget_met": report["wrapper_p50_us"] <= 100 and report["wrapper_p99_us"] <= 250,
            "diagnostics": snapshot["diagnostics"],
            "timing_correlation": cost(report["caller_events"], waits),
            "upstream_ack_p99_ms": quantile(successes, .99), "upstream_ack_max_ms": max(successes, default=None),
            "relay_samples": len(report["relay_events"]),
            "first_relay_event": report["relay_events"][0] if report["relay_events"] else None,
            "known_post_commit_delay_seconds": report["ack_delay_seconds_first_request_only"]})
        archive(source)
        archive(source.parent/"fresh.db.log")
    for source in args.input.glob("*-terminal.txt"):
        archive(source)
    for source in (args.input/"environment.json", args.input/"processes.jsonl", args.input/"stage1-python.txt"):
        archive(source)
    for name in ("measured-binary.json", "protected-after.json", "final-python.txt", "go-lint.txt", "go-race.txt", "go-build.txt"):
        if (args.input/name).exists():
            archive(args.input/name)
    (args.out/"summary.json").write_text(json.dumps(summary, indent=2)+"\n")
    (args.out/"cases.md").write_text("\n".join(cases_lines)+"\n")
    lines = ["# Recorder capture-loss Checkpoint A measurements", "",
             "Synthetic; no fix implemented. export_failed means unacknowledged, not confirmed storage loss.", "",
             "| Run | Completed / incomplete / unattempted | Attempted | Exported | Failed export | Cooldown / full drops | Queue HWM |",
             "|---|---|---:|---:|---:|---|---:|"]
    for r in summary["development_runs"]:
        c, a = r["cases"], r["accounting"]
        lines.append(f'| {r["name"]} | {c.get("completed",0)} / {c.get("incomplete",0)} / {r["unattempted"]} | '
                     f'{a["attempted"]} | {a["exported"]} | {a["export_failed"]} | {a["drop_cooldown"]} / {a["drop_full"]} | {r["queue_high_water"]} |')
    lines += ["", "## Toy controls", "",
              "| Condition | Attempted | Exported | Failed export | Cooldown / full drops | HWM / capacity | Calls/sec | Wrapper p50 / p99 us | 100/250 pass |",
              "|---|---:|---:|---:|---|---|---:|---|---|"]
    for r in summary["controls"]:
        a = r["accounting"]
        lines.append(f'| {r["name"]} | {a["attempted"]} | {a["exported"]} | {a["export_failed"]} | '
                     f'{a["drop_cooldown"]} / {a["drop_full"]} | {r["queue_high_water"]} / {r["capacity"]} | '
                     f'{r["achieved_calls_per_second"]:.3f} | {r["wrapper_p50_us"]:.3f} / {r["wrapper_p99_us"]:.3f} | {r["budget_met"]} |')
    lines += ["", "All per-case costs, including every miss, are in cases.md. Raw gzip archives restore byte-identical original JSON reports and telemetry; SHA-256s are in summary.json. Incidental local paths in listener/terminal logs alone are redacted, with original/published hashes disclosed; originals stay local. Worker source rings are sampled to retain earlier stalls; controls cap retained batches at 4096 (more than their maximum attempted span count). No engine invocation or data generation occurs in publication."]
    (args.out/"summary.md").write_text("\n".join(lines)+"\n")


if __name__ == "__main__":
    main()
