# Wrapper overhead: Stage 1 protocol and Checkpoint A

Owner request: profile only on `codex/wrapper-overhead`, based on recorder fix
`eb9225e`. Stop at Checkpoint A. No optimization, delivery-policy change,
scoring change, migration, new difference type or later milestone is authorized.
Only synthetic fixed payload and development seeds 101/103 are permitted.

## Measurement choices made before running

- Keep Python uv-pinned at 3.12.12. Record interpreter, OS, CPU and GC settings.
  This measures one Python version; it cannot establish cross-version effects.
- Use diagnostic copies compiled from the current wrapper/recorder AST. Insert
  timing scopes around original operations, not replacement implementations.
  Check fixed-clock OTLP bytes, return values, exception identity and mutable
  snapshots against the originals before measuring. Source files and policy
  remain unchanged. Profiler clocks never become application timestamps.
- Record exclusive phase wall times, inclusive context times, raw whole-call
  times and timer/empty-scope calibration. Subtract measured median observer
  costs sample by sample, retain raw/negative/clipped results, and expose the
  unattributed residual. Fractions use sums, not a sum of unrelated quantiles.
  Instrumented timings are estimates; an unchanged uninstrumented control
  measures observer perturbation. Never infer a hard scheduler guarantee.
- Separate cProfile processes from phase timing processes. Profiles retain
  function statistics, not locals or captured payloads.
- Primary fixed-payload conditions: bare function; default capture OFF;
  explicit empty probe allowlist with the recorder's fields allowed; full
  allowlist. The current API has no separate capture-enabled flag: OFF and
  empty both omit values but retain references/metadata. Do not invent a new
  runtime path to distinguish them. Full payload is the unchanged S0a four
  scalar arguments and two returned alternatives.
- Five fresh processes for each primary condition and pacing mode. Fixed
  payload uses 1,800 calls/30 s at 60 Hz or 10,000 tight calls. Include the same
  feature comparison on generated development inputs without interpreting
  capture-disabled cohorts as complete. Full lane-02 end-to-end profiling also
  uses the frozen driver on all 12 cases, five paced and five tight repetitions.
- Tight lane-driver mode substitutes only that module's sleep facade to remove
  pacing; it does not patch global time/sleep, source files, equations, seeds,
  profile or completeness checks. Record overload/incomplete cases honestly;
  do not change the queue or suppress the driver's abort.
- GC-disable and worker-disabled comparisons are experiment-only controls,
  five trials each with full capture, both pacing modes. Queue remains 256.
  Worker-disabled queues fill and drop: stratify nonfull/full observations and
  disclose that this is not an isolated causal GIL test. Natural worker encode
  and HTTP intervals, GC callbacks, caller thread CPU and wall intervals provide
  overlap evidence. They cannot uniquely separate GIL waiting from OS scheduling.
- For lane input feature controls, replay the frozen development generator's
  rows cyclically, preserving each input and checking returned values against
  the plain target. These microbenchmarks do not replace the frozen full-driver
  runs. Capture-disabled and overload samples must not become zero associations.

Every process writes to a fresh directory. Preserve every attempted, incomplete
and completed run, raw timing sample, calibration, counter and profile. Keep
initial instrumented/observer misses; do not tune measurements after results.
The 100/250 us budget remains unchanged, including every miss. GC and worker
controls are not shipped defaults. No Stage 2 begins without GO OPTIMIZE.
