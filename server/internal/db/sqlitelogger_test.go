package db

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"testing"
)

// captureHandler records log records for assertion in tests.
type captureHandler struct {
	mu    sync.Mutex
	attrs []map[string]slog.Value
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *captureHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	m := make(map[string]slog.Value)
	m["msg"] = slog.StringValue(r.Message)
	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value
		return true
	})
	h.attrs = append(h.attrs, m)
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }

func (h *captureHandler) WithGroup(name string) slog.Handler { return h }

func (h *captureHandler) recordsFor(t *testing.T, msg string) []map[string]slog.Value {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []map[string]slog.Value
	for _, m := range h.attrs {
		if m["msg"].String() == msg {
			out = append(out, m)
		}
	}
	return out
}

func (h *captureHandler) reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.attrs = nil
}

func TestNewLoggingConnector_nilLoggerUsesDefault(t *testing.T) {
	conn, err := NewLoggingConnector(":memory:", nil)
	if err != nil {
		t.Fatalf("NewLoggingConnector: %v", err)
	}
	if conn == nil {
		t.Fatal("conn is nil")
	}
	_ = conn.(*loggingConnector)
}

func TestLoggingConnector_ExecAndQueryLogged(t *testing.T) {
	handler := &captureHandler{}
	logger := slog.New(handler)

	connector, err := NewLoggingConnector(":memory:", logger)
	if err != nil {
		t.Fatalf("NewLoggingConnector: %v", err)
	}
	db := sql.OpenDB(connector)
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	recs := handler.recordsFor(t, "sql")
	if len(recs) == 0 {
		t.Fatal("expected at least one sql log record for Exec")
	}
	got := recs[len(recs)-1]
	if got["op"].String() != "exec" {
		t.Errorf("op: got %q, want exec", got["op"].String())
	}
	if got["sql"].String() != `CREATE TABLE t (id INTEGER PRIMARY KEY)` {
		t.Errorf("sql: got %q", got["sql"].String())
	}

	handler.reset()
	row := db.QueryRow(`SELECT 1`)
	var one int
	if err := row.Scan(&one); err != nil {
		t.Fatalf("query row: %v", err)
	}
	recs = handler.recordsFor(t, "sql")
	if len(recs) == 0 {
		t.Fatal("expected sql log record for QueryRow")
	}
	got = recs[len(recs)-1]
	if got["op"].String() != "query" {
		t.Errorf("op: got %q, want query", got["op"].String())
	}
	if got["sql"].String() != `SELECT 1` {
		t.Errorf("sql: got %q", got["sql"].String())
	}
}

func TestLoggingConnector_QueryWithArgsLogged(t *testing.T) {
	handler := &captureHandler{}
	logger := slog.New(handler)

	connector, err := NewLoggingConnector(":memory:", logger)
	if err != nil {
		t.Fatalf("NewLoggingConnector: %v", err)
	}
	db := sql.OpenDB(connector)
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`CREATE TABLE t (id INTEGER, name TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	handler.reset()

	_, err = db.Exec(`INSERT INTO t (id, name) VALUES (?, ?)`, 1, "alice")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	recs := handler.recordsFor(t, "sql")
	if len(recs) == 0 {
		t.Fatal("expected sql log for Exec with args")
	}
	got := recs[len(recs)-1]
	if got["op"].String() != "exec" {
		t.Errorf("op: got %q, want exec", got["op"].String())
	}
	if got["sql"].String() != `INSERT INTO t (id, name) VALUES (?, ?)` {
		t.Errorf("sql: got %q", got["sql"].String())
	}
	// args should be present (slog value for the slice)
	_, hasArgs := got["args"]
	if !hasArgs {
		t.Error("expected args attribute in log")
	}
}

func TestLoggingConnector_QueryRowsLogged(t *testing.T) {
	handler := &captureHandler{}
	logger := slog.New(handler)

	connector, err := NewLoggingConnector(":memory:", logger)
	if err != nil {
		t.Fatalf("NewLoggingConnector: %v", err)
	}
	db := sql.OpenDB(connector)
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`CREATE TABLE t (id INTEGER)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	handler.reset()

	rows, err := db.Query(`SELECT id FROM t`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	_ = rows.Close()
	recs := handler.recordsFor(t, "sql")
	if len(recs) == 0 {
		t.Fatal("expected sql log for Query")
	}
	got := recs[len(recs)-1]
	if got["op"].String() != "query" {
		t.Errorf("op: got %q, want query", got["op"].String())
	}
	if got["sql"].String() != `SELECT id FROM t` {
		t.Errorf("sql: got %q", got["sql"].String())
	}
}

func TestLoggingConnector_PingSucceeds(t *testing.T) {
	connector, err := NewLoggingConnector(":memory:", slog.Default())
	if err != nil {
		t.Fatalf("NewLoggingConnector: %v", err)
	}
	db := sql.OpenDB(connector)
	defer func() { _ = db.Close() }()
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
}
