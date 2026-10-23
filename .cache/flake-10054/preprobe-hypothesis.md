# Phase 1 pre-probe hypothesis (written 2026-10-23 America/Toronto)

## Plain reading of source and retained log only

Exact test: `test_privacy.Privacy.test_real_probe_intake_and_log_raw_copies` in `adapters/provenance-python/tests/test_privacy.py`. It starts a real Go `signalscout.exe` server on loopback using `Serving`, posts synthetic OTLP to `/v1/logs`, reads the stored run with `signalscout runs show`, then posts to `/api/annotations` and expects `urllib.error.HTTPError` with status 403.

`adapters/provenance-python/spike_support.py` selects a free port by binding `127.0.0.1:0`, closes that temporary socket, then starts the Go child bound to the chosen port and polls `/runs`. `cmd/signalscout/serve.go` denies non-GET `/api/` requests with 403 (`P0 metadata writes disabled; annotation text has no approved capture policy`). The retained `.cache/doctor-checks/python-privacy-recheck.txt` fails while `urllib` reads the status line for the final POST: `ConnectionResetError: [WinError 10054] An existing connection was forcibly closed by the remote host`.

## Hypothesis (not a finding)

The visible source-level candidate is a loopback port handoff race: after the temporary socket releases its ephemeral port and before the Go child binds it, another local process could claim or disturb that port. In a bad run, the final POST could then encounter a listener that does not return the Go handler's expected 403, or a reset connection. In a good run, the same POST should reach the intended server and produce the expected HTTP 403. This predicts intermittent resets specifically while reading the response to `/api/annotations`; evidence of the Go child still owning/responding on the chosen port in a failing run would contradict this candidate. No probe data has been collected yet.

## Exact isolated command

From repository root: `uv run --directory adapters/provenance-python --frozen python -m unittest discover -s tests -p test_privacy.py -k test_real_probe_intake_and_log_raw_copies -v`

This runs only the named test, using the documented frozen adapter environment.
