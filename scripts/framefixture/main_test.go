package main

import (
	"encoding/json"
	col "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"testing"
)

func TestFramePlainOTLPAndStaleInput(t *testing.T) {
	req, facts := generate(1, 60)
	data, err := protojson.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var decoded col.ExportTraceServiceRequest
	if err = protojson.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	spans := decoded.ResourceSpans[0].ScopeSpans[0].Spans
	if len(spans) != 120 || len(facts) != 60 {
		t.Fatal("record count")
	}
	for i := 0; i < 60; i++ {
		var d map[string]any
		if err = json.Unmarshal([]byte(spans[i*2+1].Events[0].Attributes[2].Value.GetStringValue()), &d); err != nil {
			t.Fatal(err)
		}
		if len(d["inputs"].([]any)) != 4 || len(spans[i*2+1].Events) != 5 {
			t.Fatal("missing lineage")
		}
	}
	f := facts[7].(map[string]any)
	if f["inputs"].(map[string]any)["previous_frame"] != 5 {
		t.Fatal("stale frame not exercised")
	}
	if encode(facts) == encode(func() []any { _, other := generate(2, 60); return other }()) {
		t.Fatal("seed ignored")
	}
}
