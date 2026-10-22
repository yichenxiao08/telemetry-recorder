package provenance

import (
	"fmt"
	"math"
	"strconv"

	"github.com/signalscout/signalscout/internal/model"
)

// ValidateRelations bounds work before any expression is evaluated. Type errors
// involving captured fields stay unknown, never a coerced value.
func ValidateRelations(p model.AnalysisProfile) error {
	if p.Relations == nil {
		return nil
	}
	if p.Relations.Version != 1 || len(p.Relations.Rules) < 1 || len(p.Relations.Rules) > 16 {
		return fmt.Errorf("relations require version 1 and 1..16 rules")
	}
	seen := map[string]bool{}
	for _, r := range p.Relations.Rules {
		if r.ID == "" || len(r.ID) > 128 || seen[r.ID] || r.Unit == "" || len(r.Unit) > 128 || r.Tolerance == nil || !finite(*r.Tolerance) || *r.Tolerance < 0 {
			return fmt.Errorf("relation requires unique ID, unit and explicit finite nonnegative tolerance")
		}
		seen[r.ID] = true
		nodes := 0
		for _, e := range []*model.Expression{r.When, &r.Assert} {
			if e == nil {
				continue
			}
			typ, err := validateExpression(*e, p.SelectorSyntax, 0, &nodes)
			if err != nil {
				return fmt.Errorf("relation %s: %w", r.ID, err)
			}
			if typ == "number" {
				return fmt.Errorf("relation %s requires boolean predicate", r.ID)
			}
		}
	}
	return nil
}
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func validateExpression(e model.Expression, syntax string, depth int, nodes *int) (string, error) {
	*nodes++
	if depth > 8 || *nodes > 64 {
		return "", fmt.Errorf("expression exceeds depth 8 or 64 nodes (including precondition)")
	}
	bad := func() (string, error) { return "", fmt.Errorf("invalid expression form/operator %q", e.Op) }
	if e.Op == "field" {
		if e.Number != nil || e.Boolean != nil || len(e.Args) != 0 {
			return bad()
		}
		if _, ok := SelectorParts(e.Field, syntax); !ok {
			return bad()
		}
		return "field", nil
	}
	if e.Op == "number" {
		if e.Number == nil || !finite(*e.Number) || e.Field != "" || e.Boolean != nil || len(e.Args) != 0 {
			return bad()
		}
		return "number", nil
	}
	if e.Op == "boolean" {
		if e.Boolean == nil || e.Field != "" || e.Number != nil || len(e.Args) != 0 {
			return bad()
		}
		return "boolean", nil
	}
	if e.Field != "" || e.Number != nil || e.Boolean != nil {
		return bad()
	}
	want := 2
	result, operand := "boolean", "number"
	switch e.Op {
	case "add", "sub":
		result = "number"
	case "abs":
		result = "number"
		want = 1
	case "eq", "ne":
		operand = "any"
	case "lt", "le", "gt", "ge":
	case "and", "or":
		operand = "boolean"
	case "not":
		operand = "boolean"
		want = 1
	default:
		return bad()
	}
	if len(e.Args) != want {
		return bad()
	}
	types := []string{}
	for _, arg := range e.Args {
		typ, err := validateExpression(arg, syntax, depth+1, nodes)
		if err != nil {
			return "", err
		}
		if typ != "field" && operand != "any" && typ != operand {
			return bad()
		}
		types = append(types, typ)
	}
	if operand == "any" && types[0] != "field" && types[1] != "field" && types[0] != types[1] {
		return bad()
	}
	return result, nil
}

type RelationEvaluation struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func EvaluateRelation(values map[string]any, r model.RelationRule, syntax string) RelationEvaluation {
	// Callers validate the profile first; guard malformed direct calls as well.
	if err := ValidateRelations(model.AnalysisProfile{SelectorSyntax: syntax, Relations: &model.RelationPolicy{Version: 1, Rules: []model.RelationRule{r}}}); err != nil {
		return RelationEvaluation{"unknown", "invalid declaration"}
	}
	if r.When != nil {
		v, ok := evaluate(*r.When, values, syntax, *r.Tolerance)
		b, valid := v.(bool)
		if !ok || !valid {
			return RelationEvaluation{"unknown", "precondition operand unavailable or wrong type"}
		}
		if !b {
			return RelationEvaluation{"not_applicable", "declared precondition is false"}
		}
	}
	v, ok := evaluate(r.Assert, values, syntax, *r.Tolerance)
	b, valid := v.(bool)
	if !ok || !valid {
		return RelationEvaluation{"unknown", "operand missing, capture-off/hash-only, unsupported type or nonfinite arithmetic"}
	}
	if b {
		return RelationEvaluation{"satisfied", "recorded values meet the declared relation"}
	}
	return RelationEvaluation{"violated", "recorded values do not meet the declared relation"}
}
func evaluate(e model.Expression, values map[string]any, syntax string, tolerance float64) (any, bool) {
	switch e.Op {
	case "field":
		v, ok := relationValue(values, e.Field, syntax)
		if !ok {
			return nil, false
		}
		switch n := v.(type) {
		case float64:
			return n, finite(n)
		case bool:
			return n, true
		default:
			return nil, false
		}
	case "number":
		return *e.Number, true
	case "boolean":
		return *e.Boolean, true
	}
	a, ok := evaluate(e.Args[0], values, syntax, tolerance)
	if !ok {
		return nil, false
	}
	var b any
	if len(e.Args) == 2 {
		b, ok = evaluate(e.Args[1], values, syntax, tolerance)
		if !ok {
			return nil, false
		}
	}
	x, xok := a.(float64)
	y, yok := b.(float64)
	if e.Op == "abs" && xok {
		return math.Abs(x), true
	}
	if (e.Op == "add" || e.Op == "sub") && xok && yok {
		n := x + y
		if e.Op == "sub" {
			n = x - y
		}
		return n, finite(n)
	}
	if e.Op == "eq" || e.Op == "ne" {
		same := false
		if xok && yok {
			d := math.Abs(x - y)
			if !finite(d) {
				return nil, false
			}
			lo, hi := y-tolerance, y+tolerance
			if !finite(lo) || !finite(hi) {
				return nil, false
			}
			same = x >= lo && x <= hi
		} else {
			u, uok := a.(bool)
			v, vok := b.(bool)
			if !uok || !vok {
				return nil, false
			}
			same = u == v
		}
		if e.Op == "ne" {
			same = !same
		}
		return same, true
	}
	if xok && yok {
		d := x - y
		if !finite(d) {
			return nil, false
		}
		lo, hi := y-tolerance, y+tolerance
		if !finite(lo) || !finite(hi) {
			return nil, false
		}
		switch e.Op {
		case "lt":
			return x < lo, true
		case "le":
			return x <= hi, true
		case "gt":
			return x > hi, true
		case "ge":
			return x >= lo, true
		}
	}
	u, uok := a.(bool)
	v, vok := b.(bool)
	if e.Op == "not" && uok {
		return !u, true
	}
	if uok && vok {
		switch e.Op {
		case "and":
			return u && v, true
		case "or":
			return u || v, true
		}
	}
	return nil, false
}

// Unknown element markers are reserved by the recorder. Do not navigate into
// contradictory detail beneath one, even if a malformed producer includes it.
func relationValue(values map[string]any, path, syntax string) (any, bool) {
	parts, ok := SelectorParts(path, syntax)
	if !ok {
		return nil, false
	}
	var v any = values
	for _, part := range parts {
		if m, yes := v.(map[string]any); yes {
			if _, unknown := m["unknown"]; unknown {
				return nil, false
			}
		}
		switch x := v.(type) {
		case map[string]any:
			v, ok = x[part]
			if !ok {
				return nil, false
			}
		case []any:
			i, err := strconv.Atoi(part)
			if err != nil || i < 0 || i >= len(x) {
				return nil, false
			}
			v = x[i]
		default:
			return nil, false
		}
	}
	return v, true
}
