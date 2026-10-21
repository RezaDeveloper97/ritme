package usertz

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// fakeStore is an in-memory user_profiles.timezone with call counters.
type fakeStore struct {
	mu            sync.Mutex
	zones         map[uint64]sql.NullString
	reads, writes int
	readErr       error
}

func (f *fakeStore) GetUserTimezone(_ context.Context, uid uint64) (sql.NullString, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if f.readErr != nil {
		return sql.NullString{}, f.readErr
	}
	z, ok := f.zones[uid]
	if !ok {
		return sql.NullString{}, sql.ErrNoRows
	}
	return z, nil
}

func (f *fakeStore) StoreUserTimezone(_ context.Context, arg profilestore.StoreUserTimezoneParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	if z, ok := f.zones[arg.UserID]; ok && (!z.Valid || z.String != arg.Timezone.String) {
		f.zones[arg.UserID] = arg.Timezone
		return 1, nil
	}
	return 0, nil
}

// instant is 2026-07-17T21:00Z: 00:30 on the 18th in Tehran, 23:00 on the 17th in Berlin.
var instant = time.Date(2026, 7, 17, 21, 0, 0, 0, time.UTC)

func newTestApp(m *Middleware, uid uint64) *fiber.App {
	m.logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	m.userID = func(fiber.Ctx) (uint64, bool) { return uid, uid != 0 }
	app := fiber.New()
	app.Use(m.Handler())
	app.Get("/today", func(c fiber.Ctx) error {
		return c.SendString(civildate.LocalToday(c, clock.At(instant)).String())
	})
	app.Get("/today-ctx", func(c fiber.Ctx) error { // through c.Context(), as services receive it
		return c.SendString(civildate.LocalToday(c.Context(), clock.At(instant)).String())
	})
	app.Get("/fail", func(fiber.Ctx) error { return fiber.ErrUnprocessableEntity })
	return app
}

func get(t *testing.T, app *fiber.App, path, zone string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, path, nil)
	if zone != "" {
		req.Header.Set(Header, zone)
	}
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(body)
}

func TestParse(t *testing.T) {
	for _, ok := range []string{"Asia/Tehran", "Europe/Berlin", "America/Argentina/Buenos_Aires", "UTC", "Etc/GMT+3", " Asia/Tehran "} {
		loc, valid := Parse(ok)
		assert.True(t, valid, ok)
		assert.NotNil(t, loc, ok)
	}
	for _, bad := range []string{"", "Local", "Mars/Olympus", "../../etc/passwd", "Asia/Tehran\x00", "Asia/Tehran zone",
		"+03:30", "Asia/" + string(make([]byte, 70))} {
		_, valid := Parse(bad)
		assert.False(t, valid, bad)
	}
	assert.Equal(t, "Asia/Tehran", ZoneOf(sql.NullString{}).String())
	assert.Equal(t, "Asia/Tehran", ZoneOf(sql.NullString{String: "nope", Valid: true}).String())
	assert.Equal(t, "Europe/Berlin", ZoneOf(sql.NullString{String: "Europe/Berlin", Valid: true}).String())
	assert.Equal(t, "Europe/Berlin", ZoneFor(profilestore.UserProfile{Timezone: sql.NullString{String: "Europe/Berlin", Valid: true}}).String())
}

func TestMiddleware_Precedence(t *testing.T) {
	st := &fakeStore{zones: map[uint64]sql.NullString{
		1: {},                                     // profile, no stored zone
		2: {String: "Europe/Berlin", Valid: true}, // stored Berlin
	}}
	m := New(st, nil)

	// Guest, no header → Tehran (today's behaviour); no DB read without a user.
	app := newTestApp(m, 0)
	_, body := get(t, app, "/today", "")
	assert.Equal(t, "2026-07-18", body)
	_, body = get(t, app, "/today", "Europe/Berlin")
	assert.Equal(t, "2026-07-17", body)
	_, body = get(t, app, "/today", "Asia/Tehran")
	assert.Equal(t, "2026-07-18", body)
	_, body = get(t, app, "/today", "Not/AZone") // invalid → ignored
	assert.Equal(t, "2026-07-18", body)
	m.Wait()
	assert.Zero(t, st.reads)
	assert.Zero(t, st.writes, "guests never store a zone")

	// User 1: no header, no stored zone → Tehran.
	app = newTestApp(New(st, nil), 1)
	_, body = get(t, app, "/today-ctx", "")
	assert.Equal(t, "2026-07-18", body)

	// User 2: no header → the stored zone; a header wins over it.
	m2 := New(st, nil)
	app = newTestApp(m2, 2)
	_, body = get(t, app, "/today", "")
	assert.Equal(t, "2026-07-17", body)
	_, body = get(t, app, "/today", "Asia/Tehran")
	assert.Equal(t, "2026-07-18", body)
	m2.Wait()
}

func TestMiddleware_StoresChangedZoneOnce(t *testing.T) {
	st := &fakeStore{zones: map[uint64]sql.NullString{7: {}}}
	m := New(st, nil)
	app := newTestApp(m, 7)

	for range 3 {
		_, body := get(t, app, "/today", "Europe/Berlin")
		assert.Equal(t, "2026-07-17", body)
	}
	m.Wait()
	assert.Equal(t, 1, st.writes, "the same zone again issues no UPDATE")
	assert.Equal(t, sql.NullString{String: "Europe/Berlin", Valid: true}, st.zones[7])

	_, _ = get(t, app, "/today", "Asia/Tokyo")
	m.Wait()
	assert.Equal(t, 2, st.writes)
	// After the change, a request without the header reads the newly stored zone, without a SELECT.
	_, body := get(t, app, "/today", "")
	assert.Equal(t, "2026-07-18", body) // 06:00 on the 18th in Tokyo
	assert.Zero(t, st.reads)

	// An error response still stores (it is not about the zone); an invalid header never does.
	_, _ = get(t, app, "/fail", "Europe/Paris")
	_, _ = get(t, app, "/fail", "garbage")
	m.Wait()
	assert.Equal(t, 3, st.writes)
}

// The stored-zone read failing never fails the request: Tehran.
func TestMiddleware_ReadErrorFallsBackToTehran(t *testing.T) {
	st := &fakeStore{readErr: assert.AnError}
	app := newTestApp(New(st, nil), 9)
	status, body := get(t, app, "/today", "")
	assert.Equal(t, 200, status)
	assert.Equal(t, "2026-07-18", body)
}

// A remembered zone expires, so a zone changed by another instance is picked up again.
func TestMiddleware_KnownZoneExpires(t *testing.T) {
	st := &fakeStore{zones: map[uint64]sql.NullString{3: {String: "Europe/Berlin", Valid: true}}}
	m := New(st, nil)
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return now }
	app := newTestApp(m, 3)
	_, _ = get(t, app, "/today", "")
	_, _ = get(t, app, "/today", "")
	assert.Equal(t, 1, st.reads)
	now = now.Add(knownTTL + time.Second)
	_, _ = get(t, app, "/today", "")
	assert.Equal(t, 2, st.reads)
}
