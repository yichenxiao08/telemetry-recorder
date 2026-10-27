package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/signalscout/signalscout/internal/redaction"
)

func TestPublicExperimentValidation(t *testing.T) {
	good := &redaction.Policy{Profiles: map[string]redaction.Profile{"public": {Fields: map[string]redaction.Rule{"inputs.speed": {Kind: "number"}}}}}
	db := filepath.Join(t.TempDir(), "fresh.db")
	cases := []struct {
		name, db, addr, profile string
		policy                  *redaction.Policy
		ok                      bool
	}{
		{"valid", db, "127.0.0.1:4318", "public", good, true},
		{"ipv6", db, "[::1]:4318", "public", good, true},
		{"remote", db, "0.0.0.0:4318", "public", good, false},
		{"hostname", db, "localhost:4318", "public", good, false},
		{"relative db", "old.db", "127.0.0.1:4318", "public", good, false},
		{"missing policy", db, "127.0.0.1:4318", "public", nil, false},
		{"missing profile", db, "127.0.0.1:4318", "none", good, false},
		{"non numeric", db, "127.0.0.1:4318", "public", &redaction.Policy{Profiles: map[string]redaction.Profile{"public": {Fields: map[string]redaction.Rule{"inputs.secret": {Kind: "enum"}}}}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := validatePublicExperiment(c.db, c.addr, c.policy, c.profile) == nil; got != c.ok {
				t.Fatalf("valid = %v, want %v", got, c.ok)
			}
		})
	}
}

func TestPublicExperimentRejectsExistingDatabase(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "existing.db")
	if err := os.WriteFile(db, []byte("existing-data"), 0600); err != nil {
		t.Fatal(err)
	}
	policy := filepath.Join(dir, "policy.json")
	if err := os.WriteFile(policy, []byte(`{"version":1,"profiles":{"public":{"fields":{"inputs.speed":{"kind":"number"}}}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := newServeCommand(&db)
	cmd.SetArgs([]string{"--public-data-experiment", "--capture-policy", policy, "--capture-profile", "public"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("existing database accepted")
	}
	got, err := os.ReadFile(db)
	if err != nil || string(got) != "existing-data" {
		t.Fatalf("existing file changed: %q %v", got, err)
	}
}
