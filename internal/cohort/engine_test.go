package cohort

import (
	"reflect"
	"testing"

	"github.com/signalscout/signalscout/internal/model"
)

func toyProfile() model.AnalysisProfile {
	return model.AnalysisProfile{ID: "toy", Unit: "item", Workload: "toy", Step: "toy", EnvelopeProfile: "synthetic", EnvelopeVersion: "1", Numeric: []model.NumericFeature{{Name: "alpha", Selector: "inputs.a", Unit: "u", Epsilon: 1, Floor: 1}, {Name: "beta", Selector: "inputs.b", Unit: "u", Epsilon: 1, Floor: 1}}, Invariant: model.VersionInvariant{Declared: "used", Current: "n", Used: "v", Delta: 1}}
}
func TestStableTiesNoNameBoostAndMissing(t *testing.T) {
	p, r := model.Cohort{Profile: toyProfile(), RunIDs: []string{"p"}}, model.Cohort{Profile: toyProfile(), RunIDs: []string{"r"}}
	makeValues := func(v float64) []model.DecisionObservation {
		out := []model.DecisionObservation{}
		for i := 0; i < 10; i++ {
			out = append(out, model.DecisionObservation{Values: map[string]any{"inputs": map[string]any{"a": v, "b": v}}})
		}
		return out
	}
	a, err := Compare(p, r, makeValues(3), makeValues(1))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Compare(p, r, makeValues(3), makeValues(1))
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("unstable", err)
	}
	if a.Differences[0].Feature != "alpha" || a.Differences[0].Score != a.Differences[1].Score {
		t.Fatal("tie or name boost")
	}
	p.Profile.Numeric[0].Name = "zzz"
	r.Profile = p.Profile
	c, err := Compare(p, r, makeValues(3), makeValues(1))
	if err != nil || c.Differences[0].Score != a.Differences[0].Score || c.Differences[0].Feature != "beta" {
		t.Fatal("names changed score", err)
	}
	missing := makeValues(3)
	delete(missing[0].Values["inputs"].(map[string]any), "a")
	c, err = Compare(p, r, missing, makeValues(1))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range c.Differences {
		if f.Feature == "zzz" && (f.Status != "insufficient_evidence" || f.ProblemMissing != 1 || f.Score != 0) {
			t.Fatal("missing imputed", f)
		}
	}
	r.RunIDs = []string{"p"}
	if _, err := Compare(p, r, makeValues(3), makeValues(1)); err == nil {
		t.Fatal("overlap accepted")
	}
}

func TestOverflowIsUnsupportedObservation(t *testing.T) {
	p, r := model.Cohort{Profile: toyProfile(), RunIDs: []string{"p"}}, model.Cohort{Profile: toyProfile(), RunIDs: []string{"r"}}
	problem, reference := []model.DecisionObservation{}, []model.DecisionObservation{}
	for i := 0; i < 10; i++ {
		value := 1e308
		if i < 2 {
			value = -1e308
		}
		problem = append(problem, model.DecisionObservation{Values: map[string]any{"inputs": map[string]any{"a": 0.0, "b": 0.0}}})
		reference = append(reference, model.DecisionObservation{Values: map[string]any{"inputs": map[string]any{"a": value, "b": value}}})
	}
	result, err := Compare(p, r, problem, reference)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range result.Differences {
		if f.Status != "unsupported_numeric" || f.Tier != "observation" || f.Score != 0 || f.ProblemMissing != 0 {
			t.Fatal("overflow disguised as no effect or missing", f)
		}
	}
}
