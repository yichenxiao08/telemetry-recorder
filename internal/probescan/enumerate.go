package probescan

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// EnumerationLimits bound application-retained enumeration work, not syscall time.
type EnumerationLimits struct {
	MaxEntries     int `json:"max_entries"`
	MaxPathBytes   int `json:"max_path_bytes"`
	MaxDepth       int `json:"max_depth"`
	MaxDiagnostics int `json:"max_diagnostics"`
}
type Enumeration struct {
	Limits   EnumerationLimits `json:"limits"`
	Entries  int               `json:"entries"`
	Complete bool              `json:"complete"`
}

func enumerationDefaults(l EnumerationLimits) (EnumerationLimits, error) {
	fields := []struct {
		p        *int
		def, cap int
	}{{&l.MaxEntries, 20000, 200000}, {&l.MaxPathBytes, 8388608, 33554432}, {&l.MaxDepth, 64, 128}, {&l.MaxDiagnostics, 1000, 10000}}
	for _, f := range fields {
		if *f.p == 0 {
			*f.p = f.def
		}
		if *f.p < 1 || *f.p > f.cap {
			return l, fmt.Errorf("invalid enumeration bounds")
		}
	}
	return l, nil
}
func (r *Report) diagnostic(dst *[]Entry, e Entry) {
	if len(r.Errors)+len(r.Exclusions)+len(r.Truncations) >= r.Enumeration.Limits.MaxDiagnostics {
		r.DiagnosticsOmitted++
		return
	}
	// Bound strings too, including caller-supplied parser errors. Preserve UTF-8.
	trim := func(s string) string {
		if len(s) <= 1024 {
			return s
		}
		return strings.ToValidUTF8(s[:1021], "?") + "..."
	}
	e.Path = trim(e.Path)
	e.Reason = trim(e.Reason)
	*dst = append(*dst, e)
}

type directoryReader interface {
	ReadDir(int) ([]os.DirEntry, error)
	Close() error
}

func enumerate(ctx context.Context, root string, o Options, r *Report) ([]string, bool) {
	paths := []string{}
	pathBytes := 0
	open := o.readDirectory
	if open == nil {
		open = func(p string) (directoryReader, error) { return os.Open(p) }
	}
	stopped := func(rel, reason string) bool {
		r.diagnostic(&r.Truncations, Entry{rel, reason + "; no sources admitted; eligible total unknown"})
		return false
	}
	var walk func(string, int) bool
	walk = func(dir string, depth int) bool {
		rel, _ := filepath.Rel(root, dir)
		rel = filepath.ToSlash(rel)
		if ctx.Err() != nil {
			return stopped(rel, "enumeration canceled")
		}
		if depth > r.Enumeration.Limits.MaxDepth {
			return stopped(rel, "directory depth limit")
		}
		f, err := open(dir)
		if err != nil {
			r.diagnostic(&r.Errors, Entry{rel, "unreadable directory"})
			return true
		}
		entries := []os.DirEntry{}
		startEntries := r.Enumeration.Entries
		// Commit accounting only for fully read directories; filesystem listing order
		// must not affect reported counts when a path-byte budget aborts a directory.
		abortRead := func(reason string) bool {
			_ = f.Close()
			r.Enumeration.Entries = startEntries
			return stopped(rel, reason)
		}
		// ReadDir(-1) / WalkDir would allocate the entire directory before checking.
		for {
			if ctx.Err() != nil {
				return abortRead("enumeration canceled")
			}
			batch, e := f.ReadDir(1)
			for _, d := range batch {
				if r.Enumeration.Entries >= r.Enumeration.Limits.MaxEntries {
					return abortRead("directory entry limit")
				}
				n := len(filepath.Join(dir, d.Name()))
				if n > r.Enumeration.Limits.MaxPathBytes-pathBytes {
					return abortRead("retained path byte limit")
				}
				r.Enumeration.Entries++
				pathBytes += n
				entries = append(entries, d)
			}
			if e != nil {
				_ = f.Close()
				if e != io.EOF {
					r.diagnostic(&r.Errors, Entry{rel, "unreadable directory entries"})
					return true
				}
				break
			}
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, d := range entries {
			if ctx.Err() != nil {
				return stopped(rel, "enumeration canceled")
			}
			path := filepath.Join(dir, d.Name())
			sub, _ := filepath.Rel(root, path)
			sub = filepath.ToSlash(sub)
			name := strings.ToLower(d.Name())
			excluded := name == ".venv" || name == "venv" || name == "__pycache__" || name == "vendor" || name == ".git" || name == ".cache"
			for _, x := range o.Exclude {
				x = strings.TrimSuffix(filepath.ToSlash(filepath.Clean(x)), "/")
				if sub == x || strings.HasPrefix(sub, x+"/") {
					excluded = true
				}
			}
			if excluded {
				r.diagnostic(&r.Exclusions, Entry{sub, "excluded before reading"})
				continue
			}
			i, e := d.Info()
			if e != nil {
				r.diagnostic(&r.Errors, Entry{sub, "unreadable metadata"})
				continue
			}
			if unsafeLink(i) {
				r.diagnostic(&r.Exclusions, Entry{sub, "symlink/reparse point rejected"})
				continue
			}
			if d.IsDir() {
				if !walk(path, depth+1) {
					return false
				}
			} else if strings.HasSuffix(d.Name(), ".py") {
				paths = append(paths, path)
			}
		}
		return true
	}
	if !walk(root, 0) {
		return nil, false
	}
	r.Enumeration.Complete = len(r.Errors) == 0 && r.DiagnosticsOmitted == 0
	return paths, true
}

// Cancellation is checked between reads; an in-flight filesystem syscall can block.
type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if e := r.ctx.Err(); e != nil {
		return 0, e
	}
	return r.r.Read(p)
}
