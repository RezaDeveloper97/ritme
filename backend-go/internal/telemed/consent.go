package telemed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/telemed/store"
)

// Scoped data-share consent (nbl_v17_ReviewBook «اشتراک داده‌های ریتمی با پزشک»): per booking and scope, for this
// booking's doctor only, within the visit window only, revocable at any time. Nothing is shared without an active
// consent row — AuthorizeShare is the one gate every doctor-side read (B-N7-04 / B-N7-08) must pass.

// ActiveScopes are the scopes the user currently shares on a booking (Scopes order).
func (b *Bookings) ActiveScopes(ctx context.Context, bookingID uint64) ([]string, error) {
	rows, err := b.q.ListBookingConsents(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("telemed: consents: %w", err)
	}
	active := map[string]bool{}
	for _, r := range rows {
		if !r.RevokedAt.Valid {
			active[r.Scope] = true
		}
	}
	out := []string{}
	for _, s := range Scopes {
		if active[s] {
			out = append(out, s)
		}
	}
	return out, nil
}

// SetConsent grants (true) or revokes (false) scopes on the user's booking. Granting needs an own-visit booking that
// is held or confirmed (ErrConsentNotAllowed / ErrBookingClosed); revoking always works.
func (b *Bookings) SetConsent(ctx context.Context, userID, id uint64, changes map[string]bool) (store.TelemedBooking, error) {
	now := b.now(ctx)
	bk, err := b.Booking(ctx, userID, id)
	if err != nil {
		return store.TelemedBooking{}, err
	}
	grants := false
	for _, on := range changes {
		grants = grants || on
	}
	if grants {
		switch {
		case bk.ForWhom != ForSelf:
			return store.TelemedBooking{}, ErrConsentNotAllowed
		case bk.Status != StatusHeld && bk.Status != StatusConfirmed:
			return store.TelemedBooking{}, ErrBookingClosed
		}
	}
	err = b.inTx(ctx, func(_ *sql.Tx, q *store.Queries) error {
		for _, scope := range Scopes {
			on, ok := changes[scope]
			switch {
			case !ok:
			case on:
				if err := q.UpsertConsent(ctx, store.UpsertConsentParams{BookingID: bk.ID, Scope: scope, GrantedAt: now, Stamp: nt(now)}); err != nil {
					return fmt.Errorf("telemed: grant consent: %w", err)
				}
			default:
				if err := q.RevokeConsent(ctx, store.RevokeConsentParams{Now: nt(now), BookingID: bk.ID, Scope: scope}); err != nil {
					return fmt.Errorf("telemed: revoke consent: %w", err)
				}
			}
		}
		return nil
	})
	return bk, err
}

// RevokeConsent revokes every scope of the user's booking.
func (b *Bookings) RevokeConsent(ctx context.Context, userID, id uint64) (store.TelemedBooking, error) {
	bk, err := b.Booking(ctx, userID, id)
	if err != nil {
		return store.TelemedBooking{}, err
	}
	if err := b.q.RevokeAllConsents(ctx, store.RevokeAllConsentsParams{Now: nt(b.now(ctx)), BookingID: bk.ID}); err != nil {
		return store.TelemedBooking{}, fmt.Errorf("telemed: revoke consents: %w", err)
	}
	return bk, nil
}

// ShareWindowEnd is the last moment a doctor may read what was shared on bk.
func ShareWindowEnd(bk store.TelemedBooking) time.Time { return wall(bk.EndsAt).Add(ConsentGrace) }

// AuthorizeShare is the doctor-side gate: doctorID may read scope of booking bookingID at now only when the booking
// is this doctor's, confirmed or completed, inside its visit window (until ShareWindowEnd) and the user's consent
// for scope is active. Anything else — unknown booking, another doctor, a hold, a cancelled visit, a revoked or
// never-given consent, a past window — is ErrConsentRequired (no detail leaks to the caller).
func (b *Bookings) AuthorizeShare(ctx context.Context, doctorID, bookingID uint64, scope string, now time.Time) (store.TelemedBooking, error) {
	if !slices.Contains(Scopes, scope) {
		return store.TelemedBooking{}, ErrConsentRequired
	}
	bk, err := b.q.GetBooking(ctx, bookingID)
	if errors.Is(err, sql.ErrNoRows) {
		return store.TelemedBooking{}, ErrConsentRequired
	}
	if err != nil {
		return store.TelemedBooking{}, fmt.Errorf("telemed: booking: %w", err)
	}
	switch {
	case bk.DoctorID != doctorID, bk.ForWhom != ForSelf:
		return store.TelemedBooking{}, ErrConsentRequired
	case bk.Status != StatusConfirmed && bk.Status != StatusCompleted:
		return store.TelemedBooking{}, ErrConsentRequired
	case now.After(ShareWindowEnd(bk)):
		return store.TelemedBooking{}, ErrConsentRequired
	}
	_, err = b.q.GetActiveConsent(ctx, store.GetActiveConsentParams{BookingID: bookingID, DoctorID: doctorID, Scope: scope})
	if errors.Is(err, sql.ErrNoRows) {
		return store.TelemedBooking{}, ErrConsentRequired
	}
	if err != nil {
		return store.TelemedBooking{}, fmt.Errorf("telemed: consent: %w", err)
	}
	return bk, nil
}
