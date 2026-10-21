// Package provenance reads explicitly captured decision values, not log text.
package provenance

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/signalscout/signalscout/internal/model"
)

func ValidateProfile(p model.AnalysisProfile) error {
	if p.ID == "" || p.Unit == "" || p.Workload == "" || p.Step == "" || p.EnvelopeProfile == "" || p.EnvelopeVersion == "" || len(p.Numeric)+len(p.Categories) > 32 {
		return fmt.Errorf("bounded unit/profile/workload/step required")
	}
	names := map[string]bool{}
	for _, f := range p.Numeric {
		if err := validateRole(f.Role, f.Rationale, p.RoleAuthor); err != nil {
			return err
		}
		if f.Name == "" || names[f.Name] || f.Unit == "" || !finitePositive(f.Epsilon) || !finitePositive(f.Floor) || !selectorValid(f.Selector) || f.Subtract != "" && !selectorValid(f.Subtract) {
			return fmt.Errorf("invalid numeric feature %q", f.Name)
		}
		names[f.Name] = true
	}
	for _, f := range p.Categories {
		if err := validateRole(f.Role, f.Rationale, p.RoleAuthor); err != nil {
			return err
		}
		if f.Name == "" || names[f.Name] || !selectorValid(f.Selector) {
			return fmt.Errorf("invalid categorical feature %q", f.Name)
		}
		names[f.Name] = true
	}
	for _, s := range []string{p.Invariant.Declared, p.Invariant.Current, p.Invariant.Used} {
		if !selectorValid(s) {
			return fmt.Errorf("invalid invariant selector")
		}
	}
	if !finitePositive(p.Invariant.Delta) {
		return fmt.Errorf("positive version delta required")
	}
	return nil
}
func validateRole(role, rationale, author string) error {
	if role == "" {
		if rationale != "" {
			return fmt.Errorf("role required for rationale")
		}
		return nil
	}
	if (role != "candidate" && role != "symptom") || rationale == "" || author == "" || len(rationale) > 512 || len(author) > 128 {
		return fmt.Errorf("versioned role author and rationale required")
	}
	return nil
}
func finitePositive(x float64) bool { return x > 0 && !math.IsInf(x, 0) && !math.IsNaN(x) }
func selectorValid(s string) bool   { return s != "" && len(s) <= 256 && len(strings.Split(s, ".")) <= 8 }

func Get(values map[string]any, path string) (any, bool) {
	var value any = values
	for _, key := range strings.Split(path, ".") {
		switch x := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = x[key]
			if !ok {
				return nil, false
			}
		case []any:
			i, err := strconv.Atoi(key)
			if err != nil || i < 0 || i >= len(x) {
				return nil, false
			}
			value = x[i]
		default:
			return nil, false
		}
	}
	return value, true
}

func Decode(signal model.Signal, profile model.AnalysisProfile) (model.DecisionObservation, bool) {
	out := model.DecisionObservation{RunID: signal.RunID, EvidenceID: signal.ID, TS: signal.TS, Values: map[string]any{}}
	if signal.Kind != "event" || signal.Attrs["signalscout.record.kind"] != "decision" {
		return out, false
	}
	body, ok := signal.Attrs["signalscout.record.json"].(string)
	if !ok || len(body) > 8192 {
		return out, false
	}
	var env struct {
		Version int                          `json:"version"`
		Profile struct{ ID, Version string } `json:"profile"`
		Step    struct{ ID string }          `json:"step"`
		Inputs  []struct {
			Name    string
			Capture string
			Inline  json.RawMessage
		} `json:"inputs"`
		Output struct {
			Inline  json.RawMessage
			Capture string
		} `json:"output"`
		Verdict *string        `json:"verdict"`
		Reason  *string        `json:"reason_code"`
		Chosen  map[string]any `json:"chosen"`
	}
	if json.Unmarshal([]byte(body), &env) != nil || env.Version != 1 || env.Profile.ID != profile.EnvelopeProfile || env.Profile.Version != profile.EnvelopeVersion || env.Step.ID != profile.Step || len(env.Inputs) > 32 {
		return out, false
	}
	inputs := map[string]any{}
	seen := map[string]bool{}
	for _, in := range env.Inputs {
		if seen[in.Name] {
			return out, false
		}
		seen[in.Name] = true
		if in.Capture != "allowed" || len(in.Inline) == 0 {
			continue
		}
		var value any
		if json.Unmarshal(in.Inline, &value) == nil {
			inputs[in.Name] = value
		}
	}
	out.Values["inputs"] = inputs
	if env.Output.Capture == "allowed" && len(env.Output.Inline) > 0 {
		var output any
		if json.Unmarshal(env.Output.Inline, &output) == nil {
			out.Values["output"] = output
		}
	}
	if env.Verdict != nil {
		out.Values["verdict"] = *env.Verdict
	}
	if env.Reason != nil {
		out.Values["reason_code"] = *env.Reason
	}
	out.Values["chosen"] = env.Chosen
	return out, true
}
