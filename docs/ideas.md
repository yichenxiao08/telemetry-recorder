# Ideas parking lot

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
