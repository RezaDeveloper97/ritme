package todo_test

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
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/todo"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const clientID = "0199c0de-0000-7000-8000-00000c0ffee1"

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

// fixed is the request clock: Tuesday 2026-10-06 09:00 Tehran.
var fixed = time.Date(2026, 10, 6, 9, 0, 0, 0, civildate.Tehran)

type env struct {
	db  *sql.DB
	app *fiber.App
	iss *passport.Issuer
}

func setup(t *testing.T) *env {
	t.Helper()
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))
	h := todo.NewHandlers(todo.NewService(db), clock.Fixed(fixed))

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	p := "/api/v1/todo"
	app.Get(p, locale, guard, h.Board)
	app.Post(p+"/tasks", locale, guard, h.Store)
	app.Get(p+"/tasks/:id", locale, guard, h.Show)
	app.Put(p+"/tasks/:id", locale, guard, h.Update)
	app.Delete(p+"/tasks/:id", locale, guard, h.Destroy)
	app.Post(p+"/tasks/:id/items", locale, guard, h.StoreItem)
	app.Put(p+"/tasks/:id/items/:item", locale, guard, h.UpdateItem)
	app.Delete(p+"/tasks/:id/items/:item", locale, guard, h.DestroyItem)
	app.Post(p+"/suggestions/:key/accept", locale, guard, h.AcceptSuggestion)
	app.Post(p+"/suggestions/:key/dismiss", locale, guard, h.DismissSuggestion)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now())
	require.NoError(t, err)
	return uint64(id), tok.AccessToken
}

// period logs a confirmed period starting on start (5 days).
func (e *env) period(t *testing.T, userID uint64, start string) {
	t.Helper()
	s := civildate.MustParse(start)
	_, err := e.db.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, bleeding_length, is_confirmed, is_estimated, source, created_at, updated_at)
		VALUES (?, ?, ?, 5, 1, 0, 'user_logged', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, userID, s, s.AddDays(4))
	require.NoError(t, err)
}

type response struct {
	status int
	body   map[string]any
}

func (r response) data() map[string]any {
	d, _ := r.body["data"].(map[string]any)
	return d
}

func (e *env) do(t *testing.T, method, path, token, lang string, body any) response {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rd = strings.NewReader(string(b))
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	res, err := e.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m), string(raw))
	return response{status: res.StatusCode, body: m}
}

func group(r response, key string) []any {
	for _, g := range r.data()["groups"].([]any) {
		gm := g.(map[string]any)
		if gm["key"] == key {
			return gm["tasks"].([]any)
		}
	}
	return nil
}

func TestTodo_Unauthenticated(t *testing.T) {
	e := setup(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/todo"},
		{http.MethodPost, "/api/v1/todo/tasks"},
		{http.MethodGet, "/api/v1/todo/tasks/1"},
		{http.MethodPost, "/api/v1/todo/suggestions/period_supplies/accept"},
	} {
		r := e.do(t, tc.method, tc.path, "", "", nil)
		assert.Equal(t, http.StatusUnauthorized, r.status, tc.path)
		assert.Equal(t, "unauthenticated", r.body["error_code"])
	}
}

func TestTodo_TasksItemsBoardAndIDOR(t *testing.T) {
	e := setup(t)
	_, tokA := e.user(t, "09120000001")
	_, tokB := e.user(t, "09120000002")

	// Fresh board: empty groups, no suggestion (no cycle data).
	r := e.do(t, http.MethodGet, "/api/v1/todo", tokA, "fa", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, "2026-10-06", r.data()["date"])
	assert.Nil(t, r.data()["suggestion"])
	assert.Empty(t, group(r, "today"))
	assert.Equal(t, []any{"shopping", "work", "personal", "health"}, r.data()["categories"])

	r = e.do(t, http.MethodPost, "/api/v1/todo/tasks", tokA, "fa", map[string]any{
		"title": "جلسه با تیم طراحی", "category": "work", "due_date": "2026-10-06", "due_time": "10:00",
	})
	require.Equal(t, http.StatusCreated, r.status, r.body)
	assert.Equal(t, "کار اضافه شد", r.body["message"])
	meeting := r.data()["task"].(map[string]any)
	assert.Equal(t, "10:00", meeting["due_time"])
	assert.Equal(t, false, meeting["remind"])

	r = e.do(t, http.MethodPost, "/api/v1/todo/tasks", tokA, "en", map[string]any{
		"title": "Fruit and yogurt", "category": "shopping", "due_date": "2026-10-06",
	})
	require.Equal(t, http.StatusCreated, r.status, r.body)
	list := r.data()["task"].(map[string]any)
	listID := uint64(list["id"].(float64))
	r = e.do(t, http.MethodPost, "/api/v1/todo/tasks", tokA, "en", map[string]any{
		"title": "Monthly report", "category": "work", "due_date": "2026-10-07", "due_time": "14:00", "remind": true,
	})
	require.Equal(t, http.StatusCreated, r.status, r.body)
	r = e.do(t, http.MethodPost, "/api/v1/todo/tasks", tokA, "en", map[string]any{"title": "Renew insurance", "category": "personal"})
	require.Equal(t, http.StatusCreated, r.status, r.body)

	// Validation (controller-style 422).
	r = e.do(t, http.MethodPost, "/api/v1/todo/tasks", tokA, "en", map[string]any{"title": "", "category": "food", "remind": true})
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
	assert.Equal(t, false, r.body["success"])
	errs := r.body["errors"].(map[string]any)
	assert.Contains(t, errs, "title")
	assert.Contains(t, errs, "category")

	// Items: add three, tick one.
	itemsPath := fmt.Sprintf("/api/v1/todo/tasks/%d/items", listID)
	var firstItem uint64
	for i, title := range []string{"Apples", "Yogurt", "Bread"} {
		r = e.do(t, http.MethodPost, itemsPath, tokA, "en", map[string]any{"title": title})
		require.Equal(t, http.StatusCreated, r.status, r.body)
		if i == 0 {
			firstItem = uint64(r.data()["item"].(map[string]any)["id"].(float64))
		}
	}
	assert.Equal(t, map[string]any{"total": float64(3), "open": float64(3)}, r.data()["task"].(map[string]any)["list"])
	itemPath := fmt.Sprintf("%s/%d", itemsPath, firstItem)
	r = e.do(t, http.MethodPut, itemPath, tokA, "en", map[string]any{"done": true})
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, true, r.data()["item"].(map[string]any)["done"])
	assert.Equal(t, "2026-10-06T09:00:00+03:30", r.data()["item"].(map[string]any)["done_at"])
	assert.Equal(t, float64(2), r.data()["task"].(map[string]any)["list"].(map[string]any)["open"])

	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/todo/tasks/%d", listID), tokA, "en", nil)
	require.Equal(t, http.StatusOK, r.status)
	assert.Len(t, r.data()["items"], 3)

	// Tick the meeting → stays in «امروز», struck through, after the open ones.
	meetingPath := fmt.Sprintf("/api/v1/todo/tasks/%d", uint64(meeting["id"].(float64)))
	r = e.do(t, http.MethodPut, meetingPath, tokA, "en", map[string]any{"done": true})
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, true, r.data()["task"].(map[string]any)["done"])
	assert.Equal(t, "جلسه با تیم طراحی", r.data()["task"].(map[string]any)["title"]) // partial update keeps the rest

	r = e.do(t, http.MethodGet, "/api/v1/todo", tokA, "en", nil)
	today := group(r, "today")
	require.Len(t, today, 2)
	assert.Equal(t, "Fruit and yogurt", today[0].(map[string]any)["title"])
	assert.Equal(t, true, today[1].(map[string]any)["done"])
	assert.Len(t, group(r, "tomorrow"), 1)
	assert.Len(t, group(r, "later"), 1)

	// IDOR: user B gets a uniform 404 on A's task and items and sees none of A's data.
	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, meetingPath, nil},
		{http.MethodPut, meetingPath, map[string]any{"done": false}},
		{http.MethodDelete, meetingPath, nil},
		{http.MethodPost, itemsPath, map[string]any{"title": "x"}},
		{http.MethodPut, itemPath, map[string]any{"done": false}},
		{http.MethodDelete, itemPath, nil},
	} {
		r = e.do(t, tc.method, tc.path, tokB, "en", tc.body)
		assert.Equal(t, http.StatusNotFound, r.status, tc.method+" "+tc.path)
	}
	r = e.do(t, http.MethodGet, "/api/v1/todo", tokB, "", nil)
	assert.Empty(t, group(r, "today"))

	// Delete a list item, then the list itself (cascade).
	r = e.do(t, http.MethodDelete, itemPath, tokA, "en", nil)
	require.Equal(t, http.StatusOK, r.status)
	assert.Equal(t, float64(2), r.data()["task"].(map[string]any)["list"].(map[string]any)["total"])
	r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/todo/tasks/%d", listID), tokA, "en", nil)
	require.Equal(t, http.StatusOK, r.status)
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM todo_items WHERE task_id = ?`, listID).Scan(&n))
	assert.Zero(t, n)
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/todo/tasks/%d", listID), tokA, "fa", nil)
	assert.Equal(t, http.StatusNotFound, r.status)
	assert.Equal(t, "todo_not_found", r.body["error_code"])
}

func TestTodo_CycleSuggestion(t *testing.T) {
	e := setup(t)
	idA, tokA := e.user(t, "09120000003")
	idB, tokB := e.user(t, "09120000004")
	// Last periods 2026-08-12 and 2026-09-10 (29-day cycle) → next predicted 2026-10-09, 3 days from 2026-10-06.
	for _, id := range []uint64{idA, idB} {
		e.period(t, id, "2026-08-12")
		e.period(t, id, "2026-09-10")
	}

	r := e.do(t, http.MethodGet, "/api/v1/todo", tokA, "fa", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	sg, ok := r.data()["suggestion"].(map[string]any)
	require.True(t, ok, r.data())
	assert.Equal(t, "period_supplies", sg["key"])
	assert.Equal(t, "2026-10-09", sg["period_start"])
	assert.Equal(t, float64(3), sg["days"])
	assert.Equal(t, "پریودت ۳ روز دیگر است؛ «خرید نوار بهداشتی» را اضافه کنم؟", sg["prompt"])
	assert.Equal(t, "افزودن", sg["action"])

	// An admin edit of the copy is used on the next request.
	_, err := e.db.Exec("UPDATE message_contents SET payload = JSON_SET(payload, '$.task_title', 'Buy period pads') " +
		"WHERE `group` = 'todo_suggestion' AND item_key = 'period_supplies' AND locale = 'en'")
	require.NoError(t, err)

	// Unknown key → 404; accept creates a shopping task due the day before; a second accept is 404; it is gone.
	r = e.do(t, http.MethodPost, "/api/v1/todo/suggestions/other/accept", tokA, "en", nil)
	assert.Equal(t, http.StatusNotFound, r.status)
	r = e.do(t, http.MethodPost, "/api/v1/todo/suggestions/period_supplies/accept", tokA, "en", nil)
	require.Equal(t, http.StatusCreated, r.status, r.body)
	task := r.data()["task"].(map[string]any)
	assert.Equal(t, "Buy period pads", task["title"])
	assert.Equal(t, "shopping", task["category"])
	assert.Equal(t, "2026-10-08", task["due_date"])
	assert.Equal(t, "period_supplies", task["suggestion_key"])
	assert.Nil(t, r.data()["item"])
	r = e.do(t, http.MethodPost, "/api/v1/todo/suggestions/period_supplies/accept", tokA, "en", nil)
	assert.Equal(t, http.StatusNotFound, r.status)
	assert.Equal(t, "todo_suggestion_not_found", r.body["error_code"])
	r = e.do(t, http.MethodGet, "/api/v1/todo", tokA, "en", nil)
	assert.Nil(t, r.data()["suggestion"])

	// User B already keeps a shopping list → the suggestion lands there as an item due the period start.
	r = e.do(t, http.MethodPost, "/api/v1/todo/tasks", tokB, "fa", map[string]any{"title": "لیست خرید", "category": "shopping"})
	require.Equal(t, http.StatusCreated, r.status)
	listID := uint64(r.data()["task"].(map[string]any)["id"].(float64))
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/todo/tasks/%d/items", listID), tokB, "fa", map[string]any{"title": "نان"})
	require.Equal(t, http.StatusCreated, r.status)
	r = e.do(t, http.MethodPost, "/api/v1/todo/suggestions/period_supplies/accept", tokB, "fa", nil)
	require.Equal(t, http.StatusCreated, r.status, r.body)
	item := r.data()["item"].(map[string]any)
	assert.Equal(t, "نوار بهداشتی", item["title"])
	assert.Equal(t, "2026-10-09", item["due_date"])
	assert.Equal(t, float64(listID), r.data()["task"].(map[string]any)["id"])
	assert.Equal(t, float64(2), r.data()["task"].(map[string]any)["list"].(map[string]any)["total"])
}

func TestTodo_DismissSuggestion(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000005")
	e.period(t, id, "2026-08-12")
	e.period(t, id, "2026-09-10")
	r := e.do(t, http.MethodPost, "/api/v1/todo/suggestions/period_supplies/dismiss", tok, "en", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	r = e.do(t, http.MethodGet, "/api/v1/todo", tok, "en", nil)
	assert.Nil(t, r.data()["suggestion"])
	r = e.do(t, http.MethodPost, "/api/v1/todo/suggestions/period_supplies/dismiss", tok, "en", nil)
	assert.Equal(t, http.StatusNotFound, r.status)

	// A pregnancy-mode account never gets it.
	id2, tok2 := e.user(t, "09120000006")
	e.period(t, id2, "2026-08-12")
	e.period(t, id2, "2026-09-10")
	_, err := e.db.Exec(`INSERT INTO user_profiles (user_id, cycle_duration, created_at, updated_at) VALUES (?, 29, NOW(), NOW())`, id2)
	require.NoError(t, err)
	_, err = e.db.Exec(`INSERT INTO user_life_profiles (user_id, life_mode, created_at, updated_at) VALUES (?, 'pregnancy', NOW(), NOW())`, id2)
	require.NoError(t, err)
	r = e.do(t, http.MethodGet, "/api/v1/todo", tok2, "en", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Nil(t, r.data()["suggestion"])
}
