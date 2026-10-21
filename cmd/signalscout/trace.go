package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/lineage"
	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
	"github.com/spf13/cobra"
)

type traceOutput struct {
	Scope        store.TraceScope `json:"scope"`
	Trace        lineage.Report   `json:"trace"`
	Compare      *lineage.Report  `json:"compare,omitempty"`
	TotalVisited int              `json:"total_nodes_visited"`
	NodeBudget   int              `json:"shared_node_budget"`
}

func newTraceCommand(database *string) *cobra.Command {
	var from, to, run, profilePath, compare string
	var limit, depth, nodes int
	var asJSON bool
	cmd := &cobra.Command{Use: "trace <output-ref|evidence-id> | <run-id> <output-name>", Short: "Describe one output's recorded lineage (observations only)", Args: cobra.RangeArgs(1, 2), RunE: func(cmd *cobra.Command, args []string) error {
		if from == "" || to == "" || !cmd.Flags().Changed("limit") {
			return fmt.Errorf("--from, --to and --limit are required")
		}
		start, end, err := timeBounds(from, to)
		if err != nil {
			return err
		}
		if nodes < 1 || nodes > 1000 || depth < 0 || depth > 64 {
			return fmt.Errorf("--nodes must be 1..1000; --depth must be 0..64")
		}
		target := args[0]
		if len(args) == 2 {
			if run != "" && run != args[0] {
				return fmt.Errorf("run argument conflicts with --run")
			}
			run = args[0]
			target = "value:" + run + "/" + args[1]
		}
		var profile *model.AnalysisProfile
		if profilePath != "" {
			profile, err = traceProfile(profilePath)
			if err != nil {
				return err
			}
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()
		reader, err := store.OpenReader(ctx, *database)
		if err != nil {
			return err
		}
		defer func() { _ = reader.Close() }()
		targets := []string{target}
		if compare != "" {
			targets = append(targets, compare)
		}
		data, err := reader.TraceRead(ctx, store.TraceReadRequest{Targets: targets, RunID: run, From: start, To: end, Limit: limit})
		if err != nil {
			return err
		}
		if len(args) == 2 {
			target = lineage.NamedTarget(data.Signals, run, args[1])
		}
		result := traceOutput{Scope: data.Scope, Trace: lineage.Walk(data.Signals, target, lineage.Options{Depth: depth, Nodes: nodes}, profile), NodeBudget: nodes}
		result.TotalVisited = result.Trace.Visited
		if compare != "" {
			remaining := nodes - result.TotalVisited
			other := lineage.Report{Target: compare, EvidenceTier: lineage.Tier, Nodes: []lineage.Node{}, Budget: lineage.Options{Depth: depth, Nodes: remaining}, Truncations: 1, Findings: []lineage.Finding{{Kind: "node budget", Ref: compare, Detail: "unknown: shared node budget exhausted"}}}
			if remaining > 0 {
				other = lineage.Walk(data.Signals, compare, lineage.Options{Depth: depth, Nodes: remaining}, profile)
			}
			result.Compare = &other
			result.TotalVisited += other.Visited
		}
		if asJSON {
			return jsonOutput(cmd, result)
		}
		if _, err = fmt.Fprintf(cmd.OutOrStdout(), "Negative-search scope: [%s, %s); limit %d records (+ at most one sentinel); partitions %q; read %d; native lookups %d; truncated %t\n%s\n%s", from, to, limit, data.Scope.Runs, data.Scope.RecordsRead, data.Scope.NativeLookups, data.Scope.Truncated, data.Scope.NegativeSearch, lineage.Text(result.Trace)); err != nil {
			return err
		}
		if result.Compare != nil {
			if _, err = fmt.Fprintln(cmd.OutOrStdout(), "\nDescriptive comparison; no ranking or score:\n"+lineage.SideBySide(result.Trace, *result.Compare)+"\n"+lineage.Text(*result.Compare)); err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Shared total nodes visited / budget: %d/%d\n", result.TotalVisited, nodes)
		return err
	}}
	cmd.Flags().StringVar(&from, "from", "", "Inclusive RFC3339 time (required)")
	cmd.Flags().StringVar(&to, "to", "", "Exclusive RFC3339 time (required)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Shared event/log read limit, 1..10000 (required)")
	cmd.Flags().StringVar(&run, "run", "", "Run partition for an opaque output ref")
	cmd.Flags().IntVar(&depth, "depth", 16, "Maximum backward depth, root=0 (0..64)")
	cmd.Flags().IntVar(&nodes, "nodes", 128, "Shared node budget (1..1000)")
	cmd.Flags().StringVar(&profilePath, "profile", "", "Optional analysis profile declaring the version invariant")
	cmd.Flags().StringVar(&compare, "compare-to", "", "Second output ref/evidence ID, descriptive only")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Print JSON")
	return cmd
}
func traceProfile(path string) (*model.AnalysisProfile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	body, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil {
		return nil, err
	}
	if len(body) > 65536 {
		return nil, fmt.Errorf("profile exceeds 64 KiB")
	}
	var p model.AnalysisProfile
	d := json.NewDecoder(strings.NewReader(string(body)))
	d.DisallowUnknownFields()
	if err = d.Decode(&p); err != nil {
		return nil, err
	}
	if d.Decode(new(any)) != io.EOF {
		return nil, fmt.Errorf("extra profile content")
	}
	if p.Invariant.Declared != "" {
		if err = lineage.ValidateInvariantProfile(p); err != nil {
			return nil, err
		}
	}
	return &p, nil
}
