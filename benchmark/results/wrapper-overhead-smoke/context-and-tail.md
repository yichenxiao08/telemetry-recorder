# Context and tail observations

Context values below are raw inclusive scope timings: they include child timing scopes.
Corrected exclusive context values are in report.md; neither is an unobserved production cost.
Tail means calls at or above each trial's raw p99 (ties retained). Overlap is not causal attribution.
Thread CPU accounting can be too coarse for one call; zero CPU is not proof of idle time.

| Group | Enter p50/p99 us | Exit p50/p99 us | Tail calls | Tail GC / encode / either / send overlap | Zero thread CPU calls |
|---|---:|---:|---:|---:|---:|

## No-worker queue stratification

Nonfull before a call does not ensure two free slots. These are experimental saturation costs, not successful recording costs.

| Trial | Queue group | Calls | Adjusted p50 / p99 us |
|---|---|---:|---:|
