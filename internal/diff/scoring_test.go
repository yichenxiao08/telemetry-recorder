package diff

import (
	"math"
	"testing"

	"github.com/signalscout/signalscout/internal/stats"
	"pgregory.net/rapid"
)

func TestNumericScoring(t *testing.T) {
	for _, tc := range []struct {
		name         string
		x            float64
		baseline     []float64
		floor, scale float64
		twoSided     bool
		want         float64
	}{
		{"unchanged", 10, []float64{10, 10, 10}, 5, 1, false, 0},
		{"below floor", 14.99, []float64{10, 10, 10}, 5, 1, false, 0},
		{"constant baseline", 16, []float64{10, 10, 10}, 5, 1, false, (1 - math.Exp(-2)) * .8},
		{"faster is not latency", 1, []float64{10, 10, 10}, 5, 1, false, 0},
		{"count decrease", 0, []float64{3, 3, 3}, 1, 1, true, (1 - math.Exp(-1)) * .8},
		{"count increase", 6, []float64{3, 3, 3}, 1, 1, true, (1 - math.Exp(-1)) * .8},
		{"empty", 10, nil, 5, 1, false, 0},
		{"nonfinite", math.NaN(), []float64{1}, 1, 1, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := numericScore(tc.x, tc.baseline, tc.floor, tc.scale, tc.twoSided)
			if math.Abs(got-tc.want) > 1e-12 {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestPresenceScoring(t *testing.T) {
	for _, tc := range []struct {
		present  bool
		count, n int
		want     float64
	}{
		{true, 0, 40, 41.0 / 42}, {true, 12, 40, 29.0 / 42}, {true, 40, 40, 0},
		{false, 32, 40, 33.0 / 42}, {false, 31, 40, 0}, {false, 0, 40, 0}, {true, 0, 0, 0},
	} {
		if got := presenceScore(tc.present, tc.count, tc.n); math.Abs(got-tc.want) > 1e-12 {
			t.Fatalf("%+v got %v", tc, got)
		}
	}
}

func TestScoreProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseline := rapid.SliceOfN(rapid.Float64Range(0, 10000), 10, 60).Draw(t, "baseline")
		x := rapid.Float64Range(0, 100000).Draw(t, "x")
		for _, twoSided := range []bool{false, true} {
			score := numericScore(x, baseline, 1, 1, twoSided)
			if math.IsNaN(score) || score < 0 || score > 1 {
				t.Fatalf("unbounded score %v", score)
			}
		}
		n := rapid.IntRange(10, 1000).Draw(t, "n")
		b := rapid.IntRange(0, n).Draw(t, "present")
		for _, present := range []bool{false, true} {
			s := presenceScore(present, b, n)
			if s < 0 || s > 1 || math.IsNaN(s) {
				t.Fatalf("presence %v", s)
			}
		}
	})
}

func TestScoresMonotoneInEffect(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseline := rapid.SliceOfN(rapid.Float64Range(0, 10000), 10, 50).Draw(t, "baseline")
		effect := rapid.Float64Range(0, 10000).Draw(t, "effect")
		extra := rapid.Float64Range(0, 10000).Draw(t, "extra")
		m := stats.Median(baseline)
		if numericScore(m+effect+extra, baseline, 1, 1, false)+1e-12 < numericScore(m+effect, baseline, 1, 1, false) {
			t.Fatal("larger increase scored lower")
		}
		if numericScore(m-effect-extra, baseline, 1, 1, true)+1e-12 < numericScore(m-effect, baseline, 1, 1, true) {
			t.Fatal("larger decrease scored lower")
		}
	})
}

func TestAddingCommonBaselineRunsNeverRaisesPresenceScore(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(10, 1000).Draw(t, "n")
		b := rapid.IntRange((n+1)/2, n).Draw(t, "common count")
		additional := rapid.IntRange(1, 1000).Draw(t, "identical present runs")
		if presenceScore(true, b+additional, n+additional) > presenceScore(true, b, n)+1e-12 {
			t.Fatal("common feature became more surprising")
		}
	})
}
