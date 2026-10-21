package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
)

func TestMarkCLIUsesServingWriterAndLabelsReadOnly(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "metadata.db")
	s, err := store.Open(ctx, path, store.Options{QueueSize: 2, IdleTimeout: time.Second, SweepInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	root := model.Signal{ID: "span:a", Kind: "span", TraceID: "a", Name: "work", TS: 100, End: 200, Attrs: map[string]any{}, Resource: model.Resource{ID: "r", Raw: []byte(`{}`), Attrs: map[string]any{}}, Scope: []byte(`{}`), Raw: []byte(`{}`)}
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{root}}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(metadataHandler(s))
	defer server.Close()
	command := newRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"mark", "a", "--endpoint", server.URL, "--dimension", "quality", "--value", "bad", "--created-at", "2026-10-20T00:01:00Z"})
	if err = command.ExecuteContext(ctx); err != nil {
		t.Fatal(err)
	}
	var label model.Annotation
	if err = json.Unmarshal(output.Bytes(), &label); err != nil {
		t.Fatal(err)
	}
	stored, err := s.Annotation(ctx, label.ID)
	if err != nil || stored.RecordedAt != label.RecordedAt {
		t.Fatal("ack did not expose committed revision", err)
	}
	detail, err := s.Show(ctx, "a")
	if err != nil || detail.Run.EndedAt != 200 {
		t.Fatal("mark changed interval", err)
	}
	command = newRootCommand()
	output.Reset()
	command.SetOut(&output)
	command.SetArgs([]string{"--db", path, "labels", "list", "--run", "a", "--from", "2026-01-01T00:00:00Z", "--to", "2027-01-01T00:00:00Z", "--limit", "10"})
	if err = command.ExecuteContext(ctx); err != nil || !strings.Contains(output.String(), label.ID) {
		t.Fatal(output.String(), err)
	}
	data, _ := json.Marshal(label)
	for _, tc := range []struct {
		body    string
		origin  string
		content string
	}{{string(data), "https://evil.example", "application/json"}, {string(data), "", "text/plain"}, {string(data) + " {}", "", "application/json"}, {string(data), "", "application/json"}} {
		request := httptest.NewRequest("POST", "http://local/api/annotations", strings.NewReader(tc.body))
		request.Header.Set("Content-Type", tc.content)
		request.Header.Set("Origin", tc.origin)
		response := httptest.NewRecorder()
		metadataHandler(s).ServeHTTP(response, request)
		if response.Code == http.StatusOK {
			t.Fatal("invalid metadata accepted")
		}
	}
	select {
	case err := <-s.Errors():
		t.Fatal("bad client input killed server", err)
	default:
	}
}
