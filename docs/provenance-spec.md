# Provenance capture specification (design v0)

Status: design plus source-level feasibility audit. Executable spike is pending. No new production
endpoint, table, resolver, capture policy or diff type is implemented here.
M4b is preserved in PR #1; M5–M8 remain unstarted.

## Meaning and identity

A decision record describes what a producer reports it used and chose at a step.
A lineage edge describes an output derived from an input through that step.
These are producer assertions, not independent proof of execution or causation.
Parent spans describe execution nesting; explicit lineage describes data dependence.
Domain meaning, units, stage identities and expectations live in profiles.
Host/time context is never a substitute for conversational or input lineage.

Proposed immutable logical IDs:
- decision:<run-id>/<record-id>
- lineage:<run-id>/<edge-id>
- value:<run-id>/<name>@<version>
- artifact:<opaque-id> (existing artifact-reference convention)
- annotation:<uuid> (proposed in problem-model.md)

Record and edge IDs are producer-assigned UUIDs or deterministic collision-resistant
IDs. Percent-encode individual path components. A changed payload requires a new
ID and supersedes reference. Same ID/different payload must become a conflict.
This enforcement and resolution are future work. Today only native
span:<trace>/<span>, event:<hash>/<ordinal>, log:<hash>/<ordinal> evidence resolves.
Future reports must retain logical-ID -> native-evidence mappings. Do not display
a proposed decision ID as an already functioning evidence link.

## Decision envelope

~~~json
{
  "version": 1,
  "id": "decision:RUN/RECORD",
  "run_id": "RUN",
  "profile": {"id": "lane", "version": "1"},
  "step": {"id": "lane.select", "version": "1",
           "span_ref": "span:TRACE/SPAN",
           "code_ref": {"revision": "COMMIT", "path": "relative/source", "symbol": "Select"}},
  "output": {"ref": "value:RUN/lane@1", "type": "string", "inline": "right"},
  "inputs": [
    {"ref": "value:RUN/gap@1", "type": "float64", "unit": "ratio",
     "inline": 0.51, "capture": "allowed"},
    {"ref": "artifact:opaque", "type": "document", "uri": "producer-managed-reference",
     "digest": "sha256:HEX", "capture": "reference-only"}
  ],
  "chosen": {"option": "right", "score": 0.51},
  "alternatives": [{"option": "stay", "score": 0.49, "reason_code": "below_threshold"}],
  "rule": {"id": "gap_threshold", "operator": ">=", "threshold": 0.5, "unit": "ratio"},
  "config_refs": ["config:RUN/lane_threshold"],
  "verdict": "selected",
  "reason_code": "gap_available",
  "complete": true
}
~~~

Scores use a producer/profile scale, not Signal Scout confidence. Omit unknown
scores rather than supplying zero. Rules may use an opaque versioned rule ID.
Config references must identify the snapshot actually used; the current merged
run snapshot cannot reliably represent contradictory per-service configurations.
Code locations are producer-supplied and revision-bound, never inferred from code.

A lineage record contains version, id, run_id, input_ref, output_ref, step_id,
decision_ref and relation="derived_from". Emit one edge per input/output pair.
Cross-run inputs identify their originating run. Session and previous-run links
are explicit. Record completeness and omissions. A future validator must detect
cycles, dangling references and ID conflicts with bounded graph traversal.
Incomplete capture yields unknown, not an invented complete graph.

## OTLP transport

Use existing /v1/traces span events or /v1/logs records. Event names:
signalscout.decision and signalscout.lineage. Log body is a short message,
not a second copy of captured values. Normal trace_id/span_id correlation applies.

Documented attributes:
- signalscout.record.version: integer 1
- signalscout.record.kind: decision | lineage | label
- signalscout.record.id: logical ID
- signalscout.record.json: complete envelope as UTF-8 JSON string
- signalscout.step.id, signalscout.output.ref, signalscout.chosen.option
- signalscout.verdict, signalscout.reason.code
- signalscout.session.id, signalscout.previous_run.id when applicable

The small selector fields duplicate allowlisted envelope fields for existing flat
attribute filters. JSON-in-a-string avoids SDK differences in nested span
attributes. Native arrays/kv-lists survive the decoder but are less portable
across language SDK span APIs. Proposed producer limits: 8 KiB/envelope, 32 inputs,
8 alternatives; larger collections use manifest refs with count/digest and
explicit omissions. These limits are proposed validation, not current guarantees.

No new endpoint is needed to preserve bytes. OTLP alone does not implement
logical-ID conflict checks, label semantics, graph queries, schema validation or
independent annotation timestamps. SDK truncation/dropped-event counts must mark
incompleteness. Delivery success alone does not establish complete provenance.

## Capture and privacy gate: before real personal content

Value capture is opt-in per profile AND field. Default to type, capture status
and opaque ref. Allowlisted small synthetic/nonpersonal scalars may be inline;
proposed string cap 256 UTF-8 bytes and envelope cap 8 KiB.
Large values remain external artifact references with URI, digest, size/media type,
access policy and retention. Signal Scout never copies artifacts or automatically
dereferences URIs. A digest cannot reconstruct an original input.

Redact before buffering/export AND before storage, covering body, attrs,
resource/scope, raw OTLP copies, nested record JSON, errors, review excerpts and
artifact URIs. Removing an indexed field while retaining its raw copy leaks data.
Unknown payload versions fail closed for value capture. Never capture credentials.
Plain hashes of low-entropy personal values/secrets allow dictionary attacks:
omit them or use a scoped keyed HMAC only where equality comparison is explicitly
allowed, with key/version policy. SHA-256 is suitable for permitted nonpersonal
artifact integrity. State whether a digest covers original or redacted content.

Move per-field allowlists, recursive redaction, planted-secret tests, consent,
access and retention/deletion checks from late M8 into prerequisite P0.
Until P0 passes, use synthetic data only. No production privacy gate exists today.
Human labels record judgments; no built-in quality judge is implied.

## Volume policy: design only

A. Retain small coverage metadata for every run plus a bounded, already-redacted
full-detail recent buffer; promote selected problem AND reference examples on a mark or label. Eviction is explicit and irreversible.
B. Always export a deterministic detail sample independently of labels; retain
capture probability and missing-edge information.

Recommend A for the first low-rate developer workflow, with independent reference
selection and a completeness manifest. Configure both time and byte caps only
after measurement. Late labels cannot recover evicted values. B is an explicitly
incomplete option for sustained high rates. Neither is implemented here.

Measure bytes/run/field, exporter CPU/latency, queue pressure, label delays,
eviction/promotion success, reference availability, graph completeness and privacy
exposure. Publish retained/total counts and selection probabilities where known.
Only promoting bad examples biases comparisons; do not claim population rates
from selected cohorts. Do not assume indefinite full capture at 60 fps is feasible.
The mixed-fixture 12 GB/hour extrapolation is not a scalar-only measurement.

## Smallest next slice

P0: a synthetic-first capture contract and privacy gate; P1 then adds one tiny producer
helper for one supported runtime/profile. Acceptance: plain OTLP preserves allowed
fields/IDs; conflict and truncation are visible; planted secrets reach no stored
raw/indexed field; default capture contains no user text; M4b ownership,
navigation, durability and scoring tests remain unchanged. The synthetic helper and external probes now exist only as the S0 feasibility spike; P0/P1 production gates remain unimplemented.

## Recorder safety and per-call cost contract

No host-path network I/O, queue-capacity waiting or recorder exceptions.
Bounded queue and bounded allowed-value serialization; overflow increments a
visible drop counter. Unreachable intake opens a cooldown circuit: calls become
no-op/dropped with explicit counters until retry, never waiting for a connection.
Exporter thread network operations have deadlines. run context exit records
best effort without flush/join; it never suppresses the application's own error.
Explicit flush/close is separate, timeout-bounded, and outside the hot-path budget.
A Python GIL/queue mutex and OS scheduling are not hard real-time guarantees;
"nonblocking" means no intentional I/O/capacity wait, not zero CPU time.

Budget fixed before measurement: p50 <=100 us, p99 <=250 us for small records
at 60 calls/sec and tight-loop load, reachable/unreachable, five trials.
Report serialization/enqueue wall time, counts, p50/p99 and drops. Keep misses.
The synthetic-only Python spike may demonstrate transport/safety; it does not
claim a production privacy gate or instrumentation beyond explicitly supported probes/libraries.

## Proposed record -- command wrapper (design only)

signalscout record -- <command> would start loopback intake with bounded storage,
set OTEL_EXPORTER_OTLP_ENDPOINT and protocol plus explicit capture/profile config,
run the child with native argument handling, and print a doctor summary:
runtime/integration coverage, received runs/records, drops, missing semantic
signals, privacy mode and evidence links. Propagate the child's exit code, drain
with a deadline and terminate owned intake. Preserve existing OTel providers and
allow explicit attachment to an existing recorder; no silent source mutation.
Unsupported frameworks are named, not called automatically instrumented.
The wrapper does not execute stored inputs or infer semantics. It is not built.

For comparable cheapness trials, fix the small payload before implementation:
four allowed scalar inputs, two scored alternatives and <=2 KiB envelope, plus
a stable step/output/config reference. Record payload bytes. Measure run-context
entry/exit separately and retain saturation/drop/no-op-path costs separately.
Large artifact hashing/I/O is explicitly outside this API and must not be
silently charged to or hidden inside a "nonblocking" recording call.

## S0 probe tier (2026-10-20 design amendment)

The PRIMARY path is external config-driven capture with zero target-source edits.
signalscout.yaml names Python modules/functions and per-field argument/return
allowlists, default OFF. The synthetic python -m scout_probe import hook wraps
supported synchronous functions in memory. It records call arguments before
mutation, return values, duration and exception type through the SAME bounded
record_decision/OTLP core. Unsupported targets/loaders/reloads are diagnosed.
No server or transport convention changes are needed.

A probe edge describes an argument supplied at a call boundary producing a
return; it cannot establish which parameters were actually read or discover
hidden globals/locals. The experimental complete flag covers captured call args,
not complete workflow/data-flow provenance. Rules and rejected alternatives
remain unobserved unless exposed in returns or explicitly instrumented.
The OPTIONAL second tier is record_decision for those semantic gaps only.
It must not become a prerequisite for ordinary capture.

The prototype doctor lists attached/failed/unsupported probes, configured capture
fields, serialized/dropped/exported counts, cooldown and synthetic privacy mode.
Standard installed requests tracing uses the same queue. Logging instrumentation
adds trace correlation; body export stays OFF. Other frameworks are unsupported.

Design-only CLI shapes (not Go commands implemented here):
- signalscout record --config signalscout.yaml -- <command>
- signalscout probe add module:function --capture args.field --capture return
- signalscout mark <run-or-output-ref> --problem <id> --label <value> --rubric <version>
- signalscout explain <problem-id> --reference <selection>
A future viewer Mark action would append an annotation through the existing
writer, showing reviewer/rubric, dimension and evidence refs. It must not rewrite
outcome/timestamps or manufacture a satisfactory reference. No marking UI or
these production CLI commands are built during S0.

### Executed probe boundary and clock policy

The synthetic CLI accepts a module:function entry point, not an arbitrary child
command. It neither edits source nor installs sitecustomize globally. Configuration
must identify every supported function and explicitly allow its argument names
and/or return. Doctor is required output, including unsupported targets and loss.
Safe support is limited to directly defined synchronous Python module functions.
Framework discovery, aliases already held elsewhere, async/generators, descriptors,
reload and native functions remain unsupported.

Short Windows wall-clock samples can be identical. The spike anchors each run to
UTC once, then uses monotonic elapsed nanoseconds for probe timestamps/durations
and run exit. Explicit supplied fixture timestamps remain explicit. This preserves
within-run order and durations; it does not synchronize clocks across processes
or establish sub-tick absolute order across independent runs. Standard SDK spans
keep their SDK's clock behavior. A constant-wall-clock regression test covers this.
Both original and corrected-candidate cost trials are retained with the SAME budget.

The optional call tier shares the exact recorder/queue used by probes. A missing
output ref can receive a unique run/span-local versioned ref; explicit refs remain
unchanged. Only explicit calls supply hidden rule/alternative declarations. These
are producer assertions, not independent proof that the producer's rule is true.
The minimal enriched B adds an import and a selection-semantic call (two lines);
all ordinary A/B capture works without those edits. Body capture remains OFF
unless the corresponding field is explicitly allowed. This is synthetic-only,
not a replacement for P0 recursive privacy/secret tests.
