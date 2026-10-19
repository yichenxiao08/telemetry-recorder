package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyClosedPreservesSourceAndRefusesLiveOrExisting(t *testing.T) {
	dir := t.TempDir()
	source, target := filepath.Join(dir, "source.db"), filepath.Join(dir, "copy", "target.db")
	if err := os.WriteFile(source, []byte("fixture"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := copyClosed(source, target); err != nil {
		t.Fatal(err)
	}
	if err := copyClosed(source, target); err == nil {
		t.Fatal("overwrote existing database")
	}
	for _, path := range []string{source, target} {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "fixture" {
			t.Fatalf("copy changed data: %q %v", data, err)
		}
	}
	if err := os.WriteFile(source+"-wal", []byte("uncheckpointed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := copyClosed(source, filepath.Join(dir, "live.db")); err == nil {
		t.Fatal("copied a live WAL database")
	}
}
