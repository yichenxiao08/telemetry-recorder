package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRootHelp(t *testing.T) {
	command := newRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"--help"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "signalscout") {
		t.Fatalf("missing CLI help: %s", output.String())
	}
}
