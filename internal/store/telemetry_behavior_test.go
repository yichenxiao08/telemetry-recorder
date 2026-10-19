package store

import (
	"context"
	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/telemetry"
	"testing"
	"time"
)

func TestExplorerFiltersAndPagination(t *testing.T) {
	ctx := context.Background()
	s := testStore(t, 0)
	var signals []model.Signal
	for i, id := range []string{"log:a", "log:b", "log:c"} {
		x := fixture(id, "log", "trace", "r", 10+int64(i), 10+int64(i))
		x.Body = "warning <script>"
		x.Template = "warning"
		x.Raw = []byte(`{"severityNumber":"SEVERITY_NUMBER_WARN"}`)
		x.Attrs["region"] = "local"
		signals = append(signals, x)
	}
	if err := s.Submit(ctx, model.Batch{Signals: signals}); err != nil {
		t.Fatal(err)
	}
	f := telemetry.Filter{Kind: "log", Service: "shop", From: 1, To: 100, Limit: 2, Text: "<script>", Attribute: "region", AttributeValue: "local"}
	page, err := s.SearchTelemetry(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Signals) != 2 || page.Next == nil {
		t.Fatal(page)
	}
	f.After = page.Next
	next, err := s.SearchTelemetry(ctx, f)
	if err != nil || len(next.Signals) != 1 || next.Signals[0].ID != "log:c" {
		t.Fatalf("%+v %v", next, err)
	}
	f.After = nil
	templates, err := s.LogTemplates(ctx, f)
	if err != nil || len(templates) != 1 || templates[0].Count != 3 || templates[0].Severity != "13" {
		t.Fatalf("%+v %v", templates, err)
	}
	f.Service = ""
	f.TraceID = "trace"
	if page, err = s.SearchTelemetry(ctx, f); err != nil || len(page.Signals) != 2 {
		t.Fatalf("%+v %v", page, err)
	}
	minimum := 17
	f.SeverityMin = &minimum
	if page, err = s.SearchTelemetry(ctx, f); err != nil || len(page.Signals) != 0 {
		t.Fatalf("%+v %v", page, err)
	}
}
func TestBoundedContextIncludesRulesAndRefusesTruncation(t *testing.T) {
	ctx := context.Background()
	s := testStore(t, 0)
	root := fixture("span:root", "span", "trace", "r", 10, 100)
	ambient := fixture("log:ambient", "log", "", "r", 20, 20)
	other := ambient
	other.ID = "log:other"
	if err := s.Submit(ctx, model.Batch{Signals: []model.Signal{root, ambient, other}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Maintain(ctx, time.Now().Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	signals, err := s.RunContext(ctx, "trace", 1, 101, 2)
	if err != nil || len(signals) != 2 || len(signals[0].ContextMatches) == 0 {
		t.Fatalf("%+v %v", signals, err)
	}
	if _, err = s.RunContext(ctx, "trace", 1, 101, 1); err != EvidenceLimit {
		t.Fatal(err)
	}
}
