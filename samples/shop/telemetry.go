package main

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type telemetry struct {
	Traces  *sdktrace.TracerProvider
	Metrics *sdkmetric.MeterProvider
	Logs    *sdklog.LoggerProvider
}

func newTelemetry(ctx context.Context, c config, databaseVersion string) (*telemetry, error) {
	attrs := append(c.resourceAttributes(), attribute.String("process.runtime.version", runtime.Version()))
	if databaseVersion != "" {
		attrs = append(attrs, attribute.String("dependency.postgresql.version", databaseVersion))
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/jackc/pgx/v5" || dep.Path == "go.opentelemetry.io/otel" || dep.Path == "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp" {
				attrs = append(attrs, attribute.String("dependency."+dep.Path+".version", dep.Version))
			}
		}
	}
	r := resource.NewSchemaless(attrs...)
	traceExporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(c.Endpoint))
	if err != nil {
		return nil, fmt.Errorf("trace exporter: %w", err)
	}
	metricExporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL(c.Endpoint))
	if err != nil {
		_ = traceExporter.Shutdown(ctx)
		return nil, fmt.Errorf("metric exporter: %w", err)
	}
	logExporter, err := otlploghttp.New(ctx, otlploghttp.WithEndpointURL(c.Endpoint))
	if err != nil {
		_ = traceExporter.Shutdown(ctx)
		_ = metricExporter.Shutdown(ctx)
		return nil, fmt.Errorf("log exporter: %w", err)
	}
	// Providers are wired explicitly; no process-global OTel providers are changed.
	return &telemetry{
		Traces:  sdktrace.NewTracerProvider(sdktrace.WithResource(r), sdktrace.WithSampler(sdktrace.AlwaysSample()), sdktrace.WithBatcher(traceExporter, sdktrace.WithBatchTimeout(200*time.Millisecond))),
		Metrics: sdkmetric.NewMeterProvider(sdkmetric.WithResource(r), sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(time.Second)))),
		Logs:    sdklog.NewLoggerProvider(sdklog.WithResource(r), sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter, sdklog.WithExportInterval(200*time.Millisecond)))),
	}, nil
}

func (t *telemetry) Shutdown(ctx context.Context) error {
	return errors.Join(t.Traces.Shutdown(ctx), t.Metrics.Shutdown(ctx), t.Logs.Shutdown(ctx))
}

func emitLog(ctx context.Context, logger otellog.Logger, body string, severity otellog.Severity) {
	var record otellog.Record
	record.SetTimestamp(time.Now())
	record.SetBody(otellog.StringValue(body))
	record.SetSeverity(severity)
	if severity >= otellog.SeverityError {
		record.SetSeverityText("ERROR")
	} else if severity >= otellog.SeverityWarn {
		record.SetSeverityText("WARN")
	} else {
		record.SetSeverityText("INFO")
	}
	logger.Emit(ctx, record)
}

func registerPoolMetrics(provider *sdkmetric.MeterProvider, pool *pgxpool.Pool) error {
	meter := provider.Meter("signalscout/shop")
	inUse, err := meter.Int64ObservableGauge("db.pool.in_use", metric.WithUnit("{connection}"))
	if err != nil {
		return fmt.Errorf("pool gauge: %w", err)
	}
	capacity, err := meter.Int64ObservableGauge("db.pool.capacity", metric.WithUnit("{connection}"))
	if err != nil {
		return fmt.Errorf("pool capacity: %w", err)
	}
	// pgxpool.Stat returns a concurrency-safe snapshot; callbacks do not mutate it.
	_, err = meter.RegisterCallback(func(_ context.Context, observer metric.Observer) error {
		stats := pool.Stat()
		observer.ObserveInt64(inUse, int64(stats.AcquiredConns()))
		observer.ObserveInt64(capacity, int64(stats.MaxConns()))
		return nil
	}, inUse, capacity)
	if err != nil {
		return fmt.Errorf("register pool metrics: %w", err)
	}
	return nil
}
