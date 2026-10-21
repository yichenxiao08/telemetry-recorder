# Branch and PR audit — 2026-10-21

No GitHub PR was merged. Annotated tags are immutable:

- blind-test-engine -> 072820ab28d22ae1bd7a081f58addc498be9f04b;
  tag object 0a301550f2b5bf757378063f24019c9f1750d833, pushed. Build passed;
  Actions run 39554316037 passed Windows and Ubuntu.
- blind-test-engine-2 -> cc68e4a24f89963549d702e886281ddb31413917;
  tag object b27d2e4875741b888e1d9829785069bd2122dff2, pushed **after**
  Actions 39562411063 passed Windows and Ubuntu. It contains probe usability
  fixes independently of P0. Its full Python suite passed 60 tests.

All remote branches and associated PRs at audit time:

| Remote branch | Head before final P0 documentation | PR / state | Base | Head CI |
| --- | --- | --- | --- | --- |
| main | 859487c | — | — | passed |
| codex/m4b | e260a09 | #1 merged | main | passed |
| codex/provenance-spec | 0bdd3ec | #2/#3 merged | main | passed |
| codex/slice1-lane | 05e9add | #4 open | main | passed |
| codex/lane-diagnostic | 70ce5ac | #5 open | codex/slice1-lane | passed |
| codex/lane02-contract | 37f2151 | #8 opened | codex/lane-diagnostic | passed |
| codex/lane02-taskb | bef1fc8 | #9 opened | codex/lane02-contract | passed |
| codex/recorder-capture-loss | eb9225e | #10 opened | codex/lane02-taskb | passed |
| codex/wrapper-overhead | 474f53f | #6 open | codex/recorder-capture-loss | passed |
| codex/single-output-trace | 072820a | #7 opened | codex/wrapper-overhead | passed |
| codex/probe-usability | cc68e4a | #13 opened | codex/single-output-trace | passed |
| codex/p0-spec-gates | 5a0c251 | #11 opened | codex/single-output-trace | passed |
| codex/p0-capture-privacy | aa7e6e6 integration, then final docs | #12 opened | codex/single-output-trace | final CI checked in completion report |

PRs: https://github.com/yichenxiao08/telemetry-recorder/pulls
Local-only master at 6c49592 is an old checkout reference, not an open remote PR.

Recommended owner merge order:
**#4 -> #5 -> #8 -> #9 -> #10 -> #6 -> #7 -> #13 -> #11 -> #12**.
Retarget a child to main after its parent merges. Preserve ancestry with merge
commits; squashing/rebasing a parent requires explicit coordination with all
stacked children. Independent docs #11 and usability #13 can be reviewed in
parallel, but both touch provenance-spec.md and need an ordinary docs merge.

P0 paused at ddf1add, then WIP b79d85a. Usability branched directly from the
original engine tag, not P0. P0 resumed at b79d85a and locally merged the tested
engine-2 tag at aa7e6e6, preserving both parents. The only textual merge conflict
was scout_probe.py: retain both policy settings and usability/counter/preflight
settings. No engine code was changed to integrate these branches. README,
provenance-spec.md, questions and ideas are shared documentation; the P0 code
also touches recorder, intake and the server. The annotated engine-2 tag has no
P0 implementation. Owner's two unstaged report/question edits were preserved
outside all commits, with a retained recovery stash.
