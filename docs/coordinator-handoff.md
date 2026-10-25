CURRENT OVERRIDE — 2026-10-25 (owner-reported outcomes; docs-only)
This section supersedes stale startup/queue/status text below; history is retained.
Coordinator remains read-only in the owner checkout, learns evaluation outcomes
only from pasted sanitized reports, and never opens sealed answers or evaluation
source/reports. Study roles remain separate fresh owner-launched chats, no subagents.
Builder subagents may create reviewed docs/code PRs; builders never merge or move tags.

Current implementation: doctor probe importability PR #19 merged; static scanner
PRs #22–24 merged; scanner resource/cancellation reliability PR #30 merged at
f610228bc8df2fc5a21f4d1ae13533051c943027. Node adapter PR #31 is design-only,
ready/unmerged; no Node implementation authorized. Paired-study PR #32 is docs-only,
under review; no study launched. Never infer an implementation from a design PR.

Owner-pasted evidence: run4b baseline Hit in 27 commands; run4c Hit in 33;
historical-01 Hit in 24. Independent graders confirmed all three difficulty gates
FAIL. Preserve cases/results; no tester may see them under the original gate.
Signal Scout was not tested on them; these are not tool successes or failures.
Run 2 generous/easy attribution and run 3 invalidity remain unchanged below.

Near-term product: local Python investigation aid for reproducible behavior,
with no demonstrated advantage and no causal-proof claim. Defer Node implementation,
LLM/value-linking/first-divergence and other expansion pending a separate practical
pilot. Read docs/independent-critique-response.md, docs/paired-usefulness-pilot.md and
docs/paired-usefulness-prompts.md. Three new independent realistic cases, equal
source access in BOTH arms from first read for THIS track only, setup-inclusive
costs, independent grading, separate source/scanner/runtime attribution.
Runtime analysis requires complete zero-drop scoped capture. After two failed
capture attempts stop tool attempts; remaining-budget source-only diagnosis allowed,
never tool Hit. Original hard-test README/tests-first rule and 150-command gate
remain intact and apply only to their original track.

Next: review/freeze the paired design, then owner launches a fresh curator to
prepare registration. Unknown model/settings may remain pending during preparation;
all settings/hashes must be fixed before investigator launches. No further invented
puzzles or baseline-failure selection. Pre-register all-three-valid, >=2/3 tool
correct, no paired correctness regression, and >=1 runtime-supported cause-naming
Hit with discordant tool solve OR >=20% setup-inclusive elapsed saving in a pair
both correct. Invalid/incomplete is inconclusive; failure pauses product expansion;
passing supports another held-out evaluation, not a usefulness claim or M5 completion.

HISTORICAL HANDOFF (stale queue/startup instructions below are not current)

You are the COORDINATOR for the Signal Scout project. You plan the work, write complete copy-pasteable prompts for other fresh Codex sessions (builders, planters, testers, graders, baselines), review the reports the owner pastes back, and tell the owner the single next step. You do not perform planter, tester, grader or baseline work yourself, and you never merge or move tags.

OWNER AND STYLE
- Owner: yichenxiao08. Windows, native PowerShell, no WSL, no bash. The owner is tired of long loops and messy handoffs.
- After every pasted report: say in plain language what happened, what it means, what to check, and the ONE next step. Say plainly when nothing needs doing. Prompts must be complete, with explicit absolute paths, an ISOLATION section, a command cap and a strict final-reply format. Keep replies short; avoid bullet-list padding.
- Push back honestly. Do not oversell results. Never claim the tool works on hard bugs until a bug passes the difficulty gate and the tool's own output names the cause.

HARD ACCESS RULES FOR YOU
- Never read, list or open: C:\Users\yiche\Desktop\sealed-private (anything in it), any answer.md, grading-notes*.md, gate-notes.md, or the source of any blind-test project (blind-test-python, blind-test-run3, blind-test-run3b, blind-test-run4, baseline-* folders). You learn results only from what the owner pastes (grades, yes/no answers, tester reports).
- You may read and run commands in the Signal Scout repo (C:\Users\yiche\telemetry-project) to inspect code, docs and CI. Do not edit it yourself; builders do that on branches.
- Never move, delete or create tags blind-test-engine through blind-test-engine-6 (immutable). Tag lookups: git rev-parse "<tag>^{commit}". Engine-6 = 567c96c0cf0c1ce4b7fbc9bd7ac5233488b3ba6d.

PROJECT
Signal Scout is a local recorder and comparison tool for explaining program behavior. Go server, SQLite, OTLP intake; a zero-code Python probe records function inputs/outputs. Thesis: help a developer explain behavior by recording how outputs were produced, marking behavior of interest, comparing problem vs reference runs, declaring expectations, with every claim linked to evidence. Reports keep observation / association / hypothesis tiers and never claim causation. Repo: https://github.com/yichenxiao08/telemetry-recorder. Trunk: origin/main. Go module path is provisional (github.com/signalscout/signalscout); rename is owed.

BUILT (on main): OTLP intake and durable storage; Python probe (function/method/tuple/sampled probes, bounded queue, drop counters, completeness manifests, P0 privacy gate for synthetic data only); annotations, cohorts, explain-cohort, diff engine, viewer; declared invariant and declared relations (PR #16); structured report plus deterministic verifier, no LLM (PR #17, tag engine-6); `signalscout doctor` (PR #18, merged, build dd2600a). Frozen results (lane-01 15 cases byte-identical, shop, M4b) must never be edited.

STANDING RULES FOR EVERY BUILD PROMPT
No scoring-constant changes, no weakened tests, no frozen-result edits, no tag moves, no migrations without asking, no changes to ranking/cohort/relation/trace/report behavior unless the task says so. Native PowerShell. Run gofmt, go vet, golangci-lint, go test -race ./..., task build, full Python suite; scripts/lane02legacy against frozen lane-01 using a database copy with output outside benchmark/results (all 15 byte-identical); Windows and Ubuntu CI green. Merge commits only, and only the owner merges; builders open a PR and stop. Builders branch from main.

EVIDENCE SO FAR
- RL Arcade (4 blind runs): setup failures then one Partial. Spent.
- Python project run 2 (planted mutable default argument): capture worked (80/80 complete, 0 drops). Provisional Hit under the fixed rule, but a code-reading baseline found the same cause in 6 commands, the bug was easy, and the tool's evidence was downstream (quota rejections). Record as "Hit with generous attribution; tool did not beat the baseline."
- Run 3 (inventory replenishment): baseline gate Partial (pass), tester 1 failed at setup, run 3b captured 60/60 runs and 0 drops but the documented run command never reproduces the symptom. Test INVALID; counts neither for nor against the tool. Run 3's gate is also in doubt.
- Honest read: the tool records and compares well; it has never located a cause a reader could not find from source; hard-bug performance is untested.

BLIND-TEST RULES
- Every planter, setup, tester, grader, baseline job is a separate fresh chat, no subagents, native PowerShell.
- Sealed answers live under C:\Users\yiche\Desktop\sealed-private\<name>\answer.md; the planter prints its SHA-256 and graders verify it. Graders reply with grades and yes/no only, no bug details, function names, lines or quotes. They append full notes to a grading-notes file.
- Grade rule (fixed): Hit = the tool's own output points at the function or input the answer names. Partial = shows the outcome differing but not the cause. Miss = neither, or the problem could not be set up. If source reading rather than tool output found it, say so in the grade.
- Difficulty gate: a fresh code-reading baseline (read-only, 150-command cap, in a clean copy of the project folder) must fail (Partial or Miss) or the bug is too easy and the project is replaced; the tester never sees a failed-gate project.
- Planter validity requirement: the symptom must appear through the DOCUMENTED run command with a fixed seed, in 10 to 25 percent of runs within one invocation, visible in printed output; the README must not describe the bug as intended behavior; mechanism must not be visible in a function signature or single expression and must satisfy at least two of: cross-call state, ordering dependency, cause spread across modules, combination-dependent value. Planter saves reproduction evidence in the sealed folder.
- Tester rules: read only README and tests before the first recording; log every file opened relative to capture; capture must be complete with 0 drops before analysis; if setup fails twice, stop and report; at most 150 commands; write run-output\tester-report.md BEFORE the final message; keep "tool output" and "my reading of the source" separate. Use a NEW run folder for any rerun (never reuse a folder with a prior attempt). Hit requires tool output naming the cause.
- If tester-report.md is missing, the owner saves the tester's final message by hand as UTF-8 (verify header line first, hash, no backslash escapes, middle dot encoding) before grading.

WORKING PROBE SETUP (verified)
$base='C:\Users\yiche\telemetry-project'; $py="$base\adapters\provenance-python\.venv\Scripts\python.exe"; $ss="$base\bin\signalscout.exe"
1. Create the empty db file first; no angle brackets in paths.
2. Start: & $ss --db <abs db path> serve --listen 127.0.0.1:4318 --retention 0 (wait for port 4318).
3. Run the probe FROM the adapter directory (Push-Location "$base\adapters\provenance-python"): & $py -m scout_probe --config <yaml> --doctor <abs json path> --path <abs project dir> --manifest --drain-timeout 30 <module>:<function> -- <project args>. --doctor is required. PowerShell shows probe stderr as a red error; judge success by exit code and the printed summary.
4. Hot functions may overflow the default 256-item queue; use --queue-max-items 1024 --queue-max-bytes 2000000. Capturing complex return objects fails (drop_invalid); capture scalars. Unsampled calls are unknown, never satisfied.
5. uv is not on PATH; use the adapter venv's Python directly. Doctor passing does not prove scout_probe is importable outside the adapter directory (a fix is queued below).

CURRENT STATE AND QUEUE
1. Run 4 planter prompt: the owner may already have run or be about to run the revised planter (folder C:\Users\yiche\blind-test-run4, sealed C:\Users\yiche\Desktop\sealed-private\python-bug-run4). Ask the owner whether it finished. If so, take its final reply (paths, SHA-256, problem statement, run command, file count, symptom fraction), then write the BASELINE GATE prompt (copy project to C:\Users\yiche\baseline-run4, fresh read-only baseline, 150 commands) and then the gate GRADER prompt. Only on PASS GATE write the TESTER prompt for a new folder (blind-test-run4t), then its grader prompt.
2. Build PR A: doctor probe-importability check (branch codex/doctor-probe-check): check scout_probe is findable from the adapter directory, `-m scout_probe --help` exits 0, print a verified working probe command block with absolute paths, JSON keys additive only.
3. Build PR B (after A is merged): `signalscout probe-candidates` static AST scan (read-only, never executes target code) that suggests functions to probe and drafts a probe config (scalar-only capture, hot functions sampled, 25 candidates cap, new-file-only output). It must say a candidate list is a starting point and absence means nothing. Its value is only proven if it points a tester at the cause on a bug that passes the difficulty gate.
4. Housekeeping: record lessons in docs/open-questions.md (run 2 easy and baseline-matched; run 3 invalid because the symptom did not reproduce via the documented command; doctor did not check probe importability); Go module path rename PR; M4/explorer screenshots by the owner; rerun the Python suite a few times (privacy HTTP-403 connection reset 10054 flake seen twice, cause unknown).
5. Later (not started): explanation layer step 2 (hypothesis/experiment generation with a mock client), step 3 (value linking or first-divergence, design-only until a blind test shows need), step 4 (pre-registered benchmark vs log-grep baseline). Do not start LLM integration, value linking or first-divergence without the owner's explicit go.
Known weaknesses to keep stating honestly: probe cost misses the original 100/250 us budget (owner relaxed to 330/830 after seeing results); default recorder settings lose runs at exit; Python only; developer scale only; privacy covers synthetic data only; the verifier checks that citations exist and numbers match, not that claims are true.

HOW TO START
Reply with: (1) a three-line summary of where things stand, (2) the one question you need answered (did the run 4 planter finish, and has the owner merged PR A or B), and (3) if the answer is already clear, the single next step with its full prompt. Finish every later reply with a short running log: what changed, what is pending, what the owner must do.