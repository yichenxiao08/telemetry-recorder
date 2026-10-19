package intake

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
	logscol "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	metricscol "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	tracescol "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type captured struct {
	batch model.Batch
	err   error
	calls int
}

func (c *captured) Submit(_ context.Context, b model.Batch) error {
	c.calls++
	c.batch = b
	return c.err
}

func fixtureData(t *testing.T, kind string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "otlp", kind+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func send(handler http.Handler, path, contentType string, data []byte, compressed bool) *httptest.ResponseRecorder {
	if compressed {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, _ = gz.Write(data)
		_ = gz.Close()
		data = buf.Bytes()
	}
	r := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	r.Header.Set("Content-Type", contentType)
	if compressed {
		r.Header.Set("Content-Encoding", "gzip")
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func TestAllFormatsAndGzip(t *testing.T) {
	for _, kind := range []string{"traces", "logs", "metrics"} {
		for _, format := range []string{"json", "protobuf"} {
			for _, gzipEnabled := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/gzip=%v", kind, format, gzipEnabled), func(t *testing.T) {
					data := fixtureData(t, kind)
					contentType := "application/json; charset=utf-8"
					if format == "protobuf" {
						var message proto.Message
						switch kind {
						case "traces":
							message = &tracescol.ExportTraceServiceRequest{}
						case "logs":
							message = &logscol.ExportLogsServiceRequest{}
						case "metrics":
							message = &metricscol.ExportMetricsServiceRequest{}
						}
						fixed, err := otlpJSON(data)
						if err != nil {
							t.Fatal(err)
						}
						if err = protojson.Unmarshal(fixed, message); err != nil {
							t.Fatal(err)
						}
						data, err = proto.Marshal(message)
						if err != nil {
							t.Fatal(err)
						}
						contentType = "application/x-protobuf"
					}
					queue := &captured{}
					w := send(New(queue), "/v1/"+kind, contentType, data, gzipEnabled)
					if w.Code != 200 {
						t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
					}
					if len(queue.batch.Signals) == 0 {
						t.Fatal("no records decoded")
					}
					if format == "json" && w.Header().Get("Content-Type") != "application/json" {
						t.Fatal("wrong JSON response content type")
					}
					for _, s := range queue.batch.Signals {
						if !json.Valid(s.Raw) || !json.Valid(s.Resource.Raw) || !json.Valid(s.Scope) {
							t.Fatalf("lost raw metadata: %+v", s)
						}
					}
				})
			}
		}
	}
}

func TestBackpressureAndMalformed(t *testing.T) {
	queue := &captured{err: store.QueueFull}
	handler := New(queue)
	w := send(handler, "/v1/traces", "application/json", fixtureData(t, "traces"), false)
	if w.Code != 503 || w.Header().Get("Retry-After") == "" {
		t.Fatalf("backpressure response: %+v", w)
	}
	queue.err = nil
	queue.calls = 0
	for _, tc := range []struct {
		path, ct, body string
		code           int
	}{
		{"/v1/traces", "application/json", "{", 400},
		{"/v1/logs", "application/x-protobuf", "bad-wire", 400},
		{"/v1/metrics", "text/plain", "{}", 415},
		{"/v1/traces", "application/json", `{"resourceSpans":[{"scopeSpans":[{"spans":[{"traceId":"01","spanId":"01"}]}]}]}`, 400},
		{"/v1/logs", "application/json", "{} {}", 400},
		{"/missing", "application/json", "{}", 404},
	} {
		w = send(handler, tc.path, tc.ct, []byte(tc.body), false)
		if w.Code != tc.code {
			t.Errorf("%s got %d, want %d", tc.body, w.Code, tc.code)
		}
	}
	if queue.calls != 0 {
		t.Fatal("malformed requests reached queue")
	}
	r := httptest.NewRequest(http.MethodGet, "/v1/traces", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 405 {
		t.Fatal("GET accepted")
	}
	w = send(handler, "/v1/logs", "application/json", bytes.Repeat([]byte(" "), MaxBodyBytes+1), true)
	if w.Code != 413 {
		t.Fatalf("gzip expansion got %d", w.Code)
	}
	r = httptest.NewRequest(http.MethodPost, "/v1/logs", strings.NewReader("bad gzip"))
	r.Header.Set("Content-Encoding", "gzip")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatal("invalid gzip accepted")
	}
}

type goldenRun struct {
	ID       string `json:"id"`
	Workload string `json:"workload"`
	Domain   string `json:"domain"`
	Outcome  string `json:"outcome"`
	Closed   bool   `json:"closed"`
	Spans    int    `json:"spans"`
	Events   int    `json:"events"`
	Logs     int    `json:"logs"`
	Metrics  int    `json:"metrics"`
	Context  int    `json:"context"`
}

func TestFixtureReplayGolden(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "fixtures.db"), store.Options{QueueSize: 4, IdleTimeout: time.Second, SweepInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	}()
	h := New(s)
	// Ambient context arrives first; repeat complete requests to test stable identities.
	for replay := 0; replay < 2; replay++ {
		for _, kind := range []string{"logs", "metrics", "traces"} {
			w := send(h, "/v1/"+kind, "application/json", fixtureData(t, kind), false)
			if w.Code != 200 {
				t.Fatalf("%s: %s", kind, w.Body.String())
			}
		}
	}
	if err = s.Maintain(ctx, time.Now().Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	list, err := s.List(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	actual := []goldenRun{}
	for _, run := range list {
		detail, err := s.Show(ctx, run.ID)
		if err != nil {
			t.Fatal(err)
		}
		g := goldenRun{ID: run.ID, Workload: run.Workload, Domain: run.Domain, Outcome: run.Outcome, Closed: run.Closed, Context: len(detail.Context)}
		for _, signal := range detail.Signals {
			switch signal.Kind {
			case "span":
				g.Spans++
			case "event":
				g.Events++
			case "log":
				g.Logs++
			case "metric":
				g.Metrics++
			}
		}
		actual = append(actual, g)
	}
	sort.Slice(actual, func(i, j int) bool { return actual[i].ID < actual[j].ID })
	var expected []goldenRun
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "otlp", "runs.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, expected) {
		encoded, _ := json.MarshalIndent(actual, "", "  ")
		t.Fatalf("golden mismatch:\n%s", encoded)
	}
	// A malformed request must leave persisted state untouched.
	w := send(h, "/v1/traces", "application/json", []byte("malformed"), false)
	if w.Code != 400 {
		t.Fatal("malformed accepted")
	}
	after, err := s.List(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, list) {
		t.Fatal("malformed request changed storage")
	}
}
