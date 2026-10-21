package httpx_test

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// T-M9-10 integration: a `SELECT SLEEP(6)` made with the request context is cancelled at the 5 s request deadline.
func TestRequestDeadline_CancelsSlowQuery(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set")
	}
	db, err := platformdb.OpenDSN(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var (
		queryErr error
		elapsed  time.Duration
	)
	app := fiber.New()
	app.Use(httpx.RequestDeadline(5*time.Second, nil))
	app.Get("/slow", func(c fiber.Ctx) error {
		start := time.Now()
		var v int
		queryErr = db.QueryRowContext(c.Context(), "SELECT SLEEP(6)").Scan(&v)
		elapsed = time.Since(start)
		return c.SendString("done")
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/slow", nil), fiber.TestConfig{Timeout: 15 * time.Second})
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	require.Error(t, queryErr, "the query must not run to completion")
	assert.True(t, errors.Is(queryErr, context.DeadlineExceeded), "got %v", queryErr)
	assert.GreaterOrEqual(t, elapsed, 4900*time.Millisecond)
	assert.Less(t, elapsed, 5900*time.Millisecond, "cancelled at the deadline, not after SLEEP(6)")

	// The pool is still usable after the cancelled query.
	var one int
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT 1").Scan(&one))
	assert.Equal(t, 1, one)
}
