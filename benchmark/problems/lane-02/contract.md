# Lane-02 pre-registration contract v1 — Task A review checkpoint

Status: **documentation committed before implementation or evaluation**.
Synthetic data only. Task B requires a separate owner approval of this contract.
No M5–M8, P0, conditional comparison, migration, production feature or new
scoring implementation starts in Task A. Existing lane-01 source, profiles,
constants and results stay frozen; this does not replace or repair those results.

## Provenance and source freeze

Normative generator/target source is [generator-source.md](generator-source.md),
a complete language-neutral source specification, not an executable generator.
Its exact Git-blob UTF-8/LF bytes are SHA-256 pinned in checksums.txt. This is
the conservative interpretation of the docs-only constraint: **no executable
generator exists yet**. Task B must translate it without changing equations and
record executable target/generator/profile SHA-256s in a source-freeze commit
before generating evaluation records. If the source deviates, stop for review.
The contract commit SHA identifies this document; it is reported after commit
rather than written into its own content. checksums.txt deliberately excludes
itself and pins both source specification and contract, avoiding circular hashes.

Known formulas and lane-01 misses informed this design. It is outcome-blind for
lane-02, not independently authored or a claim of real-world diagnostic accuracy.

- Reserved evaluation seeds, in order: **710021, 710023, 710027, 710029, 710033**.
- Development seeds: **101, 103 only**. No evaluation-seed data generation,
  counterfactual testing, selection, tuning or scoring before approval/freeze.
- Six variants in fixed order: matched-negative-controls, minority-stale-use,
  benign-minority-tail, within-range-subgroup, moderate-decoy, unrelated-nuisance.
- Exactly one seeded evaluation per seed/variant: **30 cases, 24,000 frames**.
  No seed replacement, rerun for better outcomes, post-result profile changes,
  chosen-tail selection or adjusted/fused score. Keep errors and incomplete cases.

## Population, target and capture

Each case contains 20 shared synthetic session clusters, 20 problem/reference
pairs per session, 400 examples per cohort. One frame/run/decision per example.
Explicit profile/unit/workload identity: lane-02-v1 / frame / lane02_target.decide.
Select reviewed synthetic labels in the same case time window, not old/new
systems; unlabeled, contradictory, missing or dropped capture stays unknown.
Pin labels, native decisions and session IDs with immutable cohort snapshots.
No implicit filter using evaluation truth or mechanism labels. No population
rates or independent-frame claims from this constructed case-control design.

The exact target and generation equations in generator-source.md govern all
variants. Crucially the target **looks up the value at previous_frame** in a
captured two-snapshot history. For affected frames, using n-2 selects -1 and
returns left; substituting n-1 selects +1 and returns right in the same branch.
The required development-only counterfactual test is specified there.
The synthetic timestamp origin is 1767225600000000000 (2026-01-01 UTC);
all offsets stay unchanged. A harness clock supplies timestamps to the
probe/exporter, never to target logic. If that cannot be done without changing
the target or an equation, stop and ask the owner.
These are independent short synthetic episodes, not continuous game histories.
No real content is captured. Capture remains default off; a future fixture-only
allowlist explicitly permits these synthetic arguments, history and return value.
Probe source changes in the plain target remain zero. The profile and synthetic
labels live outside the target; no explicit record_decision call is needed.

The minority construction intentionally discloses a second wrong-output path:
16/20 P examples/session take current_value=-1 in the current branch. This keeps
400 genuine problem outputs while only 20% involve stale use. There are no
correct outputs relabeled as problems to inflate a stale subgroup.

## Predeclared profile, roles and invariant

All listed features are captured in every record; missingness is still reported.
Role author is synthetic-lane02-contract-v1. All fields below are **candidate**
except output.lane (**symptom**, the oracle's problem selection criterion).
Role metadata is not a mechanism/decoy label and cannot affect scoring weights.

| Feature | Selector / transform | Unit | epsilon | floor | Role rationale |
|---|---|---|---:|---:|---|
| previous_age | inputs.frame minus inputs.previous_frame | frame | 1 | 1 | Candidate freshness input; actual declared-use lookup is inspectable. |
| gap | inputs.gap | ratio | .01 | .01 | Candidate branch gate; exactly matched across cohorts. |
| current_value | inputs.current_value | signed value | 1 | 1 | Candidate direct input for disclosed current-branch mechanism. |
| noise_0 through noise_6 | inputs.noise.0 through inputs.noise.6 | arbitrary | 1 | .1 | Candidates; evaluation-only truth calls them negative controls. |
| output.lane | output.lane | categorical | N/A | N/A | Symptom displayed separately under **The problem**. |

No other field is ranked in v1. Keep history, output.chosen_value, the branch
declaration and returned previous_version as evidence, not post-hoc added
features. Do not alter profile feature sets between variants. Reason/verdict
constants are not useful candidate features and are omitted in this new profile;
the lane-01 profile itself is untouched.

Invariant: declared=output.previous_used, current=inputs.frame,
used=inputs.previous_frame, expected delta=1. False declaration is not_applicable;
missing/nonintegral evidence is unknown. It is an observation of declared use,
not a score in either candidate ranking. The target's source/counterfactual
checks provide the synthetic validation of that declaration.

## Fixed scores and eligibility

**Median-only comparison stays unchanged:** minimum ten observed examples per
arm; x=median(P), m=median(R), scale=max(1.4826 MAD(R),epsilon), z=abs(x-m)/scale.
Below the declared floor return score zero. Otherwise
S=(1-exp(-z/3))*(1-(e+1)/(n_R+2)), with directional inclusive exceedance count e
exactly as frozen in internal/cohort. Categorical score remains Laplace-1 TV.
No existing constants or score behavior change. Preserve an all-feature legacy
median ranking for comparison, including the symptom's original slot.

**Separate reference-tail list:** Q uses existing interpolation at q*(n-1).
delta=max(epsilon,floor)/2; L=Q_R(.05)-delta; U=Q_R(.95)+delta. Count strictly
outside (v<L or v>U); **equality stays inside**. Compute raw a_P/n_P and a_R/n_R,
left/right counts, p=(a_P+1)/(n_P+2), r=(a_R+1)/(n_R+2), T=max(0,p-r).
Bounds [0,1]. No other smoothing, learned threshold or maximum over tails.
Need >=100 observed references, >=30 observed problems, >=five distinct known
session IDs per arm among that feature's observed numeric values; otherwise
insufficient_tail_evidence with no score/rank.
Unknown sessions/missing fields are reported, not imputed. Session IDs do not
establish independence. This is an in-sample descriptive reference-tail contrast,
not a p-value, cause probability or out-of-sample false-positive estimate.

Show **The problem**, candidate median ranking, and candidate tail ranking
separately. Tail applies only to numeric candidates here. Symptoms keep raw
counts/evidence/median score; never silently count them as candidate mechanisms.
For both positive-score candidate lists, tie-break by score descending, type
lexicographically, then stable feature key. Equal scores keep tie membership
visible; zero/no-difference and insufficient cases are unranked. Relabeling a
feature cannot change its score, only deterministic ordering inside an exact tie.
No fused score, top-ranked control, name boost or conditional comparison.

## Expectations committed before any run

Expectations below are analytic deductions from the source specification,
**not measured engine results**. Success means exposing the declared signal,
not proving general causality. Both rankings and every miss must be published.

| Variant | Median expectation | Tail expectation | Invariant expectation |
|---|---|---|---|
| matched-negative-controls | Age succeeds: P age=2, R age=1. Gap/current/noise multisets match and score zero. Age is the only positive numeric candidate. | Age succeeds: 400/400 vs 0/400 outside [.5,1.5], T=400/402. Matched controls yield zero. | P: 400 violated. R: 400 satisfied. |
| minority-stale-use | Age deliberately fails: P has 80 twos+320 ones, R 400 ones; medians both 1. Current_value succeeds (320 negative P, all positive R). Gap/noise match. | Age succeeds: 80/400 vs 0/400 outside [.5,1.5], T=80/402. Current_value also surfaces (320/400 vs 0/400 outside reference padded range); separate ranks, not a fused diagnosis. | P: 80 violated+320 N/A. R: 80 satisfied+320 N/A. |
| benign-minority-tail | Same age miss/current signal as minority. noise_0 has identical 20% elevated tails in both arms: no median association. | Age/current succeed as above. Nuisance supports lie inside the padded reference interval: nuisance T=0 exactly. This variant tests nuisance behavior in the median list only. | Same as minority. |
| within-range-subgroup (deliberate failure) | Age fails: P 80 twos, R 120 twos; both medians 1. Current_value still surfaces. | Age fails: R padded interval [.5,2.5] contains all P/R ages, so both outside counts zero and T=0. Conditional applicability is NOT added to rescue it. Current_value still surfaces. | Same as minority: R's 120 age-2 examples never use that history. Observe 80 actual P violations despite both marginal age scores missing them. |
| moderate-decoy | Age median fails; current succeeds. One UNUSED noise field shifts only +.75: its positive median association is expected and may outrank age. All other nuisance multisets match. | Age/current signals remain. Only noise_0 can score on tail, seed-dependent; it may outrank a mechanism. Retain any such miss, no promise of causal separation. | Same as minority. |
| unrelated-nuisance | Age median fails; current succeeds. Seven independent-arm nuisance fields have equal generating distributions but possible finite-sample contrasts; publish them all. | Age/current signals remain. Nuisance supports lie inside the padded reference interval, so nuisance T=0 exactly. This variant tests the median list only. | Same as minority. |

Gap's distribution is matched by pair in **every variant**, so it is not
expected to score despite its branch role. A mechanism need not differ between
cohorts. Moderate-decoy explicitly preserves an identifiability challenge at
smaller nuisance shift, not a disguised requirement to separate causes from
observationally correlated unused fields. Unrelated-nuisance is distributionally
label-independent by construction, not merely a smaller label-dependent offset.

## Task B validation and reporting, after review only

Before evaluation, implement table-driven and rapid tests using development
seeds/toy arrays only: bounds, determinism, renaming without score changes,
edge equality inside, eligibility/missing/cluster rules, constant reference,
empty data, raw left/right counts, symptom separation, and both counterfactual
checks. No test may use reserved seeds or run their generator as a side effect.
Verify no change to the frozen median formula or lane-01 reports. Check that
real intake decodes the new plain-target probe output and evidence anchors resolve.

Run all 30 cases once, fresh store, ordinary OTLP, immutable cohorts. Publish
selected/observed/missing/excluded counts, captured/dropped/exported counts,
session/shared-session counts, labels, interval endpoints/padding, a_P/n_P and
a_R/n_R, smoothed rates, median terms, both rankings, exact ties, invariant
counts, evidence IDs, timings and source/contract hashes. Distinguish attempted,
failed and completed cases; a transport failure is not a zero association.
Compute mechanism top-1/3/5 separately for median and tail candidate lists using
the predeclared evaluation truth (age only in matched; age/current elsewhere),
plus symptom-inclusive legacy median results. Report top-1 rank and the number
of positive candidates in each list beside top-5. Keep absent ranks and every
negative-control rank, including nuisance fields above known mechanisms.

Functional acceptance is exact agreement with these counts/formulas, no symptom
in candidate lists, verified development counterfactuals, native intake/evidence
fidelity, and unchanged regression checks. Record whether age reaches top 5 in
the tail list on >=4/5 seeds for each intended supported variant. Age/current
tail values are deterministic, so this is a reproducibility check, not
performance evidence; a miss is retained, never a reason to tune or replace a seed.
Do not apply that gate to the deliberately within-range failure. For unrelated
controls publish all contrasts without an invented significance cutoff. Nothing
in this contract guarantees causes rank above all decoys or proves real-world
diagnostic accuracy. Any later revision needs new untouched seeds and approval.

## Owner assumptions needing review

1. Docs-only source hashes pin a **generator specification**, with executable
   source hashes deferred to the separately approved Task B source-freeze commit.
2. Short two-snapshot episodes are sufficient to validate actual version-based
   value use here; continuous histories remain a future benchmark.
3. A disclosed current-value mechanism supplies the other 80% of minority-case
   problem outputs; correct outputs are never marked wrong to fill a cohort.
4. Conditional comparison, control fields and related variants are deferred;
   the within-range miss is retained rather than repaired by applicability filtering.
