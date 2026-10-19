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
Three exporters share one store and offer 5,000 spans, 5,000 logs, and 10,000
scalar points per second concurrently for ten seconds, for each seed 1–5. Each
exporter has one outstanding acknowledged request; absolute pacing exposes
backlog rather than allocating an unbounded client queue. Stop offering at the
deadline; unsent work is a capacity miss. Report both records committed during the
ten-second window and throughput including the final in-flight drain. Every
acknowledged record must persist. Targets require every intended record for every
signal in every seed to commit within the offered window. Separate rates are
diagnostics, not combined capacity. An incomplete first harness attempt that kept
sending unsent work after the window was aborted after 365 seconds; its failure
record is retained as `perf-00-combined-incomplete.json`, without invented rates.

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

Pending measured changes and acceptance. Keep each attempt's five-seed results,
including regressions, and do not change constants or workloads between attempts.
