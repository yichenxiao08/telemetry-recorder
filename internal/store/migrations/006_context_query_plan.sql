-- Keep interval scopes before evidence in every rule. Without explicit join
-- ordering SQLite can scan every scope for each ambient record on large stores.
DROP VIEW context_links;
DROP VIEW context_matches;
CREATE VIEW context_matches AS
 SELECT c.run_id,s.evidence_id,c.rule,c.matched_resource_id
 FROM context_scopes c CROSS JOIN resources a ON a.resource_id=c.match_value
 CROSS JOIN signals s ON s.resource_id=a.resource_id AND s.run_id=''
 AND s.kind IN ('log','metric') AND s.ts_ns>=c.start_ns AND s.ts_ns<=c.end_ns
 WHERE c.rule='exact_resource'
 UNION ALL
 SELECT c.run_id,s.evidence_id,c.rule,c.matched_resource_id
 FROM context_scopes c CROSS JOIN resources a ON json_extract(a.attrs,'$."host.name"')=c.match_value
 CROSS JOIN signals s ON s.resource_id=a.resource_id AND s.run_id=''
 AND s.kind IN ('log','metric') AND s.ts_ns>=c.start_ns AND s.ts_ns<=c.end_ns
 WHERE c.rule='host.name' AND json_type(a.attrs,'$."host.name"')='text'
 UNION ALL
 SELECT c.run_id,s.evidence_id,c.rule,c.matched_resource_id
 FROM context_scopes c CROSS JOIN resources a ON json_extract(a.attrs,'$."service.instance.id"')=c.match_value
 CROSS JOIN signals s ON s.resource_id=a.resource_id AND s.run_id=''
 AND s.kind IN ('log','metric') AND s.ts_ns>=c.start_ns AND s.ts_ns<=c.end_ns
 WHERE c.rule='service.instance.id' AND json_type(a.attrs,'$."service.instance.id"')='text';
CREATE VIEW context_links AS SELECT DISTINCT run_id,evidence_id FROM context_matches;
