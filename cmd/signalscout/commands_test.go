package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
)

func TestRunsCommandsAreReadOnly(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "command.db")
	s, err := store.Open(ctx, path, store.Options{QueueSize: 2, IdleTimeout: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	}()
	resource := model.Resource{ID: "r", Attrs: map[string]any{}, Raw: []byte(`{}`)}
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{{ID: "s", Kind: "span", TraceID: "run", Name: "work", TS: 1, End: 2, Resource: resource}}}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"runs", "list", "--workload", "work", "--outcome", "pass"}, {"runs", "show", "run"}} {
		command := newRootCommand()
		var output bytes.Buffer
		command.SetOut(&output)
		command.SetArgs(append([]string{"--db", path}, args...))
		if err = command.ExecuteContext(ctx); err != nil {
			t.Fatal(err)
		}
		if !json.Valid(output.Bytes()) || !bytes.Contains(output.Bytes(), []byte("run")) {
			t.Fatalf("bad output: %s", output.String())
		}
	}
	detail, err := s.Show(ctx, "run")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Run.Closed {
		t.Fatal("inspection closed the open run")
	}
	command := newRootCommand()
	command.SetArgs([]string{"--db", path, "runs", "show", "missing"})
	if err = command.ExecuteContext(ctx); err == nil {
		t.Fatal("missing run succeeded")
	}
	missing := filepath.Join(t.TempDir(), "missing.db")
	command = newRootCommand()
	command.SetArgs([]string{"--db", missing, "runs", "list"})
	if err = command.ExecuteContext(ctx); err == nil {
		t.Fatal("missing database succeeded")
	}
	if _, err = os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("read command created database")
	}
}

func TestServeGracefulShutdown(t *testing.T) {
	// Reserve a local port, then let the actual serve lifecycle bind it.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err = listener.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	path := filepath.Join(t.TempDir(), "serve.db")
	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, path, address, store.Options{QueueSize: 4, IdleTimeout: time.Hour}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()
	client := &http.Client{Timeout: time.Second}
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "otlp", "traces.json"))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		response, requestErr := client.Post("http://"+address+"/v1/traces", "application/json", bytes.NewReader(data))
		if requestErr == nil {
			body, _ := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("intake: %s", body)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not start: %v", requestErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown stalled")
	}
	reader, err := store.OpenReader(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	list, err := reader.List(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 4 {
		t.Fatalf("got %d runs", len(list))
	}
	for _, run := range list {
		if !run.Closed {
			t.Fatal("shutdown left run open")
		}
	}
}
