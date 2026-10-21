package db_test

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// deadlineApp serves GET /q, which runs query with the fiber.Ctx itself as the context (like ~457 service calls do),
// behind RequestDeadline(d) with /long allow-listed, and the production error handler.
func deadlineApp(pool *sql.DB, d time.Duration, query string, queryErr *error, elapsed *time.Duration) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Use(httpx.RequestDeadline(d, func(c fiber.Ctx) bool { return c.Path() == "/long" }))
	h := func(c fiber.Ctx) error {
		start := time.Now()
		var v int
		*queryErr = pool.QueryRowContext(c, query).Scan(&v)
		*elapsed = time.Since(start)
		if *queryErr != nil {
			return *queryErr
		}
		return c.SendString("done")
	}
	app.Get("/q", h)
	app.Get("/long", h)
	return app
}

func status(t *testing.T, app *fiber.App, path string) int {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, path, nil), fiber.TestConfig{Timeout: 15 * time.Second})
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode
}

// T-M9-10b acceptance: a handler passing `c` (not c.Context()) to a `SELECT SLEEP(6)` is cancelled at the 5 s
// request deadline, and the request fails with a 500, never a 401.
func TestRequestDeadline_ReachesQueriesMadeWithFiberCtx(t *testing.T) {
	pool := testdb.New(t)
	var (
		queryErr error
		elapsed  time.Duration
	)
	app := deadlineApp(pool, 5*time.Second, "SELECT SLEEP(6)", &queryErr, &elapsed)

	code := status(t, app, "/q")
	require.Error(t, queryErr, "the query must not run to completion")
	assert.True(t, errors.Is(queryErr, context.DeadlineExceeded), "got %v", queryErr)
	assert.GreaterOrEqual(t, elapsed, 4900*time.Millisecond)
	assert.Less(t, elapsed, 5900*time.Millisecond, "cancelled at the deadline, not after SLEEP(6)")
	assert.Equal(t, fiber.StatusInternalServerError, code)
	assert.NotEqual(t, fiber.StatusUnauthorized, code)

	// The pool is still usable after the cancelled query.
	var one int
	require.NoError(t, pool.QueryRowContext(context.Background(), "SELECT 1").Scan(&one))
	assert.Equal(t, 1, one)
}

// The other driver paths (prepared statement with args, exec, transaction) are bounded the same way; a short deadline
// keeps the test fast.
func TestRequestDeadline_BoundsArgsExecAndTx(t *testing.T) {
	pool := testdb.New(t)
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Use(httpx.RequestDeadline(300*time.Millisecond, nil))
	errs := map[string]error{}
	app.Get("/args", func(c fiber.Ctx) error {
		var v int
		errs["args"] = pool.QueryRowContext(c, "SELECT SLEEP(?)", 3).Scan(&v)
		return errs["args"]
	})
	app.Get("/exec", func(c fiber.Ctx) error {
		_, errs["exec"] = pool.ExecContext(c, "DO SLEEP(?)", 3)
		return errs["exec"]
	})
	app.Get("/tx", func(c fiber.Ctx) error {
		tx, err := pool.BeginTx(c, nil)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback() }()
		_, errs["tx"] = tx.ExecContext(c, "DO SLEEP(3)")
		return errs["tx"]
	})
	for _, path := range []string{"/args", "/exec", "/tx"} {
		start := time.Now()
		code := status(t, app, path)
		assert.Less(t, time.Since(start), 2*time.Second, path)
		assert.Equal(t, fiber.StatusInternalServerError, code, path)
		assert.True(t, errors.Is(errs[path[1:]], context.DeadlineExceeded), "%s: got %v", path, errs[path[1:]])
	}
}

// Long-running (allow-listed) routes and context.WithoutCancel(c) keep running past the request deadline.
func TestRequestDeadline_LongRunningAndDetachedAreNotBounded(t *testing.T) {
	pool := testdb.New(t)
	var (
		queryErr error
		elapsed  time.Duration
	)
	app := deadlineApp(pool, 300*time.Millisecond, "SELECT SLEEP(1)", &queryErr, &elapsed)
	assert.Equal(t, fiber.StatusOK, status(t, app, "/long"))
	require.NoError(t, queryErr)
	assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond)

	detached := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	detached.Use(httpx.RequestDeadline(300*time.Millisecond, nil))
	detached.Get("/q", func(c fiber.Ctx) error {
		var v int
		queryErr = pool.QueryRowContext(context.WithoutCancel(c), "SELECT SLEEP(1)").Scan(&v)
		return queryErr
	})
	assert.Equal(t, fiber.StatusOK, status(t, detached, "/q"))
	require.NoError(t, queryErr)
}
