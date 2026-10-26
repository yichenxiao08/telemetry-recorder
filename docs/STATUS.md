# Status at a glance

Updated 2026-10-26. One page; the other docs are the detailed record.

## What it is

Signal Scout is a local Python investigation aid: a Go server (OTLP intake,
SQLite), a Python probe that
records selected function inputs and outputs, run grouping, cohort comparison,
declared expectations, and a cited report with a deterministic verifier.

## What works (tested)

- OTLP intake, durable single-writer storage, run grouping, viewer, diff.
- Python probe with bounded queue, drop counters and completeness manifests.
- Declared invariants and relations; structured report and verifier; `doctor`.

## What is not shown

- That it diagnoses faster or better than ordinary debugging. Several cases
  failed the source-reading difficulty gate, including run4b, run4c and
  historical-01 (27, 33 and 24 commands respectively). Signal Scout was not
  tested on those rejected cases. Run 2's provisional Hit had generous
  downstream attribution and matched an easy source baseline. See
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

The primary pilot02 is stopped as **INCONCLUSIVE**: one valid case had both
diagnoses correct, but tool attribution Partial and investigation times of
747 seconds with the tool versus 220 ordinarily. A case-02 isolation incident
was reported; its independent audit reply exceeded budget and saved no notes.
Case 02 is unscorable and case 03 was not launched. These are owner-reported
outcomes, not an independent evidence inspection. See
[partial outcome and limitations](pilot02-outcome.md).

Preserve attempts and costs; no case replacement, rerun or automatic restart.
The [frozen protocol](paired-usefulness-pilot.md) is unchanged. Its all-three-valid
requirement was not met, so this is neither an overall product pass nor fail.
Node implementation and further expansion remain paused; a future study requires
a deliberate decision and new registration. No diagnostic advantage is shown.
