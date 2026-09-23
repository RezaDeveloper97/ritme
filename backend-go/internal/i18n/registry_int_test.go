package i18n_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// testCache is a Redis client under a unique prefix (Redis is shared with other packages).
func testCache(t *testing.T) (*cache.Client, *redis.Client, string) {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set (run `make test-int`)")
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	prefix := "ritme-go-test-" + hex.EncodeToString(b) + ":"
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	c := cache.NewFromClient(rdb, prefix)
	t.Cleanup(func() {
		_ = rdb.Del(context.Background(), prefix+i18n.CacheKey).Err()
		_ = c.Close()
	})
	return c, rdb, prefix
}

// Integration: `make test-int PKG=./internal/i18n/...`.
func TestRegistry_DatabaseAndRedisCache(t *testing.T) {
	db := testdb.New(t) // fa (default) + en, as seeded by the baseline migration
	c, rdb, prefix := testCache(t)
	ctx := context.Background()
	reg := i18n.NewRegistry(store.New(db), c, nil)

	_, err := db.ExecContext(ctx, `INSERT INTO languages (code, name, english_name, direction, is_active, is_default, sort_order)
		VALUES ('ar', 'العربية', 'Arabic', 'rtl', 1, 0, 3), ('de', 'Deutsch', 'German', 'ltr', 0, 0, 4)`)
	require.NoError(t, err)

	langs := reg.All(ctx)
	assert.Equal(t, []string{"fa", "en", "ar"}, langs.Codes(), "active rows in sort_order; inactive de excluded")
	assert.Equal(t, "fa", langs.DefaultCode())
	assert.Equal(t, "rtl", langs.Direction("ar"))
	assert.Equal(t, "ar", reg.Resolve(ctx, "ar-SA,en;q=0.5"))
	assert.Equal(t, "fa", reg.Resolve(ctx, "de"), "inactive language → default")

	raw, err := rdb.Get(ctx, prefix+"languages.registry").Result()
	require.NoError(t, err, "cached under <prefix>languages.registry")
	var cached []map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &cached))
	assert.Len(t, cached, 3)
	ttl, err := rdb.TTL(ctx, prefix+i18n.CacheKey).Result()
	require.NoError(t, err)
	assert.Less(t, int64(ttl), int64(0), "cached forever (no TTL)")

	// A write without Flush is not seen (rememberForever) …
	_, err = db.ExecContext(ctx, `UPDATE languages SET is_default = (code = 'en')`)
	require.NoError(t, err)
	assert.Equal(t, "fa", reg.DefaultCode(ctx))

	// … until the registry is flushed.
	require.NoError(t, reg.Flush(ctx))
	assert.Equal(t, "en", reg.DefaultCode(ctx))
	assert.Equal(t, "en", reg.Resolve(ctx, "xx"), "unknown locale → the new default")

	// An empty table falls back to the bootstrap pair.
	_, err = db.ExecContext(ctx, `UPDATE languages SET is_active = 0`)
	require.NoError(t, err)
	require.NoError(t, reg.Flush(ctx))
	assert.Equal(t, i18n.Bootstrap, reg.All(ctx))
}

func TestRegistry_NoCache(t *testing.T) {
	db := testdb.New(t)
	reg := i18n.NewRegistry(store.New(db), nil, nil)
	assert.Equal(t, []string{"fa", "en"}, reg.All(context.Background()).Codes())
	assert.NoError(t, reg.Flush(context.Background()))
}
