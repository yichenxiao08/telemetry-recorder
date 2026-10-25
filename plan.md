# Signal Scout: Build Plan (Go)

> This file is the source of truth for building Signal Scout. Read it fully before writing code. Work one milestone at a time, in order. Do not start a later milestone until the current one's acceptance criteria pass.

## 1. What we're building

**Signal Scout helps a developer explain any behavior or problem they want to
understand by recording how outputs were produced, tracing inputs and decisions,
comparing labeled problem examples with reference examples, and checking declared
expectations, with every claim linked to evidence.**

Failures are one way to mark a problem. A developer may instead ask about a
lane-change calculation, an unexpected choice, or replies human reviewers call
too strict. One unchanged system can supply both problem and reference examples.
Domain meaning lives in versioned profiles, not generic core heuristics.

The telemetry store and M0–M4b remain the foundation: all received OTLP signals,
run grouping, context, structural differences and bounded exploration.
Execution spans alone do not explain data dependence. Decision records and
explicit lineage must record the inputs, intermediate choices, configuration
and rejected alternatives the producer actually reports using.
Human quality judgments are annotations with provenance, never inferred from
operational success. Unlabeled examples stay unknown.

### The core loop

1. Configure supported standard telemetry, report coverage, and add explicit
   profile-defined decision/input/output signals where standard telemetry ends.
2. Store OTLP with stable native evidence, run/session links, capture completeness,
   privacy controls and external artifact references.
3. Let the developer define a problem using marks, predicates or review labels;
   explicitly select reference examples from the same or another system.
4. Follow output lineage and check developer-declared invariants without requiring
   any baseline. Missing evidence yields unknown.
5. Compare eligible cohorts, showing raw counts, sample/cluster sizes and missingness.
6. Report observations, associations and hypotheses separately, cite evidence,
   and suggest developer-controlled experiments. Never claim causation.
7. Evaluate on pre-registered real and synthetic cases, retaining misses and costs.

### Architecture

- Go backend, standard OTLP and SQLite single-writer storage remain.
- Domain adapters/helpers emit OTLP; Python and other clients never open SQLite.
- Standard integration setup covers only supported libraries; app-specific meaning
  requires explicit instrumentation. One initial runtime/profile precedes breadth.
- Existing Go shop remains a real operational benchmark; the ML adapter role is absorbed into the general helper; trainer breadth is optional. No M5, M6, M7 or M8 implementation starts in this design task.

### Non-goals (preserved and extended)

- A custom collector or database engine. Reuse OpenTelemetry Collector and SQLite.
- A polished dashboard. Keep the UI minimal and functional.
- Robotics integration (only an adapter interface and design note). A synthetic
  lane/frame fixture is a data-dependence example, not a robotics integration.
- Replay or rerun from saved inputs. Replaying synthetic OTLP into intake validates
  transport, not re-executing recorded applications.
- An artifact store. Keep URI/content-hash references; never copy large files.
- A human diagnosis-time study.
- Multi-tenant auth, accounts or cloud deployment.
- Dashboard builders, alerting or PromQL/LogQL compatibility.
- Judging quality itself: labels/predicates express developer/reviewer criteria.
- Inferring how arbitrary code produced an output without instrumentation.
- Claiming causation: observations, associations and hypotheses remain distinct.

Specifications: docs/provenance-spec.md and docs/problem-model.md.
The original failure-only thesis is superseded: normal/failing run comparison
remains an implemented useful subset, not the whole problem definition.

## 2. Tech stack (fixed unless a milestone says otherwise)

| Concern | Choice |
|---|---|
| Language | Go 1.22+ (module name `github.com/<owner>/signalscout`) |
| HTTP server | Standard library `net/http` with Go 1.22 `ServeMux` patterns. No web framework. |
| OTLP decoding | `go.opentelemetry.io/proto/otlp` types with `google.golang.org/protobuf` (`proto` for protobuf bodies, `protojson` for JSON bodies). Endpoints: `/v1/traces`, `/v1/logs`, `/v1/metrics`. |
| Storage | SQLite via `modernc.org/sqlite` (pure Go, no cgo), WAL mode, JSON stored as TEXT and queried with SQLite JSON functions |
| Migrations | Embedded `.sql` files (`embed`) applied by a small in-repo migrator |
| CLI | `github.com/spf13/cobra` |
| Viewer | `html/template` + `embed`, plus a little vanilla JS. No frontend build step. |
| Stats | In-repo `internal/stats` package (median, MAD, quantiles, CUSUM). `gonum` is allowed if a function is clearly needed. |
| Testing | `go test` with table-driven tests, golden files in `testdata/`, `pgregory.net/rapid` for property tests, `-race` in CI |
| Lint/format | `gofmt`, `go vet`, `golangci-lint` |
| Logging | Standard library `log/slog` |
| Sample web stack | Go services using `otelhttp` and `pgx`/`pgxpool` + Postgres via docker compose |
| Sample ML stack | Python 3.11+, PyTorch (CPU), `opentelemetry-sdk`, managed with `uv` |
| LLM (agent layer) | Anthropic Go SDK behind a small `LLMClient` interface so it can be mocked |

### Platform: Windows (native, no WSL)

The project owner develops on **Windows with PowerShell and does not use WSL**. Everything must work there.

- Use **Task** (`Taskfile.yml`, https://taskfile.dev) instead of `make`. Do not create a Makefile.
- **No bash scripts.** Write helper scripts (demo setup, benchmark orchestration, fixtures) in Go under `scripts/` or as subcommands of `cmd/ssbench`, run with `go run`.
- Use `path/filepath` for paths; never hardcode `/` or `\` separators. Binaries are built as `bin/signalscout.exe`.
- Add a `.gitattributes` that normalizes line endings (`* text=auto`, and `*.sh text eol=lf` if any shell file ever exists).
- Docker commands must work with Docker Desktop on Windows. Keep volume mounts simple.
- CI should run on both `ubuntu-latest` and `windows-latest` for build and tests.
- Any Python commands use `uv` and must work in PowerShell.

### Go style notes for Codex

- The project owner is newer to Go. Prefer plain, idiomatic Go: small packages, concrete types, simple interfaces defined where they are used, explicit error handling. Avoid clever generics, reflection, and deep abstraction.
- Pass `context.Context` as the first parameter to anything that does I/O.
- No package-level mutable state. Wire dependencies in `main` and pass them in.
- Wrap errors with `fmt.Errorf("...: %w", err)`.
- Comment any non-obvious concurrency (goroutines, channels, mutexes) explaining why it is safe.

## 3. Repository layout

```
signalscout/
  plan.md
  README.md
  go.mod
  Taskfile.yml
  docs/
    run-spec.md            # the run model spec (deliverable)
    diff-scoring.md        # scoring formulas
    adapter-guide.md       # how to write an adapter, incl. robotics design note
    benchmark.md           # methodology and results
    ideas.md               # parking lot for out-of-scope ideas
    open-questions.md
  cmd/
    signalscout/           # main binary: serve, runs, diff, explain
    ssbench/               # benchmark harness binary
  internal/
    intake/                # OTLP HTTP handlers, decoding to internal models
    model/                 # Run, Span, Event, MetricPoint, Artifact, Difference types
    runs/                  # run grouping, open-run tracking, outcome rules
    store/                 # SQLite access, migrations/ (embedded .sql), queries
    telemetry/             # query layer over all spans/logs/metrics, service aggregates, context attachment
    retention/             # TTL cleanup for ambient telemetry
    stats/                 # numeric helpers (median, MAD, quantile, CUSUM)
    diff/                  # feature extraction, alignment, scoring, ranking
    agent/                 # prompt building, LLMClient interface, mock, verifier
    redaction/             # scrubbing rules and tests
    viewer/                # handlers, templates/, static/ (embedded)
    config/
  adapters/
    ml-python/             # signalscout_ml package (Python)
    conformance/           # adapter conformance fixtures and checker
  samples/
    shop/                  # Go web sample (frontend, checkout, inventory) + docker compose
    trainer/               # Python ML sample project
  benchmark/
    failures/              # one directory per seeded failure
    results/               # committed result JSON and tables
  testdata/                # OTLP fixtures, golden files
```

## 4. The run model (draft; finalize in `docs/run-spec.md`)

A **run** is a profile-declared unit of execution: request, turn, frame, job or
session, with boundaries and identity. Operational outcome is separate from
problem labels. The fields and web/ML rules below describe the built M0–M4b
contract, retained unchanged; generic run profiles are proposed, not implemented.

### Revised core concepts (design)

| Concept | Meaning |
|---|---|
| Decision record | A step's output ref, declared inputs, chosen/rejected options, rule/threshold and config refs. |
| Lineage edge | An explicit input -> output derivation through a step, distinct from span parenthood. |
| Behavior of interest | A developer-selected output, omission or pattern to understand, before calling it a defect. |
| Problem | Developer-authored question and criteria for selecting examples. |
| Problem/reference cohorts | Explicit eligible sets; no old/new core requirement and no unknown-as-good shortcut. |
| Label | Outcome-rule, predicate or append-only human assertion with source/reviewer, rubric/version and evidence provenance. |
| Invariant | Declared expectation, such as input X appearing in output Y's lineage; deterministically checked without a baseline. |

See the two specifications for versioning, unknown states, capture/privacy and
proposed difference formulas. None is silently mapped onto today's HTTP outcome.

### Identity and metadata

| Field | Notes |
|---|---|
| `run_id` | Unique. For web: the trace ID of the root request. For ML: a user- or wrapper-supplied ID set as the resource attribute `signalscout.run.id`. |
| `workload` | Stable name used to find comparable runs, e.g. `POST /checkout`, `train/resnet-small`. |
| `domain` | `web` or `ml` (extensible). |
| `started_at`, `ended_at` | UTC timestamps. |
| `outcome` | `pass`, `fail`, or `degraded`. |
| `failure_marker` | Optional: span/event ID where failure was first detected. |
| `env` | Environment fingerprint: commit SHA, service versions, dependency versions, host, region. |
| `config` | Flat key-value snapshot of relevant configuration. |

### Evidence (three kinds, coexisting in one run)

- **Spans**: causal, timed steps (requests, DB queries, training steps, data loading).
- **Metrics**: periodic series (loss, queue depth, pool usage). Stored as `(name, t, value)` points.
- **Events**: discrete occurrences (retry, error, config load, batch drawn).
- **Logs**: log records, linked to a run by trace/span ID when present.
- **Context telemetry**: metrics and logs from the same resources (service instance or host) during the run's time window, attached to the run by resource and time rather than by trace ID. Examples: host CPU, memory, pool-in-use gauges, warning logs from a neighboring service.

### Artifact references

`{uri, content_hash, kind, redaction_status, retention}`. Large files are linked, never embedded.

### Domain profiles

A profile names the attributes that carry domain meaning without claiming domains are equivalent.

- **web**: `http.route`, `http.status_code`, `db.statement` (redacted), service name.
- **ml**: `ml.epoch`, `ml.step`, `ml.shard`, `ml.dataset_hash`, `ml.loss`, `ml.lr`.

### Outcome rules

- Web: `fail` if root span status is error or HTTP status >= 500. `degraded` if duration exceeds a configurable per-workload threshold.
- ML: `fail` if the wrapper marks failure, the job raises, or loss becomes non-finite. Otherwise `pass`.
- Rules live in `internal/runs/outcome.go` and must be covered by table-driven tests.

### Run grouping

- Web: group by trace ID. The root span defines workload and outcome.
- ML: group by `signalscout.run.id` across spans, metrics, and logs.
- Late-arriving data must attach to an existing run. Runs are "open" in memory until a configurable idle timeout, then finalized (outcome computed, row marked closed).

## 5. Storage schema (SQLite, initial)

Run tables first, then the telemetry tables that hold every signal (including `metric_points`, defined below).

```sql
runs(run_id TEXT PRIMARY KEY, workload TEXT, domain TEXT, started_at INTEGER, ended_at INTEGER,
     outcome TEXT, failure_marker TEXT, env TEXT /*JSON*/, config TEXT /*JSON*/, closed INTEGER)
spans(span_id TEXT PRIMARY KEY, run_id TEXT, parent_span_id TEXT, name TEXT, service TEXT,
      start_ns INTEGER, end_ns INTEGER, duration_ms REAL, status TEXT, attrs TEXT /*JSON*/)
events(event_id TEXT PRIMARY KEY, run_id TEXT, span_id TEXT, name TEXT, ts_ns INTEGER, attrs TEXT /*JSON*/)
artifacts(artifact_id TEXT PRIMARY KEY, run_id TEXT, uri TEXT, content_hash TEXT, kind TEXT,
          redaction_status TEXT, retention TEXT)
diff_results(diff_id TEXT PRIMARY KEY, failing_run_id TEXT, baseline_spec TEXT /*JSON*/,
             created_at INTEGER, results TEXT /*JSON*/)
```

Add indexes on `(workload, outcome, started_at)` for runs and on `run_id` for spans, events, and metric points.

### Telemetry tables (all signals, not only run-attached)

```sql
resources(resource_id TEXT PRIMARY KEY, service TEXT, instance TEXT, host TEXT, attrs TEXT /*JSON*/)
logs(log_id TEXT PRIMARY KEY, resource_id TEXT, run_id TEXT /*nullable*/, trace_id TEXT, span_id TEXT,
     ts_ns INTEGER, severity INTEGER, body TEXT, template TEXT, attrs TEXT /*JSON*/)
metric_series(series_id TEXT PRIMARY KEY, resource_id TEXT, name TEXT, unit TEXT, kind TEXT, attrs TEXT /*JSON*/)
metric_points(series_id TEXT, run_id TEXT /*nullable*/, ts_ns INTEGER, value REAL)
```

- Spans, logs, and metric points are all stored whether or not they belong to a run. `run_id` is nullable and set when grouping succeeds.
- `logs.template` is a normalized message (numbers, IDs, UUIDs, and quoted strings replaced by placeholders) so recurring log patterns can be counted and compared. Keep the normalizer simple and tested.
- Add indexes on `(resource_id, ts_ns)` for logs and `(series_id, ts_ns)` for metric points, and on `trace_id` for logs and spans.
- **Retention:** telemetry not attached to any run (ambient) is deleted after a configurable TTL (default 7 days) by a periodic cleanup goroutine in `internal/retention`. Run-attached evidence is kept until the run is deleted. Provide optional per-signal drop rules (e.g. drop debug-level logs) in config.
- **Context attachment:** when a run closes, link ambient logs and metric points by exact resource identity or a shared nonempty string `host.name` or `service.instance.id` with a resource seen in its spans. Do not qualify resources seen only in logs/metrics. Store interval/rule scopes per qualifying run resource, not copies or evidence-by-run rows. Compatibility views resolve every matching evidence ID, rule and qualifying span-resource ID lazily. `serve` pads the interval by default 30 s before start and 5 s after end (`--context-before`/`--context-after`; zero disables padding). Context remains many-to-many and does not change ownership or outcome; late ambient evidence is immediately visible through the views, and late run evidence updates scopes in the same writer transaction.

### Concurrency model

- SQLite allows one writer at a time. Use a **single writer goroutine** that receives batches over a channel and commits them in transactions. Intake handlers decode and enqueue; they never write directly.
- Reads (viewer, diff engine, CLI) use a separate read-only connection pool.
- Enable WAL mode and set a busy timeout.
- Intake must apply backpressure (bounded channel; return 503 with `Retry-After` when full) rather than grow without limit.

### Evidence IDs

Stable strings shown to users and cited by the agent: `span:<id>`, `event:<id>`, `log:<id>`, `series:<series_id>@<ts>`, `metric:<run_id>/<name>@<ts>`, `config:<run_id>/<key>`, `artifact:<id>`.

## 6. Diff engine design

The failing-run baseline API below is the built M3/M4b subset and remains
unchanged. The general problem/cohort/categorical/value design is specified in docs/problem-model.md. Its narrow synthetic Slice 1 subset is now owner-authorized; existing M3/M4b scoring constants remain unchanged.

**Input:** one failing run (later: a set of failing runs) and a baseline set of passing runs of the same workload.
**Output:** a ranked slice of `Difference` structs, each with a score in [0, 1], a type, a human-readable explanation, and evidence IDs.

### Baseline selection

- Default: the most recent N closed passing runs of the same workload (N=40) whose start times are strictly before the failing run. No implicit environment filter hides configuration changes.
- Support default mode, explicit `--baseline-n` (count mode), and `--baseline-before <run-id or RFC3339 time>` (before mode). The explicit cutoff is exclusive and capped at the failing run's start; a run ID resolves to its start. The count may also limit before mode. Record mode, count, original/resolved cutoff, selected run IDs and scoring version in `diff_results.baseline_spec`.
- Require a minimum baseline size (default 10); otherwise return a clear "insufficient baseline" result.

### Structural alignment

Normalize each span to a **path key**: the chain of span names from the root, e.g. `frontend.request > checkout-service.handle > inventory-service.reserve > db.pool.acquire`. Repeated siblings are aggregated (max duration and count per path). Features are computed per path key.

### Difference types and scoring

All scores are normalized to [0, 1] so they can be ranked together. Document formulas in `docs/diff-scoring.md`.

1. **Latency outlier** (per path key): robust z-score using baseline median and MAD, plus the fraction of baseline runs at or above the failing value. Score combines effect size (squashed, e.g. `1 - exp(-z/k)`) with `1 - exceedance_fraction`. Ignore paths where the absolute difference is below a floor (default 5 ms) to avoid noise.
2. **Presence/absence** (spans and events by name): if present in the failing run, score = `1 - baseline_presence_rate`. If absent but expected, score = `baseline_presence_rate` (only when rate >= 0.8). Use smoothing so 0/40 does not produce infinite confidence.
3. **Count change** (e.g. retries per run): compare to the baseline distribution using the same robust approach as latency.
4. **Event ordering change**: for event/span pairs that always occur in one order in baseline, flag a reversed order. Score by baseline consistency.
5. **Config/env difference**: for each key, if the failing value differs from the baseline mode, score = fraction of baseline sharing the mode. Also report when the change coincides with a commit.
6. **Metric divergence**: change-point detection on metric series (start with CUSUM or a two-window mean shift), compared to baseline series. Report the step/time of divergence and its lead time before the failure marker.
7. **Context metric difference**: for each context metric series (CPU, memory, pool in use, queue depth), compare summary statistics over the failing run's window (max, mean, last value) to the same statistic over the baseline runs' windows, using the robust approach from latency. Ignore series with too little variance or too few points.
8. **Context log difference**: compare log templates (and severity levels) present in the failing run's context window to the baseline windows. A template that is new or much more frequent in the failing window scores like a presence/count difference. Common templates (e.g. present in 30% of baseline windows) rank low as decoys.

### Ranking rules

- Sort by score; break ties by type priority, then by earliest occurrence in the run.
- Collapse redundant differences (a slow parent span and its slow child): keep the most specific and mention the parent in `Related`.
- Always include a **decoy check**: for each top difference, report the baseline rate so a common-but-present item ranks low (e.g. a cache miss present in 30% of passing runs).

### Explanation format

```
db.pool.acquire in checkout > inventory: 4812 ms in failing run vs. baseline p99 14 ms.
0 of 40 baseline runs exceed 100 ms.            [span:8f3a-07]
```

### Suggested core types (adjust as needed)

```go
type Difference struct {
    Type        string   // "latency", "presence", "count", "ordering", "config", "metric"
    Score       float64  // [0,1]
    Summary     string
    EvidenceIDs []string
    BaselineRate float64 // decoy check: how common this is in baseline
    Related     []string
}

type Differ interface {
    Diff(ctx context.Context, failing model.Run, baseline []model.Run) ([]Difference, error)
}
```

## 7. Agent layer design

The original specification below is retained for history. Its failure-only,
ranked-diff-only input is superseded by the proposed three-tier report contract:
bounded decision/lineage evidence, invariants, labels, coverage and cohort counts
must also be available. M7 remains unstarted. Verified citations are not verified
causes; telemetry and reviewer text remain untrusted data.

- The agent receives **only** the ranked diff (plus run metadata), never raw logs.
- Required structured output (JSON, decoded into Go structs and validated):
  - `hypothesis`: text
  - `claims[]`: each with `text` and `evidence_ids[]`
  - `experiment`: a concrete confirming experiment
  - `ruled_out[]`: differences judged irrelevant, with reason and evidence IDs
- **Verifier** (deterministic, no LLM): for every cited ID, check that it exists in the store and belongs to the failing run or the baseline summary; check that numeric claims quoted in the text match stored values within tolerance. Output per-claim `valid` / `invalid` / `unsupported`.
- Metrics emitted: citation validity rate, claim support rate.
- The LLM client sits behind a small interface with a mock implementation for tests and CI:

```go
type LLMClient interface {
    Complete(ctx context.Context, system, prompt string) (string, error)
}
```

## 8. Milestones

Each milestone lists tasks and **acceptance criteria**. A milestone is done only when all criteria pass.

### Revised order and status (proposal, 2026-10-20)

Execution boundary: Parts 1–3 and source audit are accepted. The synthetic external-probe/helper spike is executed: safety passes, recording-cost budgets have retained misses, and A/B/C fresh-store replay confirms transport breadth and manual workflow gaps. No P0–P4 or M5–M8 production implementation starts.

The requested task includes documentation plus an executed synthetic spike, not the start of a
later milestone. PR #1 is confirmed merged (e260a09), and main 859487c also includes the accepted provenance spike. Slice 1 starts at main on codex/slice1-lane.

| Order | Scope/status | Acceptance before the next slice |
|---|---|---|
| M0–M4b | Built foundation; retain historical criteria/results below, including seed-1 sampling exception and pipeline capacity miss. | Existing regression suite and Windows/Ubuntu CI remain green. |
| S0a (measured; cost miss) | External Python probes, optional helper and A recording spike only; safety checkpoint passes 19 tests. | Bounded queue/drop/no-op behavior, host-exception preservation, instrumentation LOC and five-trial 60 Hz/tight-loop p50/p99, retaining misses. |
| S0b (executed; limits retained) | Plain A/B targets via config, independent Go A producer, original C OTLP replay and existing HTTP views. No semantic engine changes. | Verify facts/refs and native anchors; report manual workflow gaps; simulated labels explicit; no schema/diff changes. |
| P0 | Capture contract and privacy gate; synthetic data until complete. | Allowlisted values round-trip; planted secrets absent from body/attrs/raw/nested envelopes/URIs; unknown schemas fail closed; capture omissions visible; no default conversation text. |
| P1 | One tiny provenance helper and guided setup for one actual supported runtime; framework coverage is explicit. | Under 15 minutes on a prepared clean fixture; repeated setup idempotent; no duplicate OTel providers/spans; decision IDs/conflicts and coverage inspectable; default-off field capture. |
| P2 | Problem records, append-only annotations and bounded explicit cohorts. | HTTP-200 problem marks do not change outcome/times; disagreement/retractions/rubric versions survive; unlabeled stays unknown; exact selected IDs/label revisions persist; same-core comparison works. |
| P3 | Bounded lineage inspection and deterministic invariants. | Cross-run versioned refs resolve; expected input missing with complete capture violates; incomplete/cyclic/dangling/evicted graphs yield integrity/unknown; no baseline required. |
| P4 | Categorical/value cohort differences from problem-model.md. | Tests first: bounded scores, coverage/cluster counts beside every association, no iteration pseudo-replication/name boosts; held-out fixtures; unchanged M3/M4b constants. |
| M5 revised | Trimmed evaluation: 4–5 cases, at least 3 problems observed in real systems, with independent review or declared expectations; synthetic cases explicitly marked. | Pre-register cases, expected evidence, cause/symptom labels, metrics and fixed constants before ranking; actual held-out runs/fresh experiments; at least one retained miss; report top-k plus invariant/coverage results over five seeds where generation applies. |
| M7 revised | Bounded observation/association/hypothesis reports and verifier; suggested experiments. | Reject fabricated/wrong-run/wrong-turn citations, misquoted numbers and causal upgrades; insufficient evidence is explicit; report with no reference cohort can use invariants only; assess against pre-registered baselines. |
| M8 revised | Remaining overhead/storage/redaction audits and write-up. | Measure scalar and decision bytes, label delays, buffer coverage and real workload capacity; publish costs/misses, final adapter conformance and reproducible PowerShell workflow. Privacy minimum already gates P0. |
| M6 superseded | General helper absorbs the ML adapter role. Trainer/failure breadth remains optional, not a separate active milestone. | A second domain uses the same OTLP contract without server special cases; preserve old ML text for history. |

M5 keeps pre-registration discipline. The existing real pool-exhaustion case can be one; other real cases need
actual recorded behavior and independent review/expectations, not cases
invented from engine features. The two planned synthetic fixtures would not satisfy
"at least 3 real". Prefer including input-lineage and multi-stage cases only when
actual observations and independent review/expectations are available. Four or five cases replace the old 6–8 web plus
6–8 ML breadth so missing primitives can be validated first. No scoring tuning.
The volume buffer/sampling policy remains a design pending measurements.

### Historical recording-cost boundary (owner override below)

The round-number 5k/5k/10k rates, revised shop capacity gate and estimated
3,000-point pipeline profile are retired as acceptance gates. Preserve every
measurement, miss and method as historical evidence. Scale is a documented
limitation, not a reason to keep expanding a telemetry product. Freeze the
service overview and metrics browser at M4b: no extensions in this roadmap.

Pre-register a provisional small-record recording-call budget: p50 <=100 us
and p99 <=250 us, measured as caller wall time at 60 calls/sec and in a tight
loop, with reachable and unreachable intake, five repetitions and reported drop
counts. Include bounded capture/serialization/enqueue; exclude startup and
explicit flush, measured separately. No network/queue-capacity wait on app paths,
no recorder exception into the host, bounded memory and visible drops/no-op
fallback. This is an engineering target, not a hard real-time guarantee of the
OS/Python scheduler. Keep any miss; do not move the budget after measuring.

Validation must span A (noisy per-frame calculation with stale prior value),
B (human-labeled staged pipeline) and C (existing shop pool exhaustion). Trimmed
M5 must pre-register 4–5 problems, at least 3 real observed problems, covering these
structures. Synthetic A/B fixtures are not real defects or actual human labels.
M6's adapter scope is absorbed by the language-neutral helper; no training-specific
server core is required. Recorder helper, marking, lineage view and invariants
come before M5. The record -- command wrapper is design-only in this task.


### Slice 1 owner authorization and implementation decisions (2026-10-20)

PR #1 is merged; branch codex/slice1-lane starts from main 859487c, which also
contains the accepted provenance spike. Only the synthetic lane-01 vertical
slice is authorized: immutable annotations, CLI marks, explicit bounded cohorts,
categorical/input-value associations and one declared-used version invariant.
These are narrow portions of the P2–P4 designs, not completion of those phases.
P0 production privacy and M5–M8 remain unstarted; no hypothesis generation.
The original Subway Surfers data is unavailable and is not reconstructed.

The owner sets reachable-60-Hz budget p50 <=330 us and p99 <=830 us AFTER seeing
the results (roughly 2%/5% of a 16.7 ms frame). All five corrected-duration trials
meet this new budget; first-candidate trials meet it in 3/5 cases. All original
100/250 us misses and raw results remain. This is an explicit post-hoc acceptance
change, not a claim against the original budget. The recorder/probe implementation
is unchanged in Slice 1; old measurements apply to their fixed measured payload,
not a claim about arbitrary field volume or end-to-end diagnosis cost.

Pre-registration b537ee1 locks problem.yaml and slice1-scoring.md BEFORE the first
new evaluation run. Seeds 1009/2027/3037/4051/5059 are distinct from development
101/103 and prior spike seeds. Base, decoy-heavy and minority-mechanism variants
remain fixed; authored_blind is explicitly outcome-blind, not independent authorship.
Numeric MAD/squash/smoothing constants and all M3/M4b constants remain unchanged.

Migration 009 adds append-only annotations and immutable cohort snapshots. Marks
post to the existing serving writer queue; CLI reads use read-only connections.
Annotation created/recorded times do not become execution/context timestamps.
Annotations require synthetic provenance and revisions cannot fork/change target
or reviewer. Active reviewer disagreement is explicit and excludes label selection.
No privacy/redaction or real-content gate is claimed.

Cohorts pin bounds/limit, unit/profile/workload, run IDs, label revisions, native
decision revisions and sessions. Selection rejects oversize, overlap and profile
mismatch; a field needs ten observed runs on EACH side before ranking. Unknown
capture is excluded, never zero. Reports contain observation/association tiers,
raw counts, selected/observed/missing counts, exclusions and cluster counts.
The API uses bounded same-origin JSON; no second operational writer is opened.

Profiles use JSON to avoid a new Go dependency. A profile-declared subtraction
(frame minus supplied previous version) makes age comparable across frames;
returned categories may be selected alongside verdict/reason/chosen. The generic
engine has no lane, gap, stale or decoy-name logic. The invariant's selectors and
delta live in the declared profile; it checks assertions, not arbitrary reads.
Capture-off outputs are unknown even if a contradictory inline value is stored;
a failing regression test and all pre-fix evaluation results are retained.

The narrow task lane-01 invokes the pinned Python client through a native Go
orchestrator; it is NOT the M5 multi-problem harness. The target is plain Python,
unchanged by probes. No general query language, mark UI, lineage graph, production
privacy, real-system replay or causal reporting is added.
### Historical milestone text (preserved)

M0–M4b below document what was built, not work to restart.
Old M5–M8 text is retained with supersession notes, not an active authorization.

### M0: Project skeleton

**Tasks**
- `go mod init`, `Taskfile.yml` tasks (`build`, `test`, `lint`, `run`), `.golangci.yml`, `.gitignore`.
- Create the repo layout above with placeholder packages and a smoke test.
- Add CI (GitHub Actions): `gofmt` check, `go vet`, `golangci-lint`, `go test -race ./...`.
- Write a `README.md` stub that links to `plan.md`.
- Create `cmd/signalscout` with a Cobra root command.

**Acceptance**
- `task test`, `task lint`, and `task build` all pass on a clean checkout, in PowerShell on Windows.
- `bin\signalscout.exe --help` prints CLI help.

### M1: OTLP intake, run grouping, storage

**Tasks**
- Implement `/v1/traces`, `/v1/logs`, `/v1/metrics` accepting protobuf and JSON (including gzip request bodies).
- Decode into internal models; write via the single-writer goroutine to SQLite with embedded migrations.
- Implement run grouping and outcome rules for the web profile (trace-based) and the ML profile (run-ID attribute).
- Implement idle-timeout finalization of runs and late-data attachment.
- Store **all** logs and metric points (not only run-attached ones) using the telemetry tables above, with resource tracking, log template normalization, and the retention cleanup goroutine.
- Implement context attachment when a run closes.
- Document how to point an OpenTelemetry Collector (or an SDK directly) at Signal Scout, with a sample Collector config.
- CLI: `signalscout serve`, `signalscout runs list`, `signalscout runs show <run_id>`.

**Acceptance**
- Replaying recorded OTLP fixtures from `testdata/` produces the expected runs, spans, events, and outcomes (golden-file tests).
- Malformed payloads return 4xx and do not corrupt storage.
- Backpressure works: when the write queue is full, intake returns 503 (tested).
- **Historical throughput acceptance (retired by the 2026-10-20 recorder budget):** The owner replaced the original round-number simultaneous 5,000 spans + 5,000 logs + 10,000 scalar points/sec gate after reviewing its measured miss. Keep those original results and separate diagnostics unchanged. Current combined profiles: twice the maximum sliding-one-second emitted record count across the saved shop-demo load phases for seeds 1 and 2 (2,304 spans + 1,212 logs + 72 scalar points/sec), and a second **estimated**, not measured, 60 fps x 50 values/frame pipeline budget of 3,000 scalar points/sec alongside that same shop span/log budget. Use serving defaults, 30-second offered windows, seeds 1–5, durable acknowledgments and verified counts; report pass/fail for every profile and seed, including unsent work. Measurement boundaries, averages and peak counts are in benchmark/results/m4b-workload-rates.json. This is an explicit workload-based target change, not a pass against the original targets.

### M2: Web sample app and seeded failure #1

**Tasks**
- Build `samples/shop` in Go: a frontend, `checkout-service`, `inventory-service`, and Postgres, via docker compose, instrumented with `otelhttp` and OpenTelemetry's Go SDK (exporting OTLP to Signal Scout).
- Add a load generator producing configurable checkout traffic with realistic variation (cache hits/misses, varied payload sizes).
- Add a failure injection switch via env vars. Seeded failure #1: `DB_POOL_SIZE=5` (pgxpool max connections) under concurrent load, causing pool-acquire timeouts in about 3% of requests.
- Record the environment fingerprint (commit SHA, versions, config) as resource attributes.
- Script `task shop-demo`: start the stack, generate 40 normal runs, enable the failure, generate more.

**Acceptance**
- After the demo script, `signalscout runs list --workload "POST /checkout" --outcome fail` shows failing runs, and passing runs exist from before the change.
- A failing run's spans include the slow pool-acquire step and a retry event.

### M3: Diff engine v1

**Tasks**
- Implement `internal/stats`, feature extraction, alignment, and the latency, presence/absence, count, and config difference types from section 6.
- Implement baseline selection and the insufficient-baseline case.
- Implement ranking, redundancy collapsing, and decoy checks.
- CLI: `signalscout diff <failing_run_id> [--baseline-n 40] [--baseline-before <run-id or RFC3339 time>]` printing the ranked list in the explanation format, with optional JSON output. Persist the selection mode and exact baseline IDs.
- Write `docs/diff-scoring.md`.
- Property tests with `rapid`: scores always in [0, 1]; scores monotone in effect size; adding identical baseline runs never raises the score of a feature that is common in baseline.

**Acceptance**
- On the M2 sample with a cutoff at the start of the failure phase, the pool-acquire latency outlier, the retry presence difference, and the pool-size config difference all rank in the top 3. The cache-miss decoy ranks below them. Also report default-mode rankings on the same failing run: for a late failure the recent passes also have pool=5, so the pool-size difference correctly disappears. Validate both modes on seed 1 and a new seed 2, without changing documented scoring constants to fit the demo. Confirm flat pool config and cache misses in passing runs. Context metric/log differences remain M4b.
- Diff computation for a 40-run baseline completes in under 1 second on a laptop.

### M4: Minimal viewer

**Tasks**
- Pages (served by `signalscout serve`): run list (filter by workload, outcome, time), single-run timeline (span tree with durations, events as markers), diff view (ranked differences with drill-down to underlying spans and the baseline distribution).
- Evidence IDs are clickable and jump to the item.
- Keep it simple: `html/template`, embedded assets, no JS framework.
- Serve the viewer from the existing `serve` listener (127.0.0.1 by default). Use auto-escaped templates, embedded assets, no external requests, and a strict Content-Security-Policy. Never treat stored telemetry as trusted HTML or script.
- Run lists have workload/outcome/start-time filters, bounded time windows, and cursor pagination. Run pages show the span tree timeline, event markers, metadata, env and config. Offer recent and explicit before-phase baselines side by side, including mode, count, IDs, scores, raw rates and linked evidence. Numeric differences include inline SVG baseline distributions with the failing value marked.
- Viewer reads use the read-only pool. On-demand diffs persist only through the serving store's existing writer queue. No context logs/metrics panels, service overview, search pages or ingest tuning in M4.

**Acceptance**
- From the run list, a user can open the failing run, open its diff, and click a difference to reach the span in the timeline, all in the browser.
- `httptest` navigation tests walk run list -> run page -> diff view -> evidence link and assert the destination anchors exist, including span/event/config links. Tests cover filtering/pagination, escaped telemetry, CSP, and persistence through the existing writer.
- `docs/viewer-walkthrough.md` gives exact PowerShell commands and browser steps. The owner will capture screenshots; a recording is not required from the agent.

### Before M4b: planned ingest performance task

- The unchanged pre-M3 preflight missed all separate targets: minimum/median spans 3,661/4,128 per second (target 5,000), logs 1,650/1,754 (target 5,000), scalar metric points 1,643/1,704 (target 10,000). Raw measurements remain in benchmark/results/m3-preflight-ingest.json.
- Before starting M4b, profile the native intake/storage path, including decode, commit, run finalization, startup backfill and context-link storage growth; make separately tested performance changes and remeasure with the unchanged per-signal method and fixed seeds. Preserve every result and report remaining misses. Do not perform this tuning during M4.

### M4b: Telemetry explorer and context telemetry

**Tasks**
- Implement `internal/telemetry`: a query layer over spans, logs, and metrics with filters (service, name, attribute, severity, status, duration range, time range, trace ID) and pagination.
- Viewer pages:
  - **Service overview**: per service and route, request rate, error rate, and p50/p95/p99 latency over a chosen time range.
  - **Span search**: filter and open any trace, even if it is not a classified run.
  - **Log search**: text and severity search, template grouping with counts, and a "logs for this trace" link.
  - **Metrics browser**: pick a series, view it as a simple line chart (inline SVG or canvas, vanilla JS).
  - **Run context panel** on the run page: correlated logs and metrics for the run's window, with the failure marker shown on the time axis.
- Implement the **context metric** and **context log** difference types in the diff engine (section 6, items 7 and 8) and show them in the diff view.
- Query results must be bounded (limits and time ranges are required) so the explorer never scans the whole database by accident.

**Acceptance**
- On the M2 sample, the explorer shows request/error/latency by service, and searching logs for the failing trace ID returns its logs.
- The run context panel shows the connection-pool gauge (or equivalent) for the failing run's window.
- Context difference types have table-driven and property tests, and decoy templates rank below true causes.
- Explorer queries over 1M stored rows return in under 500 ms for typical filters (record the numbers).
- Validate both saved M2 seeds without tuning scoring or regenerating data to fit ranks. The owner delegated handling the discovered seed-1 sampling gap: retain and report it as an explicit data-limited acceptance exception. Seed 1's 1 Hz in-use gauge recorded only zeros through the padded failure window, so no in-use difference can be claimed there. Demonstrate the in-use difference on seed 2 and controlled fixtures, and report absent ranks, incidental higher-ranked context and both baseline modes honestly. This replaces the expectation that the saved seed-1 gauge must surface despite its missing measurement; it does not turn that miss into a successful detection.

### M5: Benchmark harness and web failures

> Superseded by the trimmed, pre-registered M5 above; not started. Original breadth is retained for history.

**Tasks**
- Define the seeded-failure format in `benchmark/failures/<id>/`: `failure.yaml` (domain, root-cause label, expected evidence IDs or path keys, injection config), plus the injection config or code.
- Label expected evidence as cause/mechanism or symptom. Retry and acquire-presence differences are failure symptoms; configuration and context differences speak to mechanism. Latency shows where time was spent and is not by itself proof of a cause. M5 scoring must distinguish these labels instead of counting every high-ranked symptom as a diagnosed root cause.
- Build `cmd/ssbench`: for each failure, generate 20 to 50 passing runs plus failing run(s) against a Signal Scout instance, run the diff engine, and score top-1/top-3/top-5 and precision@k.
- Add workload variation noise so baselines are not artificially clean.
- Seed 6 to 8 web failures, for example: slow query under load, retry storm, misconfigured timeout, race between two services, dependency version regression, pool exhaustion (from M2), cache stampede, slow downstream dependency.
- Include at least 2 failures whose root cause is visible **only in context telemetry**, not in the failing run's own spans (for example, CPU saturation from a noisy co-located process, a memory-growth gauge preceding GC pauses, or a warning log from a neighboring service).
- Support an **ablation flag** that disables context telemetry in the diff engine, so the benchmark can measure its contribution.
- Output machine-readable results to `benchmark/results/` and a generated markdown summary table.

**Acceptance**
- `task benchmark-web` regenerates runs and produces a results file with top-k accuracy per failure.
- At least one failure is documented where the diff engine misses, with an analysis of why.

### M6: ML integration and failures

> Superseded: the ML adapter role is absorbed into the general helper. Training-specific examples remain optional and unstarted.

**Tasks**
- Build `adapters/ml-python` (`signalscout_ml`): a small Python context manager that sets the run ID, emits spans for data loading and training steps, emits metrics (loss, lr, grad norm), emits events (batch drawn with shard ID, checkpoint saved), and records dataset hashes and model references as artifact refs. It exports to Signal Scout via OTLP only.
- Build `samples/trainer`: a tiny model on a synthetic or small public dataset, CPU-only, finishing in a few minutes.
- Implement metric divergence (change-point) and ordering-change difference types in the diff engine.
- Seed 6 to 8 ML failures, for example: silently dropped data shard, NaN from a bad batch, learning-rate schedule bug, train/eval leakage, non-deterministic dependency update, corrupted re-exported shard.
- Extend `ssbench` to run the Python trainer as a subprocess for ML failures.

**Acceptance**
- The NaN-from-bad-shard example ranks the loss divergence, the shard-dominance difference, and the dataset-hash difference in the top 3.
- `task benchmark-ml` produces results in the same format as the web benchmark.

### M7: Agent layer and verifier

> Superseded input/report semantics are in section 7 and the revised order; not started.

**Tasks**
- Implement prompt building from the ranked diff, the structured output types, the `LLMClient` interface, an Anthropic-backed client, and the mock client.
- Implement the deterministic verifier and CLI: `signalscout explain <failing_run_id>` prints the hypothesis, claims with citations, the experiment, ruled-out items, and per-claim verification status.
- Implement two baselines for the benchmark:
  1. **Log-grep agent**: gets the failing run's raw logs/spans only, no historical baseline.
  2. **Logs-added-after-the-bug agent**: gets only runs generated after the failure begins, simulating logging added post hoc.
- Add agent scoring to `ssbench`: does the stated hypothesis name the true root cause (matched against labeled root-cause keywords and evidence IDs, with the rubric documented in `docs/benchmark.md`)? What is the citation validity rate?

**Acceptance**
- With the mock client, tests cover: all citations valid, a fabricated citation flagged, a misquoted number flagged.
- The benchmark reports root-cause hit rate and citation validity for the Signal Scout-grounded agent vs. both baselines.

### M8: Overhead, redaction, and write-up

> Privacy minimum moves before real data into P0. Remaining measurements stay later; not started.

**Tasks**
- Measure scalar-point bytes/record and hourly storage specifically: the mixed 1M fixture's roughly 1.1 KB/record extrapolates to about 12 GB/hour at 3,000 points/sec before retention, but is not a scalar-only measurement.
- Overhead measurements on the web and ML samples: added request latency p50/p99, CPU, memory, and bytes stored per run, and storage growth per hour at a fixed ingest rate with the default retention policy, with sampling on and off.
- Redaction: scrubbing rules for common secrets and PII (tokens, emails, card-like numbers, auth headers, SQL literals) applied at intake, before storage. Plant known secrets in sample traffic and measure the leak rate.
- Finalize `docs/run-spec.md`, `docs/adapter-guide.md` (including the robotics design note: why spans fit high-rate streams poorly and how the spec handles it with metric streams, event markers, and linked bag files), and `docs/benchmark.md`.
- Adapter conformance suite in `adapters/conformance/`: OTLP fixtures plus a checker that validates an adapter's output against the run spec. Run it against the web and ML adapters.
- README with a quick start, architecture diagram, and headline results table.

**Acceptance**
- Overhead report with real numbers committed to `benchmark/results/`.
- Redaction leak rate reported (target: 0 leaks on the planted set; report honestly if not).
- A new contributor can follow the README to reproduce the web demo and one benchmark run.

## 9. Benchmark metrics (what gets reported)

| Metric | Definition |
|---|---|
| Top-k accuracy | Fraction of seeded failures whose true root-cause difference appears in the top k (1, 3, 5) of the ranked diff |
| Precision@k | Fraction of the top k differences that are true causes or direct evidence of one |
| Context ablation | Top-k accuracy with context telemetry on vs. off, especially on the context-only failures. This measures whether collecting broad telemetry actually helps. |
| Agent root-cause hit rate | Fraction of failures where the agent's hypothesis matches the labeled root cause, for Signal Scout-grounded vs. log-grep vs. logs-added-after |
| Citation validity rate | Fraction of agent citations that exist and support the claim |
| Overhead | Added latency (p50/p99), CPU %, bytes per run, with and without sampling |
| Redaction leak rate | Planted secrets surviving intake / planted secrets |

Rules for honest reporting:
- Fix random seeds and publish them. Report mean and spread across repeated generations (at least 5 seeds).
- Include failures where the engine does poorly and say why.
- Have a few failures written without looking at the engine's feature set, to reduce the risk of tailoring the tests to the tool.
- Never edit results by hand. Tables are generated by code.

## 10. Working agreements for Codex

- **Read the relevant section of this plan before each task.** If something conflicts with reality, update `plan.md` in the same change and explain why.
- **One milestone at a time.** Small commits, each with tests. Do not start M(n+1) until M(n)'s acceptance criteria pass.
- **No scope creep.** If a feature is not in a milestone, do not build it. Add it to `docs/ideas.md` instead.
- **Tests first for scoring and verification logic.** These are the credibility core of the project.
- **Determinism.** Seed all randomness. Benchmarks must be reproducible from a clean checkout with a single `task` command.
- **Always run `go test -race ./...`** before declaring a task done.
- **Explainability over cleverness.** Prefer simple statistics whose output can be explained in one sentence.
- **No secrets in the repo.** LLM keys come from environment variables; CI uses the mock client.
- **Python stays on the client side.** The Python adapter and trainer talk to the Go server over OTLP only, and never read or write the database.
- **When unsure, add a question to `docs/open-questions.md`** and continue with the most conservative choice.

## 11. Historical first task for Codex (completed; superseded by revised order)

Start with **M0**, then proceed to **M1**. After M1 is complete, stop and summarize:
- what was built,
- the measured ingest throughput,
- any deviations from this plan,
- open questions for the project owner.

## 12. Revised definition of done

- Low-effort supported integration with explicit coverage and app-specific helpers.
- Trace a selected output to declared inputs/decisions/config and rejected options.
- Mark a problem, keep unknown/disagreement, select reference examples from one system.
- Check declared invariants without a baseline; compare cohorts with honest sample sizes.
- Evidence-linked tiered reports and developer-controlled falsifiable experiments.
- Pre-registered 4–5 case evaluation with at least 3 real observed problems, retained misses.
- Measured capture/privacy/storage/overhead and native Windows reproducibility.

### Historical definition of done (superseded breadth; preserved)

- Web and ML sample apps, each with at least 6 seeded failures.
- A telemetry store and minimal explorer over all traces, logs, and metrics, with retention.
- Diff engine with documented scoring, evaluated by top-k accuracy over multiple seeds, including a context-telemetry ablation.
- Agent layer with a deterministic verifier and a measured comparison against two baselines.
- Overhead and redaction numbers from real measurements.
- Published run spec, adapter guide, and benchmark methodology.
- One command per demo and per benchmark that works from a clean checkout.

## Implementation decisions and deviations (maintained by Codex; recorded after M1)

- Owner-authorized declared relations (2026-10-22): decoded escaped selectors
  opt in by profile; integer/boolean categories retain existing scoring formulas.
  Bounded cohort reads now default to 100,000 span/event rows (explicit maximum
  1,000,000), rejecting the entire request with population/scope details at a cap.
  Optional version-1 numeric/boolean relation trees check every matching recorded
  decision without a baseline; relation selection pins counts, declaration and
  native violation witnesses in existing cohort JSON. An absent version invariant
  is permitted for rule-only profiles; partial declarations remain invalid.
  Existing one-decision rankings, median/tail constants, diff types and trace
  behavior remain; no migration. First-divergence comparison is proposed only.
  See docs/declared-relations.md for exact grammar, scope and limitations.

> These record what was actually built. Where this section conflicts with the design sketches above (for example the schema in section 5), this section and the migrations in `internal/store/` are authoritative.

- M4b's first plan tests exposed the missing service/time path for spans and metric points. Migration 008 adds a partial (resource_id,kind,ts_ns) index for those two kinds only, retaining migration 007's resource-service, log-resource/time and trace/time paths. Explorer filters require an explicit bounded range and limit plus a service, explicit no-service partition, or trace; the no-service partition uses resources_service IS NULL for independent host-only telemetry browsing. Query-plan tests reject signal scans and require indexed time bounds. Run context uses the existing run/rule interval indexes because host-only resources deliberately have no service name. No throughput experiment from the parking lot is implemented.
- M4b's full race suite exposed the sample acquire timer starting before its span. Span setup could consume part of the timeout outside the recorded wait, failing the unchanged duration assertion. Start the span first, then create the acquire deadline from that span context. The timeout, retry behavior and test assertion are unchanged; the same race regression passes twenty repetitions after the correction. Saved M2 data and scoring constants are not rewritten.
- M4b context scoring is versioned as v1+context-v1 with a persisted retrospective-config-v1 window policy. Metrics use configured lookback through run end (never future lookahead), config-compatible scalar gauges/non-monotonic sums, ten observed baseline windows and two failing samples. Logs use actual run intervals. Existing M3 constants are unchanged; a fixed 1% unit-relative numeric floor/scale handles arbitrary metric units. Strongest max/mean/last summary is ranked per series, with other statistics related; log presence/count duplicates share one rank. Missing metric windows are unknown, and overlapping baseline windows are explicitly correlated. Policies and reasons were committed in docs/context-differences.md before demo ranking. CLI and viewer comparisons load bounded context on the same read snapshot and persist through their existing writer paths; pure M3 comparison remains for historical validation.
- The M4b viewer adds explicit bounded explorer forms, service/route statistics, span/trace and log navigation, template counts, scalar SVG charts and a padded run context panel with rule/source provenance and failure markers. Stable keyset pages and inclusive trace seeks keep a selected span reachable beyond the first page. Context evidence anchors resolve on run pages; templates and SVG geometry remain escaped/local under the existing strict CSP. Comparisons cap at 200 baselines/50,000 records and reject oversized inputs. The historical task diff-demo uses structural-only analysis so M3 acceptance remains reproducible; current CLI and viewer analysis include context. No M5 ablation CLI, dashboard builder, alerting, or query language is added.
- Saved-data M4b race navigation exposed the unchanged 10-second request deadline while adding both context comparisons. Its CPU profile attributes 69.14% to context reads, primarily SQLite. A single-query provenance aggregation and per-run resource reader still missed the deadline; their patches and failure logs are retained. The final reader batches overlapping windows inside one read snapshot, resolves resources/rules before points, reads each physical record once, and retains each run's exact interval/provenance. Disjoint intervals are not widened across gaps; late updates remain visible on the next snapshot. Resource metadata uses existing migration 003 host/instance indexes (unary + removes TEXT affinity while text-only guards preserve matching). Plan tests reject resource and signal scans. No ingestion path, scoring constant, deadline or test assertion is relaxed.
- M4b hosted validation initially stopped before project checks because Task setup exhausted GitHub's shared unauthenticated API limit on both operating systems. Pass the existing job token through the action's supported repo-token input and pin Task 3.54.0 to match native validation. No check is skipped or weakened.
- M4b validation retains both original M2 datasets and all sampling gaps. Both saved-data race walkthroughs pass unchanged deadlines, resolving 513 evidence links for seed 1 and 379 for seed 2. Controlled tests cover both context difference types, prior-phase contamination, sparse/missing samples and common-template decoys. Native read optimization preserves all 36 diff identities; maximum read/compute/save time is 67.78 ms. The nine actual explorer API cases over 1M records have five-repeat maxima at or below 5.4773 ms with explicit time bounds and limits. Seed 1's missing pool-in-use sample remains the documented acceptance exception; seed 2's representative pre-change pool metric ranks #3 behind config and retry. Recent baselines correctly remove it. Full details and incidental higher ranks are generated in benchmark/results/m4b-summary.md. M5 is not started.
- After migration 008, unchanged serving-default combined trials pass all five shop-budget seeds (2,304 spans + 1,212 logs + 72 scalar points/sec) and miss all five estimated mixed-pipeline seeds (same spans/logs + 3,000 points/sec). The command exits nonzero and retains verified results. Pipeline minimum/median rates are spans 1,548/1,800, logs 1,065/1,208 and points 2,543/2,731 per second. The earlier revised-profile 3/5 outcomes and original 5k/5k/10k misses remain preserved; no target, durability setting, scoring constant or ingest knob is relaxed. This does not establish sustained capacity or attribute the outcome change solely to the extra index. The 1M fixture occupies 1,217,073,152 bytes after the index, recorded separately from the older storage estimate.

- The pre-M4b performance task originally stopped with the combined gate unmet, before the owner authorized M4b and replaced that gate as recorded in M1 acceptance. Final unprofiled serving-defaults mixed-load five-seed minimum/median rates are spans 1,308/2,317, logs 1,385/2,334 and scalar points 2,651/4,652 per second, missing all simultaneous 5k/5k/10k targets. Separate diagnostics pass at 18,247/22,222, 17,303/19,347 and 17,565/19,760 respectively, without establishing combined acceptance. Seed-1 startup improves from 67.634 s native / 455.885 s race to 0.460 s / 8.460 s for first upgrade and 0.002172 s for an unchanged native restart. On the seeded real-schema 1M-record fixture, bounded service/time, trace and template-count queries have five-repeat maxima of 25.495 ms, 0.541 ms and 35.602 ms. The file occupies 1,149,222,912 bytes, with 51,562 physical scopes (16,056,320 bytes including their primary-key b-tree) representing 1,876,896 links and 3,484,277 provenance rows. Whole-history context enumeration still takes seconds. Profiles identify durable SQLite commit/Windows file I/O as the remaining bottleneck; cleanup is not a demonstrated hot spot. See docs/performance.md and the generated results summary for all attempts, limitations and the no-go recommendation. No target, durability setting, timeout or correctness assertion was relaxed.

- The pre-M4b performance task supersedes the historical context materialization notes below: migration 005 replaces evidence-by-run `context_links`/`context_matches` tables with compatible views over `context_scopes` (one interval/value per run, qualifying span resource and matching rule). Raw signals, resources, run snapshots, ownership, outcomes, evidence IDs and all provenance remain readable. Scope updates share the run transaction; ambient inserts need no context writes. A persisted padding checkpoint skips history on unchanged restart; first upgrade or changed padding rebuilds only closed run/resource scopes. No timestamp cursor can skip out-of-order evidence. Migration frees the old b-tree pages for reuse; SQLite does not automatically shrink the existing file. Measurements and remaining throughput misses are recorded in docs/performance.md. At that stage the target was simultaneous 5k/5k/10k capacity; the later owner override replaces it as recorded in M1 acceptance. Separate historical rates remain diagnostic.
- The performance task also replaces per-row signal executions with bounded 128-row parameterized inserts, without changing HTTP batch size, transaction acknowledgment or ownership checks. The fixed driver prepares SQLite bytecode on each execution even through database/sql prepared statements. Raw records and normalized fields remain JSON TEXT; WAL, synchronous NORMAL and the single-writer design are unchanged. Five-seed results and unsuccessful combined trials are retained; separate throughput does not establish simultaneous acceptance.
- The 1M-row preflight exposed a reversed join order in the shared-key context views. Migration 006 uses explicit scope/resource/evidence join order so every rule uses the indexed resource and time interval, preserving the exact logical sets. This is tested against SQLite's query plan; a global logical count is measured separately from bounded explorer-style queries. No context UI or difference type is added.
- To reduce the measured flush/write cost, the single writer coalesces at most four already-queued intake jobs without adding a timer wait. Each job retains its receipt time and savepoint; successful acknowledgments wait for the shared commit, job errors roll back only their savepoint, and commit failure acknowledges none. Maintenance and diff jobs remain ordering barriers. WAL, synchronous NORMAL, default checkpoint policy, queue bounds and shutdown draining are unchanged. Tests cover commit failure recovery and cross-job ownership conflicts.
- Migration 007 adds only the lookup indexes demonstrated necessary by the 1M-row SQL preflight: resource service name, span/log trace ID plus time, and log resource plus time. Existing run/context/retention indexes remain required. Service/time, trace-ID and log-template queries retain explicit time bounds and LIMIT 100; their measured plans and five-repeat timings are recorded without building the M4b query API or viewer. Additional write/storage cost is included in a new unchanged ingest run and file-size measurement.

- The owner namespace is unspecified, so the provisional module is `github.com/signalscout/signalscout`. Change it to the real GitHub path before publishing.
- CI uses Go 1.27 (within the Go 1.22+ requirement) and golangci-lint v2.14, matching the native development tools. Lint checks formatting without changing files.
- M1 adds resource identity, raw OTLP/scope metadata, ambient evidence, context associations, receipt timestamps and persisted grouping state. The review follow-up adds exact-resource, `host.name`, and `service.instance.id` context rules anchored only to span resources, with rule/source provenance in migration 003 and `runs show`. Host-only metric resources and padding boundaries have transaction tests. CLI padding now follows section 5; internal zero-valued padding explicitly keeps the unpadded interval used by the unchanged ingest benchmark.
- Span IDs include trace ID; event and metric IDs use canonical content hashes so valid records cannot overwrite unrelated evidence. Missing web roots yield degraded, and histogram/summary point payloads are retained without scalar flattening.
- The three signal collections use indexed common storage with SQL views named `spans`, `events` and `metric_points`, avoiding three divergent retention/context implementations. Original OTLP records remain JSON TEXT. All writes, including cleanup, share one writer.
- OTLP JSON hex identifiers are converted to protobuf JSON byte encoding before protojson decoding. Official generated OTLP collector types bring gRPC/gateway modules transitively; no gRPC server or web framework is added.
- The maintenance scheduler is a separate goroutine that coalesces timer ticks. The sole writer performs the actual idle closure and retention transactions. Open receipt timestamps are tracked in memory and recovered from persisted state.
- Root selection and failure markers are deterministic across arrival order; unknown evidence timestamps do not widen a timed context interval. Resource identity normalizes top-level attribute order. Conflicting evidence ownership is rejected atomically.
- M1 ingest measurement and review: measured by `scripts/ingestbench` (`task benchmark-ingest`) with five deterministic seeds and separate span/log/scalar-point rates. Each HTTP acknowledgment follows commit; counts are verified independently. The methodology excludes startup, final closure and large-history context scans. Three unchanged review repetitions retained all target misses; the owner authorized proceeding to M2 without tuning despite those misses. Those repetitions do not establish a pass against the separate targets.
- To improve M1 throughput without changing correctness or durability, resources are inserted once per batch and stored once by content identity; reads restore complete metadata. Run-resource membership uses current-batch pairs instead of scanning stored history. Earlier failed measurements are preserved alongside the original run that passed the historical span-only criterion; it does not establish a pass against all current separate targets.
- M2 uses parameterized Postgres inventory reads with per-cart-line simulated work (`pg_sleep`), identical in both phases. Fixed-seed carts include cache and payload variation. The load pattern includes five bulk carts followed by three normal cache misses per 100 requests; only DB_POOL_SIZE changes from 20 to 5 to trigger real acquire timeouts, with one retry. Failure frequency is measured, not synthesized.
- M2 pins SDK 1.28/logs 0.4/otelhttp 0.53 and pgx 5.7.1 for a compatible Go 1.22+ sample. Providers are explicitly wired, including correlated OTLP logs and pool gauges. These are the dependencies already allowed by section 2.
- M2 integration exposed a finalization timeout caused by repeatedly joining all retained context on every batch/tick. Operational attachment now uses changed run IDs/new ambient evidence IDs; one atomic startup backfill upgrades prior data and rebuilds links for changed padding. The 45-second demo verification limit and evidence checks were not relaxed. The three review measurements above predate this functional fix; they are preserved and are not claimed as measurements of the final M2 code.
- `task shop-demo` runs a native Go orchestrator/intake and a Compose shop (three Go services, Postgres 17.6, Collector contrib 0.147). The native compiler cross-builds the service binary; containers never open the Windows SQLite file. Each invocation uses a fresh isolated database, records its repository-relative path in `benchmark/results/shop-demo.json`, stops exporters/containers/intake on completion, and leaves the database for native CLI inspection. Use `--db` with that recorded path from the repository root when checking M2 acceptance. `.cache/` is ignored.
- Collector hostmetrics load/memory scrapers run in Docker Desktop's Linux environment. Their dedicated pipeline adds only the shared logical `host.name=docker-desktop`; these are not physical Windows-host metrics. App metrics retain pool gauges and full service resources.
- M2 integration also caught the SDK logs 0.4 exporter's exact-URL-path behavior (posting to `/` returned 404). The sample explicitly sets `/v1/logs`; a real SDK-to-intake regression test requires all three signal kinds and trace-correlated log body/template preservation. The evidence check was not removed.
- M2 acceptance passed on native Windows with Docker Desktop using seed 1 and source revision e0097ed (clean sample sources): 40/40 baseline passes, then 9/300 loaded failures (3.00%). The recorded failing run has approximately 101 ms pool-acquire spans, a retry event, trace-correlated logs, a pool=5 resource gauge, and a system.memory.usage point from a host-only resource linked by host.name. Raw generated evidence references are preserved in benchmark/results/shop-demo-seed-1.json. The stack was stopped and the native database closed successfully. That implementation stopped at M2.
- M3 v1 fixes constants in docs/diff-scoring.md before demo ranking. Duration statistics condition on observed paths (minimum 10 observations); missing paths are zeros only for counts. Count/presence duplicates and exception annotations confined to the same slow branch become Related evidence, retaining their scores and IDs. Config remains flat; environment scoring excludes identity/control/SDK metadata while preserving it in storage. These conservative feature and redundancy choices are documented and tested, without changing the sample workload or fitting constants to its database.
- M3 selection uses one read-only transaction for closed earlier passes and run-owned spans/events, excluding context. The CLI persists both successful and insufficient-baseline results through the store's writer queue. Its result-only open mode requires an existing schema and skips intake recovery, context backfill, scheduling and shutdown finalization so analysis cannot change run state or context padding. CLI elapsed time includes selection, computation, persistence and connection closure; stored computation results omit this caller-side timing (zero), which is filled in the returned output after persistence. As with the existing local single-serving-process agreement, run standalone persisted CLI analysis after stopping intake; in-process analysis can use the serving store's SaveDiff queue.
- M3 adds the narrow `task diff-demo` validation helper for an existing M2 database, with generated JSON/Markdown rankings for every failing run in both modes. This does not implement the multi-failure M5 harness. A fresh `task shop-demo -- -seed 2` passed M2 evidence checks with 40/40 baseline passes and 9/300 failures (3.00%), retaining separate seed-1 and seed-2 reports and repository-relative database paths.
- M3 acceptance passed on both seeds without changing scoring constants or the sample workload: all nine failures per seed rank DB_POOL_SIZE config #1, retry presence #2, pool-acquire latency #3 and cache.miss #4 against the pre-change window. For the latest failure, default mode ranks retry #1, latency #2 and cache.miss #3, with no pool-size difference because all 40 recent passes have pool=5. Earlier failures retain a mixed default pool history and a config difference. Across 36 comparisons, maximum elapsed times are 30.969 ms (seed 1) and 41.281 ms (seed 2), including persistence; seed 2 has the same ranks and slower timing. Generated reports are benchmark/results/diff-seed-1 and diff-seed-2 (JSON/Markdown). M3 is complete; no M4/M4b implementation was started.
- M4 adds a same-listener viewer with escaped templates, embedded CSS, inline SVG and no JavaScript. CSP disables scripts/external requests. Lists use a seven-day default/31-day maximum window and 25-row default/100-row maximum cursor pagination, backed by a start-time index. Timelines cap spans/events at 5,000 per run; viewer comparisons cap baseline runs at 200 and total structural records at 50,000, rejecting oversized inputs instead of truncating scoring. Request deadlines and two computation slots bound viewer work.
- M4 uses explicit POST comparison and the serving store's SaveDiff queue; GET navigation performs no writes. Run changes between read and persistence return a nonfatal conflict; genuine storage failures still reach the serving lifecycle. Before-phase suggestions require the explicit M2 phase/workload/config/revision fingerprint, with a user-editable run-ID/RFC3339 cutoff; unknown phases require explicit input. Query-encoded run IDs and hashed evidence anchors preserve identifiers containing URL syntax. Numeric display uses the same comparison snapshot and feature aggregation without changing M3 scoring.
- M4 navigation acceptance passes in the self-contained httptest race suite and native saved-data checks: seed 1 resolves 46 evidence links and seed 2 resolves 43, including both persisted baseline panels. The extra seed-1 saved-data test under -race failed at Go's unchanged 10-minute timeout in startup context backfill, before HTTP navigation; its portable log is benchmark/results/viewer-m2-seed-1-race.txt. This failure is retained for the planned pre-M4b performance task; no timeout, assertion, scoring constant or startup behavior was relaxed. Native saved-data test bodies took 63.46 s and 93.69 s including startup. M4 uses the written PowerShell walkthrough instead of an agent-recorded screen capture; the owner will capture screenshots. No M4b views/difference types or ingest tuning were implemented.

Slice 1 measured boundary: base cases find the two declared input mechanisms at
#2/#3; decoy-heavy pushes them to #9/#10; minority stale age is missed by the
median while the invariant flags 20/100 problem frames. All 15 cases have complete
capture. See benchmark/results/lane-01.md and lane-01-shop-regression.json.
All 72 saved shop comparisons retain identical differences/scores. These are
constructed cases, not independently verified sequential histories or real-world
accuracy. The first evaluation and confirmatory rerun remain on record.

### Lane-02 Stage 1 checkpoint (owner-authorized synthetic slice)

Docs-only contract amendment 6a3716b precedes implementation. The narrow extension
adds opt-in symptom roles and a separate reference-tail association list to the
existing cohort report/CLI, without changing median constants, lane-01 artifacts,
structural/context diff types or migrations. Development seeds 101/103 pass the
deterministic score/count/invariant expectations; within-range age remains missed.
Retain the benign finite-support premise mismatch and recording-cost misses in
docs/lane02-checkpoint-a.md. Sources are pinned in the lane-02 source-freeze
manifest; reserved seeds remain ungenerated/unscored pending explicit owner GO.
No M6–M8, P0–P4 production work or conditional comparison starts.

### Recorder capture-loss checkpoint (owner-authorized; no behavior fix)

Lane-02 is closed as attempted/incomplete due to capture loss. Preserve its lock,
original attempt, contract/spec/checksums/source freeze and published results.
The recorder-capture-loss branch adds bounded diagnostics, keeping transport,
queue/cooldown defaults and exported bytes unchanged. Five 101/103 development
repetitions complete 60 cases/96,000 acknowledged spans without loss. The original
initiating exception is not reproduced or recovered. A controlled post-commit
reply delay demonstrates timeout/cooldown amplification; a burst demonstrates
finite queue overflow. Costs still miss the unchanged 100/250 us budget. See
docs/recorder-capture-loss.md and benchmark/results/recorder-capture-loss for every
run, raw report, limitation and the proposed worker fix. STOP at Checkpoint A
pending GO FIX; no retry/accounting fix, lane-03 contract or production phase.
The earlier Stage 1 pending-GO statements above describe that historical checkpoint.

### M7 deterministic reporting first half — 2026-10-22

Owner-authorized partial implementation: version-one structured observation,
association and supplied-hypothesis reports compose existing relation,
completeness, explain-cohort, invariant and trace outputs. The read-only verifier
rebuilds scoped facts and checks citations, membership, quantities and tiers,
with explicit unsupported/unknown results and rates. No LLM client, prompt,
hypothesis generation, experiment executor, ranking change or migration is
implemented. Historical "M7 unstarted" text refers to the full agent milestone;
its deterministic first half is now built. Numeric assertions use structured
quotations rather than unreliable natural-language extraction. Synthetic authored
fixtures establish mechanics only; see docs/report-walkthrough.md and the dated
implementation note in docs/provenance-spec.md for exact limits.

### Doctor implementation note — 2026-10-22

Owner-authorized `signalscout doctor` adds a reporting-only local setup checklist
and optional version-one JSON file. It inspects binary metadata, Python tools and
dependency imports, existing file access and a transient loopback bind. A new DB
path warns without creating any file; creation/WAL permissions remain untested.
It prints a synthetic PowerShell example, installs/records nothing, and changes
no store schema, scoring or analysis behavior. See docs/doctor.md.

### Paired agent-session usefulness pilot design — 2026-10-25

Owner-authorized design-only parallel evaluation track:
[protocol](docs/paired-usefulness-pilot.md) and
[fresh-session prompts](docs/paired-usefulness-prompts.md).
DRAFT until reviewed and frozen; no evaluation execution authorized by this change.
Three paired fresh agent investigations measure correct diagnosis and setup-inclusive
costs. This is not a human diagnosis-time study, M5 completion, a replacement for
the existing 150-command hard-bug gate, or evidence of tool usefulness. Keep all
historical results and cause-naming tool-output attribution rules unchanged.

### Near-term scope narrowed after independent critique — 2026-10-25

Owner-authorized docs-only response: local Python investigation aid with no
established diagnostic advantage or causal-proof claim. Defer Node implementation,
LLM integration, value linking, first-divergence and other product expansion pending
the paired pilot; PR #31 remains unmerged design reference. See
[attributed critique response](docs/independent-critique-response.md) and
[prospective investment policy](docs/paired-usefulness-pilot.md).
For that separate pilot only, both arms read implementation source from first read;
original hard-test capture/access rules and 150-command gate stay unchanged.
Three valid pairs, >=2/3 correct tool diagnoses, no correctness regression and a
runtime-supported discordant solve or >=20% setup-inclusive elapsed advantage are
the prospective continuation bar. Incomplete is inconclusive; failure pauses
expansion; passing only supports another held-out evaluation. No runs launched.
