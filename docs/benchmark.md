# M1 ingest benchmark

Run from PowerShell:

```powershell
task benchmark-ingest
```

The Go helper in `scripts/ingestbench` generates deterministic OTLP protobuf requests for seeds 1 through 5, then sends 20,000 unique spans, 20,000 ambient logs and 20,000 ambient scalar gauge points per seed. Each trial creates a fresh temporary on-disk SQLite database and uses actual loopback HTTP requests in batches of 256. Spans form ten-span traces so grouping and root selection are exercised. Results describe this workload; they are not a guarantee for arbitrary telemetry or accumulated history.

Timing includes client HTTP requests, server decoding, normalization/grouping, bounded queue admission, and successful SQLite transaction acknowledgment (WAL, synchronous=NORMAL). Payload generation, database setup/migrations, persisted-count verification and final shutdown/finalization are excluded. Logs and metric points are ambient, so their rates do not include explicit ML grouping or run-close context joins. Retention and idle closure have separate correctness tests.

Every trial verifies the stored record count through the read-only pool. Failures stop the helper. It writes both JSON and Markdown results; tables are never edited manually. The helper's legacy exit status and `span_acceptance` field check only the original 5,000 spans/sec minimum. Current plan acceptance independently requires every trial to meet 5,000 spans/sec, 5,000 logs/sec and 10,000 scalar points/sec; use the generated minimum/median summary for all three targets. A zero exit status alone does not establish current plan acceptance. There is no sampling, compression or concurrent load in this measurement.

Committed measurements: [JSON](../benchmark/results/ingest.json), [generated table](../benchmark/results/ingest.md). The JSON records the OS, Go version, CPU identifier, logical CPU count, seeds, record count and individual trial durations. Background system load and storage performance can change results. Large-history context join cost remains unmeasured in M1; the later seeded failure benchmark is not implemented here.

## Earlier failed measurements

The complete generated reports `ingest-before-batching`, `ingest-batched`, and `ingest-normalized` are retained in `benchmark/results/` (JSON and Markdown). Each failed the unchanged minimum-across-five-seeds criterion. The first two preceded the final resource storage layout. `ingest-normalized` used the final layout and also had a below-target span trial; the final `ingest.json` run passed. Throughput varied sharply on this laptop, so the final passing result should not be interpreted as a guarantee under all machine conditions. No cause for all of that variation has been established.

Profiling showed substantial cost in SQLite writes/checkpoints. The implementation removes repeated resource inserts and a growing history scan, stores resource content once, and restores it through the reader. Signal counts, raw evidence, WAL synchronous mode, fixture expectations, and the 5,000/sec threshold are unchanged.

The M1 review repeats run the unchanged task three times, preserving m1-review-1/2/3 JSON and Markdown reports. scripts/ingestsummary generates the minimum and median across all 15 seed trials for each signal and compares every-trial minimum against the revised independent targets (5,000 spans/sec, 5,000 logs/sec, 10,000 scalar points/sec). These are separate measurements, not combined concurrent rates. No tuning is performed during the repetitions.

## Unchanged pre-M3 measurement

One additional exact `task benchmark-ingest` invocation on the accepted M2 code
is retained in [JSON](../benchmark/results/m3-preflight-ingest.json) and
[Markdown](../benchmark/results/m3-preflight-ingest.md), without tuning or changing
the benchmark workload. All three separate targets were missed. The generated
[minimum/median summary](../benchmark/results/m3-preflight-ingest-summary.md)
covers this invocation's five seed trials per signal, not the older repetitions.
The older default `ingest` report was preserved rather than overwritten.

```powershell
go run ./scripts/ingestsummary -inputs benchmark/results/m3-preflight-ingest.json -out benchmark/results/m3-preflight-ingest-summary -title 'M3 preflight: one unchanged invocation'
```

## M2 sample acceptance

Successful native Windows/Docker Desktop runs are retained separately as `benchmark/results/shop-demo-seed-1.json` and `shop-demo-seed-2.json`; `shop-demo.json` records the latest generation. Reports include the committed source revision and source dirty status, seed, actual request counts, repository-relative database path, evidence IDs, measured pool wait and host-link provenance. Their `signals` counts are the snapshot at verification, before final Collector shutdown; databases can contain later host points as shutdown drains exports. Local databases are not committed; `.cache/` is gitignored.

The first attempt failed the 45-second finalization check despite collecting 40 passing baselines and 9/300 loaded failures. A scoped context-refresh fix removed the repeated full-history rescan. The second attempt then exposed dropped logs caused by an SDK endpoint-path mismatch; the explicit `/v1/logs` setting and a real three-signal SDK regression test fixed that. Neither acceptance timeout nor evidence predicate was relaxed. Local failed-attempt databases and command logs remain under `.cache/shop-demo` for diagnosis.

M2 uses deliberate seeded cart bursts to validate a known failure mechanism. Its approximately-3% frequency is not an unbiased production failure estimate or the later multi-failure benchmark. Three unchanged M1 review repetitions were committed before these M2 fixes; no updated ingest-target success is claimed.

## M3 diff validation

Constants and redundancy rules were documented and committed in `e3f3511` before
the first demo ranking. They are unchanged for both seeds. The original seed-1
database is reused; seed 2 was generated with `task shop-demo -- -seed 2`, including
the separate Collector host resource. Both demos have 40 pre-change passing runs
and 9/300 failures. No workload or config extraction changes were necessary:
the stored run snapshot already has flat `DB_POOL_SIZE`, and baseline cache
misses exist (16/40 for seed 1 and 20/40 for seed 2).

```powershell
task diff-demo -- -demo benchmark/results/shop-demo-seed-1.json -out benchmark/results/diff-seed-1
task diff-demo -- -demo benchmark/results/shop-demo-seed-2.json -out benchmark/results/diff-seed-2
```

The helper validates every failing run against the 40 pre-change passes and its
40 most recent earlier passes, recording the exclusive cutoff, exact IDs,
baseline pool sizes, rates, complete ranked/related evidence and timing.
The latest failure is the representative used in the README commands; all
earlier failures remain in the reports, including mixed default baselines.

Generated results: [seed 1](../benchmark/results/diff-seed-1.md),
[seed 2](../benchmark/results/diff-seed-2.md), with full JSON beside each table.
All pre-change comparisons meet top-three acceptance, with the cache decoy
fourth. Default mode for the latest failure omits the pool config difference
because all 40 selected passes already use pool=5; retry and latency rank first
and second, followed by the cache decoy. This is expected baseline conditioning,
not evidence that the cache miss caused the failure. Earlier failures still
rank the pool config third against a mixed pool=20/pool=5 default history.

The ranks are identical across seeds; seed 2 is slower (maximum 41.281 ms versus
30.969 ms for seed 1). Each of the 36 comparisons is below one second, including
read connections, transactional selection/evidence loading, extraction/scoring,
result persistence and connection closure. Compilation and process startup are
excluded. These are local database timings for one failure mechanism, not a
general accuracy estimate or the M5 harness. Context logs/metrics are excluded
from extraction; no M4b difference types are implemented.
