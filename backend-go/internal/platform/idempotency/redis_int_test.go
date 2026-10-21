package idempotency_test

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/idempotency"
)

// The Lua scripts (finish / release) and SET NX against the real test Redis, not miniredis.
func TestRealRedis_ReplayAndTTL(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set")
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	c := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: addr}), "ritme-go-test-"+hex.EncodeToString(b)+":")
	t.Cleanup(func() { _ = c.Close() })

	var calls atomic.Int32
	app := fiber.New()
	app.Post("/w", idempotency.New(c, user, nil).Handler(), func(c fiber.Ctx) error {
		calls.Add(1)
		return c.Status(201).JSON(fiber.Map{"ok": true})
	})
	do := func() (int, string) {
		req := httptest.NewRequest(http.MethodPost, "/w", strings.NewReader(`{"a":1}`))
		req.Header.Set(idempotency.Header, rawKey)
		res, err := app.Test(req)
		require.NoError(t, err)
		body, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(body)
	}
	s1, b1 := do()
	s2, b2 := do()
	assert.Equal(t, 201, s1)
	assert.Equal(t, s1, s2)
	assert.Equal(t, b1, b2)
	assert.EqualValues(t, 1, calls.Load())

	key := c.Key(idempotency.Key(42, rawKey))
	ttl, err := c.Redis().TTL(t.Context(), key).Result()
	require.NoError(t, err)
	assert.InDelta(t, idempotency.TTL.Seconds(), ttl.Seconds(), 5)
	require.NoError(t, c.Redis().Del(t.Context(), key).Err())
}
