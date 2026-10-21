# Recorder capture-loss investigation — Checkpoint A

Synthetic data only, native Windows/PowerShell. Lane-02 is closed as attempted,
incomplete due to capture loss. This task adds observation and development-only
experiments; it does not reopen evaluation or implement a fix. Wait for owner
**GO FIX** after Checkpoint A. No lane-03 contract or later production work.

## Observation protocol (written before controls)

- Keep the lane-02 driver, target, generator, profile, equations and scores
  unchanged. Run the driver in development mode with 101 and 103, all six
  variants, in five serial fresh stores. The existing completeness check still
  aborts a run on loss; retain failed reports and unattempted cases explicitly.
- `capture_loss.py` delegates to that unmodified driver. It adds a bounded
  caller-timing ring and a sampling thread that drains bounded worker rings to
  JSONL. Its sidecar records actual call-phase duration and achieved calls/sec,
  explicit flush cost, per-phase snapshots, high-water mark, and worker sizes.
  JSONL contains timestamps, sizes and counters, never payload values. Original
  driver reports retain synthetic fields as before. Observation overhead and
  uncontrolled machine activity are potential confounders, not free measurement.
- No measurements are the old reserved run's missing diagnostics. Even a matching
  induced failure cannot establish the historical initiating cause.
- Exception samples: last 64; worker samples: last 256; at most 32 named class
  keys plus overflow; status keys bounded to HTTP 100–599; sizes bounded 1–32.
  Rings are in-memory bounded. On-disk experiment records are finite, fixed-call
  runs, not an unbounded recording service. Sample loss is visible in the wrapper.
- Preserve all existing counters. New categories count affected spans, not HTTP
  attempts. `flush_timeout` counts calls. A timeout does not discard queued
  items, so `flush_drop=0`; `shutdown_drop` aliases existing `drop_closed`.
  Already-accepted cooldown-suppressed batches remain `export_failed`, while
  new caller rejections remain `drop_cooldown`. Do not count them twice.
- Phases describe where an exception surfaced: encode, connect, send,
  wait-for-ack (through response headers), response. Stdlib transport observers
  run only on the exporter's thread and delegate to unchanged methods. A phase
  does not prove which remote subsystem caused a delay. Existing urllib/HTTP
  behavior, timeout 0.5 s, cooldown 1 s, queue 256 and batches at most 32 remain.
- Messages are restricted/truncated safe summaries. Arbitrary exception strings
  can include captured values, URLs or server bodies; these details are redacted.
  Keep class, underlying reason class, safe errno/status, phase and attempt time.
  Do not claim that a redacted message is the original complete exception text.

## Controlled experiments (toy data)

After the serial default repetitions, run `capture_loss_conditions.py` once in
fresh stores. A local relay forwards ordinary OTLP to the real serving binary.
It records end-to-end upstream acknowledgement time (includes decode, queue and
commit), not a separately isolated SQLite commit duration. No server code changes.

| Condition | Calls | Queue | Deliberate condition |
|---|---:|---:|---|
| real-fast | 1,800 at 60 Hz | 256 | Relay without added delay |
| real-controlled-load | 1,800 at 60 Hz | 256 | Two CPU processes plus one bounded-file fsync process on the same machine/volume |
| post-commit-slow-ack | 1,800 at 60 Hz | 256 | Delay first reply 0.65 s AFTER the real intake's commit acknowledgement |
| burst-capacity-256 | 1,000 tight-loop | 256 | First reply delayed 0.30 s, below unchanged 0.5 s timeout |
| burst-capacity-4096 | 1,000 tight-loop | 4,096 | Same 0.30 s control; explicit experimental capacity comparison, not a proposed default change |
| unreachable | 1,800 at 60 Hz | 256 | Reserved local port not listening |

For each retain counters, bounded diagnostic samples, batch-size histogram,
queue high-water mark, actual rate/wall duration, caller timing samples and relay
timestamps. Compare caller p99 with 100/250 us without changing that budget.
These are single control runs, not Stage 4's five-trial cost validation.
The frozen driver measures probe wrapper time inside an explicitly entered run
and reports that run's enter/exit separately. Toy controls use the old cost
harness convention: caller wall time minus target body, including automatic run
setup/cleanup; those context samples are also retained separately. Do not pool
these different payloads/context conventions into one cost claim.

Hypotheses are assessed separately: timeout/cooldown amplification; server
acknowledgement latency; Defender/background activity; queue capacity; and
temporal overlap of wrapper stalls/export waits. Correlation and total response
time cannot isolate SQLite or Defender. Read-only Defender status may be denied;
do not disable protection or add exclusions. Sample process CPU/RSS without
command lines or unrelated application content. Report access limits explicitly.

## Results — Checkpoint A

**Natural capture loss was not reproduced.** All five serial repetitions
completed all 12 development cases: 48,000 calls, 96,000 acknowledged spans,
zero failed exports and zero drops. Every score/count expectation still matches.
The unchanged median minority miss, deliberate within-range miss and benign
finite-support premise mismatch remain in the raw reports; repeating them does
not repair them or establish real diagnostic accuracy.

Generated [run/control tables](../benchmark/results/recorder-capture-loss/summary.md),
[all 60 per-case costs](../benchmark/results/recorder-capture-loss/cases.md) and
[machine-readable details](../benchmark/results/recorder-capture-loss/summary.json)
retain every run. The `raw/` archives restore the exact original JSON reports and
streams, with verified hashes. Only incidental absolute local paths in log copies
are redacted; originals stay local and both hashes are disclosed. All partial
stores remain under `.cache/capture-loss`, never replacing lane-02 attempt-01.

Measured call phases: **60.116–60.266 calls/sec**, **13.275–13.308 seconds**
per 800 calls. Queue high-water marks by repetition: **40, 14, 16, 16, 16**
out of 256. Sampling overflow is zero in all five runs. Worker batch duration
maxima: **335.605, 117.322, 140.617, 143.431, 123.336 ms**. These include
encode and end-to-end request/ack time, not separately timed SQLite commits.

### Failure classes and exact accounting

The post-commit-delay control induces one **TimeoutError in wait-for-ack**,
affecting two in-flight spans. The worker then suppresses **58 already-accepted
spans** during cooldown and the caller rejects **120 new spans** during cooldown:
**3,600 = 3,420 exported + 60 export_failed + 120 dropped**. Queue peak is 58,
well below 256. The upstream acknowledged the first request in **21.700 ms**;
the relay held its reply for about 650 ms after that acknowledgement. Thus a
failed client acknowledgement can follow a real commit. Keep the client's two
ambiguous outcomes unacknowledged; do not retroactively count them as delivered.

The terminal log also retains a relay **ConnectionAbortedError / WinError 10053**
when attempting the delayed reply after the client has gone away. That handler
error did not escape into target calls or abort the experiment. It was not hidden,
fixed and rerun. The recorder's observed failure is the earlier timeout.

The unreachable control records **URLError with TimeoutError reason in connect**:
40 spans in timed-out requests, 1,200 accepted spans suppressed by cooldown,
2,360 caller cooldown drops. **3,600 = 0 + 1,240 + 2,360**. There are no real
encode failures, explicit connection-error classifications or HTTP non-2xx
responses in these controls; those categories are exercised by injected tests.

Burst at capacity 256: **2,000 = 288 + 0 + 1,712 queue-full drops**, HWM 256.
At experimental capacity 4,096: **2,000 = 2,000 + 0 + 0**, HWM 1,968. Explicit
flush rises from **695.365 ms** to **4,301.014 ms**. That is a real buffering
trade-off, not evidence for increasing the serving default.

All reports use the unchanged legacy identity:
`attempted = exported + export_failed + dropped + unsettled`.
Unsettled is zero in every completed measurement. `export_failed` means
unacknowledged, not confirmed database loss; it includes software-suppressed
items that were never sent. Stage 3's stronger acknowledgement/failure/unknown
state machine is a proposal, not an implemented accounting upgrade.

### Evidence by hypothesis

| Hypothesis | Evidence and conclusion |
|---|---|
| (a) 0.5 s timeout + 1 s cooldown amplifies loss | **Supported for the induced mechanism.** One delayed acknowledgement produces two timed-out spans, 58 queued suppressions and 120 new rejections. Existing code renews cooldown while discarding queued batches. This explains amplification, not the original initiating exception, which was never retained. |
| (b) SQLite commit / Windows I/O spikes | **Not established as the initiating cause.** Default worker waits reach 335.605 ms without loss. Real relay upstream ack maxima are 99.305 ms without added load and 59.020 ms under the specified load. These aggregate decode, queue and commit; commit alone is not isolated. The delayed-ack loss occurs after commit, demonstrating that slow commit is not necessary for this failure pattern. |
| (c) Defender / background processes | **Not reproduced by the specified load.** Two CPU processes and a bounded-file fsync process produce zero loss in one 30 s trial on a 16-logical-CPU machine. This is neither full saturation nor a factorial CPU-vs-I/O study. The loaded trial being faster does not establish an improvement. Defender status returns Access denied; no exclusion/disable experiment was performed and no Defender attribution is possible. CPU/RSS samples have no command lines or personal content. |
| (d) Queue 256 too small | **Supported for the deliberate tight burst, unsupported as the recorded normal-rate rejection cause.** Increasing experimental capacity removes that burst's full-queue drops but lengthens drain time. Default peaks remain <=40; delayed-ack peak is 58; the original lane-02 report has no queue-full drop. Larger capacity alone does not remove cooldown rejection/discard behavior. |
| (e) Wrapper p99 spikes are the same event as export stalls | **Partial temporal overlap, not equivalence.** With the predeclared diagnostic >=100 ms wait definition, default-01 caller p99 is 1,727.852 us inside waits vs 919.362 us outside. Default-02 is 3,858.045 vs 698.048 us (only 86 inside samples). Yet real-fast has no >=100 ms batch and wrapper p99 922.509 us. Host scheduling/serialization spikes occur without long export waits; shared pressure and observer effects remain possible. |

The historical initiating cause remains **unknown / not reproduced**. The
reserved run lacked achieved-rate, queue and exception diagnostics, so no claim
about what changed between it and these development runs is justified. The
controlled timeout amplification is reproducible evidence for a targeted design
fix, not a forensic recovery of that missing exception.

### Cost, safety and observation limits

All **60** development cases miss the unchanged **100/250 us** budget. Wrapper
p50 spans **231.550–462.150 us**; p99 **496.015–1,791.453 us**. All six toy
controls also miss at least one limit; every value is in the generated tables.
Do not substitute the historical 330/830 us owner override or discard outliers.

Development run enter p50 is 20.550–37.200 us, exit p50 15.300–28.000 us.
The wrapper's exit observation (timestamp/sample append) is included in that
exit cost; this is not an isolated pure-recorder exit measurement. All context
p99 and flush costs remain in the raw sidecars. Toy context costs are separately
reported in summary.json. These runs are not Stage 4's five-trial fixed-payload
overhead validation and do not quantify a fix's cost: **no fix exists yet**.
Diagnostics can perturb timing and share the Python GIL; do not treat monitoring
as zero-cost or infer causation from time overlap.

The 32-test Python suite passes, including injected failure counters, bounded
storage, transport exception identity, doctor output, independent byte-level
OTLP construction and real intake capture/lineage tests. Original safety tests
still enforce nonblocking queue behavior, no host-path network/flush/join,
unchanged target returns and exception identity, default-off capture and bounded
memory. Tests/imports/CI never enable reserved evaluation. Formatting, vet,
golangci-lint, fresh `go test -race -count=1 ./...` and `task build` pass.
Hosted Windows/Ubuntu CI is confirmed separately on the committed branch.

All **25 protected files** match their pre-task working-copy hashes: evaluation
lock, attempt-01 files, contract/spec/checksums/freeze, driver/generator and old
published lane-02 artifacts (including the owner's existing local report edit).
The recorder itself intentionally changes for diagnostics; the historical source
freeze is not rewritten to pretend that these new sources were evaluated then.
Target instrumentation remains zero source changes and ten config lines.

## Proposed fix for owner review — NOT IMPLEMENTED

1. Keep queue 256, caller nonblocking behavior, socket timeout 0.5 s, durability,
   OTLP bytes and existing IDs. Do not retry encoding errors or terminal HTTP
   rejections. Do not increase capacity on the strength of a tight-loop toy.
2. Let the worker retain a transiently failed batch and make **at most one
   additional attempt**, with a **50 ms worker-only backoff** and **1.25 s total
   per-batch attempt budget**. The implementation must enforce the overall
   deadline, not assume a urllib per-socket timeout bounds DNS/headers globally.
   A stuck network operation must not create unbounded threads or retries.
3. Cooldown governs worker network attempts. While capacity remains, callers
   may enqueue without waiting; do not discard already-accepted batches merely
   because cooldown is active. Sustained outages still fill a finite queue and
   cause visible full-queue drops. No zero-loss promise under arbitrary outage.
4. Track each producer span's state exactly: acknowledged, known failed,
   dropped, or unknown/unsettled at a timeout-bounded explicit flush. Keep
   request-attempt counters separate from producer-span counts. Ambiguous
   timeout outcomes stay unknown until an acknowledgement actually resolves
   them; a drained queue is not a completeness gate.
5. A retry can duplicate a span already committed before an acknowledgement
   failed. Reuse trace/span IDs and test actual intake span/event idempotency
   before accepting the fix; do not assume this extends to future log exporters.

Trade-offs: extra worker I/O/CPU, delayed delivery, longer flush/outage draining,
and duplicate/ambiguous outcomes in exchange for surviving short transient
acknowledgement failures within a fixed memory/time budget. A one-retry policy
will still miss longer outages. Strict deadline enforcement and exact lifecycle
accounting are the riskiest implementation assumptions; both require the owner's
Stage 3 tests and Stage 4 measurements before any reliability claim.

**STOP at Checkpoint A. Await GO FIX.** No retry/buffering behavior change,
Stage 4 validation, lane-03 contract, scoring update, migration, new diff type or
later production milestone has started. This synthetic recorder investigation
does not demonstrate real-world diagnostic accuracy.
