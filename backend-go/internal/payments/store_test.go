package payments_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // G505: the stores sign with SHA1withRSA
	"crypto/x509"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/payments"
)

func storeKey(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 1024) //nolint:gosec // G403: test key, the stores use 1024/1392-bit keys
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&k.PublicKey)
	require.NoError(t, err)
	return k, base64.StdEncoding.EncodeToString(der)
}

func sign(t *testing.T, k *rsa.PrivateKey, data string) string {
	t.Helper()
	sum := sha1.Sum([]byte(data)) //nolint:gosec // G401
	sig, err := rsa.SignPKCS1v15(rand.Reader, k, crypto.SHA1, sum[:])
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(sig)
}

func TestStoreVerifier(t *testing.T) {
	k, pub := storeKey(t)
	v, err := payments.NewStoreVerifier(payments.StoreBazaar, "web.ritme.app", pub)
	require.NoError(t, err)

	good := `{"orderId":"o-1","purchaseToken":"tok_ABC.123","developerPayload":"rp1-x","packageName":"web.ritme.app","productId":"mtest","purchaseTime":1762950000000,"purchaseState":0}`
	p, err := v.Verify(good, sign(t, k, good))
	require.NoError(t, err)
	assert.Equal(t, "tok_ABC.123", p.PurchaseToken)
	assert.Equal(t, "mtest", p.ProductID)
	assert.Equal(t, "rp1-x", p.DeveloperPayload)
	assert.Equal(t, int64(1762950000000), p.PurchaseTime.UnixMilli())

	_, err = v.Verify(good+" ", sign(t, k, good))
	assert.ErrorIs(t, err, payments.ErrStoreSignature, "the signature covers the exact bytes")

	other, _ := storeKey(t)
	_, err = v.Verify(good, sign(t, other, good))
	assert.ErrorIs(t, err, payments.ErrStoreSignature, "signed by another key")

	_, err = v.Verify(good, "not base64!!")
	assert.ErrorIs(t, err, payments.ErrStoreSignature)

	pkg := `{"orderId":"o-1","purchaseToken":"t","developerPayload":"","packageName":"com.other","productId":"mtest","purchaseTime":1,"purchaseState":0}`
	_, err = v.Verify(pkg, sign(t, k, pkg))
	assert.ErrorIs(t, err, payments.ErrStorePurchase, "another app's purchase")

	refunded := `{"purchaseToken":"t","packageName":"web.ritme.app","productId":"mtest","purchaseTime":1,"purchaseState":1}`
	_, err = v.Verify(refunded, sign(t, k, refunded))
	assert.ErrorIs(t, err, payments.ErrStorePurchase, "not in the purchased state")

	none, err := payments.NewStoreVerifier(payments.StoreMyket, "web.ritme.app", "")
	require.NoError(t, err)
	assert.Nil(t, none, "no key = store off")
	_, err = none.Verify(good, sign(t, k, good))
	assert.ErrorIs(t, err, payments.ErrStoreUnavailable)

	_, err = payments.NewStoreVerifier(payments.StoreBazaar, "", "AAAA")
	assert.Error(t, err)
}

// The Café Bazaar key from the developer panel parses (1392-bit RSA, X.509).
func TestStoreVerifier_BazaarPanelKey(t *testing.T) {
	const key = "MIHNMA0GCSqGSIb3DQEBAQUAA4G7ADCBtwKBrwDiBhR4d7x9iA/9LtGE+p5fJBbI9zFYzhGCW19YutmZPVl85RmAhsXp6DSaWYyFHngXu03UFHx7Pv2r9ErwDlNv+Ea5oNoEF1r0P5Wg07Q3nNJFosajKYto+Ch5z3Zx9cpNLTSgFqHfmFN8y0kJA0LTrz5cWTRlFHejO0mkjjPUjmCPmynxj6BndhmQ4qvOUijvlRAZ88t+H7Dzt8NGfjIOg/nvQkVH2a9viFORQCkCAwEAAQ=="
	v, err := payments.NewStoreVerifier(payments.StoreBazaar, "web.ritme.app", key)
	require.NoError(t, err)
	assert.NotNil(t, v)
}
