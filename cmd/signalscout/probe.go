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
	var output string
	c := &cobra.Command{Use: "scan", Short: "Suggest functions worth probing from Python syntax", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
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
	c.Flags().StringSliceVar(&o.Exclude, "exclude", nil, "Root-relative excluded paths")
	c.Flags().StringVar(&output, "json", "", "Write JSON to a new path")
	c.Flags().IntVar(&o.Limit, "limit", 20, "Maximum candidates per section (cap 200)")
	c.Flags().StringVar(&o.ImportRoot, "import-root", "", "Explicit Python import root for ambiguous layouts")
	p.AddCommand(c)
	return p
}
