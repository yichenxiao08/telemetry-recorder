package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

// auditContext compares complete ordered sets across a migration. It writes no
// SQLite data and deliberately includes all rule/source provenance, not samples.
func auditContext(ctx context.Context, path, out string) error {
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
	type digest struct {
		Rows   int64  `json:"rows"`
		SHA256 string `json:"sha256"`
	}
	r := map[string]digest{}
	for name, query := range map[string]string{
		"signals":   "SELECT evidence_id,kind,run_id,resource_id,ts_ns,end_ns,received_at,data FROM signals ORDER BY evidence_id",
		"resources": "SELECT resource_id,attrs,raw FROM resources ORDER BY resource_id",
		"runs":      "SELECT run_id,state FROM runs ORDER BY run_id",
		"links":     "SELECT run_id,evidence_id FROM context_links ORDER BY run_id,evidence_id",
		"matches":   "SELECT run_id,evidence_id,rule,matched_resource_id FROM context_matches ORDER BY run_id,evidence_id,rule,matched_resource_id",
	} {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return err
		}
		hash := sha256.New()
		encoder := json.NewEncoder(hash)
		columns, err := rows.Columns()
		if err != nil {
			_ = rows.Close()
			return err
		}
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		var n int64
		for rows.Next() {
			if err = rows.Scan(ptrs...); err != nil {
				_ = rows.Close()
				return err
			}
			if err = encoder.Encode(values); err != nil {
				_ = rows.Close()
				return err
			}
			n++
		}
		err = rows.Err()
		closeErr := rows.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		r[name] = digest{Rows: n, SHA256: fmt.Sprintf("%x", hash.Sum(nil))}
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(out, append(data, '\n'), 0644); err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
