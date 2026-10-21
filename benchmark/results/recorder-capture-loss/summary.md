# Recorder capture-loss Checkpoint A measurements

Synthetic; no fix implemented. export_failed means unacknowledged, not confirmed storage loss.

| Run | Completed / incomplete / unattempted | Attempted | Exported | Failed export | Cooldown / full drops | Queue HWM |
|---|---|---:|---:|---:|---|---:|
| default-01 | 12 / 0 / 0 | 19200 | 19200 | 0 | 0 / 0 | 40 |
| default-02 | 12 / 0 / 0 | 19200 | 19200 | 0 | 0 / 0 | 14 |
| default-03 | 12 / 0 / 0 | 19200 | 19200 | 0 | 0 / 0 | 16 |
| default-04 | 12 / 0 / 0 | 19200 | 19200 | 0 | 0 / 0 | 16 |
| default-05 | 12 / 0 / 0 | 19200 | 19200 | 0 | 0 / 0 | 16 |

## Toy controls

| Condition | Attempted | Exported | Failed export | Cooldown / full drops | HWM / capacity | Calls/sec | Wrapper p50 / p99 us | 100/250 pass |
|---|---:|---:|---:|---|---|---:|---|---|
| burst-capacity-256 | 2000 | 288 | 0 | 0 / 1712 | 256 / 256 | 13053.365 | 54.700 / 288.243 | False |
| burst-capacity-4096 | 2000 | 2000 | 0 | 0 / 0 | 1968 / 4096 | 10472.718 | 60.200 / 382.937 | False |
| post-commit-slow-ack | 3600 | 3420 | 60 | 120 / 0 | 58 / 256 | 60.032 | 301.550 / 616.038 | False |
| real-controlled-load | 3600 | 3600 | 0 | 0 / 0 | 8 / 256 | 60.032 | 229.200 / 388.011 | False |
| real-fast | 3600 | 3600 | 0 | 0 / 0 | 12 / 256 | 60.032 | 364.700 / 922.509 | False |
| unreachable | 3600 | 0 | 1240 | 2360 / 0 | 60 / 256 | 60.032 | 290.750 / 982.144 | False |

All per-case costs, including every miss, are in cases.md. Raw gzip archives restore byte-identical original JSON reports and telemetry; SHA-256s are in summary.json. Incidental local paths in listener/terminal logs alone are redacted, with original/published hashes disclosed; originals stay local. Worker source rings are sampled to retain earlier stalls; controls cap retained batches at 4096 (more than their maximum attempted span count). No engine invocation or data generation occurs in publication.
