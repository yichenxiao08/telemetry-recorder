# Lane-02 Checkpoint A — source freeze, no evaluation

Stage 1 only. Await explicit owner **GO** before Stage 2. No reserved seed was
generated or scored. No M6–M8, P0–P4 production work, conditional comparison,
migration, new structural/context diff type or median-constant change.

## Contract and source

Docs-only Step 0 commit: **6a3716b8b1c21b3167331d0c0e7421ffc529fdfb**.
It changes only the four owner-requested items and checksums. The new timestamp
origin is 1767225600000000000 (2026-01-01 UTC), with unchanged offsets.
The harness supplies run start times; the existing probe/exporter derives event
timestamps from that run anchor and monotonic elapsed time. The plain target has
no clock or instrumentation imports, so the clock stop condition did not arise.

Canonical executable/spec/profile/config/test hashes are in
benchmark/problems/lane-02/source-freeze.json (UTF-8/LF normalization). The source
freeze commit is reported separately to avoid a self-referential commit hash.
The target, generator, profile and probe config follow the frozen equations.
Go and Python independently check PowerShell-authored development-only golden
vectors for U(), all 40 ordering positions and frame numbers in two sessions.

The generator rejects every reserved seed before hashing or constructing rows
unless the explicit evaluation-runner flag is supplied. Imports have no generation
side effects. Tests inspect test call sites and CI to prohibit that flag. The
runner additionally requires the complete reserved-seed order and unchanged,
committed source-freeze manifest; an exclusive local evaluation ledger prevents
an accidental second command using another output directory. That evaluation
path has **not** been enabled in Stage 1. There is no force/retry option.

No producer/helper safety behavior changed. Recorder queue, counters, cooldown,
network worker, exception handling and bounded flush remain the prior shared
implementation. The target lookup and current-input counterfactuals, noise
metamorphic check, and source/config counting run only on development seeds.
Target edits by instrumentation: **0 lines**; instrumentation imports/calls:
**0 lines**; config: **10 physical/nonblank lines**.

## Implemented surface

- Optional profile symptom/candidate role, author and rationale metadata. Existing
  profile JSON and cohort results omit new empty fields, preserving legacy bytes.
- The CLI has **The problem**, candidate median, and separate reference-tail
  sections. JSON retains the full legacy median report plus an opt-in presentation.
  No new viewer page or general production phase is introduced.
- Tail bounds, padding, strict comparisons, +1/+2 smoothing and eligibility are
  exactly the contract. Unknown/insufficient scores serialize as **null**, not a
  measured zero. Observed feature values determine known/shared-session counts.
- Tail citations prefer actual outside witnesses, then observed samples, bounded
  to three per arm. Legacy median citations are unchanged. The final development
  reanalysis changes only these new tail sample citations; all legacy fields,
  tail scores, counts and interval endpoints match the original development run.

## Development results (101 and 103 only)

All **12/12 cases completed** with 400 problem/400 reference examples and 20 shared
session IDs each. These IDs are constructed clusters, not proof of independent
trajectories. Both seeds have the same mechanism ranks below. Dash means
unranked, not missing-as-zero. The generated report lists every feature, positive
count, score, raw tail numerator/denominator, legacy symptom-inclusive rank,
top-1/3/5, exact ties and nuisance-above-mechanism entry.

| Variant | Candidate median | Candidate tail | Positive counts median / tail |
|---|---|---|---|
| matched-negative-controls | age #1 | age #1 | 1 / 1 |
| minority-stale-use | current #1, age — | current #1, age #2 | 1 / 2 |
| benign-minority-tail | current #1, age — | current #1, age #2 | 1 / 2 |
| within-range-subgroup | current #1, age — | current #1, age — | 1 / 1 |
| moderate-decoy | current #1, noise_0 #2, age — | current #1, age #2, noise_0 #3 | 2 / 3 |
| unrelated-nuisance | current #1, age —; seed 103 also noise_5 #2 | current #1, age #2 | 1 or 2 / 2 |

Age tail is 400/402 in matched, 80/402 in minority/benign/moderate/unrelated,
and zero in within-range. Current tail is zero in matched, 320/402 otherwise.
Invariant P/R counts are 400 violated / 400 satisfied in matched; otherwise P
80 violated+320 N/A and R 80 satisfied+320 N/A. Every deterministic age/current
score/count and invariant expectation matches. The within-range age miss remains.
These are **reproducibility checks**, not diagnostic-performance evidence.

Moderate-decoy noise_0 scores: median 0.196438111 / 0.192036136 and tail
0.146766169 / 0.174129353 for 101/103. Its raw tail counts are 59/400 vs 0/400
and 70/400 vs 0/400. It outranks unranked median age, but not tail age. Seed 103's
unrelated noise_5 has median score 0.019639565 and also outranks unranked age;
all unrelated nuisance tail scores are zero. No constant, seed, variant or
profile was changed after seeing these results.

### Retained contract-premise mismatch

The amendment says benign nuisance support is wholly inside the padded reference
interval. Seed 101 contradicts that **premise**: noise_0 has **7/400 outside on
both sides**, with upper endpoint about 3.798230205. Tail T is nevertheless
**exactly zero**, because matched multisets have equal smoothed rates. Seed 103
has zero outside. Thus the amended T=0 expectation passes, but the stated
support justification is not universally true for the finite sample. Keep this
visible; do not alter the generator/interval or silently amend the contract.
This does not change any predeclared age/current or invariant count.

### Retained test/measurement limitations

An initial unit test expected zero tail score for 0/30 vs 0/100 outside. The
specified Laplace rates actually produce 1/32-1/102 = 0.02144607843137255.
Its failure is retained in lane02-stage1-tail-firstfailure.txt. The corrected
tests assert that exact positive smoothing effect **and** zero for equal-sized
edge-only samples, preserving strict edge coverage. The formula was not changed.
Unequal-size tail comparisons may therefore show a positive smoothed contrast
without any raw outside observations; do not present this as outlier evidence.

Development used one trial per case with the full new payload, not the original
five-trial fixed-payload overhead harness. Per-case wrapper p50 ranges
234.150–346.700 us; p99 469.143–846.346 us. Against the historical 330/830 us
reference budget, 10/12 cases meet both: seed 103 moderate misses p99 (846.346 us),
and seed 103 unrelated misses p50 (346.700 us). All 12 miss the original
100/250 us reference budget. These are honest cost observations, not a new
budget pass, a five-trial replication, or a performance-tuning exercise.
Context-enter p50 is 26.5–29.5 us, exit p50 8.05–12.15 us. Explicit flush takes
5.678–102.880 ms per case; it is outside the host call path. Context p99 was
not stored in the first development run; the frozen driver records it in future
runs. No number is reconstructed from unavailable samples.

Capture totals: **9,600 decisions**, **57,600 captured arguments**, **19,200
accepted/exported OTLP spans** (one root plus one probed span per decision),
**zero drops/export failures**. Decision envelope bytes total 15,544,497
(mean about 1,619 bytes/decision). Native anchor navigation was checked during
the original run and again for the refined tail witnesses using a read-only
HTTP viewer with a writer that rejects writes. No data replay or new labels
are needed for that final verification; all **155** final link targets resolve.

## Validation and reproduction

Artifacts: benchmark/results/lane02-stage1-development.json/.md,
lane02-stage1-legacy.json and lane02-stage1-views.json. The original development
store/report remain under .cache/lane02-dev/attempt-01; development source was
refined before freeze, and final reports were reanalyzed from the same pinned
records without regeneration. The cached pre-change binary was built from
70ce5acc8db6679535dc660278215dc4fdceb22b.

All 15 frozen lane-01 full reports and native CLI **JSON and text bytes** match
that pre-change binary exactly. Before/after SHA-256s are recorded separately.
The median source, old profiles, source target and result files are unchanged.
All lane-02 source tests use development/toy data only. Table/rapid tests cover
bounds, determinism, no naming boosts, strict interval-edge equality, minima,
known/unknown clusters, missing capture, nullable scores and witness selection.
The native Python suite checks real probe OTLP decoding, harness times, lineage,
CLI role sections, insufficient-tail output and native evidence anchors.

Native PowerShell checks (no reserved seeds):

```powershell
task lint
go test -race ./...
task build
uv run --directory adapters/provenance-python --frozen python -m unittest discover -s tests -v
```

Windows and Ubuntu CI must pass on the source-freeze commit; the final checkpoint
report links that run. There is no Stage 2 command to execute at this checkpoint.

## Risks and boundary

The largest assumption is still validity of constructed, author-known synthetic
episodes as a proxy for real diagnosis. Deterministic age/current counts make
success largely arithmetic; they cannot establish real-world accuracy or
causality inference. The unchanged median misses minorities; marginal tail
coverage misses within-range subgroups and can flag unused shifted nuisances.
The finite-reference support premise and unequal-size smoothing issue need
interpretation, not post-result tuning. Proceed only after the owner reviews
these findings and explicitly replies **GO**.
