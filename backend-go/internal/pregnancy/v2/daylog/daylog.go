// Package daylog is the pregnancy v2 day log (T-M7-03): GET/PUT /pregnancy/v2/days/{date}
// and GET /pregnancy/v2/report. It stores nothing of its own: the v1 symptoms (and their
// alert rules) go through pregnancy.SaveSymptomDay, the weight through
// pregnancy.SaveWeeklyWeight, and mood / water / heartburn / constipation / visit note go to
// pregnancy_daily_extras.
package daylog

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

// Symptoms is the v2 day-log symptom list, in display order.
var Symptoms = []string{
	"nausea", "vomiting", "fatigue", "headache", "back_pain", "breast_pain", "heartburn", "constipation", "spotting",
}

// Severities are the allowed symptom severities.
var Severities = []string{"mild", "moderate", "severe"}

// extraSymptoms live in pregnancy_daily_extras; every other symptom is a v1 column.
var extraSymptoms = map[string]bool{"heartburn": true, "constipation": true}

// v1Symptoms are the Symptoms stored in pregnancy_symptom_logs.
var v1Symptoms = func() []string {
	var out []string
	for _, s := range Symptoms {
		if !extraSymptoms[s] {
			out = append(out, s)
		}
	}
	return out
}()

// MaxReportDays is the longest report range.
const MaxReportDays = 366

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

// T is the pregnancy_daylog line for key in locale (:placeholders from kv pairs).
func T(key, locale string, kv ...string) string {
	params := make(map[string]string, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		params[kv[i]] = kv[i+1]
	}
	return translator().Trans("pregnancy_daylog."+key, params, locale)
}

func attributes(locale string) []string {
	line, ok := translator().Get("pregnancy_daylog.attributes", locale)
	m, isMap := line.(phpval.Map)
	if !ok || !isMap {
		return nil
	}
	var kv []string
	for _, k := range m.Keys() {
		if v, ok := m.Get(k); ok {
			if s, ok := v.(string); ok {
				kv = append(kv, k, s)
			}
		}
	}
	return kv
}

// Evaluator runs the v2 alert rules after a save and returns the alerts it created
// (messages/pregnancyalerts.Engine, T-M7-04).
type Evaluator interface {
	Evaluate(ctx context.Context, userID uint64, now time.Time, l v2.Lang) ([]*jsonx.OrderedMap, error)
}

// Service reads and writes the day log.
type Service struct {
	q      store.Querier
	alerts Evaluator
}

// NewService wires the service; alerts may be nil (no v2 alert rules).
func NewService(q store.Querier, alerts Evaluator) *Service { return &Service{q: q, alerts: alerts} }

// Input is a validated PUT body: a nil field was absent (keep); a set field whose inner
// pointer is nil clears.
type Input struct {
	Mood      **int
	Water     **int
	Weight    **float64
	WeightRaw any // the validated weight value handed to the v1 weekly log
	VisitNote **string
	Symptoms  *map[string]string // replaces the whole set; nil map entry = absent
}

// weekOf is the 1-based pregnancy week of date, clamped to 1..MaxWeek.
func weekOf(d v2.Dating, date civildate.Date) int {
	return min(v2.MaxWeek, max(1, d.Start().DiffDays(date)/7+1))
}

func (s *Service) dating(ctx context.Context, userID uint64, today civildate.Date, locale string) (v2.Dating, error) {
	p, err := s.q.GetProfileByUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return v2.Dating{}, errNotActive(locale)
	}
	if err != nil {
		return v2.Dating{}, fmt.Errorf("pregnancy daylog: profile: %w", err)
	}
	if !p.PregnancyMode {
		return v2.Dating{}, errNotActive(locale)
	}
	d, ok := v2.Resolve(&p, today)
	if !ok {
		return v2.Dating{}, errNotActive(locale)
	}
	return d, nil
}

// errNotActive is v2's 409 {success:false, message, error_code: pregnancy_not_active}.
func errNotActive(locale string) error {
	return httpx.Fail(fiber.StatusConflict, v2.T("messages.not_active", locale), "error_code", "pregnancy_not_active")
}

func noRows[T any](row T, err error) (*T, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// v1Severities reads the v2 symptoms of a v1 symptom log (has_X true → severity, mild when unset).
func v1Severities(r *store.PregnancySymptomLog) map[string]string {
	out := map[string]string{}
	if r == nil {
		return out
	}
	cols := map[string][2]any{
		"nausea": {r.HasNausea, r.NauseaSeverity}, "vomiting": {r.HasVomiting, r.VomitingSeverity},
		"fatigue": {r.HasFatigue, r.FatigueSeverity}, "headache": {r.HasHeadache, r.HeadacheSeverity},
		"back_pain": {r.HasBackPain, r.BackPainSeverity}, "breast_pain": {r.HasBreastPain, r.BreastPainSeverity},
		"spotting": {r.HasSpotting, r.SpottingSeverity},
	}
	for name, c := range cols {
		has, _ := c[0].(sql.NullBool)
		sev, _ := c[1].(sql.NullString)
		if has.Valid && has.Bool {
			if sev.Valid && sev.String != "" {
				out[name] = sev.String
			} else {
				out[name] = "mild"
			}
		}
	}
	return out
}

func symptomsJSON(v1 map[string]string, ex *store.PregnancyDailyExtra) *jsonx.OrderedMap {
	all := map[string]string{}
	for k, v := range v1 {
		all[k] = v
	}
	if ex != nil {
		if ex.HeartburnSeverity.Valid {
			all["heartburn"] = ex.HeartburnSeverity.String
		}
		if ex.ConstipationSeverity.Valid {
			all["constipation"] = ex.ConstipationSeverity.String
		}
	}
	out := jsonx.Obj()
	for _, s := range Symptoms {
		if v, ok := all[s]; ok {
			out.Set(s, v)
		}
	}
	return out
}

func nullInt(n sql.NullInt16) any {
	if !n.Valid {
		return nil
	}
	return int(n.Int16)
}

func nullStr(n sql.NullString) any {
	if !n.Valid {
		return nil
	}
	return n.String
}

func weightNum(n sql.NullString) any {
	if !n.Valid {
		return nil
	}
	f, err := strconv.ParseFloat(n.String, 64)
	if err != nil {
		return nil
	}
	return f
}

// Day is the merged day read model.
func (s *Service) Day(ctx context.Context, userID uint64, date civildate.Date, now time.Time, locale string,
	raised []*jsonx.OrderedMap,
) (*jsonx.OrderedMap, error) {
	d, err := s.dating(ctx, userID, civildate.InTehran(now), locale)
	if err != nil {
		return nil, err
	}
	week := weekOf(d, date)
	sym, err := noRows(s.q.GetSymptomLog(ctx, store.GetSymptomLogParams{UserID: userID, LogDate: date}))
	if err != nil {
		return nil, fmt.Errorf("pregnancy daylog: symptoms: %w", err)
	}
	ex, err := noRows(s.q.GetDailyExtras(ctx, store.GetDailyExtrasParams{UserID: userID, LogDate: date}))
	if err != nil {
		return nil, fmt.Errorf("pregnancy daylog: extras: %w", err)
	}
	wl, err := noRows(s.q.GetWeeklyLog(ctx, store.GetWeeklyLogParams{UserID: userID, PregnancyWeek: int32(week)})) //nolint:gosec // 1..42
	if err != nil {
		return nil, fmt.Errorf("pregnancy daylog: weekly: %w", err)
	}
	last, err := noRows(s.q.GetV2LastWeight(ctx, userID))
	if err != nil {
		return nil, fmt.Errorf("pregnancy daylog: last weight: %w", err)
	}

	var mood, water, note, weight any
	if ex != nil {
		mood, water, note = nullInt(ex.Mood), nullInt(ex.WaterGlasses), nullStr(ex.VisitNote)
	}
	if wl != nil {
		weight = weightNum(wl.Weight)
	}
	var lastWeight any
	if last != nil {
		if v := weightNum(last.Weight); v != nil {
			lastWeight = jsonx.Obj("value", v, "date", last.LogDate.String(),
				"date_label", v2.FullDate(last.LogDate, locale), "week", int(last.PregnancyWeek))
		}
	}
	return jsonx.Obj(
		"date", date.String(),
		"date_label", v2.FullDate(date, locale),
		"week", week,
		"mood", mood,
		"symptoms", symptomsJSON(v1Severities(sym), ex),
		"water_glasses", water,
		"weight", weight,
		"visit_note", note,
		"last_weight", lastWeight,
		"alerts", alertsJSON(raised),
	), nil
}

func alertsJSON(raised []*jsonx.OrderedMap) []any {
	out := make([]any, 0, len(raised))
	for _, a := range raised {
		out = append(out, a)
	}
	return out
}

// Save applies a validated PUT and returns the merged day with the alerts this save raised.
// Writes happen only when something changes, so a resent PUT raises no alerts twice. The
// writes (symptom log + its v1 alerts, extras, weekly log + its v1 alerts) and the v2 alert
// evaluation run in one transaction (review #9, T-M2-34): a failure leaves nothing half-saved.
// A date before the pregnancy start is a 422 (review #8): it would clamp to week 1.
func (s *Service) Save(ctx context.Context, userID uint64, date civildate.Date, in Input, now time.Time, l v2.Lang,
) (*jsonx.OrderedMap, error) {
	locale := l.Locale
	d, err := s.dating(ctx, userID, civildate.InTehran(now), locale)
	if err != nil {
		return nil, err
	}
	if err := checkInPregnancy(d, date, locale); err != nil {
		return nil, err
	}

	q, commit, rollback, err := s.begin(ctx)
	if err != nil {
		return nil, err
	}
	defer rollback()
	if err := s.write(ctx, q, userID, date, d, in, now, locale); err != nil {
		return nil, err
	}
	var raised []*jsonx.OrderedMap
	if s.alerts != nil {
		if tx, ok := s.alerts.(TxEvaluator); ok {
			raised, err = tx.EvaluateOn(ctx, q, userID, now, l)
		} else {
			raised, err = s.alerts.Evaluate(ctx, userID, now, l)
		}
		if err != nil {
			return nil, err
		}
	}
	if err := commit(); err != nil {
		return nil, err
	}
	return s.Day(ctx, userID, date, now, locale, raised)
}

// TxEvaluator is an Evaluator that can run on a given querier (the day-log transaction).
type TxEvaluator interface {
	EvaluateOn(ctx context.Context, q store.Querier, userID uint64, now time.Time, l v2.Lang) ([]*jsonx.OrderedMap, error)
}

// begin opens the save transaction when the querier is a *store.Queries on a connection
// pool; otherwise (a test double, an outer transaction) it writes through s.q directly.
func (s *Service) begin(ctx context.Context) (store.Querier, func() error, func(), error) {
	sq, ok := s.q.(*store.Queries)
	if !ok {
		return s.q, func() error { return nil }, func() {}, nil
	}
	tx, qtx, err := sq.BeginTx(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("pregnancy daylog: begin: %w", err)
	}
	if tx == nil {
		return qtx, func() error { return nil }, func() {}, nil
	}
	done := false
	commit := func() error {
		done = true
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("pregnancy daylog: commit: %w", err)
		}
		return nil
	}
	rollback := func() {
		if !done {
			_ = tx.Rollback()
		}
	}
	return qtx, commit, rollback, nil
}

// checkInPregnancy refuses a day before the pregnancy start (422 on date).
func checkInPregnancy(d v2.Dating, date civildate.Date, locale string) error {
	if date.Before(d.Start()) {
		return fieldError(locale, "date", T("validation.before_pregnancy", locale))
	}
	return nil
}

// write applies in through q (the save transaction).
func (s *Service) write(ctx context.Context, q store.Querier, userID uint64, date civildate.Date, d v2.Dating, in Input,
	now time.Time, locale string,
) error {
	if in.Symptoms != nil {
		row, err := noRows(q.GetSymptomLog(ctx, store.GetSymptomLogParams{UserID: userID, LogDate: date}))
		if err != nil {
			return fmt.Errorf("pregnancy daylog: symptoms: %w", err)
		}
		cur := v1Severities(row)
		want := *in.Symptoms
		changed := false
		attrs := phpval.NewMap()
		for _, name := range v1Symptoms {
			sev, on := want[name]
			if cur[name] != sev {
				changed = true
			}
			if on {
				attrs.Set("has_"+name, true)
				attrs.Set(name+"_severity", sev)
			} else {
				attrs.Set("has_"+name, false)
				attrs.Set(name+"_severity", nil)
			}
		}
		if changed {
			// The v1 rules still write their pregnancy_alerts rows (v1 parity); the day
			// returns the v2 rule alerts.
			if _, err := pregnancy.SaveSymptomDay(ctx, q, userID, date, attrs, locale, now); err != nil {
				return err
			}
		}
	}

	if err := saveExtras(ctx, q, userID, date, in, now); err != nil {
		return err
	}

	if in.Weight != nil {
		week := weekOf(d, date)
		wl, err := noRows(q.GetWeeklyLog(ctx, store.GetWeeklyLogParams{UserID: userID, PregnancyWeek: int32(week)})) //nolint:gosec // 1..42
		if err != nil {
			return fmt.Errorf("pregnancy daylog: weekly: %w", err)
		}
		var cur *float64
		if wl != nil {
			if v, ok := weightNum(wl.Weight).(float64); ok {
				cur = &v
			}
		}
		next := *in.Weight
		same := (cur == nil && next == nil) || (cur != nil && next != nil && *cur == *next && wl.LogDate == date)
		if !same && (wl != nil || next != nil) {
			if _, err := pregnancy.SaveWeeklyWeight(ctx, q, userID, week, date, in.WeightRaw, locale, now); err != nil {
				return err
			}
		}
	}
	return nil
}

func toNullInt(p *int) sql.NullInt16 {
	if p == nil {
		return sql.NullInt16{}
	}
	return sql.NullInt16{Int16: int16(*p), Valid: true} //nolint:gosec // validated 0..15
}

func toNullStr(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}

func saveExtras(ctx context.Context, q store.Querier, userID uint64, date civildate.Date, in Input, now time.Time) error {
	touched := in.Mood != nil || in.Water != nil || in.VisitNote != nil || in.Symptoms != nil
	if !touched {
		return nil
	}
	ex, err := noRows(q.GetDailyExtras(ctx, store.GetDailyExtrasParams{UserID: userID, LogDate: date}))
	if err != nil {
		return fmt.Errorf("pregnancy daylog: extras: %w", err)
	}
	p := store.UpsertDailyExtrasParams{UserID: userID, LogDate: date, Now: sql.NullTime{Time: now, Valid: true}}
	if ex != nil {
		p.Mood, p.WaterGlasses, p.VisitNote = ex.Mood, ex.WaterGlasses, ex.VisitNote
		p.HeartburnSeverity, p.ConstipationSeverity = ex.HeartburnSeverity, ex.ConstipationSeverity
	}
	if in.Mood != nil {
		p.Mood = toNullInt(*in.Mood)
	}
	if in.Water != nil {
		p.WaterGlasses = toNullInt(*in.Water)
	}
	if in.VisitNote != nil {
		p.VisitNote = toNullStr(*in.VisitNote)
	}
	if in.Symptoms != nil {
		for name := range extraSymptoms {
			var v *string
			if sev, ok := (*in.Symptoms)[name]; ok {
				v = &sev
			}
			if name == "heartburn" {
				p.HeartburnSeverity = toNullStr(v)
			} else {
				p.ConstipationSeverity = toNullStr(v)
			}
		}
	}
	if ex != nil && ex.Mood == p.Mood && ex.WaterGlasses == p.WaterGlasses && ex.VisitNote == p.VisitNote &&
		ex.HeartburnSeverity == p.HeartburnSeverity && ex.ConstipationSeverity == p.ConstipationSeverity {
		return nil // unchanged: keep updated_at
	}
	if ex == nil && !p.Mood.Valid && !p.WaterGlasses.Valid && !p.VisitNote.Valid &&
		!p.HeartburnSeverity.Valid && !p.ConstipationSeverity.Valid {
		return nil // nothing to store
	}
	if err := q.UpsertDailyExtras(ctx, p); err != nil {
		return fmt.Errorf("pregnancy daylog: upsert extras: %w", err)
	}
	return nil
}

// Report is the doctor-PDF timeline for from..to (inclusive): logged days + weights.
func (s *Service) Report(ctx context.Context, userID uint64, from, to civildate.Date, now time.Time, locale string,
) (*jsonx.OrderedMap, error) {
	d, err := s.dating(ctx, userID, civildate.InTehran(now), locale)
	if err != nil {
		return nil, err
	}
	syms, err := s.q.ListV2SymptomLogsRange(ctx, store.ListV2SymptomLogsRangeParams{UserID: userID, DateFrom: from, DateTo: to})
	if err != nil {
		return nil, fmt.Errorf("pregnancy daylog: report symptoms: %w", err)
	}
	extras, err := s.q.ListDailyExtrasRange(ctx, store.ListDailyExtrasRangeParams{UserID: userID, DateFrom: from, DateTo: to})
	if err != nil {
		return nil, fmt.Errorf("pregnancy daylog: report extras: %w", err)
	}
	weights, err := s.q.ListV2WeightsRange(ctx, store.ListV2WeightsRangeParams{UserID: userID, DateFrom: from, DateTo: to})
	if err != nil {
		return nil, fmt.Errorf("pregnancy daylog: report weights: %w", err)
	}

	symBy := map[civildate.Date]*store.PregnancySymptomLog{}
	exBy := map[civildate.Date]*store.PregnancyDailyExtra{}
	var dates []civildate.Date
	seen := map[civildate.Date]bool{}
	add := func(dt civildate.Date) {
		if !seen[dt] {
			seen[dt] = true
			dates = append(dates, dt)
		}
	}
	for i := range syms {
		symBy[syms[i].LogDate] = &syms[i]
		add(syms[i].LogDate)
	}
	for i := range extras {
		exBy[extras[i].LogDate] = &extras[i]
		add(extras[i].LogDate)
	}
	days := []any{}
	for dt := from; !dt.After(to); dt = dt.AddDays(1) {
		if !seen[dt] {
			continue
		}
		ex := exBy[dt]
		sj := symptomsJSON(v1Severities(symBy[dt]), ex)
		var mood, water, note any
		if ex != nil {
			mood, water, note = nullInt(ex.Mood), nullInt(ex.WaterGlasses), nullStr(ex.VisitNote)
		}
		if sj.Len() == 0 && mood == nil && water == nil && note == nil {
			continue
		}
		days = append(days, jsonx.Obj(
			"date", dt.String(), "date_label", v2.FullDate(dt, locale), "week", weekOf(d, dt),
			"mood", mood, "symptoms", sj, "water_glasses", water, "visit_note", note,
		))
	}
	ws := []any{}
	for _, w := range weights {
		if v := weightNum(w.Weight); v != nil {
			ws = append(ws, jsonx.Obj("date", w.LogDate.String(), "week", int(w.PregnancyWeek), "value", v))
		}
	}
	return jsonx.Obj(
		"range", jsonx.Obj("from", from.String(), "to", to.String()),
		"range_label", v2.RangeLabel(from, to, locale),
		"generated_at", now.In(civildate.Tehran).Format(time.RFC3339),
		"profile", jsonx.Obj("weeks", d.Weeks, "days", d.Days, "week", d.CurrentWeek(), "due_date", d.Due.String()),
		"due_date", d.Due.String(),
		"days", days,
		"weights", ws,
	), nil
}

// trimNote normalises a visit note: trimmed, empty → nil.
func trimNote(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
