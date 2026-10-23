# Mock hypothesis and experiment proposals

`report propose --input report.json` reads an existing structured report and
emits the version-one supplied-hypothesis JSON envelope. It does not edit the
report, open a database, use a network, call an LLM, or run an experiment.
Import remains an explicit `report build --hypotheses` operation. The existing
store-backed verifier remains authoritative at import and verification time.

The small `hypothesis.Client` interface receives only observation and association
claims, including their scoped citations and structured quantities. Sources,
request metadata and existing hypotheses are excluded. Its output is candidates
with citations, alternatives, missing evidence, change, measure, predicted result,
and a result against the hypothesis. The only implementation is `Mock`: the same
fixed prose template and the first available citation, in report order. It does
not interpret evidence or select a useful intervention. This mock demonstrates
plumbing and rejection behavior; it does not demonstrate diagnostic ability.

The wrapper creates only proposed hypothesis-tier claims, with empty quantities
and no association binding. Every candidate must have at least one alternative.
Experiment change and measurement are labeled in the existing experiment string;
the counterresult uses the existing falsifying-result field. Proposals are text
for human review. Citations must match complete citation tuples in the report's
observation/association tiers. This local check does not authenticate the report.
The existing supplied-JSON import rebuilds facts and checks scope against storage.
Even valid citations leave every hypothesis unsupported. No tier is upgraded.

Every generated prose field is submitted to the existing deterministic verifier's
claim-text checks, including lexical causal-phrase rejection. Invalid output is
rejected with the verifier reason and no partial output, repair, retry or fallback.
The lexical policy is conservative and incomplete; it is not a semantic detector.
Report and client input are isolated so client mutation cannot change the citation
pool or caller's report. Generated ID collisions are rejected.

Input is bounded to one MiB (raw CLI bytes and indented JSON representation), and
4,096 report claims. Candidate count is one through sixteen; candidate and final
indented output each have a 64 KiB budget including the final newline. Existing
verifier field limits also apply. Oversize input or output is rejected, never
silently truncated. The implementation is deterministic for identical inputs.

## Native PowerShell

Use an existing report and its original request/database:

```powershell
go build -o bin/signalscout.exe ./cmd/signalscout
.\bin\signalscout.exe report propose --input $report |
    Set-Content -Encoding utf8 .cache/proposed-hypotheses.json
if ($LASTEXITCODE -ne 0) { throw 'Proposal rejected' }
.\bin\signalscout.exe --db $db report build --request $request `
    --hypotheses .cache/proposed-hypotheses.json --json
```

Both Windows PowerShell's UTF-8 BOM and PowerShell 7 UTF-8 are accepted. No WSL
or uv is needed. The proposal command needs only the report file.

## Future client requirements and risks

A real client is out of scope. Adding one would require explicit decisions about
credentials, data disclosure, timeouts/cancellation, response size limits enforced
while reading, structured response parsing, reproducibility and model versioning.
It must implement the interface and retain this rejection/import boundary. The
current CLI always instantiates the mock; there is no real-client switch.

Fluent unsupported claims can appear convincing without explaining recorded
behavior. Citation laundering can make an untested explanation look supported
merely by attaching genuine evidence IDs. Resolving citations demonstrates only
that cited records exist in scope. Alternative hypotheses and proposed falsifiers
are mandatory structure, not proof of meaningful scientific alternatives or a
well-designed experiment. Human review and future evaluation would be required.
