package care_test

// T-M12-32 (P-08): GET /care/today's next appointment is one bounded read — cancelled visits are skipped in SQL and
// at most one row comes back — while ListAppointmentsFrom (the family projection) still returns cancelled ones.

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/care/store"
)

// appt stores an appointment row directly (meta as given, NULL when empty) and returns its id.
func (e *env) appt(t *testing.T, uid uint64, at time.Time, meta string) uint64 {
	t.Helper()
	var m any
	if meta != "" {
		m = meta
	}
	res, err := e.db.Exec(`INSERT INTO reminders (user_id, type, title, scheduled_at, meta, created_at, updated_at)
		VALUES (?, 'appointment', 'visit', ?, ?, NOW(), NOW())`, uid, at, m)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	return uint64(id) //nolint:gosec // positive id
}

func TestNextAppointmentFrom_SkipsCancelledInSQL(t *testing.T) {
	e := setup(t)
	uid, _ := e.user(t, "09120000321")
	other, _ := e.user(t, "09120000322")
	q := store.New(e.db)
	ctx := context.Background()
	now := time.Date(2026, 10, 1, 9, 0, 0, 0, time.Local)
	next := func() []store.Reminder {
		t.Helper()
		rows, err := q.NextAppointmentFrom(ctx, store.NextAppointmentFromParams{UserID: uid, ScheduledAt: sql.NullTime{Time: now, Valid: true}})
		require.NoError(t, err)
		return rows
	}

	assert.Empty(t, next(), "no appointment → no row")

	// Past, then many cancelled ones ahead, then a scheduled one; another user's earlier visit never leaks in.
	e.appt(t, uid, now.Add(-time.Hour), `{"status":"scheduled"}`)
	for i := range 50 {
		e.appt(t, uid, now.Add(time.Duration(i+1)*time.Minute), `{"status":"cancelled"}`)
	}
	e.appt(t, other, now.Add(30*time.Second), `{"status":"scheduled"}`)
	scheduled := e.appt(t, uid, now.Add(2*time.Hour), `{"status":"scheduled"}`)
	got := next()
	require.Len(t, got, 1, "LIMIT 1: the cancelled rows ahead are not read")
	assert.Equal(t, scheduled, got[0].ID)

	// Boundaries: a row without a status (legacy) or with NULL meta reads as scheduled (care.ParseAppointment);
	// scheduled_at == now is still ahead.
	legacy := e.appt(t, uid, now.Add(90*time.Minute), `{"kind":"phone"}`)
	assert.Equal(t, legacy, next()[0].ID)
	bare := e.appt(t, uid, now.Add(80*time.Minute), "")
	assert.Equal(t, bare, next()[0].ID)
	atNow := e.appt(t, uid, now, `{"status":"scheduled"}`)
	assert.Equal(t, atNow, next()[0].ID)

	// The family projection's list keeps the cancelled ones (scheduled_at ≥ now, soonest first).
	all, err := q.ListAppointmentsFrom(ctx, store.ListAppointmentsFromParams{UserID: uid, ScheduledAt: sql.NullTime{Time: now, Valid: true}})
	require.NoError(t, err)
	assert.Len(t, all, 50+4)
	assert.Equal(t, atNow, all[0].ID)
}

func TestToday_NextAppointmentPastCancelledOnes(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000323")
	// Cancelled visits ahead of the real one (stored directly, the clock is real): next_appointment is the
	// scheduled one, read in the same 3-query budget.
	base := time.Now().Add(48 * time.Hour)
	for i := range 20 {
		e.appt(t, uid, base.Add(time.Duration(i)*time.Minute), `{"status":"cancelled"}`)
	}
	want := e.appt(t, uid, base.Add(3*time.Hour), `{"status":"scheduled"}`)
	r := e.today(t, tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	na, ok := r.data()["next_appointment"].(map[string]any)
	require.True(t, ok, r.raw)
	assert.Equal(t, fmt.Sprint(want), fmt.Sprint(uint64(na["id"].(float64))))
}
