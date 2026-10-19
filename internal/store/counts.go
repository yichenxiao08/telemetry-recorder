package store

import (
	"context"
	"fmt"
)

// Counts verifies persisted signal totals for the ingest measurement helper.
func (r *Reader) Counts(ctx context.Context) (map[string]int64, error) {
	rows, err := r.reader.QueryContext(ctx, "SELECT kind,count(*) FROM signals GROUP BY kind")
	if err != nil {
		return nil, fmt.Errorf("count evidence: %w", err)
	}
	defer func() { _ = rows.Close() }()
	result := map[string]int64{}
	for rows.Next() {
		var kind string
		var count int64
		if err = rows.Scan(&kind, &count); err != nil {
			return nil, fmt.Errorf("read count: %w", err)
		}
		result[kind] = count
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate counts: %w", err)
	}
	return result, nil
}
