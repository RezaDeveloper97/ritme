package querybudget_test

import (
	"database/sql"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

// Row growth (T-M12-30b): the T-M12-30 home-read budgets were measured on fixtures without care, to-do, checkup or
// inbox rows, so an N+1 (one query per medication, list, checkup or notification) would pass them. Each route here is
// called warm by a user holding 1 row of its kind and by one holding 20; the counts must be equal (like
// TestMessagesSelectedConstantInRules).

// rowGrowthRoutes are the budget rows checked; the value is how many queries a non-empty list may add to the row's
// warm budget (measured on an empty fixture): /home/notifications skips its page query when the count is 0.
var rowGrowthRoutes = map[string]int64{"care today": 0, "todo": 0, "todo v6": 0, "checkups home": 0, "home notifications": 1}

// seedHomeRows gives uid n rows of every kind the routes above list (benchfixtures day: 2026-10-21, Tehran):
// active daily medications each taken today, upcoming appointments, to-do lists with two items each, custom checkup
// types each with a setting and a record, and inbox notifications (half read).
func seedHomeRows(t *testing.T, db *sql.DB, uid uint64, n int) {
	t.Helper()
	for i := range n {
		res, err := db.Exec(`INSERT INTO reminders (user_id, type, title, subtitle, recurrence, recurrence_time, starts_on, is_active, created_at, updated_at)
			VALUES (?, 'medication', ?, 'one tablet', 'daily', '08:00:00', '2026-10-01', 1, NOW(), NOW())`, uid, fmt.Sprintf("Pill %d", i))
		require.NoError(t, err)
		med, err := res.LastInsertId()
		require.NoError(t, err)
		exec(t, db, `INSERT INTO reminder_intakes (user_id, reminder_id, intake_date, slot, taken_at, created_at, updated_at)
			VALUES (?, ?, '2026-10-21', '08:00', '2026-10-21 08:05:00', NOW(), NOW())`, uid, med)
		exec(t, db, `INSERT INTO reminders (user_id, type, title, scheduled_at, recurrence, is_active, created_at, updated_at)
			VALUES (?, 'appointment', ?, ?, 'none', 1, NOW(), NOW())`, uid, fmt.Sprintf("Visit %d", i),
			fmt.Sprintf("2026-11-%02d 10:00:00", 1+i%28))

		res, err = db.Exec(`INSERT INTO todo_tasks (user_id, title, category, due_date, remind, created_at, updated_at)
			VALUES (?, ?, 'shopping', ?, 0, NOW(), NOW())`, uid, fmt.Sprintf("List %d", i), fmt.Sprintf("2026-10-%02d", 20+i%3))
		require.NoError(t, err)
		task, err := res.LastInsertId()
		require.NoError(t, err)
		exec(t, db, `INSERT INTO todo_items (task_id, user_id, title, sort_order, done_at, created_at, updated_at)
			VALUES (?, ?, 'Milk', 1, NULL, NOW(), NOW()), (?, ?, 'Bread', 2, '2026-10-20 09:00:00', NOW(), NOW())`, task, uid, task, uid)

		res, err = db.Exec(`INSERT INTO checkup_types (user_id, category, title, performed_by, interval_months, sort_order, created_at, updated_at)
			VALUES (?, 'annual', ?, 'doctor', 12, ?, NOW(), NOW())`, uid, fmt.Sprintf(`{"en":"Custom %d","fa":"سفارشی %d"}`, i, i), 100+i)
		require.NoError(t, err)
		typ, err := res.LastInsertId()
		require.NoError(t, err)
		exec(t, db, `INSERT INTO user_checkup_settings (user_id, checkup_type_id, enabled, remind, created_at, updated_at)
			VALUES (?, ?, 1, 1, NOW(), NOW())`, uid, typ)
		exec(t, db, `INSERT INTO checkup_records (user_id, checkup_type_id, done_on, result, next_due_on, created_at, updated_at)
			VALUES (?, ?, '2026-03-10', 'normal', '2027-03-10', NOW(), NOW())`, uid, typ)

		var read any
		if i%2 == 1 {
			read = "2026-10-20 12:00:00"
		}
		exec(t, db, `INSERT INTO user_notifications (user_id, type, title, body, read_at, created_at, updated_at)
			VALUES (?, 'care', ?, '{"en":"body"}', ?, NOW(), NOW())`, uid, fmt.Sprintf(`{"en":"Note %d"}`, i), read)
	}
}

func TestHomeReadsConstantInRows(t *testing.T) {
	db := testdb.New(t)
	keysPath, err := filepath.Abs(keysDir)
	require.NoError(t, err)
	tokens := map[int]string{}
	for i, n := range []int{1, 20} {
		uid, tok := fixtureToken(t, db, keysPath, enums.LifeModeCycle, fmt.Sprintf("0912000%04d", 8300+i))
		seedHomeRows(t, db, uid, n)
		tokens[n] = tok
	}

	configcache.SetEnabled(true)
	defer configcache.SetEnabled(false)
	for _, name := range slices.Sorted(maps.Keys(rowGrowthRoutes)) {
		b := budgetOf(t, name)
		t.Run(name, func(t *testing.T) {
			app := newApp(t, db, keysPath, b.Engine)
			warm := func(n int) (int64, string) {
				_, _ = call(t, app, b.Path, tokens[n]) // fills the caches for this user
				return call(t, app, b.Path, tokens[n])
			}
			one, oneBody := warm(1)
			twenty, twentyBody := warm(20)
			t.Logf("%s warm: 1 row %d queries, 20 rows %d queries", b.Path, one, twenty)
			assert.Equal(t, one, twenty, "%s: query count grows with the rows (N+1)", b.Path)
			assert.LessOrEqual(t, twenty, b.Warm+rowGrowthRoutes[name], "%s: over its warm budget with rows", b.Path)
			// The seeded rows are really on the screen (the fixture is not silently empty).
			assert.NotEqual(t, oneBody, twentyBody)
		})
	}
}

// failRecorder is an assert.TestingT that records a failure instead of failing the test.
type failRecorder struct{ failed bool }

func (r *failRecorder) Errorf(string, ...any) { r.failed = true }

// TestTightenedRowFailsWhenExceeded: the rows T-M12-30b lowered sit exactly at their measured count, so one query
// more fails them. The same request with one extra query per request (a middleware, standing in for a regression)
// goes through TestQueryBudgets' own check, which must reject it; without the extra query the row passes.
func TestTightenedRowFailsWhenExceeded(t *testing.T) {
	db := testdb.New(t)
	keysPath, err := filepath.Abs(keysDir)
	require.NoError(t, err)
	_, tok := fixtureToken(t, db, keysPath, enums.LifeModeCycle, "09120008400")
	regression := func(c fiber.Ctx) error {
		if _, err := db.ExecContext(c, "SELECT 1"); err != nil {
			return err
		}
		return c.Next()
	}
	defer configcache.SetEnabled(false)
	for _, name := range []string{"todo", "todo v6"} {
		b := budgetOf(t, name)
		t.Run(name, func(t *testing.T) {
			measure := func(extra ...fiber.Handler) (int64, int64) {
				configcache.SetEnabled(false)
				app := newApp(t, db, keysPath, b.Engine, extra...)
				cold, _ := call(t, app, b.Path, tok)
				configcache.SetEnabled(true)
				_, _ = call(t, app, b.Path, tok)
				warm, _ := call(t, app, b.Path, tok)
				return cold, warm
			}
			cold, warm := measure()
			assert.Equal(t, b.Warm, warm, "the row is tight: budget = measured warm count")
			assert.True(t, withinBudget(t, b, cold, warm))

			cold, warm = measure(regression)
			rec := &failRecorder{}
			assert.False(t, withinBudget(rec, b, cold, warm), "one query more (%d / %d) must exceed %d / %d", cold, warm, b.Cold, b.Warm)
			assert.True(t, rec.failed)
		})
	}
}
