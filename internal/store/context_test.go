package store

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
)

func TestContextSharedHostAndInstanceRules(t *testing.T) {
	ctx := context.Background()
	s := testStore(t, 0)
	span := fixture("span:shop", "span", "run", "shop-resource", 10, 20)
	span.Resource.Attrs = map[string]any{"service.name": "shop", "host.name": "host-a", "service.instance.id": "shop-1"}
	// A Collector hostmetrics receiver has a host resource, not the app resource.
	hostMetric := fixture("metric:host", "metric", "", "host-only", 15, 15)
	hostMetric.Resource.Attrs = map[string]any{"host.name": "host-a"}
	instanceLog := fixture("log:instance", "log", "", "instance-only", 15, 15)
	instanceLog.Resource.Attrs = map[string]any{"service.instance.id": "shop-1"}
	exact := fixture("log:exact", "log", "", "shop-resource", 15, 15)
	exact.Resource = span.Resource
	unrelated := fixture("log:unrelated", "log", "", "unrelated", 15, 15)
	unrelated.Resource.Attrs = map[string]any{"host.name": "host-b", "service.instance.id": "shop-2"}
	empty := fixture("metric:empty", "metric", "", "empty", 15, 15)
	empty.Resource.Attrs = map[string]any{"host.name": "", "service.instance.id": ""}
	outside := hostMetric
	outside.ID = "metric:outside"
	outside.TS = 21
	outside.End = 21
	owned := hostMetric
	owned.ID = "metric:owned"
	owned.Resource = model.Resource{ID: "owned", Attrs: map[string]any{"host.name": "host-a", "signalscout.run.id": "other-run"}, Raw: []byte(`{}`)}
	if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{hostMetric, instanceLog, exact, unrelated, empty, outside, owned, span}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Maintain(ctx, time.Now().Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	detail, err := s.Show(ctx, "run")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for _, signal := range detail.Context {
		if signal.RunID != "" {
			t.Fatal("context changed ownership")
		}
		for _, match := range signal.ContextMatches {
			if match.ResourceID != span.Resource.ID {
				t.Fatalf("wrong span resource: %+v", match)
			}
			got[signal.ID] = append(got[signal.ID], match.Rule)
		}
	}
	want := map[string][]string{"metric:host": {"host.name"}, "log:instance": {"service.instance.id"}, "log:exact": {"exact_resource", "host.name", "service.instance.id"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("context rules got %+v, want %+v", got, want)
	}
	// Late ambient evidence also links to closed runs and keeps its provenance.
	late := hostMetric
	late.ID = "metric:late"
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{late}}); err != nil {
		t.Fatal(err)
	}
	detail, err = s.Show(ctx, "run")
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Context) != 4 {
		t.Fatalf("late host context not attached: %+v", detail.Context)
	}
}

func TestContextCannotUseResourcesSeenOnlyInLogs(t *testing.T) {
	ctx := context.Background()
	s := testStore(t, 0)
	log := fixture("owned-log", "log", "run", "log-resource", 10, 20)
	log.Resource.Attrs = map[string]any{"host.name": "host-a"}
	metric := fixture("ambient", "metric", "", "host-only", 15, 15)
	metric.Resource.Attrs = map[string]any{"host.name": "host-a"}
	if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{log, metric}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Maintain(ctx, time.Now().Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	detail, err := s.Show(ctx, "run")
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Context) != 0 {
		t.Fatal("log-only resource qualified as a serving span resource")
	}
	span := log
	span.ID = "span"
	span.Kind = "span"
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{span}}); err != nil {
		t.Fatal(err)
	}
	detail, err = s.Show(ctx, "run")
	if err != nil {
		t.Fatal(err)
	}
	if !detail.Run.Closed || len(detail.Context) != 1 {
		t.Fatal("late span did not qualify its host")
	}
}

func TestContextPaddingBoundaries(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "padding.db"), Options{QueueSize: 2, IdleTimeout: time.Second, SweepInterval: time.Hour, ContextBefore: 30 * time.Second, ContextAfter: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	}()
	span := fixture("span", "span", "run", "r", int64(100*time.Second), int64(101*time.Second))
	batch := model.Batch{Signals: []model.Signal{span}}
	for index, ts := range []int64{int64(70*time.Second) - 1, int64(70 * time.Second), int64(106 * time.Second), int64(106*time.Second) + 1} {
		signal := fixture(string(rune('a'+index)), "log", "", "r", ts, ts)
		batch.Signals = append(batch.Signals, signal)
	}
	if err = s.Submit(ctx, batch); err != nil {
		t.Fatal(err)
	}
	if err = s.Maintain(ctx, time.Now().Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	detail, err := s.Show(ctx, "run")
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Context) != 2 || detail.Context[0].ID != "b" || detail.Context[1].ID != "c" {
		t.Fatalf("padding boundaries: %+v", detail.Context)
	}
}

func TestContextBackfillAndChangedPaddingOnRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "restart-context.db")
	options := Options{QueueSize: 2, IdleTimeout: time.Second, SweepInterval: time.Hour, ContextBefore: 30 * time.Second}
	s, err := Open(ctx, path, options)
	if err != nil {
		t.Fatal(err)
	}
	span := fixture("span", "span", "run", "app", int64(100*time.Second), int64(101*time.Second))
	span.Resource.Attrs = map[string]any{"host.name": "host-a"}
	inside := fixture("inside", "metric", "", "host", int64(100*time.Second), int64(100*time.Second))
	inside.Resource.Attrs = map[string]any{"host.name": "host-a"}
	padding := inside
	padding.ID = "padding"
	padding.TS = int64(75 * time.Second)
	padding.End = padding.TS
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{span, inside, padding}}); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	options.ContextBefore = 0
	s, err = Open(ctx, path, options)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	}()
	detail, err := s.Show(ctx, "run")
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Context) != 1 || detail.Context[0].ID != "inside" || len(detail.Context[0].ContextMatches) != 1 {
		t.Fatalf("restart retained stale padded links: %+v", detail.Context)
	}
}
