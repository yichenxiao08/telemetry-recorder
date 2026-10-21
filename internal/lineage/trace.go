// Package lineage describes recorded single-output boundaries. It makes no causal inference.
package lineage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"strings"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/provenance"
)

const Tier = "Observations from recorded boundary capture and explicit lineage assertions; boundary capture does not prove the code read a value."

func ValidateInvariantProfile(p model.AnalysisProfile) error { return provenance.ValidateProfile(p) }

// NamedTarget accepts a run-local ref suffix, an explicit output name, or an
// unambiguous step identity. Repeated calls need their exact output ref.
func NamedTarget(signals []model.Signal, run, name string) string {
	exact := "value:" + run + "/" + name
	var matches []string
	for _, s := range signals {
		if s.RunID != run || s.Attrs["signalscout.record.kind"] != "decision" {
			continue
		}
		body, _ := s.Attrs["signalscout.record.json"].(string)
		var e Envelope
		if decode(body, &e) != nil {
			continue
		}
		if e.Output.Ref == exact {
			return exact
		}
		if e.Output.Name == name || e.Step.ID == name {
			matches = append(matches, e.Output.Ref)
		}
	}
	if len(matches) == 1 {
		return matches[0]
	}
	return "unresolved-name:" + run + "/" + name
}

func SideBySide(a, b Report) string {
	left, right := map[string]string{}, map[string]string{}
	var names []string
	seen := map[string]bool{}
	for i, r := range []Report{a, b} {
		m := left
		if i == 1 {
			m = right
		}
		if len(r.Nodes) == 0 {
			continue
		}
		n := r.Nodes[0]
		m["step"] = fmt.Sprintf("%q", n.Step.ID)
		m["output"] = display(n.Output)
		m["chosen"] = raw(n.Chosen)
		m["alternatives"] = raw(n.Alternatives)
		m["rule"] = raw(n.Rule)
		for _, key := range []string{"step", "output", "chosen", "alternatives", "rule"} {
			if !seen[key] {
				names = append(names, key)
				seen[key] = true
			}
		}
		for _, f := range n.Inputs {
			key := "input " + f.Name
			m[key] = display(f)
			if !seen[key] {
				names = append(names, key)
				seen[key] = true
			}
		}
	}
	var out strings.Builder
	fmt.Fprintf(&out, "field | left %q | right %q\n", a.Target, b.Target)
	for _, name := range names {
		l, r := left[name], right[name]
		if l == "" {
			l = "unknown"
		}
		if r == "" {
			r = "unknown"
		}
		fmt.Fprintf(&out, "%q | %s | %s\n", name, l, r)
	}
	return out.String()
}

type Options struct {
	Depth int `json:"depth"`
	Nodes int `json:"nodes"`
}
type Field struct {
	Name    string          `json:"name,omitempty"`
	Ref     string          `json:"ref,omitempty"`
	Capture string          `json:"capture,omitempty"`
	Inline  json.RawMessage `json:"inline,omitempty"`
	Digest  string          `json:"digest,omitempty"`
	Status  string          `json:"status,omitempty"`
}
type Step struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
	SpanRef string `json:"span_ref,omitempty"`
}
type Envelope struct {
	Version int    `json:"version"`
	ID      string `json:"id"`
	RunID   string `json:"run_id"`
	Profile struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	} `json:"profile"`
	Step         Step            `json:"step"`
	Inputs       []Field         `json:"inputs"`
	Output       Field           `json:"output"`
	Chosen       json.RawMessage `json:"chosen,omitempty"`
	Alternatives json.RawMessage `json:"alternatives,omitempty"`
	Rule         json.RawMessage `json:"rule,omitempty"`
	ConfigRefs   []string        `json:"config_refs,omitempty"`
}
type Edge struct {
	Version     int    `json:"version"`
	ID          string `json:"id"`
	RunID       string `json:"run_id"`
	InputRef    string `json:"input_ref"`
	OutputRef   string `json:"output_ref"`
	DecisionRef string `json:"decision_ref"`
	StepID      string `json:"step_id"`
	Relation    string `json:"relation"`
}
type Evidence struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}
type Finding struct {
	Kind   string `json:"kind"`
	Ref    string `json:"ref"`
	Detail string `json:"detail"`
}
type Invariant struct {
	Status   string `json:"status"`
	Observed int    `json:"observed"`
	Required int    `json:"required"`
}
type Node struct {
	Envelope
	Evidence  Evidence   `json:"evidence"`
	Edges     []Evidence `json:"edge_evidence"`
	Invariant Invariant  `json:"invariant"`
	Status    string     `json:"status"`
	Depth     int        `json:"depth"`
}
type Report struct {
	Target       string    `json:"target"`
	EvidenceTier string    `json:"evidence_tier"`
	Nodes        []Node    `json:"nodes"`
	Findings     []Finding `json:"findings"`
	Visited      int       `json:"visited"`
	Budget       Options   `json:"budget"`
	Truncations  int       `json:"truncations"`
}
type record struct {
	signal model.Signal
	env    Envelope
	edge   Edge
	kind   string
	body   string
}

// RunRef extracts only the documented run-qualified value/decision/lineage refs.
// Opaque refs require an explicit run partition; they never trigger history scans.
func RunRef(ref string) string {
	for _, prefix := range []string{"value:", "decision:", "lineage:"} {
		if strings.HasPrefix(ref, prefix) {
			run, _, ok := strings.Cut(strings.TrimPrefix(ref, prefix), "/")
			if ok {
				v, err := url.PathUnescape(run)
				if err == nil {
					return v
				}
			}
		}
	}
	return ""
}
func Link(s model.Signal) Evidence {
	sum := sha256.Sum256([]byte(s.ID))
	return Evidence{ID: s.ID, URL: "/run?id=" + url.QueryEscape(s.RunID) + "#e-" + hex.EncodeToString(sum[:])}
}

// decode rejects duplicate JSON keys as well as excessive nesting/size. Last-key-wins
// decoding must not silently change capture policy or identity.
func decode(body string, dst any) error {
	if len(body) > 8192 {
		return fmt.Errorf("envelope exceeds 8192 bytes")
	}
	d := json.NewDecoder(strings.NewReader(body))
	if err := tokens(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("extra JSON content")
	}
	return json.Unmarshal([]byte(body), dst)
}
func tokens(d *json.Decoder, depth int) error {
	if depth > 16 {
		return fmt.Errorf("excessive JSON depth")
	}
	v, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := v.(json.Delim)
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	for d.More() {
		if delim == '{' {
			key, err := d.Token()
			if err != nil {
				return err
			}
			k, ok := key.(string)
			if !ok || seen[k] {
				return fmt.Errorf("duplicate or invalid JSON key")
			}
			seen[k] = true
		}
		if err := tokens(d, depth+1); err != nil {
			return err
		}
	}
	_, err = d.Token()
	return err
}
func capture(f Field) Field {
	switch {
	case f.Capture == "allowed" && len(f.Inline) > 0:
		f.Status = "captured"
	case f.Capture == "off":
		f.Status = "capture-off"
		f.Inline = nil
		f.Digest = ""
	case (f.Capture == "reference-only" || f.Capture == "hash-only") && f.Digest != "":
		f.Status = "hash-only"
		f.Inline = nil
	default:
		f.Status = "missing"
		f.Inline = nil
	}
	return f
}
func invariant(e Envelope, p *model.AnalysisProfile) Invariant {
	out := Invariant{Status: "not_applicable"}
	if p == nil || p.Invariant.Declared == "" {
		return out
	}
	out = Invariant{Status: "unknown", Required: 3}
	if p.Step != e.Step.ID || p.EnvelopeProfile != "" && (p.EnvelopeProfile != e.Profile.ID || p.EnvelopeVersion != e.Profile.Version) {
		return out
	}
	inputs := map[string]any{}
	values := map[string]any{"inputs": inputs}
	for _, f := range e.Inputs {
		if f.Status == "captured" {
			var v any
			if json.Unmarshal(f.Inline, &v) == nil {
				inputs[f.Name] = v
			}
		}
	}
	if e.Output.Status == "captured" {
		var v any
		if json.Unmarshal(e.Output.Inline, &v) == nil {
			values["output"] = v
		}
	}
	paths := []string{p.Invariant.Declared, p.Invariant.Current, p.Invariant.Used}
	v := make([]any, 3)
	for i, path := range paths {
		var ok bool
		v[i], ok = provenance.Get(values, path)
		if ok && v[i] != nil {
			out.Observed++
		}
	}
	declared, ok := v[0].(bool)
	if !ok {
		return out
	}
	if !declared {
		out.Status = "not_applicable"
		return out
	}
	current, cok := v[1].(float64)
	used, uok := v[2].(float64)
	if !cok || !uok || math.IsInf(current, 0) || math.IsInf(used, 0) || math.Trunc(current) != current || math.Trunc(used) != used || math.Abs(current) > 9007199254740991 || math.Abs(used) > 9007199254740991 || p.Invariant.Delta <= 0 {
		return out
	}
	out.Status = "violated"
	if current-used == p.Invariant.Delta {
		out.Status = "satisfied"
	}
	return out
}

// Walk traverses stored producers only. Missing producers remain findings, never
// invented nodes. A shared producer is visited once; recursion has two hard bounds.
func Walk(signals []model.Signal, target string, opt Options, p *model.AnalysisProfile) Report {
	r := Report{Target: target, EvidenceTier: Tier, Budget: opt, Nodes: []Node{}, Findings: []Finding{}}
	add := func(kind, ref, detail string) { r.Findings = append(r.Findings, Finding{kind, ref, detail}) }
	if opt.Nodes < 1 || opt.Nodes > 1000 || opt.Depth < 0 || opt.Depth > 64 {
		add("invalid", target, "unknown: invalid traversal budget")
		return r
	}
	records := map[string]record{}
	bad := map[string]bool{}
	for _, s := range signals {
		kind, _ := s.Attrs["signalscout.record.kind"].(string)
		if kind != "decision" && kind != "lineage" {
			continue
		}
		body, _ := s.Attrs["signalscout.record.json"].(string)
		rec := record{signal: s, kind: kind, body: body}
		var err error
		var id, run string
		var version int
		if kind == "decision" {
			err = decode(body, &rec.env)
			id, run, version = rec.env.ID, rec.env.RunID, rec.env.Version
		} else {
			err = decode(body, &rec.edge)
			id, run, version = rec.edge.ID, rec.edge.RunID, rec.edge.Version
		}
		attrID, _ := s.Attrs["signalscout.record.id"].(string)
		if err != nil || id == "" || run != s.RunID || version != 1 || attrID != "" && attrID != id {
			add("malformed", s.ID, "unknown: invalid envelope, version or identity")
			if id != "" {
				bad[id] = true
			}
			continue
		}
		if prior, ok := records[id]; ok {
			kind := "duplicate"
			if prior.body != body {
				kind = "conflicting"
			}
			bad[id] = true
			add(kind, id, "unknown: logical ID is not unique")
			continue
		}
		records[id] = rec
	}
	outputs := map[string][]string{}
	native := map[string][]string{}
	edges := map[string][]record{}
	// Iterate the original order for deterministic reports; maps are lookup-only.
	seen := map[string]bool{}
	for _, s := range signals {
		id, _ := s.Attrs["signalscout.record.id"].(string)
		if id == "" {
			body, _ := s.Attrs["signalscout.record.json"].(string)
			var x struct{ ID string }
			_ = json.Unmarshal([]byte(body), &x)
			id = x.ID
		}
		rec, ok := records[id]
		if !ok || seen[id] {
			continue
		}
		seen[id] = true
		if rec.kind == "lineage" {
			if rec.edge.InputRef == "" || rec.edge.OutputRef == "" || rec.edge.DecisionRef == "" || rec.edge.StepID == "" || rec.edge.Relation != "derived_from" {
				bad[id] = true
				add("malformed", id, "unknown: incomplete or unsupported lineage edge")
			}
			if !bad[id] {
				edges[rec.edge.OutputRef] = append(edges[rec.edge.OutputRef], rec)
			}
			continue
		}
		e := rec.env
		names := map[string]bool{}
		valid := e.Step.ID != "" && e.Output.Ref != "" && len(e.Inputs) <= 32 && len(e.ConfigRefs) <= 32
		for _, f := range e.Inputs {
			if f.Name == "" || names[f.Name] {
				valid = false
			}
			names[f.Name] = true
		}
		for _, f := range append(append([]Field(nil), e.Inputs...), e.Output) {
			if f.Capture != "allowed" && f.Capture != "off" && f.Capture != "reference-only" && f.Capture != "hash-only" && f.Capture != "" {
				valid = false
			}
		}
		if !valid {
			bad[id] = true
			add("malformed", id, "unknown: invalid step, output or input fields")
		}
		outputs[e.Output.Ref] = append(outputs[e.Output.Ref], id)
		native[rec.signal.ID] = append(native[rec.signal.ID], id)
		native[e.Step.SpanRef] = append(native[e.Step.SpanRef], id)
	}
	for _, s := range signals {
		id, _ := s.Attrs["signalscout.record.id"].(string)
		rec, ok := records[id]
		if ok && rec.kind == "decision" && len(outputs[rec.env.Output.Ref]) > 1 && !bad[id] {
			bad[id] = true
			add("conflicting", rec.env.Output.Ref, "unknown: multiple decisions claim this output")
		}
	}
	var roots []string
	if _, ok := records[target]; ok && records[target].kind == "decision" {
		roots = []string{target}
	} else if ids := outputs[target]; len(ids) > 0 {
		roots = ids
	} else {
		roots = native[target]
	}
	state := map[string]int{}
	var visit func(string, int)
	visit = func(id string, depth int) {
		rec, ok := records[id]
		if !ok || bad[id] {
			add("missing", id, "unknown: absent or non-unique decision")
			return
		}
		if state[id] == 1 {
			add("cycle", id, "unknown: backward lineage cycle")
			return
		}
		if state[id] == 2 {
			return
		}
		if depth > opt.Depth {
			r.Truncations++
			add("depth", id, "unknown: depth budget reached")
			return
		}
		if len(r.Nodes) >= opt.Nodes {
			r.Truncations++
			add("node budget", id, "unknown: node budget reached")
			return
		}
		e := rec.env
		e.Output = capture(e.Output)
		e.Inputs = append([]Field(nil), e.Inputs...)
		for i := range e.Inputs {
			e.Inputs[i] = capture(e.Inputs[i])
		}
		n := Node{Envelope: e, Evidence: Link(rec.signal), Status: e.Output.Status, Depth: depth, Invariant: invariant(e, p), Edges: []Evidence{}}
		idx := len(r.Nodes)
		r.Nodes = append(r.Nodes, n)
		state[id] = 1
		for _, er := range edges[e.Output.Ref] {
			edge := er.edge
			if edge.DecisionRef != id {
				continue
			}
			found := false
			for _, f := range e.Inputs {
				if f.Ref == edge.InputRef && f.Ref != "" {
					found = true
				}
			}
			if !found || edge.StepID != e.Step.ID || edge.Relation != "derived_from" {
				add("malformed", edge.ID, "unknown: lineage edge disagrees with decision or relation")
				continue
			}
			r.Nodes[idx].Edges = append(r.Nodes[idx].Edges, Link(er.signal))
			producers := outputs[edge.InputRef]
			if len(producers) == 0 {
				add("dangling", edge.InputRef, "unknown: no producer in search scope")
				continue
			}
			if len(producers) != 1 {
				add("conflicting", edge.InputRef, "unknown: multiple producers")
				continue
			}
			visit(producers[0], depth+1)
		}
		for _, f := range e.Inputs {
			found := false
			for _, er := range edges[e.Output.Ref] {
				if er.edge.DecisionRef == id && er.edge.InputRef == f.Ref {
					found = true
				}
			}
			if !found {
				add("boundary", f.Ref, "recorded as input; upstream detail unknown (no explicit lineage edge)")
			}
		}
		state[id] = 2
	}
	if len(roots) != 1 {
		add("missing", target, "unknown: target absent or ambiguous within search scope")
	} else {
		visit(roots[0], 0)
	}
	r.Visited = len(r.Nodes)
	return r
}

// Text escapes recorded strings and JSON control characters for terminal safety.
func Text(r Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Evidence tier: %s\nTarget: %q\n", r.EvidenceTier, r.Target)
	for _, n := range r.Nodes {
		fmt.Fprintf(&b, "Step %q version %q [%s] output %q\n", n.Step.ID, n.Step.Version, n.Status, n.Output.Ref)
		fmt.Fprintf(&b, "  output: %s\n", display(n.Output))
		for _, f := range n.Inputs {
			fmt.Fprintf(&b, "  %q supplied to this step; recorded as input: %s\n", f.Name, display(f))
		}
		fmt.Fprintf(&b, "  chosen: %s; alternatives (recorded scores): %s; rule/threshold: %s\n", raw(n.Chosen), raw(n.Alternatives), raw(n.Rule))
		fmt.Fprintf(&b, "  config refs: %q; invariant: %s (coverage %d/%d)\n", n.ConfigRefs, n.Invariant.Status, n.Invariant.Observed, n.Invariant.Required)
		fmt.Fprintf(&b, "  evidence %q %s\n", n.Evidence.ID, n.Evidence.URL)
		for _, e := range n.Edges {
			fmt.Fprintf(&b, "  explicit lineage assertion evidence %q %s\n", e.ID, e.URL)
		}
	}
	for _, f := range r.Findings {
		fmt.Fprintf(&b, "%s %q: %s\n", f.Kind, f.Ref, f.Detail)
	}
	fmt.Fprintf(&b, "Total nodes visited: %d; node budget used: %d/%d; depth limit: %d; truncations: %d\n", r.Visited, r.Visited, r.Budget.Nodes, r.Budget.Depth, r.Truncations)
	return b.String()
}
func raw(v json.RawMessage) string {
	if len(v) == 0 {
		return "unknown"
	}
	var b bytes.Buffer
	if json.Compact(&b, v) != nil {
		return "unknown"
	}
	return b.String()
}
func display(f Field) string {
	if f.Status == "captured" {
		return "captured " + raw(f.Inline)
	}
	if f.Status == "hash-only" {
		return fmt.Sprintf("hash-only ref=%q digest=%q; value unknown", f.Ref, f.Digest)
	}
	return f.Status + "; value unknown"
}
