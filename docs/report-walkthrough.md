# Structured evidence reports (synthetic data only)

The deterministic half of revised M7 is implemented. `report build` composes
existing results; `report verify` rebuilds facts from the store and checks the
draft. There is no LLM, hypothesis generator or experiment executor.

## Reproduce in native PowerShell

```powershell
task build
go run ./scripts/reportcheck --out .cache/structured-report/demo.json
$demo = Get-Content .cache/structured-report/demo.json -Raw | ConvertFrom-Json
$db = $demo.database
$report = $demo.report_path
$request = Join-Path (Split-Path $report) 'request.json'
.\bin\signalscout.exe --db $db report build --request $request
.\bin\signalscout.exe --db $db report build --request $request --json |
    Set-Content -Encoding utf8 .cache/structured-report/draft.json
.\bin\signalscout.exe --db $db report verify --input $report --json
```

The check starts a real `serve` binary, exports ordinary JSON OTLP, verifies
`runs list` has 40 runs, creates immutable cohorts through the existing writer,
builds/verifies a report and resolves ten native evidence anchors over HTTP.
Four runs violate `output == inputs.x + 1`. Crossing violations with the stored
flag cohorts gives 2 violated / 18 satisfied for **each** flag value: the flag
has no marginal association with these minority violations. The original
operational outcomes remain passing. No success is relabeled.

The generator and independent unit fixture use their own synthetic increment
rule, rather than copying any external application. The checker can additionally
build/verify reports for all 15 frozen lane-01 cases:

```powershell
Copy-Item -LiteralPath .cache/declared-relations/lane01-copy.db -Destination .cache/structured-report/lane01-copy.db
go run ./scripts/reportcheck --lane-db .cache/structured-report/lane01-copy.db --out .cache/structured-report/acceptance.json
```

## Closed version-one request

All time bounds are integer Unix nanoseconds. They bound signal event time and
must contain the pinned cohorts' original run-start selection windows. No new
live selection or ranking is performed. Example shape (replace identifiers and
use a suitable real time window):

```json
{
  "version": 1,
  "from_ns": 1767225600000000000,
  "to_ns": 1767225660000000000,
  "record_limit": 1000,
  "cohorts": ["rules-pinned"],
  "comparisons": [{"problem": "problem-pinned", "reference": "reference-pinned"}],
  "traces": [{"target": "value:run-id/output", "run_id": "run-id", "budget": {"depth": 8, "nodes": 32}}],
  "records": [{"id": "event:native-id", "kind": "native", "run_id": "run-id"}]
}
```

`cohorts` composes their pinned relation snapshots and completeness assessments.
Legacy cohorts without manifests produce `unassessed`, not `complete`. Comparison
sources preserve the existing explain-cohort JSON, including symptom/candidate
median/tail sections and invariants. Legacy profiles without presentation sections
retain their original differences. Trace sources preserve the existing walker and
bounded read scope; no tracing behavior changes. The `profile` member of each trace
is an optional complete existing analysis profile, including its version invariant.

`records` accepts native span/event/log/metric IDs. Specify the actual owning run
if there is one; ambient signals have an empty run. A context link does not change
ownership. This version does not claim an ambient record is a run-owned record.
An optional `cohort_id` on a native record requires it to be that cohort's pinned
decision. Logical output IDs are cited only when the existing trace walker resolves
them without integrity findings or read truncation. Cohort citations are existing
cohort IDs resolved by primary key, not new evidence records.

## Report and verification contract

Version-one reports contain the original request, unchanged source outputs,
required limitation notes and claims. Only three tiers exist:

- **observation**: recorded values, rule counts, completeness classes, trace scope
  and declared-version invariant results. Unknown/insufficient states remain
  explicit. Rule counts concern recorded decisions; unsampled calls are unknown.
- **association**: only a supported existing cohort comparison finding. Alongside
  the claim are problem/reference raw numerators, observed denominators, run counts,
  session counts, shared sessions, independence-known flag, missing/excluded counts,
  exact pinned selection policy and disjoint-run overlap policy. Session identifiers
  are clustering proxies; their distinct count does not establish independence.
- **hypothesis**: only explicitly supplied JSON. Alternatives, missing evidence,
  experiment, predicted result and falsifying result are mandatory. Verification
  always returns `unsupported` for the explanation/experiment, even if citations
  resolve. Nothing in this task generates a hypothesis.

Every generated claim binds to a deterministic `fact` and exact ordered citations.
The verifier rebuilds facts; submitted source JSON or claim quantities are never
used as truth. Changing source output or removing coverage notes rejects the
document. A claim receives `valid`, `invalid` or `unsupported`, with a reason.
`valid` means structural support for the quoted evidence, not semantic truth.

All numeric quotations are in `numbers`, keyed by JSON pointer paths in the fact's
detail (RFC6901 `~0` and `~1` escaping); these are separate from envelope selectors.
Quantities must include the complete canonical numeric map. Integer quantities
match exactly. Nonintegral quantities use absolute tolerance
`1e-9 * max(1, abs(expected))`. Nonfinite values are invalid. Integers outside the
exact float64 range (absolute value greater than 2^53) are not exported as numeric
quotations; exact timestamp strings/integers remain in the original sources. Depth
above twelve is not converted into numeric assertions. Unknown element markers and
capture-off/hash-only subtrees never become numeric values. Rule `unknown` **counts**
are retained. This release does not implement arbitrary free-text number extraction:
ASCII digits in claim text are rejected; put numeric assertions in `numbers`.

The case-insensitive phrases `caused`, `root cause`, `because of` and `due to` are
rejected in claim text, including negated/quoted uses as a conservative lexical
policy. The schema has no causal field. Synonyms, words spelling out numbers,
misleading interpretation, semantic truth, causal overinterpretation in free text
and subjective reply quality cannot be verified by this mechanism. These claims
remain the author's responsibility; no diagnostic accuracy claim follows.

Citation validity rate = valid scoped, fact-related citations / submitted citations.
Claim support rate = valid claims / all claims; unsupported hypotheses remain in
the denominator. Empty denominators report zero rates, not perfect validity. An
unknown negative search with no recorded root has no invented citation and returns
`unsupported`. Other claims require citations. Invalid claims make `report verify`
exit 1 after printing its report; supported observations and explicitly unsupported
hypotheses exit 0. Input/scope/source errors also exit 1 with an error, rather than
producing a partial verification.

## Supplying a hypothesis explicitly

The input shape is `{"version":1,"claims":[...]}`. Use a distinct claim ID, tier
`hypothesis`, state `proposed`, empty `numbers`, and existing scoped citations copied
from the draft. For example:

```powershell
$draft = Get-Content $report -Raw | ConvertFrom-Json
$hypothesis = @{
    version = 1
    claims = @(@{
        id = 'supplied-experiment'; fact = ''; tier = 'hypothesis'
        text = 'A repeat with the injected offset removed is proposed'
        state = 'proposed'; evidence = $draft.claims[0].evidence; numbers = @{}
        hypothesis = @{
            alternatives = @('A mismatch in the captured input')
            missing_evidence = @('A controlled repeat without the synthetic offset')
            experiment = 'Hold inputs fixed and remove the injected offset in a separate synthetic repeat'
            predicted_result = 'The increment rule is satisfied on the previously affected decisions'
            falsifying_result = 'Those decisions still violate the increment rule'
        }
    })
}
$hypothesis | ConvertTo-Json -Depth 64 | Set-Content -Encoding utf8 .cache/structured-report/hypothesis.json
.\bin\signalscout.exe --db $db report build --request $request --hypotheses .cache/structured-report/hypothesis.json --json
```

## Bounds, immutability and remaining limits

Each request has a time range of at most 31 days and a required shared snapshot
record cap of 1..10,000. At most four distinct pinned cohorts, two comparisons,
four traces, 64 explicitly named native records, 1,000 run partitions, 4,096 claims
and 512 numeric quantities per fact. Trace calls each use the same explicit
per-source record cap, depth <=64 and nodes <=128; therefore the total read bound
is at most five record caps plus up to 64 primary-key native lookups and bounded
metadata/run lookups. This is a per-source cap, not a global trace-sharing budget.
Queries run under a ten-second deadline. Snapshot signals use `signals_run` with
time predicates and LIMIT; EXPLAIN tests require the index. Metadata is selected by
primary key with LIMIT. Every snapshot has a 32 MiB cumulative payload ceiling and
1 MiB per-signal transfer ceiling; inputs/output reports are bounded at 32 MiB.
Oversize requests/records/caps reject the whole request, without silent truncation.

Pinned relation/completeness snapshots describe their creation time. They are not
reassessed as new evidence arrives. Retention can evict pinned evidence; a later
verification then fails to rebuild and reports that explicitly. Trace sources are
separate bounded read snapshots: concurrent late arrivals or retention can change
them, making verification fail source equality. Quiesce a synthetic store when
reproducibility matters. No report is persisted, no second writer is opened, no
migration is added, no artifact URI is followed and no application is executed by
the verifier. Hypothesis tests, human labels, consent/access/deletion/retention for
real content and LLM integration remain outside this implementation.
