package lineage

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/signalscout/signalscout/internal/model"
	"pgregory.net/rapid"
)

func decision(id, output string, inputs ...Field) model.Signal {
	e := Envelope{Version: 1, ID: id, RunID: "r", Step: Step{ID: "choose"}, Output: Field{Ref: output, Capture: "allowed", Inline: json.RawMessage(`"left"`)}, Inputs: inputs}
	b, _ := json.Marshal(e)
	return model.Signal{ID: "event:" + id, Kind: "event", RunID: "r", TS: 10, Attrs: map[string]any{"signalscout.record.kind": "decision", "signalscout.record.id": id, "signalscout.record.json": string(b)}}
}
func edge(id, input, output, dec string) model.Signal {
	e := Edge{Version: 1, ID: id, RunID: "r", InputRef: input, OutputRef: output, DecisionRef: dec, StepID: "choose", Relation: "derived_from"}
	b, _ := json.Marshal(e)
	return model.Signal{ID: "event:" + id, Kind: "event", RunID: "r", TS: 10, Attrs: map[string]any{"signalscout.record.kind": "lineage", "signalscout.record.id": id, "signalscout.record.json": string(b)}}
}
func field(ref string) Field {
	return Field{Name: ref, Ref: ref, Capture: "allowed", Inline: json.RawMessage(`1`)}
}
func TestWalkCases(t *testing.T) {
	a, b := decision("a", "a"), decision("b", "b", field("a"))
	chain := []model.Signal{a, b, edge("ab", "a", "b", "b")}
	cases := []struct {
		name               string
		signals            []model.Signal
		root               string
		depth, nodes, want int
		finding            string
	}{
		{"chain", chain, "b", 8, 8, 2, ""},
		{"branch", append(append([]model.Signal{}, chain...), decision("c", "c", field("a"), field("b")), edge("ac", "a", "c", "c"), edge("bc", "b", "c", "c")), "c", 8, 8, 3, ""},
		{"missing", chain, "absent", 8, 8, 0, "missing"},
		{"dangling", []model.Signal{b, edge("ab", "a", "b", "b")}, "b", 8, 8, 1, "dangling"},
		{"cycle", []model.Signal{decision("a", "a", field("b")), b, edge("ab", "a", "b", "b"), edge("ba", "b", "a", "a")}, "b", 8, 8, 2, "cycle"},
		{"conflict", append(append([]model.Signal{}, chain...), decision("a", "other")), "b", 8, 8, 1, "conflicting"},
		{"duplicate", append(append([]model.Signal{}, chain...), a), "a", 8, 8, 0, "duplicate"},
		{"depth", chain, "b", 0, 8, 1, "depth"},
		{"budget", chain, "b", 8, 1, 1, "node budget"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := Walk(c.signals, c.root, Options{Depth: c.depth, Nodes: c.nodes}, nil)
			if len(r.Nodes) != c.want {
				t.Fatalf("nodes %d want %d: %+v", len(r.Nodes), c.want, r.Findings)
			}
			raw, _ := json.Marshal(r.Findings)
			if c.finding != "" && !strings.Contains(string(raw), c.finding) {
				t.Fatalf("missing finding %s: %s", c.finding, raw)
			}
		})
	}
}
func TestCaptureAndInvariant(t *testing.T) {
	p := &model.AnalysisProfile{Step: "choose", Invariant: model.VersionInvariant{Declared: "inputs.declared", Current: "inputs.frame", Used: "inputs.previous", Delta: 1}}
	s := decision("stale", "out", Field{Name: "declared", Capture: "allowed", Inline: json.RawMessage(`true`)}, Field{Name: "frame", Capture: "allowed", Inline: json.RawMessage(`10`)}, Field{Name: "previous", Capture: "allowed", Inline: json.RawMessage(`8`)}, Field{Name: "secret", Capture: "off", Inline: json.RawMessage(`"must not leak"`)}, Field{Name: "digest", Capture: "reference-only", Digest: "sha256:abc", Inline: json.RawMessage(`99`)}, Field{Name: "missing"})
	r := Walk([]model.Signal{s}, "out", Options{Depth: 8, Nodes: 8}, p)
	n := r.Nodes[0]
	if n.Invariant.Status != "violated" || n.Invariant.Observed != 3 {
		t.Fatalf("%+v", n.Invariant)
	}
	if n.Inputs[3].Status != "capture-off" || n.Inputs[4].Status != "hash-only" || n.Inputs[5].Status != "missing" {
		t.Fatal(n.Inputs)
	}
	raw, _ := json.Marshal(r)
	if strings.Contains(string(raw), "must not leak") || len(n.Inputs[4].Inline) > 0 {
		t.Fatal("private value leaked")
	}
}
func TestMalformedAndDuplicateKeys(t *testing.T) {
	for _, body := range []string{`{`, `{"version":1,"version":2}`, strings.Repeat("x", 8193)} {
		s := decision("x", "x")
		s.Attrs["signalscout.record.json"] = body
		r := Walk([]model.Signal{s}, "x", Options{Depth: 2, Nodes: 2}, nil)
		if len(r.Nodes) != 0 || len(r.Findings) < 2 {
			t.Fatalf("malformed accepted: %+v", r)
		}
	}
}

func TestInvariantUnknownAndEdges(t *testing.T) {
	p := &model.AnalysisProfile{Step: "choose", Invariant: model.VersionInvariant{Declared: "inputs.declared", Current: "inputs.frame", Used: "inputs.previous", Delta: 1}}
	for _, c := range []struct{ name, declared, current, used, capture, want string }{
		{"fresh", "true", "10", "9", "allowed", "satisfied"},
		{"not declared", "false", "10", "8", "allowed", "not_applicable"},
		{"off", "true", "10", "8", "off", "unknown"},
		{"wrong type", "true", "10", "\"8\"", "allowed", "unknown"},
		{"unsafe integer", "true", "9007199254740992", "9007199254740991", "allowed", "unknown"},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := decision("x", "x", Field{Name: "declared", Capture: "allowed", Inline: json.RawMessage(c.declared)}, Field{Name: "frame", Capture: "allowed", Inline: json.RawMessage(c.current)}, Field{Name: "previous", Capture: c.capture, Inline: json.RawMessage(c.used)})
			r := Walk([]model.Signal{s}, "x", Options{Depth: 4, Nodes: 4}, p)
			if r.Nodes[0].Invariant.Status != c.want {
				t.Fatal(r.Nodes[0].Invariant)
			}
		})
	}
	s := decision("a", "a")
	other := decision("b", "a")
	r := Walk([]model.Signal{s, other}, "a", Options{Depth: 4, Nodes: 4}, nil)
	if len(r.Nodes) != 0 {
		t.Fatal("conflicting output was trusted")
	}
	badEdge := edge("bad", "not-an-input", "a", "a")
	r = Walk([]model.Signal{s, badEdge}, "a", Options{Depth: 4, Nodes: 4}, nil)
	if len(r.Findings) == 0 || len(r.Nodes[0].Edges) != 0 {
		t.Fatal("mismatched edge accepted")
	}
	delete(s.Attrs, "signalscout.record.id")
	delete(other.Attrs, "signalscout.record.id")
	r = Walk([]model.Signal{s, other}, "a", Options{Depth: 4, Nodes: 4}, nil)
	if len(r.Nodes) != 0 {
		t.Fatal("conflicting output accepted via logical ID without selector attributes")
	}
	orphanEdge := edge("orphan", "x", "a", "absent")
	r = Walk([]model.Signal{s, orphanEdge}, "a", Options{Depth: 4, Nodes: 4}, nil)
	if len(r.Findings) == 0 {
		t.Fatal("dangling decision_ref silently ignored")
	}
}
func TestTraversalProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(1, 30).Draw(t, "count")
		budget := rapid.IntRange(1, 20).Draw(t, "budget")
		depth := rapid.IntRange(0, 12).Draw(t, "depth")
		var signals []model.Signal
		stored := map[string]bool{}
		for i := 0; i < count; i++ {
			id := string(rune('A' + i))
			ref := string(rune('A' + rapid.IntRange(0, count-1).Draw(t, "parent")))
			f := field(ref)
			switch rapid.IntRange(0, 3).Draw(t, "capture") {
			case 1:
				f.Capture = "off"
			case 2:
				f.Capture = "hash-only"
				f.Digest = "sha256:synthetic"
			case 3:
				f.Capture = ""
			}
			s := decision(id, id, f)
			signals = append(signals, s, edge("edge"+id, ref, id, id))
			stored[s.ID] = true
		}
		r := Walk(signals, "A", Options{Depth: depth, Nodes: budget}, nil)
		if len(r.Nodes) > budget || r.Visited != len(r.Nodes) {
			t.Fatal("budget exceeded")
		}
		for _, n := range r.Nodes {
			if !stored[n.Evidence.ID] {
				t.Fatal("invented node")
			}
			for _, f := range n.Inputs {
				if f.Status != "captured" && len(f.Inline) > 0 {
					t.Fatal("unknown became value")
				}
			}
		}
	})
}
