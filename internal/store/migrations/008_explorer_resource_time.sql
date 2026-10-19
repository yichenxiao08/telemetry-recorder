-- Explorer span/metric service partitions need time as an indexed bound.
-- 007's log and trace indexes and resources_service remain the lookup paths.
CREATE INDEX signals_resource_kind_time ON signals(resource_id,kind,ts_ns) WHERE kind IN ('span','metric');
