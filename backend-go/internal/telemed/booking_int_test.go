package telemed_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/payments"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/telemed"
)

// mclock is a movable clock.
type mclock struct {
	mu sync.Mutex
	t  time.Time
}

func (m *mclock) Now() time.Time  { m.mu.Lock(); defer m.mu.Unlock(); return m.t }
func (m *mclock) set(t time.Time) { m.mu.Lock(); m.t = t; m.mu.Unlock() }

type benv struct {
	*env
	bk    *telemed.Bookings
	fake  *payments.Fake
	clk   *mclock
	plus  map[uint64]bool
	plMu  sync.Mutex
	a     uint64
	other uint64
}

const returnURL = "https://web.ritme.test/services/bookings/return"

func setupBooking(t *testing.T, withGateway bool) *benv {
	t.Helper()
	e := admintest.New(t)
	e.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(e.DB)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, admintest.Quiet).RequireUser
	locale := i18n.Middleware(e.Registry)
	reader := catalog.NewReader(catalogstore.New(e.DB), nil, 0, admintest.Quiet)
	be := &benv{clk: &mclock{t: fixed}, plus: map[uint64]bool{}}
	opts := telemed.BookingOptions{
		Config: config.Telemed{HoldTTL: 10 * time.Minute, CallbackURL: returnURL, PlusDiscountPercent: 20},
		Clock:  be.clk, Logger: admintest.Quiet,
		Discount: func(_ context.Context, userID uint64, _ time.Time) (bool, error) {
			be.plMu.Lock()
			defer be.plMu.Unlock()
			return be.plus[userID], nil
		},
	}
	if withGateway {
		be.fake = payments.NewFake(payments.NewMemoryFakeStore(), appURL)
		opts.Gateway = payments.NewGateway(be.fake, appURL, payments.NewAllowList([]string{returnURL}), admintest.Quiet)
	}
	be.bk = telemed.NewBookings(e.DB, reader, opts)
	h := telemed.NewHandlers(be.bk.Directory(), be.clk, appURL)
	bh := telemed.NewBookingHandlers(be.bk, be.clk, appURL)
	p := "/api/v1/telemed/doctors"
	e.App.Get(p+"/:id", locale, guard, h.Show)
	e.App.Get(p+"/:id/slots", locale, guard, h.Slots)
	e.App.Post(p+"/:id/reviews", locale, guard, h.StoreReview)
	b := "/api/v1/telemed/bookings"
	e.App.Get(b, locale, guard, bh.Index)
	e.App.Get(b+"/quote", locale, guard, bh.Quote)
	e.App.Post(b, locale, guard, bh.Store)
	e.App.Post(b+"/verify", locale, guard, bh.Verify)
	e.App.Get(b+"/:id", locale, guard, bh.Show)
	e.App.Post(b+"/:id/cancel", locale, guard, bh.Cancel)
	e.App.Post(b+"/:id/reschedule", locale, guard, bh.Reschedule)
	e.App.Put(b+"/:id/consent", locale, guard, bh.UpdateConsent)
	e.App.Delete(b+"/:id/consent", locale, guard, bh.DestroyConsent)
	telemed.NewAdmin(be.bk.Directory(), media.NewDisk(e.Storage), media.DefaultOptimizer, appURL, admintest.Quiet).Routes(e.Route(), e.Kit)
	be.env = &env{Env: e, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
	be.a, be.other, _, _ = be.seed(t)
	return be
}

// book posts a self booking of doctor A's video slot at hh:mm on the fixed Tuesday.
func (be *benv) book(t *testing.T, tok, at string, extra map[string]any) response {
	t.Helper()
	body := map[string]any{"doctor_id": be.a, "mode": "video", "starts_at": "2030-01-08 " + at, "for_whom": "self"}
	for k, v := range extra {
		body[k] = v
	}
	return be.do(t, http.MethodPost, "/api/v1/telemed/bookings", tok, "fa", body)
}

// pay books, pays on the fake page and verifies; returns the booking id.
func (be *benv) pay(t *testing.T, tok, at string, extra map[string]any) uint64 {
	t.Helper()
	r := be.book(t, tok, at, extra)
	require.Equal(t, http.StatusCreated, r.status, r.body)
	bk, pay := r.obj("booking"), r.obj("payment")
	_, err := be.fake.Decide(context.Background(), pay["authority"].(string), true)
	require.NoError(t, err)
	v := be.do(t, http.MethodPost, "/api/v1/telemed/bookings/verify", tok, "fa",
		map[string]any{"reference": bk["reference"], "authority": pay["authority"]})
	require.Equal(t, http.StatusOK, v.status, v.body)
	assert.Equal(t, "confirmed", v.obj("booking")["status"])
	return id(bk["id"])
}

func TestBooking_Unauthenticated(t *testing.T) {
	be := setupBooking(t, true)
	for _, tc := range []struct{ m, p string }{
		{http.MethodGet, "/api/v1/telemed/bookings"}, {http.MethodGet, "/api/v1/telemed/bookings/quote"},
		{http.MethodPost, "/api/v1/telemed/bookings"}, {http.MethodPost, "/api/v1/telemed/bookings/verify"},
		{http.MethodGet, "/api/v1/telemed/bookings/1"}, {http.MethodPost, "/api/v1/telemed/bookings/1/cancel"},
		{http.MethodPost, "/api/v1/telemed/bookings/1/reschedule"}, {http.MethodPut, "/api/v1/telemed/bookings/1/consent"},
		{http.MethodDelete, "/api/v1/telemed/bookings/1/consent"},
	} {
		r := be.do(t, tc.m, tc.p, "", "", nil)
		assert.Equal(t, http.StatusUnauthorized, r.status, tc.p)
		assert.Equal(t, "unauthenticated", r.body["error_code"], tc.p)
	}
}

func TestBooking_HoldPayConfirmAndReplay(t *testing.T) {
	be := setupBooking(t, true)
	uid, tok := be.user(t, "Nava", "09120000001")

	q := be.do(t, http.MethodGet, fmt.Sprintf("/api/v1/telemed/bookings/quote?doctor_id=%d&mode=video", be.a), tok, "fa", nil)
	require.Equal(t, http.StatusOK, q.status, q.body)
	assert.InDelta(t, 2_000_000, q.obj("price")["total_rials"], 0)
	assert.Equal(t, false, q.data()["plus"])

	r := be.book(t, tok, "16:00", map[string]any{"share": []any{"cycle_summary"}, "reason": "ttc", "note": "سلام"})
	require.Equal(t, http.StatusCreated, r.status, r.body)
	bk := r.obj("booking")
	assert.Equal(t, "held", bk["status"])
	assert.Equal(t, "pending", bk["payment_status"])
	assert.Equal(t, map[string]any{"cycle_summary": true, "bbt_lh": false, "assistant_summaries": false}, bk["consent"])
	pay := r.obj("payment")
	require.NotEmpty(t, pay["redirect_url"])

	// The held slot is busy for everyone.
	s := be.do(t, http.MethodGet, fmt.Sprintf("/api/v1/telemed/doctors/%d/slots?mode=video&days=1", be.a), tok, "fa", nil)
	first := s.data()["days"].([]any)[0].(map[string]any)["slots"].([]any)[0].(map[string]any)
	assert.Equal(t, "16:30", first["time"])

	// Verify before paying: declined → nothing confirmed. (Use a second booking for that below.)
	_, err := be.fake.Decide(context.Background(), pay["authority"].(string), true)
	require.NoError(t, err)
	wrong := be.do(t, http.MethodPost, "/api/v1/telemed/bookings/verify", tok, "fa", map[string]any{"reference": bk["reference"], "authority": "FAKE-OTHER"})
	assert.Equal(t, http.StatusUnprocessableEntity, wrong.status)
	assert.Equal(t, "authority_mismatch", wrong.body["error_code"])
	body := map[string]any{"reference": bk["reference"], "authority": pay["authority"]}
	v := be.do(t, http.MethodPost, "/api/v1/telemed/bookings/verify", tok, "fa", body)
	require.Equal(t, http.StatusOK, v.status, v.body)
	assert.Equal(t, "confirmed", v.obj("booking")["status"])
	assert.Equal(t, false, v.data()["already_paid"])
	// Replay: idempotent, one appointment.
	v = be.do(t, http.MethodPost, "/api/v1/telemed/bookings/verify", tok, "fa", body)
	require.Equal(t, http.StatusOK, v.status)
	assert.Equal(t, true, v.data()["already_paid"])
	assert.Equal(t, 1, be.Int(`SELECT COUNT(*) FROM reminders WHERE user_id = ? AND type = 'appointment' AND scheduled_at = '2030-01-08 16:00:00'`, uid))
	assert.NotNil(t, v.obj("booking")["appointment_id"])

	next, err := be.bk.NextBooking(context.Background(), uid, fixed)
	require.NoError(t, err)
	require.NotNil(t, next)
	assert.Equal(t, "/services/bookings/"+fmt.Sprint(id(bk["id"])), next.Href)

	// Declined payment closes the hold and frees the slot.
	r = be.book(t, tok, "18:00", nil)
	require.Equal(t, http.StatusCreated, r.status, r.body)
	_, err = be.fake.Decide(context.Background(), r.obj("payment")["authority"].(string), false)
	require.NoError(t, err)
	v = be.do(t, http.MethodPost, "/api/v1/telemed/bookings/verify", tok, "fa",
		map[string]any{"reference": r.obj("booking")["reference"], "authority": r.obj("payment")["authority"]})
	assert.Equal(t, http.StatusUnprocessableEntity, v.status)
	assert.Equal(t, "payment_failed", v.body["error_code"])
	_, tok2 := be.user(t, "Sara", "09120000002")
	assert.Equal(t, http.StatusCreated, be.book(t, tok2, "18:00", nil).status, "a failed payment frees the slot")
}

func TestBooking_PlusDiscountAndAmountTamper(t *testing.T) {
	be := setupBooking(t, true)
	uid, tok := be.user(t, "Nava", "09120000001")
	be.plus[uid] = true
	r := be.book(t, tok, "16:00", nil)
	require.Equal(t, http.StatusCreated, r.status, r.body)
	assert.Equal(t, map[string]any{"price_rials": float64(2_000_000), "discount_rials": float64(400_000),
		"discount_source": "plus", "total_rials": float64(1_600_000), "currency": "IRR"}, r.obj("booking")["price"])
	// The stored total is what is verified: tampering with it is an amount mismatch, never a confirmation.
	be.Exec(`UPDATE telemed_bookings SET total_rials = 1 WHERE id = ?`, id(r.obj("booking")["id"]))
	_, err := be.fake.Decide(context.Background(), r.obj("payment")["authority"].(string), true)
	require.NoError(t, err)
	v := be.do(t, http.MethodPost, "/api/v1/telemed/bookings/verify", tok, "fa",
		map[string]any{"reference": r.obj("booking")["reference"], "authority": r.obj("payment")["authority"]})
	assert.Equal(t, http.StatusUnprocessableEntity, v.status)
	assert.Equal(t, "amount_mismatch", v.body["error_code"])
}

func TestBooking_Validation(t *testing.T) {
	be := setupBooking(t, true)
	_, tok := be.user(t, "Nava", "09120000001")
	owner, _ := be.user(t, "Other", "09120000009")
	r := be.do(t, http.MethodPost, "/api/v1/telemed/bookings", tok, "en", map[string]any{})
	require.Equal(t, http.StatusUnprocessableEntity, r.status)
	for _, k := range []string{"doctor_id", "mode", "starts_at", "for_whom"} {
		assert.Contains(t, r.body["errors"], k)
	}
	r = be.book(t, tok, "16:00", map[string]any{"for_whom": "other"})
	assert.Contains(t, r.body["errors"], "patient_name")
	r = be.book(t, tok, "16:00", map[string]any{"for_whom": "child"})
	assert.Contains(t, r.body["errors"], "child_id")
	r = be.book(t, tok, "16:00", map[string]any{"reason": "nope", "share": []any{"everything"}})
	assert.Contains(t, r.body["errors"], "reason")
	assert.Contains(t, r.body["errors"], "share.0")
	r = be.book(t, tok, "16:00", map[string]any{"for_whom": "other", "patient_name": "Mina", "share": []any{"bbt_lh"}})
	assert.Equal(t, "consent_not_allowed", r.body["error_code"])
	res := be.Exec(`INSERT INTO children (owner_id, name, birth_date, created_at, updated_at) VALUES (?, 'Ava', '2029-10-01', NOW(), NOW())`, owner)
	kid, _ := res.LastInsertId()
	r = be.book(t, tok, "16:00", map[string]any{"for_whom": "child", "child_id": kid})
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
	assert.Contains(t, r.body["errors"], "child_id", "another user's child")
	r = be.book(t, tok, "17:00", nil)
	assert.Equal(t, http.StatusConflict, r.status, "time off is not a slot")
	assert.Equal(t, "slot_unavailable", r.body["error_code"])
}

func TestBooking_DoubleBookingRace(t *testing.T) {
	be := setupBooking(t, true)
	const n = 12
	tokens := make([]uint64, n)
	for i := range n {
		tokens[i], _ = be.user(t, fmt.Sprintf("U%d", i), fmt.Sprintf("0912100%04d", i))
	}
	start := time.Date(2030, 1, 8, 16, 0, 0, 0, fixed.Location())
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok, taken := 0, 0
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			mode := telemed.ModeVideo
			if i%2 == 1 {
				mode = telemed.ModeInPerson // 16:00–16:30 overlaps the video 16:00–16:20
			}
			_, err := be.bk.Create(context.Background(), tokens[i], telemed.BookInput{
				DoctorID: be.a, Mode: mode, StartsAt: start, ForWhom: telemed.ForSelf,
			}, catalog.Localizer{Locale: "fa", Default: "fa"})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ok++
			case errors.Is(err, telemed.ErrSlotUnavailable):
				taken++
			default:
				t.Errorf("unexpected: %v", err)
			}
		}(i)
	}
	wg.Wait()
	assert.Equal(t, 1, ok)
	assert.Equal(t, n-1, taken)
	assert.Equal(t, 1, be.Int(`SELECT COUNT(*) FROM telemed_bookings WHERE status = 'held'`))
}

func TestBooking_IDOR(t *testing.T) {
	be := setupBooking(t, true)
	_, tokA := be.user(t, "A", "09120000001")
	_, tokB := be.user(t, "B", "09120000002")
	bid := be.pay(t, tokA, "16:00", nil)
	ref := be.String(`SELECT reference FROM telemed_bookings WHERE id = ?`, bid)
	p := fmt.Sprintf("/api/v1/telemed/bookings/%d", bid)
	for _, tc := range []struct {
		m, p string
		body any
	}{
		{http.MethodGet, p, nil}, {http.MethodPost, p + "/cancel", nil},
		{http.MethodPost, p + "/reschedule", map[string]any{"starts_at": "2030-01-08 16:30"}},
		{http.MethodPut, p + "/consent", map[string]any{"bbt_lh": true}}, {http.MethodDelete, p + "/consent", nil},
		{http.MethodPost, "/api/v1/telemed/bookings/verify", map[string]any{"reference": ref, "authority": "FAKE-" + ref}},
	} {
		r := be.do(t, tc.m, tc.p, tokB, "en", tc.body)
		assert.Equal(t, http.StatusNotFound, r.status, tc.p)
		assert.Equal(t, "booking_not_found", r.body["error_code"], tc.p)
	}
	r := be.do(t, http.MethodGet, "/api/v1/telemed/bookings?scope=all", tokB, "en", nil)
	assert.Empty(t, r.items())
	assert.Equal(t, "confirmed", be.String(`SELECT status FROM telemed_bookings WHERE id = ?`, bid))
}

func TestBooking_CancelWindowAndReschedule(t *testing.T) {
	be := setupBooking(t, true)
	uid, tok := be.user(t, "Nava", "09120000001")
	_, tok2 := be.user(t, "Sara", "09120000002")
	bid := be.pay(t, tok, "16:00", nil)
	be.pay(t, tok2, "18:30", nil)
	p := fmt.Sprintf("/api/v1/telemed/bookings/%d", bid)

	// Reschedule into someone else's slot: 409; into a free one: moved with its appointment.
	r := be.do(t, http.MethodPost, p+"/reschedule", tok, "fa", map[string]any{"starts_at": "2030-01-08 18:30"})
	assert.Equal(t, http.StatusConflict, r.status)
	r = be.do(t, http.MethodPost, p+"/reschedule", tok, "fa", map[string]any{"starts_at": "2030-01-08 16:30"})
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, "16:30", r.obj("booking")["time"])
	assert.Equal(t, 1, be.Int(`SELECT COUNT(*) FROM reminders WHERE user_id = ? AND scheduled_at = '2030-01-08 16:30:00'`, uid))

	// 14:30 is exactly 2 h before 16:30: closed.
	be.clk.set(time.Date(2030, 1, 8, 14, 30, 0, 0, fixed.Location()))
	r = be.do(t, http.MethodPost, p+"/cancel", tok, "fa", nil)
	assert.Equal(t, http.StatusConflict, r.status)
	assert.Equal(t, "cancel_window_closed", r.body["error_code"])
	r = be.do(t, http.MethodPost, p+"/reschedule", tok, "fa", map[string]any{"starts_at": "2030-01-08 19:00"})
	assert.Equal(t, "cancel_window_closed", r.body["error_code"])

	// One second earlier: free cancel with a refund and the appointment cancelled.
	be.clk.set(time.Date(2030, 1, 8, 14, 29, 59, 0, fixed.Location()))
	r = be.do(t, http.MethodPost, p+"/cancel", tok, "fa", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, "cancelled", r.obj("booking")["status"])
	assert.Equal(t, "refunded", r.obj("booking")["payment_status"])
	assert.Equal(t, 0, be.Int(`SELECT is_active FROM reminders WHERE user_id = ? AND type = 'appointment'`, uid))
	r = be.do(t, http.MethodPost, p+"/cancel", tok, "fa", nil)
	assert.Equal(t, "booking_closed", r.body["error_code"])
}

func TestBooking_HoldExpiryAndLatePaymentRefund(t *testing.T) {
	be := setupBooking(t, true)
	_, tok := be.user(t, "Nava", "09120000001")
	_, tok2 := be.user(t, "Sara", "09120000002")
	r := be.book(t, tok, "16:00", nil)
	require.Equal(t, http.StatusCreated, r.status)
	be.clk.set(fixed.Add(11 * time.Minute))
	exp, _, err := be.bk.Sweep(context.Background(), be.clk.Now())
	require.NoError(t, err)
	assert.Equal(t, 1, exp)
	require.Equal(t, http.StatusCreated, be.book(t, tok2, "16:00", nil).status, "a lapsed hold frees the slot")
	// The first user pays late: the slot is gone, the payment is refunded.
	_, err = be.fake.Decide(context.Background(), r.obj("payment")["authority"].(string), true)
	require.NoError(t, err)
	v := be.do(t, http.MethodPost, "/api/v1/telemed/bookings/verify", tok, "fa",
		map[string]any{"reference": r.obj("booking")["reference"], "authority": r.obj("payment")["authority"]})
	assert.Equal(t, http.StatusConflict, v.status, v.body)
	assert.Equal(t, "booking_refunded", v.body["error_code"])
	assert.Equal(t, "refunded", be.String(`SELECT payment_status FROM telemed_bookings WHERE id = ?`, id(r.obj("booking")["id"])))
}

func TestBooking_CompleteReviewAndConsent(t *testing.T) {
	be := setupBooking(t, true)
	_, tok := be.user(t, "Nava", "09120000001")
	bid := be.pay(t, tok, "16:00", map[string]any{"share": []any{"cycle_summary", "bbt_lh"}})
	ctx := context.Background()
	at := time.Date(2030, 1, 8, 15, 0, 0, 0, fixed.Location())

	_, err := be.bk.AuthorizeShare(ctx, be.a, bid, telemed.ScopeCycleSummary, at)
	require.NoError(t, err)
	_, err = be.bk.AuthorizeShare(ctx, be.a, bid, telemed.ScopeAssistantSummaries, at)
	require.ErrorIs(t, err, telemed.ErrConsentRequired, "never given")
	_, err = be.bk.AuthorizeShare(ctx, be.other, bid, telemed.ScopeCycleSummary, at)
	require.ErrorIs(t, err, telemed.ErrConsentRequired, "another doctor")

	r := be.do(t, http.MethodPut, fmt.Sprintf("/api/v1/telemed/bookings/%d/consent", bid), tok, "fa", map[string]any{"bbt_lh": false})
	require.Equal(t, http.StatusOK, r.status, r.body)
	_, err = be.bk.AuthorizeShare(ctx, be.a, bid, telemed.ScopeBBTLH, at)
	require.ErrorIs(t, err, telemed.ErrConsentRequired, "revoked")

	// Visit ends → completed, counted, reviewable.
	be.clk.set(time.Date(2030, 1, 8, 16, 21, 0, 0, fixed.Location()))
	_, done, err := be.bk.Sweep(ctx, be.clk.Now())
	require.NoError(t, err)
	assert.Equal(t, 1, done)
	assert.Equal(t, 1, be.Int(`SELECT visits_count FROM telemed_doctors WHERE id = ?`, be.a))
	d := be.do(t, http.MethodGet, fmt.Sprintf("/api/v1/telemed/doctors/%d", be.a), tok, "fa", nil)
	assert.Equal(t, true, d.obj("doctor")["can_review"])
	rv := be.do(t, http.MethodPost, fmt.Sprintf("/api/v1/telemed/doctors/%d/reviews", be.a), tok, "fa", map[string]any{"rating": 5})
	assert.Equal(t, http.StatusCreated, rv.status, rv.body)

	// Still readable within the grace window, never after it; nothing after a full revoke.
	_, err = be.bk.AuthorizeShare(ctx, be.a, bid, telemed.ScopeCycleSummary, time.Date(2030, 1, 11, 16, 20, 0, 0, fixed.Location()))
	require.NoError(t, err)
	_, err = be.bk.AuthorizeShare(ctx, be.a, bid, telemed.ScopeCycleSummary, time.Date(2030, 1, 11, 16, 20, 1, 0, fixed.Location()))
	require.ErrorIs(t, err, telemed.ErrConsentRequired, "after the visit window")
	r = be.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/telemed/bookings/%d/consent", bid), tok, "fa", nil)
	require.Equal(t, http.StatusOK, r.status)
	_, err = be.bk.AuthorizeShare(ctx, be.a, bid, telemed.ScopeCycleSummary, at)
	require.ErrorIs(t, err, telemed.ErrConsentRequired)
}

func TestBooking_PaymentUnavailable(t *testing.T) {
	be := setupBooking(t, false)
	_, tok := be.user(t, "Nava", "09120000001")
	r := be.book(t, tok, "16:00", nil)
	assert.Equal(t, http.StatusServiceUnavailable, r.status)
	assert.Equal(t, "payment_unavailable", r.body["error_code"])
	assert.Equal(t, 0, be.Int(`SELECT COUNT(*) FROM telemed_bookings`))
}
