package state_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/inputs"
	"github.com/ritme/backend-go/internal/cycle/revision"
	"github.com/ritme/backend-go/internal/cycle/state"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// recordingKV is the real Redis client that records the keys it reads.
type recordingKV struct {
	*cache.Client
	mu    sync.Mutex
	reads []string
}

func (k *recordingKV) Get(ctx context.Context, key string) (string, error) {
	k.mu.Lock()
	k.reads = append(k.reads, key)
	k.mu.Unlock()
	return k.Client.Get(ctx, key)
}

func (k *recordingKV) takeReads() []string {
	k.mu.Lock()
	defer k.mu.Unlock()
	r := k.reads
	k.reads = nil
	return r
}

func testRedis(t testing.TB) *cache.Client {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set (run `make test-int`)")
	}
	prefix := "ritme-go-test-state-" + time.Now().Format("150405.000000") + ":"
	c := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: addr}), prefix)
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func seedUser(t testing.TB, db *sql.DB, periods int) uint64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('S', '09120000031', NOW(), NOW())`)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	u := uint64(id) //nolint:gosec // auto-increment id
	_, err = db.Exec(`INSERT INTO user_profiles (user_id, period_duration, cycle_duration, calculation_status, created_at, updated_at)
		VALUES (?, 5, 28, 'completed', NOW(), NOW())`, u)
	require.NoError(t, err)
	first := civildate.FromTime(now).AddDays(-periods * 28)
	for i := range periods {
		addPeriodRow(t, db, u, first.AddDays(i*28))
	}
	return u
}

func addPeriodRow(t testing.TB, db interface {
	Exec(query string, args ...any) (sql.Result, error)
}, u uint64, start civildate.Date) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, is_confirmed, is_estimated, source, created_at, updated_at)
		VALUES (?, ?, ?, 1, 0, 'user_logged', NOW(), NOW())`, u, start.String(), start.AddDays(4).String())
	require.NoError(t, err)
}

func engineRev(t testing.TB, q *store.Queries, u uint64) uint64 {
	t.Helper()
	r, err := q.GetUserRevisions(context.Background(), u)
	if errors.Is(err, sql.ErrNoRows) {
		return 0
	}
	require.NoError(t, err)
	return r.EngineRev
}

// Coherence against MariaDB + Redis: a write bumps engine_rev in its transaction; the next For misses L1 and L2,
// reads only the new revision's key and returns the new state.
func TestForInt_WriteBumpServesNewState(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	q := store.New(db)
	kv := &recordingKV{Client: testRedis(t)}
	u := seedUser(t, db, 6)
	pr, _ := newProvider(t, q, kv, state.DefaultOptions()) // verify on

	rev := engineRev(t, q, u)
	s1, err := pr.For(ctx, u, rev, p1)
	require.NoError(t, err)
	assert.Len(t, s1.Periods(), 6)
	_, err = kv.Client.Get(ctx, state.L2Key(u, rev))
	require.NoError(t, err, "snapshot stored in Redis")

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	addPeriodRow(t, tx, u, civildate.FromTime(now).AddDays(-2))
	require.NoError(t, revision.Bump(ctx, tx, u))
	require.NoError(t, tx.Commit())
	rev2 := engineRev(t, q, u)
	require.Equal(t, rev+1, rev2)
	_ = kv.takeReads()

	s2, err := pr.For(ctx, u, rev2, p1)
	require.NoError(t, err)
	assert.Len(t, s2.Periods(), 7)
	for _, k := range kv.takeReads() {
		assert.Equal(t, state.L2Key(u, rev2), k, "only the new revision's key is read")
	}
}

// Stampede against MariaDB: 50 concurrent For calls for one cold user cost the queries of one inputs.Load
// (counted with the request query counter) and one Prepare.
func TestForInt_StampedeIsOneLoad(t *testing.T) {
	db := testdb.New(t)
	q := store.New(db)
	u := seedUser(t, db, 12)

	one, counter := metrics.WithQueryCounter(context.Background())
	_, err := inputs.Load(one, q, u, civildate.FromTime(now))
	require.NoError(t, err)
	perLoad := counter.Load()
	require.Positive(t, perLoad)

	pr, prepares := newProvider(t, q, testRedis(t), noVerify())
	ctx, counter := metrics.WithQueryCounter(context.Background())
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			_, err := pr.For(ctx, u, 9, p1)
			assert.NoError(t, err)
		})
	}
	wg.Wait()
	assert.Equal(t, perLoad, counter.Load(), "one database load for 50 callers")
	assert.Equal(t, int64(1), prepares.Load())
}

// Redis down → served from the database.
func TestForInt_RedisDown(t *testing.T) {
	db := testdb.New(t)
	u := seedUser(t, db, 4)
	down := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 50 * time.Millisecond, MaxRetries: -1}), "x:")
	t.Cleanup(func() { _ = down.Close() })
	pr, _ := newProvider(t, store.New(db), down, state.DefaultOptions())
	s, err := pr.For(context.Background(), u, 0, p1)
	require.NoError(t, err)
	assert.Len(t, s.Periods(), 4)
}

// The L2 budget with the real Redis round trip (GET + decode + Prepare of 24 periods).
func BenchmarkFor_L2HitPrepare24_Redis(b *testing.B) {
	benchL2Hit(b, testRedis(b))
}
