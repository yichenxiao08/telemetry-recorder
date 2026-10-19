package main

import "testing"

func TestReportedMinimumMedianAndTarget(t *testing.T) {
	for _, tc := range []struct {
		values      []float64
		min, median float64
		met         bool
	}{
		{[]float64{12000, 4000, 10000}, 4000, 10000, false},
		{[]float64{10000, 12000, 11000, 13000}, 10000, 11500, true},
	} {
		got, err := summarize(tc.values, 10000)
		if err != nil {
			t.Fatal(err)
		}
		if got.Minimum != tc.min || got.Median != tc.median || got.Met != tc.met {
			t.Fatalf("misreported target: %+v", got)
		}
	}
	if _, err := summarize(nil, 10000); err == nil {
		t.Fatal("missing data accepted")
	}
}
