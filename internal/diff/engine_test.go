package diff

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/signalscout/signalscout/internal/model"
)

func span(id, parent, name string, start, ms int64) model.Signal {
	return model.Signal{ID: "span:" + id, SpanID: id, ParentID: parent, Kind: "span", Name: name, TS: start, End: start + ms*1000000}
}

func TestPathsRepeatedSiblingsAndMalformedTrees(t *testing.T) {
	signals := []model.Signal{span("root", "", "root", 1, 20), span("a", "root", "step", 2, 5), span("b", "root", "step", 3, 8), span("c", "a", "leaf", 4, 2)}
	got := extract(model.RunDetail{Signals: signals})
	key := pathKey([]string{"root", "step"})
	if got.paths[key].count != 2 || got.paths[key].maximum != 8 || got.paths[key].maxID != "span:b" {
		t.Fatalf("aggregation: %+v", got.paths[key])
	}
	for i, j := 0, len(signals)-1; i < j; i, j = i+1, j-1 {
		signals[i], signals[j] = signals[j], signals[i]
	}
	if !reflect.DeepEqual(got, extract(model.RunDetail{Signals: signals})) {
		t.Fatal("arrival order changes features")
	}
	if pathKey([]string{"a > b", "c"}) == pathKey([]string{"a", "b > c"}) {
		t.Fatal("delimiter collision")
	}
	bad := extract(model.RunDetail{Signals: []model.Signal{span("a", "b", "one", 1, 1), span("b", "a", "two", 2, 1), span("c", "missing", "orphan", 3, 1)}})
	for _, f := range bad.paths {
		if len(f.path) == 0 || !strings.HasPrefix(f.path[0], "[") {
			t.Fatalf("malformed tree got valid path %v", f.path)
		}
	}
}

func comparisonFixture() model.Comparison {
	makeDetail := func(id string, fail bool) model.RunDetail {
		r := model.Run{ID: id, Workload: "work", Outcome: "pass", StartedAt: 1, Closed: true, Config: map[string]any{"POOL": 20}, Env: map[string]any{"vcs.revision": "same", "signalscout.demo.phase": "baseline"}}
		signals := []model.Signal{span("root", "", "request", 1, 10), span("child", "root", "acquire", 2, 1), {ID: "event:cache:" + id, Kind: "event", Name: "cache.miss", SpanID: "root", TS: 3}}
		if fail {
			r.Outcome = "fail"
			r.StartedAt = 100
			r.Config["POOL"] = 5
			r.Env["signalscout.demo.phase"] = "failure"
			signals[0].End = 200000001
			signals[1].End = 100000002
			signals = append(signals, model.Signal{ID: "event:retry", Kind: "event", Name: "retry", SpanID: "root", TS: 4}, model.Signal{ID: "event:exception", Kind: "event", Name: "exception", SpanID: "child", TS: 5})
		}
		return model.RunDetail{Run: r, Signals: signals}
	}
	input := model.Comparison{Failing: makeDetail("fail", true), Spec: model.BaselineSpec{Mode: "default", N: 40, Minimum: 10, BeforeNS: 100, Workload: "work", ScoringVersion: model.DiffScoringVersion}}
	for i := 0; i < 40; i++ {
		id := string(rune('a' + i))
		b := makeDetail(id, false)
		if i >= 12 {
			b.Signals = b.Signals[:2]
		}
		input.Baseline = append(input.Baseline, b)
		input.Spec.RunIDs = append(input.Spec.RunIDs, id)
	}
	return input
}

func TestEngineRanksAndCollapsesWithoutContextDifferences(t *testing.T) {
	input := comparisonFixture()
	input.Failing.Context = []model.Signal{{Kind: "log", Name: "unrelated"}, {Kind: "metric", Name: "system.cpu"}}
	got, err := Compute(input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "ok" || len(got.Differences) < 4 {
		t.Fatalf("result %+v", got)
	}
	top := map[string]bool{}
	for _, d := range got.Differences[:3] {
		top[d.Type+":"+d.Name] = true
	}
	for _, key := range []string{"config:POOL", "presence:retry", "latency:acquire"} {
		if !top[key] {
			t.Fatalf("missing %s: %+v", key, got.Differences)
		}
	}
	for _, d := range got.Differences {
		if d.Type == "latency" && d.Name == "acquire" && len(d.Related) < 2 {
			t.Fatal("slow parent/exception not retained as related")
		}
		if d.Type == "env" || d.Name == "exception" || d.Name == "request" && d.Type == "latency" {
			t.Fatalf("redundant/control difference ranked: %+v", d)
		}
		if len(d.EvidenceIDs) == 0 || d.Score < 0 || d.Score > 1 {
			t.Fatalf("invalid evidence/score: %+v", d)
		}
	}
	before, _ := json.Marshal(got.Differences)
	for i, j := 0, len(input.Baseline)-1; i < j; i, j = i+1, j-1 {
		input.Baseline[i], input.Baseline[j] = input.Baseline[j], input.Baseline[i]
	}
	other, err := Compute(input)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(other.Differences)
	if string(before) != string(after) {
		t.Fatal("baseline order changes ranking")
	}
}

func TestInsufficientAndIndependentExceptions(t *testing.T) {
	input := comparisonFixture()
	input.Baseline = input.Baseline[:9]
	input.Spec.RunIDs = input.Spec.RunIDs[:9]
	got, err := Compute(input)
	if err != nil || got.Status != "insufficient_baseline" || len(got.Differences) != 0 {
		t.Fatalf("got %+v err %v", got, err)
	}
	input = comparisonFixture()
	input.Failing.Signals = append(input.Failing.Signals, span("other", "root", "other", 6, 1), model.Signal{ID: "event:independent", Kind: "event", Name: "exception", SpanID: "other", TS: 7})
	got, err = Compute(input)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range got.Differences {
		if d.Name == "exception" {
			found = true
		}
	}
	if !found {
		t.Fatal("independent exception hidden")
	}
}

func TestConfigModeMissingNullAndCommitCorrelation(t *testing.T) {
	input := comparisonFixture()
	input.Failing.Run.Config = map[string]any{"null": nil, "present": 1}
	input.Failing.Run.Env["vcs.revision"] = "new"
	for i := range input.Baseline {
		input.Baseline[i].Run.Config = map[string]any{"present": nil, "removed": 1}
	}
	got, err := Compute(input)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, d := range got.Differences {
		if d.Type == "config" {
			found[d.Name] = true
			if !strings.Contains(d.Summary, "coincides") {
				t.Fatal("commit correlation missing")
			}
		}
	}
	for _, key := range []string{"null", "present", "removed"} {
		if !found[key] {
			t.Fatalf("missing config %s", key)
		}
	}
}
