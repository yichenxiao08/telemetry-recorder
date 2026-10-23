package main

import (
	"fmt"
	"github.com/signalscout/signalscout/internal/hypothesis"
	"github.com/signalscout/signalscout/internal/report"
	"github.com/spf13/cobra"
	"io"
	"os"
)

func newProposeCommand() *cobra.Command {
	var input string
	cmd := &cobra.Command{Use: "propose", Short: "Emit mock hypothesis JSON and text-only experiment proposals", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if input == "" {
			return fmt.Errorf("--input required")
		}
		f, err := os.Open(input)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		body, err := io.ReadAll(io.LimitReader(f, hypothesis.MaxInputBytes+1))
		if err != nil {
			return err
		}
		if len(body) > hypothesis.MaxInputBytes {
			return fmt.Errorf("input exceeds mock byte budget")
		}
		var doc report.Document
		if err = report.Decode(body, &doc); err != nil {
			return err
		}
		out, err := hypothesis.Generate(cmd.Context(), doc, hypothesis.Mock{})
		if err != nil {
			return err
		}
		return jsonOutput(cmd, out)
	}}
	cmd.Flags().StringVar(&input, "input", "", "Existing structured report JSON (at most one MiB)")
	return cmd
}
