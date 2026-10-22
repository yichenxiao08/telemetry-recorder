package report

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/signalscout/signalscout/internal/cohort"
	"github.com/signalscout/signalscout/internal/lineage"
	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/provenance"
	"github.com/signalscout/signalscout/internal/store"
)

func raw(v any) json.RawMessage { b, _ := json.Marshal(v); return b }
func numbers(v any) map[string]float64 {
	out := map[string]float64{}
	var tree any
	_ = json.Unmarshal(raw(v), &tree)
	var walk func(any, string, int)
	walk = func(v any, path string, depth int) {
		if depth > 12 {
			return
		}
		switch x := v.(type) {
		case float64:
			if !math.IsNaN(x) && !math.IsInf(x, 0) && math.Abs(x) <= 1<<53 {
				out[path] = x
			}
		case map[string]any:
			switch marker := x["unknown"].(type) {
			case string:
				return
			case bool:
				if marker {
					return
				}
			}
			if capture, ok := x["capture"].(string); ok && capture != "allowed" {
				return
			}
			for k, v := range x {
				walk(v, path+"/"+escape(k), depth+1)
			}
		case []any:
			for i, v := range x {
				walk(v, fmt.Sprintf("%s/%d", path, i), depth+1)
			}
		}
	}
	walk(tree, "", 0)
	return out
}
func escape(s string) string {
	out := ""
	for _, c := range s {
		switch c {
		case '~':
			out += "~0"
		case '/':
			out += "~1"
		default:
			out += string(c)
		}
	}
	return out
}
func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func validateRequest(q Request) error {
	if q.Version != Version || len(q.Comparisons) > 2 || len(q.Traces) > 4 || len(q.Records) > 64 {
		return fmt.Errorf("report version 1; <=2 comparisons, <=4 traces, <=64 records required")
	}
	for _, t := range q.Traces {
		if t.Target == "" || t.Budget.Depth < 0 || t.Budget.Depth > 64 || t.Budget.Nodes < 1 || t.Budget.Nodes > 128 {
			return fmt.Errorf("trace needs target, depth 0..64 and nodes 1..128")
		}
		if t.Profile != nil {
			if err := provenance.ValidateProfile(*t.Profile); err != nil {
				return err
			}
		}
	}
	return nil
}

// Build uses only existing engines and pinned cohort decisions. Source outputs
// are retained verbatim as JSON; no score, ordering or evidence policy changes.
func Build(ctx context.Context, r *store.Reader, q Request) (Document, []Fact, error) {
	doc := Document{Version: Version, Request: q, Sources: []Source{}, Claims: []Claim{}, Notes: []string{
		"Observations and associations only. Hypotheses require a separately supplied JSON file.",
		"Sampled capture is incomplete: unsampled calls remain unknown, never satisfied. Recorded counts do not estimate unseen calls.",
		"Verification checks scoped citations and structured quotations, not semantic truth, causal explanations or subjective quality.",
		"Session counts are distinct recorded session identifiers; independence is not established when independence_known is false.",
	}}
	facts := []Fact{}
	if err := validateRequest(q); err != nil {
		return doc, nil, err
	}
	rq := store.ReportRequest{From: q.From, To: q.To, Limit: q.Limit, Cohorts: append([]string{}, q.Cohorts...)}
	for _, p := range q.Comparisons {
		rq.Cohorts = append(rq.Cohorts, p.Problem, p.Reference)
	}
	// Deduplicate metadata reads before applying the source budget.
	uniq := map[string]bool{}
	for _, id := range rq.Cohorts {
		uniq[id] = true
	}
	rq.Cohorts = keys(uniq)
	for _, c := range q.Records {
		if c.Kind != "native" || c.ID == "" {
			return doc, nil, fmt.Errorf("record source needs native id")
		}
		rq.NativeIDs = append(rq.NativeIDs, c.ID)
	}
	snap, err := r.ReportRead(ctx, rq)
	if err != nil {
		return doc, nil, err
	}
	sigs := map[string]model.Signal{}
	for _, s := range snap.Signals {
		sigs[s.ID] = s
	}
	add := func(f Fact) { facts = append(facts, f); doc.Claims = append(doc.Claims, claimFor(f)) }
	source := func(id, kind string, v any) { doc.Sources = append(doc.Sources, Source{id, kind, raw(v)}) }
	native := func(id, run, cid string) (Citation, error) {
		s, ok := sigs[id]
		if !ok || s.RunID != run {
			return Citation{}, fmt.Errorf("native citation %q missing or outside stated run/time scope", id)
		}
		if cid != "" {
			c, ok := snap.Cohorts[cid]
			if !ok || c.DecisionIDs[run] != id {
				return Citation{}, fmt.Errorf("citation is not this cohort's pinned decision")
			}
		}
		return Citation{ID: id, Kind: "native", RunID: run, CohortID: cid}, nil
	}
	for _, c := range q.Records {
		actual, err := native(c.ID, c.RunID, c.CohortID)
		if err != nil {
			return doc, nil, err
		}
		s := sigs[c.ID]
		source("record/"+c.ID, "record", s)
		add(Fact{ID: "record/" + c.ID, Tier: "observation", State: "captured", Evidence: []Citation{actual}, Numbers: numbers(s), Detail: raw(s)})
	}
	for _, id := range keys(snap.Cohorts) {
		c := snap.Cohorts[id]
		citation := Citation{ID: id, Kind: "cohort", CohortID: id}
		var complete any = c.Completeness
		state := "assessed_snapshot"
		if c.Completeness == nil {
			state = "unassessed"
			complete = map[string]string{"status": "unassessed"}
		}
		source("completeness/"+id, "cohort_completeness", complete)
		add(Fact{ID: "completeness/" + id, Tier: "observation", State: state, Evidence: []Citation{citation}, Numbers: numbers(complete), Detail: raw(complete)})
		if c.Relations == nil {
			continue
		}
		rel := c.Relations
		source("relations/"+id, "relations", rel)
		emit := func(scope, run string, rows map[string]*model.RelationCounts) error {
			for _, rule := range keys(rows) {
				counts := rows[rule]
				cites := []Citation{citation}
				for _, eid := range counts.ViolationEvidence {
					s, ok := sigs[eid]
					if !ok || run != "" && s.RunID != run {
						return fmt.Errorf("pinned relation witness missing or wrong run")
					}
					cites = append(cites, Citation{ID: eid, Kind: "native", RunID: s.RunID})
				}
				add(Fact{ID: "relations/" + id + "/" + scope + "/" + rule, Tier: "observation", State: "recorded_counts", Evidence: cites, Numbers: numbers(counts), Detail: raw(counts)})
			}
			return nil
		}
		if err = emit("step", "", rel.Totals); err != nil {
			return doc, nil, err
		}
		for _, run := range keys(rel.Runs) {
			if err = emit("run/"+run, run, rel.Runs[run]); err != nil {
				return doc, nil, err
			}
		}
	}
	observations := func(c model.Cohort) ([]model.DecisionObservation, error) {
		xs := []model.DecisionObservation{}
		for _, run := range c.RunIDs {
			id := c.DecisionIDs[run]
			if _, err := native(id, run, c.ID); err != nil {
				return nil, err
			}
			o, ok := provenance.Decode(sigs[id], c.Profile)
			if !ok {
				return nil, fmt.Errorf("pinned decision unsupported")
			}
			o.Session = c.Sessions[run]
			xs = append(xs, o)
		}
		return xs, nil
	}
	for index, pair := range q.Comparisons {
		p, ref := snap.Cohorts[pair.Problem], snap.Cohorts[pair.Reference]
		ps, err := observations(p)
		if err != nil {
			return doc, nil, err
		}
		rs, err := observations(ref)
		if err != nil {
			return doc, nil, err
		}
		result, err := cohort.Compare(p, ref, ps, rs)
		if err != nil {
			return doc, nil, err
		}
		prefix := fmt.Sprintf("comparison/%d", index)
		source(prefix, "explain_cohort", result)
		emit := func(section string, index int, f cohort.Finding, detail any) error {
			cites := []Citation{{ID: p.ID, Kind: "cohort", CohortID: p.ID}, {ID: ref.ID, Kind: "cohort", CohortID: ref.ID}}
			for _, id := range f.EvidenceIDs {
				s, ok := sigs[id]
				if !ok {
					return fmt.Errorf("comparison witness missing")
				}
				cid := p.ID
				if ref.DecisionIDs[s.RunID] == id {
					cid = ref.ID
				}
				c, err := native(id, s.RunID, cid)
				if err != nil {
					return err
				}
				cites = append(cites, c)
			}
			a := &Association{ProblemNumerator: f.ProblemNumerator, ProblemDenominator: f.ProblemObserved, ReferenceNumerator: f.ReferenceNumerator, ReferenceDenominator: f.ReferenceObserved, ProblemRuns: f.ProblemRuns, ReferenceRuns: f.ReferenceRuns, ProblemSessions: f.ProblemSessions, ReferenceSessions: f.ReferenceSessions, SharedSessions: f.SharedSessions, IndependenceKnown: f.IndependenceKnown, ProblemMissing: f.ProblemMissing, ReferenceMissing: f.ReferenceMissing, Excluded: f.Exclusions, Selection: string(raw(map[string]model.CohortSelection{"problem": p.Selection, "reference": ref.Selection})), Overlap: "Disjoint run IDs required; shared sessions counted, independence not inferred."}
			add(Fact{ID: fmt.Sprintf("%s/%s/%d", prefix, section, index), Tier: f.Tier, State: f.Status, Evidence: cites, Numbers: numbers(detail), Association: a, Detail: raw(detail)})
			return nil
		}
		if result.Presentation != nil {
			for i, f := range result.Presentation.Problem {
				if err = emit("the_problem", i, f, f); err != nil {
					return doc, nil, err
				}
			}
			for i, f := range result.Presentation.Median {
				if err = emit("candidate_median", i, f, f); err != nil {
					return doc, nil, err
				}
			}
			for i, f := range result.Presentation.Tail {
				if err = emit("candidate_tail", i, f.Finding, f); err != nil {
					return doc, nil, err
				}
			}
		} else {
			for i, f := range result.Differences {
				if err = emit("legacy_median", i, f, f); err != nil {
					return doc, nil, err
				}
			}
		}
		for i, v := range result.Invariants {
			s := sigs[v.EvidenceID]
			cite, err := native(v.EvidenceID, s.RunID, "")
			if err != nil {
				return doc, nil, err
			}
			add(Fact{ID: fmt.Sprintf("%s/invariant/%d", prefix, i), Tier: "observation", State: v.Status, Evidence: []Citation{cite}, Numbers: map[string]float64{}, Detail: raw(v)})
		}
	}
	for i, t := range q.Traces {
		data, err := r.TraceRead(ctx, store.TraceReadRequest{Targets: []string{t.Target}, RunID: t.RunID, From: q.From, To: q.To, Limit: q.Limit})
		if err != nil {
			return doc, nil, err
		}
		result := lineage.Walk(data.Signals, t.Target, t.Budget, t.Profile)
		if len(result.Nodes) > 0 && t.RunID != "" && result.Nodes[0].RunID != t.RunID {
			return doc, nil, fmt.Errorf("trace target belongs to a different run")
		}
		prefix := fmt.Sprintf("trace/%d", i)
		source(prefix, "trace", struct {
			Scope store.TraceScope `json:"scope"`
			Trace lineage.Report   `json:"trace"`
		}{data.Scope, result})
		coverage := Fact{ID: prefix + "/coverage", Tier: "observation", State: "observed", Evidence: []Citation{}, Numbers: map[string]float64{"nodes_visited": float64(result.Visited), "truncations": float64(result.Truncations), "records_read": float64(data.Scope.RecordsRead)}, Detail: raw(data.Scope)}
		if len(result.Findings) > 0 || result.Truncations > 0 || data.Scope.Truncated || len(result.Nodes) == 0 {
			coverage.State = "unknown"
		}
		if len(result.Nodes) > 0 {
			coverage.Evidence = append(coverage.Evidence, Citation{ID: result.Nodes[0].Evidence.ID, Kind: "native", RunID: result.Nodes[0].RunID})
		}
		add(coverage)
		for n, node := range result.Nodes {
			cites := []Citation{{ID: node.Evidence.ID, Kind: "native", RunID: node.RunID}}
			if node.Output.Ref != "" && len(result.Findings) == 0 && !data.Scope.Truncated {
				cites = append(cites, Citation{ID: node.Output.Ref, Kind: "logical", RunID: node.RunID})
				cites = append(cites, Citation{ID: node.ID, Kind: "logical", RunID: node.RunID})
			}
			// The existing walker resolves these native and logical IDs; integrity and
			// truncation findings remain in the original source output, never discarded.
			add(Fact{ID: fmt.Sprintf("%s/node/%d", prefix, n), Tier: "observation", State: node.Status, Evidence: cites, Numbers: numbers(node), Detail: raw(node)})
			add(Fact{ID: fmt.Sprintf("%s/invariant/%d", prefix, n), Tier: "observation", State: node.Invariant.Status, Evidence: cites, Numbers: numbers(node.Invariant), Detail: raw(node.Invariant)})
		}
	}
	if len(facts) > 4096 {
		return doc, nil, fmt.Errorf("report exceeds 4096 claims; narrow source selection")
	}
	for _, f := range facts {
		if len(f.Numbers) > 512 {
			return doc, nil, fmt.Errorf("fact exceeds 512 numeric quotations")
		}
	}
	if err = Size(doc); err != nil {
		return doc, nil, err
	}
	return doc, facts, nil
}

// VerifyDocument rebuilds all facts; submitted source JSON is never trusted.
func VerifyDocument(ctx context.Context, r *store.Reader, doc Document) (Verification, error) {
	if doc.Version != Version || len(doc.Claims) > 4096 {
		return Verification{}, fmt.Errorf("version 1 and <=4096 claims required")
	}
	fresh, facts, err := Build(ctx, r, doc.Request)
	if err != nil {
		return Verification{}, err
	}
	if string(raw(doc.Sources)) != string(raw(fresh.Sources)) {
		return Verification{}, fmt.Errorf("submitted source outputs differ from rebuilt store evidence")
	}
	if string(raw(doc.Notes)) != string(raw(fresh.Notes)) {
		return Verification{}, fmt.Errorf("required coverage and verifier limitation notes changed or missing")
	}
	out := Verify(doc.Claims, facts)
	return out, nil
}
