// Synthetic spike only: read-only export of original OTLP, never saved diffs.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/signalscout/signalscout/internal/model"
	_ "modernc.org/sqlite"
)

func request(s model.Signal) (string, map[string]any, error) {
	var resource, scope map[string]json.RawMessage
	if err := json.Unmarshal(s.Resource.Raw, &resource); err != nil {
		return "", nil, err
	}
	if err := json.Unmarshal(s.Scope, &scope); err != nil {
		return "", nil, err
	}
	outer := map[string]any{"resource": resource["resource"], "schemaUrl": resource["schema_url"]}
	inner := map[string]any{"scope": scope["scope"], "schemaUrl": scope["schema_url"]}
	path, group, scopes, records := "/v1/traces", "resourceSpans", "scopeSpans", "spans"
	var record any = s.Raw
	switch s.Kind {
	case "span":
	case "log":
		path, group, scopes, records = "/v1/logs", "resourceLogs", "scopeLogs", "logRecords"
	case "metric":
		path, group, scopes, records = "/v1/metrics", "resourceMetrics", "scopeMetrics", "metrics"
		var raw struct {
			Metric map[string]json.RawMessage `json:"metric"`
			Point  json.RawMessage            `json:"point"`
		}
		if err := json.Unmarshal(s.Raw, &raw); err != nil {
			return "", nil, err
		}
		var kind string
		if err := json.Unmarshal(raw.Metric["type"], &kind); err != nil {
			return "", nil, err
		}
		keys := map[string]string{"gauge": "gauge", "sum": "sum", "histogram": "histogram", "exponential_histogram": "exponentialHistogram", "summary": "summary"}
		key, ok := keys[kind]
		if !ok {
			return "", nil, fmt.Errorf("unsupported replay metric %s", kind)
		}
		data := map[string]any{"dataPoints": []any{raw.Point}}
		if kind == "sum" || kind == "histogram" || kind == "exponential_histogram" {
			data["aggregationTemporality"] = raw.Metric["aggregation_temporality"]
		}
		if kind == "sum" {
			data["isMonotonic"] = raw.Metric["is_monotonic"]
		}
		metric := map[string]any{"name": raw.Metric["name"], "description": raw.Metric["description"], "unit": raw.Metric["unit"], key: data}
		var extra map[string]json.RawMessage
		if b := raw.Metric["extra"]; len(b) > 0 {
			if err := json.Unmarshal(b, &extra); err != nil {
				return "", nil, err
			}
			if metadata := extra["metadata"]; len(metadata) > 0 {
				metric["metadata"] = metadata
			}
		}
		record = metric
	default:
		return "", nil, fmt.Errorf("unsupported kind %s", s.Kind)
	}
	inner[records] = []any{record}
	outer[scopes] = []any{inner}
	return path, map[string]any{group: []any{outer}}, nil
}

func run(ctx context.Context) error {
	dbPath := flag.String("db", "", "existing synthetic database")
	export := flag.Bool("export", false, "export original OTLP as NDJSON")
	flag.Parse()
	db, err := sql.Open("sqlite", "file:"+*dbPath+"?mode=ro")
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	enc := json.NewEncoder(os.Stdout)
	if *export {
		rows, err := db.QueryContext(ctx, "SELECT s.data,r.raw FROM signals s JOIN resources r ON r.resource_id=s.resource_id WHERE kind IN ('span','log','metric') ORDER BY ts_ns,evidence_id LIMIT 10000")
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var raw, resourceRaw []byte
			if err := rows.Scan(&raw, &resourceRaw); err != nil {
				return err
			}
			var s model.Signal
			if err := json.Unmarshal(raw, &s); err != nil {
				return err
			}
			s.Resource.Raw = resourceRaw
			path, body, err := request(s)
			if err != nil {
				return err
			}
			if err := enc.Encode(map[string]any{"path": path, "body": body}); err != nil {
				return err
			}
		}
		return rows.Err()
	}
	output := map[string]any{}
	for name, query := range map[string]string{"runs": "SELECT run_id,workload,started_at,ended_at,outcome,config FROM runs ORDER BY started_at LIMIT 1000", "signals": "SELECT evidence_id,kind,run_id,data FROM signals ORDER BY ts_ns,evidence_id LIMIT 10000", "scopes": "SELECT run_id,rule,start_ns,end_ns FROM context_scopes LIMIT 10000"} {
		if name == "scopes" {
			var exists int
			if err := db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='context_scopes'").Scan(&exists); err != nil {
				return err
			}
			if exists == 0 {
				output[name] = []any{}
				output["legacy_context_schema"] = true
				continue
			}
		}
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return err
		}
		cols, err := rows.Columns()
		if err != nil {
			_ = rows.Close()
			return err
		}
		list := []map[string]any{}
		for rows.Next() {
			values := make([]any, len(cols))
			args := make([]any, len(cols))
			for i := range values {
				args[i] = &values[i]
			}
			if err := rows.Scan(args...); err != nil {
				_ = rows.Close()
				return err
			}
			item := map[string]any{}
			for i, k := range cols {
				item[k] = values[i]
			}
			list = append(list, item)
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return err
		}
		output[name] = list
	}
	return enc.Encode(output)
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
