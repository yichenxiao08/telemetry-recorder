package store

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
)

func TestViewerRunPaginationBoundsAndFilters(t *testing.T) {
	s := testStore(t, 0)
	ctx := context.Background()
	base := time.Now().UnixNano()
	var signals []model.Signal
	for i := 0; i < 7; i++ {
		id := fmt.Sprintf("r%d", i)
		signal := fixture("span:"+id, "span", id, "res", base+int64(i/2), base+10)
		signal.Name = "work"
		if i%2 == 0 {
			signal.Status = "error"
		}
		signals = append(signals, signal)
	}
	if err := s.Submit(ctx, model.Batch{Signals: signals}); err != nil {
		t.Fatal(err)
	}
	filter := RunFilter{From: base, To: base + 100, Limit: 2, Workload: "work"}
	seen := map[string]bool{}
	for {
		page, err := s.RunPage(ctx, filter)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Runs) > 2 {
			t.Fatal("unbounded page")
		}
		for _, r := range page.Runs {
			if seen[r.ID] {
				t.Fatal("duplicate cursor row")
			}
			seen[r.ID] = true
		}
		if page.Next == nil {
			break
		}
		filter.After = page.Next
	}
	if len(seen) != 7 {
		t.Fatalf("lost rows: %v", seen)
	}
	filter.After = nil
	filter.Outcome = "fail"
	filter.Limit = 100
	page, err := s.RunPage(ctx, filter)
	if err != nil || len(page.Runs) != 4 {
		t.Fatalf("filter: %+v %v", page, err)
	}
	for _, bad := range []RunFilter{{From: base, To: base + 1, Limit: 101}, {From: base, To: base + int64(32*24*time.Hour), Limit: 1}, {From: base, To: base, Limit: 1}, {From: base, To: base + 1, Limit: 1, Outcome: "bad"}} {
		if _, err = s.RunPage(ctx, bad); err == nil {
			t.Fatal("invalid bounds accepted")
		}
	}
}

func TestViewerStructureLimitsAndExplicitPhaseSuggestion(t *testing.T) {
	s := testStore(t, 0)
	ctx := context.Background()
	var signals []model.Signal
	for i := 0; i < 12; i++ {
		id := fmt.Sprintf("p%d", i)
		signal := fixture("span:"+id, "span", id, "base", int64(i+1), int64(i+2))
		signal.Name = "work"
		signals = append(signals, signal)
	}
	r := fixture("span:fail", "span", "fail", "changed", 100, 110)
	r.Name = "work"
	r.Status = "error"
	r.Resource.Attrs = map[string]any{"signalscout.demo.phase": "failure", "signalscout.config.DB_POOL_SIZE": 5, "vcs.revision": "same"}
	signals = append(signals, r)
	event := r
	event.Kind = "event"
	event.ID = "event:retry"
	event.Name = "retry"
	signals = append(signals, event)
	if err := s.Submit(ctx, model.Batch{Signals: signals}); err != nil {
		t.Fatal(err)
	}
	if err := s.Maintain(ctx, time.Now().Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RunStructure(ctx, "fail", 1); !errors.Is(err, EvidenceLimit) {
		t.Fatalf("limit: %v", err)
	}
	detail, err := s.RunStructure(ctx, "fail", 5)
	if err != nil || len(detail.Signals) != 2 || len(detail.Context) != 0 {
		t.Fatalf("structure %+v %v", detail, err)
	}
	before, err := s.PhaseStart(ctx, detail.Run)
	if err != nil || before != time.Unix(0, 100).UTC().Format(time.RFC3339Nano) {
		t.Fatalf("phase %s %v", before, err)
	}
	detail.Run.Env = map[string]any{}
	if before, err = s.PhaseStart(ctx, detail.Run); err != nil || before != "" {
		t.Fatal("invented phase")
	}
	if _, err = s.ViewerComparison(ctx, "fail", model.BaselineOptions{N: 201, ExplicitN: true}); err == nil {
		t.Fatal("unbounded baseline accepted")
	}
	if _, err = s.ViewerComparison(ctx, "fail", model.BaselineOptions{}); err != nil {
		t.Fatal(err)
	}
}
