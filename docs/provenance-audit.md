# Provenance feasibility audit (current M4b; no engine changes)

## Current representation

| Layer / source | What works | Loss or missing behavior |
|---|---|---|
| internal/intake/decode.go: attributes, anyValue, canonical, decode | Ordinary OTLP span events/logs retain scalar/array/kv-list attributes and canonical raw payloads; trace/span correlation. | Byte attributes become hex in searchable attrs (raw remains); duplicate keys collapse in the map; logical record IDs and JSON envelopes are opaque, not validated. |
| internal/store/migrations/001_initial.sql and internal/model/model.go | signals JSON plus raw OTLP retains decisions/edges as event/log records, stable native evidence IDs, resource/config snapshots. | No semantic decision/edge/annotation tables or logical-ID resolver, referential integrity, append-only label enforcement or graph index. Generic signal upsert is not an annotation audit log. |
| internal/runs/group.go: Identity, Observe; internal/runs/outcome.go | One trace gives one unit and records nested spans/events/logs. | Trace-only units classify as web; explicit resource run ID forces ML. Outcome is operational, not human quality. Late label logs extend run evidence interval; label time is not independent. Session links are opaque attrs. |
| internal/telemetry/query.go and internal/store/telemetry.go | Indexed service/time or trace/time span/log browsing with explicit bounds/limit and one flat attribute filter. | No event search kind, JSON-envelope field query, graph traversal, multi-label cohorts or artifact fetch. Nested JSON is inspectable text, not first-class searchable structure. |
| internal/viewer/templates/run.html, internal/viewer/templates/explorer.html; internal/viewer/explorer.go | Span tree, event markers, escaped attrs; trace-correlated logs show full record JSON; flat selectors filter logs. | Logical refs are plain text, no clickable lineage traversal or mark/annotation UI. Owned label logs require log explorer, not a quality badge. Event parenthood is not derivation. |
| internal/diff/features.go, internal/diff/engine.go; internal/store/diff.go | Span latency/count/name presence and config/env differences; M4b context metric/log differences. | Decision verdict/reason/input attrs are not extracted as proposed features; labels do not select cohorts. Closed failing run and earlier passing baselines remain required. Repair count may be visible, but no general semantic diagnosis. |
| internal/redaction/doc.go; docs/adapter-guide.md | Placeholders preserve room for later work. | No production content privacy gate, helper, setup/coverage discovery or semantic profile contract. |

For exact native ID construction see decoder.add and decoder.decode: explicit
logical IDs live in attrs; changing a record's payload can produce another native
event/log ID, not a semantic conflict. Hash preservation does not enforce
producer honesty. External artifact refs remain opaque: raw data cannot be
reconstructed from a digest, and no artifact retrieval is added.

## Spike boundaries and test plan

Planned follow-up: two standalone generators under scripts/ emit OTLP collector JSON
using only the already approved protobuf types. These plain OTLP generators remain independent of the separately requested Python helper.
All inputs, review scores, reviewer labels and expected problems are synthetic.
One unchanged configuration per fixture supplies both problem/reference examples.

Planned tests will check generator determinism, genuine threshold/stale-input execution in
the synthetic calculation, actual loop structure, logical references, deferred
simulated labels, and plain protobuf JSON decoding. The proposed spike driver will run the
real signalscout serve binary against a fresh database, post ordinary OTLP,
open existing run/explorer pages over HTTP, verify all records through the
read-only pool and records limitations. It does not open another writer.

Current UI workflow is manual: identify a problem from the synthetic oracle or
a reviewer, find the trace, inspect decision attributes/envelopes, copy refs and
follow them through other events/logs; choose reference IDs externally and
compare values manually. There is no mark/cohort workflow or deterministic
invariant checker. Do not relabel a successful HTTP response as a failure to
make existing diff accept it.

Measured results, exact PowerShell commands and observations will be added after
the replay, including tests that explicitly demonstrate unsupported operations.
## Current executable status (2026-10-20)

S0a is now executed. Historical pending/fallback statements below describe the
accepted source audit, not today's executable status. S0b results follow below
when validated; no production milestone or engine change is implied.

- uv 0.13.3 provisions CPython 3.12.12; OTel 1.28.2 / instrumentation 0.49b2
  and all transitives are locked. Native Python 3.14 only bootstrapped uv.
- Primary capture uses external YAML/import probes; A source edits = 0, config
  = 11 lines. The independent Go generator matches every decision/lineage
  envelope field after normalizing generated run identity. No graph resolver
  or semantic completeness enforcement was added.
- Safety checkpoint passed 15 tests before breadth validation. Tests cover
  queue/cooldown, bounded values and metadata, unavailable/blocked exporter,
  invalid values, original exception identity, context cleanup, actual OTLP
  decoding, unsupported callables, default capture OFF and standard integrations.
- Cost budget MISSED. The unchanged budget is p50 <=100 us / p99 <=250 us.
  See [all 30 trials](../benchmark/results/s0a-probe-overhead.json) and the
  [generated cost/context/flush tables](../benchmark/results/s0a-probe-overhead.md).
  Reachable 60 Hz median trial p50/p99 = 252.20/818.49 us (0/5 pass); worst
  trial p99 = 1741.28 us. All three 60 Hz conditions miss. Only 4/30 trials
  across all modes meet both quantile budgets. Fast tight-loop drop paths do
  not establish affordable full capture.
- Reachable 60 Hz exported all 18,000 native items (9,000 decisions plus roots).
  Reachable tight loops exported 2,112/100,000 items and dropped 97,888 full.
  Unreachable exported zero; accepted-but-failed exports remain explicit.
  Saturation retains at most one queued item and reports all loss.
- The timed wrapper includes automatic run entry/exit, copying/serialization
  and enqueue; body time only is subtracted. Context and explicit flush are
  separately reported. Fixed four-scalar/two-returned-alternative envelopes
  average about 1.19 KiB, below 2 KiB. Returning alternatives was a benchmark
  fixture choice; probes cannot invent alternatives hidden in an ordinary body.
- Standard requests tracing bridges into the same bounded queue. Logging adds
  trace correlation only; log-body export is OFF. Existing providers are
  preserved, not silently replaced. Async/generator/descriptor/alias/reload
  support and arbitrary command injection remain unsupported.
- The experimental complete flag covers capture of boundary arguments only.
  It does not mean all arguments were read or all workflow dependencies seen.
  HTTP-client instrumentation does not automatically instrument a server.

### Historical source audit (preserved)

## Execution status: user-authorized partial boundary

This pass completes Parts 1–3 and this source-level audit, using the expanded
request's explicit fallback. No fixture generator, Python helper, new transport
test, replay driver, volume policy or record wrapper was built. No fresh A/B/C
store was populated or opened in the browser. No recording-call overhead,
instrumentation line count or unreachable-recorder behavior was measured.
These are pending, not zero and not passed. Existing repository tests are
regression checks, not proof of the proposed helper contract.
uv is not currently discoverable on PATH; native Python 3.14 is available.
This does not block the design. The next spike must provision/locate uv and pin
its Python/runtime dependencies before measuring.

### What can be concluded across A, B and C

| Behavior | Existing evidence / source-level feasibility | Not established in this pass |
|---|---|---|
| A: noisy frame decision + stale previous value | OTLP attrs/envelopes can encode input value/ref versions, alternatives, threshold and explicit edges. Separate traces can be frame units. | No generated/replayed frames or invariant result. Current grouping calls them web; no frame lineage traversal. A stale reference is not inferred from timing. |
| B: staged pipeline and human labels | Spans/events can encode stage/repair structure. Correlated logs can carry a simulated review annotation. | No real human labels, fixture or helper replay; no marking/annotation/cohort UI. Recorded retrieval does not establish relevance or roboticness. |
| C: existing pool-exhaustion shop | Prior executed M2/M3/M4b results exist: seed 1 has 40 baseline passes and 9/300 loaded failures; structural config/retry/acquire evidence and explorer navigation were validated. See benchmark/results/shop-demo-seed-1.json and m4b-summary.md. | No new replay or fresh-store validation here. Seed-1 gauge sampling gap and incidental context associations remain as previously reported. |

The current common intake/store can preserve these records in principle. That
is weaker than proving one complete recorder/diagnosis experience serves all
three. The current web/ML run rules and failure-only diff selector need the
proposed profile/label changes. No lane/coaching-specific server switch should
be introduced to hide these gaps.

## Loss and completeness details

- Raw means canonical decoded OTLP, not original compressed HTTP/wire bytes.
  Canonical records retain non-indexed OTLP data such as span links; the UI
  does not resolve them into semantic lineage.
- anyValue converts searchable bytes to hex and nonfinite numbers to strings;
  duplicate attribute keys collapse. JSON rehydration in store.readSignals uses
  float64 for generic numeric attrs, so integers beyond 2^53 should use string
  refs or typed envelope JSON. Original proto JSON remains available.
- Log template normalization deliberately replaces values. Use original attrs/
  envelopes for verdict/reason/value comparison, not template placeholders.
- Trace-owned logs do not enter M3 feature extraction; only spans/events do.
  Compatibility events views do not make every log a scored event.
- Observe expands a run's interval for late evidence. A later review log may
  therefore extend execution/context time; this is a source-level prediction,
  not a fresh measured replay result. Separate annotation storage must avoid it.
- Generic upsert, retention and raw copying do not enforce immutable semantic
  annotations, privacy policy or cross-run graph completeness.

## Minimal remaining spike, in order

1. Plain Go OTLP A/B generators with deterministic tests, explicit synthetic
   oracles, unreviewed examples, opportunities/rejection reasons and versioned
   previous refs. Keep their entire emitted one-second record count inside the
   measured shop envelope. Reuse C data, preserving original databases; export/
   replay ordinary original OTLP into a fresh serving store rather than copying
   accepted diffs and calling that a replay.
2. Minimal Python helper with record_decision and run context manager, no new
   server endpoint; re-express the same A/B logic and compare captured facts.
   Tests: bounded queue, overflow/drop visibility, server-down cooldown/no-op,
   invalid values, unavailable exporter, context cleanup and preservation of the
   host's original exception. No network/wait/join in record or context exit.
3. Measure five trials per mode on a documented native runtime: 60 calls/sec
   for 30 seconds and 10,000 tight-loop calls, reachable and unreachable.
   Report caller wall-time p50/p99 including bounded copying/serialization/
   enqueue, accepted/dropped/exported counts, queue limits and explicit flush
   time separately. Do not present a fast no-op/drop path as full-record cost.
   Count instrumentation lines from imports/setup/context/API-call source spans
   with a documented counting script, excluding original application logic.
4. Open real serve run/explorer pages for A/B/C. Assert stored envelopes/refs,
   inspect native anchors, try an HTTP-200 marked turn against the unchanged
   failing-run-only selector and report the expected limitation. Show manual
   marking/reference selection outside the product; do not fake an error status.
   Record absence and before/after examples with coverage; no new diff types.

Do not rush a safety-critical asynchronous helper merely to fill benchmark
cells. Report any call-budget miss without tuning the budget or engine constants.

## Smallest next task: S0a recorder-safety spike

Implement only the synthetic Python run context + record_decision helper and A
example, using uv and plain OTLP. Acceptance: default field capture off; explicit
synthetic fields survive a reachable intake; bounded queue has visible drops;
server-down, queue-full and invalid-value calls do not raise or intentionally
wait; original host exceptions propagate unchanged; caller p50/p99 are reported
against the pre-registered budget at 60 Hz and tight-loop rates over five trials.
Publish instrumentation LOC and accepted/exported/dropped counts. Keep any miss.
No claims about B/C breadth until separate S0b replay/browser validation completes.
This is a feasibility prototype, not P1 production onboarding or permission to
capture real personal data. P0 privacy remains required before that.
