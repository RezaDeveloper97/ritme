package plus_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // G505: the stores sign with SHA1withRSA
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/payments"
	"github.com/ritme/backend-go/internal/plus"
)

type storeSigner struct{ key *rsa.PrivateKey }

func storeEnv(t *testing.T) (*env, storeSigner) {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 1024) //nolint:gosec // G403: test key
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&k.PublicKey)
	require.NoError(t, err)
	v, err := payments.NewStoreVerifier(payments.StoreBazaar, "web.ritme.app", base64.StdEncoding.EncodeToString(der))
	require.NoError(t, err)
	e := setup(t, plus.FakeGateway{})
	e.svc.WithStores(payments.StoreVerifiers{payments.StoreBazaar: v})
	return e, storeSigner{key: k}
}

// purchase is the verify body of a store-signed purchase.
func (s storeSigner) purchase(t *testing.T, token, sku, payload string, timeMs int64) string {
	t.Helper()
	data := fmt.Sprintf(`{"orderId":"o-%s","purchaseToken":"%s","developerPayload":"%s","packageName":"web.ritme.app","productId":"%s","purchaseTime":%d,"purchaseState":0}`,
		token, token, payload, sku, timeMs)
	sum := sha1.Sum([]byte(data)) //nolint:gosec // G401
	sig, err := rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA1, sum[:])
	require.NoError(t, err)
	b, err := json.Marshal(map[string]string{"store": "bazaar", "signed_data": data, "signature": base64.StdEncoding.EncodeToString(sig)})
	require.NoError(t, err)
	return string(b)
}

const t0 = int64(1758609000000) // 2026-09-23 ~10:00 Tehran

func TestStore_PlanPurchaseRenewalReplayAndOwnership(t *testing.T) {
	e, s := storeEnv(t)
	uid, tok := e.user(t, "09120000951")
	_, otherTok := e.user(t, "09120000952")

	r := e.doAt(t, now, http.MethodPost, "/api/v1/plus/store/intent", tok, "en", `{"store":"bazaar","sku":"mtest"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	payload, _ := r.data()["payload"].(string)
	assert.Equal(t, plus.StorePayload(uid), payload)
	assert.Equal(t, "plan", r.data()["kind"])

	body := s.purchase(t, "tok-1", "mtest", payload, t0)
	r = e.doAt(t, now, http.MethodPost, "/api/v1/plus/store/verify", tok, "en", body)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, false, r.data()["already_verified"])
	assert.Equal(t, "paid", r.obj("invoice")["status"])
	assert.Equal(t, "plus", r.obj("status")["tier"])

	// Five minutes, not a month.
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ? AND TIMESTAMPDIFF(MINUTE, starts_at, ends_at) = 5`, uid))

	// Replay: nothing added.
	r = e.doAt(t, now, http.MethodPost, "/api/v1/plus/store/verify", tok, "en", body)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, true, r.data()["already_verified"])
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, uid))

	// Another account cannot redeem it (payload of another user), nor a forged payload with the same token.
	r = e.doAt(t, now, http.MethodPost, "/api/v1/plus/store/verify", otherTok, "en", body)
	require.Equal(t, http.StatusConflict, r.status, r.raw)
	assert.Equal(t, "store_purchase_owned", r.body["error_code"])

	// Renewal (newer purchase_time, same token) through sync: one more period, queued after the first.
	renewal := s.purchase(t, "tok-1", "mtest", payload, t0+5*60*1000)
	var rb map[string]string
	require.NoError(t, json.Unmarshal([]byte(renewal), &rb))
	sync, _ := json.Marshal(map[string]any{"store": "bazaar", "purchases": []any{
		map[string]string{"signed_data": rb["signed_data"], "signature": rb["signature"]},
		map[string]string{"signed_data": rb["signed_data"], "signature": "AAAA"}, // tampered: skipped
	}})
	r = e.doAt(t, now, http.MethodPost, "/api/v1/plus/store/sync", tok, "en", string(sync))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.EqualValues(t, 1, r.data()["restored"])
	assert.EqualValues(t, 1, r.data()["rejected"])
	assert.Equal(t, 2, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, uid))
	assert.Equal(t, 2, e.count(t, `SELECT COUNT(*) FROM plus_receipts WHERE user_id = ? AND gateway = 'bazaar'`, uid))
}

func TestStore_RejectsBadSignatureUnknownSkuAndUnconfiguredStore(t *testing.T) {
	e, s := storeEnv(t)
	uid, tok := e.user(t, "09120000953")
	payload := plus.StorePayload(uid)

	var b map[string]string
	require.NoError(t, json.Unmarshal([]byte(s.purchase(t, "tok-2", "mtest", payload, t0)), &b))
	b["signed_data"] = b["signed_data"][:len(b["signed_data"])-1] + ` }` // altered after signing
	bad, _ := json.Marshal(b)
	r := e.doAt(t, now, http.MethodPost, "/api/v1/plus/store/verify", tok, "en", string(bad))
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "store_purchase_invalid", r.body["error_code"])

	r = e.doAt(t, now, http.MethodPost, "/api/v1/plus/store/verify", tok, "en", s.purchase(t, "tok-3", "unknown_sku", payload, t0))
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "store_product_unknown", r.body["error_code"])

	r = e.doAt(t, now, http.MethodPost, "/api/v1/plus/store/intent", tok, "en", `{"store":"myket","sku":"mtest"}`)
	require.Equal(t, http.StatusServiceUnavailable, r.status, r.raw)
	assert.Equal(t, "payment_unavailable", r.body["error_code"])
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, uid))
}

func TestStore_TrialSkuStartsTheOneTrial(t *testing.T) {
	e, s := storeEnv(t)
	uid, tok := e.user(t, "09120000954")
	payload := plus.StorePayload(uid)

	r := e.doAt(t, now, http.MethodPost, "/api/v1/plus/store/verify", tok, "en", s.purchase(t, "tok-t", "trial_subscription", payload, t0))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "trial", r.data()["kind"])
	assert.Equal(t, "trial", r.obj("status")["tier"])
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_trials WHERE user_id = ?`, uid))

	// The web trial is now used too, and a second store trial intent is refused.
	r = e.doAt(t, now, http.MethodPost, "/api/v1/plus/trial/start", tok, "en", "")
	assert.Equal(t, "trial_used", r.body["error_code"])
	r = e.doAt(t, now, http.MethodPost, "/api/v1/plus/store/intent", tok, "en", `{"store":"bazaar","sku":"trial_subscription"}`)
	assert.Equal(t, "trial_used", r.body["error_code"])
}
