package hypothesis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/signalscout/signalscout/internal/report"
)

type clientFunc func(context.Context, Input) ([]Candidate, error)

func (f clientFunc) Propose(ctx context.Context, in Input) ([]Candidate, error) { return f(ctx, in) }

func fixture() report.Document {
	return report.Document{Version: 1, Claims: []report.Claim{{ID: "observed", Tier: "observation", Text: "Recorded evidence observation", Evidence: []report.Citation{{ID: "event:synthetic", Kind: "native", RunID: "synthetic"}}}}}
}

func TestDeterministicUnsupportedAndImmutable(t *testing.T) {
	doc := fixture()
	before, _ := json.Marshal(doc)
	a, err := Generate(context.Background(), doc, Mock{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(context.Background(), doc, Mock{})
	if err != nil {
		t.Fatal(err)
	}
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	after, _ := json.Marshal(doc)
	if string(x) != string(y) || string(before) != string(after) {
		t.Fatal("nondeterministic or input mutated")
	}
	var supplied Supplied
	if err = report.Decode(x, &supplied); err != nil {
		t.Fatal(err)
	}
	v := report.Verify(supplied.Claims, []report.Fact{{Evidence: doc.Claims[0].Evidence}})
	if v.Unsupported != 1 || v.Valid != 0 || v.Invalid != 0 || supplied.Claims[0].Tier != "hypothesis" || supplied.Claims[0].State != "proposed" {
		t.Fatalf("tier changed: %+v", v)
	}
}

func TestRejectedClientOutput(t *testing.T) {
	for name, mutate := range map[string]func(*Candidate){
		"causal text":        func(c *Candidate) { c.Text = "This was caused by a condition" },
		"causal alternative": func(c *Candidate) { c.Alternatives = []string{"Due to a condition"} },
		"causal proposal":    func(c *Candidate) { c.Measure = "Find the root cause" },
		"missing citation":   func(c *Candidate) { c.Evidence[0].ID = "missing" },
		"wrong scope":        func(c *Candidate) { c.Evidence[0].RunID = "other" },
		"no citation":        func(c *Candidate) { c.Evidence = nil },
		"no alternative":     func(c *Candidate) { c.Alternatives = nil },
		"blank alternative":  func(c *Candidate) { c.Alternatives = []string{" "} },
		"no change":          func(c *Candidate) { c.Change = " " },
		"no measure":         func(c *Candidate) { c.Measure = "" },
		"no counterresult":   func(c *Candidate) { c.Against = "" },
		"oversize":           func(c *Candidate) { c.Text = strings.Repeat("x", MaxOutputBytes) },
	} {
		t.Run(name, func(t *testing.T) {
			client := clientFunc(func(ctx context.Context, in Input) ([]Candidate, error) {
				cs, e := (Mock{}).Propose(ctx, in)
				mutate(&cs[0])
				return cs, e
			})
			out, err := Generate(context.Background(), fixture(), client)
			if err == nil || len(out.Claims) != 0 {
				t.Fatalf("accepted or partial result: %+v %v", out, err)
			}
		})
	}
}

func TestBoundsAndClientIsolation(t *testing.T) {
	doc := fixture()
	called := false
	client := clientFunc(func(_ context.Context, in Input) ([]Candidate, error) {
		called = true
		in.Observations[0].Evidence[0].ID = "forged"
		return (Mock{}).Propose(context.Background(), in)
	})
	if _, err := Generate(context.Background(), doc, client); err == nil || doc.Claims[0].Evidence[0].ID != "event:synthetic" {
		t.Fatal("client changed citation pool")
	}
	doc.Notes = []string{strings.Repeat("x", MaxInputBytes)}
	called = false
	if _, err := Generate(context.Background(), doc, client); err == nil || called {
		t.Fatal("oversize input reached client")
	}
	for _, n := range []int{0, MaxClaims + 1} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			client := clientFunc(func(ctx context.Context, in Input) ([]Candidate, error) {
				cs, e := (Mock{}).Propose(ctx, in)
				out := []Candidate{}
				for i := 0; i < n; i++ {
					out = append(out, cs[0])
				}
				return out, e
			})
			if _, err := Generate(context.Background(), fixture(), client); err == nil {
				t.Fatal("accepted candidate count")
			}
		})
	}
}

func TestNoHypothesisFeedbackOrSources(t *testing.T) {
	doc := fixture()
	doc.Claims[0].Tier = "hypothesis"
	if _, err := Generate(context.Background(), doc, Mock{}); err == nil {
		t.Fatal("used hypothesis as evidence")
	}
	doc = fixture()
	doc.Claims[0].Tier = "association"
	if _, err := Generate(context.Background(), doc, Mock{}); err != nil {
		t.Fatal(err)
	}
	doc = fixture()
	doc.Claims[0].ID = "mock-hypothesis-1"
	if _, err := Generate(context.Background(), doc, Mock{}); err == nil {
		t.Fatal("accepted collision")
	}
}
