# Diff scoring v1

These choices were fixed before computing M2 rankings. They implement section 6;
they are not fitted to either demo seed. Scores describe unusual evidence, not
causal probabilities. All rates and sample sizes remain inspectable.

## Baselines

Select closed passing runs of the same workload, strictly earlier by start time
than the failing run. Sort by descending start time, then run ID, and take N.
The default is N=40; fewer than 10 selected runs yields `insufficient_baseline`.

- `default`: neither baseline flag supplied.
- `count`: `--baseline-n` supplied, with the same temporal rule.
- `before`: `--baseline-before` supplies a run ID (its start time) or RFC3339
  timestamp. Use the earlier of this cutoff and the failing run's start. The
  cutoff is exclusive. `--baseline-n` may also limit this window.

Persist mode, requested count, explicit-count flag, original cutoff, resolved
cutoff, selected IDs, workload, minimum size, and scoring version. No implicit
environment filtering: comparing changed configurations is part of the task.
Selection and evidence loading share a read transaction, excluding open runs.

In the M2 demo, the latest failure has many recent passing pool=5 runs. Its
default baseline should therefore have no pool-size config difference. That is
correct: this baseline asks why this request failed among its contemporaries.
A cutoff at the start of the failure phase instead compares against pool=20.
Early failures may still have pool=20 as the default baseline mode; always show
the actual selected runs rather than assuming all defaults have pool=5.

## Statistics and constants

| Choice | Value | Reason |
|---|---:|---|
| Default N / minimum N | 40 / 10 | Plan defaults; avoid interpreting a tiny history as normality. |
| MAD consistency factor | 1.4826 | Converts normal-distribution MAD to an estimate of standard deviation. |
| Squash scale k | 3 | Three robust standard deviations give effect `1-exp(-1)`; smooth, bounded growth without a binary threshold. |
| Latency absolute effect floor | 5 ms | Section 6's floor rejects small timing noise. |
| Minimum latency scale | 1 ms | Avoid division by zero for constant or timer-quantized baselines. |
| Minimum count scale / effect | 1 / 1 | A count is discrete; one extra occurrence is a modest effect, not infinite surprise. |
| Laplace smoothing | +1 success, +1 failure | A finite baseline cannot establish impossible or certain presence/exceedance. |
| Expected-absence threshold | 0.8 observed presence | Section 6's default; ordinary optional work should not become a missing-step alarm. |

`median` is the middle value (mean of the two middle values for even N).
`MAD = median(abs(value - median))`. Quantiles use linear interpolation at
`q*(N-1)`; p99 is descriptive, not an additional scoring threshold.

### Latency

Align spans by the complete root-to-span name chain, encoded as a JSON array to
avoid delimiter collisions. Repeated siblings contribute maximum duration and
count. Missing parents and cycles get explicit structural markers; malformed
trees never silently align with a valid root path. Use only baseline runs where
that path exists for durations; require 10 such observations. Presence is scored
separately, so an absent path is never a zero-millisecond measurement.

For a failing value x, baseline median m, scale
`s=max(1.4826*MAD, 1 ms)`, and e observations >= x among n observations:

`z=max(0,(x-m)/s)`

`score=(1-exp(-z/3)) * (1-(e+1)/(n+2))`

Return no latency difference if `x-m < 5 ms`. Only slowdowns are latency
outliers in v1. Include x, median, p99, raw exceedance e/n, observation count,
and the failing maximum-duration span ID.

### Presence and count

Presence aggregates spans and events by kind and name across a run. For b
baseline runs containing the item among N, `p=(b+1)/(N+2)`:

- Present in failure: `score=1-p` (no difference if present in every baseline).
- Absent in failure: `score=p`, only if observed b/N >= 0.8.

Count uses the latency formula with absolute effect `abs(x-m)`, minimum scale 1,
and an effect floor of one occurrence. Exceedance is directional: >=x for an
increase, <=x for a decrease. Include zeros for absent paths/events. Span counts
are aligned by path; event counts by name. Count and presence for the same item
are collapsed to the higher-scoring difference, retaining the other in Related.

Adding identical runs containing a common item can never increase its presence
surprise. The rapid property checks this presence rule. It does not claim that
adding observations cannot change a median or MAD: those statistics must reflect
the new distribution. Numeric scores are tested for bounds and monotonicity in
effect size with a fixed baseline, in both increase and decrease directions.

### Config and environment

Compare each flat key to the baseline mode, with missing values distinct from
JSON null. Choose the lexicographically first JSON encoding on mode ties for
determinism. A differing value scores `mode_count/N` exactly as in section 6.
Include baseline mode, support, failing value, and `config:<run-id>/<key>` or
`env:<run-id>/<key>` evidence (these resolve to the stored snapshot).

All config keys participate. Environment excludes resource identity/control
metadata: `service.name`, `service.instance.id`, `process.pid`, `process.command`,
`process.command_line`, `signalscout.*`, and `telemetry.sdk.*`. These describe
run grouping, service membership, instrumentation or experiment labels rather
than the environment fingerprint from section 4. Commit, dirty flag, service
version, dependencies, runtime, host, region and other environment keys remain.
No values are removed from storage. Report coincident commit changes as
correlation, not proof that a commit caused a failure. The current run snapshot
merges service resources; per-service config conflicts remain an open question.

## Ranking and redundancy

Sort by descending score, then type priority latency, presence, count, config,
env, then earliest failing occurrence, then stable feature key. The tie priority
puts directly observed timed/structural evidence before configuration clues.
Never use a feature's name (such as retry or cache miss) to boost its score.

Keep the deepest slow span path and attach slow ancestors in Related, including
their evidence IDs. Standard OTel `exception` events that all annotate this same
slow branch are also Related to that latency difference: repeating an SDK error
annotation at forwarding layers is one observation, not independent causes.
Standalone exceptions or exceptions on another branch remain presence/count
differences. Related records retain their own scores, rates and evidence IDs.

`baseline_rate` is raw presence for presence, raw directional exceedance for
latency/count, and raw mode support for config/env. Summaries name which rate is
being reported. Smoothed rates affect scoring only. Common but present items
remain in the output for the decoy check; an unchanged ubiquitous item is omitted.
Missing-item evidence cites baseline examples and the failing root/snapshot.

## Scope and validation

M3 consumes run-owned spans/events and snapshots only. It does not read or score
context logs or metrics, ordering changes, or metric divergence. Results are
persisted in `diff_results` through a writer goroutine; read selection uses a
separate read-only pool. A result-only writer must not finalize runs, refresh
context, change padding, or create a missing database. Run one operational
serving process per SQLite file, as before.

Validation records both baseline modes on the accepted seed-1 database and a
new seed-2 database. Timings include selection, evidence loading, extraction,
scoring, ranking and result persistence, excluding Go compilation and process
startup. Generated results retain each ranking; acceptance failures must remain
visible. No scoring constants may be changed to improve these demo rankings.

Property tests use the plan-approved [rapid package](https://pkg.go.dev/pgregory.net/rapid@v1.2.0).
