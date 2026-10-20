// Package telemetry defines bounded queries over all stored signals.
package telemetry

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/model"
)

type Cursor struct {
	Inclusive bool   `json:"inclusive,omitempty"`
	TS        int64  `json:"ts"`
	ID        string `json:"id"`
}
type Filter struct {
	NoService                                                                       bool
	Kind, Service, TraceID, Name, Text, Template, Status, Attribute, AttributeValue string
	From, To                                                                        int64
	Limit                                                                           int
	SeverityMin, SeverityMax                                                        *int
	DurationMin, DurationMax                                                        *float64
	After                                                                           *Cursor
}
type Page struct {
	Signals []model.Signal
	Next    *Cursor
}
type TemplateCount struct {
	Template, Severity string
	Count              int
}
type ServiceStats struct {
	Service, Route                 string
	Requests, Errors               int
	Rate, ErrorRate, P50, P95, P99 float64
}

func (f Filter) Validate() error {
	if f.From < 0 || f.From >= f.To || time.Unix(0, f.To).Sub(time.Unix(0, f.From)) > 31*24*time.Hour || f.Limit < 1 || f.Limit > 1000 {
		return fmt.Errorf("telemetry requires an explicit time range (max 31 days) and limit 1..1000")
	}
	if f.NoService && (f.Service != "" || f.TraceID != "") {
		return fmt.Errorf("no-service partition cannot be combined with service or trace")
	}
	if f.Service == "" && f.TraceID == "" && !f.NoService {
		return fmt.Errorf("select a service or trace ID for an indexed partition")
	}
	if f.Kind != "span" && f.Kind != "log" && f.Kind != "metric" {
		return fmt.Errorf("kind must be span, log or metric")
	}
	if f.TraceID != "" && f.Kind == "metric" {
		return fmt.Errorf("metric points have no trace ID index")
	}
	if f.DurationMin != nil && (*f.DurationMin < 0 || *f.DurationMin > 1e12) || f.DurationMax != nil && (*f.DurationMax < 0 || *f.DurationMax > 1e12) {
		return fmt.Errorf("invalid duration")
	}
	for _, p := range []*float64{f.DurationMin, f.DurationMax} {
		if p != nil && (math.IsNaN(*p) || math.IsInf(*p, 0)) {
			return fmt.Errorf("duration must be finite")
		}
	}
	if f.DurationMin != nil && f.DurationMax != nil && *f.DurationMin > *f.DurationMax {
		return fmt.Errorf("reversed duration range")
	}
	for _, p := range []*int{f.SeverityMin, f.SeverityMax} {
		if p != nil && (*p < 0 || *p > 24) {
			return fmt.Errorf("severity must be 0..24")
		}
	}
	if f.SeverityMin != nil && f.SeverityMax != nil && *f.SeverityMin > *f.SeverityMax {
		return fmt.Errorf("reversed severity range")
	}
	for _, s := range []string{f.Service, f.TraceID, f.Name, f.Text, f.Template, f.Status, f.Attribute, f.AttributeValue} {
		if len(s) > 2048 {
			return fmt.Errorf("filter too long")
		}
	}
	return nil
}

// All dynamic values are parameters. INDEXED BY fails visibly if a migration
// removes an access path; plan tests also check that time is a search bound.
func query(f Filter, projection, tail string, limit int) (string, []any, error) {
	if err := f.Validate(); err != nil {
		return "", nil, err
	}
	base := ` FROM resources r INDEXED BY resources_service JOIN signals s INDEXED BY signals_resource_kind_time ON s.resource_id=r.resource_id WHERE json_extract(r.attrs,'$."service.name"')=?`
	args := []any{f.Service}
	if f.NoService {
		base = strings.TrimSuffix(base, "=?") + " IS NULL"
		args = nil
	}
	if f.Kind == "log" {
		base = strings.Replace(base, "signals_resource_kind_time", "signals_log_resource_time", 1)
	}
	if f.TraceID != "" {
		base = ` FROM signals s INDEXED BY signals_trace_time JOIN resources r ON r.resource_id=s.resource_id WHERE s.kind IN ('span','log') AND json_extract(s.data,'$.trace_id')=?`
		args = []any{f.TraceID}
		if f.Service != "" {
			base += ` AND json_extract(r.attrs,'$."service.name"')=?`
			args = append(args, f.Service)
		}
	}
	base += ` AND s.kind=? AND s.ts_ns>=? AND s.ts_ns<?`
	args = append(args, f.Kind, f.From, f.To)
	// Partial-index predicates are literal so SQLite can prove applicability.
	if f.Kind == "log" {
		base += ` AND s.kind='log'`
	}
	if f.TraceID == "" && f.Kind != "log" {
		base += ` AND s.kind IN ('span','metric')`
	}
	for _, p := range []struct{ column, value string }{{"name", f.Name}, {"status", f.Status}, {"template", f.Template}} {
		if p.value != "" {
			base += ` AND json_extract(s.data,'$.` + p.column + `')=?`
			args = append(args, p.value)
		}
	}
	if f.Text != "" {
		base += ` AND instr(coalesce(json_extract(s.data,'$.body'),''),?)>0`
		args = append(args, f.Text)
	}
	if f.Attribute != "" {
		base += ` AND (EXISTS(SELECT 1 FROM json_each(s.data,'$.attrs') a WHERE a.key=? AND CAST(a.value AS TEXT)=?) OR EXISTS(SELECT 1 FROM json_each(r.attrs) a WHERE a.key=? AND CAST(a.value AS TEXT)=?))`
		args = append(args, f.Attribute, f.AttributeValue, f.Attribute, f.AttributeValue)
	}
	for _, p := range []struct {
		op string
		v  *int
	}{{">=", f.SeverityMin}, {"<=", f.SeverityMax}} {
		if p.v != nil {
			base += ` AND (` + severitySQL() + `)` + p.op + `?`
			args = append(args, *p.v)
		}
	}
	for _, p := range []struct {
		op string
		v  *float64
	}{{">=", f.DurationMin}, {"<=", f.DurationMax}} {
		if p.v != nil {
			base += ` AND (s.end_ns-s.ts_ns)/1000000.0` + p.op + `?`
			args = append(args, *p.v)
		}
	}
	if f.After != nil {
		op := ">"
		if f.After.Inclusive {
			op = ">="
		}
		base += ` AND (s.ts_ns>? OR (s.ts_ns=? AND s.evidence_id` + op + `?))`
		args = append(args, f.After.TS, f.After.TS, f.After.ID)
	}
	args = append(args, limit)
	return "SELECT " + projection + base + tail + " LIMIT ?", args, nil
}
func Query(f Filter) (string, []any, error) {
	return query(f, "s.data,r.attrs,r.raw", " ORDER BY s.ts_ns,s.evidence_id", f.Limit+1)
}
func OverviewQuery(f Filter) (string, []any, error) {
	if f.Kind != "span" || f.Service == "" || f.After != nil {
		return "", nil, fmt.Errorf("overview requires spans")
	}
	return query(f, "s.data,r.attrs,r.raw", ` AND json_extract(s.data,'$.attrs."http.route"') IS NOT NULL ORDER BY s.ts_ns,s.evidence_id`, 50001)
}
func TemplateQuery(f Filter) (string, []any, error) {
	if f.Kind != "log" || f.After != nil {
		return "", nil, fmt.Errorf("templates require logs")
	}
	return query(f, `coalesce(json_extract(s.data,'$.template'),''),`+severitySQL()+`,count(*)`, " GROUP BY 1,2 ORDER BY count(*) DESC,1,2", f.Limit)
}
