package payments_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/payments"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

const (
	apiBase   = "https://api.test"
	returnURL = "https://web.test/plus/return"
	ref       = "RPABCDEFGHIJKLMNOP"
)

var quiet = slog.New(slog.NewJSONHandler(io.Discard, nil))

func newFakeGateway(t *testing.T) (*payments.Gateway, *payments.Fake) {
	t.Helper()
	gw := payments.New(payments.Deps{
		App:       config.App{Env: "testing", URL: apiBase},
		Config:    config.Payment{Provider: "fake", ReturnURLs: []string{returnURL, "https://web.test/shop/return"}},
		Logger:    quiet,
		FakeStore: payments.NewMemoryFakeStore(),
	})
	require.NotNil(t, gw)
	fake, ok := gw.Provider().(*payments.Fake)
	require.True(t, ok)
	return gw, fake
}

func create(t *testing.T, gw *payments.Gateway, amount uint64) payments.Session {
	t.Helper()
	sess, err := gw.Create(context.Background(), payments.Request{Reference: ref, AmountRials: amount, Description: "Ritme Plus plus_1m", CallbackURL: returnURL})
	require.NoError(t, err)
	return sess
}

// ---------------------------------------------------------------- registry

func TestNew_ProviderSelection(t *testing.T) {
	deps := func(env, provider, merchant string) payments.Deps {
		return payments.Deps{
			App:    config.App{Env: env, URL: apiBase},
			Config: config.Payment{Provider: provider, ReturnURLs: []string{returnURL}, Zarinpal: config.Zarinpal{MerchantID: merchant}},
			Logger: quiet, FakeStore: payments.NewMemoryFakeStore(),
		}
	}
	assert.Equal(t, "fake", payments.New(deps("local", "", "")).Name(), "fake is the default outside production")
	assert.Nil(t, payments.New(deps("production", "", "")), "no provider in production by default → 503")
	assert.Nil(t, payments.New(deps("production", "fake", "")), "the fake is never built in production")
	assert.Nil(t, payments.New(deps("staging", "none", "")))
	assert.Nil(t, payments.New(deps("staging", "zarinpal", "")), "zarinpal stays disabled until the merchant id exists")
	assert.Equal(t, "zarinpal", payments.New(deps("production", "zarinpal", "m-123")).Name())
	assert.Nil(t, payments.New(payments.Deps{App: config.App{Env: "local"}, Logger: quiet}), "fake without Redis → unavailable")
}

// ---------------------------------------------------------------- gateway checks

func TestGateway_CreateValidatesAndRewritesCallback(t *testing.T) {
	gw, fake := newFakeGateway(t)
	ctx := context.Background()

	_, err := gw.Create(ctx, payments.Request{Reference: ref, AmountRials: 0, CallbackURL: returnURL})
	require.ErrorIs(t, err, payments.ErrInvalidRequest)
	_, err = gw.Create(ctx, payments.Request{Reference: "bad ref/..", AmountRials: 10, CallbackURL: returnURL})
	require.ErrorIs(t, err, payments.ErrInvalidRequest)
	for _, evil := range []string{"https://evil.test/plus/return", "https://web.test/other", "//evil.test", "https://user@web.test/plus/return"} {
		_, err = gw.Create(ctx, payments.Request{Reference: ref, AmountRials: 10, CallbackURL: evil})
		require.ErrorIs(t, err, payments.ErrInvalidRequest, evil)
	}

	sess := create(t, gw, 990000)
	assert.Equal(t, "FAKE-"+ref, sess.Authority)
	assert.Equal(t, apiBase+"/api/v1/payments/fake/pay/FAKE-"+ref, sess.RedirectURL)
	p, ok, err := fake.Payment(ctx, sess.Authority)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, apiBase+"/api/v1/payments/fake/return?next=https%3A%2F%2Fweb.test%2Fplus%2Freturn&reference="+ref, p.CallbackURL,
		"the provider gets the API's return URL, never the raw caller URL")

	// A second payment with the same reference is refused (one payment per order reference).
	_, err = gw.Create(ctx, payments.Request{Reference: ref, AmountRials: 990000, CallbackURL: returnURL})
	require.ErrorIs(t, err, payments.ErrInvalidRequest)
}

func TestMaskPAN(t *testing.T) {
	assert.Equal(t, "502229******5995", payments.MaskPAN("502229******5995"))
	assert.Equal(t, "502229******5995", payments.MaskPAN("5022291234565995"))
	assert.Equal(t, "6037-99**-****-1234", payments.MaskPAN("6037-9912-3456-1234"))
	assert.Empty(t, payments.MaskPAN(""))
}

// ---------------------------------------------------------------- fake provider

func TestFake_VerifyAsksItsOwnStateNotTheCallback(t *testing.T) {
	gw, fake := newFakeGateway(t)
	ctx := context.Background()
	sess := create(t, gw, 990000)

	// Before the tester decides, a forged status=OK callback does not pay.
	res, err := gw.Verify(ctx, payments.VerifyRequest{Authority: sess.Authority, AmountRials: 990000, Callback: map[string]string{"status": "OK"}})
	require.NoError(t, err)
	assert.False(t, res.Paid)

	next, err := fake.Decide(ctx, sess.Authority, true)
	require.NoError(t, err)
	u, _ := url.Parse(next)
	assert.Equal(t, "/api/v1/payments/fake/return", u.Path)
	assert.Equal(t, "OK", u.Query().Get("status"))
	assert.Equal(t, sess.Authority, u.Query().Get("authority"))

	// Paid, whatever the callback claims; replays return the same reference (idempotent verify).
	for _, cb := range []map[string]string{{"status": "NOK"}, nil, {"status": "OK"}} {
		res, err = gw.Verify(ctx, payments.VerifyRequest{Authority: sess.Authority, AmountRials: 990000, Callback: cb})
		require.NoError(t, err)
		assert.Equal(t, payments.Result{Paid: true, RefID: "FAKE-REF-" + ref, AmountRials: 990000}, res)
	}

	// A second decision does not flip a settled payment.
	next, err = fake.Decide(ctx, sess.Authority, false)
	require.NoError(t, err)
	assert.Contains(t, next, "status=OK")
}

func TestFake_AmountTamperIsReported(t *testing.T) {
	gw, fake := newFakeGateway(t)
	ctx := context.Background()
	sess := create(t, gw, 990000)
	_, err := fake.Decide(ctx, sess.Authority, true)
	require.NoError(t, err)

	// Verifying another amount: the provider reports what it actually settled, so the domain sees the mismatch.
	res, err := gw.Verify(ctx, payments.VerifyRequest{Authority: sess.Authority, AmountRials: 1000})
	require.NoError(t, err)
	assert.True(t, res.Paid)
	assert.EqualValues(t, 990000, res.AmountRials)
}

func TestFake_DeclinedAndUnknown(t *testing.T) {
	gw, fake := newFakeGateway(t)
	ctx := context.Background()
	sess := create(t, gw, 990000)
	next, err := fake.Decide(ctx, sess.Authority, false)
	require.NoError(t, err)
	assert.Contains(t, next, "status=NOK")
	res, err := gw.Verify(ctx, payments.VerifyRequest{Authority: sess.Authority, AmountRials: 990000})
	require.NoError(t, err)
	assert.False(t, res.Paid)

	res, err = gw.Verify(ctx, payments.VerifyRequest{Authority: "FAKE-RPZZZZZZZZZZZZZZZZ", AmountRials: 990000, Callback: map[string]string{"status": "OK"}})
	require.NoError(t, err)
	assert.False(t, res.Paid, "an authority the fake never issued is never paid")
	_, err = fake.Decide(ctx, "FAKE-RPZZZZZZZZZZZZZZZZ", true)
	require.ErrorIs(t, err, payments.ErrInvalidRequest)
	_, err = gw.Verify(ctx, payments.VerifyRequest{Authority: "x y", AmountRials: 1})
	require.ErrorIs(t, err, payments.ErrInvalidRequest)
}

func TestFake_Refund(t *testing.T) {
	gw, fake := newFakeGateway(t)
	ctx := context.Background()
	sess := create(t, gw, 990000)

	_, err := gw.Refund(ctx, payments.RefundRequest{Authority: sess.Authority, AmountRials: 1000})
	require.ErrorIs(t, err, payments.ErrRefundRejected, "unpaid")

	_, err = fake.Decide(ctx, sess.Authority, true)
	require.NoError(t, err)
	r, err := gw.Refund(ctx, payments.RefundRequest{Authority: sess.Authority, RefID: "FAKE-REF-" + ref, AmountRials: 400000})
	require.NoError(t, err)
	assert.Equal(t, payments.RefundResult{RefundID: "FAKE-RFD-" + ref + "-1", AmountRials: 400000}, r)
	_, err = gw.Refund(ctx, payments.RefundRequest{Authority: sess.Authority, AmountRials: 590001})
	require.ErrorIs(t, err, payments.ErrRefundRejected, "more than what is left")
	_, err = gw.Refund(ctx, payments.RefundRequest{Authority: sess.Authority, RefID: "OTHER", AmountRials: 1})
	require.ErrorIs(t, err, payments.ErrRefundRejected, "wrong bank reference")
	_, err = gw.Refund(ctx, payments.RefundRequest{Authority: sess.Authority, AmountRials: 590000})
	require.NoError(t, err)
	_, err = gw.Refund(ctx, payments.RefundRequest{Authority: sess.Authority, AmountRials: 1})
	require.ErrorIs(t, err, payments.ErrRefundRejected, "fully refunded")
}

func TestRedisFakeStore_RoundTrip(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	c := cache.NewFromClient(rdb, "ritme-go:")
	gw := payments.New(payments.Deps{
		App: config.App{Env: "staging", URL: apiBase}, Config: config.Payment{ReturnURLs: []string{returnURL}},
		Cache: c, Logger: quiet,
	})
	require.NotNil(t, gw)
	sess := create(t, gw, 5000)
	assert.True(t, mr.Exists("ritme-go:payments:fake:"+sess.Authority))
	assert.Positive(t, mr.TTL("ritme-go:payments:fake:"+sess.Authority))

	// A second process (another Gateway on the same Redis) sees the decision.
	other := payments.New(payments.Deps{App: config.App{Env: "staging", URL: apiBase}, Config: config.Payment{ReturnURLs: []string{returnURL}}, Cache: c, Logger: quiet})
	_, err := other.Provider().(*payments.Fake).Decide(context.Background(), sess.Authority, true)
	require.NoError(t, err)
	res, err := gw.Verify(context.Background(), payments.VerifyRequest{Authority: sess.Authority, AmountRials: 5000})
	require.NoError(t, err)
	assert.True(t, res.Paid)
}

// ---------------------------------------------------------------- browser routes

func app(gw *payments.Gateway) *fiber.App {
	a := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	a.Get("/api/v1/payments/:provider/return", gw.Return)
	if fh := payments.FakeHandlersFor(gw); fh != nil {
		a.Get("/api/v1/payments/fake/pay/:authority", fh.Page)
		a.Post("/api/v1/payments/fake/pay/:authority", fh.Decide)
	}
	return a
}

func send(t *testing.T, a *fiber.App, method, target, form string) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(form))
	if form != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := a.Test(req)
	require.NoError(t, err)
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func TestReturn_NormalizesAndAllowListsTheRedirect(t *testing.T) {
	gw, _ := newFakeGateway(t)
	a := app(gw)
	cases := []struct{ query, want string }{
		{"?reference=" + ref + "&next=https%3A%2F%2Fweb.test%2Fplus%2Freturn&authority=FAKE-" + ref + "&status=OK",
			returnURL + "?authority=FAKE-" + ref + "&reference=" + ref + "&status=OK"},
		// Another allow-listed page is honored.
		{"?next=https%3A%2F%2Fweb.test%2Fshop%2Freturn&status=nok", "https://web.test/shop/return?status=NOK"},
		// Open-redirect attempts fall back to the first allowed page; junk parameters are dropped.
		{"?next=https%3A%2F%2Fevil.test%2Fplus%2Freturn&reference=%3Cscript%3E&authority=a%22b&status=OK", returnURL + "?status=OK"},
		{"?next=%2F%2Fevil.test&status=weird", returnURL + "?status=NOK"},
		{"?next=https%3A%2F%2Fweb.test%2Fplus%2Freturn%3Fx%3Dhttps%3A%2F%2Fevil.test", returnURL + "?status=NOK"},
	}
	for _, c := range cases {
		resp, _ := send(t, a, http.MethodGet, "/api/v1/payments/fake/return"+c.query, "")
		require.Equal(t, http.StatusSeeOther, resp.StatusCode, c.query)
		assert.Equal(t, c.want, resp.Header.Get("Location"), c.query)
		assert.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
		assert.Equal(t, "no-referrer", resp.Header.Get("Referrer-Policy"))
	}
	resp, _ := send(t, a, http.MethodGet, "/api/v1/payments/zarinpal/return?status=OK", "")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "only the active provider answers")
}

func TestReturn_ZarinpalParameters(t *testing.T) {
	gw := payments.New(payments.Deps{
		App: config.App{Env: "production", URL: apiBase}, Logger: quiet,
		Config: config.Payment{Provider: "zarinpal", ReturnURLs: []string{returnURL}, Zarinpal: config.Zarinpal{MerchantID: "m"}},
	})
	resp, _ := send(t, app(gw), http.MethodGet, "/api/v1/payments/zarinpal/return?reference="+ref+"&next=https%3A%2F%2Fweb.test%2Fplus%2Freturn&Authority=A00000000000000000000000000217885159&Status=OK", "")
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Equal(t, returnURL+"?authority=A00000000000000000000000000217885159&reference="+ref+"&status=OK", resp.Header.Get("Location"))
	assert.Nil(t, payments.FakeHandlersFor(gw), "no TEST page with a real provider")
}

func TestFakePage_DecideFlow(t *testing.T) {
	gw, _ := newFakeGateway(t)
	a := app(gw)
	sess := create(t, gw, 1234000)
	path := strings.TrimPrefix(sess.RedirectURL, apiBase)

	resp, body := send(t, a, http.MethodGet, path, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, "TEST — FAKE GATEWAY")
	assert.Contains(t, body, "1,234,000 IRR")
	assert.Contains(t, body, ref)
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "frame-ancestors 'none'")

	resp, _ = send(t, a, http.MethodPost, path, "decision=maybe")
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	resp, _ = send(t, a, http.MethodGet, "/api/v1/payments/fake/pay/FAKE-RPZZZZZZZZZZZZZZZZ", "")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp, _ = send(t, a, http.MethodPost, "/api/v1/payments/fake/pay/FAKE-RPZZZZZZZZZZZZZZZZ", "decision=success")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	resp, _ = send(t, a, http.MethodPost, path, "decision=success")
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	loc := resp.Header.Get("Location")
	assert.True(t, strings.HasPrefix(loc, apiBase+"/api/v1/payments/fake/return?"), loc)

	// Follow the return hop: lands on the web app with the normalized parameters.
	resp, _ = send(t, a, http.MethodGet, strings.TrimPrefix(loc, apiBase), "")
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Equal(t, returnURL+"?authority=FAKE-"+ref+"&reference="+ref+"&status=OK", resp.Header.Get("Location"))
}

// ---------------------------------------------------------------- zarinpal (httptest only, never the network)

type zpServer struct {
	mu       sync.Mutex
	requests []map[string]any
	paths    []string
	respond  func(path string, body map[string]any) (int, string)
}

func newZarinpal(t *testing.T, respond func(path string, body map[string]any) (int, string)) (*payments.Gateway, *zpServer) {
	t.Helper()
	s := &zpServer{respond: respond}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		s.mu.Lock()
		s.requests = append(s.requests, body)
		s.paths = append(s.paths, r.Method+" "+r.URL.Path+" "+r.Header.Get("Content-Type"))
		s.mu.Unlock()
		status, out := s.respond(r.URL.Path, body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, out)
	}))
	t.Cleanup(srv.Close)
	gw := payments.New(payments.Deps{
		App: config.App{Env: "production", URL: apiBase}, Logger: quiet, HTTPClient: srv.Client(),
		Config: config.Payment{Provider: "zarinpal", ReturnURLs: []string{returnURL},
			Zarinpal: config.Zarinpal{MerchantID: "11111111-2222-3333-4444-555555555555", BaseURL: srv.URL}},
	})
	require.NotNil(t, gw)
	return gw, s
}

const zpAuthority = "A00000000000000000000000000217885159"

func TestZarinpal_CreateRequestShape(t *testing.T) {
	gw, s := newZarinpal(t, func(string, map[string]any) (int, string) {
		return 200, `{"data":{"code":100,"message":"Success","authority":"` + zpAuthority + `","fee_type":"Merchant","fee":100},"errors":[]}`
	})
	sess, err := gw.Create(context.Background(), payments.Request{Reference: ref, AmountRials: 2607000, Description: "Ritme Plus plus_3m", CallbackURL: returnURL})
	require.NoError(t, err)
	assert.Equal(t, zpAuthority, sess.Authority)
	assert.True(t, strings.HasSuffix(sess.RedirectURL, "/pg/StartPay/"+zpAuthority), sess.RedirectURL)

	require.Len(t, s.requests, 1)
	assert.Equal(t, "POST /pg/v4/payment/request.json application/json", s.paths[0])
	body := s.requests[0]
	assert.Equal(t, "11111111-2222-3333-4444-555555555555", body["merchant_id"])
	assert.EqualValues(t, 2607000, body["amount"], "the domain's amount, in rials")
	assert.Equal(t, "IRR", body["currency"])
	assert.Equal(t, apiBase+"/api/v1/payments/zarinpal/return?next=https%3A%2F%2Fweb.test%2Fplus%2Freturn&reference="+ref, body["callback_url"])
	assert.Equal(t, map[string]any{"order_id": ref}, body["metadata"], "no mobile/email (PII) is sent")
}

func TestZarinpal_CreateRejected(t *testing.T) {
	gw, _ := newZarinpal(t, func(string, map[string]any) (int, string) {
		return 422, `{"data":[],"errors":{"code":-9,"message":"The input params invalid, validation error.","validations":[]}}`
	})
	_, err := gw.Create(context.Background(), payments.Request{Reference: ref, AmountRials: 1000, CallbackURL: returnURL})
	require.ErrorContains(t, err, "code -9")
	assert.NotContains(t, err.Error(), "11111111", "the merchant id never ends up in errors/logs")
}

func TestZarinpal_VerifyOutcomes(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		body    string
		want    payments.Result
		wantErr string
	}{
		{"paid", 200, `{"data":{"code":100,"message":"Verified","card_hash":"x","card_pan":"502229******5995","ref_id":201,"fee_type":"Merchant","fee":0},"errors":[]}`,
			payments.Result{Paid: true, RefID: "201", CardPAN: "502229******5995", AmountRials: 2607000}, ""},
		{"replay is idempotent (101 = already verified, same ref_id)", 200, `{"data":{"code":101,"message":"Verified","card_pan":"5022291234565995","ref_id":201},"errors":[]}`,
			payments.Result{Paid: true, RefID: "201", CardPAN: "502229******5995", AmountRials: 2607000}, ""},
		{"not paid / cancelled", 200, `{"data":[],"errors":{"code":-51,"message":"Session is not valid, session is not active paid try.","validations":[]}}`, payments.Result{}, ""},
		{"amount tampered (session amount differs)", 200, `{"data":[],"errors":{"code":-50,"message":"Session is not valid, amounts values is not the same.","validations":[]}}`, payments.Result{}, ""},
		{"invalid authority", 400, `{"data":[],"errors":{"code":-54,"message":"Invalid authority.","validations":[]}}`, payments.Result{}, ""},
		{"merchant problem → error, stays pending", 200, `{"data":[],"errors":{"code":-11,"message":"Terminal is not active.","validations":[]}}`, payments.Result{}, "code -11"},
		{"gateway down", 502, `<html>bad gateway</html>`, payments.Result{}, "not JSON"},
		{"verified without ref", 200, `{"data":{"code":100,"ref_id":0},"errors":[]}`, payments.Result{}, "ref"},
		{"unexpected code", 200, `{"data":{"code":-1},"errors":[]}`, payments.Result{}, "code -1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gw, s := newZarinpal(t, func(string, map[string]any) (int, string) { return c.status, c.body })
			res, err := gw.Verify(context.Background(), payments.VerifyRequest{
				Authority: zpAuthority, AmountRials: 2607000, Callback: map[string]string{"status": "OK", "amount": "1"},
			})
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, c.want, res)
			}
			require.Len(t, s.requests, 1)
			assert.Equal(t, "POST /pg/v4/payment/verify.json application/json", s.paths[0])
			assert.EqualValues(t, 2607000, s.requests[0]["amount"], "the stored amount, never a callback parameter")
			assert.Equal(t, zpAuthority, s.requests[0]["authority"])
		})
	}
}

func TestZarinpal_VerifyAlwaysAsksEvenOnNOK(t *testing.T) {
	gw, s := newZarinpal(t, func(string, map[string]any) (int, string) {
		return 200, `{"data":{"code":100,"ref_id":77},"errors":[]}`
	})
	res, err := gw.Verify(context.Background(), payments.VerifyRequest{Authority: zpAuthority, AmountRials: 5000, Callback: map[string]string{"status": "NOK"}})
	require.NoError(t, err)
	assert.True(t, res.Paid, "the server-to-server verdict wins over the browser's status")
	assert.Len(t, s.requests, 1)
}

func TestZarinpal_RefundNotSupported(t *testing.T) {
	gw, s := newZarinpal(t, func(string, map[string]any) (int, string) { return 500, "" })
	_, err := gw.Refund(context.Background(), payments.RefundRequest{Authority: zpAuthority, RefID: "201", AmountRials: 1000})
	require.ErrorIs(t, err, payments.ErrRefundNotSupported)
	assert.Empty(t, s.requests)
}

func TestZarinpal_SandboxAndDefaults(t *testing.T) {
	assert.Equal(t, payments.ZarinpalSandboxBaseURL, payments.NewZarinpal("m", "", true, http.DefaultClient).BaseURL)
	assert.Equal(t, payments.ZarinpalBaseURL, payments.NewZarinpal("m", "", false, http.DefaultClient).BaseURL)
}
