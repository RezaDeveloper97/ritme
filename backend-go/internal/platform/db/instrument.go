package db

import (
	"context"
	"database/sql/driver"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

// Query instrumentation (T-M9-07).
//
// Deps.DB is a plain *sql.DB handed to every domain, and sqlc's DBTX is satisfied by both *sql.DB and *sql.Tx. The
// wrapper therefore sits one level lower, at the database/sql driver: Open/OpenDSN build the pool over an
// instrumented connector, so every query of every DBTX — pool or transaction, sqlc or raw — passes through
// conn/stmt below. database/sql itself still owns pooling, transactions, sql.ErrNoRows (Row.Scan) and context
// cancellation; the wrapper only forwards, times and counts. Per query it records db_query_duration_seconds{query},
// adds one to the request's metrics.QueryCounter and warns when it took longer than SlowQuery.

// Request deadline (T-M9-10b): most service calls pass the fiber.Ctx itself as their context, and it never carries a
// deadline. Every call into the base driver below therefore runs with httpx.DriverContext(ctx), the request's
// deadline-bounded context in that case, so the driver cancels the query at the request deadline (the handler then
// fails like any other DB error: 500, never 401). Metrics and logs still use the caller's ctx. database/sql's own
// wait for a free pool connection happens before the driver and keeps the caller's ctx.

// SlowQuery is the slow-query warning threshold.
const SlowQuery = 200 * time.Millisecond

// QueryRaw labels SQL without a sqlc `-- name:` header.
const QueryRaw = "raw"

// QueryName returns the sqlc query name of query (`-- name: GetPeriod :one` → GetPeriod), or QueryRaw.
func QueryName(query string) string {
	const marker = "-- name: "
	q := strings.TrimLeft(query, " \t\r\n")
	if !strings.HasPrefix(q, marker) {
		return QueryRaw
	}
	q = q[len(marker):]
	if i := strings.IndexAny(q, " \t\r\n"); i >= 0 {
		q = q[:i]
	}
	if q == "" {
		return QueryRaw
	}
	return q
}

// observe records one executed query. driver.ErrSkip is not an execution: database/sql retries through Prepare and
// the statement records it then.
func observe(ctx context.Context, query string, start time.Time, err error) {
	if errors.Is(err, driver.ErrSkip) {
		return
	}
	d := time.Since(start)
	name := QueryName(query)
	metrics.ObserveQuery(name, d)
	metrics.CountQuery(ctx)
	if d > SlowQuery {
		slog.Default().LogAttrs(ctx, slog.LevelWarn, "slow query",
			slog.String("query", name),
			slog.Float64("duration_ms", float64(d.Microseconds())/1000),
		)
	}
}

// instrumentedConnector wraps a driver.Connector so its connections are instrumented.
type instrumentedConnector struct{ base driver.Connector }

// Instrument wraps c (sql.OpenDB(Instrument(c))).
func Instrument(c driver.Connector) driver.Connector { return instrumentedConnector{base: c} }

func (c instrumentedConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.base.Connect(httpx.DriverContext(ctx))
	if err != nil {
		return nil, err
	}
	return &instrumentedConn{base: conn}, nil
}

func (c instrumentedConnector) Driver() driver.Driver { return c.base.Driver() }

// instrumentedConn forwards every optional driver interface the base conn implements; for one it lacks it answers
// the way database/sql treats a missing interface (ErrSkip / no-op), so behaviour is unchanged.
type instrumentedConn struct{ base driver.Conn }

var (
	_ driver.Conn               = (*instrumentedConn)(nil)
	_ driver.ConnPrepareContext = (*instrumentedConn)(nil)
	_ driver.ConnBeginTx        = (*instrumentedConn)(nil)
	_ driver.QueryerContext     = (*instrumentedConn)(nil)
	_ driver.ExecerContext      = (*instrumentedConn)(nil)
	_ driver.Pinger             = (*instrumentedConn)(nil)
	_ driver.SessionResetter    = (*instrumentedConn)(nil)
	_ driver.Validator          = (*instrumentedConn)(nil)
	_ driver.NamedValueChecker  = (*instrumentedConn)(nil)
)

func (c *instrumentedConn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}

func (c *instrumentedConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	var (
		st  driver.Stmt
		err error
	)
	if p, ok := c.base.(driver.ConnPrepareContext); ok {
		st, err = p.PrepareContext(httpx.DriverContext(ctx), query)
	} else {
		st, err = c.base.Prepare(query)
	}
	if err != nil {
		return nil, err
	}
	return &instrumentedStmt{base: st, conn: c.base, query: query}, nil
}

func (c *instrumentedConn) Close() error { return c.base.Close() }

//nolint:staticcheck // driver.Conn requires Begin; database/sql calls BeginTx.
func (c *instrumentedConn) Begin() (driver.Tx, error) { return c.base.Begin() }

func (c *instrumentedConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if b, ok := c.base.(driver.ConnBeginTx); ok {
		return b.BeginTx(httpx.DriverContext(ctx), opts)
	}
	if opts.Isolation != 0 || opts.ReadOnly {
		return nil, errors.New("db: driver does not support non-default transaction options")
	}
	return c.base.Begin() //nolint:staticcheck // fallback for drivers without BeginTx
}

func (c *instrumentedConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	q, ok := c.base.(driver.QueryerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	start := time.Now()
	rows, err := q.QueryContext(httpx.DriverContext(ctx), query, args)
	observe(ctx, query, start, err)
	return rows, err
}

func (c *instrumentedConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	e, ok := c.base.(driver.ExecerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	start := time.Now()
	res, err := e.ExecContext(httpx.DriverContext(ctx), query, args)
	observe(ctx, query, start, err)
	return res, err
}

func (c *instrumentedConn) Ping(ctx context.Context) error {
	if p, ok := c.base.(driver.Pinger); ok {
		return p.Ping(httpx.DriverContext(ctx))
	}
	return nil
}

func (c *instrumentedConn) ResetSession(ctx context.Context) error {
	if r, ok := c.base.(driver.SessionResetter); ok {
		return r.ResetSession(ctx)
	}
	return nil
}

func (c *instrumentedConn) IsValid() bool {
	if v, ok := c.base.(driver.Validator); ok {
		return v.IsValid()
	}
	return true
}

func (c *instrumentedConn) CheckNamedValue(nv *driver.NamedValue) error {
	if n, ok := c.base.(driver.NamedValueChecker); ok {
		return n.CheckNamedValue(nv)
	}
	return driver.ErrSkip
}

// instrumentedStmt times the executions of a prepared statement (database/sql prepares every query with arguments
// when the driver does not interpolate them, as go-sql-driver/mysql does by default).
type instrumentedStmt struct {
	base  driver.Stmt
	conn  driver.Conn // the statement's connection: database/sql falls back to its NamedValueChecker
	query string
}

var (
	_ driver.Stmt              = (*instrumentedStmt)(nil)
	_ driver.StmtQueryContext  = (*instrumentedStmt)(nil)
	_ driver.StmtExecContext   = (*instrumentedStmt)(nil)
	_ driver.NamedValueChecker = (*instrumentedStmt)(nil)
	_ driver.ColumnConverter   = (*instrumentedStmt)(nil) //nolint:staticcheck // forwarded for drivers that use it
)

func (s *instrumentedStmt) Close() error  { return s.base.Close() }
func (s *instrumentedStmt) NumInput() int { return s.base.NumInput() }

//nolint:staticcheck // driver.Stmt requires Exec; database/sql calls ExecContext.
func (s *instrumentedStmt) Exec(args []driver.Value) (driver.Result, error) {
	start := time.Now()
	res, err := s.base.Exec(args)
	observe(context.Background(), s.query, start, err)
	return res, err
}

//nolint:staticcheck // driver.Stmt requires Query; database/sql calls QueryContext.
func (s *instrumentedStmt) Query(args []driver.Value) (driver.Rows, error) {
	start := time.Now()
	rows, err := s.base.Query(args)
	observe(context.Background(), s.query, start, err)
	return rows, err
}

func (s *instrumentedStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	start := time.Now()
	dctx := httpx.DriverContext(ctx)
	var (
		rows driver.Rows
		err  error
	)
	if q, ok := s.base.(driver.StmtQueryContext); ok {
		rows, err = q.QueryContext(dctx, args)
	} else {
		var vals []driver.Value
		if vals, err = namedToValues(args); err == nil {
			if err = dctx.Err(); err == nil {
				rows, err = s.base.Query(vals) //nolint:staticcheck // fallback for drivers without StmtQueryContext
			}
		}
	}
	observe(ctx, s.query, start, err)
	return rows, err
}

func (s *instrumentedStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	start := time.Now()
	dctx := httpx.DriverContext(ctx)
	var (
		res driver.Result
		err error
	)
	if e, ok := s.base.(driver.StmtExecContext); ok {
		res, err = e.ExecContext(dctx, args)
	} else {
		var vals []driver.Value
		if vals, err = namedToValues(args); err == nil {
			if err = dctx.Err(); err == nil {
				res, err = s.base.Exec(vals) //nolint:staticcheck // fallback for drivers without StmtExecContext
			}
		}
	}
	observe(ctx, s.query, start, err)
	return res, err
}

func (s *instrumentedStmt) CheckNamedValue(nv *driver.NamedValue) error {
	if n, ok := s.base.(driver.NamedValueChecker); ok {
		return n.CheckNamedValue(nv)
	}
	if n, ok := s.conn.(driver.NamedValueChecker); ok {
		return n.CheckNamedValue(nv)
	}
	return driver.ErrSkip
}

func (s *instrumentedStmt) ColumnConverter(idx int) driver.ValueConverter { //nolint:staticcheck // see above
	if cc, ok := s.base.(driver.ColumnConverter); ok { //nolint:staticcheck // see above
		return cc.ColumnConverter(idx)
	}
	return driver.DefaultParameterConverter
}

// namedToValues mirrors database/sql's conversion for drivers without the context interfaces.
func namedToValues(named []driver.NamedValue) ([]driver.Value, error) {
	out := make([]driver.Value, len(named))
	for i, nv := range named {
		if nv.Name != "" {
			return nil, errors.New("db: driver does not support named parameters")
		}
		out[i] = nv.Value
	}
	return out, nil
}
