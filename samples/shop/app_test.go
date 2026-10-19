package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/log/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestFrontendPropagatesTraceAndDownstreamFailure(t *testing.T) {
	traceparents := make(chan string, 1)
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceparents <- r.Header.Get("traceparent")
		http.Error(w, "pool timeout", http.StatusServiceUnavailable)
	}))
	defer downstream.Close()
	exporter := tracetest.NewInMemoryExporter()
	traces := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	metrics := sdkmetric.NewMeterProvider()
	defer func() { _ = traces.Shutdown(context.Background()); _ = metrics.Shutdown(context.Background()) }()
	a := app{config: config{Role: "frontend", Downstream: downstream.URL}, telemetry: &telemetry{Traces: traces, Metrics: metrics}, logger: noop.NewLoggerProvider().Logger("test"), health: func() poolHealth { return poolHealth{Ready: true} }}
	h := a.handler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(`{"id":"test","items":[{"sku":"sku-001","quantity":1}]}`))
	h.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || len(w.Header().Get("X-Trace-ID")) != 32 {
		t.Fatalf("lost failing run status/ID: %+v", w)
	}
	if !strings.Contains(<-traceparents, w.Header().Get("X-Trace-ID")) {
		t.Fatal("downstream request lost trace context")
	}
	found := false
	for _, span := range exporter.GetSpans() {
		if span.Name == "frontend.request" {
			found = true
			if span.Parent.IsValid() {
				t.Fatal("frontend request is not root")
			}
		}
	}
	if !found {
		t.Fatal("root span missing")
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/checkout", bytes.NewBufferString("malformed")))
	if w.Code != http.StatusBadRequest {
		t.Fatal("invalid cart forwarded")
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	data, _ := io.ReadAll(w.Result().Body)
	if w.Code != http.StatusOK || !bytes.Contains(data, []byte(`"ready":true`)) {
		t.Fatalf("bad health response: %s", data)
	}
}
