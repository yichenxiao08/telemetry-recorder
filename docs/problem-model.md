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
