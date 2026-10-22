# Declared relations and decoded selectors (2026-10-22)

Synthetic observations only. A declaration checks the recorded values of **one
decision**. It neither infers code reads nor establishes a cause. It needs no
reference cohort and does not change any ranking formula.

## Selectors

Selectors address the **decoded** decision, not raw envelope JSON:

| Selector | Decoded value |
|---|---|
| `inputs.x` | allowed inline value of the input named `x` |
| `inputs.state.3` | fourth item of that input's sequence |
| `output.1` | second item of the allowed return sequence |
| `output.0.3` | fourth item inside the first returned item |
| `output.lane` | map key `lane` in a returned object |
| `chosen.option`, `verdict`, `reason_code` | recorded envelope selectors |

There is no `output.inline` or numeric index into the envelope's raw input list.
Map segments are exact keys; numeric segments index sequences only when the
current value is a sequence. Indices are zero based; out-of-range/negative indices
are unknown. There are no wildcards, recursive search or guessing between keys.

Omit `selector_syntax` to preserve the original literal `Split(".")`, including
literal backslashes. Opt into `"selector_syntax":"escaped-v1"` to use `\.` for
a literal dot and `\\` for a literal backslash. For example the JSON selector
`"inputs.self\\.score"` selects the single input name `self.score`, while
`"inputs.self.score"` traverses a nested map. Escaped syntax rejects empty
segments, dangling/unknown escapes and unknown syntax versions. Both profile
grammars have a 256-byte/eight-segment bound. The syntax is pinned in profiles
and cohorts. Trace's existing selector behavior is not extended by this task.

Categorical features accept strings, booleans and finite safe integers
(-2^53..2^53). Fractional numbers, null and structured values remain missing.
String-only comparisons preserve their original labels and ordering. If either
arm contains a boolean/integer, categories are explicitly typed (`string:`,
`boolean:`, `integer:`), so string `"1"` and integer `1` cannot collide. The
Laplace total-variation formula and constants are unchanged.

## Version 1 grammar

Profiles use the existing validated JSON file (64 KiB max). See the executable
example in `testdata/relations/profile.json`. `relations` is optional; when
present it contains `version:1` and 1..16 uniquely named rules. Each rule declares
`id`, nonempty `unit`, an explicit finite nonnegative `tolerance`, optional `when`
and boolean `assert`. Units are developer assertions; no conversion or unit
inference occurs. Use one compatible numeric unit per rule; a boolean-only rule
declares `boolean`. This does not verify dimensional correctness.

Each expression node has exactly one form, with no unrelated fields:

| `op` | Payload / arity | Result |
|---|---|---|
| `field` | `field`: validated decoded selector | recorded number/boolean |
| `number` | finite `number` literal | number |
| `boolean` | `boolean` literal | boolean |
| `add`, `sub` | two numeric `args` | number |
| `abs` | one numeric argument | number |
| `eq`, `ne` | two numbers or two booleans | boolean |
| `lt`, `le`, `gt`, `ge` | two numbers | boolean |
| `and`, `or` | two booleans | boolean |
| `not` | one boolean | boolean |

Bounds use `and(ge(value,lower),le(value,upper))`. There are no strings as rule
operands, division, function calls, imports, SQL, assignments, cross-record
references or arbitrary evaluation. A rule plus its precondition is limited to
64 nodes and depth eight (root zero). Known literal type errors are rejected;
recorded type mismatches yield unknown. A wholly absent declared-version invariant
is now allowed so a relation-only profile does not need a fabricated version
rule. A partly supplied invariant is rejected; existing invariant equations stay.

For numeric operands x,y and tolerance t, `eq` tests x in [y-t,y+t]; `ne`
negates it. `lt/le/gt/ge` test x < y-t / x <= y+t / x > y+t / x >= y-t.
Both interval edges count as inside. Boolean equality is exact. Arithmetic uses
IEEE float64; no hidden epsilon, rounding, coercion or score is added. Nonfinite
operands/intermediate values, overflowing subtraction/bounds and wrong types are
unknown. Very large magnitudes can lose precision: declare a suitable tolerance
and units instead of interpreting close floating point values as exact decimals.

Missing, off, hash-only and unsupported element capture is unknown, never zero.
The reserved recorder `unknown` marker makes its whole subtree unavailable.
Boolean operators evaluate all their operands (a true `or` cannot hide missing
capture). A false known `when` yields `not_applicable` without inspecting `assert`;
unknown `when` yields unknown. Otherwise `assert` yields satisfied or violated.

## Commands and immutable selection

```powershell
task build
go run ./scripts/relationscheck
$db = '<fresh fixture database shown in the generated run directory>'
$bounds = @('--from','<run-start lower RFC3339>','--to','<exclusive RFC3339>','--limit','40')
& .\bin\signalscout.exe --db $db relations --profile testdata/relations/profile.json @bounds
& .\bin\signalscout.exe --db $db relations --profile testdata/relations/profile.json @bounds --json
# Start serve on the same database before creating a cohort through its writer.
& .\bin\signalscout.exe --db $db cohort create minority --profile testdata/relations/profile.json @bounds --violates absolute-difference
& .\bin\signalscout.exe --db $db relations --cohort minority --json
```

Live checks require run-start [from,to), <=31 days and a run limit 1..1000. The
report and selection read all owned span/event rows for those runs, including
late decision evidence, in one read-only snapshot. `--record-limit` is explicit
when supplied, defaults to 100,000 and permits at most 1,000,000 rows, including
non-decision spans/events. SQL LIMIT reads at most that cap plus one sentinel.
The indexed bounded run window and run/kind indexes remain; no history search
or migration is added. A ten-second CLI/metadata deadline still applies.

Exceeding either cap rejects the **entire** request and creates no report/cohort.
The run-cap error reports the exact eligible count and rejected count. The record
cap reports cap, sentinel read count, and every candidate run ID rejected. No
partial cohort is silently returned; narrow bounds or deliberately raise the
record cap. Existing frozen snapshots retain their serialized limits/bytes.
Metadata CLI responses have a 32 MiB ceiling; exceeding it reports a possible
committed write and asks the caller to inspect the ID before retrying, rather
than silently parsing a truncated body.

Checks evaluate every matching decision for the profile's single step, not just
the final call. Reports show step totals and per-run rows, with all four status
counts beside the recorded-decision denominator. Runs without a supported
decision are listed separately with zero fabricated decisions. Each row samples
at most three native violation witnesses, gives exact omitted-witness counts and
the last violation ID. Witness URLs resolve to existing run/event anchors.

`cohort create --violates ID` selects runs with **any recorded violation** of
that rule. It is exclusive with mark selection and scalar equality predicates.
Normal completeness exclusions/flags still apply. The cohort pins the declaration,
full bounded candidate counts and witness samples, as well as each selected run's
last violating decision ID. `relations --cohort` reads this immutable snapshot;
it does not recheck later evidence. Reports cover the candidate population,
including excluded partial runs; selected IDs and exclusions remain explicit in
the cohort. Existing cohort comparisons still compare one pinned decision per
run (here the last violation), not an aggregate of all calls. Per-run aggregate
ranking remains parked. Cohort JSON uses the existing table and serving writer.

Sampling is incomplete capture. Counts describe recorded decisions, not total
host calls. `sampled_records` counts records carrying sampling metadata; unsampled
calls remain unknown and are not classified as satisfied. Historical records
without sampling metadata are not certified complete. Manifests/completeness
are separate evidence; absence of a violation never establishes universal
satisfaction. Results are observations; any later cohort ranking is association.

## Synthetic acceptance

The plain Python Calculator has zero Signal Scout imports. Its 18-line config
probes the run boundary, method and a separate admission function, with explicit
field/return capture only.
The original application source needs zero instrumentation edits. For frames
0..39, y=x+3 and reported=abs(y-x)+1 exactly on frames divisible by ten. The
flag `frame % 4 < 2` has this independently tested contingency table:

| | flag true | flag false |
|---|---:|---:|
| rule violated | 2 | 2 |
| rule satisfied | 18 | 18 |

Thus the flag supplies no marginal distinction for the four affected runs.
All operational runs pass. This is constructed evidence, not real diagnostic
accuracy. Before intake evaluation the analytic expected counts are: equality,
ordering and dotted-key rules 40 satisfied each; absolute difference and bounds
36 satisfied/4 violated each; conditional 34 satisfied/6 not_applicable; capture-off
40 unknown. The real-binary checker asserts those exact counts, 40 `runs list`
entries, four relation-cohort members, identical pinned/live counts, and all
violation witnesses resolving over HTTP. Results and validation attempts are
retained under `benchmark/results/declared-relations/` at publication.
The `private=7` synthetic default argument is deliberately not allowlisted; its
value is off in the exported envelope and the rule cannot observe it.

A second plain target returns an object with an embedded sequence. Across 16
admission calls it has one deliberately inverted boolean decision and two
inverted input bounds. Analytic expectations: admission 15 satisfied/1 violated;
input ordering 14 satisfied/2 violated; clipping bounds 14 satisfied/2
not_applicable; object/sequence equality 16 satisfied. The checker verifies its
separate `runs list` partition, all counts and all violation URLs. Together the
two targets resolve eleven native violation anchors. Neither target imports
Signal Scout. Independent Python tests verify both fixtures' analytic counts.

Local validation: 172 Go top-level tests in 30 test-bearing packages pass with
the race detector; 119 Python tests pass on pinned Python 3.12.12. Formatting,
vet, golangci-lint and `task build` pass. All 15 frozen lane-01 reports/JSON/text
match byte for byte on a database copy. Thirty trace CLI text/JSON outputs match
the binary independently built from base main 13509b0. All 72 saved shop
structural/context comparisons match serialized result bytes except the two
inherently variable fields `created_at`/`elapsed_ns`; those fields are explicitly
excluded, not reported as byte-identical. Frozen result files were not modified.
The initial sandboxed Python attempt's 20 existing temp-directory cleanup errors
are retained alongside the passing native run. The one-target development check
is retained; the final check adds the second target and measures flag intersections
from stored cohorts. Hosted Windows/Ubuntu CI is reported on the PR head.

Remaining scope: the existing cohort decoder evaluates supported v1 decision
span events, not decision log envelopes. Unsupported envelope/profile versions
do not become invented decision samples; runs with no supported decision are
listed separately. Cross-record/temporal rules and automatic rule authoring are
not implemented. The riskiest assumption is that a developer's declared relation
expresses intended behavior over correctly captured values; a passing check
cannot establish actual reads or overall correctness.
