package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportCommandRequiresClosedBoundedRequest(t *testing.T) {
	for _, args := range [][]string{{"report", "build"}, {"report", "verify"}} {
		root := newRootCommand()
		root.SetArgs(args)
		root.SetOut(new(bytes.Buffer))
		if err := root.Execute(); err == nil {
			t.Fatal(args)
		}
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "request.json")
	if err := os.WriteFile(p, []byte(`{"version":1,"caused_by":"no"}`), 0600); err != nil {
		t.Fatal(err)
	}
	root := newRootCommand()
	root.SetArgs([]string{"report", "build", "--request", p})
	root.SetOut(new(bytes.Buffer))
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatal(err)
	}
}
