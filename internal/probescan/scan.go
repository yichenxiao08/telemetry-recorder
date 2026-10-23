// Package probescan enumerates source without executing target code.
package probescan

import (
	"bytes"
	"context"

	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Candidate struct {
	File              string   `json:"file"`
	Line              int      `json:"line"`
	Function          string   `json:"function"`
	Signals           []string `json:"signals"`
	Selector          string   `json:"selector"`
	UnsupportedReason string   `json:"unsupported_reason"`
	Details           []Signal `json:"details"`
	Score             int      `json:"score"`
	Rank              int      `json:"rank"`
	TieCount          int      `json:"tie_count"`
	Config            string   `json:"config"`
	Hot               bool     `json:"hot"`
	Advice            string   `json:"advice"`
}
type Entry struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}
type Coverage struct {
	Eligible int   `json:"eligible"`
	Parsed   int   `json:"parsed"`
	Read     int   `json:"read"`
	Bytes    int64 `json:"bytes"`
	Complete bool  `json:"complete"`
}
type Limits struct {
	MaxFiles int   `json:"max_files"`
	MaxBytes int64 `json:"max_bytes"`
}
type Report struct {
	ProbeCommand  []string    `json:"probe_command"`
	SchemaVersion int         `json:"schema_version"`
	Candidates    []Candidate `json:"candidates"`
	Unresolved    []Candidate `json:"unresolved"`
	Coverage      Coverage    `json:"coverage"`
	Exclusions    []Entry     `json:"exclusions"`
	Errors        []Entry     `json:"errors"`
	Truncations   []Entry     `json:"truncations"`
	Limits        Limits      `json:"limits"`
}
type Options struct {
	Root, Python string
	ImportRoot   string
	Limit        int
	Analyze      Analyzer
	Exclude      []string
	Limits       Limits
	Open         func(string) (*os.File, error)
}
type Parser func(context.Context, []byte) error

func PythonParser(python string) Parser {
	return func(ctx context.Context, source []byte) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		// Only this fixed program executes. Target bytes are data on stdin.
		cmd := exec.CommandContext(ctx, python, "-E", "-s", "-B", "-c", `import ast,sys
try:
 ast.parse(sys.stdin.buffer.read())
 print("ok")
except (SyntaxError, ValueError, UnicodeError):
 print("invalid")`)
		cmd.Stdin = bytes.NewReader(source)
		cmd.Stderr = io.Discard
		cmd.WaitDelay = time.Second
		out := &bounded{}
		cmd.Stdout = out
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("parser failed or timed out")
		}
		if out.String() != "ok\n" && out.String() != "ok\r\n" {
			return fmt.Errorf("invalid Python source or parser output")
		}
		return nil
	}
}

type bounded struct{ bytes.Buffer }

func (b *bounded) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 1024 {
		return 0, fmt.Errorf("parser output bound")
	}
	return b.Buffer.Write(p)
}

func Scan(ctx context.Context, o Options, parse Parser) (Report, error) {
	r := Report{ProbeCommand: []string{}, SchemaVersion: 1, Candidates: []Candidate{}, Unresolved: []Candidate{}, Exclusions: []Entry{}, Errors: []Entry{}, Truncations: []Entry{}, Limits: o.Limits}
	if o.Limits.MaxFiles < 1 || o.Limits.MaxFiles > 2000 || o.Limits.MaxBytes < 1 || o.Limits.MaxBytes > 20000000 {
		return r, fmt.Errorf("bounds must be positive and at most 2000 files / 20000000 bytes")
	}
	if o.Limit == 0 {
		o.Limit = 20
	}
	if o.Limit < 1 || o.Limit > 200 {
		return r, fmt.Errorf("candidate limit must be 1..200")
	}
	root, err := filepath.Abs(o.Root)
	if err != nil {
		return r, err
	}
	info, err := os.Lstat(root)
	if err != nil {
		return r, err
	}
	if !info.IsDir() || unsafeLink(info) {
		return r, fmt.Errorf("root must be an ordinary directory")
	}
	// Reject linked ancestors too, rather than silently resolving them.
	for p := filepath.Dir(root); ; p = filepath.Dir(p) {
		i, e := os.Lstat(p)
		if e != nil {
			return r, e
		}
		if unsafeLink(i) {
			return r, fmt.Errorf("linked root ancestor")
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	paths := []string{}
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if e != nil {
			r.Errors = append(r.Errors, Entry{rel, "unreadable entry"})
			return nil
		}
		if path == root {
			return nil
		}
		name := strings.ToLower(d.Name())
		excluded := name == ".venv" || name == "venv" || name == "__pycache__" || name == "vendor" || name == ".git" || name == ".cache"
		for _, x := range o.Exclude {
			x = strings.TrimSuffix(filepath.ToSlash(filepath.Clean(x)), "/")
			if rel == x || strings.HasPrefix(rel, x+"/") {
				excluded = true
			}
		}
		if excluded {
			r.Exclusions = append(r.Exclusions, Entry{rel, "excluded before reading"})
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		i, e := d.Info()
		if e != nil {
			r.Errors = append(r.Errors, Entry{rel, "unreadable metadata"})
			return nil
		}
		if unsafeLink(i) {
			r.Exclusions = append(r.Exclusions, Entry{rel, "symlink/reparse point rejected"})
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".py") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return r, err
	}
	sort.Slice(paths, func(i, j int) bool { return filepath.ToSlash(paths[i]) < filepath.ToSlash(paths[j]) })
	r.Coverage.Eligible = len(paths)
	for _, path := range paths {
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if r.Coverage.Read >= o.Limits.MaxFiles {
			r.Truncations = append(r.Truncations, Entry{rel, "file limit"})
			continue
		}
		i, e := os.Lstat(path)
		if e != nil || unsafeLink(i) {
			r.Errors = append(r.Errors, Entry{rel, "unreadable or replaced entry"})
			continue
		}
		if i.Size() > o.Limits.MaxBytes-r.Coverage.Bytes {
			r.Truncations = append(r.Truncations, Entry{rel, "byte limit"})
			continue
		}
		open := o.Open
		if open == nil {
			open = os.Open
		}
		f, e := open(path)
		if e != nil {
			r.Errors = append(r.Errors, Entry{rel, "unreadable source"})
			continue
		}
		b, e := io.ReadAll(io.LimitReader(f, o.Limits.MaxBytes-r.Coverage.Bytes+1))
		_ = f.Close()
		if e != nil {
			r.Errors = append(r.Errors, Entry{rel, "unreadable source"})
			continue
		}
		if int64(len(b)) > o.Limits.MaxBytes-r.Coverage.Bytes {
			r.Truncations = append(r.Truncations, Entry{rel, "byte limit"})
			continue
		}
		r.Coverage.Read++
		r.Coverage.Bytes += int64(len(b))
		var analysis Analysis
		if o.Analyze != nil {
			analysis, e = o.Analyze(ctx, b)
		} else {
			e = parse(ctx, b)
		}
		if e != nil {
			r.Errors = append(r.Errors, Entry{rel, e.Error()})
			continue
		}
		r.Coverage.Parsed++
		if analysis.Truncated > 0 {
			r.Truncations = append(r.Truncations, Entry{rel, fmt.Sprintf("per-file analysis limit: omitted %d", analysis.Truncated)})
		}
		module, reason := moduleName(root, path, o.ImportRoot)
		for _, c := range analysis.Items {
			c.File = rel
			c.Signals = []string{}
			for j := range c.Details {
				c.Details[j].File = rel
				c.Signals = append(c.Signals, c.Details[j].Kind)
				if c.Details[j].Kind == "loop_call_site" {
					c.Hot = true
					c.Advice = hotAdvice
				}
			}
			c.Score = len(c.Signals)
			if c.UnsupportedReason == "" {
				c.UnsupportedReason = reason
			}
			if c.UnsupportedReason == "" {
				c.Selector = module + ":" + c.Function
				c.Config = fmt.Sprintf("- module: %q\n  function: %q\n  capture: {}", module, c.Function)
				if c.Hot {
					c.Config += "\n  sampling: {every_nth: 25, max_per_run: 40}"
				}
				r.Candidates = append(r.Candidates, c)
			} else {
				r.Unresolved = append(r.Unresolved, c)
			}
		}
	}
	r.Coverage.Complete = len(r.Errors) == 0 && len(r.Truncations) == 0
	finalize(&r, o.Limit)
	r.Coverage.Complete = len(r.Errors) == 0 && len(r.Truncations) == 0
	return r, nil
}
