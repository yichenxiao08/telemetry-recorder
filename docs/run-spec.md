# Run specification (M1)

Web runs use the 32-character lowercase hex trace ID. ML runs use the nonempty resource attribute `signalscout.run.id`, with `signalscout.workload` naming the workload. OTLP span status and both HTTP status attribute conventions are supported. The root is the earliest parentless span. Unknown workload is displayed explicitly.

Times are signed UTC Unix nanoseconds. OTLP zero timestamps use observed log time when provided; otherwise zero is retained rather than replaced with an invented event time. Idle timeout uses server receipt time. Closed runs stay closed when late evidence arrives; their evidence interval, context and outcome can change. Open run state survives restart.

A web run passes unless its root errors, its HTTP status is >=500, or it exceeds a configured duration threshold. A trace without a root is degraded. ML runs fail on wrapper failure (`signalscout.run.failed=true` or record `signalscout.run.outcome=fail`), an exception/job.failed event, an error span, or non-finite `ml.loss`. ML failure is sticky. Ambient context cannot change outcome.

Resources use a SHA-256 identity over their complete OTLP resource and schema URL. All raw record and scope metadata is preserved. The environment snapshot retains resource attributes; configuration uses resource keys prefixed `signalscout.config.`. Conflicting resource metadata is last accepted value; all originals remain in resources.

Ambient logs and metrics are stored with empty run ownership. On run close, same-resource ambient records inside its inclusive start/end interval become context links. Links are many-to-many and refresh for late evidence. Explicitly owned evidence is never reassigned as ambient context.

Span evidence IDs include trace ID and span ID. Span events and log IDs are stable content hashes. Metric records have a stable content hash, retaining separate series at identical names/timestamps. These extend the draft IDs to avoid collisions. The store keeps every point type in its original JSON; scalar values are also indexed as REAL, with non-finite representations stored separately.

Zero timestamps remain in raw records but do not widen an otherwise timed run's interval. Resource identity sorts top-level attributes so SDK ordering changes do not break matching. Run ownership of an existing evidence ID cannot change; such a conflicting batch is rolled back.
