# Wrapper Stage 1: every trial group

Each cell is the median of trial p50/p99s, followed by the min–max across trials.
Instrumented, calibration-subtracted estimates are separate from plain controls.
Only plain controls measure total cost without inserted phase scopes; both observers still record worker intervals.
cProfile is a separate run and is not a budget validation.

| Payload / mode / observer / capture / factor | Trials | p50 us (range) | p99 us (range) | Passes | Drops / failed / unknown |
|---|---:|---:|---:|---:|---:|
| fixed / 60hz / cprofile / full / default | 1 | 735.65 (735.65–735.65) | 1746.74 (1746.74–1746.74) | 0/1 | 0 / 0 / 0 |
| fixed / 60hz / phase / bare / default | 5 | 23.20 (14.50–26.60) | 122.21 (35.10–189.05) | 5/5 | 0 / 0 / 0 |
| fixed / 60hz / phase / empty / default | 5 | 721.95 (691.70–746.90) | 2369.79 (1892.91–2709.49) | 0/5 | 0 / 0 / 0 |
| fixed / 60hz / phase / full / default | 5 | 854.05 (820.75–947.90) | 1804.35 (1567.14–2391.35) | 0/5 | 0 / 0 / 0 |
| fixed / 60hz / phase / full / gc-disabled | 5 | 854.60 (740.55–919.70) | 1862.26 (1611.42–2109.53) | 0/5 | 0 / 0 / 0 |
| fixed / 60hz / phase / full / worker-idle | 5 | 974.15 (906.65–1169.70) | 2865.15 (2202.07–3242.73) | 0/5 | 16720 / 0 / 1280 |
| fixed / 60hz / phase / off / default | 5 | 715.50 (694.90–731.10) | 1968.24 (1781.75–2080.67) | 0/5 | 0 / 0 / 0 |
| fixed / 60hz / plain / full / default | 5 | 468.50 (382.95–499.85) | 912.87 (834.16–1662.02) | 0/5 | 0 / 0 / 0 |
| fixed / tight / cprofile / full / default | 1 | 128.80 (128.80–128.80) | 423.05 (423.05–423.05) | 0/1 | 19296 / 0 / 0 |
| fixed / tight / phase / bare / default | 5 | 0.50 (0.50–1.00) | 2.40 (2.00–2.90) | 5/5 | 0 / 0 / 0 |
| fixed / tight / phase / empty / default | 5 | 36.80 (34.50–38.40) | 389.72 (334.72–473.91) | 0/5 | 96608 / 0 / 0 |
| fixed / tight / phase / full / default | 5 | 44.30 (41.90–53.40) | 479.24 (391.91–545.91) | 0/5 | 96320 / 0 / 0 |
| fixed / tight / phase / full / gc-disabled | 5 | 44.60 (41.80–48.60) | 461.60 (435.93–501.01) | 0/5 | 96256 / 0 / 0 |
| fixed / tight / phase / full / worker-idle | 5 | 42.30 (41.50–44.60) | 411.34 (366.70–466.61) | 0/5 | 98720 / 0 / 1280 |
| fixed / tight / phase / off / default | 5 | 37.30 (23.20–53.70) | 367.40 (293.70–444.21) | 0/5 | 96640 / 0 / 0 |
| fixed / tight / plain / full / default | 5 | 49.90 (49.10–51.20) | 179.40 (156.62–208.44) | 5/5 | 97952 / 0 / 0 |
| lane-driver / 60hz / phase / full / default | 5 | 982.40 (888.10–1012.20) | 2224.61 (2033.92–2261.00) | 0/5 | 0 / 0 / 0 |
| lane-driver / tight / phase / full / default | 5 | 61.80 (58.50–63.00) | 628.65 (568.82–871.94) | 0/5 | 6408 / 0 / 0 |
| lane-inputs / 60hz / cprofile / full / default | 1 | 855.00 (855.00–855.00) | 1405.12 (1405.12–1405.12) | 0/1 | 0 / 0 / 0 |
| lane-inputs / 60hz / phase / bare / default | 5 | 20.30 (18.30–22.20) | 79.25 (42.16–135.66) | 5/5 | 0 / 0 / 0 |
| lane-inputs / 60hz / phase / empty / default | 5 | 817.30 (811.60–840.15) | 1964.23 (1802.12–2601.93) | 0/5 | 0 / 0 / 0 |
| lane-inputs / 60hz / phase / full / default | 5 | 978.60 (957.80–1047.45) | 2474.88 (2315.32–2980.54) | 0/5 | 0 / 0 / 0 |
| lane-inputs / 60hz / phase / full / gc-disabled | 5 | 999.30 (954.60–1010.10) | 2749.67 (1726.52–3855.80) | 0/5 | 0 / 0 / 0 |
| lane-inputs / 60hz / phase / full / worker-idle | 5 | 916.30 (875.35–1711.90) | 2092.61 (1442.38–3731.77) | 0/5 | 16720 / 0 / 1280 |
| lane-inputs / 60hz / phase / off / default | 5 | 841.90 (816.90–880.95) | 1999.76 (1651.93–2552.89) | 0/5 | 0 / 0 / 0 |
| lane-inputs / 60hz / plain / full / default | 5 | 498.85 (488.55–512.20) | 1315.68 (1249.73–2399.23) | 0/5 | 0 / 0 / 0 |
| lane-inputs / tight / cprofile / full / default | 1 | 164.20 (164.20–164.20) | 518.65 (518.65–518.65) | 0/1 | 19328 / 0 / 0 |
| lane-inputs / tight / phase / bare / default | 5 | 0.50 (0.50–0.50) | 1.40 (1.20–1.50) | 5/5 | 0 / 0 / 0 |
| lane-inputs / tight / phase / empty / default | 5 | 40.00 (39.60–43.10) | 393.10 (358.71–397.71) | 0/5 | 96448 / 0 / 0 |
| lane-inputs / tight / phase / full / default | 5 | 54.90 (52.80–56.30) | 988.40 (842.55–1140.83) | 0/5 | 95424 / 0 / 0 |
| lane-inputs / tight / phase / full / gc-disabled | 5 | 53.60 (51.00–56.40) | 876.42 (847.05–1092.81) | 0/5 | 95372 / 0 / 0 |
| lane-inputs / tight / phase / full / worker-idle | 5 | 51.60 (40.80–53.40) | 780.71 (659.20–896.10) | 0/5 | 98720 / 0 / 1280 |
| lane-inputs / tight / phase / off / default | 5 | 40.40 (39.20–41.90) | 368.22 (362.10–390.82) | 0/5 | 96320 / 0 / 0 |
| lane-inputs / tight / plain / full / default | 5 | 63.10 (62.30–63.20) | 174.70 (136.20–198.81) | 5/5 | 97760 / 0 / 0 |

## Exclusive phases: fixed, 60hz, full, default

Quantiles are medians of trial quantiles. Fractions divide summed exclusive times by summed adjusted call time; they need not add to 100% because residual and negatives are retained.

| Phase | p50 us | p99 us | Fraction |
|---|---:|---:|---:|
| entry_argument_access | 122.00 | 298.98 | 14.7% |
| allowlist_privacy | 54.50 | 163.72 | 6.5% |
| value_capture | 143.90 | 426.52 | 17.5% |
| envelope_json | 174.70 | 403.42 | 20.4% |
| span_lineage | 41.80 | 120.03 | 4.9% |
| clock | 13.80 | 32.80 | 1.7% |
| queue_put | 34.20 | 177.93 | 4.7% |
| exit_path | 21.90 | 63.41 | 2.7% |
| run_context_enter | 38.40 | 106.03 | 4.7% |
| run_context_exit | 8.70 | 26.80 | 1.1% |
| counter_bookkeeping | 16.10 | 44.11 | 1.9% |
| Unattributed residual | 163.60 | 395.20 | 19.3% |

## Exclusive phases: fixed, tight, full, default

Quantiles are medians of trial quantiles. Fractions divide summed exclusive times by summed adjusted call time; they need not add to 100% because residual and negatives are retained.

| Phase | p50 us | p99 us | Fraction |
|---|---:|---:|---:|
| entry_argument_access | 9.10 | 49.80 | 16.0% |
| allowlist_privacy | 1.90 | 31.50 | 5.5% |
| value_capture | 5.20 | 101.21 | 17.2% |
| envelope_json | 19.90 | 103.60 | 33.9% |
| span_lineage | 2.70 | 27.30 | 5.8% |
| clock | 1.70 | 6.90 | 3.0% |
| queue_put | 2.10 | 17.60 | 4.2% |
| exit_path | 1.30 | 11.40 | 3.0% |
| run_context_enter | 1.50 | 13.40 | 3.3% |
| run_context_exit | 0.30 | 5.60 | 0.9% |
| counter_bookkeeping | 2.50 | 9.30 | 4.6% |
| Unattributed residual | -4.90 | 92.01 | 2.6% |

## Exclusive phases: lane-inputs, 60hz, full, default

Quantiles are medians of trial quantiles. Fractions divide summed exclusive times by summed adjusted call time; they need not add to 100% because residual and negatives are retained.

| Phase | p50 us | p99 us | Fraction |
|---|---:|---:|---:|
| entry_argument_access | 118.50 | 343.81 | 12.6% |
| allowlist_privacy | 70.25 | 219.93 | 7.6% |
| value_capture | 215.20 | 614.03 | 22.6% |
| envelope_json | 202.70 | 556.07 | 20.4% |
| span_lineage | 54.60 | 194.53 | 5.6% |
| clock | 13.20 | 35.70 | 1.3% |
| queue_put | 32.90 | 149.21 | 3.8% |
| exit_path | 20.60 | 61.20 | 2.2% |
| run_context_enter | 36.00 | 100.22 | 3.8% |
| run_context_exit | 9.00 | 20.50 | 1.0% |
| counter_bookkeeping | 17.50 | 62.41 | 1.8% |
| Unattributed residual | 170.75 | 452.10 | 17.3% |

## Exclusive phases: lane-inputs, tight, full, default

Quantiles are medians of trial quantiles. Fractions divide summed exclusive times by summed adjusted call time; they need not add to 100% because residual and negatives are retained.

| Phase | p50 us | p99 us | Fraction |
|---|---:|---:|---:|
| entry_argument_access | 10.10 | 71.60 | 11.4% |
| allowlist_privacy | 1.90 | 72.20 | 6.6% |
| value_capture | 5.10 | 251.60 | 21.3% |
| envelope_json | 27.80 | 175.80 | 30.8% |
| span_lineage | 3.30 | 57.00 | 6.6% |
| clock | 1.70 | 10.40 | 2.0% |
| queue_put | 2.20 | 27.60 | 4.0% |
| exit_path | 1.30 | 18.20 | 2.2% |
| run_context_enter | 1.50 | 21.80 | 2.7% |
| run_context_exit | 0.30 | 9.80 | 0.9% |
| counter_bookkeeping | 3.00 | 18.10 | 3.4% |
| Unattributed residual | -4.80 | 157.10 | 8.0% |

## Exclusive phases: lane-driver, 60hz, full, default

Quantiles are medians of trial quantiles. Fractions divide summed exclusive times by summed adjusted call time; they need not add to 100% because residual and negatives are retained.

| Phase | p50 us | p99 us | Fraction |
|---|---:|---:|---:|
| entry_argument_access | 106.60 | 380.67 | 11.6% |
| allowlist_privacy | 71.40 | 200.61 | 7.2% |
| value_capture | 227.40 | 666.10 | 23.0% |
| envelope_json | 205.60 | 590.31 | 21.3% |
| span_lineage | 55.90 | 182.71 | 5.8% |
| clock | 7.90 | 17.10 | 0.8% |
| queue_put | 35.40 | 135.70 | 3.8% |
| exit_path | 22.50 | 65.60 | 2.4% |
| run_context_enter | 36.90 | 107.11 | 3.9% |
| run_context_exit | 10.00 | 33.20 | 1.0% |
| counter_bookkeeping | 21.30 | 60.00 | 2.2% |
| Unattributed residual | 161.60 | 444.82 | 16.9% |

## Exclusive phases: lane-driver, tight, full, default

Quantiles are medians of trial quantiles. Fractions divide summed exclusive times by summed adjusted call time; they need not add to 100% because residual and negatives are retained.

| Phase | p50 us | p99 us | Fraction |
|---|---:|---:|---:|
| entry_argument_access | 10.30 | 71.84 | 13.3% |
| allowlist_privacy | 2.40 | 48.13 | 5.6% |
| value_capture | 6.00 | 151.62 | 18.1% |
| envelope_json | 30.00 | 129.65 | 35.8% |
| span_lineage | 3.40 | 33.91 | 6.9% |
| clock | 1.00 | 4.21 | 1.2% |
| queue_put | 2.60 | 22.63 | 4.8% |
| exit_path | 1.50 | 13.22 | 2.1% |
| run_context_enter | 1.85 | 13.10 | 2.5% |
| run_context_exit | 0.40 | 6.70 | 0.8% |
| counter_bookkeeping | 3.10 | 11.80 | 4.1% |
| Unattributed residual | -2.40 | 97.50 | 4.6% |

## Individual trials (including every budget miss)

| Trial | Calls | Raw p50 / p99 us | Adjusted p50 / p99 us | Budget | Ack / failed / drops / unknown |
|---|---:|---:|---:|---|---:|
| fixed/60hz/cprofile/full/default/1 | 1800 | 735.65 / 1746.74 | 735.65 / 1746.74 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/bare/default/1 | 1800 | 17.35 / 35.10 | 17.35 / 35.10 | PASS | 0 / 0 / 0 / 0 |
| fixed/60hz/phase/bare/default/2 | 1800 | 23.20 / 189.05 | 23.20 / 189.05 | PASS | 0 / 0 / 0 / 0 |
| fixed/60hz/phase/bare/default/3 | 1800 | 24.30 / 122.21 | 24.30 / 122.21 | PASS | 0 / 0 / 0 / 0 |
| fixed/60hz/phase/bare/default/4 | 1800 | 26.60 / 77.04 | 26.60 / 77.04 | PASS | 0 / 0 / 0 / 0 |
| fixed/60hz/phase/bare/default/5 | 1800 | 14.50 / 141.56 | 14.50 / 141.56 | PASS | 0 / 0 / 0 / 0 |
| fixed/60hz/phase/empty/default/1 | 1800 | 861.15 / 2267.57 | 725.85 / 2132.27 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/empty/default/2 | 1800 | 869.55 / 2517.39 | 721.95 / 2369.79 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/empty/default/3 | 1800 | 844.85 / 2565.61 | 697.25 / 2418.01 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/empty/default/4 | 1800 | 882.20 / 2028.21 | 746.90 / 1892.91 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/empty/default/5 | 1800 | 827.00 / 2844.79 | 691.70 / 2709.49 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/full/default/1 | 1800 | 984.95 / 1827.81 | 840.95 / 1683.81 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/full/default/2 | 1800 | 998.05 / 2399.47 | 854.05 / 2255.47 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/full/default/3 | 1800 | 964.75 / 1711.14 | 820.75 / 1567.14 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/full/default/4 | 1800 | 1091.90 / 2535.35 | 947.90 / 2391.35 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/full/default/5 | 1800 | 1039.90 / 1962.75 | 881.50 / 1804.35 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/full/gc-disabled/1 | 1800 | 1039.90 / 2035.06 | 867.10 / 1862.26 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/full/gc-disabled/2 | 1800 | 1078.10 / 2121.51 | 919.70 / 1963.11 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/full/gc-disabled/3 | 1800 | 898.95 / 1997.55 | 740.55 / 1839.15 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/full/gc-disabled/4 | 1800 | 1005.40 / 2267.93 | 847.00 / 2109.53 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/full/gc-disabled/5 | 1800 | 1027.40 / 1784.22 | 854.60 / 1611.42 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/full/worker-idle/1 | 1800 | 1050.65 / 3104.90 | 906.65 / 2960.90 | MISS | 0 / 0 / 3344 / 256 |
| fixed/60hz/phase/full/worker-idle/2 | 1800 | 1307.75 / 2600.84 | 1163.75 / 2456.84 | MISS | 0 / 0 / 3344 / 256 |
| fixed/60hz/phase/full/worker-idle/3 | 1800 | 1146.95 / 3037.95 | 974.15 / 2865.15 | MISS | 0 / 0 / 3344 / 256 |
| fixed/60hz/phase/full/worker-idle/4 | 1800 | 1092.35 / 2374.88 | 919.55 / 2202.07 | MISS | 0 / 0 / 3344 / 256 |
| fixed/60hz/phase/full/worker-idle/5 | 1800 | 1328.10 / 3401.13 | 1169.70 / 3242.73 | MISS | 0 / 0 / 3344 / 256 |
| fixed/60hz/phase/off/default/1 | 1800 | 838.50 / 2035.32 | 715.50 / 1912.32 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/off/default/2 | 1800 | 817.90 / 1904.75 | 694.90 / 1781.75 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/off/default/3 | 1800 | 854.10 / 2091.24 | 731.10 / 1968.24 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/off/default/4 | 1800 | 860.60 / 2129.89 | 725.30 / 1994.59 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/phase/off/default/5 | 1800 | 848.65 / 2215.97 | 713.35 / 2080.67 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/plain/full/default/1 | 1800 | 499.85 / 1662.02 | 499.85 / 1662.02 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/plain/full/default/2 | 1800 | 468.50 / 895.47 | 468.50 / 895.47 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/plain/full/default/3 | 1800 | 382.95 / 834.16 | 382.95 / 834.16 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/plain/full/default/4 | 1800 | 463.60 / 912.87 | 463.60 / 912.87 | MISS | 3600 / 0 / 0 / 0 |
| fixed/60hz/plain/full/default/5 | 1800 | 491.75 / 1354.78 | 491.75 / 1354.78 | MISS | 3600 / 0 / 0 / 0 |
| fixed/tight/cprofile/full/default/1 | 10000 | 128.80 / 423.05 | 128.80 / 423.05 | MISS | 704 / 0 / 19296 / 0 |
| fixed/tight/phase/bare/default/1 | 10000 | 1.00 / 2.90 | 1.00 / 2.90 | PASS | 0 / 0 / 0 / 0 |
| fixed/tight/phase/bare/default/2 | 10000 | 0.90 / 2.70 | 0.90 / 2.70 | PASS | 0 / 0 / 0 / 0 |
| fixed/tight/phase/bare/default/3 | 10000 | 0.50 / 2.00 | 0.50 / 2.00 | PASS | 0 / 0 / 0 / 0 |
| fixed/tight/phase/bare/default/4 | 10000 | 0.50 / 2.30 | 0.50 / 2.30 | PASS | 0 / 0 / 0 / 0 |
| fixed/tight/phase/bare/default/5 | 10000 | 0.50 / 2.40 | 0.50 / 2.40 | PASS | 0 / 0 / 0 / 0 |
| fixed/tight/phase/empty/default/1 | 10000 | 157.50 / 457.72 | 34.50 / 334.72 | MISS | 608 / 0 / 19392 / 0 |
| fixed/tight/phase/empty/default/2 | 10000 | 161.40 / 512.72 | 38.40 / 389.72 | MISS | 704 / 0 / 19296 / 0 |
| fixed/tight/phase/empty/default/3 | 10000 | 158.30 / 525.93 | 35.30 / 402.93 | MISS | 640 / 0 / 19360 / 0 |
| fixed/tight/phase/empty/default/4 | 10000 | 159.80 / 596.91 | 36.80 / 473.91 | MISS | 736 / 0 / 19264 / 0 |
| fixed/tight/phase/empty/default/5 | 10000 | 159.90 / 478.60 | 36.90 / 355.60 | MISS | 704 / 0 / 19296 / 0 |
| fixed/tight/phase/full/default/1 | 10000 | 188.00 / 623.24 | 44.00 / 479.24 | MISS | 736 / 0 / 19264 / 0 |
| fixed/tight/phase/full/default/2 | 10000 | 193.00 / 674.80 | 49.00 / 530.80 | MISS | 736 / 0 / 19264 / 0 |
| fixed/tight/phase/full/default/3 | 10000 | 185.90 / 598.91 | 41.90 / 454.91 | MISS | 736 / 0 / 19264 / 0 |
| fixed/tight/phase/full/default/4 | 10000 | 188.30 / 689.91 | 44.30 / 545.91 | MISS | 736 / 0 / 19264 / 0 |
| fixed/tight/phase/full/default/5 | 10000 | 197.40 / 535.91 | 53.40 / 391.91 | MISS | 736 / 0 / 19264 / 0 |
| fixed/tight/phase/full/gc-disabled/1 | 10000 | 185.80 / 579.93 | 41.80 / 435.93 | MISS | 736 / 0 / 19264 / 0 |
| fixed/tight/phase/full/gc-disabled/2 | 10000 | 186.10 / 645.01 | 42.10 / 501.01 | MISS | 864 / 0 / 19136 / 0 |
| fixed/tight/phase/full/gc-disabled/3 | 10000 | 191.60 / 597.44 | 47.60 / 453.44 | MISS | 704 / 0 / 19296 / 0 |
| fixed/tight/phase/full/gc-disabled/4 | 10000 | 188.60 / 605.60 | 44.60 / 461.60 | MISS | 704 / 0 / 19296 / 0 |
| fixed/tight/phase/full/gc-disabled/5 | 10000 | 192.60 / 636.01 | 48.60 / 492.01 | MISS | 736 / 0 / 19264 / 0 |
| fixed/tight/phase/full/worker-idle/1 | 10000 | 186.70 / 583.23 | 42.70 / 439.23 | MISS | 0 / 0 / 19744 / 256 |
| fixed/tight/phase/full/worker-idle/2 | 10000 | 185.70 / 555.34 | 41.70 / 411.34 | MISS | 0 / 0 / 19744 / 256 |
| fixed/tight/phase/full/worker-idle/3 | 10000 | 188.60 / 610.61 | 44.60 / 466.61 | MISS | 0 / 0 / 19744 / 256 |
| fixed/tight/phase/full/worker-idle/4 | 10000 | 185.50 / 510.70 | 41.50 / 366.70 | MISS | 0 / 0 / 19744 / 256 |
| fixed/tight/phase/full/worker-idle/5 | 10000 | 186.30 / 516.42 | 42.30 / 372.42 | MISS | 0 / 0 / 19744 / 256 |
| fixed/tight/phase/off/default/1 | 10000 | 158.50 / 429.00 | 23.20 / 293.70 | MISS | 672 / 0 / 19328 / 0 |
| fixed/tight/phase/off/default/2 | 10000 | 157.50 / 424.71 | 34.50 / 301.71 | MISS | 704 / 0 / 19296 / 0 |
| fixed/tight/phase/off/default/3 | 10000 | 160.40 / 490.40 | 37.40 / 367.40 | MISS | 608 / 0 / 19392 / 0 |
| fixed/tight/phase/off/default/4 | 10000 | 160.30 / 567.21 | 37.30 / 444.21 | MISS | 704 / 0 / 19296 / 0 |
| fixed/tight/phase/off/default/5 | 10000 | 176.70 / 533.67 | 53.70 / 410.67 | MISS | 672 / 0 / 19328 / 0 |
| fixed/tight/plain/full/default/1 | 10000 | 49.90 / 187.64 | 49.90 / 187.64 | PASS | 384 / 0 / 19616 / 0 |
| fixed/tight/plain/full/default/2 | 10000 | 51.20 / 179.40 | 51.20 / 179.40 | PASS | 384 / 0 / 19616 / 0 |
| fixed/tight/plain/full/default/3 | 10000 | 49.30 / 208.44 | 49.30 / 208.44 | PASS | 416 / 0 / 19584 / 0 |
| fixed/tight/plain/full/default/4 | 10000 | 49.10 / 175.51 | 49.10 / 175.51 | PASS | 448 / 0 / 19552 / 0 |
| fixed/tight/plain/full/default/5 | 10000 | 49.90 / 156.62 | 49.90 / 156.62 | PASS | 416 / 0 / 19584 / 0 |
| lane-driver/60hz/phase/full/default/1 | 9600 | 1193.20 / 2405.61 | 1012.20 / 2224.61 | MISS | 19200 / 0 / 0 / 0 |
| lane-driver/60hz/phase/full/default/2 | 9600 | 1163.40 / 2440.67 | 982.40 / 2259.67 | MISS | 19200 / 0 / 0 / 0 |
| lane-driver/60hz/phase/full/default/3 | 9600 | 1179.30 / 2442.00 | 998.30 / 2261.00 | MISS | 19200 / 0 / 0 / 0 |
| lane-driver/60hz/phase/full/default/4 | 9600 | 1069.10 / 2214.92 | 888.10 / 2033.92 | MISS | 19200 / 0 / 0 / 0 |
| lane-driver/60hz/phase/full/default/5 | 9600 | 1114.90 / 2334.24 | 933.90 / 2153.24 | MISS | 19200 / 0 / 0 / 0 |
| lane-driver/tight/phase/full/default/1 | 800 | 239.50 / 770.87 | 58.50 / 589.87 | MISS | 312 / 0 / 1288 / 0 |
| lane-driver/tight/phase/full/default/2 | 800 | 242.80 / 905.39 | 61.80 / 724.39 | MISS | 320 / 0 / 1280 / 0 |
| lane-driver/tight/phase/full/default/3 | 800 | 240.35 / 809.65 | 59.35 / 628.65 | MISS | 320 / 0 / 1280 / 0 |
| lane-driver/tight/phase/full/default/4 | 800 | 244.00 / 1052.94 | 63.00 / 871.94 | MISS | 320 / 0 / 1280 / 0 |
| lane-driver/tight/phase/full/default/5 | 800 | 243.90 / 749.82 | 62.90 / 568.82 | MISS | 320 / 0 / 1280 / 0 |
| lane-inputs/60hz/cprofile/full/default/1 | 1800 | 855.00 / 1405.12 | 855.00 / 1405.12 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/bare/default/1 | 1800 | 18.50 / 135.66 | 18.50 / 135.66 | PASS | 0 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/bare/default/2 | 1800 | 18.30 / 42.16 | 18.30 / 42.16 | PASS | 0 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/bare/default/3 | 1800 | 20.30 / 108.94 | 20.30 / 108.94 | PASS | 0 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/bare/default/4 | 1800 | 21.45 / 79.25 | 21.45 / 79.25 | PASS | 0 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/bare/default/5 | 1800 | 22.20 / 46.91 | 22.20 / 46.91 | PASS | 0 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/empty/default/1 | 1800 | 999.65 / 2123.73 | 840.15 / 1964.23 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/empty/default/2 | 1800 | 976.80 / 1975.92 | 817.30 / 1816.42 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/empty/default/3 | 1800 | 991.90 / 1961.62 | 832.40 / 1802.12 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/empty/default/4 | 1800 | 989.45 / 2775.93 | 815.45 / 2601.93 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/empty/default/5 | 1800 | 971.10 / 2431.57 | 811.60 / 2272.07 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/full/default/1 | 1800 | 1184.30 / 2639.22 | 978.60 / 2433.52 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/full/default/2 | 1800 | 1144.80 / 2739.22 | 957.80 / 2552.22 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/full/default/3 | 1800 | 1159.60 / 2502.32 | 972.60 / 2315.32 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/full/default/4 | 1800 | 1201.70 / 3186.24 | 996.00 / 2980.54 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/full/default/5 | 1800 | 1253.15 / 2680.58 | 1047.45 / 2474.88 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/full/gc-disabled/1 | 1800 | 1160.30 / 2955.37 | 954.60 / 2749.67 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/full/gc-disabled/2 | 1800 | 1211.45 / 3760.96 | 1005.75 / 3555.26 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/full/gc-disabled/3 | 1800 | 1186.30 / 1913.52 | 999.30 / 1726.52 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/full/gc-disabled/4 | 1800 | 1143.85 / 2321.33 | 956.85 / 2134.33 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/full/gc-disabled/5 | 1800 | 1215.80 / 4061.50 | 1010.10 / 3855.80 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/full/worker-idle/1 | 1800 | 1917.60 / 3937.47 | 1711.90 / 3731.77 | MISS | 0 / 0 / 3344 / 256 |
| lane-inputs/60hz/phase/full/worker-idle/2 | 1800 | 1564.35 / 3580.72 | 1190.35 / 3206.72 | MISS | 0 / 0 / 3344 / 256 |
| lane-inputs/60hz/phase/full/worker-idle/3 | 1800 | 1096.45 / 1655.16 | 890.75 / 1449.46 | MISS | 0 / 0 / 3344 / 256 |
| lane-inputs/60hz/phase/full/worker-idle/4 | 1800 | 1062.35 / 1629.38 | 875.35 / 1442.38 | MISS | 0 / 0 / 3344 / 256 |
| lane-inputs/60hz/phase/full/worker-idle/5 | 1800 | 1103.30 / 2279.61 | 916.30 / 2092.61 | MISS | 0 / 0 / 3344 / 256 |
| lane-inputs/60hz/phase/off/default/1 | 1800 | 986.90 / 1796.93 | 841.90 / 1651.93 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/off/default/2 | 1800 | 1046.00 / 2726.89 | 872.00 / 2552.89 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/off/default/3 | 1800 | 986.30 / 2143.58 | 826.80 / 1984.08 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/off/default/4 | 1800 | 1054.95 / 2576.01 | 880.95 / 2402.01 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/phase/off/default/5 | 1800 | 976.40 / 2159.26 | 816.90 / 1999.76 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/plain/full/default/1 | 1800 | 488.55 / 1396.95 | 488.55 / 1396.95 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/plain/full/default/2 | 1800 | 512.20 / 2399.23 | 512.20 / 2399.23 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/plain/full/default/3 | 1800 | 498.85 / 1249.73 | 498.85 / 1249.73 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/plain/full/default/4 | 1800 | 505.80 / 1315.68 | 505.80 / 1315.68 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/60hz/plain/full/default/5 | 1800 | 489.40 / 1251.72 | 489.40 / 1251.72 | MISS | 3600 / 0 / 0 / 0 |
| lane-inputs/tight/cprofile/full/default/1 | 10000 | 164.20 / 518.65 | 164.20 / 518.65 | MISS | 672 / 0 / 19328 / 0 |
| lane-inputs/tight/phase/bare/default/1 | 10000 | 0.50 / 1.40 | 0.50 / 1.40 | PASS | 0 / 0 / 0 / 0 |
| lane-inputs/tight/phase/bare/default/2 | 10000 | 0.50 / 1.30 | 0.50 / 1.30 | PASS | 0 / 0 / 0 / 0 |
| lane-inputs/tight/phase/bare/default/3 | 10000 | 0.50 / 1.20 | 0.50 / 1.20 | PASS | 0 / 0 / 0 / 0 |
| lane-inputs/tight/phase/bare/default/4 | 10000 | 0.50 / 1.40 | 0.50 / 1.40 | PASS | 0 / 0 / 0 / 0 |
| lane-inputs/tight/phase/bare/default/5 | 10000 | 0.50 / 1.50 | 0.50 / 1.50 | PASS | 0 / 0 / 0 / 0 |
| lane-inputs/tight/phase/empty/default/1 | 10000 | 185.00 / 538.10 | 40.00 / 393.10 | MISS | 704 / 0 / 19296 / 0 |
| lane-inputs/tight/phase/empty/default/2 | 10000 | 184.90 / 540.11 | 39.90 / 395.11 | MISS | 768 / 0 / 19232 / 0 |
| lane-inputs/tight/phase/empty/default/3 | 10000 | 184.60 / 542.71 | 39.60 / 397.71 | MISS | 736 / 0 / 19264 / 0 |
| lane-inputs/tight/phase/empty/default/4 | 10000 | 187.00 / 503.71 | 42.00 / 358.71 | MISS | 672 / 0 / 19328 / 0 |
| lane-inputs/tight/phase/empty/default/5 | 10000 | 188.10 / 530.32 | 43.10 / 385.32 | MISS | 672 / 0 / 19328 / 0 |
| lane-inputs/tight/phase/full/default/1 | 10000 | 242.00 / 1327.83 | 55.00 / 1140.83 | MISS | 960 / 0 / 19040 / 0 |
| lane-inputs/tight/phase/full/default/2 | 10000 | 239.80 / 1195.53 | 52.80 / 1008.53 | MISS | 992 / 0 / 19008 / 0 |
| lane-inputs/tight/phase/full/default/3 | 10000 | 241.90 / 1091.11 | 54.90 / 904.11 | MISS | 864 / 0 / 19136 / 0 |
| lane-inputs/tight/phase/full/default/4 | 10000 | 240.50 / 1029.55 | 53.50 / 842.55 | MISS | 832 / 0 / 19168 / 0 |
| lane-inputs/tight/phase/full/default/5 | 10000 | 243.30 / 1175.40 | 56.30 / 988.40 | MISS | 928 / 0 / 19072 / 0 |
| lane-inputs/tight/phase/full/gc-disabled/1 | 10000 | 241.50 / 1087.42 | 54.50 / 900.42 | MISS | 992 / 0 / 19008 / 0 |
| lane-inputs/tight/phase/full/gc-disabled/2 | 10000 | 239.80 / 1034.05 | 52.80 / 847.05 | MISS | 896 / 0 / 19104 / 0 |
| lane-inputs/tight/phase/full/gc-disabled/3 | 10000 | 238.00 / 1063.42 | 51.00 / 876.42 | MISS | 928 / 0 / 19072 / 0 |
| lane-inputs/tight/phase/full/gc-disabled/4 | 10000 | 240.60 / 1057.64 | 53.60 / 870.64 | MISS | 896 / 0 / 19104 / 0 |
| lane-inputs/tight/phase/full/gc-disabled/5 | 10000 | 243.40 / 1279.81 | 56.40 / 1092.81 | MISS | 916 / 0 / 19084 / 0 |
| lane-inputs/tight/phase/full/worker-idle/1 | 10000 | 238.60 / 970.02 | 51.60 / 783.02 | MISS | 0 / 0 / 19744 / 256 |
| lane-inputs/tight/phase/full/worker-idle/2 | 10000 | 236.80 / 846.20 | 49.80 / 659.20 | MISS | 0 / 0 / 19744 / 256 |
| lane-inputs/tight/phase/full/worker-idle/3 | 10000 | 239.20 / 967.71 | 52.20 / 780.71 | MISS | 0 / 0 / 19744 / 256 |
| lane-inputs/tight/phase/full/worker-idle/4 | 10000 | 240.40 / 932.21 | 53.40 / 745.21 | MISS | 0 / 0 / 19744 / 256 |
| lane-inputs/tight/phase/full/worker-idle/5 | 10000 | 246.50 / 1101.80 | 40.80 / 896.10 | MISS | 0 / 0 / 19744 / 256 |
| lane-inputs/tight/phase/off/default/1 | 10000 | 186.20 / 512.52 | 41.20 / 367.52 | MISS | 736 / 0 / 19264 / 0 |
| lane-inputs/tight/phase/off/default/2 | 10000 | 184.20 / 507.10 | 39.20 / 362.10 | MISS | 736 / 0 / 19264 / 0 |
| lane-inputs/tight/phase/off/default/3 | 10000 | 186.90 / 535.82 | 41.90 / 390.82 | MISS | 736 / 0 / 19264 / 0 |
| lane-inputs/tight/phase/off/default/4 | 10000 | 185.20 / 515.41 | 40.20 / 370.41 | MISS | 704 / 0 / 19296 / 0 |
| lane-inputs/tight/phase/off/default/5 | 10000 | 185.40 / 513.22 | 40.40 / 368.22 | MISS | 768 / 0 / 19232 / 0 |
| lane-inputs/tight/plain/full/default/1 | 10000 | 62.80 / 136.20 | 62.80 / 136.20 | PASS | 448 / 0 / 19552 / 0 |
| lane-inputs/tight/plain/full/default/2 | 10000 | 63.20 / 174.70 | 63.20 / 174.70 | PASS | 416 / 0 / 19584 / 0 |
| lane-inputs/tight/plain/full/default/3 | 10000 | 63.20 / 189.11 | 63.20 / 189.11 | PASS | 480 / 0 / 19520 / 0 |
| lane-inputs/tight/plain/full/default/4 | 10000 | 63.10 / 198.81 | 63.10 / 198.81 | PASS | 480 / 0 / 19520 / 0 |
| lane-inputs/tight/plain/full/default/5 | 10000 | 62.30 / 168.20 | 62.30 / 168.20 | PASS | 416 / 0 / 19584 / 0 |
