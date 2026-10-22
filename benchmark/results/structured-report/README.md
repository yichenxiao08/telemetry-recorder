# Synthetic structured-report validation — 2026-10-22

Generated evidence for `codex/structured-evidence-report` on main 86a0e15.
No external project/report was read. These authored fixtures check mechanics,
not diagnostic accuracy or causal truth.

- `acceptance.json`: real serving binary, ordinary OTLP, 40 new synthetic runs,
  four minority violations with exactly 2/18 violated/satisfied under each flag;
  90 valid structured claims, fabricated citation invalid, ten resolved HTTP
  native anchors. Includes 295 valid claims over the accepted relation fixture
  with all 40 capture-off decisions unknown, plus report verification over each
  of the 15 frozen lane-01 cases. All citation/support rates are 1 for generated
  canonical reports; these rates measure structural consistency only.
- `lane01-legacy.json`: all 15 full frozen reports and JSON/text CLI outputs
  byte-identical against an independently built main binary, using a database
  copy. The command output was generated under `.cache`, outside frozen results.
- `regression.json`: 30 complete trace text/JSON outputs byte-identical; all 72
  shop structural/context results identical except creation/elapsed measurement
  metadata. Existing frozen result files were unchanged.
- `validation.json`, `lint.txt`, `python-suite.txt`: 185 top-level Go tests across
  31 packages pass with race checking; formatting/vet/lint/build pass; all 119
  existing pinned Python 3.12.12 tests pass natively. Baseline/new binary hashes
  and source revision are recorded. The raw Go JSON test log remains under
  `.cache/structured-report/go-race.jsonl` (not a permanent result dependency).

Reproduce with PowerShell commands in `docs/report-walkthrough.md`. The requests,
fresh stores and complete draft report are retained under the relative `.cache`
paths in `acceptance.json`; the script regenerates them on a clean run. No claims
about subjective quality or real-world diagnostic accuracy are supported.
