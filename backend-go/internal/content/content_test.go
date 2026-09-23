package content

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/cache"
)

func TestRegistryTTLBoundsTheSharedLanguageCache(t *testing.T) {
	mr := miniredis.RunT(t)
	c := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "ritme-go:")
	key := c.Key(i18n.CacheKey)
	require.NoError(t, mr.Set(key, "[]")) // written by i18n.Registry without TTL

	app := fiber.New()
	app.Get("/", RegistryTTL(c, 5*time.Minute), func(ctx fiber.Ctx) error { return ctx.SendStatus(204) })
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	require.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
	assert.Equal(t, 5*time.Minute, mr.TTL(key))

	mr.FastForward(4 * time.Minute) // a later request must not extend it
	_, err = app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	require.NoError(t, err)
	assert.Equal(t, time.Minute, mr.TTL(key))
}

func TestPhpTruthy(t *testing.T) {
	assert.False(t, phpTruthy(""))
	assert.False(t, phpTruthy("0"))
	assert.True(t, phpTruthy("00"))
	assert.True(t, phpTruthy("a"))
}
