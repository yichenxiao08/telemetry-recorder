package store

import (
	"context"
	"strings"
	"testing"
)

func TestReportReadBoundsAndQueryPlan(t *testing.T) {
	s := testStore(t, 0)
	addRun(t, s, traceID(1), 1, nil)
	for _, q := range []ReportRequest{{From: 0, To: 1000, Limit: 0}, {From: 1000, To: 0, Limit: 10}, {From: 0, To: 1000, Limit: 1, Runs: []string{traceID(1)}}} {
		if _, err := s.ReportRead(context.Background(), q); err == nil {
			t.Fatal("invalid bound/cap accepted", q)
		}
	}
	q := ReportRequest{From: 0, To: 1000, Limit: 100, Runs: []string{traceID(1)}}
	snap, err := s.ReportRead(context.Background(), q)
	if err != nil || len(snap.Signals) == 0 {
		t.Fatal(snap, err)
	}
	q.From = 1000
	q.To = 2000
	if _, err = s.ReportRead(context.Background(), q); err == nil {
		t.Fatal("run outside time range accepted")
	}
	rows, err := s.reader.QueryContext(context.Background(), "EXPLAIN QUERY PLAN "+reportSignalsSQL, traceID(1), 0, 1000, 101)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var plan string
	for rows.Next() {
		var a, b, c int
		var detail string
		if err = rows.Scan(&a, &b, &c, &detail); err != nil {
			t.Fatal(err)
		}
		plan += detail
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plan, "SCAN signals") || !strings.Contains(plan, "signals_run") || !strings.Contains(plan, "ts_ns>?") || !strings.Contains(plan, "ts_ns<?") {
		t.Fatal(plan)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	q.From = 0
	q.To = 1000
	if _, err = s.ReportRead(ctx, q); err == nil {
		t.Fatal("cancelled bounded read succeeded")
	}
}
