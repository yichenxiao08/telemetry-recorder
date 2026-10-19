package viewer

import (
	"context"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
)

type capturingWriter struct {
	s       *store.Store
	mu      sync.Mutex
	results []model.DiffResult
}

type failedWriter struct{ err error }

func (w failedWriter) SaveDiff(context.Context, model.DiffResult) error { return w.err }

type noPhase struct{ *store.Reader }

func (noPhase) PhaseStart(context.Context, model.Run) (string, error) { return "", nil }

type blockedWriter struct {
	entered chan struct{}
	release chan struct{}
}

func (w blockedWriter) SaveDiff(ctx context.Context, _ model.DiffResult) error {
	select {
	case w.entered <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-w.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestConcurrentComparisonAdmissionIsBounded(t *testing.T) {
	s, _, _ := testViewer(t)
	blocked := blockedWriter{entered: make(chan struct{}, 4), release: make(chan struct{})}
	var once sync.Once
	t.Cleanup(func() { once.Do(func() { close(blocked.release) }) })
	h, err := New(s.Reader, blocked, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan int, 2)
	for i := 0; i < 2; i++ {
		go func() {
			r := httptest.NewRequest("POST", "/run/diff?id=run-80", nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			done <- w.Code
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case <-blocked.entered:
		case <-time.After(5 * time.Second):
			t.Fatal("comparison did not reach writer")
		}
	}
	rejected := request(t, h, "POST", "/run/diff?id=run-80", "")
	if rejected.Code != 429 || rejected.Header().Get("Retry-After") == "" {
		t.Fatal("extra computation admitted")
	}
	once.Do(func() { close(blocked.release) })
	for i := 0; i < 2; i++ {
		select {
		case code := <-done:
			if code != 200 {
				t.Fatalf("admitted request returned %d", code)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("comparison did not finish")
		}
	}
}

func TestMissingPhaseExplicitOverrideAndWriterBackpressure(t *testing.T) {
	s, writer, _ := testViewer(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h, err := New(noPhase{s.Reader}, writer, logger)
	if err != nil {
		t.Fatal(err)
	}
	page := request(t, h, "POST", "/run/diff?id=run-80", "")
	if page.Code != 200 || !strings.Contains(page.Body.String(), "Supply a cutoff") || len(writer.results) != 1 || writer.results[0].BaselineSpec.Mode != "default" {
		t.Fatal("invented phase or lost default")
	}
	page = request(t, h, "POST", "/run/diff?id=run-80", "baseline_before=run-40")
	if page.Code != 200 || len(writer.results) != 3 || writer.results[1].BaselineSpec.BeforeInput != "run-40" {
		t.Fatal("run-ID cutoff semantics lost")
	}
	for _, tc := range []struct {
		err    error
		status int
	}{{store.QueueFull, 503}, {store.RunChanged, 409}, {context.DeadlineExceeded, 504}} {
		failing, err := New(s.Reader, failedWriter{tc.err}, logger)
		if err != nil {
			t.Fatal(err)
		}
		response := request(t, failing, "POST", "/run/diff?id=run-80", "")
		if response.Code != tc.status {
			t.Fatalf("writer error %v returned %d", tc.err, response.Code)
		}
		if tc.status == 503 && response.Header().Get("Retry-After") == "" {
			t.Fatal("no retry advice")
		}
	}
}

func (w *capturingWriter) SaveDiff(ctx context.Context, result model.DiffResult) error {
	if err := w.s.SaveDiff(ctx, result); err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.results = append(w.results, result)
	return nil
}

func testViewer(t *testing.T) (*store.Store, *capturingWriter, http.Handler) {
	t.Helper()
	ctx := context.Background()
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "viewer.db"), store.Options{QueueSize: 4, IdleTimeout: time.Hour, SweepInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	base := time.Now().Add(-time.Hour).UnixNano()
	var signals []model.Signal
	for i := 0; i < 81; i++ {
		id := fmt.Sprintf("run-%02d", i)
		pool := 20
		phase := "baseline"
		if i >= 40 {
			pool = 5
			phase = "failure"
		}
		resource := model.Resource{ID: fmt.Sprint(pool), Attrs: map[string]any{"signalscout.config.DB_POOL_SIZE": pool, "signalscout.demo.phase": phase, "vcs.revision": "same", "host.name": "host"}}
		ts := base + int64(i)*int64(time.Second)
		root := model.Signal{ID: "span:" + id + "/root", TraceID: id, SpanID: "root", Kind: "span", Name: `frontend <img src=x onerror=alert(1)>`, TS: ts, End: ts + int64(10*time.Millisecond), Resource: resource, Attrs: map[string]any{"http.method": "POST", "http.route": "/checkout"}}
		child := root
		child.ID = "span:" + id + "/child"
		child.SpanID = "child"
		child.ParentID = "root"
		child.Name = "db.pool.acquire"
		child.TS++
		child.End = child.TS + int64(time.Millisecond)
		if i == 80 {
			root.Status = "error"
			root.End = ts + int64(200*time.Millisecond)
			child.End = child.TS + int64(100*time.Millisecond)
		}
		signals = append(signals, root, child)
		if i%3 == 0 || i == 80 {
			signals = append(signals, model.Signal{ID: "event:" + id + "/cache", TraceID: id, SpanID: "root", Kind: "event", Name: "cache.miss", TS: ts + 2, Resource: resource})
		}
		if i == 80 {
			signals = append(signals, model.Signal{ID: "event:retry", TraceID: id, SpanID: "child", Kind: "event", Name: `retry`, TS: ts + 3, Resource: resource})
		}
	}
	if err = s.Submit(ctx, model.Batch{Signals: signals}); err != nil {
		t.Fatal(err)
	}
	if err = s.Maintain(ctx, time.Now().Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	open := model.Signal{ID: "span:open", TraceID: "open", SpanID: "root", Kind: "span", Name: "open", TS: base, End: base + 1, Resource: model.Resource{ID: "open", Attrs: map[string]any{}}}
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{open}}); err != nil {
		t.Fatal(err)
	}
	writer := &capturingWriter{s: s}
	handler, err := New(s.Reader, writer, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return s, writer, handler
}

func request(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	target, err := url.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	// Browsers keep fragments locally; they are not part of an HTTP request.
	r := httptest.NewRequest(method, target.RequestURI(), strings.NewReader(body))
	if method == http.MethodPost {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestWalkListRunDiffAndEveryEvidenceAnchor(t *testing.T) {
	s, writer, h := testViewer(t)
	list := request(t, h, "GET", "/runs?outcome=fail", "")
	if list.Code != 200 {
		t.Fatal(list.Body.String())
	}
	runLink := "/run?id=run-80"
	if !strings.Contains(list.Body.String(), `href="`+runLink+`"`) {
		t.Fatal("missing failing-run link")
	}
	page := request(t, h, "GET", runLink, "")
	if page.Code != 200 || !strings.Contains(page.Body.String(), `action="/run/diff?id=run-80#diff"`) {
		t.Fatal("missing diff form")
	}
	if len(writer.results) != 0 {
		t.Fatal("GET persisted a diff")
	}
	action := regexp.MustCompile(`method="post" action="([^"]+)"`).FindStringSubmatch(page.Body.String())
	if len(action) != 2 {
		t.Fatal("missing comparison action")
	}
	response := request(t, h, "POST", html.UnescapeString(action[1]), "")
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	body := response.Body.String()
	for _, text := range []string{"Before the failure phase", "Recent baseline", "Selected runs: 40", "baseline distribution", "DB_POOL_SIZE", "retry", "<svg", "Raw baseline rate"} {
		if !strings.Contains(body, text) {
			t.Fatalf("missing %q", text)
		}
	}
	if len(writer.results) != 2 {
		t.Fatal("did not persist both results through serving writer")
	}
	modes := map[string]bool{}
	for _, r := range writer.results {
		modes[r.BaselineSpec.Mode] = true
	}
	if !modes["default"] || !modes["before"] {
		t.Fatal("lost baseline mode")
	}
	links := regexp.MustCompile(`class="evidence-link" href="([^"]+)"`).FindAllStringSubmatch(body, -1)
	if len(links) < 3 {
		t.Fatal("missing evidence links")
	}
	sawSpan, sawEvent, sawConfig := false, false, false
	for _, match := range links {
		target, err := url.Parse(html.UnescapeString(match[1]))
		if err != nil {
			t.Fatal(err)
		}
		targetPage := request(t, h, "GET", target.RequestURI(), "")
		if targetPage.Code != 200 {
			t.Fatalf("evidence target failed %s", target)
		}
		if !strings.Contains(targetPage.Body.String(), `id="`+target.Fragment+`"`) {
			t.Fatalf("unresolved anchor %s", target)
		}
		if target.Fragment == "config" {
			sawConfig = true
		}
		if target.Fragment == anchor("event:retry") {
			sawEvent = true
		}
		if target.Fragment == anchor("span:run-80/child") {
			sawSpan = true
		}
	}
	if !sawSpan || !sawEvent || !sawConfig {
		t.Fatalf("links span=%t event=%t config=%t", sawSpan, sawEvent, sawConfig)
	}
	detail, err := s.Show(context.Background(), "open")
	if err != nil || detail.Run.Closed {
		t.Fatal("viewer finalized an open run")
	}
	css := request(t, h, "GET", "/assets/viewer.css", "")
	if !strings.Contains(css.Body.String(), ":target") {
		t.Fatal("no anchor highlight")
	}
}

func TestEscapingCSPFiltersBoundsAndSameOrigin(t *testing.T) {
	_, writer, h := testViewer(t)
	for _, path := range []string{"/runs", "/run?id=run-80", "/assets/viewer.css", "/missing"} {
		w := request(t, h, "GET", path, "")
		csp := w.Header().Get("Content-Security-Policy")
		if !strings.Contains(csp, "default-src 'none'") || !strings.Contains(csp, "script-src 'none'") || strings.Contains(csp, "unsafe-inline") {
			t.Fatalf("weak CSP %s", csp)
		}
		if strings.Contains(w.Body.String(), "<img src=x") || strings.Contains(w.Body.String(), "<script") {
			t.Fatal("unescaped telemetry")
		}
	}
	page := request(t, h, "GET", "/run?id=run-80", "")
	if !strings.Contains(page.Body.String(), "&lt;img") {
		t.Fatal("telemetry missing rather than escaped")
	}
	epoch := request(t, h, "GET", "/runs?from=1970-01-01T00:00:00Z&to=1970-01-02T00:00:00Z", "")
	if epoch.Code != 200 || !strings.Contains(epoch.Body.String(), `value="1970-01-01T00:00:00Z"`) {
		t.Fatal("epoch filter was replaced by unknown telemetry time")
	}
	for _, path := range []string{"/runs?limit=101", "/runs?cursor=bad", "/runs?from=invalid", "/runs?outcome=invalid", "/runs?from=2026-01-01T00:00:00Z&to=2026-04-01T00:00:00Z"} {
		if w := request(t, h, "GET", path, ""); w.Code != 400 {
			t.Fatalf("invalid filter returned %d: %s", w.Code, path)
		}
	}
	first := request(t, h, "GET", "/runs?limit=2", "")
	next := regexp.MustCompile(`rel="next" href="([^"]+)"`).FindStringSubmatch(first.Body.String())
	if len(next) != 2 {
		t.Fatal("no next page")
	}
	second := request(t, h, "GET", html.UnescapeString(next[1]), "")
	if second.Code != 200 || strings.Contains(second.Body.String(), `href="/run?id=run-80"`) {
		t.Fatal("pagination repeated first run")
	}
	r := httptest.NewRequest("POST", "/run/diff?id=run-80", strings.NewReader(""))
	r.Header.Set("Origin", "https://other.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 || len(writer.results) != 0 {
		t.Fatal("cross-origin compute accepted")
	}
	before := url.Values{"baseline_before": {"not-a-run"}}.Encode()
	if w = request(t, h, "POST", "/run/diff?id=run-80", before); w.Code != 400 {
		t.Fatalf("invalid cutoff %d", w.Code)
	}
}
