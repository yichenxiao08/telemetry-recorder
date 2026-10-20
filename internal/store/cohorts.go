package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/provenance"
)

func selectionBounds(s model.CohortSelection) error {
	if s.From < 0 || s.To <= s.From || time.Duration(s.To-s.From) > 31*24*time.Hour || s.Limit < 1 || s.Limit > 1000 {
		return fmt.Errorf("cohort needs explicit time range <=31 days and limit 1..1000")
	}
	return nil
}

// Decisions bounds the eligible population and raw evidence in one read snapshot.
// One final decision per step/run avoids treating loop iterations as samples.
func (r *Reader) Decisions(ctx context.Context, p model.AnalysisProfile, b model.CohortSelection) ([]model.DecisionObservation, map[string]int, error) {
	if err := provenance.ValidateProfile(p); err != nil {
		return nil, nil, err
	}
	if err := selectionBounds(b); err != nil {
		return nil, nil, err
	}
	tx, err := r.reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()
	return readDecisions(ctx, tx, p, b)
}

func readDecisions(ctx context.Context, tx *sql.Tx, p model.AnalysisProfile, b model.CohortSelection) ([]model.DecisionObservation, map[string]int, error) {
	rows, err := tx.QueryContext(ctx, "SELECT run_id FROM runs INDEXED BY runs_time WHERE started_at>=? AND started_at<? AND workload=? ORDER BY started_at,run_id LIMIT ?", b.From, b.To, p.Workload, b.Limit+1)
	if err != nil {
		return nil, nil, err
	}
	ids := []string{}
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
		return nil, nil, err
	}
	if len(ids) > b.Limit {
		return nil, nil, fmt.Errorf("eligible population exceeds explicit limit; narrow time range")
	}
	out := []model.DecisionObservation{}
	excluded := map[string]int{}
	total := 0
	for _, id := range ids {
		rows, err = tx.QueryContext(ctx, "SELECT data FROM signals WHERE run_id=? AND kind IN ('span','event') ORDER BY ts_ns,evidence_id LIMIT ?", id, 10001-total)
		if err != nil {
			return nil, nil, err
		}
		var selected *model.DecisionObservation
		session := ""
		for rows.Next() {
			total++
			if total > 10000 {
				_ = rows.Close()
				return nil, nil, fmt.Errorf("decision evidence exceeds 10000 records")
			}
			var raw string
			var signal model.Signal
			if err = rows.Scan(&raw); err != nil {
				break
			}
			if err = json.Unmarshal([]byte(raw), &signal); err != nil {
				break
			}
			if v, ok := signal.Attrs["signalscout.session.id"].(string); ok && v != "" {
				session = v
			}
			if obs, ok := provenance.Decode(signal, p); ok {
				if selected == nil || obs.TS > selected.TS || obs.TS == selected.TS && obs.EvidenceID > selected.EvidenceID {
					copy := obs
					selected = &copy
				}
			}
		}
		if err == nil {
			err = rows.Err()
		}
		_ = rows.Close()
		if err != nil {
			return nil, nil, err
		}
		if selected == nil {
			excluded["no_supported_decision"]++
			continue
		}
		selected.Session = session
		out = append(out, *selected)
	}
	return out, excluded, nil
}

func (r *Reader) SelectCohort(ctx context.Context, id string, p model.AnalysisProfile, b model.CohortSelection) (model.Cohort, error) {
	c := model.Cohort{ID: id, Profile: p, Selection: b, CreatedAt: time.Now().UnixNano(), RunIDs: []string{}, LabelIDs: []string{}, Excluded: map[string]int{}, DecisionIDs: map[string]string{}, Sessions: map[string]string{}}
	if !validText(id) {
		return c, fmt.Errorf("cohort ID required")
	}
	if err := provenance.ValidateProfile(p); err != nil {
		return c, err
	}
	if err := selectionBounds(b); err != nil {
		return c, err
	}
	if (b.Dimension == "") == (b.Predicate == "") {
		return c, fmt.Errorf("choose exactly one label/mark dimension or captured equality predicate")
	}
	tx, err := r.reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return c, err
	}
	defer func() { _ = tx.Rollback() }()
	observations, excluded, err := readDecisions(ctx, tx, p, b)
	if err != nil {
		return c, err
	}
	c.Excluded = excluded
	for _, obs := range observations {
		if b.Predicate != "" {
			value, ok := provenance.Get(obs.Values, b.Predicate)
			data, _ := json.Marshal(value)
			if !ok {
				c.Excluded["predicate_unknown"]++
				continue
			}
			if string(data) != b.Equals {
				c.Excluded["predicate_false"]++
				continue
			}
			c.RunIDs = append(c.RunIDs, obs.RunID)
			c.DecisionIDs[obs.RunID] = obs.EvidenceID
			c.Sessions[obs.RunID] = obs.Session
			continue
		}
		rows, e := tx.QueryContext(ctx, `SELECT a.id,a.value FROM annotations a WHERE a.run_id=? AND a.dimension=? AND a.retracts IS NULL
   AND NOT EXISTS(SELECT 1 FROM annotations x WHERE x.supersedes=a.id OR x.retracts=a.id) ORDER BY a.id LIMIT 1001`, obs.RunID, b.Dimension)
		if e != nil {
			return c, e
		}
		revisions := []string{}
		values := map[string]bool{}
		count := 0
		for rows.Next() {
			var label, value string
			if e = rows.Scan(&label, &value); e != nil {
				break
			}
			count++
			revisions = append(revisions, label)
			values[value] = true
		}
		if e == nil {
			e = rows.Err()
		}
		_ = rows.Close()
		if e != nil {
			return c, e
		}
		if count > 1000 {
			return c, fmt.Errorf("too many annotations per run")
		}
		if len(values) == 0 {
			c.Excluded["unlabeled_unknown"]++
			continue
		}
		if len(values) > 1 {
			c.Excluded["reviewer_disagreement"]++
			continue
		}
		if !values[b.Value] {
			c.Excluded["other_label"]++
			continue
		}
		c.RunIDs = append(c.RunIDs, obs.RunID)
		c.DecisionIDs[obs.RunID] = obs.EvidenceID
		c.Sessions[obs.RunID] = obs.Session
		c.LabelIDs = append(c.LabelIDs, revisions...)
	}
	sort.Strings(c.RunIDs)
	sort.Strings(c.LabelIDs)
	return c, nil
}

func (s *Store) SaveCohort(ctx context.Context, c model.Cohort) error {
	if !validText(c.ID) || len(c.RunIDs) > 1000 || len(c.LabelIDs) > 10000 {
		return fmt.Errorf("invalid cohort bounds")
	}
	if err := provenance.ValidateProfile(c.Profile); err != nil {
		return err
	}
	if err := selectionBounds(c.Selection); err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	var snapshot model.Cohort
	if err = json.Unmarshal(data, &snapshot); err != nil {
		return err
	}
	return s.submit(ctx, job{metadata: &metadataWrite{cohort: &snapshot}, done: make(chan error, 1)})
}

func (r *Reader) Cohort(ctx context.Context, id string) (model.Cohort, error) {
	var c model.Cohort
	var data string
	if err := r.reader.QueryRowContext(ctx, "SELECT data FROM cohorts WHERE id=?", id).Scan(&data); err != nil {
		return c, err
	}
	return c, json.Unmarshal([]byte(data), &c)
}

// CohortPair resolves pinned native decisions under one consistent read snapshot.
// A later decision or annotation cannot silently replace the selected revision.
func (r *Reader) CohortPair(ctx context.Context, pID, rID string) ([]model.Cohort, [][]model.DecisionObservation, error) {
	tx, err := r.reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()
	cohorts := []model.Cohort{}
	sets := [][]model.DecisionObservation{}
	for _, id := range []string{pID, rID} {
		var data string
		var c model.Cohort
		if err = tx.QueryRowContext(ctx, "SELECT data FROM cohorts WHERE id=?", id).Scan(&data); err != nil {
			return nil, nil, err
		}
		if err = json.Unmarshal([]byte(data), &c); err != nil {
			return nil, nil, err
		}
		observations := []model.DecisionObservation{}
		if len(c.RunIDs) > 1000 {
			return nil, nil, fmt.Errorf("oversized cohort")
		}
		for _, run := range c.RunIDs {
			var raw string
			err = tx.QueryRowContext(ctx, "SELECT data FROM signals WHERE evidence_id=? AND run_id=? AND kind='event'", c.DecisionIDs[run], run).Scan(&raw)
			if err != nil {
				return nil, nil, fmt.Errorf("pinned decision missing: %w", err)
			}
			var signal model.Signal
			if err = json.Unmarshal([]byte(raw), &signal); err != nil {
				return nil, nil, err
			}
			obs, ok := provenance.Decode(signal, c.Profile)
			if !ok {
				return nil, nil, fmt.Errorf("pinned decision is unsupported")
			}
			obs.Session = c.Sessions[run]
			observations = append(observations, obs)
		}
		cohorts = append(cohorts, c)
		sets = append(sets, observations)
	}
	return cohorts, sets, nil
}
