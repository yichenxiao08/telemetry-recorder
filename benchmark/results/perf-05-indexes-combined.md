# Simultaneous combined ingest

Simultaneous loopback protobuf exporters paced at 5000 spans, 5000 logs, 10000 scalar points/sec. Four bounded exporters (one span, one log, two disjoint alternating metric batch streams) each await durable acknowledgment; metric concurrency matches its doubled offered rate. Fixed seeds 1..5, fresh shared on-disk WAL/NORMAL store each seed, same payload generator and batch size as separate benchmark. Stop offering at the window deadline: unsent records count as capacity misses. All acknowledged counts verified. Rate denominator is max(offered window, final acknowledgment); committed_in_window also records deadline capacity. Setup, verification and shutdown excluded. Store uses unchanged diagnostic settings: idle/sweep 1h, zero padding/retention, queue 64; shutdown finalization excluded.

| Seed | Seconds incl. drain | Spans/s | Logs/s | Points/s | All targets met |
|---|---:|---:|---:|---:|---|
| 1 | 10.085 | 3452 | 3452 | 6904 | false |
| 2 | 10.072 | 3228 | 3228 | 6431 | false |
| 3 | 10.140 | 2777 | 2777 | 5554 | false |
| 4 | 10.046 | 3236 | 3211 | 6421 | false |
| 5 | 10.046 | 3262 | 3262 | 6523 | false |
