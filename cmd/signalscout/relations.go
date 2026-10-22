package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
	"github.com/spf13/cobra"
)

func newRelationsCommand(database *string) *cobra.Command {
	var profile, from, to, cohortID string
	var limit, records int
	var asJSON bool
	cmd := &cobra.Command{Use: "relations", Short: "Check declared rules over recorded decisions (observations only)", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()
		var p *model.AnalysisProfile
		var b model.CohortSelection
		if cohortID != "" {
			if profile != "" || from != "" || to != "" || cmd.Flags().Changed("limit") || cmd.Flags().Changed("record-limit") {
				return fmt.Errorf("--cohort reads a pinned relation snapshot; do not combine with live selection flags")
			}
		} else {
			if profile == "" || from == "" || to == "" || !cmd.Flags().Changed("limit") {
				return fmt.Errorf("--profile, --from, --to and --limit required")
			}
			var err error
			p, err = traceProfile(profile)
			if err != nil {
				return err
			}
			b.From, b.To, err = timeBounds(from, to)
			if err != nil {
				return err
			}
			b.Limit, b.RecordLimit = limit, records
		}
		r, err := store.OpenReader(ctx, *database)
		if err != nil {
			return err
		}
		defer func() { _ = r.Close() }()
		var report model.RelationReport
		if cohortID != "" {
			c, e := r.Cohort(ctx, cohortID)
			if e != nil {
				return e
			}
			if c.Relations == nil {
				return fmt.Errorf("cohort has no pinned relation snapshot")
			}
			report = *c.Relations
		} else {
			report, err = r.Relations(ctx, *p, b)
			if err != nil {
				return err
			}
		}
		if asJSON {
			return jsonOutput(cmd, report)
		}
		if _, err = fmt.Fprintf(cmd.OutOrStdout(), "Evidence tier: %s\nStep %s; candidate runs %d; [%d,%d); run limit %d; record limit %d\n%s\n", report.Tier, report.Step, report.CandidateRuns, report.From, report.To, report.RunLimit, report.RecordLimit, report.SamplingNote); err != nil {
			return err
		}
		printRows := func(scope string, rows map[string]*model.RelationCounts) error {
			keys := make([]string, 0, len(rows))
			for k := range rows {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				c := rows[k]
				if _, e := fmt.Fprintf(cmd.OutOrStdout(), "%s rule %s: satisfied %d/%d; violated %d/%d; unknown %d/%d; not_applicable %d/%d recorded decisions; sampled records %d; violation evidence %v; evidence omitted %d\n", scope, k, c.Satisfied, c.Total, c.Violated, c.Total, c.Unknown, c.Total, c.NotApplicable, c.Total, c.SampledRecords, c.ViolationEvidence, c.EvidenceOmitted); e != nil {
					return e
				}
				for _, u := range c.ViolationURLs {
					if _, e := fmt.Fprintln(cmd.OutOrStdout(), "  "+u); e != nil {
						return e
					}
				}
			}
			return nil
		}
		if err = printRows("step "+report.Step, report.Totals); err != nil {
			return err
		}
		ids := make([]string, 0, len(report.Runs))
		for id := range report.Runs {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			if err = printRows("run "+id, report.Runs[id]); err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Runs without a supported decision (unknown capture, no fabricated calls): %v\n", report.RunsWithoutDecision)
		return err
	}}
	f := cmd.Flags()
	f.StringVar(&profile, "profile", "", "Versioned analysis profile JSON")
	f.StringVar(&from, "from", "", "Required run-start lower bound RFC3339")
	f.StringVar(&to, "to", "", "Required exclusive run-start upper bound RFC3339")
	f.IntVar(&limit, "limit", 0, "Required candidate-run limit 1..1000")
	f.IntVar(&records, "record-limit", 0, "Span/event read cap 1..1000000; omitted uses 100000; overflow rejects entire selection")
	f.StringVar(&cohortID, "cohort", "", "Read relation snapshot pinned at cohort creation")
	f.BoolVar(&asJSON, "json", false, "Structured observation counts and violation evidence")
	return cmd
}
