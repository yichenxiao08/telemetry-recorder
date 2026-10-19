package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"
)

func profileContext(ctx context.Context, path, out, cpu, alloc string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}
	if filepath.VolumeName(absolute) != "" {
		u.Path = "/" + u.Path
	}
	db, err := sql.Open("sqlite", u.String()+"?mode=ro&_pragma=query_only(1)")
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	var f *os.File
	if cpu != "" {
		f, err = os.Create(cpu)
		if err != nil {
			return err
		}
		if err = pprof.StartCPUProfile(f); err != nil {
			_ = f.Close()
			return err
		}
	}
	deadline, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	start := time.Now()
	var count int64
	queryErr := db.QueryRowContext(deadline, "SELECT count(*) FROM context_matches").Scan(&count)
	r := struct {
		SQL     string  `json:"sql"`
		Seconds float64 `json:"seconds"`
		Count   *int64  `json:"count,omitempty"`
		Error   string  `json:"error,omitempty"`
		Method  string  `json:"method"`
	}{SQL: "SELECT count(*) FROM context_matches", Seconds: time.Since(start).Seconds(), Method: "Read-only whole-dataset provenance count, one-minute profiling deadline. Timeout is failed/censored; a missing count is not zero. This is a storage-cardinality diagnostic, not a bounded explorer query."}
	if queryErr != nil {
		r.Error = queryErr.Error()
	} else {
		r.Count = &count
	}
	if f != nil {
		pprof.StopCPUProfile()
		if err = f.Close(); err != nil {
			return err
		}
	}
	if alloc != "" {
		f, err = os.Create(alloc)
		if err != nil {
			return err
		}
		err = errors.Join(pprof.Lookup("allocs").WriteTo(f, 0), f.Close())
		if err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(out, append(data, '\n'), 0644); err != nil {
		return err
	}
	fmt.Println(string(data))
	return queryErr
}
