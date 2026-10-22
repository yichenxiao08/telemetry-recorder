# Doctor validation — 2026-10-22

Implementation source: `07f31d613ed967860b37795c69498ee1aead14ae`, based on
main `567c96c0cf0c1ce4b7fbc9bd7ac5233488b3ba6d`. Synthetic data only.

- Formatting, vet, golangci-lint (zero issues), `go test -race ./...` and
  `task build` pass. The JSON race log counts 195 top-level tests across 32
  test-bearing packages, including ten new top-level doctor/CLI tests.
- The native pinned CPython 3.12.12 suite passes all 119 tests. The initial
  sandbox attempt had 20 temp-permission errors. The first native suite and a
  focused recheck each failed the unchanged privacy HTTP-403 test with Windows
  connection reset 10054. A focused check with the prior engine passed, and two
  subsequent full suites with the doctor build passed. These observations do
  not establish the reset's cause; no Python/server assertion or behavior was
  changed. Failed and successful logs remain in `.cache/doctor-checks/`.
- `scripts/lane02legacy` uses `.cache/doctor-checks/lane01-copy.db`, a copy of
  the previously validated local lane-01 database, and writes outside frozen
  results. All 15 full reports and text/JSON outputs remain byte-identical to
  frozen lane-01 and the independently built baseline at `86a0e15`. Final report:
  `.cache/doctor-checks/lane01-legacy-final.json`. Frozen artifacts are untouched.
- The real binary was executed once with a nonexistent DB in a new OS temp
  directory outside the repository. Its text and schema-one JSON are retained
  there. Exit status 0; **11 PASS, 2 WARN, 0 FAIL**; the DB was not created.
  Binary build metadata identifies `07f31d6`, modified=false. PATH `py -3`
  and `python` report 3.14.7; uv is not on PATH. Selection correctly uses the
  existing adapter virtual environment's 3.12.12. All four dependency imports
  pass. Warnings are missing PATH uv and untested new-DB creation permissions.
  Port 127.0.0.1:4318 binds and is released. The printed PowerShell example
  parses without syntax errors; it was not executed.

The no-write test checks both directory entries and sentinel file contents/time.
Other fake-environment cases cover lookup/version failures, selection priority,
each missing dependency and its exact pinned repair, missing adapter, database
access failures, port availability, schema keys and exit behavior. JSON output
cannot overwrite files or create the DB path. Output bounds and suppressed
subprocess/error detail are also checked.

Readiness is limited to these observations. Importable dependencies may still be
incompatible, installed code is trusted during import, a port may be claimed
later, and parent existence does not establish ACL/disk/WAL/SQLite readiness.
No server, application, recording, cohort or analysis was started by doctor.
