package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/signalscout/signalscout/internal/store"
	"os"
	"strings"
	"time"
)

type ingestProfile struct {
	Name      string         `json:"name"`
	Rationale string         `json:"rationale"`
	Targets   map[string]int `json:"targets"`
}

func validateProfile(p ingestProfile) error {
	for _, k := range []string{"span", "log", "metric"} {
		if p.Targets[k] <= 0 || p.Targets[k] > 100000 {
			return fmt.Errorf("%s: invalid %s target", p.Name, k)
		}
	}
	return nil
}
func runProfiles(ctx context.Context, path, out string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var profiles []ingestProfile
	if err = json.Unmarshal(data, &profiles); err != nil {
		return err
	}
	if len(profiles) == 0 {
		return fmt.Errorf("no profiles")
	}
	type result struct {
		Profile      ingestProfile         `json:"profile"`
		Method       string                `json:"method"`
		Measurements []combinedMeasurement `json:"measurements"`
	}
	all := []result{}
	met := true
	var md strings.Builder
	fmt.Fprintln(&md, "# Revised combined profiles\n\nTargets were changed by the owner from round-number capacity goals to workload-derived budgets. Original misses remain in docs/performance.md. Pipeline is an estimate.\n\n| Profile | Seed | Spans/s | Logs/s | Points/s | All targets met |\n|---|---:|---:|---:|---:|---|")
	for _, p := range profiles {
		if err = validateProfile(p); err != nil {
			return err
		}
		r := result{Profile: p, Method: "30-second offered window; seeds 1..5; same payload generator, batch 256, four bounded exporters (1 span, 1 log, 2 metric). Serve defaults: queue64, idle5s, sweep1s, context30s/5s, retention7d; WAL/NORMAL, acknowledgment after commit. Every intended record must commit before deadline; unsent work is a miss. All acknowledged counts verified; rate denominator includes final drain."}
		counts := map[string]int{}
		for k, v := range p.Targets {
			counts[k] = v * 30
		}
		for seed := int64(1); seed <= 5; seed++ {
			m, e := combinedTrialRates(ctx, counts, 30*time.Second, p.Targets, 256, seed, store.Options{QueueSize: 64, IdleTimeout: 5 * time.Second, SweepInterval: time.Second, ContextBefore: 30 * time.Second, ContextAfter: 5 * time.Second, Retention: 7 * 24 * time.Hour})
			if e != nil {
				m.Error = e.Error()
				m.Met = false
			}
			met = met && m.Met
			r.Measurements = append(r.Measurements, m)
			fmt.Fprintf(&md, "| %s | %d | %.0f | %.0f | %.0f | %v |\n", p.Name, seed, m.Rates["span"], m.Rates["log"], m.Rates["metric"], m.Met)
			fmt.Printf("%s seed=%d met=%v\n", p.Name, seed, m.Met)
		}
		all = append(all, r)
		data, err = json.MarshalIndent(all, "", "  ")
		if err != nil {
			return err
		}
		if err = os.WriteFile(out, append(data, '\n'), 0644); err != nil {
			return err
		}
	}
	if err = os.WriteFile(strings.TrimSuffix(out, ".json")+".md", []byte(md.String()), 0644); err != nil {
		return err
	}
	if !met {
		return fmt.Errorf("revised combined targets missed; results retained")
	}
	return nil
}
