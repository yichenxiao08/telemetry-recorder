package probescan

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestEnumeration(t *testing.T) {
	for _, tc := range []struct {
		name        string
		files       int
		bytes       int64
		read, trunc int
	}{{"full", 2000, 20000000, 3, 0}, {"files", 1, 20000000, 1, 2}, {"bytes", 2000, 1, 0, 3}} {
		t.Run(tc.name, func(t *testing.T) {
			d := t.TempDir()
			for _, n := range []string{"z.py", "a.py", "m.py", "vendor/no.py"} {
				p := filepath.Join(d, n)
				_ = os.MkdirAll(filepath.Dir(p), 0700)
				_ = os.WriteFile(p, []byte("pass\n"), 0600)
			}
			o := Options{Root: d, Limits: Limits{tc.files, tc.bytes}}
			p := func(context.Context, []byte) error { return nil }
			a, e := Scan(context.Background(), o, p)
			if e != nil {
				t.Fatal(e)
			}
			b, _ := Scan(context.Background(), o, p)
			x, _ := json.Marshal(a)
			y, _ := json.Marshal(b)
			if string(x) != string(y) || a.Coverage.Read != tc.read || len(a.Truncations) != tc.trunc || len(a.Exclusions) != 1 {
				t.Fatal(a)
			}
		})
	}
}
func TestParseFailureAndExclusion(t *testing.T) {
	o := Options{Root: "testdata", Exclude: []string{"namespace"}, Limits: Limits{2000, 20000000}}
	r, e := Scan(context.Background(), o, func(context.Context, []byte) error { return fmt.Errorf("synthetic unreadable/parser failure") })
	if e != nil || r.Coverage.Complete || len(r.Errors) != r.Coverage.Read || r.Coverage.Parsed != 0 {
		t.Fatal(r, e)
	}
}
func TestEscape(t *testing.T) {
	d := t.TempDir()
	outside := t.TempDir()
	if e := os.Symlink(outside, filepath.Join(d, "escape")); e != nil {
		t.Skip("symlink privilege unavailable: " + e.Error())
	}
	r, e := Scan(context.Background(), Options{Root: d, Limits: Limits{2000, 20000000}}, func(context.Context, []byte) error { t.Fatal("read linked source"); return nil })
	if e != nil || len(r.Exclusions) != 1 {
		t.Fatal(r, e)
	}
}
func TestBounds(t *testing.T) {
	for _, l := range []Limits{{0, 1}, {1, 0}, {2001, 1}, {1, 20000001}} {
		_, e := Scan(context.Background(), Options{Root: "testdata", Limits: l}, nil)
		if e == nil {
			t.Fatal(l)
		}
	}
}
func TestJSONKeys(t *testing.T) {
	r, _ := Scan(context.Background(), Options{Root: "testdata", Limits: Limits{2000, 20000000}}, func(context.Context, []byte) error { return nil })
	b, _ := json.Marshal(r)
	var m map[string]json.RawMessage
	_ = json.Unmarshal(b, &m)
	want := []string{"schema_version", "candidates", "unresolved", "coverage", "exclusions", "errors", "truncations", "limits", "probe_command"}
	if len(m) != len(want) {
		t.Fatal(m)
	}
	for _, k := range want {
		if _, ok := m[k]; !ok {
			t.Fatal(k)
		}
	}
	if !reflect.DeepEqual(r.Candidates, []Candidate{}) {
		t.Fatal(r)
	}
}

func TestUnreadable(t *testing.T) {
	d := t.TempDir()
	_ = os.WriteFile(filepath.Join(d, "denied.py"), []byte("pass"), 0600)
	r, e := Scan(context.Background(), Options{Root: d, Limits: Limits{2000, 20000000}, Open: func(string) (*os.File, error) { return nil, os.ErrPermission }}, func(context.Context, []byte) error { t.Fatal("parsed unreadable file"); return nil })
	if e != nil || len(r.Errors) != 1 || r.Coverage.Read != 0 || r.Coverage.Complete {
		t.Fatal(r, e)
	}
}
