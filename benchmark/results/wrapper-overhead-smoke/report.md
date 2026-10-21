# Wrapper Stage 1: every trial group

Each cell is the median of trial p50/p99s, followed by the min–max across trials.
Instrumented, calibration-subtracted estimates are separate from plain controls.
Only plain controls measure total cost without inserted phase scopes; both observers still record worker intervals.
cProfile is a separate run and is not a budget validation.

| Payload / mode / observer / capture / factor | Trials | p50 us (range) | p99 us (range) | Passes | Drops / failed / unknown |
|---|---:|---:|---:|---:|---:|
| fixed / tight / phase / full / default | 1 | 59.90 (59.90–59.90) | 973.97 (973.97–973.97) | 0/1 | 19104 / 0 / 0 |

## Exclusive phases: fixed, tight, full, default

Quantiles are medians of trial quantiles. Fractions divide summed exclusive times by summed adjusted call time; they need not add to 100% because residual and negatives are retained.

| Phase | p50 us | p99 us | Fraction |
|---|---:|---:|---:|
| entry_argument_access | 10.80 | 82.80 | 10.5% |
| allowlist_privacy | 3.10 | 65.20 | 7.1% |
| value_capture | 7.90 | 222.92 | 21.8% |
| envelope_json | 23.40 | 167.10 | 22.2% |
| span_lineage | 3.60 | 52.40 | 5.9% |
| clock | 2.10 | 13.70 | 2.0% |
| queue_put | 3.00 | 32.40 | 3.9% |
| exit_path | 1.90 | 24.60 | 2.7% |
| run_context_enter | 2.30 | 29.00 | 3.0% |
| run_context_exit | 0.40 | 12.00 | 1.3% |
| counter_bookkeeping | 3.00 | 18.70 | 2.8% |
| Unattributed residual | -1.10 | 203.40 | 16.9% |

## Individual trials (including every budget miss)

| Trial | Calls | Raw p50 / p99 us | Adjusted p50 / p99 us | Budget | Ack / failed / drops / unknown |
|---|---:|---:|---:|---|---:|
| fixed/tight/phase/full/default/1 | 10000 | 203.90 / 1117.97 | 59.90 / 973.97 | MISS | 896 / 0 / 19104 / 0 |
