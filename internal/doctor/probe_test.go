package doctor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProbeChecks(t *testing.T) {
	for _, tc := range []struct {
		name, adapter, cwd            string
		helpErr, errorAt              error
		wantImport, wantCWD, wantHelp string
	}{
		{"found both", `{"found":true}`, `{"found":true}`, nil, nil, Pass, Pass, Pass},
		{"missing adapter", `{"found":false}`, `{"found":true}`, nil, nil, Fail, Pass, Pass},
		{"missing cwd", `{"found":true}`, `{"found":false}`, nil, nil, Pass, Warn, Pass},
		{"help nonzero", `{"found":true}`, `{"found":true}`, errors.New("sensitive stderr"), nil, Pass, Pass, Fail},
		{"timeout", ``, ``, context.DeadlineExceeded, context.DeadlineExceeded, Fail, Warn, Fail},
		{"oversized", ``, ``, errors.New("output exceeded limit"), errors.New("output exceeded limit"), Fail, Warn, Fail},
		{"malformed output", `{"found":true} unexpected`, `invalid`, nil, nil, Fail, Warn, Pass},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := fakeEnvironment(t)
			adapter := fakeAdapter(t)
			cwd := t.TempDir()
			e.Cwd = func() (string, error) { return cwd, nil }
			calls := 0
			e.RunIn = func(ctx context.Context, p string, a []string, dir string) (string, error) {
				calls++
				d, ok := ctx.Deadline()
				if !ok || time.Until(d) > 5*time.Second {
					t.Fatal("missing five-second deadline")
				}
				if !filepath.IsAbs(p) {
					t.Fatal("relative interpreter")
				}
				if a[len(a)-1] == "--help" {
					if dir != adapter || !strings.Contains(strings.Join(a, " "), "-E -s -B -m scout_probe --help") {
						t.Fatal("wrong help invocation")
					}
					return "sensitive help output", tc.helpErr
				}
				joined := strings.Join(a, " ")
				if !strings.Contains(joined, "-I -B -c") || !strings.Contains(joined, "importlib.util.find_spec") || strings.Contains(joined, "run_module") {
					t.Fatal("lookup executed module or was not isolated")
				}
				if dir == adapter {
					return tc.adapter, tc.errorAt
				}
				if dir == cwd {
					return tc.cwd, tc.errorAt
				}
				t.Fatal("wrong working directory")
				return "", nil
			}
			r := Inspect(context.Background(), Options{Database: filepath.Join(t.TempDir(), "missing.db"), Adapter: adapter}, e)
			for id, want := range map[string]string{"probe.importable": tc.wantImport, "probe.cwd": tc.wantCWD, "probe.help": tc.wantHelp} {
				c := findCheck(t, r, id)
				if c.Status != want {
					t.Fatalf("%s: %+v", id, c)
				}
				if want != Pass && !strings.Contains(c.Fix, "--doctor") {
					t.Fatal("missing working fix")
				}
			}
			if calls != 3 {
				t.Fatalf("calls=%d", calls)
			}
			if strings.Contains(r.Text(), "sensitive") {
				t.Fatal("diagnostic output leaked")
			}
		})
	}
}

func findCheck(t *testing.T, r Report, id string) Check {
	t.Helper()
	for _, c := range r.Checks {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("missing %s", id)
	return Check{}
}

func TestProbeSchemaAndExistingIDs(t *testing.T) {
	e := fakeEnvironment(t)
	r := Inspect(context.Background(), Options{Database: filepath.Join(t.TempDir(), "missing.db"), Adapter: fakeAdapter(t)}, e)
	want := []string{"binary", "tool.py", "tool.python", "tool.uv", "adapter", "python.recommendation", "python.selected", "dependency.yaml", "dependency.opentelemetry.sdk.trace", "dependency.opentelemetry.exporter.otlp.proto.http.trace_exporter", "dependency.requests", "database", "port", "probe.importable", "probe.cwd", "probe.help"}
	var got []string
	for _, c := range r.Checks {
		got = append(got, c.ID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ids: %v", got)
	}
	b, _ := json.Marshal(r)
	var m map[string]json.RawMessage
	_ = json.Unmarshal(b, &m)
	if len(m) != 6 || string(m["schema_version"]) != "1" {
		t.Fatal("schema changed")
	}
}

func TestWorkingProbeCommands(t *testing.T) {
	for _, args := range [][]string{nil, {"-3"}} {
		p := Python{Command: filepath.Join(t.TempDir(), "Python's $safe.exe"), Arguments: args, Adapter: fakeAdapter(t)}
		r := Report{Python: p, Examples: Examples("signalscout", p, "127.0.0.1:4318")}
		text := r.Text()
		for _, part := range []string{"Working probe command", "Push-Location -LiteralPath", "Pop-Location", "-m scout_probe", "--config", "--doctor", "--path", "--manifest", "--drain-timeout 30", "'module:function' --", "--queue-max-items", "--queue-max-bytes", "drop_invalid", "$LASTEXITCODE", "stderr"} {
			if !strings.Contains(text, part) {
				t.Fatal(part)
			}
		}
		example := strings.Join(Examples("signalscout", p, "127.0.0.1:4318"), "\n")
		if strings.Contains(example, "uv run") || !strings.Contains(example, quote(p.Command)) || !strings.Contains(example, "--doctor (Join-Path $demo 'probe-doctor.json')") {
			t.Fatal(example)
		}
	}
}

func TestReadyProbeReportingDoesNotWrite(t *testing.T) {
	dir := fakeAdapter(t)
	before := map[string]string{}
	for _, name := range []string{"scout_probe.py", "sentinel"} {
		path := filepath.Join(dir, name)
		if name == "sentinel" {
			if err := os.WriteFile(path, []byte("unchanged"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		info, _ := os.Stat(path)
		data, _ := os.ReadFile(path)
		before[name] = string(data) + info.ModTime().String()
	}
	e := fakeEnvironment(t)
	e.Cwd = func() (string, error) { return dir, nil }
	r := Inspect(context.Background(), Options{Database: filepath.Join(dir, "missing.db"), Adapter: dir}, e)
	if r.Summary.Fail != 0 {
		t.Fatal(r.Checks)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 2 {
		t.Fatal("files created", err)
	}
	for _, entry := range entries {
		info, _ := entry.Info()
		data, _ := os.ReadFile(filepath.Join(dir, entry.Name()))
		if before[entry.Name()] != string(data)+info.ModTime().String() {
			t.Fatal("file changed")
		}
	}
}

// Child modes let process tests exercise the real bounded runner with no Python,
// PATH changes, dependency imports, or target execution.
func TestDiagnosticChildProcess(t *testing.T) {
	i := -1
	for n, a := range os.Args {
		if a == "--" {
			i = n
			break
		}
	}
	if i < 0 {
		return
	}
	switch os.Args[i+1] {
	case "cwd":
		dir, _ := os.Getwd()
		fmt.Print(dir)
	case "large":
		fmt.Print(strings.Repeat("x", 32768))
	case "stderr":
		fmt.Fprint(os.Stderr, "sensitive diagnostic")
		fmt.Print("ok")
	case "wait":
		time.Sleep(10 * time.Second)
	default:
		os.Exit(2)
	}
	os.Exit(0)
}

func TestRunInBoundsAndWorkingDirectory(t *testing.T) {
	for _, mode := range []string{"cwd", "large", "stderr", "wait"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			ctx := context.Background()
			if mode == "wait" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
			}
			out, err := runCommandIn(ctx, os.Args[0], []string{"-test.run=^TestDiagnosticChildProcess$", "--", mode}, dir)
			switch mode {
			case "cwd":
				if err != nil || out != dir {
					t.Fatalf("cwd %q: %v", out, err)
				}
			case "large", "wait":
				if err == nil || out != "" {
					t.Fatalf("unbounded/timeout output %q %v", out, err)
				}
			case "stderr":
				if err != nil || out != "ok" {
					t.Fatal(out, err)
				}
			}
			entries, _ := os.ReadDir(dir)
			if len(entries) != 0 {
				t.Fatal("process check wrote files")
			}
		})
	}
}
