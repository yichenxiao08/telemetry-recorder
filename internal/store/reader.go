package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
)

type Reader struct{ reader *sql.DB }

func databaseURI(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("database path: %w", err)
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}
	if filepath.VolumeName(absolute) != "" {
		u.Path = "/" + u.Path
	}
	return u.String(), nil
}

// OpenReader never creates databases, migrates, starts writers or closes runs.
func OpenReader(ctx context.Context, path string) (*Reader, error) {
	uri, err := databaseURI(path)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", uri+"?mode=ro&_pragma=busy_timeout(5000)&_pragma=query_only(1)")
	if err != nil {
		return nil, fmt.Errorf("open reader: %w", err)
	}
	db.SetMaxOpenConns(4)
	if err = db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect reader: %w", err)
	}
	return &Reader{reader: db}, nil
}

func (r *Reader) Close() error {
	if err := r.reader.Close(); err != nil {
		return fmt.Errorf("close reader: %w", err)
	}
	return nil
}
