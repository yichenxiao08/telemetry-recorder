package stats

import (
	"math"
	"sort"
)

// Quantile interpolates at q*(n-1) without changing the input. Empty or invalid
// samples return NaN so callers cannot mistake missing data for a measured zero.
func Quantile(values []float64, q float64) float64 {
	if len(values) == 0 || math.IsNaN(q) || q < 0 || q > 1 {
		return math.NaN()
	}
	ordered := append([]float64(nil), values...)
	for _, value := range ordered {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return math.NaN()
		}
	}
	sort.Float64s(ordered)
	position := q * float64(len(ordered)-1)
	lower := int(position)
	if lower == len(ordered)-1 {
		return ordered[lower]
	}
	return ordered[lower] + (position-float64(lower))*(ordered[lower+1]-ordered[lower])
}

func Median(values []float64) float64 { return Quantile(values, .5) }

// MAD is the median absolute deviation from the sample median.
func MAD(values []float64) float64 {
	middle := Median(values)
	deviations := make([]float64, len(values))
	for index, value := range values {
		deviations[index] = math.Abs(value - middle)
	}
	return Median(deviations)
}
