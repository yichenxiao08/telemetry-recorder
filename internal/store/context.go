package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// Scopes describe the candidate context set, without copying one link for each
// ambient evidence item. Reads resolve current evidence and retain every rule.
// Scope changes commit with run changes on the same writer. Late ambient data
// therefore needs no backfill and cannot leave a stale link after an update.
func (s *Store) refreshContext(ctx context.Context, tx *sql.Tx, runIDs []string) error {
	if len(runIDs) == 0 {
		return nil
	}
	data, err := json.Marshal(runIDs)
	if err != nil {
		return fmt.Errorf("encode context scope: %w", err)
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM context_scopes WHERE run_id IN (SELECT value FROM json_each(?))", string(data)); err != nil {
		return fmt.Errorf("refresh changed run context: %w", err)
	}
	return s.insertContextScopes(ctx, tx, "r.run_id IN (SELECT value FROM json_each(?))", []any{string(data)})
}

func (s *Store) backfillContext(ctx context.Context) error {
	var before, after int64
	err := s.writer.QueryRowContext(ctx, "SELECT before_ns,after_ns FROM context_settings WHERE singleton=1").Scan(&before, &after)
	if err == nil && before == int64(s.options.ContextBefore) && after == int64(s.options.ContextAfter) {
		return nil
	}
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("read context checkpoint: %w", err)
	}
	tx, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin context backfill: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, "DELETE FROM context_scopes"); err != nil {
		return fmt.Errorf("reset context scopes: %w", err)
	}
	if err = s.insertContextScopes(ctx, tx, "1", nil); err != nil {
		return err
	}
	// This checkpoint is valid for all history: every subsequent run change
	// updates its scopes atomically. An unchanged restart reads only this row;
	// migration or changed padding rebuilds run/resource scopes, not evidence.
	if _, err = tx.ExecContext(ctx, "INSERT OR REPLACE INTO context_settings VALUES(1,?,?)", int64(s.options.ContextBefore), int64(s.options.ContextAfter)); err != nil {
		return fmt.Errorf("checkpoint context padding: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit context backfill: %w", err)
	}
	return nil
}

func (s *Store) insertContextScopes(ctx context.Context, tx *sql.Tx, scope string, args []any) error {
	rules := []struct{ name, value, predicate string }{
		{"exact_resource", "source.resource_id", "1"},
		{"host.name", `json_extract(source.attrs,'$."host.name"')`, `json_type(source.attrs,'$."host.name"')='text' AND json_extract(source.attrs,'$."host.name"')<>''`},
		{"service.instance.id", `json_extract(source.attrs,'$."service.instance.id"')`, `json_type(source.attrs,'$."service.instance.id"')='text' AND json_extract(source.attrs,'$."service.instance.id"')<>''`},
	}
	for _, rule := range rules {
		// Only fixed SQL fragments above and internal scope predicates are used.
		query := `INSERT INTO context_scopes SELECT r.run_id,?,source.resource_id,` + rule.value + `,r.started_at-?,r.ended_at+?
		 FROM runs r JOIN run_resources rr ON rr.run_id=r.run_id
		 JOIN resources source ON source.resource_id=rr.resource_id
		 WHERE r.closed=1 AND (` + scope + `) AND (` + rule.predicate + `)
		 AND EXISTS(SELECT 1 FROM signals span WHERE span.run_id=r.run_id AND span.kind='span' AND span.resource_id=source.resource_id)`
		values := append([]any{rule.name, int64(s.options.ContextBefore), int64(s.options.ContextAfter)}, args...)
		if _, err := tx.ExecContext(ctx, query, values...); err != nil {
			return fmt.Errorf("record context scope %s: %w", rule.name, err)
		}
	}
	return nil
}
