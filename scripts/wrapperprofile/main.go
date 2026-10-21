// wrapperprofile runs the measurement-only Python probe diagnostics natively.
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
	out := flag.String("out", "", "fresh output directory")
	flag.Parse()
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	if *out == "" {
		*out = filepath.Join(".cache", "wrapper-profile", time.Now().UTC().Format("20060102T150405.000000000"))
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
	command := exec.CommandContext(ctx, uv, "run", "--directory", filepath.Join(root, "adapters", "provenance-python"), "--frozen", "python", "profile_wrapper.py", "--out", absolute)
	command.Env = append(os.Environ(), "UV_PYTHON_INSTALL_DIR="+filepath.Join(root, ".cache", "uv-python"), "UV_CACHE_DIR="+filepath.Join(root, ".cache", "uv-cache"))
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err = command.Run(); err != nil {
		return fmt.Errorf("wrapper profiling: %w", err)
	}
	fmt.Println("Measurement-only results:", absolute)
	return nil
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
