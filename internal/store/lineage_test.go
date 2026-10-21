package store

import (
	"context"
	"strings"
	"testing"

	"github.com/signalscout/signalscout/internal/lineage"
	"github.com/signalscout/signalscout/internal/model"
)

func TestTraceReadBoundsAndPlans(t *testing.T) {
	s := testStore(t, 0)
	ctx := context.Background()
	root := fixture("span:trace", "span", "r", "resource", 10, 20)
	event := fixture("event:trace", "event", "r", "resource", 11, 11)
	event.Attrs = map[string]any{"signalscout.record.kind": "decision", "signalscout.record.id": "decision:r/a", "signalscout.record.json": `{"version":1,"id":"decision:r/a","run_id":"r","step":{"id":"choose"},"inputs":[],"output":{"ref":"value:r/a.return@1","capture":"off"}}`}
	if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{root, event}}); err != nil {
		t.Fatal(err)
	}
	r := &Reader{reader: s.reader}
	for _, target := range []string{"value:r/a.return@1", "event:trace"} {
		data, err := r.TraceRead(ctx, TraceReadRequest{Targets: []string{target}, From: 10, To: 20, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		report := lineage.Walk(data.Signals, target, lineage.Options{Depth: 8, Nodes: 8}, nil)
		if len(report.Nodes) != 1 {
			t.Fatalf("%+v", report)
		}
	}
	data, err := r.TraceRead(ctx, TraceReadRequest{Targets: []string{"value:r/a.return@1"}, From: 12, To: 20, Limit: 10})
	if err != nil || len(data.Signals) != 0 {
		t.Fatalf("range ignored: %+v %v", data, err)
	}
	if _, err = r.TraceRead(ctx, TraceReadRequest{Targets: []string{"opaque"}, From: 10, To: 20, Limit: 10}); err == nil {
		t.Fatal("unbounded opaque lookup")
	}
	for _, q := range []struct {
		sql   string
		args  []any
		index string
	}{
		{traceRunSQL, []any{"r", 10, 20, 11}, "signals_run"},
		{traceNativeSQL, []any{"event:trace", 10, 20}, "sqlite_autoindex_signals_1"},
	} {
		rows, err := s.reader.QueryContext(ctx, "EXPLAIN QUERY PLAN "+q.sql, q.args...)
		if err != nil {
			t.Fatal(err)
		}
		var plan strings.Builder
		for rows.Next() {
			var a, b, c int
			var detail string
			if err = rows.Scan(&a, &b, &c, &detail); err != nil {
				t.Fatal(err)
			}
			plan.WriteString(detail)
		}
		if err = rows.Err(); err != nil {
			t.Fatal(err)
		}
		_ = rows.Close()
		if !strings.Contains(plan.String(), "SEARCH") || !strings.Contains(plan.String(), q.index) || strings.Contains(plan.String(), "SCAN signals") {
			t.Fatal(plan.String())
		}
	}
}

func TestTraceReadCrossRunAndTruncation(t *testing.T) {
	s := testStore(t, 0)
	ctx := context.Background()
	makeEvent := func(id, run, body string) model.Signal {
		e := fixture(id, "event", run, "resource", 11, 11)
		e.Attrs = map[string]any{"signalscout.record.kind": "decision", "signalscout.record.json": body}
		return e
	}
	a := makeEvent("event:a", "a", `{"version":1,"id":"decision:a/1","run_id":"a","step":{"id":"s"},"inputs":[{"name":"x","ref":"value:b/1"}],"output":{"ref":"value:a/1"}}`)
	b := makeEvent("event:b", "b", `{"version":1,"id":"decision:b/1","run_id":"b","step":{"id":"s"},"inputs":[],"output":{"ref":"value:b/1"}}`)
	extra := fixture("event:extra", "event", "a", "resource", 12, 12)
	if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{a, b, extra}}); err != nil {
		t.Fatal(err)
	}
	reader := &Reader{reader: s.reader}
	data, err := reader.TraceRead(ctx, TraceReadRequest{Targets: []string{"value:a/1"}, From: 10, To: 20, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Scope.Runs) != 2 || data.Scope.RecordsRead != 3 || data.Scope.Truncated {
		t.Fatal(data.Scope)
	}
	data, err = reader.TraceRead(ctx, TraceReadRequest{Targets: []string{"value:a/1"}, From: 10, To: 20, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !data.Scope.Truncated || len(data.Signals) != 1 {
		t.Fatal(data.Scope)
	}
	data, err = reader.TraceRead(ctx, TraceReadRequest{Targets: []string{"event:missing"}, From: 10, To: 20, Limit: 10})
	if err != nil || len(data.Signals) != 0 {
		t.Fatal(data, err)
	}
}
