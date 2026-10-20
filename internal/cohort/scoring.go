// Package cohort compares explicitly selected examples. Scores are associations.
package cohort

import (
	"math"
	"sort"

	"github.com/signalscout/signalscout/internal/stats"
)

// Categorical returns Laplace-smoothed total variation and the strongest excess
// category's raw counts. No domain-specific strings affect the score.
func Categorical(problem, reference []string) (score float64, category string, pCount, rCount, k int) {
	p, r := map[string]int{}, map[string]int{}
	union := map[string]bool{}
	for _, v := range problem {
		p[v]++
		union[v] = true
	}
	for _, v := range reference {
		r[v]++
		union[v] = true
	}
	if len(problem) == 0 || len(reference) == 0 {
		return 0, "", 0, 0, len(union)
	}
	keys := make([]string, 0, len(union))
	for v := range union {
		keys = append(keys, v)
	}
	sort.Strings(keys)
	k = len(keys)
	strongest := -1.0
	for _, v := range keys {
		delta := float64(p[v]+1)/float64(len(problem)+k) - float64(r[v]+1)/float64(len(reference)+k)
		score += math.Abs(delta) / 2
		if delta > strongest {
			strongest = delta
			category = v
			pCount = p[v]
			rCount = r[v]
		}
	}
	return
}

func Numeric(problem, reference []float64, epsilon, floor float64) float64 {
	if len(problem) == 0 || len(reference) == 0 || epsilon <= 0 || floor <= 0 {
		return 0
	}
	x, m := stats.Median(problem), stats.Median(reference)
	effect := math.Abs(x - m)
	if math.IsNaN(effect) || math.IsInf(effect, 0) || effect < floor {
		return 0
	}
	e := 0
	for _, v := range reference {
		if x >= m && v >= x || x < m && v <= x {
			e++
		}
	}
	scale := math.Max(1.4826*stats.MAD(reference), epsilon)
	return (1 - math.Exp(-effect/scale/3)) * (1 - float64(e+1)/float64(len(reference)+2))
}

func VersionCheck(values map[string]any, declared, current, used string, delta float64) string {
	applicable, ok := values[declared].(bool)
	if !ok {
		return "unknown"
	}
	if !applicable {
		return "not_applicable"
	}
	n, nOK := values[current].(float64)
	v, vOK := values[used].(float64)
	if !nOK || !vOK || math.IsNaN(n) || math.IsNaN(v) || math.IsInf(n, 0) || math.IsInf(v, 0) || math.Trunc(n) != n || math.Trunc(v) != v {
		return "unknown"
	}
	if n-v == delta {
		return "satisfied"
	}
	return "violated"
}
