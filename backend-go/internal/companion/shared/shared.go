// Package shared is the /companions/* compatibility layer over the v6 member projections (engine v6 M11, T-M11-06;
// D-194, D-204): each legacy section name maps onto the allow-list DTO of its v6 domain (internal/family/projections)
// — nothing here builds a view of its own any more.
//
//	cycle        → cycle_phase  (main_phase; period_soon for the partner role; no dates or numbers, F-01, F-02)
//	pregnancy    → pregnancy    (current_week, trimester, edd)
//	meds         → meds         (title, form, dose, unit, times; today's dose status at nudge and above)
//	appointments → appointments (title, scheduled_at, location, with, status)
//	symptoms     → removed (F-03): ErrInvalidSection, the route answers 404
//
// ReadFor re-decides through family.Decide on the link's (owner, companion) snapshot, so a paused, expired or revoked
// grant returns companion.ErrProjectionDenied and no data even when the caller forgot to check.
package shared

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/cycle/state"
	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/family/projections"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Reader maps legacy sections onto the member projections.
type Reader struct {
	db *sql.DB
	b  *projections.Builder
}

// NewReader returns a Reader on db (v1 cycle source; WithEngine switches to v6).
func NewReader(db *sql.DB) *Reader { return &Reader{db: db, b: projections.New(db)} }

// WithEngine serves the cycle phase from the v6 engine state of the owner (T-M9-36, D-136). nil keeps v1.
func (r *Reader) WithEngine(s state.UserStates) *Reader {
	r.b.WithEngine(s)
	return r
}

// Domain is the v6 domain of a legacy section (false for symptoms, F-03, and the teen card, which has its own route).
func Domain(s companion.Section) (family.Domain, bool) {
	switch s {
	case companion.SectionCycle:
		return family.DomainCyclePhase, true
	case companion.SectionPregnancy:
		return family.DomainPregnancy, true
	case companion.SectionMeds:
		return family.DomainMeds, true
	case companion.SectionAppointments:
		return family.DomainAppointments, true
	}
	return family.DomainNone, false
}

// Read is the projection of one section of ownerID's data as the owner herself sees it through the member DTO (no
// link: internal callers and tests).
func (r *Reader) Read(ctx context.Context, ownerID uint64, section companion.Section, locale string, now time.Time) (any, error) {
	d, ok := Domain(section)
	if !ok {
		return nil, companion.ErrInvalidSection
	}
	return r.b.Project(ctx, family.Snapshot{OwnerID: ownerID, ActorID: ownerID}, d, locale, now)
}

// ReadFor is the projection of one section of link.OwnerID's data for the link's companion, decided on their pair
// snapshot (the request's memoised one when ctx is a fiber.Ctx).
func (r *Reader) ReadFor(ctx context.Context, link companion.Link, section companion.Section, locale string, now time.Time) (any, error) {
	d, ok := Domain(section)
	if !ok {
		return nil, companion.ErrInvalidSection
	}
	if link.CompanionUserID == 0 || link.CompanionUserID == link.OwnerID {
		return nil, companion.ErrProjectionDenied
	}
	snap, err := family.Request(ctx, r.db, func() time.Time { return now }).Pair(ctx, link.OwnerID, link.CompanionUserID)
	if err != nil {
		return nil, err
	}
	v, err := r.b.Project(ctx, snap, d, locale, now)
	if errors.Is(err, projections.ErrDenied) {
		return nil, companion.ErrProjectionDenied
	}
	return v, err
}

// PeriodForecastFor is the likely period days of link.OwnerID in [from, to] for the link's companion, or
// companion.ErrProjectionDenied (no data) when the owner has not consented on this link (preferences.period_days) or
// the cycle_phase read is not allowed for their pair. The caller audits the read.
func (r *Reader) PeriodForecastFor(ctx context.Context, link companion.Link, from, to civildate.Date, now time.Time) (projections.PeriodForecast, error) {
	none := projections.PeriodForecast{Days: []string{}}
	if !link.Preferences.PeriodDays || link.CompanionUserID == 0 || link.CompanionUserID == link.OwnerID {
		return none, companion.ErrProjectionDenied
	}
	snap, err := family.Request(ctx, r.db, func() time.Time { return now }).Pair(ctx, link.OwnerID, link.CompanionUserID)
	if err != nil {
		return none, err
	}
	v, err := r.b.PeriodForecast(ctx, snap, from, to, now)
	if errors.Is(err, projections.ErrDenied) {
		return none, companion.ErrProjectionDenied
	}
	return v, err
}
