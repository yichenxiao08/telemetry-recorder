# Probe usability record — 2026-10-21

Synthetic fixtures authored for this task only; no tester report, unseen project,
prohibited files/folders or personal content were read. Based directly on annotated
`blind-test-engine` at 072820a, separate from P0 WIP b79d85a.

Tests first: initial class-method tests failed against the tag because method
paths were unsupported. Initial argv unit test mocked `close` and expected status
0, but the new delivery accounting correctly returned 5 for unacknowledged work;
the test was corrected to use the actual local server, retaining the status-0
requirement and actual ACKs. No assertion was weakened to admit data loss.

Full Python suite: 60 tests passed before final publication. New cases cover
instance/class/static methods, no whole receiver capture, selected receiver data,
mutable snapshots, return and exception identity, super-chain suppression,
descriptor/property nonexecution, unsupported async/generator/nested/custom-access
cases, real protobuf wire values, argv isolation/restoration (including argparse
errors), normal exit-0 with zero capture, unsupported preflight, and real-server
`runs list` with six expected runs. Existing accepted frame/lane/pipeline and
trace-anchor tests remain unchanged. Original reports and scoring/trace/cohort
sources are byte-identical to the tag.

gofmt/vet/golangci-lint/race/build pass locally. Exact final Windows/Ubuntu CI and
the annotated engine-2 tag are verified in the task response after CI completes.

## One small paired cost observation (not a qualification)

`paired.json` is the unedited raw result. One trial, 20 warmup pairs plus 300
measured pairs, alternating order, 60 TOTAL calls/sec (30/kind). Four identical
scalar arguments and identical returned alternatives. Caller wall time includes
wrapper/serialization/enqueue, subtracting only the recorded target-body duration.
No profiler/worker-interval hooks, no receiver fields, no optimization and no rerun.

| Kind | p50 us | p99 us |
|---|---:|---:|
| Function | 210.95 | 421.31 |
| Instance method | 219.20 | 403.34 |

Difference of quantiles: +8.25/-17.97 us. Retain the negative p99 difference;
do not interpret quantile differences as a causal estimate. Narrow observations
miss 100/250 and fall below 330/830, but are NOT the specified five-trial budget
measurement. Full added method/receiver-state cost against both budgets remains
unmeasured. No historical result is superseded.

640/640 decisions admitted; 1,280/1,280 spans acknowledged; flush complete;
zero drops/failed/unknown. Mean envelope bytes 725,120/640 = 1,133.0; queue and
payload bounds unchanged. The measured fixture SHA-256 is
59bbe59e19f3677d9979931f1f8d3f0d6dc1b21a4e3f4584a9763b2925d57a57.
`measured-class-environment.py` preserves those exact verified bytes. A later
zero-exit CLI test helper was added to the working fixture; measured functions
and method-wrapper code were not tuned or changed after this observation.

Limitations: import preflight is not a sandbox; cached aliases and introspection
may notice wrappers. Explicit base calls and super calls in the active same-name
override chain are collapsed alike. Method identity/nesting is not data lineage.
Every outer call creates a run, not an inferred episode. Missing or unsupported
capture stays unknown. Synthetic-only; P0 is not part of this branch.
