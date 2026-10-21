package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
)

func TestTraceCLIReadOnlyAndCompare(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "trace.db")
	s, err := store.Open(ctx, path, store.Options{QueueSize: 2, IdleTimeout: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	body := `{"version":1,"id":"decision:r/a","run_id":"r","step":{"id":"choose"},"inputs":[{"name":"private","capture":"off","inline":"secret"}],"output":{"ref":"value:r/result@1","capture":"allowed","inline":"left"}}`
	resource := model.Resource{ID: "r", Attrs: map[string]any{}, Raw: []byte(`{}`)}
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{{ID: "span:r", Kind: "span", TraceID: "r", Name: "work", TS: 10, End: 20, Resource: resource}, {ID: "event:r", Kind: "event", TraceID: "r", TS: 11, End: 11, Resource: resource, Attrs: map[string]any{"signalscout.record.kind": "decision", "signalscout.record.id": "decision:r/a", "signalscout.record.json": body}}}}); err != nil {
		t.Fatal(err)
	}
	bounds := []string{"--from", "1970-01-01T00:00:00Z", "--to", "1970-01-01T00:00:01Z", "--limit", "20"}
	for _, args := range [][]string{{"trace", "value:r/result@1", "--json"}, {"trace", "event:r", "--compare-to", "value:r/result@1"}, {"trace", "r", "choose"}, {"trace", "r", "result@1", "--nodes", "1", "--compare-to", "value:r/result@1", "--json"}} {
		command := newRootCommand()
		var output bytes.Buffer
		command.SetOut(&output)
		command.SetArgs(append(append([]string{"--db", path}, args...), bounds...))
		if err = command.ExecuteContext(ctx); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(output.String(), "secret") || !strings.Contains(output.String(), "capture-off") {
			t.Fatal(output.String())
		}
		if strings.Contains(strings.Join(args, " "), "--json") {
			var result traceOutput
			if err = json.Unmarshal(output.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.TotalVisited > result.NodeBudget || result.Trace.Visited != 1 {
				t.Fatal(result)
			}
		}
	}
	for _, args := range [][]string{{"trace", "value:r/result@1"}, {"trace", "value:r/result@1", "--nodes", "0"}, {"trace", "value:r/result@1", "--limit", "0"}} {
		command := newRootCommand()
		command.SetArgs(append([]string{"--db", path}, args...))
		if err = command.ExecuteContext(ctx); err == nil {
			t.Fatal("missing/invalid bounds accepted")
		}
	}
	detail, err := s.Show(ctx, "r")
	if err != nil || detail.Run.Closed {
		t.Fatal("trace mutated run", err)
	}
}
