package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/signalscout/signalscout/internal/doctor"
)

func TestDoctorCommand(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "warnings exit zero", true: "failure exits nonzero"}[fail], func(t *testing.T) {
			dir := t.TempDir()
			db := filepath.Join(dir, "not-created.db")
			output := filepath.Join(dir, "doctor.json")
			if err := os.WriteFile(filepath.Join(dir, "scout_probe.py"), nil, 0600); err != nil {
				t.Fatal(err)
			}
			e := doctor.Environment{Cwd: func() (string, error) { return dir, nil }, RunIn: func(context.Context, string, []string, string) (string, error) { return `{"found":true}`, nil }, Version: "test", Executable: func() (string, error) { return "signalscout", nil }, Lookup: func(s string) (string, error) { return s, nil }, Stat: os.Stat, Writable: func(string) error { return nil }, Port: func(context.Context, string) error { return nil }, Run: func(_ context.Context, _ string, args []string) (string, error) {
				if args[len(args)-1] == "--version" {
					return "Python 3.12.12", nil
				}
				if fail {
					return `{}`, nil
				}
				return `{"yaml":true,"opentelemetry.sdk.trace":true,"opentelemetry.exporter.otlp.proto.http.trace_exporter":true,"requests":true}`, nil
			}}
			cmd := doctorCommand(&db, e)
			var b bytes.Buffer
			cmd.SetOut(&b)
			cmd.SetErr(&b)
			cmd.SetArgs([]string{"--adapter", dir, "--json", output})
			err := cmd.ExecuteContext(context.Background())
			if (err != nil) != fail {
				t.Fatalf("unexpected exit: %v", err)
			}
			data, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			var r doctor.Report
			if err = json.Unmarshal(data, &r); err != nil {
				t.Fatal(err)
			}
			if r.SchemaVersion != 1 || !strings.Contains(b.String(), r.Text()) {
				t.Fatal("JSON/text mismatch")
			}
			if _, err = os.Stat(db); !os.IsNotExist(err) {
				t.Fatal("DB created")
			}
			cmd = doctorCommand(&db, e)
			cmd.SetArgs([]string{"--adapter", dir, "--json", output})
			cmd.SetOut(&b)
			cmd.SetErr(&b)
			if err = cmd.Execute(); err == nil {
				t.Fatal("existing report overwritten")
			}
			after, _ := os.ReadFile(output)
			if !bytes.Equal(data, after) {
				t.Fatal("report changed")
			}
		})
	}
}

func TestDoctorJSONCannotCreateDatabase(t *testing.T) {
	db := filepath.Join(t.TempDir(), "missing.db")
	cmd := doctorCommand(&db, doctor.Environment{})
	cmd.SetArgs([]string{"--json", db})
	if err := cmd.Execute(); err == nil {
		t.Fatal("JSON used database path")
	}
	if _, err := os.Stat(db); !os.IsNotExist(err) {
		t.Fatal("database path created")
	}
}
