# Paired practical usefulness pilot

**DRAFT — design only, 2026-10-25. No study runs authorized by this document.**

This separate track asks whether access to Signal Scout improves correct causal
investigation, including its setup cost. It does not replace the original
150-command hard-bug gate, complete M5, authorize production data, or establish
usefulness. Owner-reported run4b/run4c/historical-01 baseline Hits remain failed
hard-gate cases; Signal Scout was not tested on them. Retain all prior results.
The plan's exclusion of a human diagnosis-time study remains: this is a small
agent-session pilot, not a human study or a product benchmark.

## Near-term product scope

Adopt the owner's independent critique dated 2026-10-25: a local Python
investigation aid for reproducible behavior, with no demonstrated diagnostic
advantage and no causal-proof claim. Recording selected boundaries, comparing
observations and checking declared expectations are implemented; usefulness is
unproven. Defer Node implementation, LLM integration, value linking,
first-divergence and other product expansion pending this pilot. Node design
PR #31 remains an unmerged reference. Passing this pilot supports only further
held-out evaluation, not automatic expansion. See the attributed
[critique response](independent-critique-response.md) for rationale and limits.

## Freeze before investigation

Merge/review this draft, then record a registration commit and SHA-256 manifest
before any investigator starts. Preparation may begin after the protocol is
reviewed/merged, without an already-frozen case registration. A registration with
status FROZEN must reference the reviewed merged protocol commit and its hash;
that explicit freeze supersedes this historical DRAFT banner for that pilot only.
If model/reasoning or other launch settings are unavailable, prepare cases and
save status PENDING_SETTINGS with null settings; never invent them. Preparation
is not an investigator launch. Fill pending fields and freeze before launching
any investigator. A separate fresh curator
selects exactly three new independent real historical cases, preferably different
projects, without inspecting scanner signals or selecting for baseline failure.
Take the first three eligible cases from a dated ordered shortlist of at most
six; retain all exclusions and reasons. Do not screen by baseline success,
scalar-friendliness, candidate rank, or expected tool advantage. Unsupported
capture mechanisms remain possible tool losses. If fewer than three qualify,
report an incomplete pilot; do not search indefinitely.

Eligibility: actual upstream defect and correction; readable licensed source;
local Windows reproduction under a supported Python version; no services,
credentials or personal data; synthetic inputs; practical dependency provisioning;
documented intended rule; two identical deterministic reproduction outcomes and
upstream correction removes the symptom. No required 10–25% frequency in THIS
track: report actual counts. This explicitly differs from, and changes nothing
in, the original hard-gate contract. Do not distort workloads to hit a percentage.
Cases already exposed to any investigator, including run4b/run4c/historical-01,
are contaminated: optional setup/exploratory evidence only, never primary cases.

A case custodian (fresh chat, not coordinator) prepares byte-identical clean
copies, common README/tests/symptom/seeds/run commands and offline dependency
wheels/environments. No Git history, fix tests, upstream issue/fix links or causal
hints in packets; preserve required attribution. Public project identity creates
recognition risk; investigators disclose familiarity and never search upstream.
Seal answers, upstream provenance, hashes and fixed-copy validation separately.
Curator may research upstream; investigators and graders have no internet.
Do not replace a case because either arm loses. Infrastructure-invalid pairs
remain listed, unscored with reasons, without silent replacement.

Registration must contain:

- Protocol commit/hash, ordered cases and eligibility/exclusions, target tree and
  individual file hashes, README/tests/harness hashes, seeds, exact commands,
  interpreter and offline dependency hashes, expected counts and answer hashes.
- Engine source commit, clean binary SHA-256/build metadata, adapter/lockfile
  hashes and complete scanner source manifest. The existing scanner manifest may
  predate later changes: regenerate from the frozen engine commit's Git blobs,
  include every scanner source file, and record exclusions/rules/bounds. Freeze
  generic support docs and allowed commands; no post-result tuning or rebuilds.
- Same owner-selected model, reasoning effort and tool permissions across both
  arms and all cases; fresh sessions with no inherited history/subagents.
  Record exact model/settings. Unknown settings do not block preparation; missing
  fields block investigator launches and registration freeze.
- Assigned anonymous session codes and fixed launch order: case 01 ordinary then
  tool; case 02 tool then ordinary; case 03 ordinary then tool. No report or
  diagnosis transfer; don't reveal first-arm outcomes before second arm ends.
  Three cases cannot perfectly balance order; publish that limitation.

## Layout and custody

Reserve (do not create during design) `C:\Users\yiche\signalscout-study-pilot01`.
If it exists nonempty, registration stops rather than reuses it. Each case uses
`case-01`, `case-02`, `case-03`; within each: `ordinary\project`,
`ordinary\output`, `tool\project`, `tool\output`. Engine is a frozen copy at
`C:\Users\yiche\signalscout-study-pilot01\engine`, with prepared adapter venv.
Generic docs live at `support`; registration is `registration.json` at root.
No investigator may enumerate sibling arm/case/output directories.

New sealed root for the custodian/graders only:
`C:\Users\yiche\Desktop\sealed-private\signalscout-study-pilot01`.
Use case-specific `answer.md`, `reproduction.txt`, custody mapping, and grading
notes. Coordinator only receives sanitized reports, never sealed evidence or
project/baseline source. The owner launches curator, ordinary investigator, tool
investigator, packetizer and graders in separate fresh chats. Builder subagents
may implement generic improvements later; they perform no study roles.

## Equal investigation budgets and access

Each investigator gets 150 commands and 60 minutes of investigation, whichever
comes first; reserve ten more minutes ONLY to save reports/cleanup. The same
150-command total includes reporting, so reserve ten commands. Start the clock
before first setup/packet read, record ISO UTC timestamps and command log.
One command means each top-level executable or PowerShell cmdlet invocation,
including each pipeline stage and each invocation in a loop; count failed calls,
file writes, tool calls that execute scripts, setup and reporting. Scripts must
be logged verbatim; count their external/cmdlet invocations too. Pure assignment
and shell control constructs are not commands. No opaque batch scripts to evade
budgets; preexisting target program internals are not shell commands. Record
both shell-call count and counted commands so accounting can be audited.

Approval delays, model/tool/network waits and setup remain in gross elapsed;
also log separately measured approval/platform outages for a sensitivity table,
never silently subtract them. No network except loopback intake; dependencies
are provisioned before launch equally. Engine-specific setup is charged to the
tool arm. Case-copy/dependency preparation and generic builder effort are
separate orchestration costs, published, not falsely free product setup.
The prospective investment bar below is fixed before results; no retrospective
budget extension or threshold tuning.

Both arms may read source/tests, use existing tests, temporary diagnostic scripts,
logging, debugger and local experiments on their own disposable project copy.
Log every edit and restore pristine target before final reproduction or capture;
do not repair the delivered target. No hidden answers, sibling outputs, upstream
history, assistants or network. Both see identical case facts and generic engine
support docs; ordinary may not execute Signal Scout or read its outputs.

For THIS paired track only, BOTH arms have equal implementation source access
from their first read and equal permission to debug, log and experiment. No
capture is required before ordinary reproduction or source diagnosis. This
explicitly supersedes the old asymmetric paired draft; the ORIGINAL hard-test
README/tests-before-first-capture rule is unchanged. Restore pristine source
before every capture and final reproduction, logging experiments separately.

After TWO failed setup/capture attempts, stop further tool attempts and retain
the capture failure. The tool investigator may use remaining investigation
budget for ordinary diagnosis, explicitly source-only; that diagnosis can count
for primary correctness but cannot receive a tool Hit. Keep source reasoning,
static scanner suggestions and runtime-output contributions separately attributed.
Complete zero-drop delivery remains mandatory before RUNTIME EVIDENCE ANALYSIS,
not before source reading or ordinary debugging. Failed/incomplete recordings
may be inspected only for setup diagnostics, never as explanatory evidence.

Each attempt uses a NEW `tool\output\attempt-01` or `attempt-02` directory and
new DB/config/doctor paths. No reuse or overwrite. Run scout_probe from the
adapter directory, required --doctor, --manifest and --drain-timeout 30; allowlist
scalars explicitly. Optional queue settings must be paired and logged. Do not
analyze runtime evidence until manifest coverage is complete, all drop counters zero,
delivery acknowledged and required probes invoked. Completeness is only scoped
to declared captured boundaries, not whole-program causality. Unsampled values
remain unknown. Keep scanner suggestions separate from runtime evidence.

## Measures and grading

Primary: per-case paired correct causal diagnosis (correct/partial/incorrect/
unsolved/invalid), based on sealed mechanism and causal chain, not investigator
confidence. Publish every pair and both-correct/both-unsolved/discordant counts.
No success percentage without numerator, denominator and exclusions.
An unsupported probe, capture/setup failure, drops or source-only fallback is a
retained tool outcome, not by itself an infrastructure-invalid pair. If both arms
had valid frozen packets and submit complete reports within budget, grade their
actual diagnoses even when the tool could not record. Invalid means compromised
integrity/isolation, an unusable common reproduction/environment, or missing
necessary records preventing sound grading; document the reason without erasing
the attempted pair.

Separately grade tool attribution: Hit only when preserved unedited tool output
names the sealed causal function/input with supporting context. Downstream
symptom differences are Partial. Source-derived correct diagnosis may score
correct on the primary outcome but is NOT a tool Hit. AST suspicion alone is
not runtime explanation; label static-only pointers separately. A runtime-supported
Hit additionally requires an unedited runtime artifact naming the causal function
or input with relevant context, checked against the sealed mechanism. Citations and
numbers being valid does not prove the causal explanation true.

Secondary: setup-inclusive gross elapsed, setup minutes/commands, analysis time,
commands to final submitted diagnosis, coverage/omissions/drops, configuration
and manual edits, setup failures, evidence specificity and unsupported claims.
Save first candidate diagnosis timestamp but grade the final submitted report.
Record the investigation stop timestamp when committing the final diagnosis;
no new diagnosis/evidence may be added during report-only allowance. Define
T_arm = (investigation_stop_UTC - investigation_start_UTC), in elapsed seconds,
including setup, reproduction, capture, analysis, approval and platform waits.
Report-writing/cleanup allowance is reported separately and excluded from solve
time; investigator stop must occur by 60 minutes. At deadline without diagnosis,
record unsolved/right-censored at observed budget (normally >=60 minutes), never
a fictitious solve time. Command-cap stops can censor earlier; record actual
elapsed and stopping reason. For a valid uncensored pair where BOTH diagnoses
are correct and T_ordinary > 0, saving_fraction =
(T_ordinary - T_tool) / T_ordinary. The prospective >=20% bar means
saving_fraction >= 0.20, using unrounded seconds and ordinary time as denominator.
Censored/unsolved/invalid pairs or zero denominator cannot establish speedup.
Publish raw times, reporting overhead and all waits; do not subtract setup or
outages in the investment calculation. No statistical/population inference.

Packetizer prepares random X/Y diagnoses, identical headings, strips arm/tool
names, paths and timings without changing causal claims, then hashes packets.
A fresh correctness grader sees answer and X/Y packets only. Names of mechanisms
or writing style may reveal arm: record suspected unblinding. Freeze grades
before a DIFFERENT attribution grader sees full reports/tool artifacts and checks
original claims and output provenance. Packetizer never changes factual content;
if anonymization would alter meaning, preserve it and disclose unblinding.
Coordinator receives grades/yes-no, timings and sanitized summaries only.

## Prospective investment decision, fixed before results

The independent critique's practical investment policy is adopted prospectively,
not a result or statistical threshold. PASS requires ALL of:

1. All THREE registered pairs are completed and valid (budget-ended unsolved
   reports count as completed attempts; missing reports/infrastructure-invalid
   pairs do not).
2. Tool-arm causal diagnosis is correct on at least TWO of THREE cases.
3. No paired correctness regression: if ordinary is correct and tool is not
   correct on any case, the bar fails.
4. At least ONE runtime-supported cause-naming tool Hit on the SAME case as
   either (a) tool correct/ordinary not correct, or (b) both correct with
   uncensored setup-inclusive elapsed saving >=20% by the formula above.
   A static-only pointer, source-only correct diagnosis, echoed investigator
   hypothesis, or downstream runtime difference cannot satisfy this condition.

Missing/invalid pairs make the pilot INCONCLUSIVE, never a pass. Preserve every
pair and failure, with no favorable dropping, replacements, tuned extensions,
threshold changes or retrospective case additions. If all pairs are valid but
any investment condition fails, FAIL: pause product expansion under this thesis.
Do not tune or extend this pilot in order to obtain a pass. Setup repairs, if later
justified, belong to a different registered version and do not erase this result.
PASS supports only another independent held-out evaluation; it establishes no
broad usefulness, hard-bug advantage, human usability, demand or production
readiness. Publish all outcomes, drops, unsupported values and orchestration
costs, including frozen environment preparation. No product expansion or new
implementation is authorized by this document.

Reusable role prompts: [fresh-session prompts](paired-usefulness-prompts.md).
Setup references: [doctor](doctor.md), [probe adapter](../adapters/provenance-python/README.md),
[scanner contract](probe-scan-schema-v1.md). Existing hard-gate rules remain in
[coordinator handoff](coordinator-handoff.md). Privacy remains synthetic-input,
local-only; upstream licenses do not authorize collection of real user data.
