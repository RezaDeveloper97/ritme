// Package remindersms is «یادآور پیامکی» (Ritme Plus): the medication doses and doctor-appointment reminders of
// Plus subscribers also go out by SMS, automatically. It sits outside internal/care on purpose — care is a
// no-paywall package (internal/http/paywall_guard_int_test.go) and never learns about Plus; the reminder itself
// (in-app, notification) stays free for everyone, only the SMS copy is the Plus extra.
//
// Every SMS passes the same gate as a push: the category switch («داروها» / «نوبت پزشک»), quiet hours, and
// «متن خنثی», which swaps the template for the neutral one (no neutral template configured = no SMS, never the
// regular wording). On top of that: the user's own «یادآور پیامکی» switch and a per-day cap.
//
// Providers (REMINDER_SMS_PROVIDER, config.ReminderSMS): none (default, the sender does not start) · fake (logs a
// masked number; refused in production) · gateway (Kavenegar verify/lookup with the login OTP's credentials). The
// wording lives in the gateway templates:
//
//	KAVENEGAR_TEMPLATE_MED_REMINDER          «ریتمی: وقت %token10 است (ساعت %token).»
//	KAVENEGAR_TEMPLATE_APPOINTMENT_REMINDER  «ریتمی: یادآور %token10، %token20.»
//	KAVENEGAR_TEMPLATE_REMINDER_NEUTRAL      «ریتمی: برای ساعت %token یک یادآور داری.»
//
// Nothing here logs a full phone number or a medication name.
package remindersms

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	authsms "github.com/ritme/backend-go/internal/auth/sms"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/config"
)

// Kinds of reminder SMS.
const (
	KindMedication  = "medication"
	KindAppointment = "appointment"
)

// SMS is one reminder to text.
type SMS struct {
	Kind     string
	Discreet bool   // the user's «متن خنثی»: the neutral template, nothing about her health
	Name     string // medication name / appointment title
	Clock    string // "HH:MM" of the dose or the visit
	When     string // appointments: «فردا ساعت 10:30»
}

// Sender delivers a reminder SMS.
type Sender interface {
	Name() string
	// Delivers reports whether Send sends something (false for none: the worker does not start).
	Delivers() bool
	Send(ctx context.Context, mobile string, sms SMS) error
}

// ErrNotSent: the provider is off, or the user is discreet and no neutral template is configured.
var ErrNotSent = errors.New("remindersms: not sent")

// NewSender resolves the configured provider.
func NewSender(cfg *config.Config, client *http.Client, logger *slog.Logger) (Sender, error) {
	if logger == nil {
		logger = slog.Default()
	}
	switch p := cfg.ReminderSMS.Provider; p {
	case "", config.CompanionSMSNone:
		return None{}, nil
	case config.CompanionSMSFake:
		if cfg.App.IsProduction() {
			return nil, errors.New("remindersms: the fake provider cannot be used in production")
		}
		return NewFake(logger), nil
	case config.CompanionSMSGateway:
		if client == nil {
			client = &http.Client{Timeout: 30 * time.Second}
		}
		return &Gateway{provider: authsms.NewKavenegar(cfg.SMS.Kavenegar, client, logger), cfg: cfg.ReminderSMS}, nil
	default:
		return nil, fmt.Errorf("remindersms: unknown provider %q", p)
	}
}

// None sends nothing.
type None struct{}

// Name implements Sender.
func (None) Name() string { return config.CompanionSMSNone }

// Delivers implements Sender.
func (None) Delivers() bool { return false }

// Send implements Sender.
func (None) Send(context.Context, string, SMS) error { return ErrNotSent }

// Fake logs that an SMS would have gone out (masked number, kind only) and remembers it for tests.
type Fake struct {
	logger *slog.Logger
	mu     sync.Mutex
	sent   []Sent
	// Fail makes the next sends fail (tests).
	Fail bool
}

// Sent is one Send call the fake recorded.
type Sent struct {
	Mobile string
	SMS    SMS
}

// NewFake returns the fake provider.
func NewFake(logger *slog.Logger) *Fake {
	if logger == nil {
		logger = slog.Default()
	}
	return &Fake{logger: logger}
}

// Name implements Sender.
func (*Fake) Name() string { return config.CompanionSMSFake }

// Delivers implements Sender.
func (*Fake) Delivers() bool { return true }

// Send implements Sender.
func (f *Fake) Send(ctx context.Context, mobile string, sms SMS) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Fail {
		return errors.New("remindersms: fake failure")
	}
	f.logger.InfoContext(ctx, "Reminder SMS (fake provider, not sent)", slog.String("mobile", authsms.MaskMobile(mobile)),
		slog.String("kind", sms.Kind), slog.Bool("discreet", sms.Discreet))
	f.sent = append(f.sent, Sent{Mobile: mobile, SMS: sms})
	return nil
}

// Drain returns and clears what the fake was asked to send.
func (f *Fake) Drain() []Sent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.sent
	f.sent = nil
	return out
}

type lookup interface {
	Lookup(ctx context.Context, mobile, template string, tokens map[string]string) error
}

// Gateway sends through Kavenegar verify/lookup.
type Gateway struct {
	provider lookup
	cfg      config.ReminderSMS
}

// Name implements Sender.
func (*Gateway) Name() string { return config.CompanionSMSGateway }

// Delivers implements Sender.
func (*Gateway) Delivers() bool { return true }

// Send implements Sender.
func (g *Gateway) Send(ctx context.Context, mobile string, sms SMS) error {
	template, tokens, ok := Template(g.cfg, sms)
	if !ok {
		return ErrNotSent
	}
	return g.provider.Lookup(ctx, mobile, template, tokens)
}

// Template picks the gateway template of sms and fills its tokens; ok is false when nothing may be sent (a
// discreet user without a neutral template, an unknown kind, an unset template).
func Template(cfg config.ReminderSMS, sms SMS) (template string, tokens map[string]string, ok bool) {
	regular := cfg.MedicationTemplate
	tokens = map[string]string{"token": sms.Clock, "token10": Token(sms.Name, 4)}
	switch sms.Kind {
	case KindMedication:
	case KindAppointment:
		regular = cfg.AppointmentTemplate
		tokens["token20"] = Token(sms.When, 8)
	default:
		return "", nil, false
	}
	template, ok = notifications.SMSTemplate(sms.Discreet, regular, cfg.NeutralTemplate)
	if !ok || template == "" {
		return "", nil, false
	}
	if sms.Discreet {
		tokens = map[string]string{"token": sms.Clock}
	}
	return template, tokens, true
}

// Token fits s into a lookup token that allows at most `spaces` spaces: whitespace is collapsed and the words
// past the limit are dropped; an empty result becomes "-" (the gateway refuses an empty token).
func Token(s string, spaces int) string {
	words := strings.Fields(s)
	if len(words) > spaces+1 {
		words = words[:spaces+1]
	}
	out := strings.Join(words, " ")
	if r := []rune(out); len(r) > 60 {
		out = string(r[:60])
	}
	if out == "" {
		return "-"
	}
	return out
}
