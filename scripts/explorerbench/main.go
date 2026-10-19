// explorerbench measures the actual bounded M4b reader APIs on the 1M fixture.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/signalscout/signalscout/internal/store"
	"github.com/signalscout/signalscout/internal/telemetry"
	"os"
	"time"
)

type measurement struct {
	Name   string           `json:"name"`
	Filter telemetry.Filter `json:"filter"`
	MS     []float64        `json:"milliseconds"`
	Rows   int              `json:"rows"`
	Met    bool             `json:"all_under_500ms"`
}

func run() error {
	path := flag.String("db", ".cache/performance/million.db", "Existing seeded 1M fixture")
	out := flag.String("out", "benchmark/results/m4b-million-queries.json", "Results")
	flag.Parse()
	ctx := context.Background()
	s, err := store.Open(ctx, *path, store.Options{QueueSize: 64, IdleTimeout: time.Hour, SweepInterval: time.Hour, ContextBefore: 30 * time.Second, ContextAfter: 5 * time.Second})
	if err != nil {
		return err
	}
	if err = s.Close(); err != nil {
		return err
	}
	base := telemetry.Filter{Service: "service-02", From: 1700123200000000000, To: 1700126800000000000, Limit: 100}
	results := []measurement{}
	met := true
	rc, err := store.OpenReader(ctx, *path)
	if err != nil {
		return err
	}
	counts, err := rc.Counts(ctx)
	closeErr := rc.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	total := int64(0)
	for _, n := range counts {
		total += n
	}
	if total != 1000000 {
		return fmt.Errorf("fixture has %d signals, expected 1000000", total)
	}
	for _, name := range []string{"spans_service_time", "logs_service_time", "metrics_service_time", "metrics_without_service", "attribute_filter", "trace_spans", "trace_logs", "template_counts", "service_overview"} {
		f := base
		f.Kind = "span"
		switch name {
		case "logs_service_time", "trace_logs", "template_counts":
			f.Kind = "log"
		case "metrics_service_time", "metrics_without_service":
			f.Kind = "metric"
		}
		if name == "metrics_without_service" {
			f.Service = ""
			f.NoService = true
		}
		if name == "attribute_filter" {
			f.Attribute = "http.route"
			f.AttributeValue = "/benchmark"
			minimum := 5.0
			f.DurationMin = &minimum
			f.Status = "ok"
		}
		if name == "trace_spans" || name == "trace_logs" {
			f.TraceID = "000000000000000100000000000030d5"
			f.Service = ""
		}
		m := measurement{Name: name, Filter: f, Met: true}
		// A new pool for each query's first read. OS caches are not flushed. Repeat
		// clocks include row JSON/resource decoding and Go-side overview statistics.
		r, err := store.OpenReader(ctx, *path)
		if err != nil {
			return err
		}
		for range 5 {
			start := time.Now()
			n := 0
			switch name {
			case "template_counts":
				xs, e := r.LogTemplates(ctx, f)
				err = e
				n = len(xs)
			case "service_overview":
				xs, e := r.ServiceOverview(ctx, f)
				err = e
				n = len(xs)
			default:
				page, e := r.SearchTelemetry(ctx, f)
				err = e
				n = len(page.Signals)
			}
			ms := float64(time.Since(start).Nanoseconds()) / 1e6
			if err != nil {
				_ = r.Close()
				return err
			}
			m.MS = append(m.MS, ms)
			m.Rows = n
			m.Met = m.Met && ms < 500
			if n == 0 {
				_ = r.Close()
				return fmt.Errorf("%s returned no fixture rows", name)
			}
		}
		if err = r.Close(); err != nil {
			return err
		}
		met = met && m.Met
		results = append(results, m)
	}
	report := struct {
		Database, Method string
		Queries          []measurement
	}{*path, "Existing seeded 1M-row real-schema fixture, migrated through 008; actual read APIs, time bounds and LIMIT100. Five repeats; first connection-cold, OS caches unflushed. Includes decoding, aggregation and iteration. No concurrent measurement workload.", results}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(*out, append(data, '\n'), 0644); err != nil {
		return err
	}
	fmt.Println(string(data))
	if !met {
		return fmt.Errorf("one or more explorer queries missed 500ms")
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
