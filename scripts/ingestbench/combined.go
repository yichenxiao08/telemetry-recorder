package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/intake"
	"github.com/signalscout/signalscout/internal/store"
)

var targets = map[string]int{"span": 5000, "log": 5000, "metric": 10000}

type combinedMeasurement struct {
	Seed              int64              `json:"seed"`
	Window            float64            `json:"offered_window_seconds"`
	Elapsed           float64            `json:"seconds_including_drain"`
	Rates             map[string]float64 `json:"per_second"`
	CommittedInWindow map[string]int     `json:"committed_in_window"`
	Counts            map[string]int64   `json:"persisted"`
	Met               bool               `json:"all_targets_met"`
}

func combinedTrial(ctx context.Context, count, batch int, seed int64) (result combinedMeasurement, resultErr error) {
	if count <= 0 || batch <= 0 {
		return result, fmt.Errorf("positive count and batch required")
	}
	data := map[string][][]byte{}
	for _, kind := range []string{"span", "log", "metric"} {
		n := count * targets[kind] / 5000
		var err error
		data[kind], err = payloads(kind, n, batch, seed)
		if err != nil {
			return result, err
		}
	}
	dir, err := os.MkdirTemp("", "signalscout-combined-")
	if err != nil {
		return result, err
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			resultErr = err
		}
	}()
	s, err := store.Open(ctx, filepath.Join(dir, "ingest.db"), store.Options{QueueSize: 64, IdleTimeout: time.Hour, SweepInterval: time.Hour})
	if err != nil {
		return result, err
	}
	defer func() {
		if err := s.Close(); err != nil {
			resultErr = err
		}
	}()
	server := httptest.NewServer(intake.New(s))
	defer server.Close()
	client := server.Client()
	client.Timeout = 30 * time.Second
	window := time.Duration(float64(count) / 5000 * float64(time.Second))
	type completion struct {
		kind      string
		inWindow  int
		committed int
		err       error
	}
	completed := make(chan completion, 3)
	start := time.Now()
	// Each independent exporter awaits durable acknowledgments. Absolute pacing
	// offers the target rate without hiding writer lag in a growing client queue.
	for _, kind := range []string{"span", "log", "metric"} {
		go func() {
			c := completion{kind: kind}
			defer func() { completed <- c }()
			endpoint := map[string]string{"span": "traces", "log": "logs", "metric": "metrics"}[kind]
			for index, body := range data[kind] {
				if time.Since(start) >= window {
					break
				}
				due := start.Add(time.Duration(float64(index*batch) / float64(targets[kind]) * float64(time.Second)))
				if delay := time.Until(due); delay > 0 {
					timer := time.NewTimer(delay)
					select {
					case <-timer.C:
					case <-ctx.Done():
						timer.Stop()
						c.err = ctx.Err()
						return
					}
				}
				request, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/v1/"+endpoint, bytes.NewReader(body))
				if err != nil {
					c.err = err
					return
				}
				request.Header.Set("Content-Type", "application/x-protobuf")
				response, err := client.Do(request)
				if err != nil {
					c.err = err
					return
				}
				_, readErr := io.Copy(io.Discard, response.Body)
				closeErr := response.Body.Close()
				if readErr != nil || closeErr != nil || response.StatusCode != http.StatusOK {
					c.err = fmt.Errorf("%s status %d read=%v close=%v", kind, response.StatusCode, readErr, closeErr)
					return
				}
				if time.Since(start) <= window {
					n := min(batch, count*targets[kind]/5000-index*batch)
					c.inWindow += n
				}
				c.committed += min(batch, count*targets[kind]/5000-index*batch)
			}
		}()
	}
	result = combinedMeasurement{Seed: seed, Window: window.Seconds(), Rates: map[string]float64{}, CommittedInWindow: map[string]int{}, Met: true}
	acknowledged := map[string]int{}
	for range 3 {
		c := <-completed
		if c.err != nil {
			resultErr = c.err
		}
		result.CommittedInWindow[c.kind] = c.inWindow
		acknowledged[c.kind] = c.committed
	}
	result.Elapsed = max(window.Seconds(), time.Since(start).Seconds())
	if resultErr != nil {
		return result, resultErr
	}
	result.Counts, err = s.Counts(ctx)
	if err != nil {
		return result, err
	}
	for kind, target := range targets {
		n := count * target / 5000
		if result.Counts[kind] != int64(acknowledged[kind]) {
			return result, fmt.Errorf("%s persisted %d of %d acknowledged", kind, result.Counts[kind], acknowledged[kind])
		}
		result.Rates[kind] = float64(acknowledged[kind]) / result.Elapsed
		result.Met = result.Met && result.CommittedInWindow[kind] == n
	}
	return result, nil
}

func runCombined(ctx context.Context, count, batch, seeds int, out string) error {
	if seeds < 5 {
		return fmt.Errorf("at least five fixed seeds required")
	}
	r := struct {
		At           time.Time             `json:"measured_at"`
		Go           string                `json:"go"`
		Method       string                `json:"method"`
		Batch        int                   `json:"batch_size"`
		Measurements []combinedMeasurement `json:"measurements"`
	}{At: time.Now().UTC(), Go: runtime.Version(), Batch: batch, Method: "Simultaneous loopback protobuf exporters paced at 5000 spans, 5000 logs, 10000 scalar points/sec. One durable-acknowledged request per exporter; fixed seeds 1..5, fresh shared on-disk WAL/NORMAL store each seed, same payload generator and batch size as separate benchmark. Stop offering at the window deadline: unsent records count as capacity misses. All acknowledged counts verified. Rate denominator is max(offered window, final acknowledgment); committed_in_window also records deadline capacity. Setup, verification and shutdown excluded."}
	met := true
	var md strings.Builder
	fmt.Fprintln(&md, "# Simultaneous combined ingest\n\n"+r.Method+"\n\n| Seed | Seconds incl. drain | Spans/s | Logs/s | Points/s | All targets met |\n|---|---:|---:|---:|---:|---|")
	for seed := int64(1); seed <= int64(seeds); seed++ {
		m, err := combinedTrial(ctx, count, batch, seed)
		if err != nil {
			return err
		}
		r.Measurements = append(r.Measurements, m)
		met = met && m.Met
		fmt.Fprintf(&md, "| %d | %.3f | %.0f | %.0f | %.0f | %v |\n", seed, m.Elapsed, m.Rates["span"], m.Rates["log"], m.Rates["metric"], m.Met)
		fmt.Printf("combined seed=%d span=%.0f log=%.0f metric=%.0f met=%v\n", seed, m.Rates["span"], m.Rates["log"], m.Rates["metric"], m.Met)
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(out, append(data, '\n'), 0644); err != nil {
		return err
	}
	if err = os.WriteFile(strings.TrimSuffix(out, filepath.Ext(out))+".md", []byte(md.String()), 0644); err != nil {
		return err
	}
	if !met {
		return fmt.Errorf("combined ingest targets missed; measurements retained")
	}
	return nil
}
