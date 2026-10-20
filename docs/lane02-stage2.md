# Lane-02 Stage 2 — stopped on incomplete transport

## Contract and source

Owner **GO** authorized the one-shot reserved evaluation. Contract amendment:
**6a3716b8b1c21b3167331d0c0e7421ffc529fdfb**. Source freeze:
**d704cc7b40b341df7ed02e8f564202d5f653f7f6**. All 34 canonical frozen file
hashes still match. No implementation, profile, equation or scoring constant
changed. Lane-01 and development results remain untouched. The historical
pre-GO status inside the frozen manifest is preserved, not rewritten.

## Implemented surface

No additional engine or recorder surface was implemented. The frozen runner was
invoked once with all five reserved seeds in contract order, against one fresh
store. It stopped at its existing completeness check in the first case.
The exclusive evaluation ledger remains in place. No case was retried and no
further reserved data was generated, replayed, scored or inspected through the
engine. Publishing reads saved files only; it does not open SQLite.

## Reserved results

**Attempted: 1; incomplete: 1; completed: 0; scored: 0; unattempted: 29.**
The overall runner status is failed; the first case status is incomplete, not a
completed negative result. Attempted is a total, not an additional status bucket.
The complete 30-row seed/variant table and development-format result sections
are in [the evaluation report](../benchmark/results/lane02-stage2-evaluation.md).

| Variant | Seeds attempted | Candidate median | Candidate tail | Positive counts median / tail |
|---|---|---|---|---|
| matched-negative-controls | 710021 only; incomplete | NA | NA | NA / NA |
| minority-stale-use | None | NA | NA | NA / NA |
| benign-minority-tail | None | NA | NA | NA / NA |
| within-range-subgroup | None | NA | NA | NA / NA |
| moderate-decoy | None | NA | NA | NA / NA |
| unrelated-nuisance | None | NA | NA | NA / NA |

Neither candidate list nor the symptom-inclusive legacy list was computed.
Top-1/3/5, ties, nuisance-above-mechanism entries, evidence links and invariant
counts are unavailable. No analytic age/current expectation was tested. There
is no measured deterministic mismatch, agreement or five-seed reproducibility
result. Seed-dependent nuisance results are likewise unavailable. **NA is not
zero association.** The known development misses remain in Checkpoint A.

### Retained transport failure

Seed **710021 / matched-negative-controls** completed 800 host calls and captured
4,800 arguments. The harness manifest labels 400 outputs problem and 400
reference; these are not completed store cohort selections. The runner stopped
before annotations, cohort creation, analysis, evidence checks and final doctor.

| Counter | Value |
|---|---:|
| Serialized decision envelopes | 735 |
| Envelope bytes | 1,189,437 |
| Attempted spans (root + probe) | 1,600 |
| Accepted spans | 1,470 |
| Successfully exported spans | 1,418 |
| Failed-export spans | 52 |
| Cooldown drops | 130 |
| Network requests | 419 |

Accounting: 1,418 + 52 + 130 = 1,600. Failed exports have unknown persistence;
an acknowledgement can fail after commit. They are not confirmed database
losses. Some failed-export items can be suppressed while already in cooldown,
so 52 is not a count of independent network failures. No other drop counter
was recorded. A drained flush does not recover failed exports or prove capture
completeness. The existing check correctly rejected this case.

The log contains only the listener-start message. The recorder's worker catches
export exceptions without retaining their class, message or HTTP status. Its
unchanged defaults are capacity 256, HTTP timeout 0.5 seconds and cooldown one
second. The saved evidence cannot distinguish timeout, HTTP rejection, encoding
failure or another exporter error. No root cause is asserted from counters alone.

### Retained cost misses

| Measurement | p50 (us) | p99 (us) |
|---|---:|---:|
| Probed-call wrapper | 475.750 | 2,300.580 |
| Run-context enter | 44.000 | 318.680 |
| Run-context exit | 17.800 | 86.061 |

Wrapper cost misses both the original **100/250 us** and owner-approved
historical **330/830 us** budgets. Explicit flush: **9.7015 ms**, separate from
the host path. This is one 800-call case scheduled at 60 calls/sec, not the older
five-trial fixed-payload harness. Achieved rate and phase wall duration were not
stored. Mean serialized envelope size is about 1,618.28 bytes. No cost tuning or
new safety-gate pass is claimed.

Read-only counting after failure confirms **zero source lines changed**, zero
instrumentation lines, and **ten config lines**, against the pre-run target
snapshot. Host calls completed; broader safety claims remain the regression
suite's scope, not an inference from incomplete telemetry.

## Validation and retained artifacts

The published JSON is the exact raw runner report, including all 800 harness
rows and every counter. Artifact metadata records sizes and SHA-256s for the
partial database, WAL/SHM, log, report and target snapshot, plus a portable ledger
copy. Original files remain in `.cache/lane02-evaluation/attempt-01`; the exclusive
`.cache/lane02-evaluation.lock` is retained. Do not delete it to retry.

Native PowerShell regression checks passed: formatting check, `go vet`,
`golangci-lint` (zero issues), `go test -race ./...`, `task build`, and all
25 frozen Python tests. These tests use development/toy inputs and never enable
reserved evaluation. Hosted Windows and Ubuntu CI is verified separately on
the result commit; it must not run reserved seeds.

## Risks and boundary

The riskiest immediate assumption is that the fail-open, lossy capture path can
produce complete, representative cohorts at this scheduled load. This attempt
did not. Missing records could bias associations; host success and flush success
are insufficient completeness checks. The lack of retained exporter error detail
also limits diagnosis of the recorder itself.

**No-go on claiming lane-02 evaluation acceptance or advancing from these
results.** No new production phase, fix/retry, seed replacement or replacement
evaluation is started. An owner-reviewed future protocol would be needed for
further measurement. This incomplete attempt establishes neither mechanism
ranking nor real-world diagnostic accuracy. Even complete synthetic arithmetic
would not establish causal diagnosis across real codebases.
