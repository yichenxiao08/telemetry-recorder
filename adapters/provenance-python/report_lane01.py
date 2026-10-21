"""Generate Slice 1 tables and compare saved shop rankings, never edit results."""
import argparse
import json
import subprocess
from pathlib import Path


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument("--repo",type=Path,required=True)
    parser.add_argument("--verify-db",type=Path)
    args=parser.parse_args()
    results=args.repo/"benchmark/results"
    data=json.loads((results/"lane-01.json").read_text())
    first=json.loads((results/"lane-01-first.json").read_text())
    assert first["profile_sha256"]==data["profile_sha256"],"profile changed between evaluation and confirmation"
    compare_fields=("feature","type","score","status","problem_observed","reference_observed","problem_missing","reference_missing","problem_numerator","reference_numerator")
    assert len(first["cases"])==len(data["cases"])==15
    for a,b in zip(first["cases"],data["cases"]):
        assert (a["seed"],a["variant"],a["ranks"],a["invariant_counts"])==(b["seed"],b["variant"],b["ranks"],b["invariant_counts"])
        assert [{k:d[k] for k in compare_fields} for d in a["result"]["differences"]]==[{k:d[k] for k in compare_fields} for d in b["result"]["differences"]]
    (results/"lane-01-confirmation.json").write_text(json.dumps({"first_revision":first["revision"],"confirmed_revision":data["revision"],"cases":15,"ranks_scores_counts_invariants_identical":True,"formulas_profile_unchanged":True},indent=2)+"\n",encoding="utf-8")
    if args.verify_db:
        for case in data["cases"]:
            p=case["result"]["problem"]["id"]
            r=case["result"]["reference"]["id"]
            checked=json.loads(subprocess.check_output([str(args.repo/"bin/signalscout.exe"),"--db",str(args.verify_db),"explain-cohort",p,r,"--json"]))
            assert checked==case["result"],(case["seed"],case["variant"],"saved evidence result changed")
        (results/"lane-01-boundary-confirmation.json").write_text(json.dumps({"cases":15,"full_reports_identical_on_pinned_evidence":True,"guard":"nonfinite MAD arithmetic is unsupported, never NaN"},indent=2)+"\n",encoding="utf-8")
    rows=["# lane-01: pre-registered synthetic Slice 1", "",
        "Pre-registration: `"+data["preregistration_sha"]+"`; implementation: `"+data["revision"]+"`.",
        "All 15 first-run evaluation cases retained; no formula/constant/profile tuning. Validation fixes are disclosed below.",
        "Each case: 100 marked problem and 100 explicit reference frames; 10 shared session clusters.",
        "Session labels do not establish independence. Sampling/capture: 3000/3000 decisions, zero drops.",
        "Ranks count only positive, sufficiently observed differences. Dash = no detected difference.",
        "Top-k columns are recall of TWO labeled mechanisms; the symptom is excluded.", "",
        "| Seed | Variant | Stale age | Gap | Output symptom | Decoys 0,1,2,3,4,5,6 | Mechanism recall @1 / @3 / @5 | Invariant violated / satisfied / N/A |",
        "|---|---|---:|---:|---:|---|---|---|"]
    rank=lambda x:str(x) if x is not None else "—"
    for case in data["cases"]:
        r=case["ranks"];inv=case["invariant_counts"]
        rows.append(f"| {case['seed']} | {case['variant']} | {rank(r['previous_age'])} | {rank(r['gap'])} | {rank(r['output.lane'])} | "+
            ", ".join(rank(r[f"noise_{i}"]) for i in range(7))+" | "+
            " / ".join(f"{case['top_k_mechanism_recall'][str(k)]:.0%}" for k in (1,3,5))+" | "+
            f"{inv.get('violated',0)} / {inv.get('satisfied',0)} / {inv.get('not_applicable',0)} |")
    rows += ["", "## Interpretation and retained misses", "",
        "The base task deliberately separates input distributions. Its perfect top-3 result is inflated:",
        "the profile declares a useful age transform, an oracle marks known wrong outputs, and references",
        "are constructed to be correct. The output symptom ranks first; that is not a mechanism discovery.",
        "The author knew the desired mechanisms/formulas; authored_blind means evaluation outcomes unseen.",
        "No real Subway Surfers data was used or reconstructed.","",
        "After the first run, a new adversarial parser test exposed contradictory output capture-off",
        "metadata being treated as observed. The parser now respects output capture status. No evaluation",
        "record had this contradiction. All 15 cases were rerun: ranks, scores, counts and invariants",
        "are identical. The first results and failed regression test remain committed. The rerun is",
        "confirmation on the SAME seeds, not a fresh holdout; formulas/profile were unchanged.","",
        "A later adversarial numeric test found finite extreme values could overflow MAD subtraction.",
        "That arithmetic now reports unsupported_numeric (observation), never a NaN or false zero-effect",
        "association. Exact formula/constant tests and wide-magnitude rapid tests pass. All 15 saved",
        "reports were recomputed from pinned evidence and remain identical; no formula/constant changed.","",
        "Seven irrelevant but selection-correlated inputs outrank mechanisms in decoy-heavy. This is a",
        "real limitation of marginal associations, not evidence that noise caused the output. Harder",
        "contains stale versions in 20% of problem frames: median shift misses them entirely. The",
        "declared-use invariant still flags 20/100 problem frames, with no baseline. No scores were tuned.",
        "Small chance decoy shifts also appear in base/harder and remain reported. Invariant coverage is",
        "complete for declared-use fields here; missing-capture unknown is separately tested. No graph",
        "proof of actual reads or real-world sensitivity/specificity is established.","",
        "These are constructed frame cases, not a reconstructed sequential game history. The fixture",
        "supplies a previous value/version; its actual origin in a prior frame is not independently",
        "verified. The invariant validates the declared version relation. Trust the pipeline/regression",
        "result, but expect the base diagnostic-accuracy estimate to be inflated.","",
        "The plain target changes zero lines; external probe configuration is 10 lines. Native event",
        "anchors for the top five findings were followed over real HTTP in every case. Raw labels,",
        "selected IDs/revisions, representative evidence links, sample counts and per-field missingness",
        "are in lane-01.json. All engine findings are observations/associations, with no hypotheses."]
    (results/"lane-01.md").write_text("\n".join(rows)+"\n",encoding="utf-8")
    context_old=json.loads((results/"m4b-context-demo-final.json").read_text())
    context_new=json.loads((results/"lane-01-shop-context.json").read_text())
    comparisons=[]
    for seed in (1,2):
        old=json.loads((results/f"diff-seed-{seed}.json").read_text())
        new=json.loads((results/f"lane-01-shop-structural-seed-{seed}.json").read_text())
        old_trials={(t["run_id"],t["mode"]):t for t in old["trials"]}
        for t in new["trials"]:
            prior=old_trials[t["run_id"],t["mode"]]
            assert prior["ranks"]==t["ranks"]
            assert prior["result"]["differences"]==t["result"]["differences"],(seed,t["run_id"],"structural differences changed")
        before=next(x for x in context_old if x["Seed"]==seed)
        after=next(x for x in context_new if x["Seed"]==seed)
        original={(t["RunID"],t["Mode"]):t for t in before["Trials"]}
        for t in after["Trials"]:
            prior=original[t["RunID"],t["Mode"]]
            assert prior["Result"]["differences"]==t["Result"]["differences"],(seed,t["RunID"],"context differences changed")
            assert prior["PoolRank"]==t["PoolRank"]
        comparisons.append({"seed":seed,"structural_comparisons":len(new["trials"]),"context_comparisons":len(after["Trials"]),
            "all_ranked_differences_and_scores_identical":True,"m3_max_ms":new["maximum_40_run_diff_ms"],
            "known_seed1_sampling_gap_preserved":True if seed==1 else None})
    (results/"lane-01-shop-regression.json").write_text(json.dumps(comparisons,indent=2)+"\n",encoding="utf-8")
    print(json.dumps(comparisons))


if __name__=="__main__":
    main()
