# First Python recording in native PowerShell

This copy-paste recipe uses the existing **public synthetic fixture** and CLI.
After prerequisites, it needs one PowerShell window, no uv on PATH, no manual
run-ID copying and no manually authored report request. It is not a general
application launcher, automatic probe selection or a diagnosis benchmark.

## Prerequisites

Run from the repository root in PowerShell 7 on Windows. Build with `task build`.
Prepare the locked CPython 3.12.12 environment using the
[adapter instructions](../adapters/provenance-python/README.md#native-powershell).
The block defaults to the adapter's `.venv\Scripts\python.exe`; in an isolated
checkout, change **only** `$python` to an already prepared absolute executable
path if needed. A Windows venv is not a portable directory to copy. Doctor checks
setup, installs nothing, and does not prove recording/delivery.

Synthetic data only: this uses legacy synthetic capture, **not strict P0 or a
production privacy gate**. Port 4318 must be free; an existing listener is never
stopped. The two explicitly allowlisted boundaries return scalars, avoiding the
fixture's mutable list returns. Without sampling, ten episodes make 30 policy
calls and ten reward calls: 40 automatic per-call runs, not ten episode runs.
This expected population comes from the public fixture, not inference.

## Paste the complete block

Every run creates a new output directory. Native failures stop the block. Retain
the artifacts on failure, fix the reported problem, and start a **new** run;
incomplete capture is not explanatory evidence.

```powershell
$ErrorActionPreference = 'Stop'
$repo = (Get-Location).Path
$adapter = Join-Path $repo 'adapters/provenance-python'
$python = Join-Path $adapter '.venv/Scripts/python.exe'
$ss = Join-Path $repo 'bin/signalscout.exe'
$fixture = Join-Path $adapter 'fixtures/usability'
$out = Join-Path $repo ('.cache/first-python/' + [guid]::NewGuid().ToString('N'))
if (-not [IO.File]::Exists($ss) -or -not [IO.File]::Exists($python)) {
    throw 'Build bin/signalscout.exe and prepare the locked adapter environment first.'
}
if (-not [IO.File]::Exists((Join-Path $fixture 'class_environment.py'))) {
    throw 'Run this block from the Signal Scout repository root.'
}
if ([IO.Directory]::Exists($out)) { throw 'Output collision; choose a new run.' }
$null = [IO.Directory]::CreateDirectory($out)
$db = Join-Path $out 'capture.db'
$config = Join-Path $out 'scalars.yaml'
$doctor = Join-Path $out 'probe-doctor.json'
$utf8 = [Text.UTF8Encoding]::new($false)
[IO.File]::WriteAllText($config, @'
version: 1
synthetic_only: true
service: synthetic.first-recording
probes:
  - module: class_environment
    function: Environment.policy
    capture: {args: [value], return: true}
  - module: class_environment
    function: Environment.reward
    capture: {return: true}
'@, $utf8)

# Port availability is momentary; doctor does not reserve it.
& $ss doctor --python $python --adapter $adapter --listen 127.0.0.1:4318 --json (Join-Path $out 'setup-doctor.json')
if ($LASTEXITCODE -ne 0) { throw 'Doctor failed; inspect setup-doctor.json.' }
[IO.File]::WriteAllBytes($db, [byte[]]@())
$server = $null
$captureStart = [DateTimeOffset]::UtcNow
try {
    $server = Start-Process -FilePath $ss -WindowStyle Hidden -PassThru -ArgumentList @(
        '--db', "`"$db`"", 'serve', '--listen', '127.0.0.1:4318', '--retention', '0'
    ) -RedirectStandardOutput (Join-Path $out 'server.stdout.log') -RedirectStandardError (Join-Path $out 'server.stderr.log')
    $ready = $false
    for ($i = 0; $i -lt 20; $i++) {
        if ($server.HasExited) { throw 'Own server exited; inspect server.stderr.log.' }
        try {
            $null = Invoke-WebRequest 'http://127.0.0.1:4318/runs' -TimeoutSec 1
            $ready = $true
            break
        } catch { Start-Sleep -Milliseconds 100 }
    }
    if (-not $ready -or $server.HasExited) { throw 'Own server did not become ready.' }

    Push-Location -LiteralPath $adapter
    try {
        & $python -B -m scout_probe --config $config --doctor $doctor --path $fixture --manifest --drain-timeout 30 --queue-max-items 1024 --queue-max-bytes 2000000 class_environment:main -- run --episodes 10 --seed 1 1> (Join-Path $out 'target.stdout.json') 2> (Join-Path $out 'probe.stderr.log')
        if ($LASTEXITCODE -ne 0) { throw 'Probe failed; inspect probe.stderr.log and probe-doctor.json.' }
    } finally { Pop-Location }

    $probe = Get-Content -LiteralPath $doctor -Raw | ConvertFrom-Json
    $loss = @($probe.counters.PSObject.Properties | Where-Object {
        ($_.Name.StartsWith('drop') -or $_.Name -in @('failed', 'unknown')) -and $_.Value -ne 0
    })
    if (-not $probe.drained -or $probe.records_captured -ne 40 -or $probe.runs_created -ne 40 -or $loss.Count -ne 0) {
        throw 'Capture counters are incomplete for this fixture; do not interpret evidence.'
    }
    $epoch = $probe.counters.manifest.epoch
    if (-not $epoch) { throw 'No capture-scope epoch; completeness is unknown.' }

    # The supported command is cohort completeness, not root completeness.
    $completeJson = & $ss --db $db cohort completeness --scope $epoch --runs
    if ($LASTEXITCODE -ne 0) { throw 'Persisted completeness query failed.' }
    [IO.File]::WriteAllText((Join-Path $out 'completeness.json'), ($completeJson -join "`n"), $utf8)
    $complete = $completeJson -join "`n" | ConvertFrom-Json
    if (-not $complete.population_known -or $complete.scope_expected -ne 40 -or $complete.counts.complete -ne 40 -or $complete.counts.partial -ne 0 -or $complete.counts.unknown -ne 0 -or $complete.counts.absent -ne 0) {
        throw 'Persisted scope incomplete; retain diagnostics, not an explanation.'
    }

    # Use the actual stored native decision ID, not a guessed logical reference.
    $runID = @($complete.runs.PSObject.Properties.Name | Sort-Object)[0]
    $detailJson = & $ss --db $db runs show $runID
    if ($LASTEXITCODE -ne 0) { throw 'Run inspection failed.' }
    [IO.File]::WriteAllText((Join-Path $out 'run.json'), ($detailJson -join "`n"), $utf8)
    $detail = $detailJson -join "`n" | ConvertFrom-Json
    $decision = @($detail.signals | Where-Object { $_.attrs.'signalscout.record.kind' -eq 'decision' })[0]
    if (-not $decision) { throw 'No recorded decision in selected run.' }

    # Short UTC window containing capture; reports reject windows over 31 days.
    $from = [DateTimeOffset]::new($captureStart.UtcDateTime.Date)
    $to = [DateTimeOffset]::UtcNow.AddDays(1)
    $request = @{
        version = 1; from_ns = $from.ToUnixTimeMilliseconds() * [long]1000000
        to_ns = $to.ToUnixTimeMilliseconds() * [long]1000000; record_limit = 200
        cohorts = @(); comparisons = @(); traces = @()
        records = @(@{ id = $decision.id; kind = 'native'; run_id = $runID })
    }
    $requestPath = Join-Path $out 'request.json'
    [IO.File]::WriteAllText($requestPath, ($request | ConvertTo-Json -Depth 12), $utf8)
    $reportJson = & $ss --db $db report build --request $requestPath --json
    if ($LASTEXITCODE -ne 0) { throw 'Report build failed.' }
    $reportPath = Join-Path $out 'report.json'
    [IO.File]::WriteAllText($reportPath, ($reportJson -join "`n"), $utf8)
    & $ss --db $db report verify --input $reportPath --json
    if ($LASTEXITCODE -ne 0) { throw 'Report verification failed.' }
    Write-Host "Recorded 40 complete scoped runs. Inspect artifacts in $out"
} finally {
    # This is forced cleanup, not graceful Windows console shutdown.
    # Analysis ran after acknowledged delivery, before stopping our own process.
    if ($null -ne $server -and -not $server.HasExited) {
        Stop-Process -Id $server.Id
        $server.WaitForExit()
    }
}
```

## Interpret the artifacts

`target.stdout.json` should contain ten rewards of `7`. `probe-doctor.json` retains
capture/delivery counters; `completeness.json` checks the stored expected
population. Larger queue/drain settings guarantee nothing; the gates check this
run. Doctor warnings and failures stay in the saved logs.

The report cites one actual stored decision. It does not infer the internal rule,
data dependence or a cause. Complete means complete **selected scope**, not
whole-program observation. Verification checks structural evidence/quantities,
not explanatory truth. Forced server cleanup is not graceful shutdown or a
production recording wrapper.

The separate [structured report walkthrough](report-walkthrough.md) demonstrates
cohort comparison and declared rules using its own synthetic data; this capture
does not magically produce those comparisons. For another supported application,
choose explicit probes/fields and preserve its real entry point/arguments. This
recipe neither chooses those boundaries nor demonstrates diagnosis speed/value.
