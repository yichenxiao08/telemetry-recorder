// framefixture is a plain OTLP reference producer for the synthetic frame call.
// It deliberately records call-boundary facts, not invisible local rules.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	col "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	common "go.opentelemetry.io/proto/otlp/common/v1"
	resource "go.opentelemetry.io/proto/otlp/resource/v1"
	trace "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func textAttr(key, value string) *common.KeyValue {
	return &common.KeyValue{Key: key, Value: &common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: value}}}
}
func versionAttr() *common.KeyValue {
	return &common.KeyValue{Key: "signalscout.record.version", Value: &common.AnyValue{Value: &common.AnyValue_IntValue{IntValue: 1}}}
}
func syntheticAttr() *common.KeyValue {
	return &common.KeyValue{Key: "signalscout.synthetic", Value: &common.AnyValue{Value: &common.AnyValue_BoolValue{BoolValue: true}}}
}
func encode(v any) string { b, _ := json.Marshal(v); return string(b) }

func generate(seed, count int) (*col.ExportTraceServiceRequest, []any) {
	req := &col.ExportTraceServiceRequest{}
	rs := &trace.ResourceSpans{Resource: &resource.Resource{Attributes: []*common.KeyValue{
		textAttr("service.name", "synthetic.frame.go"),
		{Key: "signalscout.synthetic", Value: &common.AnyValue{Value: &common.AnyValue_BoolValue{BoolValue: true}}},
	}}}
	scope := &trace.ScopeSpans{Scope: &common.InstrumentationScope{Name: "synthetic.plain.go"}}
	rs.ScopeSpans = []*trace.ScopeSpans{scope}
	req.ResourceSpans = []*trace.ResourceSpans{rs}
	lanes := []string{}
	facts := []any{}
	for frame := 0; frame < count; frame++ {
		prior := frame - 1
		if frame >= 2 && frame%7 == 0 {
			prior = frame - 2
		}
		previous := "left"
		if prior >= 0 {
			previous = lanes[prior]
		}
		gap := .5 + float64((frame*17+seed*13)%9-4)/100
		lane := "left"
		if gap >= .5 {
			lane = "right"
		}
		maneuver := "keep"
		if lane != previous {
			maneuver = "change_" + lane
		}
		lanes = append(lanes, lane)
		output := map[string]any{"lane": lane, "maneuver": maneuver, "used_previous_frame": prior}
		values := map[string]any{"gap": gap, "previous_lane": previous, "previous_frame": prior, "frame": frame}
		facts = append(facts, map[string]any{"inputs": values, "output": output})
		id := fmt.Sprintf("%032x", frame+1)
		tid := make([]byte, 16)
		tid[14] = byte((frame + 1) >> 8)
		tid[15] = byte(frame + 1)
		sid := []byte{0, 0, 0, 0, 0, 0, 0, 2}
		rootID := []byte{0, 0, 0, 0, 0, 0, 0, 1}
		at := uint64(1792454400000000000) + uint64(frame)*16666667
		root := &trace.Span{TraceId: tid, SpanId: rootID, Name: "frame_target.choose_lane", StartTimeUnixNano: at, EndTimeUnixNano: at + 1000, Status: &trace.Status{Code: trace.Status_STATUS_CODE_OK}}
		child := &trace.Span{TraceId: tid, SpanId: sid, ParentSpanId: rootID, Name: "frame_target.choose_lane", StartTimeUnixNano: at + 100, EndTimeUnixNano: at + 900, Status: &trace.Status{Code: trace.Status_STATUS_CODE_OK}}
		inputList := []any{}
		// Signature order matches Python; no map iteration defines evidence order.
		for _, name := range []string{"gap", "previous_lane", "previous_frame", "frame"} {
			inputList = append(inputList, map[string]any{"name": name, "ref": "value:" + id + "/0000000000000002.arg." + name + "@1", "capture": "allowed", "inline": values[name]})
		}
		outRef := "value:" + id + "/0000000000000002.return@1"
		recordID := "decision:" + id + "/0000000000000002"
		decision := map[string]any{"version": 1, "id": recordID, "run_id": id,
			"profile": map[string]any{"id": "synthetic", "version": "1"},
			"step":    map[string]any{"id": child.Name, "version": "1", "span_ref": "span:" + id + "/0000000000000002"},
			"output":  map[string]any{"ref": outRef, "capture": "allowed", "inline": output},
			"inputs":  inputList, "chosen": map[string]any{"option": "returned"}, "alternatives": []any{},
			"rule": map[string]any{"id": "unobserved"}, "config_refs": []any{}, "verdict": "returned", "reason_code": "", "complete": true}
		child.Events = append(child.Events, &trace.Span_Event{Name: "signalscout.decision", TimeUnixNano: at + 100, Attributes: []*common.KeyValue{
			textAttr("signalscout.record.kind", "decision"), textAttr("signalscout.record.id", recordID),
			textAttr("signalscout.record.json", encode(decision)), versionAttr(), syntheticAttr(),
			textAttr("signalscout.step.id", child.Name), textAttr("signalscout.output.ref", outRef),
			textAttr("signalscout.chosen.option", "returned"), textAttr("signalscout.verdict", "returned"), textAttr("signalscout.reason.code", ""),
		}})
		for i, input := range inputList {
			edge := map[string]any{"version": 1, "id": fmt.Sprintf("lineage:%s/0000000000000002-%d", id, i), "run_id": id,
				"input_ref": input.(map[string]any)["ref"], "output_ref": outRef, "step_id": child.Name,
				"decision_ref": recordID, "relation": "derived_from"}
			child.Events = append(child.Events, &trace.Span_Event{Name: "signalscout.lineage", TimeUnixNano: at + 100, Attributes: []*common.KeyValue{
				textAttr("signalscout.record.kind", "lineage"), textAttr("signalscout.record.json", encode(edge)), versionAttr(), syntheticAttr(),
				textAttr("signalscout.record.id", edge["id"].(string)),
			}})
		}
		scope.Spans = append(scope.Spans, root, child)
	}
	return req, facts
}
func run() error {
	out := flag.String("out", ".cache/provenance/frame-go", "Output directory")
	seed := flag.Int("seed", 1, "Synthetic deterministic seed")
	flag.Parse()
	req, facts := generate(*seed, 60)
	if err := os.MkdirAll(*out, 0755); err != nil {
		return err
	}
	data, err := protojson.Marshal(req)
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(*out, "traces.json"), data, 0644); err != nil {
		return err
	}
	data, err = json.MarshalIndent(map[string]any{"synthetic": true, "seed": *seed, "facts": facts}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(*out, "facts.json"), append(data, '\n'), 0644)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
