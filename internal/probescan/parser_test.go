package probescan

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRealASTNoExecution(t *testing.T) {
	p, e := exec.LookPath("python")
	if e != nil {
		p, e = exec.LookPath("python3")
	}
	if e != nil {
		t.Fatal("Python required for AST integration", e)
	}
	parse := PythonParser(p)
	d := t.TempDir()
	sentinel := filepath.Join(d, "sentinel")
	source := []byte("open(" + "r'" + sentinel + "', 'w').write('executed')\nasync def a(): pass\ndef g(): yield 1\n")
	if e = parse(context.Background(), source); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(sentinel); !os.IsNotExist(e) {
		t.Fatal("target executed")
	}
	if e = parse(context.Background(), []byte("def broken(")); e == nil {
		t.Fatal("invalid source accepted")
	}
}
