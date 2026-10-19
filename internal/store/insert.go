package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

const signalInsertRows = 128

// SQLite's Go driver prepares SQL at execution time even for database/sql
// statements. Bound multi-row inserts amortize that work and argument handling.
// All chunks still belong to the job's one transaction, with one durable ack.
type signalInsert struct {
	args []any
	rows int
}

func (b *signalInsert) add(ctx context.Context, tx *sql.Tx, args ...any) error {
	b.args = append(b.args, args...)
	b.rows++
	if b.rows == signalInsertRows {
		return b.flush(ctx, tx)
	}
	return nil
}

func (b *signalInsert) flush(ctx context.Context, tx *sql.Tx) error {
	if b.rows == 0 {
		return nil
	}
	query := "INSERT INTO signals VALUES" + strings.TrimSuffix(strings.Repeat("(?,?,?,?,?,?,?,?),", b.rows), ",") + ` ON CONFLICT(evidence_id) DO UPDATE SET data=excluded.data,received_at=excluded.received_at,ts_ns=excluded.ts_ns,end_ns=excluded.end_ns,resource_id=excluded.resource_id WHERE signals.run_id=excluded.run_id AND signals.kind=excluded.kind`
	written, err := tx.ExecContext(ctx, query, b.args...)
	if err != nil {
		return fmt.Errorf("write signals: %w", err)
	}
	changed, err := written.RowsAffected()
	if err != nil {
		return fmt.Errorf("check signal writes: %w", err)
	}
	// Each valid insert or idempotent update affects exactly one row. A rejected
	// ownership/kind conflict affects zero, so any short count rolls back every
	// chunk and run/resource update, including earlier successful chunks.
	if changed != int64(b.rows) {
		return fmt.Errorf("evidence cannot change run ownership or kind: wrote %d of %d", changed, b.rows)
	}
	clear(b.args)
	b.args = b.args[:0]
	b.rows = 0
	return nil
}
