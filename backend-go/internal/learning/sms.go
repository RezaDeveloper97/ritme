package learning

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	authsms "github.com/ritme/backend-go/internal/auth/sms"
	"github.com/ritme/backend-go/internal/platform/config"
)

// Sender delivers the «دوره برایت باز شد» SMS (LEARNING_SMS_PROVIDER, config.Learning):
//   - none: nothing is sent (Delivers false; outbox rows end as skipped/sms_disabled);
//   - fake: logs a masked number and sends nothing (default outside production, refused in production);
//   - gateway: Kavenegar verify/lookup with KAVENEGAR_TEMPLATE_COURSE_UNLOCKED, the course count as its token (the
//     login OTP's credentials; the text itself lives in the template, so it never names a course).
type Sender interface {
	Name() string
	Delivers() bool
	SendUnlocked(ctx context.Context, mobile string, courses int) error
}

// ErrSMSDisabled is returned by the none provider.
var ErrSMSDisabled = errors.New("learning: SMS disabled")

// NewSender resolves the configured provider.
func NewSender(cfg *config.Config, client *http.Client, logger *slog.Logger) (Sender, error) {
	if logger == nil {
		logger = slog.Default()
	}
	switch p := config.DefaultCompanionSMSProvider(cfg.Learning.SMSProvider, cfg.App); p {
	case config.CompanionSMSNone:
		return NoSMS{}, nil
	case config.CompanionSMSFake:
		if cfg.App.IsProduction() {
			return nil, errors.New("learning: the fake SMS provider cannot be used in production")
		}
		return NewFakeSMS(logger), nil
	case config.CompanionSMSGateway:
		if client == nil {
			client = &http.Client{Timeout: 30 * time.Second}
		}
		return &GatewaySMS{provider: authsms.NewKavenegar(cfg.SMS.Kavenegar, client, logger), template: cfg.Learning.UnlockTemplate}, nil
	default:
		return nil, fmt.Errorf("learning: unknown SMS provider %q", p)
	}
}

// NoSMS sends nothing.
type NoSMS struct{}

// Name implements Sender.
func (NoSMS) Name() string { return config.CompanionSMSNone }

// Delivers implements Sender.
func (NoSMS) Delivers() bool { return false }

// SendUnlocked implements Sender.
func (NoSMS) SendUnlocked(context.Context, string, int) error { return ErrSMSDisabled }

// FakeSMS logs that an SMS would have gone out (masked number) and remembers the numbers (tests).
type FakeSMS struct {
	logger *slog.Logger
	mu     sync.Mutex
	sent   []string
	// Fail makes the next sends fail (tests).
	Fail bool
}

// NewFakeSMS returns the fake provider.
func NewFakeSMS(logger *slog.Logger) *FakeSMS {
	if logger == nil {
		logger = slog.Default()
	}
	return &FakeSMS{logger: logger}
}

// Name implements Sender.
func (*FakeSMS) Name() string { return config.CompanionSMSFake }

// Delivers implements Sender.
func (*FakeSMS) Delivers() bool { return true }

// SendUnlocked implements Sender.
func (f *FakeSMS) SendUnlocked(ctx context.Context, mobile string, courses int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Fail {
		return errors.New("learning: fake SMS failure")
	}
	f.logger.InfoContext(ctx, "Course unlocked SMS (fake provider, not sent)",
		slog.String("mobile", authsms.MaskMobile(mobile)), slog.Int("courses", courses))
	f.sent = append(f.sent, mobile)
	return nil
}

// Sent returns and clears the numbers the fake was asked to reach.
func (f *FakeSMS) Sent() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.sent
	f.sent = nil
	return out
}

// GatewaySMS sends through Kavenegar with the course-unlocked template.
type GatewaySMS struct {
	provider authsms.Provider
	template string
}

// Name implements Sender.
func (*GatewaySMS) Name() string { return config.CompanionSMSGateway }

// Delivers implements Sender.
func (*GatewaySMS) Delivers() bool { return true }

// SendUnlocked implements Sender.
func (g *GatewaySMS) SendUnlocked(ctx context.Context, mobile string, courses int) error {
	return g.provider.SendOTP(ctx, mobile, strconv.Itoa(max(courses, 1)), g.template)
}
