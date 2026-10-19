package store

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/runs"
	_ "modernc.org/sqlite" // Registers the pure Go driver with database/sql.
)

//go:embed migrations/*.sql
var migrations embed.FS

type queueError string

func (e queueError) Error() string { return string(e) }

const QueueFull queueError = "ingest queue full"
const Closed queueError = "store closed"

type Options struct {
	QueueSize     int
	IdleTimeout   time.Duration
	Retention     time.Duration
	SweepInterval time.Duration
	Thresholds    map[string]float64
}

type job struct {
	batch *model.Batch
	now   time.Time
	done  chan error
}

type Store struct {
	*Reader
	writer          *sql.DB
	options         Options
	queue           chan job
	done            chan struct{}
	errors          chan error
	open            map[string]int64
	maintenance     chan time.Time
	stopMaintenance chan struct{}
	maintenanceDone chan struct{}
	// Admission lock prevents sending to the channel after Close closes it.
	mu        sync.RWMutex
	closed    bool
	closeOnce sync.Once
	closeErr  error
}

func Open(ctx context.Context, path string, options Options) (*Store, error) {
	if options.QueueSize <= 0 || options.IdleTimeout <= 0 || options.Retention < 0 {
		return nil, fmt.Errorf("queue size and idle timeout must be positive")
	}
	if options.SweepInterval <= 0 {
		options.SweepInterval = time.Second
	}
	uri, err := databaseURI(path)
	if err != nil {
		return nil, err
	}
	writer, err := sql.Open("sqlite", uri+"?mode=rwc&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open writer: %w", err)
	}
	writer.SetMaxOpenConns(1)
	failed := true
	defer func() {
		if failed {
			_ = writer.Close()
		}
	}()
	if _, err = writer.ExecContext(ctx, "PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;"); err != nil {
		return nil, fmt.Errorf("configure SQLite: %w", err)
	}
	if err = migrate(ctx, writer); err != nil {
		return nil, err
	}
	reader, err := OpenReader(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("open reader: %w", err)
	}
	s := &Store{Reader: reader, writer: writer, options: options, queue: make(chan job, options.QueueSize), done: make(chan struct{}), errors: make(chan error, 1), open: map[string]int64{}, maintenance: make(chan time.Time, 1), stopMaintenance: make(chan struct{}), maintenanceDone: make(chan struct{})}
	rows, err := writer.QueryContext(ctx, "SELECT run_id,last_seen FROM runs WHERE closed=0")
	if err != nil {
		_ = reader.Close()
		return nil, fmt.Errorf("recover open runs: %w", err)
	}
	for rows.Next() {
		var id string
		var last int64
		if err = rows.Scan(&id, &last); err != nil {
			break
		}
		s.open[id] = last
	}
	if err == nil {
		err = rows.Err()
	}
	_ = rows.Close()
	if err != nil {
		_ = reader.Close()
		return nil, fmt.Errorf("read open runs: %w", err)
	}
	failed = false
	// The writer goroutine owns every operational write, including timer work.
	// It drains accepted jobs before closing connections; callers never write SQL.
	go s.loop()
	go s.scheduleMaintenance()
	return s, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations(name TEXT PRIMARY KEY)"); err != nil {
		return fmt.Errorf("migration registry: %w", err)
	}
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	for _, entry := range entries {
		var count int
		if err = db.QueryRowContext(ctx, "SELECT count(*) FROM schema_migrations WHERE name=?", entry.Name()).Scan(&count); err != nil {
			return fmt.Errorf("migration lookup: %w", err)
		}
		if count > 0 {
			continue
		}
		data, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration: %w", err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration: %w", err)
		}
		if _, err = tx.ExecContext(ctx, string(data)); err == nil {
			_, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations VALUES(?)", entry.Name())
		}
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("commit migration: %w", err)
		}
	}
	return nil
}

// Submit returns only after commit. Cancellation stops waiting, not an accepted job.
func (s *Store) Submit(ctx context.Context, batch model.Batch) error {
	return s.submit(ctx, job{batch: &batch, now: time.Now(), done: make(chan error, 1)})
}
func (s *Store) Maintain(ctx context.Context, now time.Time) error {
	return s.submit(ctx, job{now: now, done: make(chan error, 1)})
}
func (s *Store) submit(ctx context.Context, j job) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return Closed
	}
	select {
	case s.queue <- j:
		s.mu.RUnlock()
	default:
		s.mu.RUnlock()
		return QueueFull
	}
	select {
	case err := <-j.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Store) Errors() <-chan error { return s.errors }

func (s *Store) loop() {
	defer close(s.done)
	for {
		select {
		case j, ok := <-s.queue:
			if !ok {
				s.closeErr = s.maintain(context.Background(), time.Now(), true)
				return
			}
			var err error
			if j.batch != nil {
				err = s.write(context.Background(), *j.batch, j.now)
			} else {
				err = s.maintain(context.Background(), j.now, false)
			}
			if err != nil {
				select {
				case s.errors <- err:
				default:
				}
			}
			j.done <- err
		case now := <-s.maintenance:
			if err := s.maintain(context.Background(), now, false); err != nil {
				select {
				case s.errors <- err:
				default:
				}
			}
		}
	}
}

// The scheduler never writes SQL. Coalescing ticks bounds maintenance backlog;
// the writer serializes idle finalization and retention with incoming batches.
func (s *Store) scheduleMaintenance() {
	defer close(s.maintenanceDone)
	ticker := time.NewTicker(s.options.SweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopMaintenance:
			return
		case now := <-ticker.C:
			select {
			case s.maintenance <- now:
			default:
			}
		}
	}
}

func (s *Store) Close() error {
	s.closeOnce.Do(func() {
		close(s.stopMaintenance)
		<-s.maintenanceDone
		s.mu.Lock()
		s.closed = true
		close(s.queue)
		s.mu.Unlock()
		<-s.done
		if err := s.reader.Close(); err != nil && s.closeErr == nil {
			s.closeErr = fmt.Errorf("close reader: %w", err)
		}
		if err := s.writer.Close(); err != nil && s.closeErr == nil {
			s.closeErr = fmt.Errorf("close writer: %w", err)
		}
	})
	return s.closeErr
}

func loadState(ctx context.Context, tx *sql.Tx, id string) (runs.State, error) {
	var data string
	var state runs.State
	err := tx.QueryRowContext(ctx, "SELECT state FROM runs WHERE run_id=?", id).Scan(&data)
	if err == sql.ErrNoRows {
		return state, nil
	}
	if err != nil {
		return state, fmt.Errorf("load run: %w", err)
	}
	if err = json.Unmarshal([]byte(data), &state); err != nil {
		return state, fmt.Errorf("decode run: %w", err)
	}
	return state, nil
}

func saveState(ctx context.Context, tx *sql.Tx, state runs.State) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode run state: %w", err)
	}
	env, err := json.Marshal(state.Run.Env)
	if err != nil {
		return fmt.Errorf("encode environment: %w", err)
	}
	config, err := json.Marshal(state.Run.Config)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	r := state.Run
	_, err = tx.ExecContext(ctx, `INSERT INTO runs VALUES(?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(run_id) DO UPDATE SET
	workload=excluded.workload,domain=excluded.domain,started_at=excluded.started_at,ended_at=excluded.ended_at,outcome=excluded.outcome,
	failure_marker=excluded.failure_marker,env=excluded.env,config=excluded.config,closed=excluded.closed,last_seen=excluded.last_seen,state=excluded.state`,
		r.ID, r.Workload, r.Domain, r.StartedAt, r.EndedAt, r.Outcome, r.FailureMarker, string(env), string(config), r.Closed, state.LastSeen, string(data))
	if err != nil {
		return fmt.Errorf("save run: %w", err)
	}
	return nil
}

func (s *Store) write(ctx context.Context, batch model.Batch, now time.Time) error {
	tx, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin ingest: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	states := map[string]runs.State{}
	resourceStmt, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO resources VALUES(?,?,?)")
	if err != nil {
		return fmt.Errorf("prepare resources: %w", err)
	}
	defer func() { _ = resourceStmt.Close() }()
	signalStmt, err := tx.PrepareContext(ctx, `INSERT INTO signals VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(evidence_id) DO UPDATE SET data=excluded.data,received_at=excluded.received_at,ts_ns=excluded.ts_ns,end_ns=excluded.end_ns`)
	if err != nil {
		return fmt.Errorf("prepare signals: %w", err)
	}
	defer func() { _ = signalStmt.Close() }()
	for _, signal := range batch.Signals {
		id, domain := runs.Identity(signal)
		signal.RunID = id
		if id != "" {
			state, ok := states[id]
			if !ok {
				state, err = loadState(ctx, tx, id)
				if err != nil {
					return err
				}
			}
			if state.Run.ID != "" && state.Run.Domain != domain {
				return fmt.Errorf("run ID %q has conflicting domains", id)
			}
			state = runs.Observe(state, signal, now, 0)
			// The root may have just supplied the workload for threshold lookup.
			state = runs.Observe(state, signal, now, s.options.Thresholds[state.Run.Workload])
			states[id] = state
		}
		attrs, err := json.Marshal(signal.Resource.Attrs)
		if err != nil {
			return fmt.Errorf("encode resource: %w", err)
		}
		if _, err = resourceStmt.ExecContext(ctx, signal.Resource.ID, string(attrs), string(signal.Resource.Raw)); err != nil {
			return fmt.Errorf("write resource: %w", err)
		}
		data, err := json.Marshal(signal)
		if err != nil {
			return fmt.Errorf("encode signal: %w", err)
		}
		if _, err = signalStmt.ExecContext(ctx, signal.ID, signal.Kind, id, signal.Resource.ID, signal.TS, signal.End, now.UnixNano(), string(data)); err != nil {
			return fmt.Errorf("write signal: %w", err)
		}
	}
	for _, state := range states {
		if err = saveState(ctx, tx, state); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO run_resources SELECT DISTINCT run_id,resource_id FROM signals WHERE run_id<>'' AND received_at=?`, now.UnixNano()); err != nil {
		return fmt.Errorf("track run resources: %w", err)
	}
	if err = refreshContext(ctx, tx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit ingest: %w", err)
	}
	for id, state := range states {
		if state.Run.Closed {
			delete(s.open, id)
		} else {
			s.open[id] = state.LastSeen
		}
	}
	return nil
}

func refreshContext(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO context_links
	SELECT r.run_id,s.evidence_id FROM runs r JOIN run_resources rr ON rr.run_id=r.run_id
	JOIN signals s ON s.resource_id=rr.resource_id AND s.run_id='' AND s.kind IN ('log','metric')
	AND s.ts_ns>=r.started_at AND s.ts_ns<=r.ended_at WHERE r.closed=1`)
	if err != nil {
		return fmt.Errorf("attach context: %w", err)
	}
	return nil
}

func (s *Store) maintain(ctx context.Context, now time.Time, force bool) error {
	tx, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin maintenance: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	cutoff := now.Add(-s.options.IdleTimeout).UnixNano()
	if force {
		cutoff = now.UnixNano()
	}
	var states []runs.State
	for id, last := range s.open {
		if force || last <= cutoff {
			state, err := loadState(ctx, tx, id)
			if err != nil {
				return err
			}
			states = append(states, state)
		}
	}
	for _, state := range states {
		state.Run.Closed = true
		if err = saveState(ctx, tx, state); err != nil {
			return err
		}
	}
	if err = refreshContext(ctx, tx); err != nil {
		return err
	}
	if s.options.Retention > 0 {
		cutoff = now.Add(-s.options.Retention).UnixNano()
		queries := []string{
			"DELETE FROM signals WHERE run_id IN (SELECT run_id FROM runs WHERE closed=1 AND last_seen<?)",
			"DELETE FROM runs WHERE closed=1 AND last_seen<?",
			"DELETE FROM signals WHERE run_id='' AND received_at<?",
		}
		for _, query := range queries {
			if _, err = tx.ExecContext(ctx, query, cutoff); err != nil {
				return fmt.Errorf("retention cleanup: %w", err)
			}
		}
		if _, err = tx.ExecContext(ctx, "DELETE FROM resources WHERE resource_id NOT IN (SELECT resource_id FROM signals) AND resource_id NOT IN (SELECT resource_id FROM run_resources)"); err != nil {
			return fmt.Errorf("resource cleanup: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit maintenance: %w", err)
	}
	for _, state := range states {
		delete(s.open, state.Run.ID)
	}
	return nil
}

func (s *Reader) List(ctx context.Context, workload, outcome string) ([]model.Run, error) {
	rows, err := s.reader.QueryContext(ctx, "SELECT state FROM runs WHERE (?='' OR workload=?) AND (?='' OR outcome=?) ORDER BY started_at DESC,run_id", workload, workload, outcome, outcome)
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	result := []model.Run{}
	for rows.Next() {
		var data string
		var state runs.State
		if err = rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("read run: %w", err)
		}
		if err = json.Unmarshal([]byte(data), &state); err != nil {
			return nil, fmt.Errorf("decode run: %w", err)
		}
		result = append(result, state.Run)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate runs: %w", err)
	}
	return result, nil
}

func (s *Reader) Show(ctx context.Context, id string) (model.RunDetail, error) {
	var result model.RunDetail
	var state runs.State
	var data string
	// A read transaction keeps run metadata and both evidence sets on one snapshot.
	tx, err := s.reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return result, fmt.Errorf("begin read: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err = tx.QueryRowContext(ctx, "SELECT state FROM runs WHERE run_id=?", id).Scan(&data); err != nil {
		return result, fmt.Errorf("find run %q: %w", id, err)
	}
	if err = json.Unmarshal([]byte(data), &state); err != nil {
		return result, fmt.Errorf("decode run: %w", err)
	}
	result.Run = state.Run
	result.Signals, err = readSignals(ctx, tx, "SELECT data FROM signals WHERE run_id=? ORDER BY ts_ns,evidence_id", id)
	if err != nil {
		return result, err
	}
	result.Context, err = readSignals(ctx, tx, "SELECT s.data FROM signals s JOIN context_links c ON c.evidence_id=s.evidence_id WHERE c.run_id=? ORDER BY s.ts_ns,s.evidence_id", id)
	if err != nil {
		return result, err
	}
	if err = tx.Commit(); err != nil {
		return result, fmt.Errorf("commit read: %w", err)
	}
	return result, nil
}

func readSignals(ctx context.Context, tx *sql.Tx, query, id string) ([]model.Signal, error) {
	rows, err := tx.QueryContext(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("query evidence: %w", err)
	}
	defer func() { _ = rows.Close() }()
	result := []model.Signal{}
	for rows.Next() {
		var data string
		var signal model.Signal
		if err = rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("read evidence: %w", err)
		}
		if err = json.Unmarshal([]byte(data), &signal); err != nil {
			return nil, fmt.Errorf("decode evidence: %w", err)
		}
		result = append(result, signal)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evidence: %w", err)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].TS == result[j].TS {
			return result[i].ID < result[j].ID
		}
		return result[i].TS < result[j].TS
	})
	return result, nil
}
