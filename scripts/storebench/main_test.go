package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/signalscout/signalscout/internal/store"
)

func TestFixtureDeterminismAndCoverage(t *testing.T) {
	for _, kind := range []string{"span", "log", "metric"} {
		a, b := signal(kind, 512, 1), signal(kind, 512, 1)
		if !reflect.DeepEqual(a, b) || !json.Valid(a.Raw) {
			t.Fatalf("invalid/non-deterministic %s", kind)
		}
		if a.ID == signal(kind, 512, 2).ID {
			t.Fatal("seed did not change identity")
		}
	}
	host := signal("metric", 0, 1)
	if len(host.Resource.Attrs) != 1 || host.Resource.Attrs["host.name"] == nil {
		t.Fatal("missing host-only fixture")
	}
	for _, q := range queries(1000000, 1) {
		if !strings.Contains(q.SQL, "LIMIT ?") || !strings.Contains(q.SQL, "ts_ns>=?") || !strings.Contains(q.SQL, "ts_ns<?") || q.Args[len(q.Args)-1] != 100 {
			t.Fatal("unbounded query", q.Name)
		}
	}
}

func TestFixtureUsesDurableStoreAndRefusesOverwrite(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "fixture.db")
	if _, err := populate(ctx, path, 64, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := populate(ctx, path, 64, 1); err == nil {
		t.Fatal("overwrote fixture")
	}
	r, err := store.OpenReader(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	counts, err := r.Counts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counts["span"] != 16 || counts["log"] != 16 || counts["metric"] != 32 {
		t.Fatal("lost fixture records", counts)
	}
}
