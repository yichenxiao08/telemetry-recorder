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
