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
