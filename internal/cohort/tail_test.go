package cohort

import (
	"encoding/json"
	"fmt"
	"github.com/signalscout/signalscout/internal/model"
	"math"
	"reflect"
	"testing"

	"pgregory.net/rapid"
)

func repeat(v float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = v
	}
	return out
}
func TestTailTerms(t *testing.T) {
	for _, tc := range []struct {
		name   string
		p, r   []float64
		ps, rs int
		status string
		count  int
		score  float64
	}{
		{"age", append(repeat(1, 320), repeat(2, 80)...), repeat(1, 400), 20, 20, "ok", 80, 80.0 / 402},
		{"edges inside equal sizes", append(repeat(.5, 50), repeat(1.5, 50)...), repeat(1, 100), 5, 5, "no_difference", 0, 0},
		{"edges inside unequal sizes smoothing", append(repeat(.5, 15), repeat(1.5, 15)...), repeat(1, 100), 5, 5, "ok", 0, 1.0/32 - 1.0/102},
		{"reference too small", repeat(2, 30), repeat(1, 99), 5, 5, "insufficient_tail_evidence", 0, 0},
		{"problem too small", repeat(2, 29), repeat(1, 100), 5, 5, "insufficient_tail_evidence", 0, 0},
		{"unknown clusters", repeat(2, 30), repeat(1, 100), 4, 5, "insufficient_tail_evidence", 0, 0},
		{"empty", nil, nil, 0, 0, "insufficient_tail_evidence", 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ReferenceTail(tc.p, tc.r, 1, 1, tc.ps, tc.rs)
			if got.Status != tc.status || got.ProblemOutside != tc.count || math.Abs(got.Score-tc.score) > 1e-12 {
				t.Fatalf("%+v", got)
			}
			if !reflect.DeepEqual(got, ReferenceTail(tc.p, tc.r, 1, 1, tc.ps, tc.rs)) {
				t.Fatal("non deterministic")
			}
		})
	}
}
func TestTailBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		p := rapid.SliceOfN(rapid.Float64Range(-1e6, 1e6), 30, 200).Draw(t, "p")
		r := rapid.SliceOfN(rapid.Float64Range(-1e6, 1e6), 100, 200).Draw(t, "r")
		v := ReferenceTail(p, r, 1, .1, 5, 5)
		if math.IsNaN(v.Score) || v.Score < 0 || v.Score > 1 {
			t.Fatalf("score %g", v.Score)
		}
		if v.ProblemLeft+v.ProblemRight != v.ProblemOutside || v.ReferenceLeft+v.ReferenceRight != v.ReferenceOutside {
			t.Fatal("counts")
		}
		if !reflect.DeepEqual(v, ReferenceTail(p, r, 1, .1, 5, 5)) {
			t.Fatal("non deterministic")
		}
	})
}

func TestTailEdgeAndMinimumProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		middle := float64(rapid.IntRange(-1000, 1000).Draw(t, "middle"))
		edges := append(repeat(middle-.5, 50), repeat(middle+.5, 50)...)
		v := ReferenceTail(edges, repeat(middle, 100), 1, 1, 5, 5)
		if v.ProblemOutside != 0 || v.ReferenceOutside != 0 || v.Score != 0 {
			t.Fatal("equality outside interval", v)
		}
		small := rapid.IntRange(0, 29).Draw(t, "small_problem")
		if ReferenceTail(repeat(middle, small), repeat(middle, 100), 1, 1, 5, 5).Status != "insufficient_tail_evidence" {
			t.Fatal("minimum problem")
		}
		clusters := rapid.IntRange(0, 4).Draw(t, "clusters")
		if ReferenceTail(edges, repeat(middle, 100), 1, 1, 5, clusters).Status != "insufficient_tail_evidence" {
			t.Fatal("minimum reference clusters")
		}
	})
}

func TestUnknownTailScoreIsNull(t *testing.T) {
	terms := ReferenceTail(nil, nil, 1, 1, 0, 0)
	finding := TailFinding{Finding: Finding{Status: terms.Status}, Terms: terms}
	data, err := json.Marshal(finding)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err = json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["score"] != nil || decoded["terms"].(map[string]any)["score"] != nil {
		t.Fatal("unknown disguised as zero", string(data))
	}
}

func TestObservedClusterEligibility(t *testing.T) {
	profile := toyProfile()
	values := []model.DecisionObservation{}
	for i := 0; i < 100; i++ {
		session := ""
		if i < 80 {
			session = "known"
		}
		values = append(values, model.DecisionObservation{Session: session, Values: map[string]any{"inputs": map[string]any{"a": 1.0}}})
	}
	numbers, sessions, unknown := numericObservations(values, profile.Numeric[0])
	if len(numbers) != 100 || len(sessions) != 1 || unknown != 20 {
		t.Fatal("observed cluster counts")
	}
	if ReferenceTail(numbers, numbers, 1, 1, len(sessions), len(sessions)).Status != "insufficient_tail_evidence" {
		t.Fatal("unknown sessions counted as clusters")
	}
	delete(values[0].Values["inputs"].(map[string]any), "a")
	numbers, sessions, unknown = numericObservations(values, profile.Numeric[0])
	if len(numbers) != 99 || len(sessions) != 1 || unknown != 20 {
		t.Fatal("missing counted as zero")
	}
}

func TestTailEvidencePrefersOutsideWitness(t *testing.T) {
	profile := toyProfile()
	xs := []model.DecisionObservation{}
	for i := 0; i < 5; i++ {
		value := 1.0
		if i == 4 {
			value = 2
		}
		xs = append(xs, model.DecisionObservation{RunID: fmt.Sprint(i), EvidenceID: "event:" + fmt.Sprint(i), Values: map[string]any{"inputs": map[string]any{"a": value}}})
	}
	f := Finding{}
	tailEvidence(&f, xs, profile.Numeric[0], TailTerms{Status: "ok", Lower: .5, Upper: 1.5})
	if len(f.EvidenceIDs) != 3 || f.EvidenceIDs[0] != "event:4" {
		t.Fatal("outlier witness absent", f)
	}
}
