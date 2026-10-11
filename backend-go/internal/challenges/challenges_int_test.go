package challenges_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
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
	"github.com/ritme/backend-go/internal/challenges"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
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
	svc *challenges.Service
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
	svc := challenges.NewService(db)
	h := challenges.NewHandlers(svc, clock.Fixed(fixed))
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	p := "/api/v1/challenges"
	app.Get(p, locale, guard, h.Index)
	app.Get(p+"/rewards", locale, guard, h.Rewards)
	app.Post(p+"/:id/join", locale, guard, h.Join)
	app.Post(p+"/:id/tasks/:task/complete", locale, guard, h.CompleteTask)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), svc: svc}
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

func (e *env) period(t *testing.T, userID uint64, start string) {
	t.Helper()
	s := civildate.MustParse(start)
	_, err := e.db.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, bleeding_length, is_confirmed, is_estimated, source, created_at, updated_at)
		VALUES (?, ?, ?, 5, 1, 0, 'user_logged', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, userID, s, s.AddDays(4))
	require.NoError(t, err)
}

// challenge inserts a v2 challenge: duration 3 days, task 1 on days 1-3, task 2 on days 2-3.
func (e *env) challenge(t *testing.T, slug string, eligibility string, startRule string, minDays int, plusDays int) (id uint64, tasks [2]uint64) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO challenges (slug, title, is_active, sort_order, created_at, updated_at) VALUES (?, '{"fa":"کشش","en":"Stretch"}', 1, 1, NOW(), NOW())`, slug)
	require.NoError(t, err)
	cid, _ := res.LastInsertId()
	_, err = e.db.Exec(`INSERT INTO challenge_programs (challenge_id, duration_days, start_rule, eligibility, reward_badge_code, reward_badge_title, reward_plus_days, completion_min_days, created_at, updated_at)
		VALUES (?, 3, ?, ?, 'calm', '{"fa":"ریتم آرام","en":"Calm rhythm"}', ?, ?, NOW(), NOW())`, cid, startRule, eligibility, plusDays, minDays)
	require.NoError(t, err)
	for i, rng := range [][2]int{{1, 3}, {2, 3}} {
		r, err := e.db.Exec(`INSERT INTO challenge_tasks (challenge_id, day_from, day_to, title, sort_order, created_at, updated_at) VALUES (?, ?, ?, '{"fa":"کار","en":"Task"}', ?, NOW(), NOW())`,
			cid, rng[0], rng[1], i)
		require.NoError(t, err)
		tid, _ := r.LastInsertId()
		tasks[i] = uint64(tid)
	}
	return uint64(cid), tasks
}

// enroll starts a run on startedOn directly (the clock is fixed, so the day of the run is set by the start date).
func (e *env) enroll(t *testing.T, userID, challengeID uint64, startedOn string) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO challenge_enrollments (user_id, challenge_id, started_on, status, created_at, updated_at) VALUES (?, ?, ?, 'active', NOW(), NOW())`,
		userID, challengeID, startedOn)
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
	res, err := e.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m), string(raw))
	return response{status: res.StatusCode, body: m}
}

func ids(r response) []float64 {
	var out []float64
	for _, c := range r.data()["challenges"].([]any) {
		out = append(out, c.(map[string]any)["id"].(float64))
	}
	return out
}

func (e *env) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(query, args...).Scan(&n))
	return n
}

func complete(e *env, t *testing.T, tok string, cid, task uint64, body any) response {
	t.Helper()
	return e.do(t, http.MethodPost, "/api/v1/challenges/"+itoa(cid)+"/tasks/"+itoa(task)+"/complete", tok, body)
}

func itoa(n uint64) string { b, _ := json.Marshal(n); return string(b) }

func TestChallenges_Unauthenticated(t *testing.T) {
	e := setup(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/challenges"},
		{http.MethodGet, "/api/v1/challenges/rewards"},
		{http.MethodPost, "/api/v1/challenges/1/join"},
		{http.MethodPost, "/api/v1/challenges/1/tasks/1/complete"},
	} {
		r := e.do(t, tc.method, tc.path, "", nil)
		assert.Equal(t, http.StatusUnauthorized, r.status, tc.path)
	}
}

func TestChallenges_Eligibility(t *testing.T) {
	e := setup(t)
	menstrual, tokM := e.user(t, "09120000001") // period started 2026-10-05: day 2, menstrual
	e.period(t, menstrual, "2026-10-05")
	_, tokN := e.user(t, "09120000002") // no cycle data: mode cycle, no phase

	all, _ := e.challenge(t, "all", `null`, "anytime", 0, 0)
	onlyPreg, _ := e.challenge(t, "preg", `{"modes_include":["pregnancy"]}`, "anytime", 0, 0)
	notCycle, _ := e.challenge(t, "notcycle", `{"modes_exclude":["cycle"]}`, "anytime", 0, 0)
	menstrualOnly, _ := e.challenge(t, "mens", `{"phases_include":["menstrual"]}`, "anytime", 0, 0)
	startMens, _ := e.challenge(t, "startmens", `null`, "menstrual", 0, 0)
	// a legacy challenge is not part of v2
	_, err := e.db.Exec(`INSERT INTO challenges (slug, title, is_active, sort_order, created_at, updated_at) VALUES ('legacy', '{"fa":"x"}', 1, 1, NOW(), NOW())`)
	require.NoError(t, err)

	r := e.do(t, http.MethodGet, "/api/v1/challenges", tokM, nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.ElementsMatch(t, []float64{float64(all), float64(menstrualOnly), float64(startMens)}, ids(r), "menstrual cycle user")

	r = e.do(t, http.MethodGet, "/api/v1/challenges", tokN, nil)
	assert.ElementsMatch(t, []float64{float64(all), float64(startMens)}, ids(r), "no phase: phase rules refuse")
	for _, c := range r.data()["challenges"].([]any) {
		m := c.(map[string]any)
		assert.Equal(t, m["id"].(float64) == float64(all), m["can_join"], "start rule menstrual cannot start without the phase")
	}

	// join: not eligible is 403, an unmet start rule 409, unknown 404
	assert.Equal(t, http.StatusForbidden, e.do(t, http.MethodPost, "/api/v1/challenges/"+itoa(onlyPreg)+"/join", tokM, nil).status)
	assert.Equal(t, http.StatusForbidden, e.do(t, http.MethodPost, "/api/v1/challenges/"+itoa(notCycle)+"/join", tokM, nil).status)
	assert.Equal(t, http.StatusForbidden, e.do(t, http.MethodPost, "/api/v1/challenges/"+itoa(menstrualOnly)+"/join", tokN, nil).status)
	r = e.do(t, http.MethodPost, "/api/v1/challenges/"+itoa(startMens)+"/join", tokN, nil)
	assert.Equal(t, http.StatusConflict, r.status)
	assert.Equal(t, "challenge_not_startable", r.body["error_code"])
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodPost, "/api/v1/challenges/99999/join", tokM, nil).status)

	r = e.do(t, http.MethodPost, "/api/v1/challenges/"+itoa(startMens)+"/join", tokM, nil)
	require.Equal(t, http.StatusCreated, r.status, r.body)
	assert.Equal(t, "2026-10-06", r.data()["challenge"].(map[string]any)["enrollment"].(map[string]any)["started_on"])
	r = e.do(t, http.MethodPost, "/api/v1/challenges/"+itoa(startMens)+"/join", tokM, nil)
	assert.Equal(t, http.StatusOK, r.status, "second join is idempotent")
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM challenge_enrollments WHERE user_id = ?`, menstrual))
}

func TestChallenges_CompletionRewardOnceAndIDOR(t *testing.T) {
	e := setup(t)
	a, tokA := e.user(t, "09120000001")
	_, tokB := e.user(t, "09120000002")
	cid, tasks := e.challenge(t, "stretch", `null`, "anytime", 2, 3) // 2 of 3 days, reward 3 Plus days
	other, otherTasks := e.challenge(t, "other", `null`, "anytime", 0, 0)
	e.enroll(t, a, cid, "2026-10-05") // today is day 2 of the run

	// not joined (B) -> 409; IDOR: B cannot touch A's run, a foreign challenge's task is 404 for A
	r := complete(e, t, tokB, cid, tasks[0], nil)
	assert.Equal(t, http.StatusConflict, r.status)
	assert.Equal(t, "challenge_not_joined", r.body["error_code"])
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM challenge_task_completions`))
	assert.Equal(t, http.StatusNotFound, complete(e, t, tokA, cid, otherTasks[0], nil).status)
	_ = other

	// bad days: the future, day 0, a day outside the task's range
	assert.Equal(t, http.StatusUnprocessableEntity, complete(e, t, tokA, cid, tasks[0], map[string]any{"day": 3}).status, "future day")
	assert.Equal(t, http.StatusUnprocessableEntity, complete(e, t, tokA, cid, tasks[0], map[string]any{"day": 0}).status)
	assert.Equal(t, http.StatusUnprocessableEntity, complete(e, t, tokA, cid, tasks[1], map[string]any{"day": 1}).status, "task 2 starts on day 2")

	// day 1 (task 1) done: 1 of 2 required days
	r = complete(e, t, tokA, cid, tasks[0], map[string]any{"day": 1})
	require.Equal(t, http.StatusOK, r.status, r.body)
	enr := r.data()["challenge"].(map[string]any)["enrollment"].(map[string]any)
	assert.EqualValues(t, 1, enr["done_days"])
	assert.EqualValues(t, 2, enr["required_days"])
	assert.Equal(t, false, r.data()["completed_now"])

	// day 2 needs both tasks: after the first only, still active
	r = complete(e, t, tokA, cid, tasks[0], nil)
	assert.Equal(t, "active", r.data()["challenge"].(map[string]any)["enrollment"].(map[string]any)["status"])
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM challenge_reward_grants`))
	// ... and the repeat is idempotent
	complete(e, t, tokA, cid, tasks[0], nil)
	assert.Equal(t, 2, e.count(t, `SELECT COUNT(*) FROM challenge_task_completions`))

	r = complete(e, t, tokA, cid, tasks[1], nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, true, r.data()["completed_now"])
	assert.Equal(t, "completed", r.data()["challenge"].(map[string]any)["enrollment"].(map[string]any)["status"])
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM challenge_reward_grants WHERE user_id = ?`, a))
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ? AND source = 'challenge'`, a))
	var days int
	require.NoError(t, e.db.QueryRow(`SELECT DATEDIFF(ends_at, starts_at) FROM plus_subscriptions WHERE user_id = ?`, a).Scan(&days))
	assert.Equal(t, 3, days)

	// repeats never grant again
	for range 3 {
		r = complete(e, t, tokA, cid, tasks[1], nil)
		require.Equal(t, http.StatusOK, r.status)
		assert.Equal(t, false, r.data()["completed_now"])
	}
	complete(e, t, tokA, cid, tasks[0], map[string]any{"day": 1})
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM challenge_reward_grants`))
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE source = 'challenge'`))

	// badge: A sees it localized, B sees none
	r = e.do(t, http.MethodGet, "/api/v1/challenges/rewards", tokA, nil)
	rewards := r.data()["rewards"].([]any)
	require.Len(t, rewards, 1)
	assert.Equal(t, "Calm rhythm", rewards[0].(map[string]any)["badge_title"])
	assert.EqualValues(t, 3, rewards[0].(map[string]any)["plus_days"])
	assert.Empty(t, e.do(t, http.MethodGet, "/api/v1/challenges/rewards", tokB, nil).data()["rewards"])
}

func TestChallenges_RewardIsRaceSafe(t *testing.T) {
	e := setup(t)
	a, _ := e.user(t, "09120000001")
	cid, tasks := e.challenge(t, "race", `null`, "anytime", 1, 5)
	e.enroll(t, a, cid, "2026-10-06")
	// day 1 has only task 1; completing it finishes the run. Fire it concurrently.
	var wg sync.WaitGroup
	today := civildate.InTehran(fixed)
	finished := make(chan bool, 12)
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := e.svc.CompleteTask(context.Background(), a, cid, tasks[0], nil, today, fixed)
			if err == nil {
				finished <- res.JustFinally
			}
		}()
	}
	wg.Wait()
	close(finished)
	n := 0
	for f := range finished {
		if f {
			n++
		}
	}
	assert.Equal(t, 1, n, "exactly one call completes the run")
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM challenge_reward_grants`))
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE source = 'challenge'`))
}

func TestChallenges_RunEnded(t *testing.T) {
	e := setup(t)
	a, tokA := e.user(t, "09120000001")
	cid, tasks := e.challenge(t, "old", `null`, "anytime", 0, 0)
	e.enroll(t, a, cid, "2026-09-01") // 3-day run long over
	r := complete(e, t, tokA, cid, tasks[0], nil)
	assert.Equal(t, http.StatusConflict, r.status)
	assert.Equal(t, "challenge_run_ended", r.body["error_code"])
	r = e.do(t, http.MethodGet, "/api/v1/challenges", tokA, nil)
	assert.Equal(t, "expired", r.data()["challenges"].([]any)[0].(map[string]any)["enrollment"].(map[string]any)["status"])
}
