// Package testsql supplies a network-free database/sql driver for unit tests.
package testsql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync/atomic"
	"testing"
)

// Behavior is configured before use. Its counters and captured options are safe
// to read while database/sql performs cancellation cleanup in another goroutine.
type Behavior struct {
	BeginErr, QueryErr, ExecErr, CommitErr, RollbackErr error
	Begins, Queries, Execs, Commits, Rollbacks          atomic.Int64
	Columns                                             []string
	Rows                                                [][]driver.Value
	BeginIsolation                                      atomic.Int64
	BeginReadOnly                                       atomic.Bool
}

// Open creates an independent in-memory connection pool and closes it at cleanup.
func Open(t testing.TB, behavior *Behavior) *sql.DB {
	t.Helper()
	if behavior == nil {
		t.Fatal("testsql: behavior is required")
	}
	db := sql.OpenDB(&connector{behavior: behavior})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close testsql: %v", err)
		}
	})
	return db
}

type connector struct{ behavior *Behavior }

func (c *connector) Connect(ctx context.Context) (driver.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &connection{behavior: c.behavior}, nil
}
func (c *connector) Driver() driver.Driver { return &sqlDriver{behavior: c.behavior} }

type sqlDriver struct{ behavior *Behavior }

func (d *sqlDriver) Open(string) (driver.Conn, error) { return &connection{behavior: d.behavior}, nil }

type connection struct{ behavior *Behavior }

func (c *connection) Prepare(query string) (driver.Stmt, error) {
	return &statement{connection: c, query: query}, nil
}
func (c *connection) Close() error { return nil }
func (c *connection) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
func (c *connection) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.behavior.Begins.Add(1)
	c.behavior.BeginIsolation.Store(int64(opts.Isolation))
	c.behavior.BeginReadOnly.Store(opts.ReadOnly)
	if c.behavior.BeginErr != nil {
		return nil, c.behavior.BeginErr
	}
	return &transaction{behavior: c.behavior}, nil
}
func (c *connection) ExecContext(ctx context.Context, _ string, _ []driver.NamedValue) (driver.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.behavior.Execs.Add(1)
	if c.behavior.ExecErr != nil {
		return nil, c.behavior.ExecErr
	}
	return driver.RowsAffected(1), nil
}
func (c *connection) QueryContext(ctx context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.behavior.Queries.Add(1)
	if c.behavior.QueryErr != nil {
		return nil, c.behavior.QueryErr
	}
	return &rows{columns: c.behavior.Columns, values: c.behavior.Rows}, nil
}

type transaction struct{ behavior *Behavior }

func (t *transaction) Commit() error   { t.behavior.Commits.Add(1); return t.behavior.CommitErr }
func (t *transaction) Rollback() error { t.behavior.Rollbacks.Add(1); return t.behavior.RollbackErr }

type statement struct {
	connection *connection
	query      string
}

func (s *statement) Close() error  { return nil }
func (s *statement) NumInput() int { return -1 }
func (s *statement) Exec(args []driver.Value) (driver.Result, error) {
	return s.ExecContext(context.Background(), namedValues(args))
}
func (s *statement) Query(args []driver.Value) (driver.Rows, error) {
	return s.QueryContext(context.Background(), namedValues(args))
}
func (s *statement) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	return s.connection.ExecContext(ctx, s.query, args)
}
func (s *statement) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	return s.connection.QueryContext(ctx, s.query, args)
}
func namedValues(values []driver.Value) []driver.NamedValue {
	args := make([]driver.NamedValue, len(values))
	for i, value := range values {
		args[i] = driver.NamedValue{Ordinal: i + 1, Value: value}
	}
	return args
}

type rows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *rows) Columns() []string { return append([]string(nil), r.columns...) }
func (r *rows) Close() error      { return nil }
func (r *rows) Next(dest []driver.Value) error {
	if r.index == len(r.values) {
		return io.EOF
	}
	row := r.values[r.index]
	if len(row) != len(dest) {
		return errors.New("testsql: row width does not match columns")
	}
	copy(dest, row)
	r.index++
	return nil
}

var (
	_ driver.Connector        = (*connector)(nil)
	_ driver.Driver           = (*sqlDriver)(nil)
	_ driver.ConnBeginTx      = (*connection)(nil)
	_ driver.ExecerContext    = (*connection)(nil)
	_ driver.QueryerContext   = (*connection)(nil)
	_ driver.StmtExecContext  = (*statement)(nil)
	_ driver.StmtQueryContext = (*statement)(nil)
)
