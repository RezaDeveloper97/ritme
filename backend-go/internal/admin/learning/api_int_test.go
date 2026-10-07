package learning_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	adminlearning "github.com/ritme/backend-go/internal/admin/learning"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/learning"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const clientID = "0199c0de-0000-7000-8000-00000c0ffee8"

// syncBuf is a goroutine-safe log sink (the audit lines are asserted).
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

type env struct {
	*admintest.Env
	iss  *passport.Issuer
	logs *syncBuf
}

// newEnv mounts the admin module and the instructor API (/api/instructor/v1: me, apply, courses) on one app.
func newEnv(t *testing.T) *env {
	t.Helper()
	e := admintest.New(t)
	logs := &syncBuf{}
	adminlearning.NewHandlers(e.DB, slog.New(slog.NewJSONHandler(logs, nil))).Routes(e.Route(), e.Kit)

	e.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(e.DB)
	iss := passport.NewIssuer(key, q, clock.Real{}, 365)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, admintest.Quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(e.DB), nil, admintest.Quiet))
	h := learning.NewHandlers(learning.NewService(learning.Options{DB: e.DB, Logger: admintest.Quiet}), clock.Real{})
	p := "/api/instructor/v1"
	e.App.Get(p+"/me", locale, guard, h.Me)
	e.App.Post(p+"/apply", locale, guard, h.Apply)
	e.App.Get(p+"/courses", locale, guard, h.RequireInstructor, h.Courses)
	return &env{Env: e, iss: iss, logs: logs}
}

func (e *env) user(mobile, name string) (uint64, string) {
	e.T.Helper()
	id, err := e.Exec("INSERT INTO users (mobile, name, created_at, updated_at) VALUES (?, ?, NOW(), NOW())", mobile, name).LastInsertId()
	require.NoError(e.T, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // test ids
	require.NoError(e.T, err)
	return uint64(id), tok.AccessToken //nolint:gosec // test ids
}

// userAPI calls the instructor API as a user (status + error_code).
func (e *env) userAPI(method, path, token string, body any) (int, map[string]any) {
	e.T.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequestWithContext(e.T.Context(), method, path, rd)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := e.App.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(e.T, err)
	defer func() { _ = res.Body.Close() }()
	raw, _ := io.ReadAll(res.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return res.StatusCode, m
}

// instructor inserts an instructor row directly.
func (e *env) instructor(userID uint64, name, status, created string) string {
	e.T.Helper()
	id, err := e.Exec(`INSERT INTO learning_instructors (user_id, display_name, title, status, created_at, updated_at)
		VALUES (?, ?, 'ماما', ?, ?, ?)`, userID, name, status, created, created).LastInsertId()
	require.NoError(e.T, err)
	return strconv.FormatInt(id, 10)
}

func (e *env) course(instructorID, title, status string) string {
	e.T.Helper()
	id, err := e.Exec(`INSERT INTO learning_courses (instructor_id, kind, title, status, published_at, created_at, updated_at)
		VALUES (?, 'course', ?, ?, '2026-10-01 10:00:00', '2026-10-01 10:00:00', '2026-10-01 10:00:00')`, instructorID, title, status).LastInsertId()
	require.NoError(e.T, err)
	return strconv.FormatInt(id, 10)
}

func (e *env) lesson(courseID, title, status, updated string) string {
	e.T.Helper()
	id, err := e.Exec(`INSERT INTO learning_lessons (course_id, kind, title, status, duration_seconds, published_at, created_at, updated_at)
		VALUES (?, 'video', ?, ?, 600, '2026-10-01 10:00:00', ?, ?)`, courseID, title, status, updated, updated).LastInsertId()
	require.NoError(e.T, err)
	return strconv.FormatInt(id, 10)
}

func (e *env) grant(instructorID, courseID string, userID uint64, phone string) {
	e.T.Helper()
	e.Exec(`INSERT INTO learning_grants (instructor_id, phone, user_id, course_id, duration, status, activated_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'unlimited', 'active', '2026-10-02 10:00:00', '2026-10-02 10:00:00', '2026-10-02 10:00:00')`,
		instructorID, phone, userID, courseID)
}

func (e *env) progress(userID uint64, lessonID, courseID string, percent int, completed bool) {
	e.T.Helper()
	var done any
	if completed {
		done = "2026-10-03 10:00:00"
	}
	e.Exec(`INSERT INTO learning_progress (user_id, lesson_id, course_id, position_seconds, percent, completed_at, last_seen_at, created_at, updated_at)
		VALUES (?, ?, ?, 0, ?, ?, NOW(), NOW(), NOW())`, userID, lessonID, courseID, percent, done)
}

func num(v any) int {
	f, _ := v.(float64)
	return int(f)
}

func TestGuards(t *testing.T) {
	e := newEnv(t)
	u, _ := e.user("09120000001", "Sara")
	ins := e.instructor(u, "Sara midwife", "pending", "2026-10-01 10:00:00")
	c := e.course(ins, "Prenatal yoga", "published")
	l := e.lesson(c, "Breathing", "published", "2026-10-01 10:00:00")

	for _, path := range []string{"/learning/stats", "/learning/instructors", "/learning/instructors/" + ins, "/learning/courses",
		"/learning/courses/" + c, "/learning/reviews"} {
		r := e.Anonymous().Get(path)
		assert.Equal(t, 401, r.Status, path)
		assert.Equal(t, "unauthenticated", r.Code(), path)
		assert.Equal(t, 200, e.As(admintest.EditorID).Get(path).Status, "editors read: "+path)
	}
	assert.Equal(t, 401, e.Anonymous().JSON(fiber.MethodPost, "/learning/instructors/"+ins+"/approve", nil).Status)

	ed := e.As(admintest.EditorID)
	for _, a := range []string{"approve", "revoke"} {
		r := ed.JSON(fiber.MethodPost, "/learning/instructors/"+ins+"/"+a, nil)
		assert.Equal(t, 403, r.Status, a)
		assert.Equal(t, "forbidden", r.Code(), a)
	}
	assert.Equal(t, "pending", e.String("SELECT status FROM learning_instructors WHERE id = ?", ins))

	noToken := e.As(admintest.SuperID)
	noToken.CSRF = ""
	assert.Equal(t, 419, noToken.JSON(fiber.MethodPost, "/learning/instructors/"+ins+"/approve", nil).Status)
	assert.Equal(t, 419, noToken.JSON(fiber.MethodPost, "/learning/lessons/"+l+"/approve", nil).Status)
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM learning_moderation_log"))

	// Editors moderate content.
	assert.Equal(t, 200, ed.JSON(fiber.MethodPost, "/learning/lessons/"+l+"/approve", nil).Status)

	for _, path := range []string{"/learning/instructors/999999", "/learning/instructors/abc", "/learning/courses/999999"} {
		assert.Equal(t, 404, ed.Get(path).Status, path)
	}
	assert.Equal(t, 404, e.As(admintest.SuperID).JSON(fiber.MethodPost, "/learning/instructors/999999/approve", nil).Status)
	assert.Equal(t, 404, ed.JSON(fiber.MethodPost, "/learning/lessons/999999/flag", map[string]any{"note": "bad claims"}).Status)
}

func TestApproveRevokeOpensAndClosesTheInstructorAPI(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user("09121234567", "Mina")
	status, body := e.userAPI(http.MethodPost, "/api/instructor/v1/apply", tok, map[string]any{"display_name": "Mina Ahmadi", "title": "ماما"})
	require.Equal(t, 201, status, body)
	status, body = e.userAPI(http.MethodGet, "/api/instructor/v1/courses", tok, nil)
	require.Equal(t, 403, status)
	assert.Equal(t, "instructor_pending", body["error_code"])

	id := strconv.Itoa(e.Int("SELECT id FROM learning_instructors WHERE user_id = ?", uid))
	super := e.As(admintest.SuperID)

	list := super.Get("/learning/instructors?status=pending")
	require.Equal(t, 200, list.Status, string(list.Raw))
	require.Len(t, list.Items(), 1)
	item := list.Items()[0].(map[string]any)
	assert.Equal(t, "Mina Ahmadi", item["display_name"])
	assert.Equal(t, "0912•••4567", item["user"].(map[string]any)["mobile"], "the instructor's mobile is masked")
	assert.NotContains(t, string(list.Raw), "09121234567")

	r := super.JSON(fiber.MethodPost, "/learning/instructors/"+id+"/approve", map[string]any{"note": "Licence checked"})
	require.Equal(t, 200, r.Status, string(r.Raw))
	assert.Equal(t, true, r.Data()["changed"])
	ins := r.Obj("instructor")
	assert.Equal(t, "approved", ins["status"])
	assert.Equal(t, map[string]any{"id": float64(1), "name": "Root"}, ins["approved_by"])
	assert.NotNil(t, ins["approved_at"])

	// The audit row and the audit line.
	assert.Equal(t, 1, e.Int(`SELECT COUNT(*) FROM learning_moderation_log WHERE action = 'instructor.approve'
		AND target_type = 'instructor' AND target_id = ? AND admin_id = 1 AND instructor_id = ? AND note = 'Licence checked'`, id, id))
	assert.Contains(t, e.logs.String(), `"audit":"learning.instructor.approve"`)
	assert.Contains(t, e.logs.String(), `"admin_id":1`)

	// Approved → the instructor API opens.
	status, body = e.userAPI(http.MethodGet, "/api/instructor/v1/courses", tok, nil)
	assert.Equal(t, 200, status, body)

	// Approving again changes nothing and logs nothing.
	r = super.JSON(fiber.MethodPost, "/learning/instructors/"+id+"/approve", nil)
	require.Equal(t, 200, r.Status)
	assert.Equal(t, false, r.Data()["changed"])
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM learning_moderation_log"))

	// Note too long → 422.
	r = super.JSON(fiber.MethodPost, "/learning/instructors/"+id+"/revoke", map[string]any{"note": strings.Repeat("x", 301)})
	assert.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "note")

	r = super.JSON(fiber.MethodPost, "/learning/instructors/"+id+"/revoke", nil)
	require.Equal(t, 200, r.Status, string(r.Raw))
	assert.Equal(t, "revoked", r.Obj("instructor")["status"])
	assert.NotNil(t, r.Obj("instructor")["revoked_at"])
	status, body = e.userAPI(http.MethodGet, "/api/instructor/v1/courses", tok, nil)
	assert.Equal(t, 403, status)
	assert.Equal(t, "instructor_required", body["error_code"])

	detail := super.Get("/learning/instructors/" + id)
	require.Equal(t, 200, detail.Status)
	hist := detail.Data()["history"].([]any)
	require.Len(t, hist, 2)
	assert.Equal(t, "instructor.revoke", hist[0].(map[string]any)["action"], "newest first")
	assert.Equal(t, "Licence checked", hist[1].(map[string]any)["note"])
}

func TestInstructorListOrderAndCounts(t *testing.T) {
	e := newEnv(t)
	u1, _ := e.user("09120000001", "A")
	u2, _ := e.user("09120000002", "B")
	u3, _ := e.user("09120000003", "C")
	old := e.instructor(u1, "Approved older", "approved", "2026-09-01 10:00:00")
	pend := e.instructor(u2, "Pending oldest", "pending", "2026-08-01 10:00:00")
	e.instructor(u3, "Revoked newest", "revoked", "2026-10-01 10:00:00")

	r := e.As(admintest.EditorID).Get("/learning/instructors")
	require.Equal(t, 200, r.Status)
	items := r.Items()
	require.Len(t, items, 3)
	assert.EqualValues(t, mustInt(pend), items[0].(map[string]any)["id"], "pending first")
	assert.Equal(t, map[string]any{"all": float64(3), "pending": float64(1), "approved": float64(1), "revoked": float64(1)}, r.Data()["counts"])

	r = e.As(admintest.EditorID).Get("/learning/instructors?status=approved&q=older")
	require.Len(t, r.Items(), 1)
	assert.EqualValues(t, mustInt(old), r.Items()[0].(map[string]any)["id"])
	r = e.As(admintest.EditorID).Get("/learning/instructors?status=bogus&q=" + url.QueryEscape("۰۰۰۰۰۳"))
	require.Len(t, r.Items(), 1, "Persian digits search the mobile")
	assert.Equal(t, "all", r.Data()["filters"].(map[string]any)["status"])
}

func mustInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func TestCoursesUsageAndReviewQueue(t *testing.T) {
	e := newEnv(t)
	iu, _ := e.user("09120000009", "Ins")
	ins := e.instructor(iu, "Dr. Rahimi", "approved", "2026-09-01 10:00:00")
	c := e.course(ins, "Birth prep", "published")
	draft := e.course(ins, "Draft course", "draft")
	l1 := e.lesson(c, "Lesson one", "published", "2026-10-01 10:00:00")
	l2 := e.lesson(c, "Lesson two", "published", "2026-10-02 10:00:00")
	e.lesson(c, "Unpublished", "draft", "2026-10-02 10:00:00")

	s1, _ := e.user("09130000001", "S1")
	s2, _ := e.user("09130000002", "S2")
	e.grant(ins, c, s1, "09130000001")
	e.grant(ins, c, s2, "09130000002")
	e.progress(s1, l1, c, 0, true)
	e.progress(s1, l2, c, 100, true)
	e.progress(s2, l1, c, 50, false)

	ed := e.As(admintest.EditorID)
	r := ed.Get("/learning/courses")
	require.Equal(t, 200, r.Status, string(r.Raw))
	require.Len(t, r.Items(), 2)
	byID := map[int]map[string]any{}
	for _, it := range r.Items() {
		m := it.(map[string]any)
		byID[num(m["id"])] = m
	}
	bc := byID[mustInt(c)]
	assert.EqualValues(t, 3, bc["lessons_count"])
	assert.EqualValues(t, 2, bc["published_lessons"])
	assert.EqualValues(t, 2, bc["review_open"], "published, never reviewed")
	assert.EqualValues(t, 2, bc["students_count"])
	assert.EqualValues(t, 63, bc["completion_percent"], "(100 + 25) / 2 rounded")
	assert.EqualValues(t, 1, bc["completed_count"])
	assert.Equal(t, map[string]any{"id": float64(mustInt(ins)), "display_name": "Dr. Rahimi", "status": "approved"}, bc["instructor"])
	assert.EqualValues(t, 0, byID[mustInt(draft)]["students_count"])
	assert.Equal(t, map[string]any{"all": float64(2), "draft": float64(1), "published": float64(1)}, r.Data()["counts"])
	assert.NotContains(t, string(r.Raw), "0913", "no student phone numbers")

	r = ed.Get("/learning/courses?status=draft&instructor_id=" + ins)
	require.Len(t, r.Items(), 1)
	assert.EqualValues(t, mustInt(draft), r.Items()[0].(map[string]any)["id"])

	d := ed.Get("/learning/courses/" + c)
	require.Equal(t, 200, d.Status, string(d.Raw))
	course := d.Obj("course")
	assert.Len(t, course["lessons"], 3)
	assert.Equal(t, map[string]any{"all": float64(2), "active": float64(2), "pending": float64(0), "expired": float64(0), "revoked": float64(0)}, course["grants"])
	assert.Equal(t, map[string]any{"students_count": float64(2), "completion_percent": float64(63), "completed_count": float64(1)}, course["usage"])
	assert.NotContains(t, string(d.Raw), "0913")

	// Review queue: the two published lessons, oldest change first.
	q := ed.Get("/learning/reviews")
	require.Equal(t, 200, q.Status, string(q.Raw))
	require.Len(t, q.Items(), 2)
	first := q.Items()[0].(map[string]any)
	assert.EqualValues(t, mustInt(l1), first["id"])
	assert.Equal(t, "pending", first["review"].(map[string]any)["state"])
	assert.Equal(t, "Birth prep", first["course"].(map[string]any)["title"])
	assert.Equal(t, map[string]any{"open": float64(2), "pending": float64(2), "changed": float64(0), "flagged": float64(0), "approved": float64(0)}, q.Data()["counts"])

	// Approve l1 → leaves the open queue.
	a := ed.JSON(fiber.MethodPost, "/learning/lessons/"+l1+"/approve", nil)
	require.Equal(t, 200, a.Status, string(a.Raw))
	rev := a.Obj("lesson")["review"].(map[string]any)
	assert.Equal(t, "approved", rev["state"])
	assert.Equal(t, map[string]any{"id": float64(2), "name": "Editor"}, rev["reviewed_by"])
	assert.Len(t, ed.Get("/learning/reviews").Items(), 1)
	assert.Len(t, ed.Get("/learning/reviews?state=approved").Items(), 1)

	// The instructor edits it later → «changed» and back in the queue.
	e.Exec("UPDATE learning_lessons SET updated_at = DATE_ADD(reviewed_at, INTERVAL 1 HOUR) WHERE id = ?", l1)
	q = ed.Get("/learning/reviews?state=changed")
	require.Len(t, q.Items(), 1)
	assert.EqualValues(t, mustInt(l1), q.Items()[0].(map[string]any)["id"])
	assert.Len(t, ed.Get("/learning/reviews").Items(), 2)

	// Flag needs a note.
	f := ed.JSON(fiber.MethodPost, "/learning/lessons/"+l2+"/flag", map[string]any{})
	assert.Equal(t, 422, f.Status)
	assert.Contains(t, f.Errors(), "note")
	f = ed.JSON(fiber.MethodPost, "/learning/lessons/"+l2+"/flag", map[string]any{"note": "Unsafe dosage claim"})
	require.Equal(t, 200, f.Status, string(f.Raw))
	assert.Equal(t, "flagged", f.Obj("lesson")["review"].(map[string]any)["state"])
	assert.Equal(t, "Unsafe dosage claim", f.Obj("lesson")["review"].(map[string]any)["note"])
	assert.Equal(t, "published", f.Obj("lesson")["status"], "a flag alone does not take the lesson down")

	// Unpublish → draft, still listed as flagged.
	u := ed.JSON(fiber.MethodPost, "/learning/lessons/"+l2+"/unpublish", map[string]any{"note": "Taken down until fixed"})
	require.Equal(t, 200, u.Status, string(u.Raw))
	assert.Equal(t, "draft", e.String("SELECT status FROM learning_lessons WHERE id = ?", l2))
	q = ed.Get("/learning/reviews?state=flagged")
	require.Len(t, q.Items(), 1)
	assert.Equal(t, "draft", q.Items()[0].(map[string]any)["status"])

	assert.Equal(t, 3, e.Int("SELECT COUNT(*) FROM learning_moderation_log WHERE target_type = 'lesson' AND instructor_id = ? AND admin_id = 2", ins))
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM learning_moderation_log WHERE action = 'lesson.unpublish' AND note = 'Taken down until fixed'"))
	assert.Contains(t, e.logs.String(), `"audit":"learning.lesson.flag"`)
	assert.NotContains(t, e.logs.String(), "Unsafe dosage", "notes stay out of the logs")

	// Stats.
	s := ed.Get("/learning/stats")
	require.Equal(t, 200, s.Status, string(s.Raw))
	assert.Equal(t, map[string]any{"all": float64(1), "pending": float64(0), "approved": float64(1), "revoked": float64(0)}, s.Data()["instructors"])
	assert.EqualValues(t, 2, s.Obj("students")["with_access"])
	assert.EqualValues(t, 2, s.Obj("students")["active_30d"])
	assert.EqualValues(t, 2, s.Obj("grants")["active"])
	assert.EqualValues(t, 2, s.Obj("completion")["lessons_completed"])
	assert.EqualValues(t, 2, s.Obj("completion")["enrolments"])
	assert.EqualValues(t, 1, s.Obj("review")["flagged"])
	assert.EqualValues(t, 1, s.Obj("review")["changed"])
}
