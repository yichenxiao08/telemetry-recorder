package probetemplate

import (
	"strings"
	"testing"
)

func TestPowerShellLiteralQuoting(t *testing.T) {
	for _, s := range []string{`C:\with spaces\python.exe`, `C:\O'Brien\日本語\python.exe`, "<selected-python.exe>"} {
		q := Quote(s)
		if q != "'"+strings.ReplaceAll(s, "'", "''")+"'" {
			t.Fatal(q)
		}
		inner := strings.TrimSuffix(strings.TrimPrefix(q, "'"), "'")
		if strings.ReplaceAll(inner, "''", "'") != s {
			t.Fatal("literal round trip")
		}
		out := strings.Join(Render(Context{Python: s, Adapter: s, ImportRoot: s, Selector: "demo:main"}), "\n")
		if !strings.Contains(out, q) || !strings.Contains(out, "--doctor") || !strings.Contains(out, " -- '<target-arguments>'") {
			t.Fatal(out)
		}
	}
}
