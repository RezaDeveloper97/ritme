package home

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/home/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

// T-M9-13b: each /home config table is a snapshot. A row written without a publish stays invisible (the snapshot is
// served), an admin write in this process (configcache.Changed, what httpadmin.Publishes calls) shows on the next
// request, and one published by a second process (another Hub on the same Redis) arrives over ritme-go:config-changed.
// Integration: `make test-int PKG=./internal/home/...`.
func TestConfigSnapshots_AdminWritesReachThisAndOtherProcesses(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set (run `make test-int`)")
	}
	sqlDB := testdb.New(t)
	q := store.New(sqlDB)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	prefix := "ritme-go-test-home-" + time.Now().Format("150405.000000") + ":"
	newClient := func() *cache.Client {
		c := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: addr}), prefix)
		t.Cleanup(func() { _ = c.Close() })
		return c
	}
	this, other := newClient(), newClient()
	configcache.SetEnabled(true)
	t.Cleanup(func() { configcache.SetEnabled(false) })
	configcache.Default.Start(ctx, this, nil)
	second := configcache.NewHub() // the second API process
	second.Start(ctx, other, nil)
	require.Eventually(t, func() bool {
		n, err := this.Redis().PubSubNumSub(ctx, this.Key(configcache.Channel)).Result()
		return err == nil && n[this.Key(configcache.Channel)] >= 2
	}, 5*time.Second, 20*time.Millisecond)

	exec := func(query string, args ...any) {
		t.Helper()
		_, err := sqlDB.ExecContext(ctx, query, args...)
		require.NoError(t, err)
	}
	cases := []struct {
		table  string
		insert func(n int)
		ids    func() []string // the marker of every visible row
	}{
		{"task_templates", func(n int) {
			exec("INSERT INTO task_templates (`key`, title, category, cycle_phase, is_active, sort_order) VALUES (?, '{\"fa\":\"t\"}', 'self_care', NULL, 1, ?)",
				marker("task", n), n)
		}, func() []string {
			rows, err := activeTaskTemplates(ctx, q, "luteal")
			require.NoError(t, err)
			out := []string{}
			for _, r := range rows {
				out = append(out, r.Key)
			}
			return out
		}},
		{"affirmations", func(n int) {
			exec("INSERT INTO affirmations (`text`, cycle_phase, is_active, sort_order) VALUES (JSON_OBJECT('fa', ?), NULL, 1, ?)", marker("aff", n), n)
		}, func() []string {
			rows, err := activeAffirmations(ctx, q, "luteal")
			require.NoError(t, err)
			out := []string{}
			for _, r := range rows {
				out = append(out, string(r.Text))
			}
			return out
		}},
		{"challenges", func(n int) {
			exec("INSERT INTO challenges (slug, title, is_active, sort_order) VALUES (?, '{\"fa\":\"c\"}', 1, ?)", marker("ch", n), n)
		}, func() []string {
			rows, err := activeChallenges(ctx, q)
			require.NoError(t, err)
			out := []string{}
			for _, r := range rows {
				out = append(out, strconv.FormatUint(r.ID, 10))
			}
			return out
		}},
		{"articles", func(n int) {
			exec("INSERT INTO articles (slug, title, is_published, sort_order) VALUES (?, '{\"fa\":\"a\"}', 1, ?)", marker("art", n), n)
		}, func() []string {
			rows, err := homeArticles(ctx, q, []string{"luteal"})
			require.NoError(t, err)
			out := []string{}
			for _, r := range rows {
				out = append(out, r.Slug)
			}
			return out
		}},
	}
	for _, tc := range cases {
		t.Run(tc.table, func(t *testing.T) {
			exec("DELETE FROM `" + tc.table + "`")
			configcache.Default.Invalidate(tc.table)
			tc.insert(1)
			require.Len(t, tc.ids(), 1, "first read loads the snapshot")

			tc.insert(2) // a direct DB write, nobody publishes
			assert.Len(t, tc.ids(), 1, "the snapshot is served")

			configcache.Changed(ctx, tc.table) // an admin write in this process
			assert.Len(t, tc.ids(), 2, "same process: next request")

			tc.insert(3)
			second.Changed(ctx, tc.table) // an admin write in the second process
			assert.Eventually(t, func() bool { return len(tc.ids()) == 3 }, 5*time.Second, 10*time.Millisecond,
				"this process sees the second one's write via ritme-go:config-changed")
		})
	}

	// A challenge that became a v2 program (challenge_programs) leaves the daily pool once published.
	t.Run("challenge_programs", func(t *testing.T) {
		before := len(cases[2].ids())
		require.Positive(t, before)
		rows, err := activeChallenges(ctx, q)
		require.NoError(t, err)
		id := rows[0].ID
		exec(`INSERT INTO challenge_programs (challenge_id, duration_days, created_at, updated_at) VALUES (?, 7, NOW(), NOW())`, id)
		configcache.Changed(ctx, "challenge_programs")
		after := cases[2].ids()
		assert.Len(t, after, before-1)
		assert.NotContains(t, after, strconv.FormatUint(id, 10))
	})
}

func marker(kind string, n int) string { return kind + "-" + strconv.Itoa(n) }
