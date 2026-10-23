package doctor

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fakeEnvironment(t *testing.T) Environment {
	t.Helper()
	dir := t.TempDir()
	return Environment{
		Lookup: func(name string) (string, error) { return name, nil },
		Cwd:    func() (string, error) { return dir, nil },
		RunIn:  func(context.Context, string, []string, string) (string, error) { return `{"found":true}`, nil },
		Run: func(_ context.Context, name string, args []string) (string, error) {
			if args[len(args)-1] == "--version" {
				if name == "uv" {
					return "uv 0.13.3", nil
				}
				return "Python 3.12.12", nil
			}
			return `{"yaml":true,"opentelemetry.sdk.trace":true,"opentelemetry.exporter.otlp.proto.http.trace_exporter":true,"requests":true}`, nil
		},
		Executable: func() (string, error) { return filepath.Join(dir, "signalscout.exe"), nil },
		Version:    "test-revision",
		Stat:       os.Stat,
		Writable:   func(string) error { return nil },
		Port:       func(context.Context, string) error { return nil },
	}
}

func fakeAdapter(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "scout_probe.py"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestChecks(t *testing.T) {
	for _, tc := range []struct {
		name, id, status string
		change           func(*Environment, *Options)
	}{
		{"binary", "binary", Pass, func(*Environment, *Options) {}},
		{"adapter absent", "adapter", Fail, func(_ *Environment, o *Options) { o.Adapter = filepath.Join(t.TempDir(), "absent") }},
		{"binary failure", "binary", Fail, func(e *Environment, _ *Options) {
			e.Executable = func() (string, error) { return "", errors.New("private diagnostic") }
		}},
		{"missing py", "tool.py", Warn, func(e *Environment, _ *Options) {
			e.Lookup = func(n string) (string, error) {
				if n == "py" {
					return "", os.ErrNotExist
				}
				return n, nil
			}
		}},
		{"missing python", "tool.python", Warn, func(e *Environment, _ *Options) {
			e.Lookup = func(n string) (string, error) {
				if n == "python" {
					return "", os.ErrNotExist
				}
				return n, nil
			}
		}},
		{"missing uv", "tool.uv", Warn, func(e *Environment, _ *Options) {
			e.Lookup = func(n string) (string, error) {
				if n == "uv" {
					return "", os.ErrNotExist
				}
				return n, nil
			}
		}},
		{"version mismatch", "python.selected", Warn, func(e *Environment, _ *Options) {
			e.Run = func(context.Context, string, []string) (string, error) { return "Python 3.14.0", nil }
		}},
		{"broken python", "python.selected", Fail, func(e *Environment, _ *Options) {
			e.Run = func(context.Context, string, []string) (string, error) { return "secret", os.ErrPermission }
		}},
		{"no python", "python.selected", Fail, func(e *Environment, _ *Options) {
			e.Lookup = func(string) (string, error) { return "", os.ErrNotExist }
		}},
		{"missing yaml", "dependency.yaml", Fail, func(e *Environment, _ *Options) {
			old := e.Run
			e.Run = func(c context.Context, n string, a []string) (string, error) {
				if a[len(a)-1] != "--version" {
					return `{}`, nil
				}
				return old(c, n, a)
			}
		}},
		{"bad import response", "dependency.requests", Fail, func(e *Environment, _ *Options) {
			old := e.Run
			e.Run = func(c context.Context, n string, a []string) (string, error) {
				if a[len(a)-1] != "--version" {
					return "private exception", nil
				}
				return old(c, n, a)
			}
		}},
		{"db absent", "database", Warn, func(*Environment, *Options) {}},
		{"db missing parent", "database", Fail, func(_ *Environment, o *Options) { o.Database = filepath.Join(o.Database, "absent", "db") }},
		{"db existing writable", "database", Pass, func(_ *Environment, o *Options) {
			if err := os.WriteFile(o.Database, []byte("sentinel"), 0600); err != nil {
				t.Fatal(err)
			}
		}},
		{"db existing denied", "database", Fail, func(e *Environment, o *Options) {
			if err := os.WriteFile(o.Database, []byte("sentinel"), 0600); err != nil {
				t.Fatal(err)
			}
			e.Writable = func(string) error { return os.ErrPermission }
		}},
		{"db directory", "database", Fail, func(_ *Environment, o *Options) { o.Database = filepath.Dir(o.Database) }},
		{"port free", "port", Pass, func(*Environment, *Options) {}},
		{"port busy", "port", Fail, func(e *Environment, _ *Options) {
			e.Port = func(context.Context, string) error { return errors.New("busy secret") }
		}},
		{"port omitted", "port", Warn, func(_ *Environment, o *Options) { o.Listen = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := fakeEnvironment(t)
			o := Options{Database: filepath.Join(t.TempDir(), "new.db"), Adapter: fakeAdapter(t), Listen: "127.0.0.1:4318"}
			tc.change(&e, &o)
			r := Inspect(context.Background(), o, e)
			found := false
			for _, c := range r.Checks {
				if c.ID == tc.id {
					found = true
					if c.Status != tc.status {
						t.Fatalf("%s: %+v", tc.id, c)
					}
				}
			}
			if !found {
				t.Fatalf("missing %s", tc.id)
			}
			b, _ := json.Marshal(r)
			if strings.Contains(string(b), "private") || strings.Contains(string(b), "secret") {
				t.Fatal("subprocess/error detail leaked")
			}
		})
	}
}

func TestReportingDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	keep := filepath.Join(dir, "keep")
	if err := os.WriteFile(keep, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(keep)
	if err != nil {
		t.Fatal(err)
	}
	e := SystemEnvironment()
	e.Lookup = func(string) (string, error) { return "", os.ErrNotExist }
	e.Port = func(context.Context, string) error { return nil }
	Inspect(context.Background(), Options{Database: filepath.Join(dir, "missing.db"), Adapter: filepath.Join(dir, "adapter")}, e)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(keep)
	b, _ := os.ReadFile(keep)
	if len(entries) != 1 || string(b) != "unchanged" || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("reporting modified filesystem")
	}
	if err := e.Writable(keep); err != nil {
		t.Fatal(err)
	}
	after, _ = os.Stat(keep)
	b, _ = os.ReadFile(keep)
	if string(b) != "unchanged" || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("write-access check modified file")
	}
}

func TestStableSchema(t *testing.T) {
	r := Inspect(context.Background(), Options{Database: filepath.Join(t.TempDir(), "new.db"), Adapter: fakeAdapter(t)}, fakeEnvironment(t))
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err = json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"schema_version", "binary", "python", "checks", "summary", "examples"} {
		if _, ok := m[key]; !ok {
			t.Fatal(key)
		}
	}
	if len(m) != 6 || string(m["schema_version"]) != "1" {
		t.Fatal(string(b))
	}
	for _, tc := range []struct {
		key   string
		names []string
	}{
		{"binary", []string{"path", "version"}},
		{"python", []string{"command", "arguments", "version", "pinned_version", "recommendation", "adapter"}},
		{"summary", []string{"pass", "warn", "fail"}},
	} {
		var nested map[string]json.RawMessage
		if err = json.Unmarshal(m[tc.key], &nested); err != nil {
			t.Fatal(err)
		}
		if len(nested) != len(tc.names) {
			t.Fatal(tc.key)
		}
		for _, name := range tc.names {
			if _, ok := nested[name]; !ok {
				t.Fatal(tc.key, name)
			}
		}
	}
	var check map[string]json.RawMessage
	cb, _ := json.Marshal(r.Checks[0])
	_ = json.Unmarshal(cb, &check)
	if len(check) != 4 {
		t.Fatal(string(cb))
	}
	for _, key := range []string{"id", "status", "message", "fix"} {
		if _, ok := check[key]; !ok {
			t.Fatal(key)
		}
	}
	if r.Summary.Fail != 0 || !reflect.DeepEqual(r.Examples, Examples(r.Binary.Path, r.Python, "127.0.0.1:4318")) {
		t.Fatal("unexpected summary/examples")
	}
}

func TestEveryDependencyFailure(t *testing.T) {
	for _, missing := range dependencies() {
		t.Run(missing.Module, func(t *testing.T) {
			e := fakeEnvironment(t)
			old := e.Run
			e.Run = func(ctx context.Context, p string, a []string) (string, error) {
				if a[len(a)-1] == "--version" {
					return old(ctx, p, a)
				}
				values := map[string]bool{"pip": true}
				for _, d := range dependencies() {
					values[d.Module] = d.Module != missing.Module
				}
				b, _ := json.Marshal(values)
				return string(b), nil
			}
			r := Inspect(context.Background(), Options{Database: filepath.Join(t.TempDir(), "new.db"), Adapter: fakeAdapter(t)}, e)
			for _, c := range r.Checks {
				if c.ID == "dependency."+missing.Module && (c.Status != Fail || !strings.Contains(c.Fix, " -3 -m pip install '"+missing.Package+"'")) {
					t.Fatalf("bad fix: %+v", c)
				}
			}
			if r.Summary.Fail != 1 {
				t.Fatalf("%+v", r.Summary)
			}
		})
	}
}

func TestVersionAndAddressValidation(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		uv          bool
	}{
		{"Python 3.12.12\n", "3.12.12", false}, {"Python 3.14.0\nsecret", "3.14.0", false},
		{"uv 0.13.3 (build)", "0.13.3", true}, {"private invalid output", "", false},
	} {
		if got := parseVersion(tc.input, tc.uv); got != tc.want {
			t.Fatalf("%q != %q", got, tc.want)
		}
	}
	for _, tc := range []struct {
		address string
		valid   bool
	}{
		{"127.0.0.1:4318", true}, {"[::1]:4318", true}, {"localhost:4318", true},
		{"127.0.0.1:0", false}, {"127.0.0.1:65536", false}, {"invalid", false}, {"0.0.0.0:4318", false}, {"example.com:4318", false},
	} {
		if validAddress(tc.address) != tc.valid {
			t.Fatal(tc.address)
		}
	}
}

func TestSelectedInterpreterAndImports(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(map[bool]string{false: "venv", true: "explicit"}[explicit], func(t *testing.T) {
			e := fakeEnvironment(t)
			adapter := fakeAdapter(t)
			venv := venvPython(adapter)
			if err := os.MkdirAll(filepath.Dir(venv), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(venv, nil, 0600); err != nil {
				t.Fatal(err)
			}
			o := Options{Database: filepath.Join(t.TempDir(), "new.db"), Adapter: adapter}
			if explicit {
				o.Python = "chosen"
			}
			old := e.Run
			var used string
			e.Run = func(c context.Context, n string, a []string) (string, error) {
				if a[len(a)-1] != "--version" {
					used = n
					if !strings.Contains(strings.Join(a, " "), "-I -B -c") {
						t.Fatal("imports not isolated/no-bytecode")
					}
				}
				return old(c, n, a)
			}
			r := Inspect(context.Background(), o, e)
			want := venv
			if explicit {
				want = absolutePython("chosen")
			}
			if used != want || r.Python.Command != want {
				t.Fatalf("got %q, want %q", used, want)
			}
		})
	}
}

func TestDependencyFixesAndPin(t *testing.T) {
	readme, err := os.ReadFile("../../adapters/provenance-python/README.md")
	if err != nil {
		t.Fatal(err)
	}
	project, err := os.ReadFile("../../adapters/provenance-python/pyproject.toml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), PinnedPython) || !strings.Contains(string(project), `requires-python = "==3.12.*"`) {
		t.Fatal("doctor recommendation drift")
	}
	for _, d := range dependencies() {
		if !strings.Contains(d.Package, "==") {
			t.Fatal("unversioned fix")
		}
	}
}

func TestOutputBound(t *testing.T) {
	var output boundedOutput
	input := strings.Repeat("x", 32768)
	n, err := output.Write([]byte(input))
	if err != nil || n != len(input) || output.text.Len() != 16384 || !output.overflow {
		t.Fatal("output not bounded")
	}
	n, err = output.Write([]byte(input))
	if err != nil || n != len(input) || output.text.Len() != 16384 {
		t.Fatal("overflow not drained")
	}
}
