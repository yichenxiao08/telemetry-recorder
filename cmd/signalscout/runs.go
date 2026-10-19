package main

import (
	"encoding/json"
	"errors"

	"github.com/signalscout/signalscout/internal/store"
	"github.com/spf13/cobra"
)

func newRunsCommand(database *string) *cobra.Command {
	root := &cobra.Command{Use: "runs", Short: "Inspect stored runs"}
	var workload, outcome string
	list := &cobra.Command{Use: "list", Short: "List runs as JSON", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) (result error) {
		reader, err := store.OpenReader(cmd.Context(), *database)
		if err != nil {
			return err
		}
		defer func() { result = errors.Join(result, reader.Close()) }()
		runs, err := reader.List(cmd.Context(), workload, outcome)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(runs)
	}}
	list.Flags().StringVar(&workload, "workload", "", "Exact workload filter")
	list.Flags().StringVar(&outcome, "outcome", "", "Outcome filter: pass, fail or degraded")
	show := &cobra.Command{Use: "show <run_id>", Short: "Show a run with evidence and context as JSON", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) (result error) {
		reader, err := store.OpenReader(cmd.Context(), *database)
		if err != nil {
			return err
		}
		defer func() { result = errors.Join(result, reader.Close()) }()
		detail, err := reader.Show(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(detail)
	}}
	root.AddCommand(list, show)
	return root
}
