# Synthetic P0 gate validation — 2026-10-21

Base blind-test-engine 072820a; P0 implementation ddf1add; safe pause WIP
b79d85a; integration aa7e6e6 merges tested blind-test-engine-2 cc68e4a locally.
No GitHub PR is merged and neither annotated engine tag is changed.

## Checks

| Check | Observation |
| --- | --- |
| Full Python suite after integration | 66 tests passed; raw portable log python-tests.txt |
| Planted-secret Go intake/storage | 0 leaked matches / 77 stored cells, 6 stored signals, 7 planted needles; 539 cell/needle comparisons |
| Producer before queue | Default OFF, nested capture, resource/name/error removal and unchanged host values/exception identity pass |
| Real producer/intake/log path | Probe output and synthetic nested log traffic persist; no planted strings in run detail; metadata POST forbidden in P0 mode |
| Raw/indexed storage check | Every column of every populated application table; JSON + protobuf + gzip for traces/logs/metrics |
| HMAC/artifact | Field scoped and second-gate stable; credentials/card/SQL refused; exact approved nonpersonal redacted artifact bytes only |
| Unknown envelope versions | Fail closed to capture-off/unknown; invalid policies rejected |
| Native Go validation | gofmt, task lint (format/vet/golangci-lint), go test -race ./..., task build passed |
| Historical tests/results | Existing tests unchanged; no code changes in cohort/diff/lineage/stats/viewer/shop or accepted fixture sources; stored lane/shop results not regenerated |
| Added policy overhead | Unmeasured against both budgets; no optimization or performance claim |

Runs 39554316037 (engine-1), 39562411063 (engine-2) and 39557904171
(separate docs proposal) passed both Windows and Ubuntu. Final P0 revision CI
is recorded by its exact SHA and run URL in the completion report.

## Retained failed observations

1. Initial synthetic planted test: one leaking stored cell in 69 checked cells.
   Cause: OTLP sfixed64 as_int was outside the reflection clear list. After adding
   sfixed kinds the same assertion passed 0/69. This observation was recorded
   during development; no original raw Go failure log was saved.
2. Expanded histogram test: one leaking stored cell in 77 checked cells. Cause:
   repeated scalar histogram arrays were not cleared. Added clear branch;
   unchanged expanded assertion passes 0/77. The raw failure is in the tool
   transcript; this written record preserves the miss.
3. New unknown-wire regression first failed because kept KeyValues retained
   their unknown protobuf bytes. Clear those bytes before filtering; both
   allowlisted scalar and record-envelope attribute cases now pass. Dedicated
   policy-package regression, not included in the 77-cell denominator.
4. Initial full Python run: 56 tests, one error, Windows WinError32 when a build
   replaced a binary concurrently used by a real-server test. Kept complete log
   initial-windows-sharing-failure.txt. Sequential runs pass; tests/assertions
   were not weakened. Latest combined suite is 66, not a claimed 56-test failure
   rerun with fewer tests.

Logs replace absolute checkout/user paths with <repo>/<user>. These finite
synthetic observations do not establish zero real-data leakage. See
docs/p0-capture.md for intentional loss, opt-in compatibility and uncovered
channels. No new reserved-seed evaluation or fixture/ranking tuning occurred.
