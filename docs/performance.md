# Performance task before M4b

This task changes storage performance only. It does not add explorer pages,
context difference types, or change diff scoring. All rates require committed
SQLite data (WAL, synchronous NORMAL), with one operational writer.

## Reproducible measurements

Run from the repository root in PowerShell. Stop other Signal Scout writers
first. `.cache/performance` contains local database copies and binary profiles;
portable JSON, generated tables, and pprof summaries live in `benchmark/results`.

```powershell
New-Item -ItemType Directory -Force .cache/performance | Out-Null
task benchmark-ingest -- -out benchmark/results/perf-00-ingest.json
task benchmark-ingest -- -cpuprofile .cache/performance/baseline-ingest.cpu -allocprofile .cache/performance/baseline-ingest.alloc -out benchmark/results/perf-00-profiled-ingest.json
task benchmark-ingest -- -combined -count 50000 -out benchmark/results/perf-00-combined.json
go run ./scripts/startupbench -source .cache/shop-demo/run-73122371/signalscout.db -db .cache/performance/seed1-before.db -out benchmark/results/perf-00-startup-native-profiled.json -cpuprofile .cache/performance/startup-before.cpu -allocprofile .cache/performance/startup-before.alloc
go tool pprof -top -nodecount=10 .cache/performance/baseline-ingest.cpu
go tool pprof -top -alloc_space -nodecount=10 .cache/performance/baseline-ingest.alloc
```

The original separate benchmark is unchanged: 20,000 unique records per signal,
256 per request, five seeds (1–5), fresh on-disk database for every trial,
sequential acknowledged requests. Setup, generation, verification and shutdown
are excluded from its throughput clock. Whole-process profiles include those
activities; source-line samples distinguish them. The added flags and Task
argument forwarding are instrumentation, not a workload change. Profiled rates
are retained separately from unprofiled comparison rates.

The new combined measurement uses the same payload generator and batch size.
Four exporters share one store and offer 5,000 spans, 5,000 logs, and 10,000
scalar points per second concurrently for ten seconds, for each seed 1–5. Each
exporter has one outstanding acknowledged request (one span, one log, two
alternating metric batch streams, matching its doubled target); absolute pacing exposes
backlog rather than allocating an unbounded client queue. Stop offering at the
deadline; unsent work is a capacity miss. Report both records committed during the
ten-second window and throughput including the final in-flight drain. Every
acknowledged record must persist. Targets require every intended record for every
signal in every seed to commit within the offered window. Separate rates are
diagnostics, not combined capacity. An incomplete first harness attempt that kept
sending unsent work after the window was aborted after 365 seconds; its failure
record is retained as `perf-00-combined-incomplete.json`, without invented rates.
The first completed combined helper used only one metric exporter. Its reports
are retained, but the final method uses two to avoid imposing equal per-kind
queue shares on a 1:1:2 offered load. The context-only code was remeasured with
this four-exporter method before further storage changes
(`perf-01-context-combined-four-exporters`). Subsequent combined comparisons use
that same method. Every completed/failed trial is now saved immediately, so a
later HTTP error cannot erase earlier results. The 30-second request timeout and
all target assertions are unchanged.

Diagnostic combined comparisons use the unchanged benchmark store options (idle
and sweep one hour, zero padding/retention). The additional final
`-combined -serve-defaults` check uses idle 5s, sweep 1s, context 30s/5s and
retention 7d, so finalization and cleanup actually run during the offered load.
Neither case establishes a production SLO or unlimited-history capacity.

Startup uses the normal intake `store.Open`, not the result-only writer. Copies
refuse an existing destination or a source with a nonempty WAL. Open and Close
are timed separately; database size and SQL row counts are outside those clocks.
The deadline is ten minutes; expiration is a failed, censored measurement.

## Before changing storage: measured findings

The unmodified store's whole-process ingest CPU profile (`perf-00-ingest-*`)
contains 72.21 seconds of samples over 68.95 seconds wall time. Cumulative source
line attribution is:

- Writer: 80.61%; context refresh: 30.61%; per-record signal insert: 30.05%;
  commit: 16.62%. These costs overlap the writer total and must not be added to it.
- SQLite preparation: 10.72%. Windows ReadFile: 20.07%; WriteFile: 6.04%.
  `runtime.cgocall` here includes Windows system calls; SQLite remains pure Go.
- Cumulative sampled allocation: 3,183.76 MiB. Store.write accounts for 29.21%,
  decoder.add for 47.00%; reflection, byte clones and JSON decoding are prominent.
  This is cumulative allocation, not peak resident memory. SQLite's manual
  allocator is not comprehensively represented by Go allocation sampling.

The saved seed-1 copy contains 4,057 signals, 340 runs, and seven resources, yet
188,559 context link rows plus 690,921 provenance rows. The closed file occupies
353,308,672 bytes. Those are measured counts, not an extrapolation to 1M rows.

Its profiled native Open takes 67.6339 seconds. Startup CPU has 66 seconds of
samples: backfill 98.73%; deleting links/cascades 25.27%, rebuilding links and
provenance 67.24%, commit 6.21%. Allocation sampling records only 6.16 MiB;
CPU/storage work, rather than Go heap churn, dominates this startup. Source-line
profiles and both flat/cumulative top tens are retained. The prior unchanged
saved-data race test timed out after 600.923 seconds before HTTP navigation
(`viewer-m2-seed-1-race.txt`); this is a failure, not a startup pass.

## Changes and final measurements

### 1. Context scopes (migration 005)

Persist at most three interval/rule/value rows per qualifying span resource in
each closed run. Resolve evidence and every matching source/rule via compatible
SQL views. Ambient writes do no context SQL. Closed-run changes refresh only that
run's scopes atomically; updated ambient timestamps/resources cannot leave stale
links. A padding checkpoint avoids any history scan on unchanged startup. An
upgrade or padding change rebuilds only run/resource scopes. This follows the
measured context cost and multiplicative row counts, rather than assuming JSON
encoding was the main bottleneck.

Unchanged separate benchmark minimum/median: spans 8,706/8,943, logs 8,676/9,522,
points 8,674/8,958 per second (`perf-01-context-summary`). Point target still
missed. First-upgrade native Open on seed 1: 0.4155 seconds versus profiled
67.6339 before; both produce exactly 188,559 logical links and 690,921 matches.
The original combined attempt completed only one trial (rounded console rates
1,047/1,071/1,095), then a later trial hit the unchanged 30-second HTTP timeout;
its raw console is retained, not presented as five completed trials. After the
scope change all five combined trials complete, but still miss all targets;
`perf-01-context-combined` retains the measured rates.

Migration keeps raw telemetry and snapshots. Existing files retain freed pages
for reuse rather than automatically shrinking; fresh-file growth and freelist
measurements distinguish this from live allocated storage. Existing data needs
no rebuild. As with other embedded migrations, normal startup must finish before
serving readers; if interrupted, retry startup to complete derived scopes.

Further measurements and independently measured changes follow below.

The complete ordered SHA-256 audits match for signals, resources, run snapshots,
logical links and every provenance row (`perf-00-context-audit` versus
`perf-01-context-audit`). Fresh original-code startup under race instrumentation
takes 455.8851 seconds, using the frozen pre-change binary and a new saved-data
copy; it completes within the unchanged ten-minute deadline. First upgrade after
the scope change takes 3.4938 seconds under race instrumentation. The older
600.923-second saved-data test failure remains retained as a separate observation.

### 2. Multi-row signal insertion

The initial CPU profile attributes 30.05% to per-row inserts and 10.72% to SQLite
preparation. Inspection of the pinned driver's `stmt.exec` confirms it prepares
and finalizes SQLite bytecode on every execution, including database/sql prepared
statements. Insert at most 128 rows (1,024 parameters) per SQL statement, within
the same job transaction. The bound limits statement/argument size without
changing the HTTP batch size or queue. A short affected-row count still rejects
ownership/kind conflicts and rolls back the entire job. Tests cover conflicts
before/at/after chunk boundaries and duplicate updates across chunks.

The unchanged separate benchmark now has minimum rates of 20,011 spans, 17,164
logs and 16,893 points/sec. Every separate target is met; this alone does not
establish simultaneous capacity. All raw trials remain in `perf-02-multirow-ingest`.

The follow-up mixed-load CPU profile (`perf-02-combined-*`) shows statement
preparation reduced to 2.63%, with Windows FlushFileBuffers at 30.39%, WriteFile
17.53% and ReadFile 7.45%. Canonical OTLP JSON accounts for 7.24%. Thus the
remaining combined limitation is primarily the durable storage path, not just
decoding. No synchronous setting, WAL checkpoint policy, raw record fidelity or
acknowledgment rule was relaxed.

### 3. Preserve time-range join order in context views (migration 006)

The first 1M-row fixture run completed storage but was interrupted before an
inspection report was saved. The retained database was verified to contain
exactly 1,000,000 signals, then measured read-only. Plans exposed the host and
instance branches starting at ambient signals and scanning scopes inside that
loop. A separate global provenance-count profile failed at its new one-minute
diagnostic deadline: 98.08% of samples were in SQLite's VDBE, with JSON lookup
and repeated comparison work. That count is missing/censored, not zero; this is
not a relaxation of the 500 ms bounded-query target.

Migration 006 fixes only join ordering: scopes, resources, then evidence through
the indexed resource/run/time range. A plan test first failed (one of three
branches used its time range), then passed for all three. On the same 1M-row
file, complete counts now finish: 1,876,896 logical links in 15.80 seconds and
3,484,277 provenance rows in 4.94 seconds, backed by only 51,562 physical scopes.
These global counts are storage diagnostics, not bounded explorer queries.

Before explorer indexes, bounded trace lookup takes up to 2,809.96 ms and
template grouping up to 1,177.22 ms; both miss 500 ms and scan signals. Service
and time filtering already meets the target. SQL, arguments, all five repeat
timings and plans are retained in `perf-02-million-before-indexes` and
`perf-03-million-context-plan`. No timings were substituted for the failures.

### 4. Coalesce already-queued transactions

The measured FlushFileBuffers/WriteFile cost supports sharing a commit across at
most four intake jobs already waiting. There is no timer wait or acknowledgment
before commit. Keep each receipt timestamp and savepoint, retain maintenance/diff
ordering barriers, and publish the open-run cache only after commit. Tests first
cover a conflicting middle job, deferred-FK commit failure/recovery, exact receipt
times, barrier ordering and draining accepted jobs. WAL, synchronous NORMAL and
the 1,000-page autocheckpoint policy have explicit regression tests.

All separate targets pass in this attempt, but combined capacity still misses.
Its first four mixed trials improve over the context-plan attempt; the fifth
trial regresses to roughly 2,001/2,001/3,977 records/sec. That trial is retained
and counts as a failure, with no selective exclusion. Generated minimum/median
tables report the variation; the read-only view-plan change is not claimed to
explain changing fresh-database write speeds.

### 5. Bounded lookup indexes (migration 007)

Add a service-name resource index, a partial span/log trace/time index, and a
partial log resource/time index. Plans first failed the new index assertions,
then used the intended lookups. Keep the required run/context/retention indexes;
do not add a general time index merely to improve a query already under budget.
On the same million records, maxima across five repeats are 25.49 ms for
service/time, 0.541 ms for trace lookup, and 35.60 ms for template grouping.
All meet 500 ms, including the first read on a new SQLite connection. OS caches
are not flushed. Zero readings in the sub-millisecond trace timings reflect
clock resolution, not a claim of zero execution cost.

The complete fixture has 25,000 runs, 11 resources, 51,562 physical scopes,
1,876,896 logical links and 3,484,277 logical rule/source rows. Context scopes
plus their primary-key b-tree occupy 16,056,320 bytes (dbstat); one small settings
page is separate. File-size and ingest costs of the lookup indexes are retained
alongside the pre-index measurements. An inspection-only report's older zero
generation duration means generation was not timed by that invocation; newer
reports omit that field. The existing dataset was retained after the interrupted
initial inspection; no row counts were extrapolated.
