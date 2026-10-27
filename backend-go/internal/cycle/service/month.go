package service

// /cycle/month on the v6 engine (T-M9-33, D-121, DECISIONS.md E-04/E-05/E-13/E-22/E-25/E-28, I-01, I-11):
//
//	auth (1 query) → ETag (as /cycle/today, plus the month and view) → If-None-Match → 304
//	 → state.Provider.For(uid, rev, params of the local day)          L1 hit = no query
//	 → State.Range(first, last, LocalToday)                           one pass, each cycle laid out once
//	 → per day: the legacy calculation keys rebuilt from the status (calculationV6; calendar view = its calendar
//	   subset) + the v6 day fields; month level: month_summary, future_cycles (today's, k ≤ 6) and limits (today's)
//
// Everything a calendar needs comes from the engine: the cycle day never wraps (a late cycle keeps counting), the days
// from predicted_next through today are period_expected and tomorrow onward follows the rebased chain (E-04), late
// beyond max_late or past future_cycles[6] nothing is predicted (E-04, E-28), a completed cycle uses the retrospective
// layout (E-22), pregnancy shows the logged history only (E-25) and hormonal contraception has no fertile day (E-13).
//
// CYCLE_ENGINE=v1 never reaches this file; shadow serves v1 and classifies the month against v6 after the response
// (monthShadowSubmit).

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/cycle/legacy"
	"github.com/ritme/backend-go/internal/cycle/recommendation"
	"github.com/ritme/backend-go/internal/cycle/shadow"
	"github.com/ritme/backend-go/internal/cycle/view"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/etag"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

// MonthViewCalendar is the compact month view (T-M1-12).
const MonthViewCalendar = "calendar"

// calendarDropped are the calculation keys the calendar view leaves out (legacy.Calculation in calendar mode).
var calendarDropped = []string{"is_luteal_spotting", "cycle_score", "age_factor", "base_probability", "symptom_score",
	"uncertainty_range", "text_flags", "daily_tips", "source_profile_data", "source_daily_log_data"}

// PeriodKind is a month day's period_kind: how a menstrual day is known.
type PeriodKind string

const (
	// PeriodLogged is a day of a logged period: a closed period, or the logged start of an open one.
	PeriodLogged PeriodKind = "logged"
	// PeriodAssumed is an open period's assumed day (after its logged start, within the end caps) or a day of the
	// declared onboarding period (I-01: an assumed range computed by the engine, never a logged row).
	PeriodAssumed PeriodKind = "assumed"
	// PeriodPredicted is a predicted period day: a future cycle (E-03/E-05) or a stale roll-forward of the
	// onboarding anchor (§2.41).
	PeriodPredicted PeriodKind = "predicted"
)

// PeriodKindOf is the period_kind of a resolved day, "" (null) when the day is not menstrual.
func PeriodKindOf(s *engine.Status) PeriodKind {
	if s.MainPhase != engine.MainPhaseMenstrual {
		return ""
	}
	a := &s.Anchors
	switch {
	case s.ResolutionSource == engine.ResolutionUserLogged:
		return PeriodLogged
	case s.ResolutionSource == engine.ResolutionUserLoggedWithAssumedEnd:
		if s.Date == a.CurrentPeriodStart {
			return PeriodLogged
		}
		return PeriodAssumed
	case s.FutureK == 0 && a.CurrentPeriodStartSource == engine.StartOnboardingDeclared:
		return PeriodAssumed
	default:
		return PeriodPredicted
	}
}

// monthFutureCycle is one month-level future_cycles item (the §2.35 shape, E-05).
type monthFutureCycle struct {
	K               int     `json:"k"`
	PeriodStart     *string `json:"period_start"`
	PeriodEnd       *string `json:"period_end"`
	FertileStart    *string `json:"fertile_start"`
	FertileEnd      *string `json:"fertile_end"`
	Ovulation       *string `json:"ovulation"`
	OvulationSource string  `json:"ovulation_source"`
	Confidence      string  `json:"confidence"`
}

// monthLimits is the E-24 limits block of the local day.
type monthLimits struct {
	FertilitySuppressed bool    `json:"fertility_suppressed"`
	SuppressionReason   *string `json:"suppression_reason"`
	EngineSuspended     bool    `json:"engine_suspended"`
	PromptPregnancyMode bool    `json:"prompt_pregnancy_mode"`
}

func dayPtr(d engine.Day) *string {
	if !d.IsSet() {
		return nil
	}
	s := d.String()
	return &s
}

func optAny(o engine.Opt[int]) any {
	if v, ok := o.Get(); ok {
		return v
	}
	return nil
}

// MonthTag is the ETag of uid's v6 month payload: the day tag's parts with the month and view in the config part (one
// tag per URL, so a tag never revalidates another month's body).
func (e *Engine) MonthTag(uid, rev uint64, year, month int, viewName string, today civildate.Date, locale string) string {
	return etag.Engine{
		UserID: uid, Rev: rev, Today: today, Locale: locale, ParamsFP: engineparams.Fingerprint(today),
		ConfigFP: fmt.Sprintf("%s.m%04d%02d%s", tipsFingerprint(), year, month, viewName),
		Build:    e.build,
	}.Tag()
}

// MonthBody is the whole 200 body of a v6 /cycle/month for the local day today.
func (e *Engine) MonthBody(ctx context.Context, uid, rev uint64, year, month int, viewName string, today civildate.Date, locale string) ([]byte, error) {
	st, err := e.stateFor(ctx, uid, rev, today)
	if err != nil {
		return nil, err
	}
	return e.monthJSON(ctx, st, year, month, viewName, today, locale)
}

// monthJSON encodes the month of st: {"success":true,"data":{calculations, calculation_status, is_recalculating,
// month_summary, future_cycles, limits}}.
func (e *Engine) monthJSON(ctx context.Context, st *engine.State, year, month int, viewName string, today civildate.Date, locale string) ([]byte, error) {
	first, last := MonthRange(year, month)
	t := engine.DayOf(today)
	days := st.Range(engine.DayOf(first), engine.DayOf(last), t)
	if days == nil {
		return nil, fmt.Errorf("cycle: month range %s..%s", first, last)
	}
	calendar := viewName == MonthViewCalendar
	noFertility := view.NoFertilityCopy(st.Mode())
	calcs := make([]*jsonx.OrderedMap, len(days))
	var sum legacy.MonthSummary
	for i := range days {
		s := &days[i]
		m, err := e.calculationV6(ctx, st, s, locale)
		if err != nil {
			return nil, err
		}
		if calendar {
			for _, k := range calendarDropped {
				m.Delete(k)
			}
		} else if err := e.rawMonthTips(ctx, m, s, noFertility); err != nil {
			return nil, err
		}
		kind := PeriodKindOf(s)
		isPeriod := kind != ""
		m.Set("main_phase", s.MainPhase.String()).
			Set("fertility_level", s.FertilityLevel.String()).
			Set("is_predicted", s.IsPredicted()).
			Set("is_period", isPeriod)
		if isPeriod {
			m.Set("period_kind", string(kind))
		} else {
			m.Set("period_kind", nil)
		}
		m.Set("days_to_period", optAny(s.DaysToPeriod)).
			Set("days_late", optAny(s.DaysLate))
		calcs[i] = m

		if fertile, _ := m.Get("is_fertile_window"); fertile == true {
			sum.FertileDays++
		}
		if isPeriod {
			sum.PeriodDays++
		}
		if pms, _ := m.Get("is_pms_window"); pms == true {
			sum.PmsDays++
		}
	}

	future := st.FutureCycles(t)
	fc := make([]monthFutureCycle, len(future))
	for i := range future {
		f := &future[i]
		fc[i] = monthFutureCycle{K: f.K, PeriodStart: dayPtr(f.PeriodStart), PeriodEnd: dayPtr(f.PeriodEnd),
			FertileStart: dayPtr(f.FertileStart), FertileEnd: dayPtr(f.FertileEnd), Ovulation: dayPtr(f.Ovulation),
			OvulationSource: f.OvulationSource.String(), Confidence: f.Confidence.String()}
	}
	l := st.Suppression(t)
	limits := monthLimits{FertilitySuppressed: l.FertilitySuppressed, EngineSuspended: l.EngineSuspended,
		PromptPregnancyMode: l.PromptPregnancyMode}
	if r := l.SuppressionReason.String(); r != "" {
		limits.SuppressionReason = &r
	}

	body := httpx.Envelope(jsonx.Obj(
		"calculations", calcs,
		// The engine has no asynchronous recalculation (D-120).
		"calculation_status", string(enums.CalculationStatusCompleted),
		"is_recalculating", false,
		"month_summary", sum,
		"future_cycles", fc,
		"limits", limits,
	))
	out, err := jsonx.Marshal(body, jsonx.UnescapedSlashes|jsonx.UnescapedUnicode)
	if err != nil {
		return nil, fmt.Errorf("cycle: encode month: %w", err)
	}
	return out, nil
}

// rawMonthTips keeps the full month view's daily_tips in the legacy month shape: the raw bilingual tips (the month
// view never localized them), from the same I-11 content phase as calculationV6.
func (e *Engine) rawMonthTips(ctx context.Context, m *jsonx.OrderedMap, s *engine.Status, noFertility bool) error {
	phase, _ := view.ContentPhase(s.Subphase)
	if !s.CycleDay.OK() {
		m.Set("daily_tips", []recommendation.Tip{})
		return nil
	}
	tips, err := e.dailyTips(ctx, phase, s.Subphase, noFertility)
	if err != nil {
		return err
	}
	if tips == nil {
		tips = []recommendation.Tip{}
	}
	m.Set("daily_tips", tips)
	return nil
}

// monthV6 answers a v6 month: 304 on a matching If-None-Match (nothing else runs), else the v6 body.
func (h *Handlers) monthV6(c fiber.Ctx, uid uint64, year, month int, viewName string) error {
	rev := revision(c)
	today := h.today(c)
	locale := i18n.ResolveLocale(c, "")
	// As dayV6: the tag names the recommendations snapshot, so it is loaded first; one request-scoped repository.
	ctx := withTips(c.Context(), recommendation.New(h.eng.tips))
	if _, err := h.eng.tipsRepo(ctx).HasContent(ctx); err != nil {
		return err
	}
	if etag.NotModified(c, h.eng.MonthTag(uid, rev, year, month, viewName, today, locale)) {
		return nil
	}
	body, err := h.eng.MonthBody(ctx, uid, rev, year, month, viewName, today, locale)
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return c.Status(fiber.StatusOK).Send(body)
}

// ---- shadow classification of month responses ----

var (
	monthShadowDiff = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "engine_shadow_month_diff_total",
		Help: "CYCLE_ENGINE=shadow: days of served v1 /cycle/month responses by difference class against v6.",
	}, []string{"class"})
	monthShadowJobs = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "engine_shadow_month_jobs_total",
		Help: "CYCLE_ENGINE=shadow month comparisons by result (compared, dropped = busy, failed).",
	}, []string{"result"})
)

func init() {
	metrics.Registry().MustRegister(monthShadowDiff, monthShadowJobs)
}

// monthShadow bounds the month comparisons like the day pool (shadow.PoolSize at once, a busy pool drops the job):
// shadow mode never queues work and never adds latency to a request. One per process (one engine per process).
var monthShadow = struct {
	slots chan struct{}
	wg    sync.WaitGroup
}{slots: make(chan struct{}, shadow.PoolSize)}

// WaitMonthShadow blocks until the running month comparisons finish (tests, shutdown).
func WaitMonthShadow() { monthShadow.wg.Wait() }

// v1MonthDay is the part of a served legacy month day the comparison reads.
type v1MonthDay struct {
	Date     string  `json:"calculation_date"`
	CycleDay *int    `json:"cycle_day"`
	Phase    *string `json:"phase"`
	OvDay    *int    `json:"estimated_ovulation_day"`
	Fertile  bool    `json:"is_fertile_window"`
	PMS      bool    `json:"is_pms_window"`
}

// ClassifyMonthDay names the difference classes of one served v1 month day against the v6 status of that day (the
// shadow.Class vocabulary; a day can have several):
//
//   - o_plus_one: v1 flags the day fertile, v6 does not, and it is the day after ovulation (v6 days_to_ovulation
//     −1, or the legacy cycle day = its ovulation day + 1);
//   - lateness: v6 counts the day late and the legacy cycle day differs (it wraps into a new cycle);
//   - suppression: v6 suppresses fertility (E-24) and the fertile flags differ;
//   - phase: the legacy phase differs from the I-11 content phase of the v6 subphase;
//   - other: the cycle day, the PMS flag or the period flag differ otherwise;
//   - new_fields_only: everything shared agrees.
func ClassifyMonthDay(v1 *v1MonthDay, s *engine.Status) []shadow.Class {
	var out []shadow.Class
	fertile6 := s.CycleDay.OK() && s.MainPhase == engine.MainPhaseFertile && !s.Limits.FertilitySuppressed
	cd6, hasCD6 := s.CycleDay.Get()
	cdDiffers := (v1.CycleDay == nil) != !hasCD6 || (v1.CycleDay != nil && *v1.CycleDay != cd6)
	oPlusOne := v1.Fertile && !fertile6 &&
		(s.DaysToOvulation.Or(0) == -1 && s.DaysToOvulation.OK() ||
			v1.CycleDay != nil && v1.OvDay != nil && *v1.CycleDay == *v1.OvDay+1)
	if oPlusOne {
		out = append(out, shadow.ClassOPlusOne)
	}
	lateness := s.DaysLate.Or(0) > 0 && cdDiffers
	if lateness {
		out = append(out, shadow.ClassLateness)
	}
	suppression := s.Limits.FertilitySuppressed && v1.Fertile != fertile6
	if suppression {
		out = append(out, shadow.ClassSuppression)
	}
	phase6, ok6 := view.ContentPhase(s.Subphase)
	v1Phase := ""
	if v1.Phase != nil {
		v1Phase = *v1.Phase
	}
	if !ok6 {
		phase6 = ""
	}
	if !oPlusOne && v1Phase != string(phase6) {
		out = append(out, shadow.ClassPhase)
	}
	period1 := v1Phase == string(enums.CyclePhaseMenstruation)
	pms6 := s.CycleDay.OK() && s.Subphase == engine.SubphasePMSPossible
	if (cdDiffers && !lateness) || v1.PMS != pms6 || period1 != (s.MainPhase == engine.MainPhaseMenstrual) {
		out = append(out, shadow.ClassOther)
	}
	if len(out) == 0 {
		out = append(out, shadow.ClassNewFieldsOnly)
	}
	return out
}

// monthShadowSubmit hands the served v1 month calculations to a month comparison after the response. The job context
// carries the request's zone and clock only (the fiber context is recycled); a busy pool drops the job.
func (h *Handlers) monthShadowSubmit(c fiber.Ctx, uid uint64, year, month int, today civildate.Date, v1Calcs []byte) {
	if h.eng.Shadow() == nil {
		return
	}
	select {
	case monthShadow.slots <- struct{}{}:
	default:
		monthShadowJobs.WithLabelValues("dropped").Inc()
		return
	}
	loc := civildate.Zone(c.Context())
	ctx := clock.WithClock(context.Background(), clock.At(clock.FromContext(c, h.clock).Now()))
	ctx = civildate.WithZone(ctx, func(context.Context) *time.Location { return loc })
	rev := revision(c)
	v1 := append([]byte(nil), v1Calcs...)
	e := h.eng
	monthShadow.wg.Add(1)
	go func() {
		defer func() {
			<-monthShadow.slots
			monthShadow.wg.Done()
		}()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := e.classifyMonth(ctx, uid, rev, year, month, today, v1); err != nil {
			monthShadowJobs.WithLabelValues("failed").Inc()
			slog.WarnContext(ctx, "engine shadow: month compare failed", slog.Any("error", err))
			return
		}
		monthShadowJobs.WithLabelValues("compared").Inc()
	}()
}

// classifyMonth computes the v6 month and counts each day's classes.
func (e *Engine) classifyMonth(ctx context.Context, uid, rev uint64, year, month int, today civildate.Date, v1Calcs []byte) error {
	var v1 []v1MonthDay
	if err := json.Unmarshal(v1Calcs, &v1); err != nil {
		return fmt.Errorf("shadow: decode v1 month: %w", err)
	}
	st, err := e.stateFor(ctx, uid, rev, today)
	if err != nil {
		return err
	}
	first, last := MonthRange(year, month)
	days := st.Range(engine.DayOf(first), engine.DayOf(last), engine.DayOf(today))
	if len(days) != len(v1) {
		return fmt.Errorf("shadow: month has %d v1 days, %d v6 days", len(v1), len(days))
	}
	for i := range days {
		for _, c := range ClassifyMonthDay(&v1[i], &days[i]) {
			monthShadowDiff.WithLabelValues(string(c)).Inc()
		}
	}
	return nil
}
