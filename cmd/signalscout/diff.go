package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/diff"
	"github.com/signalscout/signalscout/internal/model"
	"github.com/spf13/cobra"
)

func newDiffCommand(database *string) *cobra.Command {
	var n int
	var before string
	var asJSON bool
	command := &cobra.Command{Use: "diff <failing_run_id>", Short: "Rank differences against earlier passing runs", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		result, err := diff.Analyze(cmd.Context(), *database, args[0], model.BaselineOptions{N: n, ExplicitN: cmd.Flags().Changed("baseline-n"), Before: before})
		if err != nil {
			return err
		}
		if asJSON {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(result)
		}
		var text strings.Builder
		fmt.Fprintf(&text, "%s: baseline=%s selected=%d requested=%d before=%s; %.3f ms (including persistence)\n", result.ID, result.BaselineSpec.Mode, len(result.BaselineSpec.RunIDs), result.BaselineSpec.N, time.Unix(0, result.BaselineSpec.BeforeNS).UTC().Format(time.RFC3339Nano), float64(result.ElapsedNS)/float64(time.Millisecond))
		if result.Status != "ok" {
			fmt.Fprintln(&text, result.Message)
		}
		for i, d := range result.Differences {
			fmt.Fprintf(&text, "%d. %.6f %s: %s\n   [%s]\n", i+1, d.Score, d.Type, d.Summary, strings.Join(d.EvidenceIDs, ", "))
			for _, related := range d.Related {
				fmt.Fprintf(&text, "   Related (%.6f): %s [%s]\n", related.Score, related.Summary, strings.Join(related.EvidenceIDs, ", "))
			}
		}
		_, err = io.WriteString(cmd.OutOrStdout(), text.String())
		return err
	}}
	command.Flags().IntVar(&n, "baseline-n", 40, "Most recent N closed passing runs (minimum usable baseline: 10)")
	command.Flags().StringVar(&before, "baseline-before", "", "Exclusive baseline cutoff: run ID or RFC3339 timestamp")
	command.Flags().BoolVar(&asJSON, "json", false, "Print ranked differences and baseline specification as JSON")
	return command
}
