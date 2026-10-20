# Lane-02 normative generator source, version 1

This document is the **generator source specification**, not executable Python,
Go, a generated dataset, or an engine implementation. It is intentionally
language-neutral pseudocode for Task A's docs-only review. Task B must implement
these exact equations and publish the executable source SHA-256 before generating
evaluation data. A changed equation requires a new reviewed contract, not a
silent implementation choice. No reserved seed has been evaluated.

## Deterministic primitives

All text is ASCII. Integer rendering is base-10 without leading zeroes. Strings
below have no spaces. Variant names are exactly the six names in contract.md.
All indices are zero-based. SHA256 means the standard 32 digest bytes.

For seed z, variant v, session s, pair i, field f and arm a, define:

    message = "lane-02/v1|" + z + "|" + v + "|" + s + "|" + i + "|" + f + "|" + a
    digest = SHA256(ASCII(message))
    bits = unsigned_big_endian_integer(digest[0:8]) >> 11
    U(z,v,s,i,f,a) = bits / 2^53

This is a specified deterministic pseudorandom construction, not independent
sampling proven by the hash. No mutable PRNG state or seed search. U is in [0,1).
Use IEEE-754 binary64 for all calculations; thresholds below are fixed decimal
literals. There is no post-hoc rounding, clipping or rejection sampling.

## Population and inputs

For each seed/variant, build 20 sessions s=0..19. Each session has 20 pairs
i=0..19. Each pair contains one construction-arm P and one construction-arm R.
Thus each variant has 400 eventual problem and 400 reference examples. Arm is
generator-only metadata: **never pass it, a label, or expected lane to target**.

    minority = variant != "matched-negative-controls"
    previous_branch = (not minority) or (i < 4)
    if previous_branch:
        gap = 0.50 + 0.04 * U(z,v,s,i,"gap","shared")
    else:
        gap = 0.65 + 0.30 * U(z,v,s,i,"gap","shared")
    current_value = -1 if (arm == P and not previous_branch) else +1
    age = 2 if (arm == P and previous_branch) else 1
    if variant == "within-range-subgroup" and arm == R and 4 <= i < 10:
        age = 2

For j=0..6, start with:

    draw_arm = arm if variant == "unrelated-nuisance" else "shared"
    noise[j] = -1 + 2 * U(z,v,s,i,"noise_"+j,draw_arm)
    if variant == "benign-minority-tail" and j == 0 and i < 4:
        noise[j] += 3.0                 # BOTH arms, identical paired values
    if variant == "moderate-decoy" and j == 0 and arm == P:
        noise[j] += 0.75                # ONE unused nuisance, moderate shift

All other fields stay as above; in particular no +10 offset, no nuisance shift
in the unrelated variant, and no conditional-comparison variant. P/R share the
same gap multiset and, except the declared nuisance variants, the same nuisance
multisets. Pairing is intentional matching, not evidence of independent frames.

## Ordering, frame history and identity

Within each session, sort its 40 arm/pair items by ascending digest bytes of:

    SHA256(ASCII("lane-02/v1|"+z+"|"+v+"|"+s+"|"+i+"|order|"+arm))

Tie-break by arm P before R, then pair index. Sorting affects execution order
only; it does not change generated values. Both arms exist in every session.
For sorted slot k=0..39:

    frame n = 100000 + 100*s + k
    previous_frame = n - age
    history = { decimal_string(n-2): -1, decimal_string(n-1): +1 }

Each decision is a separate, explicitly synthetic **two-snapshot episode**.
history[n-2]=-1 then history[n-1]=+1 are the actual supplied older/fresh values,
not merely version metadata. Episodes do not share a physical global history;
session IDs identify construction clusters, not verified independent drives.
Capture the entire two-entry history so the lookup and counterfactual are
inspectable. A future continuous sequential-history benchmark is out of scope.

Case identity is lane-02/<seed>/<variant>; session appends /session/<s>; run
identity is the first 32 lowercase hex characters of SHA256 of ASCII case
identity + /session/<s>/pair/<i>/arm/<P-or-R>. Do not include expected labels in
target inputs or in the profile used to score. IDs support reproducibility,
not a mechanism boost. Each run contains exactly one completed decision.

Execution is at most 60 calls/sec with recording enabled. Set each case's
synthetic UTC root timestamp origin from its fixed index (seed order, then
variant order as listed in contract.md), not from cohort label. Frame slots
in session/slot order are 16,666,667 ns apart; origin is
1767225600000000000 (2026-01-01 UTC) + case_index * 60,000,000,000 ns. Use one explicit 60 s
selection window per case and eligible-population limit 800; reject overflow.
Late annotations must not affect these execution timestamps.
Synthetic timestamps are supplied by a harness clock to the probe/exporter,
never by the target. If this requires changing the target or any equation,
stop and ask the owner before proceeding.

## Plain target source specification

The future plain Python target has NO Signal Scout imports. Signature inputs
are gap, current_value, history, previous_frame, frame, noise. Its equations are:

    previous_used = gap < 0.55
    if previous_used:
        chosen_value = history[decimal_string(previous_frame)]
    else:
        chosen_value = current_value
    lane = "right" if chosen_value >= 0 else "left"
    return { lane, chosen_value, previous_used,
             previous_version: previous_frame, frame }

Noise is accepted for negative-control capture but is never read by the decision.
Missing history keys are target errors, not silently substituted fresh values;
the generator supplies both possible keys. No branch may inspect seed, variant,
construction arm, problem label, reviewer label, or feature-scoring metadata.

## Labels, exact mechanisms and development-only counterfactual check

The synthetic oracle's expected lane is right. After target completion, label
problem iff returned lane != right; otherwise reference. Verify that derived
labels agree with the intended 400/400 construction, but **derive**, never copy,
the labels. Synthetic-oracle-v2 and synthetic-reviewer-v2 are explicit simulated
sources, not human evidence or a population-quality judgment.

For each declared-use problem frame, age=2, so the actual lookup returns -1:

    gap < .55 AND history[n-2] = -1  => lane = left (wrong)
    replace ONLY previous_frame by n-1, holding history and all else fixed
    gap < .55 AND history[n-1] = +1  => lane = right (fresh counterfactual)

This fixes lane-01's metadata-only version flaw. The version changes which value
the target really consumes. The freshness invariant checks n-previous_frame=1
only where previous_used is true. A stale wrong output has a constructive
counterfactual witness; this is a controlled synthetic fact, not a general
causality inference by Signal Scout.

In minority variants, the other 16 P examples/session have gap>=.65 and
current_value=-1, which independently causes a wrong lane in the current branch.
Changing ONLY current_value to +1 makes their lane right. These are a disclosed
second mechanism, not unlabeled fillers or synthetic failures relabeled as passes.
Matched-negative-controls has only stale-use problems. References always return
right. In within-range-subgroup, six R examples/session carry age=2 but do NOT
use it (previous_used=false); they remain correct and invariant not_applicable.

Task B development tests use ONLY seeds 101 and 103 (already used development
seeds, not new evaluation seeds): for every affected declared-use frame, run
the plain target with the supplied stale version and fresh counterfactual;
assert left vs right, equal other inputs, previous_used=true, and actual lookup
values -1 vs +1. Also assert the current-branch counterfactual and noise
metamorphic invariance. These tests are specified here but NOT implemented or
executed in Task A. Never generate reserved seeds in tests/imports/CI preflight.
