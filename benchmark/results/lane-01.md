# lane-01: pre-registered synthetic Slice 1

Pre-registration: `b537ee154f004cc93a24571156fe1feb856b226f`; implementation: `3002f4c2287f2adc849abd51c008930b0aa77428`.
All 15 first-run evaluation cases retained; no formula/constant/profile tuning. Validation fixes are disclosed below.
Each case: 100 marked problem and 100 explicit reference frames; 10 shared session clusters.
Session labels do not establish independence. Sampling/capture: 3000/3000 decisions, zero drops.
Ranks count only positive, sufficiently observed differences. Dash = no detected difference.
Top-k columns are recall of TWO labeled mechanisms; the symptom is excluded.

| Seed | Variant | Stale age | Gap | Output symptom | Decoys 0,1,2,3,4,5,6 | Mechanism recall @1 / @3 / @5 | Invariant violated / satisfied / N/A |
|---|---|---:|---:|---:|---|---|---|
| 1009 | base | 3 | 2 | 1 | —, —, —, —, 6, 4, 5 | 0% / 100% / 100% | 100 / 0 / 100 |
| 1009 | decoy-heavy | 10 | 9 | 1 | 4, 5, 2, 3, 6, 8, 7 | 0% / 0% / 0% | 100 / 0 / 100 |
| 1009 | harder | — | 2 | 1 | 4, —, 3, —, 5, —, — | 0% / 50% / 50% | 20 / 80 / 100 |
| 2027 | base | 3 | 2 | 1 | —, —, 4, —, —, 6, 5 | 0% / 100% / 100% | 100 / 0 / 100 |
| 2027 | decoy-heavy | 10 | 9 | 1 | 7, 6, 2, 8, 5, 4, 3 | 0% / 0% / 0% | 100 / 0 / 100 |
| 2027 | harder | — | 2 | 1 | —, —, —, —, —, 4, 3 | 0% / 50% / 50% | 20 / 80 / 100 |
| 3037 | base | 3 | 2 | 1 | —, 5, —, 4, —, —, — | 0% / 100% / 100% | 100 / 0 / 100 |
| 3037 | decoy-heavy | 10 | 9 | 1 | 7, 2, 6, 8, 5, 4, 3 | 0% / 0% / 0% | 100 / 0 / 100 |
| 3037 | harder | — | 2 | 1 | 3, —, —, —, —, —, — | 0% / 50% / 50% | 20 / 80 / 100 |
| 4051 | base | 3 | 2 | 1 | —, —, —, —, 4, 5, — | 0% / 100% / 100% | 100 / 0 / 100 |
| 4051 | decoy-heavy | 10 | 9 | 1 | 6, 5, 4, 3, 8, 7, 2 | 0% / 0% / 0% | 100 / 0 / 100 |
| 4051 | harder | — | 2 | 1 | —, 3, —, 6, —, 4, 5 | 0% / 50% / 50% | 20 / 80 / 100 |
| 5059 | base | 3 | 2 | 1 | —, —, —, —, —, —, — | 0% / 100% / 100% | 100 / 0 / 100 |
| 5059 | decoy-heavy | 10 | 9 | 1 | 7, 8, 2, 3, 5, 4, 6 | 0% / 0% / 0% | 100 / 0 / 100 |
| 5059 | harder | — | 2 | 1 | —, —, 6, —, 5, 3, 4 | 0% / 50% / 50% | 20 / 80 / 100 |

## Interpretation and retained misses

The base task deliberately separates input distributions. Its perfect top-3 result is inflated:
the profile declares a useful age transform, an oracle marks known wrong outputs, and references
are constructed to be correct. The output symptom ranks first; that is not a mechanism discovery.
The author knew the desired mechanisms/formulas; authored_blind means evaluation outcomes unseen.
No real Subway Surfers data was used or reconstructed.

After the first run, a new adversarial parser test exposed contradictory output capture-off
metadata being treated as observed. The parser now respects output capture status. No evaluation
record had this contradiction. All 15 cases were rerun: ranks, scores, counts and invariants
are identical. The first results and failed regression test remain committed. The rerun is
confirmation on the SAME seeds, not a fresh holdout; formulas/profile were unchanged.

A later adversarial numeric test found finite extreme values could overflow MAD subtraction.
That arithmetic now reports unsupported_numeric (observation), never a NaN or false zero-effect
association. Exact formula/constant tests and wide-magnitude rapid tests pass. All 15 saved
reports were recomputed from pinned evidence and remain identical; no formula/constant changed.

Seven irrelevant but selection-correlated inputs outrank mechanisms in decoy-heavy. This is a
real limitation of marginal associations, not evidence that noise caused the output. Harder
contains stale versions in 20% of problem frames: median shift misses them entirely. The
declared-use invariant still flags 20/100 problem frames, with no baseline. No scores were tuned.
Small chance decoy shifts also appear in base/harder and remain reported. Invariant coverage is
complete for declared-use fields here; missing-capture unknown is separately tested. No graph
proof of actual reads or real-world sensitivity/specificity is established.

These are constructed frame cases, not a reconstructed sequential game history. The fixture
supplies a previous value/version; its actual origin in a prior frame is not independently
verified. The invariant validates the declared version relation. Trust the pipeline/regression
result, but expect the base diagnostic-accuracy estimate to be inflated.

The plain target changes zero lines; external probe configuration is 10 lines. Native event
anchors for the top five findings were followed over real HTTP in every case. Raw labels,
selected IDs/revisions, representative evidence links, sample counts and per-field missingness
are in lane-01.json. All engine findings are observations/associations, with no hypotheses.
