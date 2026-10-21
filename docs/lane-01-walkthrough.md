# Slice 1: synthetic lane diagnosis (PowerShell)

PR #1 is confirmed merged. Slice 1 starts from main 859487c on
codex/slice1-lane. Pre-registration b537ee1 precedes implementation/evaluation.
This uses no original Subway Surfers data and no real personal content.
P0 production privacy and M5–M8 remain unstarted.

## Reproduce all cases

From the repo root, with Go, Task and uv installed (or the accepted spike's
workspace uv installation under .cache/uv-tool):

```powershell
task lane-01
```

The Go orchestrator uses uv's pinned CPython 3.12.12/dependencies. It prints a
fresh directory containing fresh.db and report.json. All five held-out seeds
and three variants run through the unchanged probe, real OTLP intake, marks,
CLI cohort creation/explanation and HTTP event-anchor checks. No Python client
opens SQLite. Keep the pre-registration/scoring files unchanged; the harness
checks their committed state and rejects uncommitted evaluation/scoring code.

The completed confirmation dataset is .cache/lane-01/eval-002/fresh.db.
Use another output directory for a new run:

```powershell
task lane-01 -- -out .cache/lane-01/my-fresh-run
```

Capture/config changes may need new overhead measurement. Slice 1 did not change
the recorder/probe implementation. The owner-set 330/830 us budget passes all
five EXISTING corrected-duration reachable trials; this is a post-results change,
not a pass against the preserved 100/250 us budget or a new payload cost claim.

## Mark and select explicitly

Start the server in one PowerShell terminal:

```powershell
.\bin\signalscout.exe --db .cache/lane-01/eval-002/fresh.db serve --listen 127.0.0.1:4318 --retention 0
```

In another terminal at the repo root:

```powershell
$report = Get-Content benchmark/results/lane-01.json -Raw | ConvertFrom-Json
$run = ($report.cases[0].manifest | Where-Object value -eq 'problem' | Select-Object -First 1).run_id
$tag = 'manual-' + (Get-Date -Format 'yyyyMMddHHmmssfff')
$dimension = $tag + '-review'
$problem = $tag + '-problem'
$reference = $tag + '-reference'
.\bin\signalscout.exe mark $run --dimension $dimension --value bad --reviewer synthetic-developer
.\bin\signalscout.exe --db .cache/lane-01/eval-002/fresh.db labels list --run $run --from 2026-01-01T00:00:00Z --to 2027-01-01T00:00:00Z --limit 100
.\bin\signalscout.exe cohort create $problem --profile benchmark/problems/lane-01/profile.json --from 2026-10-21T00:00:00Z --to 2026-10-21T00:01:00Z --limit 200 --dimension $dimension --value bad
.\bin\signalscout.exe cohort create $reference --profile benchmark/problems/lane-01/profile.json --from 2026-10-21T00:00:00Z --to 2026-10-21T00:01:00Z --limit 200 --dimension lane-01-1009-base --value reference
.\bin\signalscout.exe --db .cache/lane-01/eval-002/fresh.db explain-cohort $problem $reference
```

The manual problem cohort has ONE reviewed run, so this intentionally reports
insufficient_evidence and descriptive observations. Unmarked frames do not become
good references. The existing benchmark cohorts have 100 reviewed members each:

```powershell
.\bin\signalscout.exe --db .cache/lane-01/eval-002/fresh.db explain-cohort lane-01-1009-base-problem lane-01-1009-base-reference
```

Mark writes always use the serving process's writer queue (`--endpoint` overrides
the default listener); they never open a second operational writer. Label/explain
reads use the read-only pool and may run while serving. Use --created-at for a
later review time, --supersedes or --retracts to append a same-reviewer revision,
and --rubric/--rubric-version for explicit provenance. Revisions never erase rows.
Conflicting reviewers remain explicit and are excluded from label selection.

Captured-scalar predicate selection is also supported, with exactly one of
label dimension/value or predicate equality. This is not SQL or a query language:

```powershell
.\bin\signalscout.exe cohort create ($tag + '-predicate') --profile benchmark/problems/lane-01/profile.json --from 2026-10-21T00:00:00Z --to 2026-10-21T00:01:00Z --limit 200 --predicate inputs.previous_frame --equals 999
```

A selected output/evidence ref may be provided to mark with --output-ref/--evidence;
the serving writer checks ownership. Cohorts pin native decision and annotation
revisions. A later mark/decision never silently changes an existing selection.

## Evidence and interpretation

Open http://127.0.0.1:4318/run?id=RUN_ID with an actual run ID from the report.
Explanation URLs include the native event anchor and resolve directly to its
decision envelope. The fixture uses explicit synthetic October 21 times, so use
explicit viewer time filters if browsing the run list. Event timestamps retain
real measured elapsed time; input values and traces are seeded, while timing
and content-derived event IDs can differ between executions.

The declared-used version checker is baseline-independent: its function only
needs captured declared/current/used values and expected delta. False declaration
is not_applicable; missing capture is unknown. explain-cohort prints those results
beside associations; the reference is used for differences, never the invariant.
The profile's version-age transform and expected delta are declared domain
knowledge, not automatically discovered code dependence.

Read benchmark/results/lane-01.md for every mechanism/symptom/decoy rank and
top-1/3/5 mechanism recall. Decoy-heavy and minority-mechanism misses are retained.
All findings are observations or associations. The symptom ranks first by
construction; useful base ranks are likely inflated and do not establish causal
diagnosis or real-world accuracy. No graphical mark UI or hypothesis generation
is implemented in this slice.
