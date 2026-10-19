package viewer

import (
	"context"
	"math"
	"testing"

	"github.com/signalscout/signalscout/internal/diff"
	"github.com/signalscout/signalscout/internal/model"
)

func TestTimelineHandlesCyclesMissingParentsAndTraceScopedIDs(t *testing.T) {
	detail := model.RunDetail{Run: model.Run{StartedAt: 10, EndedAt: 100}, Signals: []model.Signal{
		{ID: "a", Kind: "span", TraceID: "one", SpanID: "a", ParentID: "b", TS: 10, End: 20},
		{ID: "b", Kind: "span", TraceID: "one", SpanID: "b", ParentID: "a", TS: 11, End: 21},
		{ID: "c", Kind: "span", TraceID: "two", SpanID: "a", TS: 12, End: 22},
		{ID: "event:c", Kind: "event", TraceID: "two", SpanID: "a", TS: 13},
		{ID: "event:orphan", Kind: "event", SpanID: "missing", TS: 14},
	}}
	spans, orphans := timeline(detail)
	if len(spans) != 3 || len(orphans) != 1 {
		t.Fatal("lost malformed-tree evidence")
	}
	seen := map[string]bool{}
	for _, s := range spans {
		if seen[s.Anchor] {
			t.Fatal("duplicate anchor")
		}
		seen[s.Anchor] = true
		if s.Signal.ID == "c" && len(s.Events) != 1 {
			t.Fatal("event attached across traces")
		}
	}
}

func TestPlotGeometryAndConstantBaseline(t *testing.T) {
	for _, values := range [][]float64{{0, 0, 0}, {1, 2, 100}} {
		p := makePlot(diff.NumericDistribution{Values: values, RunIDs: []string{"a", "b", "c"}, Failing: 0})
		if p == nil || math.IsNaN(p.FailingX) || p.FailingX < 20 || p.FailingX > 620 {
			t.Fatal("invalid failing marker")
		}
		for _, point := range p.Points {
			if math.IsNaN(point.X) || point.X < 20 || point.X > 620 {
				t.Fatal("invalid sample marker")
			}
		}
	}
	if p := makePlot(diff.NumericDistribution{}); p != nil {
		t.Fatal("invented empty distribution")
	}
}

func TestRunIDsWithURLSyntaxRemainTextAndResolve(t *testing.T) {
	s, _, h := testViewer(t)
	for _, id := range []string{".", "..", `ml/job?x=<script>#fragment`, "a/b"} {
		signal := model.Signal{ID: "span:" + id, TraceID: id, SpanID: "root", Kind: "span", Name: "untrusted", TS: 1, End: 2, Resource: model.Resource{ID: "ids", Attrs: map[string]any{}}}
		if err := s.Submit(context.Background(), model.Batch{Signals: []model.Signal{signal}}); err != nil {
			t.Fatal(err)
		}
		page := request(t, h, "GET", runURL(id), "")
		if page.Code != 200 {
			t.Fatalf("run ID %q: %d", id, page.Code)
		}
	}
}
