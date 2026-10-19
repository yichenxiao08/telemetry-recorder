# M1 ingest measurements

Generated 2026-10-19T08:26:55Z by `task benchmark-ingest`.

Loopback HTTP protobuf, sequential batches, unique records, ten spans/trace, ambient logs and gauge points; includes decode, grouping and WAL SQLite commit (synchronous=NORMAL). Excludes payload generation, database setup, count verification and shutdown. Fresh temporary on-disk database per signal and seed.

Go go1.27.0; windows/amd64; 16 logical CPUs; Intel64 Family 6 Model 197 Stepping 2, GenuineIntel. Batch size 256, five or more fixed seeds. Rates are records per second; standard deviation is across seeds.

| Signal | Mean /s | Min /s | Max /s | Std dev |
|---|---:|---:|---:|---:|
| span | 5059 | 3605 | 8874 | 1926 |
| log | 3235 | 1757 | 5061 | 1494 |
| metric | 6078 | 5555 | 6319 | 274 |

Every span seed >= 5,000 spans/sec: **false**. Persisted counts verified after each measurement.
