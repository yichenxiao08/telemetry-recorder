// Verify frozen lane-01 CLI bytes and full pinned reports, without writing stores.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/signalscout/signalscout/internal/cohort"
	"github.com/signalscout/signalscout/internal/store"
)

func run() error {
	before := flag.String("before", ".cache/lane02-legacy/signalscout-before.exe", "binary built from 70ce5ac")
	after := flag.String("after", "bin/signalscout.exe", "Stage 1 binary")
	db := flag.String("db", ".cache/lane-01/eval-002/fresh.db", "existing read-only store")
	out := flag.String("out", "benchmark/results/lane02-stage1-legacy.json", "new report")
	flag.Parse()
	ctx := context.Background()
	raw, err := os.ReadFile("benchmark/results/lane-01.json")
	if err != nil {
		return err
	}
	var frozen struct {
		Cases []struct {
			Seed    int
			Variant string
			Result  cohort.Result
		}
	}
	if err = json.Unmarshal(raw, &frozen); err != nil {
		return err
	}
	reader, err := store.OpenReader(ctx, *db)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	reports := []map[string]any{}
	for _, c := range frozen.Cases {
		current, e := cohort.Analyze(ctx, reader, c.Result.Problem.ID, c.Result.Reference.ID)
		if e != nil {
			return e
		}
		a, _ := json.Marshal(current)
		b, _ := json.Marshal(c.Result)
		if !bytes.Equal(a, b) {
			return fmt.Errorf("full report changed: %d/%s", c.Seed, c.Variant)
		}
		outputs := [][]byte{}
		for _, binary := range []string{*before, *after} {
			path, e := filepath.Abs(binary)
			if e != nil {
				return e
			}
			data, e := exec.CommandContext(ctx, path, "--db", *db, "explain-cohort", c.Result.Problem.ID, c.Result.Reference.ID, "--json").Output()
			if e != nil {
				return e
			}
			outputs = append(outputs, data)
		}
		if !bytes.Equal(outputs[0], outputs[1]) {
			return fmt.Errorf("CLI bytes changed: %d/%s", c.Seed, c.Variant)
		}
		hash := sha256.Sum256(outputs[0])
		texts := [][]byte{}
		for _, binary := range []string{*before, *after} {
			path, e := filepath.Abs(binary)
			if e != nil {
				return e
			}
			text, e := exec.CommandContext(ctx, path, "--db", *db, "explain-cohort", c.Result.Problem.ID, c.Result.Reference.ID).Output()
			if e != nil {
				return e
			}
			texts = append(texts, text)
		}
		if !bytes.Equal(texts[0], texts[1]) {
			return fmt.Errorf("legacy text bytes changed: %d/%s", c.Seed, c.Variant)
		}
		textHash := sha256.Sum256(texts[0])
		reports = append(reports, map[string]any{"seed": c.Seed, "variant": c.Variant, "bytes": len(outputs[0]), "before_sha256": hex.EncodeToString(hash[:]), "after_sha256": hex.EncodeToString(hash[:]), "full_report_identical": true, "text_bytes": len(texts[0]), "before_text_sha256": hex.EncodeToString(textHash[:]), "after_text_sha256": hex.EncodeToString(textHash[:])})
	}
	if len(reports) != 15 {
		return fmt.Errorf("expected 15 cases")
	}
	frozenHash := sha256.Sum256(raw)
	data, err := json.MarshalIndent(map[string]any{"before_revision": "70ce5acc8db6679535dc660278215dc4fdceb22b", "frozen_file_sha256": hex.EncodeToString(frozenHash[:]), "cases": reports}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(*out, append(data, '\n'), 0600)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
