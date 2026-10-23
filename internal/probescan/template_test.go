package probescan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandTemplateContext(t *testing.T) {
	root := t.TempDir()
	adapter := filepath.Join(root, "adapter's 日本語")
	if e := os.Mkdir(adapter, 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(adapter, "scout_probe.py"), []byte("raise RuntimeError('must not execute')"), 0600); e != nil {
		t.Fatal(e)
	}
	python, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name       string
		r          Report
		importRoot string
		want       string
	}{{"single", Report{Candidates: []Candidate{{Selector: "demo:one"}}}, root, "'demo:one'"}, {"multi", Report{Candidates: []Candidate{{Selector: "demo:one"}, {Selector: "demo:two"}}}, root, "'module:function'"}, {"hidden", Report{Candidates: []Candidate{{Selector: "demo:one"}}, Truncations: []Entry{{Reason: "post-ranking limit"}}}, root, "'module:function'"}, {"unresolved", Report{Unresolved: []Candidate{{UnsupportedReason: "ambiguous layout"}}}, "", "'<absolute-target-directory>'"}} {
		t.Run(tc.name, func(t *testing.T) {
			lines := CommandTemplate(tc.r, root, tc.importRoot, adapter, python)
			text := strings.Join(lines, "\n")
			if !strings.Contains(text, tc.want) || !strings.Contains(text, "adapter''s 日本語") || !strings.Contains(text, "--doctor '<new-probe-doctor-json-path>'") {
				t.Fatal(text)
			}
			tc.r.ProbeCommand = lines
			firstText := tc.r.Text()
			a, _ := json.Marshal(tc.r)
			tc.r.ProbeCommand = CommandTemplate(tc.r, root, tc.importRoot, adapter, python)
			b, _ := json.Marshal(tc.r)
			if string(a) != string(b) {
				t.Fatal("unstable JSON")
			}
			if tc.r.Text() != firstText {
				t.Fatal("unstable text")
			}
		})
	}
	text := strings.Join(CommandTemplate(Report{}, root, "", filepath.Join(root, "absent"), filepath.Join(root, "missing.exe")), "\n")
	for _, part := range []string{"'<adapter-directory>'", "'<selected-python.exe>'", "'<absolute-target-directory>'", "'module:function'"} {
		if !strings.Contains(text, part) {
			t.Fatal(text)
		}
	}
}
