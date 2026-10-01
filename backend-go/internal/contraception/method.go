package contraception

import (
	"database/sql"

	"github.com/ritme/backend-go/internal/contraception/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Method is the user's saved method; fields that do not belong to the method are zero.
type Method struct {
	Method           string
	PackType         string         // combined pill only
	PackStartedOn    civildate.Date // pill methods
	PacksLeft        *int           // pill methods: unopened packs when counted (nil = never counted)
	PacksCountedOn   civildate.Date // the start of the pack in use when PacksLeft was counted
	InsertedOn       civildate.Date // IUD (required), implant (optional)
	IUDLifetimeYears int            // IUD
	FollowupDone     bool           // IUD: the 6-week check-up visit
	InjectedOn       civildate.Date // injection: the last one
	ReplaceOn        civildate.Date // implant: removal / replacement date set by the user
}

func dateOf(n civildate.NullDate) civildate.Date {
	if !n.Valid {
		return civildate.Date{}
	}
	return n.Date
}

func nullDate(d civildate.Date) civildate.NullDate {
	return civildate.NullDate{Date: d, Valid: !d.IsZero()}
}

// methodFromRow reads a stored row.
func methodFromRow(r store.ContraceptionMethod) Method {
	m := Method{
		Method: r.Method, PackStartedOn: dateOf(r.PackStartedOn), PacksCountedOn: dateOf(r.PacksCountedOn),
		InsertedOn: dateOf(r.InsertedOn), FollowupDone: r.FollowupDone, InjectedOn: dateOf(r.InjectedOn),
		ReplaceOn: dateOf(r.ReplaceOn),
	}
	if r.PackType.Valid {
		m.PackType = r.PackType.String
	}
	if r.PacksLeft.Valid {
		n := int(r.PacksLeft.Int16)
		m.PacksLeft = &n
	}
	if r.IudLifetimeYears.Valid {
		m.IUDLifetimeYears = int(r.IudLifetimeYears.Int16)
	}
	return m
}

// params is the upsert of m for userID.
func (m Method) params(userID uint64, now sql.NullTime) store.UpsertMethodParams {
	p := store.UpsertMethodParams{
		UserID: userID, Method: m.Method, PackStartedOn: nullDate(m.PackStartedOn),
		PacksCountedOn: nullDate(m.PacksCountedOn), InsertedOn: nullDate(m.InsertedOn), FollowupDone: m.FollowupDone,
		InjectedOn: nullDate(m.InjectedOn), ReplaceOn: nullDate(m.ReplaceOn), Now: now,
	}
	if m.PackType != "" {
		p.PackType = sql.NullString{String: m.PackType, Valid: true}
	}
	if m.PacksLeft != nil {
		p.PacksLeft = sql.NullInt16{Int16: int16(*m.PacksLeft), Valid: true} //nolint:gosec // G115: validated 0…MaxPacksLeft
	}
	if m.IUDLifetimeYears > 0 {
		p.IudLifetimeYears = sql.NullInt16{Int16: int16(m.IUDLifetimeYears), Valid: true} //nolint:gosec // G115: validated 1…MaxIUDLifetime
	}
	return p
}

// Pack is the pill pack of the method; ok=false for non-pill methods or without a pack start.
func (m Method) Pack() (Pack, bool) {
	if m.PackStartedOn.IsZero() {
		return Pack{}, false
	}
	return PackFor(m.Method, m.PackType)
}

// Refill is the pack stock on today (Known=false without a count or a pack).
func (m Method) Refill(today civildate.Date) Refill {
	p, ok := m.Pack()
	if !ok || m.PacksLeft == nil || m.PacksCountedOn.IsZero() {
		return Refill{}
	}
	return p.RefillAt(m.PackStartedOn, today, *m.PacksLeft, m.PacksCountedOn)
}

// FollowupOn is the IUD's 6-week check-up date (zero without an insertion date).
func (m Method) FollowupOn() civildate.Date {
	if !IsIUD(m.Method) || m.InsertedOn.IsZero() {
		return civildate.Date{}
	}
	return m.InsertedOn.AddDays(FollowupDays)
}

// IUDReplaceOn is insertion + lifetime (zero when either is unknown).
func (m Method) IUDReplaceOn() civildate.Date {
	if !IsIUD(m.Method) || m.InsertedOn.IsZero() || m.IUDLifetimeYears <= 0 {
		return civildate.Date{}
	}
	return addMonths(m.InsertedOn, 12*m.IUDLifetimeYears)
}

// NextInjectionOn is the last injection + 12 weeks (zero without one).
func (m Method) NextInjectionOn() civildate.Date {
	if m.Method != MethodInjection || m.InjectedOn.IsZero() {
		return civildate.Date{}
	}
	return m.InjectedOn.AddDays(InjectionDays)
}
