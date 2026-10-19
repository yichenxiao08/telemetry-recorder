package diff

import (
	"fmt"
	"github.com/signalscout/signalscout/internal/model"
	"math"
	"pgregory.net/rapid"
	"testing"
)

func contextFixture(value float64) model.Comparison {
	in := model.Comparison{Spec: model.BaselineSpec{Minimum: 10, ContextPolicy: "retrospective-config-v1", ContextLookbackNS: 30e9}}
	makeRun := func(id, outcome, pool string, v float64) model.RunDetail {
		run := model.Run{ID: id, Workload: "shop", Outcome: outcome, StartedAt: 100e9, EndedAt: 101e9, Closed: true, Config: map[string]any{"DB_POOL_SIZE": pool}}
		if outcome == "fail" {
			run.StartedAt = 200e9
			run.EndedAt = 201e9
		}
		d := model.RunDetail{Run: run}
		for i := range 2 {
			val := v
			d.Context = append(d.Context, model.Signal{ID: fmt.Sprintf("metric:%s/%d", id, i), Kind: "metric", Name: "pool", TS: run.StartedAt - int64(2-i)*1e9, Value: &val, Resource: model.Resource{Attrs: map[string]any{"service.name": "inventory", "signalscout.config.DB_POOL_SIZE": pool}}, Attrs: map[string]any{}, Raw: []byte(`{"metric":{"type":"gauge","unit":"connections"}}`)})
		}
		return d
	}
	in.Failing = makeRun("fail", "fail", "5", value)
	for i := range 40 {
		in.Baseline = append(in.Baseline, makeRun(fmt.Sprintf("b%d", i), "pass", "20", 0))
	}
	return in
}
func contextScore(t interface{ Fatal(...any) }, in model.Comparison) float64 {
	r, err := Compute(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range r.Differences {
		if d.Type == "context_metric" {
			return d.Score
		}
	}
	return 0
}
func TestContextMetricRules(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*model.Comparison)
		want   bool
	}{
		{"changed constant baseline", func(*model.Comparison) {}, true},
		{"constant everywhere", func(in *model.Comparison) {
			for i := range in.Failing.Context {
				*in.Failing.Context[i].Value = 0
			}
		}, false},
		{"sparse", func(in *model.Comparison) { in.Failing.Context = in.Failing.Context[:1] }, false},
		{"missing baseline windows", func(in *model.Comparison) {
			for i := range 31 {
				in.Baseline[i].Context = nil
			}
		}, false},
		{"monotonic counter", func(in *model.Comparison) {
			for i := range in.Failing.Context {
				in.Failing.Context[i].Raw = []byte(`{"metric":{"type":"sum","is_monotonic":true}}`)
			}
			for b := range in.Baseline {
				for i := range in.Baseline[b].Context {
					in.Baseline[b].Context[i].Raw = []byte(`{"metric":{"type":"sum","is_monotonic":true}}`)
				}
			}
		}, false},
		{"future excluded", func(in *model.Comparison) {
			for i := range in.Failing.Context {
				in.Failing.Context[i].TS = in.Failing.Run.EndedAt + 1
			}
		}, false},
		{"wrong phase config excluded", func(in *model.Comparison) {
			for i := range in.Failing.Context {
				in.Failing.Context[i].Resource.Attrs["signalscout.config.DB_POOL_SIZE"] = "20"
			}
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := contextFixture(5)
			tc.change(&in)
			if got := contextScore(t, in) > 0; got != tc.want {
				t.Fatalf("score present=%v want %v", got, tc.want)
			}
		})
	}
	in := contextFixture(5)
	bad := in.Failing.Context[0]
	v := 1000.0
	bad.ID = "metric:old-phase"
	bad.Value = &v
	bad.Resource.Attrs = map[string]any{"service.name": "inventory", "signalscout.config.DB_POOL_SIZE": "20"}
	in.Failing.Context = append(in.Failing.Context, bad)
	r, err := Compute(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range r.Differences {
		if d.Type == "context_metric" {
			if d.Value != 5 {
				t.Fatalf("previous-phase contamination: %+v", d)
			}
			for _, id := range d.EvidenceIDs {
				if id == bad.ID {
					t.Fatal("excluded point cited")
				}
			}
		}
	}
}
func TestContextProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		x := rapid.Float64Range(0, 100).Draw(t, "effect")
		a := contextScore(t, contextFixture(x))
		b := contextScore(t, contextFixture(x+1))
		if math.IsNaN(a) || a < 0 || a > 1 || b+1e-12 < a {
			t.Fatalf("scores %v %v", a, b)
		}
	})
}
func TestContextLogsDecoyAndCommonBaseline(t *testing.T) {
	in := contextFixture(0)
	log := func(id, name string, ts int64) model.Signal {
		return model.Signal{ID: id, Kind: "log", Name: "WARN", Template: name, TS: ts, Resource: model.Resource{Attrs: map[string]any{"service.name": "inventory"}}}
	}
	in.Failing.Context = []model.Signal{log("log:new", "new warning", 200e9), log("log:decoy", "common", 200e9)}
	for i := range in.Baseline {
		in.Baseline[i].Context = []model.Signal{log(fmt.Sprintf("log:b%d", i), "common", 100e9)}
	}
	r, err := Compute(in)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range r.Differences {
		if d.Type == "context_log" {
			if d.Name == "common" {
				t.Fatal("unchanged decoy scored")
			}
			if d.Name == "new warning" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("missing context warning")
	}
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 30).Draw(t, "commonCopies")
		base := presenceScore(true, 20, 40)
		if presenceScore(true, 20+n, 40+n) > base+1e-12 {
			t.Fatal("adding common baseline raised presence score")
		}
	})
}
