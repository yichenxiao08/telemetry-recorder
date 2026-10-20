package store

import (
	"context"
	"database/sql"
	"encoding/json"
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

// Unary + removes the scope column's TEXT affinity so SQLite can seek the
// existing JSON expression indexes. Both matched values are still strings:
// match_value is TEXT and each shared-key branch requires json_type='text'.
const contextResourcesQuery = `SELECT c.run_id,a.resource_id,a.attrs,a.raw,c.rule,c.matched_resource_id,c.start_ns,c.end_ns
 FROM context_scopes c CROSS JOIN resources a ON a.resource_id=c.match_value
 WHERE c.run_id IN (SELECT value FROM json_each(?)) AND c.rule='exact_resource' AND c.start_ns<=? AND c.end_ns>=?
 UNION ALL
 SELECT c.run_id,a.resource_id,a.attrs,a.raw,c.rule,c.matched_resource_id,c.start_ns,c.end_ns
 FROM context_scopes c CROSS JOIN resources a ON json_extract(a.attrs,'$."host.name"')=+c.match_value
 WHERE c.run_id IN (SELECT value FROM json_each(?)) AND c.rule='host.name' AND json_type(a.attrs,'$."host.name"')='text' AND c.start_ns<=? AND c.end_ns>=?
 UNION ALL
 SELECT c.run_id,a.resource_id,a.attrs,a.raw,c.rule,c.matched_resource_id,c.start_ns,c.end_ns
 FROM context_scopes c CROSS JOIN resources a ON json_extract(a.attrs,'$."service.instance.id"')=+c.match_value
 WHERE c.run_id IN (SELECT value FROM json_each(?)) AND c.rule='service.instance.id' AND json_type(a.attrs,'$."service.instance.id"')='text' AND c.start_ns<=? AND c.end_ns>=?
 ORDER BY 1,2,5,6 LIMIT 50001`

const contextQuery = `SELECT s.data,s.ts_ns FROM signals s INDEXED BY signals_context
 WHERE s.resource_id=? AND s.run_id='' AND s.kind IN ('log','metric')
 AND s.ts_ns>=? AND s.ts_ns<=? ORDER BY s.ts_ns,s.evidence_id LIMIT ?`

type contextBounds struct{ from, to int64 }
type contextRule struct {
	match    model.ContextMatch
	from, to int64
}
type contextResource struct {
	resource  model.Resource
	rules     map[string][]contextRule
	intervals []contextBounds
}

func mergeContextBounds(bounds []contextBounds) []contextBounds {
	sort.Slice(bounds, func(i, j int) bool {
		if bounds[i].from == bounds[j].from {
			return bounds[i].to < bounds[j].to
		}
		return bounds[i].from < bounds[j].from
	})
	out := []contextBounds{}
	for _, b := range bounds {
		if len(out) == 0 || b.from > out[len(out)-1].to {
			out = append(out, b)
		} else {
			out[len(out)-1].to = max(out[len(out)-1].to, b.to)
		}
	}
	return out
}

func readContext(ctx context.Context, tx *sql.Tx, id string, from, to int64, limit int) ([]model.Signal, error) {
	result, err := readContextWindows(ctx, tx, map[string]contextBounds{id: {from, to}}, limit, limit)
	if err != nil {
		return nil, err
	}
	return result[id], nil
}

// Cache only inside this read transaction. Overlapping windows read and decode
// each physical record once; each run still receives its exact interval/rule
// membership. Resource metadata/raw payloads are shared read-only snapshots.
// Disjoint intervals remain disjoint, so sparse baselines never fetch the gaps.
func readContextWindows(ctx context.Context, tx *sql.Tx, windows map[string]contextBounds, perRun, totalLimit int) (map[string][]model.Signal, error) {
	if len(windows) == 0 || len(windows) > 201 || perRun < 1 || perRun > 5000 || totalLimit < 0 || totalLimit > 50000 {
		return nil, fmt.Errorf("invalid context window bounds or limits")
	}
	ids := []string{}
	result := map[string][]model.Signal{}
	var first, last int64
	for id, b := range windows {
		if b.from > b.to {
			return nil, fmt.Errorf("invalid context time range")
		}
		if len(ids) == 0 {
			first, last = b.from, b.to
		}
		first = min(first, b.from)
		last = max(last, b.to)
		ids = append(ids, id)
		result[id] = []model.Signal{}
	}
	sort.Strings(ids)
	encoded, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	input := string(encoded)
	rows, err := tx.QueryContext(ctx, contextResourcesQuery, input, last, first, input, last, first, input, last, first)
	if err != nil {
		return nil, fmt.Errorf("resolve context resources: %w", err)
	}
	resources := map[string]*contextResource{}
	metadataCount := 0
	for rows.Next() {
		var runID, resourceID, attrs, raw string
		var rule contextRule
		if err = rows.Scan(&runID, &resourceID, &attrs, &raw, &rule.match.Rule, &rule.match.ResourceID, &rule.from, &rule.to); err != nil {
			break
		}
		metadataCount++
		if metadataCount > 50000 {
			err = EvidenceLimit
			break
		}
		window, ok := windows[runID]
		if !ok {
			continue
		}
		rule.from = max(rule.from, window.from)
		rule.to = min(rule.to, window.to)
		if rule.from > rule.to {
			continue
		}
		resource, ok := resources[resourceID]
		if !ok {
			resource = &contextResource{resource: model.Resource{ID: resourceID, Raw: json.RawMessage(raw)}, rules: map[string][]contextRule{}}
			if err = json.Unmarshal([]byte(attrs), &resource.resource.Attrs); err != nil {
				break
			}
			resources[resourceID] = resource
		}
		resource.rules[runID] = append(resource.rules[runID], rule)
		resource.intervals = append(resource.intervals, contextBounds{rule.from, rule.to})
	}
	if err == nil {
		err = rows.Err()
	}
	closeErr := rows.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	keys := []string{}
	for key := range resources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	scanned, total := 0, 0
	matches := map[string]int{}
	for _, key := range keys {
		resource := resources[key]
		for _, interval := range mergeContextBounds(resource.intervals) {
			records, e := tx.QueryContext(ctx, contextQuery, key, interval.from, interval.to, 50001-scanned)
			if e != nil {
				return nil, e
			}
			for records.Next() {
				var data string
				var timestamp int64
				var signal model.Signal
				if e = records.Scan(&data, &timestamp); e != nil {
					break
				}
				scanned++
				if scanned > 50000 {
					e = EvidenceLimit
					break
				}
				if e = json.Unmarshal([]byte(data), &signal); e != nil {
					break
				}
				signal.Resource = resource.resource
				for runID, rules := range resource.rules {
					record := signal
					record.ContextMatches = nil
					for _, rule := range rules {
						if timestamp >= rule.from && timestamp <= rule.to {
							record.ContextMatches = append(record.ContextMatches, rule.match)
						}
					}
					if len(record.ContextMatches) == 0 {
						continue
					}
					sort.Slice(record.ContextMatches, func(i, j int) bool {
						a, b := record.ContextMatches[i], record.ContextMatches[j]
						if a.Rule == b.Rule {
							return a.ResourceID < b.ResourceID
						}
						return a.Rule < b.Rule
					})
					matches[runID] += len(record.ContextMatches)
					total++
					if matches[runID] > 50000 || total > totalLimit || len(result[runID]) >= perRun {
						e = EvidenceLimit
						break
					}
					result[runID] = append(result[runID], record)
				}
				if e != nil {
					break
				}
			}
			if e == nil {
				e = records.Err()
			}
			closeErr = records.Close()
			if e != nil {
				return nil, e
			}
			if closeErr != nil {
				return nil, closeErr
			}
		}
	}
	for _, signals := range result {
		sort.Slice(signals, func(i, j int) bool {
			if signals[i].TS == signals[j].TS {
				return signals[i].ID < signals[j].ID
			}
			return signals[i].TS < signals[j].TS
		})
	}
	return result, nil
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

func (r *Reader) ContextRange(ctx context.Context, id string) (int64, int64, error) {
	var from, to sql.NullInt64
	err := r.reader.QueryRowContext(ctx, "SELECT min(start_ns),max(end_ns) FROM context_scopes WHERE run_id=? LIMIT 1", id).Scan(&from, &to)
	return from.Int64, to.Int64, err
}
