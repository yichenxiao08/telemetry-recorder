// Package probetemplate renders instructions; it never runs a command.
package probetemplate

import "strings"

type Context struct {
	Python, Adapter, ImportRoot, Selector string
	Arguments                             []string
}

func Quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func Block(c Context, call string) []string {
	args := append(append([]string{}, c.Arguments...), "-E", "-s", "-B", "-m", "scout_probe")
	return []string{"Push-Location -LiteralPath " + Quote(c.Adapter), "try {", "  & " + Quote(c.Python) + " " + strings.Join(args, " ") + " " + call, "  $probeExit = $LASTEXITCODE", "} finally {", "  Pop-Location", "}"}
}
func Render(c Context) []string {
	if c.Python == "" {
		return []string{"# Select a working Python and adapter before using the probe command template."}
	}
	if c.ImportRoot == "" {
		c.ImportRoot = "<absolute-target-directory>"
	}
	if c.Selector == "" {
		c.Selector = "module:function"
	}
	return Block(c, "--config '<absolute-config-path>' --doctor '<new-probe-doctor-json-path>' --path "+Quote(c.ImportRoot)+" --manifest --drain-timeout 30 "+Quote(c.Selector)+" -- '<target-arguments>'")
}
