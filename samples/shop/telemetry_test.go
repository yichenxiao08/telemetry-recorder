package main

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/intake"
	"github.com/signalscout/signalscout/internal/store"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
)

func TestSDKExportsAllSignalsAndCorrelatesLogs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "sdk.db"), store.Options{QueueSize: 16, IdleTimeout: time.Second, SweepInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	}()
	server := httptest.NewServer(intake.New(s))
	defer server.Close()
	providers, err := newTelemetry(ctx, config{Role: "frontend", Endpoint: server.URL, Host: "docker-desktop", PoolSize: 20, Revision: "test", Phase: "baseline"}, "")
	if err != nil {
		t.Fatal(err)
	}
	spanCtx, span := providers.Traces.Tracer("test").Start(ctx, "request")
	span.SetAttributes(attribute.String("http.route", "/checkout"), attribute.String("http.request.method", "POST"))
	id := span.SpanContext().TraceID().String()
	emitLog(spanCtx, providers.Logs.Logger("test"), "request 42 completed", otellog.SeverityInfo)
	span.End()
	counter, err := providers.Metrics.Meter("test").Int64Counter("shop.requests")
	if err != nil {
		t.Fatal(err)
	}
	counter.Add(ctx, 1)
	if err = providers.Metrics.ForceFlush(ctx); err != nil {
		t.Fatal(err)
	}
	if err = providers.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	counts, err := s.Counts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counts["span"] != 1 || counts["log"] != 1 || counts["metric"] < 1 {
		t.Fatalf("SDK signals lost: %+v", counts)
	}
	detail, err := s.Show(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, signal := range detail.Signals {
		if signal.Kind == "log" {
			found = true
			if signal.TraceID != id || signal.SpanID == "" || signal.Body != "request 42 completed" || signal.Template != "request {} completed" {
				t.Fatalf("log correlation/body lost: %+v", signal)
			}
		}
	}
	if !found {
		t.Fatal("log was not attached to its trace run")
	}
}
