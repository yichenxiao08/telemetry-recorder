package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
)

func TestBaselineModesUseEarlierClosedPassesAndOwnedStructure(t *testing.T) {
	ctx := context.Background()
	s := testStore(t, 0)
	var signals []model.Signal
	for i := 0; i < 45; i++ {
		id := fmt.Sprintf("p%02d", i)
		signal := fixture("span:"+id, "span", id, "r", int64(10+i), int64(11+i))
		signal.Name = "work"
		signal.SpanID = id
		signals = append(signals, signal)
	}
	for _, entry := range []struct {
		id      string
		ts      int64
		outcome string
	}{
		{"fail", 1000, "fail"}, {"future", 1001, "pass"}, {"tie", 1000, "pass"}, {"degraded", 999, "degraded"},
	} {
		signal := fixture("span:"+entry.id, "span", entry.id, "r", entry.ts, entry.ts+1)
		signal.Name = "work"
		signal.SpanID = entry.id
		if entry.outcome == "fail" {
			signal.Status = "error"
		}
		if entry.outcome == "degraded" {
			signal.ParentID = "missing"
		}
		signals = append(signals, signal)
	}
	other := fixture("span:other", "span", "other", "r", 999, 1000)
	other.Name = "other"
	signals = append(signals, other)
	ownedLog := fixture("log:fail", "log", "fail", "r", 1000, 1000)
	ownedMetric := fixture("metric:fail", "metric", "fail", "r", 1000, 1000)
	signals = append(signals, ownedLog, ownedMetric)
	if err := s.Submit(ctx, model.Batch{Signals: signals}); err != nil {
		t.Fatal(err)
	}
	if err := s.Maintain(ctx, time.Now().Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	open := fixture("span:open", "span", "open", "r", 999, 1000)
	open.Name = "work"
	if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{open}}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		options model.BaselineOptions
		mode    string
		count   int
		first   string
	}{
		{"default", model.BaselineOptions{}, "default", 40, "p44"},
		{"count", model.BaselineOptions{N: 12, ExplicitN: true}, "count", 12, "p44"},
		{"run window", model.BaselineOptions{Before: "p20"}, "before", 20, "p19"},
		{"time window", model.BaselineOptions{Before: time.Unix(0, 30).UTC().Format(time.RFC3339Nano)}, "before", 20, "p19"},
		{"window and count", model.BaselineOptions{N: 10, ExplicitN: true, Before: "p20"}, "before", 10, "p19"},
		{"future capped", model.BaselineOptions{Before: "future"}, "before", 40, "p44"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := s.Comparison(ctx, "fail", tc.options)
			if err != nil {
				t.Fatal(err)
			}
			if got.Spec.Mode != tc.mode || len(got.Baseline) != tc.count || got.Spec.RunIDs[0] != tc.first || got.Spec.ExplicitN != tc.options.ExplicitN {
				t.Fatalf("selection %+v", got.Spec)
			}
			if len(got.Failing.Signals) != 1 || len(got.Failing.Context) != 0 {
				t.Fatal("analysis loaded logs, metrics or context")
			}
			for _, b := range got.Baseline {
				if !b.Run.Closed || b.Run.Outcome != "pass" || b.Run.StartedAt >= got.Spec.BeforeNS {
					t.Fatalf("invalid baseline %+v", b.Run)
				}
			}
		})
	}
	for _, options := range []model.BaselineOptions{{N: 0, ExplicitN: true}, {N: -1, ExplicitN: true}, {Before: "missing"}, {Before: "9999-01-01T00:00:00Z"}} {
		if _, err := s.Comparison(ctx, "fail", options); err == nil {
			t.Fatalf("invalid options accepted: %+v", options)
		}
	}
	for _, id := range []string{"missing", "open", "p00", "degraded"} {
		if _, err := s.Comparison(ctx, id, model.BaselineOptions{}); err == nil {
			t.Fatalf("invalid failing run %s accepted", id)
		}
	}
}

func TestResultWriterPersistsModeWithoutIngestMaintenance(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "results.db")
	s, err := Open(ctx, path, Options{QueueSize: 2, IdleTimeout: time.Hour, SweepInterval: time.Hour, ContextBefore: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	root := fixture("span:fail", "span", "fail", "r", 10, 20)
	root.Status = "error"
	ambient := fixture("log:ambient", "log", "", "r", 9, 9)
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{root, ambient}}); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	w, err := OpenResultWriter(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	result := model.DiffResult{ID: "diff:test", FailingRunID: "fail", Status: "ok", CreatedAt: time.Now().UTC(), BaselineSpec: model.BaselineSpec{Mode: "before", BeforeInput: "anchor", RunIDs: []string{"pass"}}, Differences: []model.Difference{{Type: "config", Name: "POOL", Score: 1}}}
	if err = w.SaveDiff(ctx, result); err != nil {
		t.Fatal(err)
	}
	var spec, data string
	if err = w.reader.QueryRowContext(ctx, "SELECT baseline_spec,results FROM diff_results WHERE diff_id=?", result.ID).Scan(&spec, &data); err != nil {
		t.Fatal(err)
	}
	var saved model.DiffResult
	if err = json.Unmarshal([]byte(data), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.BaselineSpec.Mode != "before" || saved.ID != result.ID || !json.Valid([]byte(spec)) {
		t.Fatalf("lost result %s", data)
	}
	detail, err := w.Show(ctx, "fail")
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Context) != 1 {
		t.Fatal("analysis writer changed context padding")
	}
	if err = w.Submit(ctx, model.Batch{}); err == nil {
		t.Fatal("result-only writer accepted intake")
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	if err = w.SaveDiff(ctx, result); err != Closed {
		t.Fatalf("save after close: %v", err)
	}
	missing := filepath.Join(t.TempDir(), "missing.db")
	if w, err = OpenResultWriter(ctx, missing); err == nil {
		_ = w.Close()
		t.Fatal("created missing database")
	}
	if _, err = os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("analysis created a file")
	}
}
