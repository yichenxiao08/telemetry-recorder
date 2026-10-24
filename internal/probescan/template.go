package probescan

import (
	"github.com/signalscout/signalscout/internal/probetemplate"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CommandTemplate inspects paths only. It never invokes Python or doctor.
func CommandTemplate(r Report, root, importRoot, adapter, python string) []string {
	c := probetemplate.Context{Python: "<selected-python.exe>", Adapter: "<adapter-directory>", ImportRoot: "<absolute-target-directory>", Selector: "module:function"}
	directory := func(p string) string {
		if strings.ContainsAny(p, "\x00\r\n") || p == "" {
			return ""
		}
		a, e := filepath.Abs(p)
		if e != nil {
			return ""
		}
		i, e := os.Lstat(a)
		if e != nil || !i.IsDir() || unsafeLink(i) {
			return ""
		}
		return a
	}
	if a := directory(adapter); a != "" {
		if i, e := os.Lstat(filepath.Join(a, "scout_probe.py")); e == nil && i.Mode().IsRegular() {
			c.Adapter = a
		}
	}
	if !strings.ContainsAny(python, "\x00\r\n") && python != "" {
		if p, e := exec.LookPath(python); e == nil {
			if a, e := filepath.Abs(p); e == nil {
				if i, e := os.Stat(a); e == nil && i.Mode().IsRegular() {
					c.Python = a
				}
			}
		}
	}
	if importRoot != "" {
		if p := directory(importRoot); p != "" {
			c.ImportRoot = p
		}
	} else if len(r.Candidates) > 0 {
		ambiguous := false
		for _, x := range r.Unresolved {
			if strings.Contains(x.UnsupportedReason, "layout") || strings.Contains(x.UnsupportedReason, "import root") {
				ambiguous = true
			}
		}
		// Hidden entries could contain an ambiguous layout, so leave the path unknown.
		if !ambiguous && len(r.Truncations) == 0 && len(r.Errors) == 0 && r.DiagnosticsOmitted == 0 {
			if p := directory(root); p != "" {
				c.ImportRoot = p
			}
		}
	}
	if len(r.Candidates) == 1 && r.Candidates[0].Selector != "" && len(r.Truncations) == 0 && len(r.Errors) == 0 && r.DiagnosticsOmitted == 0 {
		c.Selector = r.Candidates[0].Selector
	}
	return probetemplate.Render(c)
}
