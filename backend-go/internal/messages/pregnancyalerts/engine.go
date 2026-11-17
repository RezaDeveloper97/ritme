package pregnancyalerts

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
	"sync/atomic"
	"time"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/labor"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

// Group is the message_contents group of the rules and the legend.
const Group = "pregnancy_alert"

// LegendKey is the legend row of Group.
const LegendKey = "legend"

// ListWindowDays is the window of GET /pregnancy/v2/alerts (the Today badge counts the same window).
const ListWindowDays = v2.AlertsWindowDays

//go:embed lang/*/*.json
var langFS embed.FS

// emergencyFS is the embedded copy of the EmergencyRules (one payload per rule in the row shape, plus `variants`),
// used when a rule's message_contents row is missing or disabled and for copy variants a row does not carry.
//
//go:embed emergency/*.json
var emergencyFS embed.FS

// emergencyCopy is locale → rule → payload of the embedded emergency copy.
var emergencyCopy = sync.OnceValue(func() map[string]map[string]map[string]any {
	out := map[string]map[string]map[string]any{}
	entries, err := emergencyFS.ReadDir("emergency")
	if err != nil {
		panic(err)
	}
	for _, en := range entries {
		b, err := emergencyFS.ReadFile("emergency/" + en.Name())
		if err != nil {
			panic(err)
		}
		var m map[string]map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			panic(fmt.Errorf("pregnancy alerts: emergency copy %s: %w", en.Name(), err))
		}
		out[strings.TrimSuffix(en.Name(), ".json")] = m
	}
	return out
})

// EmbeddedRule is the embedded payload of rule in locale (the fallback locale when the locale has none; nil when
// the rule has no embedded copy).
func EmbeddedRule(rule, locale string) map[string]any {
	c := emergencyCopy()
	if m, ok := c[locale][rule]; ok {
		return m
	}
	return c[lang.FallbackLocale][rule]
}

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

// T is the pregnancy_alerts line for key in locale.
func T(key, locale string) string { return translator().Trans("pregnancy_alerts."+key, nil, locale) }

// Engine evaluates, lists and acts on the v2 alerts of one user.
type Engine struct{ q store.Querier }

// New wires the engine.
func New(q store.Querier) *Engine { return &Engine{q: q} }

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// ErrNotActive: the user has no active, dated pregnancy.
var ErrNotActive = errors.New("pregnancy alerts: pregnancy not active")

// DatingOf is the user's active pregnancy dated on today (ErrNotActive without an active, dated pregnancy).
func DatingOf(ctx context.Context, q store.Querier, userID uint64, today civildate.Date) (v2.Dating, error) {
	p, err := q.GetProfileByUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return v2.Dating{}, ErrNotActive
	}
	if err != nil {
		return v2.Dating{}, fmt.Errorf("pregnancy alerts: profile: %w", err)
	}
	if !p.PregnancyMode {
		return v2.Dating{}, ErrNotActive
	}
	d, ok := v2.Resolve(&p, today)
	if !ok {
		return v2.Dating{}, ErrNotActive
	}
	return d, nil
}

// FactsWindowDays is the look-back of LoadFacts (the longest detector window, 7 days, and the list window).
const FactsWindowDays = 7

// LoadFacts loads the detectors' input of userID at now: her dating and her logs of the last FactsWindowDays days
// (symptoms, extras, weekly logs, fetal movement, the running contraction session, the kick sessions). ok is false
// without an active, dated pregnancy. q may be the caller's transaction, so the facts see its uncommitted writes.
func LoadFacts(ctx context.Context, q store.Querier, userID uint64, now time.Time) (Facts, bool, error) {
	d, err := DatingOf(ctx, q, userID, civildate.InTehran(now))
	if errors.Is(err, ErrNotActive) {
		return Facts{}, false, nil
	}
	if err != nil {
		return Facts{}, false, err
	}
	f, err := (&Engine{q: q}).facts(ctx, userID, d, FactsWindowDays, now)
	return f, err == nil, err
}

// facts loads the user's logs of the window (at least ListWindowDays).
func (e *Engine) facts(ctx context.Context, userID uint64, d v2.Dating, window int, now time.Time) (Facts, error) {
	today := d.Today
	from := today.AddDays(-(max(window, ListWindowDays) - 1))
	f := Facts{Today: today, Week: d.CurrentWeek(), WeekDay: d.Days, Source: d.Source, Days: map[civildate.Date]Day{},
		Now: now, TotalDays: d.TotalDays}

	syms, err := e.q.ListV2SymptomLogsRange(ctx, store.ListV2SymptomLogsRangeParams{UserID: userID, DateFrom: from, DateTo: today})
	if err != nil {
		return Facts{}, fmt.Errorf("pregnancy alerts: symptoms: %w", err)
	}
	for i := range syms {
		f.Days[syms[i].LogDate] = SymptomDay(&syms[i], nil)
	}
	extras, err := e.q.ListDailyExtrasRange(ctx, store.ListDailyExtrasRangeParams{UserID: userID, DateFrom: from, DateTo: today})
	if err != nil {
		return Facts{}, fmt.Errorf("pregnancy alerts: extras: %w", err)
	}
	for i := range extras {
		day := f.Days[extras[i].LogDate]
		if day == nil {
			day = Day{}
		}
		for k, v := range SymptomDay(nil, &extras[i]) {
			day[k] = v
		}
		f.Days[extras[i].LogDate] = day
	}
	if f.Weekly, err = e.q.ListV2WeeklyLogsSince(ctx, store.ListV2WeeklyLogsSinceParams{UserID: userID, DateFrom: from}); err != nil {
		return Facts{}, fmt.Errorf("pregnancy alerts: weekly: %w", err)
	}
	wl, err := e.q.GetWeeklyLog(ctx, store.GetWeeklyLogParams{UserID: userID, PregnancyWeek: int32(f.Week)}) //nolint:gosec // 1..42
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return Facts{}, fmt.Errorf("pregnancy alerts: week log: %w", err)
	default:
		f.HasWeight = wl.Weight.Valid
	}
	if f.Fetal, err = e.q.ListFetalMovements(ctx, store.ListFetalMovementsParams{
		UserID: userID, HasFrom: 1, FromValue: from.String(), HasTo: 1, ToValue: today.String(),
	}); err != nil {
		return Facts{}, fmt.Errorf("pregnancy alerts: fetal: %w", err)
	}
	cs, err := e.q.ListActiveSessionContractions(ctx, store.ListActiveSessionContractionsParams{UserID: userID})
	if err != nil {
		return Facts{}, fmt.Errorf("pregnancy alerts: contractions: %w", err)
	}
	for _, c := range cs {
		f.ContractionSession = c.SessionID
		lc := labor.Contraction{Start: c.StartedAt}
		if c.EndedAt.Valid {
			lc.End = c.EndedAt.Time
		}
		f.Contractions = append(f.Contractions, lc)
	}
	if f.Kicks, err = e.kicks(ctx, userID, today); err != nil {
		return Facts{}, err
	}
	return f, nil
}

// kicks are the ended kick sessions started in the last KickLookbackDays (plus one day for a session that expired
// after midnight) and the running session, whenever it started.
func (e *Engine) kicks(ctx context.Context, userID uint64, today civildate.Date) ([]Kick, error) {
	ended, err := e.q.ListKickSessionsBetween(ctx, store.ListKickSessionsBetweenParams{
		UserID: userID, DateFrom: today.AddDays(-KickLookbackDays).TehranMidnight(), DateTo: today.AddDays(1).TehranMidnight(),
	})
	if err != nil {
		return nil, fmt.Errorf("pregnancy alerts: kicks: %w", err)
	}
	active, err := e.q.GetActiveKickSession(ctx, userID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return nil, fmt.Errorf("pregnancy alerts: active kicks: %w", err)
	default:
		ended = append(ended, active)
	}
	out := make([]Kick, 0, len(ended))
	for _, k := range ended {
		kk := Kick{ID: k.ID, Start: k.StartedAt, Count: int(k.Kicks)}
		if k.EndedAt.Valid && !k.ActiveLock.Valid {
			kk.End = k.EndedAt.Time
		}
		if k.TenthKickAt.Valid {
			kk.TargetAt = k.TenthKickAt.Time
		}
		out = append(out, kk)
	}
	return out, nil
}

// SymptomDay reads the v2 symptoms of a v1 symptom log and / or an extras row (has_X → severity,
// "mild" when unset), including the critical symptoms.
func SymptomDay(r *store.PregnancySymptomLog, ex *store.PregnancyDailyExtra) Day {
	out := Day{}
	add := func(name string, has sql.NullBool, sev sql.NullString) {
		if has.Valid && has.Bool {
			if sev.Valid && sev.String != "" {
				out[name] = sev.String
			} else {
				out[name] = "mild"
			}
		}
	}
	if r != nil {
		add("nausea", r.HasNausea, r.NauseaSeverity)
		add("vomiting", r.HasVomiting, r.VomitingSeverity)
		add("fatigue", r.HasFatigue, r.FatigueSeverity)
		add("headache", r.HasHeadache, r.HeadacheSeverity)
		add("back_pain", r.HasBackPain, r.BackPainSeverity)
		add("breast_pain", r.HasBreastPain, r.BreastPainSeverity)
		add("dizziness", r.HasDizziness, r.DizzinessSeverity)
		add("lower_abdominal_pain", r.HasLowerAbdominalPain, r.LowerAbdominalPainSeverity)
		add("cramping", r.HasCramping, r.CrampingSeverity)
		add("pelvic_pressure", r.HasPelvicPressure, r.PelvicPressureSeverity)
		add("spotting", r.HasSpotting, r.SpottingSeverity)
		add("bleeding", r.HasBleeding, r.BleedingSeverity)
		add("fluid_leakage", r.HasFluidLeakage, r.FluidLeakageSeverity)
		add("severe_sudden_pain", r.HasSevereSuddenPain, r.SevereSuddenPainSeverity)
	}
	if ex != nil {
		if ex.HeartburnSeverity.Valid {
			out["heartburn"] = ex.HeartburnSeverity.String
		}
		if ex.ConstipationSeverity.Valid {
			out["constipation"] = ex.ConstipationSeverity.String
		}
	}
	return out
}

// ---- The message engine backend (T-M10-08, D-165) ----
//
// The alerts themselves are messages of the alerts inbox (SPEC §5.4.1): the detectors above are the fact producers
// (messages/facts/pregnancy_alerts.go), one seeded message rule per alert decides the level, and message_inbox keeps
// the one alert history per user. This package is a leaf (vitals and the safety scenarios import it), so the inbox
// side lives in pregnancyalerts/inboxalerts and installs itself here; the Engine keeps its API for its callers
// (pregnancy log saves, the v2 day log, the kick / contraction tools, vitals in pregnancy, /pregnancy/v2/alerts*).

// EvalOptions tune one evaluation.
type EvalOptions struct {
	// Daily also returns the calendar alerts (new week, weight, next visit) created now; a log save returns only the
	// alerts its data raised (the v2 TimeRules split). Every rule is evaluated either way.
	Daily bool
	// Readings are vitals readings handed over by the save that holds them (EvaluateReadings).
	Readings []Reading
	// V1 adds `trigger_symptoms` to each alert: the evidence in the v1 alert terms (EvaluateV1).
	V1 bool
}

// Backend is the inbox side: it evaluates the user's pregnancy rules, lists and acts on her alerts.
type Backend interface {
	// Evaluate evaluates the pregnancy rules over the user's data read through q (the caller's transaction, if any)
	// and returns the alerts this evaluation created or re-raised, in the v2 alert shape rendered in l.
	Evaluate(ctx context.Context, q store.Querier, userID uint64, now time.Time, l v2.Lang, opt EvalOptions) ([]*jsonx.OrderedMap, error)
	// List is GET /pregnancy/v2/alerts over the inbox (evaluate = run the daily evaluation first).
	List(ctx context.Context, q store.Querier, userID uint64, now time.Time, l v2.Lang, evaluate bool) (*jsonx.OrderedMap, error)
	// Act is POST /pregnancy/v2/alerts/{id}/actions/{action} over the inbox row id.
	Act(ctx context.Context, q store.Querier, userID, id uint64, act string, now time.Time, l v2.Lang) (*jsonx.OrderedMap, error)
}

type backendBox struct{ b Backend }

var installed atomic.Pointer[backendBox]

// InstallBackend sets the inbox backend (inboxalerts.Install; nil uninstalls).
func InstallBackend(b Backend) {
	if b == nil {
		installed.Store(nil)
		return
	}
	installed.Store(&backendBox{b: b})
}

// ErrNoBackend: the inbox backend is not installed — the alert paths fail loudly instead of silently dropping alerts.
var ErrNoBackend = errors.New("pregnancy alerts: message engine backend not installed (inboxalerts.Install)")

func backendOf() (Backend, error) {
	if b := installed.Load(); b != nil {
		return b.b, nil
	}
	return nil, ErrNoBackend
}

// Evaluate runs the rules after a log save and returns the alerts the save raised.
func (e *Engine) Evaluate(ctx context.Context, userID uint64, now time.Time, l v2.Lang) ([]*jsonx.OrderedMap, error) {
	return e.EvaluateOn(ctx, e.q, userID, now, l)
}

// EvaluateOn is Evaluate reading through q (the caller's transaction, e.g. the day-log save), so the rules see the
// caller's uncommitted writes.
func (e *Engine) EvaluateOn(ctx context.Context, q store.Querier, userID uint64, now time.Time, l v2.Lang) ([]*jsonx.OrderedMap, error) {
	b, err := backendOf()
	if err != nil {
		return nil, err
	}
	return b.Evaluate(ctx, q, userID, now, l, EvalOptions{})
}

// EvaluateV1 is EvaluateOn for the v1 log saves (POST /pregnancy/symptoms, /weekly, /fetal-movement): each alert also
// carries `trigger_symptoms`, its evidence in the v1 alert terms, for the v1 `alerts` array.
func (e *Engine) EvaluateV1(ctx context.Context, q store.Querier, userID uint64, now time.Time, l v2.Lang) ([]*jsonx.OrderedMap, error) {
	b, err := backendOf()
	if err != nil {
		return nil, err
	}
	return b.Evaluate(ctx, q, userID, now, l, EvalOptions{V1: true})
}

// EvaluateAll is the daily evaluation: it also returns the calendar alerts created now.
func (e *Engine) EvaluateAll(ctx context.Context, userID uint64, now time.Time, l v2.Lang) ([]*jsonx.OrderedMap, error) {
	b, err := backendOf()
	if err != nil {
		return nil, err
	}
	return b.Evaluate(ctx, e.q, userID, now, l, EvalOptions{Daily: true})
}

// TimeRules are the calendar alerts (legacy keys): returned by the daily evaluation only.
var TimeRules = []string{"weight_missing_week", "week_entered", "next_visit"}

// EvaluateReadings hands vitals readings of a pregnant user to the BP and GDM rules (T-M9-05) and returns the alerts
// of the readings rendered in l (raised now, or still open for the same reading). A user without an active pregnancy
// gets nothing.
func (e *Engine) EvaluateReadings(ctx context.Context, userID uint64, readings []Reading, now time.Time, l v2.Lang,
) ([]*jsonx.OrderedMap, error) {
	b, err := backendOf()
	if err != nil {
		return nil, err
	}
	return b.Evaluate(ctx, e.q, userID, now, l, EvalOptions{Readings: readings})
}

// List is GET /pregnancy/v2/alerts (HEAD passes evaluate=false so it never writes).
func (e *Engine) List(ctx context.Context, userID uint64, now time.Time, l v2.Lang, evaluate bool) (*jsonx.OrderedMap, error) {
	b, err := backendOf()
	if err != nil {
		return nil, err
	}
	return b.List(ctx, e.q, userID, now, l, evaluate)
}

// ErrNotFound: no alert with that id for the user.
var ErrNotFound = errors.New("pregnancy alerts: not found")

// Actions are the server-side alert actions.
var Actions = []string{"ack"}

// Act runs POST /pregnancy/v2/alerts/{id}/actions/{action}: ack acknowledges the alert.
func (e *Engine) Act(ctx context.Context, userID, id uint64, act string, now time.Time, l v2.Lang) (*jsonx.OrderedMap, error) {
	b, err := backendOf()
	if err != nil {
		return nil, err
	}
	return b.Act(ctx, e.q, userID, id, act, now, l)
}
