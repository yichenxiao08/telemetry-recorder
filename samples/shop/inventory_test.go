package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/signalscout/signalscout/samples/shop/workload"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/log/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type testConnection struct{ released, queries int }

func (c *testConnection) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	c.queries++
	return pgconn.NewCommandTag("SELECT 1"), nil
}
func (c *testConnection) Release() { c.released++ }

func TestAcquireTimeoutRetryAndEvidence(t *testing.T) {
	for _, recoverOnRetry := range []bool{false, true} {
		t.Run(map[bool]string{false: "failure", true: "recovery"}[recoverOnRetry], func(t *testing.T) {
			exporter := tracetest.NewInMemoryExporter()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
			defer func() { _ = provider.Shutdown(context.Background()) }()
			tracer := provider.Tracer("test")
			ctx, parent := tracer.Start(context.Background(), "inventory-service.reserve")
			calls := 0
			conn := &testConnection{}
			i := inventory{Tracer: tracer, Logger: noop.NewLoggerProvider().Logger("test"), AcquireTimeout: 5 * time.Millisecond, Acquire: func(ctx context.Context) (connection, error) {
				calls++
				if recoverOnRetry && calls == 2 {
					return conn, nil
				}
				<-ctx.Done()
				return nil, ctx.Err()
			}}
			cart := workload.Cart{ID: "bulk", Items: make([]workload.Item, 80)}
			for index := range cart.Items {
				cart.Items[index] = workload.Item{SKU: "sku-001", Quantity: 1}
			}
			err := i.reserve(ctx, cart)
			parent.End()
			if calls != 2 {
				t.Fatalf("acquired %d times", calls)
			}
			if recoverOnRetry {
				if err != nil || conn.queries != 1 || conn.released != 1 {
					t.Fatalf("recovery lost query/release: %v %+v", err, conn)
				}
			} else if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("wrong timeout: %v", err)
			}
			acquires, retries := 0, 0
			for _, span := range exporter.GetSpans() {
				if span.Name == "db.pool.acquire" {
					acquires++
					if span.Status.Code == codes.Error && span.EndTime.Sub(span.StartTime) < i.AcquireTimeout {
						t.Fatal("timeout evidence does not measure the real wait")
					}
				}
				for _, event := range span.Events {
					if event.Name == "retry" {
						retries++
					}
				}
			}
			if acquires != 2 || retries != 1 {
				t.Fatalf("missing acquire/retry evidence: %d %d", acquires, retries)
			}
		})
	}
}

func TestFingerprintIncludesPoolAndRevision(t *testing.T) {
	c := config{Role: "inventory-service", PoolSize: 5, Revision: "abc123", Phase: "failure", Host: "docker-desktop"}
	seen := map[string]string{}
	for _, attr := range c.resourceAttributes() {
		seen[string(attr.Key)] = attr.Value.Emit()
	}
	for key, want := range map[string]string{"vcs.revision": "abc123", "signalscout.config.DB_POOL_SIZE": "5", "host.name": "docker-desktop", "service.instance.id": "inventory-service-1", "signalscout.demo.phase": "failure"} {
		if seen[key] != want {
			t.Errorf("%s=%q, want %q", key, seen[key], want)
		}
	}
}
