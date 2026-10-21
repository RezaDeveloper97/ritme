package i18n_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/configcache"
)

// T-M9-13b: the language list is a configcache snapshot — one load, then no DB and no Redis read per request; Flush
// reloads it (and deletes the legacy Redis key older builds read).
func TestRegistry_SnapshotNoRedisOnHotPath(t *testing.T) {
	mr := miniredis.RunT(t)
	c := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "ritme-go:")
	require.NoError(t, mr.Set(c.Key(i18n.CacheKey), `[{"code":"xx"}]`)) // what an older build left behind: never read
	lister := withArabic()
	reg := i18n.NewRegistry(lister, c, nil).WithHub(configcache.NewHub())
	ctx := context.Background()

	before := mr.CommandCount()
	for range 5 {
		require.Equal(t, []string{"fa", "en", "ar"}, reg.All(ctx).Codes())
		require.Equal(t, "ar", reg.Resolve(ctx, "ar-SA"))
	}
	assert.Equal(t, 1, lister.calls, "one load for every request until a change")
	assert.Equal(t, before, mr.CommandCount(), "no Redis command on the hot path")

	lister.rows = lister.rows[:2]
	assert.Len(t, reg.All(ctx), 3, "a write without Flush waits for the TTL")
	require.NoError(t, reg.Flush(ctx))
	assert.False(t, mr.Exists(c.Key(i18n.CacheKey)), "Flush deletes the legacy key")
	assert.Equal(t, []string{"fa", "en"}, reg.All(ctx).Codes(), "Flush reloads the snapshot")
	assert.Equal(t, 2, lister.calls)
}

// A second registry on the same hub (another router in the process) sees a Flush through the hub.
func TestRegistry_FlushReachesOtherRegistriesOnTheHub(t *testing.T) {
	hub := configcache.NewHub()
	lister := withArabic()
	a := i18n.NewRegistry(lister, nil, nil).WithHub(hub)
	b := i18n.NewRegistry(lister, nil, nil).WithHub(hub)
	ctx := context.Background()
	require.Len(t, a.All(ctx), 3)
	require.Len(t, b.All(ctx), 3)

	lister.rows[0].IsDefault, lister.rows[1].IsDefault = false, true
	require.NoError(t, a.Flush(ctx))
	assert.Equal(t, "en", b.DefaultCode(ctx))
}

// An unreadable table before the first load yields Bootstrap and is retried; once loaded, a failed reload keeps the
// previous list.
func TestRegistry_SnapshotLoadErrors(t *testing.T) {
	hub := configcache.NewHub()
	broken := &fakeLister{err: errors.New("no such table")}
	reg := i18n.NewRegistry(broken, nil, nil).WithHub(hub)
	ctx := context.Background()
	assert.Equal(t, i18n.Bootstrap, reg.All(ctx))
	assert.Equal(t, i18n.Bootstrap, reg.All(ctx))
	assert.Equal(t, 2, broken.calls, "no snapshot stored: every request retries")

	broken.err, broken.rows = nil, withArabic().rows
	require.Len(t, reg.All(ctx), 3)
	broken.err = errors.New("db down")
	require.NoError(t, reg.Flush(ctx))
	assert.Len(t, reg.All(ctx), 3, "stale snapshot served on reload error")

	empty := i18n.NewRegistry(&fakeLister{rows: []store.ListActiveLanguagesRow{}}, nil, nil).WithHub(hub)
	assert.Equal(t, i18n.Bootstrap, empty.All(ctx), "empty table → bootstrap pair")
}
