package store

// T-M12-32 (P-07): with 5k inbox rows the GET /home/notifications list query reads through
// user_notifications_user_id_created_at_id_index (goose 00131) — no filesort — and the order the API pages in is
// unchanged: newest first, equal created_at by id ascending. Dropping the index brings the filesort back (the check
// below fails without the migration).
//
// Kept in the generated store package so EXPLAIN runs the exact sqlc query text (listUserNotifications).

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const listingIndex = "user_notifications_user_id_created_at_id_index"

type plan struct{ key, extra string }

func explainList(t *testing.T, db *sql.DB, uid uint64, limit, offset int) plan {
	t.Helper()
	rows, err := db.Query("EXPLAIN "+listUserNotifications, uid, limit, offset)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	require.NoError(t, err)
	require.True(t, rows.Next(), "EXPLAIN returned no row")
	vals := make([]sql.NullString, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	require.NoError(t, rows.Scan(ptrs...))
	require.False(t, rows.Next(), "one table, one plan row")
	require.NoError(t, rows.Err())
	var p plan
	for i, c := range cols {
		switch strings.ToLower(c) {
		case "key":
			p.key = vals[i].String
		case "extra":
			p.extra = vals[i].String
		}
	}
	return p
}

func TestUserNotificationsListing_IndexNoFilesort(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	user := func(mobile string) uint64 {
		res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('u', ?, NOW(), NOW())`, mobile)
		require.NoError(t, err)
		id, _ := res.LastInsertId()
		return uint64(id) //nolint:gosec // positive id
	}
	uid, other := user("09120000881"), user("09120000882")

	// 5k rows for her: one per minute, except every 10th minute holds a pair with the same created_at (ties) and
	// every 7th row is read. Plus 5k rows of another user interleaved by id.
	_, err := db.Exec(`INSERT INTO user_notifications (user_id, type, title, read_at, created_at, updated_at)
		SELECT ?, 'notice', JSON_OBJECT('fa', CONCAT('n', seq)),
		       IF(seq % 7 = 0, '2026-11-01 08:00:00', NULL),
		       TIMESTAMP('2026-11-01 08:00:00') - INTERVAL (seq - (seq % 10 = 1)) MINUTE, NOW()
		FROM seq_1_to_5000`, uid)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_notifications (user_id, type, title, created_at, updated_at)
		SELECT ?, 'notice', JSON_OBJECT('fa', 'x'), TIMESTAMP('2026-11-01 08:00:00') - INTERVAL seq MINUTE, NOW()
		FROM seq_1_to_5000`, other)
	require.NoError(t, err)
	_, err = db.Exec(`ANALYZE TABLE user_notifications`)
	require.NoError(t, err)

	q := New(db)
	n, err := q.CountUserNotifications(ctx, uid)
	require.NoError(t, err)
	require.EqualValues(t, 5000, n)

	// EXPLAIN: first page, a deep page and past the end all read through the listing key, no filesort.
	for _, off := range []int{0, 4980, 6000} {
		p := explainList(t, db, uid, 20, off)
		t.Logf("with the index, offset %d: key=%s extra=%q", off, p.key, p.extra)
		assert.Equal(t, listingIndex, p.key, "offset %d: %+v", off, p)
		assert.NotContains(t, strings.ToLower(p.extra), "filesort", "offset %d: %+v", off, p)
	}

	// Order unchanged: every page, concatenated, is created_at DESC then id ASC over her rows only.
	type row struct {
		id uint64
		at string
	}
	want := func() []row {
		rs, err := db.Query(`SELECT id, created_at FROM user_notifications WHERE user_id = ?`, uid)
		require.NoError(t, err)
		defer func() { _ = rs.Close() }()
		var out []row
		for rs.Next() {
			var r row
			var at sql.NullTime
			require.NoError(t, rs.Scan(&r.id, &at))
			r.at = at.Time.Format("2006-01-02 15:04:05")
			out = append(out, r)
		}
		require.NoError(t, rs.Err())
		return out
	}()
	sort.Slice(want, func(i, j int) bool {
		if want[i].at != want[j].at {
			return want[i].at > want[j].at
		}
		return want[i].id < want[j].id
	})
	var got []uint64
	for off := int32(0); ; off += 100 {
		page, err := q.ListUserNotifications(ctx, ListUserNotificationsParams{UserID: uid, Limit: 100, Offset: off})
		require.NoError(t, err)
		if len(page) == 0 {
			break
		}
		for _, r := range page {
			got = append(got, r.ID)
		}
	}
	require.Len(t, got, len(want))
	ties := 0
	for i := range want {
		require.Equal(t, want[i].id, got[i], "position %d", i)
		if i > 0 && want[i].at == want[i-1].at {
			ties++
		}
	}
	assert.Positive(t, ties, "the fixture has equal created_at values (the id tie-break is exercised)")

	// Negative: without the index the same query filesorts — the EXPLAIN check above depends on the migration.
	_, err = db.Exec("DROP INDEX " + listingIndex + " ON user_notifications")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec("CREATE INDEX " + listingIndex + " ON user_notifications (user_id, created_at DESC, id)")
	})
	p := explainList(t, db, uid, 20, 0)
	t.Logf("without the index: key=%s extra=%q", p.key, p.extra)
	assert.NotEqual(t, listingIndex, p.key)
	assert.Contains(t, strings.ToLower(p.extra), "filesort", "%+v", p)
}
