// Package sms is the adapter for transactional SMS other than the login OTP (bloom B-N4-02: the «همدم» invite).
// The login OTP keeps its own chain in internal/auth/sms; the gateway provider here reuses its Kavenegar client
// (verify/lookup with another template), so there is one set of gateway credentials.
//
// Providers (COMPANION_SMS_PROVIDER, config.Companion):
//   - none: nothing is sent; Delivers() is false and the owner shares the code herself.
//   - fake: logs a masked number and sends nothing (default outside production; refused in production).
//   - gateway: Kavenegar verify/lookup with KAVENEGAR_TEMPLATE_COMPANION_INVITE, the code as its token. No SMS.ir
//     fallback: SMS.ir has a single (login) template id, so it would deliver the wrong text.
//
// Nothing here ever logs the code or a full phone number.
package sms

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	authsms "github.com/ritme/backend-go/internal/auth/sms"
	"github.com/ritme/backend-go/internal/platform/config"
)

// InviteSender delivers a companion invite code to a mobile number.
type InviteSender interface {
	// Name is the provider id (none, fake, gateway).
	Name() string
	// Delivers reports whether SendInvite actually sends something (false for none).
	Delivers() bool
	// SendInvite sends the code; a nil error means the provider accepted it.
	SendInvite(ctx context.Context, mobile, code string) error
}

// ErrNotSent is returned by the none provider.
var ErrNotSent = errors.New("sms: invite SMS disabled")

// New resolves the configured invite provider.
func New(cfg *config.Config, client *http.Client, logger *slog.Logger) (InviteSender, error) {
	if logger == nil {
		logger = slog.Default()
	}
	provider := config.DefaultCompanionSMSProvider(cfg.Companion.SMSProvider, cfg.App)
	switch provider {
	case config.CompanionSMSNone:
		return None{}, nil
	case config.CompanionSMSFake:
		if cfg.App.IsProduction() {
			return nil, errors.New("sms: the fake invite provider cannot be used in production")
		}
		return NewFake(logger), nil
	case config.CompanionSMSGateway:
		if client == nil {
			client = &http.Client{Timeout: 30 * time.Second}
		}
		return &Gateway{provider: authsms.NewKavenegar(cfg.SMS.Kavenegar, client, logger), template: cfg.Companion.InviteTemplate}, nil
	default:
		return nil, fmt.Errorf("sms: unknown invite provider %q", provider)
	}
}

// None sends nothing.
type None struct{}

// Name implements InviteSender.
func (None) Name() string { return config.CompanionSMSNone }

// Delivers implements InviteSender.
func (None) Delivers() bool { return false }

// SendInvite implements InviteSender.
func (None) SendInvite(context.Context, string, string) error { return ErrNotSent }

// Fake logs that an invite would have gone out (masked number, no code) and remembers the last numbers it was
// asked to reach, for tests.
type Fake struct {
	logger *slog.Logger
	sent   chan string
}

// NewFake returns the fake provider.
func NewFake(logger *slog.Logger) *Fake {
	if logger == nil {
		logger = slog.Default()
	}
	return &Fake{logger: logger, sent: make(chan string, 64)}
}

// Name implements InviteSender.
func (*Fake) Name() string { return config.CompanionSMSFake }

// Delivers implements InviteSender.
func (*Fake) Delivers() bool { return true }

// SendInvite implements InviteSender.
func (f *Fake) SendInvite(ctx context.Context, mobile, _ string) error {
	f.logger.InfoContext(ctx, "Companion invite SMS (fake provider, not sent)", slog.String("mobile", authsms.MaskMobile(mobile)))
	select {
	case f.sent <- mobile:
	default:
	}
	return nil
}

// Sent drains the numbers the fake was asked to reach (tests).
func (f *Fake) Sent() []string {
	var out []string
	for {
		select {
		case m := <-f.sent:
			out = append(out, m)
		default:
			return out
		}
	}
}

// Gateway sends through Kavenegar with the invite template.
type Gateway struct {
	provider authsms.Provider
	template string
}

// Name implements InviteSender.
func (*Gateway) Name() string { return config.CompanionSMSGateway }

// Delivers implements InviteSender.
func (*Gateway) Delivers() bool { return true }

// SendInvite implements InviteSender.
func (g *Gateway) SendInvite(ctx context.Context, mobile, code string) error {
	return g.provider.SendOTP(ctx, mobile, code, g.template)
}
