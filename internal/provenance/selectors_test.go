package provenance

import "testing"

func TestSelectorSyntax(t *testing.T) {
	v := map[string]any{"inputs": map[string]any{"self.score": 7.0, `a\b`: true, "self": map[string]any{"score": 3.0}}, "output": []any{[]any{1.0, 4.0}}}
	for _, tc := range []struct {
		syntax, path string
		want         any
		valid        bool
	}{
		{"", "inputs.self.score", 3.0, true},
		{"", `inputs.self\.score`, nil, false},
		{"", `inputs.a\b`, true, true},
		{"escaped-v1", `inputs.self\.score`, 7.0, true},
		{"escaped-v1", `inputs.a\\b`, true, true},
		{"escaped-v1", "output.0.1", 4.0, true},
		{"escaped-v1", "output.-1", nil, false},
		{"escaped-v1", `inputs.a\q`, nil, false},
		{"escaped-v1", `inputs.a\`, nil, false},
		{"escaped-v1", "inputs..x", nil, false},
		{"future", "inputs.self", nil, false},
	} {
		t.Run(tc.syntax+tc.path, func(t *testing.T) {
			got, ok := GetWithSyntax(v, tc.path, tc.syntax)
			if ok != tc.valid || got != tc.want {
				t.Fatalf("got %v/%v want %v/%v", got, ok, tc.want, tc.valid)
			}
		})
	}
}

func TestCategoryScalarTypesDoNotCollide(t *testing.T) {
	seen := map[string]bool{}
	for _, v := range []any{"1", 1.0, true, "true", false, 2} {
		s, ok := Category(v)
		if !ok || seen[s] {
			t.Fatalf("category collision for %v", v)
		}
		seen[s] = true
	}
	for _, v := range []any{nil, 1.5, []any{1}, map[string]any{"unknown": "not_allowlisted"}} {
		if _, ok := Category(v); ok {
			t.Fatalf("unsupported category %v", v)
		}
	}
}
