package main

import (
	"github.com/signalscout/signalscout/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyPreservesSourceAndRefusesOverwriteOrLiveWAL(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.db")
	target := filepath.Join(dir, "copy.db")
	if err := os.WriteFile(source, []byte("saved"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := copyDatabase(source, target); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "saved" {
		t.Fatalf("%s %v", data, err)
	}
	if copyDatabase(source, target) == nil {
		t.Fatal("overwrote existing copy")
	}
	if err = os.WriteFile(source+"-wal", []byte("live"), 0644); err != nil {
		t.Fatal(err)
	}
	if copyDatabase(source, filepath.Join(dir, "new.db")) == nil {
		t.Fatal("copied live WAL database")
	}
}
func TestPoolDetectionRequiresObservedDifference(t *testing.T) {
	for _, tc := range []struct {
		value, median float64
		want          bool
	}{{5, 0, true}, {0, 0, false}, {5, 5, false}} {
		ds := []model.Difference{{Type: "context_metric", Name: "db.pool.in_use", Value: tc.value, Median: tc.median, Score: .9}}
		if poolObserved(ds, 5) != tc.want {
			t.Fatal(tc)
		}
	}
	if poolObserved(nil, 5) {
		t.Fatal("invented unobserved peak")
	}
}
