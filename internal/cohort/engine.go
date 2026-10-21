package cohort

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"sort"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/provenance"
	"github.com/signalscout/signalscout/internal/stats"
	"github.com/signalscout/signalscout/internal/store"
)

type Finding struct {
	Type               string                    `json:"type"`
	Feature            string                    `json:"feature"`
	Score              float64                   `json:"score"`
	Status             string                    `json:"status"`
	Tier               string                    `json:"tier"`
	Summary            string                    `json:"summary"`
	ProblemObserved    int                       `json:"problem_observed"`
	ReferenceObserved  int                       `json:"reference_observed"`
	ProblemMissing     int                       `json:"problem_missing"`
	ReferenceMissing   int                       `json:"reference_missing"`
	ProblemRuns        int                       `json:"problem_runs"`
	ReferenceRuns      int                       `json:"reference_runs"`
	ProblemSessions    int                       `json:"problem_sessions"`
	ReferenceSessions  int                       `json:"reference_sessions"`
	SharedSessions     int                       `json:"shared_sessions"`
	IndependenceKnown  bool                      `json:"independence_known"`
	Exclusions         map[string]map[string]int `json:"selection_exclusions"`
	ProblemNumerator   int                       `json:"problem_numerator"`
	ReferenceNumerator int                       `json:"reference_numerator"`
	EvidenceIDs        []string                  `json:"evidence_ids"`
	EvidenceURLs       []string                  `json:"evidence_urls"`
}
type InvariantResult struct {
	RunID      string `json:"run_id"`
	Status     string `json:"status"`
	EvidenceID string `json:"evidence_id"`
}
type Result struct {
	Status            string            `json:"status"`
	Problem           model.Cohort      `json:"problem"`
	Reference         model.Cohort      `json:"reference"`
	ProblemRuns       int               `json:"problem_runs"`
	ReferenceRuns     int               `json:"reference_runs"`
	ProblemSessions   int               `json:"problem_sessions"`
	ReferenceSessions int               `json:"reference_sessions"`
	UnknownSessions   int               `json:"unknown_sessions"`
	SharedSessions    int               `json:"shared_sessions"`
	Differences       []Finding         `json:"differences"`
	Invariants        []InvariantResult `json:"invariants"`
	InvariantCounts   map[string]int    `json:"invariant_counts"`
}

func Analyze(ctx context.Context, reader *store.Reader, problem, reference string) (Result, error) {
	cs, sets, err := reader.CohortPair(ctx, problem, reference)
	if err != nil {
		return Result{}, err
	}
	return Compare(cs[0], cs[1], sets[0], sets[1])
}

func Compare(p, r model.Cohort, problem, reference []model.DecisionObservation) (Result, error) {
	out := Result{Status: "ok", Problem: p, Reference: r, ProblemRuns: len(problem), ReferenceRuns: len(reference), Differences: []Finding{}, Invariants: []InvariantResult{}, InvariantCounts: map[string]int{}}
	a, _ := json.Marshal(p.Profile)
	b, _ := json.Marshal(r.Profile)
	if string(a) != string(b) {
		return out, fmt.Errorf("cohorts require identical unit/profile/workload and feature policy")
	}
	if err := provenance.ValidateProfile(p.Profile); err != nil {
		return out, err
	}
	ids := map[string]bool{}
	for _, id := range p.RunIDs {
		if ids[id] {
			return out, fmt.Errorf("duplicate cohort member")
		}
		ids[id] = true
	}
	for _, id := range r.RunIDs {
		if ids[id] {
			return out, fmt.Errorf("problem/reference cohorts overlap")
		}
		ids[id] = true
	}
	sessions := func(xs []model.DecisionObservation) int {
		unique := map[string]bool{}
		for _, x := range xs {
			if x.Session == "" {
				out.UnknownSessions++
			} else {
				unique[x.Session] = true
			}
		}
		return len(unique)
	}
	out.ProblemSessions = sessions(problem)
	out.ReferenceSessions = sessions(reference)
	shared := map[string]bool{}
	for _, x := range problem {
		if x.Session != "" {
			shared[x.Session] = true
		}
	}
	counted := map[string]bool{}
	for _, x := range reference {
		if shared[x.Session] && !counted[x.Session] {
			out.SharedSessions++
			counted[x.Session] = true
		}
	}
	if len(problem) < 10 || len(reference) < 10 {
		out.Status = "insufficient_evidence"
	}
	add := func(f Finding) {
		f.ProblemMissing = len(problem) - f.ProblemObserved
		f.ReferenceMissing = len(reference) - f.ReferenceObserved
		f.Tier = "association"
		if f.Status == "" {
			f.Status = "ok"
		}
		if f.Status == "unsupported_numeric" {
			f.Score = 0
			f.Tier = "observation"
		}
		f.ProblemRuns = len(problem)
		f.ReferenceRuns = len(reference)
		f.ProblemSessions = out.ProblemSessions
		f.ReferenceSessions = out.ReferenceSessions
		f.SharedSessions = out.SharedSessions
		f.Exclusions = map[string]map[string]int{"problem": p.Excluded, "reference": r.Excluded}
		if f.ProblemObserved < 10 || f.ReferenceObserved < 10 {
			f.Score = 0
			f.Status = "insufficient_evidence"
			f.Tier = "observation"
		}
		if f.Status == "ok" && f.Score == 0 {
			f.Status = "no_difference"
			f.Tier = "observation"
		}
		for _, xs := range [][]model.DecisionObservation{problem, reference} {
			count := 0
			for _, x := range xs {
				if count == 3 {
					break
				}
				sum := sha256.Sum256([]byte(x.EvidenceID))
				f.EvidenceIDs = append(f.EvidenceIDs, x.EvidenceID)
				f.EvidenceURLs = append(f.EvidenceURLs, "/run?id="+url.QueryEscape(x.RunID)+"#e-"+hex.EncodeToString(sum[:]))
				count++
			}
		}
		out.Differences = append(out.Differences, f)
	}
	for _, feature := range p.Profile.Categories {
		values := func(xs []model.DecisionObservation) []string {
			out := []string{}
			for _, x := range xs {
				if v, ok := provenance.Get(x.Values, feature.Selector); ok {
					if s, ok := v.(string); ok {
						out = append(out, s)
					}
				}
			}
			return out
		}
		pv, rv := values(problem), values(reference)
		score, category, pc, rc, _ := Categorical(pv, rv)
		add(Finding{Type: "categorical", Feature: feature.Name, Score: score, ProblemObserved: len(pv), ReferenceObserved: len(rv), ProblemNumerator: pc, ReferenceNumerator: rc, Summary: fmt.Sprintf("%s category %q: %d/%d problem vs %d/%d reference; Laplace total variation %.6f", feature.Name, category, pc, len(pv), rc, len(rv), score)})
	}
	for _, feature := range p.Profile.Numeric {
		values := func(xs []model.DecisionObservation) []float64 {
			out := []float64{}
			for _, x := range xs {
				v, ok := provenance.Get(x.Values, feature.Selector)
				number, valid := v.(float64)
				if !ok || !valid {
					continue
				}
				if feature.Subtract != "" {
					v, ok = provenance.Get(x.Values, feature.Subtract)
					other, valid := v.(float64)
					if !ok || !valid {
						continue
					}
					number -= other
				}
				if !math.IsNaN(number) && !math.IsInf(number, 0) {
					out = append(out, number)
				}
			}
			return out
		}
		pv, rv := values(problem), values(reference)
		score := Numeric(pv, rv, feature.Epsilon, feature.Floor)
		x, m := stats.Median(pv), stats.Median(rv)
		pc, rc := 0, 0
		for _, v := range pv {
			if x >= m && v >= m || x < m && v <= m {
				pc++
			}
		}
		for _, v := range rv {
			if x >= m && v >= x || x < m && v <= x {
				rc++
			}
		}
		summary := fmt.Sprintf("%s median %g vs %g %s; problem directional support %d/%d; reference directional exceedance %d/%d", feature.Name, x, m, feature.Unit, pc, len(pv), rc, len(rv))
		if len(pv) == 0 || len(rv) == 0 {
			summary = feature.Name + ": missing capture; no numeric effect imputed"
		}
		status := ""
		if len(pv) > 0 && len(rv) > 0 && !numericDefined(pv, rv) {
			status = "unsupported_numeric"
			summary = feature.Name + ": arithmetic overflow; observed counts retained, comparison unknown"
		}
		add(Finding{Type: "input_value", Feature: feature.Name, Score: score, Status: status, ProblemObserved: len(pv), ReferenceObserved: len(rv), ProblemNumerator: pc, ReferenceNumerator: rc, Summary: summary})
	}
	sort.Slice(out.Differences, func(i, j int) bool {
		a, b := out.Differences[i], out.Differences[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		return a.Feature < b.Feature
	})
	inv := p.Profile.Invariant
	for _, xs := range [][]model.DecisionObservation{problem, reference} {
		for _, x := range xs {
			flat := map[string]any{}
			for _, key := range []string{inv.Declared, inv.Current, inv.Used} {
				if value, ok := provenance.Get(x.Values, key); ok {
					flat[key] = value
				}
			}
			status := VersionCheck(flat, inv.Declared, inv.Current, inv.Used, inv.Delta)
			out.InvariantCounts[status]++
			out.Invariants = append(out.Invariants, InvariantResult{x.RunID, status, x.EvidenceID})
		}
	}
	return out, nil
}
