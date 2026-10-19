// ingestbench measures the M1 intake/storage path, not later failure benchmarks.
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/intake"
	"github.com/signalscout/signalscout/internal/store"
	logscol "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	metricscol "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	tracescol "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	common "go.opentelemetry.io/proto/otlp/common/v1"
	logs "go.opentelemetry.io/proto/otlp/logs/v1"
	metrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	resource "go.opentelemetry.io/proto/otlp/resource/v1"
	traces "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

type measurement struct {
	Kind      string  `json:"kind"`
	Seed      int64   `json:"seed"`
	Records   int     `json:"records"`
	Seconds   float64 `json:"seconds"`
	PerSecond float64 `json:"per_second"`
}
type summary struct {
	Mean   float64 `json:"mean"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	StdDev float64 `json:"stddev"`
}
type report struct {
	MeasuredAt     time.Time          `json:"measured_at"`
	GoVersion      string             `json:"go_version"`
	OS             string             `json:"os"`
	Arch           string             `json:"arch"`
	CPUs           int                `json:"logical_cpus"`
	CPU            string             `json:"cpu"`
	BatchSize      int                `json:"batch_size"`
	Method         string             `json:"method"`
	Measurements   []measurement      `json:"measurements"`
	Summary        map[string]summary `json:"summary"`
	SpanAcceptance bool               `json:"span_acceptance"`
}

func attr(key, value string) *common.KeyValue {
	return &common.KeyValue{Key: key, Value: &common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: value}}}
}

func payloads(kind string, count, batchSize int, seed int64) ([][]byte, error) {
	random := rand.New(rand.NewSource(seed))
	r := &resource.Resource{Attributes: []*common.KeyValue{attr("service.name", "ingest-benchmark")}}
	result := [][]byte{}
	for start := 0; start < count; start += batchSize {
		end := start + batchSize
		if end > count {
			end = count
		}
		spans := []*traces.Span{}
		records := []*logs.LogRecord{}
		points := []*metrics.NumberDataPoint{}
		for index := start; index < end; index++ {
			ts := uint64(1700000000000000000) + uint64(index)*1000000
			switch kind {
			case "span":
				traceID := make([]byte, 16)
				binary.BigEndian.PutUint64(traceID[:8], uint64(seed))
				binary.BigEndian.PutUint64(traceID[8:], uint64(index/10+1))
				spanID := make([]byte, 8)
				binary.BigEndian.PutUint64(spanID, uint64(index+1))
				var parent []byte
				if index%10 != 0 {
					parent = make([]byte, 8)
					binary.BigEndian.PutUint64(parent, uint64(index/10*10+1))
				}
				spans = append(spans, &traces.Span{TraceId: traceID, SpanId: spanID, ParentSpanId: parent, Name: "request", StartTimeUnixNano: ts, EndTimeUnixNano: ts + uint64(random.Intn(1000000)+1), Attributes: []*common.KeyValue{attr("http.request.method", "POST"), attr("http.route", "/benchmark")}})
			case "log":
				records = append(records, &logs.LogRecord{TimeUnixNano: ts, SeverityText: "INFO", Body: &common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: fmt.Sprintf("request %d completed in %d ms", index, random.Intn(100))}}})
			case "metric":
				points = append(points, &metrics.NumberDataPoint{TimeUnixNano: ts, Value: &metrics.NumberDataPoint_AsDouble{AsDouble: random.Float64()}})
			default:
				return nil, fmt.Errorf("unknown signal kind %q", kind)
			}
		}
		var message proto.Message
		switch kind {
		case "span":
			message = &tracescol.ExportTraceServiceRequest{ResourceSpans: []*traces.ResourceSpans{{Resource: r, ScopeSpans: []*traces.ScopeSpans{{Spans: spans}}}}}
		case "log":
			message = &logscol.ExportLogsServiceRequest{ResourceLogs: []*logs.ResourceLogs{{Resource: r, ScopeLogs: []*logs.ScopeLogs{{LogRecords: records}}}}}
		case "metric":
			message = &metricscol.ExportMetricsServiceRequest{ResourceMetrics: []*metrics.ResourceMetrics{{Resource: r, ScopeMetrics: []*metrics.ScopeMetrics{{Metrics: []*metrics.Metric{{Name: "queue.depth", Data: &metrics.Metric_Gauge{Gauge: &metrics.Gauge{DataPoints: points}}}}}}}}}
		}
		data, err := proto.Marshal(message)
		if err != nil {
			return nil, fmt.Errorf("encode payload: %w", err)
		}
		result = append(result, data)
	}
	return result, nil
}

func measure(ctx context.Context, kind string, count, batchSize int, seed int64) (result measurement, resultErr error) {
	data, err := payloads(kind, count, batchSize, seed)
	if err != nil {
		return result, err
	}
	directory, err := os.MkdirTemp("", "signalscout-ingest-")
	if err != nil {
		return result, fmt.Errorf("benchmark directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(directory); err != nil {
			resultErr = fmt.Errorf("remove benchmark database: %w", err)
		}
	}()
	s, err := store.Open(ctx, filepath.Join(directory, "ingest.db"), store.Options{QueueSize: 64, IdleTimeout: time.Hour, SweepInterval: time.Hour})
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
	endpoint := map[string]string{"span": "traces", "log": "logs", "metric": "metrics"}[kind]
	start := time.Now()
	for _, body := range data {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/v1/"+endpoint, bytes.NewReader(body))
		if err != nil {
			return result, fmt.Errorf("benchmark request: %w", err)
		}
		request.Header.Set("Content-Type", "application/x-protobuf")
		response, err := client.Do(request)
		if err != nil {
			return result, fmt.Errorf("benchmark intake: %w", err)
		}
		body, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr != nil {
			return result, fmt.Errorf("read response: %w", readErr)
		}
		if closeErr != nil {
			return result, fmt.Errorf("close response: %w", closeErr)
		}
		if response.StatusCode != http.StatusOK {
			return result, fmt.Errorf("intake returned %d: %s", response.StatusCode, body)
		}
	}
	elapsed := time.Since(start).Seconds()
	counts, err := s.Counts(ctx)
	if err != nil {
		return result, err
	}
	if counts[kind] != int64(count) {
		return result, fmt.Errorf("persisted %d of %d %s records", counts[kind], count, kind)
	}
	return measurement{Kind: kind, Seed: seed, Records: count, Seconds: elapsed, PerSecond: float64(count) / elapsed}, nil
}

func summarize(values []float64) summary {
	result := summary{Min: math.Inf(1)}
	for _, v := range values {
		result.Mean += v
		if v < result.Min {
			result.Min = v
		}
		if v > result.Max {
			result.Max = v
		}
	}
	result.Mean /= float64(len(values))
	for _, v := range values {
		result.StdDev += (v - result.Mean) * (v - result.Mean)
	}
	result.StdDev = math.Sqrt(result.StdDev / float64(len(values)))
	return result
}

func run(ctx context.Context, count, batchSize, seeds int, out string) error {
	if count <= 0 || batchSize <= 0 || seeds < 5 {
		return fmt.Errorf("count/batch-size must be positive; at least five seeds required")
	}
	r := report{MeasuredAt: time.Now().UTC(), GoVersion: runtime.Version(), OS: runtime.GOOS, Arch: runtime.GOARCH, CPUs: runtime.NumCPU(), CPU: os.Getenv("PROCESSOR_IDENTIFIER"), BatchSize: batchSize, Method: "Loopback HTTP protobuf, sequential batches, unique records, ten spans/trace, ambient logs and gauge points; includes decode, grouping and WAL SQLite commit (synchronous=NORMAL). Excludes payload generation, database setup, count verification and shutdown. Fresh temporary on-disk database per signal and seed.", Summary: map[string]summary{}}
	for _, kind := range []string{"span", "log", "metric"} {
		values := []float64{}
		for seed := int64(1); seed <= int64(seeds); seed++ {
			m, err := measure(ctx, kind, count, batchSize, seed)
			if err != nil {
				return err
			}
			r.Measurements = append(r.Measurements, m)
			values = append(values, m.PerSecond)
			fmt.Printf("%s seed=%d persisted=%d %.0f/s\n", kind, seed, count, m.PerSecond)
		}
		r.Summary[kind] = summarize(values)
	}
	r.SpanAcceptance = r.Summary["span"].Min >= 5000
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	if err = os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return fmt.Errorf("create results directory: %w", err)
	}
	if err = os.WriteFile(out, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("write results: %w", err)
	}
	var markdown bytes.Buffer
	fmt.Fprintf(&markdown, "# M1 ingest measurements\n\nGenerated %s by `task benchmark-ingest`.\n\n%s\n\nGo %s; %s/%s; %d logical CPUs; %s. Batch size %d, five or more fixed seeds. Rates are records per second; standard deviation is across seeds.\n\n| Signal | Mean /s | Min /s | Max /s | Std dev |\n|---|---:|---:|---:|---:|\n", r.MeasuredAt.Format(time.RFC3339), r.Method, r.GoVersion, r.OS, r.Arch, r.CPUs, r.CPU, r.BatchSize)
	for _, kind := range []string{"span", "log", "metric"} {
		s := r.Summary[kind]
		fmt.Fprintf(&markdown, "| %s | %.0f | %.0f | %.0f | %.0f |\n", kind, s.Mean, s.Min, s.Max, s.StdDev)
	}
	fmt.Fprintf(&markdown, "\nEvery span seed >= 5,000 spans/sec: **%v**. Persisted counts verified after each measurement.\n", r.SpanAcceptance)
	if err = os.WriteFile(strings.TrimSuffix(out, filepath.Ext(out))+".md", markdown.Bytes(), 0644); err != nil {
		return fmt.Errorf("write table: %w", err)
	}
	if !r.SpanAcceptance {
		return fmt.Errorf("span acceptance failed: minimum %.0f spans/sec", r.Summary["span"].Min)
	}
	return nil
}

func command() error {
	count := flag.Int("count", 20000, "Records per signal and seed")
	batch := flag.Int("batch-size", 256, "Records per HTTP request")
	seeds := flag.Int("seeds", 5, "Fixed seeds 1..N (N >= 5)")
	out := flag.String("out", filepath.Join("benchmark", "results", "ingest.json"), "Result JSON path")
	profilePath := flag.String("cpuprofile", "", "Optional CPU profile path for diagnosing a failed measurement")
	flag.Parse()
	if *profilePath != "" {
		file, err := os.Create(*profilePath)
		if err != nil {
			return fmt.Errorf("create CPU profile: %w", err)
		}
		defer func() { _ = file.Close() }()
		if err = pprof.StartCPUProfile(file); err != nil {
			return fmt.Errorf("start CPU profile: %w", err)
		}
		defer pprof.StopCPUProfile()
	}
	return run(context.Background(), *count, *batch, *seeds, *out)
}

func main() {
	if err := command(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
