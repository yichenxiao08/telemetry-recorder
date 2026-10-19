package store

import (
	"context"
	"strings"
	"testing"
)

func TestContextCountPlansUseEveryIntervalTimeRange(t *testing.T) {
	s := testStore(t, 0)
	rows, err := s.reader.QueryContext(context.Background(), "EXPLAIN QUERY PLAN SELECT count(*) FROM context_matches")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	bounded := 0
	var plan []string
	for rows.Next() {
		var a, b, c int
		var detail string
		if err = rows.Scan(&a, &b, &c, &detail); err != nil {
			t.Fatal(err)
		}
		plan = append(plan, detail)
		if strings.Contains(detail, "SEARCH s USING INDEX signals_context (resource_id=? AND run_id=? AND ts_ns>") {
			bounded++
		}
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	if bounded != 3 {
		t.Fatalf("context must use time ranges in all three rules; got %d: %v", bounded, plan)
	}
}

func TestWriterDurabilitySettings(t *testing.T) {
	s := testStore(t, 0)
	var mode string
	if err := s.writer.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
		t.Fatal("WAL changed", mode, err)
	}
	for pragma, want := range map[string]int{"synchronous": 1, "foreign_keys": 1, "wal_autocheckpoint": 1000, "busy_timeout": 5000} {
		var got int
		if err := s.writer.QueryRow("PRAGMA " + pragma).Scan(&got); err != nil || got != want {
			t.Fatalf("%s=%d want %d: %v", pragma, got, want, err)
		}
	}
}
