package main

import (
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
)

func acceptedDetail() model.RunDetail {
	value := 5.0
	app := model.Resource{ID: "app", Attrs: map[string]any{"signalscout.config.DB_POOL_SIZE": 5}}
	return model.RunDetail{Run: model.Run{ID: "run", Outcome: "fail"}, Signals: []model.Signal{
		{ID: "acquire", Kind: "span", Name: "db.pool.acquire", TS: 1, End: 1 + int64(100*time.Millisecond), Resource: app},
		{ID: "retry", Kind: "event", Name: "retry"},
		{ID: "log", Kind: "log", TraceID: "run"},
	}, Context: []model.Signal{
		{ID: "host", Kind: "metric", Name: "system.memory.usage", Value: &value, Resource: model.Resource{ID: "host", Attrs: map[string]any{"host.name": "docker-desktop"}}, ContextMatches: []model.ContextMatch{{Rule: "host.name", ResourceID: "app"}}},
		{ID: "gauge", Kind: "metric", Name: "db.pool.in_use", Value: &value, Resource: app},
	}}
}

func TestAcceptanceRequiresMeasuredEvidenceAndSeparateHostResource(t *testing.T) {
	if _, err := inspect(acceptedDetail()); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		remove func(*model.RunDetail)
	}{
		{"slow acquire", func(d *model.RunDetail) { d.Signals[0].End = d.Signals[0].TS + int64(time.Millisecond) }},
		{"retry", func(d *model.RunDetail) { d.Signals[1].Name = "other" }},
		{"correlated log", func(d *model.RunDetail) { d.Signals[2].TraceID = "" }},
		{"host rule", func(d *model.RunDetail) { d.Context[0].ContextMatches[0].Rule = "exact_resource" }},
		{"host value", func(d *model.RunDetail) { d.Context[0].Value = nil }},
		{"separate resource", func(d *model.RunDetail) { d.Context[0].Resource.ID = "app" }},
		{"pool gauge", func(d *model.RunDetail) { d.Context[1].Name = "other" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			detail := acceptedDetail()
			tc.remove(&detail)
			if _, err := inspect(detail); err == nil {
				t.Fatal("incomplete evidence passed acceptance")
			}
		})
	}
}
