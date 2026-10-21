package care_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-M9-02 (D-133, gap report 05 FAM-M2): a companion acting with for_user_id on the owner's meds and appointments
// (show / store / update / the switches-only update) gets the member allow-list projection — never the owner's notes,
// topic, result note, prep, specialty, visit stage, subtitle or any other free text. The owner still gets her full JSON.

// memberHidden are the keys (and sentinel values) a member response must never carry.
var memberHidden = []string{
	`"notes"`, `"topic"`, `"result_note"`, `"prep"`, `"stage"`, `"specialty"`, `"subtitle"`, `"care_item_key"`,
	`"remind_before"`, `"remind_at"`, `"add_to_calendar"`, `"notify"`, "SENTINEL",
}

func assertMemberProjection(t *testing.T, r response, what string) {
	t.Helper()
	for _, h := range memberHidden {
		assert.NotContains(t, r.raw, h, what)
	}
}

// familyLink makes an active partner link owner → member with edit on meds and appointments.
func (e *env) familyLink(t *testing.T, owner, member uint64) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO companions (owner_id, companion_user_id, type, status, display_name, invited_at, accepted_at, created_at, updated_at)
		VALUES (?, ?, 'partner', 'active', 'Ali', '2026-09-01 09:00:00', '2026-09-01 09:00:00', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, owner, member)
	require.NoError(t, err)
	cid, err := res.LastInsertId()
	require.NoError(t, err)
	for _, s := range []string{"meds", "appointments"} {
		_, err = e.db.Exec(`INSERT INTO companion_grants (companion_id, section, level, created_at, updated_at)
			VALUES (?, ?, 'edit', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, cid, s)
		require.NoError(t, err)
	}
}

func TestMember_Medication_ForUserIsProjection(t *testing.T) {
	e := setup(t)
	owner, ownerTok := e.user(t, "09120009201")
	_, memberTok := e.user(t, "09120009202")
	member := uint64(0)
	require.NoError(t, e.db.QueryRow(`SELECT id FROM users WHERE mobile = '09120009202'`).Scan(&member))
	e.familyLink(t, owner, member)

	id := e.create(t, ownerTok, `{"title":"Iron","dose":"400","unit":"mcg","form":"tablet","times":["08:00"],"notes":"SENTINEL_OWNER_NOTE"}`)
	forUser := fmt.Sprintf("?for_user_id=%d", owner)

	// The owner still sees her full JSON, notes included.
	r := e.do(t, http.MethodGet, medPath(id, ""), ownerTok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "SENTINEL_OWNER_NOTE", r.data()["notes"])

	// GET as the member: the projection, with what the member card needs.
	r = e.do(t, http.MethodGet, medPath(id, forUser), memberTok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assertMemberProjection(t, r, "GET")
	assert.Equal(t, "Iron", r.data()["title"])
	assert.Equal(t, "tablet", r.data()["form"])
	assert.Equal(t, "400", r.data()["dose"])
	assert.Equal(t, []any{"08:00"}, r.data()["times"])

	// POST for the owner: stored with the member's note, answered without it.
	r = e.do(t, http.MethodPost, "/api/v1/care/medications", memberTok, "en",
		fmt.Sprintf(`{"for_user_id":%d,"title":"Folic","dose":"5","unit":"mg","form":"tablet","times":["09:00"],"notes":"SENTINEL_MEMBER_NOTE"}`, owner))
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assertMemberProjection(t, r, "POST")
	created := uint64(r.data()["id"].(float64))
	var notes string
	require.NoError(t, e.db.QueryRow(`SELECT notes FROM reminders WHERE id = ? AND user_id = ?`, created, owner).Scan(&notes))
	assert.Equal(t, "SENTINEL_MEMBER_NOTE", notes, "the write itself is unchanged")

	// PUT (full update) and the switches-only PUT.
	r = e.do(t, http.MethodPut, medPath(id, ""), memberTok, "en", fmt.Sprintf(`{"for_user_id":%d,"title":"Iron 2"}`, owner))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assertMemberProjection(t, r, "PUT")
	assert.Equal(t, "Iron 2", r.data()["title"])
	r = e.do(t, http.MethodPut, medPath(id, ""), memberTok, "en", fmt.Sprintf(`{"for_user_id":%d,"is_active":false}`, owner))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assertMemberProjection(t, r, "PUT switches")
	assert.Equal(t, false, r.data()["is_active"])

	// The owner's own reads are untouched by the member's edits.
	r = e.do(t, http.MethodGet, medPath(id, ""), ownerTok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "SENTINEL_OWNER_NOTE", r.data()["notes"])
	assert.Contains(t, r.data(), "notify")
}

func TestMember_Appointment_ForUserIsProjection(t *testing.T) {
	e := setup(t)
	owner, ownerTok := e.user(t, "09120009211")
	_, memberTok := e.user(t, "09120009212")
	member := uint64(0)
	require.NoError(t, e.db.QueryRow(`SELECT id FROM users WHERE mobile = '09120009212'`).Scan(&member))
	e.familyLink(t, owner, member)

	r := e.do(t, http.MethodPost, "/api/v1/care/appointments", ownerTok, "en",
		`{"kind":"in_person","with":"Dr. Rahimi","specialty":"SENTINEL_SPECIALTY","topic":"ultrasound","title":"Checkup",
		  "scheduled_at":"2026-09-30 10:30:00","location":"Clinic A","remind_before":"1d","notes":"SENTINEL_NOTE",
		  "prep":[{"text":"SENTINEL_PREP"}]}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	id := uint64(r.data()["id"].(float64))
	r = e.do(t, http.MethodPut, apptPath(id, ""), ownerTok, "en", `{"care_item_key":"nt_scan","stage":"result","result_note":"SENTINEL_RESULT"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	// The owner's full JSON.
	d := r.data()
	assert.Equal(t, "SENTINEL_NOTE", d["notes"])
	assert.Equal(t, "ultrasound", d["topic"])
	assert.Equal(t, "SENTINEL_RESULT", d["result_note"])
	assert.Equal(t, "result", d["stage"])
	assert.Len(t, d["prep"], 1)

	forUser := fmt.Sprintf("?for_user_id=%d", owner)
	r = e.do(t, http.MethodGet, apptPath(id, forUser), memberTok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assertMemberProjection(t, r, "GET")
	assert.NotContains(t, r.raw, "ultrasound")
	d = r.data()
	assert.Equal(t, "Checkup", d["title"])
	assert.Equal(t, "Dr. Rahimi", d["with"])
	assert.Equal(t, "Clinic A", d["location"])
	assert.Equal(t, "2026-09-30 10:30:00", d["scheduled_at"])
	assert.Equal(t, "scheduled", d["status"])

	r = e.do(t, http.MethodPost, "/api/v1/care/appointments", memberTok, "en",
		fmt.Sprintf(`{"for_user_id":%d,"kind":"in_person","topic":"lab","title":"Blood test","scheduled_at":"2026-10-02 09:00:00",
		  "notes":"SENTINEL_MEMBER_NOTE"}`, owner))
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assertMemberProjection(t, r, "POST")
	assert.NotContains(t, r.raw, `"lab"`)

	r = e.do(t, http.MethodPut, apptPath(id, ""), memberTok, "en",
		fmt.Sprintf(`{"for_user_id":%d,"location":"Clinic B","notes":"SENTINEL_MEMBER_EDIT"}`, owner))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assertMemberProjection(t, r, "PUT")
	assert.Equal(t, "Clinic B", r.data()["location"])
	r = e.do(t, http.MethodPut, apptPath(id, ""), memberTok, "en", fmt.Sprintf(`{"for_user_id":%d,"is_active":false}`, owner))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assertMemberProjection(t, r, "PUT switches")
	assert.Equal(t, false, r.data()["is_active"])

	// The owner still reads everything she typed (the delegated update keeps her result note and prep).
	r = e.do(t, http.MethodGet, apptPath(id, ""), ownerTok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "SENTINEL_RESULT", r.data()["result_note"])
	assert.Len(t, r.data()["prep"], 1)
}
