# Queued transaction grouping: separate diagnostics

Minimum and median across all seed trials per signal in the listed unchanged task benchmark-ingest invocations. Signals measured separately; these rates are not simultaneous ingest capacity. Target is met only if every trial reaches it.

| Signal | Trials | Minimum /s | Median /s | Target /s | All trials met target |
|---|---:|---:|---:|---:|---|
| span | 5 | 23789 | 26326 | 5000 | true |
| log | 5 | 20496 | 21645 | 5000 | true |
| metric | 5 | 20505 | 20830 | 10000 | true |
