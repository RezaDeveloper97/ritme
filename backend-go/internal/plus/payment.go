package plus

import (
	"context"
	"net/url"
	"strings"
)

// Gateway is the payment port checkout and verify talk to. B-N2-05 builds the adapter package (the configurable
// fake provider with its local success/fail page, and one real Iranian web bank gateway); this package only owns
// the contract and a minimal in-package FakeGateway so the domain runs and is tested without network.
//
// Rules every adapter must keep:
//   - amounts are rials; the domain always sends the invoice total it computed (never a client value);
//   - Verify must ask the provider (never trust callback parameters alone for a real provider) and report the amount
//     the provider actually settled, so the domain can reject a mismatch;
//   - RefID is the provider's unique payment reference (the domain stores it with UNIQUE(gateway, ref_id));
//   - errors are transport/provider failures (the invoice stays pending and can be re-verified); a declined or
//     cancelled payment is VerifyResult{Paid: false}, not an error.
type Gateway interface {
	// Name is the provider id stored on invoices and receipts ("fake", …).
	Name() string
	// Create registers a payment and returns where to send the user.
	Create(ctx context.Context, req PaymentRequest) (PaymentSession, error)
	// Verify settles a payment the user came back from.
	Verify(ctx context.Context, req VerifyRequest) (VerifyResult, error)
}

// PaymentRequest is one invoice to charge.
type PaymentRequest struct {
	Reference   string // the invoice's public reference
	AmountRials uint64
	Description string
	CallbackURL string // the web app's return page
}

// PaymentSession is the provider's answer to Create.
type PaymentSession struct {
	Authority   string // provider payment id; the user brings it back to verify
	RedirectURL string
}

// VerifyRequest asks the provider whether Authority was paid AmountRials.
type VerifyRequest struct {
	Authority   string
	AmountRials uint64
	// Callback holds the query parameters the provider sent the user back with (lower-case keys), or nil when the
	// domain re-checks a payment on its own (restore).
	Callback map[string]string
}

// VerifyResult is the provider's verdict.
type VerifyResult struct {
	Paid        bool
	RefID       string
	CardPAN     string // masked, when the provider reports it
	AmountRials uint64 // what the provider settled
}

// FakeGateway is the in-package fake provider: stateless and deterministic. Create sends the user straight back to
// the callback with status=OK; Verify reports paid exactly when the callback says status=OK, for the requested
// amount. It never moves money — production refuses to run without a real provider (see the routes file).
type FakeGateway struct{}

// FakeName is FakeGateway's provider id.
const FakeName = "fake"

// Name implements Gateway.
func (FakeGateway) Name() string { return FakeName }

// Create implements Gateway.
func (FakeGateway) Create(_ context.Context, req PaymentRequest) (PaymentSession, error) {
	authority := "FAKE-" + req.Reference
	return PaymentSession{Authority: authority, RedirectURL: withQuery(req.CallbackURL, url.Values{
		"reference": {req.Reference}, "authority": {authority}, "status": {"OK"},
	})}, nil
}

// Verify implements Gateway.
func (FakeGateway) Verify(_ context.Context, req VerifyRequest) (VerifyResult, error) {
	if !strings.EqualFold(req.Callback["status"], "OK") || !strings.HasPrefix(req.Authority, "FAKE-") {
		return VerifyResult{}, nil
	}
	return VerifyResult{
		Paid:        true,
		RefID:       "FAKE-REF-" + strings.TrimPrefix(req.Authority, "FAKE-"),
		AmountRials: req.AmountRials,
	}, nil
}

// withQuery appends v to raw's query string (raw may already carry one).
func withQuery(raw string, v url.Values) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	for k, vals := range v {
		for _, x := range vals {
			q.Set(k, x)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}
