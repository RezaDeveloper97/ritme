package httpadmin_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/platform/configcache"
)

// A successful admin write invalidates the config caches of its tables; a rejected one does not.
func TestPublishes(t *testing.T) {
	configcache.SetEnabled(true)
	t.Cleanup(func() { configcache.SetEnabled(false) })
	var loads atomic.Int64
	c := configcache.New("publish-test", []string{"publish_test_table"}, func(context.Context) (int64, error) {
		return loads.Add(1), nil
	})
	ctx := context.Background()
	_, _ = c.Get(ctx)

	app := fiber.New()
	app.Post("/ok", httpadmin.Publishes(func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) }, "publish_test_table"))
	app.Post("/bad", httpadmin.Publishes(func(fiber.Ctx) error { return errors.New("boom") }, "publish_test_table"))
	app.Post("/422", httpadmin.Publishes(func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusUnprocessableEntity) }, "publish_test_table"))

	for _, path := range []string{"/bad", "/422"} {
		_, err := app.Test(httptest.NewRequest(fiber.MethodPost, path, nil))
		require.NoError(t, err)
	}
	v, _ := c.Get(ctx)
	assert.Equal(t, int64(1), v, "failed writes keep the snapshot")

	_, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/ok", nil))
	require.NoError(t, err)
	v, _ = c.Get(ctx)
	assert.Equal(t, int64(2), v, "the next read after a committed write reloads")
}
