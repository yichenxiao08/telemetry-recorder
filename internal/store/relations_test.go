package store

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/signalscout/signalscout/internal/model"
)

func TestRelationsSeeMinorityCallsAndPinViolatingEvidence(t *testing.T) {
	s := testStore(t, 0)
	p := toyProfile()
	zero := 0.0
	p.Relations = &model.RelationPolicy{Version: 1, Rules: []model.RelationRule{{ID: "equal", Unit: "count", Tolerance: &zero, Assert: model.Expression{Op: "eq", Args: []model.Expression{{Op: "field", Field: "inputs.x"}, {Op: "number", Number: &zero}}}}}}
	for n := 1; n <= 4; n++ {
		addRun(t, s, traceID(n), 3, nil)
	}
	// All earlier x=1 decisions violate, even though the final record satisfies.
	for n := 1; n <= 4; n++ {
		e := fixture(fmt.Sprintf("event:final%d", n), "event", traceID(n), "resource", 190, 190)
		e.Attrs = map[string]any{"signalscout.record.kind": "decision", "signalscout.record.json": `{"version":1,"profile":{"id":"synthetic","version":"1"},"step":{"id":"toy"},"inputs":[{"name":"x","capture":"allowed","inline":0}],"sampling":{"rule":{"every_nth":2},"call_index":7}}`}
		if err := s.Submit(context.Background(), model.Batch{Signals: []model.Signal{e}}); err != nil {
			t.Fatal(err)
		}
	}
	b := model.CohortSelection{From: 0, To: 1000, Limit: 4, Relation: "equal"}
	r, err := s.Relations(context.Background(), p, b)
	if err != nil {
		t.Fatal(err)
	}
	cnt := r.Totals["equal"]
	if cnt.Total != 16 || cnt.Violated != 12 || cnt.Satisfied != 4 || cnt.SampledRecords != 4 {
		t.Fatalf("%+v", cnt)
	}
	c, err := s.SelectCohort(context.Background(), "violations", p, b)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.RunIDs) != 4 || c.Relations == nil {
		t.Fatal(c)
	}
	for _, id := range c.DecisionIDs {
		if strings.Contains(id, "final") {
			t.Fatal("pinned satisfied final decision", id)
		}
	}
	if err = s.SaveCohort(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	saved, err := s.Cohort(context.Background(), c.ID)
	if err != nil || saved.Relations.Totals["equal"].Violated != 12 {
		t.Fatal(saved, err)
	}
}

func TestExplicitRecordCapRejectsWholeSelectionAndHigherDefaultReadsPast10000(t *testing.T) {
	s := testStore(t, 0)
	addRun(t, s, traceID(1), 1, nil)
	batch := model.Batch{}
	for i := 0; i < 10001; i++ {
		batch.Signals = append(batch.Signals, fixture(fmt.Sprintf("span:extra-%05d", i), "span", traceID(1), "resource", int64(300+i), int64(300+i)))
	}
	if err := s.Submit(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	b := model.CohortSelection{From: 0, To: 20000, Limit: 1, Predicate: "inputs.x", Equals: "1", RecordLimit: 10000}
	if _, err := s.SelectCohort(context.Background(), "cap", toyProfile(), b); err == nil || !strings.Contains(err.Error(), "entire selection rejected") || !strings.Contains(err.Error(), traceID(1)) {
		t.Fatal(err)
	}
	b.RecordLimit = 0
	c, err := s.SelectCohort(context.Background(), "larger", toyProfile(), b)
	if err != nil || len(c.RunIDs) != 1 {
		t.Fatal(c, err)
	}
}

func TestDecisionReadPlansStayWithinIndexedPartitions(t *testing.T) {
	s := testStore(t, 0)
	for _, tc := range []struct{ query, index string }{
		{cohortRunsSQL, "runs_time"},
		{cohortRunCountSQL, "runs_time"},
		{cohortRecordsSQL, "signals_run"},
	} {
		args := make([]any, strings.Count(tc.query, "?"))
		for i := range args {
			args[i] = 1
		}
		rows, err := s.reader.Query("EXPLAIN QUERY PLAN "+tc.query, args...)
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
			plan.WriteString(detail + "\n")
		}
		if err = rows.Err(); err != nil {
			t.Fatal(err)
		}
		_ = rows.Close()
		if !strings.Contains(plan.String(), tc.index) || strings.Contains(plan.String(), "SCAN signals") || strings.Contains(plan.String(), "SCAN runs") {
			t.Fatal(plan.String())
		}
	}
}
