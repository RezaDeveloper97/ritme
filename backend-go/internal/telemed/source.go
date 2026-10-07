package telemed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/services"
	"github.com/ritme/backend-go/internal/telemed/store"
)

// BookingHref is the in-app path of a booking's detail screen (nbl_v17_Booked, docs/night-bloom/routes.md).
func BookingHref(id uint64) string { return "/services/bookings/" + strconv.FormatUint(id, 10) }

// NextBooking implements services.BookingSource: the user's next confirmed visit that has not ended (the «خدمات»
// card). The doctor name is picked in the request's language when ctx is the request.
func (b *Bookings) NextBooking(ctx context.Context, userID uint64, now time.Time) (*services.Booking, error) {
	bk, err := b.q.NextUserBooking(ctx, store.NextUserBookingParams{UserID: userID, Now: now})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("telemed: next booking: %w", err)
	}
	doctor, err := b.q.GetDoctor(ctx, bk.DoctorID)
	if err != nil {
		return nil, fmt.Errorf("telemed: doctor: %w", err)
	}
	return &services.Booking{
		ID: bk.ID, Kind: bk.Mode, ProviderName: textOf(localizerOf(ctx), doctor.Name), StartsAt: wall(bk.StartsAt),
		DurationMinutes: int(bk.DurationMinutes), Href: BookingHref(bk.ID),
	}, nil
}

// localizerOf is the request's localizer when ctx is a Fiber request, else the default-language pick.
func localizerOf(ctx context.Context) catalog.Localizer {
	if c, ok := ctx.(fiber.Ctx); ok {
		langs := i18n.LanguagesOf(c)
		return catalog.Localizer{Locale: i18n.Locale(c), Default: langs.DefaultCode(), Langs: langs}
	}
	def := i18n.Bootstrap.DefaultCode()
	return catalog.Localizer{Locale: def, Default: def, Langs: i18n.Bootstrap}
}

var _ services.BookingSource = (*Bookings)(nil)
