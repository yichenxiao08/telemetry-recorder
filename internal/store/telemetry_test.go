package store

import (
	"context"
	"github.com/signalscout/signalscout/internal/telemetry"
	"strings"
	"testing"
)

func TestExplorerQueryPlans(t *testing.T) {
	s := testStore(t, 0)
	for _, kind := range []string{"span", "log", "metric"} {
		for _, mode := range []string{"service", "trace", "templates", "overview"} {
			if mode == "trace" && kind == "metric" || mode == "templates" && kind != "log" || mode == "overview" && kind != "span" {
				continue
			}
			t.Run(kind+"/"+mode, func(t *testing.T) {
				f := telemetry.Filter{Kind: kind, Service: "shop", From: 1, To: 1000, Limit: 100}
				if mode == "trace" {
					f.TraceID = "t"
				}
				q, args, err := telemetry.Query(f)
				if mode == "templates" {
					q, args, err = telemetry.TemplateQuery(f)
				}
				if mode == "overview" {
					q, args, err = telemetry.OverviewQuery(f)
				}
				if err != nil {
					t.Fatal(err)
				}
				rows, err := s.reader.QueryContext(context.Background(), "EXPLAIN QUERY PLAN "+q, args...)
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = rows.Close() }()
				var plan strings.Builder
				for rows.Next() {
					var a, b, c int
					var d string
					if err = rows.Scan(&a, &b, &c, &d); err != nil {
						t.Fatal(err)
					}
					plan.WriteString(d + "\n")
				}
				if err = rows.Err(); err != nil {
					t.Fatal(err)
				}
				p := plan.String()
				index := "signals_resource_kind_time"
				if kind == "log" {
					index = "signals_log_resource_time"
				}
				if mode == "trace" {
					index = "signals_trace_time"
				} else if !strings.Contains(p, "resources_service") {
					t.Fatal(p)
				}
				if !strings.Contains(p, index) || !strings.Contains(p, "ts_ns>?") || strings.Contains(p, "SCAN s") {
					t.Fatalf("unbounded plan: %s", p)
				}
			})
		}
	}
}
