package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Zarinpal gateway origins (docs: https://www.zarinpal.com/docs/paymentGateway/).
const (
	ZarinpalBaseURL        = "https://payment.zarinpal.com"
	ZarinpalSandboxBaseURL = "https://sandbox.zarinpal.com"
)

// Zarinpal result codes this adapter acts on.
const (
	zpOK              = 100 // request accepted / payment verified
	zpAlreadyVerified = 101 // verify replay: already verified, same ref_id
)

// zpDeclined are verify errors that mean "not paid" (the payment stays unpaid; no retry will change it):
// -50 amount differs from the session, -51 not paid / cancelled, -53 session of another merchant,
// -54 invalid authority, -55 transaction not found.
var zpDeclined = map[int]bool{-50: true, -51: true, -53: true, -54: true, -55: true}

// maxResponse caps how much of a gateway response is read.
const maxResponse = 64 << 10

// Zarinpal is the Zarinpal v4 REST provider:
//
//	POST {base}/pg/v4/payment/request.json {merchant_id, amount, currency, description, callback_url, metadata}
//	  → {data: {code: 100, authority}, errors: []}; the user pays at {base}/pg/StartPay/{authority}
//	callback: {callback_url}&Authority=…&Status=OK|NOK
//	POST {base}/pg/v4/payment/verify.json {merchant_id, amount, authority}
//	  → {data: {code: 100|101, ref_id, card_pan, …}} or {data: [], errors: {code: -51, message}}
//
// Verify sends the domain's stored amount; Zarinpal answers -50 when it differs from the session, so the settled
// amount reported back is always the verified one. Refunds go through Zarinpal's separate GraphQL API with an
// access token, which this adapter does not hold: Refund returns ErrRefundNotSupported (refund from the panel).
type Zarinpal struct {
	BaseURL    string
	merchantID string
	client     *http.Client
}

// NewZarinpal builds the provider. baseURL "" means production (or the sandbox when sandbox is true).
func NewZarinpal(merchantID, baseURL string, sandbox bool, client *http.Client) *Zarinpal {
	if baseURL == "" {
		baseURL = ZarinpalBaseURL
		if sandbox {
			baseURL = ZarinpalSandboxBaseURL
		}
	}
	return &Zarinpal{BaseURL: strings.TrimRight(baseURL, "/"), merchantID: merchantID, client: client}
}

// Name implements Provider.
func (*Zarinpal) Name() string { return ZarinpalName }

type zpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type zpRequestData struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Authority string `json:"authority"`
}

type zpVerifyData struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	RefID   json.Number `json:"ref_id"`
	CardPAN string      `json:"card_pan"`
}

// Create implements Provider. No PII is sent (metadata carries only the order reference).
func (z *Zarinpal) Create(ctx context.Context, req Request) (Session, error) {
	desc := req.Description
	if desc == "" {
		desc = req.Reference
	}
	var data zpRequestData
	zerr, err := z.call(ctx, "/pg/v4/payment/request.json", map[string]any{
		"merchant_id": z.merchantID, "amount": req.AmountRials, "currency": Currency,
		"description": desc, "callback_url": req.CallbackURL,
		"metadata": map[string]string{"order_id": req.Reference},
	}, &data)
	if err != nil {
		return Session{}, err
	}
	if zerr != nil || data.Code != zpOK || data.Authority == "" {
		return Session{}, fmt.Errorf("zarinpal: request rejected (%s)", describe(zerr, data.Code))
	}
	return Session{Authority: data.Authority, RedirectURL: z.BaseURL + "/pg/StartPay/" + url.PathEscape(data.Authority)}, nil
}

// Verify implements Provider. It always asks Zarinpal (server to server), whatever the callback said.
func (z *Zarinpal) Verify(ctx context.Context, req VerifyRequest) (Result, error) {
	var data zpVerifyData
	zerr, err := z.call(ctx, "/pg/v4/payment/verify.json", map[string]any{
		"merchant_id": z.merchantID, "amount": req.AmountRials, "authority": req.Authority,
	}, &data)
	switch {
	case err != nil:
		return Result{}, err
	case zerr != nil && zpDeclined[zerr.Code]:
		return Result{}, nil
	case zerr != nil:
		return Result{}, fmt.Errorf("zarinpal: verify failed (%s)", describe(zerr, 0))
	case data.Code != zpOK && data.Code != zpAlreadyVerified:
		return Result{}, fmt.Errorf("zarinpal: verify failed (%s)", describe(nil, data.Code))
	case data.RefID.String() == "" || data.RefID.String() == "0":
		return Result{}, errors.New("zarinpal: verified without ref_id")
	}
	return Result{Paid: true, RefID: data.RefID.String(), CardPAN: MaskPAN(data.CardPAN), AmountRials: req.AmountRials}, nil
}

// Refund implements Provider: not supported (see the type doc).
func (*Zarinpal) Refund(context.Context, RefundRequest) (RefundResult, error) {
	return RefundResult{}, ErrRefundNotSupported
}

// ReturnParams implements Provider: Zarinpal appends Authority and Status.
func (*Zarinpal) ReturnParams(q map[string]string) (string, string) {
	return q["authority"], q["status"]
}

// call POSTs body as JSON and decodes {data, errors}. data is an object on success and [] on failure; errors is []
// on success and an object on failure. Any HTTP status is accepted as long as the body has that shape.
func (z *Zarinpal) call(ctx context.Context, path string, body any, data any) (*zpError, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("zarinpal: encode: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, z.BaseURL+path, bytes.NewReader(raw))
	if err != nil {
		return nil, errors.New("zarinpal: build request failed")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := z.client.Do(req)
	if err != nil {
		// The error text carries the URL only (no merchant id: it is in the body).
		return nil, fmt.Errorf("zarinpal: transport: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse))
	if err != nil {
		return nil, fmt.Errorf("zarinpal: read: %w", err)
	}
	var env struct {
		Data   json.RawMessage `json:"data"`
		Errors json.RawMessage `json:"errors"`
	}
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, fmt.Errorf("zarinpal: HTTP %d, not JSON", resp.StatusCode)
	}
	if isObject(env.Errors) {
		var zerr zpError
		if err := json.Unmarshal(env.Errors, &zerr); err != nil {
			return nil, fmt.Errorf("zarinpal: HTTP %d, bad errors object", resp.StatusCode)
		}
		return &zerr, nil
	}
	if !isObject(env.Data) {
		return nil, fmt.Errorf("zarinpal: HTTP %d, no data", resp.StatusCode)
	}
	if err := json.Unmarshal(env.Data, data); err != nil {
		return nil, fmt.Errorf("zarinpal: HTTP %d, bad data object", resp.StatusCode)
	}
	return nil, nil
}

func isObject(raw json.RawMessage) bool {
	t := bytes.TrimSpace(raw)
	return len(t) > 0 && t[0] == '{'
}

func describe(zerr *zpError, code int) string {
	if zerr != nil {
		return fmt.Sprintf("code %d: %s", zerr.Code, clip(zerr.Message))
	}
	return fmt.Sprintf("code %d", code)
}

func clip(s string) string {
	if len(s) > 120 {
		return s[:120] + "…"
	}
	return s
}
