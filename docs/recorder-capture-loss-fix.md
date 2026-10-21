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

Earlier diagnostic tests that explicitly required cooldown discarding or
truthy flush after failed delivery were replaced with stronger preservation
and completeness assertions. Counter-category unit tests disable retries to
isolate one attempt; dedicated tests enforce the release policy's retry bound.
Initial failures are retained in validation artifacts, not hidden.

Results will be appended after measurement; no reliability or budget claim is
made by this protocol. No scoring, engine, migration or later milestone work.
