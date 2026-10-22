package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/completeness"
	"github.com/signalscout/signalscout/internal/model"
)

const (
	maxScopesPerCohort = 64
	maxListedIDs       = 8192
	// scopeLookahead lets a scope whose first record is its tail (emitted at
	// process shutdown) be found even when the cohort window ends before it.
	scopeLookahead = time.Hour
)

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// runEvidence is what readDecisions saw for one candidate run.
type runEvidence struct {
	RunID       string
	HasDecision bool
	Evidence    completeness.Evidence
	Epoch       string // from the run's own manifest, when present
}

// observeSignal folds one stored span into a run's completeness evidence.
func (e *runEvidence) observeSignal(s model.Signal) {
	if s.Kind != "span" {
		return
	}
	m, is, err := completeness.ParseManifest(s.Attrs)
	switch {
	case !is:
		e.Evidence.StoredSpans++
	case err != nil:
		e.Evidence.ManifestError = err.Error()
	default:
		e.Evidence.Manifest = m
		e.Epoch = strings.ToLower(m.Epoch)
	}
}

func addVerdict(c *model.CompletenessCounts, v completeness.Verdict) {
	switch v.Status {
	case completeness.Complete:
		c.Complete++
	case completeness.Partial:
		c.Partial++
	case completeness.Unknown:
		c.Unknown++
	case completeness.Absent:
		c.Absent++
	default:
		c.Unassessed++
	}
	c.LostItems += v.LostItems
	c.UnknownItems += v.UnknownItems
	c.DetailLossItems += v.DetailLoss
	if v.MixedUnknown {
		c.PartialWithUnknown++
	}
}

func loadScope(ctx context.Context, q queryer, epoch string) (completeness.Scope, error) {
	rows, err := q.QueryContext(ctx, "SELECT data FROM signals WHERE run_id=? AND kind='span' ORDER BY ts_ns,evidence_id LIMIT ?", epoch, completeness.MaxScopeChunks+1)
	if err != nil {
		return completeness.Scope{}, err
	}
	var records []*completeness.ScopeRecord
	count := 0
	for rows.Next() {
		count++
		var raw string
		var signal model.Signal
		if err = rows.Scan(&raw); err != nil {
			break
		}
		if err = json.Unmarshal([]byte(raw), &signal); err != nil {
			break
		}
		if record, is, e := completeness.ParseScope(signal.Attrs); is && e == nil {
			records = append(records, record)
		}
	}
	if err == nil {
		err = rows.Err()
	}
	_ = rows.Close()
	if err != nil {
		return completeness.Scope{}, err
	}
	return completeness.AssembleScope(epoch, records, count > completeness.MaxScopeChunks), nil
}

// findScopeEpochs lists capture-scope traces started in or shortly after the window.
func findScopeEpochs(ctx context.Context, q queryer, from, to int64) ([]string, bool, error) {
	rows, err := q.QueryContext(ctx, "SELECT run_id FROM runs INDEXED BY runs_workload WHERE workload=? AND started_at>=? AND started_at<? ORDER BY started_at,run_id LIMIT ?",
		completeness.ScopeWorkload, from, to+int64(scopeLookahead), maxScopesPerCohort+1)
	if err != nil {
		return nil, false, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	if err == nil {
		err = rows.Err()
	}
	_ = rows.Close()
	if err != nil {
		return nil, false, err
	}
	if len(ids) > maxScopesPerCohort {
		return ids[:maxScopesPerCohort], true, nil
	}
	return ids, false, nil
}

func runStored(ctx context.Context, q queryer, id string) (bool, error) {
	var one int
	err := q.QueryRowContext(ctx, "SELECT 1 FROM runs WHERE run_id=?", id).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

type assessment struct {
	Verdicts        map[string]completeness.Verdict
	Counts          model.CompletenessCounts
	Scopes          []string
	PopulationKnown bool
	HasEvidence     bool
}

// assess classifies every candidate run of a cohort population and the runs
// that relevant capture scopes expected but the store never saw.
func assess(ctx context.Context, q queryer, from, to int64, evidence []runEvidence) (assessment, error) {
	a := assessment{Verdicts: map[string]completeness.Verdict{}}
	candidates := map[string]bool{}
	referenced := map[string]bool{}
	epochs := map[string]bool{}
	for _, e := range evidence {
		candidates[e.RunID] = true
		if e.Evidence.Manifest != nil || e.Evidence.ManifestError != "" {
			a.HasEvidence = true
		}
		if e.Epoch != "" {
			referenced[e.Epoch], epochs[e.Epoch] = true, true
		}
	}
	found, truncated, err := findScopeEpochs(ctx, q, from, to)
	if err != nil {
		return a, err
	}
	for _, id := range found {
		epochs[id] = true
	}
	list := make([]string, 0, len(epochs))
	for epoch := range epochs {
		list = append(list, epoch)
	}
	sort.Strings(list)
	if len(list) > maxScopesPerCohort {
		list, truncated = list[:maxScopesPerCohort], true
	}
	known := !truncated
	inScope := map[string]bool{}
	loaded := map[string]bool{}
	var scopes []completeness.Scope
	for _, epoch := range list {
		scope, err := loadScope(ctx, q, epoch)
		if err != nil {
			return a, err
		}
		relevant := referenced[epoch]
		for id := range scope.IDs {
			relevant = relevant || candidates[id]
		}
		if !relevant {
			continue
		}
		scopes = append(scopes, scope)
		loaded[epoch] = true
		a.Scopes = append(a.Scopes, epoch)
		for id := range scope.IDs {
			inScope[id] = true
		}
		known = known && scope.PopulationKnown
	}
	for epoch := range referenced {
		known = known && loaded[epoch]
	}
	a.PopulationKnown = known && len(scopes) > 0
	a.HasEvidence = a.HasEvidence || len(scopes) > 0
	for _, e := range evidence {
		ev := e.Evidence
		ev.Seen, ev.InScope = true, inScope[e.RunID]
		v := completeness.Classify(ev)
		a.Verdicts[e.RunID] = v
		addVerdict(&a.Counts, v)
	}
	checked := 0
	for _, scope := range scopes {
		ids := make([]string, 0, len(scope.IDs))
		for id := range scope.IDs {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			if _, done := a.Verdicts[id]; done || candidates[id] {
				continue
			}
			if checked++; checked > maxListedIDs {
				a.PopulationKnown = false
				break
			}
			stored, err := runStored(ctx, q, id)
			if err != nil {
				return a, err
			}
			if stored {
				continue // stored under another workload: not a candidate, not absent
			}
			v := completeness.Classify(completeness.Evidence{InScope: true})
			a.Verdicts[id] = v
			addVerdict(&a.Counts, v)
		}
	}
	return a, nil
}

// requireComplete is the strict gate: complete capture cannot be demonstrated
// while any candidate is unknown or unassessed, any expected run is absent, or
// the run population itself is unknown.
func (a assessment) requireComplete() error {
	c := a.Counts
	if c.Unknown+c.Unassessed+c.Absent == 0 && a.PopulationKnown {
		return nil
	}
	return fmt.Errorf("complete capture required but not demonstrated: unknown=%d unassessed=%d absent=%d population_known=%v (partial=%d complete=%d)",
		c.Unknown, c.Unassessed, c.Absent, a.PopulationKnown, c.Partial, c.Complete)
}

// ScopeReport is the read-only completeness of one capture scope's listed runs.
type ScopeReport struct {
	Epoch           string                   `json:"epoch"`
	ScopeFinal      bool                     `json:"scope_final"`
	ScopeExpected   int                      `json:"scope_expected"`
	ListedRuns      int                      `json:"listed_runs"`
	PopulationKnown bool                     `json:"population_known"`
	ScopeReason     string                   `json:"scope_reason,omitempty"`
	Counts          model.CompletenessCounts `json:"counts"`
	Reasons         map[string]int           `json:"reasons"`
	Runs            map[string]string        `json:"runs"`
}

// Completeness classifies every run the scope lists, regardless of workload.
// The scope trace itself is never one of the runs counted.
func (r *Reader) Completeness(ctx context.Context, epoch string) (ScopeReport, error) {
	epoch = strings.ToLower(epoch)
	report := ScopeReport{Epoch: epoch, Reasons: map[string]int{}, Runs: map[string]string{}}
	tx, err := r.reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return report, err
	}
	defer func() { _ = tx.Rollback() }()
	scope, err := loadScope(ctx, tx, epoch)
	if err != nil {
		return report, err
	}
	report.ScopeFinal, report.ScopeExpected, report.PopulationKnown, report.ScopeReason = scope.Final, scope.Expected, scope.PopulationKnown, scope.Reason
	report.ListedRuns = len(scope.IDs)
	ids := make([]string, 0, len(scope.IDs))
	for id := range scope.IDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) > maxListedIDs {
		ids, report.PopulationKnown = ids[:maxListedIDs], false
	}
	for _, id := range ids {
		ev := completeness.Evidence{InScope: true}
		if ev.Seen, err = runStored(ctx, tx, id); err != nil {
			return report, err
		}
		if ev.Seen {
			holder := runEvidence{RunID: id}
			var raw string
			err = tx.QueryRowContext(ctx, "SELECT data FROM signals WHERE evidence_id=?", "span:"+id+"/"+completeness.ManifestSpanID).Scan(&raw)
			switch {
			case err == nil:
				var signal model.Signal
				if err = json.Unmarshal([]byte(raw), &signal); err != nil {
					return report, err
				}
				holder.observeSignal(signal)
			case !errors.Is(err, sql.ErrNoRows):
				return report, err
			}
			var spans int64
			if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM signals WHERE run_id=? AND kind='span'", id).Scan(&spans); err != nil {
				return report, err
			}
			ev = holder.Evidence
			ev.Seen, ev.InScope = true, true
			if ev.Manifest != nil || ev.ManifestError != "" {
				spans-- // the manifest span is not an acknowledged item
			}
			ev.StoredSpans = spans
		}
		v := completeness.Classify(ev)
		addVerdict(&report.Counts, v)
		report.Reasons[string(v.Status)+": "+v.Reason]++
		report.Runs[id] = string(v.Status)
	}
	return report, nil
}
