package db

import (
	"context"
	"database/sql/driver"
	"fmt"
	"log/slog"

	sqlite3 "github.com/mattn/go-sqlite3"
)

// loggingConnector implements driver.Connector by opening the sqlite3 driver
// and wrapping the connection to log all SQL.
type loggingConnector struct {
	dsn    string
	logger *slog.Logger
}

// loggingConn wraps driver.Conn to provide statement logging.
type loggingConn struct {
	conn   driver.Conn
	logger *slog.Logger
}

// loggingStmt wraps driver.Stmt to log Exec/Query and their args.
type loggingStmt struct {
	stmt   driver.Stmt
	query  string
	logger *slog.Logger
}

// NewLoggingConnector returns a driver.Connector that logs all SQL (query and args)
// using the given logger. Use sql.OpenDB(connector) to get a *sql.DB that logs.
// If logger is nil, slog.Default() is used.
func NewLoggingConnector(dsn string, logger *slog.Logger) (driver.Connector, error) {
	if logger == nil {
		logger = slog.Default()
	}
	return &loggingConnector{dsn: dsn, logger: logger}, nil
}

// Driver implements driver.Connector.
func (c *loggingConnector) Driver() driver.Driver {
	return &loggingDriver{}
}

// Connect implements driver.Connector.
func (c *loggingConnector) Connect(ctx context.Context) (driver.Conn, error) {
	underlying := &sqlite3.SQLiteDriver{}
	conn, err := underlying.Open(c.dsn)
	if err != nil {
		return nil, err
	}
	return &loggingConn{conn: conn, logger: c.logger}, nil
}

// loggingDriver satisfies Connector.Driver(); opening is done via OpenDB(connector).
type loggingDriver struct{}

// Open implements driver.Driver; opening via this driver is not supported (use OpenDB(connector)).
func (d *loggingDriver) Open(name string) (driver.Conn, error) {
	return nil, fmt.Errorf("sqlite3-log: use sql.OpenDB(NewLoggingConnector(...)) instead of sql.Open")
}

// Prepare implements driver.Conn.
func (c *loggingConn) Prepare(query string) (driver.Stmt, error) {
	stmt, err := c.conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &loggingStmt{stmt: stmt, query: query, logger: c.logger}, nil
}

// PrepareContext implements driver.ConnPrepareContext.
func (c *loggingConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if prep, ok := c.conn.(driver.ConnPrepareContext); ok {
		stmt, err := prep.PrepareContext(ctx, query)
		if err != nil {
			return nil, err
		}
		return &loggingStmt{stmt: stmt, query: query, logger: c.logger}, nil
	}
	return c.Prepare(query)
}

// Close implements driver.Conn.
func (c *loggingConn) Close() error {
	return c.conn.Close()
}

// Begin implements driver.Conn.
func (c *loggingConn) Begin() (driver.Tx, error) {
	//nolint:staticcheck // SA1019 – required when underlying conn does not implement ConnBeginTx
	return c.conn.Begin()
}

// BeginTx implements driver.ConnBeginTx.
func (c *loggingConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if beginTx, ok := c.conn.(driver.ConnBeginTx); ok {
		return beginTx.BeginTx(ctx, opts)
	}
	//nolint:staticcheck // SA1019 – fallback when underlying conn does not implement ConnBeginTx
	return c.conn.Begin()
}

// Exec implements driver.Stmt.
func (s *loggingStmt) Exec(args []driver.Value) (driver.Result, error) {
	s.logQuery("exec", args)
	//nolint:staticcheck // SA1019 – required when underlying stmt does not implement StmtExecContext
	return s.stmt.Exec(args)
}

// ExecContext implements driver.StmtExecContext.
func (s *loggingStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	s.logQuery("exec", namedValuesToSlice(args))
	execCtx, ok := s.stmt.(driver.StmtExecContext)
	if !ok {
		vals := namedValuesToValues(args)
		//nolint:staticcheck // SA1019 – fallback when underlying stmt does not implement StmtExecContext
		return s.stmt.Exec(vals)
	}
	return execCtx.ExecContext(ctx, args)
}

// Query implements driver.Stmt.
func (s *loggingStmt) Query(args []driver.Value) (driver.Rows, error) {
	s.logQuery("query", args)
	//nolint:staticcheck // SA1019 – required when underlying stmt does not implement StmtQueryContext
	return s.stmt.Query(args)
}

// QueryContext implements driver.StmtQueryContext.
func (s *loggingStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	s.logQuery("query", namedValuesToSlice(args))
	queryCtx, ok := s.stmt.(driver.StmtQueryContext)
	if !ok {
		vals := namedValuesToValues(args)
		//nolint:staticcheck // SA1019 – fallback when underlying stmt does not implement StmtQueryContext
		return s.stmt.Query(vals)
	}
	return queryCtx.QueryContext(ctx, args)
}

// Close implements driver.Stmt.
func (s *loggingStmt) Close() error {
	return s.stmt.Close()
}

// NumInput implements driver.Stmt (optional); -1 means unknown.
func (s *loggingStmt) NumInput() int {
	if n, ok := s.stmt.(interface{ NumInput() int }); ok {
		return n.NumInput()
	}
	return -1
}

func (s *loggingStmt) logQuery(op string, args interface{}) {
	attrs := []any{
		"op", op,
		"sql", s.query,
		"args", args,
	}
	s.logger.Debug("sql", attrs...)
}

func namedValuesToSlice(args []driver.NamedValue) []interface{} {
	out := make([]interface{}, len(args))
	for i, a := range args {
		if a.Name != "" {
			out[i] = a.Name + "=" + formatArg(a.Value)
		} else {
			out[i] = formatArg(a.Value)
		}
	}
	return out
}

func namedValuesToValues(args []driver.NamedValue) []driver.Value {
	out := make([]driver.Value, len(args))
	for i := range args {
		out[i] = args[i].Value
	}
	return out
}

func formatArg(v interface{}) string {
	if v == nil {
		return "NULL"
	}
	switch t := v.(type) {
	case []byte:
		return string(t)
	default:
		return fmt.Sprint(t)
	}
}
