package store

import (
	"strings"
	"testing"
)

// These are SQL performance preflights, not an M4b query API or viewer.
func TestBoundedLookupPlansUseTraceAndLogIndexes(t *testing.T) {
	s := testStore(t, 0)
	for _, tc := range []struct {
		name, query string
		args        []any
		indexes     []string
	}{
		{"trace", `SELECT evidence_id,data FROM signals WHERE kind IN ('span','log') AND json_extract(data,'$.trace_id')=? AND ts_ns>=? AND ts_ns<? ORDER BY ts_ns,evidence_id LIMIT ?`, []any{"trace", 0, 1000, 100}, []string{"signals_trace_time"}},
		{"templates", `SELECT json_extract(s.data,'$.template') AS template,count(*) FROM signals s JOIN resources r ON r.resource_id=s.resource_id WHERE s.kind='log' AND json_extract(r.attrs,'$."service.name"')=? AND s.ts_ns>=? AND s.ts_ns<? GROUP BY template ORDER BY count(*) DESC,template LIMIT ?`, []any{"shop", 0, 1000, 100}, []string{"resources_service", "signals_log_resource_time"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := s.reader.Query("EXPLAIN QUERY PLAN "+tc.query, tc.args...)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = rows.Close() }()
			var plan strings.Builder
			for rows.Next() {
				var a, b, c int
				var detail string
				if err = rows.Scan(&a, &b, &c, &detail); err != nil {
					t.Fatal(err)
				}
				plan.WriteString(detail)
				plan.WriteByte('\n')
			}
			if err = rows.Err(); err != nil {
				t.Fatal(err)
			}
			for _, index := range tc.indexes {
				if !strings.Contains(plan.String(), index) {
					t.Fatalf("missing %s: %s", index, plan.String())
				}
			}
		})
	}
}
