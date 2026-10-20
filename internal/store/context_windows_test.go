package store

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
)

func TestMergedContextIntervalsRetainGaps(t *testing.T) {
	got := mergeContextBounds([]contextBounds{{40, 50}, {10, 20}, {15, 30}})
	want := []contextBounds{{10, 30}, {40, 50}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("merged %v want %v", got, want)
	}
}
func TestContextSnapshotPreservesPerRunMembershipAndLateUpdates(t *testing.T) {
	ctx := context.Background()
	s := testStore(t, 0)
	a := fixture("span:a", "span", "a", "r1", 10, 30)
	a.Resource.Attrs["host.name"] = "host"
	b := fixture("span:b", "span", "b", "r2", 70, 90)
	b.Resource.Attrs["host.name"] = "host"
	value := 1.0
	point := fixture("metric:early", "metric", "", "ambient", 20, 20)
	point.Resource.Attrs["host.name"] = "host"
	point.Value = &value
	gap := point
	gap.ID = "metric:gap"
	gap.TS = 50
	gap.End = 50
	late := point
	late.ID = "metric:late"
	late.TS = 80
	late.End = 80
	if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{a, b, point, gap, late}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Maintain(ctx, time.Now().Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	tx, err := s.reader.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	windows := map[string]contextBounds{"a": {1, 100}, "b": {1, 100}}
	contexts, err := readContextWindows(ctx, tx, windows, 1, 2)
	if err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b"} {
		legacy, err := s.Show(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(legacy.Context, contexts[id]) {
			t.Fatalf("wrong context for %s: %+v", id, contexts[id])
		}
		if contexts[id][0].RunID != "" {
			t.Fatal("context became owned")
		}
	}
	tx, err = s.reader.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = readContextWindows(ctx, tx, windows, 1, 1)
	_ = tx.Rollback()
	if err != EvidenceLimit {
		t.Fatal("accepted partial logical context", err)
	}
	value = 2
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{point}}); err != nil {
		t.Fatal(err)
	}
	updated, err := s.RunContext(ctx, "a", 1, 100, 1)
	if err != nil || *updated[0].Value != 2 || *contexts["a"][0].Value != 1 {
		t.Fatalf("cache escaped its snapshot: %+v %v", updated, err)
	}
}
