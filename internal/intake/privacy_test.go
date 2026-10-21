package intake

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/redaction"
	"github.com/signalscout/signalscout/internal/store"
	logscol "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	metricscol "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	tracescol "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	common "go.opentelemetry.io/proto/otlp/common/v1"
	logs "go.opentelemetry.io/proto/otlp/logs/v1"
	metrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	resource "go.opentelemetry.io/proto/otlp/resource/v1"
	trace "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestP0PlantedSecretsAbsentFromEveryStoredColumn(t *testing.T) {
	secrets := []string{"sk-syntheticSECRET123456789", "person@example.invalid", "4111111111111111", "Bearer syntheticAuth123", "synthetic_sql_literal", "https://artifact.invalid/?token=planted", "splitperson@example.invalid"}
	p, err := redaction.Parse([]byte(`{"version":1,"profiles":{"synthetic":{"fields":{"attributes.safe.number":{"kind":"number"}}}}}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "privacy.db")
	s, err := store.Open(context.Background(), path, store.Options{QueueSize: 4, IdleTimeout: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	kv := func(key, value string) *common.KeyValue {
		return &common.KeyValue{Key: key, Value: &common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: value}}}
	}
	text := strings.Join(secrets, " ") + " SELECT 'synthetic_sql_literal'"
	attrs := []*common.KeyValue{kv("Authorization", text), kv("review.excerpt", text), kv("exception.message", text), kv("artifact.uri", text), kv("piece.one", "splitperson@"), kv("piece.two", "example.invalid"), kv(text, text), kv("signalscout.record.json", `{"version":1,"inputs":[{"name":"review","capture":"allowed","inline":{"nested":"`+text+`"}}],"error":"`+text+`"}`), {Key: "safe.number", Value: &common.AnyValue{Value: &common.AnyValue_DoubleValue{DoubleValue: 10}}}}
	res := &resource.Resource{Attributes: attrs}
	scope := &common.InstrumentationScope{Name: text, Version: text, Attributes: attrs}
	traces := &tracescol.ExportTraceServiceRequest{ResourceSpans: []*trace.ResourceSpans{{Resource: res, SchemaUrl: secrets[5], ScopeSpans: []*trace.ScopeSpans{{Scope: scope, SchemaUrl: text, Spans: []*trace.Span{{TraceId: []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, SpanId: []byte{0, 0, 0, 0, 0, 0, 0, 1}, Name: text, StartTimeUnixNano: 10, EndTimeUnixNano: 20, Attributes: attrs, Status: &trace.Status{Message: text}, Events: []*trace.Span_Event{{TimeUnixNano: 11, Name: text, Attributes: attrs}}}}}}}}}
	logRequest := &logscol.ExportLogsServiceRequest{ResourceLogs: []*logs.ResourceLogs{{Resource: res, ScopeLogs: []*logs.ScopeLogs{{Scope: scope, LogRecords: []*logs.LogRecord{{TimeUnixNano: 12, Body: &common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: text}}, SeverityText: text, Attributes: attrs}, {TimeUnixNano: 13, Body: &common.AnyValue{Value: &common.AnyValue_IntValue{IntValue: 4111111111111111}}, Attributes: attrs}}}}}}}
	point := &metrics.NumberDataPoint{TimeUnixNano: 14, Attributes: attrs, Value: &metrics.NumberDataPoint_AsInt{AsInt: 4111111111111111}}
	metric := &metrics.Metric{Name: text, Description: text, Unit: text, Data: &metrics.Metric_Gauge{Gauge: &metrics.Gauge{DataPoints: []*metrics.NumberDataPoint{point}}}}
	metricScope := &metrics.ScopeMetrics{Scope: scope, Metrics: []*metrics.Metric{metric}}
	metricRequest := &metricscol.ExportMetricsServiceRequest{ResourceMetrics: []*metrics.ResourceMetrics{{Resource: res, ScopeMetrics: []*metrics.ScopeMetrics{metricScope}}}}
	handler := NewWithPolicy(s, p, "synthetic")
	for _, item := range []struct {
		path    string
		message proto.Message
	}{{"traces", traces}, {"logs", logRequest}, {"metrics", metricRequest}} {
		for _, encoding := range []string{"json", "proto"} {
			var body []byte
			if encoding == "json" {
				body, err = protojson.Marshal(item.message)
			} else {
				body, err = proto.Marshal(item.message)
			}
			if err != nil {
				t.Fatal(err)
			}
			content := "application/json"
			if encoding == "proto" {
				content = "application/x-protobuf"
			}
			response := send(handler, "/v1/"+item.path, content, body, true)
			if response.Code != 200 {
				t.Fatalf("%s %s: %d %s", item.path, encoding, response.Code, response.Body.String())
			}
		}
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	tables, err := db.Query("SELECT name FROM sqlite_schema WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for tables.Next() {
		var name string
		if err = tables.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if err = tables.Err(); err != nil {
		t.Fatal(err)
	}
	_ = tables.Close()
	checked, leaks := 0, 0
	for _, name := range names {
		rows, err := db.Query(`SELECT * FROM "` + strings.ReplaceAll(name, `"`, `""`) + `"`)
		if err != nil {
			t.Fatal(err)
		}
		columns, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			values := make([]any, len(columns))
			ptrs := make([]any, len(columns))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err = rows.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}
			for i, v := range values {
				checked++
				text := fmt.Sprint(v)
				if b, ok := v.([]byte); ok {
					text = string(b)
				}
				for _, secret := range secrets {
					if strings.Contains(text, secret) {
						leaks++
						t.Errorf("LEAK table=%s column=%s", name, columns[i])
					}
				}
			}
		}
		if err = rows.Err(); err != nil {
			t.Fatal(err)
		}
		_ = rows.Close()
	}
	if checked == 0 {
		t.Fatal("vacuous storage test")
	}
	var count int
	var data string
	if err = db.QueryRow("SELECT count(*) FROM signals").Scan(&count); err != nil || count == 0 {
		t.Fatal("no records persisted", err)
	}
	if err = db.QueryRow("SELECT data FROM signals WHERE kind='span' LIMIT 1").Scan(&data); err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if json.Unmarshal([]byte(data), &record) != nil {
		t.Fatal("invalid raw JSON")
	}
	t.Logf("synthetic planted-set leaks=%d; stored columns checked=%d; records=%d; placements across traces/logs/metrics JSON+protobuf+gzip", leaks, checked, count)
}
