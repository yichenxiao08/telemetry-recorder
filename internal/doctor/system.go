package doctor

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"time"
)

func SystemEnvironment() Environment {
	return Environment{Lookup: exec.LookPath, Run: runCommand, Executable: os.Executable, Version: buildVersion(), Stat: os.Stat,
		Writable: func(path string) error {
			f, err := os.OpenFile(path, os.O_RDWR, 0)
			if err != nil {
				return err
			}
			return f.Close()
		},
		Port: func(ctx context.Context, address string) error {
			var config net.ListenConfig
			listener, err := config.Listen(ctx, "tcp", address)
			if err != nil {
				return err
			}
			return listener.Close()
		},
	}
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	version := info.Main.Version
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			version += " revision=" + s.Value
		}
		if s.Key == "vcs.modified" {
			version += " modified=" + s.Value
		}
	}
	return strings.TrimSpace(version)
}

// Bounded output and deadlines prevent an unhealthy interpreter from flooding
// diagnostics or waiting indefinitely. Stderr and unparsed output are not shown.
func runCommand(ctx context.Context, path string, args []string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.WaitDelay = time.Second
	var output boundedOutput
	cmd.Stdout = &output
	cmd.Stderr = io.Discard
	err := cmd.Run()
	if output.overflow {
		return "", fmt.Errorf("diagnostic output exceeded limit")
	}
	return output.text.String(), err
}

type boundedOutput struct {
	text     strings.Builder
	overflow bool
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := 16384 - b.text.Len()
	if len(p) > remaining {
		p = p[:remaining]
		b.overflow = true
	}
	_, _ = b.text.Write(p)
	return n, nil
}
