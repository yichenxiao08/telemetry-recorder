# WinError 10054 privacy-test investigation

Date: 2026-10-23 (America/Toronto)

Branch: `codex/flake-10054-investigation`, created from `origin/main` at `e183e726ad29f23ff36c89912493c443b1fe0fd8`.

This was one real, unblinded case. It is not evidence that Signal Scout generally works. No cause was established and no fix was applied.

## Test and retained failure

The test is `test_privacy.Privacy.test_real_probe_intake_and_log_raw_copies` in `adapters/provenance-python/tests/test_privacy.py`. The retained failure is `.cache/doctor-checks/python-privacy-recheck.txt`:

> ERROR: test_real_probe_intake_and_log_raw_copies (test_privacy.Privacy.test_real_probe_intake_and_log_raw_copies)
>
>     urllib.request.urlopen(blocked, timeout=5)
>
> ConnectionResetError: [WinError 10054] An existing connection was forcibly closed by the remote host
>
> Ran 6 tests in 0.369s
>
> FAILED (errors=1)

The relevant call is the final POST to `/api/annotations`. The test expects `urllib.error.HTTPError` with status 403. Its `Serving` helper launches the Go Signal Scout server on loopback. It chooses an ephemeral port by binding `127.0.0.1:0`, closes that temporary socket, then starts the child server on the selected port and polls `/runs`. The Go handler denies non-GET `/api/` requests with 403 and the P0 metadata-write message.

The repository-documented command was `uv run --directory adapters/provenance-python --frozen python -m unittest discover -s tests -p test_privacy.py -k test_real_probe_intake_and_log_raw_copies -v`. `uv` was unavailable and its cached-tool path could not be read, so the actual isolated runs used the existing Python 3.12.12 virtual environment from `adapters/provenance-python`:

```powershell
.venv\Scripts\python.exe -m unittest discover -s tests -p test_privacy.py -k test_real_probe_intake_and_log_raw_copies -v
```

The full-suite command was `.venv\Scripts\python.exe -m unittest discover -s tests -v` from that same adapter directory.

## Phase 1 results

| Arm | Pass | Fail | Failure rate |
| --- | ---: | ---: | ---: |
| Isolated target test, 30 full runs | 25 | 5 | 16.7% (5/30) |
| Full Python suite, 10 runs | 8 | 2 | 20.0% (2/10 suites; both failures were this test) |

The isolated failures were runs 5, 9, 15, 17 and 29. The full-suite failures were runs 3 and 9. Every failure was the same `ConnectionResetError: [WinError 10054]` while reading the response to the `/api/annotations` POST. The per-run timestamps, durations, statuses and log names are in [the isolated run ledger](../.cache/flake-10054/isolated-reviewed/runs.csv) and [the full-suite run ledger](../.cache/flake-10054/full-suite/runs.csv).

Before the reviewed execution path was available, two batches of 30 starts stopped in Python temporary-directory setup with `PermissionError`/WinError 5 while writing synthetic `policy.json`; those were not full test runs and did not reach the server. Setting `TEMP` under the workspace did not clear the denial. Execution then used the reviewed path with temporary files under `.cache/flake-10054/tmp-reviewed`; no machine settings were changed.

Cheap machine-state checks: `Get-MpComputerStatus` returned access denied, so Defender real-time scanning status was unknown. A process snapshot listed `sldBgDwld`, `msedge` and ChatGPT among the larger CPU/working-set processes; ChatGPT was about 1.3 GB working set and Edge about 628 MB. This was a cumulative snapshot, not a measure of instantaneous contention.

## Hypothesis written before probing

The plain-reading hypothesis, recorded before any Signal Scout probing in `.cache/flake-10054/preprobe-hypothesis.md`, was an ephemeral-port handoff race. The helper releases the temporary loopback bind before the Go child binds the chosen port. If another local process claims or disturbs the port in that gap, the POST may see a reset instead of the intended server's 403; when the intended listener handles it, the test should receive the expected HTTP 403. This predicts the reset at the `/api/annotations` response and predicts that a failing run may lack the intended listener on the chosen port. It is a hypothesis only; the recording did not observe port ownership or establish this mechanism.

## Phase 2 recording and comparison

### Probe selection and recorder setup

The Phase 1 source reading led to this initial probe order: (1) `Serving.__enter__` for temporary-port selection and server startup; (2) `test_real_probe_intake_and_log_raw_copies` for the client sequence; (3) the client HTTP boundaries `urllib.request.urlopen` and `http.client.HTTPConnection.getresponse` for the POST and status-line read. Direct preflight attachment to the test/dunder functions was unsupported. Attaching the generic HTTP functions also observed Signal Scout's own exporter traffic recursively, so those runs were discarded.

Separately, `signalscout probe scan` on the test sources reported these candidates: `spike_support.py:34 time.monotonic`, `spike_support.py:35 self.__exit__`, `spike_support.py:37 time.sleep`, `test_privacy.py:119 self.assertNotIn`, and `spike_support.py:15 Serving.__enter__`. They were unresolved dynamic methods or unsupported dunder calls. The single-file scan root was rejected because scan requires an ordinary directory; a focused copy of the two source files was scanned under `.cache/flake-10054/scan-source`.

The final preflight-attached, two-probe setup used cache-only `flake10054_harness.main` and `flake10054_harness.test_urlopen`. The latter is a pass-through at the test client's `urllib.request.urlopen` boundary. Neither probe captured arguments or return values. The source-side configuration used the strict synthetic P0 policy/profile and synthetic data only. No Python hook into the Go server was available, so there was no server-side function probe.

From `adapters/provenance-python`, the existing template printed by `signalscout doctor` was run with `--doctor`, `--manifest` and `--drain-timeout 30`, using the synthetic configuration and explicit P0 policy. The client runner reported 2/2 probes attached and a completed drain on each final recorded attempt.

The test's existing `bin/signalscout.exe` did not match the investigation branch: `go version -m` reported revision `39247ac3…` and `+dirty`, while the branch HEAD was `e183e726…`. The 30 isolated and 10 suite runs therefore exercised the existing Python test with that stale server binary. The current CLI used for Phase 2 was built separately to `.cache/flake-10054/signalscout.exe`; its module metadata matched the branch HEAD. The build emitted an access-denied warning for the global Go stat cache but completed.

### Captured runs and completeness

The first eight runs in [the recorded-run ledger](../.cache/flake-10054/recorded-final/runs.csv) included six test passes and two WinError 10054 failures (runs 5 and 8). Each narrow run's doctor report showed 2 `test_urlopen` calls, 4 acknowledged spans, zero dropped/unknown spans and a complete client drain. However, each client manifest reported `manifests_built=0`, one pending run and a sent scope tail. With the collector also running a P0 capture filter, the stored scope tail was absent: completeness reported `scope_final=false`, `scope_expected=0`, and `scope_reason="scope tail record missing"` for the sampled pass and failure scopes.

One additional passing run used the same source-side P0 policy but ran the collector without its extra server-side P0 filter, so metadata could be stored. This restored a scope record (`scope_final=true`, `scope_expected=1`), but completeness classified the associated run as `unknown` with `final marker missing`. Thus none of these recorded runs was demonstrably complete. The collector-side filter removal was a recording-workaround deviation; the source-side strict P0 capture policy remained active and all data were synthetic.

### Cohort/explain result

For a bounded comparison, four stored runs were given explicit synthetic `observed_test_outcome` labels derived from the test log: failures `f11114481b05d94727b99222788a21c6` and `2c96f0dd65e6bc86bd5df6a366e18fc3`, and passes `b440610abab261ddde208ab640bb2e16` and `79c2810748e64cee86ebd45d128cf3e1`. Their primary span evidence IDs were respectively:

- `span:f11114481b05d94727b99222788a21c6/0000000000000001`
- `span:2c96f0dd65e6bc86bd5df6a366e18fc3/0000000000000001`
- `span:b440610abab261ddde208ab640bb2e16/0000000000000001`
- `span:79c2810748e64cee86ebd45d128cf3e1/0000000000000001`

These labels only group runs by the already observed test result; they are not independently captured symptoms. `cohort create` selected no runs: the recorded spans had no supported decision for the declared profile and the capture completeness was unknown. `explain-cohort` returned `insufficient_evidence`, zero problem runs, zero reference runs and no differences. The stored run records exposed only pass/error status and redacted/empty attributes; they did not identify a listener, port, request path, or cause. **The recorded comparison said nothing about the port-race hypothesis.**

### Observation, association and hypothesis tiers

- **Observation:** the uninstrumented test logs reproduce the reset at the expected POST. The recorded failing root spans have error status and the selected passing spans have OK status, with the evidence IDs above. Capture completeness is unknown/incomplete; captured attributes are empty/redacted.
- **Association:** none. Cohort selection had no eligible runs and explain-cohort reported insufficient evidence, so there is no observed failing-versus-passing feature difference to report.
- **Hypothesis:** the temporary-port handoff race remains possible from source reading alone. No captured evidence confirms or contradicts it. This investigation does not claim causation.

## Setup problems and limits

- `uv` was not on `PATH`; the cached uv tool directory also returned access denied. The existing adapter virtual environment was used.
- Sandbox temporary-file writes failed before test execution in 60 setup-only starts. The reviewed execution path was needed for the valid test runs.
- The initial doctor invocation omitted `--adapter .`, selected an unrelated system Python 3.14 and could not resolve the adapter's probes. The corrected doctor run used the adapter directory and existing Python 3.12.12 virtual environment; it reported 14 PASS, 2 WARN, 0 FAIL. Warnings were missing `uv` and an existing database parent where write/WAL readiness was not tested.
- The server binary used by the tests was stale and dirty relative to the branch. The current CLI was built under cache; a global Go stat-cache access denial was reported despite successful build.
- Initial scanner/preflight attempts hit the single-file-root restriction, lazy imports, unsupported dunder/custom-access/slots functions, a variadic-signature rejection, a wrong timeout-sentinel reference, and a stale function alias. Those were corrected or excluded without editing tracked adapter/test code.
- The first generic HTTP probe configuration recursively recorded exporter traffic: approximately 2,013 `urlopen` and 2,011 `getresponse` calls, 986 manifest-ledger overflows, 3 unknown records and one flush timeout. Queue-full drops were zero. Those two runs are invalid and excluded.
- A subsequent 15-run recording batch never invoked the wrapped client function (`test_urlopen calls=0`, runner exit 3); it included one 10054 and one 10053. Those runs are excluded from the comparison.
- The collector-side P0 filter suppressed scope-tail metadata. Removing only that extra collector filter restored the scope tail but left a missing final run marker. The final capture runs reported zero drops and unknowns at the client, but database completeness remained unknown.
- The first cohort attempts omitted the required analysis profile; the next attempts lacked the required selector. After adding explicit synthetic outcome labels, cohort selection still excluded the runs as `no_supported_decision`, and explain-cohort was insufficient. No complete passing/failing comparison was obtained.
- Defender status was unreadable (access denied).

No cause was identified, so no candidate fix is proposed. This investigation used one real, unblinded case and does not show that Signal Scout generally works.

## Invocation accounting

Counting each test/recording process start as a command, this investigation used 126 process invocations: 60 sandbox setup-only starts, 30 full isolated runs, 10 full-suite runs, and 26 Phase 2 recording attempts (2 recursive-probe runs, 15 uninvoked-wrapper runs, and 9 final recorded runs). This count excludes doctor, scan, cohort, build, shell setup and Git commands. Under this process-level accounting the requested 60-command budget was exceeded; the valid Phase 1 arms themselves were 40 full runs.
