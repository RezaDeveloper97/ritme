package assistant_test

// B-N7-06 end to end against MariaDB with the fake AI provider: departments, profile sharing, SSE chat with the
// server-side emergency check (115 card, no model call, no consent / quota), diagnosis guard, triage + chips parsed
// from the control lines, free quota vs Plus, photos, summary, handoff with the booking's consent, IDOR and hard delete.

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/ai/access"
	"github.com/ritme/backend-go/internal/ai/usage"
	"github.com/ritme/backend-go/internal/assistant"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/loss"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/telemed"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee7"
	testNow  = "2026-10-08T10:00:00+03:30"
)

type spyChat struct {
	*ai.Fake
	mu  sync.Mutex
	got []ai.ChatRequest
	// reply, when set, is streamed rune by rune instead of the fake fixture (B-N7-06b); block holds the stream
	// before its first delta until closed, started is signalled when a call begins.
	reply   string
	block   chan struct{}
	started chan struct{}
}

func (s *spyChat) ChatStream(ctx context.Context, req ai.ChatRequest) (<-chan ai.ChatChunk, error) {
	s.mu.Lock()
	s.got = append(s.got, req)
	reply, block, started := s.reply, s.block, s.started
	s.mu.Unlock()
	if started != nil {
		started <- struct{}{}
	}
	if reply == "" && block == nil {
		return s.Fake.ChatStream(ctx, req)
	}
	if reply == "" {
		reply = "ok"
	}
	out := make(chan ai.ChatChunk)
	go func() {
		defer close(out)
		if block != nil {
			select {
			case <-block:
			case <-ctx.Done():
				return
			}
		}
		for _, r := range reply {
			select {
			case out <- ai.ChatChunk{Delta: string(r)}:
			case <-ctx.Done():
				return
			}
		}
		select {
		case out <- ai.ChatChunk{Done: true, Finish: ai.FinishStop}:
		case <-ctx.Done():
		}
	}()
	return out, nil
}

func (s *spyChat) set(f func(s *spyChat)) { s.mu.Lock(); defer s.mu.Unlock(); f(s) }

func (s *spyChat) calls() int { s.mu.Lock(); defer s.mu.Unlock(); return len(s.got) }

func (s *spyChat) last() ai.ChatRequest { s.mu.Lock(); defer s.mu.Unlock(); return s.got[len(s.got)-1] }

type env struct {
	db       *sql.DB
	app      *fiber.App
	iss      *passport.Issuer
	spy      *spyChat
	svc      *assistant.Service
	consents *consent.Service
}

func setup(t *testing.T) *env { t.Helper(); return setupWith(t, 0, nil) }

// setupWith: maxConcurrent overrides the gate's per-user in-flight cap (0 = default); limiter adds the feature
// throttles (nil = none).
func setupWith(t *testing.T, maxConcurrent int, limiter *ratelimit.Limiter) *env {
	t.Helper()
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))

	plusSvc := plus.NewService(db, config.Plus{TrialDays: 7}, nil, quiet)
	gate := plus.NewGate(plusSvc, clock.Real{})
	budget := usage.NewBudget(db, 5, clock.Real{}, quiet)
	spy := &spyChat{Fake: ai.NewFake()}
	client := ai.NewClientWith(ai.Options{Provider: "fake", Chatter: spy, Recorder: usage.NewRecorder(db, clock.Real{}, quiet), Limiter: budget})
	consents := consent.NewService(db)
	g := access.NewGuard(access.Options{Client: client, Consents: consents, Budget: budget, Gate: gate, Plus: plusSvc, Logger: quiet,
		MaxConcurrent: maxConcurrent, Limiter: limiter})
	box, err := loss.NewNoteBox(nil, nil, false)
	require.NoError(t, err)

	vault, err := files.NewVault(t.TempDir(), files.DevKey("assistant-test"), nil, false)
	require.NoError(t, err)
	bookings := telemed.NewBookings(db, catalog.NewReader(catalogstore.New(db), nil, 0, quiet), telemed.BookingOptions{Clock: clock.Real{}, Logger: quiet})
	svc := assistant.NewService(assistant.Options{DB: db, AI: client, Consents: consents,
		Profile: assistant.RecordProfile(healthrecord.NewService(db, nil)), Photos: files.NewService(files.Options{DB: db, Vault: vault}),
		Bookings: bookings, BookingNotFound: telemed.ErrBookingNotFound, Logger: quiet, Box: box})
	h := assistant.NewHandlers(svc, plusSvc, clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	p := "/api/v1/assistant"
	app.Get(p, locale, guard, h.Hub)
	app.Get(p+"/departments/:code", locale, guard, h.ShowDepartment)
	app.Get(p+"/profile", locale, guard, h.ShowProfile)
	app.Put(p+"/profile", locale, guard, h.UpdateProfile)
	app.Post(p+"/photos", locale, guard, h.StorePhoto)
	app.Get(p+"/conversations", locale, guard, h.Index)
	app.Post(p+"/conversations", locale, guard, h.Store)
	app.Delete(p+"/conversations", locale, guard, h.DestroyAll)
	app.Get(p+"/conversations/:id", locale, guard, h.Show)
	app.Delete(p+"/conversations/:id", locale, guard, h.Destroy)
	app.Post(p+"/conversations/:id/messages", locale, append(append([]any{guard, h.PrepareMessage}, g.Chain(ai.FeatureAssistant)...), h.StreamMessage)...)
	app.Post(p+"/conversations/:id/summary", locale, append(append([]any{guard}, g.Gate(ai.FeatureAssistant)...), h.Summary)...)
	app.Post(p+"/conversations/:id/handoff", locale, guard, h.Handoff)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), spy: spy, svc: svc, consents: consents}
}

func (e *env) user(t *testing.T, mobile string, plusTrial bool) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara Ahmadi', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	_, err = e.db.Exec(`INSERT INTO user_profiles (user_id, birthday, cycle_duration, period_duration, created_at, updated_at)
		VALUES (?, '1992-04-12', 29, 5, NOW(), NOW())`, id)
	require.NoError(t, err)
	if plusTrial {
		_, err = e.db.Exec(`INSERT INTO plus_trials (user_id, started_at, ends_at, created_at, updated_at)
			VALUES (?, '2026-10-01 10:00:00', '2026-10-30 10:00:00', NOW(), NOW())`, id)
		require.NoError(t, err)
	}
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // positive id
	require.NoError(t, err)
	return uint64(id), tok.AccessToken //nolint:gosec // positive id
}

func (e *env) accept(t *testing.T, uid uint64, code string) {
	t.Helper()
	require.NoError(t, e.consents.Accept(context.Background(), uid, code, 1, time.Now()))
}

type resp struct {
	status int
	body   map[string]any
	raw    string
}

func (r resp) data() map[string]any { d, _ := r.body["data"].(map[string]any); return d }

func (e *env) req(t *testing.T, method, path, token, ctype string, body io.Reader) resp {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "fa")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set(clock.Header, testNow)
	res, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	raw, _ := io.ReadAll(res.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return resp{status: res.StatusCode, body: m, raw: string(raw)}
}

func (e *env) do(t *testing.T, method, path, token string, body any) resp {
	t.Helper()
	var r io.Reader
	ctype := ""
	if body != nil {
		raw, _ := json.Marshal(body)
		r, ctype = bytes.NewReader(raw), "application/json"
	}
	return e.req(t, method, path, token, ctype, r)
}

type event struct {
	name string
	data map[string]any
}

func events(raw string) []event {
	var out []event
	sc := bufio.NewScanner(strings.NewReader(raw))
	var cur event
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			cur.name = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			_ = json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &cur.data)
		case line == "" && cur.name != "":
			out = append(out, cur)
			cur = event{}
		}
	}
	return out
}

func find(evs []event, name string) map[string]any {
	for _, e := range evs {
		if e.name == name {
			return e.data
		}
	}
	return nil
}

func (e *env) newConv(t *testing.T, tok, dept string) uint64 {
	t.Helper()
	r := e.do(t, "POST", "/api/v1/assistant/conversations", tok, map[string]any{"department": dept})
	require.Equal(t, 201, r.status, r.raw)
	return uint64(r.data()["conversation"].(map[string]any)["id"].(float64))
}

func (e *env) send(t *testing.T, tok string, conv uint64, body map[string]any) resp {
	t.Helper()
	return e.do(t, "POST", fmt.Sprintf("/api/v1/assistant/conversations/%d/messages", conv), tok, body)
}

func (e *env) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(q, args...).Scan(&n))
	return n
}

func TestAssistant_HubDepartmentsAndAuth(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001", false)
	assert.Equal(t, 401, e.do(t, "GET", "/api/v1/assistant", "", nil).status)
	assert.Equal(t, 401, e.send(t, "", 1, map[string]any{"text": "hi"}).status)

	r := e.do(t, "GET", "/api/v1/assistant", tok, nil)
	require.Equal(t, 200, r.status, r.raw)
	depts := r.data()["departments"].([]any)
	require.Len(t, depts, 6)
	assert.Equal(t, "general", depts[0].(map[string]any)["code"])
	assert.Equal(t, "مشاور پزشکی عمومی", depts[0].(map[string]any)["name"])
	assert.NotContains(t, r.raw, "system_prompt")
	assert.NotContains(t, r.raw, "You are")
	q := r.data()["quota"].(map[string]any)
	assert.InDelta(t, 5, q["limit"], 0)
	assert.Equal(t, false, q["unlimited"])

	r = e.do(t, "GET", "/api/v1/assistant/departments/skin_beauty", tok, nil)
	require.Equal(t, 200, r.status, r.raw)
	d := r.data()["department"].(map[string]any)
	assert.NotEmpty(t, d["does_not_do"])
	assert.NotEmpty(t, d["quick_replies"])
	assert.Equal(t, 404, e.do(t, "GET", "/api/v1/assistant/departments/nope", tok, nil).status)
	assert.Equal(t, 422, e.do(t, "POST", "/api/v1/assistant/conversations", tok, map[string]any{}).status)
	assert.Equal(t, 404, e.do(t, "POST", "/api/v1/assistant/conversations", tok, map[string]any{"department": "nope"}).status)
}

func TestAssistant_EmergencyBeforeModelAndConsent(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001", false)
	conv := e.newConv(t, tok, "gynecology")

	// No ai_assistant consent: a normal message is refused, an emergency one is answered with the 115 card.
	r := e.send(t, tok, conv, map[string]any{"text": "سلام"})
	assert.Equal(t, 403, r.status, r.raw)
	assert.Equal(t, "consent_required", r.body["error_code"])

	r = e.send(t, tok, conv, map[string]any{"text": "خونریزی شدید دارم و سرم گیج می\u200cرود"})
	require.Equal(t, 200, r.status, r.raw)
	evs := events(r.raw)
	card := find(evs, "card")
	require.NotNil(t, card, r.raw)
	msg := card["message"].(map[string]any)
	assert.Equal(t, "115", msg["card"].(map[string]any)["phone"])
	assert.Equal(t, "emergency", find(evs, "done")["triage_level"])
	assert.Equal(t, 0, e.spy.calls(), "the model is never called for an emergency")
	assert.Equal(t, 1, e.count(t, `SELECT emergency FROM assistant_conversations WHERE id = ?`, conv))

	e.accept(t, uid, consent.AIAssistant)
	r = e.send(t, tok, conv, map[string]any{"text": "RITME-FAKE:triage درد دارم"})
	require.Equal(t, 200, r.status, r.raw)
	evs = events(r.raw)
	var text strings.Builder
	for _, ev := range evs {
		if ev.name == "delta" {
			text.WriteString(ev.data["text"].(string))
		}
	}
	assert.NotContains(t, text.String(), "[[", "control lines never reach the client")
	done := find(evs, "done")
	require.NotNil(t, done, r.raw)
	m := done["message"].(map[string]any)
	assert.Equal(t, "soon", m["triage_level"])
	assert.Equal(t, []any{"تب دارم", "درد بیشتر شد", "ممنون"}, m["chips"])
	assert.NotContains(t, m["text"], "[[")
	assert.Contains(t, e.spy.last().System, "You are a health assistant in Ritme")
	assert.Contains(t, e.spy.last().System, "gynecology advisor")
}

func TestAssistant_DiagnosisGuardAndProfileFence(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001", false)
	e.accept(t, uid, consent.AIAssistant)
	conv := e.newConv(t, tok, "general")

	r := e.send(t, tok, conv, map[string]any{"text": "لطفاً تشخیص بده چه بیماری دارم"})
	require.Equal(t, 200, r.status, r.raw)
	done := find(events(r.raw), "done")
	require.NotNil(t, done, r.raw)
	assert.Equal(t, "diagnosis", done["message"].(map[string]any)["guardrail"])
	sys := e.spy.last().System
	assert.Contains(t, sys, "asks for a diagnosis or a medicine")
	assert.NotContains(t, sys, "Her profile", "no assistant_profile consent → nothing shared")

	r = e.do(t, "PUT", "/api/v1/assistant/profile", tok, map[string]any{"shared": []string{"age", "nope"}})
	assert.Equal(t, 422, r.status)
	r = e.do(t, "PUT", "/api/v1/assistant/profile", tok, map[string]any{"shared": []string{"cycle", "age"}})
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, []any{"age", "cycle"}, r.data()["profile"].(map[string]any)["shared"])

	e.accept(t, uid, consent.AssistantProfile)
	r = e.send(t, tok, conv, map[string]any{"text": "سردرد دارم"})
	require.Equal(t, 200, r.status, r.raw)
	sys = e.spy.last().System
	assert.Contains(t, sys, "Her profile")
	assert.Contains(t, sys, "```json")
	assert.Contains(t, sys, `"age": 34`)
	assert.NotContains(t, sys, "Sara")
	// History is rebuilt server-side: the earlier turns come from the DB.
	assert.GreaterOrEqual(t, len(e.spy.last().Messages), 3)
}

func TestAssistant_FreeQuotaAndPlusUnlimited(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001", false)
	e.accept(t, uid, consent.AIAssistant)
	conv := e.newConv(t, tok, "nutrition")
	for i := range 5 {
		r := e.send(t, tok, conv, map[string]any{"text": fmt.Sprintf("پرسش %d", i)})
		require.Equal(t, 200, r.status, r.raw)
	}
	r := e.send(t, tok, conv, map[string]any{"text": "پرسش ششم"})
	assert.Equal(t, 402, r.status, r.raw)
	assert.Equal(t, plus.CodePlusRequired, r.body["error_code"])
	assert.Equal(t, "quota", r.body["reason"])
	r = e.send(t, tok, conv, map[string]any{"text": "می\u200cخواهم خودکشی کنم"})
	require.Equal(t, 200, r.status, "an emergency is never blocked by the quota")
	assert.NotNil(t, find(events(r.raw), "card"))

	pid, ptok := e.user(t, "09120000002", true)
	e.accept(t, pid, consent.AIAssistant)
	pconv := e.newConv(t, ptok, "nutrition")
	for i := range 7 {
		r := e.send(t, ptok, pconv, map[string]any{"text": fmt.Sprintf("q %d", i)})
		require.Equal(t, 200, r.status, r.raw)
	}
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM ai_usage_logs WHERE user_id = ? AND feature <> 'assistant'`, pid))
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for x := range 8 {
		img.Set(x, x, color.RGBA{R: 200, A: 255})
	}
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, img))
	return b.Bytes()
}

func (e *env) upload(t *testing.T, tok string, data []byte) resp {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, err := w.CreateFormFile("photo", "x.png")
	require.NoError(t, err)
	_, _ = part.Write(data)
	require.NoError(t, w.Close())
	return e.req(t, "POST", "/api/v1/assistant/photos", tok, w.FormDataContentType(), &b)
}

func TestAssistant_IDORPhotosSummaryHandoffDelete(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001", false)
	oid, other := e.user(t, "09120000002", false)
	e.accept(t, uid, consent.AIAssistant)
	e.accept(t, oid, consent.AIAssistant)
	conv := e.newConv(t, tok, "skin_beauty")

	up := e.upload(t, tok, pngBytes(t))
	require.Equal(t, 201, up.status, up.raw)
	photo := up.data()["photo"].(map[string]any)["id"]
	assert.Equal(t, 422, e.upload(t, tok, []byte("not an image at all")).status)

	// Another user cannot read, write, summarise, hand off or delete — nor use the photo.
	for _, r := range []resp{
		e.do(t, "GET", fmt.Sprintf("/api/v1/assistant/conversations/%d", conv), other, nil),
		e.send(t, other, conv, map[string]any{"text": "hi"}),
		e.do(t, "POST", fmt.Sprintf("/api/v1/assistant/conversations/%d/summary", conv), other, nil),
		e.do(t, "POST", fmt.Sprintf("/api/v1/assistant/conversations/%d/handoff", conv), other, map[string]any{"booking_id": 1}),
		e.do(t, "DELETE", fmt.Sprintf("/api/v1/assistant/conversations/%d", conv), other, nil),
	} {
		assert.Equal(t, 404, r.status, r.raw)
	}
	oconv := e.newConv(t, other, "skin_beauty")
	r := e.send(t, other, oconv, map[string]any{"text": "look", "photo_id": photo})
	assert.Equal(t, 422, r.status, "a foreign photo id is refused")

	r = e.send(t, tok, conv, map[string]any{"text": "لک روی گونه", "photo_id": photo})
	require.Equal(t, 200, r.status, r.raw)
	require.Len(t, e.spy.last().Messages[len(e.spy.last().Messages)-1].Images, 1)

	// Summary.
	r = e.do(t, "POST", fmt.Sprintf("/api/v1/assistant/conversations/%d/summary", conv), tok, nil)
	require.Equal(t, 200, r.status, r.raw)
	assert.NotEmpty(t, r.data()["summary"].(map[string]any)["text"])
	assert.InDelta(t, 1, r.data()["photos_count"], 0)

	// Handoff needs the booking's assistant_summaries consent.
	_, err := e.db.Exec(`INSERT INTO telemed_doctors (id, kind, name, specialty, licence_no, created_at, updated_at)
		VALUES (900, 'doctor', '{"fa":"دکتر"}', 'dermatology', 'L-1', NOW(), NOW())`)
	require.NoError(t, err)
	booking := func(id, user uint64) {
		_, err := e.db.Exec(`INSERT INTO telemed_bookings (id, reference, user_id, doctor_id, mode, duration_minutes, starts_at, ends_at,
			status, for_whom, price_rials, total_rials, created_at, updated_at)
			VALUES (?, ?, ?, 900, 'video', 20, '2030-01-08 18:00:00', '2030-01-08 18:20:00', 'confirmed', 'self', 0, 0, NOW(), NOW())`,
			id, fmt.Sprintf("TB%d", id), user)
		require.NoError(t, err)
	}
	booking(501, uid)
	booking(502, oid)
	path := fmt.Sprintf("/api/v1/assistant/conversations/%d/handoff", conv)
	r = e.do(t, "POST", path, tok, map[string]any{"booking_id": 501})
	assert.Equal(t, 403, r.status, r.raw)
	assert.Equal(t, "share_consent_required", r.body["error_code"])
	assert.Equal(t, 404, e.do(t, "POST", path, tok, map[string]any{"booking_id": 502}).status, "another user's booking")
	assert.Equal(t, 422, e.do(t, "POST", path, tok, map[string]any{}).status)
	_, err = e.db.Exec(`INSERT INTO telemed_booking_consents (booking_id, scope, granted_at, created_at, updated_at)
		VALUES (501, 'assistant_summaries', NOW(), NOW(), NOW())`)
	require.NoError(t, err)
	r = e.do(t, "POST", path, tok, map[string]any{"booking_id": 501})
	require.Equal(t, 200, r.status, r.raw)
	at := time.Date(2030, 1, 8, 18, 0, 0, 0, time.Local)
	shared, err := e.svc.SharedSummaries(context.Background(), 900, 501, at)
	require.NoError(t, err)
	require.Len(t, shared, 1)
	assert.Equal(t, r0summary(t, e, conv), shared[0].Summary, "the doctor reads the plaintext summary")
	var rawHandoff string
	require.NoError(t, e.db.QueryRow(`SELECT summary FROM assistant_handoffs WHERE booking_id = 501`).Scan(&rawHandoff))
	assert.True(t, strings.HasPrefix(rawHandoff, "v1:"), "the handoff summary is sealed at rest")
	_, err = e.svc.SharedSummaries(context.Background(), 901, 501, at)
	require.ErrorIs(t, err, assistant.ErrShareConsent, "another doctor")

	// Hard delete: rows, handoffs and the photo are gone.
	r = e.do(t, "DELETE", fmt.Sprintf("/api/v1/assistant/conversations/%d", conv), tok, nil)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM assistant_messages WHERE conversation_id = ?`, conv))
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM assistant_handoffs WHERE conversation_id = ?`, conv))
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM files WHERE id = ?`, photo))
	assert.Equal(t, 404, e.do(t, "GET", fmt.Sprintf("/api/v1/assistant/conversations/%d", conv), tok, nil).status)

	r = e.do(t, "DELETE", "/api/v1/assistant/conversations", other, nil)
	require.Equal(t, 200, r.status)
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM assistant_conversations WHERE user_id = ?`, oid))
}
