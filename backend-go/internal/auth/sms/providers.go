package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/ritme/backend-go/internal/platform/config"
)

// Default gateway base URLs (overridable for tests).
const (
	KavenegarBaseURL = "https://api.kavenegar.com/v1"
	SMSIRBaseURL     = "https://api.sms.ir/v1"
)

// maxBody caps how much of a gateway response is read.
const maxBody = 64 << 10

// KavenegarProvider is Providers\KavenegarProvider::sendOtp:
// GET {base}/{api_key}/verify/lookup.json?receptor&token&template, success when the JSON
// has return.status === 200.
type KavenegarProvider struct {
	BaseURL   string
	apiKey    string
	templates map[string]string
	client    *http.Client
	logger    *slog.Logger
}

// NewKavenegar builds the provider (templates: login_otp → KAVENEGAR_TEMPLATE_LOGIN_OTP).
func NewKavenegar(cfg config.Kavenegar, client *http.Client, logger *slog.Logger) *KavenegarProvider {
	return &KavenegarProvider{
		BaseURL:   KavenegarBaseURL,
		apiKey:    cfg.APIKey,
		templates: map[string]string{TemplateLoginOTP: cfg.TemplateLoginOTP},
		client:    client,
		logger:    logger,
	}
}

// Name implements Provider.
func (p *KavenegarProvider) Name() string { return Kavenegar }

// SendOTP implements Provider.
func (p *KavenegarProvider) SendOTP(ctx context.Context, mobile, code, template string) error {
	name, ok := p.templates[template]
	if !ok || name == "" {
		name = template
	}
	q := url.Values{}
	q.Set("receptor", mobile)
	q.Set("token", code)
	q.Set("template", name)
	// The API key is part of the path: never log this URL.
	u := p.BaseURL + "/" + url.PathEscape(p.apiKey) + "/verify/lookup.json?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return errors.New("kavenegar: build request failed")
	}
	status, body, err := do(p.client, req)
	if err != nil {
		return fmt.Errorf("kavenegar: %w", err)
	}
	if status < 200 || status > 299 {
		return fmt.Errorf("kavenegar: HTTP %d", status)
	}
	var data struct {
		Return struct {
			Status json.RawMessage `json:"status"`
		} `json:"return"`
	}
	if json.Unmarshal(body, &data) != nil || string(data.Return.Status) != "200" {
		return fmt.Errorf("kavenegar: rejected (return.status=%s)", clip(data.Return.Status))
	}
	return nil
}

// IRProvider is Providers\SmsIrProvider::sendOtp: POST {base}/send/verify with
// X-API-KEY and {mobile, templateId, parameters:[{name:"OTPCODE", value}]}, success when
// the JSON has status === 1.
type IRProvider struct {
	BaseURL    string
	apiKey     string
	templateID int
	client     *http.Client
	logger     *slog.Logger
}

// NewSMSIR builds the provider.
func NewSMSIR(cfg config.SMSIR, client *http.Client, logger *slog.Logger) *IRProvider {
	return &IRProvider{BaseURL: SMSIRBaseURL, apiKey: cfg.APIKey, templateID: cfg.TemplateID, client: client, logger: logger}
}

// Name implements Provider.
func (p *IRProvider) Name() string { return SMSIR }

// SendOTP implements Provider. The template argument is unused: SMS.ir has one template id.
func (p *IRProvider) SendOTP(ctx context.Context, mobile, code, _ string) error {
	type param struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	payload, err := json.Marshal(struct {
		Mobile     string  `json:"mobile"`
		TemplateID int     `json:"templateId"`
		Parameters []param `json:"parameters"`
	}{mobile, p.templateID, []param{{Name: "OTPCODE", Value: code}}})
	if err != nil {
		return fmt.Errorf("smsir: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/send/verify", bytes.NewReader(payload))
	if err != nil {
		return errors.New("smsir: build request failed")
	}
	req.Header.Set("X-API-KEY", p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	status, body, err := do(p.client, req)
	if err != nil {
		return fmt.Errorf("smsir: %w", err)
	}
	if status < 200 || status > 299 {
		return fmt.Errorf("smsir: HTTP %d", status)
	}
	var data struct {
		Status json.RawMessage `json:"status"`
	}
	if json.Unmarshal(body, &data) != nil || string(data.Status) != "1" {
		return fmt.Errorf("smsir: rejected (status=%s)", clip(data.Status))
	}
	return nil
}

// LogProvider is Providers\LogSmsProvider: logs instead of sending (contract stack, local
// dev). It refuses to exist in production. The code is not logged.
type LogProvider struct{ logger *slog.Logger }

// NewLogProvider fails when appEnv is production.
func NewLogProvider(appEnv string, logger *slog.Logger) (*LogProvider, error) {
	if appEnv == "production" {
		return nil, errors.New("sms: the log SMS provider cannot be used in production")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &LogProvider{logger: logger}, nil
}

// Name implements Provider.
func (p *LogProvider) Name() string { return Log }

// SendOTP implements Provider.
func (p *LogProvider) SendOTP(ctx context.Context, mobile, _ string, template string) error {
	p.logger.InfoContext(ctx, "OTP SMS (log provider, not sent)",
		slog.String("mobile", MaskMobile(mobile)), slog.String("template", template))
	return nil
}

// do sends req and reads a bounded body. Transport errors are reduced to their kind so a
// URL carrying the API key never ends up in an error message.
func do(client *http.Client, req *http.Request) (int, []byte, error) {
	resp, err := client.Do(req) //nolint:gosec // G704: fixed gateway URLs from config
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			return 0, nil, fmt.Errorf("request failed: %w", ue.Err)
		}
		return 0, nil, errors.New("request failed")
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read body: %w", err)
	}
	return resp.StatusCode, body, nil
}

// clip shortens a raw JSON value for an error message. Statuses are compared as raw
// literals because PHP checks `=== 200` / `=== 1` (an int: not "200", not 200.0).
func clip(raw json.RawMessage) string {
	if len(raw) > 32 {
		return string(raw[:32]) + "…"
	}
	return string(raw)
}
