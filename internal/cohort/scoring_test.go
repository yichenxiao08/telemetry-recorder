package cohort

import (
	"math"
	"testing"

	"pgregory.net/rapid"
)

func TestFrozenScores(t *testing.T) {
	for _, tc := range []struct {
		p, r []string
		want float64
	}{{[]string{"x", "x"}, []string{"x", "x"}, 0}, {[]string{"x", "x"}, []string{"y", "y"}, .5}, {[]string{"x", "y"}, []string{"y", "x"}, 0}} {
		got, _, _, _, _ := Categorical(tc.p, tc.r)
		if math.Abs(got-tc.want) > 1e-12 {
			t.Fatalf("score %g want %g", got, tc.want)
		}
	}
	p := []float64{1, 1, 1}
	r := []float64{1, 1, 1}
	if Numeric(p, r, 1, 1) != 0 {
		t.Fatal("identical numeric distributions")
	}
	if Numeric([]float64{3, 3, 3}, r, 1, 1) <= Numeric([]float64{2, 2, 2}, r, 1, 1) {
		t.Fatal("effect monotonicity")
	}
}

func TestNumericFormulaTable(t *testing.T) {
	zero := make([]float64, 10)
	for _, tc := range []struct {
		name                 string
		p, r                 []float64
		epsilon, floor, want float64
	}{
		{"increase", []float64{3, 3, 3}, zero, 1, 1, (1 - math.Exp(-1)) * 11 / 12},
		{"decrease", []float64{-3, -3, -3}, zero, 1, 1, (1 - math.Exp(-1)) * 11 / 12},
		{"effect floor", []float64{3}, zero, 1, 4, 0},
		{"MAD factor", []float64{3}, []float64{-2, -1, 0, 1, 2, -2, -1, 0, 1, 2}, .01, .1, (1 - math.Exp(-3/1.4826/3)) * 11 / 12},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Numeric(tc.p, tc.r, tc.epsilon, tc.floor); math.Abs(got-tc.want) > 1e-12 {
				t.Fatalf("got %g want %g", got, tc.want)
			}
		})
	}
}

func TestScoreProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(10, 100).Draw(t, "n")
		a := rapid.IntRange(0, n).Draw(t, "a")
		b := rapid.IntRange(0, n).Draw(t, "b")
		makeSample := func(k int) []string {
			x := make([]string, n)
			for i := range x {
				if i < k {
					x[i] = "yes"
				} else {
					x[i] = "no"
				}
			}
			return x
		}
		p, r := makeSample(a), makeSample(b)
		score, _, _, _, _ := Categorical(p, r)
		if score < 0 || score > 1 || math.IsNaN(score) {
			t.Fatal(score)
		}
		same, _, _, _, _ := Categorical(p, p)
		if same != 0 {
			t.Fatal("identical excess")
		}
		if a >= b && a < n {
			next, _, _, _, _ := Categorical(makeSample(a+1), r)
			if next+1e-12 < score {
				t.Fatal("nonmonotone categorical effect")
			}
		}
		effect := rapid.Float64Range(0, 100).Draw(t, "effect")
		reference := []float64{-1, 0, 1, 2, 3, 4, 5, 6, 7, 8}
		x := Numeric([]float64{10 + effect}, reference, .1, .01)
		y := Numeric([]float64{11 + effect}, reference, .1, .01)
		if x < 0 || y > 1 || x > y+1e-12 {
			t.Fatal(x, y)
		}
	})
}

func TestInvariantStates(t *testing.T) {
	for _, tc := range []struct {
		values map[string]any
		want   string
	}{{map[string]any{"used": true, "n": 3.0, "v": 2.0}, "satisfied"}, {map[string]any{"used": true, "n": 3.0, "v": 1.0}, "violated"}, {map[string]any{"used": true, "n": 3.0}, "unknown"}, {map[string]any{"used": false}, "not_applicable"}, {map[string]any{}, "unknown"}} {
		if got := VersionCheck(tc.values, "used", "n", "v", 1); got != tc.want {
			t.Fatalf("%v: %s", tc.values, got)
		}
	}
}
