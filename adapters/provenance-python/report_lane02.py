"""Publish existing lane-02 records; never generates data or opens SQLite."""
import argparse
import hashlib
import json
import subprocess
import copy
from collections import Counter
from pathlib import Path

from lane02 import rankings, raw_numeric


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=Path, required=True)
    parser.add_argument("--json", type=Path, required=True)
    parser.add_argument("--markdown", type=Path, required=True)
    args = parser.parse_args()
    report = json.loads(args.input.read_text())
    root = Path(__file__).resolve().parents[2]
    binary = root/"bin/signalscout.exe"
    database = args.input.parent/"fresh.db"
    profile = json.loads((root/"benchmark/problems/lane-02/profile.json").read_text())
    report["database"] = database.relative_to(root).as_posix()
    report["case_counts"] = dict(Counter(case["status"] for case in report["cases"]))
    report["case_counts"]["unattempted"] = 0 if report["status"] == "completed" else None
    report["confirmed_binary_sha256"] = hashlib.sha256(binary.read_bytes()).hexdigest()
    lines = ["# Lane-02 Stage 1 development results", "", "Synthetic only; seeds 101/103. Reserved seeds ungenerated/unscored.",
             "Both rankings are separate. Deterministic tail ranks are reproducibility checks, not diagnostic-performance evidence.",
             "", "| Seed / variant | Median positives / mechanisms | Tail positives / mechanisms | Tail top 1 | Score/count mismatches |",
             "|---|---|---|---|---|"]
    for case in report["cases"]:
        if case["status"] != "completed":
            continue
        previous = case["result"]
        command = [str(binary), "--db", str(database), "explain-cohort", previous["problem"]["id"], previous["reference"]["id"]]
        current = json.loads(subprocess.check_output(command+["--json"]))
        old_comparison, new_comparison = copy.deepcopy(previous), copy.deepcopy(current)
        # Development source refinement prefers actual outside tail witnesses;
        # all legacy bytes/statistics and every tail term must still agree.
        for comparison in (old_comparison, new_comparison):
            for finding in comparison["presentation"]["candidate_tail"]:
                finding.pop("evidence_ids")
                finding.pop("evidence_urls")
        if old_comparison != new_comparison:
            raise RuntimeError("final binary changed a development report; retain original")
        case["result"] = current
        case["tail_witness_samples_refined"] = True
        rendered = subprocess.check_output(command).decode()
        for heading in ("The problem", "Candidate median", "Separate reference-tail"):
            if heading not in rendered:
                raise RuntimeError("missing separate CLI display")
        case["confirmed_final_binary"] = True
        case["rankings"] = rankings(current, case["variant"])
        case["raw_numeric"] = raw_numeric(case["manifest"], profile)
        case["contract_premise_misses"] = [{"feature": f["feature"], "problem_outside": f["terms"]["problem_outside"],
                                             "reference_outside": f["terms"]["reference_outside"]}
                                            for f in current["presentation"]["candidate_tail"]
                                            if case["variant"] in ("benign-minority-tail", "unrelated-nuisance")
                                            and f["feature"].startswith("noise_")
                                            and (f["terms"]["problem_outside"] or f["terms"]["reference_outside"])]
        # Full pinned native IDs/results remain. Raw input copies stay in the saved
        # development store and original report rather than duplicating all values.
        case["manifest_sha256"] = hashlib.sha256(json.dumps(case.pop("manifest"), sort_keys=True).encode()).hexdigest()
        m, t = case["rankings"]["median"], case["rankings"]["tail"]
        lines.append(f'| {case["seed"]} / {case["variant"]} | {m["positive_candidates"]} / {m["mechanism_ranks"]} | '
                     f'{t["positive_candidates"]} / {t["mechanism_ranks"]} | {t["top_1_feature"]} | {case["expectation_mismatches"]} |')
    for case in report["cases"]:
        if case["status"] != "completed":
            continue
        lines += ["", f'## {case["seed"]} / {case["variant"]}', "",
                  f'Capture: `{case["capture"]}`. Wrapper/context timings ns: `{case["overhead_ns"]}`.',
                  f'Invariant counts by arm: `{case["invariants_by_arm"]}`.',
                  f'Support-premise misses (scores unchanged): `{case["contract_premise_misses"]}`.', ""]
        for name in ("legacy_median", "median", "tail"):
            r = case["rankings"][name]
            lines += [f'### {name}', "", f'Positive count {r["positive_candidates"]}; top 1 {r["top_1_feature"]}; recall@1/3/5 `{r["recall"]}`.',
                      f'Ranks/scores: `{r["ranked"]}`.', f'Ties including unranked zero observations: `{r["exact_score_ties"]}`.',
                      f'Nuisance above mechanism: `{r["nuisance_above_mechanism"]}`.', ""]
        lines += ["| Tail feature | Raw P outside / observed | Raw R outside / observed | Interval | Score |", "|---|---|---|---|---|"]
        for f in case["result"]["presentation"]["candidate_tail"]:
            t = f["terms"]
            lines.append(f'| {f["feature"]} | {t["problem_outside"]}/{f["problem_observed"]} | '
                         f'{t["reference_outside"]}/{f["reference_observed"]} | [{t["lower"]:.9g}, {t["upper"]:.9g}] | {f["score"]:.9g} |')
    args.json.write_text(json.dumps(report, indent=2)+"\n", encoding="utf-8")
    args.markdown.write_text("\n".join(lines)+"\n", encoding="utf-8")


if __name__ == "__main__":
    main()
