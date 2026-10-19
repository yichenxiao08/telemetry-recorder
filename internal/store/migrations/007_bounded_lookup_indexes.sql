-- Measured 1M-row trace lookups and service/template counts scanned signals.
-- Restrict hot signal indexes to the kinds queried. Existing run, context and
-- retention indexes remain necessary; service/time alone already met 500 ms.
CREATE INDEX resources_service ON resources(json_extract(attrs,'$."service.name"'));
CREATE INDEX signals_trace_time ON signals(json_extract(data,'$.trace_id'),ts_ns)
 WHERE kind IN ('span','log');
CREATE INDEX signals_log_resource_time ON signals(resource_id,ts_ns) WHERE kind='log';
