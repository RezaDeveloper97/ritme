// Package sms delivers OTP codes (App\Services\Sms): a primary provider (SMS_PROVIDER,
// default kavenegar) and the fallback list ["smsir"] minus the primary, tried in order until
// one reports success.
//
// Providers never log the code, the API key or a full phone number.
package sms

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ritme/backend-go/internal/platform/config"
)

// Provider names (config/sms.php).
const (
	Kavenegar = "kavenegar"
	SMSIR     = "smsir"
	Log       = "log"
)

// Fallbacks is config('sms.fallback').
var Fallbacks = []string{SMSIR}

// TemplateLoginOTP is the template key of the login OTP.
const TemplateLoginOTP = "login_otp"

// Provider sends OTP codes through one gateway. A nil error means the gateway accepted it.
type Provider interface {
	Name() string
	SendOTP(ctx context.Context, mobile, code, template string) error
}

// ErrAllFailed is returned when every provider failed.
var ErrAllFailed = errors.New("sms: all OTP providers failed")

// Service is SmsService::sendOtp with its fallback chain.
type Service struct {
	providers []Provider // primary first
	logger    *slog.Logger
}

// NewServiceWith builds a service over explicit providers (primary first).
func NewServiceWith(logger *slog.Logger, providers ...Provider) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{providers: providers, logger: logger}
}

// NewService resolves the configured primary provider and the fallbacks. It fails for an
// unknown provider name and for the log provider in production.
func NewService(cfg config.SMS, appEnv string, client *http.Client, logger *slog.Logger) (*Service, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second} // Laravel Http default timeout
	}
	primary, err := resolve(cfg.Provider, cfg, appEnv, client, logger)
	if err != nil {
		return nil, err
	}
	providers := []Provider{primary}
	for _, name := range Fallbacks {
		if name == cfg.Provider {
			continue
		}
		p, err := resolve(name, cfg, appEnv, client, logger)
		if err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return NewServiceWith(logger, providers...), nil
}

func resolve(name string, cfg config.SMS, appEnv string, client *http.Client, logger *slog.Logger) (Provider, error) {
	switch name {
	case Kavenegar:
		return NewKavenegar(cfg.Kavenegar, client, logger), nil
	case SMSIR:
		return NewSMSIR(cfg.SMSIR, client, logger), nil
	case Log:
		return NewLogProvider(appEnv, logger)
	default:
		return nil, fmt.Errorf("sms: unknown SMS provider: %s", name)
	}
}

// Providers returns the provider names in the order they are tried.
func (s *Service) Providers() []string {
	out := make([]string, len(s.providers))
	for i, p := range s.providers {
		out[i] = p.Name()
	}
	return out
}

// SendOTP tries each provider in order and stops at the first success.
func (s *Service) SendOTP(ctx context.Context, mobile, code, template string) error {
	for i, p := range s.providers {
		if i > 0 {
			s.logger.InfoContext(ctx, "Trying fallback OTP provider",
				slog.String("provider", p.Name()), slog.String("mobile", MaskMobile(mobile)))
		}
		err := p.SendOTP(ctx, mobile, code, template)
		if err == nil {
			s.logger.InfoContext(ctx, "OTP sent successfully",
				slog.String("provider", p.Name()), slog.String("mobile", MaskMobile(mobile)))
			return nil
		}
		s.logger.ErrorContext(ctx, "OTP provider failed",
			slog.String("provider", p.Name()), slog.String("mobile", MaskMobile(mobile)), slog.String("error", err.Error()))
	}
	s.logger.ErrorContext(ctx, "All OTP providers failed", slog.String("mobile", MaskMobile(mobile)))
	return ErrAllFailed
}

// MaskMobile keeps the operator prefix and the last 3 digits: 09123456789 → 0912****789.
func MaskMobile(m string) string {
	if len(m) < 8 {
		return "***"
	}
	return m[:4] + "****" + m[len(m)-3:]
}
