# Wrapper Stage 1: every trial group

Each cell is the median of trial p50/p99s, followed by the min–max across trials.
Instrumented, calibration-subtracted estimates are separate from plain controls.
Only plain controls measure total cost without inserted phase scopes; both observers still record worker intervals.
cProfile is a separate run and is not a budget validation.

| Payload / mode / observer / capture / factor | Trials | p50 us (range) | p99 us (range) | Passes | Drops / failed / unknown |
|---|---:|---:|---:|---:|---:|
| fixed / 60hz / phase / bare / default | 5 | 15.80 (11.60–17.70) | 104.31 (70.43–123.70) | 5/5 | 0 / 0 / 0 |

## Individual trials (including every budget miss)

| Trial | Calls | Raw p50 / p99 us | Adjusted p50 / p99 us | Budget | Ack / failed / drops / unknown |
|---|---:|---:|---:|---|---:|
| fixed/60hz/phase/bare/default/1 | 1800 | 11.60 / 70.43 | 11.60 / 70.43 | PASS | 0 / 0 / 0 / 0 |
| fixed/60hz/phase/bare/default/2 | 1800 | 15.75 / 73.77 | 15.75 / 73.77 | PASS | 0 / 0 / 0 / 0 |
| fixed/60hz/phase/bare/default/3 | 1800 | 17.10 / 123.70 | 17.10 / 123.70 | PASS | 0 / 0 / 0 / 0 |
| fixed/60hz/phase/bare/default/4 | 1800 | 17.70 / 104.31 | 17.70 / 104.31 | PASS | 0 / 0 / 0 / 0 |
| fixed/60hz/phase/bare/default/5 | 1800 | 15.80 / 115.36 | 15.80 / 115.36 | PASS | 0 / 0 / 0 / 0 |
