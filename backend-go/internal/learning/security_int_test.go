package learning_test

// B-N8-01b — courses security fixes (audit of B-N8-01): enumeration, caps, cooldown, rename review, SMS skips,
// bounded claim, allow-list gate.

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/learning"
	"github.com/ritme/backend-go/internal/learning/store"
)

func (e *env) grant(t *testing.T, tok string, cid int, phones ...string) response {
	t.Helper()
	return e.do(t, http.MethodPost, "/api/instructor/v1/grants", tok, map[string]any{
		"phones": phones, "scope": "course", "target_id": cid, "duration": "unlimited",
	})
}

func (e *env) instructorID(t *testing.T, mobile string) uint64 {
	t.Helper()
	var id uint64
	require.NoError(t, e.db.QueryRow(`SELECT i.id FROM learning_instructors i JOIN users u ON u.id = i.user_id WHERE u.mobile = ?`, mobile).Scan(&id))
	return id
}

// bulkGrants inserts n pending grants of the instructor (phones 0915xxxxxxx unless phone is set).
func (e *env) bulkGrants(t *testing.T, insID uint64, cid, n int, phone string, created time.Time) {
	t.Helper()
	vals := make([]string, 0, n)
	args := make([]any, 0, n*4)
	for i := range n {
		p := phone
		if p == "" {
			p = fmt.Sprintf("0915%07d", i)
		}
		vals = append(vals, "(?, ?, ?, 'unlimited', 'pending', ?, ?)")
		args = append(args, insID, p, cid, created, created)
	}
	_, err := e.db.Exec(`INSERT INTO learning_grants (instructor_id, phone, course_id, duration, status, created_at, updated_at) VALUES `+strings.Join(vals, ","), args...)
	require.NoError(t, err)
}

func strip(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		if k != "id" && k != "phone" {
			out[k] = v
		}
	}
	return out
}

// H1: the same answer (and the same list row) for a number with and without an account, until the student opens a
// course of the instructor.
func TestGrant_NoPhoneEnumeration(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila")
	cid, _ := e.publishedCourse(t, itok, "Course", 1)
	_, stok := e.user(t, "09121110001")

	reg := e.grant(t, itok, cid, "09121110001")
	unk := e.grant(t, itok, cid, "09129990001")
	require.Equal(t, 201, reg.status, reg.raw)
	require.Equal(t, 201, unk.status, unk.raw)
	assert.Equal(t, strip(unk.data()["grants"].([]any)[0].(map[string]any)), strip(reg.data()["grants"].([]any)[0].(map[string]any)))
	assert.Equal(t, unk.data()["summary"], reg.data()["summary"])
	assert.Equal(t, unk.body["message"], reg.body["message"])

	r := e.do(t, http.MethodGet, "/api/instructor/v1/students", itok, nil)
	require.Equal(t, 200, r.status, r.raw)
	rows := r.data()["students"].([]any)
	require.Len(t, rows, 2)
	assert.Equal(t, strip(rows[0].(map[string]any)), strip(rows[1].(map[string]any)))
	assert.EqualValues(t, 2, r.data()["counts"].(map[string]any)["pending"])
	r = e.do(t, http.MethodGet, "/api/instructor/v1/courses/"+fmt.Sprint(cid), itok, nil)
	require.Equal(t, 200, r.status, r.raw)

	// She opens the course: now the instructor sees her.
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/learning/courses/%d", cid), stok, nil)
	require.Equal(t, 200, r.status, r.raw)
	r = e.do(t, http.MethodGet, "/api/instructor/v1/students?status=active", itok, nil)
	rows = r.data()["students"].([]any)
	require.Len(t, rows, 1)
	assert.Equal(t, true, rows[0].(map[string]any)["registered"])
	assert.Equal(t, "User 0001", rows[0].(map[string]any)["student_name"])
}

// H1: a target with no published course opens nothing and texts nobody.
func TestGrant_RefusedForEmptyTarget(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila")
	r := e.do(t, http.MethodPost, "/api/instructor/v1/courses", itok, map[string]any{"kind": "course", "title": "Draft"})
	require.Equal(t, 201, r.status, r.raw)
	draft := idOf(t, r.data()["course"].(map[string]any)["id"])
	r = e.grant(t, itok, draft, "09129990001")
	require.Equal(t, 422, r.status, r.raw)
	assert.Contains(t, r.body["errors"].(map[string]any), "target_id")

	r = e.do(t, http.MethodPost, "/api/instructor/v1/groups", itok, map[string]any{"name": "Empty"})
	require.Equal(t, 201, r.status, r.raw)
	gid := idOf(t, r.data()["group"].(map[string]any)["id"])
	r = e.do(t, http.MethodPost, "/api/instructor/v1/grants", itok, map[string]any{"phones": []string{"09129990001"}, "scope": "group", "target_id": gid, "duration": "30"})
	require.Equal(t, 422, r.status, r.raw)
	assert.Equal(t, 0, e.count(t, "SELECT COUNT(*) FROM learning_grants"))
	assert.Equal(t, 0, e.count(t, "SELECT COUNT(*) FROM learning_sms_outbox"))
}

// H1 / M2: at most MaxNewGrantsPerDay grants per instructor per 24 h.
func TestGrant_DailyCap(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila")
	cid, _ := e.publishedCourse(t, itok, "Course", 1)
	e.bulkGrants(t, e.instructorID(t, "09120000001"), cid, learning.MaxNewGrantsPerDay-1, "", t0.Add(-time.Hour))

	r := e.grant(t, itok, cid, "09129990001", "09129990002")
	require.Equal(t, 422, r.status, r.raw)
	assert.Contains(t, r.body["errors"].(map[string]any), "phones")
	r = e.grant(t, itok, cid, "09129990001")
	require.Equal(t, 201, r.status, r.raw)

	e.clock.Set(t0.Add(24 * time.Hour))
	r = e.grant(t, itok, cid, "09129990002")
	require.Equal(t, 201, r.status, r.raw)
}

// M2: an instructor whose last-24 h outbox reached SMSPerInstructorPerDay queues no more SMS (the grant still opens).
func TestGrant_SMSEnqueueCap(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila")
	cid, _ := e.publishedCourse(t, itok, "Course", 1)
	insID := e.instructorID(t, "09120000001")
	e.bulkGrants(t, insID, cid, 1, "09150000000", t0.Add(-48*time.Hour))
	var gid uint64
	require.NoError(t, e.db.QueryRow("SELECT id FROM learning_grants WHERE instructor_id = ?", insID).Scan(&gid))
	vals := make([]string, 0, learning.SMSPerInstructorPerDay)
	for range learning.SMSPerInstructorPerDay {
		vals = append(vals, fmt.Sprintf("(%d, 'sent', '2026-10-07 09:00:00', 1, '2026-10-07 09:00:00', '2026-10-07 09:00:00')", gid))
	}
	_, err := e.db.Exec("INSERT INTO learning_sms_outbox (grant_id, status, due_at, attempts, created_at, updated_at) VALUES " + strings.Join(vals, ","))
	require.NoError(t, err)

	r := e.grant(t, itok, cid, "09129990001")
	require.Equal(t, 201, r.status, r.raw)
	newID := idOf(t, r.data()["grants"].([]any)[0].(map[string]any)["id"])
	assert.Equal(t, 0, e.count(t, "SELECT COUNT(*) FROM learning_sms_outbox WHERE grant_id = ?", newID))
}

// M2: the dispatcher picks round-robin over instructors.
func TestListDueSMS_FairPerInstructor(t *testing.T) {
	e := setup(t)
	atok := e.instructor(t, "09120000001", "Ins A")
	btok := e.instructor(t, "09120000002", "Ins B")
	ca, _ := e.publishedCourse(t, atok, "A", 1)
	cb, _ := e.publishedCourse(t, btok, "B", 1)
	r := e.grant(t, atok, ca, "09129990001", "09129990002", "09129990003", "09129990004")
	require.Equal(t, 201, r.status, r.raw)
	r = e.grant(t, btok, cb, "09129990005")
	require.Equal(t, 201, r.status, r.raw)

	ids, err := store.New(e.db).ListDueSMS(context.Background(), store.ListDueSMSParams{DueBefore: t0, LeaseBefore: sqlTime(t0), Limit: 2})
	require.NoError(t, err)
	require.Len(t, ids, 2)
	assert.Equal(t, 2, e.count(t, `SELECT COUNT(DISTINCT a.instructor_id) FROM learning_sms_outbox o JOIN learning_grants a ON a.id = o.grant_id WHERE o.id IN (?, ?)`, ids[0], ids[1]))
}

// M3: revoke → re-add within 7 days sends no new notice / SMS; after the window it does.
func TestGrant_ReaddCooldown(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila")
	cid, _ := e.publishedCourse(t, itok, "Course", 1)
	_, _ = e.user(t, "09121110001")
	r := e.grant(t, itok, cid, "09121110001")
	require.Equal(t, 201, r.status, r.raw)
	gid := idOf(t, r.data()["grants"].([]any)[0].(map[string]any)["id"])
	_, err := e.svc.DispatchSMS(context.Background(), t0)
	require.NoError(t, err)
	e.sms.Sent()

	for i := range 3 { // spam loop: revoke / re-add
		r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/instructor/v1/grants/%d", gid), itok, nil)
		require.Equal(t, 200, r.status, r.raw)
		r = e.grant(t, itok, cid, "09121110001")
		require.Equal(t, 201, r.status, "round %d: %s", i, r.raw)
	}
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM user_notifications WHERE type = 'learning'"))
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM learning_sms_outbox"))
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM learning_grants WHERE status = 'active'"), "the grant itself re-opens")

	r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/instructor/v1/grants/%d", gid), itok, nil)
	require.Equal(t, 200, r.status, r.raw)
	e.clock.Set(t0.Add(learning.ReaddCooldown + time.Hour))
	r = e.grant(t, itok, cid, "09121110001")
	require.Equal(t, 201, r.status, r.raw)
	assert.Equal(t, 2, e.count(t, "SELECT COUNT(*) FROM user_notifications WHERE type = 'learning'"))
	assert.Equal(t, 2, e.count(t, "SELECT COUNT(*) FROM learning_sms_outbox"))
}

// M3 STOP list + L5 revoked instructor: outbox rows end skipped, nothing is sent.
func TestDispatchSMS_SuppressedAndRevokedInstructor(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila")
	cid, _ := e.publishedCourse(t, itok, "Course", 1)
	require.NoError(t, e.svc.SuppressPhone(context.Background(), "+989129990001", "support", t0))
	r := e.grant(t, itok, cid, "09129990001", "09129990002")
	require.Equal(t, 201, r.status, r.raw)
	sent, err := e.svc.DispatchSMS(context.Background(), t0)
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
	assert.Equal(t, []string{"09129990002"}, e.sms.Sent())
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM learning_sms_outbox WHERE status = 'skipped' AND reason = ?", learning.ReasonSuppressed))

	r = e.grant(t, itok, cid, "09129990003")
	require.Equal(t, 201, r.status, r.raw)
	require.NoError(t, e.svc.SetInstructorStatus(context.Background(), e.instructorID(t, "09120000001"), learning.InstructorRevoked, 0, t0))
	sent, err = e.svc.DispatchSMS(context.Background(), t0)
	require.NoError(t, err)
	assert.Equal(t, 0, sent)
	assert.Empty(t, e.sms.Sent())
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM learning_sms_outbox WHERE status = 'skipped' AND reason = ?", learning.ReasonInstructor))
}

// L4: an approved instructor who renames herself goes back to review; a bio edit keeps the approval.
func TestApply_RenameBackToPending(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila")
	r := e.do(t, http.MethodPost, "/api/instructor/v1/apply", itok, map[string]any{"display_name": "Leila", "title": "ماما", "bio": "New bio"})
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "approved", r.data()["instructor"].(map[string]any)["status"])

	r = e.do(t, http.MethodPost, "/api/instructor/v1/apply", itok, map[string]any{"display_name": "Dr. Leila", "title": "ماما"})
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "pending", r.data()["instructor"].(map[string]any)["status"])
	r = e.do(t, http.MethodGet, "/api/instructor/v1/courses", itok, nil)
	assert.Equal(t, 403, r.status)
	assert.Equal(t, learning.ErrorCodeInstructorPending, r.body["error_code"])
}

// Allow-list gate: any status other than approved (here an unknown one) is refused.
func TestRequireInstructor_AllowList(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila")
	_, err := e.db.Exec("UPDATE learning_instructors SET status = 'suspended'")
	require.NoError(t, err)
	r := e.do(t, http.MethodGet, "/api/instructor/v1/courses", itok, nil)
	assert.Equal(t, 403, r.status)
	assert.Equal(t, learning.ErrorCodeInstructorRequired, r.body["error_code"])
	_, err = e.db.Exec("UPDATE learning_instructors SET status = 'revoked'")
	require.NoError(t, err)
	r = e.do(t, http.MethodGet, "/api/instructor/v1/courses", itok, nil)
	assert.Equal(t, learning.ErrorCodeInstructorRequired, r.body["error_code"])
}

// L6: one claim activates at most 100 pending grants; the rest follow on the next call.
func TestClaimPending_Bounded(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila")
	cid, _ := e.publishedCourse(t, itok, "Course", 1)
	e.bulkGrants(t, e.instructorID(t, "09120000001"), cid, 150, "09129990001", t0)
	uid, _ := e.user(t, "09129990001")
	n, err := e.svc.ClaimPending(context.Background(), uid, "09129990001", t0)
	require.NoError(t, err)
	assert.Equal(t, 100, n)
	n, err = e.svc.ClaimPending(context.Background(), uid, "09129990001", t0)
	require.NoError(t, err)
	assert.Equal(t, 50, n)
}

func sqlTime(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }
