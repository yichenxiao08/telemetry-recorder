# Workflow transfer assessment: Flappy Bird

2026-10-26. Bounded public integration assessment at Signal Scout `b0503f4` and
rlenv `fdd92f7`. This is not a diagnostic study, ML evaluation, or usefulness claim.
No reward, policy, environment, recorder or analysis code was changed.

## Question and prospective criterion

The proposed second question is different from Snake repetition: **does each
nonterminal Flappy physics step return reward within `[0.01, 1.11]`, while a
terminal step returns `-1`?** This bound follows from the existing alive bonus,
optional obstacle-pass bonus and clamped distance shaping. A future numeric declaration should explicitly use a small tolerance, such as
`1e-12` reward units, rather than exact decimal equality. It checks a declared
single-call contract; it does not prove which reward component caused behavior.
The criterion was saved before attachment validation: use an existing bounded
headless entry point, make zero target/launcher changes, record complete zero-loss
scope, then run a native relation and a second question against the saved DB.
If there is no suitable entry point, stop rather than add a new driver.

## Three distinct results

| Layer | Finding | What it does not establish |
| --- | --- | --- |
| `scripts/rewardaudit` | Intentionally fixed to the prepared Snake v2 contract | It is not a generic RL evaluation command |
| Existing Flappy call boundaries | `_step_single(action)` has scalar arguments/receiver fields and scalar return items available to generic probes | An attachable boundary is not an executed or complete recording |
| Existing workload entry points | No launcher-free, bounded headless Flappy evaluation was demonstrated | This is not evidence that Signal Scout cannot instrument Flappy |

The [reward-audit helper](../scripts/rewardaudit/main.go) requires one exact v2
specification hash and fixed Snake seeds/conditions. Its
[capture source](../scripts/rewardaudit/capture.py) imports `study.harness`, creates
an 11-input/3-action agent and uses the Snake experiment's evaluation. Changing
only a probe config cannot turn that workload into Flappy. This is a deliberate
example boundary, not a failed general-purpose feature.

## Existing target entry points

The exact inspected upstream revision is
[`fdd92f7e722a00a6a74da1e8ef6b9c00b1f6fb04`](https://github.com/yichenxiao08/rlenv/tree/fdd92f7e722a00a6a74da1e8ef6b9c00b1f6fb04).

- [`value/main.py`](https://github.com/yichenxiao08/rlenv/blob/fdd92f7e722a00a6a74da1e8ef6b9c00b1f6fb04/value/main.py)
  exposes `main()`, but hardcodes `VISUALIZE = True`, constructs a renderer,
  starts training and enters an unbounded playback loop. The nonvisual branch is
  not a command-line selection; this assessment did not edit it or execute it.
- [`app.py`](https://github.com/yichenxiao08/rlenv/blob/fdd92f7e722a00a6a74da1e8ef6b9c00b1f6fb04/app.py)
  starts `RLArcade().mainloop()`. Its `Runner` is a useful existing library class,
  but needs constructed environment/agent objects and a caller to drive/reset it.
  It is not an existing zero-argument bounded headless entry function.
- [`value/train.py`](https://github.com/yichenxiao08/rlenv/blob/fdd92f7e722a00a6a74da1e8ef6b9c00b1f6fb04/value/train.py)
  has `train_loop(env, dqn, buffer, epsilon, action_size, N, recorder=None)`.
  It requires live objects and performs training; it is not the requested frozen
  headless evaluation driver.
- The prepared public `study.harness` is Snake-specific, not a reusable Flappy
  driver. It was inspected as preparation code, not treated as upstream source.

The current [probe entry interface](../adapters/provenance-python/scout_probe.py)
imports `module:function`, requires a plain synchronous top-level function, sets
`sys.argv`, then calls `entry()` without constructing application objects. It
cannot invoke `Environment._step_single` as an entry point or instantiate the
Flappy environment merely from a capture declaration. An application can supply
its own ordinary bounded entry function; that is separate workload preparation.

## Generic probe and relation compatibility

The existing Flappy
[`Environment._step_single`](https://github.com/yichenxiao08/rlenv/blob/fdd92f7e722a00a6a74da1e8ef6b9c00b1f6fb04/value/modules/flappy/environment.py)
returns `(state, reward, done)`. `return_items: [1, 2]` can select its scalar reward
and done values without capturing the nested state. `action` and explicitly
allowlisted scalar `self.y`, `self.velocity` and `self.score` are supported call
boundary candidates. These fields are selected measurements, not inferred data
flow. A method being attachable does not certify that every dynamic result is
serializable or that delivery will complete at game-loop rates.

The [declared relation grammar](declared-relations.md) can express the proposed
bounds using `output.1`, `output.2`, `when`, numeric comparisons and conjunctions.
It cannot join the separate `get_state()` distance result to the reward call to
prove the exact shaping equation. No cross-record value-linking feature was
added or claimed. No capture/relations counts, witness, saved-data follow-up or
throughput result is claimed for Flappy in this assessment.

An attachment-only validation loads the exact config with the frozen loader,
installs the generic injector, imports the unchanged Flappy environment module,
and inspects attachment status without constructing/stepping an environment or
running any entry point. The check passed: one configured method attached, zero
calls and zero records. Config SHA-256:
`19c6365d6458ce69a834b57a7f28502912d6e9e16f4120376d423dc6103ea72c`.
The result and inspected source-file hashes are kept in local validation artifacts. It is not a Flappy launcher, behavioral recording or native relation
execution. No GUI or training was started.

## Minimal future contract (design only)

Separate **target workload preparation** from **generic recording**:

1. The application supplies a bounded, seeded, headless zero-argument entry
   function (with ordinary `sys.argv` if necessary), fixed inputs/checkpoints,
   expected completion, and a documented run command.
2. Capture configuration selects existing fields/return items. A declared profile
   states the question and its units; missing/unsupported/sampled data stays unknown.
3. Reusable orchestration handles configuration preflight, fresh output, owned
   intake lifecycle, scope/delivery checks, native relation selection and evidence.
   It should not embed Snake state/action sizes, checkpoint layout or a single
   fixture hash if it is advertised as general.
4. A launcher-free transfer demonstration then needs complete recorded scope and
   a second saved-data question. Until that is executed, compatibility is bounded
   to static/attachment evidence, not a successful behavioral transfer.

No generic runner, Flappy driver, framework adapter or new analysis was implemented.
A future target driver would be explicit workload code, not evidence that capture
required new per-step instrumentation. Whether that preparation cost is worthwhile
remains untested. The prior Snake workflow and all earlier results are unchanged.
