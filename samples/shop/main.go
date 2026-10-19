package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func run(ctx context.Context) (result error) {
	c, err := readConfig()
	if err != nil {
		return err
	}
	var pool *pgxpool.Pool
	var version string
	if c.Role == "inventory-service" {
		settings, err := pgxpool.ParseConfig(c.DatabaseURL)
		if err != nil {
			return fmt.Errorf("database configuration: %w", err)
		}
		settings.MaxConns = int32(c.PoolSize)
		pool, err = pgxpool.NewWithConfig(ctx, settings)
		if err != nil {
			return fmt.Errorf("open pool: %w", err)
		}
		defer pool.Close()
		if err = pool.QueryRow(ctx, "SHOW server_version").Scan(&version); err != nil {
			return fmt.Errorf("database readiness: %w", err)
		}
	}
	t, err := newTelemetry(ctx, c, version)
	if err != nil {
		return err
	}
	defer func() {
		flush, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		result = errors.Join(result, t.Shutdown(flush))
	}()
	a := &app{config: c, telemetry: t, logger: t.Logs.Logger("signalscout/shop"), health: func() poolHealth { return poolHealth{Ready: true} }}
	if pool != nil {
		if err = registerPoolMetrics(t.Metrics, pool); err != nil {
			return err
		}
		a.inventory = &inventory{Tracer: t.Traces.Tracer("signalscout/shop"), Logger: a.logger, AcquireTimeout: 100 * time.Millisecond, Acquire: func(ctx context.Context) (connection, error) { return pool.Acquire(ctx) }}
		a.health = func() poolHealth {
			stats := pool.Stat()
			return poolHealth{Ready: true, InUse: stats.AcquiredConns(), Capacity: stats.MaxConns()}
		}
	}
	server := &http.Server{Addr: c.Address, Handler: a.handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	finished := make(chan error, 1)
	// Server result is buffered so shutdown cannot leave a goroutine blocked.
	go func() { finished <- server.ListenAndServe() }()
	slog.Info("shop service listening", "service", c.Role, "address", c.Address, "pool_size", c.PoolSize, "phase", c.Phase)
	select {
	case <-ctx.Done():
	case err = <-finished:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve shop: %w", err)
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = server.Shutdown(shutdown); err != nil {
		_ = server.Close()
		return fmt.Errorf("shop shutdown: %w", err)
	}
	return nil
}

func main() {
	health := flag.Bool("healthcheck", false, "Check this container's HTTP readiness")
	flag.Parse()
	if *health {
		client := &http.Client{Timeout: time.Second}
		response, err := client.Get("http://127.0.0.1:8080/healthz")
		if err != nil {
			os.Exit(1)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		return
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx); err != nil {
		slog.Error("shop stopped", "error", err)
		os.Exit(1)
	}
}
