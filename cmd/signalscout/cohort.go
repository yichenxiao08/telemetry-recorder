package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/cohort"
	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
	"github.com/spf13/cobra"
)

type cohortRequest struct {
	ID        string                `json:"id"`
	Profile   model.AnalysisProfile `json:"profile"`
	Selection model.CohortSelection `json:"selection"`
}

func decodeOne(d *json.Decoder, value any) error {
	if err := d.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("one JSON value required")
	}
	return nil
}

func jsonOutput(cmd *cobra.Command, value any) error {
	e := json.NewEncoder(cmd.OutOrStdout())
	e.SetIndent("", "  ")
	return e.Encode(value)
}
func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func metadataPost(ctx context.Context, endpoint, path string, value any, out any) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "http" || u.Host == "" || u.User != nil {
		return fmt.Errorf("HTTP serving endpoint required")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(endpoint, "/")+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("use the serving writer endpoint (start serve first): %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return fmt.Errorf("metadata HTTP %d: %s", response.StatusCode, body)
	}
	return json.Unmarshal(body, out)
}
func timeBounds(from, to string) (int64, int64, error) {
	a, err := time.Parse(time.RFC3339Nano, from)
	if err != nil {
		return 0, 0, err
	}
	b, err := time.Parse(time.RFC3339Nano, to)
	if err != nil {
		return 0, 0, err
	}
	return a.UnixNano(), b.UnixNano(), nil
}

func newMarkCommand() *cobra.Command {
	a := model.Annotation{Source: "cli-mark", Synthetic: true, RubricID: "manual", RubricVersion: "1"}
	var endpoint, created string
	cmd := &cobra.Command{Use: "mark <run_id>", Short: "Append a synthetic review through the serving writer", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		a.ID, err = randomID()
		if err != nil {
			return err
		}
		a.RunID = args[0]
		a.CreatedAt = time.Now().UnixNano()
		if created != "" {
			t, e := time.Parse(time.RFC3339Nano, created)
			if e != nil {
				return e
			}
			a.CreatedAt = t.UnixNano()
		}
		var result model.Annotation
		if err = metadataPost(cmd.Context(), endpoint, "/api/annotations", a, &result); err != nil {
			return err
		}
		return jsonOutput(cmd, result)
	}}
	f := cmd.Flags()
	f.StringVar(&endpoint, "endpoint", "http://127.0.0.1:4318", "Existing serving writer endpoint")
	f.StringVar(&a.Dimension, "dimension", "", "Assessment dimension")
	f.StringVar(&a.Value, "value", "", "Assessment value")
	f.StringVar(&a.Reviewer, "reviewer", "synthetic-developer", "Synthetic reviewer identity")
	f.StringVar(&a.RubricID, "rubric", "manual", "Rubric ID")
	f.StringVar(&a.RubricVersion, "rubric-version", "1", "Rubric version")
	f.StringVar(&a.OutputRef, "output-ref", "", "Observed output ref")
	f.StringSliceVar(&a.EvidenceIDs, "evidence", nil, "Native evidence IDs belonging to run")
	f.StringVar(&created, "created-at", "", "Review RFC3339 timestamp, independent of execution")
	f.StringVar(&a.Supersedes, "supersedes", "", "Prior same-reviewer annotation ID")
	f.StringVar(&a.Retracts, "retracts", "", "Prior same-reviewer annotation ID")
	return cmd
}
func newLabelsCommand(database *string) *cobra.Command {
	root := &cobra.Command{Use: "labels", Short: "Read append-only synthetic assertions"}
	var from, to, run string
	var limit int
	list := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		a, b, err := timeBounds(from, to)
		if err != nil {
			return err
		}
		reader, err := store.OpenReader(cmd.Context(), *database)
		if err != nil {
			return err
		}
		defer func() { _ = reader.Close() }()
		labels, err := reader.Annotations(cmd.Context(), run, a, b, limit)
		if err != nil {
			return err
		}
		return jsonOutput(cmd, labels)
	}}
	list.Flags().StringVar(&from, "from", "", "Required recorded-at lower bound")
	list.Flags().StringVar(&to, "to", "", "Required exclusive recorded-at upper bound")
	list.Flags().IntVar(&limit, "limit", 0, "Required limit 1..1000")
	list.Flags().StringVar(&run, "run", "", "Target run ID")
	root.AddCommand(list)
	return root
}
func newCohortCommand() *cobra.Command {
	root := &cobra.Command{Use: "cohort", Short: "Pin a bounded synthetic problem or reference selection"}
	var profile, from, to, endpoint string
	request := cohortRequest{}
	create := &cobra.Command{Use: "create <id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		request.ID = args[0]
		file, err := os.Open(profile)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		decoder := json.NewDecoder(io.LimitReader(file, 65537))
		decoder.DisallowUnknownFields()
		if info, e := file.Stat(); e != nil || info.Size() > 65536 {
			return fmt.Errorf("profile file exceeds bounds or cannot be read")
		}
		if err = decodeOne(decoder, &request.Profile); err != nil {
			return err
		}
		request.Selection.From, request.Selection.To, err = timeBounds(from, to)
		if err != nil {
			return err
		}
		var c model.Cohort
		if err = metadataPost(cmd.Context(), endpoint, "/api/cohorts", request, &c); err != nil {
			return err
		}
		return jsonOutput(cmd, c)
	}}
	f := create.Flags()
	f.StringVar(&endpoint, "endpoint", "http://127.0.0.1:4318", "Existing serving writer endpoint")
	f.StringVar(&profile, "profile", "", "Analysis profile JSON file")
	f.StringVar(&from, "from", "", "Required run start lower bound")
	f.StringVar(&to, "to", "", "Required exclusive run start upper bound")
	f.IntVar(&request.Selection.Limit, "limit", 0, "Required eligible-population limit 1..1000")
	f.StringVar(&request.Selection.Dimension, "dimension", "", "Mark/label dimension")
	f.StringVar(&request.Selection.Value, "value", "", "Selected active mark/label value")
	f.StringVar(&request.Selection.Predicate, "predicate", "", "Captured value selector for equality")
	f.StringVar(&request.Selection.Equals, "equals", "", "Expected JSON scalar for predicate")
	root.AddCommand(create)
	return root
}
func newExplainCohortCommand(database *string) *cobra.Command {
	var asJSON bool
	var viewerURL string
	cmd := &cobra.Command{Use: "explain-cohort <problem-cohort> <reference-cohort>", Short: "Report observations and cohort associations; no causal claims", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		reader, err := store.OpenReader(cmd.Context(), *database)
		if err != nil {
			return err
		}
		defer func() { _ = reader.Close() }()
		result, err := cohort.Analyze(cmd.Context(), reader, args[0], args[1])
		if err != nil {
			return err
		}
		if asJSON {
			return jsonOutput(cmd, result)
		}
		var text strings.Builder
		fmt.Fprintf(&text, "%s; %d problem runs / %d sessions vs %d reference runs / %d sessions; unknown sessions %d; shared clusters %d; independence unknown\nSelection exclusions: problem=%v reference=%v\n", result.Status, result.ProblemRuns, result.ProblemSessions, result.ReferenceRuns, result.ReferenceSessions, result.UnknownSessions, result.SharedSessions, result.Problem.Excluded, result.Reference.Excluded)
		for i, d := range result.Differences {
			fmt.Fprintf(&text, "%d. %.6f %s [%s/%s]: %s\n   observed %d/%d problem vs %d/%d reference; missing %d/%d; sessions %d/%d; shared clusters %d; independence unknown\n   [%s]\n", i+1, d.Score, d.Type, d.Tier, d.Status, d.Summary, d.ProblemObserved, d.ProblemRuns, d.ReferenceObserved, d.ReferenceRuns, d.ProblemMissing, d.ReferenceMissing, d.ProblemSessions, d.ReferenceSessions, d.SharedSessions, strings.Join(d.EvidenceIDs, ", "))
			for _, u := range d.EvidenceURLs {
				fmt.Fprintln(&text, "   "+strings.TrimRight(viewerURL, "/")+u)
			}
		}
		fmt.Fprintf(&text, "Declared-version invariant (observation, no baseline): %v\n", result.InvariantCounts)
		_, err = io.WriteString(cmd.OutOrStdout(), text.String())
		return err
	}}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Structured report with pinned selections, sample sizes and invariant evidence")
	cmd.Flags().StringVar(&viewerURL, "viewer-url", "http://127.0.0.1:4318", "Viewer origin for clickable evidence links in text output")
	return cmd
}

func metadataHandler(s *store.Store) http.Handler {
	mux := http.NewServeMux()
	handle := func(path string, run func(context.Context, *json.Decoder) (any, error)) {
		mux.HandleFunc("POST "+path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				http.Error(w, "same-origin JSON required", http.StatusForbidden)
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || u.Scheme != "http" || u.Host != r.Host {
					http.Error(w, "same origin required", http.StatusForbidden)
					return
				}
			}
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536))
			decoder.DisallowUnknownFields()
			value, err := run(ctx, decoder)
			if err != nil {
				status := 400
				if err == store.QueueFull {
					status = 503
					w.Header().Set("Retry-After", "1")
				}
				http.Error(w, err.Error(), status)
				return
			}
			_ = json.NewEncoder(w).Encode(value)
		})
	}
	handle("/api/annotations", func(ctx context.Context, d *json.Decoder) (any, error) {
		var a model.Annotation
		if err := decodeOne(d, &a); err != nil {
			return nil, err
		}
		a.RecordedAt = time.Now().UnixNano()
		if err := s.SaveAnnotation(ctx, a); err != nil {
			return nil, err
		}
		return s.Annotation(ctx, a.ID)
	})
	handle("/api/cohorts", func(ctx context.Context, d *json.Decoder) (any, error) {
		var request cohortRequest
		if err := decodeOne(d, &request); err != nil {
			return nil, err
		}
		c, err := s.SelectCohort(ctx, request.ID, request.Profile, request.Selection)
		if err != nil {
			return nil, err
		}
		if err = s.SaveCohort(ctx, c); err != nil {
			return nil, err
		}
		return c, nil
	})
	return mux
}
