package viewer

import (
	"context"
	"fmt"
	"github.com/signalscout/signalscout/internal/model"
	"html"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestExplorerContextAndEvidenceNavigation(t *testing.T) {
	s, writer, h := testViewer(t)
	ctx := context.Background()
	detail, err := s.RunStructure(ctx, "run-80", 5000)
	if err != nil {
		t.Fatal(err)
	}
	base := detail.Run.StartedAt - 80*int64(time.Second)
	resource := model.Resource{ID: "ambient-service", Attrs: map[string]any{"service.name": "inventory-service", "host.name": "host"}}
	signals := []model.Signal{}
	for i := range 81 {
		for j := range 2 {
			v := 0.0
			if i == 80 {
				v = 5
			}
			ts := base + int64(i)*int64(time.Second) + int64(j+1)*1e6
			signals = append(signals, model.Signal{ID: fmt.Sprintf("metric:ui/%d/%d", i, j), Kind: "metric", Name: "pool", TS: ts, End: ts, Value: &v, Resource: resource, Attrs: map[string]any{}, Raw: []byte(`{"metric":{"type":"gauge"}}`)})
		}
	}
	signals = append(signals, model.Signal{ID: "log:ui-danger", Kind: "log", Name: "WARN", TS: detail.Run.StartedAt + 3e6, Body: "<script>alert(1)</script>", Template: "new warning", Resource: resource, Raw: []byte(`{"severityNumber":"SEVERITY_NUMBER_WARN"}`)})
	free := model.Signal{ID: "span:free/root", Kind: "span", TraceID: "free", SpanID: "root", Name: "unclassified worker", TS: detail.Run.StartedAt + 4e6, End: detail.Run.StartedAt + 5e6, Resource: model.Resource{ID: "job", Attrs: map[string]any{"service.name": "inventory-service", "signalscout.run.id": "job-x"}}, Attrs: map[string]any{}}
	signals = append(signals, free)
	hostValue := 2.0
	host := free
	host.ID = "metric:host-only"
	host.Kind = "metric"
	host.Name = "host.load"
	host.TraceID = ""
	host.Value = &hostValue
	host.Resource = model.Resource{ID: "host-only", Attrs: map[string]any{"host.name": "host"}}
	host.Raw = []byte(`{"metric":{"type":"gauge"}}`)
	signals = append(signals, host)
	if err = s.Submit(ctx, model.Batch{Signals: signals}); err != nil {
		t.Fatal(err)
	}
	q := url.Values{"service": {"inventory-service"}, "from": {timeValue(detail.Run.StartedAt - 1e9)}, "to": {timeValue(detail.Run.EndedAt + 1e9)}, "limit": {"100"}}
	for _, mode := range []string{"services", "spans", "logs", "templates", "metrics"} {
		q.Set("mode", mode)
		page := request(t, h, "GET", "/telemetry?"+q.Encode(), "")
		if page.Code != 200 {
			t.Fatalf("%s: %d %s", mode, page.Code, page.Body.String())
		}
		if strings.Contains(page.Body.String(), "<script>") {
			t.Fatal("stored HTML executed")
		}
		if !strings.Contains(page.Header().Get("Content-Security-Policy"), "script-src 'none'") {
			t.Fatal("weak CSP")
		}
	}
	q.Set("mode", "metrics")
	q.Del("service")
	q.Set("no_service", "1")
	hostPage := request(t, h, "GET", "/telemetry?"+q.Encode(), "")
	if hostPage.Code != 200 || !strings.Contains(hostPage.Body.String(), "host.load") {
		t.Fatal("host-only resource is not browsable")
	}
	q.Del("no_service")
	q.Set("service", "inventory-service")
	q.Set("mode", "metrics")
	q.Set("limit", "1")
	first := request(t, h, "GET", "/telemetry?"+q.Encode(), "")
	next := regexp.MustCompile(`rel="next" href="([^"]+)"`).FindStringSubmatch(first.Body.String())
	if len(next) != 2 {
		t.Fatal("missing telemetry pagination")
	}
	if page := request(t, h, "GET", html.UnescapeString(next[1]), ""); page.Code != 200 {
		t.Fatal(page.Body.String())
	}
	q.Set("trace", "free")
	q.Del("service")
	q.Set("limit", "100")
	tracePage := request(t, h, "GET", "/trace?"+q.Encode(), "")
	if tracePage.Code != 200 || !strings.Contains(tracePage.Body.String(), `id="`+anchor(free.ID)+`"`) {
		t.Fatal("trace lookup depended on run identity")
	}
	if strings.Count(tracePage.Body.String(), `id="`+anchor(free.ID)+`"`) != 1 {
		t.Fatal("duplicate trace anchor")
	}
	page := request(t, h, "GET", "/run?id=run-80", "")
	if page.Code != 200 || !strings.Contains(page.Body.String(), "Red line: failure marker") || !strings.Contains(page.Body.String(), "&lt;script&gt;") || !strings.Contains(page.Body.String(), "host.name") {
		t.Fatal("missing escaped context, rules or marker")
	}
	compared := request(t, h, "POST", "/run/diff?id=run-80", "")
	if compared.Code != 200 || len(writer.results) != 2 || !strings.Contains(compared.Body.String(), "context_metric") || !strings.Contains(compared.Body.String(), "context_log") {
		t.Fatal(compared.Body.String())
	}
	links := regexp.MustCompile(`class="evidence-link" href="([^"]+)"`).FindAllStringSubmatch(compared.Body.String(), -1)
	found := false
	for _, match := range links {
		target, err := url.Parse(html.UnescapeString(match[1]))
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(target.Fragment, "e-") {
			response := request(t, h, "GET", target.RequestURI(), "")
			if response.Code != 200 || !strings.Contains(response.Body.String(), `id="`+target.Fragment+`"`) {
				t.Fatalf("unresolved context anchor %s", target)
			}
			if target.Fragment == anchor("log:ui-danger") {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("no clickable context log evidence")
	}
	for _, path := range []string{"/telemetry?mode=logs", "/telemetry?mode=metrics&limit=1", "/trace?from=bad", "/telemetry?mode=bad"} {
		if response := request(t, h, "GET", path, ""); response.Code != 400 {
			t.Fatalf("unbounded query accepted %s", path)
		}
	}
}

func TestFocusedTraceBeyondFirstPage(t *testing.T) {
	s, _, h := testViewer(t)
	ctx := context.Background()
	base := time.Now().Add(-time.Hour).UnixNano()
	signals := []model.Signal{}
	for i := range 1001 {
		signals = append(signals, model.Signal{ID: fmt.Sprintf("span:large/%04d", i), Kind: "span", TraceID: "large", SpanID: fmt.Sprint(i), Name: "worker", TS: base, End: base + 1, Resource: model.Resource{ID: "large-job", Attrs: map[string]any{"service.name": "worker", "signalscout.run.id": "large-job"}}})
	}
	if err := s.Submit(ctx, model.Batch{Signals: signals}); err != nil {
		t.Fatal(err)
	}
	last := signals[len(signals)-1]
	rows := contextRows([]model.Signal{last}, base-1, base+100)
	u, err := url.Parse(rows[0].TraceURL)
	if err != nil {
		t.Fatal(err)
	}
	page := request(t, h, "GET", u.RequestURI(), "")
	if page.Code != 200 || !strings.Contains(page.Body.String(), `id="`+u.Fragment+`"`) {
		t.Fatal("selected span fell outside first trace page")
	}
}
