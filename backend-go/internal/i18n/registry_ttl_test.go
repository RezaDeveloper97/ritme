package i18n_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/cache"
)

// T-M9-03 B5: the cached language list expires, so a direct DB edit is picked up; Flush still drops it at once.
func TestRegistry_CacheHasTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	c := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "ritme-go:")
	key := c.Key(i18n.CacheKey)
	reg := i18n.NewRegistry(withArabic(), c, nil)
	ctx := context.Background()

	require.Len(t, reg.All(ctx), 3)
	require.True(t, mr.Exists(key))
	ttl := mr.TTL(key)
	assert.Positive(t, ttl)
	assert.LessOrEqual(t, ttl, i18n.CacheTTL)

	require.NoError(t, reg.Flush(ctx))
	assert.False(t, mr.Exists(key))
}
