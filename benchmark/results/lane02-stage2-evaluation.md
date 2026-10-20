# Lane-02 Stage 2 reserved-seed results

Synthetic only. One invocation after owner GO; stopped on transport/capture incompleteness. No retry, generation replay, analysis replay, seed replacement or tuning. One attempted/incomplete case, zero completed cases, zero scored cases, 29 unattempted cases. Attempted is a total, not a disjoint status bucket.

Contract commit: 6a3716b8b1c21b3167331d0c0e7421ffc529fdfb. Source-freeze commit: d704cc7b40b341df7ed02e8f564202d5f653f7f6. All 34 canonical source/spec/profile/config/test hashes still match. The frozen manifest retains its historical pre-GO status; it has not been rewritten.

Same reporting columns as development; NA means unavailable, never zero association. Median, tail and symptom-inclusive legacy lists were not computed. No cohorts were created or annotations posted: the frozen runner stopped before those steps. Harness labels are 400 problem / 400 reference, not completed store selections. Full 800-row harness manifest and exact raw counters remain in the accompanying JSON.

| Seed / variant | Status | Median positives / mechanisms | Tail positives / mechanisms | Tail top 1 | Score/count mismatches |
|---|---|---|---|---|---|
| 710021 / matched-negative-controls | incomplete | NA | NA | NA | Not tested |
| 710021 / minority-stale-use | unattempted | NA | NA | NA | Not tested |
| 710021 / benign-minority-tail | unattempted | NA | NA | NA | Not tested |
| 710021 / within-range-subgroup | unattempted | NA | NA | NA | Not tested |
| 710021 / moderate-decoy | unattempted | NA | NA | NA | Not tested |
| 710021 / unrelated-nuisance | unattempted | NA | NA | NA | Not tested |
| 710023 / matched-negative-controls | unattempted | NA | NA | NA | Not tested |
| 710023 / minority-stale-use | unattempted | NA | NA | NA | Not tested |
| 710023 / benign-minority-tail | unattempted | NA | NA | NA | Not tested |
| 710023 / within-range-subgroup | unattempted | NA | NA | NA | Not tested |
| 710023 / moderate-decoy | unattempted | NA | NA | NA | Not tested |
| 710023 / unrelated-nuisance | unattempted | NA | NA | NA | Not tested |
| 710027 / matched-negative-controls | unattempted | NA | NA | NA | Not tested |
| 710027 / minority-stale-use | unattempted | NA | NA | NA | Not tested |
| 710027 / benign-minority-tail | unattempted | NA | NA | NA | Not tested |
| 710027 / within-range-subgroup | unattempted | NA | NA | NA | Not tested |
| 710027 / moderate-decoy | unattempted | NA | NA | NA | Not tested |
| 710027 / unrelated-nuisance | unattempted | NA | NA | NA | Not tested |
| 710029 / matched-negative-controls | unattempted | NA | NA | NA | Not tested |
| 710029 / minority-stale-use | unattempted | NA | NA | NA | Not tested |
| 710029 / benign-minority-tail | unattempted | NA | NA | NA | Not tested |
| 710029 / within-range-subgroup | unattempted | NA | NA | NA | Not tested |
| 710029 / moderate-decoy | unattempted | NA | NA | NA | Not tested |
| 710029 / unrelated-nuisance | unattempted | NA | NA | NA | Not tested |
| 710033 / matched-negative-controls | unattempted | NA | NA | NA | Not tested |
| 710033 / minority-stale-use | unattempted | NA | NA | NA | Not tested |
| 710033 / benign-minority-tail | unattempted | NA | NA | NA | Not tested |
| 710033 / within-range-subgroup | unattempted | NA | NA | NA | Not tested |
| 710033 / moderate-decoy | unattempted | NA | NA | NA | Not tested |
| 710033 / unrelated-nuisance | unattempted | NA | NA | NA | Not tested |

## 710021 / matched-negative-controls

Capture: 800 host calls finished; 4,800 argument captures; 735 serialized decision envelopes; 1,189,437 envelope bytes (about 1,618.28 per serialized envelope). Attempted spans 1,600; accepted 1,470; exported successfully 1,418; export_failed 52; drop_cooldown 130; network requests 419. Root and probe spans are separate queue items. All counters are spans unless explicitly described as decisions/envelopes/arguments. No other drop counters were recorded.

130 + 52 + 1,418 = 1,600. The 52 failed-export spans have unknown persistence: a timeout/lost acknowledgement could occur after server commit. Do not call them confirmed database losses or infer which decisions are complete. Flush drained pending work but cannot restore failed exports. No read/recovery/analysis of the partial store was performed.

| Measurement | p50 (us) | p99 (us) | Budget/result |
|---|---:|---:|---|
| Probed-call wrapper | 475.750 | 2300.580 | Misses original 100/250 and historical 330/830 |
| Run-context enter | 44.000 | 318.680 | Separate; no added budget |
| Run-context exit | 17.800 | 86.061 | Separate; no added budget |

Explicit flush: 9.7015 ms. This is one 800-call case scheduled at 60 calls/sec, nominally 13.33 seconds; achieved rate and phase wall duration were not stored. It is not the older five-trial 30-second/tight-loop overhead experiment. No new claim about safety under all failure modes follows from these cost samples.

### legacy_median

Positive count, top-1 feature, mechanism ranks, recall@1/3/5, ranked scores, exact ties, nuisance-above-mechanism entries, selected/observed/missing/excluded/session counts and evidence IDs: **NA, not computed**. Missing telemetry is not a zero association.

### median

Positive count, top-1 feature, mechanism ranks, recall@1/3/5, ranked scores, exact ties, nuisance-above-mechanism entries, selected/observed/missing/excluded/session counts and evidence IDs: **NA, not computed**. Missing telemetry is not a zero association.

### tail

Positive count, top-1 feature, mechanism ranks, recall@1/3/5, ranked scores, exact ties, nuisance-above-mechanism entries, selected/observed/missing/excluded/session counts and evidence IDs: **NA, not computed**. Missing telemetry is not a zero association.

## Analytic expectations and retained misses

- No deterministic age/current count, interval, invariant, median or tail expectation was evaluated. No deterministic mismatch is claimed, and no agreement is claimed. Top-5 reproducibility across five seeds is not assessable for any variant.
- Minority, benign, within-range, moderate and unrelated variants are entirely unattempted. Every remaining seed/variant is enumerated above. Seed 710021 matched consumed its one attempt; the evaluation stopped without continuing after the incomplete case.
- Transport completeness failed. Both wrapper cost budgets were missed. Native evidence links were not reached or verified for reserved data. Doctor and final instrumentation reporting in the runner were not reached; separate read-only counting confirms zero target edits/instrumentation lines and ten config lines.
- Development findings remain on record, not substituted for evaluation: median minority misses, deliberate within-range age miss, nuisance-above-mechanism cases, finite-reference benign-support counterexample and unequal-size smoothing limitation.

## Failure evidence and boundary

The server log has only the successful listener-start message (queue size 64). The recorder uses a 0.5-second HTTP timeout, one-second cooldown and capacity 256; its worker catches export exceptions and records counts without retaining the exception or HTTP status. Some export_failed items can be suppressed while already in cooldown; 52 is not a count of distinct network failures. This run establishes failed acknowledgement/export accounting and cooldown drops, not whether the initial cause was a timeout, HTTP rejection, encoding error or other exporter exception. No timeout, batch, queue, engine, source, scoring or logging change was made to investigate by rerunning.

The raw report, partial database plus WAL/SHM, listener log, target snapshot and exclusive evaluation ledger remain under .cache. Published artifact metadata includes their paths/sizes/SHA-256s and a portable copy of the ledger; the ledger is not removed. The report JSON is a byte-for-byte copy of the frozen runner output. Publishing only reads saved files and does not import the generator, run the engine or open SQLite.

Riskiest assumption now: the lossy, fail-open probe/export path can provide complete, representative cohorts even at the scheduled synthetic rate. Completeness cannot be inferred from host success or flush success. No mechanism-ranking, causal-separation or real-world diagnostic-accuracy conclusion is supported by this incomplete evaluation. Stop here; no production phase or replacement evaluation is authorized.
