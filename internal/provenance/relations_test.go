package provenance

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/signalscout/signalscout/internal/model"
	"pgregory.net/rapid"
)

func relation(t *testing.T, expr string) model.RelationRule {
	t.Helper()
	var r model.RelationRule
	if err := json.Unmarshal([]byte(`{"id":"r","unit":"synthetic units","tolerance":0.1,"assert":`+expr+`}`), &r); err != nil {
		t.Fatal(err)
	}
	return r
}
func TestRelationShapesAndUnknown(t *testing.T) {
	values := map[string]any{"inputs": map[string]any{"a": 2.0, "b": 3.0, "off": map[string]any{"unknown": "not_allowlisted"}, "yes": true}, "output": []any{[]any{1.0, 2.0, 3.0, 4.0}, 3.0}}
	for _, tc := range []struct{ name, expr, want string }{
		{"equality", `{"op":"eq","args":[{"op":"field","field":"inputs.a"},{"op":"number","number":2.1}]}`, "satisfied"},
		{"nested tuple abs difference", `{"op":"eq","args":[{"op":"field","field":"output.1"},{"op":"abs","args":[{"op":"sub","args":[{"op":"field","field":"output.0.3"},{"op":"field","field":"output.0.0"}]}]}]}`, "satisfied"},
		{"ordering", `{"op":"gt","args":[{"op":"field","field":"inputs.a"},{"op":"field","field":"inputs.b"}]}`, "violated"},
		{"bounds", `{"op":"and","args":[{"op":"ge","args":[{"op":"field","field":"inputs.a"},{"op":"number","number":1}]},{"op":"le","args":[{"op":"field","field":"inputs.a"},{"op":"number","number":3}]}]}`, "satisfied"},
		{"boolean", `{"op":"not","args":[{"op":"field","field":"inputs.yes"}]}`, "violated"},
		{"missing", `{"op":"eq","args":[{"op":"field","field":"inputs.missing"},{"op":"number","number":0}]}`, "unknown"},
		{"off", `{"op":"eq","args":[{"op":"field","field":"inputs.off"},{"op":"number","number":0}]}`, "unknown"},
		{"no short circuit unknown", `{"op":"or","args":[{"op":"boolean","boolean":true},{"op":"field","field":"inputs.missing"}]}`, "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := relation(t, tc.expr)
			p := model.AnalysisProfile{Relations: &model.RelationPolicy{Version: 1, Rules: []model.RelationRule{r}}}
			if err := ValidateRelations(p); err != nil {
				t.Fatal(err)
			}
			got := EvaluateRelation(values, r, "")
			if got.Status != tc.want {
				t.Fatalf("%+v", got)
			}
		})
	}
	r := relation(t, `{"op":"field","field":"inputs.missing"}`)
	b := false
	r.When = &model.Expression{Op: "boolean", Boolean: &b}
	if got := EvaluateRelation(values, r, ""); got.Status != "not_applicable" {
		t.Fatal(got)
	}
	r.When = &model.Expression{Op: "field", Field: "inputs.missing"}
	if got := EvaluateRelation(values, r, ""); got.Status != "unknown" {
		t.Fatal(got)
	}
}
func TestRelationValidationAndFiniteArithmetic(t *testing.T) {
	r := relation(t, `{"op":"eq","args":[{"op":"field","field":"inputs.a"},{"op":"number","number":0}]}`)
	for _, change := range []func(*model.RelationPolicy){
		func(p *model.RelationPolicy) { p.Version = 2 },
		func(p *model.RelationPolicy) { p.Rules[0].Tolerance = nil },
		func(p *model.RelationPolicy) { n := -1.0; p.Rules[0].Tolerance = &n },
		func(p *model.RelationPolicy) { p.Rules[0].Assert.Op = "eval" },
		func(p *model.RelationPolicy) { p.Rules[0].Assert.Args = nil },
		func(p *model.RelationPolicy) { p.Rules[0].Assert.Args[0].Number = new(float64) },
		func(p *model.RelationPolicy) { p.Rules = append(p.Rules, p.Rules[0]) },
		func(p *model.RelationPolicy) { p.Rules[0].Unit = "" },
	} {
		raw, _ := json.Marshal(model.RelationPolicy{Version: 1, Rules: []model.RelationRule{r}})
		var p model.RelationPolicy
		_ = json.Unmarshal(raw, &p)
		change(&p)
		if ValidateRelations(model.AnalysisProfile{Relations: &p}) == nil {
			t.Fatal("invalid relation accepted", p)
		}
	}
	r.Assert = model.Expression{Op: "eq", Args: []model.Expression{{Op: "add", Args: []model.Expression{{Op: "field", Field: "a"}, {Op: "field", Field: "a"}}}, {Op: "field", Field: "a"}}}
	if got := EvaluateRelation(map[string]any{"a": math.MaxFloat64}, r, ""); got.Status != "unknown" {
		t.Fatal(got)
	}
}
func TestRelationProperties(t *testing.T) {
	r := relation(t, `{"op":"eq","args":[{"op":"field","field":"a"},{"op":"field","field":"b"}]}`)
	rapid.Check(t, func(t *rapid.T) {
		a := float64(rapid.IntRange(-10000, 10000).Draw(t, "a"))
		b := float64(rapid.IntRange(-10000, 10000).Draw(t, "b"))
		v := map[string]any{"a": a, "b": b}
		one, two := EvaluateRelation(v, r, ""), EvaluateRelation(v, r, "")
		if one != two {
			t.Fatal("nondeterministic")
		}
		delete(v, "b")
		if EvaluateRelation(v, r, "").Status != "unknown" {
			t.Fatal("unknown became value")
		}
	})
}

func TestUnknownMarkerCannotExposeNestedValue(t *testing.T) {
	r := relation(t, `{"op":"eq","args":[{"op":"field","field":"output.0.value"},{"op":"number","number":7}]}`)
	v := map[string]any{"output": []any{map[string]any{"unknown": "not_allowlisted", "value": 7.0}}}
	if got := EvaluateRelation(v, r, ""); got.Status != "unknown" {
		t.Fatal(got)
	}
}

func TestRelationExpressionWorkBounds(t *testing.T) {
	r := relation(t, `{"op":"field","field":"b"}`)
	for n := 0; n < 10; n++ {
		r.Assert = model.Expression{Op: "not", Args: []model.Expression{r.Assert}}
	}
	if ValidateRelations(model.AnalysisProfile{Relations: &model.RelationPolicy{Version: 1, Rules: []model.RelationRule{r}}}) == nil {
		t.Fatal("deep tree accepted")
	}
	var tree func(int) model.Expression
	tree = func(d int) model.Expression {
		if d == 0 {
			return model.Expression{Op: "field", Field: "b"}
		}
		return model.Expression{Op: "and", Args: []model.Expression{tree(d - 1), tree(d - 1)}}
	}
	r.Assert = tree(6)
	if ValidateRelations(model.AnalysisProfile{Relations: &model.RelationPolicy{Version: 1, Rules: []model.RelationRule{r}}}) == nil {
		t.Fatal("wide tree accepted")
	}
}
