// Package projections builds what a family member may read of an owner's data (engine v6 M11, T-M11-06; SPEC §1.7;
// docs/engine-v6/family.md §4; DECISIONS.md F-01, F-02, F-03, F-21; D-194).
//
// Every domain has its own DTO, an explicit allow-list struct: only the fields declared on it can ever leave the
// server. Nothing is copied from the owner's own resources (no map is passed through, no owner JSON is reused), so a
// field added to an owner endpoint never reaches a member by accident. Never shared, in any domain: private notes,
// mood and EPDS, sex, lab results, chats, full cycle history, cycle numbers (day, length, lateness, confidence) and
// fertility data outside the partner-only TTC switch.
//
// Project re-decides (family.Decide, read) before it loads anything: a denied, paused (stale mode epoch), expired or
// revoked grant returns ErrDenied and no data. The legacy `symptoms` section has no domain any more (F-03).
package projections

import (
	"context"
	"database/sql"
	"errors"
	"time"

	carestore "github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/cycle/state"
	"github.com/ritme/backend-go/internal/family"
	healthlogstore "github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
	"github.com/ritme/backend-go/internal/vitals"
)

// ErrDenied: family.Decide refused the read. Callers answer with family.DenyError (no data).
var ErrDenied = errors.New("projections: read denied")

// ErrUnknownDomain: the domain has no member projection (404).
var ErrUnknownDomain = errors.New("projections: unknown domain")

// Domains are the member-readable domains, in display order.
var Domains = []family.Domain{family.DomainCyclePhase, family.DomainTTCFertility, family.DomainPregnancy,
	family.DomainFetalActivity, family.DomainAppointments, family.DomainMeds, family.DomainVitals}

// Readable reports whether d has a member projection.
func Readable(d family.Domain) bool {
	for _, x := range Domains {
		if x == d {
			return true
		}
	}
	return false
}

// Builder loads and projects the owner's data. Every query is scoped by the owner's id.
type Builder struct {
	db     *sql.DB
	cycle  CycleSource
	care   carestore.Querier
	preg   pregstore.Querier
	logs   healthlogstore.Querier
	vitals *vitals.Service
}

// New returns a Builder on db with the v1 cycle source (WithEngine switches to the v6 engine).
func New(db *sql.DB) *Builder {
	return &Builder{db: db, cycle: v1Cycle(db), care: carestore.New(db), preg: pregstore.New(db),
		logs: healthlogstore.New(db), vitals: vitals.NewService(db)}
}

// WithEngine reads the cycle phase from the v6 engine state of the owner (T-M9-36). nil keeps v1.
func (b *Builder) WithEngine(s state.UserStates) *Builder {
	if s != nil {
		b.cycle = v6Cycle{states: s}
	}
	return b
}

// WithCycleSource replaces the cycle source (tests).
func (b *Builder) WithCycleSource(s CycleSource) *Builder {
	b.cycle = s
	return b
}

// Project is the member projection of domain d of s.OwnerID's data at now, for s.ActorID. It decides first: anything
// but an Allow on read is ErrDenied, and nothing is loaded.
func (b *Builder) Project(ctx context.Context, s family.Snapshot, d family.Domain, locale string, now time.Time) (any, error) {
	if !Readable(d) {
		return nil, ErrUnknownDomain
	}
	if family.Decide(s, family.Subject{}, d, family.ActionRead, now).Outcome != family.Allow {
		return nil, ErrDenied
	}
	today := civildate.InTehran(now)
	switch d {
	case family.DomainCyclePhase:
		f, err := b.cycle.Facts(ctx, s.OwnerID, today)
		if err != nil {
			return nil, err
		}
		return CyclePhaseOf(f, s, now), nil
	case family.DomainTTCFertility:
		f, err := b.cycle.Facts(ctx, s.OwnerID, today)
		if err != nil {
			return nil, err
		}
		return TTCOf(f), nil
	case family.DomainPregnancy:
		return b.pregnancy(ctx, s.OwnerID, locale, today)
	case family.DomainFetalActivity:
		return b.fetal(ctx, s.OwnerID, now)
	case family.DomainAppointments:
		return b.appointments(ctx, s.OwnerID, now)
	case family.DomainMeds:
		return b.meds(ctx, s, locale, now)
	case family.DomainVitals:
		return b.vitalsOf(ctx, s.OwnerID, today)
	}
	return nil, ErrUnknownDomain
}

func dateOrNil(d civildate.Date) *string {
	if d.IsZero() {
		return nil
	}
	s := d.String()
	return &s
}
