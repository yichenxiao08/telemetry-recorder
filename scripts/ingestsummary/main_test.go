package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReportedMinimumMedianAndTarget(t *testing.T) {
	for _, tc := range []struct {
		values      []float64
		min, median float64
		met         bool
	}{
		{[]float64{12000, 4000, 10000}, 4000, 10000, false},
		{[]float64{10000, 12000, 11000, 13000}, 10000, 11500, true},
	} {
		got, err := summarize(tc.values, 10000)
		if err != nil {
			t.Fatal(err)
		}
		if got.Minimum != tc.min || got.Median != tc.median || got.Met != tc.met {
			t.Fatalf("misreported target: %+v", got)
		}
	}
	if _, err := summarize(nil, 10000); err == nil {
		t.Fatal("missing data accepted")
	}
}

func TestSummaryAcceptsOneUnchangedInvocation(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "input.json")
	var measurements []trial
	for _, kind := range []string{"span", "log", "metric"} {
		for i := 0; i < 5; i++ {
			measurements = append(measurements, trial{Kind: kind, PerSecond: float64(4000 + i*1000)})
		}
	}
	data, err := json.Marshal(map[string]any{"measurements": measurements})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(input, data, 0600); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(directory, "summary")
	if err = run([]string{input}, base, "Current code"); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(base + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var got report
	if err = json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	for kind, rate := range got.Signals {
		if rate.Trials != 5 || rate.Minimum != 4000 || rate.Median != 6000 || rate.Met {
			t.Fatalf("%s: %+v", kind, rate)
		}
	}
}

func TestSummaryReportsSimultaneousRates(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "combined.json")
	trials := []map[string]any{}
	for i := 0; i < 5; i++ {
		trials = append(trials, map[string]any{"per_second": map[string]float64{"span": 5000, "log": 5000, "metric": float64(8000 + i*1000)}})
	}
	data, err := json.Marshal(map[string]any{"measurements": trials})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(input, data, 0600); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(dir, "summary")
	if err = run([]string{input}, base, "Combined"); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(base + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var r report
	if err = json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	if !r.Signals["span"].Met || r.Signals["metric"].Minimum != 8000 || r.Signals["metric"].Median != 10000 || r.Signals["metric"].Met {
		t.Fatal("misreported mixed capacity", r)
	}
}
