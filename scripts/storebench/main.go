// storebench validates storage growth and bounded SQL on a seeded real-schema fixture.
// It deliberately does not implement the M4b query API or viewer.
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/store"
)

const epoch int64 = 1700000000000000000

func signal(kind string, index int, seed int64) model.Signal {
	service := index / 256 % 10
	attrs := map[string]any{"service.name": fmt.Sprintf("service-%02d", service), "host.name": fmt.Sprintf("host-%d", service%2)}
	resourceID := service + 1
	if kind == "metric" && index/256%4 == 0 {
		attrs = map[string]any{"host.name": fmt.Sprintf("host-%d", service%2)}
		resourceID = 11 + service%2
	}
	resourceRaw, _ := json.Marshal(attrs)
	ts := epoch + int64(index)*int64(time.Second)
	hash := sha256.Sum256(fmt.Appendf(nil, "%s/%d/%d", kind, seed, index))
	s := model.Signal{ID: fmt.Sprintf("%s:%x/0", kind, hash), Kind: kind, TS: ts, End: ts, Name: "queue.depth", Resource: model.Resource{ID: fmt.Sprintf("%064x", resourceID), Attrs: attrs, Raw: resourceRaw}, Attrs: map[string]any{}, Scope: json.RawMessage(`{"scope":{"name":"storebench"}}`)}
	switch kind {
	case "span":
		s.TraceID = fmt.Sprintf("%016x%016x", seed, index/10+1)
		s.SpanID = fmt.Sprintf("%016x", index+1)
		s.ID = "span:" + s.TraceID + "/" + s.SpanID
		if index%10 != 0 {
			s.ParentID = fmt.Sprintf("%016x", index/10*10+1)
		}
		s.Name = "request"
		s.Status = "ok"
		s.End = ts + int64(time.Millisecond)*(1+int64(index%13))
		s.Attrs = map[string]any{"http.request.method": "POST", "http.route": "/benchmark"}
		s.Raw = json.RawMessage(fmt.Sprintf(`{"name":"request","startTimeUnixNano":"%d","endTimeUnixNano":"%d","traceId":"%s","spanId":"%s"}`, ts, s.End, s.TraceID, s.SpanID))
	case "log":
		if index%4 == 0 {
			s.TraceID = fmt.Sprintf("%016x%016x", seed, index/10+1)
			s.SpanID = fmt.Sprintf("%016x", index+1)
		}
		s.Name = "INFO"
		s.Template = fmt.Sprintf("operation-%c request {num} completed", 'a'+index%8)
		s.Body = fmt.Sprintf("operation-%c request %d completed", 'a'+index%8, index)
		s.Raw = json.RawMessage(fmt.Sprintf(`{"timeUnixNano":"%d","severityText":"INFO","body":{"stringValue":%q}}`, ts, s.Body))
	case "metric":
		v := float64((index*31+int(seed))%1000) / 10
		s.Value = &v
		s.Raw = json.RawMessage(fmt.Sprintf(`{"metric":{"name":"queue.depth","type":"gauge"},"point":{"timeUnixNano":"%d","asDouble":%g}}`, ts, v))
	}
	return s
}

func populate(ctx context.Context, path string, total int, seed int64) (float64, error) {
	if total <= 0 || total%4 != 0 {
		return 0, fmt.Errorf("count must be a positive multiple of four")
	}
	if _, err := os.Stat(path); err == nil {
		return 0, fmt.Errorf("refuse to overwrite existing dataset")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return 0, err
	}
	s, err := store.Open(ctx, path, store.Options{QueueSize: 64, IdleTimeout: time.Hour, SweepInterval: time.Hour, ContextBefore: 30 * time.Second, ContextAfter: 5 * time.Second})
	if err != nil {
		return 0, err
	}
	start := time.Now()
	for _, kind := range []string{"span", "log", "metric"} {
		count := total / 4
		if kind == "metric" {
			count *= 2
		}
		for first := 0; first < count; first += 256 {
			batch := model.Batch{}
			for index := first; index < min(first+256, count); index++ {
				batch.Signals = append(batch.Signals, signal(kind, index, seed))
			}
			if err = s.Submit(ctx, batch); err != nil {
				_ = s.Close()
				return 0, err
			}
		}
		fmt.Printf("stored %s %d\n", kind, count)
	}
	// Closure is included in fixture generation, never in ingest throughput.
	err = s.Close()
	return time.Since(start).Seconds(), err
}

type query struct {
	Name, SQL string
	Args      []any
}
type timing struct {
	Name         string    `json:"name"`
	SQL          string    `json:"sql"`
	Args         []any     `json:"arguments"`
	Plan         []string  `json:"query_plan"`
	Milliseconds []float64 `json:"milliseconds"`
	Rows         int       `json:"rows"`
	Met          bool      `json:"all_under_500ms"`
}
type report struct {
	Database            string           `json:"database"`
	Seed                int64            `json:"seed"`
	Go                  string           `json:"go"`
	Method              string           `json:"method"`
	GenerationSeconds   float64          `json:"generation_seconds_including_close"`
	Bytes               int64            `json:"database_bytes"`
	PageSize            int64            `json:"page_size"`
	FreePages           int64            `json:"free_pages"`
	Counts              map[string]int64 `json:"counts"`
	ContextObjectType   string           `json:"context_links_object_type"`
	ContextCountSeconds float64          `json:"context_count_seconds"`
	Queries             []timing         `json:"queries"`
}

func queries(total int, seed int64) []query {
	middle := total / 8
	from := epoch + int64(max(0, middle-1800))*int64(time.Second)
	to := epoch + int64(middle+1800)*int64(time.Second)
	return []query{
		{"service_time", `SELECT s.evidence_id,s.kind,s.ts_ns,s.data FROM signals s JOIN resources r ON r.resource_id=s.resource_id WHERE json_extract(r.attrs,'$."service.name"')=? AND s.ts_ns>=? AND s.ts_ns<? ORDER BY s.ts_ns,s.evidence_id LIMIT ?`, []any{"service-02", from, to, 100}},
		{"trace_id", `SELECT evidence_id,kind,ts_ns,data FROM signals WHERE kind IN ('span','log') AND json_extract(data,'$.trace_id')=? AND ts_ns>=? AND ts_ns<? ORDER BY ts_ns,evidence_id LIMIT ?`, []any{fmt.Sprintf("%016x%016x", seed, middle/10+1), from, to, 100}},
		{"log_templates", `SELECT json_extract(s.data,'$.template') AS template,count(*) FROM signals s JOIN resources r ON r.resource_id=s.resource_id WHERE s.kind='log' AND json_extract(r.attrs,'$."service.name"')=? AND s.ts_ns>=? AND s.ts_ns<? GROUP BY template ORDER BY count(*) DESC,template LIMIT ?`, []any{"service-02", max(epoch, epoch+int64(middle-43200)*int64(time.Second)), epoch + int64(middle+43200)*int64(time.Second), 100}},
	}
}

func inspect(ctx context.Context, path string, total int, seed int64) (report, error) {
	r := report{Database: filepath.ToSlash(path), Seed: seed, Go: runtime.Version(), Counts: map[string]int64{}, Method: "Seeded real-schema storage fixture (not an HTTP throughput measurement): 25% spans (10/trace), 25% logs (one quarter trace-correlated), 50% gauge points; ten services on two hosts, every fourth metric batch host-only. One-second event spacing, 30s/5s context padding. Full raw synthetic records, durable writer batches of 256 and shutdown finalization. Query clock includes query, row decoding, iteration and close, excludes EXPLAIN. Five repeats on one read-only connection; first is connection-cold, OS filesystem caches are not flushed. Required time windows and LIMIT 100. File size is after all writers close/checkpoint; free pages reported separately. Context view counts are logical links, scope counts are physical rows."}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return r, err
	}
	u := url.URL{Scheme: "file", Path: "/" + filepath.ToSlash(absolute)}
	if filepath.VolumeName(absolute) == "" {
		u.Path = filepath.ToSlash(absolute)
	}
	db, err := sql.Open("sqlite", u.String()+"?mode=ro&_pragma=query_only(1)")
	if err != nil {
		return r, err
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	info, err := os.Stat(path)
	if err != nil {
		return r, err
	}
	r.Bytes = info.Size()
	if err = db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&r.PageSize); err != nil {
		return r, err
	}
	if err = db.QueryRowContext(ctx, "PRAGMA freelist_count").Scan(&r.FreePages); err != nil {
		return r, err
	}
	for _, table := range []string{"signals", "runs", "resources", "context_scopes", "context_links", "context_matches"} {
		var exists int
		if err = db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE name=?", table).Scan(&exists); err != nil {
			return r, err
		}
		if exists == 0 {
			continue
		}
		start := time.Now()
		var count int64
		if err = db.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil {
			return r, err
		}
		r.Counts[table] = count
		if strings.HasPrefix(table, "context_") {
			r.ContextCountSeconds += time.Since(start).Seconds()
		}
	}
	if r.Counts["signals"] != int64(total) {
		return r, fmt.Errorf("fixture has %d signals, expected %d", r.Counts["signals"], total)
	}
	if err = db.QueryRowContext(ctx, "SELECT type FROM sqlite_master WHERE name='context_links'").Scan(&r.ContextObjectType); err != nil {
		return r, err
	}
	for _, q := range queries(total, seed) {
		// Each query starts with a new SQLite page cache. OS caches remain warm;
		// the subsequent four repeats reuse this connection deliberately.
		if err = db.Close(); err != nil {
			return r, err
		}
		db, err = sql.Open("sqlite", u.String()+"?mode=ro&_pragma=query_only(1)")
		if err != nil {
			return r, err
		}
		db.SetMaxOpenConns(1)
		t := timing{Name: q.Name, SQL: q.SQL, Args: q.Args, Met: true}
		rows, err := db.QueryContext(ctx, "EXPLAIN QUERY PLAN "+q.SQL, q.Args...)
		if err != nil {
			return r, err
		}
		for rows.Next() {
			var a, b, c int
			var line string
			if err = rows.Scan(&a, &b, &c, &line); err != nil {
				_ = rows.Close()
				return r, err
			}
			t.Plan = append(t.Plan, line)
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return r, err
		}
		for range 5 {
			start := time.Now()
			rows, err = db.QueryContext(ctx, q.SQL, q.Args...)
			if err != nil {
				return r, err
			}
			count := 0
			columns, err := rows.Columns()
			if err != nil {
				_ = rows.Close()
				return r, err
			}
			values := make([]any, len(columns))
			ptrs := make([]any, len(columns))
			for i := range values {
				ptrs[i] = &values[i]
			}
			for rows.Next() {
				if err = rows.Scan(ptrs...); err != nil {
					_ = rows.Close()
					return r, err
				}
				count++
			}
			err = rows.Err()
			closeErr := rows.Close()
			if err != nil {
				return r, err
			}
			if closeErr != nil {
				return r, closeErr
			}
			elapsed := float64(time.Since(start)) / float64(time.Millisecond)
			t.Milliseconds = append(t.Milliseconds, elapsed)
			t.Rows = count
			t.Met = t.Met && elapsed < 500
			if count == 0 || count > 100 {
				return r, fmt.Errorf("%s returned invalid row count %d", q.Name, count)
			}
		}
		r.Queries = append(r.Queries, t)
	}
	return r, nil
}

func run() error {
	path := flag.String("db", ".cache/performance/million.db", "Dataset path")
	out := flag.String("out", "benchmark/results/perf-million.json", "Report path")
	count := flag.Int("count", 1000000, "Signal count (multiple of four)")
	seed := flag.Int64("seed", 1, "Published fixture seed")
	create := flag.Bool("create", false, "Generate fixture; refuses existing destination")
	audit := flag.Bool("audit-context", false, "Hash complete raw data and context sets in an existing closed database; do not migrate")
	flag.Parse()
	ctx := context.Background()
	if *audit {
		return auditContext(ctx, *path, *out)
	}
	var seconds float64
	var err error
	if *create {
		seconds, err = populate(ctx, *path, *count, *seed)
		if err != nil {
			return err
		}
	}
	r, err := inspect(ctx, *path, *count, *seed)
	if err != nil {
		return err
	}
	r.GenerationSeconds = seconds
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(*out, append(data, '\n'), 0644); err != nil {
		return err
	}
	fmt.Println(string(data))
	for _, q := range r.Queries {
		if !q.Met {
			return fmt.Errorf("1M-query target missed; measurements retained")
		}
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
