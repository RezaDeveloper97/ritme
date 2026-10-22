package migrations_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	checkupstore "github.com/ritme/backend-go/internal/checkups/store"
	labstore "github.com/ritme/backend-go/internal/labs/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	vitalstore "github.com/ritme/backend-go/internal/vitals/store"
)

const healthTargetsVersion int64 = 99 // 00099_health_targets_confirmation_test_kind.sql

func insertLabReport(t *testing.T, conn *sql.DB, uid uint64, verifiedAt any) uint64 {
	t.Helper()
	res, err := conn.Exec(`INSERT INTO lab_reports (user_id, source, status, verified_at, created_at, updated_at)
		VALUES (?, 'upload', 'ready', ?, NOW(), NOW())`, uid, verifiedAt)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return uint64(id)
}

func insertLabMarkerRow(t *testing.T, conn *sql.DB, labID, uid uint64, name string) uint64 {
	t.Helper()
	res, err := conn.Exec(`INSERT INTO lab_markers (lab_id, user_id, name, source, created_at, updated_at)
		VALUES (?, ?, ?, 'extracted', NOW(), NOW())`, labID, uid, name)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return uint64(id)
}

// T-M10-17: the backfill confirms only markers of verified reports; the new table and columns are writable through
// their queries, cascade with the user, and Down restores the 00098 schema without touching older rows.
func TestHealthTargetsConfirmationMigration_UpDown(t *testing.T) {
	ctx := context.Background()
	conn := testdb.New(t)
	p, err := platformdb.NewMigrator(conn)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	uid := newV6User(t, conn, "09129990170")
	other := newV6User(t, conn, "09129990171")

	// Rows written before the migration: one verified report, one not.
	_, err = p.DownTo(ctx, healthTargetsVersion-1)
	require.NoError(t, err)
	require.False(t, columnExists(t, conn, "lab_markers", "confirmed_at"))
	verified := insertLabReport(t, conn, uid, "2026-10-01 09:30:00")
	unverified := insertLabReport(t, conn, uid, nil)
	mv1 := insertLabMarkerRow(t, conn, verified, uid, "Hb")
	mv2 := insertLabMarkerRow(t, conn, verified, uid, "TSH")
	mu := insertLabMarkerRow(t, conn, unverified, uid, "Ferritin")
	res, err := conn.Exec(`INSERT INTO checkup_types (category, title, performed_by, interval_months, created_at, updated_at)
		VALUES ('screening', '{"fa":"پاپ اسمیر"}', 'doctor', 36, NOW(), NOW())`)
	require.NoError(t, err)
	typeID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = p.Up(ctx)
	require.NoError(t, err)

	// Backfill: confirmed_at = verified_at on the verified report's markers only.
	lq := labstore.New(conn)
	want := time.Date(2026, 10, 1, 9, 30, 0, 0, civildate.Tehran)
	for _, id := range []uint64{mv1, mv2} {
		m, err := lq.GetLabMarker(ctx, labstore.GetLabMarkerParams{ID: id, LabID: verified, UserID: uid})
		require.NoError(t, err)
		require.True(t, m.ConfirmedAt.Valid, "marker %d of a verified report is confirmed", id)
		assert.True(t, want.Equal(m.ConfirmedAt.Time), "confirmed_at = verified_at, got %v", m.ConfirmedAt.Time)
	}
	m, err := lq.GetLabMarker(ctx, labstore.GetLabMarkerParams{ID: mu, LabID: unverified, UserID: uid})
	require.NoError(t, err)
	assert.False(t, m.ConfirmedAt.Valid, "unverified report: not confirmed")

	// ConfirmLabMarker: user-scoped.
	n, err := lq.ConfirmLabMarker(ctx, labstore.ConfirmLabMarkerParams{ID: mu, UserID: other, Now: v6Now()})
	require.NoError(t, err)
	assert.Zero(t, n, "another user's confirm never touches her marker")
	n, err = lq.ConfirmLabMarker(ctx, labstore.ConfirmLabMarkerParams{ID: mu, UserID: uid, Now: v6Now()})
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)
	m, err = lq.GetLabMarker(ctx, labstore.GetLabMarkerParams{ID: mu, LabID: unverified, UserID: uid})
	require.NoError(t, err)
	assert.True(t, m.ConfirmedAt.Valid)

	// vital_targets: upsert keeps one row per context, list is user-scoped, delete per context.
	vq := vitalstore.New(conn)
	require.NoError(t, vq.UpsertVitalTarget(ctx, vitalstore.UpsertVitalTargetParams{UserID: uid, Context: "fasting",
		LowMgdl: sql.NullInt16{Int16: 80, Valid: true}, HighMgdl: sql.NullInt16{Int16: 130, Valid: true}, SetBy: "user", Now: v6Now()}))
	require.NoError(t, vq.UpsertVitalTarget(ctx, vitalstore.UpsertVitalTargetParams{UserID: uid, Context: "fasting",
		LowMgdl: sql.NullInt16{Int16: 70, Valid: true}, HighMgdl: sql.NullInt16{Int16: 95, Valid: true}, SetBy: "doctor",
		Note: sql.NullString{String: "دکتر", Valid: true}, Now: v6Now()}))
	require.NoError(t, vq.UpsertVitalTarget(ctx, vitalstore.UpsertVitalTargetParams{UserID: uid, Context: "post_meal",
		HighMgdl: sql.NullInt16{Int16: 140, Valid: true}, SetBy: "doctor", Now: v6Now()}))
	targets, err := vq.ListVitalTargets(ctx, uid)
	require.NoError(t, err)
	require.Len(t, targets, 2)
	assert.Equal(t, "fasting", targets[0].Context)
	assert.Equal(t, sql.NullInt16{Int16: 70, Valid: true}, targets[0].LowMgdl)
	assert.Equal(t, sql.NullInt16{Int16: 95, Valid: true}, targets[0].HighMgdl)
	assert.Equal(t, "doctor", targets[0].SetBy)
	assert.Equal(t, "دکتر", targets[0].Note.String)
	assert.Equal(t, "post_meal", targets[1].Context)
	assert.False(t, targets[1].LowMgdl.Valid, "open lower bound")
	none, err := vq.ListVitalTargets(ctx, other)
	require.NoError(t, err)
	assert.Empty(t, none)
	n, err = vq.DeleteVitalTarget(ctx, vitalstore.DeleteVitalTargetParams{UserID: other, Context: "fasting"})
	require.NoError(t, err)
	assert.Zero(t, n)
	n, err = vq.DeleteVitalTarget(ctx, vitalstore.DeleteVitalTargetParams{UserID: uid, Context: "post_meal"})
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)

	// checkup_types.informational defaults to 0 on existing rows; checkup_records.test_kind round-trips.
	cq := checkupstore.New(conn)
	ct, err := cq.GetCheckupTypeForUser(ctx, checkupstore.GetCheckupTypeForUserParams{ID: uint64(typeID), LifeMode: "cycle"})
	require.NoError(t, err)
	assert.False(t, ct.Informational)
	_, err = conn.Exec(`UPDATE checkup_types SET informational = 1 WHERE id = ?`, typeID)
	require.NoError(t, err)
	ct, err = cq.GetCheckupTypeForUser(ctx, checkupstore.GetCheckupTypeForUserParams{ID: uint64(typeID), LifeMode: "cycle"})
	require.NoError(t, err)
	assert.True(t, ct.Informational)

	recID, err := cq.InsertCheckupRecord(ctx, checkupstore.InsertCheckupRecordParams{UserID: uid, CheckupTypeID: uint64(typeID),
		DoneOn: civildate.MustParse("2026-10-10"), Result: "normal", TestKind: sql.NullString{String: "hpv", Valid: true},
		CreatedAt: v6Now(), UpdatedAt: v6Now()})
	require.NoError(t, err)
	rec, err := cq.GetCheckupRecord(ctx, checkupstore.GetCheckupRecordParams{ID: uint64(recID), UserID: uid})
	require.NoError(t, err)
	assert.Equal(t, "hpv", rec.TestKind.String)
	require.NoError(t, cq.UpdateCheckupRecord(ctx, checkupstore.UpdateCheckupRecordParams{ID: uint64(recID), UserID: uid,
		DoneOn: civildate.MustParse("2026-10-11"), Result: "normal", TestKind: sql.NullString{String: "cotest", Valid: true},
		UpdatedAt: v6Now()}))
	rec, err = cq.GetCheckupRecord(ctx, checkupstore.GetCheckupRecordParams{ID: uint64(recID), UserID: uid})
	require.NoError(t, err)
	assert.Equal(t, "cotest", rec.TestKind.String)

	// FK cascade: deleting the user deletes her targets.
	_, err = conn.Exec(`DELETE FROM users WHERE id = ?`, uid)
	require.NoError(t, err)
	var c int
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM vital_targets WHERE user_id = ?`, uid).Scan(&c))
	assert.Zero(t, c, "vital_targets cascades")

	// Down restores 00098: table and columns gone, the older catalog row stays.
	require.NoError(t, vq.UpsertVitalTarget(ctx, vitalstore.UpsertVitalTargetParams{UserID: other, Context: "bedtime",
		LowMgdl: sql.NullInt16{Int16: 90, Valid: true}, SetBy: "user", Now: v6Now()}))
	_, err = p.DownTo(ctx, healthTargetsVersion-1)
	require.NoError(t, err)
	assert.False(t, tableExists(t, conn, "vital_targets"))
	assert.False(t, columnExists(t, conn, "lab_markers", "confirmed_at"))
	assert.False(t, columnExists(t, conn, "checkup_records", "test_kind"))
	assert.False(t, columnExists(t, conn, "checkup_types", "informational"))
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM checkup_types WHERE id = ?`, typeID).Scan(&c))
	assert.Equal(t, 1, c)

	_, err = p.Up(ctx)
	require.NoError(t, err)
	assert.True(t, tableExists(t, conn, "vital_targets"))
	assert.True(t, columnExists(t, conn, "lab_markers", "confirmed_at"))
	assert.True(t, columnExists(t, conn, "checkup_records", "test_kind"))
	assert.True(t, columnExists(t, conn, "checkup_types", "informational"))
}
