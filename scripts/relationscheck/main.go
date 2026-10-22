// Synthetic declared-relation acceptance through the real serving binary.
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
	"runtime"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/model"
)

func run() error {
	binary := flag.String("binary", "bin/signalscout.exe", "real serve binary")
	defaultPython := filepath.Join("adapters", "provenance-python", ".venv", "Scripts", "python.exe")
	if runtime.GOOS != "windows" {
		defaultPython = filepath.Join("adapters", "provenance-python", ".venv", "bin", "python")
	}
	python := flag.String("python", defaultPython, "pinned Python 3.12 interpreter")
	out := flag.String("out", ".cache/declared-relations/acceptance.json", "generated result outside frozen results")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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
	py, err := filepath.Abs(*python)
	if err != nil {
		return err
	}
	adapter, err := filepath.Abs(filepath.Join("adapters", "provenance-python"))
	if err != nil {
		return err
	}
	fixture := filepath.Join(adapter, "fixtures", "relations")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	address := listener.Addr().String()
	if err = listener.Close(); err != nil {
		return err
	}
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
	for n := 0; n < 100; n++ {
		res, e := client.Get(endpoint + "/")
		if e == nil {
			_ = res.Body.Close()
			ready = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !ready {
		return fmt.Errorf("real server did not become ready; see %s", dir)
	}
	raw, err := os.ReadFile(filepath.Join(fixture, "signalscout.yaml"))
	if err != nil {
		return err
	}
	config := filepath.Join(dir, "signalscout.yaml")
	if err = os.WriteFile(config, []byte(strings.ReplaceAll(string(raw), "http://127.0.0.1:4318", endpoint)), 0600); err != nil {
		return err
	}
	from := time.Now().Add(-time.Second).UTC().Format(time.RFC3339Nano)
	probe := exec.CommandContext(ctx, py, "-m", "scout_probe", "--config", config, "--doctor", filepath.Join(dir, "doctor.json"), "--manifest", "--drain-timeout", "30", "--path", fixture, "relation_target:main", "--", "--cases", "40")
	probe.Dir = adapter
	output, err := probe.CombinedOutput()
	if e := os.WriteFile(filepath.Join(dir, "probe.txt"), output, 0600); e != nil {
		return e
	}
	if err != nil {
		return fmt.Errorf("probe failed: %w: %s", err, output)
	}
	to := time.Now().Add(time.Second).UTC().Format(time.RFC3339Nano)
	call := func(args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, bin, append([]string{"--db", db}, args...)...)
		return cmd.Output()
	}
	raw, err = call("runs", "list", "--workload", "relation_target.run_case")
	if err != nil {
		return err
	}
	var runs []model.Run
	if err = json.Unmarshal(raw, &runs); err != nil {
		return err
	}
	if len(runs) != 40 {
		return fmt.Errorf("runs list got %d, expected 40", len(runs))
	}
	for _, r := range runs {
		if r.Outcome != "pass" {
			return fmt.Errorf("synthetic operational outcome changed: %s", r.Outcome)
		}
	}
	profile := filepath.Join("testdata", "relations", "profile.json")
	bounds := []string{"--profile", profile, "--from", from, "--to", to, "--limit", "40"}
	raw, err = call(append([]string{"relations", "--json"}, bounds...)...)
	if err != nil {
		return err
	}
	var report model.RelationReport
	if err = json.Unmarshal(raw, &report); err != nil {
		return err
	}
	expected := map[string][4]int{"flag-equality": {40, 0, 0, 0}, "absolute-difference": {36, 4, 0, 0}, "bounds": {36, 4, 0, 0}, "ordering": {40, 0, 0, 0}, "literal-dotted-key": {40, 0, 0, 0}, "conditional": {34, 0, 0, 6}, "capture-off": {0, 0, 40, 0}}
	for id, want := range expected {
		got := report.Totals[id]
		if got == nil || got.Total != 40 || [4]int{got.Satisfied, got.Violated, got.Unknown, got.NotApplicable} != want {
			return fmt.Errorf("%s counts mismatch: %+v want %v", id, got, want)
		}
	}
	raw, err = call("runs", "list", "--workload", "range_target.admit")
	if err != nil {
		return err
	}
	var rangeRuns []model.Run
	if err = json.Unmarshal(raw, &rangeRuns); err != nil {
		return err
	}
	if len(rangeRuns) != 16 {
		return fmt.Errorf("admission fixture has %d runs, expected 16", len(rangeRuns))
	}
	rangeBounds := append([]string{}, bounds...)
	rangeBounds[1] = filepath.Join("testdata", "relations", "bounds-profile.json")
	raw, err = call(append([]string{"relations", "--json"}, rangeBounds...)...)
	if err != nil {
		return err
	}
	var rangeReport model.RelationReport
	if err = json.Unmarshal(raw, &rangeReport); err != nil {
		return err
	}
	for id, want := range map[string][4]int{"admission": {15, 1, 0, 0}, "ordered-bounds": {14, 2, 0, 0}, "clipped": {14, 0, 0, 2}, "object-sequence": {16, 0, 0, 0}} {
		got := rangeReport.Totals[id]
		if got == nil || got.Total != 16 || [4]int{got.Satisfied, got.Violated, got.Unknown, got.NotApplicable} != want {
			return fmt.Errorf("admission fixture %s mismatch: %+v expected %v", id, got, want)
		}
	}
	raw, err = call(append([]string{"cohort", "create", "minority", "--endpoint", endpoint, "--violates", "absolute-difference"}, bounds...)...)
	if err != nil {
		return err
	}
	var c model.Cohort
	if err = json.Unmarshal(raw, &c); err != nil {
		return err
	}
	if len(c.RunIDs) != 4 {
		return fmt.Errorf("violation cohort got %d, expected 4", len(c.RunIDs))
	}
	violating := map[string]bool{}
	for _, id := range c.RunIDs {
		violating[id] = true
	}
	truth := map[string]int{}
	for _, value := range []string{"true", "false"} {
		data, e := call(append([]string{"cohort", "create", "flag-" + value, "--endpoint", endpoint, "--predicate", "inputs.flag", "--equals", value}, bounds...)...)
		if e != nil {
			return e
		}
		var flags model.Cohort
		if e = json.Unmarshal(data, &flags); e != nil {
			return e
		}
		if len(flags.RunIDs) != 20 {
			return fmt.Errorf("flag %s cohort has %d runs", value, len(flags.RunIDs))
		}
		for _, id := range flags.RunIDs {
			key := "satisfied_flag_" + value
			if violating[id] {
				key = "violated_flag_" + value
			}
			truth[key]++
		}
		if truth["violated_flag_"+value] != 2 || truth["satisfied_flag_"+value] != 18 {
			return fmt.Errorf("stored truth table mismatch: %v", truth)
		}
	}
	raw, err = call("relations", "--cohort", "minority", "--json")
	if err != nil {
		return err
	}
	var pinned model.RelationReport
	if err = json.Unmarshal(raw, &pinned); err != nil {
		return err
	}
	a, _ := json.Marshal(report)
	b, _ := json.Marshal(pinned)
	if !bytes.Equal(a, b) {
		return fmt.Errorf("pinned counts differ from live checks")
	}
	anchors := 0
	for _, checkedReport := range []model.RelationReport{report, rangeReport} {
		for _, rows := range checkedReport.Runs {
			for _, count := range rows {
				for _, link := range count.ViolationURLs {
					parts := strings.SplitN(link, "#", 2)
					res, e := client.Get(endpoint + parts[0])
					if e != nil {
						return e
					}
					body, e := io.ReadAll(io.LimitReader(res.Body, 4<<20))
					_ = res.Body.Close()
					if e != nil {
						return e
					}
					if res.StatusCode != 200 || !bytes.Contains(body, []byte(`id="`+parts[1]+`"`)) {
						return fmt.Errorf("unresolved evidence %s", link)
					}
					anchors++
				}
			}
		}
	}
	repo, err := os.Getwd()
	if err != nil {
		return err
	}
	database, err := filepath.Rel(repo, db)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(map[string]any{"synthetic": true, "database": filepath.ToSlash(database), "real_serve": true, "runs_list_count": len(runs), "admission_fixture_runs": len(rangeRuns), "admission_report": rangeReport, "violation_cohort_runs": len(c.RunIDs), "evidence_anchors_resolved": anchors, "source_lines_changed_for_probes": 0, "flag_truth_table": truth, "truth_table_source": "Stored violating cohort crossed with the two stored flag cohorts, checked against independent Python fixture expectations", "report": report}, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(*out, append(data, '\n'), 0600); err != nil {
		return err
	}
	fmt.Printf("40 arithmetic + 16 admission runs; 4 arithmetic violating runs; %d real HTTP evidence anchors; report %s\n", anchors, *out)
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
