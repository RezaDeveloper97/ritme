package sms_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth/sms"
	"github.com/ritme/backend-go/internal/platform/config"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type gateway struct {
	mu    sync.Mutex
	calls []string // "kavenegar" / "smsir" in call order
	reqs  []*http.Request
	body  []string
}

// fake serves both gateways; kavReply / irReply are the JSON bodies they answer with.
func fake(t *testing.T, g *gateway, kavStatus int, kavReply string, irStatus int, irReply string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		g.mu.Lock()
		g.reqs = append(g.reqs, r)
		g.body = append(g.body, string(b))
		g.mu.Unlock()
		if r.URL.Path == "/send/verify" {
			g.calls = append(g.calls, sms.SMSIR)
			w.WriteHeader(irStatus)
			_, _ = w.Write([]byte(irReply))
			return
		}
		g.calls = append(g.calls, sms.Kavenegar)
		w.WriteHeader(kavStatus)
		_, _ = w.Write([]byte(kavReply))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func service(t *testing.T, srv *httptest.Server, primary string) *sms.Service {
	t.Helper()
	cfg := config.SMS{
		Provider:  primary,
		Kavenegar: config.Kavenegar{APIKey: "KEY/1", TemplateLoginOTP: "1507703"},
		SMSIR:     config.SMSIR{APIKey: "IRKEY", TemplateID: 511293},
	}
	kav := sms.NewKavenegar(cfg.Kavenegar, srv.Client(), quiet)
	kav.BaseURL = srv.URL
	ir := sms.NewSMSIR(cfg.SMSIR, srv.Client(), quiet)
	ir.BaseURL = srv.URL
	if primary == sms.SMSIR {
		return sms.NewServiceWith(quiet, ir)
	}
	return sms.NewServiceWith(quiet, kav, ir)
}

func TestNewService_FallbackOrder(t *testing.T) {
	svc, err := sms.NewService(config.SMS{Provider: sms.Kavenegar}, "production", nil, quiet)
	require.NoError(t, err)
	assert.Equal(t, []string{"kavenegar", "smsir"}, svc.Providers())

	// The primary is removed from the fallback list.
	svc, err = sms.NewService(config.SMS{Provider: sms.SMSIR}, "production", nil, quiet)
	require.NoError(t, err)
	assert.Equal(t, []string{"smsir"}, svc.Providers())

	svc, err = sms.NewService(config.SMS{Provider: sms.Log}, "local", nil, quiet)
	require.NoError(t, err)
	assert.Equal(t, []string{"log", "smsir"}, svc.Providers())

	_, err = sms.NewService(config.SMS{Provider: sms.Log}, "production", nil, quiet)
	require.Error(t, err, "the log provider refuses production")
	_, err = sms.NewService(config.SMS{Provider: "twilio"}, "local", nil, quiet)
	require.Error(t, err)
}

func TestSendOTP_PrimarySucceeds(t *testing.T) {
	g := &gateway{}
	srv := fake(t, g, 200, `{"return":{"status":200,"message":"ok"}}`, 200, `{"status":1}`)
	require.NoError(t, service(t, srv, sms.Kavenegar).SendOTP(context.Background(), "09121234567", "4821", sms.TemplateLoginOTP))
	assert.Equal(t, []string{"kavenegar"}, g.calls)

	r := g.reqs[0]
	assert.Equal(t, http.MethodGet, r.Method)
	assert.Equal(t, "/KEY%2F1/verify/lookup.json", r.URL.EscapedPath())
	assert.Equal(t, "09121234567", r.URL.Query().Get("receptor"))
	assert.Equal(t, "4821", r.URL.Query().Get("token"))
	assert.Equal(t, "1507703", r.URL.Query().Get("template"))
}

func TestSendOTP_FallsBackToSMSIR(t *testing.T) {
	for name, kav := range map[string]struct {
		status int
		body   string
	}{
		"http error":      {500, `{}`},
		"status not 200":  {200, `{"return":{"status":418}}`},
		"status string":   {200, `{"return":{"status":"200"}}`},
		"status float":    {200, `{"return":{"status":200.0}}`},
		"not json":        {200, `<html>`},
		"missing return":  {200, `{}`},
		"no body at all":  {204, ``},
		"redirect status": {302, ``},
	} {
		t.Run(name, func(t *testing.T) {
			g := &gateway{}
			srv := fake(t, g, kav.status, kav.body, 200, `{"status":1,"message":"ok"}`)
			require.NoError(t, service(t, srv, sms.Kavenegar).SendOTP(context.Background(), "09121234567", "4821", sms.TemplateLoginOTP))
			assert.Equal(t, []string{"kavenegar", "smsir"}, g.calls)

			r := g.reqs[1]
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "IRKEY", r.Header.Get("X-API-KEY"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			var body map[string]any
			require.NoError(t, json.Unmarshal([]byte(g.body[1]), &body))
			assert.Equal(t, map[string]any{
				"mobile": "09121234567", "templateId": float64(511293),
				"parameters": []any{map[string]any{"name": "OTPCODE", "value": "4821"}},
			}, body)
		})
	}
}

func TestSendOTP_AllFail(t *testing.T) {
	g := &gateway{}
	srv := fake(t, g, 200, `{"return":{"status":411}}`, 200, `{"status":0}`)
	err := service(t, srv, sms.Kavenegar).SendOTP(context.Background(), "09121234567", "4821", sms.TemplateLoginOTP)
	require.ErrorIs(t, err, sms.ErrAllFailed)
	assert.Equal(t, []string{"kavenegar", "smsir"}, g.calls)
}

func TestSendOTP_ErrorsNeverContainTheKeyedURL(t *testing.T) {
	kav := sms.NewKavenegar(config.Kavenegar{APIKey: "SECRETKEY"}, &http.Client{}, quiet)
	kav.BaseURL = "http://127.0.0.1:1" // connection refused
	err := kav.SendOTP(context.Background(), "09121234567", "4821", sms.TemplateLoginOTP)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "SECRETKEY")
	assert.NotContains(t, err.Error(), "4821")
}

func TestMaskMobile(t *testing.T) {
	assert.Equal(t, "0912****567", sms.MaskMobile("09121234567"))
	assert.Equal(t, "***", sms.MaskMobile("0912"))
}
