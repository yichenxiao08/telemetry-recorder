# Signal Scout

Signal Scout receives all OpenTelemetry traces, logs and metrics, stores them in SQLite, and groups evidence into workload runs. M0 and M1 provide intake and run inspection. See [plan.md](plan.md) for the full milestone plan; diffing and explanations belong to later milestones.

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

Web runs are grouped by trace ID. ML runs require resource `signalscout.run.id` and can set `signalscout.workload`. Ambient logs and metrics are retained with no fabricated run. On close, matching-resource ambient evidence within the run interval is linked as context. Context does not change ownership or outcome. See [docs/run-spec.md](docs/run-spec.md) and [docs/open-questions.md](docs/open-questions.md) for the exact choices and remaining ambiguities.

Raw telemetry is retained. Log template normalization is a derived field, not redaction; the broader redaction milestone is M8.

## Validation and measurement

```powershell
task test
task lint
task build
```

CI runs formatting, vet, lint, race tests and build on Ubuntu and Windows. Synthetic OTLP recordings in `testdata/otlp` cover both run profiles and ambient data; replay results are checked against a hand-authored golden summary.
