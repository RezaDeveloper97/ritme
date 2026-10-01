// Package payments is the payment-gateway adapter (B-N2-05): one Provider interface, a registry that picks the
// provider from config (PAYMENT_PROVIDER), and a Gateway wrapper that every domain talks to.
//
// Providers:
//   - fake (default outside production): a local TEST gateway. Create sends the user to a success/fail page served
//     by this API (/api/v1/payments/fake/pay/{authority}); Verify asks the fake's own state (Redis), never the
//     callback. It supports refunds. PAYMENT_PROVIDER=fake with APP_ENV=production is refused at start-up.
//   - zarinpal: the Zarinpal v4 REST web gateway (payment.zarinpal.com/pg/v4). Disabled — payments answer 503 —
//     until ZARINPAL_MERCHANT_ID exists in the server's .env. Refunds are not supported through this adapter.
//   - none (default in production): no provider; checkout/verify answer 503 payment_unavailable.
//
// Café Bazaar and Myket in-app billing only work inside their Android stores, so they are out of scope (the web
// app and the Android WebView shell pay through a web bank gateway).
//
// Flow: Gateway.Create → the user pays at RedirectURL → the provider redirects the browser to
// /api/v1/payments/{provider}/return, which only normalizes the parameters and 303-redirects to an allow-listed
// web-app page (?reference&authority&status) — it changes no state. The web app then calls the domain's
// authenticated verify endpoint, which settles server-to-server through Gateway.Verify with the amount stored on
// the invoice (never a query parameter).
//
// Payment log: every Create/Verify/Refund writes one structured slog record ("payments: <op>") with provider,
// reference, authority, amount, outcome and duration. Durable state lives with the domain (plus_invoices /
// plus_receipts). Card numbers (other than the provider's masked PAN, which is never logged), phone numbers and
// merchant ids are never logged.
package payments

import (
	"context"
	"errors"
)

// Provider ids (PAYMENT_PROVIDER values).
const (
	FakeName     = "fake"
	ZarinpalName = "zarinpal"
)

// Currency is the only currency the adapter charges: integer Iranian rials.
const Currency = "IRR"

// Errors.
var (
	// ErrRefundNotSupported is returned by providers whose API this adapter cannot refund through.
	ErrRefundNotSupported = errors.New("payments: refund not supported by this provider")
	// ErrInvalidRequest is a request the adapter refuses before calling the provider.
	ErrInvalidRequest = errors.New("payments: invalid request")
	// ErrRefundRejected is a refund the provider refused (unpaid, already refunded, over the paid amount).
	ErrRefundRejected = errors.New("payments: refund rejected")
)

// Request is one payment to register. Amounts are rials; domains always send the total they computed.
type Request struct {
	Reference   string // the domain's public order reference (unique per payment)
	AmountRials uint64
	Description string
	// CallbackURL is the web-app page the user finally returns to; it must be on the PLUS_CALLBACK_URL /
	// PAYMENT_RETURN_URLS allow-list. Providers receive the API's own return URL here instead.
	CallbackURL string
}

// Session is where to send the user.
type Session struct {
	Authority   string // provider payment id; the user brings it back to verify
	RedirectURL string
}

// VerifyRequest asks the provider whether Authority was paid AmountRials (the domain's stored total).
type VerifyRequest struct {
	Authority   string
	AmountRials uint64
	// Callback holds the parameters the user came back with (lower-case keys), or nil on a re-check. Real providers
	// and the fake ignore it for the verdict: Verify always asks the provider.
	Callback map[string]string
}

// Result is the provider's verdict. A declined/cancelled payment is Result{Paid: false}, not an error; errors are
// transport or provider failures (the payment stays pending and can be verified again). Verify is idempotent: a
// second verify of a settled payment returns the same RefID.
type Result struct {
	Paid        bool
	RefID       string // the provider's unique payment reference
	CardPAN     string // masked (first 6 / last 4), when the provider reports it
	AmountRials uint64 // what the provider settled
}

// RefundRequest refunds (part of) a verified payment.
type RefundRequest struct {
	Authority   string
	RefID       string
	AmountRials uint64
	Reason      string
}

// RefundResult is an accepted refund.
type RefundResult struct {
	RefundID    string
	AmountRials uint64
}

// Provider is one gateway implementation.
type Provider interface {
	Name() string
	Create(ctx context.Context, req Request) (Session, error)
	Verify(ctx context.Context, req VerifyRequest) (Result, error)
	Refund(ctx context.Context, req RefundRequest) (RefundResult, error)
	// ReturnParams extracts the payment id and the OK/NOK status from the query the provider sent the browser back
	// with (keys lower-cased). The values are untrusted hints for the web app, never a verdict.
	ReturnParams(query map[string]string) (authority, status string)
}
