package diff

import (
	"math"

	"github.com/signalscout/signalscout/internal/stats"
)

// Constants and their reasons are fixed in docs/diff-scoring.md.
const (
	madConsistency   = 1.4826
	effectScale      = 3.0
	latencyFloorMS   = 5.0
	latencyScaleMS   = 1.0
	countScale       = 1.0
	expectedPresence = .8
)

func presenceScore(present bool, count, n int) float64 {
	if n <= 0 || count < 0 || count > n {
		return 0
	}
	p := float64(count+1) / float64(n+2)
	if present {
		if count == n {
			return 0
		}
		return 1 - p
	}
	if float64(count)/float64(n) < expectedPresence {
		return 0
	}
	return p
}

func directionalExceedances(x, median float64, baseline []float64, twoSided bool) int {
	count := 0
	for _, value := range baseline {
		if twoSided && x < median {
			if value <= x {
				count++
			}
		} else if value >= x {
			count++
		}
	}
	return count
}

func numericScore(x float64, baseline []float64, floor, minimumScale float64, twoSided bool) float64 {
	if len(baseline) == 0 || math.IsNaN(x) || math.IsInf(x, 0) || minimumScale <= 0 {
		return 0
	}
	middle := stats.Median(baseline)
	if math.IsNaN(middle) {
		return 0
	}
	effect := x - middle
	if twoSided {
		effect = math.Abs(effect)
	}
	if effect < floor || effect <= 0 {
		return 0
	}
	scale := math.Max(madConsistency*stats.MAD(baseline), minimumScale)
	z := effect / scale
	exceed := directionalExceedances(x, middle, baseline, twoSided)
	rarity := 1 - float64(exceed+1)/float64(len(baseline)+2)
	return math.Max(0, math.Min(1, -math.Expm1(-z/effectScale)*rarity))
}
