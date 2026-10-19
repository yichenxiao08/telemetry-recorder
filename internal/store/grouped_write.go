package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/signalscout/signalscout/internal/runs"
)

const maxIngestJobsPerCommit = 4

// Only the writer calls writeJobs. Each admitted job keeps its own receipt time
// and savepoint; every successful caller waits for the shared commit. A failed
// job cannot roll back its neighbors. A failed commit acknowledges none of them.
func (s *Store) writeJobs(ctx context.Context, jobs []job) []error {
	errs := make([]error, len(jobs))
	if len(jobs) == 0 {
		return errs
	}
	failAll := func(err error) []error {
		for i := range errs {
			errs[i] = errors.Join(errs[i], err)
		}
		return errs
	}
	tx, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return failAll(fmt.Errorf("begin ingest: %w", err))
	}
	defer func() { _ = tx.Rollback() }()
	updates := make([]map[string]runs.State, len(jobs))
	grouped := len(jobs) > 1
	for i, j := range jobs {
		if grouped {
			if _, err = tx.ExecContext(ctx, "SAVEPOINT ingest_job"); err != nil {
				return failAll(err)
			}
		}
		updates[i], errs[i] = s.writeBatch(ctx, tx, *j.batch, j.now)
		if errs[i] != nil {
			updates[i] = nil
			if !grouped {
				return errs
			}
			if _, err = tx.ExecContext(ctx, "ROLLBACK TO ingest_job"); err != nil {
				return failAll(err)
			}
		}
		if grouped {
			if _, err = tx.ExecContext(ctx, "RELEASE ingest_job"); err != nil {
				return failAll(err)
			}
		}
	}
	if err = tx.Commit(); err != nil {
		// Some SQLite commit errors (e.g. deferred foreign keys) leave the native
		// transaction active after database/sql marks Tx done. Clear it before
		// returning this connection to subsequent jobs on the same sole writer.
		_, _ = s.writer.ExecContext(context.Background(), "ROLLBACK")
		return failAll(fmt.Errorf("commit ingest: %w", err))
	}
	for _, states := range updates {
		for id, state := range states {
			if state.Run.Closed {
				delete(s.open, id)
			} else {
				s.open[id] = state.LastSeen
			}
		}
	}
	return errs
}

func (s *Store) reportError(err error) {
	if err != nil && !errors.Is(err, RunChanged) {
		select {
		case s.errors <- err:
		default:
		}
	}
}

func (s *Store) completeJob(j job, err error) { s.reportError(err); j.done <- err }
