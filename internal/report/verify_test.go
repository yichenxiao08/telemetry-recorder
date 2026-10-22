package report

import (
	"encoding/json"
	"reflect"
	"testing"

	"pgregory.net/rapid"
)

func testFact() Fact {
	return Fact{ID: "value", Tier: "observation", State: "captured", Evidence: []Citation{{ID: "event:one", RunID: "one", Kind: "native"}}, Numbers: map[string]float64{"value": 3}, Detail: json.RawMessage(`{"value":3}`)}
}

func TestVerifierFailures(t *testing.T) {
	f := testFact()
	for _, tc := range []struct {
		name   string
		change func(*Claim)
		want   string
	}{
		{"valid", func(*Claim) {}, "valid"},
		{"fabricated", func(c *Claim) { c.Evidence[0].ID = "event:fake" }, "invalid"},
		{"wrong run", func(c *Claim) { c.Evidence[0].RunID = "other" }, "invalid"},
		{"wrong cohort", func(c *Claim) { c.Evidence[0].CohortID = "other" }, "invalid"},
		{"number", func(c *Claim) { c.Numbers["value"] = 4 }, "invalid"},
		{"causal", func(c *Claim) { c.Text = "This caused the failure" }, "invalid"},
		{"root cause", func(c *Claim) { c.Text = "The root cause is recorded" }, "invalid"},
		{"because", func(c *Claim) { c.Text = "The result changed because of this input" }, "invalid"},
		{"due", func(c *Claim) { c.Text = "Failure is due to the input" }, "invalid"},
		{"association needs comparison", func(c *Claim) { c.Tier = "association"; c.Association = &Association{} }, "unsupported"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := claimFor(f)
			tc.change(&c)
			v := Verify([]Claim{c}, []Fact{f})
			if v.Claims[0].Status != tc.want {
				t.Fatalf("%+v", v)
			}
		})
	}
}

func TestAssociationDenominatorsAndUnknown(t *testing.T) {
	f := testFact()
	f.Tier = "association"
	f.State = "ok"
	f.Association = &Association{ProblemNumerator: 2, ProblemDenominator: 10, ReferenceNumerator: 1, ReferenceDenominator: 10, ProblemRuns: 10, ReferenceRuns: 10, Selection: "pinned", Overlap: "disjoint runs; shared sessions reported"}
	c := claimFor(f)
	c.Association = nil
	if Verify([]Claim{c}, []Fact{f}).Claims[0].Status != "invalid" {
		t.Fatal("missing denominator accepted")
	}
	f = testFact()
	f.State = "unknown"
	c = claimFor(f)
	c.State = "satisfied"
	if Verify([]Claim{c}, []Fact{f}).Claims[0].Status != "invalid" {
		t.Fatal("unknown upgraded")
	}
	c = claimFor(f)
	if Verify([]Claim{c}, []Fact{f}).Claims[0].Status != "valid" {
		t.Fatal("unknown not retained")
	}
}

func TestClosedSchema(t *testing.T) {
	for _, body := range []string{`{"version":1,"caused_by":"x"}`, `{"version":1} {}`} {
		var d Document
		if Decode([]byte(body), &d) == nil {
			t.Fatal(body)
		}
	}
	if validateRequest(Request{Version: 2}) == nil {
		t.Fatal("unknown version accepted")
	}
}

func TestVerifierProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		f := testFact()
		f.Numbers["value"] = rapid.Float64Range(-1e6, 1e6).Draw(t, "value")
		c := claimFor(f)
		a, b := Verify([]Claim{c}, []Fact{f}), Verify([]Claim{c}, []Fact{f})
		if !reflect.DeepEqual(a, b) || a.Claims[0].Status != "valid" {
			t.Fatal(a, b)
		}
		c.Evidence[0].ID = "event:absent" + rapid.String().Draw(t, "suffix")
		if Verify([]Claim{c}, []Fact{f}).Claims[0].Status != "invalid" {
			t.Fatal("fabrication supported")
		}
		f.State = "unknown"
		c = claimFor(f)
		c.State = "satisfied"
		if Verify([]Claim{c}, []Fact{f}).Claims[0].Status != "invalid" {
			t.Fatal("unknown upgraded")
		}
	})
}

func TestUnknownNumbersStayUnknown(t *testing.T) {
	for _, v := range []any{map[string]any{"unknown": true, "value": 7}, map[string]any{"capture": "off", "inline": 7}, map[string]any{"capture": "hash-only", "inline": 7}} {
		if len(numbers(v)) != 0 {
			t.Fatal("unknown converted to numeric value", v)
		}
	}
	if numbers(map[string]any{"unknown": 3})["/unknown"] != 3 {
		t.Fatal("unknown COUNT hidden as if it were a capture marker")
	}
}

func TestNumericToleranceAndPowerShellBOM(t *testing.T) {
	if !sameNumbers(map[string]float64{"x": 3.5 + 1e-10}, map[string]float64{"x": 3.5}) {
		t.Fatal("documented tolerance rejected")
	}
	if sameNumbers(map[string]float64{"x": 3.5 + 1e-6}, map[string]float64{"x": 3.5}) {
		t.Fatal("misquotation accepted")
	}
	if sameNumbers(map[string]float64{"count": 3 + 1e-10}, map[string]float64{"count": 3}) {
		t.Fatal("integer quantity not exact")
	}
	var d Document
	if err := Decode(append([]byte{0xef, 0xbb, 0xbf}, []byte(`{"version":1}`)...), &d); err != nil || d.Version != 1 {
		t.Fatal(d, err)
	}
}
