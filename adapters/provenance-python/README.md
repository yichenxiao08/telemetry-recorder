# Synthetic probe spike (S0, not a production adapter)

Pinned environment: uv 0.13.3, CPython 3.12.12, OTel SDK/proto HTTP 1.28.2;
the full dependency set is locked in uv.lock. The host Python 3.14 bootstraps uv
only. Standard requests/logging instrumentations are pinned at 0.49b2.
PyYAML is used only for bounded safe configuration. No Go dependencies changed.

Primary interface: python -m scout_probe, not an edit to the target source.
signalscout.yaml names directly defined synchronous module functions or methods
of top-level plain Python classes, with explicit argument/return/receiver-field
allowlists. Default capture is OFF. Fixed signatures are required. Unsupported
callables are reported with reasons; there is no arbitrary-code inference.

## Runner and method usability (2026-10-21)

The entry point is a plain synchronous `module:function`. Runner options go before
the explicit `--` separator; everything after it belongs only to the target.
Before importing the target, the runner sets `sys.argv` to the resolved target
script filename (module name if no source origin), followed by those target args.
It restores the previous argv object and search path on normal/error exits.
Nonzero target exits and exceptions propagate. A normal `SystemExit(0)` still
passes through capture checks, so zero capture cannot silently return success.

~~~powershell
task build
$repo = (Get-Location).Path
New-Item -ItemType Directory -Force .cache/probe-usability | Out-Null
$doctor = Join-Path $repo '.cache/probe-usability/doctor.json'
# In a separate PowerShell window, start the existing intake:
.\bin\signalscout.exe --db .cache/probe-usability/manual.db serve --retention 0
# With intake running, preflight (dry import, no entry-point execution):
uv run --directory adapters/provenance-python --frozen python -m scout_probe --config fixtures/usability/signalscout.yaml --doctor $doctor --path fixtures/usability --preflight class_environment:main -- run --episodes 10 --seed 1
# Actual target run:
uv run --directory adapters/provenance-python --frozen python -m scout_probe --config fixtures/usability/signalscout.yaml --doctor $doctor --path fixtures/usability class_environment:main -- run --episodes 10 --seed 1
.\bin\signalscout.exe --db .cache/probe-usability/manual.db runs list
~~~

Preflight resolves/imports the module, validates the entry and attachments, and
reports the exact isolated argv in doctor JSON. It does not execute `main`,
validate an argparse parser defined inside it, or sandbox import-time side effects.
Use a fresh process: wrappers and aliases are process-scoped, not hot-reloadable.

Config `function: Environment.step` supports instance methods; classmethod and
staticmethod descriptors are preserved. `capture: {args: [action], self: [score,
steps], return: true}` captures ONLY the two named builtin-valued instance-dict
attributes before the call. `cls: [family]` similarly permits named class data.
Implicit self/cls objects are never captured. Property/descriptor reads are never
used to obtain state. Missing or unsupported values remain uncaptured and visible
in counters. `complete` covers recorded explicit arguments/selected fields only,
not all receiver state or proof of code reads. No new privacy gate is included.

Only a class's directly defined method can be configured: use the declaring base
for inherited methods. If both overriding child and base methods are configured,
the same receiver/name's active child->base chain records the outermost override
once. The base invocation is counted and `probe_super_suppressed` reports omitted
duplicate recording. This also covers an explicit base call within that chain;
it does not inspect Python bytecode to prove a `super()` expression. Different
method names still record nested steps. Identity is compared with `is`, never
user equality/hash. Active method state is context-local, reset in finally,
bounded to 64 calls; receiver field/MRO/envelope/queue budgets remain bounded.

Unsupported: generators/async/async generators; properties and custom descriptors;
nested classes/functions; C/native/builtin callables; variadic or oversized
signatures; custom signatures/aliases; constructors/dunder probes, custom
metaclasses/access hooks/slots/oversized MROs; inherited aliases configured on a
subclass; unsupported loaders; reload; references cached before attachment.
Descriptor introspection never invokes the property's getter. Introspection or
identity-sensitive callers may observe wrapper functions/extra traceback frames.

Human-readable stderr and doctor JSON always report configured/attached/failed
probes, reasons, invocation counts, captured decisions, run contexts and transport
counters. Context creation and queue admission are not proof of stored delivery.
Each outer probed call creates a run; episodes are not automatically grouped.

| Runner status | Meaning |
|---|---|
| 0 | Successful supported execution and flush, or successful dry preflight |
| 2 | argparse usage error; target's nonzero exit is preserved |
| 3 | Unsupported, unattached, or (after execution) configured but uninvoked probe |
| 4 | Executed target produced zero admitted probe decisions or zero probe run contexts |
| 5 | Incomplete flush or recorded drops/failed/unknown delivery |

Zero capture has precedence over partial coverage, then delivery status. Unexpected
target exceptions keep their original exception/exit semantics after reporting.
Default-off capture and synthetic-only restrictions still apply.

One exploratory paired measurement is in `benchmark/results/probe-usability`:
300 samples/kind, one trial, 60 TOTAL calls/sec, same four scalars and returned
alternatives, receiver capture OFF. Function p50/p99 210.95/421.31 us; instance
method 219.20/403.34 us. All 640 warmup+measured decisions and 1,280 spans were
acknowledged, with zero drops. Quantile differences are +8.25/-17.97 us, not a
causal or budget qualification. Full added method/receiver-capture cost against
100/250 or 330/830 budgets is UNMEASURED. No optimization or repeat trial occurred.

The import hook replaces configured function objects in memory only.
Aliases retained before attachment, module reload semantics, opaque C/native
calls and custom introspection-sensitive code need separate validation; do not
call them transparently supported. Additional traceback wrapper frames are
expected; original exception objects and application frames are preserved.
The CLI is a synthetic Python prototype. The production signalscout record,
probe add, mark and explain commands remain design-only.

## Native PowerShell

From the repository root, with uv on PATH:

~~~powershell
task build
go run ./scripts/framefixture -out .cache/provenance/frame-go
uv sync --project adapters/provenance-python --python 3.12.12 --frozen
uv run --directory adapters/provenance-python --frozen python -m unittest discover -s tests -v
~~~

Start intake in one window:

~~~powershell
.\bin\signalscout.exe --db .cache/provenance/manual.db serve --retention 0
~~~

In another window (no target edits):

~~~powershell
$repo = (Get-Location).Path
$doctor = Join-Path $repo '.cache/provenance/frame-doctor.json'
uv run --directory adapters/provenance-python --frozen python -m scout_probe --config fixtures/signalscout.yaml --path fixtures --doctor $doctor frame_target:main
~~~

Use a fresh database for each full validation. Automated test/measurement
support starts the real Go binary on an ephemeral loopback port. It opens no
second operational writer. Closing the support process is not a production
record wrapper and does not implement saved-application replay.

## What the probe records

A call boundary: arguments before mutation (only allowed builtin values),
return value, duration and exception type. Return objects are never replaced;
host exceptions are re-raised unchanged. No exception message or user object
repr/serializer is called. Body execution timing is separate from recording cost.

Decision/lineage envelopes use the existing OTLP attributes and same bounded
recorder queue. Their edges mean "argument supplied to this call returning this
output"; they do not prove every argument influenced the output. Unobserved
locals, globals, rules, rejected alternatives and rationale remain unknown.
The envelope's complete flag describes captured call-argument values only, not
a complete application data-flow graph. Never use it alone to prove an invariant.

The optional record_decision method is retained for producer-declared semantics.
Default field capture remains off. This prototype is NOT a recursive privacy
gate: real personal/conversational data is prohibited until P0.

Standard OTel requests tracing bridges to the SAME queue and preserves only
method/status metadata; URL/query/body capture is off. Standard logging
instrumentation supplies trace-context correlation without changing message
formatting. Log body export is OFF, and other HTTP clients/frameworks are not
claimed instrumented. Doctor reports these coverage boundaries explicitly.

## Safety and cost measurement

No host-path network/queue-capacity wait. The daemon owns HTTP/protobuf work.
Queue capacity and payload/structure limits bound retained work. Drops, failed
exports and cooldown/no-op calls remain visible. Context exit does not flush or
join and never suppresses the application's exception. Explicit flush has a
deadline; drain success is not delivery success when export_failed is nonzero.
Python runtime/queue locks and scheduling are not hard real-time guarantees.

The fixed cost target has four scalar inputs and RETURNED two scored alternatives.
The probe does not invent unreturned alternatives. The serialized envelope is
checked <=2 KiB; actual bytes are recorded. Thresholds stay p50<=100 us and
p99<=250 us. Primary timing is caller wall time minus ONLY target-body execution,
including automatic run creation/exit, snapshotting, JSON, enqueue and cleanup.
Entry/exit and explicit flush times are also shown separately.

Five trials each: 1800 calls over 30 seconds at 60 Hz, and 10000 tight-loop calls,
for reachable real intake, unreachable intake and a deliberately saturated queue.
The saturated queue has capacity1 and a stopped exporter. It is an injected fault,
not a successful export. Cooldown/no-op paths are never presented as full capture.
Counts are native queue items (decision spans AND root spans); serialized is
decision-envelope count. Source hashes and queue state make drops auditable.
A/B fixture emission fits the measured shop envelope; tight-loop saturation is
the explicitly requested stress test, not a claimed supported workload rate.

count_instrumentation.py counts AST source spans of recorder imports, calls and
run-context headers, excluding original application bodies. It compares a
pre-run source snapshot to post-run source and reports physical plus nonblank
config lines. No source modification is inferred merely from in-memory wrapping.
## Reproduce synthetic breadth (PowerShell, repository root)

The source shop database must already exist at the path in breadth.py. This is
a replay of saved synthetic shop OTLP, not a live shop execution or copied diff.
Use a fresh output directory each time. Only Go tools open SQLite, read-only.

```powershell
task build
go build -o bin/provenancespike.exe ./scripts/provenancespike
go run ./scripts/framefixture -out .cache/provenance/frame-go
$env:UV_PYTHON_INSTALL_DIR = Join-Path (Get-Location) '.cache/uv-python'
$env:UV_CACHE_DIR = Join-Path (Get-Location) '.cache/uv-cache'
$uv = Join-Path (Get-Location) '.cache/uv-tool/bin/uv.exe'
$spikeOut = Join-Path (Get-Location) ('.cache/provenance/replay-' + (Get-Date -Format 'yyyyMMdd-HHmmss'))
& $uv run --directory adapters/provenance-python --frozen python -m unittest discover -s tests -v
& $uv run --directory adapters/provenance-python --frozen python breadth.py --out $spikeOut
$snapshot = Join-Path $spikeOut 'snapshot.json'
$examples = Join-Path $spikeOut 'examples.json'
& $uv run --directory adapters/provenance-python --frozen python report_s0b.py --input $snapshot --out $examples
.\bin\signalscout.exe --db (Join-Path $spikeOut 'fresh.db') serve --listen 127.0.0.1:4318 --retention 0
```

Open http://127.0.0.1:4318/runs. Copy a run ID from report.json or examples.json,
open its run page, expand event attributes and inspect signalscout.record.json.
For review logs, use the logs explorer with trace ID, explicit October 2026 time
bounds and limit 100. Copy logical refs manually; only native evidence anchors
resolve. Mark/reference/semantic explain controls are design only.

The overhead harness has no concurrent replay workload. Use a separate fresh DB
and absolute output paths; run measure.py only after safety tests pass. Original
and monotonic-duration results retain the same rates, five seeds and budget.

## Synthetic tuple returns and deterministic sampling

Capture remains OFF unless explicitly requested. Whole `capture.return: true`
accepts builtin tuples and standard namedtuples as ordered JSON arrays. Existing
list limits apply: 32 items per sequence, depth 6, total traversal budget 256,
and decision envelope 8192 bytes. Fields keep their original position; namedtuple
field names are configuration selectors, not a new wire shape. Existing scalar,
list and dict function/method wire values are unchanged without these options.

```yaml
probes:
  - module: tuple_environment
    function: loop
  - module: tuple_environment
    function: Environment.step
    capture:
      return_items: [0, 1]  # alternatively [state, reward] for a namedtuple
    sampling:
      every_nth: 10
      max_per_run: 4
```

Use either whole `return: true` or `return_items`, not both. An explicit item
allowlist enables only the selected zero-based indices or namedtuple field names;
indices and names form a union. An empty list enables no elements. Other slots
are `{"unknown":"not_allowlisted"}` without a drop. Item allowlists require a
sequence; scalar/dict returns are rejected rather than exposed. Unsupported or
oversized tuple elements (including nested sequence values) become explicit
`{"unknown":"<reason>"}` markers. The output has `element_drops`; doctor counters
include `drop_element` and reason counts. Numpy scalars/arrays stay unsupported
with `numpy_unsupported`; numpy is not a dependency. Whole oversized sequences,
invalid root values and envelopes over budget still cause `drop_invalid`.
Existing list/dict root rejection behavior remains unchanged. No user repr,
serializer, iterator or property is invoked. Snapshots do not retain caller data.

Sampling is optional, per configured probe and explicit recorder run. Index 1
is selected, then 1+N, 1+2N, etc. `max_per_run` caps selection opportunities,
including selected calls whose capture later fails; the cap never retries a lost
record. When both rules exist they intersect. Counts reset for each run; nested
calls of the same probe share its counter. A standalone probe call creates its
own run, so group repeated steps with an outer probed loop or explicit run to
sample across them. Super-chain suppression retains its existing semantics.
Calls skipped by sampling execute normally before any argument binding or value
serialization. A skipped frame supplies no span; nested captured calls retain
the nearest captured parent. This does not establish value links.

Selected records add `sampling.rule` and one-based `sampling.call_index` and set
`complete: false`. Doctor and stderr report per-probe calls, admitted decision
records, rule and captured/seen fraction; run root attributes retain per-probe
run totals, rule and fraction. Doctor/stderr retain only the latest 32 sampled
probe/run summaries with an explicit omitted count; aggregate counters remain
exact. Captured means queue-admitted, not necessarily persisted: also inspect
ACK, queue drops and flush completion. Sampling-out is counted separately and
cannot cause exit 5. Element/queue/invalid/delivery drops still can. Statuses
remain zero capture 4, coverage gap 3, capture/delivery loss 5 (existing priority).
Host exceptions and nonzero target exits keep their original identity/status.

Capture is incomplete. Unsampled calls and unknown elements provide no evidence
for invariant satisfaction or absence claims. Existing trace, lane, pipeline,
ranking, cohort and diff behavior is unchanged; a trace of a captured sample is
only that boundary observation. Do not interpret sampled traces as a full run.

Synthetic fixture: `fixtures/tuple_environment.py` has plain RL-style steps
returning `(state, reward, done)` and a high-rate loop. Run the full unittest suite
with Python 3.12; new tests also verify real serve, runs list and bounded trace.
`measure_tuple_sampling.py` retains one optional small paired local cost harness;
it does not optimize or establish a supported rate or performance budget.
