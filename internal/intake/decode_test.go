package intake

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/signalscout/signalscout/internal/runs"
	metricscol "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	common "go.opentelemetry.io/proto/otlp/common/v1"
	resource "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestMetricVariantsAndMetadata(t *testing.T) {
	data := []byte(`{"resourceMetrics":[{"scopeMetrics":[{"metrics":[
	{"name":"gauge","gauge":{"dataPoints":[{"asInt":"9223372036854775807","timeUnixNano":"1"}]}},
	{"name":"counter","sum":{"aggregationTemporality":2,"isMonotonic":true,"dataPoints":[{"asDouble":3,"timeUnixNano":"2"}]}},
	{"name":"histogram","metadata":[{"key":"extra","value":{"stringValue":"kept"}}],"histogram":{"aggregationTemporality":2,"dataPoints":[{"timeUnixNano":"3","count":"2","sum":3,"bucketCounts":["1","1"],"explicitBounds":[1]}]}},
	{"name":"exponential","exponentialHistogram":{"aggregationTemporality":1,"dataPoints":[{"timeUnixNano":"4","count":"1","scale":0,"positive":{"offset":1,"bucketCounts":["1"]}}]}},
	{"name":"summary","summary":{"dataPoints":[{"timeUnixNano":"5","count":"1","sum":1,"quantileValues":[{"quantile":0.5,"value":1}]}]}},
	{"name":"nonfinite","gauge":{"dataPoints":[{"asDouble":"Infinity","timeUnixNano":"6"}]}}
	]}]}]}`)
	var message metricscol.ExportMetricsServiceRequest
	if err := protojson.Unmarshal(data, &message); err != nil {
		t.Fatal(err)
	}
	batch, err := decode(&message, runs.NewNormalizer())
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Signals) != 6 {
		t.Fatalf("lost metric points: %d", len(batch.Signals))
	}
	if batch.Signals[0].Value == nil || !bytes.Contains(batch.Signals[0].Raw, []byte("9223372036854775807")) {
		t.Fatal("lost exact int64 original")
	}
	for _, index := range []int{2, 3, 4} {
		if batch.Signals[index].Value != nil {
			t.Fatal("invented scalar for aggregate")
		}
		if !json.Valid(batch.Signals[index].Raw) {
			t.Fatal("invalid aggregate JSON")
		}
	}
	if !bytes.Contains(batch.Signals[2].Raw, []byte("kept")) {
		t.Fatal("lost metric metadata")
	}
	if batch.Signals[5].Value != nil || batch.Signals[5].ValueText != "Infinity" {
		t.Fatal("lost nonfinite value")
	}
}

func TestResourceAttributeOrderDoesNotBreakContext(t *testing.T) {
	attrs := []*common.KeyValue{{Key: "service.name", Value: &common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: "shop"}}}, {Key: "host.name", Value: &common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: "localhost"}}}}
	d := &decoder{}
	first := d.makeResource(&resource.Resource{Attributes: attrs}, "schema")
	second := d.makeResource(&resource.Resource{Attributes: []*common.KeyValue{attrs[1], attrs[0]}}, "schema")
	if d.err != nil {
		t.Fatal(d.err)
	}
	if first.ID != second.ID {
		t.Fatal("resource identity depends on attribute order")
	}
	if bytes.Equal(first.Raw, second.Raw) {
		t.Fatal("raw metadata order was not preserved")
	}
}

func TestDuplicateLogOccurrencesAndStableIDs(t *testing.T) {
	queue := &captured{}
	h := New(queue)
	data := []byte(`{"resourceLogs":[{"scopeLogs":[{"logRecords":[{"body":{"stringValue":"same"}},{"body":{"stringValue":"same"}}]}]}]}`)
	w := send(h, "/v1/logs", "application/json", data, false)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	first := queue.batch.Signals
	if len(first) != 2 || first[0].ID == first[1].ID {
		t.Fatal("identical records in one batch collapsed")
	}
	w = send(h, "/v1/logs", "application/json", data, false)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if first[0].ID != queue.batch.Signals[0].ID || first[1].ID != queue.batch.Signals[1].ID {
		t.Fatal("replay changed IDs")
	}
}
