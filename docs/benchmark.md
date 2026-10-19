# M1 ingest benchmark

Run from PowerShell:

```powershell
task benchmark-ingest
```

The Go helper in `scripts/ingestbench` generates deterministic OTLP protobuf requests for seeds 1 through 5, then sends 20,000 unique spans, 20,000 ambient logs and 20,000 ambient scalar gauge points per seed. Each trial creates a fresh temporary on-disk SQLite database and uses actual loopback HTTP requests in batches of 256. Spans form ten-span traces so grouping and root selection are exercised. Results describe this workload; they are not a guarantee for arbitrary telemetry or accumulated history.

Timing includes client HTTP requests, server decoding, normalization/grouping, bounded queue admission, and successful SQLite transaction acknowledgment (WAL, synchronous=NORMAL). Payload generation, database setup/migrations, persisted-count verification and final shutdown/finalization are excluded. Logs and metric points are ambient, so their rates do not include explicit ML grouping or run-close context joins. Retention and idle closure have separate correctness tests.

Every trial verifies the stored record count through the read-only pool. Failures stop the helper. It writes both JSON and Markdown results; tables are never edited manually. Acceptance requires the minimum of all five span trials to reach 5,000 spans/sec. There is no sampling, compression or concurrent load in this measurement.

Committed measurements: [JSON](../benchmark/results/ingest.json), [generated table](../benchmark/results/ingest.md). The JSON records the OS, Go version, CPU identifier, logical CPU count, seeds, record count and individual trial durations. Background system load and storage performance can change results. Large-history context join cost remains unmeasured in M1; the later seeded failure benchmark is not implemented here.

## Earlier failed measurements

The complete generated reports `ingest-before-batching`, `ingest-batched`, and `ingest-normalized` are retained in `benchmark/results/` (JSON and Markdown). Each failed the unchanged minimum-across-five-seeds criterion. The first two preceded the final resource storage layout. `ingest-normalized` used the final layout and also had a below-target span trial; the final `ingest.json` run passed. Throughput varied sharply on this laptop, so the final passing result should not be interpreted as a guarantee under all machine conditions. No cause for all of that variation has been established.

Profiling showed substantial cost in SQLite writes/checkpoints. The implementation removes repeated resource inserts and a growing history scan, stores resource content once, and restores it through the reader. Signal counts, raw evidence, WAL synchronous mode, fixture expectations, and the 5,000/sec threshold are unchanged.

The M1 review repeats run the unchanged task three times, preserving m1-review-1/2/3 JSON and Markdown reports. scripts/ingestsummary generates the minimum and median across all 15 seed trials for each signal and compares every-trial minimum against the revised independent targets (5,000 spans/sec, 5,000 logs/sec, 10,000 scalar points/sec). These are separate measurements, not combined concurrent rates. No tuning is performed during the repetitions.

## M2 sample acceptance

The successful native Windows/Docker Desktop run is recorded in `benchmark/results/shop-demo.json`, including the committed source revision and source dirty status, seed, actual request counts, database path, evidence IDs, measured pool wait and host-link provenance. Its `signals` counts are the snapshot at verification, before final Collector shutdown; the database can contain later host points as shutdown drains exports. The local database is not committed.

The first attempt failed the 45-second finalization check despite collecting 40 passing baselines and 9/300 loaded failures. A scoped context-refresh fix removed the repeated full-history rescan. The second attempt then exposed dropped logs caused by an SDK endpoint-path mismatch; the explicit `/v1/logs` setting and a real three-signal SDK regression test fixed that. Neither acceptance timeout nor evidence predicate was relaxed. Local failed-attempt databases and command logs remain under `.cache/shop-demo` for diagnosis.

M2 uses deliberate seeded cart bursts to validate a known failure mechanism. Its approximately-3% frequency is not an unbiased production failure estimate or the later multi-failure benchmark. Three unchanged M1 review repetitions were committed before these M2 fixes; no updated ingest-target success is claimed.
