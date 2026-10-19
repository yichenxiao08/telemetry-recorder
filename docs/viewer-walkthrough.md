# M4 viewer walkthrough (PowerShell)

Run commands from the repository root on native Windows. The viewer and OTLP
intake share `signalscout serve`, defaulting to `127.0.0.1:4318`.
Docker is only required when generating new shop data.

## 1. Start with an accepted local demo

Stop any other Signal Scout process using this database. In PowerShell:

```powershell
task build
$demo = Get-Content -Raw .\benchmark\results\shop-demo-seed-1.json | ConvertFrom-Json
Test-Path $demo.database
.\bin\signalscout.exe --db $demo.database serve --listen 127.0.0.1:4318 --retention 0
```

`Test-Path` should return `True`. Databases live under ignored `.cache/` and are
not included in a fresh checkout. If it returns `False`, generate fresh data
with Docker Desktop's Linux engine running, then use the latest report:

```powershell
task shop-demo -- -seed 1
$demo = Get-Content -Raw .\benchmark\results\shop-demo.json | ConvertFrom-Json
.\bin\signalscout.exe --db $demo.database serve --listen 127.0.0.1:4318 --retention 0
```

`--retention 0` preserves the demo while you inspect it. Stop this server with
Ctrl+C when finished. The before-phase suggestion comes from explicit stored
phase metadata; no benchmark JSON is loaded by the viewer.

Wait for the **OTLP and viewer HTTP listening** message before opening the
browser. Normal store startup rebuilds context links before listening; this can
be slow on retained demo data, especially under `-race`. Startup backfill and
context-link storage growth are part of the planned performance task before
M4b. No startup or ingest tuning is included in M4.

## 2. Run list → failing run

In a second PowerShell window:

```powershell
Start-Process 'http://127.0.0.1:4318/runs'
```

1. Set **Workload** to `POST /checkout` and **Outcome** to `Fail`.
2. The default time window covers the last seven days. For older saved runs,
   enter UTC RFC3339 bounds covering the report's `completed_at`, for example
   `2026-10-19T00:00:00Z` through `2026-10-20T00:00:00Z` for the committed demos.
   Start is inclusive; end is exclusive. Windows may cover at most 31 days.
3. Choose **Filter runs**, then open the latest failing run.
4. Check metadata, the span tree, durations, and event diamonds. Click a diamond
   to jump to its event details and highlight them. Span attributes are native
   expandable details. Config and environment snapshots are below the tree.
5. To check pagination, clear Outcome, set Rows to `2`, filter, and use **Next
   page**. Filters and exact time bounds remain fixed across pages; **First
   page** restarts the same filtered list.

For a direct link to the latest failure in the database currently being served,
repeat the `$demo` assignment in the second window, then run:

```powershell
$failures = .\bin\signalscout.exe --db $demo.database runs list --workload 'POST /checkout' --outcome fail | ConvertFrom-Json
$runId = [uri]::EscapeDataString($failures[0].run_id)
Start-Process "http://127.0.0.1:4318/run?id=$runId"
```

`runs list` uses a read-only connection and can run while serving. Standalone
CLI `diff` opens its own result writer and should run with this server stopped.

## 3. Compare both baselines

1. Choose **Compare baselines** in the run page's navigation.
2. Inspect **Before the failure phase**. For M2, a suggested cutoff is the first
   run with the same workload, config, revision and explicit failure-phase label.
   You may replace it with a run ID or an RFC3339 timestamp. This has the same
   exclusive-cutoff semantics as CLI `--baseline-before`, capped at this failing
   run's start. It is a metadata suggestion, not automatic cause detection.
3. Choose the **Compare baselines** button. This explicit POST computes and
   persists results through the serving process's existing writer queue.
4. Inspect the two panels side by side (stacked on narrow windows). Each shows
   baseline mode, selected count, requested count, resolved cutoff, and an
   expandable list of selected run IDs. Those IDs open the corresponding runs.
5. The pre-change panel on the accepted demos ranks pool configuration, retry
   presence and acquisition latency first through third, with cache miss fourth.
   The latest failure's recent panel uses pool=5 passes, so its pool-size config
   difference disappears. Retry and latency rank first and second, followed by
   cache miss. Early failures can still have mixed recent pool history.

Without explicit phase metadata, the before-phase panel asks for a cutoff;
the recent panel still works. Fewer than ten usable passing runs produces an
explicit insufficient-baseline diagnostic. Reopening a run does not recompute
results; use the button again to compare current snapshots.

## 4. Diff → distribution → evidence

1. Each difference shows type, score, raw baseline rate, explanation and evidence
   IDs. For latency the rate is baseline exceedance among runs with that path;
   for count it is directional exceedance, for presence it is prevalence, and
   for config/env it is mode support. These rates are not causal probabilities.
2. Expand **Inspect baseline distribution** on a latency or count difference.
   Blue dots represent baseline values; the red vertical line marks the failing
   value. The table gives exact selected-run values and links. Latency omits
   absent paths; counts include them as zeros. Expand **Related evidence** to
   inspect collapsed parent delays or repeated annotations and their charts.
3. Click the acquisition span's evidence ID: its timeline row is highlighted.
4. Return to the comparison and click the retry event's ID: the event is
   highlighted. Baseline evidence links open the appropriate baseline run.
5. Click a config evidence ID: its run's config snapshot is opened and
   highlighted. Environment evidence similarly targets the environment section.

The viewer contains no context log/metric panels, service overview or search.
Those are M4b. Scores rank unusual evidence; retry/acquire presence are symptoms,
while configuration and future context differences can support a mechanism.

## Automated acceptance and headers

```powershell
go test -race ./...
task lint
task build
```

The self-contained `httptest` navigation tests always run, covering list → run →
both diff panels → every evidence anchor, including config and baseline links.
They also check escaping, CSP, pagination, malformed trees, URL-safe IDs,
missing phase metadata, writer backpressure, and bounded concurrent requests.
The serve lifecycle test checks that viewer routes and OTLP coexist.

To repeat the optional check against each saved M2 database, stop `serve` first:

```powershell
$env:SIGNALSCOUT_VIEWER_DEMO = (Resolve-Path .\benchmark\results\shop-demo-seed-1.json).Path
go test ./internal/viewer -run '^TestStoredM2Walkthrough$' -count=1 -v
$env:SIGNALSCOUT_VIEWER_DEMO = (Resolve-Path .\benchmark\results\shop-demo-seed-2.json).Path
go test ./internal/viewer -run '^TestStoredM2Walkthrough$' -count=1 -v
Remove-Item Env:SIGNALSCOUT_VIEWER_DEMO
```

These checks open the dataset with one serving writer, use `httptest` requests,
and save the two diff results through that queue. They do not generate traffic
or tune ingest. Skip the optional commands if the local databases do not exist;
the ordinary fixture-based navigation test still runs in CI.

Additional race-instrumented saved-data check: seed 1 hit Go's unchanged
ten-minute test timeout inside startup context backfill, before any viewer HTTP
request. The failure is retained in
`benchmark/results/viewer-m2-seed-1-race.txt`; neither timeout nor assertions were
relaxed. The same seed-1 navigation test passed natively, resolving 46 evidence
links. The full self-contained race suite passes. Large-history startup under
the race detector remains unresolved for the planned performance task; the
native commands above verify the owner-facing runtime separately.

Native saved-data checks passed for both committed demos: seed 1 resolved 46
evidence links and seed 2 resolved 43, each through both persisted baseline
panels. Their test bodies took 63.46 s and 93.69 s including normal startup.
These durations are integration/startup observations, not per-diff latency.

While the server is running, headers can also be inspected with:

```powershell
$response = Invoke-WebRequest -UseBasicParsing 'http://127.0.0.1:4318/runs'
$response.Headers['Content-Security-Policy']
```

Expected policy starts with `default-src 'none'`, disables scripts, permits only
the embedded same-origin CSS, and forbids framing and external connections.
No stored telemetry is inserted as trusted HTML, CSS or script. CSS `:target`
implements highlighting without JavaScript.

Limits: 25 rows by default/100 maximum per list page; 5,000 structural records
per timeline; 200 baseline runs and 50,000 structural records per comparison;
two simultaneous computations. Oversized comparisons return a visible error
instead of a partial ranking. Requests have a ten-second deadline.

The owner will capture screenshots of the list, timeline, both panels,
distribution and highlighted evidence. No screen capture is claimed here.
