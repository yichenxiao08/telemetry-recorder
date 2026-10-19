# M1 ingest measurements

Generated 2026-10-19T18:39:10Z by `task benchmark-ingest`.

Loopback HTTP protobuf, sequential batches, unique records, ten spans/trace, ambient logs and gauge points; includes decode, grouping and WAL SQLite commit (synchronous=NORMAL). Excludes payload generation, database setup, count verification and shutdown. Fresh temporary on-disk database per signal and seed.

Go go1.27.0; windows/amd64; 16 logical CPUs; Intel64 Family 6 Model 197 Stepping 2, GenuineIntel. Batch size 256, five or more fixed seeds. Rates are records per second; standard deviation is across seeds.

| Signal | Mean /s | Min /s | Max /s | Std dev |
|---|---:|---:|---:|---:|
| span | 9123 | 8706 | 9648 | 349 |
| log | 9635 | 8676 | 10704 | 650 |
| metric | 8915 | 8674 | 9127 | 194 |

Every span seed >= 5,000 spans/sec: **true**. Persisted counts verified after each measurement.
