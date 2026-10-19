CREATE TABLE resources(resource_id TEXT PRIMARY KEY, attrs TEXT NOT NULL, raw TEXT NOT NULL);
CREATE TABLE runs(run_id TEXT PRIMARY KEY, workload TEXT NOT NULL, domain TEXT NOT NULL,
 started_at INTEGER NOT NULL, ended_at INTEGER NOT NULL, outcome TEXT NOT NULL,
 failure_marker TEXT NOT NULL, env TEXT NOT NULL, config TEXT NOT NULL, closed INTEGER NOT NULL,
 last_seen INTEGER NOT NULL, state TEXT NOT NULL);
CREATE INDEX runs_workload ON runs(workload, outcome, started_at);
CREATE INDEX runs_retention ON runs(closed, last_seen);
CREATE TABLE signals(evidence_id TEXT PRIMARY KEY, kind TEXT NOT NULL CHECK(kind IN ('span','event','log','metric')),
 run_id TEXT NOT NULL, resource_id TEXT NOT NULL REFERENCES resources(resource_id),
 ts_ns INTEGER NOT NULL, end_ns INTEGER NOT NULL, received_at INTEGER NOT NULL, data TEXT NOT NULL);
CREATE INDEX signals_run ON signals(run_id, kind, ts_ns);
CREATE INDEX signals_context ON signals(resource_id, run_id, ts_ns);
CREATE INDEX signals_retention ON signals(run_id, received_at);
CREATE TABLE run_resources(run_id TEXT REFERENCES runs(run_id) ON DELETE CASCADE,
 resource_id TEXT REFERENCES resources(resource_id), PRIMARY KEY(run_id,resource_id));
CREATE TABLE context_links(run_id TEXT REFERENCES runs(run_id) ON DELETE CASCADE,
 evidence_id TEXT REFERENCES signals(evidence_id) ON DELETE CASCADE, PRIMARY KEY(run_id,evidence_id));
CREATE VIEW spans AS SELECT evidence_id,run_id,json_extract(data,'$.span_id') AS span_id,
 json_extract(data,'$.parent_span_id') AS parent_span_id,json_extract(data,'$.name') AS name,
 ts_ns AS start_ns,end_ns,(end_ns-ts_ns)/1000000.0 AS duration_ms,
 json_extract(data,'$.status') AS status,json_extract(data,'$.attrs') AS attrs FROM signals WHERE kind='span';
CREATE VIEW events AS SELECT evidence_id AS event_id,run_id,json_extract(data,'$.span_id') AS span_id,
 json_extract(data,'$.name') AS name,ts_ns,json_extract(data,'$.attrs') AS attrs FROM signals WHERE kind IN ('event','log');
CREATE VIEW metric_points AS SELECT evidence_id,run_id,json_extract(data,'$.name') AS name,
 ts_ns,json_extract(data,'$.value') AS value,json_extract(data,'$.attrs') AS attrs FROM signals WHERE kind='metric';
CREATE TABLE artifacts(artifact_id TEXT PRIMARY KEY,run_id TEXT,uri TEXT,content_hash TEXT,kind TEXT,redaction_status TEXT,retention TEXT);
CREATE TABLE diff_results(diff_id TEXT PRIMARY KEY,failing_run_id TEXT,baseline_spec TEXT,created_at INTEGER,results TEXT);
