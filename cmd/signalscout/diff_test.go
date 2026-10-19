package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
)

func TestDiffCommandsAndInsufficientBaseline(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "diff.db")
	s, err := store.Open(ctx, path, store.Options{QueueSize: 2, IdleTimeout: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	var signals []model.Signal
	for i := 0; i < 41; i++ {
		id := fmt.Sprintf("run%02d", i)
		signal := model.Signal{ID: "span:" + id, SpanID: id, TraceID: id, Kind: "span", Name: "work", TS: int64(i+1) * 1000000000, End: int64(i+1)*1000000000 + 1000000, Resource: model.Resource{ID: "r", Attrs: map[string]any{}}}
		if i == 40 {
			signal.Status = "error"
			signal.End += 100000000
		}
		signals = append(signals, signal)
	}
	if err = s.Submit(ctx, model.Batch{Signals: signals}); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args         []string
		mode, status string
	}{
		{nil, "default", "ok"}, {[]string{"--baseline-n", "20"}, "count", "ok"}, {[]string{"--baseline-before", "run20"}, "before", "ok"}, {[]string{"--baseline-before", "1970-01-01T00:00:21Z"}, "before", "ok"}, {[]string{"--baseline-n", "9"}, "count", "insufficient_baseline"},
	} {
		command := newRootCommand()
		var output bytes.Buffer
		command.SetOut(&output)
		args := []string{"--db", path, "diff", "run40", "--json"}
		command.SetArgs(append(args, tc.args...))
		if err = command.ExecuteContext(ctx); err != nil {
			t.Fatal(err)
		}
		var got model.DiffResult
		if err = json.Unmarshal(output.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.BaselineSpec.Mode != tc.mode || got.Status != tc.status || got.ID == "" || got.ElapsedNS <= 0 {
			t.Fatalf("bad diff %s", output.String())
		}
		if tc.status == "insufficient_baseline" && !strings.Contains(got.Message, "require 10") {
			t.Fatal("missing diagnostic")
		}
	}
	command := newRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"--db", path, "diff", "run40"})
	if err = command.ExecuteContext(ctx); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "baseline=default") || !strings.Contains(output.String(), "span:run40") {
		t.Fatalf("no explanation/evidence: %s", output.String())
	}
	command = newRootCommand()
	command.SetArgs([]string{"--db", path, "diff", "run40", "--baseline-n", "0"})
	if err = command.ExecuteContext(ctx); err == nil {
		t.Fatal("invalid count accepted")
	}
}
