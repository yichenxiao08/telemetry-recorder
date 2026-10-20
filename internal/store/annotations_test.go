package store

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
)

func TestLateAnnotationDoesNotExtendExecutionOrContext(t *testing.T) {
	ctx := context.Background()
	s := testStore(t, 0)
	root := fixture("span:a", "span", "a", "r", 100, 200)
	if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{root}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Maintain(ctx, time.Now().Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	before, err := s.Show(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	scopes := func() string {
		var data string
		if err := s.reader.QueryRowContext(ctx, "SELECT json_group_array(json_object('start',start_ns,'end',end_ns)) FROM context_scopes WHERE run_id='a'").Scan(&data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	beforeScope := scopes()
	a := model.Annotation{ID: "review-1", RunID: "a", Dimension: "quality", Value: "bad", Source: "human-mark", Reviewer: "synthetic", Synthetic: true, RubricID: "toy", RubricVersion: "1", CreatedAt: 200 + int64(time.Minute)}
	if err := s.SaveAnnotation(ctx, a); err != nil {
		t.Fatal(err)
	}
	after, err := s.Show(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) || beforeScope != scopes() {
		t.Fatal("annotation changed execution or context")
	}
	if err := s.SaveAnnotation(ctx, a); err == nil {
		t.Fatal("duplicate annotation overwritten")
	}
	a.ID = "review-2"
	a.Reviewer = "other"
	a.Value = "good"
	if err := s.SaveAnnotation(ctx, a); err != nil {
		t.Fatal(err)
	}
	labels, err := s.Annotations(ctx, "a", 0, time.Now().Add(time.Hour).UnixNano(), 10)
	if err != nil || len(labels) != 2 {
		t.Fatalf("disagreement lost: %v %v", labels, err)
	}
	if labels[0].RecordedAt == 0 {
		t.Fatal("missing server recorded time")
	}
	if _, err := s.writer.ExecContext(ctx, "UPDATE annotations SET value='fake'"); err == nil {
		t.Fatal("update accepted")
	}
	if _, err := s.writer.ExecContext(ctx, "DELETE FROM annotations"); err == nil {
		t.Fatal("delete accepted")
	}
	// Retractions append; the target row survives.
	a.ID = "review-3"
	a.Retracts = "review-2"
	if err := s.SaveAnnotation(ctx, a); err != nil {
		t.Fatal(err)
	}
	labels, err = s.Annotations(ctx, "a", 0, time.Now().Add(time.Hour).UnixNano(), 10)
	if err != nil || len(labels) != 3 {
		t.Fatal(labels, err)
	}
	b, _ := json.Marshal(labels)
	if len(b) == 0 {
		t.Fatal("empty labels")
	}
}
