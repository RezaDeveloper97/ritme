package db_test

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// queryHistogram returns the sample count of db_query_duration_seconds{query=name}.
func queryHistogram(t *testing.T, name string) uint64 {
	t.Helper()
	mfs, err := metrics.Registry().Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() != "db_query_duration_seconds" {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "query" && lp.GetValue() == name {
					return m.GetHistogram().GetSampleCount()
				}
			}
		}
	}
	return 0
}

// The per-request counter equals the number of DB calls the route makes — across the paths database/sql takes:
// no-arg query (driver QueryContext), query with args (prepared statement), exec with args, and a transaction.
func TestInstrument_PerRequestCounterMatchesDBCalls(t *testing.T) {
	pool := testdb.New(t)

	var counted int64
	app := fiber.New()
	app.Use(metrics.HTTP())
	app.Get("/api/v1/probe/:id", func(c fiber.Ctx) error {
		ctx := c.Context()
		var n int
		if err := pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM languages").Scan(&n); err != nil { // 1
			return err
		}
		var code string
		err := pool.QueryRowContext(ctx, "-- name: ProbeLanguage :one\nSELECT code FROM languages WHERE id = ?", c.Params("id")).Scan(&code) // 2
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if _, err := pool.ExecContext(ctx, "UPDATE languages SET sort_order = sort_order WHERE id = ?", -1); err != nil { // 3
			return err
		}
		tx, err := pool.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback() }()
		if err := tx.QueryRowContext(ctx, "SELECT 1").Scan(&n); err != nil { // 4
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE languages SET sort_order = sort_order WHERE code = ?", "zz"); err != nil { // 5
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		counted = metrics.RequestQueries(c)
		return c.SendString(strconv.Itoa(n))
	})

	before := queryHistogram(t, "ProbeLanguage")
	resp, err := app.Test(httptest.NewRequest(nethttp.MethodGet, "/api/v1/probe/1", nil))
	require.NoError(t, err)
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	assert.Equal(t, int64(5), counted, "5 DB calls → counter 5 (begin/commit are not queries)")
	assert.Equal(t, before+1, queryHistogram(t, "ProbeLanguage"), "sqlc name label from the -- name: header")
	assert.Positive(t, queryHistogram(t, "raw"), "SQL without a header is labelled raw")
}

// The wrapper keeps database/sql semantics: sql.ErrNoRows, rollback, context cancellation; queries outside a request
// are timed but count nowhere.
func TestInstrument_PreservesSemantics(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()

	var code string
	err := pool.QueryRowContext(ctx, "SELECT code FROM languages WHERE id = ?", -1).Scan(&code)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	var before, after int
	require.NoError(t, pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM languages").Scan(&before))
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx,
		"INSERT INTO languages (code, name, english_name) VALUES (?, ?, ?)", "xx", "Test", "Test")
	require.NoError(t, err)
	var inTx int
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM languages").Scan(&inTx))
	assert.Equal(t, before+1, inTx, "the tx sees its own write")
	require.NoError(t, tx.Rollback())
	require.NoError(t, pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM languages").Scan(&after))
	assert.Equal(t, before, after, "rollback undid the insert")

	cctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = pool.ExecContext(cctx, "SELECT SLEEP(?)", 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 3*time.Second, "cancellation interrupts the query")

	// The pool still works after the cancelled query.
	require.NoError(t, pool.PingContext(ctx))
	var one int
	require.NoError(t, pool.QueryRowContext(ctx, "SELECT ?", 1).Scan(&one))
	assert.Equal(t, 1, one)
}

// A query above 200 ms logs a "slow query" warning naming the query (never its SQL or arguments).
func TestInstrument_SlowQueryWarns(t *testing.T) {
	pool := testdb.New(t)
	var logs strings.Builder
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	_, err := pool.ExecContext(context.Background(), "-- name: SlowProbe :exec\nSELECT SLEEP(?)", 0.25)
	require.NoError(t, err)
	assert.Contains(t, logs.String(), `"level":"WARN","msg":"slow query","query":"SlowProbe"`)
	assert.NotContains(t, logs.String(), "SLEEP")
}
