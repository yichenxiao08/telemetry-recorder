// Package report composes existing evidence. It never generates hypotheses.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/signalscout/signalscout/internal/lineage"
	"github.com/signalscout/signalscout/internal/model"
)

const Version = 1
const NumberTolerance = 1e-9

func Size(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if len(b) > 32<<20 {
		return fmt.Errorf("report exceeds 32 MiB; narrow sources")
	}
	return nil
}

type Citation struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	RunID    string `json:"run_id,omitempty"`
	CohortID string `json:"cohort_id,omitempty"`
}
type Association struct {
	ProblemNumerator     int                       `json:"problem_numerator"`
	ProblemDenominator   int                       `json:"problem_denominator"`
	ReferenceNumerator   int                       `json:"reference_numerator"`
	ReferenceDenominator int                       `json:"reference_denominator"`
	ProblemRuns          int                       `json:"problem_runs"`
	ReferenceRuns        int                       `json:"reference_runs"`
	ProblemSessions      int                       `json:"problem_independent_sessions"`
	ReferenceSessions    int                       `json:"reference_independent_sessions"`
	SharedSessions       int                       `json:"shared_sessions"`
	IndependenceKnown    bool                      `json:"independence_known"`
	ProblemMissing       int                       `json:"problem_missing"`
	ReferenceMissing     int                       `json:"reference_missing"`
	Excluded             map[string]map[string]int `json:"excluded"`
	Selection            string                    `json:"selection_policy"`
	Overlap              string                    `json:"overlap_policy"`
}

// Missing zero-valued denominators must not be indistinguishable from explicit
// zero. All association quantities and policies are required on JSON input.
func (a *Association) UnmarshalJSON(body []byte) error {
	type plain Association
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return err
	}
	for _, k := range []string{"problem_numerator", "problem_denominator", "reference_numerator", "reference_denominator", "problem_runs", "reference_runs", "problem_independent_sessions", "reference_independent_sessions", "shared_sessions", "independence_known", "problem_missing", "reference_missing", "excluded", "selection_policy", "overlap_policy"} {
		if v, ok := fields[k]; !ok || bytes.Equal(v, []byte("null")) {
			return fmt.Errorf("association requires explicit %s", k)
		}
	}
	var value plain
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if err := d.Decode(&value); err != nil {
		return err
	}
	*a = Association(value)
	return nil
}

type Hypothesis struct {
	Alternatives    []string `json:"alternatives"`
	MissingEvidence []string `json:"missing_evidence"`
	Experiment      string   `json:"experiment"`
	Predicted       string   `json:"predicted_result"`
	Falsifying      string   `json:"falsifying_result"`
}
type Claim struct {
	ID          string             `json:"id"`
	Fact        string             `json:"fact"`
	Tier        string             `json:"tier"`
	Text        string             `json:"text"`
	State       string             `json:"state"`
	Evidence    []Citation         `json:"evidence"`
	Numbers     map[string]float64 `json:"numbers"`
	Association *Association       `json:"association,omitempty"`
	Hypothesis  *Hypothesis        `json:"hypothesis,omitempty"`
}
type Fact struct {
	ID          string             `json:"id"`
	Tier        string             `json:"tier"`
	State       string             `json:"state"`
	Evidence    []Citation         `json:"evidence"`
	Numbers     map[string]float64 `json:"numbers"`
	Association *Association       `json:"association,omitempty"`
	Detail      json.RawMessage    `json:"detail"`
}
type Comparison struct {
	Problem   string `json:"problem"`
	Reference string `json:"reference"`
}
type Trace struct {
	Target  string                 `json:"target"`
	RunID   string                 `json:"run_id"`
	Profile *model.AnalysisProfile `json:"profile,omitempty"`
	Budget  lineage.Options        `json:"budget"`
}
type Request struct {
	Version     int          `json:"version"`
	From        int64        `json:"from_ns"`
	To          int64        `json:"to_ns"`
	Limit       int          `json:"record_limit"`
	Cohorts     []string     `json:"cohorts"`
	Comparisons []Comparison `json:"comparisons"`
	Traces      []Trace      `json:"traces"`
	Records     []Citation   `json:"records"`
}
type Source struct {
	ID   string          `json:"id"`
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}
type Document struct {
	Version int      `json:"version"`
	Request Request  `json:"request"`
	Sources []Source `json:"sources"`
	Claims  []Claim  `json:"claims"`
	Notes   []string `json:"notes"`
}
type ClaimResult struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}
type Verification struct {
	Claims               []ClaimResult `json:"claims"`
	Valid                int           `json:"valid"`
	Invalid              int           `json:"invalid"`
	Unsupported          int           `json:"unsupported"`
	Citations            int           `json:"citations"`
	ValidCitations       int           `json:"valid_citations"`
	CitationValidityRate float64       `json:"citation_validity_rate"`
	ClaimSupportRate     float64       `json:"claim_support_rate"`
}

// Decode bounds every input and rejects extra fields and trailing documents.
func Decode(body []byte, v any) error {
	if len(body) > 32<<20 {
		return fmt.Errorf("input exceeds 32 MiB")
	}
	// Windows PowerShell 5 writes an UTF-8 BOM with Set-Content -Encoding utf8.
	body = bytes.TrimPrefix(body, []byte{0xef, 0xbb, 0xbf})
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return fmt.Errorf("extra JSON content")
	}
	return nil
}
