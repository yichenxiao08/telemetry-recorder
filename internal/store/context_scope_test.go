package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/runs"
)

func TestContextStorageScalesWithRunResourcesNotEvidence(t *testing.T) {
	ctx := context.Background()
	s := testStore(t, 0)
	batch := model.Batch{}
	for i := 0; i < 20; i++ {
		span := fixture(string(rune('A'+i)), "span", string(rune('A'+i)), "app", 100, 200)
		span.Resource.Attrs = map[string]any{"host.name": "host", "service.instance.id": "instance"}
		batch.Signals = append(batch.Signals, span)
	}
	for i := 0; i < 100; i++ {
		log := fixture(string(rune(1000+i)), "log", "", "app", 150, 150)
		log.Resource = batch.Signals[0].Resource
		batch.Signals = append(batch.Signals, log)
	}
	if err := s.Submit(ctx, batch); err != nil {
		t.Fatal(err)
	}
	if err := s.Maintain(ctx, time.Now().Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	for table, want := range map[string]int{"context_scopes": 60, "context_links": 2000, "context_matches": 6000} {
		var count int
		if err := s.reader.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("%s = %d, want %d", table, count, want)
		}
	}
	var object string
	if err := s.reader.QueryRowContext(ctx, "SELECT type FROM sqlite_master WHERE name='context_links'").Scan(&object); err != nil || object != "view" {
		t.Fatal("evidence links are still materialized", object, err)
	}
	late := fixture("late", "metric", "", "host-only", 150, 150)
	late.Resource.Attrs = map[string]any{"host.name": "host"}
	if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{late}}); err != nil {
		t.Fatal(err)
	}
	detail, err := s.Show(ctx, "A")
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Context) != 101 {
		t.Fatalf("late evidence not linked: %d", len(detail.Context))
	}
	var physical int
	if err = s.reader.QueryRowContext(ctx, "SELECT count(*) FROM context_scopes").Scan(&physical); err != nil || physical != 60 {
		t.Fatal("ambient data expanded physical links", physical, err)
	}
}

func TestContextMigrationFromMaterializedSchema(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "v4.db")
	uri, err := databaseURI(path)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", uri+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE schema_migrations(name TEXT PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() > "004_viewer_run_time.sql" {
			continue
		}
		data, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(data)); err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec("INSERT INTO schema_migrations VALUES(?)", entry.Name()); err != nil {
			t.Fatal(err)
		}
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	span := fixture("span", "span", "run", "app", 100, 200)
	span.Resource.Attrs = map[string]any{"host.name": "h", "service.instance.id": "i"}
	log := fixture("log", "log", "", "app", 150, 150)
	log.Resource = span.Resource
	attrs, _ := json.Marshal(span.Resource.Attrs)
	if _, err = tx.Exec("INSERT INTO resources VALUES(?,?,?)", "app", string(attrs), "{}"); err != nil {
		t.Fatal(err)
	}
	state := runs.Observe(runs.State{}, span, time.Now(), 0)
	state.Run.Closed = true
	if err = saveState(ctx, tx, state); err != nil {
		t.Fatal(err)
	}
	for _, signal := range []model.Signal{span, log} {
		signal.RunID, _ = runs.Identity(signal)
		signal.Resource = model.Resource{ID: "app"}
		data, _ := json.Marshal(signal)
		if _, err = tx.Exec("INSERT INTO signals VALUES(?,?,?,?,?,?,?,?)", signal.ID, signal.Kind, signal.RunID, "app", signal.TS, signal.End, time.Now().UnixNano(), string(data)); err != nil {
			t.Fatal(err)
		}
	}
	for _, query := range []string{"INSERT INTO run_resources VALUES('run','app')", "INSERT INTO context_links VALUES('run','log')", "INSERT INTO context_matches VALUES('run','log','exact_resource','app'),('run','log','host.name','app'),('run','log','service.instance.id','app')"} {
		if _, err = tx.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	opts := Options{QueueSize: 2, IdleTimeout: time.Hour, SweepInterval: time.Hour}
	for range 2 {
		s, err := Open(ctx, path, opts)
		if err != nil {
			t.Fatal(err)
		}
		detail, err := s.Show(ctx, "run")
		if err != nil {
			t.Fatal(err)
		}
		if len(detail.Signals) != 1 || len(detail.Context) != 1 || len(detail.Context[0].ContextMatches) != 3 || !detail.Run.Closed {
			t.Fatalf("migration lost evidence: %+v", detail)
		}
		var before, after int64
		if err = s.reader.QueryRow("SELECT before_ns,after_ns FROM context_settings WHERE singleton=1").Scan(&before, &after); err != nil || before != 0 || after != 0 {
			t.Fatal("missing restart checkpoint", err)
		}
		rows, err := s.reader.Query("PRAGMA foreign_key_check")
		if err != nil {
			t.Fatal(err)
		}
		if rows.Next() {
			t.Fatal("migration broke foreign keys")
		}
		_ = rows.Close()
		if err = s.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestContextReflectsUpdatedAmbientTimeAndServingResource(t *testing.T) {
	ctx := context.Background()
	s := testStore(t, 0)
	span := fixture("span", "span", "run", "app-a", 100, 200)
	span.Resource.Attrs = map[string]any{"host.name": "a"}
	log := fixture("log", "log", "", "host-a", 150, 150)
	log.Resource.Attrs = map[string]any{"host.name": "a"}
	if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{span, log}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Maintain(ctx, time.Now().Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	assertCount := func(want int) {
		t.Helper()
		d, err := s.Show(ctx, "run")
		if err != nil {
			t.Fatal(err)
		}
		if len(d.Context) != want {
			t.Fatalf("context count=%d want %d", len(d.Context), want)
		}
	}
	assertCount(1)
	log.TS, log.End = 201, 201
	if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{log}}); err != nil {
		t.Fatal(err)
	}
	assertCount(0)
	log.TS, log.End = 150, 150
	if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{log}}); err != nil {
		t.Fatal(err)
	}
	assertCount(1)
	span.Resource.ID = "app-b"
	span.Resource.Attrs = map[string]any{"host.name": "b"}
	if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{span}}); err != nil {
		t.Fatal(err)
	}
	assertCount(0)
}
