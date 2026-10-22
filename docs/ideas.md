# Ideas parking lot

## Owner closeout and single-output trace — 2026-10-21

- Optimization NO-GO: retain all wrapper results/misses and both budgets. One
  time-boxed binding-plan experiment may be reconsidered only if cost blocks
  real use; not authorized now. Current observer-free cost remains unknown.
- Design explicit cross-call value identity separately: today's probe argument
  refs are not upstream return identities. Never infer dependence from nesting,
  equal values or timing. A future opaque-ref index could improve ergonomics;
  current bounded run-partition reads need no migration.
- Artifact dereferencing, digest verification, identifying eviction versus
  never-captured data and a viewer trace entry point remain deferred. Existing
  run/event anchors work. No cohort, scoring or causal diagnosis is implied.

- Authentication and multi-tenant isolation (explicit non-goal for current plan).
- Sampling and retention policy configuration UI (beyond current cleanup configuration).
- Broader cross-signal correlation heuristics; current behavior requires explicit context.

## Parked throughput experiments (not part of M4b)

- Defender exclusion experiment: compare the same durable benchmark with and without an explicitly approved, narrowly scoped database-directory exclusion. Windows flush/write dominates the profile, but antivirus overhead has not been established. Do not change host protection as part of this milestone.
- Background checkpointing: measure whether moving checkpoints out of intake reduces tail stalls, preserving WAL/NORMAL, bounded WAL growth, shutdown behavior and acknowledgment after commit.
- Group-commit window: measure a small bounded waiting window versus the current four already-queued jobs, including low-load acknowledgment latency and queue pressure. No window is added now.
- Compact scalar storage: measure typed scalar columns/series metadata against duplicated JSON TEXT while retaining raw-record fidelity and existing-data readability. This requires its own migration and correctness study.

Storage estimate: the measured mixed 1M-record fixture uses about 1.1 KB/record.
Applying that mixed-record estimate to 3,000 scalar points/sec gives roughly
12 GB/hour before retention. This is an estimate, not measured scalar-only
storage. M8 must measure scalar-point storage, series cardinality, retention
and growth per hour rather than assume this extrapolation is accurate.

## Revised thesis parking lot (design only)

- Full-detail recent buffering and promotion on labels, versus declared sampling:
  measure label delays, byte caps, eviction, reference availability and graph
  completeness before choosing defaults. Do not implement in this spike.
- Additional runtime/framework integrations after one measured supported helper.
  Arbitrary-code semantic inference and automatic quality judgments remain non-goals.
- Revision-bound source navigation and richer lineage graph visualization after
  basic logical-ID resolution/invariants; preserve a bounded, simple viewer.
- Resume training/ML breadth only when needed. The lane/frame synthetic fixture
  does not authorize robotics integration or application replay.

- Design-only signalscout record -- command wrapper and doctor summary: reuse
  existing OTLP integrations, report unsupported libraries and missing app signals.
  No wrapper, metrics-browser extension or service-overview expansion in this task.

## Synthetic zero-source-change probe follow-ups (parked)

- Expand supported Python callables only after behavior-preservation tests for
  async, generators, descriptors, module aliases and reload. Report unsupported
  coverage explicitly; do not infer complete data flow.
- Standard logging export and other HTTP/framework integrations need separate
  privacy/safety tests. The spike only bridges requests traces and logging context.
- Diagnose the measured probe cost before production rollout. Do not change the
  100/250 us budget or use fast cooldown/drop paths as full-capture performance.

- Keep annotation execution time separate from review/ingestion time: the real
  S0 replay confirms a trace-owned review log extends run end and context scope.
  Fix only in a separately authorized production slice; do not change today's
  web/ML grouping or relabel successes to access the failing-only diff.
- Add bounded logical-ref resolution and selection/mark UI before claiming a
  low-effort diagnosis experience. Boundary argument edges cannot establish actual
  reads or cross-step object identity; profile assertions and internal alternatives
  may still require optional explicit calls.
- Keep monotonic durations anchored to UTC in future recorders. Measure clock
  uncertainty across runtimes/processes rather than infer sub-tick order from
  coincident wall-clock timestamps. S0 keeps the original failed fidelity test
  and both unchanged-budget cost passes.

## Slice 1 follow-ups (parked)

- Independent real-world cases and author-separated held-out evaluation before
  claiming diagnostic accuracy. lane-01 is a deliberately separable synthetic
  case-control task with pre-known mechanisms; keep decoy-heavy/minority misses.
- Graph-based identity/read validation and clustered uncertainty estimates; scalar
  version declarations are not proof of real data dependence or independence.
- Richer cohort matching, mark UI and annotation retention/deletion policy only
  after separately authorized scope and P0 privacy. Current annotations are
  synthetic-only and append-only; no real-content capture gate is claimed.

## Lane-01 diagnostic follow-ups (design parked, 2026-10-20)

- Review only two candidate additions: common-support conditional comparisons
  and reference-tail coverage. Full formulas, minima, failure modes and an
  unexecuted fresh-seed plan are in problem-model.md Proposed v1. No new engine
  type, scorer, migration or production phase is authorized by this analysis.
- Treat unused label-correlated inputs as observationally nonidentifiable unless
  stronger read/use evidence or a developer-controlled intervention is available.
  Session matching alone cannot remove the within-session synthetic +10 offset.
- Independent sequential histories and author-separated variants remain needed.
  Do not replace lane-01, search seeds, or tune constants to make it successful.
- Potential later uncertainty calibration, cluster-aware resampling, automated
  control selection, conditional tails and interventions require separate review.
  They are not additional proposed v1 features to implement in this task.

## Owner-deferred conditional comparison

Conditional comparison is **deferred, not part of the approved lane-02 work**.
It requires developer-declared pre-output controls, cannot provide an aggregate
on disjoint support, and adds sparse-stratum, selection and presentation
complexity. Do not implement controls, conditional scores, or conditional-comparison
variants. Preserve the prior proposal as history; roles and reference-tail
coverage are the only approved additions, and Task B still awaits contract review.
The within-range subgroup miss stays visible rather than being rescued with
conditional applicability filtering. See benchmark/problems/lane-02/contract.md.

## Lane-02 Stage 1 observations (parked)

- Finite empirical reference quantiles need not contain all nuisance support
  after padding. Keep the benign 7/400 vs 7/400 outside counterexample and exact
  zero score; do not change this frozen fixture or its interval to erase it.
- Unequal-size Laplace contrasts can be positive with zero raw outside counts.
  A future separately reviewed calibration design may address interpretation;
  no smoothing change, significance claim or fused score is authorized here.
- Current development cost measurements are single trials on a new payload and
  retain two historical-budget misses. No recorder/ingestion tuning is part of
  this slice. Any further cost task needs an explicit fixed-payload protocol.
- Independent real histories/author-separated cases are still needed before
  claiming diagnostic accuracy. Conditional comparison remains deferred; no
  M6–M8 or P0–P4 production phase starts. Stage 2 awaits explicit GO.

## Lane-02 Stage 2 observations (parked; no implementation)

- The owner-authorized one-shot run stopped on the first incomplete transport
  case. Keep it and the 29 unattempted cases; do not retry or silently resume.
  Future measurement needs a separately approved protocol, not reuse of the
  reserved evaluation to obtain a passing result.
- Consider bounded exporter diagnostics retaining error class/status and
  occurrence counts without payloads. Current aggregate export_failed and
  cooldown counters cannot identify the initiating failure. This is a proposal,
  not permission to change the frozen recorder or perform new measurements.
- Capture completeness must be explicit before interpreting cohort rankings.
  Ack failures can have unknown persistence; any future reconciliation design
  must distinguish confirmed success, confirmed rejection and unknown outcome.
- Retain wrapper 475.750/2,300.580 us budget misses alongside prior observations.
  No tuning, conditional comparison, migration, new diff type or production
  phase is authorized. Full results: docs/lane02-stage2.md.

## Recorder capture-loss Checkpoint A (parked until GO FIX)

- Proposed, not implemented: one bounded worker retry for transient failures,
  worker-only backoff, retaining accepted items through cooldown and allowing
  nonblocking admission while finite capacity remains. Keep 256 as the default;
  a 4,096 toy burst succeeding is not justification for a larger default.
- Exact producer-span lifecycle accounting must distinguish acknowledgement,
  confirmed rejection, caller drop and unknown persistence; draining is not
  delivery. Verify trace/span/event idempotency before retrying ambiguous results.
- A total retry deadline cannot be inferred from urllib's socket timeout. Review
  DNS/header/slow-operation behavior and bound any auxiliary worker mechanism;
  no unbounded thread-per-timeout approach. These are Stage 3 implementation
  questions, not changes already made.
- Retain the unhelpful controlled CPU/I/O trial, all 60 cost-budget misses, the
  induced post-commit loss, and the relay WinError 10053 traceback. No Defender
  exclusion test, profiling expansion or scoring/throughput tuning is added.
- Future reliability validation needs the owner's Stage 3/4 approval. Do not write
  a lane-03 contract or start later production milestones from this investigation.

## Recorder capture-loss GO FIX follow-up (Stage 3–4)

The Checkpoint A retry/accounting proposal above was approved and implemented
in `a8453cb`. See docs/recorder-capture-loss-fix.md and its saved measurements
for the final validation; the earlier proposal remains historical evidence.

- Proposed only: require ten consecutive clean full development repetitions
  before considering a future reserved evaluation. Exact counts, visible
  unknowns, protected artifacts, evidence links and unchanged score bytes are
  separate gates from the unchanged 100/250 us overhead budget. A miss retains
  the attempt and resets the qualifying streak. No lane-03 contract is written.
- Parked: any durable local spool or acknowledgement reconciliation. A bounded
  RAM queue plus one retry cannot promise delivery across arbitrary outages or
  process exit. Such work needs explicit privacy, disk, recovery and lifecycle
  requirements; this fix does not introduce it.
- Parked: extending duplicate guarantees to log/metric exporters. The current
  acceptance test proves span/event idempotency with stable trace/span IDs only.
- Parked: stronger OS network cancellation. The current single daemon operation
  slot quarantines a stuck call without growing threads; queued data can remain
  unknown until a timeout-bounded flush returns or the process exits.
- Keep cost misses and every paired pre/post trial on record. Do not expand the
  queue, change the payload, drop diagnostics or relax the budget to make a
  future run qualify. Any future host-cost optimization needs its own measured
  proposal. No scoring/engine change or production milestone is implied.
- Final measurements: ten clean development repetitions, but all paced trials
  still miss the fixed cost budget. Retain the strict fresh-byte mismatch due
  to event timestamp identity separately from the successful evidence-validated
  analytic comparison. Do not stabilize IDs or alter clocks/scoring merely to
  make that reporting assumption true. Reviewed results are in
  benchmark/results/recorder-capture-loss-fix-reviewed; the first publication
  remains in benchmark/results/recorder-capture-loss-fix.

## Wrapper cost investigation: approval boundary

- Stage 1 measures only. The diagnostic AST copies are observers, not an
  optimized recorder or an alternative production path. No Stage 2 change is
  approved until the owner replies GO OPTIMIZE after Checkpoint A.
- Rank proposed savings using both phase estimates and separate cProfile/plain
  controls. Warm empty-scope subtraction does not remove cold observer costs;
  do not promise measured phase fractions as achievable savings.
- Preserve mutable-value snapshots, exact wire bytes, default capture OFF and
  the fixed queue/retry/deadline/four-state policy in any later proposal.
- Defer interpreter comparisons, OS scheduling/power experiments and native
  capture mechanisms. This task's single Python 3.12.12 measurement cannot
  establish their causal contribution or justify a new budget.
- Checkpoint A: investigate binding plans first (8–10% optimistic profile
  proxy), then encoder/static-metadata reuse, then owned-input duplicate
  snapshots. These are proposals only, not demonstrated savings. Distinct
  lineage messages cannot simply be removed. Serialization-on-worker remains
  deferred pending validation/admission-accounting and mutable snapshot design.
- Keep lower-overhead external timing/profiling as a future measurement idea:
  current fine scopes materially perturb costs, and thread CPU accounting is
  too coarse to prove GIL or scheduler causes. Do not turn a calibrated phase
  estimate into a production budget claim. Full results: docs/wrapper-overhead.md.

## Real-case gaps proposals only — 2026-10-21

- Prioritize explicit sampled/partial/unknown run capture manifests and exact
  terminal accounting before analysis on burst capture. Bounded worker retries
  cannot make an arbitrary unpaced producer lossless; missing manifests stay unknown.
- A separate bounded item-and-byte buffer experiment may absorb finite bursts
  after GO BUILD. Sampling-before-snapshot already exists; do not reopen wrapper
  optimization or the closed overhead budget task to implement it again.
- Extend selectors with literal-dot/backslash escapes, make invariant declarations
  optional, preserve Subtract and equality predicates, then add only bounded
  numeric predicates/aggregates/expressions with pinned source evidence. No eval,
  query language, new score/diff type, unit changes or feature-name boosts.

  Update 2026-10-22: escaped selectors and optional invariants are implemented;
  a bounded relation grammar supports declared single-decision inequalities and
  arithmetic. Aggregate ranking and general derived numeric features stay parked.
- Durable disk spooling, adaptive/random sampling, automatic threshold search and
  general expression languages remain deferred. Eighty aggregate run observations
  cannot meet the unchanged tail minimums; a larger independent synthetic population
  would need a new owner-approved design, never reused lane-02 reserved seeds.

## Probe usability follow-ups (deferred, 2026-10-21)

- A subprocess import sandbox or application-declared argument-parser hook could
  improve preflight. Current dry import cannot validate main-local argparse or
  prevent module-level side effects. Do not run whole targets as preflight.
- Broader callable support (native/async/generators, descriptors, inherited aliases,
  custom access/metaclasses, reload) needs separate semantics/tests; do not silently
  skip it. Automatic episode grouping and receiver-object provenance remain out
  of scope. Boundary capture does not establish reads or return-to-argument links.
- Measure receiver-field and method cost properly only if authorized later. Keep
  the one paired observation and all historical budget misses; no optimization.

## Deferred after the P0 synthetic gate (2026-10-21)

- Approve consent, access control, deletion and real-data retention policies
  separately before any real content. An allowlist does not establish that a
  developer's numeric/enum field is nonpersonal; arbitrary free-text redaction
  by pattern alone is not a safe capture design.
- A policy for opaque generated provenance identities, typed semantic metadata,
  annotation text and scalar metrics could recover strict-mode utility. The
  current gate conservatively omits those fields and blocks HTTP metadata writes;
  do not infer missing provenance or enable direct metadata writers as a bypass.
- Future P0 qualification needs cross-language policy conformance, wider
  adversarial/covert-encoding tests, key provisioning/rotation and clear store
  policy ownership. No migrations or production rollout are included here.
- Measure added redaction caller cost separately if authorized. Current policy
  cost is unmeasured and wrapper budgets remain missed/unresolved; do not use
  the pre-policy method observation as a privacy-mode budget claim.
- The return-to-argument identity proposal lives on codex/p0-spec-gates, design
  only. Mutable version tracking is unresolved; equal primitives stay unknown
  without a declared link/tag. Do not implement it as part of this gate.

## Parked after synthetic tuple/sampling work (2026-10-21)

- Application outcome field -> run status: define explicit mapping and conflict
  semantics before implementing; an RL done/result flag does not currently alter
  native run status.
- Bulk mark-from-file: define bounded validation, preview and failure accounting
  before implementing batch annotations.
- General behavior expectations need a separate contract from the declared-version
  invariant; do not reinterpret existing invariant/ranking/cohort/diff results.
- Future sampling consumers could make incompleteness more visible. Current
  metadata and documentation establish no evidence for skipped calls; do not
  infer negative claims or implement new trace behavior in this task.
# Parked after declared relations — 2026-10-22

- First-divergence comparison is design only in provenance-spec.md. Stable
  same-seed call identity, contiguous capture and unchanged-version controls must
  pass a small authored-fixture experiment before any implementation.
- Per-run aggregate ranking remains deferred. `--violates` checks all recorded
  decisions but existing comparison features use one pinned violation witness;
  do not present that as an aggregate. Numeric cohort selection can currently be
  expressed only as an explicit declared relation, not an ad hoc predicate DSL.
- Wider rule operators (division, strings, cross-record temporal relations), unit
  conversion/inference, automatic rule discovery and aggregate hypothesis/ranking
  are out of this slice. Keep the grammar closed; collect a concrete need first.
- A paginated immutable record manifest could replace large one-snapshot reads
  later. For now hard caps reject the whole selection; no partial cohort or
  silent truncation. No ingestion/wrapper optimization or overhead claim added.
