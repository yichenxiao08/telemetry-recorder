package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
)

func newRootCommand() *cobra.Command {
	root := &cobra.Command{Use: "signalscout", Short: "Store telemetry and inspect workload runs", SilenceUsage: true, SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	var database string
	root.PersistentFlags().StringVar(&database, "db", "signalscout.db", "SQLite database path")
	root.AddCommand(newServeCommand(&database), newRunsCommand(&database), newDiffCommand(&database))
	root.AddCommand(newMarkCommand(), newLabelsCommand(&database), newCohortCommand(&database), newExplainCohortCommand(&database))
	root.AddCommand(newTraceCommand(&database))
	root.CompletionOptions.DisableDefaultCmd = true
	return root
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := newRootCommand().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
