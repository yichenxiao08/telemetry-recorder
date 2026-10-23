# Local setup check

From the repository root in native PowerShell:

```powershell
task build
.\bin\signalscout.exe doctor
.\bin\signalscout.exe --db C:\path\existing-parent\signalscout.db doctor --listen 127.0.0.1:4318
$report = Join-Path ([IO.Path]::GetTempPath()) ('signalscout-doctor-' + [guid]::NewGuid() + '.json')
.\bin\signalscout.exe doctor --json $report
```

The checklist uses PASS, WARN and FAIL. Any FAIL returns exit status 1; warnings
alone return 0. The JSON file has `schema_version: 1`, `binary`, `python`, `checks`,
`summary` and `examples`. Each check has `id`, `status`, `message` and `fix`.
Text and JSON come from the same report. JSON is written only when requested,
requires a new file, and cannot use the database path. Existing files are never
overwritten. Output contains local executable/adapter/database paths and versions;
review those paths before sharing. Environment variables and subprocess error
text are not included.

Doctor reports the executing binary's path, Go module build version and embedded
VCS revision/dirty state where available. Development builds have no release
version; missing build metadata is explicit. It checks `py -3`, `python` and `uv`
on PATH separately. The adapter README recommends uv, CPython **3.12.12** and the
frozen lockfile. Selection is an existing adapter `.venv`, then `py -3`, then
`python`; `--python <executable>` overrides selection. `--adapter <directory>`
defaults to `adapters/provenance-python` relative to the working directory.
Outside the repository, supply the adapter's absolute path. A selected Python
that differs from 3.12.12 warns, including a different 3.12 patch version.
The adapter's `scout_probe.py` must be a regular accessible file; doctor checks
its presence without importing it. A missing adapter is a FAIL.

The selected interpreter runs isolated, bytecode-disabled `importlib` checks for
PyYAML, OTel SDK trace, OTel OTLP/HTTP trace exporter and requests. It suppresses
import stdout/stderr and reports booleans only. A missing/broken dependency is a
FAIL with a PowerShell `-m pip install` command pinned to the adapter's package
versions. If pip is also absent, the printed command first runs `-m ensurepip`.
These are repair suggestions; doctor never installs, syncs uv, downloads Python,
imports the probe or runs the application. The recommended frozen `uv sync`
command restores the full locked environment, including instrumentation packages;
doctor's four imports alone do not certify all instrumentations or version
compatibility. Trusted installed Python/packages may execute initialization code
during import. Subprocesses have five-second deadlines and bounded captured output.

Database checks never open SQLite or read telemetry. An existing regular file is
opened read/write **without create, truncate or write**, then closed. This tests
file access, not SQLite validity, locking, directory ACLs for WAL files, free
space, or durable commits. If the file does not exist, doctor checks only that its
parent directory exists and reports **WARN: create/write permissions untested**.
It creates neither a database nor a scratch file. A missing/inaccessible parent
is a FAIL. This conservative choice preserves the requested reporting-only
contract; actually proving creation permission requires a write.

The optional port check binds the requested loopback address and immediately
closes it without accepting connections. It starts no HTTP server and sends no
network requests. Use `--listen ''` to skip it. A numeric port in 1..65535 and a
loopback IP or localhost are required. Availability is a momentary observation,
not a reservation for a subsequent serve process.

Doctor prints a synthetic tiny Python script, allowlisted probe config, empty DB
file creation, a generated serve command for a second PowerShell window, probe
invocation and `runs list`. It executes none of these instructions. The example
needs the README's uv setup; wait for serve to listen before invoking the probe.
PASS means only that the stated setup checks succeeded. It does not establish
capture, delivery, run grouping, completeness, analysis quality or privacy safety.
Synthetic data only continues to apply.
