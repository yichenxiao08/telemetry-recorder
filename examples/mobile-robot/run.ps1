param(
    [string]$Python = 'C:\Users\yiche\telemetry-project\adapters\provenance-python\.venv\Scripts\python.exe',
    [string]$Out = ''
)
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $false
$repo = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
$adapter = Join-Path $repo 'adapters/provenance-python'
$ss = Join-Path $repo 'bin/signalscout.exe'
$fixture = $PSScriptRoot
if (-not [IO.File]::Exists($ss) -or -not [IO.File]::Exists($Python)) {
    throw 'Build bin/signalscout.exe and prepare the locked Python adapter environment.'
}
if (-not $Out) { $Out = Join-Path $repo ('.cache/mobile-robot/' + [guid]::NewGuid().ToString('N')) }
if ([IO.Directory]::Exists($Out) -or [IO.File]::Exists($Out)) { throw 'Output must be a new path.' }
$null = [IO.Directory]::CreateDirectory($Out)
$Out = (Resolve-Path $Out).Path
$utf8 = [Text.UTF8Encoding]::new($false)

foreach ($name in @('reference', 'regression')) {
    $root = Join-Path $Out $name
    $null = [IO.Directory]::CreateDirectory($root)
    $db = Join-Path $root 'capture.db'
    $doctor = Join-Path $root 'doctor.json'
    $plain = & $Python -B (Join-Path $fixture 'robot_sim.py') --controller $name
    if ($LASTEXITCODE -ne 0) { throw 'Uninstrumented simulation failed.' }
    [IO.File]::WriteAllText((Join-Path $root 'uninstrumented.json'), ($plain -join "`n"), $utf8)
    [IO.File]::WriteAllBytes($db, [byte[]]@())
    $server = $null
    $start = [DateTimeOffset]::UtcNow.AddSeconds(-2).ToString('yyyy-MM-ddTHH:mm:ss.fffffffZ')
    try {
        # Never stop another process listening on the configured port.
        $listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, 4318)
        try { $listener.Start() } catch { throw 'Port 4318 is unavailable.' } finally { $listener.Stop() }
        $server = Start-Process -FilePath $ss -WindowStyle Hidden -PassThru -ArgumentList @(
            '--db', "`"$db`"", 'serve', '--listen', '127.0.0.1:4318', '--retention', '0'
        ) -RedirectStandardOutput (Join-Path $root 'server.stdout.log') -RedirectStandardError (Join-Path $root 'server.stderr.log')
        $ready = $false
        for ($i = 0; $i -lt 50; $i++) {
            if ($server.HasExited) { throw 'Owned server exited; inspect server.stderr.log.' }
            try { $null = Invoke-WebRequest 'http://127.0.0.1:4318/runs' -TimeoutSec 1; $ready = $true; break }
            catch { Start-Sleep -Milliseconds 100 }
        }
        if (-not $ready) { throw 'Owned server did not become ready.' }
        Push-Location -LiteralPath $adapter
        try {
            & $Python -B -m scout_probe --config (Join-Path $fixture 'signalscout.yaml') --doctor $doctor --path $fixture --manifest --drain-timeout 30 --queue-max-items 1024 --queue-max-bytes 2000000 robot_sim:main -- --controller $name 1> (Join-Path $root 'simulation.json') 2> (Join-Path $root 'probe.stderr.log')
            if ($LASTEXITCODE -ne 0) { throw 'Probe failed; inspect its retained logs.' }
        } finally { Pop-Location }
        $stop = [DateTimeOffset]::UtcNow.AddSeconds(2).ToString('yyyy-MM-ddTHH:mm:ss.fffffffZ')
        $captured = Get-Content -LiteralPath (Join-Path $root 'simulation.json') -Raw
        if (($captured.Trim()) -cne (($plain -join "`n").Trim())) { throw 'Instrumented behavior differs from uninstrumented simulation.' }
        $probe = Get-Content -LiteralPath $doctor -Raw | ConvertFrom-Json
        if (-not $probe.drained.complete -or -not $probe.drained.drained -or $probe.runs_created -ne 12 -or $probe.records_captured -ne 12 -or $probe.drained.failed -ne 0 -or $probe.drained.dropped -ne 0 -or $probe.drained.unknown -ne 0 -or $probe.drained.attempted -ne $probe.drained.acknowledged) { throw 'Incomplete capture; inspect doctor.json.' }
        $counters = $probe.counters
        $required = @('attempted', 'accepted', 'exported', 'acknowledged', 'failed', 'dropped', 'unknown', 'queue_size', 'retained_items', 'drop_bytes', 'drop_full', 'manifest')
        foreach ($key in $required) {
            if ($null -eq $counters.PSObject.Properties[$key]) { throw "Missing counter $key." }
        }
        foreach ($key in @('failed', 'dropped', 'unknown', 'queue_size', 'retained_items', 'drop_bytes', 'drop_full')) {
            if ($counters.$key -ne 0) { throw "Capture counter $key is nonzero." }
        }
        foreach ($key in @('attempted', 'accepted', 'exported')) {
            if ($counters.$key -ne $counters.acknowledged) { throw 'Capture delivery counters disagree.' }
        }
        if ($counters.acknowledged -ne $probe.drained.acknowledged -or @($probe.probes).Count -ne 1 -or $probe.probes[0].status -ne 'attached' -or $probe.probes[0].calls -ne 12 -or $probe.probes[0].records_captured -ne 12) { throw 'Probe population or delivery differs.' }
        $scope = $counters.manifest.epoch
        if (-not $scope -or -not $counters.manifest.scope_tail_sent -or $counters.manifest.expected_runs -ne 12) { throw 'Incomplete capture scope.' }
        $completeJson = & $ss --db $db cohort completeness --scope $scope --runs
        if ($LASTEXITCODE -ne 0) { throw 'Completeness query failed.' }
        [IO.File]::WriteAllText((Join-Path $root 'completeness.json'), ($completeJson -join "`n"), $utf8)
        $complete = $completeJson -join "`n" | ConvertFrom-Json
        if (-not $complete.population_known -or $complete.scope_expected -ne 12 -or $complete.counts.complete -ne 12 -or $complete.counts.partial -ne 0 -or $complete.counts.unknown -ne 0 -or $complete.counts.absent -ne 0) { throw 'Persisted scope incomplete.' }
        $relationsJson = & $ss --db $db relations --profile (Join-Path $fixture 'profile.json') --from $start --to $stop --limit 20 --record-limit 10000 --json
        if ($LASTEXITCODE -ne 0) { throw 'Relations query failed.' }
        [IO.File]::WriteAllText((Join-Path $root 'relations.json'), ($relationsJson -join "`n"), $utf8)
        $relations = $relationsJson -join "`n" | ConvertFrom-Json
        $counts = $relations.step_counts.'stop-near-obstacle'
        if ($relations.candidate_runs -ne 12 -or $counts.recorded_decisions -ne 12 -or $counts.unknown -ne 0 -or $counts.sampled_records -ne 0) { throw 'Incomplete relation analysis.' }
        $expectedViolations = if ($name -eq 'reference') { 0 } else { 2 }
        if ($counts.violated -ne $expectedViolations -or $counts.satisfied -eq 0) { throw 'Declared-rule result differs from the fixture expectation.' }
        Write-Host "$name : 12/12 complete; rule satisfied $($counts.satisfied), violated $($counts.violated), not applicable $($counts.not_applicable); see $root"
    } finally {
        if ($null -ne $server -and -not $server.HasExited) {
            Stop-Process -Id $server.Id
            $server.WaitForExit()
        }
    }
}
Write-Host "All evidence retained at $Out"