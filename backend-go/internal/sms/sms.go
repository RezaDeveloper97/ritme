// Package sms is the adapter for transactional SMS other than the login OTP (bloom B-N4-02: the «همدم» invite).
// The login OTP keeps its own chain in internal/auth/sms; the gateway provider here reuses its Kavenegar client
// (verify/lookup with another template), so there is one set of gateway credentials.
//
// Providers (COMPANION_SMS_PROVIDER, config.Companion):
//   - none: nothing is sent; Delivers() is false and the owner shares the code herself.
//   - fake: logs a masked number and sends nothing (default outside production; refused in production).
//   - gateway: Kavenegar verify/lookup with KAVENEGAR_TEMPLATE_COMPANION_INVITE, the code as its token. No SMS.ir
//     fallback: SMS.ir has a single (login) template id, so it would deliver the wrong text. With the owner's
//     discreet flag on, KAVENEGAR_TEMPLATE_COMPANION_INVITE_NEUTRAL; not sent at all when that is unset (CB-PRIV-01).
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
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/config"
)

// InviteSender delivers a companion invite code to a mobile number.
type InviteSender interface {
	// Name is the provider id (none, fake, gateway).
	Name() string
	// Delivers reports whether SendInvite actually sends something (false for none).
	Delivers() bool
	// SendInvite sends the code; a nil error means the provider accepted it. discreet is the inviting owner's
	// «اعلان‌های محرمانه» flag (notifications.Discreet, CB-PRIV-01): the neutral template is used when it is on.
	SendInvite(ctx context.Context, mobile, code string, discreet bool) error
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
		return &Gateway{
			provider: authsms.NewKavenegar(cfg.SMS.Kavenegar, client, logger),
			template: cfg.Companion.InviteTemplate, neutral: cfg.Companion.InviteTemplateNeutral,
		}, nil
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
func (None) SendInvite(context.Context, string, string, bool) error { return ErrNotSent }

// Fake logs that an invite would have gone out (masked number, no code) and remembers the last numbers it was
// asked to reach, for tests.
type Fake struct {
	logger *slog.Logger
	sent   chan Invite
}

// Invite is one SendInvite call the fake recorded (never the code).
type Invite struct {
	Mobile   string
	Discreet bool
}

// NewFake returns the fake provider.
func NewFake(logger *slog.Logger) *Fake {
	if logger == nil {
		logger = slog.Default()
	}
	return &Fake{logger: logger, sent: make(chan Invite, 64)}
}

// Name implements InviteSender.
func (*Fake) Name() string { return config.CompanionSMSFake }

// Delivers implements InviteSender.
func (*Fake) Delivers() bool { return true }

// SendInvite implements InviteSender.
func (f *Fake) SendInvite(ctx context.Context, mobile, _ string, discreet bool) error {
	f.logger.InfoContext(ctx, "Companion invite SMS (fake provider, not sent)",
		slog.String("mobile", authsms.MaskMobile(mobile)), slog.Bool("discreet", discreet))
	select {
	case f.sent <- Invite{Mobile: mobile, Discreet: discreet}:
	default:
	}
	return nil
}

// Invites drains the invites the fake was asked to send (tests).
func (f *Fake) Invites() []Invite {
	var out []Invite
	for {
		select {
		case m := <-f.sent:
			out = append(out, m)
		default:
			return out
		}
	}
}

// Sent drains the numbers the fake was asked to reach (tests).
func (f *Fake) Sent() []string {
	var out []string
	for _, inv := range f.Invites() {
		out = append(out, inv.Mobile)
	}
	return out
}

// Gateway sends through Kavenegar with the invite template, or its neutral variant for a discreet owner.
type Gateway struct {
	provider authsms.Provider
	template string
	neutral  string // "" = no neutral variant registered at the gateway
}

// Name implements InviteSender.
func (*Gateway) Name() string { return config.CompanionSMSGateway }

// Delivers implements InviteSender.
func (*Gateway) Delivers() bool { return true }

// SendInvite implements InviteSender.
// A discreet owner's invite needs the neutral template: without one nothing is sent (ErrNotSent — no silent fallback
// to the regular wording), so the owner shares the code herself.
func (g *Gateway) SendInvite(ctx context.Context, mobile, code string, discreet bool) error {
	template, ok := notifications.SMSTemplate(discreet, g.template, g.neutral)
	if !ok {
		return ErrNotSent
	}
	return g.provider.SendOTP(ctx, mobile, code, template)
}
