package report

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
)

var causal = regexp.MustCompile(`(?i)\b(caused|root\s+cause|because\s+of|due\s+to)\b`)
var numericProse = regexp.MustCompile(`[0-9]`)

func claimFor(f Fact) Claim {
	// Copy maps/slices so editing a draft cannot change the fact checked against it.
	out := Claim{ID: f.ID, Fact: f.ID, Tier: f.Tier, State: f.State, Evidence: append([]Citation{}, f.Evidence...), Numbers: map[string]float64{}}
	for k, v := range f.Numbers {
		out.Numbers[k] = v
	}
	if f.Association != nil {
		a := *f.Association
		a.Excluded = map[string]map[string]int{}
		for group, counts := range f.Association.Excluded {
			a.Excluded[group] = map[string]int{}
			for k, v := range counts {
				a.Excluded[group][k] = v
			}
		}
		out.Association = &a
	}
	out.Text = "Recorded evidence observation"
	if f.Tier == "association" {
		out.Text = "Recorded cohort association"
	}
	return out
}
func claimShape(c Claim) error {
	if c.ID == "" || len(c.ID) > 512 || c.Text == "" || len(c.Text) > 4096 || len(c.Evidence) > 128 || (len(c.Evidence) == 0 && (c.State != "unknown" || c.Tier != "observation")) {
		return fmt.Errorf("bounded id, text and evidence required")
	}
	if causal.MatchString(c.Text) {
		return fmt.Errorf("causal phrasing rejected")
	}
	if numericProse.MatchString(c.Text) {
		return fmt.Errorf("numeric assertions must use structured numbers, not prose")
	}
	if c.Tier != "observation" && c.Tier != "association" && c.Tier != "hypothesis" {
		return fmt.Errorf("unknown tier")
	}
	if c.Tier == "hypothesis" {
		if c.State != "proposed" || len(c.Numbers) != 0 || c.Association != nil {
			return fmt.Errorf("supplied hypotheses must remain proposed with no asserted numeric quantities or association")
		}
		h := c.Hypothesis
		if h == nil || len(h.Alternatives) == 0 || len(h.MissingEvidence) == 0 || strings.TrimSpace(h.Experiment) == "" || strings.TrimSpace(h.Predicted) == "" || strings.TrimSpace(h.Falsifying) == "" {
			return fmt.Errorf("hypothesis needs alternatives, missing evidence, experiment, predicted and falsifying results")
		}
		for _, s := range append(append([]string{}, h.Alternatives...), h.MissingEvidence...) {
			if strings.TrimSpace(s) == "" || len(s) > 4096 {
				return fmt.Errorf("hypothesis alternatives and missing evidence need bounded nonempty text")
			}
		}
		if len(h.Alternatives) > 16 || len(h.MissingEvidence) > 16 || len(h.Experiment) > 4096 || len(h.Predicted) > 4096 || len(h.Falsifying) > 4096 {
			return fmt.Errorf("hypothesis detail budget exceeded")
		}
	} else if c.Hypothesis != nil {
		return fmt.Errorf("hypothesis fields require hypothesis tier")
	}
	return nil
}
func sameNumbers(a, b map[string]float64) bool {
	if len(a) != len(b) {
		return false
	}
	for k, x := range a {
		y, ok := b[k]
		if !ok || math.IsNaN(x) || math.IsInf(x, 0) || math.IsNaN(y) || math.IsInf(y, 0) || (math.Trunc(y) == y && x != y) || math.Abs(x-y) > NumberTolerance*math.Max(1, math.Abs(y)) {
			return false
		}
	}
	return true
}

func consistentAssociation(a *Association) bool {
	if a == nil {
		return true
	}
	if a.ProblemNumerator < 0 || a.ReferenceNumerator < 0 || a.ProblemDenominator < 0 || a.ReferenceDenominator < 0 || a.ProblemNumerator > a.ProblemDenominator || a.ReferenceNumerator > a.ReferenceDenominator || a.ProblemDenominator+a.ProblemMissing != a.ProblemRuns || a.ReferenceDenominator+a.ReferenceMissing != a.ReferenceRuns || a.ProblemMissing < 0 || a.ReferenceMissing < 0 || a.ProblemSessions < 0 || a.ReferenceSessions < 0 || a.ProblemSessions > a.ProblemRuns || a.ReferenceSessions > a.ReferenceRuns || a.SharedSessions < 0 || a.SharedSessions > a.ProblemSessions || a.SharedSessions > a.ReferenceSessions || a.Selection == "" || a.Overlap == "" {
		return false
	}
	for _, counts := range a.Excluded {
		for _, n := range counts {
			if n < 0 {
				return false
			}
		}
	}
	return true
}

// Verify checks structural support against facts rebuilt from a bounded store
// snapshot. Valid means supported as quoted, never semantic or causal truth.
func Verify(claims []Claim, facts []Fact) Verification {
	out := Verification{Claims: []ClaimResult{}}
	byID := map[string]Fact{}
	all := map[Citation]bool{}
	for _, f := range facts {
		byID[f.ID] = f
		for _, id := range f.Evidence {
			all[id] = true
		}
	}
	seen := map[string]bool{}
	for _, c := range claims {
		result := ClaimResult{ID: c.ID, Status: "valid", Reason: "structured quotation and scoped citations match rebuilt evidence"}
		f, exists := byID[c.Fact]
		validIDs := true
		for _, id := range c.Evidence {
			out.Citations++
			allowed := false
			pool := f.Evidence
			if c.Tier == "hypothesis" {
				allowed = all[id]
			} else {
				for _, want := range pool {
					if want == id {
						allowed = true
						break
					}
				}
			}
			if allowed {
				out.ValidCitations++
			} else {
				validIDs = false
			}
		}
		switch {
		case seen[c.ID]:
			result.Status, result.Reason = "invalid", "duplicate claim id"
		case claimShape(c) != nil:
			result.Status, result.Reason = "invalid", claimShape(c).Error()
		case !validIDs:
			result.Status, result.Reason = "invalid", "citation absent, outside scope, wrong run/cohort or unrelated to this fact"
		case c.Tier == "hypothesis":
			result.Status, result.Reason = "unsupported", "supplied hypothesis only; citations resolve but experiment and explanation are untested"
		case !exists:
			result.Status, result.Reason = "invalid", "unknown fact binding"
		case c.Tier != f.Tier:
			result.Status, result.Reason = "unsupported", "claim tier does not match rebuilt evidence; association requires a supported cohort comparison"
		case !consistentAssociation(c.Association):
			result.Status, result.Reason = "invalid", "inconsistent association counts, denominators, sessions or policies"
		case c.State != f.State:
			result.Status, result.Reason = "invalid", "state mismatch; unknown or insufficient evidence cannot be upgraded"
		case !sameNumbers(c.Numbers, f.Numbers):
			result.Status, result.Reason = "invalid", "quoted numbers or required quantities differ from rebuilt evidence"
		case !reflect.DeepEqual(c.Association, f.Association):
			result.Status, result.Reason = "invalid", "association numerators, denominators, sample sizes, sessions, missing/excluded counts or policies missing/mismatched"
		case !reflect.DeepEqual(c.Evidence, f.Evidence):
			result.Status, result.Reason = "invalid", "required evidence omitted, reordered or duplicated"
		case len(c.Evidence) == 0:
			result.Status, result.Reason = "unsupported", "unknown within the declared negative-search scope; no recorded detail to cite, no absence conclusion"
		}
		seen[c.ID] = true
		switch result.Status {
		case "valid":
			out.Valid++
		case "invalid":
			out.Invalid++
		case "unsupported":
			out.Unsupported++
		}
		out.Claims = append(out.Claims, result)
	}
	if out.Citations > 0 {
		out.CitationValidityRate = float64(out.ValidCitations) / float64(out.Citations)
	}
	if len(claims) > 0 {
		out.ClaimSupportRate = float64(out.Valid) / float64(len(claims))
	}
	return out
}
