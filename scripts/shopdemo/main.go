// shopdemo orchestrates Docker from a native Go process and verifies the stored
// M2 evidence. SQLite stays on the host; containers communicate only over OTLP.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/signalscout/signalscout/internal/intake"
	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
	"github.com/signalscout/signalscout/samples/shop/load"
)

type evidence struct {
	RunID          string               `json:"run_id"`
	AcquireID      string               `json:"acquire_span_id"`
	AcquireMS      float64              `json:"acquire_ms"`
	RetryID        string               `json:"retry_event_id"`
	HostMetricID   string               `json:"host_metric_id"`
	HostMetricName string               `json:"host_metric_name"`
	HostResourceID string               `json:"host_resource_id"`
	HostMatches    []model.ContextMatch `json:"host_matches"`
	PoolGaugeID    string               `json:"pool_gauge_id"`
	TraceLogID     string               `json:"trace_log_id"`
}
type report struct {
	CompletedAt      time.Time        `json:"completed_at"`
	Revision         string           `json:"revision"`
	SourceDirty      bool             `json:"source_dirty"`
	Database         string           `json:"database"`
	Seed             int64            `json:"seed"`
	BaselineRequests int              `json:"baseline_requests"`
	BaselinePassing  int              `json:"baseline_passing"`
	FailureRequests  int              `json:"failure_requests"`
	FailureCount     int              `json:"failure_count"`
	FailurePercent   float64          `json:"failure_percent"`
	FailureRunIDs    []string         `json:"failure_run_ids"`
	Signals          map[string]int64 `json:"signals"`
	Evidence         evidence         `json:"evidence"`
	Acceptance       bool             `json:"acceptance"`
}

type runner struct {
	environment []string
	log         *os.File
}

func (r runner) command(ctx context.Context, name string, args ...string) error {
	fmt.Println(name, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = r.environment
	cmd.Stdout = r.log
	cmd.Stderr = r.log
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s (see %s): %w", name, strings.Join(args, " "), r.log.Name(), err)
	}
	return nil
}
func (r runner) compose(ctx context.Context, args ...string) error {
	prefix := []string{"compose", "--project-name", "signalscout-shop", "--file", filepath.Join("samples", "shop", "compose.yml")}
	return r.command(ctx, "docker", append(prefix, args...)...)
}

func output(ctx context.Context, name string, args ...string) (string, error) {
	data, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %s: %w", name, strings.Join(args, " "), data, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func pause(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

func waitHealth(ctx context.Context, url string, capacity int) error {
	deadline, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 2 * time.Second}
	for {
		r, err := http.NewRequestWithContext(deadline, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("readiness request: %w", err)
		}
		response, err := client.Do(r)
		if err == nil {
			var health struct {
				Capacity int `json:"capacity"`
			}
			if capacity > 0 {
				err = json.NewDecoder(response.Body).Decode(&health)
			} else {
				_, err = io.Copy(io.Discard, response.Body)
			}
			_ = response.Body.Close()
			if err == nil && response.StatusCode == http.StatusOK && (capacity == 0 || capacity == health.Capacity) {
				return nil
			}
		}
		if err = pause(deadline, 100*time.Millisecond); err != nil {
			return fmt.Errorf("readiness %s: %w", url, err)
		}
	}
}

func waitRecorded(ctx context.Context, s *store.Store, result load.Result, poolSize int) error {
	deadline, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	for {
		list, err := s.List(deadline, "POST /checkout", "")
		if err != nil {
			return err
		}
		byID := map[string]model.Run{}
		for _, run := range list {
			byID[run.ID] = run
		}
		ready := true
		for _, request := range result.Records {
			run, ok := byID[request.TraceID]
			if !ok || !run.Closed {
				ready = false
				break
			}
			want := "pass"
			if request.Status >= 500 {
				want = "fail"
			}
			if run.Outcome != want {
				return fmt.Errorf("run %s outcome %s, HTTP result requires %s", run.ID, run.Outcome, want)
			}
			if fmt.Sprint(run.Config["DB_POOL_SIZE"]) != fmt.Sprint(poolSize) || run.Env["signalscout.demo.phase"] != result.Phase {
				return fmt.Errorf("run %s has wrong pool/phase fingerprint", run.ID)
			}
		}
		if ready {
			return nil
		}
		if err = pause(deadline, 200*time.Millisecond); err != nil {
			return fmt.Errorf("waiting for %s runs to finalize: %w", result.Phase, err)
		}
	}
}

func inspect(detail model.RunDetail) (evidence, error) {
	e := evidence{RunID: detail.Run.ID}
	spanResources := map[string]bool{}
	for _, signal := range detail.Signals {
		if signal.Kind == "span" {
			spanResources[signal.Resource.ID] = true
		}
		if signal.Kind == "span" && signal.Name == "db.pool.acquire" && signal.End-signal.TS >= int64(80*time.Millisecond) {
			e.AcquireID = signal.ID
			e.AcquireMS = float64(signal.End-signal.TS) / float64(time.Millisecond)
		}
		if signal.Kind == "event" && signal.Name == "retry" {
			e.RetryID = signal.ID
		}
		if signal.Kind == "log" && signal.TraceID == detail.Run.ID {
			e.TraceLogID = signal.ID
		}
	}
	for _, signal := range detail.Context {
		if signal.Name == "db.pool.in_use" && signal.Value != nil && fmt.Sprint(signal.Resource.Attrs["signalscout.config.DB_POOL_SIZE"]) == "5" {
			e.PoolGaugeID = signal.ID
		}
		if signal.Kind != "metric" || signal.Value == nil || !strings.HasPrefix(signal.Name, "system.") || len(signal.Resource.Attrs) != 1 || signal.Resource.Attrs["host.name"] != "docker-desktop" || spanResources[signal.Resource.ID] {
			continue
		}
		for _, match := range signal.ContextMatches {
			if match.Rule == "host.name" && spanResources[match.ResourceID] {
				e.HostMetricID = signal.ID
				e.HostMetricName = signal.Name
				e.HostResourceID = signal.Resource.ID
				e.HostMatches = signal.ContextMatches
				break
			}
		}
	}
	if detail.Run.Outcome != "fail" || e.AcquireID == "" || e.RetryID == "" || e.HostMetricID == "" || e.PoolGaugeID == "" || e.TraceLogID == "" {
		return e, fmt.Errorf("incomplete failure/context evidence: %+v", e)
	}
	return e, nil
}

func demo(ctx context.Context, seed int64, failureCount int) (result report, resultErr error) {
	if failureCount < 100 || failureCount%100 != 0 {
		return result, fmt.Errorf("failure-count must be a positive multiple of 100")
	}
	engine, err := output(ctx, "docker", "info", "--format", "{{.OSType}}/{{.Architecture}}")
	if err != nil {
		return result, err
	}
	arch := "amd64"
	if engine == "linux/aarch64" || engine == "linux/arm64" {
		arch = "arm64"
	} else if engine != "linux/x86_64" && engine != "linux/amd64" {
		return result, fmt.Errorf("demo requires a Linux Docker engine, got %s", engine)
	}
	result.Seed = seed
	result.Revision, err = output(ctx, "git", "rev-parse", "HEAD")
	if err != nil {
		return result, err
	}
	dirty, err := output(ctx, "git", "status", "--porcelain", "--", "samples/shop", "scripts/shopdemo", "scripts/shopload", "go.mod", "go.sum", "Taskfile.yml")
	if err != nil {
		return result, err
	}
	result.SourceDirty = dirty != ""
	root, err := os.Getwd()
	if err != nil {
		return result, fmt.Errorf("workspace: %w", err)
	}
	dataRoot := filepath.Join(root, ".cache", "shop-demo")
	if err = os.MkdirAll(dataRoot, 0755); err != nil {
		return result, fmt.Errorf("demo data directory: %w", err)
	}
	directory, err := os.MkdirTemp(dataRoot, "run-")
	if err != nil {
		return result, fmt.Errorf("isolated demo directory: %w", err)
	}
	result.Database = filepath.Join(directory, "signalscout.db")
	logFile, err := os.Create(filepath.Join(directory, "commands.log"))
	if err != nil {
		return result, fmt.Errorf("command log: %w", err)
	}
	defer func() { _ = logFile.Close() }()
	r := runner{environment: os.Environ(), log: logFile}
	buildDir := filepath.Join(root, ".cache", "shop-build")
	if err = os.MkdirAll(buildDir, 0755); err != nil {
		return result, fmt.Errorf("build directory: %w", err)
	}
	build := r
	build.environment = append(append([]string(nil), os.Environ()...), "GOOS=linux", "GOARCH="+arch, "CGO_ENABLED=0")
	if err = build.command(ctx, "go", "build", "-o", filepath.Join(buildDir, "shop"), "./samples/shop"); err != nil {
		return result, err
	}
	dockerfile, err := os.ReadFile(filepath.Join(root, "samples", "shop", "Dockerfile"))
	if err != nil {
		return result, fmt.Errorf("read Dockerfile: %w", err)
	}
	if err = os.WriteFile(filepath.Join(buildDir, "Dockerfile"), dockerfile, 0644); err != nil {
		return result, fmt.Errorf("stage Dockerfile: %w", err)
	}
	s, err := store.Open(ctx, result.Database, store.Options{QueueSize: 64, IdleTimeout: time.Second, SweepInterval: 250 * time.Millisecond, ContextBefore: 30 * time.Second, ContextAfter: 5 * time.Second})
	if err != nil {
		return result, err
	}
	defer func() { resultErr = errors.Join(resultErr, s.Close()) }()
	listener, err := net.Listen("tcp", "0.0.0.0:14318")
	if err != nil {
		return result, fmt.Errorf("demo OTLP listener: %w", err)
	}
	server := &http.Server{Handler: intake.New(s), ReadHeaderTimeout: 5 * time.Second}
	// A single native intake instance owns SQLite for the whole demo. Containers
	// export through the Collector; no Linux process opens the Windows DB file.
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Serve(listener) }()
	defer func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
			resultErr = errors.Join(resultErr, fmt.Errorf("intake shutdown: %w", err))
		}
		if err := <-serverDone; !errors.Is(err, http.ErrServerClosed) {
			resultErr = errors.Join(resultErr, err)
		}
	}()
	phaseEnv := func(phase, pool string) []string {
		return append(append([]string(nil), os.Environ()...), "GIT_COMMIT="+result.Revision, fmt.Sprintf("GIT_DIRTY=%t", result.SourceDirty), "SHOP_HOST_NAME=docker-desktop", "SHOP_PHASE="+phase, "DB_POOL_SIZE="+pool)
	}
	r.environment = phaseEnv("baseline", "20")
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		if err := r.compose(cleanup, "stop", "--timeout", "15"); err != nil {
			resultErr = errors.Join(resultErr, err)
		}
	}()
	if err = r.compose(ctx, "up", "--detach", "--build", "--wait", "--wait-timeout", "90"); err != nil {
		return result, err
	}
	if err = waitHealth(ctx, "http://127.0.0.1:13133/", 0); err != nil {
		return result, err
	}
	if err = waitHealth(ctx, "http://127.0.0.1:18082/healthz", 20); err != nil {
		return result, err
	}
	if err = pause(ctx, 2*time.Second); err != nil {
		return result, err
	}
	fmt.Println("Generating 40 passing baseline requests (pool=20)")
	baseline, err := load.Run(ctx, load.Options{URL: "http://127.0.0.1:18080/checkout", InventoryHealthURL: "http://127.0.0.1:18082/healthz", Count: 40, Concurrency: 8, Seed: seed, Phase: "baseline", Bursts: true})
	if err != nil {
		return result, err
	}
	result.BaselineRequests = baseline.Requests
	result.BaselinePassing = baseline.Passing
	if baseline.Passing != 40 {
		return result, fmt.Errorf("baseline produced %d passing requests, need 40", baseline.Passing)
	}
	if err = waitRecorded(ctx, s, baseline, 20); err != nil {
		return result, err
	}
	if err = pause(ctx, 5*time.Second); err != nil {
		return result, err
	}
	if err = r.compose(ctx, "stop", "--timeout", "15", "frontend", "checkout-service", "inventory-service"); err != nil {
		return result, err
	}
	r.environment = phaseEnv("failure", "5")
	if err = r.compose(ctx, "up", "--detach", "--no-deps", "--force-recreate", "--wait", "--wait-timeout", "90", "inventory-service", "checkout-service", "frontend"); err != nil {
		return result, err
	}
	if err = waitHealth(ctx, "http://127.0.0.1:18082/healthz", 5); err != nil {
		return result, err
	}
	if err = pause(ctx, 2*time.Second); err != nil {
		return result, err
	}
	fmt.Printf("Generating %d loaded requests (pool=5)\n", failureCount)
	failing, err := load.Run(ctx, load.Options{URL: "http://127.0.0.1:18080/checkout", InventoryHealthURL: "http://127.0.0.1:18082/healthz", Count: failureCount, Concurrency: 8, Seed: seed, Phase: "failure", Bursts: true})
	if err != nil {
		return result, err
	}
	result.FailureRequests = failing.Requests
	result.FailureCount = failing.Failing
	result.FailurePercent = 100 * float64(failing.Failing) / float64(failing.Requests)
	if result.FailurePercent < 2 || result.FailurePercent > 4 {
		return result, fmt.Errorf("observed failure rate %.2f%% is outside the intended approximately 3%% (2..4%%) band", result.FailurePercent)
	}
	if err = r.compose(ctx, "stop", "--timeout", "15", "frontend", "checkout-service", "inventory-service"); err != nil {
		return result, err
	}
	if err = waitRecorded(ctx, s, failing, 5); err != nil {
		return result, err
	}
	if err = pause(ctx, 5*time.Second); err != nil {
		return result, err
	}
	for _, record := range failing.Records {
		if record.Status < 500 {
			continue
		}
		result.FailureRunIDs = append(result.FailureRunIDs, record.TraceID)
		if result.Evidence.RunID == "" {
			detail, err := s.Show(ctx, record.TraceID)
			if err != nil {
				return result, err
			}
			result.Evidence, err = inspect(detail)
			if err != nil {
				return result, err
			}
		}
	}
	result.Signals, err = s.Counts(ctx)
	if err != nil {
		return result, err
	}
	select {
	case err := <-s.Errors():
		return result, fmt.Errorf("intake storage failed: %w", err)
	default:
	}
	result.CompletedAt = time.Now().UTC()
	result.Acceptance = true
	return result, nil
}

func main() {
	seed := flag.Int64("seed", 1, "Fixed cart workload seed")
	count := flag.Int("failure-count", 300, "Loaded request count, a multiple of 100")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	result, err := demo(ctx, *seed, *count)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if result.Database != "" {
			fmt.Fprintln(os.Stderr, "Partial telemetry and command log:", filepath.Dir(result.Database))
		}
		os.Exit(1)
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err == nil {
		err = os.WriteFile(filepath.Join("benchmark", "results", "shop-demo.json"), append(data, '\n'), 0644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("M2 acceptance passed: %d baseline passes, %d/%d failures (%.2f%%).\nDatabase: %s\n", result.BaselinePassing, result.FailureCount, result.FailureRequests, result.FailurePercent, result.Database)
}
