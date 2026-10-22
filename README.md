# Signal Scout

Decoded decision selectors and declared relations: see
[the syntax, closed grammar and PowerShell walkthrough](docs/declared-relations.md).
`inputs.x`, `output.1` and `output.0.3` address captured decoded values.
Literal dotted input names require the opt-in `escaped-v1` profile syntax.
`signalscout relations` checks one step's recorded calls without a baseline;
`cohort create --violates <rule>` pins runs with recorded violations.

Signal Scout receives all OpenTelemetry traces, logs and metrics, stores them in SQLite, and groups evidence into workload runs. M0–M4b provide intake, a seeded shop demo, ranked structural and context differences against passing history, and a local viewer and telemetry explorer. See [plan.md](plan.md) for the full milestone plan; agent explanations belong to later milestones.

## Windows / PowerShell quick start

Install Go 1.22+ (development/CI currently Go 1.27), Task 3, and golangci-lint 2.14. The Go race detector needs a native C compiler even though the production binary and SQLite driver require no cgo. The build task sets `CGO_ENABLED=0`.

```powershell
task test
task lint
task build
.\bin\signalscout.exe serve --db .\signalscout.db
```

In another PowerShell window:

```powershell
.\bin\signalscout.exe runs list --db .\signalscout.db
.\bin\signalscout.exe runs list --db .\signalscout.db --workload 'POST /checkout' --outcome fail
.\bin\signalscout.exe runs show '<run_id>' --db .\signalscout.db
```

`serve` binds to `127.0.0.1:4318`. Use `--listen` to change it. `Ctrl+C` stops HTTP intake, drains accepted writes, and closes open runs. `--idle-timeout 5s`, `--sweep-interval 1s`, `--retention 168h`, and `--queue-size 64` are the defaults. `--retention 0` disables retention. A per-workload latency rule can be repeated:

```powershell
.\bin\signalscout.exe serve --threshold 'POST /checkout=2s'
```

Open the viewer in another PowerShell window:

```powershell
Start-Process 'http://127.0.0.1:4318/runs'
```

It provides bounded run filters and pagination, span/event timelines, snapshots,
two baseline diff panels, numeric distributions and highlighted evidence links.
Templates escape telemetry; CSS is embedded and CSP disables scripts/external
requests. On-demand diffs use the serving writer queue. See the full
[PowerShell walkthrough](docs/viewer-walkthrough.md), including saved demo commands.
The explorer adds context logs/metrics, service overview and search. The completed
performance task and the owner's revised workload gates are recorded in
[docs/performance.md](docs/performance.md); remaining misses are preserved.

The queue is bounded in batches, each decoded request limited to 16 MiB. Success means the batch committed. Queue saturation or storage failure returns 503 with `Retry-After: 1`; malformed input returns 4xx. Retrying identical requests is idempotent. Processes with accepted batches can still lose them if killed before commit; the HTTP client receives no success in that case. WAL uses `synchronous=NORMAL`, which does not promise survival of the most recent commits after power loss.

## Send telemetry

An SDK with an OTLP HTTP exporter can send directly. Set these variables before starting your instrumented application (Signal Scout does not instrument it for you):

```powershell
$env:OTEL_EXPORTER_OTLP_PROTOCOL = 'http/protobuf'
$env:OTEL_EXPORTER_OTLP_ENDPOINT = 'http://127.0.0.1:4318'
```

Alternatively, run an OpenTelemetry Collector using [docs/collector-config.yaml](docs/collector-config.yaml):

```powershell
otelcol --config .\docs\collector-config.yaml
```

The Collector receives gRPC on 4317 and HTTP on 4319, forwarding every signal to Signal Scout on 4318. No sampling processor is configured. An SDK sending via this Collector should use port 4319 for HTTP. Signal Scout accepts protobuf and OTLP JSON at `/v1/traces`, `/v1/logs`, `/v1/metrics`, including gzip bodies. OTLP JSON trace/span IDs use hex; original decoded records, resources, scopes, exemplars, links and all metric point types are retained.

Web runs are grouped by trace ID. ML runs require resource `signalscout.run.id` and can set `signalscout.workload`. Ambient logs and metrics are retained with no fabricated run. On close, ambient evidence matching a span resource exactly or by host.name/service.instance.id is linked as context, with rule provenance. The default context window pads the run by 30 seconds before and 5 seconds after; use --context-before/--context-after to change it. Context does not change ownership or outcome. See [docs/run-spec.md](docs/run-spec.md) and [docs/open-questions.md](docs/open-questions.md) for the exact choices and remaining ambiguities.

Raw telemetry is retained. Log template normalization is a derived field, not redaction; the broader redaction milestone is M8.

## Validation and measurement

```powershell
task test
task lint
task build
```

CI runs formatting, vet, lint, race tests and build on Ubuntu and Windows. Synthetic OTLP recordings in `testdata/otlp` cover both run profiles and ambient data; replay results are checked against a hand-authored golden summary.

To reproduce M1 ingest measurements:

```powershell
task benchmark-ingest
```

See [docs/benchmark.md](docs/benchmark.md) for what is timed and
[docs/performance.md](docs/performance.md) for historical misses, improvements and
the revised combined profiles. Older reports are retained. These are intake
measurements, not the later multi-failure benchmark.


## M2 shop demo

```powershell
task shop-demo
$demo = Get-Content -Raw .\benchmark\results\shop-demo.json | ConvertFrom-Json
.\bin\signalscout.exe runs list --db $demo.database --workload 'POST /checkout' --outcome fail
.\bin\signalscout.exe runs show $demo.evidence.run_id --db $demo.database
```

Requires a running Docker Desktop Linux engine. The task produces 40 passing baselines, changes the pool from 20 to 5, sends 300 loaded requests, and verifies actual failure evidence plus separate host-resource context. It stops its stack and native intake before returning; the database remains for CLI inspection. See [samples/shop/README.md](samples/shop/README.md) for topology, seeded workload, manual commands and limitations.

## M3 diff (PowerShell)

Run from the repository root after intake has stopped. The accepted seed reports
reference local databases under ignored `.cache/`; regenerate with `task shop-demo`
on a fresh checkout. `task diff-demo` validates all failing runs in both modes and
generates the cutoff and representative run ID used below.

```powershell
task build
task diff-demo -- -demo benchmark/results/shop-demo-seed-1.json -out benchmark/results/diff-seed-1
$demo = Get-Content -Raw .\benchmark\results\shop-demo-seed-1.json | ConvertFrom-Json
$check = Get-Content -Raw .\benchmark\results\diff-seed-1.json | ConvertFrom-Json

# Default: recent passing runs strictly before this failure
.\bin\signalscout.exe --db $demo.database diff $check.representative_run_id

# Explicit count (recorded as count mode)
.\bin\signalscout.exe --db $demo.database diff $check.representative_run_id --baseline-n 40

# Pre-change window (a run ID can also be supplied as the cutoff)
.\bin\signalscout.exe --db $demo.database diff $check.representative_run_id --baseline-before $check.failure_phase_start
```

Add `--json` for the exact baseline specification, evidence IDs, related
differences and scores. Fewer than 10 passes produces an explicit
`insufficient_baseline` status and diagnostic. Results are persisted without
changing run finalization or context links. The standalone CLI follows the
existing one-writer-process agreement; stop `serve` before persisted analysis.

The pre-change window ranks the pool config, retry and acquire latency in the
top three for all failures in both [seed 1](benchmark/results/diff-seed-1.md) and
[seed 2](benchmark/results/diff-seed-2.md). For the latest failure, default mode
uses recent pool=5 passes, so the pool-size difference correctly disappears.
The common cache miss then ranks third, behind retry and latency; its presence
is not proof of a cause. See [scoring formulas](docs/diff-scoring.md).

To generate and validate a new seed-2 database (updates the seed-2 reports):

```powershell
task shop-demo -- -seed 2
Copy-Item .\benchmark\results\shop-demo.json .\benchmark\results\shop-demo-seed-2.json
task diff-demo -- -demo benchmark/results/shop-demo-seed-2.json -out benchmark/results/diff-seed-2
```

The historical M3 comparison excludes context. Current CLI and viewer
comparisons include the M4b extension described below.

## Telemetry explorer and context comparisons (M4b)

`signalscout serve` also serves `/telemetry` and the run context panel. Queries
require a time range, a result limit and a service, trace or explicit no-service
partition. Metrics use local SVG charts; logs retain trace correlation, severity
and template counts. Context differences use config-compatible retrospective
windows, with exact baseline IDs and clickable evidence.

See [the PowerShell explorer walkthrough](docs/explorer-walkthrough.md),
[context scoring policy](docs/context-differences.md), and
[generated M4b measurements](benchmark/results/m4b-summary.md).
The original combined-ingest targets were missed and then changed by the owner;
the revised profiles also retain their per-seed misses. Saved seed 1 did not
sample an in-use peak; its missing context detection is reported explicitly.

## Check local setup

Run `.\bin\signalscout.exe doctor` after `task build` from this repository's root.
It reports the binary, Python tools/imports, database-path access and loopback port
availability, then prints a tiny synthetic PowerShell example. Use `--json <new-path>`
for a versioned report. It starts no server or recording and installs nothing.
New database paths warn because creation permissions are not tested by writing.
See [doctor details and limits](docs/doctor.md).
