# Real-case gaps — Stage 1 / Checkpoint A

2026-10-21. Synthetic/development data only. **Design, not implementation.**
Stop here pending the owner's literal `GO BUILD`. No selector, cohort, scoring,
trace, probe, recorder or schema behavior is changed. Lane-02 remains closed;
no lock, contract, freeze, published result or reserved seed is touched.

## Source boundary and versions

- Requested design branch: codex/real-case-gaps, from latest local/remote
  codex/wrapper-overhead **474f53fb58fbc4df875371b9831a2b8ce506ce29**.
  That runner rejects Class.method and predates tuple/sampling support.
- Measured engine, isolated and unchanged:
  **a2c2cee3861c3eb6131415469c86df3b1d81b5d5** (engine-3). The owner supplied
  this as the original scenario's engine version. No behavior was cherry-picked
  onto the design branch. This is an explicitly disclosed measurement dependency.
- The owner supplied the single 27-line probe config and exact target-only
  folder/command. Only that YAML was copied from artifacts; no explanation JSON,
  trace, doctor output or blind-test report was read. The folder was inventoried
  before copying. No answer/sealed entries appeared. TESTER_README.md was left
  unopened and excluded, together with other README files, editor settings,
  bytecode, Git metadata and the generated scenario_results.csv.
- The 26 copied Python files hash identically to their sources before/after.
  Target source edits: **0**. Target implementation was not opened or inspected.
  Dependency metadata says the scenario uses only Python's standard library.
  CPython **3.12.12** and the existing locked adapter environment were used.
- No rankings, cohorts or new profile were created. The revealed defect is not
  encoded in the engine, selectors, formulas or design. This is no longer blind.

## Unchanged reproduction

80 episodes, seed 20261020, unpaced, fresh real SQLite store; serving defaults,
queue capacity 256, unchanged config. The YAML SHA-256 is
`dcad6b2669f3c36f3c91099505510fa7fb464ac4561400c5131fef923ed96f18`.

| Observation | Owner's supplied historical summary | New plain reproduction |
| --- | ---: | ---: |
| Target rows / failures / successes | 80 / 17 / 63 | 80 / 17 / 63 |
| Attempted spans | 1,834 | 1,834 |
| Queue-full drops | 1,554 | 1,464 |
| Acknowledged / failed / unknown | Not supplied | 370 / 0 / 0 |
| Stored run IDs | 14 | 19 |
| Runner exit | 5 | 5 |

The loss differs and is retained without tuning/retrying to match it. Stored run
IDs are **not a count of complete episodes**. The existing engine has no durable
per-run completeness manifest. CSV bytes are identical across baseline, plain
and separately instrumented execution. Plain elapsed wall time including startup
of the Python runner and flush: 2.069184 s; not a steady-state throughput claim.

Per-probe counts from the plain run:

| Probe | Calls seen | Sampled out | Selected | Serialized | Admitted | Admission drops |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| run_scenario.run_episode | 80 | 0 | 80 | 80 | 15 | 65 |
| Environment.step | 20,580 | 19,743 | 837 | 837 | 170 | 667 |
| run_scenario.select_action | 20,580 | 19,743 | 837 | 837 | 170 | 667 |
| Separate automatic root spans | — | — | — | — | 15 | 65 |

Selected = calls minus the probe's sampled_out counter. Here drop_invalid=0 and
the sum selected equals the native serialized counter (1,754), so serialized
counts can be attributed exactly. Admitted counts come from per-probe doctor
counters; rejected = serialized minus admitted. All admission failures in this
run are queue-full. Eighty separate root spans bring attempted to 1,834.
Four-state conservation: **1,834 = 370 acknowledged + 0 failed + 1,464 dropped +
0 unknown**. Sampling exclusions are outside that span-attempt denominator.
Per-probe ACK distribution is not recorded and is not invented from admission.
The doctor's per-run summary ring retains 32 entries and reports 128 omissions
here; it is not a complete per-run audit. Global per-probe counters are intact.

## Before-queue timing: separate diagnostic, not an optimization

diagnose_burst.py compiles private AST timing copies of the unchanged engine.
Tests verify fixed-record OTLP bytes match, exceptions propagate identically,
and counters retain bounded aggregate memory. Source files and target functions
are not rewritten. No payload is printed by the diagnostic. Timers/locks perturb
cost and thread scheduling, so the instrumented run is **not** the reproduction
and is not compared with the closed overhead budgets.

| Instrumented phase mean, microseconds | Episode | Step | Select action |
| --- | ---: | ---: | ---: |
| Sampling branch, per call | 0.706 | 1.259 | 1.373 |
| Argument snapshot/binding, per selected call | 33.443 | 40.565 | 51.615 |
| Decision/lineage JSON total, per selected call | 48.623 | 49.922 | 40.800 |
| Record preparation exclusive, per selected call | 118.6 | 131.1 | 91.6 |
| Selected wrapper, excluding target body | 411.969 | 316.581 | 282.352 |
| Sampled-out wrapper, excluding target body | — | 7.406 | 7.438 |
| Queue admission, per item | 7.655 | 6.837 | 8.479 |

Timing values are rounded from the retained JSON. The outer method guard is not
included in wrapper measurements. Argument snapshot includes recursive snapshot
children; inclusive/exclusive phases must not be summed together. JSON encoding
is a separate selected-call cost: 3.890 ms episode, 41.785 ms step, 34.150 ms action
across all decision/lineage JSON calls. OTLP protobuf encoding occurs on the
network worker, **after** caller enqueue: 15 batches / 36.959 ms in this diagnostic.
It is not a caller-side pre-queue cost.

Sampling **already precedes** argument/receiver snapshot, return capture and JSON
in a2c2cee. Each high-rate probe has 20,580 sampling-branch invocations, but only
837 argument-snapshot invocations. Sampled-out calls are much cheaper in this
diagnostic, but not free: counters, context lookup, branching and method guards
remain. No budget pass is claimed and no wrapper optimization is proposed here.

The diagnostic stored 29 run IDs, acknowledged 452 spans and dropped 1,382, with
zero failed/unknown and exit 5. Its 80 / 17 / 63 CSV is byte-identical. The larger
capture is an **observer effect**, not a fix or evidence of improved throughput.
One plain run and one diagnostic run are retained; no best-run selection.

## A. Burst capture options and recommendation

| Option | Benefit | Trade-off / recommendation |
| --- | --- | --- |
| Sample before snapshot/serialize | Avoid expensive work on excluded calls | Already implemented in measured engine; keep it. Do not rebuild/tune the wrapper. Deterministic selection can alias periodic behavior; report actual indices and coverage, never causal capture probabilities or random-sampling estimates. |
| Larger bounded buffer | Absorb a finite producer burst | Candidate for a separately approved operational experiment, not a wrapper optimization. Bound both item count and retained bytes including in-flight batches. Choose from declared workload/field limits, not the observed 1,464 drop count. Sustained excess still loses data. |
| Worker-side waiting/draining | Worker may wait/retry/export while caller never waits | Already has one network operation slot and bounded retry. A waiting worker cannot prevent a full producer queue. Do not reinterpret this as backpressure on the host or add unbounded staging. Durable spooling is a larger deferred design. |
| Run completeness manifest | Prevent a partial capture from masquerading as a complete sample | Required first. Missing final manifest means unknown; any confirmed lost selected item means partial. Unsampled calls are sampling omissions, not queue drops. |

At the configured caps, 80 episodes x (40 step + 40 action + 1 episode + 1 root)
is at most 6,560 span attempts if this nesting holds. This is a finite declared
upper bound, not a default allocation recommendation or an assertion that all
payloads fit a safe memory budget. Current measured total is only 1,834 and still
overwhelms the queue. A finite nonblocking recorder cannot promise zero loss for
arbitrary unpaced workloads: reject/flag partial rather than hide this constraint.

### Proposed completeness/accounting contract

Keep distinct populations: calls seen; selected calls; unsampled calls; invalid
elements; serialized decisions; root spans; admitted spans; terminal ACK/failed/
dropped/unknown. Define denominators in every report. Terminal network failures
can be unknown after ambiguous delivery; queue rejection is a confirmed local
drop. A drained queue is not proof of ACK. Duplicate retries must stay idempotent.

Proposed versioned capture manifest per run/probe: rule and indices/counts,
selected/admitted/rejected counts, per-state terminal counts, element detail
loss, and producer epoch/sequence/final marker. A final marker is eligible only
after all referenced selected evidence settles; if marker delivery fails or the
process crashes, status is unknown. Keep bounded manifest state; exhausted state
capacity is itself visible unknown coverage, never silently complete. Caller
exit must still never flush/join/network/wait for space or alter an exception.

Store manifests as ordinary explicitly owned telemetry using existing JSON/raw
storage and writer queue, not an extra SQLite writer. Root arrival alone cannot
prove completeness. Do not map partial capture to an application failure/outcome.
Sampling state and transport completeness are separate: a successfully exported
sample is still incomplete for absence/version-coverage claims.

A bounded capture-scope manifest must also count expected/seen run IDs: a run
lost entirely cannot be discovered from its absent per-run manifest. Exceeding
the scope inventory budget reports an unknown run population, never a smaller
apparently complete denominator. Do not infer that 19 stored IDs were the only
episodes executed when producer/target observations establish 80.

New opt-in cohort selection should exclude confirmed partial runs by default,
and reject unknown completeness when complete capture is required. An explicit
include-partial option must retain the flag/counts beside every aggregate and
selection. Historical runs without manifests remain unassessed, not implicitly
complete; preserve legacy archived profiles/results unchanged.

## B. Profile selectors, optional invariants, derived features

### Literal keys

Options: replace all paths with JSON Pointer; add bracket syntax; add escapes.
Recommend an opt-in `selector_syntax: escaped-v1` profile field, omitted on all
legacy profiles. Without it, keep the current literal split behavior exactly,
including backslashes that were valid literal key characters. This prevents a
new escape parser from changing an old selector's meaning. With the explicit
version, use the escaped path grammar: unescaped dot separates
segments, `\.` is a literal dot, `\\` a literal backslash. Thus
`inputs.self\.score` addresses the single input name self.score, whereas
`inputs.self.score` retains its existing nested meaning. JSON profile files must
escape the backslash (`"inputs.self\\.score"`). Reject dangling/unknown escapes;
retain <=256 bytes / <=8 decoded segments and bounded sequence indices. No
fallback lookup that guesses between dotted and nested keys. Test multiple dots,
literal backslashes, empty/malformed segments and map-key versus sequence-index
ambiguity; preserve all valid historical selectors.
Pin the syntax version in cohort/profile snapshots. Malformed-escape rejection
applies only to escaped-v1, not retroactively to legacy profiles.

### Optional invariant

Current ValidateProfile requires three selectors and a positive delta. Recommend
an optional complete invariant object: absent/null means **no invariant declared**,
not satisfied and not a fabricated unknown evaluation. A present object must pass
all current validation; partly supplied objects are errors. Invariant values and
coverage remain observations. Keep declared-version semantics/formula unchanged;
do not introduce a general behavior invariant in this task. Existing profiles
and frozen result JSON must remain byte-identical on the legacy path.

### Declared derived values

Source audit correction: NumericFeature already has **Subtract**, applied by both
median and tail extraction. Difference is not wholly missing. Recommend preserving
that exact legacy path and adding an opt-in, nonrecursive expression object for
`difference(a,b)`, `abs_difference(a,b)` and `ratio(a,b)` only. No eval, calls,
imports, assignments, feature-name boosts or expressions over other expressions.
Operands are validated selectors, with bounded numbers of features/reads.

Require declared output unit, epsilon and floor as now; require compatible input
units for difference and explicit ratio units. Ratio additionally declares a
positive denominator tolerance; abs(b) <= tolerance gives null, never a clamp.
Missing, capture-off, hash-only, unsupported, nonfinite or overflowed operands
give null plus a reason. Do not replace them with zero. Evidence pins both source
field selectors and native IDs; an expression is an analyst declaration about
recorded values, not proof of code reads or causation. Existing input_value and
separate tail outputs suffice; no new difference type/formula/constant.

## C. Cohort predicates and per-run aggregates

Source audit correction: captured **scalar equality** predicates already exist.
The missing part is declared numeric inequalities/ranges. Options are an SQL-like
query language or a finite typed predicate. Recommend only `lt/le/eq/ge/gt` and
explicit interval endpoint inclusion over one recorded/derived/aggregated finite
numeric value. No query language, SQL fragments, hidden threshold search or eval.
Missing/invalid value is predicate_unknown, not false. Require explicit time
bounds/limit, pin selection/profile/threshold/provenance and run membership.
Selecting on total return is descriptive outcome-based selection and may create
selection associations; publish it separately from human/scenario marks.

Current Decisions scans a bounded population/evidence budget but pins only the
latest native event per step/run. Recommend opt-in count/min/max/mean/first/last
over **all captured decisions for that step** in each selected run. Keep one
observation per run, not one independent sample per loop iteration. First/last
order by timestamp then native evidence ID, with source sequence when recorded;
unknown clocks/sequence coverage remain disclosed. Define count_present separately
from count_selected/seen and numeric-observed count. Empty numeric aggregates are
null (count_present may be zero only with sufficient completeness to establish it).
Mean uses deterministic order and checked finite arithmetic; no silent overflow.

Every aggregate pins its full bounded input evidence set, operator and coverage
metadata under a consistent read snapshot. Example citations must not substitute
for the complete membership manifest. If query/evidence budget is exceeded,
return an explicit error/incomplete aggregate; do not silently take LIMIT and
call it all decisions. Keep current time/limit/query-index constraints, and add
EXPLAIN tests for any new SQL. New aggregate membership may use optional versioned
fields in the existing cohorts.data JSON; old DecisionIDs stay readable.

Current readDecisions has a **10,000 span/event record budget across the selected
population**, including unrelated steps and lineage events. Correcting queue loss
can expose this second bound. At the declared maximum there may be 6,560 spans
plus many events, so complete capture is not proof that a single current cohort
read can aggregate it all. Preserve the budget/error; do not raise it silently or
weaken its test. Design explicit bounded windows/pages with immutable membership
and a consistent snapshot, or report the aggregate unavailable. If efficient
bounded filtering requires a new schema/index, invoke the migration stop rule.

Sampled aggregates are named **sampled min/max/mean/first/last/count**, not true
episode summaries. Observed first/last may miss real endpoints; no reweighting
can recover deterministic cap-excluded data without extra assumptions. Surface
selected/seen/ACK/detail-loss counts and all unknowns. An invariant may report a
witness violation on a captured call, but cannot report universal satisfaction
or absence across unsampled calls.

## D. Tail eligibility: keep minimums

Keep the unchanged implementation's **>=30 problem, >=100 reference numeric
observations and >=5 known sessions in each arm**. Sampling does not manufacture
independent observations. Under the recommended per-run aggregate unit, an
80-episode split can never meet 30+100: **0% eligible even with perfect capture**.
The reported 17 failure / 63 success split separately misses both count minima.
Nineteen stored runs in the plain reproduction cannot improve that. A single
fixed seed does not establish five independent sessions; the config declares no
session identity that would justify that claim.

For any future dataset, report per-feature nP/nR, observed/unknown sessions and
eligibility as collected, not only total captured calls. At best, if known-safe
complete per-run coverage fractions are qP/qR, collecting 30/qP and 100/qR runs
is a planning lower bound, not a guarantee or an independence claim. Do not
inflate n using within-episode sampled steps or lower minima for this case.
Stage 3 should retain insufficient_tail_evidence when ineligible; a demanded
tail rank is then unavailable, not a zero association or a test failure to tune.

## Recommended build order after GO BUILD

1. Preserve accounting, manifests and explicit partial/sampled/unknown status;
   test terminal state conservation and duplicate/ambiguous delivery. This guards
   all later selections from silent partial capture.
2. Run one separately approved larger-buffer experiment bounded by items AND
   bytes, holding sampling/payload/workload constant. Keep failure results. Do not
   tune wrapper overhead or silently change sampling to rescue acceptance.
3. Add escaped selectors and optional invariants, legacy serialization unchanged.
4. Add pinned sampled per-run aggregates and numeric predicates with bounded
   queries/EXPLAIN tests; do not conflate marks and outcome-based predicates.
5. Extend declared derived features minimally, reusing Subtract where appropriate.
   Preserve median/tail formulas and both source evidence refs.
6. Only then perform the owner-approved non-blind acceptance retest, with config
   fixed before ranking. Report honest tail ineligibility for 80 per-run units.

Each step needs the specified table/property/regression tests and separate
before/after measurements. No Stage 2/3 command has been run. The lane-02 closure
conflicts with later requested development reruns if interpreted as forbidding
those too; before GO BUILD record the narrow development-only exception and never
touch historical attempts/contracts/results or reserved seeds.

## Migration assessment and stop condition

No migration appears unavoidable: capture manifests fit ordinary telemetry JSON;
profiles, predicates, pinned aggregates and completeness snapshots can be versioned
inside existing cohorts.data. This is a proposal, not proof of indexed scalability.
Do not persist derived state into unrelated run env/config as an implicit schema.
If bounded/indexed query plans or durable capture bookkeeping require a new table,
column or index, **stop and ask before adding an embedded migration**.

The riskiest assumption is that enough useful per-run evidence survives finite,
nonblocking capture to support a comparison. This run disproves treating stored
run IDs as complete capture, and 80 runs cannot qualify the existing tail. One
revealed, non-blind scenario does not establish general diagnostic accuracy.
