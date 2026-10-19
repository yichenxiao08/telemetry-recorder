# Context windows and scoring decisions (fixed before demo validation)

Attachment retains the configured 30-second lookback and 5-second lookahead,
including all matching exact-resource, host and instance rules. It is candidate
context, not ownership or causal proof.

For differences, compare equal policies per selected run. Logs use the actual
run interval [start,end]. Scalar metrics use the retrospective context window
[start minus configured lookback,end], excluding the future lookahead. The shop
SDK samples gauges once per second, so an actual 200 ms request interval alone
cannot reliably characterize pool state. The retrospective window is fixed for
all runs before validating demo ranks, not expanded until a desired point is
found. Require the metric resource's flat signalscout.config.* values, when
present, to agree with that run's config. This prevents a pool=20 sample from
the previous phase being attributed to a pool=5 run through a shared host.
Resources without config (host-only metrics) remain eligible. Padding and
rejected records remain visible in the context panel with provenance.

Series identity is service + metric name + point attributes + unit/kind from
the original metric, excluding resource phase/config/instance IDs. Otherwise a
changed config would make comparable gauges disappear into different series.
Summarize max, mean and last; compare with the identical summary in each
baseline run's retrospective window. Require ten baseline windows and two finite failing
samples; no zero-imputation for missing
metric windows. Ignore a series with no variation across failing and baseline
values; a constant baseline followed by a real change remains useful.

Reuse M3 numeric scoring (MAD factor 1.4826, effect scale 3), two-sided, with
relative unit floor and scale: 1% of max(abs(baseline median),1). This is a
dimensionless noise threshold, fixed before inspecting demo rankings. Keep the
strongest max/mean/last observation per series; retain the other statistics as
related evidence. No metric-name bonus or demo-fitted constant is permitted.

Context logs use template + numeric severity + service identity. Use only logs
inside each actual interval, with the same resource/config check. Presence uses
M3 smoothing; counts use M3 count scoring. Missing logs are zeros, missing
metrics are unknown. Collapse presence/count duplicates. Overlapping baseline
windows are correlated observations, not independent experimental replications;
report window counts, exact IDs and this limitation. Common templates remain
decoys. The context panel still shows padded candidate context and the failure
marker so users can inspect exclusions.

The gauge is evidence about pool state, not proof that configuration caused a
failure. M5 must distinguish mechanism and symptoms. Sampling can miss peaks;
report absence honestly instead of synthesizing a sample or expanding the
window after seeing results.
