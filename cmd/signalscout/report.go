package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	evidence "github.com/signalscout/signalscout/internal/report"
	"github.com/signalscout/signalscout/internal/store"
	"github.com/spf13/cobra"
)

func reportFile(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	body, err := io.ReadAll(io.LimitReader(f, (32<<20)+1))
	if err != nil {
		return err
	}
	return evidence.Decode(body, v)
}

func newReportCommand(database *string) *cobra.Command {
	root := &cobra.Command{Use: "report", Short: "Compose and verify structured evidence (no generated hypotheses)"}
	var request, input, hypotheses string
	var buildJSON, verifyJSON bool
	build := &cobra.Command{Use: "build", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if request == "" {
			return fmt.Errorf("--request required")
		}
		var q evidence.Request
		if err := reportFile(request, &q); err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()
		r, err := store.OpenReader(ctx, *database)
		if err != nil {
			return err
		}
		defer func() { _ = r.Close() }()
		doc, facts, err := evidence.Build(ctx, r, q)
		if err != nil {
			return err
		}
		if hypotheses != "" {
			var supplied struct {
				Version int              `json:"version"`
				Claims  []evidence.Claim `json:"claims"`
			}
			if err = reportFile(hypotheses, &supplied); err != nil {
				return err
			}
			if supplied.Version != 1 || len(supplied.Claims) > 64 {
				return fmt.Errorf("hypothesis input requires version 1 and <=64 claims")
			}
			for _, c := range supplied.Claims {
				if c.Tier != "hypothesis" {
					return fmt.Errorf("supplied input accepts hypothesis tier only")
				}
			}
			doc.Claims = append(doc.Claims, supplied.Claims...)
		}
		v := evidence.Verify(doc.Claims, facts)
		if err = evidence.Size(doc); err != nil {
			return err
		}
		if v.Invalid > 0 {
			return fmt.Errorf("invalid draft claims: %+v", v.Claims)
		}
		if buildJSON {
			return jsonOutput(cmd, doc)
		}
		for _, c := range doc.Claims {
			if _, err = fmt.Fprintf(cmd.OutOrStdout(), "%s [%s; %s] %s\n  evidence: %v\n  quantities: %v\n", c.ID, c.Tier, c.State, c.Text, c.Evidence, c.Numbers); err != nil {
				return err
			}
			if c.Association != nil {
				if err = jsonOutput(cmd, c.Association); err != nil {
					return err
				}
			}
			if c.Hypothesis != nil {
				if err = jsonOutput(cmd, c.Hypothesis); err != nil {
					return err
				}
			}
		}
		for _, note := range doc.Notes {
			if _, err = fmt.Fprintln(cmd.OutOrStdout(), note); err != nil {
				return err
			}
		}
		return nil
	}}
	verify := &cobra.Command{Use: "verify", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if input == "" {
			return fmt.Errorf("--input required")
		}
		var doc evidence.Document
		if err := reportFile(input, &doc); err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()
		r, err := store.OpenReader(ctx, *database)
		if err != nil {
			return err
		}
		defer func() { _ = r.Close() }()
		v, err := evidence.VerifyDocument(ctx, r, doc)
		if err != nil {
			return err
		}
		if verifyJSON {
			err = jsonOutput(cmd, v)
		} else {
			for _, c := range v.Claims {
				if _, err = fmt.Fprintf(cmd.OutOrStdout(), "%s: %s — %s\n", c.ID, c.Status, c.Reason); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Claims: valid %d; invalid %d; unsupported %d; support rate %.6f\nCitations: %d/%d; validity rate %.6f\n", v.Valid, v.Invalid, v.Unsupported, v.ClaimSupportRate, v.ValidCitations, v.Citations, v.CitationValidityRate)
		}
		if err != nil {
			return err
		}
		if v.Invalid > 0 {
			return fmt.Errorf("report has invalid claims")
		}
		return nil
	}}
	build.Flags().StringVar(&request, "request", "", "Versioned bounded source selection JSON")
	build.Flags().StringVar(&hypotheses, "hypotheses", "", "Optional supplied hypothesis JSON; never generated")
	build.Flags().BoolVar(&buildJSON, "json", false, "Structured report")
	verify.Flags().StringVar(&input, "input", "", "Report JSON to verify against the store")
	verify.Flags().BoolVar(&verifyJSON, "json", false, "Per-claim results and report-level rates")
	root.AddCommand(build, verify)
	return root
}
