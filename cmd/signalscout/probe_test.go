package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestScanRefusesOverwrite(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "existing.json")
	_ = os.WriteFile(p, []byte("original"), 0600)
	c := newProbeCommand()
	c.SetOut(&bytes.Buffer{})
	c.SetArgs([]string{"scan", "--root", d, "--json", p, "--python", "unused"})
	if e := c.Execute(); e == nil {
		t.Fatal("overwrite accepted")
	}
	b, _ := os.ReadFile(p)
	if string(b) != "original" {
		t.Fatal(string(b))
	}
}

func TestLimitValidation(t *testing.T) {
	for _, v := range []string{"0", "-1", "201"} {
		c := newProbeCommand()
		c.SetArgs([]string{"scan", "--limit", v})
		if err := c.Execute(); err == nil {
			t.Fatal("accepted limit", v)
		}
	}
}
