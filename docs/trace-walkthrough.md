# Single-output trace — 2026-10-21

Synthetic observations only. Branch: `codex/single-output-trace`, based on
wrapper closeout `474f53f`. The measurement record is open as
[PR #6](https://github.com/yichenxiao08/telemetry-recorder/pull/6) into
`codex/recorder-capture-loss`; it is not merged. Optimization is deferred.

## PowerShell

Go/Task and the pinned uv spike environment are required. Use a fresh output
directory. The harness uses the unchanged accepted lane-01 target and fixture B,
ordinary OTLP, the real binary, read-only CLI and HTTP viewer checks.

```powershell
task build
$out = Join-Path (Get-Location) '.cache/trace-walkthrough'
uv run --directory adapters/provenance-python --frozen python trace_walkthrough.py --out $out
$db = Join-Path $out 'fresh.db'
$bounds = @('--from', '2026-01-01T00:00:00Z', '--to', '2027-01-01T00:00:00Z', '--limit', '200')
$stale = 'value:abababababababababababababababab/0000000000000002.return@1'
$fresh = 'value:acacacacacacacacacacacacacacacac/0000000000000002.return@1'
$profile = 'benchmark/problems/lane-01/profile.json'
& .\bin\signalscout.exe --db $db trace $stale @bounds --profile $profile
& .\bin\signalscout.exe --db $db trace $stale @bounds --profile $profile --compare-to $fresh
& .\bin\signalscout.exe --db $db trace $stale @bounds --profile $profile --json
& .\bin\signalscout.exe --db $db trace abababababababababababababababab lane_target.decide @bounds
& .\bin\signalscout.exe --db $db serve --listen 127.0.0.1:4318 --retention 0
```

Prefix printed evidence URLs with `http://127.0.0.1:4318` to highlight their
event anchors. Native evidence IDs from `report.json` also work as trace args.
Ctrl+C stops serving. Choose appropriate time bounds for other stores.

## Contract and limitations

- Required inclusive `--from`, exclusive `--to`, `--limit` 1–10,000. The shared
  read budget counts all inspected event/log rows, plus one sentinel. At most
  two native primary-key lookups resolve run partitions. Run-qualified backward
  refs discover at most 128 partitions; no history scan. Opaque refs need `--run`.
  Run + output name accepts a run-local ref suffix, explicit name or unique
  step identity. Repeated calls need an exact ref; ambiguity remains unknown.
- Existing `signals_run`/primary-key indexes, EXPLAIN tests, one read-only
  transaction, 10-second deadline; no writer or migration. Depth defaults to
  16/root=0, nodes to 128, maxima 64/1,000. Comparison shares the node budget.
- Every node reports capture status. Only allowed inline values appear; off,
  hash-only and missing detail stays unknown. Captured JSON null is distinct.
  Step/version, output, chosen option, producer alternatives/scores, rules,
  config refs and native evidence URLs appear when recorded. Missing metadata
  is unknown. Empty alternatives do not prove there were none. Config/artifact
  refs are printed without dereferencing. Owned logs receive run timeline anchors.
- Cycles, dangling refs, duplicate/conflicting IDs or producers, malformed,
  oversized/deep JSON and duplicate keys produce integrity findings. Non-unique
  identities are not trusted. Unsupported versions/relations remain unknown.
  Missing producers are findings, not invented nodes. Local captured values
  may coexist with findings; this does not certify a complete graph.
- Inputs are **supplied to this step / recorded as input**. Even explicit
  `derived_from` probe edges are producer assertions, not proof of a read.
  Nesting/equal values never create inferred edges. Missing capture, eviction,
  excluded time and read limits are unknown within the printed negative-search
  scope, never proof of global absence.
- Optional profile: per-output declared-version invariant, four statuses and
  selector coverage, no baseline. Missing/off/hashed or unsafe integer values
  give unknown; no profile gives not_applicable. Comparison is descriptive only.

## Acceptance observations

Retained under `benchmark/results/single-output-trace/`: zero changed target
lines, zero drops, 6 fixture-B decisions, 27/27 decision/edge anchors resolving
over HTTP. Lane-01 stale/fresh: violated/satisfied, both coverage 3/3.

```text
Evidence tier: Observations from recorded boundary capture and explicit lineage assertions; boundary capture does not prove the code read a value.
Step "lane_target.decide" version "1" [captured]
  output: captured {"lane":"left","previous_used":true,"previous_version":8,"frame":10}
  "previous_frame" supplied to this step; recorded as input: captured 8
  "frame" supplied to this step; recorded as input: captured 10
  invariant: violated (coverage 3/3)
  dangling argument ref: unknown: no producer in search scope
Total nodes visited: 1; node budget used: 1/100; depth limit: 16; truncations: 0
```

The fresh version is 9; **both lanes are left**. Lane-01's previous-frame value
remains metadata; this is not a counterfactual lane-effect result. Both accepted
probes allocate argument refs without upstream producer identities, so each
output traces one decision, with upstream detail unknown. Fixture B exposes
stages, not rejected alternatives/internal rules/actual reads. The riskiest
assumption is reliable producer value identity/version claims across calls.

Tests first: tables for all requested graphs, privacy states, malformed data,
bounds and invariant cases; rapid properties for bounded termination, stored-node
membership and unknown never becoming inline data. Store/CLI tests cover time,
limits, cross-run discovery, query plans, comparisons and no mutation. The real
binary acceptance is also a Python unittest in CI.

Local gofmt/vet/lint/race/build pass; pinned Python 3.12.12 passes all 52 tests.
Two sandboxed attempts failed on Python-created temp directory permissions,
including unchanged tests. The authorized native run passed unchanged; retain
all attempts. Exact hosted Windows/Ubuntu CI is reported at publication.
No overhead rerun, optimization, schema, cohort, diff or scoring change. No
reserved seeds or blind-test content used. M5–M8/P0 remain unstarted.
