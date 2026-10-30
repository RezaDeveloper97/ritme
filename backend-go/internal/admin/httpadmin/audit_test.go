package httpadmin

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

type fakeAudit struct {
	err  error
	rows []store.InsertAdminAuditLogParams
}

func (f *fakeAudit) InsertAdminAuditLog(_ context.Context, arg store.InsertAdminAuditLogParams) error {
	if f.err != nil {
		return f.err
	}
	f.rows = append(f.rows, arg)
	return nil
}

func healthApp(k *Kit) *fiber.App {
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Get("/read", func(c fiber.Ctx) error {
		if k != nil {
			c.Locals(kitKey, k)
		}
		if err := AuditHealthRead(c, quiet, 10, 7); err != nil {
			return err
		}
		return OK(c, map[string]any{"cycle_duration": 29})
	})
	return app
}

// T-M10-33: a health-data read whose audit row is not written answers 503 and no data (fail closed).
func TestAuditHealthRead_FailsClosed(t *testing.T) {
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	for name, k := range map[string]*Kit{
		"insert fails": {audit: &fakeAudit{err: errors.New("db down")}, logger: quiet},
		"no writer":    {logger: quiet},
		"no kit":       nil,
	} {
		res, err := healthApp(k).Test(httptest.NewRequest(fiber.MethodGet, "/read", nil))
		require.NoError(t, err, name)
		body, _ := io.ReadAll(res.Body)
		assert.Equal(t, fiber.StatusServiceUnavailable, res.StatusCode, name)
		assert.Contains(t, string(body), CodeAuditUnavailable, name)
		assert.NotContains(t, string(body), "cycle_duration", name)
	}

	w := &fakeAudit{}
	res, err := healthApp(&Kit{audit: w, logger: quiet}).Test(httptest.NewRequest(fiber.MethodGet, "/read", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, res.StatusCode)
	require.Len(t, w.rows, 1)
	assert.Equal(t, "health_access.read", w.rows[0].Action)
	assert.Equal(t, uint64(10), w.rows[0].TargetID)
	assert.JSONEq(t, `{"request_id":7}`, w.rows[0].Detail.String)
}
