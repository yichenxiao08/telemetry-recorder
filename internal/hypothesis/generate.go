// Package hypothesis proposes text through a bounded client; it never edits reports.
package hypothesis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/signalscout/signalscout/internal/report"
)

const MaxInputBytes = 1 << 20
const MaxOutputBytes = 64 << 10
const MaxClaims = 16

type Input struct {
	Observations []report.Claim `json:"observations"`
	Associations []report.Claim `json:"associations"`
}

type Candidate struct {
	Text            string            `json:"text"`
	Evidence        []report.Citation `json:"evidence"`
	Alternatives    []string          `json:"alternatives"`
	MissingEvidence []string          `json:"missing_evidence"`
	Change          string            `json:"change"`
	Measure         string            `json:"measure"`
	Predicted       string            `json:"predicted_result"`
	Against         string            `json:"falsifying_result"`
}

// Client implementations return candidates only. Generate validates all output.
type Client interface {
	Propose(context.Context, Input) ([]Candidate, error)
}

// Mock is a fixed template, independent of the meaning of the input text.
type Mock struct{}

func (Mock) Propose(_ context.Context, in Input) ([]Candidate, error) {
	for _, claims := range [][]report.Claim{in.Observations, in.Associations} {
		for _, c := range claims {
			if len(c.Evidence) > 0 {
				return []Candidate{{
					Text:            "A difference in recorded conditions is a proposed explanation to examine",
					Evidence:        []report.Citation{c.Evidence[0]},
					Alternatives:    []string{"A difference in capture coverage is another proposed explanation"},
					MissingEvidence: []string{"Matched repeats with consistent capture coverage"},
					Change:          "Vary a declared input condition in a separately reviewed matched repeat",
					Measure:         "Compare recorded outcomes and capture coverage across matched repeats",
					Predicted:       "The recorded outcome differs across the declared conditions",
					Against:         "The recorded outcome remains the same across the declared conditions with consistent coverage",
				}}, nil
			}
		}
	}
	return nil, fmt.Errorf("mock requires at least one cited observation or association")
}

// Supplied is exactly the existing report build --hypotheses JSON envelope.
type Supplied struct {
	Version int            `json:"version"`
	Claims  []report.Claim `json:"claims"`
}

func bound(v any, limit int) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if len(b)+1 > limit {
		return fmt.Errorf("size exceeds %d bytes; rejected", limit)
	}
	return nil
}

// Generate checks report-local citations only, not evidence truth. Importing via
// report build rechecks citations against rebuilt store facts as before.
func Generate(ctx context.Context, doc report.Document, client Client) (Supplied, error) {
	empty := Supplied{}
	if doc.Version != report.Version {
		return empty, fmt.Errorf("report version must be one")
	}
	if err := bound(doc, MaxInputBytes); err != nil {
		return empty, err
	}
	if len(doc.Claims) > 4096 {
		return empty, fmt.Errorf("too many input claims")
	}
	in := Input{Observations: []report.Claim{}, Associations: []report.Claim{}}
	facts := []report.Fact{}
	seen := map[string]bool{}
	for _, c := range doc.Claims {
		seen[c.ID] = true
		switch c.Tier {
		case "observation":
			in.Observations = append(in.Observations, c)
		case "association":
			in.Associations = append(in.Associations, c)
		default:
			continue
		}
		facts = append(facts, report.Fact{ID: c.ID, Evidence: append([]report.Citation{}, c.Evidence...)})
	}
	// Deep-copy the client input: a client cannot change the report or citation pool.
	b, err := json.Marshal(in)
	if err != nil {
		return empty, err
	}
	var clientInput Input
	if err = json.Unmarshal(b, &clientInput); err != nil {
		return empty, err
	}
	if client == nil {
		return empty, fmt.Errorf("client required")
	}
	candidates, err := client.Propose(ctx, clientInput)
	if err != nil {
		return empty, err
	}
	if len(candidates) == 0 || len(candidates) > MaxClaims {
		return empty, fmt.Errorf("requires one to %d candidates", MaxClaims)
	}
	if err = bound(candidates, MaxOutputBytes); err != nil {
		return empty, err
	}
	out := Supplied{Version: report.Version, Claims: []report.Claim{}}
	for i, c := range candidates {
		id := fmt.Sprintf("mock-hypothesis-%d", i+1)
		if seen[id] {
			return empty, fmt.Errorf("generated claim id collision: %s", id)
		}
		if strings.TrimSpace(c.Change) == "" || strings.TrimSpace(c.Measure) == "" {
			return empty, fmt.Errorf("candidate %d requires change and measure", i+1)
		}
		claim := report.Claim{ID: id, Tier: "hypothesis", State: "proposed", Text: c.Text, Evidence: c.Evidence, Numbers: map[string]float64{}, Hypothesis: &report.Hypothesis{
			Alternatives: c.Alternatives, MissingEvidence: c.MissingEvidence,
			Experiment: "Change: " + c.Change + "; Measure: " + c.Measure,
			Predicted:  c.Predicted, Falsifying: c.Against,
		}}
		// Apply the existing lexical policy to every generated prose field, including
		// alternatives and proposals. Reject unchanged; never attempt repair.
		texts := []string{c.Text, c.Change, c.Measure, c.Predicted, c.Against}
		texts = append(texts, c.Alternatives...)
		texts = append(texts, c.MissingEvidence...)
		for _, text := range texts {
			check := claim
			check.Text = text
			v := report.Verify([]report.Claim{check}, facts)
			if v.Invalid != 0 || v.Unsupported != 1 {
				return empty, fmt.Errorf("candidate %d rejected: %+v", i+1, v.Claims)
			}
		}
		out.Claims = append(out.Claims, claim)
	}
	if err = bound(out, MaxOutputBytes); err != nil {
		return empty, err
	}
	return out, nil
}
