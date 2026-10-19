DROP VIEW spans;
CREATE VIEW spans AS SELECT s.evidence_id,s.run_id,json_extract(s.data,'$.span_id') AS span_id,
 json_extract(s.data,'$.trace_id') AS trace_id,json_extract(s.data,'$.parent_span_id') AS parent_span_id,
 json_extract(s.data,'$.name') AS name,json_extract(r.attrs,'$."service.name"') AS service,
 s.ts_ns AS start_ns,s.end_ns,(s.end_ns-s.ts_ns)/1000000.0 AS duration_ms,
 json_extract(s.data,'$.status') AS status,json_extract(s.data,'$.attrs') AS attrs
 FROM signals s JOIN resources r ON r.resource_id=s.resource_id WHERE s.kind='span';
DROP VIEW events;
CREATE VIEW events AS SELECT evidence_id AS event_id,run_id,json_extract(data,'$.span_id') AS span_id,
 json_extract(data,'$.name') AS name,ts_ns,json_extract(data,'$.attrs') AS attrs,
 json_extract(data,'$.body') AS body,json_extract(data,'$.template') AS template
 FROM signals WHERE kind IN ('event','log');
