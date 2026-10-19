-- Preserve raw telemetry and runs. Replace the multiplicative evidence x run
-- materialization with at most three rules per qualifying span resource.
DROP TABLE context_matches;
DROP TABLE context_links;
CREATE TABLE context_scopes(
 run_id TEXT NOT NULL REFERENCES runs(run_id) ON DELETE CASCADE,
 rule TEXT NOT NULL CHECK(rule IN ('exact_resource','host.name','service.instance.id')),
 matched_resource_id TEXT NOT NULL REFERENCES resources(resource_id),
 match_value TEXT NOT NULL,
 start_ns INTEGER NOT NULL,
 end_ns INTEGER NOT NULL,
 PRIMARY KEY(run_id,rule,matched_resource_id));
CREATE TABLE context_settings(
 singleton INTEGER PRIMARY KEY CHECK(singleton=1),
 before_ns INTEGER NOT NULL, after_ns INTEGER NOT NULL);
-- UNION ALL keeps the three resource lookups indexable. Each provenance row is
-- unique by scope and signal. DISTINCT collapses multiple matching rules only
-- in the evidence-level compatibility view.
CREATE VIEW context_matches AS
 SELECT c.run_id,s.evidence_id,c.rule,c.matched_resource_id
 FROM context_scopes c JOIN resources a ON a.resource_id=c.match_value
 JOIN signals s ON s.resource_id=a.resource_id AND s.run_id=''
 AND s.kind IN ('log','metric') AND s.ts_ns>=c.start_ns AND s.ts_ns<=c.end_ns
 WHERE c.rule='exact_resource'
 UNION ALL
 SELECT c.run_id,s.evidence_id,c.rule,c.matched_resource_id
 FROM context_scopes c JOIN resources a ON json_extract(a.attrs,'$."host.name"')=c.match_value
 JOIN signals s ON s.resource_id=a.resource_id AND s.run_id=''
 AND s.kind IN ('log','metric') AND s.ts_ns>=c.start_ns AND s.ts_ns<=c.end_ns
 WHERE c.rule='host.name' AND json_type(a.attrs,'$."host.name"')='text'
 UNION ALL
 SELECT c.run_id,s.evidence_id,c.rule,c.matched_resource_id
 FROM context_scopes c JOIN resources a ON json_extract(a.attrs,'$."service.instance.id"')=c.match_value
 JOIN signals s ON s.resource_id=a.resource_id AND s.run_id=''
 AND s.kind IN ('log','metric') AND s.ts_ns>=c.start_ns AND s.ts_ns<=c.end_ns
 WHERE c.rule='service.instance.id' AND json_type(a.attrs,'$."service.instance.id"')='text';
CREATE VIEW context_links AS SELECT DISTINCT run_id,evidence_id FROM context_matches;
