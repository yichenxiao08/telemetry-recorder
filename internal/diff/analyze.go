package diff

import (
	"context"
	"errors"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
)

// Analyze includes reading, computation, persistence and connection closure in
// the reported time. Compilation and process startup are outside this interval.
func Analyze(ctx context.Context, path, id string, options model.BaselineOptions) (model.DiffResult, error) {
	start := time.Now()
	var result model.DiffResult
	reader, err := store.OpenReader(ctx, path)
	if err != nil {
		return result, err
	}
	input, readErr := reader.ContextComparison(ctx, id, options)
	if err = errors.Join(readErr, reader.Close()); err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	result, err = Compute(input)
	if err != nil {
		return result, err
	}
	if err = persist(ctx, path, result); err != nil {
		return result, err
	}
	// Persistence stores the deterministic computation and baseline specification.
	// This caller's elapsed time is added only after the writer has also closed.
	result.ElapsedNS = time.Since(start).Nanoseconds()
	return result, nil
}

func persist(ctx context.Context, path string, result model.DiffResult) (err error) {
	writer, err := store.OpenResultWriter(ctx, path)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, writer.Close()) }()
	return writer.SaveDiff(ctx, result)
}
