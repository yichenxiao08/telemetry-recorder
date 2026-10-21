package cohort

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/signalscout/signalscout/internal/model"
	"pgregory.net/rapid"
)

func roleComparison(t interface {
	Helper()
	Fatal(...any)
}, name string, missing bool) Result {
	t.Helper()
	profile := toyProfile()
	profile.ReferenceTail = true
	profile.RoleAuthor = "development-test"
	profile.Numeric[0].Name = name
	profile.Numeric[0].Role = "candidate"
	profile.Numeric[0].Rationale = "captured input"
	profile.Numeric[1].Role = "symptom"
	profile.Numeric[1].Rationale = "selection output"
	cohorts := []model.Cohort{{Profile: profile}, {Profile: profile}}
	sets := [][]model.DecisionObservation{nil, nil}
	for arm := range 2 {
		for i := 0; i < 100; i++ {
			id := fmt.Sprintf("%d/%d", arm, i)
			values := map[string]any{"a": float64(arm * 2), "b": float64(arm * 2)}
			if missing && i >= 80 {
				delete(values, "a")
			}
			sets[arm] = append(sets[arm], model.DecisionObservation{RunID: id, Session: fmt.Sprint(i % 5), Values: map[string]any{"inputs": values}})
			cohorts[arm].RunIDs = append(cohorts[arm].RunIDs, id)
		}
	}
	result, err := Compare(cohorts[0], cohorts[1], sets[0], sets[1])
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func TestRolesAndMissingTail(t *testing.T) {
	result := roleComparison(t, "alpha", false)
	p := result.Presentation
	if len(result.Differences) != 2 || len(p.Problem) != 1 || p.Problem[0].Feature != "beta" || len(p.Median) != 1 || len(p.Tail) != 1 || p.Tail[0].Feature != "alpha" {
		t.Fatal("roles", p)
	}
	if !reflect.DeepEqual(result, roleComparison(t, "alpha", false)) {
		t.Fatal("determinism")
	}
	missing := roleComparison(t, "alpha", true).Presentation.Tail[0]
	if missing.ReferenceObserved != 80 || missing.ReferenceMissing != 20 || missing.Status != "insufficient_tail_evidence" {
		t.Fatal("missing counted as zero", missing)
	}
}
func TestRoleNamesNeverBoost(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := "x_" + rapid.StringMatching("[a-z]{1,12}").Draw(t, "name")
		a := roleComparison(t, "alpha", false)
		b := roleComparison(t, name, false)
		if a.Presentation.Tail[0].Score != b.Presentation.Tail[0].Score || a.Presentation.Median[0].Score != b.Presentation.Median[0].Score {
			t.Fatal("name boost")
		}
	})
}
func TestRolesRejectMissingAuthorAndUnknownRole(t *testing.T) {
	for _, role := range []string{"symptom", "decoy", "cause"} {
		p := toyProfile()
		p.Numeric[0].Role = role
		p.Numeric[0].Rationale = "test"
		if _, err := Compare(model.Cohort{Profile: p}, model.Cohort{Profile: p}, nil, nil); err == nil {
			t.Fatal("invalid role metadata accepted", role)
		}
	}
}
