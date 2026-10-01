package plus_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/payments"
	"github.com/ritme/backend-go/internal/platform/config"
)

// The domain wired to the real adapter (internal/payments, B-N2-05) with the fake provider, as the routes file
// does: verify is decided by the provider's own state, never by the callback status the client posts.
func adapterGateway(t *testing.T) (*payments.Gateway, *payments.Fake) {
	t.Helper()
	gw := payments.New(payments.Deps{
		App:    config.App{Env: "testing", URL: "http://api.test"},
		Config: config.Payment{Provider: "fake", ReturnURLs: []string{cfg.CallbackURL}},
		Plus:   cfg, Logger: quiet, FakeStore: payments.NewMemoryFakeStore(),
	})
	require.NotNil(t, gw)
	return gw, gw.Provider().(*payments.Fake)
}

func TestAdapter_PayOnTestPageThenVerifyAndReplay(t *testing.T) {
	gw, fake := adapterGateway(t)
	e := setup(t, gw)
	uid, tok := e.user(t, "09120000901")

	ref, authority, r := e.checkout(t, now, tok, `{"plan_id":1,"total":1}`)
	assert.Equal(t, "FAKE-"+ref, authority)
	assert.Equal(t, "http://api.test/api/v1/payments/fake/pay/FAKE-"+ref, r.obj("payment")["redirect_url"])

	_, err := fake.Decide(context.Background(), authority, true)
	require.NoError(t, err)

	// The client claims NOK: the provider says paid, and the provider wins.
	r = e.doAt(t, "2026-09-23T10:05:00+03:30", http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "NOK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "paid", r.obj("invoice")["status"])
	assert.Equal(t, "FAKE-REF-"+ref, r.obj("invoice", "receipt")["ref_id"])
	assert.EqualValues(t, 1089000, r.obj("invoice", "receipt")["amount"])

	// Double verify (replay) → 200 already_verified, nothing added.
	for range 2 {
		r = e.doAt(t, "2026-09-23T10:06:00+03:30", http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "OK"))
		require.Equal(t, http.StatusOK, r.status, r.raw)
		assert.Equal(t, true, r.data()["already_verified"])
	}
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, uid))
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_receipts WHERE user_id = ?`, uid))

	// The fake supports refunds of the settled payment.
	rf, err := gw.Refund(context.Background(), payments.RefundRequest{Authority: authority, RefID: "FAKE-REF-" + ref, AmountRials: 1089000})
	require.NoError(t, err)
	assert.EqualValues(t, 1089000, rf.AmountRials)
}

func TestAdapter_ForgedOKCallbackDoesNotPay(t *testing.T) {
	gw, _ := adapterGateway(t)
	e := setup(t, gw)
	uid, tok := e.user(t, "09120000902")
	ref, authority, _ := e.checkout(t, now, tok, `{"plan_id":1}`)

	// The user never paid on the TEST page but posts status=OK.
	r := e.doAt(t, "2026-09-23T10:05:00+03:30", http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "OK"))
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "payment_failed", r.body["error_code"])
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, uid))
}

func TestAdapter_AmountTamperIsRejected(t *testing.T) {
	gw, fake := adapterGateway(t)
	e := setup(t, gw)
	uid, tok := e.user(t, "09120000903")
	ref, authority, _ := e.checkout(t, now, tok, `{"plan_id":1}`)
	_, err := fake.Decide(context.Background(), authority, true)
	require.NoError(t, err)

	// The provider settled 1,089,000 rials; the invoice now says more (tampered row / mismatched session).
	_, err = e.db.Exec(`UPDATE plus_invoices SET total_rials = total_rials + 1000 WHERE reference = ?`, ref)
	require.NoError(t, err)
	r := e.doAt(t, "2026-09-23T10:05:00+03:30", http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "OK"))
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "amount_mismatch", r.body["error_code"])
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, uid))
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM plus_receipts WHERE user_id = ?`, uid))
}

func TestAdapter_RestoreSettlesAPaidCheckout(t *testing.T) {
	gw, fake := adapterGateway(t)
	e := setup(t, gw)
	uid, tok := e.user(t, "09120000904")
	_, authority, _ := e.checkout(t, now, tok, `{"plan_id":1}`)

	// Not paid yet: restore leaves it pending.
	r := e.doAt(t, "2026-09-23T10:05:00+03:30", http.MethodPost, "/api/v1/plus/restore", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.EqualValues(t, 0, r.data()["restored"])

	// Paid at the bank, return page lost: restore settles it server to server.
	_, err := fake.Decide(context.Background(), authority, true)
	require.NoError(t, err)
	r = e.doAt(t, "2026-09-23T10:06:00+03:30", http.MethodPost, "/api/v1/plus/restore", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.EqualValues(t, 1, r.data()["restored"])
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, uid))
}
