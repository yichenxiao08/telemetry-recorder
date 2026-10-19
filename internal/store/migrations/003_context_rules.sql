CREATE INDEX resources_host ON resources(json_extract(attrs,'$."host.name"'));
CREATE INDEX resources_instance ON resources(json_extract(attrs,'$."service.instance.id"'));
CREATE TABLE context_matches(
 run_id TEXT NOT NULL,
 evidence_id TEXT NOT NULL,
 rule TEXT NOT NULL CHECK(rule IN ('exact_resource','host.name','service.instance.id')),
 matched_resource_id TEXT NOT NULL REFERENCES resources(resource_id),
 PRIMARY KEY(run_id,evidence_id,rule,matched_resource_id),
 FOREIGN KEY(run_id,evidence_id) REFERENCES context_links(run_id,evidence_id) ON DELETE CASCADE);
-- Old membership could include resources seen only in correlated logs. Context
-- must be grounded in a resource that appears in the run's spans.
DELETE FROM run_resources WHERE NOT EXISTS(
 SELECT 1 FROM signals s WHERE s.run_id=run_resources.run_id
 AND s.resource_id=run_resources.resource_id AND s.kind='span');
DELETE FROM context_links WHERE NOT EXISTS(
 SELECT 1 FROM signals s JOIN run_resources rr ON rr.resource_id=s.resource_id
 WHERE s.evidence_id=context_links.evidence_id AND rr.run_id=context_links.run_id);
INSERT INTO context_matches SELECT c.run_id,c.evidence_id,'exact_resource',s.resource_id
 FROM context_links c JOIN signals s ON s.evidence_id=c.evidence_id;
