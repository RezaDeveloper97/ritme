package usertz

import (
	"context"
	"database/sql"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/db/testdb"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// countingDB counts the statements that reach MariaDB, by kind.
type countingDB struct {
	*sql.DB
	updates, selects atomic.Int32
}

func (c *countingDB) count(q string) {
	switch {
	case strings.Contains(q, "UPDATE"):
		c.updates.Add(1)
	case strings.Contains(q, "SELECT"):
		c.selects.Add(1)
	}
}

func (c *countingDB) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	c.count(q)
	return c.DB.ExecContext(ctx, q, args...)
}

func (c *countingDB) QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row {
	c.count(q)
	return c.DB.QueryRowContext(ctx, q, args...)
}

func newUserWithProfile(t *testing.T, db *sql.DB, mobile string) uint64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO users (mobile, created_at, updated_at) VALUES (?, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_profiles (user_id, created_at, updated_at) VALUES (?, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, id)
	require.NoError(t, err)
	return uint64(id)
}

func storedZone(t *testing.T, db *sql.DB, uid uint64) (sql.NullString, string) {
	t.Helper()
	var zone sql.NullString
	var updatedAt string
	require.NoError(t, db.QueryRow(`SELECT timezone, updated_at FROM user_profiles WHERE user_id = ?`, uid).Scan(&zone, &updatedAt))
	return zone, updatedAt
}

// A valid new zone is stored once; the same zone again issues no UPDATE; a changed zone is stored again; updated_at
// is never touched (no revision bump, no profile change).
func TestIntegration_StoresChangedZoneOnce(t *testing.T) {
	raw := testdb.New(t)
	db := &countingDB{DB: raw}
	uid := newUserWithProfile(t, raw, "09120000001")
	m := New(profilestore.New(db), nil)
	app := newTestApp(m, uid)

	// No header, nothing stored → Tehran, one SELECT, no UPDATE.
	_, body := get(t, app, "/today", "")
	assert.Equal(t, "2026-07-18", body)
	m.Wait()
	assert.EqualValues(t, 0, db.updates.Load())
	assert.EqualValues(t, 1, db.selects.Load())

	for range 3 {
		_, body = get(t, app, "/today", "Europe/Berlin")
		assert.Equal(t, "2026-07-17", body)
	}
	m.Wait()
	assert.EqualValues(t, 1, db.updates.Load(), "the same zone again → no UPDATE")
	zone, updatedAt := storedZone(t, raw, uid)
	assert.Equal(t, sql.NullString{String: "Europe/Berlin", Valid: true}, zone)
	assert.Equal(t, "2026-09-23 09:00:00", strings.Replace(updatedAt[:19], "T", " ", 1))

	// A fresh process (another instance, a restart) knows nothing: the conditional UPDATE matches no row.
	m2 := New(profilestore.New(db), nil)
	app2 := newTestApp(m2, uid)
	_, _ = get(t, app2, "/today", "Europe/Berlin")
	m2.Wait()
	assert.EqualValues(t, 2, db.updates.Load())
	n, err := profilestore.New(raw).StoreUserTimezone(context.Background(), profilestore.StoreUserTimezoneParams{
		Timezone: sql.NullString{String: "Europe/Berlin", Valid: true}, UserID: uid,
	})
	require.NoError(t, err)
	assert.Zero(t, n, "same zone → the WHERE matches nothing")

	// Without the header the new process reads the stored zone.
	m3 := New(profilestore.New(raw), nil)
	_, body = get(t, newTestApp(m3, uid), "/today", "")
	assert.Equal(t, "2026-07-17", body)

	// A changed zone is stored again.
	_, _ = get(t, app, "/today", "Asia/Tehran")
	m.Wait()
	assert.EqualValues(t, 3, db.updates.Load())
	zone, _ = storedZone(t, raw, uid)
	assert.Equal(t, "Asia/Tehran", zone.String)
}

// No profile row yet: nothing is created (the UPDATE touches 0 rows) and the request answers normally.
func TestIntegration_NoProfileCreatesNothing(t *testing.T) {
	raw := testdb.New(t)
	res, err := raw.Exec(`INSERT INTO users (mobile, created_at, updated_at) VALUES ('09120000002', '2026-09-23 09:00:00', '2026-09-23 09:00:00')`)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	m := New(profilestore.New(raw), nil)
	status, body := get(t, newTestApp(m, uint64(id)), "/today", "Europe/Berlin")
	m.Wait()
	assert.Equal(t, 200, status)
	assert.Equal(t, "2026-07-17", body)
	var count int
	require.NoError(t, raw.QueryRow(`SELECT COUNT(*) FROM user_profiles WHERE user_id = ?`, id).Scan(&count))
	assert.Zero(t, count)
}
