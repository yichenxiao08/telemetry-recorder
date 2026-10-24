package main

import (
	"encoding/json"
	"fmt"
	"github.com/signalscout/signalscout/internal/probescan"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
)

func newProbeCommand() *cobra.Command {
	p := &cobra.Command{Use: "probe"}
	var o probescan.Options
	var output, adapter string
	c := &cobra.Command{Use: "scan", Short: "Suggest functions worth probing from Python syntax", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		for _, name := range []string{"max-entries", "max-path-bytes", "max-depth", "max-diagnostics"} {
			if c.Flags().Changed(name) {
				v, _ := c.Flags().GetInt(name)
				if v < 1 {
					return fmt.Errorf("%s must be positive", name)
				}
			}
		}
		if o.Limit < 1 || o.Limit > 200 {
			return fmt.Errorf("candidate limit must be 1..200")
		}
		if o.Python == "" {
			o.Python = filepath.FromSlash("adapters/provenance-python/.venv/Scripts/python.exe")
			if _, e := os.Stat(o.Python); e != nil {
				o.Python = "python"
			}
			path, e := exec.LookPath(o.Python)
			if e != nil {
				return e
			}
			o.Python = path
		}
		o.Analyze = probescan.PythonAnalyzer(o.Python)
		r, e := probescan.Scan(c.Context(), o, probescan.PythonParser(o.Python))
		if e != nil {
			return e
		}
		r.ProbeCommand = probescan.CommandTemplate(r, o.Root, o.ImportRoot, adapter, o.Python)
		b, e := json.MarshalIndent(r, "", "  ")
		if e != nil {
			return e
		}
		if output != "" {
			f, e := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if e != nil {
				return e
			}
			_, e = f.Write(append(b, '\n'))
			ce := f.Close()
			if e != nil {
				return e
			}
			if ce != nil {
				return ce
			}
		}
		_, e = fmt.Fprint(c.OutOrStdout(), r.Text())
		return e
	}}
	c.Flags().StringVar(&o.Root, "root", ".", "Source root")
	c.Flags().StringVar(&o.Python, "python", "", "Parser Python executable (no target imports)")
	c.Flags().IntVar(&o.Limits.MaxFiles, "max-files", 2000, "Maximum source files read (cap 2000)")
	c.Flags().Int64Var(&o.Limits.MaxBytes, "max-bytes", 20000000, "Maximum source bytes read (cap 20000000)")
	c.Flags().IntVar(&o.Enumeration.MaxEntries, "max-entries", 20000, "Enumeration entries including non-Python files (cap 200000)")
	c.Flags().IntVar(&o.Enumeration.MaxPathBytes, "max-path-bytes", 8388608, "Cumulative full entry path bytes (cap 33554432)")
	c.Flags().IntVar(&o.Enumeration.MaxDepth, "max-depth", 64, "Directory depth (cap 128)")
	c.Flags().IntVar(&o.Enumeration.MaxDiagnostics, "max-diagnostics", 1000, "Retained diagnostic entries across arrays (cap 10000)")
	c.Flags().StringSliceVar(&o.Exclude, "exclude", nil, "Root-relative excluded paths")
	c.Flags().StringVar(&output, "json", "", "Write JSON to a new path")
	c.Flags().IntVar(&o.Limit, "limit", 20, "Maximum candidates per section (cap 200)")
	c.Flags().StringVar(&o.ImportRoot, "import-root", "", "Explicit Python import root for ambiguous layouts")
	c.Flags().StringVar(&adapter, "adapter", "adapters/provenance-python", "Probe adapter directory for printed instructions (not executed)")
	p.AddCommand(c)
	return p
}
