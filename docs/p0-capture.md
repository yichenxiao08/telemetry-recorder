# P0 capture gate — synthetic only, 2026-10-21

For the separate explicitly opted-in public-data experiment, see
[public-data-experiment.md](public-data-experiment.md). This document describes
the existing synthetic-only mode.

This is an **opt-in synthetic privacy gate**, not permission to use personal
data. The owner delegated the compatibility choice: retain the existing synthetic
mode and add an explicit policy mode, strict default OFF within that mode. Legacy
mode is not privacy-safe. Neither mode supplies consent, authentication, access
control, deletion, or a real-data retention policy.

## Enable both gates

Use a fresh store; existing databases are not scrubbed. No migration is added.
The server selects one explicit profile for all incoming records; an untrusted
producer cannot select its own permissive profile.

```powershell
task build
.\bin\signalscout.exe --db .cache/p0-synthetic.db serve --capture-policy .\testdata\privacy\policy.json --capture-profile synthetic
```

In the producer's existing signalscout.yaml, set `capture_policy` to the policy
path relative to that config file and `capture_profile: synthetic`. The probe's
argument/return/receiver allowlists and the policy must both allow a field.
Receiver capture still requires individual builtin attributes; whole self/cls
capture is forbidden. See adapters/provenance-python/README.md for argv and probe
commands. Absence of the P0 policy preserves the historical synthetic wire path.

## Validated format

JSON version 1; maximum 64 KiB, 32 profiles, 128 fields per profile. Duplicate
keys, unknown keys/versions, credential field names and invalid rules are errors.
An empty fields map captures no application fields. Unknown/malformed decision
envelopes become capture-off/unknown, never a copy of the original payload.

Paths include `inputs.frame`, `inputs.self.score`, `output.lane`,
`attributes.safe.number`, `resource.attributes.service.name`, and `scope.otlp.name`.
Only explicitly allowlisted leaves survive recursive map traversal. Arrays,
arbitrary free text, log bodies and unapproved metadata are not supported capture
types. Rule kinds:

| Kind | Policy and boundary |
| --- | --- |
| number | Finite numeric value, absolute value below 10^10; bool separate. This is a typed allowlist, not a personal-value detector. |
| boolean | Actual boolean only. |
| enum | Exact match in at most 32 approved, short nonpersonal literals; credential/email/card/SQL/URI-like literals rejected. The profile author must approve the semantics. |
| hmac | Explicit `equality: true`, `value_class: email`, runtime key of at least 32 bytes, and `key_version`. No general string hashing. Credentials/card/SQL rejected. |
| artifact-sha256 | Exact approved nonpersonal literal, `nonpersonal: true`, `hash_basis: redacted`; hashes those permitted redacted bytes only. No URI fetching, arbitrary files or original-content hashing. |

HMAC-SHA256 scope includes domain, key version, profile and field path. Set the
same base64 key via `SIGNALSCOUT_P0_HMAC_KEY` in producer and server environments;
do not put it in policy, source, telemetry or command history. Rotation requires a
new key/version; tokens across fields/profiles/versions are not comparable.
Existing correctly formatted versioned tokens are preserved on the second gate,
so this is not token authentication or proof of equality to an original value.
No plain hash of a low-entropy personal value is provided. Policy/config files
themselves must not contain secrets or personal data.

## Enforcement and fidelity

The producer sanitizes a snapshot **before buffering/export**. Caller values,
return identity and exception identity remain unchanged. Failure increments a
privacy drop counter rather than raising into the application. The intake
sanitizes decoded protobuf before normalized/indexed records, resource IDs,
templates, raw canonical copies or writer submission are made. JSON, protobuf
and gzip use that same gate; raw unredacted copies are never retained alongside
sanitized indexed fields. Unknown wire fields are discarded, including on kept
attributes. Bodies, error/review text, resource/scope strings, schema/artifact
URIs, nested record JSON and split unallowlisted attributes are removed.

This conservative first policy intentionally loses detail: decision IDs/refs,
step identity, chosen/rule/config metadata and lineage declarations are omitted
from sanitized envelopes. A fixed detail field says identity and semantic
metadata are unknown. Capture-off fields are omitted, not replaced with numeric
zero. OTLP metric values and repeated numeric payloads are OFF in this version;
there is no scalar-metric allow rule yet. This can make strict-mode provenance
or invariants unavailable. It does not change trace/ranking algorithms or invent
unknown values. Native trace/span IDs, timestamps, flags and protocol enums
remain transport metadata; covert encoding in IDs/timestamps is not addressed.

Existing resource/evidence content hashes are computed after sanitization; they
are storage identities, not a personal-data equality feature. A hostile policy
author could misdeclare a personal value as a number or a nonpersonal literal.
These boundaries are why this remains synthetic only rather than a promise to
detect credentials in arbitrary encodings or free text.

Strict-mode HTTP metadata writes return 403: annotations and cohort metadata have
no approved text policy. GET/HEAD remain available; legacy synthetic APIs and M4b
ownership/navigation are unchanged. Existing stores, direct CLI metadata writes,
unrelated exporters, application logs outside this producer, console output,
process memory and external artifacts are not retroactively protected. Do not
mix legacy writers or direct metadata CLI writes into a strict-mode database.

## Measured checks and misses

- Full combined Python suite: **66 passed**, including real probe -> intake ->
  store, method fields, exception identity, capture before queue, and rejection
  of annotation POSTs. Existing lane/pipeline/trace/safety tests unchanged.
- Go planted-secret storage check: **0 leaks / 77 stored cells** in six signal
  records, seven planted strings (539 cell/string comparisons). Every column of
  every populated application table is inspected, including raw and indexed
  representations. Traffic covers traces/logs/metrics, JSON/protobuf and gzip;
  duplicate delivery remains idempotent. This is a finite synthetic check, not a
  statistical real-data leak estimate.
- Tests first exposed one leaking cell in the initial 69-cell fixture: OTLP
  `sfixed64` scalar values were not cleared. The corrected fixture passed 0/69.
  Adding histogram arrays exposed one leak in the expanded 77-cell fixture;
  repeated scalar payloads are now cleared and it passes 0/77. A separate test
  reproduced retained unknown protobuf payloads on kept KeyValues; those are
  now cleared too. These misses are retained in the validation record.
- An initial Python run failed with Windows executable-sharing errors while
  task build replaced a server binary used by tests. Sequential unchanged suite
  and build passed. Preserve that failed run; do not weaken assertions.
- gofmt, go vet, golangci-lint, go test -race ./..., task build pass locally.
  CI is checked on both Windows and Ubuntu at the final revision.
- Added P0 probe cost is **unmeasured against both 100/250 us and 330/830 us
  budgets**. Redaction adds producer work; the separate method-cost observation
  predates this policy and cannot qualify it. No optimization or budget change.

See benchmark/results/p0-capture-privacy/validation.md and python-tests.txt.
The riskiest assumption is that an explicitly approved typed field is actually
nonpersonal in the target domain. A default-OFF gate reduces exposure but cannot
validate that semantic assertion. Real data still requires a separately approved
privacy/access/retention design and stronger tests.
