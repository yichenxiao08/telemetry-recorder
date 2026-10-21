# Recorder capture-loss fix: Stage 3–4 protocol

Synthetic development/toy data only. Owner approved GO FIX after Checkpoint A.
The earlier findings and every unsuccessful measurement remain in
[the Checkpoint A report](recorder-capture-loss.md). Reserved evaluation stays
closed and incomplete; none of its files, seeds or results are rerun or changed.

## Fixed policy, before validation

Queue capacity 256, socket timeout 0.5 s, maximum batch 32 and OTLP encoding
are unchanged. One additional attempt, 50 ms worker backoff, and a 1.25 s total
logical batch deadline are fixed before the ten development repetitions.
Encoding errors and terminal HTTP rejections are not retried. Cooldown waits
on the worker, preserves queued spans, and admits callers while capacity remains.
Outages can still cause visible queue-full drops. No host path network, queue
capacity wait, flush or join is introduced.

The delivery worker uses one daemon network worker and one operation slot.
A socket/DNS operation that outlives the total deadline is quarantined rather
than spawning replacement threads. It may outlive the logical deadline; queued
data waits in bounded memory until it returns. The expired batch stays unknown,
even after a late reply. This bounds retries, threads and memory without claiming
Python can forcibly cancel a blocking OS operation. Context exit never joins.

Explicit timeout-bounded flush returns cumulative producer counts:
`attempted = acknowledged + failed + dropped + unknown`. Pending spans and
ambiguous persistence are unknown. A drained queue alone is not completeness;
the report is truthy only when drained with zero failed, dropped and unknown.
Attempt diagnostics count each network attempt, separately from producer spans.
Legacy `export_failed` counts final unsuccessful spans, not recovered retries.
Legacy cooldown-drop/suppression counters remain available and now stay zero.

## Predeclared validation

Run all 12 development cases (101/103, six frozen variants) ten times through
the unchanged driver, each repetition in a fresh real-intake store. Preserve
all raw reports, diagnostics, timelines, statuses and failures. Do not retry a
failed repetition invisibly. Compare full scores/counts against Checkpoint A;
verify frozen lane-01 JSON/text bytes separately on all 15 existing cases.

Report these named acceptance tests separately:

1. `test_cooldown_does_not_discard_queued_batches`: retain 40 admitted items
   through cooldown, admit another, then acknowledge all 41 without any drop.
2. `test_duplicate_delivery_is_idempotent_after_lost_ack`: delay a reply after
   real intake commit, retry identical bytes/IDs, verify two commits leave one
   stored span and one event, and acknowledge the producer span only once.

Use `validate_capture_fix.py costs` for five trials per condition/mode of the
unchanged fixed-payload S0a harness: reachable, unreachable, saturated; 60 Hz
for 30 s and 10,000 tight-loop calls. Each trial is a separate process/store,
so outage workers cannot affect subsequent trials. Pair Checkpoint A and fixed
recorder sources, alternating order by trial parity, without concurrent load.
Payload, probe, timeout and queue settings are identical. Retain every miss
against p50 <=100 us and p99 <=250 us; report context enter/exit and flush
separately. Saturated capacity1 intentionally has no worker and must drop.
After these serial trials, repeat the unchanged Checkpoint A toy slow-ack
control (first response delayed 0.65 s after real commit) and capacity256 burst
control (1,000 tight calls, first response delayed 0.30 s). Preserve both raw
reports and logs, including any expected relay client-disconnect traceback.
These controls do not replace the five-trial cost harness or tune capacity.

Earlier diagnostic tests that explicitly required cooldown discarding or
truthy flush after failed delivery were replaced with stronger preservation
and completeness assertions. Counter-category unit tests disable retries to
isolate one attempt; dedicated tests enforce the release policy's retry bound.
Initial failures are retained in validation artifacts, not hidden.
Report boundaries explicitly copy `dict(flush_report)` before JSON encoding:
Python's indented encoder otherwise treats a false-valued dict subclass as
empty. The existing S0a/diagnostic harnesses receive only this serialization
adaptation; call loops, fixed payload, timing, pacing and transport are unchanged.
The frozen lane-02 driver is untouched. This adaptation was added before the
paired cost trials; earlier clean development flushes were truthy and retained
all their fields. A regression test covers both complete and incomplete reports.

Measured results follow below. No scoring, engine, migration or later milestone
work is included.

## Proposed future zero-drop gate — design only

Before authorizing any new reserved evaluation, require **ten consecutive full
development repetitions**, each all 12 cases at the registered 60 calls/sec
method on the declared serving defaults and machine/software fingerprint.
Every case must serialize exactly 800 decisions / 4,800 argument captures,
attempt 1,600 producer spans, and finish its explicit flush with 1,600
acknowledged, zero failed, zero dropped, zero unknown and zero invalid captures.
The four-state identity must hold per case and repetition. All analytic
expectations, unchanged-score regressions, evidence links and protected hashes
must pass. Report observed rates, phase durations and queue high-water marks;
never substitute the intended rate for the achieved rate.

Retain injected outage, saturation, slow-acknowledgement, cooldown-preservation,
duplicate-idempotency and bounded-deadline safety controls separately. Expected
drops in an explicitly saturated control must be exact and visible; that control
cannot qualify as a clean development repetition. A false completeness report
must never be used as a zero association. Keep the separate five-trial
100/250 us overhead requirement and report it independently: a clean transport
streak does not waive a cost miss.

On any unexpected loss, unknown persistence, mismatch or safety failure: retain
the fresh store/report and all attempted/unattempted statuses, stop that proposed
evaluation, reset the qualifying streak, investigate before a new streak, and
obtain owner authorization for any new reserved attempt. Never delete a lock,
replace a seed, silently rerun a case or alter a budget. This is a proposed gate,
not a lane-03 contract, not approval to create one, and not a zero-loss guarantee
under arbitrary outages. The closed lane-02 evaluation remains incomplete.

## Stage 3–4 measured results

Implementation: `a8453cb9104adf5ed9e2946a107cfef46b3ea414`. The scheduled run
was deleted at the owner's request and this work ran immediately. The recorder
source stayed unchanged throughout measurement (SHA-256
`6605df842c639f57397e44a4c5457b886dc21eaf63fb4a8336bb5365953d5f01`).
No timeout, queue, score, generator, target or serving configuration was tuned.

Full [reviewed results](../benchmark/results/recorder-capture-loss-fix-reviewed/summary.md),
[all 120 cases](../benchmark/results/recorder-capture-loss-fix-reviewed/cases.md),
[all 60 overhead trials](../benchmark/results/recorder-capture-loss-fix-reviewed/overhead.md)
and raw gzip archives retain every result and miss. An initial, over-strict
fresh-report comparison is preserved separately in
[the first publication](../benchmark/results/recorder-capture-loss-fix/summary.md).
No workload was rerun to repair that comparison.

### Root cause and the two explicit acceptance results

**The natural reserved-run initiating failure was not reproduced.** Five old
and ten fixed development repetitions are clean. The evidenced mechanism is
the induced slow-acknowledgement timeout followed by cooldown discard/drop
amplification; it is not proof of a SQLite or Defender initiating cause.

| Acceptance | Result | Evidence |
|---|---|---|
| Cooldown no longer discards queued batches | PASS | 40 items remain queued during controlled cooldown; another is admitted; all 41 acknowledged, zero failed/dropped/unknown. No send occurs before cooldown is released. |
| Duplicate-delivery idempotency | PASS | First real commit's reply is delayed beyond the client timeout. Two requests have identical bytes/IDs and receive real intake acknowledgements. Store has one span plus one event, never four signals; producer acknowledgement count is one. |

These are separately named tests and separately saved acceptance records, not
inferred from a successful ordinary run. Span/event idempotency does not assert
log or metric exporter idempotency. Retry, encode/rejection, sustained outage,
saturation, stuck network, bounded threads, exact flush and host-behavior tests
also pass. The stuck-operation test checks that a late reply never credits the
expired batch, even after later queued batches succeed.

### Ten development repetitions

All **120/120 cases** complete: **192,000 attempted = 192,000 acknowledged +
0 failed + 0 dropped + 0 unknown**. Every repetition has 12 complete cases and
19,200 acknowledged spans. No invalid captures or final unsuccessful exports.
Queue high-water marks by repetition: **22, 24, 26, 26, 52, 40, 26, 10, 12, 12**
against capacity 256. Per-case accounting and every cost are in the linked table.

Measured call-phase rate is **60.113–60.519 calls/sec**, duration
**13.219–13.308 s** for 800 calls. This observer interval starts at the first
run context and ends before flush; the frozen driver schedules 60 Hz before
generator preparation, so its initial catch-up can slightly raise the observed
phase rate. Intended rate is not substituted for measurement. High-water marks
are cumulative within a repetition, not reset per case.

All **120 development wrappers miss** 100/250 us: p50 **152.800–331.950 us**,
p99 **330.632–2,080.013 us**. These observed manual-context costs are distinct
from the fixed-payload S0a trials below. Target changes remain **0 source lines**,
instrumentation **0 lines**, config **10 lines**. Source/generator/profile hashes
and the reserved-seed refusal test remain intact.

### Five-trial cost comparison, unchanged 100/250 us budget

Medians below are medians of five trial quantiles, not pooled calls. Each pair
uses the same payload and harness, alternates old/fixed order and has a fresh
process/store; no measurement workloads overlap. Python is uv-pinned **3.12.12**.
Four scalar inputs, two returned scored alternatives, mean JSON envelope
**1,188.714–1,189.778 bytes**, below 2 KiB. This is envelope size, not total OTLP
wire bytes including root spans/lineage. All payload/source hashes are saved.

| Condition / mode | Before p50 / p99 us | Fixed p50 / p99 us | Fixed passes / 5 |
|---|---:|---:|---:|
| Reachable, 60 Hz for 30 s | 206.900 / 403.825 | 206.600 / 404.555 | 0 |
| Reachable, 10,000 tight calls | 45.000 / 143.606 | 44.400 / 142.404 | 5 |
| Unreachable, 60 Hz for 30 s | 217.600 / 978.697 | 274.450 / 815.099 | 0 |
| Unreachable, 10,000 tight calls | 44.200 / 131.705 | 44.000 / 126.205 | 5 |
| Saturated, 60 Hz for 30 s | 348.250 / 835.211 | 339.450 / 836.224 | 0 |
| Saturated, 10,000 tight calls | 44.400 / 133.210 | 44.000 / 137.303 | 5 |

The fix leaves observed reachable medians close to the paired baseline. In the
unreachable paced condition p50 increases **56.850 us** and p99 decreases
**163.598 us**; admission now continues capture/serialization during cooldown
instead of discarding early. This is a policy/cost trade-off, not evidence of
free recording or a controlled attribution of every OS timing fluctuation.
Every paced trial misses at least one limit. Tight-loop passes coexist with
large queue drops; do not use those passes to claim reliable high-rate capture.

Context cost is measured separately (medians of five trial quantiles):

| Fixed condition / mode | Enter p50 / p99 us | Exit p50 / p99 us | Explicit flush median s |
|---|---:|---:|---:|
| Reachable, 60 Hz | 36.800 / 79.174 | 4.000 / 10.800 | 0.000066 |
| Reachable, tight | 2.700 / 9.100 | 1.800 / 5.400 | 0.307275 |
| Unreachable, 60 Hz | 46.300 / 146.008 | 6.300 / 19.601 | 10.010393 |
| Unreachable, tight | 2.700 / 7.900 | 1.800 / 5.100 | 10.012215 |

Saturation context/flush costs and every individual trial are in the results.
The explicit outage flush reaches its 10 s wait deadline; observed wall time
includes scheduler/reporting overhead. Context exit does not wait for export.
Preserving queued spans deliberately lengthens outage flush compared with
discarding them. Its false result and unknown counts are retained.

### Drops, failure states and unchanged toy controls

All fixed-trial four-state identities balance and all counts are nonnegative.
Each fixed paced reachable trial has 3,600 acknowledgements with no loss. In
each of the five paced unreachable trials: **3,600 = 0 ack + 578 failed +
2,894 dropped + 128 unknown** at explicit flush. Each tight unreachable trial:
**20,000 = 0 + 160 + 19,712 + 128**. Capacity1/no-worker saturation gives
**3,600 = 0 + 0 + 3,599 + 1**, or **20,000 = 0 + 0 + 19,999 + 1** for tight
calls. These are expected adverse controls, not clean-delivery passes.

Reachable tight-loop trials acknowledge **352–384** of 20,000 attempts and drop
**19,616–19,648**, with zero failed/unknown. Every per-trial count, accepted count,
diagnostic class and flush report is retained; no incomplete sample becomes a
zero association. Queue full, not a larger hidden buffer, absorbs overload.

| Unchanged control | Before | Fixed |
|---|---|---|
| First reply delayed 650 ms after real commit, 1,800 paced calls | 3,420 ack; 60 unsuccessful (2 timeout + 58 queued suppression); 120 caller cooldown drops | 3,600 ack; zero failed/drop/unknown; one retry; timeout diagnostic covers 2 spans; HWM 70 |
| 1,000 tight calls, capacity256, first reply delayed 300 ms | 288 ack; 1,712 queue-full drops | 288 ack; 1,712 queue-full drops; zero failed/unknown; HWM 256 |

The slow relay again logs Windows **10053** when replying to the already-timed-out
client; its traceback is retained. The later acknowledgement resolves the retry
without raising into the host. Arbitrary sustained outages still lose data;
the fix addresses the evidenced transient amplification, not infinite buffering.

### Byte regressions and the strict-comparison miss

All **15 frozen lane-01** full reports and CLI JSON/text bytes match their saved
versions. All **12 existing lane-02 development** full JSON reports (canonical
bytes including creation times) and original text bytes also match through
read-only analysis. Both historical stores' database hashes stay unchanged.

The initial fresh-replay comparison, excluding only two creation clocks,
**fails 120/120**. That result stays on record. Its reporting assumption was
wrong: `scout_probe.py` adds measured caller offsets to the harness run clock;
event raw timestamps therefore vary, and `internal/intake/decode.go` hashes
raw event identity into evidence IDs. Evidence IDs, cohort decision-ID values
and URL anchors vary across new captures. Neither algorithm was changed.

A **separate** analytic comparison validates each evidence-to-cohort-run
mapping and exact original URL/anchor hash, substitutes stable run identities
only in those reference fields, and retains all scores, terms, counts, ordering,
cohort membership, values, manifests, coverage, ranks and invariant results.
It matches **120/120** byte-for-byte canonical projections. No analytic value
is excluded. Tests retain the strict rejection and require changed scores,
counts, values, unknown evidence or invalid URLs to fail the separate check.
The old `score_count_bytes_match` JSON key retains the strict result; the new
`analytic_bytes_match` key is the separately stated score/count comparison.
This is not a claim that full new-capture reports are byte-identical.

All **25 protected files** retain their pre-task working-copy hashes, including
the closed evaluation lock/attempt and the owner's existing published-report
edit. The original lane-02 contract, generator/spec/checksums/source-freeze and
published results are untouched. The owner's open-question edits remain local;
only this task's appended questions are included in its commit.

### Checks, remaining misses and recommendation

**44 Python tests pass**, plus both explicit acceptance tests reported separately.
Tracked Go formatting, `go vet`, golangci-lint, fresh `go test -race -count=1
./...` and `task build` pass. Hosted Windows/Ubuntu CI passed for `a8453cb`
(Actions 39524192620); the final validation commit is checked separately before
the closing report.

**Delivery fix: accepted by these synthetic safety/development checks. Cost
gate: NO-GO.** All paced cost trials and all 120 development-case wrappers miss
the unchanged budget. Do not begin a future reserved evaluation or production
phase from this result. The ten-run clean transport streak alone cannot waive
the cost requirement or authorize a lane-03 contract.

Riskiest assumption: a future application's stalls and capture costs will fit
the finite worker budget and queue. A single quarantined OS call can outlive the
logical deadline; a process exit can lose pending unknown spans. There is no
durable spool, arbitrary-outage guarantee, proof of the original triggering
failure, broad cross-codebase cost claim or real-world diagnostic accuracy here.
