// startupbench measures normal intake startup on a closed, saved database copy.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/signalscout/signalscout/internal/store"
)

type report struct {
	Database     string           `json:"database"`
	Go           string           `json:"go"`
	Seconds      float64          `json:"startup_seconds"`
	CloseSeconds float64          `json:"close_seconds"`
	Error        string           `json:"error,omitempty"`
	Bytes        int64            `json:"database_bytes"`
	Counts       map[string]int64 `json:"counts"`
}

func copyClosed(source, destination string) (resultErr error) {
	if info, err := os.Stat(source + "-wal"); err == nil && info.Size() != 0 {
		return fmt.Errorf("source has a nonempty WAL; stop its writer before copying")
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, in.Close()) }()
	if err = os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	return errors.Join(err, out.Close())
}

func run() error {
	source := flag.String("source", "", "Optional closed source database; destination must not exist")
	path := flag.String("db", "", "Database copy to measure (normal writer startup)")
	out := flag.String("out", "", "Result JSON path")
	cpu := flag.String("cpuprofile", "", "CPU profile of Open only")
	alloc := flag.String("allocprofile", "", "Cumulative allocation profile through Open")
	timeout := flag.Duration("timeout", 10*time.Minute, "Startup deadline; expiration is a failed measurement")
	flag.Parse()
	if *path == "" || *out == "" {
		return fmt.Errorf("db and out required")
	}
	if *source != "" {
		if err := copyClosed(*source, *path); err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	var profile *os.File
	if *cpu != "" {
		var err error
		profile, err = os.Create(*cpu)
		if err != nil {
			return err
		}
		if err = pprof.StartCPUProfile(profile); err != nil {
			_ = profile.Close()
			return err
		}
	}
	start := time.Now()
	s, openErr := store.Open(ctx, *path, store.Options{QueueSize: 64, IdleTimeout: time.Hour, SweepInterval: time.Hour, ContextBefore: 30 * time.Second, ContextAfter: 5 * time.Second})
	r := report{Database: filepath.ToSlash(*path), Go: runtime.Version(), Seconds: time.Since(start).Seconds(), Counts: map[string]int64{}}
	if profile != nil {
		pprof.StopCPUProfile()
		if err := profile.Close(); err != nil {
			return err
		}
	}
	if *alloc != "" {
		f, err := os.Create(*alloc)
		if err != nil {
			return err
		}
		err = errors.Join(pprof.Lookup("allocs").WriteTo(f, 0), f.Close())
		if err != nil {
			return err
		}
	}
	if openErr != nil {
		r.Error = openErr.Error()
	} else {
		start = time.Now()
		if err := s.Close(); err != nil {
			r.Error = err.Error()
		}
		r.CloseSeconds = time.Since(start).Seconds()
	}
	info, err := os.Stat(*path)
	if err != nil {
		return err
	}
	r.Bytes = info.Size()
	// Counts are outside the startup timing and use a read-only connection.
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(*path)+"?mode=ro")
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	for _, table := range []string{"signals", "runs", "resources", "context_links", "context_matches"} {
		var count int64
		if err = db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil {
			return err
		}
		r.Counts[table] = count
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(*out, append(data, '\n'), 0644); err != nil {
		return err
	}
	fmt.Println(string(data))
	if r.Error != "" {
		return errors.New(r.Error)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
