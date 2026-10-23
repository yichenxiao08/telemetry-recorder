# Doctor probe checks validation — 2026-10-23

Implementation source: `1b56e494e43fc37d33241db8625205da092f0c55`, branched
from main after PR #18. Synthetic data only; no target executed by doctor.

- Formatting, vet, golangci-lint (zero issues), `go test -race ./...`, and
  `task build` pass. The race log contains 201 top-level tests in 32 test-bearing
  packages. All 119 pinned Python 3.12.12 tests pass without assertion changes.
- Fake-environment cases cover probe found/not found from the adapter, missing
  CWD lookup, nonzero help exit, timeout, excessive output and malformed lookup
  JSON. They assert explicit working directories, isolated/no-bytecode lookup,
  bytecode-disabled help, five-second deadlines and suppressed diagnostic text.
  Existing JSON keys and check IDs remain; three probe IDs are appended under
  schema one. Both working templates include the required --doctor flag and
  selected absolute Python; no example depends on uv being on PATH.
- Test subprocesses exercise the real bounded runner using only the Go test
  binary and test-created temp directories: CWD selection, stderr suppression,
  16 KiB stdout overflow and timeout. They create no files. Full ready-check
  orchestration also preserves directory entries and sentinel contents/times.
- `scripts/lane02legacy` reads a database COPY at
  `.cache/doctor-probe-checks/lane01-copy.db`. All 15 full reports and CLI
  text/JSON outputs remain byte-identical to frozen lane-01 and the independently
  built baseline at `86a0e15`. Its output is
  `.cache/doctor-probe-checks/lane01-legacy.json`; no frozen result is modified.
- One real binary execution from the repository root used a fresh OS temp
  directory outside the repository for text/JSON and a nonexistent database.
  Build metadata identifies `1b56e49`, modified=false. It selected the existing
  adapter's Python 3.12.12 and returned **13 PASS, 3 WARN, 0 FAIL**, exit 0.
  `probe.importable` and `probe.help` pass from the adapter; `probe.cwd` warns
  from the repository root with the working Push-Location command. Other
  warnings are missing PATH uv and untested new-DB creation permissions.
  The database remains absent. PowerShell parsing finds zero syntax errors in
  the printed commands; the example/target commands were not executed.

Local logs remain in `.cache/doctor-probe-checks/`. Help runs the trusted adapter
for argparse help and imports its dependencies; it is not a sandbox for arbitrary
initialization code. Find-spec lookup itself never imports scout_probe. None of
these results demonstrates capture, delivery or successful analysis.
