package doctor

import (
	"context"
	"encoding/json"
	"github.com/signalscout/signalscout/internal/probetemplate"
	"path/filepath"
	"strings"
	"time"
)

func probeChecks(ctx context.Context, p Python, ready bool, e Environment, add func(string, string, string, string)) {
	fix := strings.Join(WorkingProbeCommand(p), "\n")
	lookup := func(dir string) bool {
		if !ready {
			return false
		}
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		out, err := e.RunIn(checkCtx, p.Command, appendArgs(p.Arguments, "-I", "-B", "-c", probeSpecScript()), dir)
		var result struct {
			Found bool `json:"found"`
		}
		return err == nil && json.Unmarshal([]byte(out), &result) == nil && result.Found
	}
	if lookup(p.Adapter) {
		add("probe.importable", Pass, "scout_probe is findable from the adapter directory without importing it; capture is untested.", "")
	} else {
		add("probe.importable", Fail, "scout_probe could not be located from the adapter directory; lookup may have failed or timed out. Diagnostic output is withheld.", fix)
	}
	cwd, err := e.Cwd()
	if err != nil {
		add("probe.cwd", Warn, "INFO: current working directory could not be determined; probe lookup there is unknown.", fix)
	} else if lookup(cwd) {
		add("probe.cwd", Pass, "INFO: scout_probe is also findable from the current working directory: "+cwd, "")
	} else {
		add("probe.cwd", Warn, "INFO: scout_probe is not findable from the current working directory, or lookup failed: "+cwd+". Use the adapter directory.", fix)
	}
	if ready {
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_, err = e.RunIn(checkCtx, p.Command, appendArgs(p.Arguments, "-E", "-s", "-B", "-m", "scout_probe", "--help"), p.Adapter)
		if err == nil {
			add("probe.help", Pass, "scout_probe --help exited zero from the adapter directory; no target was run and capture is untested.", "")
			return
		}
	}
	add("probe.help", Fail, "scout_probe --help failed, timed out or exceeded the output bound; diagnostic output is withheld.", fix)
}

func probeSpecScript() string {
	// -I excludes CWD. Add only the directory explicitly being checked, matching
	// -m's directory visibility without relying on PYTHONPATH or user site hooks.
	return `import importlib.util, json, os, sys
sys.path.insert(0, os.getcwd())
print(json.dumps({"found": importlib.util.find_spec("scout_probe") is not None}))`
}

func templateContext(p Python) probetemplate.Context {
	return probetemplate.Context{Python: p.Command, Adapter: p.Adapter, Arguments: p.Arguments}
}
func probeBlock(p Python, call string) []string { return probetemplate.Block(templateContext(p), call) }
func WorkingProbeCommand(p Python) []string     { return probetemplate.Render(templateContext(p)) }

func absolutePython(path string) string {
	if path == "" {
		return ""
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	return absolute
}
