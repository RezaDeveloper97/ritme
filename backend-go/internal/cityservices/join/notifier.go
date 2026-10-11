package join

import (
	"context"
	"log/slog"
)

// Notice is everything the submit confirmation SMS («درخواستت رسید … کد پیگیری») may carry: the account's verified
// login mobile (users.mobile, never the unverified step-1 contact_phone) and the tracking code.
// No business detail, no account id beyond the row id for the log.
type Notice struct {
	ApplicationID uint64
	Mobile        string
	TrackingCode  string
}

// Notifier sends the submit confirmation. A failure never fails the submit: the row is already in the review queue.
type Notifier interface {
	Submitted(ctx context.Context, n Notice) error
}

// FakeNotifier is the default provider: it logs the application id only (never the mobile or the tracking code).
type FakeNotifier struct{ Logger *slog.Logger }

// Submitted logs the application id.
func (f FakeNotifier) Submitted(ctx context.Context, n Notice) error {
	l := f.Logger
	if l == nil {
		l = slog.Default()
	}
	l.InfoContext(ctx, "join: application submitted (fake sms)", slog.Uint64("application_id", n.ApplicationID))
	return nil
}

// NewNotifier picks the provider by name. Only `fake` exists until a business-SMS template is configured (an SMS
// gateway template is an open question of CB-DIR-04); anything else falls back to it with a warning.
func NewNotifier(provider string, logger *slog.Logger) Notifier {
	if provider != "" && provider != "fake" && logger != nil {
		logger.Warn("join: unknown notifier provider, using fake", slog.String("provider", provider))
	}
	return FakeNotifier{Logger: logger}
}
