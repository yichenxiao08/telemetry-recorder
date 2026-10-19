package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
)

func testStore(t *testing.T, retention time.Duration) *Store {
	t.Helper()
	s, err := Open(context.Background(), filepath.Join(t.TempDir(), "test.db"), Options{QueueSize: 2, IdleTimeout: time.Second, Retention: retention, SweepInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s
}

func fixture(id, kind, trace, resource string, ts, end int64) model.Signal {
	return model.Signal{ID: id, Kind: kind, TraceID: trace, Name: "request", TS: ts, End: end, Resource: model.Resource{ID: resource, Attrs: map[string]any{"service.name": "shop"}, Raw: []byte(`{}`)}, Attrs: map[string]any{}, Raw: []byte(`{}`), Scope: []byte(`{}`)}
}

func TestPersistFinalizeLateContextAndRetention(t *testing.T) {
	ctx := context.Background()
	s := testStore(t, time.Hour)
	root := fixture("span:a", "span", "a", "r", 10, 20)
	ambient := fixture("log:a", "log", "", "r", 15, 15)
	outside := fixture("metric:outside", "metric", "", "r", 21, 21)
	other := fixture("log:other", "log", "", "other", 15, 15)
	if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{ambient, outside, other, root}}); err != nil {
		t.Fatal(err)
	}
	detail, err := s.Show(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Run.Closed || len(detail.Context) != 0 {
		t.Fatalf("premature finalization: %+v", detail)
	}
	if err = s.Maintain(ctx, time.Now().Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	detail, err = s.Show(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if !detail.Run.Closed || len(detail.Signals) != 1 || len(detail.Context) != 1 || detail.Context[0].ID != ambient.ID {
		t.Fatalf("close/context mismatch: %+v", detail)
	}
	if detail.Signals[0].Resource.Attrs["service.name"] != "shop" || string(detail.Signals[0].Resource.Raw) != "{}" || detail.Context[0].Resource.Attrs["service.name"] != "shop" {
		t.Fatal("resource normalization lost original metadata")
	}
	var service string
	if err = s.reader.QueryRowContext(ctx, "SELECT service FROM spans WHERE run_id='a'").Scan(&service); err != nil {
		t.Fatal(err)
	}
	if service != "shop" {
		t.Fatalf("span view service=%q", service)
	}
	root.Status = "error"
	root.End = 22
	late := fixture("log:late", "log", "", "r", 18, 18)
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{root, late}}); err != nil {
		t.Fatal(err)
	}
	detail, err = s.Show(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if !detail.Run.Closed || detail.Run.Outcome != "fail" || len(detail.Context) != 3 {
		t.Fatalf("late evidence mismatch: %+v", detail)
	}
	if _, err = s.reader.ExecContext(ctx, "DELETE FROM signals"); err == nil {
		t.Fatal("read pool allowed writes")
	}
	if err = s.Maintain(ctx, time.Now().Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	list, err := s.List(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatal("retention left closed runs")
	}
	for _, table := range []string{"signals", "resources", "context_links", "run_resources"} {
		var count int
		if err = s.reader.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s left %d records", table, count)
		}
	}
}

func TestTransactionRollbackAndDuplicates(t *testing.T) {
	s := testStore(t, 0)
	ctx := context.Background()
	root := fixture("span:a", "span", "a", "r", 10, 20)
	bad := root
	bad.ID = "bad"
	bad.Kind = "invalid"
	if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{root, bad}}); err == nil {
		t.Fatal("invalid batch committed")
	}
	list, err := s.List(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatal("rollback left runs")
	}
	for i := 0; i < 2; i++ {
		if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{root}}); err != nil {
			t.Fatal(err)
		}
	}
	detail, err := s.Show(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Signals) != 1 {
		t.Fatal("duplicate replay created records")
	}
	conflict := root
	conflict.Resource.Attrs = map[string]any{"signalscout.run.id": "ml-run"}
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{conflict}}); err == nil {
		t.Fatal("evidence changed run ownership")
	}
	list, err = s.List(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatal("ownership conflict left a phantom run")
	}
}

func TestQueueFullAndConcurrentClose(t *testing.T) {
	s := testStore(t, 0)
	ctx := context.Background()
	// Occupying the sole connection pauses the consumer without production hooks.
	connection, err := s.writer.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for i := 0; i < 3; i++ {
		workers.Add(1)
		go func() { defer workers.Done(); _ = s.Submit(ctx, model.Batch{}) }()
		time.Sleep(10 * time.Millisecond)
	}
	deadline := time.Now().Add(time.Second)
	for len(s.queue) != cap(s.queue) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if err = s.Submit(ctx, model.Batch{}); err != QueueFull {
		t.Errorf("got %v, want QueueFull", err)
	}
	if err = connection.Close(); err != nil {
		t.Fatal(err)
	}
	workers.Wait()
	workers.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer workers.Done()
			if err := s.Close(); err != nil {
				t.Error(err)
			}
		}()
	}
	workers.Wait()
	if err = s.Submit(ctx, model.Batch{}); err != Closed {
		t.Fatalf("submit after close: %v", err)
	}
}

func TestRestartAndMigrations(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "space in name.db")
	options := Options{QueueSize: 2, IdleTimeout: time.Second, SweepInterval: time.Hour}
	s, err := Open(ctx, path, options)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{fixture("span:a", "span", "a", "r", 1, 10)}}); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(ctx, path, options)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	}()
	late := fixture("span:b", "span", "a", "r", 2, 9)
	late.ParentID = "a"
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{late}}); err != nil {
		t.Fatal(err)
	}
	detail, err := s.Show(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if !detail.Run.Closed || len(detail.Signals) != 2 {
		t.Fatalf("restart lost closed run: %+v", detail)
	}
}

func TestMaintenanceSchedulerAndOpenRunRecovery(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "recovery.db")
	options := Options{QueueSize: 2, IdleTimeout: 50 * time.Millisecond, SweepInterval: 10 * time.Millisecond}
	s, err := Open(ctx, path, options)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{fixture("s", "span", "a", "r", 1, 2)}}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		detail, err := s.Show(ctx, "a")
		if err != nil {
			t.Fatal(err)
		}
		if detail.Run.Closed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("scheduler did not close idle run")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	// Simulate durable open state left by an interrupted process using test-only SQL.
	uri, err := databaseURI(path)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", uri+"?mode=rw")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, `UPDATE runs SET closed=0,state=json_set(state,'$.Run.closed',json('false'))`); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(ctx, path, options)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	}()
	deadline = time.Now().Add(2 * time.Second)
	for {
		detail, err := s.Show(ctx, "a")
		if err != nil {
			t.Fatal(err)
		}
		if detail.Run.Closed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("recovered open run not finalized")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRetentionPreservesOpenRuns(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "retention.db"), Options{QueueSize: 2, IdleTimeout: 24 * time.Hour, Retention: time.Hour, SweepInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	}()
	first := fixture("s1", "span", "a", "r", 10, 20)
	second := fixture("s2", "span", "b", "r", 12, 22)
	ambient := fixture("ambient", "metric", "", "r", 15, 15)
	if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{first, second, ambient}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Maintain(ctx, time.Now().Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	list, err := s.List(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Closed || list[1].Closed {
		t.Fatal("retention removed or closed open runs")
	}
	// Re-ingest ambient evidence after its independent retention interval expired.
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{ambient}}); err != nil {
		t.Fatal(err)
	}
	if err = s.Maintain(ctx, time.Now().Add(25*time.Hour)); err != nil {
		t.Fatal(err)
	}
	// The run and signal retention horizons intentionally expire here as well.
	list, err = s.List(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatal("expired closed runs survived retention")
	}
}

func TestContextCanBeSharedByOverlappingRuns(t *testing.T) {
	s := testStore(t, 0)
	ctx := context.Background()
	batch := model.Batch{Signals: []model.Signal{fixture("s1", "span", "a", "r", 10, 20), fixture("s2", "span", "b", "r", 12, 22), fixture("ambient", "metric", "", "r", 15, 15)}}
	if err := s.Submit(ctx, batch); err != nil {
		t.Fatal(err)
	}
	if err := s.Maintain(ctx, time.Now().Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b"} {
		detail, err := s.Show(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if len(detail.Context) != 1 || detail.Context[0].RunID != "" {
			t.Fatalf("context was reassigned instead of linked: %+v", detail)
		}
	}
}
