package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/intake"
	"github.com/signalscout/signalscout/internal/store"
	"github.com/signalscout/signalscout/internal/viewer"
	"github.com/spf13/cobra"
)

func newServeCommand(database *string) *cobra.Command {
	options := store.Options{}
	var address string
	var thresholds []string
	command := &cobra.Command{Use: "serve", Short: "Receive OTLP telemetry and serve the local viewer", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		options.Thresholds = map[string]float64{}
		for _, value := range thresholds {
			workload, duration, ok := strings.Cut(value, "=")
			if !ok || workload == "" {
				return fmt.Errorf("threshold must be workload=duration")
			}
			parsed, err := time.ParseDuration(duration)
			if err != nil {
				return fmt.Errorf("threshold %s: %w", value, err)
			}
			if parsed <= 0 {
				return fmt.Errorf("threshold must be positive")
			}
			options.Thresholds[workload] = float64(parsed) / float64(time.Millisecond)
		}
		if options.QueueSize <= 0 || options.IdleTimeout <= 0 || options.SweepInterval <= 0 || options.Retention < 0 || options.ContextBefore < 0 || options.ContextAfter < 0 {
			return fmt.Errorf("queue, idle timeout and sweep interval must be positive; retention cannot be negative")
		}
		logger := slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), nil))
		return serve(cmd.Context(), *database, address, options, logger)
	}}
	flags := command.Flags()
	flags.StringVar(&address, "listen", "127.0.0.1:4318", "OTLP and viewer HTTP listen address")
	flags.IntVar(&options.QueueSize, "queue-size", 64, "Maximum queued batches")
	flags.DurationVar(&options.IdleTimeout, "idle-timeout", 5*time.Second, "Close a run after inactivity")
	flags.DurationVar(&options.Retention, "retention", 7*24*time.Hour, "Evidence retention; 0 disables cleanup")
	flags.DurationVar(&options.SweepInterval, "sweep-interval", time.Second, "Idle and retention maintenance interval")
	flags.DurationVar(&options.ContextBefore, "context-before", 30*time.Second, "Ambient context padding before the run")
	flags.DurationVar(&options.ContextAfter, "context-after", 5*time.Second, "Ambient context padding after the run")
	flags.StringArrayVar(&thresholds, "threshold", nil, "Degrade workload above duration, e.g. 'POST /checkout=2s'; repeatable")
	return command
}

func serve(ctx context.Context, database, address string, options store.Options, logger *slog.Logger) (result error) {
	s, err := store.Open(ctx, database, options)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, s.Close()) }()
	view, err := viewer.New(s.Reader, s, logger)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle("/v1/", intake.New(s))
	mux.Handle("/", view)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, MaxHeaderBytes: 1 << 20}
	// A buffered result channel lets Serve finish even if shutdown wins the select.
	finished := make(chan error, 1)
	go func() { finished <- server.Serve(listener) }()
	logger.Info("OTLP and viewer HTTP listening", "address", listener.Addr().String(), "viewer", "http://"+listener.Addr().String()+"/runs", "database", database, "queue_size", options.QueueSize)
	select {
	case <-ctx.Done():
	case err = <-s.Errors():
		result = fmt.Errorf("storage writer: %w", err)
		logger.Error("storage writer failed", "error", err)
	case err = <-finished:
		if !errors.Is(err, http.ErrServerClosed) {
			result = fmt.Errorf("HTTP server: %w", err)
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = server.Shutdown(shutdownCtx); err != nil {
		result = errors.Join(result, fmt.Errorf("HTTP shutdown: %w", err), server.Close())
	}
	return result
}
