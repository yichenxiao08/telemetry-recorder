// diffdemo validates only M3 against an existing M2 shop database. It is not
// the multi-failure benchmark harness reserved for M5.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/diff"
	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
)

type ranks struct {
	Pool    int `json:"pool_config"`
	Retry   int `json:"retry_presence"`
	Latency int `json:"pool_acquire_latency"`
	Decoy   int `json:"cache_miss_presence"`
}

type trial struct {
	RunID               string           `json:"run_id"`
	Mode                string           `json:"mode"`
	Ranks               ranks            `json:"ranks"`
	BaselinePools       map[string]int   `json:"baseline_pool_sizes"`
	BaselineCacheMisses int              `json:"baseline_cache_miss_runs"`
	WindowAcceptance    *bool            `json:"window_acceptance,omitempty"`
	Result              model.DiffResult `json:"result"`
}

type report struct {
	MeasuredAt     time.Time `json:"measured_at"`
	Revision       string    `json:"revision"`
	EngineDirty    bool      `json:"engine_dirty"`
	DemoRevision   string    `json:"demo_revision"`
	Seed           int64     `json:"seed"`
	Source         string    `json:"source"`
	Database       string    `json:"database"`
	Representative string    `json:"representative_run_id"`
	FailureStart   string    `json:"failure_phase_start"`
	FailingPool    string    `json:"flat_failing_pool_size"`
	ScoringVersion string    `json:"scoring_version"`
	MaximumMS      float64   `json:"maximum_40_run_diff_ms"`
	Trials         []trial   `json:"trials"`
	Acceptance     bool      `json:"acceptance"`
}

func findRanks(differences []model.Difference) ranks {
	var result ranks
	for i, d := range differences {
		switch {
		case d.Type == "config" && d.Name == "DB_POOL_SIZE":
			result.Pool = i + 1
		case d.Type == "presence" && d.Name == "retry":
			result.Retry = i + 1
		case d.Type == "latency" && d.Name == "db.pool.acquire":
			result.Latency = i + 1
		case d.Type == "presence" && d.Name == "cache.miss":
			result.Decoy = i + 1
		}
	}
	return result
}

func windowAccepted(result model.DiffResult, r ranks, cacheMisses int) bool {
	if result.Status != "ok" || len(result.BaselineSpec.RunIDs) != 40 || result.ElapsedNS >= int64(time.Second) || cacheMisses <= 0 {
		return false
	}
	for _, rank := range []int{r.Pool, r.Retry, r.Latency} {
		if rank < 1 || rank > 3 || r.Decoy <= rank {
			return false
		}
	}
	return true
}

func portable(path string) (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("workspace: %w", err)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("result path: %w", err)
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil {
		return "", fmt.Errorf("relative result path: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("result path is outside repository")
	}
	return filepath.ToSlash(relative), nil
}

func gitOutput(ctx context.Context, args ...string) (string, error) {
	data, err := exec.CommandContext(ctx, "git", args...).Output()
	if err != nil {
		return "", fmt.Errorf("git metadata: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func inspectBaseline(input model.Comparison) (map[string]int, int, error) {
	pools := map[string]int{}
	cacheMisses := 0
	for _, baseline := range input.Baseline {
		pool, ok := baseline.Run.Config["DB_POOL_SIZE"]
		if !ok {
			return pools, cacheMisses, fmt.Errorf("baseline %s has no flat DB_POOL_SIZE", baseline.Run.ID)
		}
		pools[fmt.Sprint(pool)]++
		for _, s := range baseline.Signals {
			if s.Kind == "event" && s.Name == "cache.miss" {
				cacheMisses++
				break
			}
		}
	}
	return pools, cacheMisses, nil
}

func validate(ctx context.Context, source string) (result report, resultErr error) {
	data, err := os.ReadFile(source)
	if err != nil {
		return result, fmt.Errorf("read demo report: %w", err)
	}
	var demo struct {
		Database   string `json:"database"`
		Seed       int64  `json:"seed"`
		Revision   string `json:"revision"`
		Acceptance bool   `json:"acceptance"`
	}
	if err = json.Unmarshal(data, &demo); err != nil {
		return result, fmt.Errorf("decode demo report: %w", err)
	}
	if !demo.Acceptance {
		return result, fmt.Errorf("demo acceptance did not pass")
	}
	result.Database, err = portable(demo.Database)
	if err != nil {
		return result, err
	}
	result.Source, err = portable(source)
	if err != nil {
		return result, err
	}
	result.Revision, err = gitOutput(ctx, "rev-parse", "HEAD")
	if err != nil {
		return result, err
	}
	dirty, err := gitOutput(ctx, "status", "--porcelain", "--", "internal/diff", "internal/stats", "internal/store", "internal/model", "internal/runs", "cmd/signalscout", "go.mod", "go.sum")
	if err != nil {
		return result, err
	}
	result.EngineDirty = dirty != ""
	result.Seed = demo.Seed
	result.DemoRevision = demo.Revision
	result.ScoringVersion = model.DiffScoringVersion
	reader, err := store.OpenReader(ctx, result.Database)
	if err != nil {
		return result, err
	}
	defer func() {
		if err := reader.Close(); err != nil {
			resultErr = err
		}
	}()
	list, err := reader.List(ctx, "POST /checkout", "")
	if err != nil {
		return result, err
	}
	var failures []model.Run
	var failureStart int64
	for _, run := range list {
		if run.Env["signalscout.demo.phase"] == "failure" && (failureStart == 0 || run.StartedAt < failureStart) {
			failureStart = run.StartedAt
		}
		if run.Closed && run.Outcome == "fail" {
			failures = append(failures, run)
		}
	}
	if len(failures) == 0 || failureStart == 0 {
		return result, fmt.Errorf("missing failing runs or failure-phase fingerprint")
	}
	result.Representative = failures[0].ID
	pool, ok := failures[0].Config["DB_POOL_SIZE"]
	if !ok {
		return result, fmt.Errorf("failure has no flat DB_POOL_SIZE")
	}
	result.FailingPool = fmt.Sprint(pool)
	result.FailureStart = time.Unix(0, failureStart).UTC().Format(time.RFC3339Nano)
	result.Acceptance = true
	for _, run := range failures {
		for _, mode := range []string{"before", "default"} {
			options := model.BaselineOptions{}
			if mode == "before" {
				options.Before = result.FailureStart
			}
			input, err := reader.Comparison(ctx, run.ID, options)
			if err != nil {
				return result, err
			}
			pools, cacheMisses, err := inspectBaseline(input)
			if err != nil {
				return result, err
			}
			computed, err := diff.AnalyzeStructural(ctx, result.Database, run.ID, options)
			if err != nil {
				return result, err
			}
			t := trial{RunID: run.ID, Mode: mode, Result: computed, Ranks: findRanks(computed.Differences), BaselinePools: pools, BaselineCacheMisses: cacheMisses}
			milliseconds := float64(computed.ElapsedNS) / float64(time.Millisecond)
			if milliseconds > result.MaximumMS {
				result.MaximumMS = milliseconds
			}
			if computed.Status != "ok" || len(computed.BaselineSpec.RunIDs) != 40 || computed.ElapsedNS >= int64(time.Second) {
				result.Acceptance = false
			}
			if mode == "before" {
				accepted := windowAccepted(computed, t.Ranks, cacheMisses)
				t.WindowAcceptance = &accepted
				if fmt.Sprint(input.Failing.Run.Config["DB_POOL_SIZE"]) != "5" || pools["20"] != 40 {
					accepted = false
				}
				if !accepted {
					result.Acceptance = false
				}
			}
			result.Trials = append(result.Trials, t)
		}
	}
	result.MeasuredAt = time.Now().UTC()
	return result, nil
}

func markdown(result report) []byte {
	var text bytes.Buffer
	fmt.Fprintf(&text, "# M3 shop diff validation: seed %d\n\nGenerated from `%s`; database `%s`. Engine revision `%s` (dirty=%t); demo revision `%s`. Scoring %s was fixed before ranking.\n\nRepresentative: latest failing run `%s`; flat DB_POOL_SIZE=%s. Failure-phase cutoff: `%s`.\n\nAll %d comparisons select 40 baselines. Maximum elapsed time including selection, evidence loading, extraction, scoring, persistence and connection closure: **%.3f ms**. Compilation/process startup excluded. Acceptance: **%t**.\n\n", result.Seed, result.Source, result.Database, result.Revision, result.EngineDirty, result.DemoRevision, result.ScoringVersion, result.Representative, result.FailingPool, result.FailureStart, len(result.Trials), result.MaximumMS, result.Acceptance)
	fmt.Fprintln(&text, "| Run | Mode | Pool rank | Retry rank | Acquire rank | Cache-miss rank | Baseline cache misses | Time ms | Window accepted |\n|---|---|---:|---:|---:|---:|---:|---:|---|")
	for _, t := range result.Trials {
		accepted := "n/a"
		if t.WindowAcceptance != nil {
			accepted = fmt.Sprint(*t.WindowAcceptance)
		}
		fmt.Fprintf(&text, "| %s | %s | %d | %d | %d | %d | %d/40 | %.3f | %s |\n", t.RunID, t.Mode, t.Ranks.Pool, t.Ranks.Retry, t.Ranks.Latency, t.Ranks.Decoy, t.BaselineCacheMisses, float64(t.Result.ElapsedNS)/float64(time.Millisecond), accepted)
	}
	fmt.Fprintln(&text, "\nRank 0 means absent from the ranked output. Window acceptance requires the three requested differences in the top 3, a real cache-miss decoy below them, and time below one second. Default mode is reported without requiring a pool-size difference.")
	for _, t := range result.Trials {
		if t.RunID != result.Representative {
			continue
		}
		pools, _ := json.Marshal(t.BaselinePools)
		fmt.Fprintf(&text, "\n## Representative: %s\n\nBaseline pool sizes: `%s`; cache misses %d/40.\n\n| Rank | Type | Feature | Score | Raw baseline rate |\n|---:|---|---|---:|---:|\n", t.Mode, pools, t.BaselineCacheMisses)
		for i, d := range t.Result.Differences {
			fmt.Fprintf(&text, "| %d | %s | %s | %.6f | %.4f |\n", i+1, d.Type, d.Name, d.Score, d.BaselineRate)
		}
	}
	for _, t := range result.Trials {
		if t.RunID == result.Representative && t.Mode == "default" {
			if t.Ranks.Pool == 0 {
				fmt.Fprintln(&text, "\nThe latest failure's pool-size snapshot agrees with its recent passing baseline mode, so the pool-size config difference correctly disappears.")
			} else {
				fmt.Fprintf(&text, "\nThe latest failure's pool-size config difference remains at rank %d; see the actual baseline pool sizes above.\n", t.Ranks.Pool)
			}
		}
	}
	fmt.Fprintln(&text, "\nThe explicit window uses the pre-change phase. Cache misses and pool acquisitions also occur in passing requests; the tables retain their raw baseline rates and ranks. Earlier failures can have a mixed default baseline and a pool-size difference. No context metric or context log differences are computed.")
	return text.Bytes()
}

func run(ctx context.Context, source, out string) error {
	result, err := validate(ctx, source)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("encode validation: %w", err)
	}
	if err = os.WriteFile(out+".json", append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("write validation: %w", err)
	}
	if err = os.WriteFile(out+".md", markdown(result), 0644); err != nil {
		return fmt.Errorf("write table: %w", err)
	}
	fmt.Printf("Seed %d: acceptance=%t, %d comparisons, max %.3f ms; latest failure %s\n", result.Seed, result.Acceptance, len(result.Trials), result.MaximumMS, result.Representative)
	for _, t := range result.Trials {
		if t.RunID == result.Representative {
			fmt.Printf("%s ranks: pool=%d retry=%d acquire=%d cache.miss=%d; baseline pools=%v\n", t.Mode, t.Ranks.Pool, t.Ranks.Retry, t.Ranks.Latency, t.Ranks.Decoy, t.BaselinePools)
		}
	}
	if !result.Acceptance {
		return fmt.Errorf("M3 acceptance missed; results retained in %s.json", out)
	}
	return nil
}

func main() {
	source := flag.String("demo", "benchmark/results/shop-demo.json", "Existing accepted M2 demo report")
	out := flag.String("out", "benchmark/results/diff-demo", "Output path without extension")
	flag.Parse()
	if err := run(context.Background(), *source, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
