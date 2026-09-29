// Package calendar is GET /api/v1/pregnancy/v2/calendar (T-M7-05): the month grid (visit / week-start /
// today markers), the M3 appointments of the month as pregnancy visits, the next visit and the admin
// care plan with each item's window (week_from–week_to → dates from the due date), state
// (done / booked / to_book, from appointments linked by meta.care_item_key) and suggested date.
// The month is in the locale's calendar (Jalali for fa).
package calendar

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

//go:embed lang/*/*.json
var langFS embed.FS

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err)
	}
	return t
})

// T is the pregnancy_calendar line for key in locale.
func T(key, locale string) string {
	return translator().Trans("pregnancy_calendar."+key, nil, locale)
}

// Month is a month of the locale's calendar with its Gregorian first and last day.
type Month struct {
	Year, Month int
	First, Last civildate.Date
}

// MonthOf builds the month (y, m) of the locale's calendar.
func MonthOf(y, m int, locale string) Month {
	if v2.IsJalali(locale) {
		first := engine.FromJalali(y, m, 1)
		return Month{Year: y, Month: m, First: first, Last: engine.JalaliMonthEnd(first)}
	}
	first := civildate.New(y, time.Month(m), 1)
	return Month{Year: y, Month: m, First: first, Last: civildate.New(y, time.Month(m)+1, 1).AddDays(-1)}
}

// MonthContaining is the locale-calendar month that contains d.
func MonthContaining(d civildate.Date, locale string) Month {
	y, m, _ := v2.CalendarDate(d, locale)
	return MonthOf(y, m, locale)
}

// Visit is an M3 appointment as a pregnancy visit.
type Visit struct {
	ID           uint64
	Title        string
	At           time.Time // Tehran
	Date         civildate.Date
	CareItemKey  string
	Stage        string
	ResultNote   string
	Doctor       string
	Place        string
	RemindBefore string
}

type visitMeta struct {
	CareItemKey  *string `json:"care_item_key"`
	Stage        *string `json:"stage"`
	ResultNote   *string `json:"result_note"`
	With         *string `json:"with"`
	Location     *string `json:"location"`
	RemindBefore *string `json:"remind_before"`
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// VisitsOf parses the appointment rows (soonest first).
func VisitsOf(rows []store.ListV2CalendarAppointmentsRow) []Visit {
	out := make([]Visit, 0, len(rows))
	for _, r := range rows {
		if !r.ScheduledAt.Valid {
			continue
		}
		var m visitMeta
		if r.Meta.Valid {
			_ = json.Unmarshal(r.Meta.V, &m)
		}
		at := r.ScheduledAt.Time.In(civildate.Tehran)
		stage := deref(m.Stage)
		if stage != "booked" && stage != "done" && stage != "result" {
			stage = ""
		}
		out = append(out, Visit{
			ID: r.ID, Title: r.Title, At: at, Date: civildate.FromTime(at),
			CareItemKey: deref(m.CareItemKey), Stage: stage, ResultNote: deref(m.ResultNote),
			Doctor: deref(m.With), Place: deref(m.Location), RemindBefore: deref(m.RemindBefore),
		})
	}
	return out
}

// Care item states.
const (
	StateDone   = "done"
	StateBooked = "booked"
	StateToBook = "to_book"
)

// ItemState is a care item resolved against the user's visits and dating.
type ItemState struct {
	State     string
	From, To  civildate.Date // window dates
	Visit     *Visit         // the linked visit shown (done one, else the next booked, else the last)
	Suggested *civildate.Date
}

// ResolveItem computes a care item's window, state and suggested date. A linked visit with stage
// done|result makes the item done; any other linked (non-cancelled) visit makes it booked. The
// suggested date (to_book only) is the window start, or today when the window is already open;
// none once the window has ended.
func ResolveItem(weekFrom, weekTo int, key string, visits []Visit, d v2.Dating) ItemState {
	from, _ := d.WeekRange(max(1, weekFrom))
	_, to := d.WeekRange(max(weekFrom, weekTo))
	st := ItemState{State: StateToBook, From: from, To: to}
	var done, next, last *Visit
	for i := range visits {
		v := &visits[i]
		if key == "" || v.CareItemKey != key {
			continue
		}
		if (v.Stage == "done" || v.Stage == "result") && done == nil {
			done = v
		}
		if next == nil && !v.Date.Before(d.Today) {
			next = v
		}
		last = v
	}
	switch {
	case done != nil:
		st.State, st.Visit = StateDone, done
	case next != nil:
		st.State, st.Visit = StateBooked, next
	case last != nil:
		st.State, st.Visit = StateBooked, last
	default:
		if !d.Today.After(to) {
			s := from
			if s.Before(d.Today) {
				s = d.Today
			}
			st.Suggested = &s
		}
	}
	return st
}

// Service builds the calendar.
type Service struct{ q store.Querier }

// NewService wires the service.
func NewService(q store.Querier) *Service { return &Service{q: q} }

func errNotActive(locale string) error {
	return httpx.Fail(fiber.StatusConflict, v2.T("messages.not_active", locale), "error_code", "pregnancy_not_active")
}

// weekOn is the 1-based week containing date (0 before the pregnancy start).
// sourceNote is the care-plan caveat («… ممکنه پزشکت برنامهٔ متفاوتی بده.») followed by the dating
// basis sentence (design audit E2); null when neither has copy. Each part is the admin-edited
// pregnancy_setup/calendar_note text (`plan_note`, `basis_<source>`; admin is nil without a live
// row) and falls back to the pregnancy_calendar lang line.
func sourceNote(source string, admin map[string]any, loc string) any {
	var parts []string
	for _, p := range [][2]string{{"plan_note", "plan_note"}, {"basis_" + source, "source_note." + source}} {
		s, _ := admin[p[0]].(string)
		if s = strings.TrimSpace(s); s == "" {
			s = T(p[1], loc)
		}
		if s != "" && !strings.HasPrefix(s, "pregnancy_calendar.") {
			parts = append(parts, s)
		}
	}
	return nullIf(strings.Join(parts, " "))
}

// ageAt is the gestational age on date, «۱۱ هفته و ۱ روز»; nil before the pregnancy start.
func ageAt(d v2.Dating, date civildate.Date, loc string) any {
	n := d.Start().DiffDays(date)
	if n < 0 {
		return nil
	}
	return v2.AgeLabel(n/7, n%7, loc)
}

func weekOn(d v2.Dating, date civildate.Date) int {
	n := d.Start().DiffDays(date)
	if n < 0 {
		return 0
	}
	return min(v2.MaxWeek, n/7+1)
}

func nullIf(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func dateOrNil(d *civildate.Date) any {
	if d == nil {
		return nil
	}
	return d.String()
}

func labelOrNil(d *civildate.Date, locale string) any {
	if d == nil {
		return nil
	}
	return v2.FullDate(*d, locale)
}

// Calendar is GET /pregnancy/v2/calendar for month (nil = the month of today).
func (s *Service) Calendar(ctx context.Context, userID uint64, month *[2]int, now time.Time, l v2.Lang) (*jsonx.OrderedMap, error) {
	today := civildate.InTehran(now)
	p, err := s.q.GetProfileByUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !p.PregnancyMode) {
		return nil, errNotActive(l.Locale)
	}
	if err != nil {
		return nil, fmt.Errorf("pregnancy calendar: profile: %w", err)
	}
	d, ok := v2.Resolve(&p, today)
	if !ok {
		return nil, errNotActive(l.Locale)
	}
	rows, err := s.q.ListV2CalendarAppointments(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("pregnancy calendar: appointments: %w", err)
	}
	items, err := s.q.ListActiveCareItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("pregnancy calendar: care items: %w", err)
	}

	note, err := v2.MessagePayload(ctx, s.q, v2.SetupGroup, v2.CalendarNoteItem, l)
	if err != nil {
		return nil, fmt.Errorf("pregnancy calendar: %w", err)
	}

	m := MonthContaining(today, l.Locale)
	if month != nil {
		m = MonthOf(month[0], month[1], l.Locale)
	}
	visits := VisitsOf(rows)
	byKey := map[string]store.PregnancyCareItem{}
	for _, it := range items {
		byKey[it.Key] = it
	}
	return build(d, m, visits, items, byKey, note, l), nil
}

func build(d v2.Dating, m Month, visits []Visit, items []store.PregnancyCareItem, byKey map[string]store.PregnancyCareItem,
	note map[string]any, l v2.Lang,
) *jsonx.OrderedMap {
	loc := l.Locale
	visitJSON := func(v Visit) *jsonx.OrderedMap {
		var prep any
		if it, ok := byKey[v.CareItemKey]; ok && it.Prep.Valid {
			prep = nullIf(i18n.PickString(it.Prep.V, l.Locale, l.Default))
		}
		week := weekOn(d, v.Date)
		var weekAny, weekLabel any
		if week > 0 {
			weekAny, weekLabel = week, v2.WeekLabel(week, loc)
		}
		return jsonx.Obj(
			"appointment_id", v.ID, "care_item_key", nullIf(v.CareItemKey), "title", v.Title,
			"date", v.Date.String(), "date_label", v2.FullDate(v.Date, loc), "time", v.At.Format("15:04"),
			"week", weekAny, "week_label", weekLabel, "age_label", ageAt(d, v.Date, loc), "stage", nullIf(v.Stage), "prep", prep,
			"doctor", nullIf(v.Doctor), "place", nullIf(v.Place), "remind_before", nullIf(v.RemindBefore),
			"result_note", nullIf(v.ResultNote), "days_until", d.Today.DiffDays(v.Date),
		)
	}

	hasVisit := map[civildate.Date]bool{}
	monthVisits := []any{}
	var next any
	for _, v := range visits {
		if next == nil && !v.Date.Before(d.Today) {
			next = visitJSON(v)
		}
		if v.Date.Before(m.First) || v.Date.After(m.Last) {
			continue
		}
		hasVisit[v.Date] = true
		monthVisits = append(monthVisits, visitJSON(v))
	}

	days := []any{}
	start := d.Start()
	for day := m.First; !day.After(m.Last); day = day.AddDays(1) {
		var weekStart any
		if n := start.DiffDays(day); n >= 0 && n%7 == 0 && n/7+1 <= v2.MaxWeek {
			weekStart = n/7 + 1
		}
		days = append(days, jsonx.Obj(
			"date", day.String(), "has_visit", hasVisit[day], "week_start", weekStart, "is_today", day == d.Today,
		))
	}

	var weekRange any
	if wf, wt := weekOn(d, m.First), weekOn(d, m.Last); wt > 0 {
		weekRange = jsonx.Obj("from", max(1, wf), "to", wt)
	}

	plan := []any{}
	for _, it := range items {
		st := ResolveItem(int(it.WeekFrom), int(it.WeekTo), it.Key, visits, d)
		var date, dateLabel, apptID any
		if st.Visit != nil {
			date, dateLabel, apptID = st.Visit.Date.String(), v2.FullDate(st.Visit.Date, loc), st.Visit.ID
		}
		var prep any
		if it.Prep.Valid {
			prep = nullIf(i18n.PickString(it.Prep.V, l.Locale, l.Default))
		}
		plan = append(plan, jsonx.Obj(
			"key", it.Key, "title", i18n.PickString(it.Title, l.Locale, l.Default), "kind", it.Kind,
			"week_from", int(it.WeekFrom), "week_to", int(it.WeekTo),
			"window", jsonx.Obj("from", st.From.String(), "to", st.To.String()),
			"window_label", v2.RangeLabel(st.From, st.To, loc),
			"state", st.State, "date", date, "date_label", dateLabel,
			"suggested_date", dateOrNil(st.Suggested), "suggested_date_label", labelOrNil(st.Suggested, loc),
			"appointment_id", apptID, "prep", prep,
		))
	}

	return jsonx.Obj(
		"month", fmt.Sprintf("%04d-%02d", m.Year, m.Month),
		"month_label", v2.MonthLabel(m.Year, m.Month, loc),
		"calendar", map[bool]string{true: "jalali", false: "gregorian"}[v2.IsJalali(loc)],
		"from", m.First.String(),
		"to", m.Last.String(),
		"today", d.Today.String(),
		"week_range", weekRange,
		"days", days,
		"visits", monthVisits,
		"next_visit", next,
		"care_plan", plan,
		"source_note", sourceNote(d.Source, note, loc),
	)
}
