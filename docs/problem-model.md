# Problem model (design v0; Slice 1 subset implemented)

Extends the failure-oriented design while preserving M3/M4b scoring. See the Slice 1 implemented boundary below; other sections describe the broader design.
No new schema or scoring code is implemented in the feasibility spike.

## Problems and labels

A problem is a developer-authored question with a profile/version, unit of
analysis, bounded eligible population, selection rule and optional expectations.
It may concern any behavior. A run may be a request, turn, frame, job or session.
Operational outcome and assessed quality remain independent dimensions.

A label is a scoped assertion from an outcome rule, deterministic predicate or
human annotation. Automated sources record rule/version and evidence; they never
pretend to be human reviewers. A direct mark is also an annotation.

Proposed append-only annotation:
id, target run/output/decision refs, dimension, value, reviewer/source,
rubric_id/version, created_at, recorded_at, evidence_refs, rationale_ref,
supersedes/retracts and source/version.
Corrections append; they never rewrite execution timestamps. Reviewer disagreement
remains explicit unless an adjudication policy is selected. Simulated reviewers
are marked synthetic and do not establish actual human validity.
Future annotation storage is separate from run execution, with the existing
single writer queue and read-only cohort queries. No table is added now.
OTLP label logs in the spike demonstrate transport only.

Unlabeled = unknown, not satisfactory/pass. Conflicts, retractions and incomplete
predicates are distinct from explicit negative labels. A label proves only that
its source made that judgment, not objective answer quality.

## Cohort selection

A problem cohort is a persisted set from direct marks, versioned predicates or
specified reviewer/rubric labels. A reference cohort explicitly selects examples
with the same unit/workload/profile and a stated reference interpretation,
e.g. reviewed satisfactory. The same core/version can supply both sets.
An implementation change is not required. Keep M3's historical pass baseline mode.

Pin selection time, filters/version, exact run and label-revision IDs, exclusions,
capture completeness and reference policy. Reject overlap in disjoint comparisons.
Require time bounds and limits. Match declared intent/stage/context features when
appropriate and report exclusions; do not silently match away a config feature
under study. Unknown labels/detail never become implicit reference examples.
Session/drive IDs define clusters: frames and conversational turns are correlated.
A session-level run requires a profile, not accidental trace/ML or host grouping.

## Invariants: no baseline needed

An invariant declares scope/profile/version, applicability and an expected
relation: "If input X was declared used at this step, output Y must include X in
its transitive derived_from lineage through that step."
References identify versions: a lane from frame N-2 differs from frame N-1.
A profile may declare an expected previous-frame ref or typed input equality
predicate. No arbitrary query language is required.

Bounded graph evaluation yields:
- satisfied: applicable, complete capture and the path exists;
- violated: applicable, complete capture and the path demonstrably absent;
- unknown: missing/evicted detail, unresolved refs, budget exceeded or an
  untrusted completeness declaration;
- not_applicable: antecedent known false.

Cycles, ID conflicts and malformed graphs yield integrity findings plus unknown,
never a silent pass. Persist rule/version, coverage, evidence and negative search
scope. A completeness assertion is not independent proof of instrumentation
fidelity. No baseline minimum is required. Violations show disagreement with a
developer's expectation, not proof of a cause.

## Proposed differences: design only, no constant tuning

Align profile + stable step ID + field + declared per-run reduction.
Use the last completed verdict/option, or explicit presence/count reduction for
loops. Do not treat repair iterations as independent runs.
Missing fields remain unknown with coverage; explicit "none"/"declined" is a value.

Require ten observed runs per cohort for ranked associations, retaining the
existing conservative minimum as a proposal. Smaller cohorts show descriptive
counts/invariants with insufficient-evidence status. Existing rules do not change.

### Verdict, reason code and chosen option

For problem/reference observed sizes nP/nR, category counts a/b, and K categories
in their observed union:
p=(a+1)/(nP+K), q=(b+1)/(nR+K).
Per-category excess score=max(0,p-q); whole-field score=0.5*sum(abs(p_c-q_c)).
Both lie in [0,1]. Rank the whole field once; show its strongest excess category
and retain others as related. Laplace +1 follows existing smoothing discipline.
Show raw a/nP and b/nR. No name-based bonus. This measures association, not
whether a verdict or reason was correct.

### Input value shifts

For a finite scalar with units, let x=problem median, m=reference median,
s=max(1.4826*MAD(reference), positive profile-declared epsilon in those units),
z=abs(x-m)/s.
For an increase, e counts reference values >=x; for a decrease, <=x:
score=(1-exp(-z/3))*(1-(e+1)/(nR+2)).
Require a profile-declared absolute effect floor. Missing units/epsilon yields
unsupported, not an invented universal floor. Reuse M3 MAD factor 1.4826,
squash 3 and smoothing +1/+2. No engine constant changes.
Report distributions, direction, raw e/nR, coverage and decision/input refs.
Hash-only values support permitted equality comparison, not numeric shifts.

Future acceptance: table/rapid tests for bounds, identical cohorts yielding no
categorical excess, monotone excess with fixed reference, unknown on missing
observations, loop aggregation, stable ties and no feature-name boosts.
Numeric monotonicity holds with a fixed reference, as in M3. Scores are not
significance levels or causal probabilities.

## Report tiers and sample size rule

1. Observation: directly supported value, recorded step/lineage, annotation or
   invariant result; cite evidence and any transformation.
2. Association: a measured difference between explicit cohorts. EVERY association
   states both raw numerators/denominators, unique run count, independent
   session/drive count, missing/excluded count, selection and overlap policy
   beside the claim. Unknown independence is stated explicitly. No population
   frequency claims from hand-picked examples.
3. Hypothesis: possible mechanism, alternatives and missing evidence, with a
   developer-controlled experiment, predicted result and a falsifying result.

Signal Scout makes no causal claim at any tier. Citation verification checks
membership/value fidelity, not semantic truth or causality.
"Reviewer A marked four replies too strict" is supportable; "these replies are
objectively too strict" is not implied. No reference cohort means no association;
within-run expectations and instrumentation suggestions remain available.
Suggested experiments do not implement saved-input replay or autonomous edits.

## Absence and different-from-before questions

"Why did X not happen?" requires a recorded opportunity: candidate alternatives,
applicability/gating decisions, rejected-option reasons, expected inputs and
capture coverage. A missing event alone cannot distinguish skipped, dropped,
inapplicable or uninstrumented work. With a complete applicable decision, report
the recorded rejection reason; with a declared expectation check its invariant.
Otherwise report unknown and ask for the specific missing signal.

"Why different from before?" selects explicit time/revision/config reference
cohorts with comparable profile/unit/task and pinned IDs. A before/after change
is an association with temporal/config differences, never a causal conclusion.
The M3 historical baseline modes remain useful for operational failures. A
single unchanging system can also compare marked and reviewed reference examples.

## Proposed small implementation order

P0 capture/privacy -> P1 general helper/guided setup -> P2 marking, append-only
labels/cohorts -> P3 lineage view/invariants -> P4 cohort differences ->
trimmed pre-registered M5 -> adapted M7 reports -> remaining M8 measurements.
M6's adapter role is absorbed into the general helper; training breadth is optional.
Each requires authorization. This pass completes design and the source audit;
the executable helper/replays remain pending under the explicit fallback. No M5–M8 starts.

Structured report types permit only observation, association and hypothesis.
A verifier can reject an unsupported type, missing denominator or wrong evidence
membership. It cannot guarantee detection of every causal overinterpretation in
free text or settle subjective quality. Keep that limitation visible.

## Slice 1 implemented boundary (2026-10-20)

The owner explicitly authorizes a synthetic-only slice through marking, cohorts,
categorical/input shifts and ONE declared-used version invariant. This supersedes
this document's historical "design only/pending" statements for that narrow subset.
P0 production privacy and M5–M8 remain unstarted; no hypothesis generation.

Migration 009 adds append-only annotations and immutable cohort snapshots.
CLI marks post JSON to the EXISTING serving writer; reads use the read-only pool.
Created/recorded review times do not become telemetry/run/context times. Revisions
retain run/dimension/source/reviewer, have one successor, and remain inspectable.
Conflicting active label values exclude a run from label cohorts; no implicit
adjudication. Predicate selection is bounded captured-scalar JSON equality only.

Cohorts require explicit start-time bounds <=31 days and eligible-population limit
1..1000. Oversize is an error, not silent selection truncation. Store exact run,
label and native decision revisions, unit/profile/workload, selection/exclusions,
and session IDs. The last decision per configured step/run is chosen when creating
cohorts; later decisions never replace it in a saved selection. Overlap or profile
mismatch is rejected. Raw event reads are bounded to 10,000 records per selection;
annotation reads are bounded; labels are never inferred from log-template text.

The formulas above are implemented unchanged in internal/cohort. Numeric units,
epsilon and absolute floor come from a validated immutable profile; subtraction
of two captured scalars is a declared generic feature transform. Returned string
categories are selectable alongside verdict/reason/chosen fields. Score zero is
an observation with no difference, not a ranked discovery. Each feature needs ten
observed examples per cohort; missing capture is excluded with explicit coverage,
never imputed zero. Reports retain raw numerator/denominator, selected/observed/
missing run counts, selection exclusions and session counts beside associations.
Session IDs identify clusters, not proven independence; shared sessions are explicit.

The invariant is parameterized by declared-use/current/used selectors and delta;
missing/nonintegral values yield unknown, a false declaration is not_applicable,
and a matching/differing declared version yields satisfied/violated. It is an
observation without a baseline, not a proof of arbitrary reads or causation.

Still design: semantic-ID conflict handling, full graph traversal, artifact access,
privacy/redaction, richer predicates/cohort matching, graphical mark UI, categorical
reason enrichment, durable report history, hypotheses and suggested experiments.
The exact lane evaluation contract is frozen at b537ee1 in problem.yaml and
slice1-scoring.md; authored_blind is explicitly outcome-blind, not independently
written. The engine never reads benchmark mechanism/symptom/decoy labels.

Finite captured values can overflow floating-point median/MAD subtraction.
Slice 1 explicitly emits unsupported_numeric observations for undefined numeric
arithmetic, preserving observed counts and avoiding NaN or a false no-difference
association. This guard changes no formula/constants or valid lane-01 score;
all 15 complete reports were recomputed from pinned evidence and remain identical.

## Owner decision: v1 narrowed, Task A review checkpoint

Symptom/candidate roles and the separate reference-tail list below are approved
in principle, with implementation gated on review of the docs-only lane-02
contract in benchmark/problems/lane-02/contract.md. Task A writes and commits
that contract first; Task B has NOT started and needs a separate owner approval.
Conditional comparison is **deferred**, including its control fields and variants.
The earlier two-addition proposal and variant table below are historical design,
superseded for execution by lane-02's six-variant contract. Do not implement or
run its session-adjustment/disjoint-support variants. No M5–M8 or P0 starts.
Frozen lane-01 source, profiles, constants and results remain unchanged.

## Proposed v1 (historical proposal; narrowed by owner decision above)

The lane-01 diagnostic is in docs/lane-01-diagnostic.md and generated raw tables
in benchmark/results/lane-01-diagnostic.md/.json. Heavy-case decoys have genuine
large marginal shifts introduced by cohort construction; positive-score ties
are not responsible. Harder loses the 20% age subgroup at the median. These
findings motivate **at most two additions** below. Existing scores/constants,
profiles and results stay frozen. This section specifies proposals, not working
features, a causal model or permission to start P0/M5–M8.

### 1. Conditional comparison with explicit common-support reporting

Question: does an association persist within comparable, declared contexts?
The profile predeclares at most two captured, pre-output adjustment fields,
categorical levels or numeric bin edges, and the comparison question. Use
domain-reviewed nuisance/context fields, not evaluation mechanism/decoy labels.
Session regime may be a control when it spans multiple clusters; a unique
session ID cannot meet the per-stratum five-cluster rule and yields unsupported.
Reject the candidate itself and all symptom/selection-output fields as controls.
Do not automatically condition on the top-ranked fields: selecting and testing
on the same observations is circular, and controlling a mediator or collider can
destroy a real mechanism or create an association. Automatic control selection
would require a separately reviewed, cluster-disjoint discovery/validation design.

Let h denote the fixed cross-product stratum. Eligible h needs at least ten
observed problem and ten reference examples and five distinct session IDs in
each arm. Unknown session IDs are not independence. Missing controls/values are
excluded with counts, never a zero stratum. Compute the **unchanged v0** numeric
score S_h (same unit epsilon, floor, 1.4826, k=3 and +1/+2), or Laplace-smoothed
categorical TV within h. Let H be eligible strata, a_h observed reference count,
and w_h=a_h/sum_H a_h. Report S_cond=sum_H w_h S_h, in [0,1], together with
all stratum medians/counts/directions and marginal scores. Preserve stratum
directions; a high absolute summary cannot imply one common directional effect.
The weights standardize to the explicitly chosen reference population, never
inverse tiny propensity weights. Per-stratum smoothing is exactly the existing
formula; no extra fitted shrinkage or post-ranking feature boosts.

Report supported fractions c_P=sum_H n_Ph/n_P and c_R=sum_H n_Rh/n_R.
Require both >=0.8 and at least five known clusters per arm overall. Otherwise
the aggregate is **unsupported_overlap**, with no rank or zero-score claim;
show supported-stratum observations only. Even when valid, state that the
estimand covers H and print excluded strata. Each stratum retains its own
cluster counts/shared IDs. No independent p-value is inferred from frame count.
The thresholds are proposed engineering rules to freeze before new evaluation,
not guarantees of precision. A changed proposal requires a new version.

New failure modes: sparse strata, bin-boundary sensitivity, lack of positivity,
unmeasured confounding, noisy control capture, conditioning on mediators/colliders,
and dependence within sessions. Simpson reversals may reflect different
questions rather than mistakes. A lower conditional score does not prove a
field irrelevant. Lane-01's disjoint gap support would make gap adjustment
unsupported; this is an expected limitation, not a false diagnosis to suppress.
Session adjustment cannot remove the deliberately within-session decoy offset.

### 2. Reference-tail coverage for minority shifts

Question: how many problem examples lie outside the reference's ordinary range,
even when the central median is unchanged? For each predeclared numeric field,
compute L=Q_reference(0.05)-delta and U=Q_reference(0.95)+delta using the existing
interpolated quantile definition. Here delta=max(profile epsilon, profile floor)/2
in the declared unit, frozen before evaluation. Strictly outside means x<L or
x>U; equality stays inside. The padding prevents tiny quantization changes
from creating a tail and handles zero-MAD/constant references. The full interval,
raw unpadded quantiles and delta must be displayed.

Let a_P/a_R be raw numbers outside this **same fixed reference interval** among
n_P/n_R observed values. Laplace rates p=(a_P+1)/(n_P+2), r=(a_R+1)/(n_R+2).
Proposed score T=max(0,p-r), in [0,1]. Report left/right counts separately and
the combined count once; no post-hoc choice of whichever tail scores higher.
This is an in-sample descriptive reference-tail contrast, not an out-of-sample
false-positive estimate, significance test or probability of a cause. The
reference estimates both the interval and its own tail count; that optimism
must stay explicit. No minimum observed tail count is silently imputed.

Require >=100 observed references (nominally five per 5% tail), >=30 problems,
and >=five known session IDs per arm. Below these counts report insufficient
tail evidence without a score/rank. Preserve missing/excluded/shared-session
counts and label-selection policy. These minima are feasibility guards only.
T does not boost names, mechanism labels, output roles or known demo features;
all eligible fields get the same rule. It is reported separately from median
score, with its own ranked list, **not** max(median,T) or a tuned fused score.
Two lists cost attention and multiple comparisons; do not hide that cost.

New failure modes: legitimate heavy tails, a contaminated/shifted reference,
subgroups within the reference interval, minorities smaller than sampling noise,
cluster concentration, high dimensional chance contrasts, and padding that hides
small real effects. It will also rank truly shifted but computationally unused
decoys. It addresses median blindness, not causal identifiability. Reference
intervals are marginal unless an explicitly authorized conditional-tail extension
is reviewed later; that extension is not a third proposal in this task.

### Symptom declarations and presentation

A proposed versioned profile role is symptom or candidate (default candidate),
with selector, rationale and author recorded **before** evaluation. Output
quality/selection criteria such as output.lane are declared symptoms: show them
under **The problem**, with raw counts, labels, evidence and existing score;
exclude them from the ranked candidate-mechanism lists and their denominators.
Retain original combined ranks for historical comparisons. Excluding a symptom
is presentation, not evidence that remaining fields are causes. Never use a
benchmark's hidden mechanism/decoy labels to suppress or boost candidates.
Store role versions with future reports; reject a field declared both symptom
and adjustment control. Changing roles after outcomes requires a visible new
profile/version and cannot rewrite prior acceptance results. This is metadata
and presentation design, not an implemented scoring addition.

### Fresh preregistration proposal (no execution)

Reserve seeds **710021, 710023, 710027, 710029, 710033** for the new evaluation.
These are absent from the prior fixture source/preregistrations; neither data
generation nor scoring on them has run. Keep 101/103 and all lane-01 evaluation
seeds solely as old/development evidence. No seed may be replaced for poor ranks.

Before implementing/running evaluation, commit a separate lane-02 contract with
generator equations, checksums, unit/resolution/roles, controls/bin edges,
sampling/label rules, known mechanism/symptom/negative-control labels, expected
failure conditions and this seed list. Have the owner review it; retain an
author-separated review if available. Then freeze scorer and fixture revisions
before the first new engine run. No retrospective modification of lane-01.

Planned population: 20 synthetic sessions per seed/variant, 40 decision examples
per session, 800 total, balanced 20 problem/20 reference in each session. Session
is a supplied dependence cluster, not an independent frame history. Use a plain
target whose gap mechanism and declared version use are visible, record all
fields, and keep output/synthetic labels separate. The preregistered generator
must explain how labels arise; an assembly flag is never an argument to the
target. Matching stress variants intentionally relax within-session balance as
declared below. Freeze exact equations before any engine execution; this is
the review plan, not an already frozen/runnable lane-02 benchmark.

| Planned variant | Construction and expectation to freeze |
|---|---|
| Matched negative controls | Interleaved sessions/time, nuisance fields independent of labels; gap shifts and 100% stale subgroup. Check median/tail evidence, no output in candidate list. |
| Session nuisance imbalance | Half of sessions have 30 problem/10 reference, half 10/30; one nuisance level shifts by session regime only and is unused by target. Every session has both labels. The two declared regime strata each span ten clusters. Regime adjustment should reduce the marginal contrast; report any residual and supported fractions. |
| Minority stale use | Balanced sessions; 20% of problem examples use a stale declared version, none of references. The target must actually take the declared-use branch. Expect zero median contrast but excess tail/invariant observations, not proof of a fixed output under intervention. |
| Benign minority tail | Same minority distribution in both arms, including a shifted nuisance tail unrelated to target use. Measure false associations and any tail rank, retain all outcomes. |
| Deliberate failure: disjoint support | Label-correlated unused nuisance shift like lane-01; candidate controls fully separate cohorts. Expect conditional unsupported_overlap; tail may still rank decoys high. Failing to recover causal relevance is an expected miss, never converted into a pass for diagnosis. |
| Deliberate failure: within-range subgroup | Reference already spans both subgroup values; problem minority changes while both medians remain equal and values stay within the padded interval. Expect median and tail misses; declared invariant can still expose a violation if supplied. |

Future acceptance: table tests/rapid for bounds, deterministic bins, padding,
missingness, support exclusions, no renaming effects, unchanged host/core
regressions and frozen lane-01 outputs. Evaluate every seed/variant once with
fixed policies: raw and symptom-separated top-1/3/5 mechanism recall, negative
control ranks/rates, per-stratum coverage, tail counts, invariant results,
unknown/unsupported counts and wall time. Publish each candidate list separately,
including ties and all misses. For planned supported cases require the declared
mechanisms within top 5 in at least 4/5 seeds; publish failures rather than tune.
For null controls publish false-positive contrasts without inventing calibrated
significance. Require unsupported on every deliberately empty-support case.
Do not set causal-accuracy acceptance on the nonidentifiable case. Any new design
after looking at these seeds needs another untouched evaluation set and approval.
