package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/stats"
	"github.com/signalscout/signalscout/internal/telemetry"
)

func (r *Reader) SearchTelemetry(ctx context.Context, f telemetry.Filter) (telemetry.Page, error) {
	page := telemetry.Page{}
	q, args, err := telemetry.Query(f)
	if err != nil {
		return page, err
	}
	tx, err := r.reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return page, err
	}
	defer func() { _ = tx.Rollback() }()
	page.Signals, err = readSignals(ctx, tx, q, args...)
	if err != nil {
		return page, err
	}
	if len(page.Signals) > f.Limit {
		page.Signals = page.Signals[:f.Limit]
		s := page.Signals[len(page.Signals)-1]
		page.Next = &telemetry.Cursor{TS: s.TS, ID: s.ID}
	}
	return page, tx.Commit()
}
func (r *Reader) LogTemplates(ctx context.Context, f telemetry.Filter) ([]telemetry.TemplateCount, error) {
	q, args, err := telemetry.TemplateQuery(f)
	if err != nil {
		return nil, err
	}
	rows, err := r.reader.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query templates: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []telemetry.TemplateCount{}
	for rows.Next() {
		var x telemetry.TemplateCount
		if err = rows.Scan(&x.Template, &x.Severity, &x.Count); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Reader) ServiceOverview(ctx context.Context, f telemetry.Filter) ([]telemetry.ServiceStats, error) {
	q, args, err := telemetry.OverviewQuery(f)
	if err != nil {
		return nil, err
	}
	tx, err := r.reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	signals, err := readSignals(ctx, tx, q, args...)
	if err != nil {
		return nil, err
	}
	if len(signals) > 50000 {
		return nil, EvidenceLimit
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	values := map[string][]float64{}
	errors := map[string]int{}
	for _, s := range signals {
		route := fmt.Sprint(s.Attrs["http.route"])
		values[route] = append(values[route], float64(s.End-s.TS)/1e6)
		status := s.Attrs["http.response.status_code"]
		if status == nil {
			status = s.Attrs["http.status_code"]
		}
		code, _ := status.(float64)
		if s.Status == "error" || code >= 500 {
			errors[route]++
		}
	}
	keys := []string{}
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) > f.Limit {
		return nil, EvidenceLimit
	}
	out := []telemetry.ServiceStats{}
	for _, key := range keys {
		v := values[key]
		n := len(v)
		out = append(out, telemetry.ServiceStats{Service: f.Service, Route: key, Requests: n, Errors: errors[key], Rate: float64(n) / (float64(f.To-f.From) / 1e9), ErrorRate: float64(errors[key]) / float64(n), P50: stats.Quantile(v, .5), P95: stats.Quantile(v, .95), P99: stats.Quantile(v, .99)})
	}
	return out, nil
}

const contextQuery = `SELECT s.data,r.attrs,r.raw FROM signals s JOIN resources r ON r.resource_id=s.resource_id WHERE s.evidence_id IN (SELECT c.evidence_id FROM context_matches c JOIN signals e ON e.evidence_id=c.evidence_id WHERE c.run_id=? AND e.ts_ns>=? AND e.ts_ns<=?) AND s.ts_ns>=? AND s.ts_ns<=? ORDER BY s.ts_ns,s.evidence_id LIMIT ?`

func readContext(ctx context.Context, tx *sql.Tx, id string, from, to int64, limit int) ([]model.Signal, error) {
	if from > to || limit < 1 || limit > 5000 {
		return nil, fmt.Errorf("context requires valid bounds and limit 1..5000")
	}
	signals, err := readSignals(ctx, tx, contextQuery, id, from, to, from, to, limit+1)
	if err != nil {
		return nil, err
	}
	if len(signals) > limit {
		return nil, EvidenceLimit
	}
	// Fetch provenance once for the bounded run/window, not once per record.
	rows, err := tx.QueryContext(ctx, `SELECT c.evidence_id,c.rule,c.matched_resource_id FROM context_matches c JOIN signals e ON e.evidence_id=c.evidence_id WHERE c.run_id=? AND e.ts_ns>=? AND e.ts_ns<=? ORDER BY c.evidence_id,c.rule,c.matched_resource_id LIMIT 50001`, id, from, to)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	byID := map[string][]model.ContextMatch{}
	count := 0
	for rows.Next() {
		var evidence string
		var match model.ContextMatch
		if err = rows.Scan(&evidence, &match.Rule, &match.ResourceID); err != nil {
			return nil, err
		}
		count++
		if count > 50000 {
			return nil, EvidenceLimit
		}
		byID[evidence] = append(byID[evidence], match)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	for i := range signals {
		signals[i].ContextMatches = byID[signals[i].ID]
	}
	return signals, nil
}
func (r *Reader) RunContext(ctx context.Context, id string, from, to int64, limit int) ([]model.Signal, error) {
	tx, err := r.reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	out, err := readContext(ctx, tx, id, from, to, limit)
	if err != nil {
		return nil, err
	}
	return out, tx.Commit()
}
