//go:build windows

package probescan

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestJunctionEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "must_not_read.py"), []byte("raise RuntimeError('outside')"), 0600); err != nil {
		t.Fatal(err)
	}
	junction := filepath.Join(root, "escape")
	// mklink is a cmd builtin. Only test-created absolute paths are passed;
	// creating a directory junction requires no symlink privilege.
	cmd := exec.Command("cmd.exe", "/d", "/c", "mklink", "/J", junction, outside)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("junction creation must succeed: %v: %s", err, output)
	}
	r, err := Scan(context.Background(), Options{Root: root, Limits: Limits{2000, 20000000}}, func(context.Context, []byte) error { t.Fatal("parsed outside root through junction"); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if r.Coverage.Eligible != 0 || r.Coverage.Read != 0 || len(r.Exclusions) != 1 || r.Exclusions[0] != (Entry{Path: "escape", Reason: "symlink/reparse point rejected"}) {
		t.Fatalf("junction not rejected: %+v", r)
	}
}
