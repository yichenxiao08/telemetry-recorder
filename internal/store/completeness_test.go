package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/signalscout/signalscout/internal/completeness"
	"github.com/signalscout/signalscout/internal/model"
)

const toyEpoch = "ee00000000000000000000000000ee00"

func traceID(n int) string { return fmt.Sprintf("%032x", n) }

func good(n int64) completeness.Bucket {
	return completeness.Bucket{Seen: n, Selected: n, Serialized: n, Admitted: n, Ack: n}
}

func manifestSignal(trace string, buckets map[string]completeness.Bucket) model.Signal {
	body, err := json.Marshal(completeness.Manifest{V: 1, Epoch: toyEpoch, Seq: 1, Final: true, Buckets: buckets})
	if err != nil {
		panic(err)
	}
	s := fixture("span:"+trace+"/"+completeness.ManifestSpanID, "span", trace, "resource", 200, 200)
	s.SpanID, s.ParentID, s.Name = completeness.ManifestSpanID, "0000000000000001", "signalscout.capture.manifest"
	s.Attrs = map[string]any{completeness.AttrKind: completeness.KindManifest, completeness.AttrJSON: string(body)}
	return s
}

func scopeSignal(chunk int, ids []string, final bool, expected, chunks int) model.Signal {
	record := completeness.ScopeRecord{V: 1, Epoch: toyEpoch, Chunk: chunk, IDs: ids, Final: final}
	if final {
		record.Chunks, record.Expected = chunks, expected
	}
	body, err := json.Marshal(record)
	if err != nil {
		panic(err)
	}
	s := fixture(fmt.Sprintf("span:%s/%016x", toyEpoch, chunk+1), "span", toyEpoch, "resource", 300, 300)
	s.SpanID, s.Name = fmt.Sprintf("%016x", chunk+1), completeness.ScopeWorkload
	s.Attrs = map[string]any{completeness.AttrKind: completeness.KindScope, completeness.AttrJSON: string(body)}
	return s
}

// addRun stores a root span, `children` decision spans with one decision event
// each, and optionally the run's manifest span.
func addRun(t *testing.T, s *Store, trace string, children int, manifest map[string]completeness.Bucket) {
	t.Helper()
	root := fixture("span:"+trace+"/0000000000000001", "span", trace, "resource", 100, 200)
	root.SpanID = "0000000000000001"
	signals := []model.Signal{root}
	for i := 0; i < children; i++ {
		id := fmt.Sprintf("%016x", i+2)
		child := fixture("span:"+trace+"/"+id, "span", trace, "resource", 120, 130)
		child.SpanID, child.ParentID, child.Name = id, "0000000000000001", "toy"
		event := fixture("event:"+trace+"/"+id, "event", trace, "resource", int64(150+i), int64(150+i))
		event.SpanID = id
		body := `{"version":1,"profile":{"id":"synthetic","version":"1"},"step":{"id":"toy"},"inputs":[{"name":"x","capture":"allowed","inline":1}],"chosen":{"option":"done"}}`
		event.Attrs = map[string]any{"signalscout.record.kind": "decision", "signalscout.record.json": body}
		signals = append(signals, child, event)
	}
	if manifest != nil {
		signals = append(signals, manifestSignal(trace, manifest))
	}
	if err := s.Submit(context.Background(), model.Batch{Signals: signals}); err != nil {
		t.Fatal(err)
	}
}

func toyProfile() model.AnalysisProfile {
	return model.AnalysisProfile{ID: "toy", Unit: "item", Workload: "request", Step: "toy", EnvelopeProfile: "synthetic", EnvelopeVersion: "1",
		Numeric:   []model.NumericFeature{{Name: "value", Selector: "inputs.x", Unit: "u", Epsilon: 1, Floor: 1}},
		Invariant: model.VersionInvariant{Declared: "used", Current: "n", Used: "v", Delta: 1}}
}

func selection() model.CohortSelection {
	return model.CohortSelection{From: 0, To: 1000, Limit: 50, Predicate: "inputs.x", Equals: "1"}
}

// mixedStore holds one run of every classification the scope can produce.
func mixedStore(t *testing.T) *Store {
	s := testStore(t, 0)
	partial := good(2)
	partial.Seen, partial.Selected, partial.Serialized, partial.Admitted, partial.Ack = 3, 3, 3, 2, 2
	partial.Rejected, partial.Dropped = 1, 1
	unknown := good(2)
	unknown.Ack, unknown.Unknown = 1, 1
	both := partial
	both.Ack, both.Unknown = 1, 1
	addRun(t, s, traceID(1), 2, map[string]completeness.Bucket{"@root": good(1), "p": good(2)}) // complete
	addRun(t, s, traceID(2), 2, map[string]completeness.Bucket{"@root": good(1), "p": partial}) // partial
	addRun(t, s, traceID(3), 2, map[string]completeness.Bucket{"@root": good(1), "p": unknown}) // unknown items
	addRun(t, s, traceID(4), 2, nil)                                                            // lost marker, listed by scope
	addRun(t, s, traceID(5), 2, nil)                                                            // legacy, unlisted
	addRun(t, s, traceID(7), 2, map[string]completeness.Bucket{"@root": good(1), "p": both})    // partial and unknown
	ids := []string{traceID(1), traceID(2), traceID(3), traceID(4), traceID(6), traceID(7)}     // 6 never stored
	if err := s.Submit(context.Background(), model.Batch{Signals: []model.Signal{scopeSignal(0, ids, true, 6, 1)}}); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestScopeReportClassifiesEveryRunAndNeverCountsTheScopeTrace(t *testing.T) {
	s := mixedStore(t)
	report, err := s.Completeness(context.Background(), toyEpoch)
	if err != nil {
		t.Fatal(err)
	}
	c := report.Counts
	if c.Complete != 1 || c.Partial != 2 || c.Unknown != 2 || c.Absent != 1 || c.Unassessed != 0 || c.PartialWithUnknown != 1 {
		t.Fatalf("counts: %+v", c)
	}
	if report.ListedRuns != 6 || report.ScopeExpected != 6 || !report.PopulationKnown || !report.ScopeFinal {
		t.Fatalf("scope: %+v", report)
	}
	if got := c.Complete + c.Partial + c.Unknown + c.Absent + c.Unassessed; got != report.ListedRuns {
		t.Fatalf("denominator %d != %d listed runs; the scope trace leaked into the count", got, report.ListedRuns)
	}
	if _, listed := report.Runs[toyEpoch]; listed {
		t.Fatal("scope trace counted as a run")
	}
	want := map[string]string{traceID(1): "complete", traceID(2): "partial", traceID(3): "unknown", traceID(4): "unknown", traceID(6): "absent", traceID(7): "partial"}
	for id, status := range want {
		if report.Runs[id] != status {
			t.Errorf("%s: got %s want %s", id, report.Runs[id], status)
		}
	}
	if c.LostItems != 2 || c.UnknownItems != 2 {
		t.Fatalf("items: %+v", c)
	}
}

func TestMissingScopeTailMakesPopulationUnknownNotComplete(t *testing.T) {
	s := testStore(t, 0)
	addRun(t, s, traceID(1), 2, map[string]completeness.Bucket{"@root": good(1), "p": good(2)})
	if err := s.Submit(context.Background(), model.Batch{Signals: []model.Signal{scopeSignal(0, []string{traceID(1), traceID(2)}, false, 0, 0)}}); err != nil {
		t.Fatal(err)
	}
	report, err := s.Completeness(context.Background(), toyEpoch)
	if err != nil {
		t.Fatal(err)
	}
	if report.PopulationKnown || report.ScopeFinal || report.Counts.Absent != 1 || report.Counts.Complete != 1 {
		t.Fatalf("%+v", report)
	}
	c, err := s.SelectCohort(context.Background(), "c", toyProfile(), selection())
	if err != nil {
		t.Fatal(err)
	}
	if c.Completeness == nil || c.Completeness.PopulationKnown {
		t.Fatalf("an incomplete scope inventory was presented as a known population: %+v", c.Completeness)
	}
}

func TestCohortDefaultExcludesPartialAndKeepsEveryCountVisible(t *testing.T) {
	s := mixedStore(t)
	ctx := context.Background()
	c, err := s.SelectCohort(ctx, "default", toyProfile(), selection())
	if err != nil {
		t.Fatal(err)
	}
	if c.Excluded["partial_capture"] != 2 {
		t.Fatalf("excluded: %v", c.Excluded)
	}
	for _, id := range c.RunIDs {
		if id == traceID(2) || id == traceID(7) {
			t.Fatalf("partial run %s selected by default", id)
		}
	}
	cc := c.Completeness
	if cc == nil || cc.Mode != "exclude_partial" {
		t.Fatalf("completeness: %+v", cc)
	}
	// Candidates are the six stored runs of the workload; the scope trace and the absent run are not among them.
	if cc.Candidates.Complete != 1 || cc.Candidates.Partial != 2 || cc.Candidates.Unknown != 2 || cc.Candidates.Unassessed != 1 || cc.Candidates.Absent != 1 || cc.Candidates.PartialWithUnknown != 1 {
		t.Fatalf("candidates: %+v", cc.Candidates)
	}
	if cc.Selected.Partial != 0 || cc.Selected.Complete != 1 || cc.Selected.Unknown != 2 || cc.Selected.Unassessed != 1 {
		t.Fatalf("selected: %+v", cc.Selected)
	}
	if cc.RunStatus[traceID(1)] != "complete" || cc.RunStatus[traceID(4)] != "unknown" || cc.RunStatus[traceID(5)] != "unassessed" {
		t.Fatalf("status: %v", cc.RunStatus)
	}
}

func TestIncludePartialKeepsPartialRunsAndTheirCounts(t *testing.T) {
	s := mixedStore(t)
	b := selection()
	b.IncludePartial = true
	c, err := s.SelectCohort(context.Background(), "inc", toyProfile(), b)
	if err != nil {
		t.Fatal(err)
	}
	if c.Excluded["partial_capture"] != 0 || len(c.RunIDs) != 6 {
		t.Fatalf("runs %v excluded %v", c.RunIDs, c.Excluded)
	}
	cc := c.Completeness
	if cc.Mode != "include_partial" || cc.Selected.Partial != 2 || cc.Selected.PartialWithUnknown != 1 || cc.Selected.LostItems != 2 || cc.Selected.UnknownItems != 2 {
		t.Fatalf("%+v", cc)
	}
	if cc.RunStatus[traceID(2)] != "partial" {
		t.Fatalf("partial run status hidden: %v", cc.RunStatus)
	}
}

func TestRequireCompleteRejectsUnknownUnassessedAbsentAndUnknownPopulation(t *testing.T) {
	ctx := context.Background()
	b := selection()
	b.RequireComplete = true
	if _, err := mixedStore(t).SelectCohort(ctx, "strict", toyProfile(), b); err == nil || !strings.Contains(err.Error(), "unknown=2") || !strings.Contains(err.Error(), "unassessed=1") {
		t.Fatalf("unknown/unassessed runs accepted: %v", err)
	}
	legacy := testStore(t, 0)
	addRun(t, legacy, traceID(1), 2, nil)
	if _, err := legacy.SelectCohort(ctx, "legacy", toyProfile(), b); err == nil || !strings.Contains(err.Error(), "unassessed=1") {
		t.Fatalf("legacy run treated as complete: %v", err)
	}
	both := b
	both.IncludePartial = true
	if _, err := legacy.SelectCohort(ctx, "contradiction", toyProfile(), both); err == nil {
		t.Fatal("contradictory flags accepted")
	}
	// Every candidate complete, scope final and matching: accepted, with the mode pinned.
	clean := testStore(t, 0)
	addRun(t, clean, traceID(1), 2, map[string]completeness.Bucket{"@root": good(1), "p": good(2)})
	addRun(t, clean, traceID(2), 2, map[string]completeness.Bucket{"@root": good(1), "p": good(2)})
	if err := clean.Submit(ctx, model.Batch{Signals: []model.Signal{scopeSignal(0, []string{traceID(1), traceID(2)}, true, 2, 1)}}); err != nil {
		t.Fatal(err)
	}
	c, err := clean.SelectCohort(ctx, "ok", toyProfile(), b)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.RunIDs) != 2 || c.Completeness.Mode != "require_complete" || !c.Completeness.PopulationKnown {
		t.Fatalf("%+v", c)
	}
	// One absent run is enough to refuse a "complete" denominator.
	missing := testStore(t, 0)
	addRun(t, missing, traceID(1), 2, map[string]completeness.Bucket{"@root": good(1), "p": good(2)})
	if err = missing.Submit(ctx, model.Batch{Signals: []model.Signal{scopeSignal(0, []string{traceID(1), traceID(2)}, true, 2, 1)}}); err != nil {
		t.Fatal(err)
	}
	if _, err = missing.SelectCohort(ctx, "gap", toyProfile(), b); err == nil || !strings.Contains(err.Error(), "absent=1") {
		t.Fatalf("absent run ignored: %v", err)
	}
}

func TestLostMarkerAndLostScopeNeverYieldComplete(t *testing.T) {
	ctx := context.Background()
	s := testStore(t, 0)
	addRun(t, s, traceID(1), 2, nil) // marker lost, scope lost
	c, err := s.SelectCohort(ctx, "c", toyProfile(), selection())
	if err != nil {
		t.Fatal(err)
	}
	if c.Completeness != nil {
		t.Fatalf("no evidence at all must not fabricate a completeness block: %+v", c.Completeness)
	}
	b := selection()
	b.RequireComplete = true
	if _, err = s.SelectCohort(ctx, "strict", toyProfile(), b); err == nil {
		t.Fatal("run without marker or scope accepted as complete")
	}
}

func TestDuplicateManifestAndScopeDeliveryIsIdempotent(t *testing.T) {
	s := mixedStore(t)
	before, err := s.Completeness(context.Background(), toyEpoch)
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{traceID(1), traceID(2), traceID(3), traceID(4), traceID(6), traceID(7)}
	for i := 0; i < 2; i++ {
		addRun(t, s, traceID(1), 2, map[string]completeness.Bucket{"@root": good(1), "p": good(2)})
		if err = s.Submit(context.Background(), model.Batch{Signals: []model.Signal{scopeSignal(0, ids, true, 6, 1)}}); err != nil {
			t.Fatal(err)
		}
	}
	after, err := s.Completeness(context.Background(), toyEpoch)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(before)
	b, _ := json.Marshal(after)
	if string(a) != string(b) {
		t.Fatalf("duplicate delivery changed the report:\n%s\n%s", a, b)
	}
}

func TestScopeTraceIsNeverACohortPopulation(t *testing.T) {
	s := mixedStore(t)
	p := toyProfile()
	p.Workload = completeness.ScopeWorkload
	if _, err := s.SelectCohort(context.Background(), "scope", p, selection()); err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("scope workload selectable: %v", err)
	}
	if _, _, err := s.Decisions(context.Background(), p, selection()); err == nil {
		t.Fatal("scope workload readable as decisions")
	}
	// A normal profile never lists the scope trace among its candidates.
	c, err := s.SelectCohort(context.Background(), "normal", toyProfile(), selection())
	if err != nil {
		t.Fatal(err)
	}
	for id := range c.Completeness.RunStatus {
		if id == toyEpoch {
			t.Fatal("scope trace in cohort")
		}
	}
	total := c.Completeness.Candidates
	if got := total.Complete + total.Partial + total.Unknown + total.Unassessed; got != 6 {
		t.Fatalf("candidate denominator %d includes more than the six stored runs", got)
	}
}

func TestOversizedOrForeignManifestIsUnknownNeverComplete(t *testing.T) {
	s := testStore(t, 0)
	addRun(t, s, traceID(1), 2, map[string]completeness.Bucket{"@root": good(1), "p": good(2)})
	bad := manifestSignal(traceID(1), map[string]completeness.Bucket{"@root": good(1)})
	bad.Attrs[completeness.AttrJSON] = strings.Repeat("x", completeness.MaxJSONBytes+1)
	if err := s.Submit(context.Background(), model.Batch{Signals: []model.Signal{bad}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Submit(context.Background(), model.Batch{Signals: []model.Signal{scopeSignal(0, []string{traceID(1)}, true, 1, 1)}}); err != nil {
		t.Fatal(err)
	}
	report, err := s.Completeness(context.Background(), toyEpoch)
	if err != nil {
		t.Fatal(err)
	}
	if report.Runs[traceID(1)] != "unknown" {
		t.Fatalf("%+v", report)
	}
}

// legacyCohort mirrors model.Cohort exactly as it was before completeness
// existed. Legacy cohort bytes must equal what this struct produces.
type legacyCohort struct {
	ID          string                `json:"id"`
	Profile     model.AnalysisProfile `json:"profile"`
	Selection   legacySelection       `json:"selection"`
	CreatedAt   int64                 `json:"created_at"`
	RunIDs      []string              `json:"run_ids"`
	LabelIDs    []string              `json:"label_ids"`
	DecisionIDs map[string]string     `json:"decision_ids"`
	Sessions    map[string]string     `json:"sessions"`
	Excluded    map[string]int        `json:"excluded"`
}
type legacySelection struct {
	From      int64  `json:"from"`
	To        int64  `json:"to"`
	Limit     int    `json:"limit"`
	Dimension string `json:"dimension,omitempty"`
	Value     string `json:"value,omitempty"`
	Predicate string `json:"predicate,omitempty"`
	Equals    string `json:"equals,omitempty"`
}

func TestLegacyCohortBytesAreIdentical(t *testing.T) {
	s := testStore(t, 0)
	for n := 1; n <= 3; n++ {
		addRun(t, s, traceID(n), 2, nil)
	}
	c, err := s.SelectCohort(context.Background(), "legacy", toyProfile(), selection())
	if err != nil {
		t.Fatal(err)
	}
	if c.Completeness != nil || len(c.RunIDs) != 3 {
		t.Fatalf("legacy cohort grew completeness: %+v", c)
	}
	c.CreatedAt = 42
	got, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	old := legacyCohort{ID: c.ID, Profile: c.Profile, Selection: legacySelection{From: c.Selection.From, To: c.Selection.To, Limit: c.Selection.Limit,
		Predicate: c.Selection.Predicate, Equals: c.Selection.Equals}, CreatedAt: 42, RunIDs: c.RunIDs, LabelIDs: c.LabelIDs,
		DecisionIDs: c.DecisionIDs, Sessions: c.Sessions, Excluded: c.Excluded}
	want, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("legacy bytes changed:\n got %s\nwant %s", got, want)
	}
	for _, key := range []string{"completeness", "include_partial", "require_complete", "partial_capture"} {
		if strings.Contains(string(got), key) {
			t.Fatalf("legacy cohort mentions %q", key)
		}
	}
	// Saved cohorts read back unchanged.
	if err = s.SaveCohort(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	read, err := s.Cohort(context.Background(), "legacy")
	if err != nil {
		t.Fatal(err)
	}
	again, _ := json.Marshal(read)
	if string(again) != string(got) {
		t.Fatalf("round trip changed bytes:\n%s\n%s", again, got)
	}
}

func TestCompletenessQueriesUseIndexes(t *testing.T) {
	s := testStore(t, 0)
	for _, tc := range []struct{ name, query, index string }{
		{"scope records", "SELECT data FROM signals WHERE run_id=? AND kind='span' ORDER BY ts_ns,evidence_id LIMIT ?", "signals_run"},
		{"scope discovery", "SELECT run_id FROM runs INDEXED BY runs_workload WHERE workload=? AND started_at>=? AND started_at<? ORDER BY started_at,run_id LIMIT ?", "runs_workload"},
		{"manifest by id", "SELECT data FROM signals WHERE evidence_id=?", "sqlite_autoindex_signals"},
		{"span count", "SELECT count(*) FROM signals WHERE run_id=? AND kind='span'", "signals_run"},
		{"run exists", "SELECT 1 FROM runs WHERE run_id=?", "sqlite_autoindex_runs"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := make([]any, strings.Count(tc.query, "?"))
			for i := range args {
				args[i] = 1
			}
			rows, err := s.reader.Query("EXPLAIN QUERY PLAN "+tc.query, args...)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = rows.Close() }()
			var plan strings.Builder
			for rows.Next() {
				var a, b, c int
				var detail string
				if err = rows.Scan(&a, &b, &c, &detail); err != nil {
					t.Fatal(err)
				}
				plan.WriteString(detail + "\n")
			}
			if !strings.Contains(plan.String(), tc.index) || strings.Contains(plan.String(), "SCAN signals") || strings.Contains(plan.String(), "SCAN runs") {
				t.Fatalf("plan for %s: %s", tc.name, plan.String())
			}
		})
	}
}
