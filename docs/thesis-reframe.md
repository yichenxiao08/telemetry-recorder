# Proposed thesis reframe

**PROPOSED — awaiting owner approval.** This is a proposal, not an approved
change. Nothing here is claimed as proven. No code or scoring change is proposed
as part of this document.

## Current thesis, preserved verbatim

The current thesis in `plan.md` is:

> Signal Scout helps a developer explain any behavior or problem they want to
> understand by recording how outputs were produced, tracing inputs and decisions,
> comparing labeled problem examples with reference examples, and checking declared
> expectations, with every claim linked to evidence.

## Results to date

The following records the results available on 2026-10-23 for runs 2 through 6.

- **Run 2, mutable default argument:** capture worked for 80/80 runs with zero
  drops. It was a provisional Hit. A code-reading baseline found the cause in 6
  commands. The bug was easy; attribution to the tool was generous, and the tool
  did not beat the baseline.
- **Run 3, inventory replenishment:** invalid. The documented command did not
  reproduce the symptom; all 60 CLI runs gave identical totals. It does not
  count for or against the tool.
- **Run 4, retrybench, seed 41, 80 runs:** the documented command reproduced the
  symptom in 12/80 runs (15%). The baseline reported Found in 6 of 150 commands
  and named a cache key that omits a value the calculation depends on. The
  existing evaluation record says grader confirmation is pending. The gate
  failed because the bug was too easy; the tester run was not spent.
- **Run 5, kitcheck, seed 42, 100 runs:** a fresh planter's bug reproduced
  identically on repeat in 19/100 runs (19%), within the required 10–25% window.
  The project had 9 files. Answer hash:
  `eb74d8ae9d327fc3b21a550dc40ac5601d5df203395a8842177c7dca71667ec1`. The
  baseline named the cause in 5/150 commands: a position in the SKU-sorted plan
  was reused as an index into the original cart, across fulfillment and
  inventory. The official grader said Found; the baseline named the answer's
  function or input: yes. The gate failed because the bug was too easy; the
  tester run was not spent.
- **Run 6, BugsInPy, three deterministic offline reproductions:** all three
  reproduced 20/20 on Python 3.12.12.
  - `pysnooper` bug 1, answer hash
    `8bf8d2c3d41c06ebca7d86782f40c2d1aeaf3e4844b1aa8c586bd88edea88064`: the
    baseline named a specific cause in 5 commands. It was not graded. The
    preparer added in-process compatibility shims, so this is a baseline
    self-report only.
  - `tqdm` bug 2, answer hash
    `a33c835a571e6d638786d66a7d6323e8e6f74a6c70dec89d9f98b0a252d65cd8`: the
    baseline named a specific cause in 13 commands. The official result was
    Found; it named the answer's function or input: yes.
  - `tqdm` bug 3, answer hash
    `0352142f154f16b55f86796c77baa9b7d9a999e2f89a42a57b1aaeeaf716af58`: the
    baseline named a specific cause in 11 commands. The official result was
    Found; it named the answer's function or input: yes.
  - The gate failed on both graded bugs. They were real, unplanted bugs, so
    planter design is not the whole explanation. A likely common factor is
    that a small deterministic reproducer with a precise numeric symptom is
    easy to localize by reading. Because these are public bugs, a model might
    recall them rather than find them by reading; the two cannot be separated
    in this evaluation. No scanner evaluation or tester run was spent, and the
    scanner source remains unfrozen for evaluation.

## Proposed narrower claim

Signal Scout is an evidence layer for explaining behavior. It does not locate
causes. It records runs and shows which runs differ, how often, and under what
recorded inputs, and checks whether a stated hypothesis is consistent with
recorded evidence, with every claim linked to evidence and tiers kept separate.

## Proposed pre-registered test

Fix this test design before collecting any results. Choose a bug where the
obvious reading of the source is plausible but wrong, so a confident reader's
explanation is incorrect or incomplete. Compare three arms on the same
reproduction and recorded runs:

1. A code-reading baseline.
2. A log-grep baseline.
3. Signal Scout.

Arm (c) succeeds only when it detects that an explanation is inconsistent with
recorded runs, or quantifies the behavior correctly, in a case where arm (a) is
confidently wrong and arm (b) fails to detect that inconsistency or quantify
the behavior correctly. Fix the specific bug, evidence, run population, prompts
or instructions, confidence rubric, capture checks and scoring rules before
results are collected.

## What would count against the proposed claim

- The reader's explanation is correct.
- The log-grep baseline matches Signal Scout.
- Signal Scout's output cannot be used without already knowing the answer.
- Capture misses the runs relevant to the explanation.

## Known weaknesses

- The narrower claim is easier to meet and may not be interesting.
- The claim competes with ordinary logging and notebooks.
- The scan's purpose shifts from finding the bug's function to suggesting what
  is worth recording; that purpose needs its own evaluation.
- Constructing misleading bugs has the same planter-bias risk as before.
