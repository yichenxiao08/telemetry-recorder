# Recorder capture-loss fix: measured results

Synthetic development/toy data only; no reserved evaluation rerun.

## Separate acceptance results

- **cooldown preserves queued batches: passed**. Flush: `{"acknowledged": 41, "attempted": 41, "complete": true, "drained": true, "dropped": 0, "failed": 0, "unknown": 0}`.
- **duplicate delivery idempotency: passed**. Flush: `{"acknowledged": 1, "attempted": 1, "complete": true, "drained": true, "dropped": 0, "failed": 0, "unknown": 0}`.
  2 real committed requests; 2 stored signals (one span, one event); identical request SHA-256s.

## Every development repetition

| Run | Case statuses | Attempted | Ack | Failed | Dropped | Unknown | Legacy export_failed | HWM / 256 | Identity / nonnegative |
|---|---|---:|---:|---:|---:|---:|---:|---:|---|
| default-01 | {'completed': 12, 'unattempted': 0} | 19200 | 19200 | 0 | 0 | 0 | 0 | 22 | True / True |
| default-02 | {'completed': 12, 'unattempted': 0} | 19200 | 19200 | 0 | 0 | 0 | 0 | 24 | True / True |
| default-03 | {'completed': 12, 'unattempted': 0} | 19200 | 19200 | 0 | 0 | 0 | 0 | 26 | True / True |
| default-04 | {'completed': 12, 'unattempted': 0} | 19200 | 19200 | 0 | 0 | 0 | 0 | 26 | True / True |
| default-05 | {'completed': 12, 'unattempted': 0} | 19200 | 19200 | 0 | 0 | 0 | 0 | 52 | True / True |
| default-06 | {'completed': 12, 'unattempted': 0} | 19200 | 19200 | 0 | 0 | 0 | 0 | 40 | True / True |
| default-07 | {'completed': 12, 'unattempted': 0} | 19200 | 19200 | 0 | 0 | 0 | 0 | 26 | True / True |
| default-08 | {'completed': 12, 'unattempted': 0} | 19200 | 19200 | 0 | 0 | 0 | 0 | 10 | True / True |
| default-09 | {'completed': 12, 'unattempted': 0} | 19200 | 19200 | 0 | 0 | 0 | 0 | 12 | True / True |
| default-10 | {'completed': 12, 'unattempted': 0} | 19200 | 19200 | 0 | 0 | 0 | 0 | 12 | True / True |

## Paired five-trial cost aggregates

Medians of the five trial quantiles, not pooled call quantiles. Every trial/miss and four-state count is in [overhead.md](overhead.md).

| Condition / mode | Source | Trials | p50 / p99 us | Pass | Context enter p50 / p99 us | Context exit p50 / p99 us | Flush s |
|---|---|---:|---|---:|---|---|---:|
| reachable / 60hz | checkpoint-a | 5 | 206.900 / 403.825 | 0 | 38.300 / 123.441 | 3.700 / 10.100 | 0.000011 |
| reachable / 60hz | fixed | 5 | 206.600 / 404.555 | 0 | 36.800 / 79.174 | 4.000 / 10.800 | 0.000066 |
| reachable / tight | checkpoint-a | 5 | 45.000 / 143.606 | 5 | 2.700 / 8.700 | 1.900 / 5.501 | 0.280996 |
| reachable / tight | fixed | 5 | 44.400 / 142.404 | 5 | 2.700 / 9.100 | 1.800 / 5.400 | 0.307275 |
| unreachable / 60hz | checkpoint-a | 5 | 217.600 / 978.697 | 0 | 47.300 / 210.033 | 7.100 / 22.204 | 0.000030 |
| unreachable / 60hz | fixed | 5 | 274.450 / 815.099 | 0 | 46.300 / 146.008 | 6.300 / 19.601 | 10.010393 |
| unreachable / tight | checkpoint-a | 5 | 44.200 / 131.705 | 5 | 2.700 / 8.200 | 1.800 / 5.400 | 0.068843 |
| unreachable / tight | fixed | 5 | 44.000 / 126.205 | 5 | 2.700 / 7.900 | 1.800 / 5.100 | 10.012215 |
| saturated / 60hz | checkpoint-a | 5 | 348.250 / 835.211 | 0 | 53.700 / 169.879 | 8.800 / 21.009 | 0.014286 |
| saturated / 60hz | fixed | 5 | 339.450 / 836.224 | 0 | 52.150 / 187.012 | 8.600 / 22.202 | 0.016216 |
| saturated / tight | checkpoint-a | 5 | 44.400 / 133.210 | 5 | 2.700 / 8.800 | 1.900 / 5.200 | 0.014383 |
| saturated / tight | fixed | 5 | 44.000 / 137.303 | 5 | 2.600 / 8.501 | 1.800 / 5.200 | 0.020929 |

## Unchanged toy controls

| Control | Attempted | Ack | Failed | Dropped | Unknown | HWM | Retry spans / attempts |
|---|---:|---:|---:|---:|---:|---:|---|
| burst-capacity-256 | 2000 | 288 | 0 | 1712 | 0 | 256 | see raw attempt diagnostics |
| post-commit-slow-ack | 3600 | 3600 | 0 | 0 | 0 | 70 | see raw attempt diagnostics |

All [120 per-case results](cases.md) include scores/count regression status, rate, high-water mark and every cost-budget miss. Raw JSON reports/sidecars/timelines are byte-preserved gzip archives; only incidental local paths in metadata/logs are redacted with both hashes in summary.json. SQLite stores stay local. Fresh replay projection excludes only two cohort creation clocks; the separate existing-store regression includes those clocks and compares original text bytes. See the main document for the proposed gate and limitations.
