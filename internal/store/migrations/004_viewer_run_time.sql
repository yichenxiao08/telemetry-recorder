-- Cursor pagination without an unbounded sort when no workload is selected.
CREATE INDEX runs_time ON runs(started_at DESC,run_id);
