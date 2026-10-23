package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProposeCommand(t *testing.T) {
	p := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(p, []byte(`{"version":1,"claims":[{"id":"observed","tier":"observation","text":"Recorded observation","evidence":[{"id":"event:synthetic","kind":"native"}]}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	var previous string
	for i := 0; i < 2; i++ {
		root := newRootCommand()
		out := new(bytes.Buffer)
		root.SetOut(out)
		root.SetArgs([]string{"--db", filepath.Join(t.TempDir(), "absent.db"), "report", "propose", "--input", p})
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), `"tier": "hypothesis"`) || (i > 0 && out.String() != previous) {
			t.Fatal(out.String())
		}
		previous = out.String()
	}
}
