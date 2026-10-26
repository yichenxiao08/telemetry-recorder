package main

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/provenance"
	"github.com/signalscout/signalscout/internal/store"
)

func TestDeclaredProfileDistinguishesRewardAndUnknown(t *testing.T) {
	var p model.AnalysisProfile
	if e := json.Unmarshal(profileSource, &p); e != nil {
		t.Fatal(e)
	}
	if e := provenance.ValidateProfile(p); e != nil {
		t.Fatal(e)
	}
	rule := p.Relations.Rules[3]
	for _, tc := range []struct {
		reward               any
		repeat, apple, death bool
		want                 string
	}{
		{1.0, true, false, false, "violated"},
		{-1.0, true, false, false, "satisfied"},
		{0.5, true, false, false, "violated"},
		{1.0, false, false, false, "not_applicable"},
		{10.0, true, true, false, "not_applicable"},
		{-100.0, true, false, true, "not_applicable"},
		{map[string]any{"unknown": "capture_off"}, true, false, false, "unknown"},
	} {
		v := map[string]any{"inputs": map[string]any{"apple": tc.apple, "death": tc.death, "repeated_configuration": tc.repeat, "actual_reward": tc.reward}}
		got := provenance.EvaluateRelation(v, rule, "")
		if got.Status != tc.want {
			t.Fatalf("%+v: %+v", tc, got)
		}
	}
}
func TestConfigHasRequiredGateAndNoWholeObjectCapture(t *testing.T) {
	cfg := config("http://127.0.0.1:12345")
	if cfg["version"] != 1 || cfg["synthetic_only"] != true {
		t.Fatal(cfg)
	}
	raw, e := json.Marshal(cfg)
	if e != nil {
		t.Fatal(e)
	}
	var c struct {
		Probes []struct {
			Capture struct {
				Return bool
				Args   []string
			}
		}
	}
	if e = json.Unmarshal(raw, &c); e != nil {
		t.Fatal(e)
	}
	if len(c.Probes) != 2 || len(c.Probes[1].Capture.Args) != 4 || c.Probes[0].Capture.Return || c.Probes[1].Capture.Return {
		t.Fatal(c)
	}
}
func TestLossCompletenessAndSamplingFailClosed(t *testing.T) {
	var d doctor
	if e := readJSON("testdata/doctor.json", &d); e != nil {
		t.Fatal(e)
	}

	if _, e := checkDoctor(d); e != nil {
		t.Fatal(e)
	}
	delete(d.Counters, "acknowledged")
	if _, e := checkDoctor(d); e == nil {
		t.Fatal("lost capture accepted")
	}
	c := store.ScopeReport{PopulationKnown: true, ScopeFinal: true, ScopeExpected: 10, ListedRuns: 10, Counts: model.CompletenessCounts{Complete: 10}}
	if e := checkCompleteness(c); e != nil {
		t.Fatal(e)
	}
	c.Counts.UnknownItems = 1
	if e := checkCompleteness(c); e == nil {
		t.Fatal("unknown accepted")
	}
	r := model.RelationReport{CandidateRuns: 10, Runs: map[string]map[string]*model.RelationCounts{}, Totals: map[string]*model.RelationCounts{}}
	for i := 0; i < 10; i++ {
		r.Runs[string(rune('a'+i))] = map[string]*model.RelationCounts{}
	}
	for _, id := range []string{"apple-events", "alive-steps", "configuration-repeats", "positive-reward-on-repeat"} {
		r.Totals[id] = &model.RelationCounts{Total: 20, Satisfied: 20}
		for _, run := range r.Runs {
			run[id] = &model.RelationCounts{Total: 2, Satisfied: 2}
		}
	}
	if e := checkRelations(r); e != nil {
		t.Fatal(e)
	}
	r.Totals["positive-reward-on-repeat"].Unknown = 1
	if e := checkRelations(r); e == nil {
		t.Fatal("unknown relation accepted")
	}
	r.Totals["positive-reward-on-repeat"].Unknown = 0
	r.Totals["positive-reward-on-repeat"].SampledRecords = 1
	if e := checkRelations(r); e == nil {
		t.Fatal("sample accepted")
	}
}
func TestFreshOutputHashAndInputBounds(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "file")
	if e := writeNew(p, []byte("original")); e != nil {
		t.Fatal(e)
	}
	if e := writeNew(p, []byte("changed")); e == nil {
		t.Fatal("overwrote evidence")
	}
	if e := verify(p, "not-a-hash"); e == nil {
		t.Fatal("accepted changed input")
	}
	if _, e := within(root, "../escape"); e == nil {
		t.Fatal("escaped project")
	}
	if _, e := within(root, "study\\v2\\spec.json"); e != nil {
		t.Fatal(e)
	}
	for _, pace := range []float64{math.NaN(), math.Inf(1), -1, 101} {
		if e := execute(context.Background(), options{PaceMS: pace}); e == nil {
			t.Fatal(pace)
		}
	}
	if b, e := os.ReadFile(p); e != nil || string(b) != "original" {
		t.Fatal(string(b), e)
	}
}
func TestActualNativeRelationResponse(t *testing.T) {
	var r model.RelationReport
	if e := readJSON("testdata/relations.json", &r); e != nil {
		t.Fatal(e)
	}
	if e := checkRelations(r); e != nil {
		t.Fatal(e)
	}
	if r.Totals["apple-events"].Violated != 96 || r.Totals["positive-reward-on-repeat"].NotApplicable != 1329 {
		t.Fatal("public fixture arithmetic changed")
	}
	r.Totals["apple-events"].Satisfied++
	if e := checkRelations(r); e == nil {
		t.Fatal("inconsistent totals accepted")
	}
}
