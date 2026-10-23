# WinError 10054 investigation, round 2

Date: 2026-10-23 (America/Toronto). One real, unblinded case with synthetic traffic only; this is not evidence that Signal Scout generally works.

PR #28 was confirmed merged. This branch started from clean origin/main at `40628db33ffdb3f88fa5dd599e8d2865437d21f9`. No tracked application, adapter, test, scanner, frozen result, module path or tag was changed. All scratch changes and evidence are under `.cache/flake-10054-r2`.

## Phase 0: why earlier capture was empty

No new test runs were used for this analysis. The earlier config explicitly set `capture_policy: policy.json`, `capture_profile: synthetic`, and `capture: {}` on both probes. Its entire policy leaf set was `{"output.lane":{"kind":"enum","values":["left","right"]}}`. Neither probe captured arguments/returns, so this did not expose the needed port, body, status or timing fields.

More importantly, producer `Policy.envelope()` unconditionally reconstructed the decision with inputs/output and `"detail":"redacted; identity and semantic metadata unknown"`. This discards identity/profile/step metadata even if scalar leaves are approved. `Policy.queued()` reconstructs items from trace/span/parent/start/end/error, sanitized name/attributes/events, omitting the manifest ledger reference. The prior strict intake filter also removed scope attributes; the old evidence showed a missing scope tail. Removing just intake filtering restored the scope but left a missing run-final marker. Thus three restrictions mattered: capture OFF, deliberately removed envelope metadata, and manifest bookkeeping lost on the producer path. It was not simply a missing scalar allowlist. This matches the P0 docs: IDs/semantic metadata are deliberately omitted, lineage is unknown; legacy mode is not privacy-safe.

## Phase 1: clean binary and isolated baseline

`git status --porcelain` was empty before building. The binary actually used by the unmodified test's `Serving` helper was rebuilt at `bin/signalscout.exe`, then copied under cache for collector/CLI use. `go version -m` confirmed the above revision and `vcs.modified=false` (Go 1.27.0). SHA-256:

```text
1db6a7662bfaac6805dcc9916d575e467a06cbceda73c22916804ea2253251c3
```

The native Python 3.12.12 virtual environment ran exactly 30 fresh isolated processes from the adapter directory:

```powershell
& .venv\Scripts\python.exe -m unittest discover -s tests -p test_privacy.py -k test_real_probe_intake_and_log_raw_copies -v
```

26 passed; 4 failed (13.3%), all WinError 10054 at the blocked `/api/annotations` response. Failure indices: 3, 15, 25, 26. All 30 reached the test body. Process durations in seconds: min 0.567, median 0.611, max 1.705, total 19.403. Per-run timestamps/durations/logs are in [phase1/results.json](../.cache/flake-10054-r2/phase1/results.json).

## Hypothesis before probing

[plain-reading-hypothesis.txt](../.cache/flake-10054-r2/plain-reading-hypothesis.txt) was written at **2026-10-23T21:05:10.020882Z**, before any round-2 probing. SHA-256:

```text
db597352410fb6f142cd8c957414d18d2c14ab0940011bda8ed74fab9a0b75b5
```

Reading found that strict P0 middleware writes 403 without explicitly reading the POST body. Successful readiness, intake and store-read operations precede the reset, making the old startup-port-gap hypothesis less persuasive for this late request. The new hypothesis is that early rejection/body/connection handling is associated with the reset. Prediction: changing only the blocked POST body to empty bytes, while retaining POST and the unchanged 403 assertion, yields zero resets in 30 isolated trials. Persistent resets would contradict that narrow prediction. This is not a claim about the exact TCP mechanism.

The suggested listener-hold example was not used: the current Go server cannot bind the same address while the temporary listener remains bound, absent socket handoff support. Instead, the source-reading hypothesis was tested with the scratch-only body perturbation. No tracked-code change was needed.

## Phase 2: ordinary scratch-only experiment

Only the blocked request's `json.dumps({"review": secrets}).encode()` was replaced by `b""` in a cached copy of test_privacy.py. All other source and the HTTPError/403 assertion were retained. The copied test still used the same clean server binary and strict P0 policy. The change is retained in scratch/change.json and test_privacy_scratch.py; it is not an application fix and does not exercise the original nonempty blocked payload.

**30/30 passed; 0/30 failed**, versus 4/30 in Phase 1. Process durations: min 0.550, median 0.602, max 0.708, total 18.607 seconds. The ordinary perturbation removed the observed flake in this sample. Sequential, nonrandomized arms and 30 trials do not establish that it will never recur or identify the server/network mechanism. See [phase2/results.json](../.cache/flake-10054-r2/phase2/results.json).

## Phase 3: scalar capture that can see

Targets and fields were specified beforehand in [capture-plan.txt](../.cache/flake-10054-r2/capture-plan.txt), from source reading, not probe-scan output. SHA-256: `e6428ae2fd8be0b789c3b514aa6baa6eca2e5c010769fa27e8415279eca96bee`.

The scratch harness retained the original test body. A private urllib facade observed only test-owned calls, avoiding exporter recursion. A scratch Serving subclass observed successful readiness and copied server logs; it did not inspect actual bind timing or port ownership. Probes:

- `r2_capture.main`: capture its controlled result dictionary only: failed, error_number, test_ms, startup_ms, ready, port, body_bytes, status, reset, request_ms.
- `r2_capture.observe`: allowlist scalar arguments kind, port, body_bytes, status, reset, elapsed_ms. No request/response objects, raw URLs, exception text, synthetic secret payload or host attributes were captured.

The outer recorder **omitted capture_policy/capture_profile and therefore left strict P0 for legacy mode**, as did its collector. This synthetic-only exception preserves decision identities and manifests. It is not a privacy-safe policy for real data. The test's own producer and target server retained their original strict P0 policy. No blanket real-data authorization follows. Existing probe capture remains OFF except the above explicit fields.

Every recording ran from the adapter directory via `python -m scout_probe`, with `--doctor`, `--manifest`, `--drain-timeout 30`, and the original queue defaults; no queue enlargement or sampling was used. Each process ran one original test body. The wrapper caught reset exceptions into an explicit failed scalar; consequently the outer CLI exits and root run outcomes are pass, and selection uses **output.failed**, not the outer run outcome. No successful test was relabelled a failure.

The stop condition was met at **19 recordings: 5 failures / 14 passes** (26.3%). Instrumentation changes timing, so this rate should not be compared as an equivalent treatment arm to Phase 1. Doctor reports: 19/19 completed drains, 19 built run manifests and 19 scope tails, 95/95 attempted ordinary spans acknowledged, 0 failed/dropped/unknown, no pending ledgers. Store completeness independently classified **19/19 complete**, 0 partial/unknown/absent/unassessed, 0 lost/unknown/detail-loss items. No replays or discarded test runs were used. Evidence: phase3/doctor-*.json and [all-completeness.json](../.cache/flake-10054-r2/phase3/all-completeness.json).

### Cohort/explain result and tiers

The same bounded profile/time window was used for both cohorts (2026-10-23T20:00:00Z through 2026-10-24T00:00:00Z; limit 100), requiring complete capture. Predicates selected output.failed=true (5 runs; excluded 14 predicate_false) and false (14 runs; excluded 5). Features were scalar readiness/reset/status, body bytes, port and timing. Epsilon and floor were 1 in the feature's stated unit. No engine scoring constants were changed or tuned.

The comparison is populated, but its status is **insufficient_evidence**: independent session IDs were not recorded (0 known problem/reference sessions, 19 unknown), so the engine retained observation tier and zero scores. Capture completeness is now solved for these runs; session independence is not. No IDs were invented after observing the result.

| Observation | Failing tests | Passing tests |
| --- | --- | --- |
| Ready | true, 5/5 | true, 14/14 |
| Reset | true, 5/5 | true, 0/14 |
| HTTP status | unknown/not received (stored sentinel 0), 5/5 | 403, 14/14 |
| Body bytes, median | 184 | 184 |
| Readiness duration, median ms | 183.329 | 183.923 |
| Blocked-request duration, median ms | 11.674 | 8.344 |
| Total test duration, median ms | 280.556 | 324.843 |
| Port, median | 59173 | 59237 |

These are descriptive observed values, not supported associations or causes. Status 0 means no HTTP status was received, never a valid HTTP response. Reset/status differences are selected failure symptoms. Numeric timing/port differences have no supported mechanism interpretation. Per-feature observed counts are 5 and 14 with zero missing fields; session independence is unknown. Complete IDs, raw numerators and summaries are in [explain.json](../.cache/flake-10054-r2/phase3/explain.json).

Example native evidence: failure run `3f477f12d4f609ee6acf9e022c8aeba3`, `event:2ea72ce3d75df2eb16a9dbcba02c30b5e50c805104f3d2b60ccd277c9c5d989e/0`; passing run `27e37b87b909632d1af7096a237d1775`, `event:7b82c26b37939a8c7e7f740983058c4a960136485b335a856ada3240845ef18e/0`.

- **Observation:** the recorded table and scoped event IDs above. The tool exposed complete scalar records, unlike round 1.
- **Association:** none supported; insufficient independent-session evidence. No observation is promoted.
- **Hypothesis:** the Phase-2 experiment supports the narrow body-dependent prediction. The recorded comparison neither confirms nor contradicts that mechanism: it contains only original-body runs, and body size is constant. Readiness in all 19 weakens a simple startup-failure account but does not inspect port ownership.

The tool added structured, citable timing measurements and completeness accounting. It did not supply diagnostic information beyond reading and ordinary test/server logs that identified the mechanism. The useful experimental result came from the ordinary scratch perturbation, not a Signal Scout explanation. The exact mechanism remains unresolved.

## Setup failures retained

1. Source lookups initially used nonexistent tests/spike_support.py, internal/cohort/profile.go, internal/privacy, and internal/provenance/profile.go / a PowerShell-incompatible rg glob. Correct locations were read; no test runs spent.
2. After recording 19, `runs list --json` failed (unknown flag; runs list emits JSON already). The scratch orchestrator stopped its collector; all recording drains had completed. Collector was restarted for analysis only. No test rerun.
3. Analysis looked for doctor.recorder instead of doctor.counters, causing KeyError after one successful CLI read. Corrected scratch script and restarted collector; retained CLI ledger.
4. First analysis profile used floor 0, rejected by validation (positive floor required). Corrected to floor 1 before any valid comparison; no scoring result was seen/tuned before that correction.

5. The first staged diff check found doubled CR line endings in Python-captured logs and trailing whitespace in build/hash output. A follow-up normalizes display copies only; exact original bytes and hashes are retained in raw-artifacts.zip and evidence-normalization.json. No test or recording was repeated.

No temporary-permission failures, probe attachment failures, drops or test-body setup failures occurred in round 2. No tracked-code changes were needed. Collector starts/stops and queries can affect timing, so the recording arm is not an unbiased failure-rate estimate.

## Command accounting and retained artifacts

**79 counted runs of budget 80**: 30 unmodified isolated test bodies + 30 scratch-copy test bodies + 19 one-test recording invocations. The test body inside a recording is counted once, not twice. No full-suite repetitions, pilots or extra unrecorded target executions were performed. The 5-failure/5-pass recording target was met; one counted run remains unused.

Separately, outside that budget: one Go build, one go-version inspection, four collector server starts, and **46 analysis CLI invocations** (45 retained in analysis-cli-ledger.json plus the initial invalid runs-list invocation). These include repeated read-only lists/completeness queries after setup failures, two valid cohort writes, one rejected cohort write, and one explain call. Test-owned server starts are test setup within their respective body runs. Shell/source inspection, scratch-file generation, hashing, one SQLite backup, Git and PR operations are setup/administration; they were not individually counted as test/recording runs. HTTP readiness polling is setup, not a target test body.

Per-phase ledgers and logs, hypothesis/plan hashes, doctor reports, cohort outputs and scratch source are retained under the cache directory. A quiescent SQLite backup `phase3/recordings-snapshot.db` preserves WAL transactions for review; its hash is recorded beside it. The locally built executable and transient original DB/WAL/SHM/bytecode are not committed; the binary's hash/build metadata and portable SQLite snapshot are retained. No scanner manifest regeneration or unrelated suite run was performed because no tracked implementation changed.

`git diff --check` was run before committing. Only this doc and selected new files under `.cache/flake-10054-r2` are committed. This remains one real, unblinded case and is not evidence that Signal Scout generally works.
