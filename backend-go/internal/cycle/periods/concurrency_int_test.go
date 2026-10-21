package periods_test

// T-M9-04 / D-131: concurrent and replayed period writes over HTTP (guard → idempotency →
// handler) against MariaDB and the test Redis. No write may 500, the engine must never see two
// overlapping real periods, and an Idempotency-Key replay is answered from Redis.

import (
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/cycle/periods"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/idempotency"
)

const clientID = "0199c0de-0000-7000-8000-00000c0ffe04"

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

// fixed is the request clock: Tuesday 2026-10-06 09:00 Tehran.
var fixed = time.Date(2026, 10, 6, 9, 0, 0, 0, civildate.Tehran)

type env struct {
	db  *sql.DB
	app *fiber.App
	iss *passport.Issuer
}

// testCache is a client on the test Redis under a unique prefix; redisAddr "" → skip.
func testCache(t *testing.T, addr string) *cache.Client {
	t.Helper()
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	c := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: addr, MaxRetries: -1, DialTimeout: 200 * time.Millisecond}),
		"ritme-go-test-"+hex.EncodeToString(b)+":")
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func setup(t *testing.T, c *cache.Client) *env {
	t.Helper()
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))
	idem := idempotency.New(c, auth.CurrentUserID, quiet).Handler()
	h := periods.NewHandlers(periods.NewService(db), clock.Fixed(fixed))

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	g := app.Group("/api/v1/cycle")
	g.Post("/period/start", locale, guard, idem, h.Start)
	g.Post("/period/end", locale, guard, idem, h.End)
	g.Post("/period", locale, guard, idem, h.Store)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(t.Context(), uint64(id), time.Now()) //nolint:gosec // G115: test ids are small
	require.NoError(t, err)
	return uint64(id), tok.AccessToken //nolint:gosec // G115: test ids are small
}

type reply struct {
	status int
	body   string
	header http.Header
}

func (e *env) post(t *testing.T, path, token, idemKey, body string) reply {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if idemKey != "" {
		req.Header.Set(idempotency.Header, idemKey)
	}
	res, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		return reply{status: -1, body: err.Error()}
	}
	b, _ := io.ReadAll(res.Body)
	return reply{status: res.StatusCode, body: string(b), header: res.Header}
}

type periodRow struct {
	start string
	end   sql.NullString
}

func (e *env) periods(t *testing.T, uid uint64) []periodRow {
	t.Helper()
	rows, err := e.db.Query(`SELECT DATE_FORMAT(period_start_date, '%Y-%m-%d'), DATE_FORMAT(period_end_date, '%Y-%m-%d')
		FROM cycle_histories WHERE user_id = ? AND is_estimated = 0 ORDER BY period_start_date`, uid)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out []periodRow
	for rows.Next() {
		var r periodRow
		require.NoError(t, rows.Scan(&r.start, &r.end))
		out = append(out, r)
	}
	require.NoError(t, rows.Err())
	return out
}

func parallel(n int, fn func(i int) reply) []reply {
	out := make([]reply, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range n {
		wg.Go(func() {
			<-start
			out[i] = fn(i)
		})
	}
	close(start)
	wg.Wait()
	return out
}

func redisAddr(t *testing.T) string {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set")
	}
	return addr
}

func TestConcurrentStart_SameDate_OneRowNo500(t *testing.T) {
	e := setup(t, testCache(t, redisAddr(t)))
	uid, tok := e.user(t, "09120000401")

	replies := parallel(20, func(int) reply {
		return e.post(t, "/api/v1/cycle/period/start", tok, "", `{"date":"2026-10-01"}`)
	})
	for i, r := range replies {
		assert.Contains(t, []int{200, 201}, r.status, "request %d: %s", i, r.body)
	}
	rows := e.periods(t, uid)
	require.Len(t, rows, 1)
	assert.Equal(t, "2026-10-01", rows[0].start)
	assert.False(t, rows[0].end.Valid)
}

func TestConcurrentStore_OverlappingRanges_NoOverlapLosers422(t *testing.T) {
	e := setup(t, testCache(t, redisAddr(t)))
	uid, tok := e.user(t, "09120000402")

	// 20 different start dates, each a 7-day range: every pair overlaps its neighbours.
	replies := parallel(20, func(i int) reply {
		start := civildate.MustParse("2026-09-10").AddDays(i)
		body := fmt.Sprintf(`{"start_date":%q,"end_date":%q}`, start.String(), start.AddDays(6).String())
		return e.post(t, "/api/v1/cycle/period", tok, "", body)
	})
	ok, refused := 0, 0
	for i, r := range replies {
		switch r.status {
		case 200:
			ok++
		case 422:
			refused++
			assert.Contains(t, r.body, `"code":"period_overlap"`, "request %d", i)
		default:
			t.Errorf("request %d: status %d %s", i, r.status, r.body)
		}
	}
	rows := e.periods(t, uid)
	assert.Equal(t, len(rows), ok, "every 200 is one row")
	assert.Equal(t, 20, ok+refused)
	for i := 1; i < len(rows); i++ {
		prevEnd := rows[i-1].start
		if rows[i-1].end.Valid {
			prevEnd = rows[i-1].end.String
		}
		assert.Greater(t, rows[i].start, prevEnd, "periods %v and %v overlap", rows[i-1], rows[i])
	}
	assert.GreaterOrEqual(t, refused, 17, "at most three disjoint 7-day ranges fit in 26 days")
}

func TestIdempotencyKey_ReplayIsByteIdenticalAndWritesOnce(t *testing.T) {
	e := setup(t, testCache(t, redisAddr(t)))
	uid, tok := e.user(t, "09120000403")
	body := `{"start_date":"2026-09-20","end_date":"2026-09-24"}`

	first := e.post(t, "/api/v1/cycle/period", tok, "outbox-1", body)
	require.Equal(t, 200, first.status, first.body)
	// The row changes after the first answer; a replay must not see it.
	_, err := e.db.Exec(`UPDATE cycle_histories SET period_end_date = '2026-09-25' WHERE user_id = ?`, uid)
	require.NoError(t, err)

	second := e.post(t, "/api/v1/cycle/period", tok, "outbox-1", body)
	assert.Equal(t, first.status, second.status)
	assert.Equal(t, first.body, second.body, "byte-identical replay")
	assert.Equal(t, "true", second.header.Get(idempotency.ReplayedHeader))
	assert.Len(t, e.periods(t, uid), 1)

	reused := e.post(t, "/api/v1/cycle/period", tok, "outbox-1", `{"start_date":"2026-09-21","end_date":"2026-09-24"}`)
	assert.Equal(t, 422, reused.status, reused.body)
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(reused.body), &got))
	assert.Equal(t, idempotency.CodeReused, got["code"])
	assert.Len(t, e.periods(t, uid), 1)
}

func TestIdempotencyKey_ConcurrentReplays_OneWrite(t *testing.T) {
	e := setup(t, testCache(t, redisAddr(t)))
	uid, tok := e.user(t, "09120000404")

	replies := parallel(20, func(int) reply {
		return e.post(t, "/api/v1/cycle/period/start", tok, "tap-tap", `{"date":"2026-10-02"}`)
	})
	for i, r := range replies {
		assert.Equal(t, 200, r.status, "request %d: %s", i, r.body)
		assert.Equal(t, replies[0].body, r.body)
	}
	assert.Len(t, e.periods(t, uid), 1)
}

func TestIdempotencyKey_RedisDown_StillSucceeds(t *testing.T) {
	e := setup(t, testCache(t, "127.0.0.1:1")) // nothing listens there
	uid, tok := e.user(t, "09120000405")

	r := e.post(t, "/api/v1/cycle/period/start", tok, "offline-1", `{"date":"2026-10-01"}`)
	assert.Equal(t, 200, r.status, r.body)
	r = e.post(t, "/api/v1/cycle/period/start", tok, "offline-1", `{"date":"2026-10-01"}`)
	assert.Equal(t, 200, r.status, r.body, "the replay is processed normally (idempotent start)")
	assert.Len(t, e.periods(t, uid), 1)
}

func TestConcurrentWrites_OtherUserUnaffected(t *testing.T) {
	e := setup(t, testCache(t, redisAddr(t)))
	a, tokA := e.user(t, "09120000406")
	b, tokB := e.user(t, "09120000407")

	replies := parallel(10, func(i int) reply {
		tok := tokA
		if i%2 == 1 {
			tok = tokB
		}
		return e.post(t, "/api/v1/cycle/period/start", tok, "same-key", `{"date":"2026-10-01"}`)
	})
	for i, r := range replies {
		assert.Equal(t, 200, r.status, "request %d: %s", i, r.body)
	}
	assert.Len(t, e.periods(t, a), 1)
	assert.Len(t, e.periods(t, b), 1, "the same Idempotency-Key of another user is a different key")
}
