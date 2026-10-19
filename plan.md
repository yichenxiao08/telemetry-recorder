# Signal Scout: Build Plan (Go)

> This file is the source of truth for building Signal Scout. Read it fully before writing code. Work one milestone at a time, in order. Do not start a later milestone until the current one's acceptance criteria pass.

## 1. What we're building

**Signal Scout learns what normal runs of a workload look like, then explains how a failing run diverged, with every explanation linked to the evidence behind it.**

It targets intermittent failures: ones that pass on rerun, appear only under load, or show up after a change. The useful evidence is a comparison against passing runs, so history must be collected before the bug appears.

The deliverable is a working system plus a reproducible benchmark with real measurements. The benchmark is the centerpiece; the viewer is deliberately minimal.

### The core loop

1. Instrumented apps emit standard OpenTelemetry (OTLP) data.
2. Signal Scout groups telemetry into **runs** (one web request, one training job) and stores them.
3. Given one failing run and a baseline of passing runs of the same workload, the **diff engine** ranks what differed by how well it discriminates failure.
4. An **evidence-grounded agent** proposes a hypothesis from the ranked diff, citing evidence IDs. A **verifier** checks every citation.
5. A **benchmark** of seeded failures with known root causes measures all of this.

### Architecture

- **Go backend** (single binary): OTLP intake, run grouping, storage, diff engine, agent layer, verifier, viewer, CLI.
- **Python client-side pieces**: the ML adapter (`signalscout_ml`) and the sample trainer, because that is where instrumented ML code lives.
- **Go sample web app**: instrumented services plus Postgres.
- The Python and Go sides communicate **only over OTLP and HTTP**. Python never touches the database file.

### Non-goals (do NOT build these)

- A custom collector or database engine. Reuse the OpenTelemetry Collector and SQLite.
- A polished dashboard. Keep the UI minimal and functional.
- Robotics integration (only a documented adapter interface and a design note).
- Replay or rerun from saved inputs.
- An artifact store. Artifacts are references (URI + content hash) only; never copy large files.
- A human diagnosis-time study.
- Multi-tenant auth, user accounts, or cloud deployment.

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

A **run** is one execution of a **workload** with a start, an end, and an outcome.

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

```sql
runs(run_id TEXT PRIMARY KEY, workload TEXT, domain TEXT, started_at INTEGER, ended_at INTEGER,
     outcome TEXT, failure_marker TEXT, env TEXT /*JSON*/, config TEXT /*JSON*/, closed INTEGER)
spans(span_id TEXT PRIMARY KEY, run_id TEXT, parent_span_id TEXT, name TEXT, service TEXT,
      start_ns INTEGER, end_ns INTEGER, duration_ms REAL, status TEXT, attrs TEXT /*JSON*/)
events(event_id TEXT PRIMARY KEY, run_id TEXT, span_id TEXT, name TEXT, ts_ns INTEGER, attrs TEXT /*JSON*/)
metric_points(run_id TEXT, name TEXT, ts_ns INTEGER, value REAL, attrs TEXT /*JSON*/)
artifacts(artifact_id TEXT PRIMARY KEY, run_id TEXT, uri TEXT, content_hash TEXT, kind TEXT,
          redaction_status TEXT, retention TEXT)
diff_results(diff_id TEXT PRIMARY KEY, failing_run_id TEXT, baseline_spec TEXT /*JSON*/,
             created_at INTEGER, results TEXT /*JSON*/)
```

Add indexes on `(workload, outcome, started_at)` for runs and on `run_id` for spans, events, and metric points.

### Concurrency model

- SQLite allows one writer at a time. Use a **single writer goroutine** that receives batches over a channel and commits them in transactions. Intake handlers decode and enqueue; they never write directly.
- Reads (viewer, diff engine, CLI) use a separate read-only connection pool.
- Enable WAL mode and set a busy timeout.
- Intake must apply backpressure (bounded channel; return 503 with `Retry-After` when full) rather than grow without limit.

### Evidence IDs

Stable strings shown to users and cited by the agent: `span:<id>`, `event:<id>`, `metric:<run_id>/<name>@<ts>`, `config:<run_id>/<key>`, `artifact:<id>`.

## 6. Diff engine design

**Input:** one failing run (later: a set of failing runs) and a baseline set of passing runs of the same workload.
**Output:** a ranked slice of `Difference` structs, each with a score in [0, 1], a type, a human-readable explanation, and evidence IDs.

### Baseline selection

- Default: the most recent N passing runs of the same workload (N=40), optionally filtered to the same environment where possible.
- Configurable, and recorded in `diff_results.baseline_spec`.
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
- Document how to point an OpenTelemetry Collector (or an SDK directly) at Signal Scout, with a sample Collector config.
- CLI: `signalscout serve`, `signalscout runs list`, `signalscout runs show <run_id>`.

**Acceptance**
- Replaying recorded OTLP fixtures from `testdata/` produces the expected runs, spans, events, and outcomes (golden-file tests).
- Malformed payloads return 4xx and do not corrupt storage.
- Backpressure works: when the write queue is full, intake returns 503 (tested).
- Ingest throughput of at least 5,000 spans/sec on a laptop, measured with a Go benchmark or load script. Record the number; do not tune beyond this yet.

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
- CLI: `signalscout diff <failing_run_id> [--baseline-n 40]` printing the ranked list in the explanation format.
- Write `docs/diff-scoring.md`.
- Property tests with `rapid`: scores always in [0, 1]; scores monotone in effect size; adding identical baseline runs never raises the score of a feature that is common in baseline.

**Acceptance**
- On the M2 sample, the pool-acquire latency outlier, the retry presence difference, and the pool-size config difference all rank in the top 3. The cache-miss decoy ranks below them.
- Diff computation for a 40-run baseline completes in under 1 second on a laptop.

### M4: Minimal viewer

**Tasks**
- Pages (served by `signalscout serve`): run list (filter by workload, outcome, time), single-run timeline (span tree with durations, events as markers), diff view (ranked differences with drill-down to underlying spans and the baseline distribution).
- Evidence IDs are clickable and jump to the item.
- Keep it simple: `html/template`, embedded assets, no JS framework.

**Acceptance**
- From the run list, a user can open the failing run, open its diff, and click a difference to reach the span in the timeline, all in the browser.
- A short screen capture of this walkthrough is saved in `docs/`.

### M5: Benchmark harness and web failures

**Tasks**
- Define the seeded-failure format in `benchmark/failures/<id>/`: `failure.yaml` (domain, root-cause label, expected evidence IDs or path keys, injection config), plus the injection config or code.
- Build `cmd/ssbench`: for each failure, generate 20 to 50 passing runs plus failing run(s) against a Signal Scout instance, run the diff engine, and score top-1/top-3/top-5 and precision@k.
- Add workload variation noise so baselines are not artificially clean.
- Seed 6 to 8 web failures, for example: slow query under load, retry storm, misconfigured timeout, race between two services, dependency version regression, pool exhaustion (from M2), cache stampede, slow downstream dependency.
- Output machine-readable results to `benchmark/results/` and a generated markdown summary table.

**Acceptance**
- `task benchmark-web` regenerates runs and produces a results file with top-k accuracy per failure.
- At least one failure is documented where the diff engine misses, with an analysis of why.

### M6: ML integration and failures

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

**Tasks**
- Overhead measurements on the web and ML samples: added request latency p50/p99, CPU, memory, and bytes stored per run, with sampling on and off.
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

## 11. First task for Codex

Start with **M0**, then proceed to **M1**. After M1 is complete, stop and summarize:
- what was built,
- the measured ingest throughput,
- any deviations from this plan,
- open questions for the project owner.

## 12. Definition of done for the whole project

- Web and ML sample apps, each with at least 6 seeded failures.
- Diff engine with documented scoring, evaluated by top-k accuracy over multiple seeds.
- Agent layer with a deterministic verifier and a measured comparison against two baselines.
- Overhead and redaction numbers from real measurements.
- Published run spec, adapter guide, and benchmark methodology.
- One command per demo and per benchmark that works from a clean checkout.

## Implementation decisions and deviations

- The supplied file was named `plan (1).md`; `plan.md` is the working source of truth. The original is preserved.
- The owner namespace is unspecified, so the provisional module is `github.com/signalscout/signalscout`.
- CI uses Go 1.27 (within the Go 1.22+ requirement) and golangci-lint v2.14, matching the native development tools. Lint checks formatting without changing files.
- M1 adds resource identity, raw OTLP/scope metadata, ambient evidence, context associations, receipt timestamps and persisted grouping state to the initial schema. Context attaches only on exact resource and evidence-interval match; ownership stays separate.
- Span IDs include trace ID; event and metric IDs use canonical content hashes so valid records cannot overwrite unrelated evidence. Missing web roots yield degraded, and histogram/summary point payloads are retained without scalar flattening.
- The three signal collections use indexed common storage with SQL views named spans, events and metric_points, avoiding three divergent retention/context implementations. Original OTLP records remain JSON TEXT. All writes, including cleanup, share one writer.
- OTLP JSON hex identifiers are converted to protobuf JSON byte encoding before protojson decoding. Official generated OTLP collector types bring gRPC/gateway modules transitively; no gRPC server or web framework is added.
- The maintenance scheduler is a separate goroutine that coalesces timer ticks. The sole writer performs the actual idle closure and retention transactions. Open receipt timestamps are tracked in memory and recovered from persisted state.
- Root selection and failure markers are deterministic across arrival order; unknown evidence timestamps do not widen a timed context interval. Resource identity normalizes top-level attribute order. Conflicting evidence ownership is rejected atomically.
