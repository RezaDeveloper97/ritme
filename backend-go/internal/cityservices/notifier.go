package cityservices

import (
	"context"
	"log/slog"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// BookingNotice is everything a venue (or the ops channel) may ever be told about a request. It is deliberately typed
// with allow-listed fields only: no account id, no health, cycle, pregnancy or child medical data (extras.md §4).
type BookingNotice struct {
	BookingID     uint64
	ListingID     uint64
	ContactName   string
	ContactPhone  string
	PreferredDate civildate.Date
	ChildAgeRange string
	Note          string
}

// BookingNotifier is the only door from a request toward a venue. A failure never fails the request: the row is
// already in the admin queue, which is the source of truth.
type BookingNotifier interface {
	BookingRequested(ctx context.Context, n BookingNotice) error
}

// FakeNotifier is the default provider: it logs ids only (never a name, phone or note).
type FakeNotifier struct{ Logger *slog.Logger }

// BookingRequested logs the request ids.
func (f FakeNotifier) BookingRequested(ctx context.Context, n BookingNotice) error {
	l := f.Logger
	if l == nil {
		l = slog.Default()
	}
	l.InfoContext(ctx, "cityservices: booking requested (fake notifier)",
		slog.Uint64("booking_id", n.BookingID), slog.Uint64("listing_id", n.ListingID))
	return nil
}

// NewNotifier picks the provider by name; only `fake` exists today, anything else falls back to it.
func NewNotifier(provider string, logger *slog.Logger) BookingNotifier {
	if provider != "" && provider != "fake" && logger != nil {
		logger.Warn("cityservices: unknown notifier provider, using fake", slog.String("provider", provider))
	}
	return FakeNotifier{Logger: logger}
}
