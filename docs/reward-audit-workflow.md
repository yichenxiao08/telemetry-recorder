# Record once: declared reward audit

This bounded example captures a **prepared public Snake experiment**, then uses
existing native `relations` to compare declared reward/task observations. One
command handles configuration, preflight, owned-server lifecycle, all six frozen
policies, completeness, exact evaluation parity, analysis and saved evidence.
There is no separate investigator-authored rollup or manually assembled report
request. This is reusable orchestration for this exact v2 fixture contract, not
a general application launcher, automatic instrumentation or cause discovery.

## Prepared inputs

Run from the Signal Scout repository root after `task build`:

```powershell
go run ./scripts/rewardaudit --project 'C:\path\to\prepared-rlenv-study' --python 'C:\path\to\prepared-capture-env\Scripts\python.exe'
```

`--project` is the existing unblinded rlenv study root containing `study/harness.py`,
`study/spec.json`, `study/v2/spec-v2.json`, its artifact manifest, six checkpoints
and saved results. No checkpoints are downloaded, trained, selected or bundled.
The fixed v2 spec SHA-256 is
`edde69f07acc176e2976017b8c4097e40813830f80c678f191106d28e153146f`.
Its declared source hashes and all six checkpoint/result hashes are checked before
capture and again after completion. The manifest is a supplied provenance input,
not independent proof of upstream history. Only run trusted project code.

The interpreter must be a prepared **CPython 3.12.12** environment with the locked
Python adapter dependencies plus Torch `2.9.0+cpu` and NumPy `2.2.6`, as used for
the frozen evaluation. See the [adapter dependency setup](../adapters/provenance-python/README.md).
Do not copy a Windows venv, silently upgrade dependencies or install into a shared
environment to make a run pass. Probe preflight validates attachment/configuration
before each policy executes. Missing dependencies fail with retained logs; the
command installs nothing. It uses the selected Python directly, so uv need not be
on PATH at execution time. The command reads project Python files and executes
its evaluator: this is not a sandbox for untrusted models/code.

Optional `--binary` and `--adapter` select matching prepared engine artifacts;
defaults are `bin/signalscout.exe` and `adapters/provenance-python` in this checkout.
`--out` must name a NEW directory; an existing path, even empty, is rejected.
Default output is a fresh timestamped `.cache/reward-audit/` directory. Every failure
retains logs and stops; do not retry against that output.

## What is supplied, what is computed

The public capture launcher supplies four explicitly allowlisted scalar values:
apple event, death, exact configuration repeat, and the **actual reward of that
policy's condition**. Repeat means the post-transition body/direction/apple
configuration has already occurred within that episode; death transitions do not
count as repeats. These measurement meanings are example-specific declarations.
They are not inferred code dependencies. Policy/checkpoint/training/evaluation
seed context is captured at the episode boundary and retained in each DB's files.
The evaluator's outcomes are compared exactly to saved results for every episode.

The native relation engine counts every matching transition. The final table
copies its counts directly, without recomputing them in Python:

- Apple events: violations of the declared `apple == false` check.
- Alive steps: satisfactions of `death == false`.
- Repeats: violations of `repeated_configuration == false`.
- Positive reward on repeated nonprogress: violations of `actual_reward <= 0`
  when no apple, no death and a repeated configuration are recorded.

The last check is a developer-declared question, **not a rule the tool discovered**.
Not-applicable steps are shown separately, so an original policy with no repeated
nonprogress is not represented as proof of a universally satisfied rule. Unknown
or sampled relation records prevent this workflow from publishing a complete audit.
Transitions are not independent policy-training samples: this experiment has
three training-seed condition pairs and ten held-out episodes per policy.
No reward sum, episode-level pattern join or causal explanation is claimed.

## Saved evidence and a new question without rerunning

Each of the six policies has its own ten-episode SQLite DB, doctor, parity file,
completeness response, full native relations JSON and exact CLI arguments. An
`audit.json` records input/output hashes, per-policy counts and capture durations;
`audit.md` contains the comparison and witness evidence IDs/URLs. Each row includes
complete population and recorded-step denominators. Existing native relation
bounds remain: at most three violation witnesses are retained per row, with the
exact omitted-witness count; full per-run counts remain available.

**Viewer URLs are scoped to the database.** Start the viewer using the DB named
in that policy's section, then open its witness URL. A URL from another policy's
DB is not interchangeable. The workflow stops its owned server after each capture;
it does not leave six viewers running. Example:

```powershell
& .\bin\signalscout.exe --db 'C:\actual\audit\variant-11\capture.db' serve --listen 127.0.0.1:4318 --retention 0
```

The audit prints exact saved-data `relations` commands with that policy's time
bounds. To ask **how often any step received positive actual reward**, reuse that
command and change only its profile to the generated `profile-all-steps.json`.
This second declared question removes the repeated-nonprogress precondition. It
queries the same saved DB; no Python, checkpoint loading or evaluation rerun is
needed. No arbitrary query language or new ranking engine is introduced.

## Capacity and failure semantics

All six policies / 60 episodes are required. Captures use independent scopes/DBs
so native relations can analyze the complete population within existing limits.
Every policy must show ten complete persisted runs, final known scope population,
all selected probe calls captured, equal attempted/accepted/exported/acknowledged
counters, and zero losses/unknowns. Failed or incomplete capture is retained as
setup evidence and is not interpreted. Missing source/manifest/hash, changed
checkpoint, evaluation mismatch, unknown relations or limit overflow fails closed.

The default **12 ms pause per transition** is explicit pacing, not transparent
recording. `--pace-ms` accepts finite 0..100; zero is allowed only as an explicit
capacity experiment and does not guarantee delivery. Queue bounds are 1,024 retained
items / 8 MiB estimated bytes; these are not RSS guarantees. A 30-second drain and
15-minute overall command timeout are bounds, not assurances of capture success.
This can add several minutes to an evaluation that ordinarily takes seconds.
Record capture and preparation costs; do not infer developer-time savings.

The helper stops only its own child server. Cleanup is forced process termination,
not graceful Windows console shutdown; analysis occurs after acknowledged delivery.
Ephemeral port selection is not reservation, so an intervening bind can still fail.
No existing listener is stopped. SQLite artifacts, including any WAL/SHM files,
are retained together; do not copy only a live `.db` file and expect a valid snapshot.

This audit uses native relation reports. It does **not** raise the structured
report verifier's 10,000-record cap or claim the entire audit is a verified
structured report. Legacy synthetic capture is not a production privacy gate.
The previous exploratory/paired results and failures remain unchanged. This
workflow demonstrates recording and native checking of declared observations,
not causation, independent mechanism discovery or comparative usefulness.
