package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/runs"
)

func readRun(ctx context.Context, tx *sql.Tx, id string) (model.Run, error) {
	var data string
	var state runs.State
	if err := tx.QueryRowContext(ctx, "SELECT state FROM runs WHERE run_id=?", id).Scan(&data); err != nil {
		return state.Run, fmt.Errorf("find run %q: %w", id, err)
	}
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return state.Run, fmt.Errorf("decode run %q: %w", id, err)
	}
	return state.Run, nil
}

// Comparison loads only M3's owned spans/events. Context is deliberately absent.
// A single read transaction freezes selection and evidence at the same snapshot.
func (r *Reader) Comparison(ctx context.Context, id string, options model.BaselineOptions) (model.Comparison, error) {
	return r.comparison(ctx, id, options, false)
}

// ViewerComparison refuses oversized inputs instead of computing a partial diff.
func (r *Reader) ViewerComparison(ctx context.Context, id string, options model.BaselineOptions) (model.Comparison, error) {
	if options.N > 200 {
		return model.Comparison{}, fmt.Errorf("viewer baseline limit is 200 runs")
	}
	return r.comparison(ctx, id, options, true)
}

func (r *Reader) comparison(ctx context.Context, id string, options model.BaselineOptions, bounded bool) (model.Comparison, error) {
	result := model.Comparison{}
	n := options.N
	if !options.ExplicitN && n == 0 {
		n = 40
	}
	if n <= 0 {
		return result, fmt.Errorf("baseline-n must be positive")
	}
	tx, err := r.reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return result, fmt.Errorf("begin comparison: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	result.Failing.Run, err = readRun(ctx, tx, id)
	if err != nil {
		return result, err
	}
	failing := result.Failing.Run
	if !failing.Closed || failing.Outcome != "fail" {
		return result, fmt.Errorf("diff requires a closed failing run: %s", id)
	}
	result.Spec = model.BaselineSpec{Mode: "default", N: n, ExplicitN: options.ExplicitN, BeforeInput: options.Before, BeforeNS: failing.StartedAt, Workload: failing.Workload, Minimum: model.MinimumBaseline, RunIDs: []string{}, ScoringVersion: model.DiffScoringVersion}
	if options.ExplicitN {
		result.Spec.Mode = "count"
	}
	if options.Before != "" {
		result.Spec.Mode = "before"
		var cutoff int64
		if timestamp, parseErr := time.Parse(time.RFC3339Nano, options.Before); parseErr == nil {
			cutoff = timestamp.UnixNano()
			if !time.Unix(0, cutoff).Equal(timestamp) {
				return result, fmt.Errorf("baseline-before timestamp outside nanosecond range")
			}
		} else {
			anchor, err := readRun(ctx, tx, options.Before)
			if err != nil {
				return result, fmt.Errorf("baseline-before must be an existing run ID or RFC3339 time: %w", err)
			}
			cutoff = anchor.StartedAt
		}
		if cutoff < result.Spec.BeforeNS {
			result.Spec.BeforeNS = cutoff
		}
	}
	rows, err := tx.QueryContext(ctx, "SELECT state FROM runs WHERE workload=? AND outcome='pass' AND closed=1 AND started_at<? ORDER BY started_at DESC,run_id LIMIT ?", failing.Workload, result.Spec.BeforeNS, n)
	if err != nil {
		return result, fmt.Errorf("select baseline: %w", err)
	}
	for rows.Next() {
		var data string
		var state runs.State
		if err = rows.Scan(&data); err != nil {
			break
		}
		if err = json.Unmarshal([]byte(data), &state); err != nil {
			break
		}
		result.Baseline = append(result.Baseline, model.RunDetail{Run: state.Run})
		result.Spec.RunIDs = append(result.Spec.RunIDs, state.Run.ID)
	}
	if err == nil {
		err = rows.Err()
	}
	_ = rows.Close()
	if err != nil {
		return result, fmt.Errorf("read baseline: %w", err)
	}
	const evidenceQuery = "SELECT s.data,r.attrs,r.raw FROM signals s JOIN resources r ON r.resource_id=s.resource_id WHERE s.run_id=? AND s.kind IN ('span','event') ORDER BY s.ts_ns,s.evidence_id"
	load := func(runID string) ([]model.Signal, error) {
		if bounded {
			return readStructure(ctx, tx, runID, ViewerSignalLimit)
		}
		return readSignals(ctx, tx, evidenceQuery, runID)
	}
	result.Failing.Signals, err = load(id)
	if err != nil {
		return result, err
	}
	total := len(result.Failing.Signals)
	for i := range result.Baseline {
		b := &result.Baseline[i]
		b.Signals, err = load(b.Run.ID)
		if err != nil {
			return result, err
		}
		total += len(b.Signals)
		if bounded && total > 50000 {
			return result, EvidenceLimit
		}
	}
	if err = tx.Commit(); err != nil {
		return result, fmt.Errorf("commit comparison snapshot: %w", err)
	}
	return result, nil
}

type storedDiff struct {
	id, runID, spec, results string
	createdAt                int64
}

// SaveDiff encodes before enqueueing so cancellation cannot leave mutable caller
// slices in the writer queue. Accepted writes drain even if the caller leaves.
func (s *Store) SaveDiff(ctx context.Context, result model.DiffResult) error {
	if result.ID == "" || result.FailingRunID == "" {
		return fmt.Errorf("diff identity is required")
	}
	spec, err := json.Marshal(result.BaselineSpec)
	if err != nil {
		return fmt.Errorf("encode baseline spec: %w", err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode diff result: %w", err)
	}
	stored := storedDiff{id: result.ID, runID: result.FailingRunID, spec: string(spec), results: string(data), createdAt: result.CreatedAt.UnixNano()}
	return s.submit(ctx, job{result: &stored, done: make(chan error, 1)})
}

func (s *Store) writeDiff(ctx context.Context, result storedDiff) error {
	tx, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin diff write: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	written, err := tx.ExecContext(ctx, `INSERT INTO diff_results(diff_id,failing_run_id,baseline_spec,created_at,results)
	 SELECT ?,?,?,?,? FROM runs WHERE run_id=? AND closed=1 AND outcome='fail'
	 ON CONFLICT(diff_id) DO UPDATE SET baseline_spec=excluded.baseline_spec,created_at=excluded.created_at,results=excluded.results`, result.id, result.runID, result.spec, result.createdAt, result.results, result.runID)
	if err != nil {
		return fmt.Errorf("persist diff: %w", err)
	}
	count, err := written.RowsAffected()
	if err != nil {
		return fmt.Errorf("count diff writes: %w", err)
	}
	if count != 1 {
		return fmt.Errorf("failing run no longer exists or is not closed/failing")
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit diff: %w", err)
	}
	return nil
}
