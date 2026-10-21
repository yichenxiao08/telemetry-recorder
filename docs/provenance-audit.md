# Provenance feasibility audit (current M4b; no engine changes)

Current execution supersedes the historical pending plan: see **Current executable
status** and **S0b breadth findings** below. All examples/reviewers are synthetic;
M5–M8 and P0–P4 production work remain stopped.

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

S0a is executed; S0b findings are below. Historical pending/fallback statements describe the accepted source audit. No production milestone or engine change is implied.

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
- FIRST candidate cost budget MISSED. The unchanged budget is p50 <=100 us / p99 <=250 us.
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
  average about 1.19 KB (1.16 KiB), below 2 KiB. Returning alternatives was a benchmark
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

## S0b breadth findings (synthetic replay; no engine changes)

The replay driver is adapters/provenance-python/breadth.py. It launches the real
signalscout serve binary with a fresh database, default grouping/idle/context
settings and retention disabled to preserve the deliberately old C timestamps.
Only Go read-only tools inspect SQLite; Python sends ordinary OTLP/HTTP.
[Raw replay evidence](../benchmark/results/s0b-replay.json) and
[manual evidence examples](../benchmark/results/s0b-examples.json) retain exact
native IDs, inputs, outputs, selected references, counters and scope timestamps.

| Behavior | What a developer can inspect | Manual work and limits |
|---|---|---|
| A, frame loop | 60 frame runs, four captured arguments, returned lane/maneuver/used_previous_frame, argument-boundary edges and native event anchors. Go and probe envelopes match field by field after generated identity normalization. | Configure one function (11 lines), identify the suspicious frame from a declared expectation, inspect its event JSON, copy refs and pick adjacent reference frames. No automatic previous-version resolver, hidden threshold/alternative capture or invariant engine. |
| B, one staged core | 12 HTTP-200 turns, nested context/retrieve/select/draft/review/repair calls; repair loops, declines with/without evidence, input history and selected context. Four simulated reviews arrive separately; eight turns remain unlabeled/unknown. | Configure seven functions (26 nonblank lines). Find a run, inspect stages/events and log-explorer labels, copy output/native refs and choose reference IDs outside the product. No Mark UI, quality badge, cohort selection or human-quality inference. A client HTTP span is a separate trace; the stdlib server is not auto-instrumented. |
| B, optional semantic tier | Same replies, plus a declared first-retrieval selection rule and explicitly rejected candidate IDs. | Two added source/instrumentation lines (import + call), separate enriched copy; 26 nonblank / 27 physical config lines. This does not expose the outer context-limit/strictness policy. Producer assertions are not independent proof. |
| C, existing shop | Ordinary original OTLP replay preserves 1,960 spans, 376 embedded events, 1,029 logs and 692 metric points, all 4,057 native IDs. Config/env, acquire/retry structure and available host context remain inspectable. | Existing Go application instrumentation is reused; zero source changes in this spike does NOT mean C was automatically instrumented. Saved diffs are never copied. The saved source currently has 692 metrics versus 567 in the older report; export includes all 692, without inferring the reason for the discrepancy. Seed-1 gauge sampling gap remains. |

A/B source snapshots and AST/diff counters prove zero target edits for the plain
variants, not a hypothetical integration LOC estimate. B's optional two lines are
counted against the plain original. The injector itself changes zero source bytes
in either variant. Its setup/config is explicit; discovering useful function names
and capture fields remains developer work. Body capture defaults OFF.

### Absence, before/after, and coverage

- Manual absence A: frame 2 returns keep with gap 0.48 and the captured prior
  lane. There is no change maneuver in that observed output. This establishes
  an observed absence at that call, not why an arbitrary hidden branch was skipped.
- Stale state A: frame 42 receives previous_frame=40, while adjacent frame 41 is
  recorded. Given the fixture's declared N-1 expectation this is a suspicious
  supplied version. The probe cannot prove who wrote/read the stale value.
- Near-threshold A: adjacent observed inputs/returned lanes can be compared using
  the generated before/after examples. The hidden exact 0.5 threshold is NOT
  recovered by observing a few values. No rule or rejected option is invented.
- Legitimate decline B: turn 0 receives empty retrieval evidence and returns a
  decline, without a repair call. Repair was attached and observed in other
  turns; this absence has explicit scope (one captured turn, seven attached
  functions, zero reported loss). It does not establish global coverage.
- Successful repair B: turn 1 records draft decline -> review repair -> repair
  answer -> review accept. This within-turn before/after is not a causal estimate
  or a separate independent sample per stage. Turn 2 deliberately truncates
  context; that negative control must not be called an accidental context bug.
- Unrepaired decline B: turn 3 has supplied synthetic relevant-evidence IDs but
  still declines. Relevance and too-strictness are the fixture/reviewer's declared
  judgments; neither follows from a 200 status or document ID alone.
- C's before/after config example selects an actual earlier pool=20 pass and a
  pool=5 failure. This is an association with acquire/retry evidence, not proof
  that configuration caused the failure. No new experiment was secretly run.

Every example includes its native event/span evidence in s0b-examples.json.
No A/B calls were dropped in the reachable replay. There is one synthetic session
per fixture; frames/turns are correlated and the sample is too thin for population
quality rates or calibrated causal claims. Rules hidden in bodies, actual reads,
external state, multi-session generalization and real reviewer validity remain
unknown. Optional semantics improve coverage, not truth guarantees.

### Late reviews and existing diff boundary

After default idle finalization, separate simulated label logs use review time
60 seconds after their target run end. The measured too-strict run's ended_at
and exact-resource context-scope end both increase by exactly 60 seconds; its
start and operational pass outcome remain unchanged. The original span timings
remain execution evidence, but the run-level interval is contaminated by review
arrival. This confirms the source audit prediction. Independent annotation time
is a future P2 requirement, not fixed by special-casing this fixture.

POST /run/diff for that actual HTTP-200 passing turn returns 409, "Comparison
requires a closed failing run." We do not change its outcome to bypass the guard.
Operational comparison C remains the already implemented subset; labels and
boundary values do not become scored features or automatic explanations.

### Fidelity correction, retained misses and decision

The first replay exposed identical wall-clock timestamps on sub-tick Windows
calls. A new test with a constant wall clock initially fails with zero duration;
monotonic elapsed time anchored to UTC fixes probe/root durations and nested
order. The failed regression log, first replay, and both full 30-trial overhead
passes are retained. The first inspector attempt also failed on the legacy
source context schema; a later comparison harness attempt was stopped for
quadratic snapshot decoding. Those harness failures are retained and corrected,
not asserted to have passed. No engine, schema, scoring, timeout or assertion
was weakened.

The production CLI record/probe add/mark/explain and viewer marking remain
DESIGN ONLY, with shapes in provenance-spec.md. No arbitrary command injector,
privacy gate, label store, reference-cohort UI, graph resolver or causal agent
was built. Logging only adds context correlation, not automatic log-body export.

Riskiest assumption: call-boundary arguments/returns could be mistaken for complete
semantic provenance. A supplied argument edge does not prove an actual read, and
operational success does not validate a review label. Current capture must be
presented as scoped observations with unknown internal dependence.

Recommendation: NO-GO for advancing on the claim that the recorder is cheap and
production-ready. The fixed cost gate has misses and P0 privacy is unimplemented.
Any P0 implementation requires separate authorization and an explicit decision
about the retained cost/coverage gaps; this spike does not start it.

One unchanged Go core stores A, B and C and serves their ordinary evidence.
A/B need zero target changes for useful boundary observations; B needs two optional
lines for otherwise hidden selection semantics. C reuses existing OTLP rather
than demonstrating Go zero-code injection. No behavior-specific server path was
needed. This supports representational breadth, not automatic diagnosis of any
codebase, cheap sustained capture, causal truth or production privacy.

## Corrected-duration candidate: final cost result

[All 30 final trials](../benchmark/results/s0a-probe-overhead-monotonic.json) and
[generated quantile/context/flush table](../benchmark/results/s0a-probe-overhead-monotonic.md)
use the SAME fixed payload, seeds, rates and 100/250 us budget as the first pass.
This rerun follows a timestamp-fidelity correction, not a budget or scoring change.
Do not attribute all timing variation to that correction; no CPU profile isolates it.

| Condition / rate | Median trial p50 / p99 us | Trials within budget | Exported / attempted native items |
|---|---:|---:|---:|
| reachable / 60 Hz | 205.15 / 390.31 | 0/5 | 18,000 / 18,000 |
| reachable / tight | 43.30 / 129.10 | 5/5 | 1,888 / 100,000 |
| unreachable / 60 Hz | 197.45 / 593.31 | 0/5 | 0 / 18,000 |
| unreachable / tight | 42.90 / 135.20 | 5/5 | 0 / 100,000 |
| saturated / 60 Hz | 272.15 / 708.41 | 0/5 | 0 / 18,000 |
| saturated / tight | 42.90 / 135.31 | 5/5 | 0 / 100,000 |

Reachable tight loops drop 98,112 full; unreachable 60 Hz drops 11,796 during
cooldown and fails all 6,204 accepted exports; unreachable tight loops drop
98,560 full and fail 1,440 exports (no cooldown calls in those fast trials).
Saturation accepts one pending item per trial, dropping the other 17,995/99,995
items; exporter is deliberately disabled and flush times out as expected.
A native item is a root or decision span, not a decision-only count.

Context entry/exit median p50 at reachable 60 Hz is 39.30/3.50 us; worst trial
p99 is 147.51/9.30 us. Explicit flush maximum is 0.091942 s there and 0.498532 s
in reachable tight loops. The saturated 0.01 s deadline returns within 0.025612 s
on this OS (scheduler granularity is not a hard real-time guarantee).
Full per-trial counts, context quantiles, failed exports and payload bytes are
retained in JSON. Safety checkpoint passes 19 tests. Cost acceptance FAILS:
all 60 Hz trials miss, even when delivery succeeds. Fast loss paths are not
full-capture throughput. No target, timeout, assertion or scoring constant moved.

Measurements cover the fixed probe payload and its automatic run context, not
end-to-end HTTP instrumentation cost or all B stages as one transaction. Those
costs are unmeasured, not assumed equal. Absolute cross-process clock order and
real reviewer validity also remain unestablished. The replay keeps the current
web/ML run classifications; no lane/coaching-specific engine behavior was added.

## Owner budget decision — 2026-10-20 (after results)

The owner replaces the reachable-60-Hz budget with p50 <=330 us / p99 <=830 us,
representing approximately 2%/5% of a 16.7 ms frame. This budget was set AFTER
seeing the measurements; it is not a pre-registered success against 100/250 us.
The corrected-duration reachable trials all meet 330/830 us (median trial
205.15/390.31 us; worst p99 399.50 us). Original 100/250 us misses, original
candidate trials, unreachable/saturation losses and all raw results remain.
The first-candidate reachable trials meet the new budget in 3/5 cases: all p50 values pass; p99 fails in trials 1 and 5. This decision authorizes a synthetic Slice 1 only, not P0
privacy work or M5–M8. No new overhead measurement is needed for this budget-only
change; any future recorder path change requires measurement again.
