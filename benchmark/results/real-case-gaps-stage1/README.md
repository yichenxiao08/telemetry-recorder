# Stage 1 capture reproduction and design evidence

Synthetic, owner-authorized, non-blind reproduction on a target-only copy.
No blind-test report was read and no target source is published here.

- Design branch base: 474f53fb58fbc4df875371b9831a2b8ce506ce29.
- Actual unchanged measured engine: a2c2cee3861c3eb6131415469c86df3b1d81b5d5,
  isolated detached worktree. No engine source edits.
- signalscout.yaml is the copied original, not reconstructed or tuned. SHA-256:
  dcad6b2669f3c36f3c91099505510fa7fb464ac4561400c5131fef923ed96f18.
- One unprobed baseline, one plain capture, one separate instrumented diagnostic:
  80 episodes, seed 20261020, unpaced. Each produces 17 failures / 63 successes.
  target-output-hashes.json confirms all three generated CSVs are byte-identical.
- Before/after hashes for all 26 copied Python files match; zero target edits.
  Detailed target-only hash inventory and original raw outputs remain in the
  local ignored .cache/real-case-gaps, not public source snapshots.
- Plain result: 19 stored run IDs, exit 5, ACK/failed/drop/unknown =
  370/0/1464/0 out of 1834 attempted spans. The owner supplied a historical
  14-run / 1554-drop summary; this reproduction differs and is retained as-is.
- Diagnostic result: 29 run IDs, exit 5, 452/0/1382/0 out of 1834. Instrumentation
  changes scheduling. Never present this as an improvement or combine its counts
  with the plain run. Timer overhead is not calibrated or budget-qualified.

Artifacts:

- plain-summary.json and plain-doctor.json: ordinary runner output and derived
  per-probe accounting. Selected=seen-sampled_out. Per-probe serialized counts
  are exact only after the explicit global consistency check passes.
- profiled-summary.json, profiled-doctor.json and profiled-phases.json: private
  AST-copy observation with separate counts and bounded timing aggregates.
- Config, hashes, and two Python test logs. Absolute checkout prefixes in public
  copies are normalized to <repo>; original JSON is retained locally unchanged.

Tests: design branch full Python suite **53 passed**; measured engine unchanged
full suite **75 passed**, including method/tuple/sampling, lane/pipeline and real
trace anchors. The two new diagnostic tests pass against both source versions:
fixed-record protobuf bytes identical, exception identity, bounded aggregates.
No original test or frozen score/report was edited. The full suites' existing
development-only unit checks are not a reserved-seed evaluation or a rerun of the
12-case lane-02 evaluation. No Stage 2/3 acceptance retest was run.

Local gofmt check, go vet, golangci-lint (0 issues), go test -race ./... and task
build passed on the design branch. Hosted Windows/Ubuntu CI is verified after
publication, by exact commit in the checkpoint response.

Retained setup failures before measurement: an unavailable cached shell prefix
produced 'undefinedtask' before any build ran; explicitly restoring tool paths
fixed it. The first diagnostic unit-test invocation used the repository root
without the adapter import path and failed to import diagnose_burst; running
from the documented adapter directory passed. No assertions were weakened and
neither setup failure is counted as a workload trial.

See docs/real-case-gaps.md. This is **Checkpoint A**, not an implemented burst fix
or evidence that a single revealed case establishes diagnostic accuracy.
