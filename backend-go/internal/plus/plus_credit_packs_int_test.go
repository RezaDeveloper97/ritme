package plus_test

// Credit packs (goose 00125): extra AI credits bought through the normal invoice → gateway → verify path. The pack
// invoice grants its credits exactly once (replayed and concurrent verifies), a tampered amount grants nothing,
// teen accounts cannot buy, an unknown or inactive pack is a 422 on pack_id.

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/plus"
)

// blockUsers is a commercial policy that refuses the listed accounts (a teen-mode account in production).
type blockUsers map[uint64]bool

func (b blockUsers) AllowsCommercial(_ context.Context, userID uint64) (bool, error) {
	return !b[userID], nil
}

func packEnv(t *testing.T, gw plus.Gateway, blocked blockUsers) *env {
	t.Helper()
	e := setup(t, gw)
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(e.db), nil, quiet))
	h := plus.NewHandlers(e.svc, clock.Real{}).WithCommercial(blocked)
	e.app.Get("/api/v1/plus/credit-packs", locale, h.Packs)
	e.app.Post("/api/v1/plus/credit-packs/checkout", locale, e.guard, h.CheckoutPack)
	return e
}

func (e *env) packCheckout(t *testing.T, token, body string) (string, string, response) {
	t.Helper()
	r := e.do(t, http.MethodPost, "/api/v1/plus/credit-packs/checkout", token, "en", body)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	ref, _ := r.obj("invoice")["reference"].(string)
	auth, _ := r.obj("payment")["authority"].(string)
	return ref, auth, r
}

func (e *env) extraCredits(t *testing.T, userID uint64) int {
	t.Helper()
	return e.count(t, `SELECT COALESCE(SUM(remaining), 0) FROM ai_credit_grants WHERE user_id = ? AND kind = 'extra'`, userID)
}

func TestCreditPacks_ListAndPreview(t *testing.T) {
	e := packEnv(t, plus.FakeGateway{}, nil)
	r := e.do(t, http.MethodGet, "/api/v1/plus/credit-packs", "", "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"currency":"IRR","vat_rate_bps":1000,"packs":[
		{"id":1,"code":"credits_50","title":"۵۰ اعتبار","badge":null,"credits":50,"price":490000,"is_highlighted":false},
		{"id":2,"code":"credits_150","title":"۱۵۰ اعتبار","badge":"محبوب\u200cترین","credits":150,"price":1290000,"is_highlighted":true},
		{"id":3,"code":"credits_400","title":"۴۰۰ اعتبار","badge":null,"credits":400,"price":2990000,"is_highlighted":false}]}`,
		mustJSON(t, r.data()))

	_, tok := e.user(t, "09120001101")
	r = e.do(t, http.MethodPost, "/api/v1/plus/credit-packs/checkout", tok, "en", `{"pack_id":1,"preview":true}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"pack":{"id":1,"code":"credits_50","title":"50 credits","badge":null,"credits":50,"price":490000,"is_highlighted":false},
		"currency":"IRR","subtotal":490000,"vat_rate_bps":1000,"vat":49000,"total":539000}`, mustJSON(t, r.obj("quote")))
	assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM plus_invoices`), "a preview writes nothing")

	// Unknown and inactive packs are a validation error on pack_id.
	_, err := e.db.Exec(`UPDATE ai_credit_packs SET is_active = 0 WHERE id = 3`)
	require.NoError(t, err)
	for _, body := range []string{`{"pack_id":999}`, `{"pack_id":3}`, `{"pack_id":3,"preview":true}`} {
		r = e.do(t, http.MethodPost, "/api/v1/plus/credit-packs/checkout", tok, "en", body)
		require.Equal(t, http.StatusUnprocessableEntity, r.status, body+" "+r.raw)
		assert.Contains(t, r.body["errors"], "pack_id", body)
	}
	r = e.do(t, http.MethodPost, "/api/v1/plus/credit-packs/checkout", tok, "en", `{}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.body["errors"], "pack_id")
	r = e.do(t, http.MethodGet, "/api/v1/plus/credit-packs", "", "en", "")
	assert.Len(t, r.data()["packs"], 2, "an inactive pack is not listed")
	assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM plus_invoices`))
}

func TestCreditPacks_CheckoutVerifyGrantsOnce(t *testing.T) {
	e := packEnv(t, plus.FakeGateway{}, nil)
	uid, tok := e.user(t, "09120001102")
	ref, authority, r := e.packCheckout(t, tok, `{"pack_id":1,"plan_id":2}`)
	inv := r.obj("invoice")
	assert.Equal(t, "credit_pack", inv["kind"])
	assert.EqualValues(t, 50, inv["credits"])
	assert.Equal(t, map[string]any{"id": float64(1), "code": "credits_50", "title": "50 credits", "credits": float64(50)}, inv["credit_pack"])
	assert.Nil(t, inv["plan"], "a pack invoice buys no plan, whatever the body says")
	assert.EqualValues(t, 539000, inv["total"])
	assert.Equal(t, "pending", inv["status"])
	assert.Zero(t, e.extraCredits(t, uid), "nothing before the payment")

	r = e.do(t, http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "OK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, false, r.data()["already_verified"])
	assert.Equal(t, "paid", r.obj("invoice")["status"])
	assert.Equal(t, "credit_pack", r.obj("invoice")["kind"])
	assert.EqualValues(t, 539000, r.obj("invoice", "receipt")["amount"])
	assert.Equal(t, "free", r.obj("status")["tier"], "a pack is not a subscription")
	assert.Equal(t, 50, e.extraCredits(t, uid))
	assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, uid))

	// A replayed verify grants nothing more.
	r = e.do(t, http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "OK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, true, r.data()["already_verified"])
	assert.Equal(t, 50, e.extraCredits(t, uid))

	// The history lists the pack invoice with its kind.
	r = e.do(t, http.MethodGet, "/api/v1/plus/history", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	list, _ := r.body["data"].([]any)
	require.Len(t, list, 1)
	assert.Equal(t, "credit_pack", list[0].(map[string]any)["kind"])
	assert.EqualValues(t, 50, list[0].(map[string]any)["credits"])

	// A second pack adds up; the credits stay when the user later subscribes.
	ref2, auth2, _ := e.packCheckout(t, tok, `{"pack_id":2}`)
	r = e.do(t, http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref2, auth2, "OK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, 200, e.extraCredits(t, uid))
	pref, pauth, _ := e.checkout(t, now, tok, `{"plan_id":1}`)
	r = e.do(t, http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(pref, pauth, "OK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "plus", r.obj("status")["tier"])
	assert.Equal(t, 200, e.extraCredits(t, uid))
}

func TestCreditPacks_ConcurrentVerifyGrantsOnce(t *testing.T) {
	e := packEnv(t, plus.FakeGateway{}, nil)
	uid, tok := e.user(t, "09120001103")
	ref, authority, _ := e.packCheckout(t, tok, `{"pack_id":1}`)
	var wg sync.WaitGroup
	errs := make([]error, 8)
	already := make([]bool, 8)
	for i := range errs {
		wg.Go(func() {
			res, err := e.svc.Verify(context.Background(), uid, ref, authority, map[string]string{"status": "OK"}, tNow)
			errs[i], already[i] = err, res.AlreadyPaid
		})
	}
	wg.Wait()
	fresh := 0
	for i, err := range errs {
		require.NoError(t, err)
		if !already[i] {
			fresh++
		}
	}
	assert.Equal(t, 1, fresh)
	assert.Equal(t, 50, e.extraCredits(t, uid))
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM ai_credit_grants WHERE user_id = ?`, uid))
}

func TestCreditPacks_AmountMismatchGrantsNothing(t *testing.T) {
	e := packEnv(t, tamperGateway{amountDelta: -1000}, nil)
	uid, tok := e.user(t, "09120001104")
	ref, authority, _ := e.packCheckout(t, tok, `{"pack_id":1}`)
	r := e.do(t, http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "OK"))
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "amount_mismatch", r.body["error_code"])
	assert.Zero(t, e.extraCredits(t, uid))
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_invoices WHERE reference = ? AND status = 'failed'`, ref))

	// A declined payment grants nothing either; a different user cannot verify the invoice.
	ok := packEnv(t, plus.FakeGateway{}, nil)
	uid, tok = ok.user(t, "09120001105")
	_, other := ok.user(t, "09120001106")
	ref, authority, _ = ok.packCheckout(t, tok, `{"pack_id":1}`)
	r = ok.do(t, http.MethodPost, "/api/v1/plus/verify", other, "en", verifyBody(ref, authority, "OK"))
	require.Equal(t, http.StatusNotFound, r.status, r.raw)
	r = ok.do(t, http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "NOK"))
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "payment_failed", r.body["error_code"])
	assert.Zero(t, ok.extraCredits(t, uid))
}

func TestCreditPacks_TeenBlocked(t *testing.T) {
	blocked := blockUsers{}
	e := packEnv(t, plus.FakeGateway{}, blocked)
	uid, tok := e.user(t, "09120001107")
	blocked[uid] = true
	for _, body := range []string{`{"pack_id":1}`, `{"pack_id":1,"preview":true}`} {
		r := e.do(t, http.MethodPost, "/api/v1/plus/credit-packs/checkout", tok, "en", body)
		require.Equal(t, http.StatusForbidden, r.status, r.raw)
		assert.Equal(t, plus.ErrorCodeTeenCommercialBlocked, r.body["error_code"])
	}
	assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM plus_invoices`))
}

func TestCreditPacks_Unauthenticated(t *testing.T) {
	e := packEnv(t, plus.FakeGateway{}, nil)
	r := e.do(t, http.MethodPost, "/api/v1/plus/credit-packs/checkout", "", "en", `{"pack_id":1}`)
	assert.Equal(t, http.StatusUnauthorized, r.status)
	assert.JSONEq(t, `{"message":"Unauthenticated.","error_code":"unauthenticated"}`, r.raw)
}
