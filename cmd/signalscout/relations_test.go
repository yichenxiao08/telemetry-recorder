package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
)

func TestRelationsCLIReportsRecordedDenominatorsAndUnknowns(t *testing.T) {
	ctx := context.Background()
	db := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(ctx, db, store.Options{QueueSize: 2, IdleTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	r := model.Resource{ID: "r", Attrs: map[string]any{}, Raw: []byte(`{}`)}
	e := model.Signal{ID: "event:r", Kind: "event", TraceID: "r", TS: 11, End: 11, Resource: r, Attrs: map[string]any{"signalscout.record.kind": "decision", "signalscout.record.json": `{"version":1,"profile":{"id":"synthetic","version":"1"},"step":{"id":"choose"},"output":{"capture":"off","inline":7}}`}}
	if err = s.Submit(ctx, model.Batch{Signals: []model.Signal{{ID: "span:r", Kind: "span", TraceID: "r", Name: "work", TS: 10, End: 20, Resource: r}, e}}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "profile.json")
	if err = os.WriteFile(path, []byte(`{"id":"relations-test","unit":"call","workload":"work","step":"choose","envelope_profile":"synthetic","envelope_version":"1","relations":{"version":1,"rules":[{"id":"positive","unit":"count","tolerance":0,"assert":{"op":"ge","args":[{"op":"field","field":"output"},{"op":"number","number":0}]}}]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"--db", db, "relations", "--profile", path, "--from", "1970-01-01T00:00:00Z", "--to", "1970-01-01T00:00:01Z", "--limit", "10"}
	for _, asJSON := range []bool{false, true} {
		cmd := newRootCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		a := append([]string{}, args...)
		if asJSON {
			a = append(a, "--json")
		}
		cmd.SetArgs(a)
		if err = cmd.ExecuteContext(ctx); err != nil {
			t.Fatal(err)
		}
		if asJSON {
			var report model.RelationReport
			if err = json.Unmarshal(buf.Bytes(), &report); err != nil || report.Totals["positive"].Unknown != 1 {
				t.Fatal(report, err)
			}
		} else if !strings.Contains(buf.String(), "unknown 1/1") || !strings.Contains(buf.String(), "unsampled calls are unknown") {
			t.Fatal(buf.String())
		}
	}
}
