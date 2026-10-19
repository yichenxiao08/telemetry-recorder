// workloadrates measures saved demo load phases; it never opens a writer.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "modernc.org/sqlite"
)

func peak(times []int64) int {
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	best, left := 0, 0
	for right, t := range times {
		for t-times[left] >= int64(time.Second) {
			left++
		}
		best = max(best, right-left+1)
	}
	return best
}

type phase struct {
	Seed    int                `json:"seed"`
	Phase   string             `json:"phase"`
	Start   int64              `json:"start_ns"`
	End     int64              `json:"end_ns"`
	Counts  map[string]int     `json:"counts"`
	Average map[string]float64 `json:"average_per_second"`
	Peak    map[string]int     `json:"peak_sliding_one_second"`
}
type profile struct {
	Name      string         `json:"name"`
	Rationale string         `json:"rationale"`
	Targets   map[string]int `json:"targets"`
}

func measure(ctx context.Context, report string) ([]phase, error) {
	data, err := os.ReadFile(report)
	if err != nil {
		return nil, err
	}
	var source struct {
		Seed     int
		Database string
	}
	if err = json.Unmarshal(data, &source); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(source.Database)+"?mode=ro&_pragma=query_only(1)")
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()
	var out []phase
	for _, name := range []string{"baseline", "failure"} {
		p := phase{Seed: source.Seed, Phase: name, Counts: map[string]int{}, Average: map[string]float64{}, Peak: map[string]int{}}
		err = db.QueryRowContext(ctx, `SELECT min(started_at),max(ended_at) FROM runs WHERE json_extract(env,'$."signalscout.demo.phase"')=?`, name).Scan(&p.Start, &p.End)
		if err != nil {
			return nil, err
		}
		rows, err := db.QueryContext(ctx, `SELECT kind,ts_ns FROM signals WHERE kind IN ('span','log','metric') AND ts_ns>=? AND ts_ns<=? AND (kind!='metric' OR json_type(data,'$.value') IN ('integer','real')) ORDER BY ts_ns`, p.Start, p.End)
		if err != nil {
			return nil, err
		}
		times := map[string][]int64{}
		for rows.Next() {
			var k string
			var t int64
			if err = rows.Scan(&k, &t); err != nil {
				break
			}
			times[k] = append(times[k], t)
		}
		if err == nil {
			err = rows.Err()
		}
		_ = rows.Close()
		if err != nil {
			return nil, err
		}
		for _, k := range []string{"span", "log", "metric"} {
			p.Counts[k] = len(times[k])
			p.Average[k] = float64(len(times[k])) / (float64(p.End-p.Start) / 1e9)
			p.Peak[k] = peak(times[k])
		}
		out = append(out, p)
	}
	return out, nil
}
func run() error {
	out := flag.String("out", "benchmark/results/m4b-workload-rates.json", "Measured phase rates")
	flag.Parse()
	all := []phase{}
	targets := map[string]int{"span": 1, "log": 1, "metric": 1}
	for _, seed := range []int{1, 2} {
		p, err := measure(context.Background(), fmt.Sprintf("benchmark/results/shop-demo-seed-%d.json", seed))
		if err != nil {
			return err
		}
		all = append(all, p...)
		for _, phase := range p {
			for k, v := range phase.Peak {
				targets[k] = max(targets[k], 2*v)
			}
		}
	}
	report := struct {
		Method string  `json:"method"`
		Phases []phase `json:"phases"`
	}{"Event-time records within each phase's earliest run start through latest run end. All resources (including host-only) included by timestamp, scalar metric values only. Peak = maximum count in any half-open sliding 1s window, not phase average. Targets = twice the largest per-signal peak across both phases and both saved seeds; maxima may occur at different times. Export delivery timestamps are not available, so this is workload emission rate, not exporter burst rate.", all}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(*out, append(data, '\n'), 0644); err != nil {
		return err
	}
	frame := map[string]int{"span": targets["span"], "log": targets["log"], "metric": max(3000, targets["metric"])}
	profiles := []profile{{"shop-2x-peak", "Measured saved seeds 1 and 2; twice each maximum sliding-one-second signal count.", targets}, {"pipeline-60fps-estimate", "Estimate, not a measured workload: 60 frames/sec x 50 scalar values = 3000 points/sec; retain shop 2x span/log traffic concurrently, and use the larger metric budget.", frame}}
	data, err = json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("benchmark/results/m4b-ingest-targets.json", append(data, '\n'), 0644)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
