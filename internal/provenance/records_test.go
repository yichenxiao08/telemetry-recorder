package provenance

import (
	"github.com/signalscout/signalscout/internal/model"
	"testing"
)

func TestCaptureMissingUnknownVersionsAndArrays(t *testing.T) {
	p := model.AnalysisProfile{Step: "toy", EnvelopeProfile: "synthetic", EnvelopeVersion: "1"}
	s := model.Signal{ID: "event:toy", Kind: "event", Attrs: map[string]any{"signalscout.record.kind": "decision", "signalscout.record.json": `{"version":1,"profile":{"id":"synthetic","version":"1"},"step":{"id":"toy"},"inputs":[{"name":"a","capture":"off","inline":7},{"name":"b","capture":"allowed","inline":[3,4]}],"output":{"inline":{"option":"x"}},"verdict":"done"}`}}
	obs, ok := Decode(s, p)
	if !ok {
		t.Fatal("valid envelope rejected")
	}
	if _, ok := Get(obs.Values, "inputs.a"); ok {
		t.Fatal("capture off exposed")
	}
	if v, ok := Get(obs.Values, "inputs.b.1"); !ok || v != 4.0 {
		t.Fatal(v, ok)
	}
	p.EnvelopeVersion = "2"
	if _, ok := Decode(s, p); ok {
		t.Fatal("unknown version captured")
	}
}

func TestOutputDeclaredOffIsUnknown(t *testing.T) {
	p := model.AnalysisProfile{Step: "toy", EnvelopeProfile: "synthetic", EnvelopeVersion: "1"}
	s := model.Signal{Kind: "event", Attrs: map[string]any{"signalscout.record.kind": "decision", "signalscout.record.json": `{"version":1,"profile":{"id":"synthetic","version":"1"},"step":{"id":"toy"},"output":{"capture":"off","inline":{"option":"contradictory"}}}`}}
	observation, ok := Decode(s, p)
	if !ok {
		t.Fatal("metadata should remain inspectable")
	}
	if _, ok := Get(observation.Values, "output.option"); ok {
		t.Fatal("declared-off capture treated as observed")
	}
}
