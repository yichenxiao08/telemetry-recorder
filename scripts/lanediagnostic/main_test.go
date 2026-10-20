package main

import (
	"math"
	"testing"
)

func TestFrozenFormulaTerms(t *testing.T) {
	for _, tc := range []struct {
		name  string
		p, r  []float64
		floor float64
		z     float64
		e     int
		zero  bool
	}{
		{"full shift", []float64{2, 2, 2, 2, 2}, []float64{1, 1, 1, 1, 1}, 1, 1, 0, false},
		{"minority hidden", []float64{1, 1, 1, 1, 2}, []float64{1, 1, 1, 1, 1}, 1, 0, 5, true},
		{"floor", []float64{1.05}, []float64{1}, .1, .05, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := numeric(tc.p, tc.r, 1, tc.floor)
			if math.Abs(x.Z-tc.z) > 1e-12 || x.E != tc.e || x.BelowFloor != tc.zero {
				t.Fatalf("terms: %+v", x)
			}
			want := x.Squash * x.ExceedanceFactor
			if tc.zero {
				want = 0
			}
			if x.Score != want {
				t.Fatalf("score %g != %g", x.Score, want)
			}
		})
	}
}
