package care_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/care"
)

// seedReminders inserts n bare rows of type for userID in one statement.
func (e *env) seedReminders(t *testing.T, userID uint64, typ, scheduledAt string, n int) {
	t.Helper()
	rows := make([]string, n)
	args := make([]any, 0, n*3)
	for i := range rows {
		rows[i] = "(?, ?, 'x', ?, 'none', 1, '2026-09-01 09:00:00', '2026-09-01 09:00:00')"
		var at any
		if scheduledAt != "" {
			at = scheduledAt
		}
		args = append(args, userID, typ, at)
	}
	_, err := e.db.Exec(`INSERT INTO reminders (user_id, type, title, scheduled_at, recurrence, is_active, created_at, updated_at)
		VALUES `+strings.Join(rows, ","), args...)
	require.NoError(t, err)
}

func assertLimit(t *testing.T, r response, message string) {
	t.Helper()
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	assert.Equal(t, message, r.body["message"])
	assert.Equal(t, care.ErrorCodeLimitReached, r.body["error_code"])
	assert.Equal(t, map[string]any{"limit": []any{message}}, r.body["errors"])
}

func TestMedication_Cap(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000901")
	_, other := e.user(t, "09120000902")
	e.seedReminders(t, uid, "medication", "", care.MaxMedications-1)

	// One below the cap: still fine.
	id := e.create(t, tok, folic)

	// At the cap: 422 with the localized line, nothing inserted.
	r := e.do(t, http.MethodPost, "/api/v1/care/medications", tok, "fa", folic)
	assertLimit(t, r, care.T("messages.medication_limit", "fa"))
	assert.Contains(t, r.body["message"], "یادآورهای دارو")
	r = e.do(t, http.MethodPost, "/api/v1/care/medications", tok, "en", folic)
	assertLimit(t, r, "You have reached the maximum number of medication reminders. Delete one you no longer need to add a new one.")
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM reminders WHERE user_id = ? AND type = 'medication'`, uid).Scan(&n))
	assert.Equal(t, care.MaxMedications, n)

	// The cap is per user.
	e.create(t, other, folic)

	// Deleting one frees a slot.
	r = e.do(t, http.MethodDelete, medPath(id, ""), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	e.create(t, tok, folic)
}

func TestAppointment_Cap(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000903")
	// 100 upcoming (after `now`) scheduled appointments.
	e.seedReminders(t, uid, "appointment", "2026-10-10 10:00:00", care.MaxUpcomingAppointments)

	r := e.do(t, http.MethodPost, "/api/v1/care/appointments", tok, "en", nt)
	assertLimit(t, r, "You have reached the maximum number of appointments. Delete one you no longer need to add a new one.")

	// A past visit (logged after the fact) doesn't count against the upcoming cap…
	past := strings.Replace(nt, "2026-09-30 10:30:00", "2026-09-01 10:30:00", 1)
	e.createAppt(t, tok, past)

	// …but every row counts against the hard ceiling.
	e.seedReminders(t, uid, "appointment", "2026-01-01 10:00:00", care.MaxAppointments-care.MaxUpcomingAppointments-1)
	r = e.do(t, http.MethodPost, "/api/v1/care/appointments", tok, "fa", past)
	assertLimit(t, r, care.T("messages.appointment_limit", "fa"))
	assert.Contains(t, r.body["message"], "سقف تعداد")

	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM reminders WHERE user_id = ? AND type = 'appointment'`, uid).Scan(&n))
	assert.Equal(t, care.MaxAppointments, n, fmt.Sprintf("user %d", uid))
}
