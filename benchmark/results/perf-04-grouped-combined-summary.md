# Queued transaction grouping: simultaneous ingest

Minimum and median per signal across all five seeds of the listed simultaneous combined-ingest reports. Rates include in-flight drain; missing/unsent work is a target miss. All signals in every seed must meet 5000/5000/10000 per second. These are not sums of separate-signal measurements.

| Signal | Trials | Minimum /s | Median /s | Target /s | All trials met target |
|---|---:|---:|---:|---:|---|
| span | 5 | 2001 | 3308 | 5000 | false |
| log | 5 | 2001 | 3308 | 5000 | false |
| metric | 5 | 3977 | 6615 | 10000 | false |
