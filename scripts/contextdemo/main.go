// contextdemo validates M4b against saved M2 datasets, without regenerating them.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/signalscout/signalscout/internal/diff"
	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
	"github.com/signalscout/signalscout/internal/telemetry"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type trial struct {
	RunID, Mode string
	PoolRank    int
	Above       []string
	ElapsedMS   float64
	Result      model.DiffResult
}
type demoReport struct {
	EngineRevision, SourceRevision string
	EngineDirty                    bool
	RepresentativePoolObserved     bool
	GaugeSamples                   []gaugeSample
	Seed                           int
	Database                       string
	Representative                 string
	Source                         string
	Trials                         []trial
	TraceLogs                      int
	ServiceStats                   []telemetry.ServiceStats
	BeforeAllPoolVisible           bool
}
type gaugeSample struct {
	ID            string
	TS            int64
	Value         float64
	PoolConfig    string
	BeforeFailEnd bool
}

func copyDatabase(source, dest string) error {
	if info, err := os.Stat(source + "-wal"); err == nil && info.Size() > 0 {
		return fmt.Errorf("source has a nonempty WAL")
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	closeErr := out.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func measure(ctx context.Context, seed int) (r demoReport, resultErr error) {
	r = demoReport{Seed: seed, BeforeAllPoolVisible: true}
	if revision, e := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output(); e == nil {
		r.EngineRevision = strings.TrimSpace(string(revision))
	}
	status, e := exec.CommandContext(ctx, "git", "status", "--porcelain", "--", "internal/diff", "internal/store", "internal/model", "internal/telemetry", "scripts/contextdemo").Output()
	r.EngineDirty = e != nil || len(status) > 0
	source := fmt.Sprintf("benchmark/results/shop-demo-seed-%d.json", seed)
	data, err := os.ReadFile(source)
	if err != nil {
		return r, err
	}
	var demo struct {
		Revision string   `json:"revision"`
		Database string   `json:"database"`
		Failures []string `json:"failure_run_ids"`
	}
	if err = json.Unmarshal(data, &demo); err != nil {
		return r, err
	}
	r.Source = demo.Database
	r.SourceRevision = demo.Revision
	r.Database = filepath.ToSlash(filepath.Join(".cache", "m4b", fmt.Sprintf("seed-%d.db", seed)))
	if _, err = os.Stat(r.Database); os.IsNotExist(err) {
		if err = copyDatabase(demo.Database, r.Database); err != nil {
			return r, err
		}
	}
	s, err := store.Open(ctx, r.Database, store.Options{QueueSize: 64, IdleTimeout: time.Hour, SweepInterval: time.Hour, ContextBefore: 30 * time.Second, ContextAfter: 5 * time.Second})
	if err != nil {
		return r, err
	}
	defer func() { resultErr = errors.Join(resultErr, s.Close()) }()
	latest := int64(0)
	for _, id := range demo.Failures {
		d, err := s.RunStructure(ctx, id, 5000)
		if err != nil {
			return r, err
		}
		if d.Run.StartedAt > latest {
			latest = d.Run.StartedAt
			r.Representative = id
		}
	}
	for _, id := range demo.Failures {
		d, err := s.RunStructure(ctx, id, 5000)
		if err != nil {
			return r, err
		}
		before, err := s.PhaseStart(ctx, d.Run)
		if err != nil {
			return r, err
		}
		for _, mode := range []string{"before", "default"} {
			opts := model.BaselineOptions{}
			if mode == "before" {
				opts.Before = before
			}
			start := time.Now()
			input, err := s.ContextComparison(ctx, id, opts)
			if err != nil {
				return r, err
			}
			result, err := diff.Compute(input)
			if err != nil {
				return r, err
			}
			if err = s.SaveDiff(ctx, result); err != nil {
				return r, err
			}
			t := trial{RunID: id, Mode: mode, ElapsedMS: float64(time.Since(start).Nanoseconds()) / 1e6, Result: result}
			for i, x := range result.Differences {
				if x.Type == "context_metric" && x.Name == "db.pool.in_use" {
					t.PoolRank = i + 1
					for _, above := range result.Differences[:i] {
						t.Above = append(t.Above, above.Type+": "+above.Name)
					}
					break
				}
			}
			if mode == "before" && t.PoolRank == 0 {
				r.BeforeAllPoolVisible = false
			}
			r.Trials = append(r.Trials, t)
		}
	}
	d, err := s.RunStructure(ctx, r.Representative, 5000)
	if err != nil {
		return r, err
	}
	logs, err := s.SearchTelemetry(ctx, telemetry.Filter{Kind: "log", TraceID: r.Representative, From: d.Run.StartedAt - 1e9, To: d.Run.EndedAt + 1e9, Limit: 1000})
	if err != nil {
		return r, err
	}
	r.TraceLogs = len(logs.Signals)
	if r.TraceLogs == 0 {
		return r, fmt.Errorf("seed %d trace log lookup returned no records", seed)
	}
	capacity, e := strconv.ParseFloat(fmt.Sprint(d.Run.Config["DB_POOL_SIZE"]), 64)
	if e != nil || capacity <= 0 {
		return r, fmt.Errorf("missing flat pool config")
	}
	for _, t := range r.Trials {
		if t.RunID == r.Representative && t.Mode == "before" {
			r.RepresentativePoolObserved = poolObserved(t.Result.Differences, capacity)
		}
	}
	if seed == 2 && !r.RepresentativePoolObserved {
		return r, fmt.Errorf("seed 2 representative pool-at-limit context difference is missing")
	}
	contextSignals, e := s.RunContext(ctx, r.Representative, d.Run.StartedAt-30e9, d.Run.EndedAt+5e9, 5000)
	if e != nil {
		return r, e
	}
	for _, point := range contextSignals {
		if point.Name == "db.pool.in_use" && point.Value != nil {
			r.GaugeSamples = append(r.GaugeSamples, gaugeSample{ID: point.ID, TS: point.TS, Value: *point.Value, PoolConfig: fmt.Sprint(point.Resource.Attrs["signalscout.config.DB_POOL_SIZE"]), BeforeFailEnd: point.TS <= d.Run.EndedAt})
		}
	}
	r.ServiceStats, err = s.ServiceOverview(ctx, telemetry.Filter{Kind: "span", Service: "frontend", From: latest - 60e9, To: d.Run.EndedAt + 1e9, Limit: 100})
	return r, err
}
func poolObserved(ds []model.Difference, capacity float64) bool {
	for _, d := range ds {
		if d.Type == "context_metric" && d.Name == "db.pool.in_use" && d.Value >= capacity && d.Median < d.Value && d.Score > 0 && d.Score <= 1 {
			return true
		}
	}
	return false
}
func run() error {
	out := flag.String("out", "benchmark/results/m4b-context-demo.json", "JSON result")
	flag.Parse()
	ctx := context.Background()
	all := []demoReport{}
	for _, seed := range []int{1, 2} {
		r, err := measure(ctx, seed)
		if err != nil {
			return err
		}
		all = append(all, r)
		data, err := json.MarshalIndent(all, "", "  ")
		if err != nil {
			return err
		}
		if err = os.WriteFile(*out, append(data, '\n'), 0644); err != nil {
			return err
		}
		fmt.Printf("seed=%d representative=%s before_all_pool_visible=%v trace_logs=%d\n", seed, r.Representative, r.BeforeAllPoolVisible, r.TraceLogs)
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
