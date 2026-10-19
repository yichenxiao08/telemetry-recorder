# Simultaneous combined ingest

Simultaneous loopback protobuf exporters paced at 5000 spans, 5000 logs, 10000 scalar points/sec. Four bounded exporters (one span, one log, two disjoint alternating metric batch streams) each await durable acknowledgment; metric concurrency matches its doubled offered rate. Fixed seeds 1..5, fresh shared on-disk WAL/NORMAL store each seed, same payload generator and batch size as separate benchmark. Stop offering at the window deadline: unsent records count as capacity misses. All acknowledged counts verified. Rate denominator is max(offered window, final acknowledgment); committed_in_window also records deadline capacity. Setup, verification and shutdown excluded. Store uses serve defaults: idle 5s, sweep 1s, context 30s before/5s after, retention 7d, queue 64; finalization/cleanup run during intake.

| Seed | Seconds incl. drain | Spans/s | Logs/s | Points/s | All targets met |
|---|---:|---:|---:|---:|---|
| 1 | 30.133 | 1308 | 1385 | 2651 | false |
| 2 | 30.055 | 2317 | 2334 | 4676 | false |
| 3 | 30.110 | 2449 | 2491 | 4948 | false |
| 4 | 30.265 | 2326 | 2343 | 4652 | false |
| 5 | 30.126 | 1615 | 1683 | 3263 | false |
