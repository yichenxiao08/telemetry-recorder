# Signal Scout

Signal Scout stores OpenTelemetry telemetry, groups signals into runs, and compares failing runs to passing baselines. See [plan.md](plan.md) for the build plan and milestone status.

## Development

Requires Go 1.22+ and [Task](https://taskfile.dev). From PowerShell:

```powershell
task test
task lint
task build
.\bin\signalscout.exe --help
```

Milestone: M0 skeleton and M1 OTLP intake, grouping, and SQLite storage.
