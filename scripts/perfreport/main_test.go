package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComparisonUsesMeasuredMinimumMedianAndFailure(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "summary.md")
	data := `{"signals":{"span":{"minimum":4000,"median":6000,"target":5000,"all_trials_met_target":false},"log":{"minimum":5000,"median":7000,"target":5000,"all_trials_met_target":true},"metric":{"minimum":8000,"median":10000,"target":10000,"all_trials_met_target":false}}}`
	if err := os.WriteFile(filepath.Join(dir, "perf-00-summary.json"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generate(dir, out); err != nil {
		t.Fatal(err)
	}
	result, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "span | 4000 | 6000 | 5000 | false") || !strings.Contains(string(result), "metric | 8000 | 10000 | 10000 | false") {
		t.Fatal("misreported target")
	}
}
