package report

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/cohort"
	"github.com/signalscout/signalscout/internal/lineage"
	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
)

func reportFixture(t *testing.T) (*store.Reader, Request) {
	t.Helper()
	ctx := context.Background()
	db := filepath.Join(t.TempDir(), "synthetic.db")
	s, err := store.Open(ctx, db, store.Options{QueueSize: 2, IdleTimeout: time.Second, SweepInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	one := 1.0
	p := model.AnalysisProfile{ID: "report-fixture", Unit: "decision", Workload: "request", Step: "adjust", EnvelopeProfile: "synthetic.report", EnvelopeVersion: "1", RoleAuthor: "synthetic", ReferenceTail: true, Numeric: []model.NumericFeature{{Name: "input", Selector: "inputs.x", Unit: "count", Epsilon: 1, Floor: 1, Role: "candidate", Rationale: "Synthetic recorded input"}}, Categories: []model.CategoryFeature{{Name: "flag", Selector: "inputs.flag", Role: "symptom", Rationale: "Synthetic output flag"}}, Invariant: model.VersionInvariant{Declared: "inputs.declared", Current: "inputs.current", Used: "inputs.used", Delta: 1}, Relations: &model.RelationPolicy{Version: 1, Rules: []model.RelationRule{{ID: "increment", Unit: "count", Tolerance: &one, Assert: model.Expression{Op: "gt", Args: []model.Expression{{Op: "field", Field: "output"}, {Op: "field", Field: "inputs.x"}}}}}}}
	for i := 0; i < 24; i++ {
		run := fmt.Sprintf("run-%02d", i)
		x := 0.0
		if i < 12 {
			x = 10
		}
		output := x + 3
		if i%6 == 0 {
			output = x
		}
		envelope := map[string]any{"version": 1, "id": "decision:" + run + "/adjust", "run_id": run, "profile": map[string]string{"id": "synthetic.report", "version": "1"}, "step": map[string]string{"id": "adjust"}, "inputs": []any{map[string]any{"name": "x", "capture": "allowed", "inline": x}, map[string]any{"name": "flag", "capture": "allowed", "inline": i%4 < 2}, map[string]any{"name": "group", "capture": "allowed", "inline": i < 12}}, "output": map[string]any{"name": "adjust", "ref": "value:" + run + "/adjust", "capture": "allowed", "inline": output}}
		attrs := map[string]any{"signalscout.record.kind": "decision", "signalscout.record.json": string(raw(envelope)), "signalscout.session.id": fmt.Sprintf("session-%02d", i)}
		root := model.Signal{ID: "span:" + run, Kind: "span", TraceID: run, Name: "request", TS: 100, End: 200, Attrs: map[string]any{"signalscout.session.id": fmt.Sprintf("session-%02d", i)}, Resource: model.Resource{ID: "synthetic", Attrs: map[string]any{"service.name": "synthetic"}}}
		event := root
		event.Kind = "event"
		event.ID = "event:" + run
		event.TS = 150
		event.Attrs = attrs
		if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{root, event}}); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range []struct{ id, value string }{{"problem", "true"}, {"reference", "false"}} {
		c, e := s.SelectCohort(ctx, item.id, p, model.CohortSelection{From: 0, To: 1000, Limit: 24, Predicate: "inputs.group", Equals: item.value})
		if e != nil {
			t.Fatal(e)
		}
		if e = s.SaveCohort(ctx, c); e != nil {
			t.Fatal(e)
		}
	}
	r, err := store.OpenReader(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r, Request{Version: 1, From: 0, To: 1000, Limit: 1000, Cohorts: []string{"problem", "reference"}, Comparisons: []Comparison{{Problem: "problem", Reference: "reference"}}, Traces: []Trace{{Target: "event:run-00", RunID: "run-00", Profile: &p, Budget: lineage.Options{Nodes: 16, Depth: 4}}}, Records: []Citation{{ID: "event:run-00", Kind: "native", RunID: "run-00"}}}
}

func TestBuildPreservesExistingOutputs(t *testing.T) {
	r, q := reportFixture(t)
	ctx := context.Background()
	doc, facts, err := Build(ctx, r, q)
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) == 0 {
		t.Fatal("no facts")
	}
	if v := Verify(doc.Claims, facts); v.Invalid != 0 || v.Unsupported != 0 || v.Valid != len(facts) {
		t.Fatal(v)
	}
	again, _, err := Build(ctx, r, q)
	if err != nil || !reflect.DeepEqual(doc, again) {
		t.Fatal("nondeterministic", err)
	}
	expected, err := cohort.Analyze(ctx, r, "problem", "reference")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	unknown := 0
	for _, s := range doc.Sources {
		if s.Kind == "explain_cohort" {
			found = true
			if string(s.Data) != string(raw(expected)) {
				t.Fatal("comparison output changed")
			}
		}
	}
	for _, c := range doc.Claims {
		if c.State == "unknown" {
			unknown++
		}
		if c.Association != nil && c.State == "insufficient_tail_evidence" && c.Tier != "observation" {
			t.Fatal("insufficient upgraded")
		}
	}
	if !found || unknown == 0 {
		t.Fatal("comparison or unknown invariants missing")
	}
	v, err := VerifyDocument(ctx, r, doc)
	if err != nil || v.Invalid != 0 {
		t.Fatal(v, err)
	}
	doc.Claims[0].Evidence[0].CohortID = "reference"
	v, err = VerifyDocument(ctx, r, doc)
	if err != nil || v.Invalid == 0 {
		t.Fatal("wrong scope supported", v, err)
	}
	doc.Sources[0].Data = json.RawMessage(`{}`)
	if _, err = VerifyDocument(ctx, r, doc); err == nil {
		t.Fatal("tampered source accepted")
	}
}

func TestBuildScopeAndCaps(t *testing.T) {
	r, q := reportFixture(t)
	for _, change := range []func(*Request){func(q *Request) { q.From = 151 }, func(q *Request) { q.Limit = 1 }, func(q *Request) { q.To = 0 }, func(q *Request) { q.Limit = 10001 }} {
		c := q
		change(&c)
		if _, _, err := Build(context.Background(), r, c); err == nil {
			t.Fatal("scope/cap ignored", c)
		}
	}
}

func TestMissingTraceRemainsUnsupportedUnknown(t *testing.T) {
	r, q := reportFixture(t)
	q.Comparisons = nil
	q.Cohorts = nil
	q.Records = nil
	q.Traces[0].Target = "event:missing"
	doc, _, err := Build(context.Background(), r, q)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Claims) != 1 || doc.Claims[0].State != "unknown" {
		t.Fatal(doc.Claims)
	}
	v, err := VerifyDocument(context.Background(), r, doc)
	if err != nil || v.Unsupported != 1 || v.Valid != 0 {
		t.Fatal(v, err)
	}
}

func TestNativeLogAndMetricCitations(t *testing.T) {
	ctx := context.Background()
	db := filepath.Join(t.TempDir(), "signals.db")
	s, err := store.Open(ctx, db, store.Options{QueueSize: 2, IdleTimeout: time.Second, SweepInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	val := 3.5
	for _, kind := range []string{"log", "metric"} {
		signal := model.Signal{ID: kind + ":synthetic", Kind: kind, TS: 100, Name: "synthetic", Body: "synthetic text", Value: &val, Resource: model.Resource{ID: "synthetic"}}
		if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{signal}}); err != nil {
			t.Fatal(err)
		}
	}
	r, err := store.OpenReader(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	q := Request{Version: 1, From: 0, To: 1000, Limit: 2, Records: []Citation{{ID: "log:synthetic", Kind: "native"}, {ID: "metric:synthetic", Kind: "native"}}}
	doc, _, err := Build(ctx, r, q)
	if err != nil {
		t.Fatal(err)
	}
	v, err := VerifyDocument(ctx, r, doc)
	if err != nil || v.Valid != 2 {
		t.Fatal(v, err)
	}
	doc.Claims[0].Numbers["/value"] = 3.6
	v, err = VerifyDocument(ctx, r, doc)
	if err != nil || v.Invalid != 1 {
		t.Fatal(v, err)
	}
}

func TestExplicitAssociationFieldsAndHypotheses(t *testing.T) {
	var a Association
	if Decode([]byte(`{"problem_numerator":0}`), &a) == nil {
		t.Fatal("missing zero denominator accepted")
	}
	f := testFact()
	c := claimFor(f)
	c.ID = "supplied"
	c.Tier = "hypothesis"
	c.State = "proposed"
	c.Numbers = nil
	c.Hypothesis = &Hypothesis{Alternatives: []string{"A different threshold"}, MissingEvidence: []string{"A controlled repeat"}, Experiment: "Repeat with a fixed input", Predicted: "The recorded result repeats", Falsifying: "The recorded result differs"}
	v := Verify([]Claim{c}, []Fact{f})
	if v.Unsupported != 1 || v.Valid != 0 || v.ValidCitations != 1 {
		t.Fatal(v)
	}
	c.Hypothesis.Falsifying = ""
	if Verify([]Claim{c}, []Fact{f}).Invalid != 1 {
		t.Fatal("incomplete hypothesis accepted")
	}
}
