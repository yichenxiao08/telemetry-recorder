package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/signalscout/signalscout/samples/shop/workload"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
)

// connection is the small part of pgxpool.Conn used by inventory. It allows
// timeout/retry tests without replacing the real database in the demo.
type connection interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Release()
}
type inventory struct {
	Acquire        func(context.Context) (connection, error)
	Tracer         trace.Tracer
	Logger         otellog.Logger
	AcquireTimeout time.Duration
}

func (i *inventory) reserve(ctx context.Context, cart workload.Cart) error {
	if workload.CacheHit(cart) {
		trace.SpanFromContext(ctx).AddEvent("cache.hit")
		return nil
	}
	trace.SpanFromContext(ctx).AddEvent("cache.miss")
	var conn connection
	for attempt := 0; attempt < 2; attempt++ {
		// Start evidence before the timed operation so span setup cannot consume
		// part of the acquire budget outside the recorded wait.
		spanCtx, span := i.Tracer.Start(ctx, "db.pool.acquire", trace.WithAttributes(attribute.Int("retry.attempt", attempt)))
		acquireCtx, cancel := context.WithTimeout(spanCtx, i.AcquireTimeout)
		var err error
		conn, err = i.Acquire(acquireCtx)
		cancel()
		if err == nil {
			span.End()
			break
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "connection pool acquire failed")
		span.End()
		if attempt == 1 || ctx.Err() != nil {
			return fmt.Errorf("acquire connection: %w", err)
		}
		trace.SpanFromContext(ctx).AddEvent("retry", trace.WithAttributes(attribute.Int("retry.attempt", 1)))
		emitLog(ctx, i.Logger, "connection pool acquire timed out; retrying", otellog.SeverityWarn)
		select {
		case <-time.After(10 * time.Millisecond):
		case <-ctx.Done():
			return fmt.Errorf("retry canceled: %w", ctx.Err())
		}
	}
	defer conn.Release()
	queryCtx, span := i.Tracer.Start(ctx, "inventory.db.reserve", trace.WithAttributes(attribute.String("db.system", "postgresql"), attribute.String("db.statement", "SELECT pg_sleep($1), count(*) FROM inventory WHERE sku = ANY($2)")))
	defer span.End()
	skus := make([]string, len(cart.Items))
	for index, item := range cart.Items {
		skus[index] = item.SKU
	}
	_, err := conn.Exec(queryCtx, "SELECT pg_sleep($1), count(*) FROM inventory WHERE sku = ANY($2)", workload.QueryDelay(cart).Seconds(), skus)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "inventory query failed")
		return fmt.Errorf("reserve inventory: %w", err)
	}
	return nil
}
