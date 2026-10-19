package viewer

import (
	"context"
	"encoding/json"
	"html"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/store"
)

// This opt-in check uses a saved M2 dataset; ordinary CI always runs the
// self-contained navigation fixture instead. Stop serve before using this test.
func TestStoredM2Walkthrough(t *testing.T) {
	source := os.Getenv("SIGNALSCOUT_VIEWER_DEMO")
	if source == "" {
		t.Skip("set SIGNALSCOUT_VIEWER_DEMO to an M2 report to check its local database")
	}
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	var demo struct {
		Database    string    `json:"database"`
		Seed        int       `json:"seed"`
		CompletedAt time.Time `json:"completed_at"`
	}
	if err = json.Unmarshal(data, &demo); err != nil {
		t.Fatal(err)
	}
	path := demo.Database
	if !filepath.IsAbs(path) {
		path = filepath.Join("..", "..", filepath.FromSlash(path))
	}
	ctx := context.Background()
	s, err := store.Open(ctx, path, store.Options{QueueSize: 4, IdleTimeout: time.Hour, SweepInterval: time.Hour, ContextBefore: 30 * time.Second, ContextAfter: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	from, to := demo.CompletedAt.Add(-24*time.Hour), demo.CompletedAt.Add(24*time.Hour)
	page, err := s.RunPage(ctx, store.RunFilter{Workload: "POST /checkout", Outcome: "fail", From: from.UnixNano(), To: to.UnixNano(), Limit: 1})
	if err != nil || len(page.Runs) != 1 {
		t.Fatalf("missing demo failure: %v", err)
	}
	writer := &capturingWriter{s: s}
	h, err := New(s.Reader, writer, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	query := url.Values{"outcome": {"fail"}, "from": {from.Format(time.RFC3339Nano)}, "to": {to.Format(time.RFC3339Nano)}}
	list := request(t, h, "GET", "/runs?"+query.Encode(), "")
	target := runURL(page.Runs[0].ID)
	if list.Code != 200 || !strings.Contains(list.Body.String(), `href="`+html.EscapeString(target)+`"`) {
		t.Fatal("list did not link to failure")
	}
	run := request(t, h, "GET", target, "")
	if run.Code != 200 {
		t.Fatal(run.Body.String())
	}
	form := regexp.MustCompile(`method="post" action="([^"]+)"`).FindStringSubmatch(run.Body.String())
	if len(form) != 2 {
		t.Fatal("missing comparison form")
	}
	result := request(t, h, "POST", html.UnescapeString(form[1]), "")
	if result.Code != 200 {
		t.Fatal(result.Body.String())
	}
	if len(writer.results) != 2 {
		t.Fatal("did not save both comparisons")
	}
	for _, saved := range writer.results {
		if len(saved.BaselineSpec.RunIDs) != 40 {
			t.Fatal("incorrect baseline size")
		}
	}
	links := regexp.MustCompile(`class="evidence-link" href="([^"]+)"`).FindAllStringSubmatch(result.Body.String(), -1)
	if len(links) == 0 || !strings.Contains(result.Body.String(), `class="failing"`) {
		t.Fatal("missing evidence or failing-value plot")
	}
	for _, link := range links {
		u, err := url.Parse(html.UnescapeString(link[1]))
		if err != nil {
			t.Fatal(err)
		}
		destination := request(t, h, "GET", u.RequestURI(), "")
		if destination.Code != 200 || !strings.Contains(destination.Body.String(), `id="`+u.Fragment+`"`) {
			t.Fatalf("unresolved demo evidence: %s", u)
		}
	}
	t.Logf("seed=%d run=%s: list -> run -> both persisted baseline panels -> %d resolved evidence links", demo.Seed, page.Runs[0].ID, len(links))
}
