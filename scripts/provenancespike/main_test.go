package main

import (
	"encoding/json"
	"testing"

	"github.com/signalscout/signalscout/internal/model"
	metricscol "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	tracecol "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestOriginalOTLPReplay(t *testing.T) {
	s := model.Signal{Kind: "span", Resource: model.Resource{Raw: json.RawMessage(`{"schema_url":"","resource":{}}`)}, Scope: json.RawMessage(`{"schema_url":"","scope":{}}`), Raw: json.RawMessage(`{"name":"synthetic","traceId":"11111111111111111111111111111111","spanId":"1111111111111111"}`)}
	path, body, err := request(s)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	var traces tracecol.ExportTraceServiceRequest
	if err := protojson.Unmarshal(b, &traces); err != nil {
		t.Fatal(err)
	}
	if path != "/v1/traces" || traces.ResourceSpans[0].ScopeSpans[0].Spans[0].Name != "synthetic" {
		t.Fatal("lost span")
	}
	s.Kind = "metric"
	s.Raw = json.RawMessage(`{"metric":{"name":"synthetic.gauge","description":"","unit":"1","type":"gauge"},"point":{"timeUnixNano":"123","asDouble":5}}`)
	_, body, err = request(s)
	if err != nil {
		t.Fatal(err)
	}
	b, err = json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	var metrics metricscol.ExportMetricsServiceRequest
	if err := protojson.Unmarshal(b, &metrics); err != nil {
		t.Fatal(err)
	}
	points := metrics.ResourceMetrics[0].ScopeMetrics[0].Metrics[0].GetGauge().DataPoints
	if len(points) != 1 || points[0].GetAsDouble() != 5 {
		t.Fatal("lost point")
	}
	s.Raw = json.RawMessage(`{"metric":{"type":"unsupported"}}`)
	if _, _, err := request(s); err == nil {
		t.Fatal("unsupported types must not be silently lost")
	}
}
