package probescan

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

//go:embed analyze.py
var analysisProgram string

type Signal struct {
	Kind string `json:"kind"`
	File string `json:"file"`
	Line int    `json:"line"`
	Fact string `json:"fact"`
	Tier string `json:"tier"`
}
type Analysis struct {
	Items     []Candidate `json:"items"`
	Truncated int         `json:"truncated"`
}
type Analyzer func(context.Context, []byte) (Analysis, error)

func PythonAnalyzer(python string) Analyzer {
	return func(ctx context.Context, source []byte) (Analysis, error) {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, python, "-E", "-s", "-B", "-c", analysisProgram)
		cmd.Stdin = bytes.NewReader(source)
		cmd.Stderr = io.Discard
		cmd.WaitDelay = time.Second
		out := &analysisBuffer{}
		cmd.Stdout = out
		if err := cmd.Run(); err != nil {
			return Analysis{}, fmt.Errorf("AST analysis failed, invalid source, timed out or exceeded output cap")
		}
		var a Analysis
		if err := json.Unmarshal(out.Bytes(), &a); err != nil {
			return a, fmt.Errorf("invalid AST analysis output")
		}
		return a, nil
	}
}

type analysisBuffer struct{ bytes.Buffer }

func (b *analysisBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 4*1024*1024 {
		return 0, fmt.Errorf("analysis output cap")
	}
	return b.Buffer.Write(p)
}

func moduleName(root, path, importRoot string) (string, string) {
	base := root
	if importRoot != "" {
		var e error
		base, e = filepath.Abs(importRoot)
		if e != nil {
			return "", "invalid import root"
		}
	}
	rel, e := filepath.Rel(base, path)
	if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "file outside import root"
	}
	if importRoot == "" {
		for dir := filepath.Dir(path); dir != root; dir = filepath.Dir(dir) {
			if _, e := os.Stat(filepath.Join(dir, "__init__.py")); e != nil {
				return "", "ambiguous layout: explicit --import-root required"
			}
		}
	}
	parts := strings.Split(strings.TrimSuffix(filepath.ToSlash(rel), ".py"), "/")
	if parts[len(parts)-1] == "__init__" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) == 0 {
		return "", "package root requires explicit parent --import-root"
	}
	for _, p := range parts {
		for i, r := range p {
			valid := r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9'
			if !valid {
				return "", "module component is not an identifier"
			}
		}
	}
	return strings.Join(parts, "."), ""
}

func finalize(r *Report, limit int) {
	less := func(a, b Candidate) bool {
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Function < b.Function
	}
	for section, list := range []*[]Candidate{&r.Candidates, &r.Unresolved} {
		sort.Slice(*list, func(i, j int) bool { return less((*list)[i], (*list)[j]) })
		for i := range *list {
			c := &(*list)[i]
			c.Rank = i + 1
			for _, other := range *list {
				if other.Score == c.Score {
					c.TieCount++
				}
			}
		}
		if len(*list) > limit {
			r.Truncations = append(r.Truncations, Entry{Path: []string{"primary", "unresolved"}[section], Reason: fmt.Sprintf("post-ranking limit: omitted %d", len(*list)-limit)})
			*list = (*list)[:limit]
		}
	}
}
func (r Report) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Parsed %d/%d eligible files; read %d bytes. Complete: %t. Excluded: %d; errors: %d.\n", r.Coverage.Parsed, r.Coverage.Eligible, r.Coverage.Bytes, r.Coverage.Complete, len(r.Exclusions), len(r.Errors))
	fmt.Fprintln(&b, "Scan limits apply before ranking; candidate limits apply after ranking (primary and unresolved independently).")
	for _, e := range r.Truncations {
		fmt.Fprintf(&b, "Truncated %s: %s\n", e.Path, e.Reason)
	}
	for n, list := range [][]Candidate{r.Candidates, r.Unresolved} {
		title := "Candidates"
		if n == 1 {
			title = "Unresolved/unsupported (bounded separately)"
		}
		fmt.Fprintln(&b, title+":")
		for _, c := range list {
			fmt.Fprintf(&b, "%d %s:%d %s score=%d tied=%d\n", c.Rank, c.File, c.Line, c.Function, c.Score, c.TieCount)
			for _, s := range c.Details {
				fmt.Fprintf(&b, "  %s %s %s:%d: worth probing because %s\n", s.Tier, s.Kind, s.File, s.Line, s.Fact)
			}
			if c.Selector == "" {
				fmt.Fprintf(&b, "  no selector: %s\n", c.UnsupportedReason)
			} else {
				fmt.Fprintf(&b, "  selector: %s\n  config: %s\n", c.Selector, c.Config)
			}
			if c.Hot {
				fmt.Fprintln(&b, "  INCOMPLETE sampling suggestion: every_nth 25, max_per_run 40. Unsampled calls are unknown for absence and invariant claims. Optional paired --queue-max-items <items> --queue-max-bytes <bytes>; memory estimate >= items * average serialized record bytes, plus object/worker overhead. No queue size is a default or delivery guarantee.")
			}
		}
	}
	return b.String()
}
