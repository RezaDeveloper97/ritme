package home

// /home on the v6 engine (T-M9-34, D-122, DECISIONS.md I-08, I-11): under CYCLE_ENGINE=shadow|v6 the cycle facts of
// the page come from the shared engine state (state.UserStates, T-M9-36) instead of the legacy engine:
//
//	week_calendar     State.Range over the Saturday-to-Friday week (the cycle day never wraps; period_kind,
//	                  main_phase, fertility_level, days_late per day — the same values /cycle/date shows)
//	next_period       days_to_period / days_late from the status (a late cycle is late, not a new cycle)
//	cycle_prediction  today_ui.prediction_card: fertile O−5..O and ovulation with its source (null under
//	                  suppression), the period range, the PMS window; no section without a card
//	challenge         the v6 cycle day (no wrap); a day beyond every day-targeted challenge gets none
//	cycle_summary     health check from clinical_flags / cycle_regularity (FIGO, I-08): never 21–45 as "normal"
//	my_cycles         digest of the actual periods only (the rows the engine reads)
//	HasCycleData      the engine has an anchor (a real period or the declared onboarding start)
//
// Every other section reads the same status through a calculation rebuilt from it (v6Calculation: content phase,
// cycle day, flags, the admin daily tips of the content phase), so tasks, affirmations, articles, the smart tip and the
// message system see one engine. CYCLE_ENGINE=v1 never reaches this file.

import (
	"slices"
	"strconv"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/legacy"
	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/cycle/recommendation"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/view"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

// engineView is the request's v6 engine view: the prepared state, the user-local today and the memoised statuses.
type engineView struct {
	st    *engine.State
	today engine.Day
	days  map[civildate.Date]engine.Status
}

func newEngineView(st *engine.State, today civildate.Date) *engineView {
	return &engineView{st: st, today: engine.DayOf(today), days: map[civildate.Date]engine.Status{}}
}

// status is the resolved status of d for the local today (memoised).
func (v *engineView) status(d civildate.Date) engine.Status {
	if s, ok := v.days[d]; ok {
		return s
	}
	s := v.st.Resolve(engine.DayOf(d), v.today)
	v.days[d] = s
	return s
}

// hasAnchor reports whether the engine has an anchor: a real period or the declared onboarding start (I-01).
func (v *engineView) hasAnchor() bool {
	return len(v.st.Periods()) > 0 || v.st.Declared().LastPeriodStart.OK()
}

// fertile is calculationV6's is_fertile_window: a cycle day in the fertile main phase, fertility not suppressed.
func fertile(s *engine.Status) bool {
	return s.CycleDay.OK() && s.MainPhase == engine.MainPhaseFertile && !s.Limits.FertilitySuppressed
}

// isOvulationDay reports whether s's day is the (unsuppressed) estimated ovulation.
func isOvulationDay(s *engine.Status) bool {
	ov := s.Anchors.EstimatedOvulationDate
	return ov.IsSet() && ov == s.Date && !s.Limits.FertilitySuppressed
}

func optInt(o engine.Opt[int]) any {
	if v, ok := o.Get(); ok {
		return v
	}
	return nil
}

func dayOrNil(d engine.Day) any {
	if !d.IsSet() {
		return nil
	}
	return d.String()
}

// lengthSource maps an engine length source onto the legacy effective-source enum (view.LegacyFromV6's mapping).
func lengthSource(s engine.LengthSource) enums.EffectiveSource {
	switch s {
	case engine.LengthHistory:
		return enums.EffectiveSourceRecentValidCycles
	case engine.LengthDeclared:
		return enums.EffectiveSourceProfile
	default:
		return enums.EffectiveSourceDefault
	}
}

// fertilityLevel is the v6 fertility level the user may see: unknown for a mode without fertility copy (teen,
// menopause), under suppression and on period_expected (DailyCardV6's rule).
func (v *engineView) fertilityLevel(s *engine.Status) enums.FertilityLevel {
	if view.NoFertilityCopy(v.st.Mode()) || s.Limits.FertilitySuppressed || s.MainPhase == engine.MainPhasePeriodExpected {
		return enums.FertilityLevelUnknown
	}
	return enums.FertilityLevel(s.FertilityLevel.String())
}

// ---------------------------------------------------------------------------
// the calculation the other sections and the message system read

// v6Calculation is the legacy calculation of date rebuilt from the v6 status (calculationV6's fields): Complete only
// with a cycle day and a content phase (I-11); scores, text flags and the uncertainty range have no v6 value and stay
// zero. withTips adds the admin daily tips of the content phase (the recommendations section).
func (hc *Context) v6Calculation(date civildate.Date, withTips bool) (legacy.Calculation, error) {
	s := hc.v6.status(date)
	c := legacy.Calculation{Date: date}
	cd, ok := s.CycleDay.Get()
	phase, hasPhase := view.ContentPhase(s.Subphase)
	if !ok || !hasPhase {
		return c, nil
	}
	c.Complete = true
	c.Day, c.Phase, c.CurrentSubphase = cd, phase, enums.CycleSubphase(s.Subphase.String())
	if ov, start := s.Anchors.EstimatedOvulationDate, s.Anchors.CurrentPeriodStart; ov.IsSet() && start.IsSet() && !s.Limits.FertilitySuppressed {
		c.OvulationDay = int(ov-start) + 1
	}
	c.CycleLength = s.Metrics.EffectiveCycleLength
	c.IsFertileWindow = fertile(&s)
	c.IsPmsWindow = s.Subphase == engine.SubphasePMSPossible
	c.IsPeriodTomorrow = s.DaysToPeriod.OK() && s.DaysToPeriod.Or(0) == 1
	switch s.Metrics.CycleRegularity {
	case engine.RegularityRegular:
		c.Variability = enums.CycleVariabilityRegular
	case engine.RegularityIrregular:
		c.Variability = enums.CycleVariabilityIrregular
	}
	if withTips {
		tips, err := hc.v6Tips(phase, s.Subphase)
		if err != nil {
			return legacy.Calculation{}, err
		}
		c.DailyTips = tips
	}
	return c, nil
}

// v6Tips are the admin recommendations of the content phase without symptom triggers, fertility tips dropped for a
// mode without fertility copy (calculationV6's dailyTips over the snapshot's request-scoped repository).
func (hc *Context) v6Tips(phase enums.CyclePhase, sub engine.Subphase) ([]recommendation.Tip, error) {
	repo := hc.Snapshot.Tips
	if repo == nil {
		return nil, nil
	}
	has, err := repo.HasContent(hc.Ctx)
	if err != nil || !has {
		return nil, err
	}
	tips, err := repo.ForDay(hc.Ctx, phase, enums.CycleSubphase(sub.String()), nil)
	if err != nil || !view.NoFertilityCopy(hc.v6.st.Mode()) {
		return tips, err
	}
	return slices.DeleteFunc(tips, func(t recommendation.Tip) bool { return t.Type == "fertility" }), nil
}

// v6DailyCard is the /cycle/today daily card of the context date (view.DailyCardV6 over the same status and today_ui,
// T-M9-32): the smart tip falls back to its copy, since v6 has no text flags.
func (hc *Context) v6DailyCard() *view.DailyCard {
	s := hc.v6.status(hc.Date)
	ui := hc.v6.st.TodayUI(s)
	return view.DailyCardV6(view.CardV6Input{
		Status: &s, UI: &ui, Today: hc.v6.today, Locale: hc.Locale,
		NoFertilityCopy: view.NoFertilityCopy(hc.v6.st.Mode()),
	})
}

// ---------------------------------------------------------------------------
// week_calendar

// weekDayV6 is one week_calendar day of a resolved status: the legacy keys (phase = I-11 content phase, is_period =
// any menstrual day with a period_kind, is_fertile / is_ovulation off under suppression) plus the v6 day fields.
func (v *engineView) weekDayV6(s *engine.Status, contextDate civildate.Date, locale string) *jsonx.OrderedMap {
	day := s.Date.Civil()
	phase := ""
	if p, ok := view.ContentPhase(s.Subphase); ok && s.CycleDay.OK() {
		phase = string(p)
	}
	var phaseValue, kindValue any
	if phase != "" {
		phaseValue = phase
	}
	kind := cycleservice.PeriodKindOf(s)
	if kind != "" {
		kindValue = string(kind)
	}
	return jsonx.Obj(
		"date", day.String(),
		"weekday_iso", day.ISOWeekday(),
		"is_today", day == contextDate,
		"cycle_day", optInt(s.CycleDay),
		"phase", phaseValue,
		"phase_label", phaseLabel(phase, locale),
		"color", phaseColor(phase),
		"is_period", kind != "",
		"is_fertile", fertile(s),
		"is_ovulation", isOvulationDay(s),
		"main_phase", s.MainPhase.String(),
		"fertility_level", string(v.fertilityLevel(s)),
		"period_kind", kindValue,
		"is_predicted", s.IsPredicted(),
		"days_late", optInt(s.DaysLate),
	)
}

func (s weekCalendarSection) buildV6(hc *Context) (*Rendered, error) {
	weekStart := hc.Date.StartOfWeek()
	days := make([]*jsonx.OrderedMap, 0, 7)
	if hc.HasCycleData() {
		for _, st := range hc.v6.st.Range(engine.DayOf(weekStart), engine.DayOf(weekStart.AddDays(6)), hc.v6.today) {
			days = append(days, hc.v6.weekDayV6(&st, hc.Date, hc.Locale))
		}
	} else {
		for i := range 7 {
			day := weekStart.AddDays(i)
			days = append(days, jsonx.Obj(
				"date", day.String(), "weekday_iso", day.ISOWeekday(), "is_today", day == hc.Date,
				"cycle_day", nil, "phase", nil, "phase_label", nil, "color", nil,
				"is_period", false, "is_fertile", false, "is_ovulation", false,
				"main_phase", engine.MainPhaseUnknown.String(), "fertility_level", string(enums.FertilityLevelUnknown),
				"period_kind", nil, "is_predicted", true, "days_late", nil,
			))
		}
	}
	return &Rendered{
		Key: s.Key(), Type: "week_calendar", Order: s.Order(),
		Data: jsonx.Obj("week_start", weekStart.String(), "days", days),
	}, nil
}

// ---------------------------------------------------------------------------
// next_period

func (s nextPeriodSection) buildV6(hc *Context) (*Rendered, error) {
	st := hc.v6.status(hc.Date)
	cycleDay, ok := st.CycleDay.Get()
	if !ok || !hc.HasCycleData() {
		return nil, nil
	}
	cycleLength := st.Metrics.EffectiveCycleLength
	daysUntil := max(0, st.DaysToPeriod.Or(0))
	late := st.DaysLate.Or(0)
	next := st.Anchors.PredictedNextPeriodStart
	progress := 0
	if cycleLength > 0 {
		progress = int(phpround.Round(float64(cycleDay)/float64(cycleLength)*100, 0))
	}
	level := hc.v6.fertilityLevel(&st)

	var title, subtitle *string
	switch {
	case late > 0:
		n := strconv.Itoa(late)
		title = str(hc.T("پریودت "+hc.Num(late)+" روز دیر کرده", "Your period is "+n+" days late"))
		if next.IsSet() {
			title2 := hc.T("تاریخ پیش‌بینی‌شده "+next.String()+" بود", "It was expected on "+next.String()) //nolint:staticcheck // ST1018: Persian text (ZWNJ)
			subtitle = &title2
		}
	default:
		n := strconv.Itoa(daysUntil)
		title = str(hc.T(n+" روز تا پریود بعدی", n+" days to next period"))
		if next.IsSet() {
			subtitle = str(hc.T("پریود بعدی شما حدوداً "+next.String(), "Your next period is around "+next.String()))
		}
	}
	return &Rendered{
		Key: s.Key(), Type: "next_period", Order: s.Order(),
		Title: title,
		Data: jsonx.Obj(
			"days_until_period", daysUntil,
			"next_period_date", dayOrNil(next),
			"cycle_day", cycleDay,
			"cycle_length", cycleLength,
			"progress_percent", min(100, max(0, progress)),
			"is_in_period", st.MainPhase == engine.MainPhaseMenstrual,
			"is_period_tomorrow", st.DaysToPeriod.OK() && daysUntil == 1,
			"is_fertile_window", fertile(&st),
			"uncertainty_range", nil,
			"pregnancy_probability", jsonx.Obj("percent", nil, "level", string(level), "label", level.Label(hc.Locale)),
			"days_to_period", optInt(st.DaysToPeriod),
			"days_late", optInt(st.DaysLate),
			"main_phase", st.MainPhase.String(),
		),
		Subtitle: subtitle,
		Action:   action("log_period_start", hc.T("شروع پریود", "Log period start")),
	}, nil
}

// ---------------------------------------------------------------------------
// cycle_prediction

func (s cyclePredictionSection) buildV6(hc *Context) (*Rendered, error) {
	st := hc.v6.status(hc.Date)
	ui := hc.v6.st.TodayUI(st)
	card, ok := ui.PredictionCard.Get()
	if !ok || !hc.HasCycleData() {
		return nil, nil
	}
	date := engine.DayOf(hc.Date)
	until := func(d engine.Day) int { return max(0, int(d-date)) }

	fertileCard := jsonx.Obj("key", "fertile_window", "label", hc.T("پنجره باروری", "Fertile window"),
		"start_date", nil, "end_date", nil, "days_until", nil, "is_active", false)
	if f, ok := card.Fertile.Get(); ok {
		fertileCard.Set("start_date", f.Start.String()).Set("end_date", f.End.String()).
			Set("days_until", until(f.Start)).Set("is_active", date >= f.Start && date <= f.End)
	}
	ovulationCard := jsonx.Obj("key", "ovulation", "label", hc.T("تخمک‌گذاری", "Ovulation"), //nolint:staticcheck // ST1018: Persian text (ZWNJ)
		"date", nil, "days_until", nil, "source", nil)
	if ov, ok := card.Ovulation.Get(); ok {
		ovulationCard.Set("date", ov.Date.String()).Set("days_until", until(ov.Date)).Set("source", ov.Source.String())
	}
	cards := []*jsonx.OrderedMap{
		fertileCard,
		ovulationCard,
		jsonx.Obj(
			"key", "next_period",
			"label", hc.T("پریود بعدی", "Next period"),
			"date", card.Period.Start.String(),
			"days_until", until(card.Period.Start),
			"end_date", card.Period.End.String(),
			"days_late", optInt(st.DaysLate),
		),
	}
	return &Rendered{
		Key: s.Key(), Type: "cycle_prediction", Order: s.Order(),
		Data: jsonx.Obj(
			"cards", cards,
			"uncertainty_range", nil,
			"pms_window", jsonx.Obj("start", card.PMS.Start.String(), "end", card.PMS.End.String()),
			"cycle_length", card.CycleLength,
			"cycle_regularity", card.CycleRegularity.String(),
		),
	}, nil
}

// ---------------------------------------------------------------------------
// my_cycles

// actualHistories are the history rows the engine reads as periods (inputs.mapPeriods, I-01, I-02): no estimated row,
// no onboarding seed.
func actualHistories(hs []model.History) []model.History {
	out := make([]model.History, 0, len(hs))
	for _, h := range hs {
		if h.IsEstimated || h.Source == string(enums.DataSourceUserProfileConfirmed) {
			continue
		}
		out = append(out, h)
	}
	return out
}

// v6Digest is the history digest over the actual periods with the engine's validity bounds (§2.8).
func (hc *Context) v6Digest() *Digest {
	if hc.digest == nil {
		p := hc.v6.st.Params()
		hc.digest = newDigest(actualHistories(hc.Snapshot.Histories), digestBounds{p.CycleMin, p.CycleMax, p.PeriodMin, p.PeriodMax})
	}
	return hc.digest
}

func (s myCyclesSection) buildV6(hc *Context) (*Rendered, error) {
	st := hc.v6.status(hc.Date)
	cycleDay, ok := st.CycleDay.Get()
	start := st.Anchors.CurrentPeriodStart
	if !ok || !start.IsSet() || !hc.HasCycleData() {
		return nil, nil
	}
	digest := hc.v6Digest()
	_, src := hc.v6.st.CycleLength()
	current := jsonx.Obj(
		"id", nil,
		"cycle_day", cycleDay,
		"started_at", start.String(),
		"period_end_date", nil,
		"period_length", nil,
		"is_ongoing", false,
		"cycle_length", st.Metrics.EffectiveCycleLength,
		"cycle_length_source", string(lengthSource(src)),
	)
	if rec, found := digest.StartingOn(start.Civil()); found {
		var end any
		if !rec.End.IsZero() {
			end = rec.End.String()
		}
		current.Set("id", rec.ID).
			Set("period_end_date", end).
			Set("period_length", intOrNil(rec.PeriodLength)).
			Set("is_ongoing", rec.End.IsZero())
	}
	previous := []*jsonx.OrderedMap{}
	for _, c := range digest.Previous(myCyclesPreviousLimit) {
		previous = append(previous, c.JSON())
	}
	return &Rendered{
		Key: s.Key(), Type: "my_cycles", Order: s.Order(),
		Title: str(hc.T("سیکل‌های من", "My cycles")), //nolint:staticcheck // ST1018: Persian text (ZWNJ)
		Data: jsonx.Obj(
			"current", current,
			"previous_count", len(digest.Previous(-1)),
			"previous", previous,
			"averages", jsonx.Obj(
				"cycle_length", intOrNil(digest.AverageCycleLength()),
				"period_length", intOrNil(digest.AveragePeriodLength()),
				"based_on_cycles", len(digest.ValidCycleLengths()),
			),
		),
		Action: action("add_previous", hc.T("ثبت سیکل‌های قبلی", "Add previous cycles")), //nolint:staticcheck // ST1018: Persian text (ZWNJ)
	}, nil
}

// ---------------------------------------------------------------------------
// cycle_summary: the health check on clinical_flags / cycle_regularity (I-08)

// healthFlag is one health-check verdict: the FIGO flag ("" = no basis) and the item status it maps to
// (normal | outside_range | unknown). Only a FIGO "normal" / "regular" is ever "normal" (§2.40: never 21–45).
type healthFlag struct{ flag, status string }

func frequencyFlag(f engine.ClinicalFlags) healthFlag {
	v, ok := f.Frequency.Get()
	switch {
	case !ok:
		return healthFlag{"", "unknown"}
	case v == engine.FrequencyNormal:
		return healthFlag{v.String(), "normal"}
	default:
		return healthFlag{v.String(), "outside_range"}
	}
}

func durationFlag(f engine.ClinicalFlags) healthFlag {
	v, ok := f.Duration.Get()
	switch {
	case !ok:
		return healthFlag{"", "unknown"}
	case v == engine.DurationNormal:
		return healthFlag{v.String(), "normal"}
	default:
		return healthFlag{v.String(), "outside_range"}
	}
}

func regularityFlag(r engine.CycleRegularity) healthFlag {
	switch r {
	case engine.RegularityRegular:
		return healthFlag{r.String(), "normal"}
	case engine.RegularityIrregular:
		return healthFlag{r.String(), "outside_range"}
	default:
		return healthFlag{r.String(), "unknown"}
	}
}

// label is the user-facing verdict of the flag (copy reviewed by the main session, T-M9-34).
func (f healthFlag) label(hc *Context) string {
	switch f.flag {
	case "normal":
		return hc.T("در محدودهٔ معمول", "Within the usual range")
	case "frequent":
		return hc.T("کوتاه‌تر از محدودهٔ معمول", "Shorter than the usual range") //nolint:staticcheck // ST1018: Persian text (ZWNJ)
	case "infrequent":
		return hc.T("طولانی‌تر از محدودهٔ معمول", "Longer than the usual range") //nolint:staticcheck // ST1018: Persian text (ZWNJ)
	case "prolonged":
		return hc.T("طولانی‌تر از معمول", "Longer than usual") //nolint:staticcheck // ST1018: Persian text (ZWNJ)
	case "regular":
		return hc.T("منظم", "Regular")
	case "irregular":
		return hc.T("نامنظم", "Irregular")
	}
	return hc.T("هنوز مشخص نیست", "Not known yet")
}

// healthItem is a v6 health-check item: the v1 item keys with no normal_range, plus the FIGO flag.
type healthItem struct {
	key, label               string
	value, valueMin, valueMx *int
	text, unit, unitLabel    *string
	hint                     *string
	verdict                  healthFlag
}

func (it healthItem) json(hc *Context) *jsonx.OrderedMap {
	var flag any
	if it.verdict.flag != "" {
		flag = it.verdict.flag
	}
	return jsonx.Obj(
		"key", it.key,
		"label", it.label,
		"value", intOrNil(it.value),
		"value_min", intOrNil(it.valueMin),
		"value_max", intOrNil(it.valueMx),
		"text", strOrNil(it.text),
		"unit", strOrNil(it.unit),
		"unit_label", strOrNil(it.unitLabel),
		"status", it.verdict.status,
		"status_label", it.verdict.label(hc),
		"hint", strOrNil(it.hint),
		"normal_range", nil,
		"flag", flag,
	)
}

func daysItem(hc *Context, key, label string, value *int, verdict healthFlag, hint, emptyHint *string) healthItem {
	h := hint
	if value == nil {
		h = emptyHint
	}
	return healthItem{key: key, label: label, value: value, unit: str("days"), unitLabel: str(hc.T("روز", "days")), hint: h, verdict: verdict}
}

func (s cycleSummarySection) buildV6(hc *Context) (*Rendered, error) {
	st := hc.v6.status(hc.Date)
	if !st.CycleDay.OK() || !hc.HasCycleData() {
		return nil, nil
	}
	digest := hc.v6Digest()
	freq, dur := frequencyFlag(st.ClinicalFlags), durationFlag(st.ClinicalFlags)
	reg := regularityFlag(st.Metrics.CycleRegularity)
	lastCycle, lastPeriod := digest.LastCycleLength(), digest.LastPeriodLength()
	validCycles := len(digest.ValidCycleLengths())

	var spread *int
	if v, ok := st.Metrics.CycleVariability.Get(); ok && st.Metrics.CycleRegularity != engine.RegularityInsufficientData {
		spread = &v
	}
	variability := healthItem{
		key: "cycle_variability", label: hc.T("نوسان طول سیکل", "Cycle length variation"),
		value: spread, text: str(reg.label(hc)), verdict: reg,
	}
	if spread != nil {
		variability.unit, variability.unitLabel = str("days"), str(hc.T("روز", "days"))
		variability.hint = str(hc.T("اختلاف کوتاه‌ترین و بلندترین سیکل اخیر: "+hc.Num(*spread)+" روز", //nolint:staticcheck // ST1018: Persian text (ZWNJ)
			"Recent cycles differ by "+strconv.Itoa(*spread)+" days"))
	} else {
		variability.hint = str(hc.T("برای سنجش نظم، چند چرخهٔ کامل دیگر لازم است.", //nolint:staticcheck // ST1018: Persian text (ZWNJ)
			"A few more complete cycles are needed to judge regularity."))
	}

	items := []healthItem{
		daysItem(hc, "last_cycle_length", hc.T("طول سیکل قبلی", "Last cycle length"), lastCycle, freq, nil,
			str(hc.T("با ثبت پریود بعدی، طول این سیکل مشخص می‌شود.", "Log your next period to learn this cycle’s length."))), //nolint:staticcheck // ST1018: Persian text (ZWNJ)
		daysItem(hc, "last_period_duration", hc.T("مدت پریود قبلی", "Last period duration"), lastPeriod, dur, nil,
			str(hc.T("پایان پریود را ثبت کنید تا مدت آن محاسبه شود.", "Record when your period ends to see its duration."))),
		variability,
	}
	if validCycles >= minCyclesForAggregate {
		items = append(items, daysItem(hc, "average_cycle_length", hc.T("میانگین طول سیکل", "Average cycle length"),
			digest.AverageCycleLength(), freq,
			str(hc.T("بر پایه "+hc.Num(validCycles)+" سیکل ثبت‌شده", "Based on "+strconv.Itoa(validCycles)+" recorded cycles")), nil)) //nolint:staticcheck // ST1018: Persian text (ZWNJ)
		items = append(items, healthItem{
			key: "cycle_length_range", label: hc.T("کوتاه‌ترین تا بلندترین سیکل", "Shortest to longest cycle"), //nolint:staticcheck // ST1018: Persian text (ZWNJ)
			valueMin: digest.ShortestCycle(), valueMx: digest.LongestCycle(), unit: str("days"), unitLabel: str(hc.T("روز", "days")),
			verdict: reg,
		})
	}
	list := make([]*jsonx.OrderedMap, len(items))
	for i, it := range items {
		list[i] = it.json(hc)
	}
	var frequency, duration, absent any
	if v, ok := st.ClinicalFlags.Frequency.Get(); ok {
		frequency = v.String()
	}
	if v, ok := st.ClinicalFlags.Duration.Get(); ok {
		duration = v.String()
	}
	if v, ok := st.ClinicalFlags.Absent.Get(); ok {
		absent = v
	}
	return &Rendered{
		Key: s.Key(), Type: "cycle_summary", Order: s.Order(),
		Title: str(hc.T("خلاصه سیکل", "Cycle summary")),
		Data: jsonx.Obj(
			"items", list,
			"based_on_cycles", validCycles,
			"has_history", lastCycle != nil || lastPeriod != nil,
			"normal_ranges", nil,
			"clinical_flags", jsonx.Obj("frequency", frequency, "duration", duration, "absent", absent),
			"cycle_regularity", st.Metrics.CycleRegularity.String(),
		),
		Action: action("view_more", hc.T("مشاهده بیشتر", "View more")),
	}, nil
}
