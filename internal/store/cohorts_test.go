package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/provenance"
)

func TestCohortPinsRevisionsUnknownDisagreementAndPredicate(t *testing.T) {
	ctx := context.Background()
	s := testStore(t, 0)
	p := model.AnalysisProfile{ID: "toy", Unit: "item", Workload: "request", Step: "toy", EnvelopeProfile: "synthetic", EnvelopeVersion: "1", Numeric: []model.NumericFeature{{Name: "value", Selector: "inputs.x", Unit: "u", Epsilon: 1, Floor: 1}}, Invariant: model.VersionInvariant{Declared: "used", Current: "n", Used: "v", Delta: 1}}
	for i, id := range []string{"a", "b", "c"} {
		root := fixture("span:"+id, "span", id, "resource", 100, 200)
		root.Attrs["signalscout.session.id"] = "toy-session"
		event := fixture("event:"+id, "event", id, "resource", 150, 150)
		body := map[string]any{"version": 1, "profile": map[string]any{"id": "synthetic", "version": "1"}, "step": map[string]any{"id": "toy"}, "inputs": []any{map[string]any{"name": "x", "capture": "allowed", "inline": float64(i + 3)}}, "chosen": map[string]any{"option": "done"}}
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		event.Attrs = map[string]any{"signalscout.record.kind": "decision", "signalscout.record.json": string(data)}
		if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{root, event}}); err != nil {
			t.Fatal(err)
		}
	}
	mark := func(id, run, reviewer, value string) {
		a := model.Annotation{ID: id, RunID: run, Dimension: "d", Value: value, Source: "toy", Reviewer: reviewer, Synthetic: true, RubricID: "toy", RubricVersion: "1", CreatedAt: 1}
		if err := s.SaveAnnotation(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	mark("a1", "a", "one", "bad")
	mark("b1", "b", "one", "bad")
	mark("b2", "b", "two", "good")
	bound := model.CohortSelection{From: 0, To: 1000, Limit: 3, Dimension: "d", Value: "bad"}
	c, err := s.SelectCohort(ctx, "selected", p, bound)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.RunIDs) != 1 || c.RunIDs[0] != "a" || len(c.LabelIDs) != 1 || c.LabelIDs[0] != "a1" || c.Excluded["unlabeled_unknown"] != 1 || c.Excluded["reviewer_disagreement"] != 1 {
		t.Fatalf("selection: %+v", c)
	}
	if err = s.SaveCohort(ctx, c); err != nil {
		t.Fatal(err)
	}
	if err = s.SaveCohort(ctx, c); err == nil {
		t.Fatal("cohort overwritten")
	}
	bound.Dimension = ""
	bound.Predicate = "inputs.x"
	bound.Equals = "5"
	reference, err := s.SelectCohort(ctx, "reference", p, bound)
	if err != nil {
		t.Fatal(err)
	}
	if len(reference.RunIDs) != 1 || reference.RunIDs[0] != "c" || len(reference.LabelIDs) != 0 {
		t.Fatal(reference)
	}
	if err = s.SaveCohort(ctx, reference); err != nil {
		t.Fatal(err)
	}
	mark("a2", "a", "two", "good")
	pair, sets, err := s.CohortPair(ctx, "selected", "reference")
	if err != nil {
		t.Fatal(err)
	}
	if len(pair[0].LabelIDs) != 1 || pair[0].LabelIDs[0] != "a1" {
		t.Fatal("selection revised by later label")
	}
	if v, ok := provenance.Get(sets[0][0].Values, "inputs.x"); !ok || v != 3.0 {
		t.Fatal("value lost", v)
	}
	// New decision evidence does not replace a cohort's native decision revision.
	later := fixture("event:later", "event", "a", "resource", 300, 300)
	later.Attrs = map[string]any{"signalscout.record.kind": "decision", "signalscout.record.json": `{"version":1,"profile":{"id":"synthetic","version":"1"},"step":{"id":"toy"},"inputs":[{"name":"x","capture":"allowed","inline":99}]}`}
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{later}}); err != nil {
		t.Fatal(err)
	}
	_, sets, err = s.CohortPair(ctx, "selected", "reference")
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := provenance.Get(sets[0][0].Values, "inputs.x"); v != 3.0 {
		t.Fatal("pinned decision changed")
	}
	bound.Limit = 1
	if _, err = s.SelectCohort(ctx, "too-small", p, bound); err == nil {
		t.Fatal("silently truncated population")
	}
	bound.From = bound.To
	if _, err = s.SelectCohort(ctx, "bad-time", p, bound); err == nil {
		t.Fatal("missing bounds accepted")
	}
	select {
	case err := <-s.Errors():
		t.Fatal("validation stopped serving", err)
	default:
	}
	if err = s.Maintain(ctx, time.Now().Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
}
