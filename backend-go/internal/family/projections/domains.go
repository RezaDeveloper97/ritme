package projections

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/care"
	carestore "github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/family"
	healthlogstore "github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
	"github.com/ritme/backend-go/internal/pregnancy"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
	"github.com/ritme/backend-go/internal/pregnancy/labor"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
	"github.com/ritme/backend-go/internal/pregnancy/tools"
	"github.com/ritme/backend-go/internal/vitals"
)

// ---------------------------------------------------------------- pregnancy

// Pregnancy is the pregnancy DTO: the current week, the trimester and the approximate due date (date only). Never
// the LMP, the dating source, scans, notes or alert content.
type Pregnancy struct {
	CurrentWeek *int    `json:"current_week"`
	Trimester   *int    `json:"trimester"`
	EDD         *string `json:"edd"`
}

func (b *Builder) pregnancy(ctx context.Context, ownerID uint64, locale string, today civildate.Date) (any, error) {
	p, err := pregnancy.LoadProfile(ctx, b.preg, ownerID)
	if err != nil {
		return nil, err
	}
	if p == nil || !p.PregnancyMode {
		return Pregnancy{}, nil
	}
	return PregnancyOf(calc.New(p, locale, today).Status()), nil
}

// PregnancyOf copies the allow-listed fields of a calculator status.
func PregnancyOf(st calc.Status) Pregnancy {
	var out Pregnancy
	if st.CurrentWeek > 0 {
		w := st.CurrentWeek
		out.CurrentWeek = &w
	}
	if st.GestationalAge.Valid {
		t := st.GestationalAge.Trimester
		out.Trimester = &t
	}
	if st.DueDate != nil {
		out.EDD = dateOrNil(st.DueDate.Date)
	}
	return out
}

// ---------------------------------------------------------------- fetal activity

// FetalActivity is the fetal_activity DTO (pregnancy only): today's kick-count summary and the last contraction
// session's 5-1-1 status. Never free text or alert evidence.
type FetalActivity struct {
	KicksToday             KicksToday          `json:"kicks_today"`
	LastContractionSession *ContractionSummary `json:"last_contraction_session"`
}

// KicksToday: how many kick sessions were started today and whether one reached the target in time.
type KicksToday struct {
	Date     string  `json:"date"`
	Sessions int     `json:"sessions"`
	Status   *string `json:"status"` // completed | short | null (no session)
}

// ContractionSummary is the last session: contraction count, the last contraction's start, the 5-1-1 status.
type ContractionSummary struct {
	Count         int  `json:"count"`
	LastAt        any  `json:"last_at"` // ISO 8601 (+03:30) or null
	InProgress    bool `json:"in_progress"`
	FiveOneOneMet bool `json:"five_one_one_met"`
}

// Kick statuses.
const (
	KicksCompleted = "completed"
	KicksShort     = "short"
)

func (b *Builder) fetal(ctx context.Context, ownerID uint64, now time.Time) (any, error) {
	today := civildate.InTehran(now)
	ended, err := b.preg.ListKickSessionsBetween(ctx, pregstore.ListKickSessionsBetweenParams{
		UserID: ownerID, DateFrom: today.TehranMidnight(), DateTo: today.AddDays(1).TehranMidnight(),
	})
	if err != nil {
		return nil, fmt.Errorf("projections: kicks: %w", err)
	}
	sessions := len(ended)
	if a, err := b.preg.GetActiveKickSession(ctx, ownerID); err == nil {
		if civildate.InTehran(a.StartedAt) == today {
			sessions++
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("projections: active kicks: %w", err)
	}
	out := FetalActivity{KicksToday: KicksToday{Date: today.String(), Sessions: sessions}}
	if sessions > 0 {
		st := KicksShort
		if tools.SummarizeDay(ended).Normal {
			st = KicksCompleted
		}
		out.KicksToday.Status = &st
	}

	sess, inProgress, err := b.lastContractionSession(ctx, ownerID)
	if err != nil || sess == nil {
		return out, err
	}
	cs, err := b.preg.ListSessionContractions(ctx, pregstore.ListSessionContractionsParams{SessionID: sess.ID, UserID: ownerID})
	if err != nil {
		return nil, fmt.Errorf("projections: contractions: %w", err)
	}
	sum := ContractionSummary{Count: len(cs), InProgress: inProgress, FiveOneOneMet: sess.AlertAt.Valid}
	lc := make([]labor.Contraction, 0, len(cs))
	for _, c := range cs {
		x := labor.Contraction{Start: c.StartedAt}
		if c.EndedAt.Valid {
			x.End = c.EndedAt.Time
		}
		lc = append(lc, x)
	}
	if len(cs) > 0 {
		sum.LastAt = jsonx.ISO8601(cs[len(cs)-1].StartedAt.In(civildate.Tehran))
		if !sum.FiveOneOneMet {
			sum.FiveOneOneMet = labor.FiveOneOne(lc, labor.ParamsOn(civildate.InTehran(sess.StartedAt))).Met
		}
	}
	out.LastContractionSession = &sum
	return out, nil
}

func (b *Builder) lastContractionSession(ctx context.Context, ownerID uint64) (*pregstore.PregnancyContractionSession, bool, error) {
	a, err := b.preg.GetActiveContractionSession(ctx, ownerID)
	switch {
	case err == nil:
		return &a, true, nil
	case !errors.Is(err, sql.ErrNoRows):
		return nil, false, fmt.Errorf("projections: active contractions: %w", err)
	}
	list, err := b.preg.ListContractionSessions(ctx, pregstore.ListContractionSessionsParams{UserID: ownerID, Limit: 1})
	if err != nil {
		return nil, false, fmt.Errorf("projections: contraction sessions: %w", err)
	}
	if len(list) == 0 {
		return nil, false, nil
	}
	return &list[0], false, nil
}

// ---------------------------------------------------------------- appointments

// Appointment is one appointments DTO item: title, time, place, who it is with, status. Never the notes, topic,
// result note, prep, specialty, visit stage, booking id or a private (loss follow-up) appointment.
type Appointment struct {
	ID          uint64  `json:"id"`
	Title       string  `json:"title"`
	ScheduledAt *string `json:"scheduled_at"`
	Location    *string `json:"location"` // the place
	With        *string `json:"with"`
	Status      string  `json:"status"`
}

const wallClock = "2006-01-02 15:04:05"

func (b *Builder) appointments(ctx context.Context, ownerID uint64, now time.Time) (any, error) {
	rows, err := b.care.ListAppointmentsFrom(ctx, carestore.ListAppointmentsFromParams{
		UserID: ownerID, ScheduledAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("projections: appointments: %w", err)
	}
	return AppointmentsOf(rows, now), nil
}

// AppointmentsOf is the upcoming, non-private appointments (scheduled or cancelled) as DTO items.
func AppointmentsOf(rows []carestore.Reminder, now time.Time) []Appointment {
	out := []Appointment{}
	for _, r := range rows {
		a := care.ParseAppointment(r)
		if a.Meta.Private || !r.ScheduledAt.Valid || r.ScheduledAt.Time.Before(now) {
			continue
		}
		at := r.ScheduledAt.Time.In(civildate.Tehran).Format(wallClock)
		out = append(out, Appointment{ID: r.ID, Title: r.Title, ScheduledAt: &at, Location: nonEmpty(a.Meta.Location),
			With: nonEmpty(a.Meta.With), Status: a.Meta.Status})
	}
	return out
}

func nonEmpty(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	v := *s
	return &v
}

// ---------------------------------------------------------------- meds

// Med is one meds DTO item: name (title, the legacy key), form, dose (a number and a known unit only), schedule times; today's dose status
// only for a member at nudge or above. Never notes, a free-text subtitle, dose or unit, reminder settings or history.
type Med struct {
	ID        uint64        `json:"id"`
	Title     string        `json:"title"` // the medication name
	Form      *string       `json:"form"`
	Dose      *string       `json:"dose"`
	Unit      *string       `json:"unit"`
	UnitLabel *string       `json:"unit_label"`
	Times     []string      `json:"times"`
	Today     *[]DoseStatus `json:"today,omitempty"`
}

// DoseStatus is one of today's slots.
type DoseStatus struct {
	Time   string `json:"time"`
	Status string `json:"status"` // taken | missed | pending
}

// Dose statuses.
const (
	DoseTaken   = "taken"
	DoseMissed  = "missed"
	DosePending = "pending"
)

// DoseMissedAfter: a slot not logged this long after its time is missed (family.md §6).
const DoseMissedAfter = 60 * time.Minute

func (b *Builder) meds(ctx context.Context, s family.Snapshot, locale string, now time.Time) (any, error) {
	rows, err := b.care.ListActiveMedications(ctx, s.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("projections: medications: %w", err)
	}
	withToday := family.Decide(s, family.Subject{}, family.DomainMeds, family.ActionNudge, now).Outcome != family.Deny
	var intakes []carestore.ListIntakesOnDateRow
	if withToday {
		if intakes, err = b.care.ListIntakesOnDate(ctx, carestore.ListIntakesOnDateParams{
			UserID: s.OwnerID, IntakeDate: civildate.InTehran(now),
		}); err != nil {
			return nil, fmt.Errorf("projections: intakes: %w", err)
		}
	}
	return MedsOf(rows, intakes, withToday, locale, now), nil
}

// MedsOf builds the meds DTO items; today's status only when withToday.
func MedsOf(rows []carestore.Reminder, intakes []carestore.ListIntakesOnDateRow, withToday bool, locale string, now time.Time) []Med {
	type key struct {
		id   uint64
		slot string
	}
	taken := map[key]bool{}
	for _, in := range intakes {
		taken[key{in.ReminderID, in.Slot}] = true
	}
	today := civildate.InTehran(now)
	out := []Med{}
	for _, r := range rows {
		if !r.IsActive {
			continue
		}
		m := care.ParseMedication(r)
		item := Med{ID: r.ID, Title: r.Title, Times: slices.Clone(m.Meta.Times)}
		if slices.Contains(care.Forms, m.Meta.Form) {
			f := m.Meta.Form
			item.Form = &f
		}
		if m.Meta.Dose != nil && doseNumber(*m.Meta.Dose) {
			d := *m.Meta.Dose
			item.Dose = &d
		}
		if m.Meta.Unit != nil && slices.Contains(care.Units, *m.Meta.Unit) {
			u := *m.Meta.Unit
			item.Unit = &u
			if label, ok := care.Label("units", u, locale); ok {
				item.UnitLabel = &label
			}
		}
		if item.Times == nil {
			item.Times = []string{}
		}
		if withToday {
			list := []DoseStatus{}
			if m.Covers(today) {
				for _, slot := range m.Meta.Times {
					list = append(list, DoseStatus{Time: slot, Status: doseStatus(taken[key{r.ID, slot}], today, slot, now)})
				}
			}
			item.Today = &list
		}
		out = append(out, item)
	}
	return out
}

func doseStatus(taken bool, today civildate.Date, slot string, now time.Time) string {
	if taken {
		return DoseTaken
	}
	at, err := time.ParseInLocation("2006-01-02 15:04", today.String()+" "+slot, civildate.Tehran)
	if err == nil && !now.Before(at.Add(DoseMissedAfter)) {
		return DoseMissed
	}
	return DosePending
}

// doseNumber reports whether a stored dose is a plain number ("400", "0.5"); anything else is free text.
func doseNumber(s string) bool {
	if s == "" || len(s) > 12 {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil && s[0] != '+' && s[0] != '-'
}

// ---------------------------------------------------------------- vitals

// Vitals is the vitals DTO: the latest weight, blood pressure and glucose, each value + category + date (no history,
// no time of day, no targets, notes or alert reasons). Weight has no classification in the vitals domain: its
// category is always null.
type Vitals struct {
	Weight  *Reading `json:"weight"`
	BP      *BP      `json:"bp"`
	Glucose *Reading `json:"glucose"`
}

// Reading is a single-value reading in the owner's unit.
type Reading struct {
	Value    float64 `json:"value"`
	Unit     string  `json:"unit"`
	Category *string `json:"category"`
	Date     string  `json:"date"`
}

// BP is the latest blood pressure.
type BP struct {
	Systolic  int     `json:"systolic"`
	Diastolic int     `json:"diastolic"`
	Category  *string `json:"category"`
	Date      string  `json:"date"`
}

// WeightLookbackDays: the latest weight is looked for in the log of the last 90 days.
const WeightLookbackDays = vitals.LatestLogLookbackDays

func (b *Builder) vitalsOf(ctx context.Context, ownerID uint64, today civildate.Date) (any, error) {
	latest, err := b.vitals.Latest(ctx, ownerID, today)
	if err != nil {
		return nil, err
	}
	logs, err := b.logs.ListLogEntriesBetween(ctx, healthlogstore.ListLogEntriesBetweenParams{
		UserID: ownerID, DateFrom: today.AddDays(-WeightLookbackDays), DateTo: today,
	})
	if err != nil {
		return nil, fmt.Errorf("projections: weight: %w", err)
	}
	return VitalsOf(latest, logs), nil
}

// VitalsOf copies the allow-listed fields of the latest readings and the newest logged weight.
func VitalsOf(latest map[string]*vitals.Reading, logs []healthlogstore.HealthLogEntry) Vitals {
	var out Vitals
	for i := len(logs) - 1; i >= 0; i-- { // ordered by date ascending: the newest last
		e := logs[i]
		if e.Category != "measurements" || e.Param != "weight" || e.Item != "" || !e.ValueNum.Valid {
			continue
		}
		v, err := strconv.ParseFloat(e.ValueNum.String, 64)
		if err != nil || v <= 0 {
			continue
		}
		out.Weight = &Reading{Value: phpround.Round(v, 1), Unit: "kg", Date: e.LogDate.String()}
		break
	}
	if r := latest[vitals.TypeBP]; r != nil && r.Systolic > 0 && r.Diastolic > 0 {
		out.BP = &BP{Systolic: int(r.Systolic), Diastolic: int(r.Diastolic), Category: category(r.Class()), Date: r.Date.String()}
	}
	if r := latest[vitals.TypeGlucose]; r != nil && r.MgDl > 0 {
		unit := r.Unit
		if !slices.Contains(vitals.GlucoseUnits, unit) {
			unit = vitals.UnitMgDl
		}
		out.Glucose = &Reading{Value: vitals.InUnit(r.MgDl, unit), Unit: unit, Category: category(r.Class()), Date: r.Date.String()}
	}
	return out
}

func category(c string) *string {
	if c == "" {
		return nil
	}
	return &c
}
