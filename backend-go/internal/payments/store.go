package payments

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // G505: the stores sign purchases with SHA1withRSA; we only verify their signature
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Store ids (the `store` of plus_store_products / plus_store_purchases and the client's billing channel).
const (
	StoreBazaar = "bazaar"
	StoreMyket  = "myket"
)

// Store billing errors.
var (
	// ErrStoreUnavailable is a store whose key is not configured.
	ErrStoreUnavailable = errors.New("payments: store billing unavailable")
	// ErrStoreSignature is a purchase whose signature does not verify with the store key.
	ErrStoreSignature = errors.New("payments: store purchase signature invalid")
	// ErrStorePurchase is a well-signed purchase this app must not accept (other package, not purchased, bad JSON).
	ErrStorePurchase = errors.New("payments: store purchase rejected")
)

// StorePurchase is a store purchase whose signature verified. Only these fields are trusted.
type StorePurchase struct {
	Store            string
	OrderID          string
	PurchaseToken    string
	ProductID        string // the SKU
	DeveloperPayload string
	PurchaseTime     time.Time
}

// StoreVerifier checks a purchase the Android store's billing library returned (Café Bazaar Poolakey, Myket IAB):
// the store signs the purchase JSON (`originalJson`) with SHA1withRSA, and the app's RSA public key from the
// store's developer panel verifies it. The key is public; it is not a secret.
//
// Server-to-server checks with the store API (Bazaar Pishkhan, Myket developer API) can wrap a verifier later; the
// signature check stays first either way.
type StoreVerifier struct {
	store       string
	packageName string
	key         *rsa.PublicKey
}

// NewStoreVerifier parses base64Key (the X.509 SubjectPublicKeyInfo the store panel shows). It returns nil, nil
// when base64Key is empty (store not configured).
func NewStoreVerifier(store, packageName, base64Key string) (*StoreVerifier, error) {
	base64Key = strings.Join(strings.Fields(base64Key), "")
	if base64Key == "" {
		return nil, nil
	}
	der, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("payments: %s RSA key is not base64: %w", store, err)
	}
	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("payments: %s RSA key: %w", store, err)
	}
	rk, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("payments: %s key is not RSA", store)
	}
	return &StoreVerifier{store: store, packageName: packageName, key: rk}, nil
}

// Store is the store id.
func (v *StoreVerifier) Store() string { return v.store }

// storeJSON is the purchase JSON both stores sign (Google IAB v3 shape).
type storeJSON struct {
	OrderID          string      `json:"orderId"`
	PurchaseToken    string      `json:"purchaseToken"`
	DeveloperPayload string      `json:"developerPayload"`
	PackageName      string      `json:"packageName"`
	ProductID        string      `json:"productId"`
	PurchaseTime     json.Number `json:"purchaseTime"`
	PurchaseState    *int        `json:"purchaseState"`
}

// Verify checks signature over signedData, then the package name and the purchase state, and returns the parsed
// purchase. signedData must be the exact JSON string the store returned (no re-encoding).
func (v *StoreVerifier) Verify(signedData, signature string) (StorePurchase, error) {
	if v == nil {
		return StorePurchase{}, ErrStoreUnavailable
	}
	if len(signedData) == 0 || len(signedData) > 8<<10 {
		return StorePurchase{}, fmt.Errorf("%w: signed data size", ErrStorePurchase)
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature))
	if err != nil {
		return StorePurchase{}, ErrStoreSignature
	}
	sum := sha1.Sum([]byte(signedData)) //nolint:gosec // G401: verifying the store's SHA1withRSA signature
	if err := rsa.VerifyPKCS1v15(v.key, crypto.SHA1, sum[:], sig); err != nil {
		return StorePurchase{}, ErrStoreSignature
	}
	var j storeJSON
	dec := json.NewDecoder(strings.NewReader(signedData))
	dec.UseNumber()
	if err := dec.Decode(&j); err != nil {
		return StorePurchase{}, fmt.Errorf("%w: JSON", ErrStorePurchase)
	}
	switch {
	case v.packageName != "" && j.PackageName != v.packageName:
		return StorePurchase{}, fmt.Errorf("%w: package %q", ErrStorePurchase, j.PackageName)
	case j.PurchaseState != nil && *j.PurchaseState != 0:
		return StorePurchase{}, fmt.Errorf("%w: purchase state %d", ErrStorePurchase, *j.PurchaseState)
	case !authorityPattern.MatchString(j.PurchaseToken) && !tokenPattern(j.PurchaseToken):
		return StorePurchase{}, fmt.Errorf("%w: purchase token", ErrStorePurchase)
	case j.ProductID == "" || len(j.ProductID) > 100:
		return StorePurchase{}, fmt.Errorf("%w: product id", ErrStorePurchase)
	}
	ms, err := j.PurchaseTime.Int64()
	if err != nil || ms <= 0 {
		return StorePurchase{}, fmt.Errorf("%w: purchase time", ErrStorePurchase)
	}
	return StorePurchase{
		Store: v.store, OrderID: clipTo(j.OrderID, 191), PurchaseToken: j.PurchaseToken, ProductID: j.ProductID,
		DeveloperPayload: j.DeveloperPayload, PurchaseTime: time.UnixMilli(ms),
	}, nil
}

// tokenPattern accepts the store tokens that carry '.', ':' or '=' (Bazaar/Myket tokens are opaque), up to 191.
func tokenPattern(s string) bool {
	if s == "" || len(s) > 191 {
		return false
	}
	for _, r := range s {
		ok := r == '-' || r == '_' || r == '.' || r == ':' || r == '=' || r == '+' || r == '/' ||
			(r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		if !ok {
			return false
		}
	}
	return true
}

func clipTo(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// StoreVerifiers holds the configured stores (missing key = not in the map).
type StoreVerifiers map[string]*StoreVerifier

// Get returns the verifier for store, or nil.
func (m StoreVerifiers) Get(store string) *StoreVerifier { return m[store] }
