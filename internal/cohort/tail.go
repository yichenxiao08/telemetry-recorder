package cohort

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"net/url"
	"sort"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/provenance"
	"github.com/signalscout/signalscout/internal/stats"
)

// TailTerms is a separate descriptive association; it does not change median scores.
type TailTerms struct {
	Status           string  `json:"status"`
	Score            float64 `json:"score"`
	Q05              float64 `json:"q05"`
	Q95              float64 `json:"q95"`
	Delta            float64 `json:"delta"`
	Lower            float64 `json:"lower"`
	Upper            float64 `json:"upper"`
	ProblemLeft      int     `json:"problem_left"`
	ProblemRight     int     `json:"problem_right"`
	ProblemOutside   int     `json:"problem_outside"`
	ReferenceLeft    int     `json:"reference_left"`
	ReferenceRight   int     `json:"reference_right"`
	ReferenceOutside int     `json:"reference_outside"`
	ProblemRate      float64 `json:"problem_smoothed_rate"`
	ReferenceRate    float64 `json:"reference_smoothed_rate"`
}

// Unknown comparisons serialize a null score, never a measured zero association.
func (t TailTerms) MarshalJSON() ([]byte, error) {
	type plain TailTerms
	var score *float64
	if t.Status == "ok" || t.Status == "no_difference" {
		score = &t.Score
	}
	return json.Marshal(struct {
		plain
		Score *float64 `json:"score"`
	}{plain(t), score})
}

func ReferenceTail(p, r []float64, epsilon, floor float64, ps, rs int) TailTerms {
	t := TailTerms{Status: "insufficient_tail_evidence"}
	if len(p) < 30 || len(r) < 100 || ps < 5 || rs < 5 {
		return t
	}
	t.Status = "unsupported_numeric"
	for _, xs := range [][]float64{p, r} {
		for _, v := range xs {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return t
			}
		}
	}
	if epsilon <= 0 || floor <= 0 || math.IsNaN(epsilon) || math.IsNaN(floor) || math.IsInf(epsilon, 0) || math.IsInf(floor, 0) {
		return t
	}
	q05, q95 := stats.Quantile(r, .05), stats.Quantile(r, .95)
	delta := math.Max(epsilon, floor) / 2
	lo, hi := q05-delta, q95+delta
	for _, v := range []float64{q05, q95, lo, hi} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return t
		}
	}
	t.Q05, t.Q95, t.Delta, t.Lower, t.Upper = q05, q95, delta, lo, hi
	for _, v := range p {
		if v < lo {
			t.ProblemLeft++
		}
		if v > hi {
			t.ProblemRight++
		}
	}
	for _, v := range r {
		if v < lo {
			t.ReferenceLeft++
		}
		if v > hi {
			t.ReferenceRight++
		}
	}
	t.ProblemOutside = t.ProblemLeft + t.ProblemRight
	t.ReferenceOutside = t.ReferenceLeft + t.ReferenceRight
	t.ProblemRate = float64(t.ProblemOutside+1) / float64(len(p)+2)
	t.ReferenceRate = float64(t.ReferenceOutside+1) / float64(len(r)+2)
	t.Score = math.Max(0, t.ProblemRate-t.ReferenceRate)
	t.Status = "ok"
	if t.Score == 0 {
		t.Status = "no_difference"
	}
	return t
}

type TailFinding struct {
	Finding
	Terms                    TailTerms `json:"terms"`
	ProblemUnknownSessions   int       `json:"problem_unknown_sessions"`
	ReferenceUnknownSessions int       `json:"reference_unknown_sessions"`
}

func (t TailFinding) MarshalJSON() ([]byte, error) {
	type plain TailFinding
	var score *float64
	if t.Status == "ok" || t.Status == "no_difference" {
		score = &t.Score
	}
	return json.Marshal(struct {
		plain
		Score *float64 `json:"score"`
	}{plain(t), score})
}

type Presentation struct {
	Problem []Finding     `json:"the_problem"`
	Median  []Finding     `json:"candidate_median"`
	Tail    []TailFinding `json:"candidate_tail"`
}

func observedNumber(x model.DecisionObservation, f model.NumericFeature) (float64, bool) {
	a, ok := provenance.Get(x.Values, f.Selector)
	v, valid := a.(float64)
	if !ok || !valid {
		return 0, false
	}
	if f.Subtract != "" {
		a, ok = provenance.Get(x.Values, f.Subtract)
		other, valid := a.(float64)
		if !ok || !valid {
			return 0, false
		}
		v -= other
	}
	return v, !math.IsNaN(v) && !math.IsInf(v, 0)
}
func numericObservations(xs []model.DecisionObservation, f model.NumericFeature) ([]float64, map[string]bool, int) {
	values := []float64{}
	sessions := map[string]bool{}
	unknown := 0
	for _, x := range xs {
		v, valid := observedNumber(x, f)
		if !valid {
			continue
		}
		values = append(values, v)
		if x.Session == "" {
			unknown++
		} else {
			sessions[x.Session] = true
		}
	}
	return values, sessions, unknown
}
func tailEvidence(base *Finding, xs []model.DecisionObservation, f model.NumericFeature, t TailTerms) {
	selected := map[string]bool{}
	// Prefer actual outside witnesses, then fill the bounded sample with observed
	// values. This changes only new tail citations, never legacy median evidence.
	for _, outsideOnly := range []bool{true, false} {
		for _, x := range xs {
			if len(selected) == 3 {
				return
			}
			v, valid := observedNumber(x, f)
			if !valid || selected[x.EvidenceID] {
				continue
			}
			if outsideOnly && (t.Status != "ok" && t.Status != "no_difference" || !(v < t.Lower || v > t.Upper)) {
				continue
			}
			selected[x.EvidenceID] = true
			sum := sha256.Sum256([]byte(x.EvidenceID))
			base.EvidenceIDs = append(base.EvidenceIDs, x.EvidenceID)
			base.EvidenceURLs = append(base.EvidenceURLs, "/run?id="+url.QueryEscape(x.RunID)+"#e-"+hex.EncodeToString(sum[:]))
		}
	}
}
func present(profile model.AnalysisProfile, findings []Finding, p, r []model.DecisionObservation) *Presentation {
	if !profile.ReferenceTail && profile.RoleAuthor == "" {
		return nil
	}
	out := &Presentation{Problem: []Finding{}, Median: []Finding{}, Tail: []TailFinding{}}
	roles := map[string]string{}
	for _, f := range profile.Numeric {
		roles[f.Name] = f.Role
	}
	for _, f := range profile.Categories {
		roles[f.Name] = f.Role
	}
	for _, f := range findings {
		if roles[f.Feature] == "symptom" {
			out.Problem = append(out.Problem, f)
		} else {
			out.Median = append(out.Median, f)
		}
	}
	if !profile.ReferenceTail {
		return out
	}
	for _, f := range profile.Numeric {
		if f.Role == "symptom" {
			continue
		}
		pv, ps, pu := numericObservations(p, f)
		rv, rs, ru := numericObservations(r, f)
		var base Finding
		for _, x := range findings {
			if x.Feature == f.Name {
				base = x
				break
			}
		}
		terms := ReferenceTail(pv, rv, f.Epsilon, f.Floor, len(ps), len(rs))
		base.Score, base.Status = terms.Score, terms.Status
		base.ProblemObserved, base.ReferenceObserved = len(pv), len(rv)
		base.ProblemMissing, base.ReferenceMissing = len(p)-len(pv), len(r)-len(rv)
		base.ProblemSessions, base.ReferenceSessions = len(ps), len(rs)
		base.SharedSessions = 0
		for session := range ps {
			if rs[session] {
				base.SharedSessions++
			}
		}
		base.ProblemNumerator, base.ReferenceNumerator = terms.ProblemOutside, terms.ReferenceOutside
		base.Summary = "Reference-tail association; strict outside padded reference interval, not causal relevance"
		base.EvidenceIDs, base.EvidenceURLs = nil, nil
		tailEvidence(&base, p, f, terms)
		tailEvidence(&base, r, f, terms)
		base.Tier = "association"
		if terms.Status != "ok" {
			base.Tier = "observation"
		}
		out.Tail = append(out.Tail, TailFinding{base, terms, pu, ru})
	}
	sort.Slice(out.Tail, func(i, j int) bool {
		a, b := out.Tail[i], out.Tail[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		return a.Feature < b.Feature
	})
	return out
}
