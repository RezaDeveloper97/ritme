package checkups_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/checkups"
	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/checkups/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

func exec(t *testing.T, db *sql.DB, q string, args ...any) int64 {
	t.Helper()
	res, err := db.ExecContext(context.Background(), q, args...)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

func newUser(t *testing.T, db *sql.DB, mobile string) uint64 {
	t.Helper()
	return uint64(exec(t, db, "INSERT INTO users (mobile, created_at, updated_at) VALUES (?, NOW(), NOW())", mobile)) //nolint:gosec // test ids
}

// The migration seeds the six default types with bilingual copy, and every seeded row is a valid
// engine type.
func TestSeededCatalog(t *testing.T) {
	db := testdb.New(t)
	q := store.New(db)
	rows, err := q.ListActiveCheckupTypesForUser(context.Background(), 0)
	require.NoError(t, err)

	keys := []string{}
	for _, r := range rows {
		keys = append(keys, r.Key.String)
		assert.False(t, r.UserID.Valid, r.Key.String)
		var title map[string]string
		require.NoError(t, json.Unmarshal(r.Title, &title))
		assert.NotEmpty(t, title["fa"], r.Key.String)
		assert.NotEmpty(t, title["en"], r.Key.String)
		assert.True(t, r.Why.Valid && r.PrepSteps.Valid, r.Key.String)
		assert.Positive(t, r.IntervalMonths)
	}
	assert.Equal(t, []string{"breast_self_exam", "clinical_breast_exam", "pap_smear", "blood_test", "dentist", "mammography"}, keys)

	self := checkups.TypeFromRow(rows[0])
	assert.True(t, self.CycleTimed())
	assert.Equal(t, engine.Type{ID: 1, Key: "breast_self_exam", Category: engine.CategoryMonthly, IntervalMonths: 1,
		CycleDayFrom: 7, CycleDayTo: 10, RemindLeadDays: 3, HideInPregnancy: true, IsActive: true, SortOrder: 1}, self)
	mammo := checkups.TypeFromRow(rows[5])
	assert.Equal(t, 12, mammo.IntervalMonths)
	assert.Equal(t, 24, mammo.IntervalMonthsMax)
	assert.Equal(t, 40, mammo.AgeMin)

	var findings []struct {
		Key       string `json:"key"`
		Exclusive bool   `json:"exclusive"`
	}
	require.NoError(t, json.Unmarshal(rows[0].FindingOptions.V, &findings))
	require.NotEmpty(t, findings)
	assert.Equal(t, "none", findings[0].Key)
	assert.True(t, findings[0].Exclusive)
}

// EngineInputs reads only the user's own rows (plus the shared catalog) and feeds the engine.
func TestEngineInputs_ScopedToUser(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	alice := newUser(t, db, "09120000001")
	bob := newUser(t, db, "09120000002")

	bobCustom := exec(t, db, `INSERT INTO checkup_types (user_id, category, title, performed_by, interval_months, created_at, updated_at)
		VALUES (?, 'custom', '{"fa":"چکاپ باب"}', 'doctor', 3, NOW(), NOW())`, bob)
	aliceCustom := exec(t, db, `INSERT INTO checkup_types (user_id, category, title, performed_by, interval_months, sort_order, created_at, updated_at)
		VALUES (?, 'custom', '{"fa":"چکاپ آلیس"}', 'doctor', 3, 10, NOW(), NOW())`, alice)
	exec(t, db, `INSERT INTO checkup_types (category, title, performed_by, interval_months, is_active, created_at, updated_at)
		VALUES ('annual', '{"fa":"غیرفعال"}', 'lab', 12, 0, NOW(), NOW())`)

	// Alice: blood test done, dentist overridden, Pap switched off. Bob: records of his own.
	exec(t, db, `INSERT INTO checkup_records (user_id, checkup_type_id, done_on, result, created_at, updated_at)
		VALUES (?, 4, '2026-02-01', 'normal', NOW(), NOW())`, alice)
	exec(t, db, `INSERT INTO checkup_records (user_id, checkup_type_id, done_on, result, next_due_on, created_at, updated_at)
		VALUES (?, 5, '2026-05-01', 'normal', '2026-09-20', NOW(), NOW())`, alice)
	exec(t, db, `INSERT INTO user_checkup_settings (user_id, checkup_type_id, enabled, remind, created_at, updated_at)
		VALUES (?, 3, 0, 1, NOW(), NOW())`, alice)
	exec(t, db, `INSERT INTO checkup_records (user_id, checkup_type_id, done_on, result, created_at, updated_at)
		VALUES (?, 2, '2026-09-01', 'normal', NOW(), NOW())`, bob)
	exec(t, db, `INSERT INTO user_checkup_settings (user_id, checkup_type_id, enabled, remind, created_at, updated_at)
		VALUES (?, 1, 0, 0, NOW(), NOW())`, bob)

	in, err := checkups.EngineInputs(ctx, store.New(db), alice)
	require.NoError(t, err)

	typeIDs := []uint64{}
	for _, ty := range in.Types {
		typeIDs = append(typeIDs, ty.ID)
		assert.NotEqual(t, uint64(bobCustom), ty.ID, "bob's custom checkup leaked to alice") //nolint:gosec // test ids
	}
	assert.Equal(t, []uint64{1, 2, 3, 4, 5, 6, uint64(aliceCustom)}, typeIDs) //nolint:gosec // test ids
	require.Len(t, in.Records, 2)
	for _, r := range in.Records {
		assert.NotEqual(t, uint64(2), r.TypeID, "bob's record leaked to alice")
	}
	assert.Equal(t, []engine.Setting{{TypeID: 3, Enabled: false, Remind: true}}, in.Settings)

	in.Birthday = civildate.New(1992, time.June, 15)
	now := clock.At(time.Date(2026, time.September, 23, 12, 0, 0, 0, civildate.Tehran))
	res := engine.Evaluate(in, now)

	status := map[uint64]engine.Status{}
	for _, it := range res.Items {
		status[it.TypeID] = it.Status
	}
	assert.Equal(t, map[uint64]engine.Status{
		1: engine.StatusDue, 2: engine.StatusDue, 3: engine.StatusDisabled, 4: engine.StatusUpToDate,
		5: engine.StatusOverdue, 6: engine.StatusNotYet, uint64(aliceCustom): engine.StatusDue, //nolint:gosec // test ids
	}, status)
	assert.Equal(t, engine.Summary{Total: 6, UpToDate: 2, Due: 3, Overdue: 1}, res.Summary)
}

// Deleting a user removes her records, settings and custom checkups (FK cascade).
func TestUserDeleteCascades(t *testing.T) {
	db := testdb.New(t)
	alice := newUser(t, db, "09120000003")
	exec(t, db, `INSERT INTO checkup_types (user_id, category, title, performed_by, interval_months, created_at, updated_at)
		VALUES (?, 'custom', '{"fa":"x"}', 'self', 1, NOW(), NOW())`, alice)
	exec(t, db, `INSERT INTO checkup_records (user_id, checkup_type_id, done_on, result, created_at, updated_at)
		VALUES (?, 1, '2026-09-01', 'normal', NOW(), NOW())`, alice)
	exec(t, db, `INSERT INTO user_checkup_settings (user_id, checkup_type_id, created_at, updated_at) VALUES (?, 1, NOW(), NOW())`, alice)

	_, err := db.ExecContext(context.Background(), "DELETE FROM users WHERE id = ?", alice)
	require.NoError(t, err)
	for table, want := range map[string]int{"checkup_types": 6, "checkup_records": 0, "user_checkup_settings": 0} {
		var n int
		require.NoError(t, db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&n))
		assert.Equal(t, want, n, table)
	}
}
