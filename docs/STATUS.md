# Status at a glance

Updated 2026-10-25. One page; the other docs are the detailed record.

## What it is

Signal Scout is a local recorder and comparison tool for explaining program
behavior: a Go server (OTLP intake, SQLite), a zero-code Python probe that
records selected function inputs and outputs, run grouping, cohort comparison,
declared expectations, and a cited report with a deterministic verifier.

## What works (tested)

- OTLP intake, durable single-writer storage, run grouping, viewer, diff.
- Python probe with bounded queue, drop counters and completeness manifests.
- Declared invariants and relations; structured report and verifier; `doctor`.

## What is not shown

- That it diagnoses faster or better than reading the source. Every real bug
  tried so far was found by a code-reading baseline in 5 to 13 commands, and the
  tool's own evidence was downstream of the cause. See
  [thesis-reframe.md](thesis-reframe.md) and
  [independent-critique-response.md](independent-critique-response.md).
- Production use, real-data privacy (P0 covers synthetic data only), non-Python
  runtimes, or probe cost within the original 100/250 us budget.

## Frozen: never edit

`benchmark/results/**` (lane-01, shop, M4b and later reports) and the
`blind-test-engine*` tags. Scripts and docs cite these paths and hashes.

## Known repo weight

About 1,900 tracked files under `benchmark/results` (17 MB) and about 150 MB of
history. Moving them to a release artifact would break frozen paths, so it needs
its own owner-approved change with hash manifests, not a drive-by cleanup.

## Next

A single pre-registered pilot on a bug where reading is a poor strategy
(rare, statistical, or version-regression behavior), with a kill criterion set
before results. No new features until it reports.
