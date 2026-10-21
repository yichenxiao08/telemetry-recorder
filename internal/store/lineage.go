package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/signalscout/signalscout/internal/lineage"
	"github.com/signalscout/signalscout/internal/model"
)

// Only the envelope attributes are transferred, with an 8 KiB payload ceiling.
// LIMIT counts all event/log rows inspected, including non-provenance records.
const traceRunSQL = `SELECT evidence_id,kind,run_id,ts_ns,
 json_extract(data,'$.attrs."signalscout.record.kind"'),
 json_extract(data,'$.attrs."signalscout.record.id"'),
 CASE WHEN length(CAST(json_extract(data,'$.attrs."signalscout.record.json"') AS BLOB))<=8192
 THEN json_extract(data,'$.attrs."signalscout.record.json"') ELSE '' END
 FROM signals INDEXED BY signals_run WHERE run_id=? AND kind IN ('event','log') AND ts_ns>=? AND ts_ns<?
 ORDER BY kind,ts_ns,evidence_id LIMIT ?`
const traceNativeSQL = `SELECT run_id FROM signals WHERE evidence_id=? AND ts_ns>=? AND ts_ns<? LIMIT 1`

type TraceReadRequest struct {
	Targets  []string
	RunID    string
	From, To int64
	Limit    int
}
type TraceScope struct {
	From           int64    `json:"from_ns"`
	To             int64    `json:"to_ns"`
	Limit          int      `json:"limit"`
	RecordsRead    int      `json:"records_read"`
	NativeLookups  int      `json:"native_lookups"`
	Runs           []string `json:"run_partitions"`
	Truncated      bool     `json:"truncated"`
	PartitionLimit int      `json:"partition_limit"`
	NegativeSearch string   `json:"negative_search"`
}
type TraceData struct {
	Signals []model.Signal
	Scope   TraceScope
}

func (r *Reader) TraceRead(ctx context.Context, q TraceReadRequest) (TraceData, error) {
	out := TraceData{Scope: TraceScope{From: q.From, To: q.To, Limit: q.Limit, Runs: []string{}, NegativeSearch: "Not found means unknown within these run partitions, [from,to), and the shared record limit; it does not establish absence outside this scope or distinguish eviction from missing capture."}}
	out.Scope.PartitionLimit = 128
	if q.From >= q.To || q.Limit < 1 || q.Limit > 10000 || len(q.Targets) < 1 || len(q.Targets) > 2 {
		return out, fmt.Errorf("explicit increasing time bounds, limit 1..10000 and one or two targets required")
	}
	tx, err := r.reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback() }()
	var queue []string
	queued := map[string]bool{}
	enqueue := func(run string) {
		if run != "" && !queued[run] {
			queued[run] = true
			queue = append(queue, run)
		}
	}
	for _, target := range q.Targets {
		run := lineage.RunRef(target)
		native := strings.HasPrefix(target, "event:") || strings.HasPrefix(target, "log:") || strings.HasPrefix(target, "span:")
		if run == "" && native {
			out.Scope.NativeLookups++
			err = tx.QueryRowContext(ctx, traceNativeSQL, target, q.From, q.To).Scan(&run)
			if err != nil && err != sql.ErrNoRows {
				return out, err
			}
		}
		if run == "" {
			run = q.RunID
		}
		if run == "" && native {
			continue
		} // Missing native evidence is unknown, not a history search.
		if run == "" {
			return out, fmt.Errorf("target %q needs --run (or a run-qualified value ref/native evidence ID present in the supplied time range)", target)
		}
		enqueue(run)
	}
	for len(queue) > 0 {
		if out.Scope.RecordsRead >= q.Limit || len(out.Scope.Runs) >= out.Scope.PartitionLimit {
			out.Scope.Truncated = true
			break
		}
		run := queue[0]
		queue = queue[1:]
		out.Scope.Runs = append(out.Scope.Runs, run)
		remaining := q.Limit - out.Scope.RecordsRead
		rows, err := tx.QueryContext(ctx, traceRunSQL, run, q.From, q.To, remaining+1)
		if err != nil {
			return out, err
		}
		var loaded []model.Signal
		for rows.Next() {
			var s model.Signal
			var kind, id, body sql.NullString
			if err = rows.Scan(&s.ID, &s.Kind, &s.RunID, &s.TS, &kind, &id, &body); err != nil {
				_ = rows.Close()
				return out, err
			}
			if len(loaded) >= remaining {
				out.Scope.Truncated = true
				break
			}
			s.Attrs = map[string]any{"signalscout.record.kind": kind.String, "signalscout.record.id": id.String, "signalscout.record.json": body.String}
			loaded = append(loaded, s)
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return out, err
		}
		out.Scope.RecordsRead += len(loaded)
		out.Signals = append(out.Signals, loaded...)
		// Discover only documented run-qualified refs. No global JSON/history lookup.
		// Invalid envelopes are reported by the walker; discovery cannot fetch artifacts.
		for _, s := range loaded {
			body, _ := s.Attrs["signalscout.record.json"].(string)
			var refs struct {
				Inputs   []lineage.Field `json:"inputs"`
				InputRef string          `json:"input_ref"`
			}
			if json.Unmarshal([]byte(body), &refs) == nil && len(refs.Inputs) <= 32 {
				enqueue(lineage.RunRef(refs.InputRef))
				for _, f := range refs.Inputs {
					enqueue(lineage.RunRef(f.Ref))
				}
			}
		}
		if out.Scope.Truncated {
			break
		}
	}
	return out, tx.Commit()
}
