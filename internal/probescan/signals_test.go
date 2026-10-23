package probescan

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func analyzerForTest(t *testing.T) Analyzer {
	t.Helper()
	p, e := exec.LookPath("python")
	if e != nil {
		t.Fatal(e)
	}
	return PythonAnalyzer(p)
}
func TestSignalsProvenance(t *testing.T) {
	a := analyzerForTest(t)
	b, e := os.ReadFile("testdata/signals/positive.py")
	if e != nil {
		t.Fatal(e)
	}
	r, e := a(context.Background(), b)
	if e != nil {
		t.Fatal(e)
	}
	expected := map[string]map[string]int{"partial": {"module_mutable_state": 8, "partial_dict_key": 8, "shared_access": 8}, "cached": {"cache_decorators": 11}, "bounded_cache": {"cache_decorators": 15}, "defaulted": {"mutable_defaults": 19}, "write_shared": {"module_mutable_state": 23, "shared_access": 23}, "cold": {"loop_call_site": 29}, "Top.method": {"mutable_defaults": 33}}
	for name, want := range expected {
		found := false
		for _, c := range r.Items {
			if c.Function != name {
				continue
			}
			found = true
			got := map[string]int{}
			for _, s := range c.Details {
				got[s.Kind] = s.Line
				if s.Fact == "" || s.Tier == "" {
					t.Fatal(s)
				}
				if s.Kind == "partial_dict_key" && s.Tier != "association" {
					t.Fatal(s)
				}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%s got %v want %v", name, got, want)
			}
		}
		if !found {
			t.Fatal("missing", name)
		}
	}
	for _, name := range []string{"asynchronous", "generator", "outer.inner", "Property.value", "unknown.step"} {
		found := false
		for _, c := range r.Items {
			if c.Function == name && c.UnsupportedReason != "" {
				found = true
			}
		}
		if !found {
			t.Fatal("unsupported missing", name)
		}
	}
}
func TestNegativeAndAliasMiss(t *testing.T) {
	a := analyzerForTest(t)
	for _, file := range []string{"negative.py", "alias_miss.py"} {
		b, _ := os.ReadFile(filepath.Join("testdata/signals", file))
		r, e := a(context.Background(), b)
		if e != nil || len(r.Items) != 0 {
			t.Fatal(file, r, e)
		}
	}
}
func TestSelectorsRankingAndStableJSON(t *testing.T) {
	a := analyzerForTest(t)
	o := Options{Root: "testdata/signals", ImportRoot: "testdata/signals", Limit: 2, Limits: Limits{2000, 20000000}, Analyze: a}
	r, e := Scan(context.Background(), o, nil)
	if e != nil {
		t.Fatal(e)
	}
	other, e := Scan(context.Background(), o, nil)
	if e != nil {
		t.Fatal(e)
	}
	x, _ := json.Marshal(r)
	y, _ := json.Marshal(other)
	if string(x) != string(y) {
		t.Fatal("unstable")
	}
	if len(r.Candidates) != 2 || len(r.Unresolved) != 2 || r.Candidates[0].Function != "partial" || r.Candidates[0].Score != 3 || r.Candidates[0].Selector != "positive:partial" {
		t.Fatal(r)
	}
	if !strings.Contains(r.Text(), "post-ranking limit") || !strings.Contains(r.Text(), "no selector") || strings.Contains(r.Text(), "Push-Location") {
		t.Fatal(r.Text())
	}
	o.Limit = 200
	r, e = Scan(context.Background(), o, nil)
	if e != nil {
		t.Fatal(e)
	}
	for _, c := range r.Candidates {
		if c.Score == 1 && c.TieCount < 2 {
			t.Fatal("tie missing")
		}
		if c.Hot && !strings.Contains(c.Advice, "Unsampled calls are unknown") {
			t.Fatal("missing JSON warning", c)
		}
		if !strings.Contains(c.Config, "capture: {}") {
			t.Fatal(c)
		}
	}
}
func TestLayoutSelectors(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{"pkg/__init__.py", "pkg/functions.py", "src/ns/functions.py"} {
		p := filepath.Join(root, path)
		if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(p, nil, 0600); e != nil {
			t.Fatal(e)
		}
	}
	for _, tc := range []struct{ path, importRoot, want string }{{"pkg/functions.py", "", "pkg.functions"}, {"src/ns/functions.py", "", ""}, {"src/ns/functions.py", filepath.Join(root, "src"), "ns.functions"}} {
		got, reason := moduleName(root, filepath.Join(root, tc.path), tc.importRoot)
		if got != tc.want || got == "" && reason == "" {
			t.Fatal(tc, got, reason)
		}
	}
}
func TestAnalysisOutputBounds(t *testing.T) {
	a := analyzerForTest(t)
	src := strings.Repeat("", 0)
	for i := 0; i < 300; i++ {
		src += "def f" + strings.Repeat("x", i) + "(a=[]): pass\n"
	}
	r, e := a(context.Background(), []byte(src))
	if e != nil || len(r.Items) != 256 || r.Truncated != 44 {
		t.Fatal(len(r.Items), r.Truncated, e)
	}
}

func TestCompleteKeyAndPureLoop(t *testing.T) {
	src := []byte("CACHE={}\ndef memo(a,b):\n CACHE[(a,b)]=a+b\n return CACHE[(a,b)]\ndef pure(x): return x*2\nfor i in range(3): pure(i)\n")
	r, e := analyzerForTest(t)(context.Background(), src)
	if e != nil {
		t.Fatal(e)
	}
	for _, c := range r.Items {
		for _, s := range c.Details {
			if s.Kind == "partial_dict_key" || c.Function == "pure" && s.Kind != "loop_call_site" {
				t.Fatal(c)
			}
		}
	}
}
