package cohort

import (
	"testing"

	"github.com/signalscout/signalscout/internal/model"
)

func TestIntegerBooleanCategoriesAndEscapedKeys(t *testing.T) {
	for _, pair := range [][2]any{{1.0, 2.0}, {true, false}, {"1", 1.0}, {"true", true}} {
		p := model.Cohort{Profile: toyProfile(), RunIDs: []string{"p"}}
		p.Profile.Numeric = nil
		p.Profile.Categories = []model.CategoryFeature{{Name: "category", Selector: `inputs.self\.score`}}
		p.Profile.SelectorSyntax = "escaped-v1"
		r := model.Cohort{Profile: p.Profile, RunIDs: []string{"r"}}
		makeSet := func(v any) []model.DecisionObservation {
			xs := []model.DecisionObservation{}
			for i := 0; i < 10; i++ {
				xs = append(xs, model.DecisionObservation{Values: map[string]any{"inputs": map[string]any{"self.score": v}}})
			}
			return xs
		}
		out, err := Compare(p, r, makeSet(pair[0]), makeSet(pair[1]))
		if err != nil {
			t.Fatal(err)
		}
		f := out.Differences[0]
		if f.Score <= 0 || f.ProblemObserved != 10 || f.ReferenceObserved != 10 || f.ProblemNumerator != 10 || f.ReferenceNumerator != 0 {
			t.Fatalf("%v: %+v", pair, f)
		}
	}
}
