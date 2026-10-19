package store

import (
	"context"
	"fmt"
	"testing"

	"github.com/signalscout/signalscout/internal/model"
)

func TestLargeBatchOwnershipConflictRollsBackEveryChunk(t *testing.T) {
	for _, position := range []int{0, 127, 128, 259} {
		t.Run(fmt.Sprint(position), func(t *testing.T) {
			ctx := context.Background()
			s := testStore(t, 0)
			original := fixture("existing", "span", "original-run", "r", 100, 200)
			if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{original}}); err != nil {
				t.Fatal(err)
			}
			batch := model.Batch{}
			for i := 0; i < 260; i++ {
				batch.Signals = append(batch.Signals, fixture(fmt.Sprint(i), "log", "new-run", "r", 150, 150))
			}
			conflict := original
			conflict.TraceID = "new-run"
			batch.Signals[position] = conflict
			if err := s.Submit(ctx, batch); err == nil {
				t.Fatal("accepted conflicting ownership")
			}
			counts, err := s.Counts(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if counts["span"] != 1 || counts["log"] != 0 {
				t.Fatal("partial chunk committed", counts)
			}
			list, err := s.List(ctx, "", "")
			if err != nil {
				t.Fatal(err)
			}
			if len(list) != 1 || list[0].ID != "original-run" {
				t.Fatal("partial run state committed", list)
			}
		})
	}
}

func TestLargeBatchDuplicateUpdatesRemainIdempotent(t *testing.T) {
	ctx := context.Background()
	s := testStore(t, 0)
	batch := model.Batch{}
	for i := 0; i < 260; i++ {
		batch.Signals = append(batch.Signals, fixture(fmt.Sprint(i), "span", "run", "r", int64(i+100), int64(i+101)))
	}
	update := batch.Signals[0]
	update.Status = "error"
	update.End = 1000
	update.Raw = []byte(`{"updated":true}`)
	batch.Signals = append(batch.Signals, update)
	for range 2 {
		if err := s.Submit(ctx, batch); err != nil {
			t.Fatal(err)
		}
	}
	detail, err := s.Show(ctx, "run")
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Signals) != 260 || detail.Run.Outcome != "fail" || detail.Run.EndedAt != 1000 {
		t.Fatalf("lost duplicate update: %+v", detail.Run)
	}
	if string(detail.Signals[0].Raw) != `{"updated":true}` {
		t.Fatal("lost raw update")
	}
}
