# Proposed static probe scan

Design only, 2026-10-23. No scanner is implemented by this document.

## Scope and non-goals

Suggest where a developer could capture useful function boundaries in a Python project. Deterministic local AST analysis reads source only: no imports, execution, network or LLM. It reports “worth probing because X”; a signal establishes neither a bug nor causation. It does not diagnose a failure, infer dynamic values, rewrite code, or replace human review.

## CLI and output

Proposed: `signalscout probe scan --root <directory> --limit 20 --max-files 2000 --max-bytes 20000000 --json <new-path>`. Bounds must be positive and capped; defaults are shown here. Exclude virtual environments, caches, vendor trees and explicit `--exclude` paths before reading. Reject symlink/reparse traversal outside root. Parse errors and unreadable files are reported with coverage counts. Stop at deterministic path-order limits, state truncation and never imply complete coverage.

Text lists rank, repository-relative file:line, qualified function name, score, each triggering signal and a ready-to-use module:function selector where resolvable. JSON schema version 1 contains candidates, signals, selector or unsupported reason, coverage, exclusions, truncations and limits. Sort paths ordinally and use stable tie breaks. A candidate with an unresolved package layout gets no invented selector. A top-level class method uses module:Class.method; unsupported async/generator/nested/descriptor cases remain visible with reasons and no runnable suggestion.

Example text: `1 src/demo/cache.py:12 cache.lookup — worth probing because it mutates module cache; selector demo.cache:lookup`. Resolve modules relative to an explicit `--import-root`, or an unambiguous ordinary package root. Namespace packages and ambiguous src layouts require that flag. Include a suggested config entry; selector alone does not configure capture. Field capture stays OFF until explicitly allowlisted.

Print the same working probe command template as doctor, using its selected Python and adapter resolution. The scan itself never runs doctor subprocess checks or target code; unresolved interpreter/adapter paths stay explicit placeholders. Proposed PowerShell shape:

```powershell
signalscout probe scan --root 'C:\project' --import-root 'C:\project\src' --limit 20
Push-Location -LiteralPath '<adapter-directory>'
try {
  & '<selected-python.exe>' -E -s -B -m scout_probe --config '<config.yaml>' --doctor '<new-doctor.json>' --path '<import-root>' --manifest --drain-timeout 30 'module:main' -- '<target-arguments>'
  $probeExit = $LASTEXITCODE
} finally {
  Pop-Location
}
```

No uv or WSL assumption. Hot candidates recommend deterministic sampling first (for example every_nth 25, max_per_run 40), prominently incomplete. Optional paired queue settings are shown as placeholders, with item and byte memory estimates; no universal queue size is prescribed. The earlier scenario-specific 3,280 items is not a general default. Warn that queue bounds, drain timeout and scalar allowlists do not guarantee capture; unsampled calls are unknown for absence and invariant claims.

## Signal catalogue

| Signal | Detects | Misses | False-positive risk |
| --- | --- | --- | --- |
| Module mutable state | Module assignments to list/dict/set literals or constructors, and functions directly referencing those names | Aliases, imported state, custom mutable types | Read-only constants and intentional registries |
| Cache decorators | Syntactic functools.cache/lru_cache decorators, including directly resolvable import aliases | Reexports and runtime decorators | Correct memoization |
| Partial dict key | Direct module/local persistent dictionary subscript keys syntactically reference fewer formal parameters than the function body uses | Semantic dependencies, helpers, aliases, implicit state | Unused contextual parameters and deliberately equivalent keys; never calls a key incorrect |
| Loop call site | Direct resolvable function/method calls inside for/while comprehensions or loop bodies | Dynamic dispatch, callbacks, recursion, external callers | Ordinary efficient loops; lexical nesting is not a measured rate |
| Shared access | Direct reads/subscript writes or recognized mutator calls on module mutable names or self attributes | Interprocedural ownership, aliases, object lifetime | Intended object state; self alone does not prove sharing across calls |
| Mutable defaults | List/dict/set literals and recognized mutable constructors in defaults | Custom factories, indirectly mutable objects | Intentional accumulation |

Keep direct syntactic observations separate from inferred associations: a potential partial-key candidate explicitly states the syntactic subset and uncertainty. If exported to structured reports, AST locations are source observations, not native telemetry evidence IDs; inferred diagnostic hypotheses require human-supplied alternatives and experiments. No tier is automatically upgraded.

## Ranking rule

Freeze before evaluation: one point each for module mutable access, cache decorator, partial-key syntax, loop-call site, shared writes and mutable default. Count each signal once per candidate; no field names, project labels or bug hints affect scores. Shared access and module access can overlap: disclose both rather than claiming independent evidence. Rank descending total, then normalized path, line and qualified name. Report ties and each signal separately. Weights are a prioritization heuristic, not probabilities or statistical scores. Scan limits apply before ranking, candidate limit after ranking; print both sources of truncation.

## Synthetic fixture plan

Write new fixtures without examining any blind-test source or report. Include correct and incorrect caching, intentional mutable defaults, read-only dictionaries, alias-mediated state deliberately missed, instance-local state, pure hot loops, nested callbacks, generator/async exclusions, package/src layouts, malformed files and resource-limit cases. Fixtures assert exact signal provenance, deterministic ordering, selector correctness, no execution (top-level side-effect sentinel), no external reads and bounded output. Keep a correct case for every signal and at least one failure outside this signal catalogue.

## Pre-registered evaluation

Before scanning any evaluation project, commit scanner source hash, rules, fixture hashes, exclusions and budgets. An independent author supplies a fresh synthetic planted project never used for development, retains the answer separately and runs grading. The developer does not open the answer. Register top N = 5 and 20; record eligible function count, ties, parse coverage, unsupported targets, truncations and whether each planted function appears at either cutoff. Compare the unchanged naive baseline: every directly resolved function called inside a loop, sorted by path/line/name at the same cutoffs. Report absolute ranks and all misses, including defects whose relevant function is unprobeable. A single project is feasibility evidence only. Require multiple independently authored projects before claiming general usefulness; no post-result tuning or reruns count as holdouts. Prior blind runs 2–4 are excluded from fixtures and evaluation.

## Known weaknesses

The synthetic module_read_only.py control declares a module-level dictionary that is only read by read_only, never mutated or aliased. It still produces module_mutable_state (one point), with no other signal. This is a known false positive: the signal recognizes a mutable container declaration and reference, not observed mutation or a bug.

Scores cluster at 1 to 2. Ties are broken by path and line (then qualified name), so top-N membership within a tie is arbitrary with respect to any bug. A good top-20 result must be read with the tie counts.

Enumeration has no hard memory or time bound before caps apply. File and byte caps bound source reads; walking and collecting eligible paths can still consume unbounded time and memory on a sufficiently large tree.

Static syntax cannot identify runtime ownership, actual call frequency or which boundary carries useful evidence. Dynamic Python, aliases and dependencies can defeat resolution. More signals may merely select larger stateful functions. Candidate recall does not show that recording those boundaries explains a failure. Default-off capture still requires developer choices and may omit necessary fields. Source paths and identifiers can themselves be sensitive; keep output local and explicitly exclude private trees.

## Ordered implementation proposal

1. Freeze schema, bounds, path policy and synthetic fixtures; review the contract.
2. Build safe file enumeration and AST parsing with coverage and no-execution tests.
3. Add direct-name resolution and individual signals with negative controls.
4. Add deterministic ranking, text/JSON and selector/config suggestions.
5. Reuse doctor command-template rendering; validate PowerShell quoting and unsupported paths.
6. Freeze source, then conduct the independent pre-registered evaluation and publish every miss before considering additional signals.

## Questions for the owner

- Should unresolved dynamic candidates appear in the primary list or a separate bounded section?
- Is explicit --import-root acceptable for ambiguous layouts?
- Who will author and grade the independent fixtures without sharing their answers?
- What project-size budget is representative for the intended users?
- Is top-5/top-20 recall sufficient to justify a later usability trial, and what minimum coverage should that trial require?

## Owner decisions — 2026-10-23

The future planter and grader are separate fresh Codex chats; the answer stays sealed. First evaluation is feasibility evidence only. Before results, the usability bar is fixed: planted function in the top 20 on most evaluation projects, beating the unchanged naive loop-call baseline at that cutoff, with at least 90% parse coverage of eligible files and every gap stated. Unresolved dynamic candidates belong in a separate bounded section. Ambiguous layouts require explicit --import-root; otherwise print no selector and a reason. Size defaults remain 2000 files and 20 MB.

## Steps 3-4 implementation notes — 2026-10-23

The scanner now uses embedded fixed Python AST analysis; target source remains stdin data, never imported or executed. Ranking is one point for each distinct signal (module_mutable_state, cache_decorators, partial_dict_key, loop_call_site, shared_access, mutable_defaults). Sort descending total, then relative slash path, line and qualified function name. Ties retain ordinal ranks and report their full pre-limit group size. There are no name boosts or bug labels. Partial-key syntax is an uncertain association, all other signal facts are observations. The CLI limit defaults to 20 and caps at 200 per independently bounded primary/unresolved section. No command template or evaluation is implemented.

Schema v1 additions are documented in probe-scan-schema-v1.md. Analysis returns at most 256 candidate entries per file, with an explicit truncation count, and caps subprocess output at 4 MiB (five-second deadline unchanged). Analysis overflow/failure becomes an error with incomplete coverage. This analysis cap precedes ranking, so a later strong candidate may be omitted; report it as incomplete, never as absent. Enumeration still has no hard memory/time bound. Candidate printing is bounded by admitted records and subprocess output, not by a fixed total text-byte cap.

Conservative resolution scope: same-file direct function calls and class-name method calls, direct functools imports/aliases, syntactic module containers and class-declared shared mutable attributes. Instance receivers are unresolved at loop call sites. Local-only dictionaries and instance attributes without class-declared mutable storage do not establish shared state. Read-only module mutable literals can still trigger module_mutable_state; a local read-only dictionary is the negative control. Correct cache decorators remain worth probing; complete-key memoization is negative only for partial_dict_key. Pure hot loops trigger loop_call_site but no shared/cache signal. These correct cases do not imply a defect. Partial keys are recognized only on direct module dictionaries; implicit state, constants-only keys, aliases and local persistent objects are known misses. Locally rebound module names are excluded. Rebinding/import shadowing beyond that conservative scope, call dispatch, aliases and interprocedural ownership remain limitations.

Signal misses: module_mutable_state misses imported/custom/aliased containers; cache_decorators misses reexports and dynamically installed decorators; partial_dict_key misses semantic dependencies and alias/implicit keys; loop_call_site misses callbacks, cross-module calls and dynamic receivers (reported separately when a syntactic attribute call is visible); shared_access misses aliased mutations and shared objects acquired through helpers; mutable_defaults misses custom or indirectly mutable defaults. A synthetic off-by-one defect and an alias-mediated mutation deliberately produce no candidate. No blind project or report was consulted.

## Step 5 notes — 2026-10-23

Doctor and scan now share internal/probetemplate rendering. Pre-extraction golden report text covers no interpreter, a quoted Unicode/space path, and the py launcher; doctor bytes remain unchanged. Scan adds probe_command (string array) to schema v1 and appends the same block to text, explicitly noting --doctor is required. Config, doctor-output and target arguments remain placeholders.

Template context uses only filesystem/PATH inspection, never Python version/import/help execution or doctor checks. Existing AST parsing still runs its fixed parser as before. Adapter resolution requires an existing directory with ordinary scout_probe.py; unresolved paths get explicit placeholders. --adapter can supply a directory; --python remains the parser interpreter. --path uses a statically resolvable explicit import root, otherwise an inferred unambiguous scan root, otherwise a placeholder. An exactly-one-primary selector is filled only without errors/truncation, so --limit 1 cannot silently choose among hidden candidates. Multiple candidates keep module:function. Templates do not establish runnable target arguments, capture success or dependency availability. No command is executed by rendering, no uv assumption, and no evaluation was started.
