# Context and tail observations

Context values below are raw inclusive scope timings: they include child timing scopes.
Corrected exclusive context values are in report.md; neither is an unobserved production cost.
Tail means calls at or above each trial's raw p99 (ties retained). Overlap is not causal attribution.
Thread CPU accounting can be too coarse for one call; zero CPU is not proof of idle time.

| Group | Enter p50/p99 us | Exit p50/p99 us | Tail calls | Tail GC / encode / either / send overlap | Zero thread CPU calls |
|---|---:|---:|---:|---:|---:|
| fixed / 60hz / cprofile / full / default | not measured | not measured | 18 | 3 / 0 / 3 / 15 | 1749/1800 |
| fixed / 60hz / phase / bare / default | not measured | not measured | 91 | 0 / 0 / 0 / 0 | 9000/9000 |
| fixed / 60hz / phase / empty / default | 52.10 / 182.32 | 28.30 / 96.01 | 90 | 7 / 0 / 7 / 76 | 8828/9000 |
| fixed / 60hz / phase / full / default | 57.40 / 161.90 | 27.90 / 83.11 | 90 | 8 / 0 / 8 / 70 | 8817/9000 |
| fixed / 60hz / phase / full / gc-disabled | 58.90 / 157.11 | 28.20 / 69.01 | 90 | 0 / 0 / 0 / 82 | 8840/9000 |
| fixed / 60hz / phase / full / worker-idle | 70.25 / 246.58 | 32.00 / 130.92 | 90 | 6 / 0 / 6 / 0 | 8782/9000 |
| fixed / 60hz / phase / off / default | 53.25 / 144.72 | 28.30 / 95.10 | 90 | 8 / 0 / 8 / 75 | 8888/9000 |
| fixed / 60hz / plain / full / default | not measured | not measured | 90 | 6 / 0 / 6 / 71 | 8934/9000 |
| fixed / tight / cprofile / full / default | not measured | not measured | 100 | 8 / 13 / 19 / 86 | 9914/10000 |
| fixed / tight / phase / bare / default | not measured | not measured | 597 | 0 / 0 / 0 / 0 | 49997/50000 |
| fixed / tight / phase / empty / default | 6.90 / 23.20 | 6.90 / 19.60 | 500 | 48 / 63 / 110 / 448 | 49450/50000 |
| fixed / tight / phase / full / default | 7.10 / 23.70 | 6.90 / 19.80 | 500 | 45 / 71 / 115 / 461 | 49330/50000 |
| fixed / tight / phase / full / gc-disabled | 7.00 / 25.80 | 6.80 / 20.80 | 500 | 0 / 76 / 76 / 434 | 49317/50000 |
| fixed / tight / phase / full / worker-idle | 6.90 / 25.60 | 6.90 / 20.80 | 500 | 42 / 0 / 42 / 0 | 49352/50000 |
| fixed / tight / phase / off / default | 6.90 / 21.90 | 6.80 / 17.00 | 502 | 39 / 58 / 96 / 445 | 49425/50000 |
| fixed / tight / plain / full / default | not measured | not measured | 500 | 48 / 15 / 63 / 444 | 49824/50000 |
| lane-driver / 60hz / phase / full / default | 65.30 / 185.10 | 30.60 / 89.80 | 480 | 140 / 1 / 141 / 440 | 46723/48000 |
| lane-driver / tight / phase / full / default | 7.90 / 26.70 | 7.30 / 21.80 | 40 | 10 / 10 / 18 / 31 | 3929/4000 |
| lane-inputs / 60hz / cprofile / full / default | not measured | not measured | 18 | 2 / 0 / 2 / 16 | 1791/1800 |
| lane-inputs / 60hz / phase / bare / default | not measured | not measured | 90 | 0 / 0 / 0 / 0 | 8997/9000 |
| lane-inputs / 60hz / phase / empty / default | 54.50 / 138.40 | 28.90 / 99.53 | 90 | 8 / 0 / 8 / 81 | 8779/9000 |
| lane-inputs / 60hz / phase / full / default | 53.90 / 143.56 | 28.80 / 68.34 | 90 | 7 / 0 / 7 / 78 | 8749/9000 |
| lane-inputs / 60hz / phase / full / gc-disabled | 54.10 / 141.73 | 29.50 / 87.75 | 90 | 0 / 0 / 0 / 84 | 8734/9000 |
| lane-inputs / 60hz / phase / full / worker-idle | 49.00 / 250.41 | 28.20 / 98.61 | 90 | 5 / 0 / 5 / 0 | 8657/9000 |
| lane-inputs / 60hz / phase / off / default | 56.90 / 137.84 | 29.10 / 90.71 | 90 | 9 / 0 / 9 / 84 | 8832/9000 |
| lane-inputs / 60hz / plain / full / default | not measured | not measured | 90 | 6 / 0 / 6 / 82 | 8928/9000 |
| lane-inputs / tight / cprofile / full / default | not measured | not measured | 100 | 6 / 11 / 17 / 82 | 9886/10000 |
| lane-inputs / tight / phase / bare / default | not measured | not measured | 556 | 0 / 0 / 0 / 0 | 49997/50000 |
| lane-inputs / tight / phase / empty / default | 6.80 / 19.90 | 6.80 / 18.30 | 500 | 38 / 68 / 104 / 463 | 49353/50000 |
| lane-inputs / tight / phase / full / default | 7.10 / 34.70 | 6.90 / 30.10 | 500 | 39 / 102 / 141 / 469 | 49012/50000 |
| lane-inputs / tight / phase / full / gc-disabled | 7.10 / 34.30 | 6.90 / 29.40 | 500 | 0 / 103 / 103 / 455 | 49041/50000 |
| lane-inputs / tight / phase / full / worker-idle | 7.00 / 30.90 | 6.90 / 28.30 | 500 | 43 / 0 / 43 / 0 | 49093/50000 |
| lane-inputs / tight / phase / off / default | 6.90 / 19.10 | 6.80 / 17.50 | 501 | 39 / 72 / 110 / 460 | 49362/50000 |
| lane-inputs / tight / plain / full / default | not measured | not measured | 500 | 35 / 26 / 61 / 457 | 49785/50000 |

## No-worker queue stratification

Nonfull before a call does not ensure two free slots. These are experimental saturation costs, not successful recording costs.

| Trial | Queue group | Calls | Adjusted p50 / p99 us |
|---|---|---:|---:|
| fixed-60hz-phase-full-worker-idle-1 | nonfull_before | 128 | 707.75 / 1768.72 |
| fixed-60hz-phase-full-worker-idle-1 | full_before | 1672 | 918.05 / 2984.67 |
| fixed-60hz-phase-full-worker-idle-2 | nonfull_before | 128 | 901.85 / 2447.17 |
| fixed-60hz-phase-full-worker-idle-2 | full_before | 1672 | 1179.10 / 2451.59 |
| fixed-60hz-phase-full-worker-idle-3 | nonfull_before | 128 | 673.15 / 1784.52 |
| fixed-60hz-phase-full-worker-idle-3 | full_before | 1672 | 992.65 / 2890.40 |
| fixed-60hz-phase-full-worker-idle-4 | nonfull_before | 128 | 480.90 / 1246.30 |
| fixed-60hz-phase-full-worker-idle-4 | full_before | 1672 | 938.05 / 2222.61 |
| fixed-60hz-phase-full-worker-idle-5 | nonfull_before | 128 | 1168.75 / 2050.06 |
| fixed-60hz-phase-full-worker-idle-5 | full_before | 1672 | 1170.45 / 3260.06 |
| fixed-tight-phase-full-worker-idle-1 | nonfull_before | 128 | 56.80 / 607.51 |
| fixed-tight-phase-full-worker-idle-1 | full_before | 9872 | 42.60 / 432.03 |
| fixed-tight-phase-full-worker-idle-2 | nonfull_before | 128 | 57.10 / 563.50 |
| fixed-tight-phase-full-worker-idle-2 | full_before | 9872 | 41.60 / 408.83 |
| fixed-tight-phase-full-worker-idle-3 | nonfull_before | 128 | 55.00 / 476.71 |
| fixed-tight-phase-full-worker-idle-3 | full_before | 9872 | 44.50 / 463.43 |
| fixed-tight-phase-full-worker-idle-4 | nonfull_before | 128 | 46.90 / 248.57 |
| fixed-tight-phase-full-worker-idle-4 | full_before | 9872 | 41.50 / 366.79 |
| fixed-tight-phase-full-worker-idle-5 | nonfull_before | 128 | 46.75 / 576.89 |
| fixed-tight-phase-full-worker-idle-5 | full_before | 9872 | 42.30 / 370.81 |
| lane-inputs-60hz-phase-full-worker-idle-1 | nonfull_before | 128 | 1144.45 / 2622.55 |
| lane-inputs-60hz-phase-full-worker-idle-1 | full_before | 1672 | 1757.70 / 3780.59 |
| lane-inputs-60hz-phase-full-worker-idle-2 | nonfull_before | 128 | 1075.45 / 2562.85 |
| lane-inputs-60hz-phase-full-worker-idle-2 | full_before | 1672 | 1193.95 / 3209.86 |
| lane-inputs-60hz-phase-full-worker-idle-3 | nonfull_before | 128 | 307.95 / 1030.75 |
| lane-inputs-60hz-phase-full-worker-idle-3 | full_before | 1672 | 896.60 / 1480.65 |
| lane-inputs-60hz-phase-full-worker-idle-4 | nonfull_before | 128 | 307.65 / 1143.84 |
| lane-inputs-60hz-phase-full-worker-idle-4 | full_before | 1672 | 879.90 / 1467.75 |
| lane-inputs-60hz-phase-full-worker-idle-5 | nonfull_before | 128 | 604.35 / 1347.83 |
| lane-inputs-60hz-phase-full-worker-idle-5 | full_before | 1672 | 939.30 / 2107.62 |
| lane-inputs-tight-phase-full-worker-idle-1 | nonfull_before | 128 | 56.40 / 141.21 |
| lane-inputs-tight-phase-full-worker-idle-1 | full_before | 9872 | 51.60 / 786.40 |
| lane-inputs-tight-phase-full-worker-idle-2 | nonfull_before | 128 | 64.95 / 198.46 |
| lane-inputs-tight-phase-full-worker-idle-2 | full_before | 9872 | 49.70 / 659.75 |
| lane-inputs-tight-phase-full-worker-idle-3 | nonfull_before | 128 | 56.20 / 267.85 |
| lane-inputs-tight-phase-full-worker-idle-3 | full_before | 9872 | 52.10 / 781.85 |
| lane-inputs-tight-phase-full-worker-idle-4 | nonfull_before | 128 | 63.85 / 597.89 |
| lane-inputs-tight-phase-full-worker-idle-4 | full_before | 9872 | 53.20 / 745.43 |
| lane-inputs-tight-phase-full-worker-idle-5 | nonfull_before | 128 | 58.00 / 819.00 |
| lane-inputs-tight-phase-full-worker-idle-5 | full_before | 9872 | 40.70 / 895.03 |
