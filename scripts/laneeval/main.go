// laneeval is the narrow synthetic lane-01 orchestrator, not the M5 harness.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func run(ctx context.Context) error {
	out := flag.String("out", "", "fresh output directory (default under .cache)")
	flag.Parse()
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	if *out == "" {
		*out = filepath.Join(".cache", "lane-01", time.Now().UTC().Format("20060102T150405.000000000"))
	}
	absolute, err := filepath.Abs(*out)
	if err != nil {
		return err
	}
	uv, err := exec.LookPath("uv")
	if err != nil {
		uv = filepath.Join(root, ".cache", "uv-tool", "bin", "uv.exe")
		if _, err = os.Stat(uv); err != nil {
			return fmt.Errorf("uv required: %w", err)
		}
	}
	command := exec.CommandContext(ctx, uv, "run", "--directory", filepath.Join(root, "adapters", "provenance-python"), "--frozen", "python", "lane01.py", "--out", absolute)
	command.Env = append(os.Environ(), "UV_PYTHON_INSTALL_DIR="+filepath.Join(root, ".cache", "uv-python"), "UV_CACHE_DIR="+filepath.Join(root, ".cache", "uv-cache"))
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err = command.Run(); err != nil {
		return fmt.Errorf("lane-01: %w", err)
	}
	fmt.Println("Synthetic lane-01 results:", filepath.Join(*out, "report.json"))
	return nil
}
func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
