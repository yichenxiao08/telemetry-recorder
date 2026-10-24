package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

func TestScanTemplateStableCommandOutput(t *testing.T) {
	d := t.TempDir()
	sentinel := filepath.Join(d, "sentinel")
	source := "open(r'" + sentinel + "','w').write('executed')\nSTATE={}\ndef read(key): return STATE[key]\n"
	if e := os.WriteFile(filepath.Join(d, "demo.py"), []byte(source), 0600); e != nil {
		t.Fatal(e)
	}
	output := t.TempDir() // Keep the scanned tree identical across runs, including non-Python entries.
	texts := []string{}
	jsons := []string{}
	for _, name := range []string{"one.json", "two.json"} {
		c := newProbeCommand()
		var out bytes.Buffer
		c.SetOut(&out)
		c.SetArgs([]string{"scan", "--root", d, "--import-root", d, "--adapter", filepath.Join(d, "missing-adapter"), "--json", filepath.Join(output, name)})
		if e := c.Execute(); e != nil {
			t.Fatal(e)
		}
		b, e := os.ReadFile(filepath.Join(output, name))
		if e != nil {
			t.Fatal(e)
		}
		texts = append(texts, out.String())
		jsons = append(jsons, string(b))
	}
	if texts[0] != texts[1] || jsons[0] != jsons[1] {
		t.Fatal("unstable CLI output")
	}
	if _, e := os.Stat(sentinel); !os.IsNotExist(e) {
		t.Fatal("target executed")
	}
	if !strings.Contains(texts[0], "'demo:read'") || !strings.Contains(texts[0], "--doctor is required") {
		t.Fatal(texts[0])
	}
}
