// Read-only regression checks against the accepted synthetic stores/results.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/signalscout/signalscout/internal/cohort"
	"github.com/signalscout/signalscout/internal/diff"
	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
)

type shopReport struct {
	Seed     int
	Database string
	Trials   []struct{ Result model.DiffResult }
}
type check struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
	Hash  string `json:"canonical_results_sha256"`
}

func shop(ctx context.Context, report shopReport, withContext bool) (check, error) {
	r, err := store.OpenReader(ctx, report.Database)
	if err != nil {
		return check{}, err
	}
	defer func() { _ = r.Close() }()
	result := check{Name: fmt.Sprintf("shop seed %d context=%t", report.Seed, withContext)}
	digest := sha256.New()
	for _, trial := range report.Trials {
		want := trial.Result
		spec := want.BaselineSpec
		opts := model.BaselineOptions{N: spec.N, ExplicitN: spec.ExplicitN, Before: spec.BeforeInput}
		var input model.Comparison
		if withContext {
			input, err = r.ContextComparison(ctx, want.FailingRunID, opts)
		} else {
			input, err = r.Comparison(ctx, want.FailingRunID, opts)
		}
		if err != nil {
			return result, err
		}
		got, e := diff.Compute(input)
		if e != nil {
			return result, e
		}
		// Only wall-clock measurement metadata differs by definition. IDs,
		// selections, ranks, scores, summaries and all evidence must match.
		got.CreatedAt, got.ElapsedNS = want.CreatedAt, want.ElapsedNS
		a, _ := json.Marshal(got)
		b, _ := json.Marshal(want)
		if !bytes.Equal(a, b) {
			return result, fmt.Errorf("shop result changed: %s/%s", want.FailingRunID, spec.Mode)
		}
		_, _ = digest.Write(a)
		result.Count++
	}
	result.Hash = hex.EncodeToString(digest.Sum(nil))
	return result, nil
}
func run() error {
	before := flag.String("before", ".cache/declared-relations/signalscout-main.exe", "main's executable")
	after := flag.String("after", "bin/signalscout.exe", "new executable")
	db := flag.String("db", ".cache/declared-relations/lane01-copy.db", "copy of frozen lane store")
	out := flag.String("out", ".cache/declared-relations/regression.json", "result outside frozen files")
	flag.Parse()
	ctx := context.Background()
	checks := []check{}
	for seed := 1; seed <= 2; seed++ {
		raw, err := os.ReadFile(fmt.Sprintf("benchmark/results/diff-seed-%d.json", seed))
		if err != nil {
			return err
		}
		var r shopReport
		if err = json.Unmarshal(raw, &r); err != nil {
			return err
		}
		c, err := shop(ctx, r, false)
		if err != nil {
			return err
		}
		checks = append(checks, c)
	}
	raw, err := os.ReadFile("benchmark/results/lane-01-shop-context.json")
	if err != nil {
		return err
	}
	var reports []shopReport
	if err = json.Unmarshal(raw, &reports); err != nil {
		return err
	}
	for _, r := range reports {
		c, e := shop(ctx, r, true)
		if e != nil {
			return e
		}
		checks = append(checks, c)
	}
	raw, err = os.ReadFile("benchmark/results/lane-01.json")
	if err != nil {
		return err
	}
	var frozen struct {
		Cases []struct{ Result cohort.Result }
	}
	if err = json.Unmarshal(raw, &frozen); err != nil {
		return err
	}
	trace := check{Name: "lane-01 trace CLI JSON and text"}
	digest := sha256.New()
	for _, c := range frozen.Cases {
		if len(c.Result.Problem.RunIDs) == 0 {
			return fmt.Errorf("empty frozen cohort")
		}
		id := c.Result.Problem.DecisionIDs[c.Result.Problem.RunIDs[0]]
		for _, jsonFlag := range []bool{false, true} {
			var outputs [][]byte
			for _, binary := range []string{*before, *after} {
				absolute, e := filepath.Abs(binary)
				if e != nil {
					return e
				}
				args := []string{"--db", *db, "trace", id, "--profile", "benchmark/problems/lane-01/profile.json", "--from", "2026-01-01T00:00:00Z", "--to", "2027-01-01T00:00:00Z", "--limit", "200"}
				if jsonFlag {
					args = append(args, "--json")
				}
				data, e := exec.CommandContext(ctx, absolute, args...).Output()
				if e != nil {
					return e
				}
				outputs = append(outputs, data)
			}
			if !bytes.Equal(outputs[0], outputs[1]) {
				return fmt.Errorf("trace bytes changed: %s", id)
			}
			_, _ = digest.Write(outputs[0])
			trace.Count++
		}
	}
	trace.Hash = hex.EncodeToString(digest.Sum(nil))
	checks = append(checks, trace)
	data, err := json.MarshalIndent(map[string]any{"synthetic": true, "checks": checks, "shop_metadata_exclusion": "Only created_at and elapsed_ns vary by execution; all other serialized result bytes must match frozen reports.", "trace_comparison": "Entire CLI output bytes, no normalization."}, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(*out, append(data, '\n'), 0600); err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
