# Paired practical usefulness pilot

**DRAFT — design only, 2026-10-25. No study runs authorized by this document.**

This separate track asks whether access to Signal Scout improves correct causal
investigation, including its setup cost. It does not replace the original
150-command hard-bug gate, complete M5, authorize production data, or establish
usefulness. Owner-reported run4b/run4c/historical-01 baseline Hits remain failed
hard-gate cases; Signal Scout was not tested on them. Retain all prior results.
The plan's exclusion of a human diagnosis-time study remains: this is a small
agent-session pilot, not a human study or a product benchmark.

## Freeze before investigation

Merge/review this draft, then record a registration commit and SHA-256 manifest
before any investigator starts. A registration with status FROZEN must reference the reviewed merged protocol commit and its hash; that explicit freeze supersedes this historical DRAFT banner for that registered pilot only. Never execute an unregistered draft. A separate fresh curator
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
  Record exact model/settings. Missing registration fields block launches.
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
No fixed speedup threshold or retrospective budget extension.

Both arms may read source/tests, use existing tests, temporary diagnostic scripts,
logging, debugger and local experiments on their own disposable project copy.
Log every edit and restore pristine target before final reproduction or capture;
do not repair the delivered target. No hidden answers, sibling outputs, upstream
history, assistants or network. Both see identical case facts and generic engine
support docs; ordinary may not execute Signal Scout or read its outputs.

The tool arm retains the standing first-capture restriction: before its first
COMPLETE zero-drop recording, read only target README and tests (plus frozen
generic tool docs and scanner output), never implementation source. Ordinary
has no artificial source-reading restriction. This asymmetry can disadvantage
the tool and must be reported; results are about this workflow, not isolated
algorithm superiority. After complete capture, source reading is allowed and
must be attributed separately. If no complete capture, stop after two failed
setup/capture attempts and retain the failure. Do not source-diagnose around it.

Each attempt uses a NEW `tool\output\attempt-01` or `attempt-02` directory and
new DB/config/doctor paths. No reuse or overwrite. Run scout_probe from the
adapter directory, required --doctor, --manifest and --drain-timeout 30; allowlist
scalars explicitly. Optional queue settings must be paired and logged. Do not
analyze recordings until manifest coverage is complete, all drop counters zero,
delivery acknowledged and required probes invoked. Completeness is only scoped
to declared captured boundaries, not whole-program causality. Unsampled values
remain unknown. Keep scanner suggestions separate from runtime evidence.

## Measures and grading

Primary: per-case paired correct causal diagnosis (correct/partial/incorrect/
unsolved/invalid), based on sealed mechanism and causal chain, not investigator
confidence. Publish every pair and both-correct/both-unsolved/discordant counts.
No success percentage without numerator, denominator and exclusions.

Separately grade tool attribution: Hit only when preserved unedited tool output
names the sealed causal function/input with supporting context. Downstream
symptom differences are Partial. Source-derived correct diagnosis may score
correct on the primary outcome but is NOT a tool Hit. AST suspicion alone is
not runtime explanation; label static-only pointers separately. Citations and
numbers being valid does not prove the causal explanation true.

Secondary: setup-inclusive gross elapsed, setup minutes/commands, analysis time,
commands to final submitted diagnosis, coverage/omissions/drops, configuration
and manual edits, setup failures, evidence specificity and unsupported claims.
Save first candidate diagnosis timestamp but grade the final submitted report.
At deadline report unsolved/censored time (>=60 minutes), not a fictitious solve
time. Report report-writing overhead separately. Compare time only for pairs
where both are correct; other pairs retain correctness/censoring. No speed-only
claim, significance claim or population generalization from three cases.

Packetizer prepares random X/Y diagnoses, identical headings, strips arm/tool
names, paths and timings without changing causal claims, then hashes packets.
A fresh correctness grader sees answer and X/Y packets only. Names of mechanisms
or writing style may reveal arm: record suspected unblinding. Freeze grades
before a DIFFERENT attribution grader sees full reports/tool artifacts and checks
original claims and output provenance. Packetizer never changes factual content;
if anonymization would alter meaning, preserve it and disclose unblinding.
Coordinator receives grades/yes-no, timings and sanitized summaries only.

## Pilot-end decision, fixed before results

Publish all outcomes, omissions, failures and setup/orchestration costs. If no
correct tool diagnosis backed by cause-naming tool output, stop claims of
explanation advantage; inspect workflow limits before expanding. If capture/setup
blocks a case, prioritize generic setup repair with a new engine version and a
new registered study, never overwrite this pilot. If a correctly grounded tool
arm solves a case ordinary does not, or both solve but tool takes less total
effort, treat it as a reason to expand held-out evaluation, not proof. If the
tool loses correctness or costs more, retain that and avoid claiming speedup.
Mixed results warrant narrow claims and targeted design work, not a tuned pass
threshold. No LLM/first-divergence implementation is authorized here.

Reusable role prompts: [fresh-session prompts](paired-usefulness-prompts.md).
Setup references: [doctor](doctor.md), [probe adapter](../adapters/provenance-python/README.md),
[scanner contract](probe-scan-schema-v1.md). Existing hard-gate rules remain in
[coordinator handoff](coordinator-handoff.md). Privacy remains synthetic-input,
local-only; upstream licenses do not authorize collection of real user data.
