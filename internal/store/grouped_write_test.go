package store

import (
	"context"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
)

func TestGroupedWritesIsolateJobErrorsAndKeepReceiptTimes(t *testing.T) {
	ctx := context.Background()
	s := testStore(t, 0)
	now := time.Now()
	first := fixture("first", "span", "run", "r", 10, 20)
	bad := first
	bad.TraceID = "conflict"
	last := fixture("last", "span", "run", "r", 12, 25)
	jobs := []job{{batch: &model.Batch{Signals: []model.Signal{first}}, now: now}, {batch: &model.Batch{Signals: []model.Signal{bad}}, now: now.Add(time.Second)}, {batch: &model.Batch{Signals: []model.Signal{last}}, now: now.Add(2 * time.Second)}}
	errs := s.writeJobs(ctx, jobs)
	if errs[0] != nil || errs[1] == nil || errs[2] != nil {
		t.Fatal("job error isolation failed", errs)
	}
	detail, err := s.Show(ctx, "run")
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Signals) != 2 || detail.Run.EndedAt != 25 {
		t.Fatal("lost successful jobs", detail)
	}
	if _, ok := s.open["conflict"]; ok {
		t.Fatal("failed job changed open cache")
	}
	if s.open["run"] != now.Add(2*time.Second).UnixNano() {
		t.Fatal("receipt time changed")
	}
	for id, want := range map[string]int64{"first": now.UnixNano(), "last": now.Add(2 * time.Second).UnixNano()} {
		var got int64
		if err = s.reader.QueryRow("SELECT received_at FROM signals WHERE evidence_id=?", id).Scan(&got); err != nil || got != want {
			t.Fatalf("receipt %s=%d want %d: %v", id, got, want, err)
		}
	}
}

func TestGroupedCommitFailureAcknowledgesNoSuccessAndRecovers(t *testing.T) {
	ctx := context.Background()
	s := testStore(t, 0)
	// A deferred FK fails COMMIT after both jobs have executed successfully.
	_, err := s.writer.Exec(`CREATE TABLE commit_guard(id TEXT REFERENCES resources(resource_id) DEFERRABLE INITIALLY DEFERRED);
	CREATE TRIGGER force_commit_failure AFTER INSERT ON signals WHEN new.evidence_id='guarded' BEGIN INSERT INTO commit_guard VALUES('missing'); END;`)
	if err != nil {
		t.Fatal(err)
	}
	a, b := fixture("good", "span", "a", "r", 10, 20), fixture("guarded", "span", "b", "r", 10, 20)
	errs := s.writeJobs(ctx, []job{{batch: &model.Batch{Signals: []model.Signal{a}}, now: time.Now()}, {batch: &model.Batch{Signals: []model.Signal{b}}, now: time.Now()}})
	if errs[0] == nil || errs[1] == nil {
		t.Fatal("acknowledged failed commit", errs)
	}
	counts, err := s.Counts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counts["span"] != 0 || len(s.open) != 0 {
		t.Fatal("commit failure left partial state", counts, s.open)
	}
	if _, err = s.writer.Exec("DROP TRIGGER force_commit_failure"); err != nil {
		t.Fatal(err)
	}
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{a}}); err != nil {
		t.Fatal("writer did not recover", err)
	}
}

func TestWriterPreservesMaintenanceBarrierAndDrainsAcceptedJobs(t *testing.T) {
	s := testStore(t, 0)
	now := time.Now()
	first, last := fixture("first", "span", "run", "r", 10, 20), fixture("last", "span", "run", "r", 12, 25)
	jobs := []job{{batch: &model.Batch{Signals: []model.Signal{first}}, now: now, done: make(chan error, 1)}, {now: now.Add(2 * time.Second), done: make(chan error, 1)}, {batch: &model.Batch{Signals: []model.Signal{last}}, now: now.Add(3 * time.Second), done: make(chan error, 1)}}
	for _, j := range jobs {
		s.queue <- j
	}
	for _, j := range jobs {
		if err := <-j.done; err != nil {
			t.Fatal(err)
		}
	}
	detail, err := s.Show(context.Background(), "run")
	if err != nil {
		t.Fatal(err)
	}
	if !detail.Run.Closed || len(detail.Signals) != 2 {
		t.Fatal("maintenance reordered across ingest", detail)
	}
	late := fixture("drained", "span", "run", "r", 15, 30)
	j := job{batch: &model.Batch{Signals: []model.Signal{late}}, now: now.Add(4 * time.Second), done: make(chan error, 1)}
	s.queue <- j
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if err = <-j.done; err != nil {
		t.Fatal("accepted job not drained", err)
	}
}
