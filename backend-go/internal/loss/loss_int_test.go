package loss_test

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
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/loss"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/profile"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ff10e"
	now      = "2026-09-23T10:00:00+03:30"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type env struct {
	db  *sql.DB
	app *fiber.App
	iss *passport.Issuer
}

func setup(t *testing.T, notesDisabled bool) *env {
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
	notes, err := loss.NewNoteBox(nil, nil, notesDisabled)
	require.NoError(t, err)
	h := loss.NewHandlers(loss.NewService(loss.Deps{
		DB: db, Catalog: catalog.NewReader(catalogstore.New(db), nil, 0, quiet),
		Modes: profile.NewOnboardingHandlers(db, clock.Real{}), Notes: notes, Logger: quiet,
	}), clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/loss", locale, guard, h.Show)
	app.Post("/api/v1/loss", locale, guard, h.Store)
	app.Delete("/api/v1/loss", locale, guard, h.Destroy)
	app.Delete("/api/v1/loss/note", locale, guard, h.DestroyNote)
	app.Put("/api/v1/loss/followup", locale, guard, h.Followup)
	app.Post("/api/v1/loss/moods", locale, guard, h.StoreMood)
	app.Get("/api/v1/loss/note", locale, guard, h.ShowNote)
	app.Put("/api/v1/loss/note", locale, guard, h.UpdateNote)
	app.Put("/api/v1/loss/next-step", locale, guard, h.NextStep)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // test id
	require.NoError(t, err)
	return uint64(id), tok.AccessToken //nolint:gosec // test id
}

// pregnant gives the user an active pregnancy (LMP 2026-08-02 → week 8) with open alerts, a prenatal medication
// «تا پایان بارداری», an upcoming pregnancy visit, an ordinary medication and an ordinary appointment.
func (e *env) pregnant(t *testing.T, userID uint64) {
	t.Helper()
	exec := func(q string, args ...any) {
		_, err := e.db.Exec(q, args...)
		require.NoError(t, err)
	}
	exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, cycle_mode, age_source, lmp_date, onboarding_completed, created_at, updated_at)
		VALUES (?, 1, 0, 'lmp', '2026-08-02', 1, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, userID)
	exec(`INSERT INTO user_life_profiles (user_id, life_mode, ivf_iui, track_contraception, created_at, updated_at)
		VALUES (?, 'pregnancy', 0, 0, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, userID)
	for _, lvl := range []string{"emergency", "warning"} {
		exec(`INSERT INTO pregnancy_alerts (user_id, alert_level, alert_type, title, message, created_at, updated_at)
			VALUES (?, ?, 'v2:test', 't', 'm', '2026-09-20 09:00:00', '2026-09-20 09:00:00')`, userID, lvl)
	}
	exec(`INSERT INTO reminders (user_id, type, title, recurrence, is_active, meta, created_at, updated_at) VALUES
		(?, 'medication', 'Folic acid', 'daily', 1, '{"v":1,"duration":"pregnancy_end"}', '2026-09-01 09:00:00', '2026-09-01 09:00:00'),
		(?, 'medication', 'Iron', 'daily', 1, '{"v":1,"duration":"ongoing"}', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, userID, userID)
	exec(`INSERT INTO reminders (user_id, type, title, scheduled_at, recurrence, is_active, meta, created_at, updated_at) VALUES
		(?, 'appointment', 'NT scan', '2026-10-10 10:00:00', 'none', 1, '{"v":1,"kind":"in_person","topic":"ultrasound","remind_before":"1d","status":"scheduled","care_item_key":"nt_scan"}', '2026-09-01 09:00:00', '2026-09-01 09:00:00'),
		(?, 'appointment', 'Dentist', '2026-10-11 10:00:00', 'none', 1, '{"v":1,"kind":"in_person","topic":"other","remind_before":"1d","status":"scheduled"}', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, userID, userID)
}

func (e *env) scalar(t *testing.T, q string, args ...any) any {
	t.Helper()
	var v any
	require.NoError(t, e.db.QueryRow(q, args...).Scan(&v))
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return v
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

func (r response) obj(keys ...string) map[string]any {
	m := r.data()
	for _, k := range keys {
		m, _ = m[k].(map[string]any)
	}
	return m
}

func (e *env) do(t *testing.T, method, path, token, body string, at ...string) response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	ts := now
	if len(at) > 0 {
		ts = at[0]
	}
	req.Header.Set(clock.Header, ts)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return response{status: resp.StatusCode, body: m, raw: string(raw)}
}

func TestLoss_RecordStopsPregnancyContentDurably(t *testing.T) {
	e := setup(t, false)
	uid, tok := e.user(t, "09120004001")
	e.pregnant(t, uid)

	// Nothing recorded yet: all null, 200.
	r := e.do(t, http.MethodGet, "/api/v1/loss", tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["loss"])
	assert.EqualValues(t, 0, r.data()["losses_count"])

	r = e.do(t, http.MethodPost, "/api/v1/loss", tok, `{"type":"early_miscarriage","occurred_on":"2026-09-20"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	l := r.obj("loss")
	assert.Equal(t, "early_miscarriage", l["type"])
	assert.Equal(t, "2026-09-20", l["occurred_on"])
	assert.Equal(t, true, l["content_stopped"])
	assert.Equal(t, false, l["companion_notified"])
	assert.Equal(t, false, r.data()["recurrent_hint"])
	assert.Equal(t, "2026-10-04", r.obj("followup", "visit")["suggested_on"], "visit ≈ 2 weeks after the loss date")

	// Pregnancy mode off (the flag every pregnancy reader keys on), alerts closed, mode back to cycle.
	assert.EqualValues(t, 0, e.scalar(t, `SELECT pregnancy_mode FROM pregnancy_profiles WHERE user_id = ?`, uid))
	assert.EqualValues(t, 0, e.scalar(t, `SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ? AND is_dismissed = 0`, uid))
	assert.Equal(t, "cycle", e.scalar(t, `SELECT life_mode FROM user_life_profiles WHERE user_id = ?`, uid))
	// Pregnancy-only reminders paused / cancelled; the ordinary ones untouched.
	assert.EqualValues(t, 0, e.scalar(t, `SELECT is_active FROM reminders WHERE user_id = ? AND title = 'Folic acid'`, uid))
	assert.EqualValues(t, 1, e.scalar(t, `SELECT is_active FROM reminders WHERE user_id = ? AND title = 'Iron'`, uid))
	assert.Equal(t, "cancelled", e.scalar(t, `SELECT JSON_UNQUOTE(JSON_EXTRACT(meta, '$.status')) FROM reminders WHERE user_id = ? AND title = 'NT scan'`, uid))
	assert.Equal(t, "scheduled", e.scalar(t, `SELECT JSON_UNQUOTE(JSON_EXTRACT(meta, '$.status')) FROM reminders WHERE user_id = ? AND title = 'Dentist'`, uid))
	assert.EqualValues(t, 2, e.scalar(t, `SELECT JSON_LENGTH(paused_reminders) FROM pregnancy_losses WHERE user_id = ?`, uid))

	// A second POST the same day corrects the event, never a second loss (no false recurrent hint).
	r = e.do(t, http.MethodPost, "/api/v1/loss", tok, `{"type":"chemical"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "chemical", r.obj("loss")["type"])
	assert.EqualValues(t, 1, r.data()["losses_count"])
	assert.EqualValues(t, 2, e.scalar(t, `SELECT JSON_LENGTH(paused_reminders) FROM pregnancy_losses WHERE user_id = ?`, uid))

	// A loss on another day is a new event: recurrent hint at 2.
	r = e.do(t, http.MethodPost, "/api/v1/loss", tok, `{}`, "2026-09-30T10:00:00+03:30")
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, "unspecified", r.obj("loss")["type"])
	assert.EqualValues(t, 2, r.data()["losses_count"])
	assert.Equal(t, true, r.data()["recurrent_hint"])
}

func TestLoss_FollowupMoodNoteNextStep(t *testing.T) {
	e := setup(t, false)
	uid, tok := e.user(t, "09120004002")
	e.pregnant(t, uid)
	require.Equal(t, http.StatusCreated, e.do(t, http.MethodPost, "/api/v1/loss", tok, `{"type":"early_miscarriage"}`).status)

	// Beta test next week + a visit: two care appointments.
	r := e.do(t, http.MethodPut, "/api/v1/loss/followup", tok, `{"beta_next_on":"2026-09-30","visit_at":"2026-10-07 11:30"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	beta := r.obj("followup", "beta")
	assert.Equal(t, "2026-09-30", beta["next_on"])
	appt, _ := beta["appointment"].(map[string]any)
	require.NotNil(t, appt, r.raw)
	assert.Equal(t, "2026-09-30 09:00:00", appt["scheduled_at"])
	assert.Equal(t, "Lab test", appt["title"], "a neutral title")
	visit, _ := r.obj("followup", "visit")["appointment"].(map[string]any)
	require.NotNil(t, visit)
	assert.Equal(t, "2026-10-07 11:30:00", visit["scheduled_at"])
	assert.EqualValues(t, 2, e.scalar(t, `SELECT COUNT(*) FROM reminders WHERE user_id = ? AND type = 'appointment' AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.topic')) IN ('lab','checkup')`, uid))

	// Moving the beta day moves the same appointment.
	r = e.do(t, http.MethodPut, "/api/v1/loss/followup", tok, `{"beta_next_on":"2026-10-01"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "2026-10-01 09:00:00", r.obj("followup", "beta")["appointment"].(map[string]any)["scheduled_at"])
	assert.EqualValues(t, 1, e.scalar(t, `SELECT COUNT(*) FROM reminders WHERE user_id = ? AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.topic')) = 'lab'`, uid))

	// Beta negative: next test cleared, its appointment cancelled; bleeding stopped today.
	r = e.do(t, http.MethodPut, "/api/v1/loss/followup", tok, `{"beta_negative":true,"bleeding_stopped":true}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	beta = r.obj("followup", "beta")
	assert.Equal(t, true, beta["negative"])
	assert.Equal(t, "2026-09-23", beta["negative_on"])
	assert.Nil(t, beta["next_on"])
	assert.Nil(t, beta["appointment"])
	assert.Equal(t, "2026-09-23", r.obj("followup", "bleeding")["stopped_on"])

	// Validation: a past beta day, an unknown mood, a bad next step.
	r = e.do(t, http.MethodPut, "/api/v1/loss/followup", tok, `{"beta_next_on":"2026-09-01"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.raw, `"beta_next_on"`)
	r = e.do(t, http.MethodPost, "/api/v1/loss/moods", tok, `{"mood":"happy"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	r = e.do(t, http.MethodPut, "/api/v1/loss/next-step", tok, `{"choice":"later"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)

	// Mood: one per day, last answer wins.
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPost, "/api/v1/loss/moods", tok, `{"mood":"sad"}`).status)
	r = e.do(t, http.MethodPost, "/api/v1/loss/moods", tok, `{"mood":"numb"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "numb", r.obj("mood")["today"])
	assert.Len(t, r.obj("mood")["recent"], 1)

	// Private note: encrypted at rest, readable only through the API.
	const secret = "a few words only for me"
	r = e.do(t, http.MethodPut, "/api/v1/loss/note", tok, `{"note":"`+secret+`"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, secret, r.data()["note"])
	stored, _ := e.scalar(t, `SELECT private_note FROM pregnancy_losses WHERE user_id = ?`, uid).(string)
	assert.NotContains(t, stored, "words")
	assert.True(t, strings.HasPrefix(stored, "v1:"))
	r = e.do(t, http.MethodGet, "/api/v1/loss/note", tok, "")
	assert.Equal(t, secret, r.data()["note"])
	assert.Equal(t, true, e.do(t, http.MethodGet, "/api/v1/loss", tok, "").obj("note")["has_note"])
	r = e.do(t, http.MethodPut, "/api/v1/loss/note", tok, `{"note":null}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["note"])

	// Next step ttc → TTC mode (user_goal ttc); nothing → cycle.
	r = e.do(t, http.MethodPut, "/api/v1/loss/next-step", tok, `{"choice":"ttc"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "ttc", r.obj("loss")["next_step"])
	assert.Equal(t, "ttc", e.scalar(t, `SELECT life_mode FROM user_life_profiles WHERE user_id = ?`, uid))
	assert.Equal(t, "ttc", e.scalar(t, `SELECT user_goal FROM user_profiles WHERE user_id = ?`, uid))
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPut, "/api/v1/loss/next-step", tok, `{"choice":"nothing"}`).status)
	assert.Equal(t, "cycle", e.scalar(t, `SELECT life_mode FROM user_life_profiles WHERE user_id = ?`, uid))
	assert.EqualValues(t, 0, e.scalar(t, `SELECT pregnancy_mode FROM pregnancy_profiles WHERE user_id = ?`, uid), "a next step never re-opens pregnancy")
}

// User isolation: another account sees no loss, cannot write follow-up / mood / note / next step on it (404), and
// cannot read the note; a note copied into another row does not decrypt. 401 without a token.
func TestLoss_Isolation(t *testing.T) {
	e := setup(t, false)
	sara, saraTok := e.user(t, "09120004003")
	_, ninaTok := e.user(t, "09120004004")
	e.pregnant(t, sara)
	require.Equal(t, http.StatusCreated, e.do(t, http.MethodPost, "/api/v1/loss", saraTok, `{"type":"ectopic"}`).status)
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPut, "/api/v1/loss/note", saraTok, `{"note":"mine"}`).status)

	r := e.do(t, http.MethodGet, "/api/v1/loss", ninaTok, "")
	require.Equal(t, http.StatusOK, r.status)
	assert.Nil(t, r.data()["loss"])
	for _, c := range []struct{ method, path, body string }{
		{http.MethodPut, "/api/v1/loss/followup", `{"bleeding_stopped":true}`},
		{http.MethodPost, "/api/v1/loss/moods", `{"mood":"sad"}`},
		{http.MethodGet, "/api/v1/loss/note", ``},
		{http.MethodPut, "/api/v1/loss/note", `{"note":"x"}`},
		{http.MethodPut, "/api/v1/loss/next-step", `{"choice":"cycle"}`},
	} {
		r := e.do(t, c.method, c.path, ninaTok, c.body)
		assert.Equal(t, http.StatusNotFound, r.status, c.path+" "+r.raw)
		assert.Equal(t, "loss_not_found", r.body["error_code"], c.path)
	}
	assert.EqualValues(t, 0, e.scalar(t, `SELECT COUNT(*) FROM pregnancy_loss_moods`))
	assert.Nil(t, e.scalar(t, `SELECT bleeding_stopped_on FROM pregnancy_losses WHERE user_id = ?`, sara))

	// Nina records her own loss; Sara's note copied into Nina's row does not decrypt there (bound to user + row).
	require.Equal(t, http.StatusCreated, e.do(t, http.MethodPost, "/api/v1/loss", ninaTok, `{}`).status)
	_, err := e.db.Exec(`UPDATE pregnancy_losses n JOIN pregnancy_losses s ON s.user_id = ? SET n.private_note = s.private_note WHERE n.user_id <> ?`, sara, sara)
	require.NoError(t, err)
	r = e.do(t, http.MethodGet, "/api/v1/loss/note", ninaTok, "")
	assert.Equal(t, http.StatusConflict, r.status, r.raw)
	assert.Equal(t, "note_unreadable", r.body["error_code"])
	assert.NotContains(t, r.raw, "mine")
	assert.Equal(t, "mine", e.do(t, http.MethodGet, "/api/v1/loss/note", saraTok, "").data()["note"])
	// An unreadable note can be deleted; Nina's DELETEs never touch Sara's rows.
	require.Equal(t, http.StatusOK, e.do(t, http.MethodDelete, "/api/v1/loss/note", ninaTok, "").status)
	assert.Nil(t, e.do(t, http.MethodGet, "/api/v1/loss/note", ninaTok, "").data()["note"])
	require.Equal(t, http.StatusOK, e.do(t, http.MethodDelete, "/api/v1/loss", ninaTok, "").status)
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodDelete, "/api/v1/loss", ninaTok, "").status)
	assert.Equal(t, "mine", e.do(t, http.MethodGet, "/api/v1/loss/note", saraTok, "").data()["note"])
	assert.EqualValues(t, 1, e.scalar(t, `SELECT COUNT(*) FROM pregnancy_losses WHERE user_id = ?`, sara))

	for _, p := range []string{"/api/v1/loss", "/api/v1/loss/note"} {
		r := e.do(t, http.MethodGet, p, "", "")
		assert.Equal(t, http.StatusUnauthorized, r.status)
		assert.Equal(t, "unauthenticated", r.body["error_code"])
	}
}

func TestLoss_NotesDisabledInProduction(t *testing.T) {
	e := setup(t, true)
	_, tok := e.user(t, "09120004005")
	require.Equal(t, http.StatusCreated, e.do(t, http.MethodPost, "/api/v1/loss", tok, `{}`).status)
	for _, c := range []struct{ method, body string }{{http.MethodGet, ""}, {http.MethodPut, `{"note":"x"}`}} {
		r := e.do(t, c.method, "/api/v1/loss/note", tok, c.body)
		assert.Equal(t, http.StatusServiceUnavailable, r.status, r.raw)
		assert.Equal(t, "note_unavailable", r.body["error_code"])
	}
	assert.Equal(t, http.StatusOK, e.do(t, http.MethodDelete, "/api/v1/loss/note", tok, "").status, "erasing needs no key")
}

// DELETE /loss erases the newest record with its moods and note and never resurrects pregnancy content; a new loss
// clears the earlier record's note; concurrent first POSTs record one loss.
func TestLoss_EraseAndConcurrency(t *testing.T) {
	e := setup(t, false)
	uid, tok := e.user(t, "09120004007")
	e.pregnant(t, uid)

	// Two simultaneous first POSTs (double tap): one loss.
	done := make(chan int, 2)
	for range 2 {
		go func() { done <- e.do(t, http.MethodPost, "/api/v1/loss", tok, `{"type":"early_miscarriage"}`).status }()
	}
	got := []int{<-done, <-done}
	assert.ElementsMatch(t, []int{http.StatusCreated, http.StatusOK}, got)
	assert.EqualValues(t, 1, e.scalar(t, `SELECT COUNT(*) FROM pregnancy_losses WHERE user_id = ?`, uid))

	require.Equal(t, http.StatusOK, e.do(t, http.MethodPut, "/api/v1/loss/note", tok, `{"note":"first"}`).status)
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPut, "/api/v1/loss/followup", tok, `{"visit_at":"2026-10-07 11:30"}`).status)
	assert.EqualValues(t, 1, e.scalar(t, `SELECT JSON_EXTRACT(meta, '$.private') = true FROM reminders WHERE user_id = ? AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.topic')) = 'checkup'`, uid))
	assert.Equal(t, "Doctor visit", e.scalar(t, `SELECT title FROM reminders WHERE user_id = ? AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.topic')) = 'checkup'`, uid), "neutral title")

	// A second loss on another day: the first record's note is cleared.
	r := e.do(t, http.MethodPost, "/api/v1/loss", tok, `{}`, "2026-09-30T10:00:00+03:30")
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.EqualValues(t, 0, e.scalar(t, `SELECT COUNT(*) FROM pregnancy_losses WHERE user_id = ? AND private_note IS NOT NULL`, uid))
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPost, "/api/v1/loss/moods", tok, `{"mood":"sad"}`, "2026-09-30T10:00:00+03:30").status)

	// Erase the newest: its moods go, the earlier record becomes the newest, pregnancy stays stopped.
	r = e.do(t, http.MethodDelete, "/api/v1/loss", tok, "", "2026-09-30T10:00:00+03:30")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.EqualValues(t, 1, r.data()["losses_count"])
	assert.Equal(t, "early_miscarriage", r.obj("loss")["type"])
	assert.EqualValues(t, 0, e.scalar(t, `SELECT COUNT(*) FROM pregnancy_loss_moods WHERE user_id = ?`, uid))
	require.Equal(t, http.StatusOK, e.do(t, http.MethodDelete, "/api/v1/loss", tok, "").status)
	assert.Nil(t, e.do(t, http.MethodGet, "/api/v1/loss", tok, "").data()["loss"])
	assert.EqualValues(t, 0, e.scalar(t, `SELECT pregnancy_mode FROM pregnancy_profiles WHERE user_id = ?`, uid))
	assert.EqualValues(t, 0, e.scalar(t, `SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ? AND is_dismissed = 0`, uid))
	assert.EqualValues(t, 0, e.scalar(t, `SELECT is_active FROM reminders WHERE user_id = ? AND title = 'Folic acid'`, uid))
	assert.Equal(t, "cycle", e.scalar(t, `SELECT life_mode FROM user_life_profiles WHERE user_id = ?`, uid))
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodDelete, "/api/v1/loss/note", tok, "").status)
	r = e.do(t, http.MethodDelete, "/api/v1/loss", "", "")
	assert.Equal(t, http.StatusUnauthorized, r.status)
}

func TestLoss_RecordWithoutPregnancyKeepsMode(t *testing.T) {
	e := setup(t, false)
	uid, tok := e.user(t, "09120004006")
	_, err := e.db.Exec(`INSERT INTO user_life_profiles (user_id, life_mode, ivf_iui, track_contraception, created_at, updated_at)
		VALUES (?, 'ttc', 0, 0, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)
	require.NoError(t, err)
	r := e.do(t, http.MethodPost, "/api/v1/loss", tok, `{"type":"chemical","occurred_on":"2026-09-24"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, "a future date is refused: "+r.raw)
	r = e.do(t, http.MethodPost, "/api/v1/loss", tok, `{"type":"chemical"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, "ttc", e.scalar(t, `SELECT life_mode FROM user_life_profiles WHERE user_id = ?`, uid), "no active pregnancy: the mode is hers to choose")
	assert.Nil(t, r.obj("loss")["occurred_on"])
}
