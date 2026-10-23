// Package doctor reports local setup readiness without opening a telemetry store.
package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const (
	Pass         = "PASS"
	Warn         = "WARN"
	Fail         = "FAIL"
	PinnedPython = "3.12.12"
)

type Options struct{ Database, Adapter, Python, Listen string }

// Environment makes machine checks independently testable without changing PATH.
type Environment struct {
	Lookup     func(string) (string, error)
	Run        func(context.Context, string, []string) (string, error)
	RunIn      func(context.Context, string, []string, string) (string, error)
	Cwd        func() (string, error)
	Executable func() (string, error)
	Version    string
	Stat       func(string) (os.FileInfo, error)
	Writable   func(string) error
	Port       func(context.Context, string) error
}

type Check struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Fix     string `json:"fix"`
}
type Binary struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}
type Python struct {
	Command        string   `json:"command"`
	Arguments      []string `json:"arguments"`
	Version        string   `json:"version"`
	PinnedVersion  string   `json:"pinned_version"`
	Recommendation string   `json:"recommendation"`
	Adapter        string   `json:"adapter"`
}
type Summary struct {
	Pass int `json:"pass"`
	Warn int `json:"warn"`
	Fail int `json:"fail"`
}
type Report struct {
	SchemaVersion int      `json:"schema_version"`
	Binary        Binary   `json:"binary"`
	Python        Python   `json:"python"`
	Checks        []Check  `json:"checks"`
	Summary       Summary  `json:"summary"`
	Examples      []string `json:"examples"`
}

type dependency struct{ Module, Package string }

func dependencies() []dependency {
	return []dependency{
		{"yaml", "PyYAML==6.0.3"},
		{"opentelemetry.sdk.trace", "opentelemetry-sdk==1.28.2"},
		{"opentelemetry.exporter.otlp.proto.http.trace_exporter", "opentelemetry-exporter-otlp-proto-http==1.28.2"},
		{"requests", "requests==2.34.2"},
	}
}

func venvPython(adapter string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(adapter, ".venv", "Scripts", "python.exe")
	}
	return filepath.Join(adapter, ".venv", "bin", "python")
}

// Inspect does not install packages, open SQLite, start serve, or write files.
// A new database's creation permissions remain explicitly untested.
func Inspect(ctx context.Context, o Options, e Environment) Report {
	adapter, _ := filepath.Abs(o.Adapter)
	r := Report{SchemaVersion: 1, Python: Python{Arguments: []string{}, PinnedVersion: PinnedPython, Adapter: adapter,
		Recommendation: "Adapter README recommends uv with CPython 3.12.12 and --frozen (uv.lock)."}, Checks: []Check{}}
	add := func(id, status, message, fix string) {
		r.Checks = append(r.Checks, Check{id, status, message, fix})
		switch status {
		case Pass:
			r.Summary.Pass++
		case Warn:
			r.Summary.Warn++
		case Fail:
			r.Summary.Fail++
		}
	}
	r.Binary.Version = e.Version
	path, err := e.Executable()
	if err != nil {
		add("binary", Fail, "Could not resolve the running binary path.", "Rebuild with task build.")
	} else {
		r.Binary.Path = path
		status := Pass
		if e.Version == "" {
			r.Binary.Version = "unknown"
			status = Warn
		}
		add("binary", status, r.Binary.Version+"; "+path, "")
	}
	available := map[string]string{}
	for _, name := range []string{"py", "python", "uv"} {
		p, lookupErr := e.Lookup(name)
		if lookupErr != nil {
			add("tool."+name, Warn, name+" is not on PATH.", "")
			continue
		}
		available[name] = p
		args := []string{"--version"}
		if name == "py" {
			args = []string{"-3", "--version"}
		}
		out, runErr := e.Run(ctx, p, args)
		version := parseVersion(out, name == "uv")
		if runErr != nil || version == "" {
			add("tool."+name, Warn, name+" exists at "+p+"; its version could not be checked.", "")
		} else {
			add("tool."+name, Pass, name+" "+version+"; "+p, "")
		}
	}
	if info, statErr := e.Stat(filepath.Join(adapter, "scout_probe.py")); statErr != nil || !info.Mode().IsRegular() {
		add("adapter", Fail, "Probe adapter scout_probe.py is missing or inaccessible: "+adapter, "Pass --adapter <directory> for this repository's adapters/provenance-python.")
	} else {
		add("adapter", Pass, "Probe adapter file is present: "+adapter+"; lookup and help are checked separately.", "")
	}
	selected := o.Python
	if selected != "" {
		selected, err = e.Lookup(selected)
		if err != nil {
			selected = ""
		}
	} else if info, statErr := e.Stat(venvPython(adapter)); statErr == nil && info.Mode().IsRegular() {
		selected = venvPython(adapter)
	} else if p := available["py"]; p != "" {
		selected = p
		r.Python.Arguments = []string{"-3"}
	} else {
		selected = available["python"]
	}
	selected = absolutePython(selected)
	r.Python.Command = selected
	add("python.recommendation", Pass, r.Python.Recommendation, "uv sync --project "+quote(adapter)+" --python "+PinnedPython+" --frozen")
	ready := false
	if selected == "" {
		add("python.selected", Fail, "No usable Python command was selected.", "Install the README's pinned environment or pass --python <executable>.")
	} else {
		out, runErr := e.Run(ctx, selected, appendArgs(r.Python.Arguments, "--version"))
		r.Python.Version = parseVersion(out, false)
		if runErr != nil || r.Python.Version == "" {
			add("python.selected", Fail, "Selected interpreter could not report a Python version.", "Use --python <executable> for a working Python 3.12 interpreter.")
		} else {
			ready = true
			status := Pass
			message := "Selected " + selected + " " + strings.Join(r.Python.Arguments, " ") + "; Python " + r.Python.Version
			if r.Python.Version != PinnedPython {
				status = Warn
				message += "; differs from adapter pin " + PinnedPython
			}
			add("python.selected", status, message, "")
		}
	}
	imports := map[string]bool{}
	if ready {
		out, runErr := e.Run(ctx, selected, appendArgs(r.Python.Arguments, "-I", "-B", "-c", importScript()))
		if runErr == nil {
			if err := json.Unmarshal([]byte(out), &imports); err != nil {
				imports = map[string]bool{}
			}
		}
	}
	for _, d := range dependencies() {
		if ready && imports[d.Module] {
			add("dependency."+d.Module, Pass, d.Module+" imports in the selected Python.", "")
		} else {
			fix := "Select a working Python first."
			if selected != "" {
				prefix := "& " + quote(selected) + " " + strings.Join(r.Python.Arguments, " ")
				fix = prefix + " -m pip install " + quote(d.Package)
				if !imports["pip"] {
					fix = prefix + " -m ensurepip; " + fix
				}
			}
			add("dependency."+d.Module, Fail, d.Module+" is missing or could not be imported; diagnostic text is withheld.", fix)
		}
	}
	status, message := databaseCheck(o.Database, e)
	add("database", status, message, "")
	if o.Listen == "" {
		add("port", Warn, "No port requested; availability was not checked.", "")
	} else if !validAddress(o.Listen) {
		add("port", Fail, "Use a loopback IP or localhost and a numeric port in 1..65535.", "Choose the same loopback --listen address for doctor and serve.")
	} else if err = e.Port(ctx, o.Listen); err != nil {
		add("port", Fail, "Could not bind "+o.Listen+"; it may be busy, invalid, or disallowed.", "Choose a free --listen address for doctor and serve.")
	} else {
		add("port", Pass, o.Listen+" could be bound and was immediately released; availability can change.", "")
	}
	probeChecks(ctx, r.Python, ready, e, add)
	r.Examples = Examples(r.Binary.Path, r.Python, o.Listen)
	return r
}

func validAddress(address string) bool {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func databaseCheck(path string, e Environment) (string, string) {
	p, err := filepath.Abs(path)
	if err != nil || path == "" {
		return Fail, "Database path is invalid."
	}
	info, err := e.Stat(p)
	if err == nil {
		if !info.Mode().IsRegular() {
			return Fail, "Database path is not a regular file: " + p
		}
		if e.Writable(p) != nil {
			return Fail, "Existing database file cannot be opened read/write: " + p
		}
		return Pass, "Existing file opens read/write without changing contents: " + p + "; SQLite format, locks and WAL permissions are not tested."
	}
	if !os.IsNotExist(err) {
		return Fail, "Cannot inspect database path: " + p
	}
	parent, err := e.Stat(filepath.Dir(p))
	if err != nil || !parent.IsDir() {
		return Fail, "Database parent directory is missing or inaccessible: " + filepath.Dir(p)
	}
	return Warn, "Database does not exist; parent directory exists: " + p + ". No file was created; create/write permission, SQLite and WAL readiness remain untested."
}

func parseVersion(out string, uv bool) string {
	pattern := `(?m)^Python ([0-9]+\.[0-9]+\.[0-9]+)(?:\s|$)`
	if uv {
		pattern = `(?m)^uv ([0-9]+\.[0-9]+\.[0-9]+)(?:\s|$)`
	}
	m := regexp.MustCompile(pattern).FindStringSubmatch(out)
	if len(m) != 2 {
		return ""
	}
	return m[1]
}
func appendArgs(prefix []string, args ...string) []string {
	return append(append([]string{}, prefix...), args...)
}
func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func importScript() string {
	return `import contextlib, importlib, json, os
results = {}
with open(os.devnull, 'w') as sink, contextlib.redirect_stdout(sink), contextlib.redirect_stderr(sink):
    for name in ['yaml', 'opentelemetry.sdk.trace', 'opentelemetry.exporter.otlp.proto.http.trace_exporter', 'requests', 'pip']:
        try:
            importlib.import_module(name)
            results[name] = True
        except BaseException:
            results[name] = False
print(json.dumps(results))`
}

// Examples are instructions only. No target or probe is executed by doctor.
func Examples(binary string, python Python, address string) []string {
	if binary == "" {
		binary = "signalscout"
	}
	if !validAddress(address) {
		address = "127.0.0.1:4318"
	}
	lines := []string{"Working probe command (template; replace placeholders and fix failed checks first):"}
	lines = append(lines, WorkingProbeCommand(python)...)
	lines = append(lines,
		"# PowerShell may show probe stderr as a red error; judge $probeExit and the printed probe summary.",
		"# For hot functions, --queue-max-items and --queue-max-bytes are optional and must be supplied together; they do not guarantee delivery.",
		"# Complex return objects can fail capture (drop_invalid); explicitly allowlisted scalars are safer. Help/import checks do not prove capture.",
		"PowerShell example:",
		"# Synthetic example only. Run these yourself; doctor ran none of them.",
		"$demo = Join-Path ([IO.Path]::GetTempPath()) ('signalscout-example-' + [guid]::NewGuid())",
		"New-Item -ItemType Directory -Path $demo | Out-Null",
		"$db = Join-Path $demo 'example.db'",
		"New-Item -ItemType File -Path $db | Out-Null",
		"Set-Content -Path (Join-Path $demo 'scout_example.py') -Encoding ascii -Value 'def main(): return 1 + 1'",
		"@'\nversion: 1\nsynthetic_only: true\nendpoint: http://"+address+"\nservice: synthetic.doctor\nprobes:\n  - module: scout_example\n    function: main\n    capture:\n      return: true\n'@ | Set-Content -Path (Join-Path $demo 'signalscout.yaml') -Encoding ascii",
		"# Copy the next command's OUTPUT into a second PowerShell window; leave serve running:",
		"Write-Output ("+quote("& "+quote(binary)+" --db '")+" + $db.Replace(\"'\", \"''\") + "+quote("' serve --listen "+quote(address)+" --retention 0")+")",
		"# Back in this window after serve is listening, use the selected Python:",
	)
	if python.Command != "" {
		lines = append(lines, probeBlock(python, "--config (Join-Path $demo 'signalscout.yaml') --doctor (Join-Path $demo 'probe-doctor.json') --path $demo --manifest --drain-timeout 30 scout_example:main --")...)
	} else {
		lines = append(lines, "# Select a working Python before launching this synthetic example.")
	}
	return append(lines,
		"& "+quote(binary)+" --db $db runs list",
		"# These checks establish dependency/path readiness only, not successful probe capture or analysis.",
	)
}

func (r Report) Text() string {
	var b strings.Builder
	fmt.Fprintln(&b, "Signal Scout doctor (schema 1)")
	for _, c := range r.Checks {
		fmt.Fprintf(&b, "%s %s: %s\n", c.Status, c.ID, c.Message)
		if c.Fix != "" {
			fmt.Fprintf(&b, "  Next: %s\n", c.Fix)
		}
	}
	fmt.Fprintf(&b, "Summary: %d PASS, %d WARN, %d FAIL\n", r.Summary.Pass, r.Summary.Warn, r.Summary.Fail)
	fmt.Fprintln(&b, "Checks do not prove probe capture or analysis; no server or recording was started.")
	for _, line := range r.Examples {
		fmt.Fprintln(&b, line)
	}
	return b.String()
}
