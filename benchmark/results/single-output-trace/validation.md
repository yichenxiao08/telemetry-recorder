# Single-output trace validation — 2026-10-21

Synthetic data only. No performance reruns, source-target modifications,
reserved-seed evaluations, scoring changes, migrations or cohort/diff changes.

- `task lint`: formatting check, `go vet ./...`, golangci-lint: 0 issues.
- `go test -race ./...`: all packages passed, including new trace graph,
  store, CLI tests and unchanged cohort/diff suites.
- `go test ./internal/lineage -args '-rapid.checks=1000'`: passed 1,000 graph
  property cases (bounded termination, stored-node membership, privacy/unknown).
- `task build`: passed.
- Pinned Python 3.12.12 full unittest discovery: 52 passed in 9.835 seconds.
  Includes real serving binary/OTLP/read-only CLI and 27 HTTP anchor assertions,
  plus unchanged recorder safety/cooldown/idempotency tests.
- The first two sandboxed Python attempts encountered PermissionError on
  Python-created temporary directories, including existing tests. Relocating
  TMP did not solve the sandbox ACL restriction. The authorized native execution
  passed with the same assertions. Full logs retained; machine-local diagnostic
  paths normalized to `<repo>`/`<temp>` for portability.
- A first optional 1,000-case invocation used an unquoted PowerShell dotted
  argument and failed with `flag provided but not defined: -rapid`. The quoted
  `-args '-rapid.checks=1000'` invocation above passed. No test was weakened.

`report.json` retains the real acceptance outputs and original target hashes;
`stale-trace.txt` retains the full stale/fresh descriptive comparison. Both
returned lanes are left; the version check is not a causal lane-effect test.
The argument producer refs are dangling, explicitly unknown. These are accepted
spike/lane-01 limitations, not fabricated complete lineage.

Hosted Windows/Ubuntu Actions results are reported with the final commit and
run URL in the task response. This file does not claim an unobserved CI pass.
