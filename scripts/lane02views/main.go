// Validate existing development evidence over HTTP with a read-only store and
// a writer that rejects all writes. No generator or intake replay is invoked.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/signalscout/signalscout/internal/cohort"
	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
	"github.com/signalscout/signalscout/internal/viewer"
)

type noWriter struct{}

func (noWriter) SaveDiff(context.Context, model.DiffResult) error {
	return fmt.Errorf("development HTTP verifier forbids writes")
}
func run() error {
	db := flag.String("db", ".cache/lane02-dev/attempt-01/fresh.db", "existing development store")
	input := flag.String("input", "benchmark/results/lane02-stage1-development.json", "development reports")
	output := flag.String("out", "benchmark/results/lane02-stage1-views.json", "verification results")
	flag.Parse()
	raw, err := os.ReadFile(*input)
	if err != nil {
		return err
	}
	var report struct {
		Evaluation bool
		Cases      []struct {
			Seed    int
			Variant string
			Result  cohort.Result
		}
	}
	if err = json.Unmarshal(raw, &report); err != nil {
		return err
	}
	if report.Evaluation {
		return fmt.Errorf("stage 1 verifier accepts development only")
	}
	ctx := context.Background()
	reader, err := store.OpenReader(ctx, *db)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	handler, err := viewer.New(reader, noWriter{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		return err
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	cases := []map[string]any{}
	for _, c := range report.Cases {
		if c.Seed != 101 && c.Seed != 103 {
			return fmt.Errorf("non development seed")
		}
		result, e := cohort.Analyze(ctx, reader, c.Result.Problem.ID, c.Result.Reference.ID)
		if e != nil {
			return e
		}
		links := map[string]bool{}
		for _, f := range result.Differences {
			for _, u := range f.EvidenceURLs {
				links[u] = true
			}
		}
		for _, f := range result.Presentation.Tail {
			for _, u := range f.EvidenceURLs {
				links[u] = true
			}
		}
		for link := range links {
			parts := strings.SplitN(link, "#", 2)
			if len(parts) != 2 {
				return fmt.Errorf("anchor missing")
			}
			response, e := http.Get(server.URL + parts[0])
			if e != nil {
				return e
			}
			body, e := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if e != nil {
				return e
			}
			if response.StatusCode != 200 || !strings.Contains(string(body), `id="`+parts[1]+`"`) {
				return fmt.Errorf("evidence anchor unresolved: %s", link)
			}
		}
		cases = append(cases, map[string]any{"seed": c.Seed, "variant": c.Variant, "verified_links": len(links), "read_only": true})
	}
	data, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(*output, append(data, '\n'), 0600)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
