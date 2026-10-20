# Synthetic probe spike (S0, not a production adapter)

Pinned environment: uv 0.13.3, CPython 3.12.12, OTel SDK/proto HTTP 1.28.2;
the full dependency set is locked in uv.lock. The host Python 3.14 bootstraps uv
only. Standard requests/logging instrumentations are pinned at 0.49b2.
PyYAML is used only for bounded safe configuration. No Go dependencies changed.

Primary interface: python -m scout_probe, not an edit to the target source.
signalscout.yaml names directly defined synchronous module functions and explicit
argument/return capture allowlists. Default capture is OFF. Supported functions
must have fixed signatures; generators, async functions, class/descriptor paths,
builtins and unsupported loaders are reported. No arbitrary-code inference.

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
