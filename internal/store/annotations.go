package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"modernc.org/sqlite"
)

type metadataWrite struct {
	annotation *model.Annotation
	cohort     *model.Cohort
}

const MetadataRejected queueError = "metadata validation rejected"

func validText(value string) bool { return value != "" && len(value) <= 256 }

func (s *Store) SaveAnnotation(ctx context.Context, annotation model.Annotation) error {
	if !validText(annotation.ID) || !validText(annotation.RunID) || !validText(annotation.Dimension) || !validText(annotation.Value) || !validText(annotation.Source) || !validText(annotation.Reviewer) || !validText(annotation.RubricID) || !validText(annotation.RubricVersion) || !annotation.Synthetic || annotation.CreatedAt <= 0 || len(annotation.EvidenceIDs) > 32 || len(annotation.OutputRef) > 512 || annotation.Supersedes != "" && annotation.Retracts != "" {
		return fmt.Errorf("valid synthetic annotation identity, rubric, time and bounds required")
	}
	// Snapshot caller-owned slices before queue admission. Only the existing writer
	// commits metadata; accepted jobs drain on shutdown just like intake/diffs.
	annotation.EvidenceIDs = append([]string(nil), annotation.EvidenceIDs...)
	annotation.RecordedAt = time.Now().UnixNano()
	return s.submit(ctx, job{metadata: &metadataWrite{annotation: &annotation}, done: make(chan error, 1)})
}

func (s *Store) writeMetadata(ctx context.Context, metadata metadataWrite) (result error) {
	// Constraint/validation mistakes reject this metadata request. Disk, schema
	// and commit errors still reach the serving lifecycle instead of being hidden.
	defer func() {
		var e *sqlite.Error
		if errors.As(result, &e) && e.Code()&255 == 19 {
			result = fmt.Errorf("%w: %v", MetadataRejected, result)
		}
	}()
	tx, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if a := metadata.annotation; a != nil {
		for _, id := range []string{a.Supersedes, a.Retracts} {
			if id == "" {
				continue
			}
			var run, dimension, reviewer, source string
			if err = tx.QueryRowContext(ctx, "SELECT run_id,dimension,reviewer,source FROM annotations WHERE id=?", id).Scan(&run, &dimension, &reviewer, &source); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("%w: revision missing", MetadataRejected)
				}
				return err
			}
			if run != a.RunID || dimension != a.Dimension || reviewer != a.Reviewer || source != a.Source {
				return fmt.Errorf("%w: revision must retain target, dimension, source and reviewer", MetadataRejected)
			}
			var revised int
			if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM annotations WHERE supersedes=? OR retracts=?", id, id).Scan(&revised); err != nil {
				return err
			}
			if revised > 0 {
				return fmt.Errorf("%w: annotation already revised", MetadataRejected)
			}
		}
		for _, id := range a.EvidenceIDs {
			var owner string
			if err = tx.QueryRowContext(ctx, "SELECT run_id FROM signals WHERE evidence_id=?", id).Scan(&owner); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("%w: evidence missing", MetadataRejected)
				}
				return err
			}
			if owner != a.RunID {
				return fmt.Errorf("%w: evidence belongs to another run", MetadataRejected)
			}
		}
		if a.OutputRef != "" {
			var count int
			if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM signals WHERE run_id=? AND kind='event' AND json_extract(data,'$.attrs."signalscout.output.ref"')=?`, a.RunID, a.OutputRef).Scan(&count); err != nil {
				return err
			}
			if count == 0 {
				return fmt.Errorf("%w: output ref not observed in target run", MetadataRejected)
			}
		}
		data, e := json.Marshal(a)
		if e != nil {
			return e
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO annotations VALUES(?,?,?,?,?,?,?,?,NULLIF(?,''),NULLIF(?,''))", a.ID, a.RunID, a.Dimension, a.Value, a.Source, a.Reviewer, a.RecordedAt, string(data), a.Supersedes, a.Retracts)
	} else if c := metadata.cohort; c != nil {
		data, e := json.Marshal(c)
		if e != nil {
			return e
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO cohorts VALUES(?,?,?)", c.ID, c.CreatedAt, string(data))
	} else {
		return fmt.Errorf("empty metadata job")
	}
	if err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	return tx.Commit()
}

// Annotations lists immutable revisions; disagreement/retractions remain visible.
func (r *Reader) Annotations(ctx context.Context, run string, from, to int64, limit int) ([]model.Annotation, error) {
	if from < 0 || to <= from || limit < 1 || limit > 1000 {
		return nil, fmt.Errorf("explicit label time range and limit 1..1000 required")
	}
	query := "SELECT data FROM annotations WHERE recorded_at>=? AND recorded_at<?"
	args := []any{from, to}
	if run != "" {
		query += " AND run_id=?"
		args = append(args, run)
	}
	query += " ORDER BY recorded_at,id LIMIT ?"
	args = append(args, limit)
	rows, err := r.reader.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []model.Annotation{}
	for rows.Next() {
		var data string
		var a model.Annotation
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(data), &a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Reader) Annotation(ctx context.Context, id string) (model.Annotation, error) {
	var a model.Annotation
	var data string
	if err := r.reader.QueryRowContext(ctx, "SELECT data FROM annotations WHERE id=?", id).Scan(&data); err != nil {
		return a, err
	}
	return a, json.Unmarshal([]byte(data), &a)
}
