package stats

import (
	"math"
	"reflect"
	"testing"
)

func TestStatistics(t *testing.T) {
	for _, tc := range []struct {
		name             string
		values           []float64
		median, mad, p99 float64
	}{
		{"singleton", []float64{4}, 4, 0, 4},
		{"odd", []float64{100, 1, 3}, 3, 2, 98.06},
		{"even", []float64{8, 2, 4, 6}, 5, 2, 7.94},
		{"constant", []float64{2, 2, 2}, 2, 0, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := append([]float64(nil), tc.values...)
			if Median(tc.values) != tc.median || MAD(tc.values) != tc.mad || math.Abs(Quantile(tc.values, .99)-tc.p99) > 1e-10 {
				t.Fatalf("median=%v MAD=%v p99=%v", Median(tc.values), MAD(tc.values), Quantile(tc.values, .99))
			}
			if !reflect.DeepEqual(before, tc.values) {
				t.Fatal("input mutated")
			}
		})
	}
	if !math.IsNaN(Median(nil)) || !math.IsNaN(MAD(nil)) || !math.IsNaN(Quantile([]float64{1}, -1)) {
		t.Fatal("invalid sample not rejected")
	}
}
