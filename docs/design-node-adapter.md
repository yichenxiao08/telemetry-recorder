# Node.js / TypeScript scalar capture adapter proposal

Status: design only, 2026-10-23. No adapter, package, command, runtime test or
performance result is delivered by this document. Implementation requires a
separate owner GO. Synthetic data only; the production privacy gate remains open.
This extends runtime coverage, not evidence of hard-bug usefulness.

## Decision and first supported profile

Build an explicit wrapper API, not a module patcher. The developer changes a
selected function definition/export to use `recorder.wrapSync(spec, fn)` and
opens a run with `recorder.withRun(name, callback)`. Names are proposed API.
The wrapper receives positional argument names from the declaration; it never
parses function source, infers TypeScript types or discovers functions.

First slice: Node 24.x on native Windows PowerShell and Ubuntu; implementation
must pin an exact supported patch and npm version in its lockfile/CI before
acceptance. Ordinary, synchronous user functions, zero to 32 positional arguments,
scalar inputs/returns, one process, explicit run scope, ESM `.mjs` / `.js` with
`type: module`, or CommonJS `.cjs` / `.js` with `type: commonjs`. A dual ESM/CJS
package must resolve to the SAME recorder state, not two independent singletons.
Use application-owned recorder instances rather than import-time initialization.

TypeScript is build-first: a pinned `tsc` build emits JS using NodeNext module
resolution, target ES2022 or newer, preserving async functions. Run emitted JS;
`.mts` -> `.mjs` and `.cts` -> `.cjs` profiles need explicit fixture coverage.
Do not claim direct `.ts`, tsx/ts-node, bundlers, transpiled generators, browser,
Deno, Bun, Electron, workers, cluster, native addons or serverless support.
The runtime's type stripping is not a full tsconfig-aware compiler; this proposal
intentionally avoids making it another supported build profile.

Support excludes constructors, decorators, getters/setters, proxies, generators,
custom thenables, bound/native functions, monkey-patched intrinsics and callers
that require original function identity/name/length/toString/stack shape.
Methods/receiver snapshots are deferred; an ordinary function's `this` still
passes through unchanged. Users must route calls through the returned wrapper.
Cached original aliases and recursive calls to an unwrapped lexical definition
remain invisible. Report this coverage limit; do not claim arbitrary coverage.

## Mechanism alternatives

| Mechanism | Feasibility and cost | Decision |
| --- | --- | --- |
| Explicit wrapper at definition/export | Works with ESM and CJS without rewriting imports. Source edits and changed function identity are visible; private lexical calls need explicit routing. | First slice. |
| CJS preload/export replacement | Early `--require` setup can initialize telemetry and replace selected mutable exports, but cached/destructured aliases and module-local calls bypass replacement. | Defer; not equivalent coverage. |
| ESM loader/source transform | Loader hooks can transform source before evaluation; initialization must precede target imports. Imported bindings/namespace objects are not a general writable function registry. Rewriting needs parser/source-map/version policy and can change semantics. | Defer. |
| OTel automatic library instrumentation | Useful for supported HTTP/database libraries, not arbitrary function scalar values. SDK/provider ownership must be coordinated. | Separate integration; no automatic dependency added. |

The recommendation is an engineering inference from documented module semantics,
not a tested interception result. Official references:
[Node module hooks](https://nodejs.org/api/module.html#customization-hooks),
[ESM interoperability](https://nodejs.org/api/esm.html#interoperability-with-commonjs),
[OTel JS instrumentation](https://opentelemetry.io/docs/languages/js/instrumentation/).

## Behavior and async stopping point

A normal non-async wrapper uses Reflect.apply with the original receiver and
arguments exactly once. Capture only declared primitive positions before invoking
the target. Return the exact original result synchronously; rethrow the exact
thrown value, including primitive throws. Recording failures must not replace host
results/errors. Do not read error message, stack, constructor or user properties;
record only a fixed thrown/not-thrown indicator. No blanket catch around target
execution that mistakes a host error for a recorder failure.

If a sync-declared target returns an object (including a Promise), mark output
unsupported and return it untouched. Never read `.then`, call toJSON, enumerate
properties or add a rejection handler. Such calls record the synchronous boundary
only, not asynchronous completion, duration or success. Async functions must not
be registered as supported sync functions. Do not inspect hostile function objects
in an attempt to prove their runtime category; declarations are trusted config.

A later, separately authorized `wrapAsync` may observe declared native async
functions returning native Promises. It would preserve fulfillment value/rejection
identity but must return a chained Promise, changing Promise identity, microtask
ordering and potentially unhandled-rejection behavior. Returning the original
Promise while attaching observers also changes rejection observability. Neither
path is transparent; reject Promise-identity/rejection-sensitive callers until a
specific contract is accepted. Do not implement async capture in the first slice.
[Node Promise type detection](https://nodejs.org/api/util.html#utiltypesispromisevalue)
does not solve these semantic differences or make custom thenables safe.

Use AsyncLocalStorage.run with a fresh context per run/call, never a process-global
current run or enterWith. Shared run ledger/call counters are allocated at entry;
parent call context is immutable and restored on return/throw. Node callbacks
created inside a scope can inherit its context even after synchronous return:
close the ledger on withRun return and reject late descendants as `run_closed`;
never attach them to a new run. Async entry points are unsupported in this slice.
A future async run scope must await declared completion and define detached-task
policy explicitly. Callback/custom scheduler context loss yields unknown coverage,
not guessed parentage. See [AsyncLocalStorage run](https://nodejs.org/api/async_context.html#asynclocalstoragerunstore-callback-args).

## Reuse of the existing contract

Read [provenance-spec](provenance-spec.md),
[Python probe contract](../adapters/provenance-python/README.md),
[synthetic privacy gate](p0-capture.md) and
[report walkthrough](report-walkthrough.md) as current authority. Preserve their
unknown semantics; no Go migration, new endpoint or runtime-specific ranking.

| Contract | Proposed Node encoding and required equivalence |
| --- | --- |
| Run/root | Random nonzero 16-byte trace ID per explicit run, root span ID `0000000000000001`; root/child timestamps and error status follow existing Python trace-root run grouping. Do not invent HTTP outcome or declare ordinary output defects operational failures. |
| Call identity | Unique nonzero 8-byte span IDs within trace, excluding reserved manifest ID. Use collision-free bounded per-run allocator; call IDs are allocated on entry and never reused after capture loss. Per-probe invocation index is not a total concurrent execution order. |
| Decision | Event `signalscout.decision`, `signalscout.record.version=1`, kind/id/json, step/output/selector attrs, synthetic marker. JSON contains version/id/run_id/profile/step/output/inputs/chosen/alternatives/rule/config_refs/verdict/reason_code/complete matching existing decoder. |
| Values | Inputs have explicit `name`, opaque run/call-qualified `ref`, `capture`; only allowed primitive fields have `inline`. Output uses unique run/call-qualified ref. Missing/off/unsupported is never zero, null or an invented empty string. Actual allowed null remains a value. |
| Unknown semantics | Generic call has no inferred domain choice/rule. Use fixed nonsemantic chosen option `returned` or `threw`, empty alternatives/rule/config refs; document these as boundary markers, never rationale. Error has output capture off. |
| Lineage | Same version-one events/attrs; one declared argument-to-output edge per supplied input, meaning boundary assertion only. Do not infer shared value identity across calls. |
| Evidence | Native `span:TRACE/SPAN` and stored event hash/ordinal resolve through existing readers. Logical decision/value refs are not native evidence URLs. Do not manufacture Go event hashes in JS. |
| Analysis | Existing trace, relations, explicit cohorts, invariants and report verifier consume the decoded fields with declared profiles/selectors. No runtime name boosts, new scoring, value linking or first-divergence claim. |
| Completeness | Port version-one manifest/scope wire contract exactly from capture_manifest.py, including root/probe buckets, reserved span IDs, epoch/scope population and final tail. Independent fixture equivalence must establish existing Go completeness readers accept it. |

Set SDK limits above declared bounded envelope/event limits or count any truncation
as detail loss. Timestamp nanoseconds travel as OTLP integer strings/bytes, never
JS Number arithmetic beyond safe precision; use BigInt internally for timestamps
and counters, not captured values. Use wall-time origin plus monotonic elapsed
clock; timestamp order is not proof of causal/data dependence.

`complete` describes declared boundary fields only, never all program state.
Require both allowed inputs AND supported/allowed output, no detail loss and no
sampling for a Node complete boundary; transport manifests remain separately
required. This is more conservative than Python's legacy inputs-only complete
flag and must be covered by equivalence tests, not silently interpreted as a
full graph. Read/probe compatibility does not imply byte-identical record JSON:
compare normalized meaning with declared IDs/times, and pin native fixture bytes
where transport identities are fixed.

## Default-off capture and privacy

Two gates: explicit per-function argument-position/output allowlists AND a
versioned field policy. With either absent, capture no values. No environment,
argv, paths, source text, exception text or free-form logs in telemetry. Trusted
static step/field metadata is length bounded and policy checked. Profile/field
names are not a secret detector. Resource/scope/service metadata is declared,
not copied wholesale from the host environment.

Initial allowed values: null, booleans, finite numbers (safe integers only when
integral), and exact approved short nonpersonal string enums. Strings have a
256 UTF-8-byte cap; at most 32 inputs, 8 KiB envelope, bounded names/config/probes.
Reject bigint, symbol, undefined, NaN, infinities, unsafe integers, arrays, nested
objects, functions, boxed primitives and proxies without coercion/introspection.
Keep -0 handling explicit in fixtures: JSON collapses it to 0, so mark -0
unsupported rather than claim sign preservation. No fallback stringify/repr,
truncation, plain secret hash or arbitrary artifact URI. Reject oversize before
retaining values; estimate string length bounds before allocating UTF-8 buffers.
Policy rejection emits only a bounded reason code and unknown/off marker.

Sanitize before serialization, queueing, worker messages and diagnostic output;
only bounded sanitized snapshots cross to exporter. No original target objects
survive a call. Strict intake still sanitizes before raw/indexed storage.
Existing P0 drops identity/semantic metadata and blocks metadata writes, so strict
P0 mode does NOT promise usable provenance/cohort parity. Initial useful synthetic
conformance uses declared synthetic transport mode plus producer policy; test
strict P0 separately and report its unknown/stripped results honestly. Any widening
of intake policy is a separate reviewed task. HMAC, recursive structures, real
personal data, consent and production retention/access remain out of scope.

## Export, accounting and bounds

Prefer a private, explicit OTLP/HTTP traces transport in one exporter worker,
reusing the Python wire contract, rather than silently registering another global
OTel provider. Application SDK ownership remains external. No auto-instrumentation,
provider replacement or duplicate trace export. Existing external parent trace
bridging is deferred: a Node adapter run is its own root, and this limitation must
be visible. Dependency/version choice for OTLP encoding is pinned at implementation;
handwritten protobuf is not authorized by this proposal.

Proposed initial ceilings: 32 probes; 1,024 live ledgers; depth 64; 256 retained
spans and 4 MiB sanitized payload, queued PLUS in-flight; batch 32; one HTTP request
in flight. Reserve before enqueue/worker postMessage; worker message backlog is
part of retained memory, not another unbounded queue. At most 256 manifest outbox
items with existing scope chunk/population caps. Reject new admissions without
waiting on capacity. Bounded CPU serialization remains on the caller; HTTP and
protobuf batch work stays off the application path. Object/worker overhead and RSS
must be measured separately from byte estimates. No hard real-time claim.

Manifest counters retain seen/sampled_out/selected/serialized/admitted/rejected/
ack/failed/unknown/detail_loss. Root and manifest delivery have separate accounting.
Sampling is not offered initially: every supported invocation is selected.
Later every-N sampling must choose before inspecting values, preserve denominators,
mark incomplete, and never imply skipped values were absent. Diagnostic summaries
are capped; aggregate counts are exact within an explicitly checked integer bound.
Overflow refuses capture and marks scope unknown instead of wrapping counters.

ACK means intake HTTP durable acknowledgment, not queue acceptance. A timeout
following possible commit is unknown; never count it as confirmed drop or retry
with new IDs. Initial implementation should use one attempt/no automatic retries;
503/non-ACK/worker failure are visible, never swallowed into success. Adding Python's
bounded stable-ID retry is separate work after equivalence/fault tests.

Explicit `await recorder.close({deadlineMs: 2000})` stops admissions, drains spans,
then final manifests/tail within ONE deadline, joins/stops the worker and emits a
bounded sanitized summary. Validate deadline 0..60,000 ms. No network/capacity wait
inside target calls; close is explicitly blocking asynchronous lifecycle work.
Exit/crash/SIGKILL/forced process.exit can lose data; no async exit-hook success
promise. Missing final tail/worker termination yields unknown. Target exceptions
and nonzero exit status take priority over telemetry status. Proposed fixture
runner statuses align with Python: zero capture 4, coverage gap 3, delivery loss 5,
otherwise 0; target errors keep their value/status. Library returns health summary,
never changes application exit status on its own.

## Proposed PowerShell workflow (not implemented or verified)

These exact paths/package/API/fixture names are implementation acceptance targets,
not commands available in this PR. Implementation must deliver the referenced
files, package-lock and fixture entry point before claiming repeatability.
Prepared prerequisites: pinned Node 24/npm, Go/Task, clean synthetic checkout.
No published npm package, global install, npx download, NODE_OPTIONS loader or
application source auto-edit is proposed.

```powershell
Set-Location C:\Users\yiche\telemetry-project
task build
npm ci --prefix .\adapters\provenance-node
npm run build --prefix .\adapters\provenance-node
New-Item -ItemType Directory -Force .cache\node-synthetic | Out-Null
# Window 1; this fresh database path must not already exist:
.\bin\signalscout.exe --db .cache\node-synthetic\capture.db serve --retention 0
```

```powershell
# Window 2, after intake reports ready:
Set-Location C:\Users\yiche\telemetry-project
# Fixture imports wrapSync/withRun; config fields default OFF unless declared.
# Fixture owns close in finally, preserves target errors, writes bounded health JSON.
node .\adapters\provenance-node\dist\fixtures\scalar-main.mjs --config .\adapters\provenance-node\fixtures\scalar-policy.json --endpoint http://127.0.0.1:4318/v1/traces --health .cache\node-synthetic\health.json --seed 101
.\bin\signalscout.exe --db .cache\node-synthetic\capture.db runs list
```

Fixture config declares version 1, synthetic profile, fixed step ID, positional
names `amount`/`rate`, capture args `[amount, rate]`, output off by default; its
explicit test variant permits numeric output. No YAML/source inference. Doctor
initially reports exact Node version, config/schema validity, expected wrappers
registered versus invoked and unsupported profile. Do not make existing Python
doctor checks imply Node readiness. A later additive Node preflight may inspect
trusted package metadata but must not execute an arbitrary target on a dry check;
imports/build tools can execute code and must be described accordingly.

## Implementation gates and stopping points

1. Owner GO authorizes only scalar sync adapter + synthetic fixtures, locked
   dependencies, docs and tests. Stop before automatic interception or async.
2. Paired deterministic plain/wrapped fixtures on Windows/Ubuntu: same stdout,
   values/identity, receiver, mutations, thrown primitive/object identity and
   target exit. Detect no new target invocation, serialization hook or getter.
   Unsupported return values pass through intact with visible omissions.
3. Contract conformance: independent Python/Node synthetic producers with fixed
   IDs/times and same scalar profile resolve native anchors; declared relations,
   trace/cohort/invariant/report outputs agree in meaning. Unknown/off fields,
   missing output, unsupported value and strict P0 stripping stay unknown.
4. Capture: 100 explicit runs x 4 calls, nested and sequential variants; exact
   expected root/decision/lineage counts, 0 drops/failures/unknown delivery,
   complete scope tail, valid per-run accounting and stored-value equality.
   Force queue full, oversized envelope, worker death, unreachable intake,
   503 and timeout-after-commit: bounded memory, honest failed/unknown counts,
   no false complete manifest, host outputs preserved. No fixture-specific tuning.
5. Privacy: planted secrets at every disallowed argument/output/metadata/error
   location; inspect sanitized worker payload, diagnostics and every populated
   stored cell including raw OTLP/record JSON. Zero leaked planted strings.
   Unknown policy/schema fails closed before capture. Do not call this production
   privacy; unrelated application logging and process memory remain external.
6. Scope tests: nested calls, recursive wrapped path, exception restoration,
   two ALS contexts scheduled to interleave synchronous calls, late callbacks to
   closed scopes, unsupported async run, and cached original alias. Assert exact
   ownership/no cross-run bleed and report bypass/late omissions. Async completion
   correctness is NOT passed by these tests. A separate async proposal must first
   settle Promise identity/rejection semantics, then test interleaved fulfillment,
   rejection, cancellation and detached work before support is claimed.
7. Lifecycle/provider tests: one worker/transport, idempotent close, second recorder
   isolation, mixed ESM/CJS import of same instance, existing application provider
   untouched, no duplicate adapter spans, explicit close timeout/worker termination
   and target-failure precedence. No forced-exit completion guarantee.
8. Cost pre-registration before measurements: identical scalar payload and call
   count, five paired plain/wrapped repetitions at 60 calls/sec and tight loop,
   reachable/unreachable intake. Report caller p50/p99, total process CPU/RSS,
   throughput, bytes/record, queue peaks/drops and separate startup/flush time.
   Retain original provisional 100/250 us and owner-adjusted reachable-60-Hz
   330/830 us reference lines as historical Python budgets, not a Node pass claim.
   Node release cost budget must be owner-approved BEFORE timing; until then
   publish measurements as unqualified, never move thresholds after results.
9. Standing Go/Python checks and frozen lane-01 DB-copy 15/15 byte equivalence;
   no new Go analysis, migration or frozen hash changes. Windows/Ubuntu CI green.
   If any gate fails, keep the slice experimental and report the failure.

Only after this checkpoint may the owner consider async support, richer values
or integration ergonomics. Independent difficulty-gated bug evaluation remains
separate; adapter mechanics and synthetic fixtures prove neither diagnosis nor
causation. No LLM, scanner ranking, source rewrite or evaluation-project access.

## Open implementation decisions

Pin Node/npm/TypeScript and OTLP encoding dependencies in the implementation PR;
verify licensing/support and OTel compatibility there. Confirm dual package state
and worker accounting with a small first checkpoint. Seek a pre-measurement Node
cost budget; retain all misses. Promise semantics, shared external trace/provider
bridging and production privacy remain explicitly deferred, not hidden blockers
for the synchronous synthetic slice.

Additional primary build references:
[Node TypeScript limits](https://nodejs.org/docs/latest-v24.x/api/typescript.html),
[TypeScript NodeNext behavior](https://www.typescriptlang.org/docs/handbook/modules/reference.html#node16-node18-node20-nodenext),
[TypeScript source maps](https://www.typescriptlang.org/tsconfig/sourceMap.html).
Compiled code_ref identifies emitted revision/path/symbol; optional original TS
location must be declared and map-verified. Source maps aid debugging, not runtime
function interception or evidence of a code read. Do not export source map source
content/absolute paths or fetch remote maps. Initial mapping may remain unknown.
