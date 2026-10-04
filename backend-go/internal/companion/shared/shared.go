// Package shared builds the access-filtered views of an owner's data that a companion may read (bloom B-N4-02), one
// per companion section, by reusing the owning domains (cycle engine, taxonomy-v2 day log, care reminders,
// pregnancy calculator). It never decides access: callers check companion.Service.Level first and audit the read.
//
// Each view is a deliberate subset of what the owner sees herself:
//   - cycle: today's cycle day, phase and the next-period prediction (no logs, no fertility-test values);
//   - symptoms: the last SymptomDays days of the day log, limited to SymptomCategories (pain, mood, symptoms, sleep,
//     energy/appetite, skin & hair) — never sex, discharge, urogenital, measurements, bleeding detail or free-text
//     notes;
//   - meds: the owner's active medication reminders (the /care resource);
//   - appointments: the owner's upcoming appointments (the /care resource) except private ones (loss follow-ups);
//   - pregnancy: the pregnancy status (week, due date, trimester) or is_active=false.
//
// The views are mode-aware (B-N4-08b, CMP-H1 / CMP-M1 of docs/security/bloom-companion.md): ReadFor gets the link, so
// a pregnancy / postpartum owner whose link has no pregnancy grant (and every menopause owner) gets the neutral cycle
// view, the symptoms view keeps only what the cycle-mode log offers (mode.go), lateness past MaxLateDays is never
// shown, and the fertile phase is mapped away for owners whose mode never gets fertility copy (teen).
package shared

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/care"
	carestore "github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/resolver"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// SymptomDays is how many days (today included) the symptoms view covers.
const SymptomDays = 7

// SymptomCategories are the taxonomy-v2 categories the symptoms section shares («علائم و حال روزانه»).
var SymptomCategories = []string{"pain", "mood", "symptoms", "sleep", "appetite_energy", "skin_hair"}

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
	today := civildate.InTehran(now)
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
		return r.pregnancyView(ctx, ownerID, locale, today)
	default:
		return nil, companion.ErrInvalidSection
	}
}

func (r *Reader) cycleView(ctx context.Context, ownerID uint64, today civildate.Date, scope viewScope) (any, error) {
	if scope.neutralCycle() {
		return neutralCycle(today), nil
	}
	sn, err := r.cycle.Load(ctx, ownerID, today, today, today)
	if err != nil {
		return nil, err
	}
	profile := sn.EngineProfile()
	m := metrics.Calculate(sn.Histories, profile)
	st := resolver.Resolve(sn.Histories, profile, today, today, m)
	var cycleLength, nextStart, cycleDay, daysLate any
	if st.EffectiveCycleLength > 0 {
		cycleLength = st.EffectiveCycleLength
	}
	if !st.PredictedNextPeriodStart.IsZero() {
		nextStart = st.PredictedNextPeriodStart
	}
	cycleDay, daysLate = st.CycleDay, st.DaysLate
	if tooLate(st.DaysLate) { // a long lateness is an inference, not a cycle state (CMP-H1)
		cycleDay, daysLate = nil, nil
	}
	return jsonx.Obj(
		"date", today,
		"has_data", len(sn.Histories) > 0 || sn.Profile != nil,
		"cycle_day", cycleDay,
		"cycle_length", cycleLength,
		"main_phase", sharedPhase(st.MainPhase, profile != nil && profile.NoFertilityCopy),
		"days_to_period", st.DaysToPeriod,
		"days_late", daysLate,
		"predicted_next_period_start", nextStart,
		"confidence", st.Confidence,
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
		out = append(out, care.ParseMedication(row).JSON(locale))
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
			out = append(out, a.JSON(now))
		}
	}
	return out, nil
}

func (r *Reader) pregnancyView(ctx context.Context, ownerID uint64, locale string, today civildate.Date) (any, error) {
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
