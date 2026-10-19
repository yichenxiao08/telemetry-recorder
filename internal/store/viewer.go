package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/runs"
)

const ViewerSignalLimit = 5000

const EvidenceLimit queueError = "run or comparison exceeds the viewer evidence limit"

type RunCursor struct {
	StartedAt int64  `json:"started_at"`
	ID        string `json:"id"`
}
type RunFilter struct {
	Workload, Outcome string
	From, To          int64
	Limit             int
	After             *RunCursor
}
type RunPage struct {
	Runs []model.Run
	Next *RunCursor
}

func (r *Reader) RunPage(ctx context.Context, f RunFilter) (RunPage, error) {
	result := RunPage{Runs: []model.Run{}}
	if f.Limit < 1 || f.Limit > 100 || f.From >= f.To || time.Unix(0, f.To).Sub(time.Unix(0, f.From)) > 31*24*time.Hour {
		return result, fmt.Errorf("list requires 1..100 rows and a time range of at most 31 days")
	}
	if f.Outcome != "" && f.Outcome != "pass" && f.Outcome != "fail" && f.Outcome != "degraded" {
		return result, fmt.Errorf("invalid outcome")
	}
	query := "SELECT state FROM runs WHERE started_at>=? AND started_at<?"
	args := []any{f.From, f.To}
	if f.Workload != "" {
		query += " AND workload=?"
		args = append(args, f.Workload)
	}
	if f.Outcome != "" {
		query += " AND outcome=?"
		args = append(args, f.Outcome)
	}
	if f.After != nil {
		query += " AND (started_at<? OR (started_at=? AND run_id>?))"
		args = append(args, f.After.StartedAt, f.After.StartedAt, f.After.ID)
	}
	query += " ORDER BY started_at DESC,run_id LIMIT ?"
	args = append(args, f.Limit+1)
	rows, err := r.reader.QueryContext(ctx, query, args...)
	if err != nil {
		return result, fmt.Errorf("page runs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var data string
		var state runs.State
		if err = rows.Scan(&data); err != nil {
			return result, fmt.Errorf("read page: %w", err)
		}
		if err = json.Unmarshal([]byte(data), &state); err != nil {
			return result, fmt.Errorf("decode page: %w", err)
		}
		result.Runs = append(result.Runs, state.Run)
	}
	if err = rows.Err(); err != nil {
		return result, fmt.Errorf("iterate page: %w", err)
	}
	if len(result.Runs) > f.Limit {
		result.Runs = result.Runs[:f.Limit]
		last := result.Runs[len(result.Runs)-1]
		result.Next = &RunCursor{StartedAt: last.StartedAt, ID: last.ID}
	}
	return result, nil
}

func readStructure(ctx context.Context, tx *sql.Tx, id string, limit int) ([]model.Signal, error) {
	signals, err := readSignals(ctx, tx, "SELECT s.data,r.attrs,r.raw FROM signals s JOIN resources r ON r.resource_id=s.resource_id WHERE s.run_id=? AND s.kind IN ('span','event') ORDER BY s.ts_ns,s.evidence_id LIMIT ?", id, limit+1)
	if err == nil && len(signals) > limit {
		return nil, EvidenceLimit
	}
	return signals, err
}

func (r *Reader) RunStructure(ctx context.Context, id string, limit int) (model.RunDetail, error) {
	result := model.RunDetail{}
	if limit < 1 || limit > ViewerSignalLimit {
		return result, fmt.Errorf("invalid structure limit")
	}
	tx, err := r.reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return result, fmt.Errorf("begin structure read: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	result.Run, err = readRun(ctx, tx, id)
	if err != nil {
		return result, err
	}
	result.Signals, err = readStructure(ctx, tx, id, limit)
	if err != nil {
		return result, err
	}
	if err = tx.Commit(); err != nil {
		return result, fmt.Errorf("commit structure read: %w", err)
	}
	return result, nil
}

// PhaseStart suggests a cutoff only from an explicit demo phase fingerprint.
// It does not infer a universal failure phase from outcomes or error timestamps.
func (r *Reader) PhaseStart(ctx context.Context, run model.Run) (string, error) {
	if run.Env["signalscout.demo.phase"] != "failure" {
		return "", nil
	}
	config, err := json.Marshal(run.Config)
	if err != nil {
		return "", fmt.Errorf("phase config: %w", err)
	}
	var revision any
	if text, ok := run.Env["vcs.revision"].(string); ok {
		revision = text
	}
	var started int64
	err = r.reader.QueryRowContext(ctx, `SELECT started_at FROM runs WHERE workload=? AND started_at<=? AND config=? AND json_extract(env,'$."signalscout.demo.phase"')='failure' AND json_extract(env,'$."vcs.revision"') IS ? ORDER BY started_at,run_id LIMIT 1`, run.Workload, run.StartedAt, string(config), revision).Scan(&started)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("suggest phase cutoff: %w", err)
	}
	return time.Unix(0, started).UTC().Format(time.RFC3339Nano), nil
}
