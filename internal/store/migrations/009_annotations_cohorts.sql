CREATE TABLE annotations(
 id TEXT PRIMARY KEY, run_id TEXT NOT NULL REFERENCES runs(run_id),
 dimension TEXT NOT NULL, value TEXT NOT NULL, source TEXT NOT NULL,
 reviewer TEXT NOT NULL, recorded_at INTEGER NOT NULL, data TEXT NOT NULL,
 supersedes TEXT REFERENCES annotations(id), retracts TEXT REFERENCES annotations(id));
CREATE INDEX annotations_run_time ON annotations(run_id,recorded_at,id);
CREATE INDEX annotations_dimension_value ON annotations(dimension,value,recorded_at,id);
CREATE INDEX annotations_time ON annotations(recorded_at,id);
CREATE UNIQUE INDEX annotations_supersedes ON annotations(supersedes) WHERE supersedes IS NOT NULL;
CREATE UNIQUE INDEX annotations_retracts ON annotations(retracts) WHERE retracts IS NOT NULL;
CREATE TRIGGER annotations_no_update BEFORE UPDATE ON annotations BEGIN SELECT RAISE(ABORT,'append-only annotations'); END;
CREATE TRIGGER annotations_no_delete BEFORE DELETE ON annotations BEGIN SELECT RAISE(ABORT,'append-only annotations'); END;
CREATE TABLE cohorts(id TEXT PRIMARY KEY,created_at INTEGER NOT NULL,data TEXT NOT NULL);
CREATE TRIGGER cohorts_no_update BEFORE UPDATE ON cohorts BEGIN SELECT RAISE(ABORT,'immutable cohorts'); END;
CREATE TRIGGER cohorts_no_delete BEFORE DELETE ON cohorts BEGIN SELECT RAISE(ABORT,'immutable cohorts'); END;
