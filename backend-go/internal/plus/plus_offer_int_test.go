package plus_test

// B-N2-06: Plus gating, the trial offer (status, home banner, trial sheet, checkout) and the trial usage lines.
// Every time comes from X-Test-Now / an explicit now, so trial start, expiry and the countdown are deterministic.

import (
	"context"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/home"
	homestore "github.com/ritme/backend-go/internal/home/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	msgstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/plus"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// offerEnv adds the B-N2-06 routes to the plus test app: the trial sheet, two gated probes and the home overview.
func offerEnv(t *testing.T) *env {
	t.Helper()
	e := setup(t, plus.FakeGateway{})
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(e.db), nil, quiet))
	h := plus.NewHandlers(e.svc, clock.Real{})
	gate := plus.NewGate(e.svc, clock.Real{})
	g := e.guard
	ok := func(c fiber.Ctx) error { return c.JSON(fiber.Map{"ok": true}) }
	e.app.Get("/api/v1/plus/trial", locale, g, h.TrialSheet)
	e.app.Get("/probe/deep", locale, g, gate.Require(plus.DeepAnalysis), ok)
	e.app.Get("/probe/lab", locale, g, gate.Require(plus.LabAI), ok)
	e.app.Get("/probe/assistant", locale, g, gate.Require(plus.AssistantUnlimited), ok)
	hh := home.NewHandlers(home.Deps{
		Queries: homestore.New(e.db), Messages: msgstore.New(e.db), Pregnancy: pstore.New(e.db), Logger: quiet, Plus: e.svc,
	}, cycleservice.New(e.db, nil), clock.Real{})
	e.app.Get("/api/v1/home/cycle-overview", locale, g, hh.CycleOverview)
	return e
}

func TestTrialOffer_StatusBannerSheetAndExpiry(t *testing.T) {
	e := offerEnv(t)
	uid, tok := e.user(t, "09120000901")
	ctx := context.Background()

	// No trial: no offer anywhere.
	r := e.do(t, http.MethodGet, "/api/v1/plus/status", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["trial_offer"])
	r = e.do(t, http.MethodGet, "/api/v1/home/cycle-overview", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Contains(t, r.data(), "plus_trial_offer")
	assert.Nil(t, r.data()["plus_trial_offer"])

	// Trial from 2026-09-28 10:00 to 2026-10-05 10:00 (it straddles two quota months).
	start := "2026-09-28T10:00:00+03:30"
	r = e.doAt(t, start, http.MethodPost, "/api/v1/plus/trial/start", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"percent":50,"ends_at":"2026-10-05T10:00:00+03:30","seconds_left":604800,"days_left":7,
		"countdown":{"days":7,"hours":0,"minutes":0},"currency":"IRR",
		"plan":{"id":2,"code":"plus_3m","title":"3 months","badge":"Most popular","duration_months":3,"price":2370000,
			"monthly_price":790000,"savings_percent":20,"is_highlighted":true,"offer_price":1185000,"offer_monthly_price":395000}}`,
		mustJSON(t, r.data()["trial_offer"]))

	// The home banner: two days later, «۰۵ روز ۱۴ ساعت ۲۲ دقیقه»-style countdown, localized plan.
	at := "2026-09-29T19:38:00+03:30"
	r = e.doAt(t, at, http.MethodGet, "/api/v1/home/cycle-overview", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	banner, _ := r.data()["plus_trial_offer"].(map[string]any)
	require.NotNil(t, banner, r.raw)
	assert.EqualValues(t, 50, banner["percent"])
	assert.Equal(t, map[string]any{"days": float64(5), "hours": float64(14), "minutes": float64(22)}, banner["countdown"])
	assert.EqualValues(t, 5*86400+14*3600+22*60, banner["seconds_left"])
	assert.EqualValues(t, 6, banner["days_left"])
	assert.Equal(t, "۳ ماهه", banner["plan"].(map[string]any)["title"])

	// Usage on both sides of the month boundary counts on the sheet; /plus/usage stays monthly.
	_, err := e.svc.Consume(ctx, uid, plus.DeepAnalysis, mustTime("2026-09-29T09:00:00+03:30"))
	require.NoError(t, err)
	_, err = e.svc.Consume(ctx, uid, plus.DeepAnalysis, mustTime("2026-09-30T09:00:00+03:30"))
	require.NoError(t, err)
	at = "2026-10-02T12:00:00+03:30"
	for range 3 {
		_, err = e.svc.Consume(ctx, uid, plus.AssistantUnlimited, mustTime(at))
		require.NoError(t, err)
	}
	_, err = e.svc.Consume(ctx, uid, plus.DeepAnalysis, mustTime(at))
	require.NoError(t, err)
	_, err = e.svc.Consume(ctx, uid, plus.LabAI, mustTime(at))
	require.NoError(t, err)

	r = e.doAt(t, at, http.MethodGet, "/api/v1/plus/trial", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "trial", d["tier"])
	assert.Equal(t, map[string]any{"started_at": start, "ends_at": "2026-10-05T10:00:00+03:30", "is_active": true,
		"days_left": float64(3), "seconds_left": float64(2*86400 + 22*3600)}, d["trial"])
	assert.Equal(t, false, d["trial_available"])
	assert.EqualValues(t, 50, d["offer"].(map[string]any)["percent"])
	plans, _ := d["plans"].([]any)
	require.Len(t, plans, 3)
	assert.EqualValues(t, 495000, plans[0].(map[string]any)["offer_price"])
	assert.EqualValues(t, 1950000, plans[2].(map[string]any)["offer_price"])
	assert.JSONEq(t, `{"since":"2026-09-01","features":[
		{"key":"plus.deep_analysis","used":3,"unlimited":true,"plus_limit":null},
		{"key":"plus.lab_ai","used":1,"unlimited":false,"plus_limit":10},
		{"key":"plus.assistant_unlimited","used":3,"unlimited":true,"plus_limit":null},
		{"key":"plus.pdf_share","used":0,"unlimited":true,"plus_limit":null},
		{"key":"plus.visit_discount","used":0,"unlimited":true,"plus_limit":null},
		{"key":"plus.voice_log","used":0,"unlimited":true,"plus_limit":null}]}`, mustJSON(t, d["usage"]))
	r = e.doAt(t, at, http.MethodGet, "/api/v1/plus/usage", tok, "en", "")
	assert.EqualValues(t, 1, entitlement(t, r.data()["features"], "plus.deep_analysis")["used"], "monthly counter")

	// One second before the end the offer still runs; at the end instant it is gone everywhere.
	r = e.doAt(t, "2026-10-05T09:59:59+03:30", http.MethodGet, "/api/v1/plus/status", tok, "en", "")
	assert.EqualValues(t, 1, r.obj("trial_offer")["seconds_left"])
	end := "2026-10-05T10:00:00+03:30"
	r = e.doAt(t, end, http.MethodGet, "/api/v1/plus/status", tok, "en", "")
	assert.Equal(t, "free", r.data()["tier"])
	assert.Nil(t, r.data()["trial_offer"])
	r = e.doAt(t, end, http.MethodGet, "/api/v1/home/cycle-overview", tok, "en", "")
	assert.Nil(t, r.data()["plus_trial_offer"])
	r = e.doAt(t, end, http.MethodGet, "/api/v1/plus/trial", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["offer"])
	assert.Nil(t, r.data()["plans"].([]any)[0].(map[string]any)["offer_price"])
	assert.Equal(t, false, r.obj("trial")["is_active"])
}

func TestTrialOffer_CheckoutAppliesItServerSide(t *testing.T) {
	e := offerEnv(t)
	_, tok := e.user(t, "09120000902")
	ctx := context.Background()
	e.discount(t, "RITME10", plus.DiscountPercent, 10, nil, 1, nil)
	e.discount(t, "BIG60", plus.DiscountPercent, 60, nil, 1, nil)

	// Before the trial: list price.
	r := e.do(t, http.MethodPost, "/api/v1/plus/checkout", tok, "en", `{"plan_id":2,"preview":true}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.EqualValues(t, 0, r.obj("quote")["discount"])
	assert.Nil(t, r.obj("quote")["discount_source"])
	assert.Nil(t, r.obj("quote")["trial_offer"])

	r = e.do(t, http.MethodPost, "/api/v1/plus/trial/start", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)

	// During the trial: half price, VAT on the discounted price; the client never sends an amount.
	r = e.do(t, http.MethodPost, "/api/v1/plus/checkout", tok, "en", `{"plan_id":2,"preview":true}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"plan":{"id":2,"code":"plus_3m","title":"3 months","duration_months":3},"currency":"IRR",
		"subtotal":2370000,"discount":1185000,"discount_code":null,"vat_rate_bps":1000,"vat":118500,"total":1303500,
		"discount_source":"trial_offer","trial_offer":{"percent":50,"ends_at":"2026-09-30T10:00:00+03:30","offer_price":1185000}}`,
		mustJSON(t, r.obj("quote")))

	// Codes never stack with the offer: the larger discount wins.
	r = e.do(t, http.MethodPost, "/api/v1/plus/checkout", tok, "en", `{"plan_id":2,"preview":true,"discount_code":"ritme10"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "trial_offer", r.obj("quote")["discount_source"])
	assert.Nil(t, r.obj("quote")["discount_code"])
	assert.EqualValues(t, 1185000, r.obj("quote")["discount"])
	r = e.do(t, http.MethodPost, "/api/v1/plus/checkout", tok, "en", `{"plan_id":2,"preview":true,"discount_code":"BIG60"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "code", r.obj("quote")["discount_source"])
	assert.Equal(t, "BIG60", r.obj("quote")["discount_code"])
	assert.Nil(t, r.obj("quote")["trial_offer"])
	assert.EqualValues(t, 1422000, r.obj("quote")["discount"])
	// An invalid code is still rejected.
	r = e.do(t, http.MethodPost, "/api/v1/plus/checkout", tok, "en", `{"plan_id":2,"preview":true,"discount_code":"NOPE"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)

	// The admin percent drives the price (and 0 turns the offer off).
	require.NoError(t, e.svc.SetTrialOfferPercent(ctx, 20, tNow))
	r = e.do(t, http.MethodPost, "/api/v1/plus/checkout", tok, "en", `{"plan_id":1,"preview":true}`)
	assert.EqualValues(t, 198000, r.obj("quote")["discount"])
	assert.EqualValues(t, 20, r.obj("quote")["trial_offer"].(map[string]any)["percent"])
	require.NoError(t, e.svc.SetTrialOfferPercent(ctx, 0, tNow))
	r = e.do(t, http.MethodPost, "/api/v1/plus/checkout", tok, "en", `{"plan_id":1,"preview":true}`)
	assert.EqualValues(t, 0, r.obj("quote")["discount"])
	r = e.do(t, http.MethodGet, "/api/v1/plus/status", tok, "en", "")
	assert.Nil(t, r.data()["trial_offer"])
	require.ErrorIs(t, e.svc.SetTrialOfferPercent(ctx, 101, tNow), plus.ErrInvalidSetting)
	require.ErrorIs(t, e.svc.SetTrialOfferPercent(ctx, -1, tNow), plus.ErrInvalidSetting)
	// An unreadable row falls back to the default 50 % instead of breaking checkout.
	_, err := e.db.Exec(`UPDATE plus_settings SET value = 'half' WHERE ` + "`key`" + ` = 'trial_offer_percent'`)
	require.NoError(t, err)
	pct, err := e.svc.TrialOfferPercent(ctx)
	require.NoError(t, err)
	assert.Equal(t, plus.DefaultTrialOfferPercent, pct)
	require.NoError(t, e.svc.SetTrialOfferPercent(ctx, 50, tNow))

	// The real checkout writes the offer price into the invoice and the payment settles it.
	ref, authority, r := e.checkout(t, now, tok, `{"plan_id":2}`)
	inv := r.obj("invoice")
	assert.EqualValues(t, 1185000, inv["discount"])
	assert.EqualValues(t, 1303500, inv["total"])
	assert.Nil(t, inv["discount_code"])
	assert.Equal(t, "trial_offer", inv["discount_source"])
	r = e.do(t, http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "OK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.EqualValues(t, 1303500, r.obj("invoice", "receipt")["amount"])
	assert.Equal(t, "plus", r.obj("status")["tier"])
	assert.Nil(t, r.obj("status")["trial_offer"], "a subscriber has no offer")
	r = e.do(t, http.MethodGet, "/api/v1/plus/history", tok, "en", "")
	assert.Equal(t, "trial_offer", r.body["data"].([]any)[0].(map[string]any)["discount_source"])
	// Subscribed during the trial: the next purchase is list price again.
	r = e.do(t, http.MethodPost, "/api/v1/plus/checkout", tok, "en", `{"plan_id":2,"preview":true}`)
	assert.EqualValues(t, 0, r.obj("quote")["discount"])
}

func TestTrialOffer_ExpiredTrialPaysListPrice(t *testing.T) {
	e := offerEnv(t)
	_, tok := e.user(t, "09120000903")
	r := e.do(t, http.MethodPost, "/api/v1/plus/trial/start", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.doAt(t, "2026-09-30T09:59:59+03:30", http.MethodPost, "/api/v1/plus/checkout", tok, "en", `{"plan_id":3,"preview":true}`)
	assert.EqualValues(t, 1950000, r.obj("quote")["discount"], "last second of the trial")
	_, _, r = e.checkout(t, "2026-09-30T10:00:00+03:30", tok, `{"plan_id":3}`)
	assert.EqualValues(t, 0, r.obj("invoice")["discount"])
	assert.Nil(t, r.obj("invoice")["discount_source"])
	assert.EqualValues(t, 4290000, r.obj("invoice")["total"])
}

func TestGate_PlusRequiredAndQuotas(t *testing.T) {
	e := offerEnv(t)
	uid, tok := e.user(t, "09120000904")
	ctx := context.Background()

	r := e.do(t, http.MethodGet, "/probe/deep", "", "en", "")
	assert.Equal(t, http.StatusUnauthorized, r.status)
	assert.JSONEq(t, `{"message":"Unauthenticated.","error_code":"unauthenticated"}`, r.raw)

	// Free: locked features → 402 plus_required.
	r = e.do(t, http.MethodGet, "/probe/deep", tok, "en", "")
	assert.Equal(t, http.StatusPaymentRequired, r.status)
	assert.JSONEq(t, `{"success":false,"message":"This feature is part of Ritme Plus.","error_code":"plus_required",
		"feature":"plus.deep_analysis","reason":"locked","limit":null,"resets_at":null}`, r.raw)
	r = e.do(t, http.MethodGet, "/probe/lab", tok, "fa", "")
	assert.Equal(t, http.StatusPaymentRequired, r.status)
	assert.Equal(t, "این امکان بخشی از ریتمی پلاس است.", r.body["message"])

	// Free assistant allowance: 5 a month, then plus_required with the quota reason.
	r = e.do(t, http.MethodGet, "/probe/assistant", tok, "en", "")
	assert.Equal(t, http.StatusOK, r.status, r.raw)
	for range 5 {
		_, err := e.svc.Consume(ctx, uid, plus.AssistantUnlimited, tNow)
		require.NoError(t, err)
	}
	r = e.do(t, http.MethodGet, "/probe/assistant", tok, "en", "")
	assert.Equal(t, http.StatusPaymentRequired, r.status)
	assert.JSONEq(t, `{"success":false,"message":"This feature is part of Ritme Plus.","error_code":"plus_required",
		"feature":"plus.assistant_unlimited","reason":"quota","limit":5,"resets_at":"2026-10-01T00:00:00+03:30"}`, r.raw)

	// Trial: everything open; a spent Plus quota is 429 plus_quota_exceeded (upgrading does not help).
	require.NoError(t, e.svc.StartTrial(ctx, uid, tNow))
	for _, p := range []string{"/probe/deep", "/probe/lab", "/probe/assistant"} {
		r = e.do(t, http.MethodGet, p, tok, "en", "")
		assert.Equal(t, http.StatusOK, r.status, p)
	}
	for range 10 {
		_, err := e.svc.Consume(ctx, uid, plus.LabAI, tNow)
		require.NoError(t, err)
	}
	r = e.do(t, http.MethodGet, "/probe/lab", tok, "en", "")
	assert.Equal(t, http.StatusTooManyRequests, r.status)
	assert.JSONEq(t, `{"success":false,"message":"You have reached this month's limit for this feature. It renews at the start of next month.",
		"error_code":"plus_quota_exceeded","feature":"plus.lab_ai","reason":"quota","limit":10,"resets_at":"2026-10-01T00:00:00+03:30"}`, r.raw)

	// Trial over: locked again.
	r = e.doAt(t, "2026-09-30T10:00:00+03:30", http.MethodGet, "/probe/deep", tok, "en", "")
	assert.Equal(t, http.StatusPaymentRequired, r.status)
	assert.Equal(t, "plus_required", r.body["error_code"])

	// Another user is unaffected by this user's trial or counters (no cross-user leakage).
	_, tok2 := e.user(t, "09120000905")
	r = e.do(t, http.MethodGet, "/probe/assistant", tok2, "en", "")
	assert.Equal(t, http.StatusOK, r.status)
	r = e.do(t, http.MethodGet, "/probe/deep", tok2, "en", "")
	assert.Equal(t, http.StatusPaymentRequired, r.status)

	assert.Panics(t, func() { plus.NewGate(e.svc, clock.Real{}).Require("plus.nope") })
}

func TestGate_GateErrorMapsConsume(t *testing.T) {
	e := offerEnv(t)
	uid, tok := e.user(t, "09120000906")
	gate := plus.NewGate(e.svc, clock.Real{})
	g := e.guard
	e.app.Post("/probe/consume-lab", g, func(c fiber.Ctx) error {
		ent, err := e.svc.Consume(c, uid, plus.LabAI, tNow)
		if err != nil {
			return gate.GateError(c, err, ent)
		}
		return c.JSON(fiber.Map{"remaining": ent.Remaining})
	})
	r := e.do(t, http.MethodPost, "/probe/consume-lab", tok, "en", "")
	assert.Equal(t, http.StatusPaymentRequired, r.status, r.raw)
	assert.Equal(t, "plus_required", r.body["error_code"])
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM plus_usage_counters WHERE user_id = ?`, uid), "nothing counted")
}
