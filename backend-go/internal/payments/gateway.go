package payments

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// Gateway wraps the active Provider with the checks every domain relies on: request validation, the return-URL
// allow-list, the provider-facing return URL, PAN masking and the structured payment log. It implements the
// domains' payment ports (plus.Gateway).
type Gateway struct {
	p            Provider
	callbackBase string // public API origin, no trailing slash
	returns      *AllowList
	logger       *slog.Logger
}

// NewGateway wraps p. callbackBase is the API's public origin; returns the allow-list of final web-app pages.
func NewGateway(p Provider, callbackBase string, returns *AllowList, logger *slog.Logger) *Gateway {
	return &Gateway{p: p, callbackBase: strings.TrimRight(callbackBase, "/"), returns: returns, logger: logger}
}

// Provider returns the wrapped provider.
func (g *Gateway) Provider() Provider { return g.p }

// Name is the provider id stored on invoices and receipts.
func (g *Gateway) Name() string { return g.p.Name() }

var (
	referencePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	authorityPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,191}$`)
)

// ReturnURL is the provider-facing callback: this API's return endpoint, carrying the order reference and the
// allow-listed final page (re-checked on the way back).
func (g *Gateway) ReturnURL(reference, next string) string {
	return g.callbackBase + "/api/v1/payments/" + url.PathEscape(g.p.Name()) + "/return?" +
		url.Values{"reference": {reference}, "next": {next}}.Encode()
}

// Create validates req, swaps its CallbackURL for the API's return URL and registers the payment.
func (g *Gateway) Create(ctx context.Context, req Request) (Session, error) {
	start := time.Now()
	switch {
	case !referencePattern.MatchString(req.Reference):
		return Session{}, fmt.Errorf("%w: reference", ErrInvalidRequest)
	case req.AmountRials == 0:
		return Session{}, fmt.Errorf("%w: amount must be positive", ErrInvalidRequest)
	}
	next, ok := g.returns.Match(req.CallbackURL)
	if !ok {
		g.logger.ErrorContext(ctx, "payments: callback URL is not on the return allow-list", "provider", g.Name(), "reference", req.Reference)
		return Session{}, fmt.Errorf("%w: callback URL not allowed", ErrInvalidRequest)
	}
	preq := req
	preq.CallbackURL = g.ReturnURL(req.Reference, next)
	sess, err := g.p.Create(ctx, preq)
	g.log(ctx, "create", start, err, "reference", req.Reference, "authority", sess.Authority, "amount_rials", req.AmountRials)
	if err != nil {
		return Session{}, err
	}
	if !authorityPattern.MatchString(sess.Authority) || sess.RedirectURL == "" {
		return Session{}, fmt.Errorf("payments: %s returned an invalid session", g.Name())
	}
	return sess, nil
}

// Verify asks the provider about req.Authority for the domain's stored amount. The verdict never comes from the
// callback parameters.
func (g *Gateway) Verify(ctx context.Context, req VerifyRequest) (Result, error) {
	start := time.Now()
	if !authorityPattern.MatchString(req.Authority) || req.AmountRials == 0 {
		return Result{}, fmt.Errorf("%w: authority or amount", ErrInvalidRequest)
	}
	res, err := g.p.Verify(ctx, req)
	if err == nil && res.Paid && res.RefID == "" {
		err = fmt.Errorf("payments: %s reported a payment without a reference", g.Name())
	}
	outcome := "declined"
	if res.Paid {
		outcome = "paid"
	}
	g.log(ctx, "verify", start, err, "authority", req.Authority, "amount_rials", req.AmountRials,
		"outcome", outcome, "settled_rials", res.AmountRials, "amount_matches", res.AmountRials == req.AmountRials, "ref_id", res.RefID)
	if err != nil {
		return Result{}, err
	}
	res.CardPAN = MaskPAN(res.CardPAN)
	return res, nil
}

// Refund refunds (part of) a verified payment. Providers without refunds return ErrRefundNotSupported.
func (g *Gateway) Refund(ctx context.Context, req RefundRequest) (RefundResult, error) {
	start := time.Now()
	if !authorityPattern.MatchString(req.Authority) || req.AmountRials == 0 {
		return RefundResult{}, fmt.Errorf("%w: authority or amount", ErrInvalidRequest)
	}
	res, err := g.p.Refund(ctx, req)
	g.log(ctx, "refund", start, err, "authority", req.Authority, "amount_rials", req.AmountRials, "refund_id", res.RefundID)
	return res, err
}

func (g *Gateway) log(ctx context.Context, op string, start time.Time, err error, attrs ...any) {
	attrs = append([]any{"provider", g.Name(), "duration_ms", time.Since(start).Milliseconds()}, attrs...)
	if err != nil && !errors.Is(err, ErrRefundNotSupported) {
		g.logger.WarnContext(ctx, "payments: "+op, append(attrs, "error", err.Error())...)
		return
	}
	g.logger.InfoContext(ctx, "payments: "+op, attrs...)
}

// MaskPAN keeps only the first 6 and last 4 digits of a card number; anything already masked passes through.
func MaskPAN(pan string) string {
	pan = strings.TrimSpace(pan)
	digits := 0
	for _, r := range pan {
		if unicode.IsDigit(r) {
			digits++
		}
	}
	if digits <= 10 { // already masked (502229******5995 has 10 digits) or not a PAN
		return pan
	}
	b := []rune(pan)
	seen := 0
	for i, r := range b {
		if !unicode.IsDigit(r) {
			continue
		}
		seen++
		if seen > 6 && seen <= digits-4 {
			b[i] = '*'
		}
	}
	return string(b)
}

// AllowList holds the web-app pages a payment may return to, compared on scheme, host and path (the query is
// dropped). The first entry is the fallback.
type AllowList struct {
	urls []*url.URL
}

// NewAllowList parses raw; unparsable or relative entries are skipped.
func NewAllowList(raw []string) *AllowList {
	a := &AllowList{}
	for _, r := range raw {
		u, err := url.Parse(strings.TrimSpace(r))
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
			continue
		}
		a.urls = append(a.urls, &url.URL{Scheme: u.Scheme, Host: u.Host, Path: u.Path})
	}
	return a
}

// Match returns the canonical allow-listed URL equal to raw (scheme, host, path), if any.
func (a *AllowList) Match(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	for _, w := range a.urls {
		if strings.EqualFold(u.Scheme, w.Scheme) && strings.EqualFold(u.Host, w.Host) && u.Path == w.Path && u.User == nil {
			return w.String(), true
		}
	}
	return "", false
}

// Resolve is Match, falling back to the first entry ("" when the list is empty).
func (a *AllowList) Resolve(raw string) string {
	if m, ok := a.Match(raw); ok {
		return m
	}
	if len(a.urls) == 0 {
		return ""
	}
	return a.urls[0].String()
}
