# Pilot02 partial outcome: inconclusive

2026-10-26. This is a documentation summary of sanitized reports pasted by the
owner and the coordinator's stopping decision, not an independent inspection of
study evidence. No study folders, source, sealed answers or grading notes were
opened to prepare it. No causal details are reproduced here.

## Decision and scope

Stop primary pilot02 and preserve its registration, attempts, costs and results.
Do not replace cases, rerun arms, reset budgets or investigate case 03 to recover
a favorable outcome. The [frozen protocol](paired-usefulness-pilot.md) requires
three valid completed pairs; missing or invalid pairs make the pilot
**INCONCLUSIVE**, never a pass. This is not an overall product pass/fail and does
not complete M5. The stopping decision does not resolve the missing audit record.

Product expansion remains paused: no Node implementation, LLM integration,
value linking or first-divergence work is justified by this pilot. A future study
would require a deliberate decision and a new independent registration, not an
automatic repair/replacement loop or retrospective criteria changes.

## Registration reported by the owner

- Protocol commit: `929614097be4e591fc21cfcb035534b9a06ddf18`.
- Engine commit: `0ff7ff6288d2aca189cc9a232d22427d96d44e93`.
- Frozen registration snapshot SHA-256:
  `B84FDA3937394E81D06D076543F6D50661B6EF5A10BE8370BC67DBCE46E5D04D`.
- Three cases prepared; logical case 03 mapped to `case-03b`.
- Investigator settings: owner-declared `gpt-6-luna` / `xhigh`, without
  independent platform introspection.

These identifiers are copied from supplied reports, not newly verified here.
The original hard-bug gate and all previous results remain unchanged.

## Case 01: valid pair, no demonstrated advantage

The owner-pasted blinded correctness report graded both diagnoses correct.
The subsequent attribution report classified the pair VALID, with no material
procedure violation; suspected unblinding and the correctness-grading limitation
were confidence limitations. It reported complete zero-drop capture, source-derived
diagnosis, tool attribution **Partial**, and **no runtime-supported cause-naming
tool Hit**. The investigator's earlier self-reported cause-naming claim was not
confirmed by grading.

| Reported measure | Ordinary arm | Tool arm |
| --- | ---: | ---: |
| Investigation seconds, uncensored | 220 | 747 |
| Counted commands | 45 | 138 |
| Correct causal diagnosis | correct | correct |
| Separate setup duration | not measured | not measured |

Using the registered setup-inclusive investigation measure,
`(220 - 747) / 220 = -527/220`, approximately `-2.39545`; the tool arm took
approximately `3.39545` times as long. This pair supplies negative evidence for
an advantage on this case: neither a discordant correct solve nor a runtime-backed
speed benefit was shown. It is not a general failure claim or a hard-bug success.

Timing limitations remain. Setup was not separately measured, the attribution
grader flagged a timing-comparison limitation, and tool report-writing measurements
used different endpoints (earlier 413 seconds versus 269.441 seconds to initial
content finalization, with final hash-check overhead unmeasured). Report-writing
time is excluded from the registered investigation-time calculation; no missing
duration is reconstructed or subtracted.

## Case 02: stopped attempt and incomplete procedure audit

The tool investigator reported an isolation boundary crossing while filtering the
manifest and stopped before reproduction or capture. It reported 29 seconds,
14 commands, snapshot hash MATCH, assigned-packet integrity NOT CHECKED, and zero
capture attempts. NOT CHECKED is not a demonstrated hash mismatch. This is not
a diagnostic or recording failure by Signal Scout.

A separate owner-pasted procedure-audit reply reported input hashes PASS,
other-arm diagnosis/results exposure YES, case implementation source access NO,
material isolation violation YES and comparison invalidation YES. However, that
reply had status UNRESOLVED, saved no audit notes/hash, and exceeded its command
cap at 41/30. It therefore constitutes a **reported material cross-arm exposure
with an incomplete independent audit record**, not a fully documented or newly
verified determination. The inconsistency and overrun are retained, not repaired
by inference. No hidden content is disclosed or inspected in this summary.

Case 02 is not a valid scored pair for this pilot. No case-02 ordinary-arm result
was supplied, and its ordinary investigation is not launched as a replacement or
recovery. The coordinator stops the primary pilot without asserting that every
detail of the reported exposure has been established.

## Case 03 and interpretation

Case 03 was prepared but neither investigation was launched. It remains unscored.
There is one reported valid pair, one stopped/unscorable pair with incomplete
audit evidence, and one unlaunched pair. Missing/invalid pairs make the overall
pilot inconclusive under its existing rule; they are not favorable exclusions.

Retain prior incomplete preparations, budget overruns, missing timing and accounting
corrections as preparation history. They are not completed investigations, and
their causes are not inferred here. Software regression/CI checks and a generic
workflow rehearsal establish mechanics within their scope, not usefulness.

The only valid paired result supplied so far shows correct diagnoses in both
arms, longer tool investigation time, and Partial runtime attribution. It does
not establish diagnostic advantage, production readiness, human usability or
demand. The incomplete pilot also cannot establish a population-wide lack of
value. Preserve both statements without upgrading either into a product verdict.
