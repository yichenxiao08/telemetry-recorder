package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/signalscout/signalscout/internal/model"
)

const reportSignalsSQL = `SELECT CASE WHEN length(CAST(data AS BLOB))<=1048576 THEN data ELSE '' END FROM signals INDEXED BY signals_run WHERE run_id=? AND kind IN ('span','event','log','metric') AND ts_ns>=? AND ts_ns<? ORDER BY kind,ts_ns,evidence_id LIMIT ?`
const reportNativeSQL = `SELECT CASE WHEN length(CAST(data AS BLOB))<=1048576 THEN data ELSE '' END FROM signals WHERE evidence_id=? AND ts_ns>=? AND ts_ns<? LIMIT 1`

type ReportRequest struct {
	From, To                 int64
	Limit                    int
	Cohorts, Runs, NativeIDs []string
}
type ReportSnapshot struct {
	Cohorts map[string]model.Cohort
	Signals []model.Signal
}

// ReportRead is a single read-only snapshot with a shared row and byte budget.
// A cap rejects the whole snapshot. Metadata uses primary-key lookups; signal
// queries always include explicit event-time bounds and LIMIT.
func (r *Reader) ReportRead(ctx context.Context, q ReportRequest) (ReportSnapshot, error) {
	out := ReportSnapshot{Cohorts: map[string]model.Cohort{}, Signals: []model.Signal{}}
	if q.From < 0 || q.To <= q.From || q.To-q.From > int64(31*24*time.Hour) || q.Limit < 1 || q.Limit > 10000 || len(q.Cohorts) > 4 || len(q.Runs) > 1000 || len(q.NativeIDs) > 64 {
		return out, fmt.Errorf("report requires <=31 day bounds, record limit 1..10000, <=4 cohorts, <=1000 runs, <=64 native ids")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	tx, err := r.reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback() }()
	runs := map[string]bool{}
	bytesRead := 0
	for _, run := range q.Runs {
		if run == "" {
			return out, fmt.Errorf("empty run partition")
		}
		runs[run] = true
	}
	for _, id := range q.Cohorts {
		var raw string
		if err = tx.QueryRowContext(ctx, "SELECT data FROM cohorts WHERE id=? AND length(CAST(data AS BLOB))<=33554432 LIMIT 1", id).Scan(&raw); err != nil {
			return out, err
		}
		bytesRead += len(raw)
		if bytesRead > 32<<20 {
			return out, fmt.Errorf("snapshot byte budget exceeded")
		}
		var c model.Cohort
		if err = json.Unmarshal([]byte(raw), &c); err != nil {
			return out, err
		}
		if c.Selection.From < q.From || c.Selection.To > q.To || len(c.RunIDs) > 1000 {
			return out, fmt.Errorf("cohort outside declared report scope")
		}
		out.Cohorts[id] = c
		for _, run := range c.RunIDs {
			runs[run] = true
		}
		if c.Relations != nil {
			if len(c.Relations.Runs) > 1000 {
				return out, fmt.Errorf("oversized relation snapshot")
			}
			for run := range c.Relations.Runs {
				runs[run] = true
			}
		}
	}
	if len(runs) > 1000 {
		return out, fmt.Errorf("report run partition budget exceeded")
	}
	ids := make([]string, 0, len(runs))
	for run := range runs {
		ids = append(ids, run)
	}
	sort.Strings(ids)
	seen := map[string]bool{}
	add := func(raw string) error {
		if raw == "" {
			return fmt.Errorf("record exceeds 1 MiB transfer limit; snapshot rejected")
		}
		bytesRead += len(raw)
		if bytesRead > 32<<20 {
			return fmt.Errorf("snapshot byte budget exceeded")
		}
		var s model.Signal
		if e := json.Unmarshal([]byte(raw), &s); e != nil {
			return e
		}
		if !seen[s.ID] {
			seen[s.ID] = true
			out.Signals = append(out.Signals, s)
		}
		if len(out.Signals) > q.Limit {
			return fmt.Errorf("report record cap %d exceeded; entire snapshot rejected", q.Limit)
		}
		return nil
	}
	for _, run := range ids {
		var found string
		if err = tx.QueryRowContext(ctx, "SELECT run_id FROM runs WHERE run_id=? AND started_at>=? AND started_at<? LIMIT 1", run, q.From, q.To).Scan(&found); err != nil {
			return out, fmt.Errorf("run %s outside scope or missing: %w", run, err)
		}
		rows, e := tx.QueryContext(ctx, reportSignalsSQL, run, q.From, q.To, q.Limit+1-len(out.Signals))
		if e != nil {
			return out, e
		}
		for rows.Next() {
			var raw string
			if err = rows.Scan(&raw); err == nil {
				err = add(raw)
			}
			if err != nil {
				break
			}
		}
		if err == nil {
			err = rows.Err()
		}
		_ = rows.Close()
		if err != nil {
			return out, err
		}
	}
	for _, id := range q.NativeIDs {
		if seen[id] {
			continue
		}
		var raw string
		err = tx.QueryRowContext(ctx, reportNativeSQL, id, q.From, q.To).Scan(&raw)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return out, err
		}
		if err = add(raw); err != nil {
			return out, err
		}
	}
	return out, tx.Commit()
}
