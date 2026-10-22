// Synthetic structured-report acceptance through ordinary OTLP and a real server.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/cohort"
	"github.com/signalscout/signalscout/internal/lineage"
	"github.com/signalscout/signalscout/internal/model"
	evidence "github.com/signalscout/signalscout/internal/report"
	"github.com/signalscout/signalscout/internal/store"
)

func encode(v any) []byte { b, _ := json.Marshal(v); return b }
func attr(k string, v any) map[string]any {
	value := map[string]any{}
	switch x := v.(type) {
	case string:
		value["stringValue"] = x
	case bool:
		value["boolValue"] = x
	}
	return map[string]any{"key": k, "value": value}
}
func profile() model.AnalysisProfile {
	zero := 0.0
	one := 1.0
	return model.AnalysisProfile{ID: "synthetic-report-v1", Unit: "decision", Workload: "synthetic.adjust", Step: "adjust", EnvelopeProfile: "synthetic.report", EnvelopeVersion: "1", RoleAuthor: "synthetic-v1", ReferenceTail: true, Numeric: []model.NumericFeature{{Name: "input", Selector: "inputs.x", Unit: "count", Epsilon: 1, Floor: 1, Role: "candidate", Rationale: "Synthetic recorded input"}}, Categories: []model.CategoryFeature{{Name: "flag", Selector: "inputs.flag", Role: "symptom", Rationale: "Synthetic operational flag unrelated to the rule"}}, Invariant: model.VersionInvariant{Declared: "inputs.declared", Current: "inputs.current", Used: "inputs.used", Delta: 1}, Relations: &model.RelationPolicy{Version: 1, Rules: []model.RelationRule{{ID: "increment", Unit: "count", Tolerance: &zero, Assert: model.Expression{Op: "eq", Args: []model.Expression{{Op: "field", Field: "output"}, {Op: "add", Args: []model.Expression{{Op: "field", Field: "inputs.x"}, {Op: "number", Number: &one}}}}}}}}}
}
func send(ctx context.Context, client *http.Client, endpoint string, start int64) error {
	spans := []any{}
	for i := 0; i < 40; i++ {
		run := fmt.Sprintf("%032x", i+1)
		x := 0
		if i < 20 {
			x = 10
		}
		out := x + 1
		if i%10 == 0 {
			out = x + 3
		}
		fields := []any{map[string]any{"name": "x", "capture": "allowed", "inline": x}, map[string]any{"name": "flag", "capture": "allowed", "inline": i%4 < 2}, map[string]any{"name": "group", "capture": "allowed", "inline": i < 20}, map[string]any{"name": "declared", "capture": "off"}}
		env := map[string]any{"version": 1, "id": "decision:" + run + "/adjust", "run_id": run, "profile": map[string]string{"id": "synthetic.report", "version": "1"}, "step": map[string]string{"id": "adjust"}, "inputs": fields, "output": map[string]any{"name": "adjust", "ref": "value:" + run + "/adjust", "capture": "allowed", "inline": out}}
		ts := start + int64(i)*1e6
		spans = append(spans, map[string]any{"traceId": run, "spanId": fmt.Sprintf("%016x", i+1), "name": "synthetic.adjust", "startTimeUnixNano": strconv.FormatInt(ts, 10), "endTimeUnixNano": strconv.FormatInt(ts+500000, 10), "status": map[string]any{"code": 1}, "attributes": []any{attr("signalscout.session.id", fmt.Sprintf("synthetic-session-%d", i))}, "events": []any{map[string]any{"name": "signalscout.decision", "timeUnixNano": strconv.FormatInt(ts+100000, 10), "attributes": []any{attr("signalscout.record.kind", "decision"), attr("signalscout.record.id", env["id"]), attr("signalscout.record.json", string(encode(env)))}}}})
	}
	payload := map[string]any{"resourceSpans": []any{map[string]any{"resource": map[string]any{"attributes": []any{attr("service.name", "synthetic.reportcheck")}}, "scopeSpans": []any{map[string]any{"spans": spans}}}}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/v1/traces", bytes.NewReader(encode(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("intake %d: %s", res.StatusCode, body)
	}
	return nil
}

func lane(ctx context.Context, path string) ([]any, error) {
	if path == "" {
		return []any{}, nil
	}
	body, err := os.ReadFile("benchmark/results/lane-01.json")
	if err != nil {
		return nil, err
	}
	var frozen struct {
		Cases []struct {
			Seed    int
			Variant string
			Result  cohort.Result
		}
	}
	if err = json.Unmarshal(body, &frozen); err != nil {
		return nil, err
	}
	r, err := store.OpenReader(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	checks := []any{}
	for _, c := range frozen.Cases {
		p, ref := c.Result.Problem, c.Result.Reference
		target := p.DecisionIDs[p.RunIDs[0]]
		q := evidence.Request{Version: 1, From: p.Selection.From, To: p.Selection.To, Limit: 10000, Comparisons: []evidence.Comparison{{Problem: p.ID, Reference: ref.ID}}, Traces: []evidence.Trace{{Target: target, RunID: p.RunIDs[0], Profile: &p.Profile, Budget: lineage.Options{Depth: 16, Nodes: 128}}}}
		doc, _, err := evidence.Build(ctx, r, q)
		if err != nil {
			return nil, err
		}
		v, err := evidence.VerifyDocument(ctx, r, doc)
		if err != nil || v.Invalid != 0 || v.Unsupported != 0 {
			return nil, fmt.Errorf("lane report: %+v %v", v, err)
		}
		checks = append(checks, map[string]any{"seed": c.Seed, "variant": c.Variant, "claims": v.Valid, "citations": v.ValidCitations, "citation_validity_rate": v.CitationValidityRate, "claim_support_rate": v.ClaimSupportRate})
	}
	return checks, nil
}

func acceptedRelations(ctx context.Context, path string) (any, error) {
	if path == "" {
		return nil, nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var saved struct {
		Database string               `json:"database"`
		Report   model.RelationReport `json:"report"`
	}
	if err = json.Unmarshal(body, &saved); err != nil {
		return nil, err
	}
	r, err := store.OpenReader(ctx, saved.Database)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	c, err := r.Cohort(ctx, "minority")
	if err != nil {
		return nil, err
	}
	q := evidence.Request{Version: 1, From: saved.Report.From, To: saved.Report.To, Limit: 10000, Cohorts: []string{"minority"}, Comparisons: []evidence.Comparison{{Problem: "flag-true", Reference: "flag-false"}}, Traces: []evidence.Trace{{Target: c.DecisionIDs[c.RunIDs[0]], RunID: c.RunIDs[0], Profile: &c.Profile, Budget: lineage.Options{Depth: 8, Nodes: 32}}}}
	doc, _, err := evidence.Build(ctx, r, q)
	if err != nil {
		return nil, err
	}
	v, err := evidence.VerifyDocument(ctx, r, doc)
	if err != nil || v.Invalid != 0 || v.Unsupported != 0 {
		return nil, fmt.Errorf("existing relations report: %+v %v", v, err)
	}
	return map[string]any{"valid_claims": v.Valid, "citation_validity_rate": v.CitationValidityRate, "claim_support_rate": v.ClaimSupportRate, "capture_off_unknown": saved.Report.Totals["capture-off"].Unknown, "original_rule_counts_retained": true}, nil
}

func run() error {
	binary := flag.String("binary", "bin/signalscout.exe", "real serving binary")
	out := flag.String("out", ".cache/structured-report/acceptance.json", "new synthetic result")
	laneDB := flag.String("lane-db", "", "optional COPY of the frozen lane-01 database")
	relationsResult := flag.String("relations-result", "", "optional fresh output from the unchanged scripts/relationscheck")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := os.MkdirAll(filepath.Dir(*out), 0700); err != nil {
		return err
	}
	dir, err := os.MkdirTemp(filepath.Dir(*out), "fixture-")
	if err != nil {
		return err
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return err
	}
	bin, err := filepath.Abs(*binary)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	address := listener.Addr().String()
	_ = listener.Close()
	endpoint := "http://" + address
	db := filepath.Join(dir, "fresh.db")
	log, err := os.Create(filepath.Join(dir, "serve.log"))
	if err != nil {
		return err
	}
	defer func() { _ = log.Close() }()
	server := exec.CommandContext(ctx, bin, "--db", db, "serve", "--listen", address, "--retention", "0")
	server.Stdout, server.Stderr = log, log
	if err = server.Start(); err != nil {
		return err
	}
	defer func() { _ = server.Process.Kill(); _ = server.Wait() }()
	client := &http.Client{Timeout: 5 * time.Second}
	ready := false
	for i := 0; i < 100; i++ {
		res, e := client.Get(endpoint + "/")
		if e == nil {
			_ = res.Body.Close()
			ready = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !ready {
		return fmt.Errorf("server not ready; see %s", dir)
	}
	start := time.Now().UnixNano()
	if err = send(ctx, client, endpoint, start); err != nil {
		return err
	}
	call := func(args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, bin, append([]string{"--db", db}, args...)...)
		data, e := cmd.CombinedOutput()
		if e != nil {
			return nil, fmt.Errorf("%v: %w: %s", args, e, data)
		}
		return data, nil
	}
	data, err := call("runs", "list", "--workload", "synthetic.adjust")
	if err != nil {
		return err
	}
	var runs []model.Run
	if err = json.Unmarshal(data, &runs); err != nil || len(runs) != 40 {
		return fmt.Errorf("runs list: %d, %v", len(runs), err)
	}
	p := profile()
	profilePath := filepath.Join(dir, "profile.json")
	if err = os.WriteFile(profilePath, encode(p), 0600); err != nil {
		return err
	}
	from, to := start-int64(time.Second), start+int64(time.Second)
	bounds := []string{"--profile", profilePath, "--from", time.Unix(0, from).UTC().Format(time.RFC3339Nano), "--to", time.Unix(0, to).UTC().Format(time.RFC3339Nano), "--limit", "40"}
	cohorts := map[string]model.Cohort{}
	for _, selection := range []struct {
		id   string
		args []string
	}{{"synthetic-p", []string{"--predicate", "inputs.group", "--equals", "true"}}, {"synthetic-r", []string{"--predicate", "inputs.group", "--equals", "false"}}, {"synthetic-v", []string{"--violates", "increment"}}, {"synthetic-flag-true", []string{"--predicate", "inputs.flag", "--equals", "true"}}, {"synthetic-flag-false", []string{"--predicate", "inputs.flag", "--equals", "false"}}} {
		args := append([]string{"cohort", "create", selection.id, "--endpoint", endpoint}, selection.args...)
		args = append(args, bounds...)
		b, e := call(args...)
		if e != nil {
			return e
		}
		var c model.Cohort
		if e = json.Unmarshal(b, &c); e != nil {
			return e
		}
		cohorts[c.ID] = c
	}
	violating := cohorts["synthetic-v"]
	if len(violating.RunIDs) != 4 {
		return fmt.Errorf("expected four minority violations")
	}
	truth := map[string]int{}
	for _, label := range []string{"true", "false"} {
		selected := cohorts["synthetic-flag-"+label]
		for _, id := range selected.RunIDs {
			key := "satisfied_flag_" + label
			for _, v := range violating.RunIDs {
				if v == id {
					key = "violated_flag_" + label
				}
			}
			truth[key]++
		}
		if truth["violated_flag_"+label] != 2 || truth["satisfied_flag_"+label] != 18 {
			return fmt.Errorf("unexpected flag table: %v", truth)
		}
	}
	q := evidence.Request{Version: 1, From: from, To: to, Limit: 1000, Cohorts: []string{"synthetic-v"}, Comparisons: []evidence.Comparison{{Problem: "synthetic-p", Reference: "synthetic-r"}}, Traces: []evidence.Trace{{Target: violating.DecisionIDs[violating.RunIDs[0]], RunID: violating.RunIDs[0], Profile: &p, Budget: lineage.Options{Depth: 8, Nodes: 32}}}}
	request := filepath.Join(dir, "request.json")
	if err = os.WriteFile(request, encode(q), 0600); err != nil {
		return err
	}
	data, err = call("report", "build", "--request", request, "--json")
	if err != nil {
		return err
	}
	var doc evidence.Document
	if err = evidence.Decode(data, &doc); err != nil {
		return err
	}
	reportPath := filepath.Join(dir, "report.json")
	if err = os.WriteFile(reportPath, data, 0600); err != nil {
		return err
	}
	data, err = call("report", "verify", "--input", reportPath, "--json")
	if err != nil {
		return err
	}
	var verified evidence.Verification
	if err = json.Unmarshal(data, &verified); err != nil || verified.Invalid != 0 || verified.Unsupported != 0 || verified.Valid == 0 {
		return fmt.Errorf("verification: %+v %v", verified, err)
	}
	anchors := 0
	for _, c := range doc.Claims {
		for _, id := range c.Evidence {
			if id.Kind != "native" || id.RunID == "" || anchors >= 10 {
				continue
			}
			link := lineage.Link(model.Signal{ID: id.ID, RunID: id.RunID})
			parts := strings.SplitN(link.URL, "#", 2)
			res, e := client.Get(endpoint + parts[0])
			if e != nil {
				return e
			}
			body, e := io.ReadAll(io.LimitReader(res.Body, 4<<20))
			_ = res.Body.Close()
			if e != nil || res.StatusCode != 200 || !bytes.Contains(body, []byte(`id="`+parts[1]+`"`)) {
				return fmt.Errorf("anchor failed: %s %v", link.URL, e)
			}
			anchors++
		}
	}
	r, err := store.OpenReader(ctx, db)
	if err != nil {
		return err
	}
	doc.Claims[0].Evidence[0].ID = "fabricated"
	negative, err := evidence.VerifyDocument(ctx, r, doc)
	_ = r.Close()
	if err != nil || negative.Invalid != 1 {
		return fmt.Errorf("fabricated-id verification: %+v %v", negative, err)
	}
	checks, err := lane(ctx, *laneDB)
	if err != nil {
		return err
	}
	accepted, err := acceptedRelations(ctx, *relationsResult)
	if err != nil {
		return err
	}
	repo, err := os.Getwd()
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(repo, db)
	if err != nil {
		return err
	}
	result := map[string]any{"synthetic_only": true, "real_serve": true, "database": filepath.ToSlash(relative), "runs_list_count": 40, "minority_violations": 4, "flag_truth_table": truth, "verification": verified, "fabricated_id_invalid": negative.Invalid, "resolved_native_anchors": anchors, "lane_reports": checks, "report_path": filepath.ToSlash(filepath.Join(filepath.Dir(relative), "report.json"))}
	result["accepted_relation_fixture_report"] = accepted
	body, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(*out, append(body, '\n'), 0600); err != nil {
		return err
	}
	fmt.Printf("40 synthetic runs, four violations, independent flag table; %d supported claims; %d anchors; %d lane cases; %s\n", verified.Valid, anchors, len(checks), *out)
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
