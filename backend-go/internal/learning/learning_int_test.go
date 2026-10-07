package learning_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/learning"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const clientID = "0199c0de-0000-7000-8000-00000c0ffee1"

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

// mutClock is the request clock of the learning handlers (tests move it).
type mutClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *mutClock) Now() time.Time  { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *mutClock) Set(t time.Time) { c.mu.Lock(); defer c.mu.Unlock(); c.t = t }

type nopJobs struct{}

func (nopJobs) Dispatch(context.Context, string, any) error { return nil }

type env struct {
	db    *sql.DB
	app   *fiber.App
	iss   *passport.Issuer
	svc   *learning.Service
	sms   *learning.FakeSMS
	clock *mutClock
}

// t0: Wednesday 2026-10-07 10:00 Tehran (outside the default quiet hours).
var t0 = time.Date(2026, 10, 7, 10, 0, 0, 0, civildate.Tehran)

func setup(t *testing.T) *env {
	t.Helper()
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	iss := passport.NewIssuer(key, q, clock.Real{}, 365)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	langs := i18n.NewRegistry(i18nstore.New(db), nil, quiet)
	locale := i18n.Middleware(langs)
	fake := learning.NewFakeSMS(quiet)
	clk := &mutClock{t: t0}
	svc := learning.NewService(learning.Options{
		DB: db, SMS: fake, Logger: quiet,
		Languages: func(ctx context.Context) []string { return langs.All(ctx).Codes() },
	})
	h := learning.NewHandlers(svc, clk)
	ah := auth.NewHandlers(q, iss, nopJobs{}, clk, 30, quiet)
	ah.OnSignup(svc.SignupHook())

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Post("/api/v1/auth/send-otp", locale, ah.SendOTP)
	app.Post("/api/v1/auth/verify-otp", locale, ah.VerifyOTP)
	s := "/api/v1/learning"
	app.Get(s+"/courses", locale, guard, h.MyCourses)
	app.Get(s+"/courses/:id", locale, guard, h.StudentCourse)
	app.Get(s+"/lessons/:id", locale, guard, h.ShowLesson)
	app.Put(s+"/lessons/:id/progress", locale, guard, h.SaveProgress)
	app.Get(s+"/unlocked/:grant", locale, guard, h.Unlocked)
	p := "/api/instructor/v1"
	ins := h.RequireInstructor
	app.Get(p+"/me", locale, guard, h.Me)
	app.Post(p+"/apply", locale, guard, h.Apply)
	app.Get(p+"/courses", locale, guard, ins, h.Courses)
	app.Post(p+"/courses", locale, guard, ins, h.StoreCourse)
	app.Get(p+"/courses/:id", locale, guard, ins, h.ShowCourse)
	app.Put(p+"/courses/:id", locale, guard, ins, h.UpdateCourse)
	app.Delete(p+"/courses/:id", locale, guard, ins, h.DestroyCourse)
	app.Post(p+"/courses/:id/chapters", locale, guard, ins, h.StoreChapter)
	app.Put(p+"/courses/:id/chapters/:chapter", locale, guard, ins, h.UpdateChapter)
	app.Delete(p+"/courses/:id/chapters/:chapter", locale, guard, ins, h.DestroyChapter)
	app.Post(p+"/courses/:id/lessons", locale, guard, ins, h.StoreLesson)
	app.Put(p+"/courses/:id/lessons/:lesson", locale, guard, ins, h.UpdateLesson)
	app.Delete(p+"/courses/:id/lessons/:lesson", locale, guard, ins, h.DestroyLesson)
	app.Get(p+"/groups", locale, guard, ins, h.Groups)
	app.Post(p+"/groups", locale, guard, ins, h.StoreGroup)
	app.Get(p+"/groups/:id", locale, guard, ins, h.ShowGroup)
	app.Put(p+"/groups/:id", locale, guard, ins, h.UpdateGroup)
	app.Delete(p+"/groups/:id", locale, guard, ins, h.DestroyGroup)
	app.Get(p+"/students", locale, guard, ins, h.Students)
	app.Post(p+"/grants", locale, guard, ins, h.StoreGrants)
	app.Delete(p+"/grants/:id", locale, guard, ins, h.DestroyGrant)
	return &env{db: db, app: app, iss: iss, svc: svc, sms: fake, clock: clk}
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES (?, ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, "User "+mobile[7:], mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now())
	require.NoError(t, err)
	return uint64(id), tok.AccessToken
}

// instructor creates an approved instructor account.
func (e *env) instructor(t *testing.T, mobile, name string) string {
	t.Helper()
	id, tok := e.user(t, mobile)
	r := e.do(t, http.MethodPost, "/api/instructor/v1/apply", tok, map[string]any{"display_name": name, "title": "ماما"})
	require.Equal(t, 201, r.status, r.raw)
	ins, ok, err := e.svc.InstructorByUser(context.Background(), id)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, e.svc.SetInstructorStatus(context.Background(), ins.ID, learning.InstructorApproved, 1, t0))
	return tok
}

type response struct {
	status int
	body   map[string]any
	raw    string
}

func (r response) data() map[string]any {
	d, _ := r.body["data"].(map[string]any)
	return d
}

func (e *env) do(t *testing.T, method, path, token string, body any) response {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rd = strings.NewReader(string(b))
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return response{status: resp.StatusCode, body: m, raw: string(raw)}
}

func idOf(t *testing.T, v any) int {
	t.Helper()
	f, ok := v.(float64)
	require.True(t, ok, "id %v", v)
	return int(f)
}

// publishedCourse creates a published course with one chapter and n published video lessons (600 s each).
func (e *env) publishedCourse(t *testing.T, tok, title string, n int) (int, []int) {
	t.Helper()
	r := e.do(t, http.MethodPost, "/api/instructor/v1/courses", tok, map[string]any{"kind": "course", "title": title, "status": "published"})
	require.Equal(t, 201, r.status, r.raw)
	cid := idOf(t, r.data()["course"].(map[string]any)["id"])
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/instructor/v1/courses/%d/chapters", cid), tok, map[string]any{"title": "Chapter 1"})
	require.Equal(t, 201, r.status, r.raw)
	ch := idOf(t, r.data()["chapter"].(map[string]any)["id"])
	var lessons []int
	for i := range n {
		r = e.do(t, http.MethodPost, fmt.Sprintf("/api/instructor/v1/courses/%d/lessons", cid), tok, map[string]any{
			"kind": "video", "title": fmt.Sprintf("Lesson %d", i+1), "chapter_id": ch, "duration_seconds": 600, "status": "published",
		})
		require.Equal(t, 201, r.status, r.raw)
		lessons = append(lessons, idOf(t, r.data()["lesson"].(map[string]any)["id"]))
	}
	return cid, lessons
}

func (e *env) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(query, args...).Scan(&n))
	return n
}

// Acceptance: a grant for a number without an account is pending and becomes active when that number signs up.
func TestPendingGrantActivatesOnSignup(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila M")
	cid, lessons := e.publishedCourse(t, itok, "Natural birth prep", 2)

	r := e.do(t, http.MethodPost, "/api/instructor/v1/grants", itok, map[string]any{
		"phones": []string{"0912 999 0099"}, "scope": "course", "target_id": cid, "duration": "30",
	})
	require.Equal(t, 201, r.status, r.raw)
	g := r.data()["grants"].([]any)[0].(map[string]any)
	assert.Equal(t, "pending", g["status"])
	assert.Equal(t, "09129990099", g["phone"])
	assert.Equal(t, false, g["registered"])
	assert.Nil(t, g["expires_at"])
	grantID := idOf(t, g["id"])
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM learning_sms_outbox WHERE grant_id = ? AND status = 'pending'", grantID))

	// The invite SMS goes out through the fake provider.
	sent, err := e.svc.DispatchSMS(context.Background(), t0)
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
	assert.Equal(t, []string{"09129990099"}, e.sms.Sent())

	// Signup with the OTP: the hook activates the grant, 30 days from now.
	e.clock.Set(t0.Add(48 * time.Hour))
	r = e.do(t, http.MethodPost, "/api/v1/auth/send-otp", "", map[string]any{"mobile": "09129990099"})
	require.Equal(t, 200, r.status, r.raw)
	var code string
	require.NoError(t, e.db.QueryRow("SELECT code FROM otp_verifications WHERE mobile = '09129990099' ORDER BY id DESC LIMIT 1").Scan(&code))
	r = e.do(t, http.MethodPost, "/api/v1/auth/verify-otp", "", map[string]any{"mobile": "09129990099", "code": code})
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, true, r.data()["new_user"])
	tok := r.data()["access_token"].(string)

	var status string
	var expires sql.NullTime
	require.NoError(t, e.db.QueryRow("SELECT status, expires_at FROM learning_grants WHERE id = ?", grantID).Scan(&status, &expires))
	assert.Equal(t, "active", status)
	assert.True(t, expires.Time.Equal(t0.Add(48*time.Hour).AddDate(0, 0, 30)), expires.Time)
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM user_notifications WHERE type = 'learning' AND action_url = ?", fmt.Sprintf("/learn/unlocked/%d", grantID)))

	r = e.do(t, http.MethodGet, "/api/v1/learning/courses", tok, nil)
	require.Equal(t, 200, r.status, r.raw)
	courses := r.data()["courses"].([]any)
	require.Len(t, courses, 1)
	c := courses[0].(map[string]any)
	assert.Equal(t, "Natural birth prep", c["title"])
	assert.EqualValues(t, 30, c["access"].(map[string]any)["days_left"])
	assert.EqualValues(t, 2, c["lessons_count"])
	assert.Nil(t, r.data()["continue"])

	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/learning/unlocked/%d", grantID), tok, nil)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "Leila M", r.data()["instructor"].(map[string]any)["name"])

	// The instructor now sees the student by name.
	r = e.do(t, http.MethodGet, "/api/instructor/v1/students?status=active", itok, nil)
	require.Equal(t, 200, r.status, r.raw)
	require.Len(t, r.data()["students"].([]any), 1)
	assert.EqualValues(t, 1, r.data()["counts"].(map[string]any)["active"])

	// Progress: position + derived percent, never down, completion sticks.
	path := fmt.Sprintf("/api/v1/learning/lessons/%d/progress", lessons[0])
	r = e.do(t, http.MethodPut, path, tok, map[string]any{"position_seconds": 300})
	require.Equal(t, 200, r.status, r.raw)
	assert.EqualValues(t, 50, r.data()["progress"].(map[string]any)["percent"])
	assert.EqualValues(t, 25, r.data()["course_percent"])
	r = e.do(t, http.MethodPut, path, tok, map[string]any{"position_seconds": 60})
	require.Equal(t, 200, r.status, r.raw)
	assert.EqualValues(t, 50, r.data()["progress"].(map[string]any)["percent"])
	assert.EqualValues(t, 60, r.data()["progress"].(map[string]any)["position_seconds"])
	r = e.do(t, http.MethodGet, "/api/v1/learning/courses", tok, nil)
	k := r.data()["continue"].(map[string]any)
	assert.EqualValues(t, lessons[0], k["lesson"].(map[string]any)["id"])
	assert.EqualValues(t, 540, k["remaining_seconds"])
	r = e.do(t, http.MethodPut, path, tok, map[string]any{"position_seconds": 590})
	assert.Equal(t, true, r.data()["progress"].(map[string]any)["completed"])
}

func TestSignupClaim_LazyAndOnlyOnce(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila")
	cid, _ := e.publishedCourse(t, itok, "Course", 1)
	r := e.do(t, http.MethodPost, "/api/instructor/v1/grants", itok, map[string]any{
		"phones": []string{"09129990098"}, "scope": "course", "target_id": cid, "duration": "unlimited",
	})
	require.Equal(t, 201, r.status, r.raw)
	// An account created by another path (no hook): the student endpoint claims it.
	_, tok := e.user(t, "09129990098")
	r = e.do(t, http.MethodGet, "/api/v1/learning/courses", tok, nil)
	require.Equal(t, 200, r.status, r.raw)
	require.Len(t, r.data()["courses"].([]any), 1)
	assert.Equal(t, true, r.data()["courses"].([]any)[0].(map[string]any)["access"].(map[string]any)["unlimited"])
	r = e.do(t, http.MethodGet, "/api/v1/learning/courses", tok, nil)
	require.Equal(t, 200, r.status)
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM user_notifications WHERE type = 'learning'"))
}

func TestRegisteredStudent_ActiveAtOnceAndRenewal(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila")
	cid, _ := e.publishedCourse(t, itok, "Course", 1)
	_, stok := e.user(t, "09121110001")
	grant := map[string]any{"phones": []string{"09121110001"}, "scope": "course", "target_id": cid, "duration": "until", "until": "2026-10-09"}
	r := e.do(t, http.MethodPost, "/api/instructor/v1/grants", itok, grant)
	require.Equal(t, 201, r.status, r.raw)
	g := r.data()["grants"].([]any)[0].(map[string]any)
	assert.Equal(t, "active", g["status"])
	assert.EqualValues(t, 1, r.data()["summary"].(map[string]any)["active"])
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM user_notifications WHERE type = 'learning'"))

	r = e.do(t, http.MethodPost, "/api/instructor/v1/grants", itok, grant) // same target again: renewal, no new row
	require.Equal(t, 201, r.status, r.raw)
	assert.Equal(t, true, r.data()["grants"].([]any)[0].(map[string]any)["renewed"])
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM learning_grants"))
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM user_notifications WHERE type = 'learning'"))

	// Expired: the course stays listed (flagged), lessons answer 403.
	e.clock.Set(time.Date(2026, 10, 10, 9, 0, 0, 0, civildate.Tehran))
	r = e.do(t, http.MethodGet, "/api/v1/learning/courses", stok, nil)
	require.Equal(t, 200, r.status)
	acc := r.data()["courses"].([]any)[0].(map[string]any)["access"].(map[string]any)
	assert.Equal(t, true, acc["expired"])
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/learning/courses/%d", cid), stok, nil)
	require.Equal(t, 200, r.status)
	lid := idOf(t, r.data()["chapters"].([]any)[0].(map[string]any)["lessons"].([]any)[0].(map[string]any)["id"])
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/learning/lessons/%d", lid), stok, nil)
	assert.Equal(t, 403, r.status)
	assert.Equal(t, learning.ErrorCodeAccessExpired, r.body["error_code"])

	// Revoked: gone from the list.
	gid := idOf(t, g["id"])
	r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/instructor/v1/grants/%d", gid), itok, nil)
	require.Equal(t, 200, r.status, r.raw)
	r = e.do(t, http.MethodGet, "/api/v1/learning/courses", stok, nil)
	assert.Empty(t, r.data()["courses"])
}

func TestGroupGrant_LockedChapterAndDrafts(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila")
	c1, _ := e.publishedCourse(t, itok, "One", 1)
	c2, _ := e.publishedCourse(t, itok, "Two", 1)
	r := e.do(t, http.MethodPost, "/api/instructor/v1/groups", itok, map[string]any{"name": "Mehr 1405", "course_ids": []int{c1}})
	require.Equal(t, 201, r.status, r.raw)
	gid := idOf(t, r.data()["group"].(map[string]any)["id"])
	_, stok := e.user(t, "09121110002")
	r = e.do(t, http.MethodPost, "/api/instructor/v1/grants", itok, map[string]any{
		"phones": []string{"09121110002", "09129990097"}, "scope": "group", "target_id": gid, "duration": "90",
	})
	require.Equal(t, 201, r.status, r.raw)
	r = e.do(t, http.MethodGet, "/api/v1/learning/courses", stok, nil)
	require.Len(t, r.data()["courses"].([]any), 1)

	// A course added to the group later is open to its members at once.
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/instructor/v1/groups/%d", gid), itok, map[string]any{"course_ids": []int{c1, c2}})
	require.Equal(t, 200, r.status, r.raw)
	assert.EqualValues(t, 1, r.data()["pending_signups"])
	require.Len(t, r.data()["members"].([]any), 2)
	r = e.do(t, http.MethodGet, "/api/v1/learning/courses", stok, nil)
	require.Len(t, r.data()["courses"].([]any), 2)
	assert.Equal(t, "Mehr 1405", r.data()["courses"].([]any)[0].(map[string]any)["access"].(map[string]any)["group"].(map[string]any)["name"])

	// Locked chapter: listed with its unlock time, the lesson answers 403.
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/instructor/v1/courses/%d/chapters", c2), itok, map[string]any{"title": "Later", "unlock_at": "2026-10-15 08:00"})
	require.Equal(t, 201, r.status, r.raw)
	ch := idOf(t, r.data()["chapter"].(map[string]any)["id"])
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/instructor/v1/courses/%d/lessons", c2), itok, map[string]any{"kind": "audio", "title": "Locked", "chapter_id": ch, "status": "published"})
	require.Equal(t, 201, r.status, r.raw)
	locked := idOf(t, r.data()["lesson"].(map[string]any)["id"])
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/instructor/v1/courses/%d/lessons", c2), itok, map[string]any{"kind": "pdf", "title": "Draft"})
	require.Equal(t, 201, r.status, r.raw)
	draft := idOf(t, r.data()["lesson"].(map[string]any)["id"])

	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/learning/courses/%d", c2), stok, nil)
	require.Equal(t, 200, r.status, r.raw)
	chs := r.data()["chapters"].([]any)
	require.Len(t, chs, 2)
	assert.Equal(t, true, chs[1].(map[string]any)["locked"])
	assert.EqualValues(t, 2, r.data()["course"].(map[string]any)["lessons_count"]) // the draft is hidden
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/learning/lessons/%d", locked), stok, nil)
	assert.Equal(t, 403, r.status)
	assert.Equal(t, learning.ErrorCodeChapterLocked, r.body["error_code"])
	assert.Equal(t, "2026-10-15T08:00:00+03:30", r.body["unlock_at"])
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/learning/lessons/%d", draft), stok, nil)
	assert.Equal(t, 404, r.status)

	e.clock.Set(time.Date(2026, 10, 15, 8, 0, 0, 0, civildate.Tehran))
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/learning/lessons/%d", locked), stok, nil)
	assert.Equal(t, 200, r.status, r.raw)
}

func TestInstructorIDOR(t *testing.T) {
	e := setup(t)
	atok := e.instructor(t, "09120000001", "Instructor A")
	btok := e.instructor(t, "09120000002", "Instructor B")
	cid, lessons := e.publishedCourse(t, atok, "A's course", 1)
	r := e.do(t, http.MethodGet, fmt.Sprintf("/api/instructor/v1/courses/%d", cid), atok, nil)
	ch := idOf(t, r.data()["chapters"].([]any)[0].(map[string]any)["id"])
	r = e.do(t, http.MethodPost, "/api/instructor/v1/groups", atok, map[string]any{"name": "A group", "course_ids": []int{cid}})
	gid := idOf(t, r.data()["group"].(map[string]any)["id"])
	r = e.do(t, http.MethodPost, "/api/instructor/v1/grants", atok, map[string]any{"phones": []string{"09129990096"}, "scope": "course", "target_id": cid, "duration": "unlimited"})
	grant := idOf(t, r.data()["grants"].([]any)[0].(map[string]any)["id"])

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, fmt.Sprintf("/api/instructor/v1/courses/%d", cid)},
		{http.MethodPut, fmt.Sprintf("/api/instructor/v1/courses/%d", cid)},
		{http.MethodDelete, fmt.Sprintf("/api/instructor/v1/courses/%d", cid)},
		{http.MethodPost, fmt.Sprintf("/api/instructor/v1/courses/%d/chapters", cid)},
		{http.MethodPut, fmt.Sprintf("/api/instructor/v1/courses/%d/chapters/%d", cid, ch)},
		{http.MethodDelete, fmt.Sprintf("/api/instructor/v1/courses/%d/chapters/%d", cid, ch)},
		{http.MethodPut, fmt.Sprintf("/api/instructor/v1/courses/%d/lessons/%d", cid, lessons[0])},
		{http.MethodDelete, fmt.Sprintf("/api/instructor/v1/courses/%d/lessons/%d", cid, lessons[0])},
		{http.MethodGet, fmt.Sprintf("/api/instructor/v1/groups/%d", gid)},
		{http.MethodPut, fmt.Sprintf("/api/instructor/v1/groups/%d", gid)},
		{http.MethodDelete, fmt.Sprintf("/api/instructor/v1/groups/%d", gid)},
		{http.MethodDelete, fmt.Sprintf("/api/instructor/v1/grants/%d", grant)},
	} {
		r := e.do(t, tc.method, tc.path, btok, map[string]any{"title": "x", "name": "x"})
		assert.Equal(t, 404, r.status, "%s %s: %s", tc.method, tc.path, r.raw)
	}
	// B cannot grant A's course or put it in a group, and sees none of A's students.
	r = e.do(t, http.MethodPost, "/api/instructor/v1/grants", btok, map[string]any{"phones": []string{"09129990095"}, "scope": "course", "target_id": cid, "duration": "unlimited"})
	assert.Equal(t, 422, r.status, r.raw)
	r = e.do(t, http.MethodPost, "/api/instructor/v1/groups", btok, map[string]any{"name": "B", "course_ids": []int{cid}})
	assert.Equal(t, 422, r.status, r.raw)
	r = e.do(t, http.MethodGet, "/api/instructor/v1/students", btok, nil)
	assert.Empty(t, r.data()["students"])
	r = e.do(t, http.MethodGet, "/api/instructor/v1/courses", btok, nil)
	assert.Empty(t, r.data()["courses"])
	// A's course is untouched.
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM learning_courses"))
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM learning_grants WHERE status = 'pending'"))
}

func TestStudentIDOR_AndGuards(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila")
	cid, lessons := e.publishedCourse(t, itok, "Course", 1)
	_, s1 := e.user(t, "09121110003")
	_, s2 := e.user(t, "09121110004")
	r := e.do(t, http.MethodPost, "/api/instructor/v1/grants", itok, map[string]any{"phones": []string{"09121110003"}, "scope": "course", "target_id": cid, "duration": "unlimited"})
	grant := idOf(t, r.data()["grants"].([]any)[0].(map[string]any)["id"])

	// A student without a grant gets the same 404 as for a missing id.
	for _, p := range []string{
		fmt.Sprintf("/api/v1/learning/courses/%d", cid), fmt.Sprintf("/api/v1/learning/lessons/%d", lessons[0]),
		fmt.Sprintf("/api/v1/learning/unlocked/%d", grant),
	} {
		r = e.do(t, http.MethodGet, p, s2, nil)
		assert.Equal(t, 404, r.status, p)
	}
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/learning/lessons/%d/progress", lessons[0]), s2, map[string]any{"position_seconds": 10})
	assert.Equal(t, 404, r.status)
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/learning/lessons/%d", lessons[0]), s1, nil)
	assert.Equal(t, 200, r.status, r.raw)

	// Students are not instructors: 403 (never 401); a pending applicant gets instructor_pending.
	r = e.do(t, http.MethodGet, "/api/instructor/v1/courses", s1, nil)
	assert.Equal(t, 403, r.status)
	assert.Equal(t, learning.ErrorCodeInstructorRequired, r.body["error_code"])
	r = e.do(t, http.MethodGet, "/api/instructor/v1/me", s1, nil)
	require.Equal(t, 200, r.status)
	assert.Nil(t, r.data()["instructor"])
	r = e.do(t, http.MethodPost, "/api/instructor/v1/apply", s1, map[string]any{"display_name": "Sara"})
	require.Equal(t, 201, r.status, r.raw)
	assert.Equal(t, "pending", r.data()["instructor"].(map[string]any)["status"])
	r = e.do(t, http.MethodGet, "/api/instructor/v1/courses", s1, nil)
	assert.Equal(t, 403, r.status)
	assert.Equal(t, learning.ErrorCodeInstructorPending, r.body["error_code"])

	// 401 body without a token.
	r = e.do(t, http.MethodGet, "/api/v1/learning/courses", "", nil)
	assert.Equal(t, 401, r.status)
	assert.Equal(t, "unauthenticated", r.body["error_code"])
}

func TestValidationAndRules(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila")
	r := e.do(t, http.MethodPost, "/api/instructor/v1/courses", itok, map[string]any{"kind": "book", "title": ""})
	require.Equal(t, 422, r.status, r.raw)
	errs := r.body["errors"].(map[string]any)
	assert.Contains(t, errs, "kind")
	assert.Contains(t, errs, "title")

	r = e.do(t, http.MethodPost, "/api/instructor/v1/courses", itok, map[string]any{"kind": "standalone", "title": "Meditation"})
	require.Equal(t, 201, r.status, r.raw)
	sid := idOf(t, r.data()["course"].(map[string]any)["id"])
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/instructor/v1/courses/%d/chapters", sid), itok, map[string]any{"title": "x"})
	assert.Equal(t, 422, r.status)
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/instructor/v1/courses/%d/lessons", sid), itok, map[string]any{"kind": "audio", "title": "Track"})
	require.Equal(t, 201, r.status, r.raw)
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/instructor/v1/courses/%d/lessons", sid), itok, map[string]any{"kind": "audio", "title": "Second"})
	assert.Equal(t, 422, r.status)

	r = e.do(t, http.MethodPost, "/api/instructor/v1/grants", itok, map[string]any{"phones": []string{"12"}, "scope": "course", "target_id": sid, "duration": "30"})
	assert.Equal(t, 422, r.status)
	assert.Contains(t, r.body["errors"].(map[string]any), "phones.0")
}

func TestDispatchSMS_Policy(t *testing.T) {
	e := setup(t)
	itok := e.instructor(t, "09120000001", "Leila")
	cid, _ := e.publishedCourse(t, itok, "Course", 1)
	uid, _ := e.user(t, "09121110005")
	_, err := e.db.Exec(`INSERT INTO notification_preferences (user_id, categories, quiet_hours_enabled, quiet_start, quiet_end, neutral_copy, created_at, updated_at)
		VALUES (?, '{"learning":false}', 1, '23:00:00', '08:00:00', 1, NOW(), NOW())`, uid)
	require.NoError(t, err)
	r := e.do(t, http.MethodPost, "/api/instructor/v1/grants", itok, map[string]any{
		"phones": []string{"09121110005", "09129990094"}, "scope": "course", "target_id": cid, "duration": "unlimited",
	})
	require.Equal(t, 201, r.status, r.raw)

	// 23:30: the unregistered number is deferred to 08:00 (default quiet hours); the user switched learning off.
	night := time.Date(2026, 10, 7, 23, 30, 0, 0, civildate.Tehran)
	sent, err := e.svc.DispatchSMS(context.Background(), night)
	require.NoError(t, err)
	assert.Equal(t, 0, sent)
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM learning_sms_outbox WHERE status = 'skipped' AND reason = 'category_off'"))
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM learning_sms_outbox WHERE status = 'pending' AND due_at = '2026-10-08 08:00:00'"))

	sent, err = e.svc.DispatchSMS(context.Background(), night.Add(time.Hour)) // still quiet, not due
	require.NoError(t, err)
	assert.Equal(t, 0, sent)

	// A provider failure is retried later.
	e.sms.Fail = true
	morning := time.Date(2026, 10, 8, 8, 0, 0, 0, civildate.Tehran)
	sent, err = e.svc.DispatchSMS(context.Background(), morning)
	require.NoError(t, err)
	assert.Equal(t, 0, sent)
	e.sms.Fail = false
	sent, err = e.svc.DispatchSMS(context.Background(), morning.Add(learning.SMSRetryAfter))
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
	assert.Equal(t, []string{"09129990094"}, e.sms.Sent())
}
