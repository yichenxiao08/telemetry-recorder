package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/doctor"
	"github.com/spf13/cobra"
)

func newDoctorCommand(database *string) *cobra.Command {
	return doctorCommand(database, doctor.SystemEnvironment())
}

func doctorCommand(database *string, environment doctor.Environment) *cobra.Command {
	var options doctor.Options
	var jsonPath string
	cmd := &cobra.Command{Use: "doctor", Short: "Report local Python, dependency, database and port readiness", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			options.Database = *database
			for _, value := range []string{options.Database, options.Adapter, options.Python, options.Listen, jsonPath} {
				if strings.ContainsAny(value, "\x00\r\n") {
					return fmt.Errorf("doctor paths and addresses cannot contain control characters")
				}
			}
			if jsonPath != "" {
				jsonAbs, err := filepath.Abs(jsonPath)
				if err != nil {
					return err
				}
				dbAbs, err := filepath.Abs(options.Database)
				if err != nil {
					return err
				}
				if strings.EqualFold(jsonAbs, dbAbs) {
					return fmt.Errorf("JSON output must differ from the database path")
				}
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 35*time.Second)
			defer cancel()
			result := doctor.Inspect(ctx, options, environment)
			if _, err := fmt.Fprint(cmd.OutOrStdout(), result.Text()); err != nil {
				return err
			}
			if jsonPath != "" {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return err
				}
				// Never overwrite an existing file, including a database or a report.
				file, err := os.OpenFile(jsonPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
				if err != nil {
					return fmt.Errorf("create doctor JSON (choose a new path): %w", err)
				}
				_, writeErr := file.Write(append(data, '\n'))
				closeErr := file.Close()
				if writeErr != nil {
					return writeErr
				}
				if closeErr != nil {
					return closeErr
				}
			}
			if result.Summary.Fail > 0 {
				return fmt.Errorf("doctor found %d failed checks", result.Summary.Fail)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&options.Adapter, "adapter", "adapters/provenance-python", "Probe adapter directory (uses its existing .venv when available)")
	cmd.Flags().StringVar(&options.Python, "python", "", "Explicit Python executable; otherwise adapter .venv, py -3, then python")
	cmd.Flags().StringVar(&options.Listen, "listen", "127.0.0.1:4318", "Loopback serve address to bind and immediately release; empty skips check")
	cmd.Flags().StringVar(&jsonPath, "json", "", "Write schema-versioned JSON to a new file (never overwrite)")
	return cmd
}
