# Wrapper overhead: Stage 1 protocol and Checkpoint A

Owner request: profile only on `codex/wrapper-overhead`, based on recorder fix
`eb9225e`. Stop at Checkpoint A. No optimization, delivery-policy change,
scoring change, migration, new difference type or later milestone is authorized.
Only synthetic fixed payload and development seeds 101/103 are permitted.

## Measurement choices made before running

- Keep Python uv-pinned at 3.12.12. Record interpreter, OS, CPU and GC settings.
  This measures one Python version; it cannot establish cross-version effects.
- Use diagnostic copies compiled from the current wrapper/recorder AST. Insert
  timing scopes around original operations, not replacement implementations.
  Check fixed-clock OTLP bytes, return values, exception identity and mutable
  snapshots against the originals before measuring. Source files and policy
  remain unchanged. Profiler clocks never become application timestamps.
- Record exclusive phase wall times, inclusive context times, raw whole-call
  times and timer/empty-scope calibration. Subtract measured median observer
  costs sample by sample, retain raw/negative/clipped results, and expose the
  unattributed residual. Fractions use sums, not a sum of unrelated quantiles.
  Instrumented timings are estimates; an unchanged uninstrumented control
  measures observer perturbation. Never infer a hard scheduler guarantee.
- Separate cProfile processes from phase timing processes. Profiles retain
  function statistics, not locals or captured payloads.
- Primary fixed-payload conditions: bare function; default capture OFF;
  explicit empty probe allowlist with the recorder's fields allowed; full
  allowlist. The current API has no separate capture-enabled flag: OFF and
  empty both omit values but retain references/metadata. Do not invent a new
  runtime path to distinguish them. Full payload is the unchanged S0a four
  scalar arguments and two returned alternatives.
- Five fresh processes for each primary condition and pacing mode. Fixed
  payload uses 1,800 calls/30 s at 60 Hz or 10,000 tight calls. Include the same
  feature comparison on generated development inputs without interpreting
  capture-disabled cohorts as complete. Full lane-02 end-to-end profiling also
  uses the frozen driver on all 12 cases, five paced and five tight repetitions.
- Tight lane-driver mode substitutes only that module's sleep facade to remove
  pacing; it does not patch global time/sleep, source files, equations, seeds,
  profile or completeness checks. Record overload/incomplete cases honestly;
  do not change the queue or suppress the driver's abort.
- GC-disable and worker-disabled comparisons are experiment-only controls,
  five trials each with full capture, both pacing modes. Queue remains 256.
  Worker-disabled queues fill and drop: stratify nonfull/full observations and
  disclose that this is not an isolated causal GIL test. Natural worker encode
  and HTTP intervals, GC callbacks, caller thread CPU and wall intervals provide
  overlap evidence. They cannot uniquely separate GIL waiting from OS scheduling.
- For lane input feature controls, replay the frozen development generator's
  rows cyclically, preserving each input and checking returned values against
  the plain target. These microbenchmarks do not replace the frozen full-driver
  runs. Capture-disabled and overload samples must not become zero associations.

Every process writes to a fresh directory. Preserve every attempted, incomplete
and completed run, raw timing sample, calibration, counter and profile. Keep
initial instrumented/observer misses; do not tune measurements after results.
The 100/250 us budget remains unchanged, including every miss. GC and worker
controls are not shipped defaults. No Stage 2 begins without GO OPTIMIZE.

### Preserved harness failure before the complete suite

Attempt `stage1-01` at harness commit `60eb69d` stopped after summary generation
failed: GC callbacks appended to a deque while a Python generator iterated it.
Calls had finished and raw timings were saved. The suite parent was stopped;
the in-flight child was allowed to finish. All attempted files and failures are
retained, including bare-function trials. This is a reporting defect, not a
recorder safety failure. The correction snapshots the observation deque once
before serialization/analysis, with a regression test. A fresh complete suite
uses the corrected harness. Attempt-01 source metadata read at report time can
reflect this edit for its in-flight child; its loaded code is the old commit.
Do not combine interrupted-attempt trials with the five-trial complete suite.

## Interpretation limits and reporting clarifications

- Phase scopes add Python calls, allocations, counters and GC work. Calibration
  subtracts a warmed median empty-scope cost, not its entire distribution or a
  hypothetical cold-cache cost. Negative corrections and residuals stay visible;
  clipped projections are secondary. A calibrated phase estimate is not a
  production budget pass. Use separate plain controls for that comparison.
- The diagnostic copies also execute inactive timing scopes in worker-called
  methods. Worker interval hooks are present in plain controls too. Observer
  perturbation can change worker contention and GC; interval overlap is not
  proof that GC, GIL or scheduler behavior caused a spike.
- Windows thread CPU accounting can be quantized more coarsely than one short
  call. Report zero CPU samples and CPU increments exceeding wall intervals.
  Wall-minus-thread-CPU is not a reliable fine-grained scheduler/GIL diagnosis.
- The lane-driver boundary includes external run entry/exit and its existing
  overhead-list append, unlike the automatic-context microbenchmark. This tiny
  driver bookkeeping remains in the residual. Compare these populations by name.
- The fixed microbenchmark keeps previous_frame=1 across fresh trials (same
  four-scalar, two-returned-alternative payload and target). Frame advances as in
  S0a. It is a fixed-payload phase study, not the later paired Stage 3 comparison.
- Every experimental no-worker queue stays at 256. Full/nonfull groups are
  reported separately, but a nonfull pre-call queue does not guarantee that both
  spans fit. Neither overload nor capture-OFF cohorts establish delivery or
  diagnostic-accuracy acceptance. No real personal data or reserved generation.
- Feature-control lane microbenchmarks consume a flattened sequence: 1,800 paced
  calls cover seed 101 matched controls, minority stale use and the first 200
  benign-minority rows. They do not cover all 12 cases at 60 Hz. Tight 10,000-call
  controls cover the entire 9,600-row sequence and repeat its first 400 rows.
  The separate frozen-driver full-capture runs provide the all-case paced
  measurement. Do not claim all-case OFF/empty controls or balanced microcohorts.
- Trials use the committed blocked order, not randomized/interleaved controls.
  The machine is a working Windows desktop, not an isolated scheduler lab.
  Small status reads and the branch push occur during the suite; bulk report
  generation, tests and final builds are deferred until it finishes.

## Candidate changes for review only

1. Precompute a binding plan for the already-supported signatures, with the
   existing binding path as the fallback. Test positional/keyword mixtures,
   defaults, invalid calls, original exception identity and all wire bytes.
   The allowlist itself is already precomputed; that is not a new opportunity.
2. Reuse immutable JSON encoder configuration/static metadata where byte tests
   prove identical ordering, escaping, floats and separators. The five fixed
   payload JSON messages are distinct (one decision, four lineage edges), not
   five redundant encodings of the same object. Do not promise to remove them.
3. Investigate duplicate snapshots only for internally owned captured inputs.
   Keep validation and snapshots for public inputs and returned mutable values.
   This has greater correctness risk, and the saving on four scalars may be tiny.

Moving serialization to the worker is deferred: oversized/invalid payload
handling and admission accounting currently happen before queueing. A proposal
must preserve these semantics and snapshot safety before it is an optimization.
Cheaper timestamps cannot silently change time-derived wire values or evidence
IDs. None of these proposals is implemented or approved by this report.

## Checkpoint A results

Measured with harness `d591045` (Python 3.12.12, native Windows), 154 fresh
processes: 140 five-trial feature/control runs, four separate cProfile runs,
and ten frozen-driver attempts. The complete matrix and individual budget
misses are in [the report](../benchmark/results/wrapper-overhead/report.md).
Raw samples/calibration, profiles, counters, failures and source hashes are
alongside it. The interrupted attempt and preliminary smoke run are separate
directories, never pooled with the five-trial results.

### Total cost without inserted phase scopes

These controls retain boundary/worker observations; they are not an observer-free
hardware floor. Quantiles below are medians of the five trial quantiles, in us.

| Payload | Mode | p50 | p99 | Budget passes | Ack / attempted |
|---|---|---:|---:|---:|---:|
| Fixed S0a | 60 Hz | 468.50 | 912.87 | 0/5 | 18,000 / 18,000 |
| Lane inputs | 60 Hz | 498.85 | 1,315.68 | 0/5 | 18,000 / 18,000 |
| Fixed S0a | tight | 49.90 | 179.40 | 5/5 | 2,048 / 100,000 |
| Lane inputs | tight | 63.10 | 174.70 | 5/5 | 2,240 / 100,000 |

The tight cost passes are lossy, not successful recording gates: 97,952 and
97,760 dropped spans respectively. There are zero failed/unknown spans in
these four groups. Fixed paced trial ranges are p50 382.95–499.85 and p99
834.16–1,662.02; lane paced ranges are 488.55–512.20 / 1,249.73–2,399.23.
Median explicit flush seconds: fixed paced .01565, lane paced .00336,
fixed tight .30859, lane tight .30720. Flush is outside caller timing.

Five full paced driver repetitions complete all 60 cases, 48,000 calls and
96,000 acknowledged spans, with zero failed/dropped/unknown. Analytic checks
and evidence anchors pass. All five tight driver attempts abort on their first
800-call case: 6,408 drops across 8,000 attempted spans; 55 cases are unattempted.
These are incomplete transport cases, not zero associations. Driver phase
estimates miss the budget in all ten attempts. Every snapshot's four-state
identity is exact; no observation buffer overflows.

### Phase evidence and observer cost

All six full-capture phase tables, p50/p99 and fractions are in report.md.
For fixed 60 Hz, corrected exclusive time fractions are JSON 20.4%, value
capture 17.5%, entry/argument access 14.7%, privacy 6.5%, lineage 4.9%, queue
4.7%, context enter 4.7%, exit path 2.7%, counters 1.9%, clocks 1.7%, context
exit 1.1%, and residual 19.3% (rounding applies). Do not sum phase quantiles.

Median empty-scope internal cost is .1 us (trial range .1–.3); whole empty
scope is 1 us (1–2.5). All raw calibration arrays and negative corrections are
retained. Corrected full phase estimates still exceed plain controls: fixed
60 Hz 854.05/1,804.35 versus 468.50/912.87; lane 978.60/2,474.88 versus
498.85/1,315.68. Warm subtraction does not remove cold observer/GC/cache costs.
This is a material measurement limitation, not a production slowdown or speedup.

Raw inclusive context enter/exit p50/p99: fixed paced 57.4/161.9 and
27.9/83.11 us; full paced driver 65.3/185.1 and 30.6/89.8 us. These include
child scopes, so they are diagnostic costs. Plain controls did not retain
separate context samples. See context-and-tail.md for every group and the
separate corrected exclusive context values in report.md.

The three corroborated production cost areas in separate cProfile runs are:

1. JSON: 20.1% of fixed paced wrapper cumulative time; 21.6% for lane paced.
   There are five/seven distinct JSON records per decision respectively.
2. Bounded snapshots/validation: 17.0% / 20.9%, including necessary work.
3. Signature binding plus defaults: 9.5% / 8.4%.

Full function tables and raw pstats are retained. Background network frames
also appear in profiles and are not caller I/O. Tight cProfile reports one
extra wrapper/JSON invocation compared with exact producer counters; retain
that profiler anomaly and use producer counters for accounting. These fractions
are diagnostic proxies, not exact CPU attribution or guaranteed savings.

### Tail findings and process controls

For each plain paced payload, 6/90 calls at/above raw p99 overlap GC, none
overlap encoding; 84/90 have neither overlap. HTTP send overlaps 71/90 fixed
and 82/90 lane tail calls, but asynchronous network overlap is not host waiting.
On the full paced driver, GC/encode union overlaps 141/480 tails; 339 do not.
No unique GC/GIL/scheduler cause is established for the remaining spikes.

Thread CPU is zero in 8,934/9,000 fixed and 8,928/9,000 lane plain paced calls.
Quantization prevents a fine-grained wall-minus-CPU scheduling diagnosis.
Disabling GC does not consistently improve tails: fixed phase p99
1,804→1,862 us, lane 2,475→2,750. No-worker results are mixed and confounded
by full queues; full/nonfull costs and visible unknowns are published. One
Python version cannot establish interpreter-version effects. The measurement
does not prove a scheduler, Defender, power-state or GIL explanation.

### Recommendation and approval boundary

An empirical warmed reference is about 50–63 us p50 / 175–179 us p99, while
dropping almost all spans. This is not a theoretical Python floor or evidence
of lossless 60 Hz performance. A planning floor near 50–65 us for this warmed
payload is plausible; a lower floor or a portable p99 guarantee is unproved.
There is no evidence here that safe small changes alone achieve 100/250 us
in paced use. Keep the budget and all misses; do not claim qualification.

Review candidates in the order above. Binding's optimistic profile proxy is
at most roughly 8–10% of caller time (about 40–50 us of the observed paced
median); real savings must be measured and will retain some binding work.
Encoder reuse addresses only a subset of the 20–28% JSON category. Duplicate
snapshot savings are an unknown subset of 17–21%, likely small on four scalars;
mutable/validation risk is higher. No defensible precise saving is established
for either. Moving serialization or changing clocks has greater semantic risk.

The riskiest assumption is that intrusive phase proportions predict savings in
the ordinary wrapper. They do not yet. A later approved change needs paired
ordinary-wrapper trials, exact byte/snapshot/exception tests and lossless
development replays. No optimization or Stage 2/3 validation has been run.

All 25 protected hashes match the task-start snapshot, including the owner's
existing unpublished result edit. Recorder, probe, target, generator, profile,
driver and engine are unchanged. Local formatting/vet/lint/race/build and all
51 Python tests pass, including cooldown-preserves-queued-batches and duplicate
delivery idempotency separately. Harness CI on Windows/Ubuntu passes at
https://github.com/yichenxiao08/telemetry-recorder/actions/runs/39542433782.
Final publication CI is verified in the checkpoint response.

This establishes synthetic observations on one desktop, not real-world
diagnostic accuracy, cross-codebase cost or a revised budget. STOP at
Checkpoint A; wait for the owner's GO OPTIMIZE.

## Owner decision — 2026-10-21 (America/Toronto)

**NO-GO on optimization.** Close this branch as a measurement record, keeping
every result and miss. The current lossless paced observation misses both the
original 100/250 us budget and the owner-set 330/830 us budget. Optimization is
deferred. One time-boxed binding-plan experiment may be revisited if recording
cost blocks real use; this note does not authorize that experiment.

### Why 205/390 and 468.5/912.9 are not interchangeable

The earlier corrected-duration S0a reachable-60-Hz result is 205.15/390.31 us
(benchmark/results/s0a-probe-overhead-monotonic.md). It passed the owner's
post-hoc 330/830 us limits, while missing 100/250 us. Preserve that historical
pass against its stated budget and method.

The newer plain controls use the recorder fix through eb9225e. They retain
worker encode/send interval hooks, GC callbacks and caller boundary/thread-CPU
observations even though they omit phase scopes. They therefore are not the
old untouched S0a harness. They also retain per-call samples and query queue
depth before calls. The shared target still returns four scalar inputs' result
and two scored alternatives; its source is unchanged. The newer fixed input
keeps previous_frame=1 across trials, whereas S0a used trial numbers 1–5;
service metadata differs. Frame counts/rate and small payload shape remain the
same, but these are not byte-identical whole-workload trials.

The same fixed-S0a harness was already run in the recorder-fix paired study:
before 206.900/403.825, fixed 206.600/404.555 us at reachable 60 Hz
(docs/recorder-capture-loss-fix.md). That is evidence against attributing the
new 468.50/912.87 solely to the recorder fix. Machine state was not held fixed
across the older and newer studies: different times on a working Windows
desktop, scheduling/background activity and unmeasured cache/power/thermal
state can contribute. No controlled measurement isolates those effects or
the interval hooks' contribution. Do not claim a unique explanation.

Use **468.50/912.87 us as the latest conservative current observation under
its documented harness**, and retain 205.15/390.31 (and fixed 206.60/404.56)
as historical, method-specific observations. The observer-free current cost
and the exact size of any recorder regression remain unknown. The earlier
number cannot qualify current use by itself; the newer number cannot prove
that the recorder intrinsically doubled in cost. No additional benchmark was
run for this reconciliation. Both current lossless payload controls miss
330/830 as well as 100/250. Optimization remains deferred.
