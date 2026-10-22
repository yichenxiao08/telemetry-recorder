# Open questions and conservative choices

The choices below are implemented unless marked pending validation. Product choices remain open for the owner; they do not block implementation.

| Ambiguity or risk | Conservative choice |
|---|---|
| Module owner and supplied filename | Provisional module `github.com/signalscout/signalscout`; the supplied `plan (1).md` is preserved and `plan.md` is the working specification. |
| Run grouping across domains | Resource `signalscout.run.id` identifies ML runs; otherwise a nonzero trace ID identifies a web run. Logs with trace IDs can arrive before their spans. Reject domain collisions rather than merge unrelated runs. Ambient records never manufacture runs. |
| Root selection and incomplete traces | Choose the earliest parentless span, with stable ID as tie breaker. Workload is HTTP method plus route, falling back to span name. Missing root yields degraded rather than claiming a known passing request. Support old and current HTTP attribute names. |
| Idle timeout vs late data | Idle time uses server receipt time, not event time. Persist run state, recover open runs on restart, and finalize after configurable idle timeout. Late explicit evidence updates a closed run and recomputes outcome without reopening it. Shutdown drains accepted batches and closes open runs. |
| Single SQLite writer and acknowledgment | One goroutine owns all operational writes, including finalization and retention. Embedded migrations run before serving requests. Intake submits bounded batches and waits for transaction commit; a full queue returns 503 plus Retry-After. Accepted jobs continue if clients disconnect. Disk failures are returned as 503 and logged, never hidden. |
| Duplicate records and schema identities | Span identity includes trace ID because span IDs alone are not globally unique. Other record identities hash canonical OTLP record, resource, and scope metadata. Replayed identical records are idempotent. Explicitly conflicting span updates use the latest accepted record; ML failure is sticky. |
| Resource tracking and fidelity | Hash full canonical resource plus schema URL; retain resource, scope, original span/log/point JSON, and metric descriptors. Preserve all valid OTLP signal kinds, including histograms, summaries, exponential histograms, exemplars, links, and non-finite values. Aggregate metric points remain one record with full raw JSON; do not invent a scalar. |
| OTLP decoding and limits | Use official protobuf types and protojson (unknown JSON fields tolerated for forward compatibility). Accept protobuf or JSON and gzip only; bound compressed and decoded size at 16 MiB. Reject malformed data, invalid trace/span IDs, and timestamps outside SQLite signed nanoseconds before enqueue. Return the matching OTLP success response format. |
| Context attachment on close | Explicit ownership wins. Link ambient logs/points by exact identity or shared nonempty string `host.name` or `service.instance.id` with a resource seen in the run's spans. Keep every applicable rule and source resource in `context_matches`, also returned by `runs show`. Matching is exact and case sensitive; missing, empty or nonstring values never qualify a shared-key rule. Resources seen only in logs/metrics cannot qualify. Links remain many-to-many; late evidence refreshes them without changing outcome or ownership. |
| Shared host/instance collisions | Shared host names can bring in neighboring-service noise; reused instance IDs can over-correlate. Keep this as inspectable candidate context with provenance, not proof of a cause. Do not match on service.name alone or infer transitive host relationships. |
| Context padding | `serve` now follows the latest plan: 30 seconds before and 5 seconds after the evidence interval, configurable with `--context-before` and `--context-after`. Internal Options use explicit durations; zero means no padding. Benchmark settings remain unchanged. Padding admits nearby noise; late records inside the padded window may add links after close. |
| Outcome rules | Web fails only for root OTEL error or HTTP >=500; optional per-workload duration threshold makes it degraded. ML failure includes explicit wrapper failure, error spans, exception/job.failed events, and non-finite ml.loss metrics or attributes. Failure marker is the earliest known failure evidence. |
| Log template normalization | Preserve original log body and attributes; derive a deterministic template replacing quoted values, UUIDs, IPv4 addresses and numeric tokens. Templates are not identities or redaction. IPv6 and arbitrary opaque IDs remain unchanged until examples justify additional rules. |
| Retention scope | Configurable wall-clock retention removes closed runs and owned evidence by last receipt time; independent ambient retention uses receipt time. Never purge open runs. Cascade context links and remove unreferenced resources. Zero retention disables cleanup. All deletions go through the writer. |
| Retention vs extremely late data | Once retention deletes an identity, a new explicit record may create that run again. Permanent tombstones would prevent this but grow without bound; defer until there is a concrete requirement. |
| Durability vs throughput measurement | Benchmark complete HTTP decode, queue admission and SQLite commit, with WAL synchronous NORMAL. Record mode and hardware. Report spans, logs and scalar metric points separately; do not substitute queue admission for persisted throughput. |
| Local tools | Go 1.27.0, Task 3.54.0 and golangci-lint 2.14.0 were located and run. Use workspace caches within the sandbox. Race detector needs a native C toolchain even though production SQLite is pure Go; production build disables cgo. |
| Secret handling | Broad redaction is M8. Raw evidence is deliberately retained in M1; use only local trusted telemetry until that milestone. |

Identical logs/points have no source-issued unique ID. Occurrence suffixes preserve duplicates inside a request; identical repeated requests are treated as exporter retries. Rebatching identical records can therefore change duplicate interpretation. This remains an OTLP identity ambiguity.

Additional operational limits: run one serving process per local SQLite file. Context refresh now scopes operational work to changed closed runs and new ambient records; a startup backfill still scans history. Large-history cost is not measured by the fresh-database throughput benchmark. Retention bounds history, but longer-history tuning is deferred. Resource identity ignores top-level attribute order while preserving the original raw metadata. Unknown (zero) evidence timestamps do not widen an otherwise timed run's context interval.

Performance remains environment-sensitive: a run of the final storage layout dipped below the span target, while the original acceptance run passed all five span seeds under the historical span-only criterion. Previous complete failed reports are retained under benchmark/results. These fresh-database results do not establish a sustained production service level or a pass against all revised separate targets.

The revised M1 ingest targets are diagnostic review follow-ups. Three unchanged repetitions and all misses are retained; minimum/median summaries are generated by scripts/ingestsummary. Simultaneous combined ingest capacity is still unmeasured. Per the owner's explicit instructions, proceed to M2 after reporting these results without tuning to force a pass.

M2 choices:
- The sample query is a parameterized inventory read with pg_sleep modeling per-cart-line reservation work. Baseline and failing phases use the same work and seeded cart variation; DB_POOL_SIZE is the failure switch. Five bulk carts occupy real connections, then three ordinary cache-miss carts attempt acquisition; the intended rate is three failures per 100 requests. Actual results must be measured, not forced by HTTP status injection.
- Acquisition has a 100 ms timeout and one retry after 10 ms. The demo reports actual timeout spans and retry events; normal request cancellation is kept separate.
- Go's standard OpenTelemetry providers are passed explicitly. Trace, metric, and log exporters use the Collector; no global providers or custom collector are added. Versions are pinned to the compatible Go SDK 1.28 / logs 0.4 release family and otelhttp 0.53, plus pgx 5.7.1.
- The native Go demo runner owns the Windows SQLite intake lifecycle while Docker runs only the sample and Collector. This avoids sharing a live SQLite WAL between Windows and Linux container processes. The database stays local for native CLI inspection after the run.
- Collector host load/memory metrics describe the Docker Desktop Linux environment, not the physical Windows OS. A shared logical host.name (default docker-desktop) connects them to app span resources; the host metric resource contains only that host.name.

- The first actual M2 demo produced 40 baseline passes and 9/300 failures but failed its unchanged 45-second finalization check. Repeated full-history context joins starved the single writer. Context refresh is now scoped to changed runs/new ambient IDs; startup performs one atomic backfill, also removing stale links when padding changes. This fix happened after the three unchanged M1 review measurements; those reports do not measure the final M2 store implementation, and no revised-throughput pass is claimed.
- The 30-second lookback can include pool gauges from the previous demo phase. Raw resources/config and matching provenance are retained; M2 verifies a pool=5 point exists but makes no context-diff claim. Those difference types remain in M4b.
- M2 acceptance is complete. Remaining limits: the designed burst workload is not a production failure-rate estimate; host metrics describe Docker Desktop's Linux environment; shared host/instance names and padded windows can include unrelated or prior-phase telemetry; startup backfill and large-history/context-query storage costs are not yet benchmarked.

M3 preflight: one unchanged `task benchmark-ingest` invocation on the accepted M2 implementation is saved as `benchmark/results/m3-preflight-ingest.json` with a generated minimum/median summary. All separate targets were missed: spans 3,661/4,128 per second (target 5,000), logs 1,650/1,754 (target 5,000), scalar metric points 1,643/1,704 (target 10,000). These five-seed minima/medians are independent of the three older review repetitions. No tuning was applied. Simultaneous capacity and sustained production throughput remain unresolved.

Plan cleanup: the checked-in starting plan already contained only one section 5 context bullet, one separate-signal acceptance bullet, and no "Pending follow-up" sentence. The overlapping M1 measurement/review deviation notes were merged without changing their meaning. Generated demo database paths are now repository-relative; existing result values are preserved. `.cache/` was already gitignored.

## M3 choices (fixed before demo ranking)

- Baselines exclude open runs, degraded runs, other workloads and starts at/after the failing run. Explicit windows are exclusive and cannot admit future runs. A cutoff run ID supplies its start time; `--baseline-n` may additionally limit a window. Modes and exact IDs are persisted. No implicit environment filter can hide changed configuration. Fewer than 10 runs is an explicit insufficient-baseline result.
- Span latency uses name-chain paths and only observed durations, with at least 10 path observations; absent paths contribute zero only to counts. Cycles/orphans receive structural markers. Numeric floors, smoothing, tie order and all scoring constants are recorded in docs/diff-scoring.md before computing M2 rankings.
- Presence surprise, not arbitrary numeric distribution changes, is the property that cannot rise when adding identical common baseline runs. Median/MAD may change when the baseline distribution changes. Numeric scores have separate boundedness and effect-monotonicity properties.
- Identity/control resource attributes are not config/env changes: exclude service membership, process identity, signalscout control/experiment labels and SDK metadata from environment scoring while preserving them in storage. All flat config keys and genuine environment fingerprint keys remain. Conflicting per-service values in the current merged run snapshot remain unresolved; do not infer a service-specific configuration without evidence.
- Standard exception annotations on the same slow branch are related evidence of its latency anomaly; independent exceptions remain presence/count differences. This avoids treating errors repeated by forwarding layers as independent causes. No feature name receives a score boost.
- Inspection of the accepted M2 database confirms flat `Run.Config["DB_POOL_SIZE"]` (20 before the change, 5 after it). The original 40 passing runs contain `cache.miss` in 16 runs; the decoy is real. No config extraction or workload change is needed. Use the latest failing run for both-mode acceptance; early failures need not have 40 recent pool=5 passes.
- Diff reads must exclude context to keep M3 scoped and avoid loading thousands of unrelated context links. Result persistence must use a writer goroutine without intake maintenance or padding changes; missing databases must not be created by analysis. Large-history scaling remains outside the 40-run timing criterion.
- Standalone CLI diff persistence follows the existing one-writer-process agreement: stop intake first. Later server-side analysis can use the existing serving store's queue. A cross-process command API/file ownership protocol is deferred. Persisted diff snapshots may reference baseline evidence later removed by retention; M7 verification must detect missing evidence rather than trust stale citations. Neither ambiguity changes M3 scores.

M3 validation is complete on both seeds, all nine failures per seed in both
modes, with unchanged constants. The pre-change window always ranks config,
retry, acquisition latency, then the cache decoy. Both latest failures' default
baselines use pool=5 and omit that config difference, leaving the common cache
miss at rank 3 behind the two stronger observations. A moderate surprise score
and relative rank do not establish causation; broader failure accuracy remains
M5. Seed 2 has identical ranks but slower maximum timing (41.281 ms vs 30.969 ms).
Baseline cache misses are confirmed in 16/40 and 20/40 pre-change passes,
respectively. Generated reports preserve all ranks and timings.

Remaining choices are the module namespace, service-specific snapshot conflicts,
cross-process writer ownership, evidence changes/deletion after a saved diff,
shared-host context noise, large-history costs, and the separate ingest target
misses. No scoring acceptance failure is hidden, and no context metric/log
difference types or later milestone features were implemented.

## M4 choices and remaining questions

- Viewer reads use the existing read-only pool; only computed diff results are persisted, through the serving store's existing SaveDiff queue. No second writer is opened. Computation is an explicit same-origin POST; ordinary navigation does not recompute or persist a diff.
- A universal failure phase is not defined. Suggest a cutoff only for the explicit M2 failure-phase fingerprint, using its workload, config and revision; the user can override it with a run ID or RFC3339 time under the existing baseline-before semantics. Without that fingerprint, require an explicit cutoff for the before-phase panel. Repeated phases with identical fingerprints can share a suggestion; inspect and override the displayed cutoff rather than treating it as a detected causal boundary.
- Run lists default to the last seven days, use inclusive start/exclusive end, allow at most 31 days per window, and use keyset pagination (25 rows by default, maximum 100). Individual timelines allow 5,000 spans/events; comparisons allow at most 200 baseline runs and 50,000 total structural records. Oversized evidence is rejected visibly instead of producing a partial diff. Viewer requests have a deadline and concurrent computations are bounded.
- Templates escape telemetry as text. Assets are embedded and served locally; CSP disables scripts, external requests, framing and inline styles. Inline SVG uses numeric geometry and escaped labels, with CSS :target highlighting for evidence navigation. Unknown or deleted evidence remains visibly unavailable rather than pointing to an invented anchor.
- Numeric drill-down uses the same snapshot and feature aggregation as scoring. Latency omits absent paths; counts include zeros. This is display data only: scoring constants, persisted difference identities and M3 rankings remain unchanged. Context logs/metrics, service overview and search stay in M4b.
- Pre-M3 preflight missed every separate target: minimum/median spans 3,661/4,128 per second (target 5,000), logs 1,650/1,754 (target 5,000), scalar metric points 1,643/1,704 (target 10,000). A separate performance task is planned before M4b: profile decode/commit/finalization, startup backfill and context-link storage growth, test changes, rerun the unchanged seeded measurement and preserve all results. M4 does not tune intake. The saved seed-1 database is approximately 353 MB and its normal startup context rebuild takes minutes under the race detector; this cost is not part of M3 diff timing and is not hidden by changing startup behavior for the viewer.
- Retry and acquire-presence differences are failure symptoms. Configuration and context differences speak to mechanism; latency localizes delay without establishing causality. M5 failure definitions must label cause/mechanism versus symptom, and evaluate them separately. The viewer must not label a high score as a proven cause.
- M4 acceptance uses httptest navigation, anchor/security/queue tests and docs/viewer-walkthrough.md. The owner will capture screenshots; no screen recording is claimed. Module namespace, per-service snapshot conflicts, evidence changes/retention after a saved diff, shared-host context noise and large-history costs remain open.
- The additional saved seed-1 test under -race failed at Go's unchanged ten-minute timeout in startup context backfill before viewer HTTP began; the portable failure log is retained in benchmark/results/viewer-m2-seed-1-race.txt. The same dataset's native walkthrough passed, resolving 46 evidence links. The required self-contained race navigation suite, lint and build pass. No assertions/timeouts or startup storage behavior were weakened to hide this unresolved large-history issue; it is part of the planned performance task before M4b.
- Native saved-data walkthroughs pass on both demos: seed 1 resolves 46 evidence links and seed 2 resolves 43, each after persisting both baseline panels through the existing serving queue. Their test bodies took 63.46 s and 93.69 s including startup; these are not M3 diff timings. M4 acceptance is complete with the saved-data race startup failure explicitly retained as an unresolved performance issue. No later milestone was started.

## Pre-M4b performance task

This section supersedes the earlier unresolved full-history-backfill and materialized-link notes. Migration 005 stores run/resource/rule intervals and exposes compatible context link/provenance views. All applicable rules remain inspectable; ambient records stay unowned and never affect outcome. Late evidence resolves immediately, including updates that move a record out of an interval. Run changes and scopes commit together on the sole writer. A persisted padding checkpoint avoids rescanning history on ordinary restart; padding changes rebuild scopes, not evidence-by-run rows. Raw signals, snapshots, resources and complete logical context sets are hash-audited before/after on seed 1.

- Combined capacity is now explicitly 5,000 spans + 5,000 logs + 10,000 scalar points per second simultaneously. The unchanged separate benchmark remains diagnostic, with the same five seeds, record counts, batch size, and durable acknowledgments. Neither separate rates nor a legacy span-only exit status establish combined acceptance.
- Shared host/instance collisions remain candidate context, not causal proof. Resolving logical links lazily bounds physical growth by run/span-resource/rule count, but an unbounded query over every context combination can still be expensive. M4b must require run/time/LIMIT bounds and measure its own queries.
- SQLite reuses pages freed by migration; the saved database file does not shrink automatically. Report both file size and freelist/live pages. A quiescent offline compaction may reclaim the file later; it is not required to preserve existing data.
- Embedded schema migration precedes derived-scope initialization. Normal startup waits for that initialization; interrupted upgrades must be retried before serving readers. An unchanged restart uses the persisted checkpoint, and no timestamp cursor can skip late records.
- Profiling findings, unchanged baselines, failed combined attempts, measurements and the final go/no-go recommendation are recorded in docs/performance.md. Original user edits earlier in this file are preserved separately from this appended performance section.
- Final combined capacity remains a blocker: unprofiled five-seed minimum/median spans 1,308/2,317, logs 1,385/2,334 and scalar points 2,651/4,652 per second miss all simultaneous targets. Separate checks pass. Stop with a no-go recommendation for M4b; retain WAL/NORMAL, durable acknowledgments and all existing assertions. A follow-up performance experiment must address the measured commit/file-I/O bottleneck rather than assume retention cleanup is responsible.
- The 1M-row fixture meets the three bounded query budgets, but has only eleven resources and synthetic one-second spacing. High resource cardinality, long ML intervals, cold OS caches and M4b-specific context aggregations remain unvalidated. Whole-history logical counts still take seconds; future explorer work must retain time/LIMIT bounds and measure its actual query plans.
- The supplied public GitHub repository was empty. The owner explicitly approved the initial publication; the committed project is now on main in yichenxiao08/telemetry-recorder. Hosted ubuntu-latest and windows-latest jobs pass for a16740f (run 39325377076), with step evidence in benchmark/results/perf-05-hosted-ci.json. The provisional module path remains recorded above and was not silently renamed during the performance task; align it with the published namespace in a separately scoped cleanup.

## M4b choices

- The owner overrides the earlier no-go and replaces round-number throughput gates with workload-derived profiles. Keep the old results as misses. Shop rates use event timestamps inside each load phase and a sliding one-second peak, not exporter arrival bursts; twice the largest peak per signal across both saved seeds defines the shared budget. The pipeline profile is an estimate of 3,000 scalar points/sec plus the shop span/log budget. Report every seed even if a revised profile misses; no ingestion tuning or further target reduction is authorized here.
- Explorers require an explicit time range and limit plus a named service, explicit missing-service partition, or trace ID. The missing-service partition uses migration 007's service-expression index with IS NULL, so host-only telemetry remains independently browsable. Service overview requires a named service. Additional filters are SQL parameters applied within that partition. Reject excessive windows/limits and oversized aggregation inputs rather than silently sampling them. Query timings cover the seeded small-record fixture with warm OS caches, not arbitrary large payloads or high resource cardinality.
- Context attachment remains padded and fully inspectable. Log scoring uses actual run windows; scalar scoring uses the retrospective context window (configured lookback through run end), excluding future lookahead and config-incompatible resources. The SDK's 1 Hz gauges cannot reliably characterize a 200 ms request in isolation. Every baseline uses the identical policy. See docs/context-differences.md, fixed before demo ranking. Overlapping windows are correlated; missing metrics are unknown, not zero, and sparse gauges can miss peaks. No demo-specific scoring boost or window expansion after validation is permitted.
- Park Defender exclusions, background checkpoints, a commit waiting window and compact scalar storage in docs/ideas.md. The 12 GB/hour scalar estimate is extrapolated from a mixed fixture, not a measured scalar-only rate; M8 must measure it.
- Saved seed 1 contains only zero-valued db.pool.in_use samples through the padded failure window; no scorer can recover its unsampled bursts. The owner delegated the choice after this was reported. Keep both saved datasets, report the seed-1 absence as a data-limited acceptance exception, and validate the mechanism on seed 2 plus controlled tests. Do not regenerate a better-looking seed or tune constants. Seed 2's latest pre-change comparison ranks in-use #3 behind pool config and retry; recent baselines already share pool=5 and its sampled peak, so the difference disappears. First-burst seed-2 failures also precede the nonzero sample. Incidental host-memory differences can rank highly and are not proven causes. M5/M8 must measure sampling coverage and cause-vs-symptom accuracy.
- Saved-data race navigation required a bounded read optimization: resolve indexed resource/rule metadata once, merge overlapping intervals inside the read snapshot, and preserve exact per-run membership/provenance. No cross-request cache hides late updates. Caps remain 5,000 context records per run, 50,000 comparison records and bounded metadata/provenance; very large or high-cardinality windows are rejected visibly. Saved-data navigation remains expensive under the race detector; the small-resource 1M fixture does not establish cold-cache or large-payload performance.
- Final schema combined trials pass the measured shop budget in all five seeds but miss the estimated mixed pipeline budget in all five. Earlier revised trials passed only three of five for each profile and remain retained. Treat the 3,000-point mixed profile as unsupported; do not reduce its target or execute parked experiments during M4b. A real pipeline workload and sustained capacity/storage measurements remain future work. The new explorer index's write cost is included in the final run, but the change in outcomes is not a controlled causal attribution.

## Provenance design and feasibility choices (2026-10-20)

- M4b is not merged; PR #1 preserves it. The provenance branch is based on M4b,
  not stale main; a later PR must disclose this dependency. Do not merge either
  without owner direction. M5–M8 remain unstarted.
- Real Flèche code/data is not available in this checkout. Both new producers
  are entirely synthetic; simulated review labels are not real human judgments
  or real M5 failures. No question blocks the design/spike.
- Generic run units are not implemented: use one trace per frame/turn and retain
  today's web classification. Do not misuse signalscout.run.id to label frames
  as ML or mark HTTP errors to force quality-based comparisons.
- Logical decision/lineage/annotation IDs are proposed aliases over native evidence,
  not working resolvers. Capture completeness, graph validation and append-only
  annotation semantics require later work; OTLP preservation alone supplies none.
- Value capture remains synthetic-only until P0 recursive redaction/allowlisting
  covers raw copies and nested record JSON. Hashes are not anonymization or a way
  to inspect missing inputs. Sensitive low-entropy values default to omission.
- Prefer a bounded redacted detail buffer plus independent reference promotion,
  with coverage metadata for all runs. Label delay/eviction and selection bias
  are the riskiest assumption; no buffer or sampling implementation is authorized.
- Cohort formulas are design-only with existing M3 constants unchanged. Unknown
  labels/detail stay unknown; correlated frames/turns require cluster counts.
  Invariants require declared completeness and can work without references.

- Expanded request supersedes the earlier deferrable-ML wording: the ML adapter
  role is absorbed into a general OTLP helper. Throughput profiles are historical
  measurements only; per-call budget is fixed before measuring. Service overview
  and metrics browser are frozen. A/B synthetic structures plus real shop C
  validate representational breadth, not three real defects or causal accuracy.

- The expanded Part 4 is deferred at the owner's explicit fallback boundary:
  Parts 1–3 and a source audit are complete. A/B generators, Python helper,
  fresh A/B/C replay/browser checks and five-trial cheapness/safety measurements
  remain pending. No synthetic transport or overhead result is claimed.
  uv was not found on PATH; locate/provision it before that separately scoped spike.
- The p50 <=100 us / p99 <=250 us small-record budget is provisional and fixed
  before any helper measurement. OS scheduling and runtime locks cannot provide
  a hard real-time guarantee; distinguish no intentional I/O/capacity wait from
  zero wall-time stalls, and report full/drop/no-op paths separately.
- Capture fidelity is an independent risk: a producer can record incomplete or
  incorrect inputs even with a complete-looking graph. Invariants validate
  declared provenance, not arbitrary code or causes. Report coverage and trust.

## S0 executable probe spike (2026-10-20)

- The latest request makes external config/import probes primary; record_decision
  is retained as an optional semantic tier. Target source must stay unchanged.
  No P0–P4 or M5–M8 implementation is authorized.
- Stage 0 pins uv 0.13.3, CPython 3.12.12, OTel SDK/exporter/proto 1.28.2 and
  matching instrumentation 0.49b2. PyYAML 6.0.3 parses safe config; uv.lock pins
  transitives. Native Python 3.14 only bootstrapped uv.
- S0a safety checkpoint passes 15 tests, including the real Go intake. Thirty
  cost trials retain all misses: zero reachable 60 Hz trials meet 100/250 us.
  Budget failure is separate from host-safety failure and does not authorize
  performance tuning or hiding dropped provenance.
- Probes observe call boundaries, not actual variable reads or hidden rules.
  Treat their lineage/completeness as scoped observations, never as proof of
  full data dependence. Standard logging supplies correlation only; body export
  is OFF. Async/generators/descriptors/custom signatures/reloads are unsupported.
- Doctor metadata is bounded to one current status per configured probe.
  Reload is explicitly unsupported; aliases held elsewhere cannot be retroactively
  patched. The timed wrapper/recorder methods were unchanged by this importer-only
  correction after the measurement.

- S0b's ordinary shop replay preserves all 4,057 original native evidence IDs
  (1,960 spans, 376 events, 1,029 logs, 692 points). The saved database has more
  metric points than its older 567-point report; current stored data, not the
  stale summary, defines replay coverage. Do not silently discard the difference.
- Plain A/B require zero target-source edits, with 11/26 nonblank config lines.
  Optional B adds two lines for hidden selection semantics. Other internal policy,
  actual variable reads and cross-run value identity remain outside probe coverage.
- The measured late-label run/context ends both extend 60 seconds while outcome
  stays pass. Independent annotation time and mark/reference UI are deferred;
  a successful marked turn still gets 409 from failing-only diff.
- Windows wall timestamps can collapse short calls to zero. The synthetic helper
  now anchors UTC once and uses monotonic elapsed time; a failing regression test
  and a second full, unchanged-budget cost measurement are retained. Cross-process
  clock synchronization and complete semantic lineage remain unknown.
- Final duration-corrected cost pass still misses every 60 Hz budget trial:
  reachable median p50/p99 205.15/390.31 us; unreachable 197.45/593.31 us;
  saturated 272.15/708.41 us. Tight loops meet timing only with substantial loss.
  The final 19-test safety checkpoint passes. Budget/privacy/semantic-coverage
  gaps remain; no P0 implementation or production rollout is authorized.

## Slice 1 conservative choices (2026-10-20)

- PR #1 and the provenance spike are merged on main (859487c); Slice 1 branches
  from that main. No real Subway Surfers data exists; do not reconstruct it.
- The owner changed the reachable 60 Hz budget to 330/830 us after seeing results.
  All five corrected-duration trials meet it; first-candidate trials meet it in
  3/5 cases. Original 100/250 misses and saturation losses remain unchanged.
- Annotation writes use the serving process API and existing writer queue. CLI
  marking never opens another operational writer. Annotation created/recorded
  times are separate from telemetry and cannot widen execution/context scopes.
- Synthetic-only annotations require a rubric, source and reviewer. Revisions
  append, retain target/dimension/source/reviewer and cannot fork. Disagreement
  remains visible; label selection excludes conflicting active values. Unlabeled
  runs remain unknown. Predicates support captured scalar equality only.
- Cohorts pin unit/profile/workload, selection window, labels, native decision
  revisions and sessions. Require bounds/limit; reject oversized populations
  rather than truncate silently. Explicit reference cohorts cannot overlap the
  problem cohort. Per-feature observed minimum is ten on each side.
- A previous-version age is a profile-declared subtraction of two captured
  scalars; this knowledge lives in the frozen profile, never engine name logic.
  The version invariant checks producer-declared usage, not arbitrary reads.
- authored_blind means evaluation outcomes unseen. The requested mechanisms and
  formulas are known, and synthetic separability is deliberate; this is NOT
  independent blind validation or real-world diagnostic accuracy.

- Slice 1 evaluation completes all five seeds and three variants with 3000/3000
  captured decisions and no drops. Base mechanisms rank #2/#3 behind the output
  symptom; decoy-heavy pushes them to #9/#10; harder median age has no detected
  shift while the invariant flags 20/100 problem frames. These misses are retained.
- Base success is likely inflated: separable synthetic populations, an oracle,
  a declared age transform and known mechanisms. It is not an independently
  authored blind test or evidence of real-world diagnosis/causation.
- A post-first-run regression found contradictory output capture-off metadata
  was interpreted as observed. The parser now treats it as unknown. Every first
  result is retained; all 15 confirmatory cases have identical ranks, scores,
  counts and invariant results. No formula/profile/seed was tuned. Confirmation
  reused the same seeds and is not a fresh holdout.
- Predicate scalar equality is normalized (5 and 5.0 match); structured values
  are rejected. Its failed validation test is retained. Label-selected evaluation
  does not use this predicate branch. No richer query language is introduced.
- Both saved shop seeds retain all 72 M3/M4b ranked differences and scores exactly.
  The seed-1 gauge sampling miss and incidental context observations remain.
- The declaration-based invariant itself needs no reference, but this slice's
  CLI presents it alongside cohort comparison. A standalone invariant-only CLI
  and graphical mark UI remain parked; there is no graph-read proof or causal report.
- The frame cases are constructed case-control examples, not a reconstructed
  sequential game history. Previous-value origin is not independently verified;
  the invariant checks declared versions. Treat this as pipeline validation;
  base diagnostic accuracy is likely inflated. Independent histories remain future work.
- A later adversarial test found finite extreme values can overflow MAD subtraction
  and produce NaN. The numeric path now marks unsupported arithmetic as an
  observation, with observed counts retained. Formula constants are unchanged;
  wide-magnitude rapid and exact formula tests pass. Recomputing all 15 complete
  reports from pinned evidence yields identical results. The failed test is retained.

## Lane-01 diagnostic and proposed v1 (2026-10-20)

- Slice 1 is not merged into refreshed main 859487c; this analysis branches from
  origin/codex/slice1-lane at 05e9add as requested. Existing lane-01 artifacts,
  formulas, profiles and results remain immutable. Only new diagnostic outputs
  and review design are added; no M5–M8, P0, migration or difference type starts.
- Decoy-heavy intentionally shifts every unused noise input +10 for constructed
  problem examples. Both labels share all sessions/time blocks; the artifact is
  selection-correlated input construction, not references from another phase.
  Fair as an adversarial association test; unfair as causal-recovery acceptance
  from marginal values alone. Keep this limitation and every original miss.
- Harder has complete capture but its median erases the 20% stale subgroup in
  every seed. The existing declared-use invariant reports 20/100 violations.
  The statistic is responsible for this miss; actual historical reads remain
  unverified. No scoring-constant change follows from this diagnostic.
- Positive scores in heavy are distinct: exponential compression exists but
  did not decide the mechanism ranks; shared e=0 multiplies all by 101/102.
  Zero-score observations are unranked, not discoveries resolved by names.
- Proposed conditional comparisons need reviewed pre-output controls and common
  support; fully separating controls yield unsupported, never an adjusted cause
  claim. Tail contrast is descriptive and may also flag unused decoys. Sample
  minima, interval padding and support thresholds require owner review before
  a separately frozen evaluation; two lists are not fused opportunistically.
- Symptom roles must be declared/versioned before evaluation. Hidden benchmark
  decoy labels must never select or weight the candidate list. Historical ranks
  stay visible. Real-world mechanism identification is still unvalidated.
- Seeds 710021/710023/710027/710029/710033 are reserved in a design-only fresh
  preregistration proposal. No data generation or engine run on them occurred.
  Need approval for implementation and the final generator equations/contract.
- The plain target consults gap/previous_lane for lane, but only returns the
  supplied previous_frame as metadata. Changing its version alone cannot fix
  lane. Keep the stale-age mechanism label as an asserted freshness violation,
  not a verified causal effect; future sequential histories must establish use.

## Lane-02 Task A contract checkpoint

- Owner approves symptom roles and reference-tail coverage only; conditional
  comparison is deferred because controls need developer declarations, disjoint
  support is unsupported, and complexity is unjustified for this slice. Task A
  is docs-only; Task B requires another approval. No M5–M8 or P0 starts.
- Both slice1-lane and lane-diagnostic are unmerged into refreshed main. Branch
  codex/lane02-contract starts from lane-diagnostic. Review PR #4 is Slice 1 to
  main; #5 is diagnostic to Slice 1. The app returned 403; the signed-in browser
  created both PRs. No merge is performed in this task.
- The generator is pinned as a normative source specification, not executable
  code, to satisfy the docs-only boundary. Its checksum and contract checksum
  use canonical UTF-8/LF Git-blob bytes. Task B must freeze executable source
  hashes separately before generating any reserved-seed data. Needs review.
- Actual version lookup replaces lane-01's metadata-only flaw: older history
  value -1 versus fresh +1 flips the lane in the declared-use branch. Task B
  development-only counterfactual tests on 101/103 must verify this before eval.
- Minority problem cohorts use a disclosed current-value mechanism for the
  other 80% of wrong outputs. Within-range references contain unused older
  version values, deliberately hiding age from the marginal tail interval.
  These are short independent synthetic episodes, not continuous histories.
- Seeds 710021/710023/710027/710029/710033 remain ungenerated and unscored.
  Six variants, equations, profile, roles, floors, expectations and failure
  policies are predeclared in benchmark/problems/lane-02/contract.md and its
  generator-source.md. No post-result changes, seed search or fused score.

## Lane-02 Checkpoint A / Stage 1

- Docs-only amendment 6a3716b changes exactly the requested timestamp origin,
  tail-expectation wording, reproducibility reporting and harness-clock clause.
  Harness run times already anchor probe events, so no target/clock equation
  change was needed. Source freeze is benchmark/problems/lane-02/source-freeze.json.
- Only development seeds 101/103 ran: all 12 cases complete, deterministic
  age/current and invariant counts match, zero drops. Reserved seeds remain
  blocked unless the explicit evaluation flag and frozen-source checks pass;
  tests/imports/CI never enable it. Stage 2 awaits the owner's literal GO.
- The amended benign support premise is false for seed 101: seven noise_0
  values outside on both sides. Equal matched counts still give exactly T=0.
  Record this premise miss without changing any fixture, equation or score.
- Specified unequal-size smoothing can give positive score with no raw outside
  values: 0/30 vs 0/100 yields 1/32-1/102. The initial test failure is retained;
  tests now assert the exact specified effect plus edge equality on equal sizes.
  Unknown scores serialize null; observed feature sessions govern eligibility.
- Roles/tail are opt-in existing-cohort report metadata. CLI shows The problem
  separately; no new viewer page, registered structural/context diff type,
  migration, conditional comparison or production phase is added. Legacy CLI
  JSON/text bytes and all 15 lane-01 full reports remain identical.
- Development source refinement prefers true tail outlier citations. Final
  reanalysis from the same pinned records preserves all statistics and legacy
  output; the original development report remains cached. Read-only HTTP checks
  verify the refined links without replaying data or using a second writer.
- Single-trial development overhead retains two historical 330/830 us misses;
  none meets original 100/250 us. Do not equate these new-payload observations
  with the old five-trial protocol or claim a recording-cost gate passed.

## Lane-02 Stage 2 evaluation

- **Incomplete one-shot transport:** after GO, seed 710021 matched stopped with
  130 cooldown drops and 52 failed-export spans (1,418 successful of 1,600
  attempts). Conservative choice: retain the incomplete case and 29 unattempted
  cases, do not rerun/resume, and report all rankings/count expectations as NA.
- **Unknown exporter cause/persistence:** no exception/status detail was retained;
  the server log only records startup. Conservative choice: do not infer timeout,
  queue pressure or database loss. Failed acknowledgement may follow commit.
  Park bounded diagnostics for future review; change no frozen source.
- **Cost and acceptance:** wrapper 475.750/2,300.580 us misses original 100/250
  and historical 330/830 us budgets; one case is not the five-trial harness.
  Conservative choice: no cost pass, no lane-02 evaluation acceptance and no
  real-world diagnostic claim. See docs/lane02-stage2.md for raw counts/hashes.

## Recorder capture-loss Checkpoint A

- Lane-02 stays closed as attempted/incomplete due to capture loss; the lock,
  attempt-01, contract, generator/spec/checksums/freeze and published artifacts
  retain their pre-task hashes. Existing owner edits remain local. New diagnostics
  intentionally change the recorder, not the historical source-freeze manifest.
- Five development repetitions (101/103, six variants) complete 60 cases and
  acknowledge 96,000 spans without drops or failed exports. Natural loss is not
  reproduced, so the historical initiating exception remains unknown.
- A deliberate 650 ms post-commit reply delay causes two timed-out spans, 58
  queued cooldown suppressions and 120 caller cooldown drops. Conservative
  interpretation: timeout/cooldown amplification is evidenced, not the original
  SQLite/Defender cause. Failed acknowledgements can follow real commits.
- Defender status is denied by Windows. Added two-process CPU plus bounded-file
  fsync load has zero loss in one trial. No security settings/exclusions change;
  do not attribute original loss to Defender or claim background load is harmless.
- Queue 256 fills in the tight burst (1,712 drops); 4,096 avoids those drops but
  drains longer. Conservative choice: keep default 256 pending owner review;
  normal-rate peaks <=40 and the delayed-ack peak 58 do not support a capacity fix.
- Exception messages use truncated safe summaries because arbitrary exception
  text can contain payloads, URLs or response bodies. Keep class/reason/status,
  phase and timestamp, while documenting the redaction rather than claiming raw
  message retention. Flush timeout is not a drop; counters keep that distinction.
- All 60 development cases and all six controls miss 100/250 us on at least one
  quantile. No budget change or fix-cost claim. Context exit includes observer
  sampling; raw costs and every miss stay published. Strict global network
  deadline enforcement remains a proposal requiring proof, not an implemented
  property of urllib's per-socket timeout.
- Stop at Checkpoint A: the bounded worker-retry/queue-admission/accounting design
  in docs/recorder-capture-loss.md needs GO FIX. No behavior fix or Stage 4 work,
  lane-03 contract, scoring change or production phase starts here.

## Recorder capture-loss: GO FIX Stage 3–4

This section supersedes the proposal/awaiting-approval status above. Owner
approved GO FIX, including separate cooldown-preservation and duplicate-delivery
acceptance tests. The implementation is `a8453cb`; protocol, measured results
and remaining misses are in docs/recorder-capture-loss-fix.md and
benchmark/results/recorder-capture-loss-fix.

- The original reserved-run initiating failure remains unproven. The induced
  slow-acknowledgement/cooldown amplification is evidenced; do not attribute it
  to SQLite or Defender without an isolated measurement. Keep lane-02 closed
  as attempted/incomplete and preserve all old artifacts and owner edits.
- Queue capacity remains 256; one worker retry, 50 ms backoff and a 1.25 s
  logical batch budget are fixed. A quarantined OS network call can outlive
  that budget. Conservative choice: one I/O slot, no replacement threads, no
  late acknowledgement credit, bounded pending memory and explicit unknowns.
  No RAM queue can guarantee delivery across arbitrary outages/process exit.
- Flush now exposes exact acknowledged/failed/dropped/unknown producer counts;
  its truth means completeness, not drain. Invalid captures stay separately
  visible. Per-attempt diagnostics can include recovered timeouts without a
  final failed producer span. Conservative choice: unknown never means delivered.
- Python's indented JSON encoder checks dict truthiness. Structured flush
  reports must be copied with `dict(report)` before encoding; all spike report
  boundaries do so and a regression test covers incomplete reports. External
  callers using `flush(...) is True` need the documented completeness check.
  This remains a synthetic helper API, not a production compatibility promise.
- Real intake duplicate span/event delivery is tested with stable IDs and two
  committed requests. Log/metric exporter idempotency is not established here;
  defer any broader duplicate guarantee.
- Keep the 100/250 us overhead budget and every paired-trial miss. A clean
  transport streak cannot waive the cost gate or establish that recording is
  cheap in another application. No scoring or performance tuning is authorized.
- A future zero-drop gate is proposed only: ten consecutive full clean
  development repetitions, exact expected counts, unchanged analytic outputs,
  resolved evidence and passing safety controls. Preserve a miss, reset the
  qualifying streak and obtain owner authorization before any reserved attempt.
  No lane-03 contract or later milestone is created by this task.
- Final results: benchmark/results/recorder-capture-loss-fix-reviewed. All
  120 new development cases acknowledge 192,000 spans without loss; all paced
  overhead trials still miss 100/250 us. No cost-gate override is inferred.
- The first strict fresh-byte comparison (retained in the original results
  directory) fails because measured timestamp offsets change event hashes and
  anchors. Existing-store JSON/text bytes match; a separate evidence-validated
  analytic comparison preserves every score/count/value and matches all 120
  replays. Conservative choice: retain both results, never call new raw reports
  byte-identical, and do not change clocks, engine IDs or scores to erase it.

## Wrapper overhead Stage 1 (measurement only)

- The current probe has no independent capture-enabled switch. Conservative
  choice: measure omitted capture (OFF), explicit empty allowlist and full
  allowlist as configured today; do not invent a new runtime fast path.
- Fine-grained timing changes allocation, GC and worker contention. Keep raw
  timing, warmed empty-scope corrections, negative corrections, residuals and
  separate plain controls. Phase estimates alone cannot certify the budget.
- Worker-idle experiments fill the unchanged 256-item queue. Report full and
  nonfull groups, drops and unknowns; do not mistake saturation for an isolated
  causal test of GIL cost. GC/network overlap cannot identify a unique cause.
- Keep the interrupted profiling attempt and reporting failure. Its fix takes
  one observation-buffer snapshot; no shipped wrapper/recorder change follows.
- Checkpoint A stops before optimization; retain the unchanged 100/250 us
  budget and every miss. No reserved seeds, later milestones or lane-03 work.
- Final Stage 1: plain lossless paced medians miss at 468.50/912.87 us (fixed)
  and 498.85/1,315.68 us (lane inputs). Warm tight passes drop about 98% of
  spans. All 60 paced driver cases complete; five tight cases are incomplete
  and 55 unattempted. Retain these separately; never infer zero association.
- Fine timing remains intrusive after calibration; plain controls did not save
  separate context timings. Paced OFF/empty input controls cover only the
  sequence prefix, not all 12 cases. Conservative choice: disclose coverage
  and observer limitations, and do not promise precise optimization savings.
- Most plain paced p99 calls have no GC/encode overlap; quantized thread CPU
  cannot distinguish GIL/scheduling. GC-disable has no consistent benefit.
  Keep these causes unresolved; any optimization still needs GO OPTIMIZE.

## Single-output trace — 2026-10-21

- Opaque IDs have no global index. Conservative choice: require --run, explicit
  bounds and a read limit; discover only bounded run-qualified partitions.
  No migration/history scan. Missing is unknown in the printed search scope.
- A/B probe argument refs have no upstream producer identity. Show dangling refs,
  never infer edges from nesting/equality. This is the riskiest assumption for
  useful deep tracing across unmodified applications. Artifact refs are not fetched.
- Conflicts/duplicates/malformed envelopes/cycles produce integrity findings.
  No arbitrary winner. Captured local values do not certify graph completeness.
- Optional profiles evaluate declared versions only on captured fields. Off,
  hash-only, missing and unsafe integer values stay unknown. Lane-01 stale/fresh
  is violated/satisfied (3/3), but both lanes are left; its known metadata-only
  limitation is preserved. No counterfactual causal conclusion follows.
- Wrapper optimization is NO-GO. Latest conservative method-specific observation
  468.50/912.87 us misses both budgets. Historical 205.15/390.31 passed 330/830;
  fix-only paired data was 206.60/404.56. Harness hooks, payload details and
  uncontrolled machine state differ. Current observer-free cost is unknown.
  No overhead rerun; a future time-boxed binding experiment is deferred.

## Probe usability — 2026-10-21

- Supported method attachment is limited to plain top-level classes and directly
  declared synchronous methods. Properties, custom descriptors/access/metaclasses,
  slots, nested classes, native and async/generator callables stay unsupported.
  Default receiver capture is OFF; selected builtin dict fields only, never self.
- Child->base same-name chains on one receiver record the outermost configured
  override once. Suppressed calls remain counted. Explicit base calls in the same
  chain are treated the same; no bytecode inference that super() was used.
- Preflight is a dry import, not a sandbox or validation of a parser inside main.
  It reports isolated argv/attachments. Module import side effects are a risk.
- Queue-admitted decisions and created contexts are not proof of persisted runs;
  stderr includes ACK/flush/loss counters. Zero capture now exits 4, coverage gaps
  3, delivery problems 5. Target nonzero exits/exceptions remain unchanged.
- One paired 300-sample trial: function 210.95/421.31 us, instance method
  219.20/403.34 us p50/p99, zero drops; not budget qualification. Full method/state
  overhead is unmeasured. No optimization or repeated trial is authorized here.
- Riskiest assumption: monkeypatching plain class descriptors is transparent enough
  for ordinary callers. Identity/introspection-sensitive code and cached aliases
  remain outside the supported claim. Synthetic data only; P0 is separate.

## P0 capture gate — 2026-10-21

- Owner delegated compatibility choice: retain legacy synthetic operation and add
  explicit P0 policy mode, strict default OFF within it. Legacy is NOT privacy-
  safe. Real data remains forbidden; use a fresh store, no mixed legacy writers.
- Strict mode deliberately drops identity/semantic envelope metadata and metric
  values. Provenance/invariant results can be unknown; metadata HTTP writes are
  blocked. A safe semantic-field policy and store policy ownership are unresolved.
- Approved typed fields still require trustworthy domain classification. Numeric
  personal values or credentials disguised as safe enums cannot be ruled out by
  syntax alone. P0 is not consent, access control, deletion or retention policy;
  it does not solve arbitrary free-text redaction, covert protocol IDs, console
  output, direct metadata CLI writes, external artifacts or existing stored data.
- HMAC scope/key/version rules are explicit, but provisioning/rotation and token
  authenticity are not productionized. Artifact hashes cover approved redacted
  nonpersonal bytes only. No general personal hash or URI fetch is supported.
- Planted synthetic storage test ends at 0/77 cells; tests exposed sfixed64,
  repeated-histogram and unknown-wire gaps before fixes. Passing this finite set
  is not proof of zero real-world leakage. Added policy cost is unmeasured against
  both budgets, and no optimization is authorized.
- The lane-02 seeds were already attempted historically. Docs-only reconfirmation
  cannot create fresh holdouts; no reserved seed is rerun. Any new evaluation
  needs untouched owner-approved seeds. Linking proposal remains design only.
