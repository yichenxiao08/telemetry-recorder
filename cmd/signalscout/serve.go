package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/httpdrain"
	"github.com/signalscout/signalscout/internal/intake"
	"github.com/signalscout/signalscout/internal/redaction"
	"github.com/signalscout/signalscout/internal/store"
	"github.com/signalscout/signalscout/internal/viewer"
	"github.com/spf13/cobra"
)

func newServeCommand(database *string) *cobra.Command {
	options := store.Options{}
	var address string
	var thresholds []string
	var policyPath, profile string
	var publicExperiment bool
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
		var policy *redaction.Policy
		var policyDigest string
		if policyPath != "" {
			f, err := os.Open(policyPath)
			if err != nil {
				return err
			}
			data, err := io.ReadAll(io.LimitReader(f, 65537))
			_ = f.Close()
			if err != nil {
				return err
			}
			var key []byte
			if raw := os.Getenv("SIGNALSCOUT_P0_HMAC_KEY"); raw != "" {
				key, err = base64.StdEncoding.DecodeString(raw)
				if err != nil {
					return fmt.Errorf("P0 key must be base64")
				}
			}
			policy, err = redaction.Parse(data, key)
			if err != nil {
				return err
			}
			if !policy.HasProfile(profile) {
				return fmt.Errorf("known --capture-profile required")
			}
			digest := sha256.Sum256(data)
			policyDigest = hex.EncodeToString(digest[:])
		} else if profile != "" {
			return fmt.Errorf("--capture-profile requires --capture-policy")
		}
		if publicExperiment {
			if err := validatePublicExperiment(*database, address, policy, profile); err != nil {
				return err
			}
			// Reserve the database atomically; never open a prior store in this mode.
			f, err := os.OpenFile(*database, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
			if err != nil {
				return fmt.Errorf("public experiment requires a new database: %w", err)
			}
			if err = f.Close(); err != nil {
				return err
			}
			return servePublicPolicy(cmd.Context(), *database, address, options, logger, policy, profile, policyDigest)
		}
		return servePolicy(cmd.Context(), *database, address, options, logger, policy, profile)
	}}
	flags := command.Flags()
	flags.StringVar(&policyPath, "capture-policy", "", "Enable strict synthetic P0 capture with this validated JSON policy (legacy mode otherwise)")
	flags.StringVar(&profile, "capture-profile", "", "Explicit P0 capture profile")
	flags.BoolVar(&publicExperiment, "public-data-experiment", false, "Experimental local capture of explicitly approved public data; numeric policy required")
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
	return servePolicy(ctx, database, address, options, logger, nil, "")
}

func servePolicy(ctx context.Context, database, address string, options store.Options, logger *slog.Logger, policy *redaction.Policy, profile string) (result error) {
	return servePublicPolicy(ctx, database, address, options, logger, policy, profile, "")
}

func validatePublicExperiment(database, address string, policy *redaction.Policy, profile string) error {
	if !filepath.IsAbs(database) || policy == nil || !policy.HasProfile(profile) {
		return fmt.Errorf("public experiment requires absolute new database and strict policy/profile")
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return fmt.Errorf("public experiment requires a literal loopback listen address")
	}
	fields := policy.Profiles[profile].Fields
	if len(fields) == 0 {
		return fmt.Errorf("public experiment requires numeric field allowlist")
	}
	for path, rule := range fields {
		if rule.Kind != "number" || (!strings.HasPrefix(path, "inputs.") && path != "output" && !strings.HasPrefix(path, "output.")) {
			return fmt.Errorf("public experiment permits numeric input/output fields only")
		}
	}
	return nil
}

func servePublicPolicy(ctx context.Context, database, address string, options store.Options, logger *slog.Logger, policy *redaction.Policy, profile, publicDigest string) (result error) {
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
	intakeHandler := intake.NewWithPolicy(s, policy, profile)
	if publicDigest != "" {
		mux.Handle("/v1/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-SignalScout-Public-Policy") != publicDigest || r.Header.Get("X-SignalScout-Public-Profile") != profile {
				httpdrain.Error(w, r, "public policy mismatch", http.StatusForbidden)
				return
			}
			intakeHandler.ServeHTTP(w, r)
		}))
		mux.HandleFunc("/public-capture-policy", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_ = json.NewEncoder(w).Encode(map[string]string{"mode": "public-data-experiment", "profile": profile, "sha256": publicDigest})
		})
	} else {
		mux.Handle("/v1/", intakeHandler)
	}
	metadata := metadataHandler(s)
	if policy != nil {
		mux.Handle("/api/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" && r.Method != "HEAD" {
				httpdrain.Error(w, r, "P0 metadata writes disabled; annotation text has no approved capture policy", http.StatusForbidden)
				return
			}
			metadata.ServeHTTP(w, r)
		}))
	} else {
		mux.Handle("/api/", metadata)
	}
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
