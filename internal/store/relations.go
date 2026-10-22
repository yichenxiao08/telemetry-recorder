package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/provenance"
)

func newRelationCounts() *model.RelationCounts {
	return &model.RelationCounts{ViolationEvidence: []string{}, ViolationURLs: []string{}, UnknownReasons: map[string]int{}}
}
func newRelationReport(p model.AnalysisProfile, b model.CohortSelection) *model.RelationReport {
	cap := b.RecordLimit
	if cap == 0 {
		cap = 100000
	}
	r := &model.RelationReport{Version: 1, Tier: "observation", Step: p.Step, Profile: p, From: b.From, To: b.To, RunLimit: b.Limit, RecordLimit: cap, RunsWithoutDecision: []string{}, Totals: map[string]*model.RelationCounts{}, Runs: map[string]map[string]*model.RelationCounts{}, SamplingNote: "Counts are recorded decisions, not total host calls. Sampled capture is incomplete: unsampled calls are unknown, never satisfied or evidence of absence. Without a complete capture manifest, missing calls cannot be counted. Boundary capture does not prove the code read a value."}
	for _, rule := range p.Relations.Rules {
		r.Totals[rule.ID] = newRelationCounts()
	}
	return r
}
func relationRun(r *model.RelationReport, id string) map[string]*model.RelationCounts {
	if r.Runs[id] == nil {
		r.Runs[id] = map[string]*model.RelationCounts{}
		for _, rule := range r.Profile.Relations.Rules {
			r.Runs[id][rule.ID] = newRelationCounts()
		}
	}
	return r.Runs[id]
}
func observeRelations(r *model.RelationReport, p model.AnalysisProfile, obs model.DecisionObservation) {
	rows := relationRun(r, obs.RunID)
	for _, rule := range p.Relations.Rules {
		result := provenance.EvaluateRelation(obs.Values, rule, p.SelectorSyntax)
		for _, c := range []*model.RelationCounts{r.Totals[rule.ID], rows[rule.ID]} {
			c.Total++
			if obs.Sampled {
				c.SampledRecords++
			}
			switch result.Status {
			case "satisfied":
				c.Satisfied++
			case "not_applicable":
				c.NotApplicable++
			case "violated":
				c.Violated++
				c.LastViolation = obs.EvidenceID
				if len(c.ViolationEvidence) < 3 {
					c.ViolationEvidence = append(c.ViolationEvidence, obs.EvidenceID)
					sum := sha256.Sum256([]byte(obs.EvidenceID))
					c.ViolationURLs = append(c.ViolationURLs, "/run?id="+url.QueryEscape(obs.RunID)+"#e-"+hex.EncodeToString(sum[:]))
				} else {
					c.EvidenceOmitted++
				}
			default:
				c.Unknown++
				c.UnknownReasons[result.Reason]++
			}
		}
	}
}
func finishRelations(r *model.RelationReport, evidence []runEvidence) {
	r.CandidateRuns = len(evidence)
	for _, e := range evidence {
		relationRun(r, e.RunID)
		if !e.HasDecision {
			r.RunsWithoutDecision = append(r.RunsWithoutDecision, e.RunID)
		}
	}
}

// Relations evaluates every matching decision in one bounded read snapshot.
// It does not turn unobserved/sampled-out calls into successful checks.
func (r *Reader) Relations(ctx context.Context, p model.AnalysisProfile, b model.CohortSelection) (model.RelationReport, error) {
	if err := provenance.ValidateProfile(p); err != nil {
		return model.RelationReport{}, err
	}
	if err := selectionBounds(b); err != nil {
		return model.RelationReport{}, err
	}
	if p.Relations == nil {
		return model.RelationReport{}, fmt.Errorf("profile has no declared relations")
	}
	tx, err := r.reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return model.RelationReport{}, err
	}
	defer func() { _ = tx.Rollback() }()
	out := newRelationReport(p, b)
	_, _, evidence, err := walkDecisions(ctx, tx, p, b, func(obs model.DecisionObservation) { observeRelations(out, p, obs) })
	if err != nil {
		return model.RelationReport{}, err
	}
	finishRelations(out, evidence)
	return *out, nil
}
