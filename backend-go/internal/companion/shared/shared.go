// Package shared builds the access-filtered views of an owner's data that a companion may read (bloom B-N4-02), one
// per companion section, by reusing the owning domains (cycle engine, taxonomy-v2 day log, care reminders,
// pregnancy calculator). It never decides access: callers check companion.Service.Level first and audit the read.
//
// Each view is a deliberate subset of what the owner sees herself:
//   - cycle: today's cycle day, phase and the next-period prediction (no logs, no fertility-test values);
//   - symptoms: the last SymptomDays days of the day log, limited to SymptomCategories (pain, symptoms, sleep,
//     energy/appetite, skin & hair) — never mood or anything EPDS-derived (SPEC v6 §1.7 never-shared, T-M9-02), sex,
//     discharge, urogenital, measurements, bleeding detail or free-text notes;
//   - meds: the owner's active medications as the member projection (care.Medication.MemberJSON: no notes, no
//     free-text labels);
//   - appointments: the owner's upcoming appointments as the member projection (care.Appointment.MemberJSON: no
//     notes, topic, result note or prep) except private ones (loss follow-ups);
//   - pregnancy: the pregnancy status (week, due date, trimester) while the owner's effective mode is pregnancy (not
//     postpartum), else is_active=false.
//
// The views are mode-aware (B-N4-08b, CMP-H1 / CMP-M1 of docs/security/bloom-companion.md): ReadFor gets the link, so
// a pregnancy / postpartum owner whose link has no pregnancy grant (and every menopause owner) gets the neutral cycle
// view, the symptoms view keeps only what the cycle-mode log offers (mode.go), lateness is never shown (no days_late,
// no lateness-derived number: T-M9-02, FAM-M1), and the fertile phase is mapped away for owners whose mode never gets fertility copy (teen).
package shared

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/care"
	carestore "github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/resolver"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/state"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// SymptomDays is how many days (today included) the symptoms view covers.
const SymptomDays = 7

// SymptomCategories are the taxonomy-v2 categories the symptoms section shares («علائم و حال روزانه»).
// Mood is never shared (SPEC v6 §1.7; gap report 05 FAM-H1) — removing the whole section is decision F-03 (M11).
var SymptomCategories = []string{"pain", "symptoms", "sleep", "appetite_energy", "skin_hair"}

// DB is what the readers query through (a *sql.DB).
type DB interface {
	carestore.DBTX
}

// Reader builds the section views.
type Reader struct {
	cycle    *cycleservice.Service
	healthlg *healthlog.Service
	care     *carestore.Queries
	preg     *pregstore.Queries
	states   state.UserStates // nil = CYCLE_ENGINE=v1: the cycle view keeps the v1.1 resolver
}

// WithEngine serves the cycle view from the v6 engine state of the owner (T-M9-36, D-136). nil keeps v1.
func (r *Reader) WithEngine(s state.UserStates) *Reader {
	r.states = s
	return r
}

// NewReader returns a Reader on db.
func NewReader(db DB) *Reader {
	return &Reader{
		cycle:    cycleservice.New(db, nil),
		healthlg: healthlog.NewService(db),
		care:     carestore.New(db),
		preg:     pregstore.New(db),
	}
}

// Read is ReadFor on a link without grants (no pregnancy grant): the strictest mode-aware view of one section of
// ownerID's data at now in locale.
func (r *Reader) Read(ctx context.Context, ownerID uint64, section companion.Section, locale string, now time.Time) (any, error) {
	return r.ReadFor(ctx, companion.Link{OwnerID: ownerID}, section, locale, now)
}

// ReadFor is the view of one section of link.OwnerID's data at now in locale, as the link's companion may see it:
// the link's grants decide whether pregnancy-derived signals may show in the cycle and symptoms views.
func (r *Reader) ReadFor(ctx context.Context, link companion.Link, section companion.Section, locale string, now time.Time) (any, error) {
	ownerID := link.OwnerID
	today := civildate.LocalToday(ctx, clock.At(now))
	switch section {
	case companion.SectionCycle:
		scope, err := r.scope(ctx, link)
		if err != nil {
			return nil, err
		}
		return r.cycleView(ctx, ownerID, today, scope)
	case companion.SectionSymptoms:
		scope, err := r.scope(ctx, link)
		if err != nil {
			return nil, err
		}
		return r.symptomsView(ctx, ownerID, today, scope)
	case companion.SectionMeds:
		return r.medsView(ctx, ownerID, locale)
	case companion.SectionAppointments:
		return r.appointmentsView(ctx, ownerID, now)
	case companion.SectionPregnancy:
		scope, err := r.scope(ctx, link)
		if err != nil {
			return nil, err
		}
		return r.pregnancyView(ctx, ownerID, locale, today, scope)
	default:
		return nil, companion.ErrInvalidSection
	}
}

func (r *Reader) cycleView(ctx context.Context, ownerID uint64, today civildate.Date, scope viewScope) (any, error) {
	if scope.neutralCycle() {
		return neutralCycle(today), nil
	}
	if r.states != nil {
		return r.cycleViewV6(ctx, ownerID, today)
	}
	sn, err := r.cycle.Load(ctx, ownerID, today, today, today)
	if err != nil {
		return nil, err
	}
	profile := sn.EngineProfile()
	m := metrics.Calculate(sn.Histories, profile)
	st := resolver.Resolve(sn.Histories, profile, today, today, m)
	var cycleLength, nextStart, cycleDay, daysToPeriod any
	if st.EffectiveCycleLength > 0 {
		cycleLength = st.EffectiveCycleLength
	}
	if !st.PredictedNextPeriodStart.IsZero() {
		nextStart = st.PredictedNextPeriodStart
	}
	cycleDay, daysToPeriod = st.CycleDay, st.DaysToPeriod
	if late(st.DaysLate) { // lateness is a pregnancy signal, never a member's business (T-M9-02, FAM-M1)
		cycleDay, daysToPeriod, nextStart = nil, nil, nil
	}
	return jsonx.Obj(
		"date", today,
		"has_data", len(sn.Histories) > 0 || sn.Profile != nil,
		"cycle_day", cycleDay,
		"cycle_length", cycleLength,
		"main_phase", sharedPhase(st.MainPhase, profile != nil && profile.NoFertilityCopy),
		"days_to_period", daysToPeriod,
		"predicted_next_period_start", nextStart,
		"confidence", st.Confidence,
	), nil
}

// cycleViewV6 is the same subset from the v6 engine (T-M9-36, D-136; the keys and their meaning are unchanged, nothing
// is added). The lateness rule of T-M9-02 holds: a late owner (days_late ≥ 1) gets no cycle day, days_to_period or
// prediction. Fertility suppression (limits, E-24: hormonal contraception, pregnancy, postpartum) maps the fertile
// phase away exactly like a mode without fertility copy, and the flag itself is never passed on.
func (r *Reader) cycleViewV6(ctx context.Context, ownerID uint64, today civildate.Date) (any, error) {
	es, err := r.states.ForUser(ctx, ownerID, today)
	if err != nil {
		return nil, err
	}
	d := engine.DayOf(today)
	st := es.Resolve(d, d)
	decl := es.Declared()
	hasData := len(es.Periods()) > 0 || decl.LastPeriodStart.OK() || decl.CycleLength.OK() || decl.PeriodLength.OK()

	var cycleLength, nextStart, cycleDay, daysToPeriod any
	if st.Metrics.EffectiveCycleLength > 0 {
		cycleLength = st.Metrics.EffectiveCycleLength
	}
	if st.Anchors.PredictedNextPeriodStart.IsSet() {
		nextStart = st.Anchors.PredictedNextPeriodStart.Civil()
	}
	if v, ok := st.CycleDay.Get(); ok {
		cycleDay = v
	}
	if v, ok := st.DaysToPeriod.Get(); ok {
		daysToPeriod = v
	}
	if st.DaysLate.Or(0) > 0 { // lateness is a pregnancy signal, never a member's business (T-M9-02, FAM-M1)
		cycleDay, daysToPeriod, nextStart = nil, nil, nil
	}
	phase := enums.MainPhase(st.MainPhase.String())
	return jsonx.Obj(
		"date", today,
		"has_data", hasData,
		"cycle_day", cycleDay,
		"cycle_length", cycleLength,
		"main_phase", sharedPhase(phase, es.Mode() == engine.ModeTeen || st.Limits.FertilitySuppressed),
		"days_to_period", daysToPeriod,
		"predicted_next_period_start", nextStart,
		"confidence", st.Confidence.String(),
	), nil
}

func (r *Reader) symptomsView(ctx context.Context, ownerID uint64, today civildate.Date, scope viewScope) (any, error) {
	from := today.AddDays(-(SymptomDays - 1))
	days, err := r.healthlg.Range(ctx, ownerID, from, today)
	if err != nil {
		return nil, err
	}
	modes := scope.symptomModes()
	out := make([]*jsonx.OrderedMap, 0, len(days))
	for i := len(days) - 1; i >= 0; i-- { // newest first
		var kept []taxonomy.Entry
		for _, e := range days[i].Entries {
			if slices.Contains(SymptomCategories, e.Category) && sharedEntry(e, modes) {
				kept = append(kept, e)
			}
		}
		if len(kept) == 0 {
			continue
		}
		out = append(out, jsonx.Obj("date", days[i].Date, "categories", taxonomy.DayJSON(kept)))
	}
	return jsonx.Obj("from", from, "to", today, "days", out), nil
}

func (r *Reader) medsView(ctx context.Context, ownerID uint64, locale string) (any, error) {
	rows, err := r.care.ListActiveMedications(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("shared: medications: %w", err)
	}
	out := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, row := range rows {
		out = append(out, care.ParseMedication(row).MemberJSON(locale))
	}
	return out, nil
}

func (r *Reader) appointmentsView(ctx context.Context, ownerID uint64, now time.Time) (any, error) {
	rows, err := r.care.ListAppointments(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("shared: appointments: %w", err)
	}
	out := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, row := range rows {
		if a := care.ParseAppointment(row); a.Upcoming(now) && !a.Meta.Private { // CB-LOSS-01: owner-only follow-ups
			out = append(out, a.MemberJSON(now))
		}
	}
	return out, nil
}

func (r *Reader) pregnancyView(ctx context.Context, ownerID uint64, locale string, today civildate.Date, scope viewScope) (any, error) {
	if !scope.pregnant() { // postpartum, a loss or no pregnancy: nothing (T-M9-02, FAM-M3)
		return jsonx.Obj("is_active", false), nil
	}
	p, err := pregnancy.LoadProfile(ctx, r.preg, ownerID)
	if err != nil {
		return nil, err
	}
	if p == nil || !p.PregnancyMode {
		return jsonx.Obj("is_active", false), nil
	}
	st := calc.New(p, locale, today).Status()
	var trimester, due any
	if st.GestationalAge.Valid {
		trimester = st.GestationalAge.Trimester
	}
	if st.DueDate != nil {
		due = st.DueDate.JSON()
	}
	return jsonx.Obj(
		"is_active", true,
		"current_week", st.CurrentWeek,
		"trimester", trimester,
		"gestational_age", st.GestationalAge.JSON(),
		"estimated_due_date", due,
	), nil
}

// late reports whether the owner is past her predicted period: then every number that would count the lateness
// (cycle day past the length, days to period, a past prediction) stays null.
func late(daysLate *int) bool { return daysLate != nil && *daysLate > 0 }
