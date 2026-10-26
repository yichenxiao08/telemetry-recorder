// Command rewardaudit records a fixed public Snake evaluator and composes native
// declared-relation reports. It neither trains nor infers a reward mechanism.
package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/provenance"
	"github.com/signalscout/signalscout/internal/store"
)

//go:embed capture.py
var captureSource []byte

//go:embed profile.json
var profileSource []byte

const frozenSpec = "edde69f07acc176e2976017b8c4097e40813830f80c678f191106d28e153146f"

type specification struct {
	Version       int      `json:"version"`
	Conditions    []string `json:"conditions"`
	TrainingSeeds []int    `json:"training_seeds"`
	Evaluation    struct {
		HeldoutSeeds []int `json:"heldout_seeds"`
	} `json:"evaluation"`
	Common struct {
		MaxEpisodeSteps int `json:"max_episode_steps"`
	} `json:"common"`
	Hashes map[string]string `json:"hashes"`
}
type policy struct {
	ID            string `json:"id"`
	Condition     string `json:"condition"`
	Seed          int    `json:"seed"`
	Checkpoint    string `json:"checkpoint"`
	CheckpointSHA string `json:"checkpoint_sha256"`
	Result        string `json:"result"`
	ResultSHA     string `json:"result_sha256"`
}
type workload struct {
	Project      string  `json:"project_root"`
	Policy       policy  `json:"policy"`
	Seeds        []int   `json:"heldout_seeds"`
	PaceMS       float64 `json:"pace_ms"`
	ParityOutput string  `json:"parity_output"`
}
type row struct {
	Policy         policy                           `json:"policy"`
	Database       string                           `json:"database"`
	From           string                           `json:"from"`
	To             string                           `json:"to"`
	Episodes       int                              `json:"complete_episodes"`
	CaptureSeconds float64                          `json:"capture_seconds"`
	ParitySHA      string                           `json:"parity_sha256"`
	DoctorSHA      string                           `json:"doctor_sha256"`
	RelationsSHA   string                           `json:"relations_sha256"`
	Counts         map[string]*model.RelationCounts `json:"native_step_counts"`
}
type summary struct {
	Version        int               `json:"version"`
	Scope          string            `json:"scope"`
	BinarySHA      string            `json:"binary_sha256"`
	Binary         string            `json:"binary"`
	SpecSHA        string            `json:"spec_sha256"`
	InputHashes    map[string]string `json:"input_sha256"`
	Profile        string            `json:"profile"`
	PaceMS         float64           `json:"pace_ms"`
	QueueItems     int               `json:"queue_items"`
	QueueBytes     int               `json:"queue_bytes"`
	Rows           []row             `json:"policies"`
	ElapsedSeconds float64           `json:"elapsed_seconds"`
}

type options struct {
	Project, Python, Binary, Adapter, Out string
	PaceMS                                float64
}

func sha(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		return "", e
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func readJSON(path string, dst any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	if len(b) > 4<<20 {
		return fmt.Errorf("JSON input exceeds 4 MiB: %s", path)
	}
	return json.Unmarshal(b, dst)
}
func writeNew(path string, b []byte) error {
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	_, e = f.Write(b)
	c := f.Close()
	if e != nil {
		return e
	}
	return c
}
func save(path string, value any) error {
	b, e := json.MarshalIndent(value, "", "  ")
	if e != nil {
		return e
	}
	return writeNew(path, append(b, '\n'))
}
func within(root, relative string) (string, error) {
	relative = strings.ReplaceAll(relative, "\\", "/")
	p := filepath.Join(root, filepath.FromSlash(relative))
	rel, e := filepath.Rel(root, p)
	if e != nil || filepath.IsAbs(relative) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("input path outside project: %s", relative)
	}
	return p, nil
}
func verify(path, expected string) error {
	actual, e := sha(path)
	if e != nil {
		return e
	}
	if actual != strings.ToLower(expected) {
		return fmt.Errorf("hash differs: %s", path)
	}
	return nil
}
func validateSpec(s specification) error {
	if s.Version != 2 || s.Common.MaxEpisodeSteps != 256 || len(s.Conditions) != 2 || s.Conditions[0] != "original" || s.Conditions[1] != "variant" || len(s.TrainingSeeds) != 3 || len(s.Evaluation.HeldoutSeeds) != 10 {
		return errors.New("only the fixed v2 six-policy/60-episode Snake contract is supported")
	}
	if fmt.Sprint(s.TrainingSeeds) != "[11 22 33]" || fmt.Sprint(s.Evaluation.HeldoutSeeds) != "[1001 1002 1003 1004 1005 1006 1007 1008 1009 1010]" || len(s.Hashes) == 0 {
		return errors.New("v2 seed/source contract differs")
	}
	return nil
}
func prepare(o options) (specification, []policy, map[string]string, error) {
	var s specification
	sp := filepath.Join(o.Project, "study", "v2", "spec-v2.json")
	if e := verify(sp, frozenSpec); e != nil {
		return s, nil, nil, e
	}
	if e := readJSON(sp, &s); e != nil {
		return s, nil, nil, e
	}
	if e := validateSpec(s); e != nil {
		return s, nil, nil, e
	}
	hashes := map[string]string{}
	for rel, want := range s.Hashes {
		p, e := within(o.Project, rel)
		if e != nil {
			return s, nil, nil, e
		}
		if e = verify(p, want); e != nil {
			return s, nil, nil, e
		}
		hashes[p] = want
	}
	var manifest map[string]string
	if e := readJSON(filepath.Join(o.Project, "study", "v2", "artifact-hashes-v2.json"), &manifest); e != nil {
		return s, nil, nil, e
	}
	normalized := map[string]string{}
	for k, v := range manifest {
		normalized[strings.ReplaceAll(k, "\\", "/")] = v
	}
	policies := []policy{}
	for _, condition := range s.Conditions {
		for _, seed := range s.TrainingSeeds {
			id := fmt.Sprintf("%s-%d", condition, seed)
			base := "study/v2/" + id + "/"
			p := policy{ID: id, Condition: condition, Seed: seed, Checkpoint: filepath.Join(o.Project, filepath.FromSlash(base+"checkpoint.pt")), Result: filepath.Join(o.Project, filepath.FromSlash(base+"result.json")), CheckpointSHA: normalized[base+"checkpoint.pt"]}
			for _, rel := range []string{base + "checkpoint.pt", base + "result.json"} {
				want := normalized[rel]
				if want == "" {
					return s, nil, nil, fmt.Errorf("input manifest missing %s", rel)
				}
				path, _ := within(o.Project, rel)
				if e := verify(path, want); e != nil {
					return s, nil, nil, e
				}
				hashes[path] = want
			}
			p.ResultSHA = normalized[base+"result.json"]
			policies = append(policies, p)
		}
	}
	hashes[sp] = frozenSpec
	return s, policies, hashes, nil
}

func runCommand(ctx context.Context, dir, prefix, executable string, env []string, args ...string) error {
	if e := save(prefix+".command.json", map[string]any{"executable": executable, "args": args, "working_directory": dir}); e != nil {
		return e
	}
	out, e := os.OpenFile(prefix+".stdout", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	defer func() { _ = out.Close() }()
	stderr, e := os.OpenFile(prefix+".stderr", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	defer func() { _ = stderr.Close() }()
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = dir
	cmd.Stdout = out
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), env...)
	cmd.WaitDelay = 2 * time.Second
	if e = cmd.Run(); e != nil {
		return fmt.Errorf("command failed; retained logs %s: %w", prefix, e)
	}
	return nil
}

type probeDoctor struct {
	Module   string `json:"module"`
	Function string `json:"function"`
	Status   string `json:"status"`
	Calls    int    `json:"calls"`
	Records  int    `json:"records_captured"`
}
type drainResult struct {
	Drained      bool `json:"drained"`
	Complete     bool `json:"complete"`
	Attempted    int  `json:"attempted"`
	Acknowledged int  `json:"acknowledged"`
	Failed       int  `json:"failed"`
	Dropped      int  `json:"dropped"`
	Unknown      int  `json:"unknown"`
}
type doctor struct {
	Drained  drainResult                `json:"drained"`
	Runs     int                        `json:"runs_created"`
	Records  int                        `json:"records_captured"`
	Counters map[string]json.RawMessage `json:"counters"`
	Probes   []probeDoctor              `json:"probes"`
}

func requiredCount(m map[string]json.RawMessage, key string) (int, error) {
	raw, ok := m[key]
	var n int
	if !ok {
		return 0, fmt.Errorf("missing capture counter %s", key)
	}
	if e := json.Unmarshal(raw, &n); e != nil || n < 0 {
		return 0, fmt.Errorf("invalid capture counter %s", key)
	}
	return n, nil
}
func checkDoctor(d doctor) (string, error) {
	if !d.Drained.Drained || !d.Drained.Complete || d.Drained.Failed != 0 || d.Drained.Dropped != 0 || d.Drained.Unknown != 0 || d.Drained.Attempted != d.Drained.Acknowledged || d.Runs != 10 || len(d.Probes) != 2 {
		return "", errors.New("probe scope was not ten complete episodes/two attached probes")
	}
	sum := 0
	seen := map[string]int{}
	for _, p := range d.Probes {
		if p.Module != "reward_capture" || p.Status != "attached" || p.Calls == 0 || p.Calls != p.Records {
			return "", errors.New("probe identity/calls not fully captured")
		}
		seen[p.Function]++
		sum += p.Records
		if p.Function == "episode_run" && p.Calls != 10 {
			return "", errors.New("episode probe population differs")
		}
	}
	if seen["episode_run"] != 1 || seen["step_observation"] != 1 || sum != d.Records {
		return "", errors.New("required probe identities/record count differ")
	}
	for _, key := range []string{"failed", "dropped", "unknown", "drop_bytes", "drop_full", "queue_size", "retained_items"} {
		n, e := requiredCount(d.Counters, key)
		if e != nil {
			return "", e
		}
		if n != 0 {
			return "", fmt.Errorf("capture loss/retained item: %s", key)
		}
	}
	ack, e := requiredCount(d.Counters, "acknowledged")
	if e != nil {
		return "", e
	}
	if ack < sum || ack != d.Drained.Acknowledged {
		return "", errors.New("acknowledgments below recorded decisions")
	}
	for _, key := range []string{"attempted", "accepted", "exported"} {
		n, err := requiredCount(d.Counters, key)
		if err != nil {
			return "", err
		}
		if n != ack {
			return "", errors.New("capture delivery counters disagree")
		}
	}
	var manifest map[string]json.RawMessage
	if e = json.Unmarshal(d.Counters["manifest"], &manifest); e != nil {
		return "", errors.New("missing scope manifest")
	}
	var epoch string
	var tail bool
	if e = json.Unmarshal(manifest["epoch"], &epoch); e != nil || epoch == "" {
		return "", errors.New("missing scope epoch")
	}
	if e = json.Unmarshal(manifest["scope_tail_sent"], &tail); e != nil || !tail {
		return "", errors.New("scope tail not sent")
	}
	for key, want := range map[string]int{"records_acked": 11, "records_failed": 0, "records_unknown": 0, "expected_runs": 10, "pending_runs": 0, "outbox": 0, "ledger_overflow": 0, "population_overflow": 0} {
		n, err := requiredCount(manifest, key)
		if err != nil {
			return "", err
		}
		if n != want {
			return "", fmt.Errorf("scope counter differs: %s", key)
		}
	}
	return epoch, nil
}
func checkCompleteness(c store.ScopeReport) error {
	n := c.Counts
	if !c.PopulationKnown || !c.ScopeFinal || c.ScopeExpected != 10 || c.ListedRuns != 10 || n.Complete != 10 || n.Partial != 0 || n.Unknown != 0 || n.Absent != 0 || n.Unassessed != 0 || n.LostItems != 0 || n.UnknownItems != 0 || n.DetailLossItems != 0 {
		return errors.New("persisted capture scope incomplete; no interpretation permitted")
	}
	return nil
}
func validCounts(c *model.RelationCounts) bool {
	return c != nil && c.Total > 0 && c.Total <= 100000 && c.Satisfied >= 0 && c.Violated >= 0 && c.NotApplicable >= 0 && c.Unknown == 0 && c.SampledRecords == 0 && c.Satisfied+c.Violated+c.NotApplicable == c.Total && len(c.ViolationEvidence) == len(c.ViolationURLs) && len(c.ViolationEvidence) <= 3 && len(c.ViolationEvidence) <= c.Violated && c.EvidenceOmitted == c.Violated-len(c.ViolationEvidence)
}
func checkRelations(r model.RelationReport) error {
	if r.CandidateRuns != 10 || len(r.Runs) != 10 || len(r.RunsWithoutDecision) != 0 {
		return errors.New("native relation population differs from ten episodes")
	}
	rules := []string{"apple-events", "alive-steps", "configuration-repeats", "positive-reward-on-repeat"}
	total := -1
	for _, id := range rules {
		c := r.Totals[id]
		if !validCounts(c) {
			return fmt.Errorf("native relation %s missing/inconsistent/unknown/sampled", id)
		}
		if total >= 0 && total != c.Total {
			return errors.New("native rule denominators differ")
		}
		total = c.Total
		sum := model.RelationCounts{}
		for _, counts := range r.Runs {
			run := counts[id]
			if !validCounts(run) {
				return fmt.Errorf("native per-run relation %s invalid", id)
			}
			sum.Total += run.Total
			sum.Satisfied += run.Satisfied
			sum.Violated += run.Violated
			sum.NotApplicable += run.NotApplicable
		}
		if sum.Total != c.Total || sum.Satisfied != c.Satisfied || sum.Violated != c.Violated || sum.NotApplicable != c.NotApplicable {
			return errors.New("native per-run totals disagree")
		}
	}
	return nil
}

func config(endpoint string) map[string]any {
	return map[string]any{"version": 1, "synthetic_only": true, "endpoint": endpoint, "service": "synthetic.snake.reward-audit", "probes": []any{
		map[string]any{"module": "reward_capture", "function": "episode_run", "capture": map[string]any{"args": []string{"policy_id", "training_seed", "evaluation_seed"}}},
		map[string]any{"module": "reward_capture", "function": "step_observation", "capture": map[string]any{"args": []string{"apple", "death", "repeated_configuration", "actual_reward"}}},
	}}
}

func capturePolicy(ctx context.Context, o options, s specification, p policy, root string) (row, error) {
	result := row{Policy: p, Database: filepath.Join(root, "capture.db"), Episodes: 10}
	fail := func(e error) (row, error) { return result, e }
	if e := os.Mkdir(root, 0700); e != nil {
		return fail(e)
	}
	wp := filepath.Join(root, "workload.json")
	if e := save(wp, workload{Project: o.Project, Policy: p, Seeds: s.Evaluation.HeldoutSeeds, PaceMS: o.PaceMS, ParityOutput: filepath.Join(root, "parity.json")}); e != nil {
		return fail(e)
	}
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		return fail(e)
	}
	address := l.Addr().String()
	if e = l.Close(); e != nil {
		return fail(e)
	}
	cfg := filepath.Join(root, "config.json")
	if e = save(cfg, config("http://"+address)); e != nil {
		return fail(e)
	}
	env := []string{"SIGNALSCOUT_REWARD_WORKLOAD=" + wp, "PYTHONDONTWRITEBYTECODE=1"}
	probeArgs := []string{"-B", "-m", "scout_probe", "--config", cfg, "--doctor", filepath.Join(root, "preflight.json"), "--path", o.Out, "--preflight", "reward_capture:main"}
	if e = runCommand(ctx, o.Adapter, filepath.Join(root, "preflight"), o.Python, env, probeArgs...); e != nil {
		return fail(e)
	}
	log, e := os.OpenFile(filepath.Join(root, "server.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return fail(e)
	}
	defer func() { _ = log.Close() }()
	if e = writeNew(result.Database, nil); e != nil {
		return fail(e)
	}
	server := exec.CommandContext(ctx, o.Binary, "--db", result.Database, "serve", "--listen", address, "--retention", "0")
	server.Stdout = log
	server.Stderr = log
	if e = server.Start(); e != nil {
		return fail(e)
	}
	exited := make(chan error, 1)
	go func() { exited <- server.Wait() }()
	defer func() {
		_ = server.Process.Kill()
		select {
		case <-exited:
		case <-time.After(5 * time.Second):
		}
	}()
	client := http.Client{Timeout: time.Second}
	ready := false
	for i := 0; i < 50; i++ {
		select {
		case err := <-exited:
			return fail(fmt.Errorf("own intake exited before capture: %v", err))
		default:
		}
		response, err := client.Get("http://" + address + "/runs")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == 200 {
				ready = true
				break
			}
		}
		select {
		case <-ctx.Done():
			return fail(ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
	if !ready {
		return fail(errors.New("own intake not ready; inspect server.log"))
	}
	result.From = time.Now().Add(-time.Second).UTC().Format(time.RFC3339Nano)
	started := time.Now()
	probeArgs = []string{"-B", "-m", "scout_probe", "--config", cfg, "--doctor", filepath.Join(root, "doctor.json"), "--path", o.Out, "--manifest", "--drain-timeout", "30", "--queue-max-items", "1024", "--queue-max-bytes", "8388608", "reward_capture:main"}
	if e = runCommand(ctx, o.Adapter, filepath.Join(root, "capture"), o.Python, env, probeArgs...); e != nil {
		return fail(e)
	}
	result.CaptureSeconds = time.Since(started).Seconds()
	result.To = time.Now().Add(time.Second).UTC().Format(time.RFC3339Nano)
	var d doctor
	if e = readJSON(filepath.Join(root, "doctor.json"), &d); e != nil {
		return fail(e)
	}
	epoch, e := checkDoctor(d)
	if e != nil {
		return fail(e)
	}
	if e = runCommand(ctx, o.Out, filepath.Join(root, "completeness"), o.Binary, nil, "--db", result.Database, "cohort", "completeness", "--scope", epoch, "--runs"); e != nil {
		return fail(e)
	}
	var c store.ScopeReport
	if e = readJSON(filepath.Join(root, "completeness.stdout"), &c); e != nil {
		return fail(e)
	}
	if e = checkCompleteness(c); e != nil {
		return fail(e)
	}
	var parity struct {
		Matched  bool              `json:"all_exact_matches"`
		Episodes []json.RawMessage `json:"episodes"`
	}
	if e = readJSON(filepath.Join(root, "parity.json"), &parity); e != nil {
		return fail(e)
	}
	if !parity.Matched || len(parity.Episodes) != 10 {
		return fail(errors.New("frozen evaluation parity not established"))
	}
	if e = runCommand(ctx, o.Out, filepath.Join(root, "relations"), o.Binary, nil, "--db", result.Database, "relations", "--profile", filepath.Join(o.Out, "profile.json"), "--from", result.From, "--to", result.To, "--limit", "10", "--record-limit", "100000", "--json"); e != nil {
		return fail(e)
	}
	var relations model.RelationReport
	if e = readJSON(filepath.Join(root, "relations.stdout"), &relations); e != nil {
		return fail(e)
	}
	if e = checkRelations(relations); e != nil {
		return fail(e)
	}
	result.Counts = relations.Totals
	for name, dst := range map[string]*string{"doctor.json": &result.DoctorSHA, "parity.json": &result.ParitySHA, "relations.stdout": &result.RelationsSHA} {
		*dst, e = sha(filepath.Join(root, name))
		if e != nil {
			return fail(e)
		}
	}
	return result, nil
}

func markdown(s summary) string {
	var b strings.Builder
	fmt.Fprintln(&b, "# Declared Snake reward audit\n\nObservation only: supplied scalar measurements, native relation counts. No inferred cause or usefulness claim.")
	fmt.Fprintln(&b, "| Policy | Complete episodes | Recorded steps | Apple events | Alive steps | Repeated configurations | Positive actual reward on repeated nonprogress | Not applicable | Unknown |\n|---|---:|---:|---:|---:|---:|---:|---:|---:|")
	for _, r := range s.Rows {
		c := r.Counts
		positive := c["positive-reward-on-repeat"]
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %d | %d | %d | %d |\n", r.Policy.ID, r.Episodes, positive.Total, c["apple-events"].Violated, c["alive-steps"].Satisfied, c["configuration-repeats"].Violated, positive.Violated, positive.NotApplicable, positive.Unknown)
	}
	fmt.Fprintf(&b, "\nPacing: %.3f ms per recorded transition; bounded queue: %d items / %d estimated bytes. Pacing preserves this deterministic evaluation's state outcomes, not timing. Per-policy capture durations and hashes are in audit.json. Counts are transitions, not independent training samples. Three training-seed pairs; ten episodes each.\n", s.PaceMS, s.QueueItems, s.QueueBytes)
	for _, r := range s.Rows {
		fmt.Fprintf(&b, "\n## %s: inspect saved data\n\nDatabase: `%s`\n\nNative relation file: `%s` (SHA-256 `%s`). Each policy has its own DB; viewer witness URLs below require serving THIS database. No root-relative URL is globally unique.\n", r.Policy.ID, r.Database, filepath.Join(filepath.Dir(r.Database), "relations.stdout"), r.RelationsSHA)
		counts := r.Counts["positive-reward-on-repeat"]
		for i, id := range counts.ViolationEvidence {
			fmt.Fprintf(&b, "\n- `%s` — `%s`\n", id, counts.ViolationURLs[i])
		}
		fmt.Fprintf(&b, "\nAdditional violation witnesses omitted by native engine: %d. Full per-run native counts are saved.\n", counts.EvidenceOmitted)
		fmt.Fprintf(&b, "\n```powershell\n& '%s' --db '%s' relations --profile '%s' --from '%s' --to '%s' --limit 10 --record-limit 100000 --json\n```\n", strings.ReplaceAll(s.Binary, "'", "''"), strings.ReplaceAll(r.Database, "'", "''"), strings.ReplaceAll(s.Profile, "'", "''"), r.From, r.To)
	}
	fmt.Fprintln(&b, "\nRepeat/reward/event meanings are declared in the example launcher, not inferred. This is legacy synthetic capture, not a production privacy gate. No structured-report cap is raised: this audit consists of existing native relations reports, not a verified structured causal report.")
	return b.String()
}

func execute(ctx context.Context, o options) error {
	started := time.Now()
	if math.IsNaN(o.PaceMS) || math.IsInf(o.PaceMS, 0) || o.PaceMS < 0 || o.PaceMS > 100 {
		return errors.New("pace-ms must be finite in 0..100; zero does not guarantee delivery")
	}
	for _, p := range []*string{&o.Project, &o.Python, &o.Binary, &o.Adapter, &o.Out} {
		v, e := filepath.Abs(*p)
		if e != nil {
			return e
		}
		*p = v
	}
	s, policies, hashes, e := prepare(o)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(o.Out), 0700); e != nil {
		return e
	}
	if e = os.Mkdir(o.Out, 0700); e != nil {
		return fmt.Errorf("fresh output required: %w", e)
	}
	if e = writeNew(filepath.Join(o.Out, "reward_capture.py"), captureSource); e != nil {
		return e
	}
	if e = writeNew(filepath.Join(o.Out, "profile.json"), profileSource); e != nil {
		return e
	}
	var profile model.AnalysisProfile
	if e = json.Unmarshal(profileSource, &profile); e != nil {
		return e
	}
	if e = provenance.ValidateProfile(profile); e != nil {
		return e
	}
	var followup model.AnalysisProfile
	if e = json.Unmarshal(profileSource, &followup); e != nil {
		return e
	}
	followup.ID = "snake-all-positive-reward-v1"
	followup.Relations.Rules[3].ID = "positive-reward-all-steps"
	followup.Relations.Rules[3].When = nil
	if e = save(filepath.Join(o.Out, "profile-all-steps.json"), followup); e != nil {
		return e
	}

	output := summary{Version: 1, Scope: "synthetic declared observations; not cause discovery", SpecSHA: frozenSpec, InputHashes: hashes, Binary: o.Binary, Profile: filepath.Join(o.Out, "profile.json"), PaceMS: o.PaceMS, QueueItems: 1024, QueueBytes: 8388608, Rows: []row{}}
	if output.BinarySHA, e = sha(o.Binary); e != nil {
		return e
	}
	if e = save(filepath.Join(o.Out, "inputs.json"), output); e != nil {
		return e
	}
	for _, p := range policies {
		fmt.Printf("Recording %s (10 held-out episodes)...\n", p.ID)
		r, err := capturePolicy(ctx, o, s, p, filepath.Join(o.Out, p.ID))
		if err != nil {
			_ = save(filepath.Join(o.Out, "failure.json"), map[string]any{"policy": p.ID, "error": err.Error(), "completed_policies": len(output.Rows), "elapsed_seconds": time.Since(started).Seconds()})
			return err
		}
		output.Rows = append(output.Rows, r)
	}
	for path, want := range hashes {
		if e = verify(path, want); e != nil {
			return e
		}
	}
	output.ElapsedSeconds = time.Since(started).Seconds()
	if e = save(filepath.Join(o.Out, "audit.json"), output); e != nil {
		return e
	}
	// Store executable explicitly in the saved command examples, without shell evaluation.
	text := markdown(output)
	if e = writeNew(filepath.Join(o.Out, "audit.md"), []byte(text)); e != nil {
		return e
	}
	fmt.Printf("Complete: 6 policies / 3 training-seed pairs / 60 episodes; inspect %s\n", filepath.Join(o.Out, "audit.md"))
	return nil
}
func main() {
	var o options
	flag.StringVar(&o.Project, "project", "", "prepared public rlenv study root (trusted Python code)")
	flag.StringVar(&o.Python, "python", "", "prepared CPython 3.12 with frozen adapter + Torch dependencies")
	flag.StringVar(&o.Binary, "binary", "bin/signalscout.exe", "built Signal Scout binary")
	flag.StringVar(&o.Adapter, "adapter", "adapters/provenance-python", "adapter sources matching binary")
	flag.StringVar(&o.Out, "out", filepath.Join(".cache", "reward-audit", time.Now().UTC().Format("20060102T150405.000000000")), "NEW output directory; never reused")
	flag.Float64Var(&o.PaceMS, "pace-ms", 12, "explicit pacing per transition, 0..100ms; not transparent capture")
	flag.Parse()
	if o.Project == "" || o.Python == "" {
		fmt.Fprintln(os.Stderr, "--project and --python required")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	if e := execute(ctx, o); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
