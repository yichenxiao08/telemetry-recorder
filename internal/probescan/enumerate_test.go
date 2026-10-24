package probescan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type repeatedDirectory struct {
	entry            os.DirEntry
	calls, remaining int
	closed           bool
}

func (d *repeatedDirectory) ReadDir(n int) ([]os.DirEntry, error) {
	if n != 1 {
		panic("unbounded directory read")
	}
	d.calls++
	if d.remaining == 0 {
		return nil, io.EOF
	}
	d.remaining--
	return []os.DirEntry{d.entry}, nil
}
func (d *repeatedDirectory) Close() error { d.closed = true; return nil }
func TestEnumerationStopsBeforeHugeFlatAllocation(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "ordinary.txt"), nil, 0600)
	entries, _ := os.ReadDir(root)
	var last *repeatedDirectory
	o := Options{Root: root, Limits: Limits{2000, 20000000}, Enumeration: EnumerationLimits{MaxEntries: 7}, readDirectory: func(string) (directoryReader, error) {
		last = &repeatedDirectory{entry: entries[0], remaining: 1000000}
		return last, nil
	}}
	run := func() Report {
		r, e := Scan(context.Background(), o, func(context.Context, []byte) error { t.Fatal("partial source admitted"); return nil })
		if e != nil {
			t.Fatal(e)
		}
		if last.calls != 8 || !last.closed || r.Enumeration.Entries != 0 || r.Coverage.EligibleKnown || r.Coverage.Complete || len(r.Truncations) != 1 {
			t.Fatal(r, last)
		}
		return r
	}
	a, _ := json.Marshal(run())
	b, _ := json.Marshal(run())
	if string(a) != string(b) {
		t.Fatal("not deterministic")
	}
}
func TestEnumerationBudgets(t *testing.T) {
	for _, kind := range []string{"entries", "bytes", "depth"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			_ = os.WriteFile(filepath.Join(root, "a.py"), []byte("pass"), 0600)
			_ = os.MkdirAll(filepath.Join(root, "b", "c", "d"), 0700)
			for i := 0; i < 10; i++ {
				_ = os.WriteFile(filepath.Join(root, fmt.Sprintf("z%d.txt", i)), nil, 0600)
			}
			l := EnumerationLimits{}
			switch kind {
			case "entries":
				l.MaxEntries = 5
			case "bytes":
				l.MaxPathBytes = len(root) + 2
			case "depth":
				l.MaxDepth = 1
			}
			r, e := Scan(context.Background(), Options{Root: root, Limits: Limits{2000, 20000000}, Enumeration: l}, func(context.Context, []byte) error { t.Fatal("source admitted on overflow"); return nil })
			if e != nil || r.Coverage.Read != 0 || r.Coverage.EligibleKnown || r.Enumeration.Complete || len(r.Truncations) != 1 {
				t.Fatal(r, e)
			}
		})
	}
}
func TestDiagnosticOverflowAndCancellation(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 8; i++ {
		_ = os.WriteFile(filepath.Join(root, fmt.Sprintf("%d.py", i)), []byte("pass"), 0600)
	}
	o := Options{Root: root, Limits: Limits{2000, 20000000}, Enumeration: EnumerationLimits{MaxDiagnostics: 2}}
	r, e := Scan(context.Background(), o, func(context.Context, []byte) error { return fmt.Errorf("%s", strings.Repeat("x", 10000)) })
	if e != nil || len(r.Errors) != 2 || r.DiagnosticsOmitted != 6 || r.Coverage.Complete || len(r.Errors[0].Reason) > 1030 {
		t.Fatal(r, e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, e = Scan(ctx, o, func(context.Context, []byte) error { t.Fatal("parsed after cancel"); return nil })
	if e != nil || r.Coverage.EligibleKnown || r.Coverage.Complete || r.Coverage.Read != 0 {
		t.Fatal(r, e)
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	r, e = Scan(ctx, o, func(context.Context, []byte) error { calls++; cancel(); return nil })
	if e != nil || calls != 1 || r.Coverage.Complete || r.Coverage.Parsed != 0 {
		t.Fatal(r, e, calls)
	}
}
func TestGlobalOrdinalSourceOrder(t *testing.T) {
	root := t.TempDir()
	names := []string{"a.py", "a/z.py", "a-1.py", "z.py"}
	for _, p := range names {
		_ = os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0700)
		_ = os.WriteFile(filepath.Join(root, p), []byte(p), 0600)
	}
	var got []string
	r, e := Scan(context.Background(), Options{Root: root, Limits: Limits{2000, 20000000}}, func(_ context.Context, b []byte) error { got = append(got, string(b)); return nil })
	if e != nil || !r.Coverage.Complete || !reflect.DeepEqual(got, []string{"a-1.py", "a.py", "a/z.py", "z.py"}) {
		t.Fatal(r, e, got)
	}
}
func TestEnumerationLimitValidation(t *testing.T) {
	for _, l := range []EnumerationLimits{{MaxEntries: -1}, {MaxEntries: 200001}, {MaxPathBytes: 33554433}, {MaxDepth: 129}, {MaxDiagnostics: 10001}} {
		if _, e := enumerationDefaults(l); e == nil {
			t.Fatal(l)
		}
	}
}

type cancelDirectory struct {
	directoryReader
	cancel context.CancelFunc
}

func (d cancelDirectory) ReadDir(n int) ([]os.DirEntry, error) {
	v, e := d.directoryReader.ReadDir(n)
	d.cancel()
	return v, e
}
func TestCancelDuringDirectoryRead(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "a.py"), nil, 0600)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o := Options{Root: root, Limits: Limits{2000, 20000000}, readDirectory: func(p string) (directoryReader, error) { f, e := os.Open(p); return cancelDirectory{f, cancel}, e }}
	r, e := Scan(ctx, o, func(context.Context, []byte) error { t.Fatal("parsed canceled enumeration"); return nil })
	if e != nil || r.Coverage.Complete || r.Coverage.EligibleKnown || len(r.Truncations) != 1 {
		t.Fatal(r, e)
	}
}

type orderedDirectory struct{ entries []os.DirEntry }

func (d *orderedDirectory) ReadDir(n int) ([]os.DirEntry, error) {
	if n != 1 {
		panic("unbounded read")
	}
	if len(d.entries) == 0 {
		return nil, io.EOF
	}
	e := d.entries[0]
	d.entries = d.entries[1:]
	return []os.DirEntry{e}, nil
}
func (d *orderedDirectory) Close() error { return nil }
func TestPathBudgetReportIgnoresDirectoryOrder(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "a.py"), nil, 0600)
	_ = os.WriteFile(filepath.Join(root, "long_filename.py"), nil, 0600)
	entries, _ := os.ReadDir(root)
	outputs := []string{}
	for _, order := range [][]os.DirEntry{{entries[0], entries[1]}, {entries[1], entries[0]}} {
		o := Options{Root: root, Limits: Limits{2000, 20000000}, Enumeration: EnumerationLimits{MaxPathBytes: len(root) + 10}, readDirectory: func(string) (directoryReader, error) {
			return &orderedDirectory{append([]os.DirEntry{}, order...)}, nil
		}}
		r, e := Scan(context.Background(), o, func(context.Context, []byte) error { t.Fatal("read partial enumeration"); return nil })
		if e != nil {
			t.Fatal(e)
		}
		b, _ := json.Marshal(r)
		outputs = append(outputs, string(b))
	}
	if outputs[0] != outputs[1] {
		t.Fatal(outputs)
	}
}
