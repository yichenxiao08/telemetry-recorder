// ingestsummary reports the three unchanged M1 review repetitions.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type trial struct {
	Kind      string  `json:"kind"`
	PerSecond float64 `json:"per_second"`
}
type rates struct {
	Trials  int     `json:"trials"`
	Minimum float64 `json:"minimum"`
	Median  float64 `json:"median"`
	Target  float64 `json:"target"`
	Met     bool    `json:"all_trials_met_target"`
}
type report struct {
	Sources []string         `json:"sources"`
	Method  string           `json:"method"`
	Signals map[string]rates `json:"signals"`
}

func summarize(values []float64, target float64) (rates, error) {
	if len(values) == 0 {
		return rates{}, fmt.Errorf("no measured values")
	}
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	median := ordered[len(ordered)/2]
	if len(ordered)%2 == 0 {
		median = (ordered[len(ordered)/2-1] + median) / 2
	}
	return rates{Trials: len(ordered), Minimum: ordered[0], Median: median, Target: target, Met: ordered[0] >= target}, nil
}

func run(inputs []string, base, title string) error {
	result := report{Method: "Minimum and median across all seed trials per signal in the listed unchanged task benchmark-ingest invocations. Signals measured separately; these rates are not simultaneous ingest capacity. Target is met only if every trial reaches it.", Signals: map[string]rates{}}
	values := map[string][]float64{}
	for _, path := range inputs {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read repetition: %w", err)
		}
		var source struct {
			Measurements []trial `json:"measurements"`
		}
		if err = json.Unmarshal(data, &source); err != nil {
			return fmt.Errorf("decode repetition: %w", err)
		}
		counts := map[string]int{}
		for _, m := range source.Measurements {
			if m.PerSecond <= 0 {
				return fmt.Errorf("invalid %s rate", m.Kind)
			}
			values[m.Kind] = append(values[m.Kind], m.PerSecond)
			counts[m.Kind]++
		}
		for _, kind := range []string{"span", "log", "metric"} {
			if counts[kind] != 5 {
				return fmt.Errorf("%s: expected five %s trials, got %d", path, kind, counts[kind])
			}
		}
		result.Sources = append(result.Sources, filepath.ToSlash(path))
	}
	targets := map[string]float64{"span": 5000, "log": 5000, "metric": 10000}
	for kind, target := range targets {
		s, err := summarize(values[kind], target)
		if err != nil {
			return err
		}
		result.Signals[kind] = s
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("encode summary: %w", err)
	}
	if err = os.WriteFile(base+".json", append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}
	var table bytes.Buffer
	fmt.Fprintf(&table, "# %s\n\n%s\n\n| Signal | Trials | Minimum /s | Median /s | Target /s | All trials met target |\n|---|---:|---:|---:|---:|---|\n", title, result.Method)
	for _, kind := range []string{"span", "log", "metric"} {
		s := result.Signals[kind]
		fmt.Fprintf(&table, "| %s | %d | %.0f | %.0f | %.0f | %v |\n", kind, s.Trials, s.Minimum, s.Median, s.Target, s.Met)
	}
	if err = os.WriteFile(base+".md", table.Bytes(), 0644); err != nil {
		return fmt.Errorf("write summary table: %w", err)
	}
	fmt.Print(table.String())
	return nil
}

func main() {
	inputs := flag.String("inputs", "benchmark/results/m1-review-1.json,benchmark/results/m1-review-2.json,benchmark/results/m1-review-3.json", "Comma-separated input reports")
	out := flag.String("out", "benchmark/results/m1-review-summary", "Output path without extension")
	title := flag.String("title", "M1 review: three unchanged repetitions", "Generated table title")
	flag.Parse()
	if err := run(strings.Split(*inputs, ","), *out, *title); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
