package service_test

// The engine cache never changes a result: on the contract fixtures, every persona's day and month
// payloads are identical with the cache off, on (miss) and on (hit), and after period / profile
// writes the cached path returns the fresh result, never a stale entry.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/cache"
	"github.com/ritme/backend-go/internal/cycle/periods"
	"github.com/ritme/backend-go/internal/cycle/service"
	platformcache "github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

var personas = []uint64{1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015, 1016, 1018}

func redisCache(t *testing.T) *platformcache.Client {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set")
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	prefix := fmt.Sprintf("ritme-go-test-cycle-%d:", time.Now().UnixNano())
	c := platformcache.NewFromClient(rdb, prefix)
	t.Cleanup(func() {
		ctx := context.Background()
		keys, _ := rdb.Keys(ctx, prefix+"*").Result()
		if len(keys) > 0 {
			_ = rdb.Del(ctx, keys...).Err()
		}
		_ = rdb.Close()
	})
	return c
}

type payloads struct{ off, on *service.Service }

func (p payloads) day(t *testing.T, uid uint64, date, today civildate.Date, locale string) {
	t.Helper()
	ctx := context.Background()
	want, err := p.off.DayJSON(ctx, uid, date, today, locale)
	require.NoError(t, err)
	for pass := range 2 { // miss, then hit
		got, err := p.on.DayJSON(ctx, uid, date, today, locale)
		require.NoError(t, err)
		require.JSONEq(t, string(want.JSON), string(got.JSON), "user %d %s %s pass %d", uid, date, locale, pass)
		assert.Equal(t, want.Status, got.Status)
	}
}

func (p payloads) month(t *testing.T, uid uint64, y, m int, view string, today civildate.Date, locale string) {
	t.Helper()
	ctx := context.Background()
	want, err := p.off.MonthJSON(ctx, uid, y, m, view, today, locale)
	require.NoError(t, err)
	for pass := range 2 {
		got, err := p.on.MonthJSON(ctx, uid, y, m, view, today, locale)
		require.NoError(t, err)
		require.JSONEq(t, string(want.JSON), string(got.JSON), "user %d %d-%d %s %s pass %d", uid, y, m, view, locale, pass)
	}
}

func TestEngineCacheOnEqualsOff(t *testing.T) {
	rc := redisCache(t)
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "..", "contract", "fixtures", "dump.sql"))

	on := cache.New(rc, true, nil)
	p := payloads{off: service.New(db, cache.New(rc, false, nil)), on: service.New(db, on)}
	today := civildate.MustParse("2026-09-23")

	for _, uid := range personas {
		for _, locale := range []string{"fa", "en", "ar"} {
			for _, d := range []string{"2026-09-01", "2026-09-23", "2026-10-12"} {
				p.day(t, uid, civildate.MustParse(d), today, locale)
			}
			for _, view := range []string{"full", "calendar"} {
				p.month(t, uid, 2026, 9, view, today, locale)
			}
		}
	}
	keys, err := rc.Redis().Keys(context.Background(), rc.Key("cycle-engine:*")).Result()
	require.NoError(t, err)
	assert.NotEmpty(t, keys, "the cached path really went through Redis")
}

func TestEngineCacheNeverServesStaleResults(t *testing.T) {
	rc := redisCache(t)
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "..", "contract", "fixtures", "dump.sql"))

	p := payloads{off: service.New(db, cache.New(rc, false, nil)), on: service.New(db, cache.New(rc, true, nil))}
	ps := periods.NewService(db)
	ctx := context.Background()
	today := civildate.MustParse("2026-09-23")
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)
	check := func() {
		t.Helper()
		for _, uid := range []uint64{1004, 1010} {
			p.day(t, uid, today, today, "fa")
			p.month(t, uid, 2026, 9, "full", today, "en")
			p.month(t, uid, 2026, 9, "calendar", today, "fa")
		}
	}

	check()
	_, err := ps.Start(ctx, 1010, today, now) // overdue_10 starts a period today
	require.NoError(t, err)
	check()
	_, err = ps.End(ctx, 1010, &today, now.Add(time.Minute)) // end does not bump calculation_version
	require.NoError(t, err)
	check()
	end := civildate.MustParse("2026-09-18")
	_, err = ps.Update(ctx, 1004, 100406, civildate.MustParse("2026-09-12"), &end, now)
	require.NoError(t, err)
	check()
	var n int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM daily_health_logs WHERE user_id = 1004 AND log_date = '2026-09-23'").Scan(&n))
	require.Equal(t, 1, n, "fixture has a log today")
	// A write that bypasses the app (no version bump, same second): the hashed inputs still differ.
	_, err = db.ExecContext(ctx, "UPDATE daily_health_logs SET fatigue = 1 WHERE user_id = 1004 AND log_date = '2026-09-23'")
	require.NoError(t, err)
	check()
	_, err = db.ExecContext(ctx, "UPDATE recommendations SET is_active = 1 - is_active ORDER BY id LIMIT 1")
	require.NoError(t, err)
	check()
	_, err = db.ExecContext(ctx, "UPDATE user_profiles SET cycle_duration = 31 WHERE user_id = 1004")
	require.NoError(t, err)
	check()
}
