# Simultaneous combined ingest

Simultaneous loopback protobuf exporters paced at 5000 spans, 5000 logs, 10000 scalar points/sec. One durable-acknowledged request per exporter; fixed seeds 1..5, fresh shared on-disk WAL/NORMAL store each seed, same payload generator and batch size as separate benchmark. Stop offering at the window deadline: unsent records count as capacity misses. All acknowledged counts verified. Rate denominator is max(offered window, final acknowledgment); committed_in_window also records deadline capacity. Setup, verification and shutdown excluded.

| Seed | Seconds incl. drain | Spans/s | Logs/s | Points/s | All targets met |
|---|---:|---:|---:|---:|---|
| 1 | 10.079 | 3378 | 3404 | 3404 | false |
| 2 | 10.106 | 2989 | 3014 | 3065 | false |
| 3 | 10.080 | 2819 | 2819 | 2819 | false |
| 4 | 10.071 | 3457 | 3457 | 3457 | false |
| 5 | 10.023 | 3601 | 3601 | 3678 | false |
