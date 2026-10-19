# Simultaneous combined ingest

Simultaneous loopback protobuf exporters paced at 5000 spans, 5000 logs, 10000 scalar points/sec. Four bounded exporters (one span, one log, two disjoint alternating metric batch streams) each await durable acknowledgment; metric concurrency matches its doubled offered rate. Fixed seeds 1..5, fresh shared on-disk WAL/NORMAL store each seed, same payload generator and batch size as separate benchmark. Stop offering at the window deadline: unsent records count as capacity misses. All acknowledged counts verified. Rate denominator is max(offered window, final acknowledgment); committed_in_window also records deadline capacity. Setup, verification and shutdown excluded. Store uses unchanged diagnostic settings: idle/sweep 1h, zero padding/retention, queue 64; shutdown finalization excluded.

| Seed | Seconds incl. drain | Spans/s | Logs/s | Points/s | All targets met |
|---|---:|---:|---:|---:|---|
| 1 | 10.086 | 2487 | 2513 | 4975 | false |
| 2 | 10.093 | 2892 | 2892 | 5783 | false |
| 3 | 10.061 | 2443 | 2468 | 4885 | false |
| 4 | 10.036 | 2347 | 2347 | 4643 | false |
| 5 | 10.078 | 2235 | 2261 | 4496 | false |
