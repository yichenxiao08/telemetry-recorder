# M4b explorer walkthrough (native PowerShell)

Run from the repository root. The saved demo reports reference ignored local
databases, so they must still exist. The helper copies them into `.cache/m4b/`
and upgrades only those copies; it never regenerates or edits saved telemetry.

```powershell
task context-demo
task build
$demos = Get-Content -Raw .\benchmark\results\m4b-context-demo.json | ConvertFrom-Json
$demo = $demos[0] # Use [1] for seed 2.
.\bin\signalscout.exe --db $demo.Database serve --listen 127.0.0.1:4318 --retention 0
```

In another PowerShell window:

```powershell
$demos = Get-Content -Raw .\benchmark\results\m4b-context-demo.json | ConvertFrom-Json
$demo = $demos[0]
$run = [uri]::EscapeDataString($demo.Representative)
Start-Process "http://127.0.0.1:4318/run?id=$run#context"
```

1. Inspect **Run context**. Resources are charted separately so phase/config
   fingerprints remain visible. Red lines show the failure marker. Each point
   links to its record, and every record lists its attachment rule/source.
2. Compare both baselines. Context metric and log differences use the policy
   in [context-differences.md](context-differences.md). Click evidence IDs to
   highlight their records on the failing or baseline run page. Padded context
   remains visible even when scoring excludes it.
3. Open **Telemetry explorer**. Enter `frontend` as Service, a range covering
   the demo (for the saved data: `2026-10-19T00:00:00Z` to
   `2026-10-20T00:00:00Z`), and Limit `100`. Choose service/route overview.
   Request rate uses the entire selected range; error fraction is errors divided
   by matching route-bearing server spans. Quantiles are exact over the bounded
   input, capped at 50,000 spans; oversized requests are rejected visibly.
4. Choose span search, filter name/status/duration/signal or resource attributes, and open a
   trace. Trace lookup uses telemetry, not a classified-run lookup. Use **Logs
   for this trace** to inspect its correlated logs. The latest failing traces
   in each saved seed have four logs.
5. Choose log search for `inventory-service`, optionally filter literal text or
   numeric severity (WARN=13, ERROR=17). Template counts aggregate the selected
   window and are limited by group count. Search result pages use stable
   timestamp/evidence-ID cursors. Aggregations do not accept row cursors.
6. Choose metrics, service `inventory-service`, name `db.pool.in_use`. The name
   and attribute filters select series; separate charts preserve resource and
   point attributes. Select **Resources without service.name** with the service
   field empty to browse host-only metrics independently of runs; use the
   `host.name` resource attribute filter to select a host. Charts display only
   the current bounded page. Histogram and
   other non-scalar records remain inspectable as records without scalar charts.

Every explorer query needs explicit From, To and Limit plus a service, explicit
no-service partition, or trace. Maximum range is 31 days, maximum page is 1,000.
The no-service partition uses migration 007's service-expression index with
IS NULL; host-only resources deliberately have no service label. No arbitrary
query language, alerting or dashboard builder exists.
All assets are embedded, templates escape telemetry, and CSP disables scripts
and external requests. No browser-side database writer is used.

Stop serve with Ctrl+C before standalone CLI diffs or helper invocations:

```powershell
$before = $demo.Trials | Where-Object { $_.RunID -eq $demo.Representative -and $_.Mode -eq 'before' } | Select-Object -First 1
$cutoff = $before.Result.baseline_spec.before_input
.\bin\signalscout.exe --db $demo.Database diff $demo.Representative --baseline-before $cutoff
.\bin\signalscout.exe --db $demo.Database diff $demo.Representative
task benchmark-explorer
```

The explorer benchmark requires the existing million-record fixture. To build
a new one, use a fresh destination and pass that same path to the query task:

```powershell
task benchmark-store -- -create -count 1000000 -seed 1 -db .cache/performance/new-million.db -out benchmark/results/new-million.json
task benchmark-explorer -- -db .cache/performance/new-million.db -out benchmark/results/new-million-explorer.json
```

Do not run timed measurements concurrently. Filesystem caches are not flushed;
the first read for each query opens a new reader pool.
