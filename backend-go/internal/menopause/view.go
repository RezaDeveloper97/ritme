package menopause

import (
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/checkups"
	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

func ptr[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}

func strOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func publicOrNil(it *catalog.Item, loc catalog.Localizer) any {
	if it == nil {
		return nil
	}
	return loc.Public(*it)
}

// ProfileJSON is GET|PUT /menopause/profile and the profile block of /menopause/today.
func ProfileJSON(st Stage, tip *catalog.Item, loc catalog.Localizer) *jsonx.OrderedMap {
	return jsonx.Obj(
		"stage", strOrNil(st.Stage),
		"stored_stage", strOrNil(st.Answers.Stage),
		"last_period", ptr(st.LastPeriod),
		"surgical", ptr(st.Surgical),
		"hrt", ptr(st.HRT),
		"months_without_period", ptr(st.MonthsWithoutPeriod),
		"suggested_stage", strOrNil(st.Suggested),
		"needs_stage", st.NeedsStage(),
		"post_menopausal", st.PostMenopausal(),
		"tip", publicOrNil(tip, loc),
	)
}

// FlashJSON is one hot flash.
func FlashJSON(f Flash, now time.Time) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", f.ID,
		"started_at", jsonx.ISO8601(f.StartedAt),
		"running", f.Running(),
		"duration_s", ptr(f.Duration),
		"elapsed_s", f.Elapsed(now),
		"severity", strOrNil(f.Severity),
		"night", f.Night,
		"sweat", f.Sweat,
		"triggers", jsonx.List(f.Triggers),
	)
}

func runningJSON(f *Flash, now time.Time) any {
	if f == nil {
		return nil
	}
	return FlashJSON(*f, now)
}

// FlashDayJSON is GET /menopause/hot-flashes: the day's tiles, the running timer and the list.
func FlashDayJSON(d FlashDay, now time.Time) *jsonx.OrderedMap {
	items := make([]*jsonx.OrderedMap, len(d.Flashes))
	for i, f := range d.Flashes {
		items[i] = FlashJSON(f, now)
	}
	return jsonx.Obj(
		"date", d.Date,
		"count", d.Count(),
		"night_count", d.NightCount(),
		"avg_duration_s", ptr(d.AvgDuration()),
		"running", runningJSON(d.Running, now),
		"items", items,
	)
}

func jalali(d civildate.Date) (int, int) {
	jy, jm, _ := engine.ToJalali(d)
	return jy, jm
}

// BandJSON is one meno_score_bands band.
func BandJSON(b Band, loc catalog.Localizer) *jsonx.OrderedMap {
	return jsonx.Obj("code", b.Item.Code, "title", loc.Text(b.Item.Title), "min", b.Min, "max", b.Max)
}

func bandOrNil(sc Scale, total int, loc catalog.Localizer) any {
	if b := sc.Band(total); b != nil {
		return BandJSON(*b, loc)
	}
	return nil
}

// ScoreJSON is one questionnaire with its band, domains and delta.
func ScoreJSON(e ScoreEntry, sc Scale, loc catalog.Localizer) *jsonx.OrderedMap {
	jy, jm := jalali(e.Month)
	domains := make([]*jsonx.OrderedMap, 0, len(Domains))
	for _, d := range Domains {
		domains = append(domains, jsonx.Obj("code", d, "score", e.Subtotals[d], "max", sc.DomainMax(d)))
	}
	answers := jsonx.NewObject()
	for _, code := range sc.Codes() {
		if v, ok := e.Answers[code]; ok {
			answers.Set(code, v)
		}
	}
	rest := make([]string, 0, len(e.Answers))
	for code := range e.Answers { // answers to questions no longer active, by code
		if _, ok := answers.Get(code); !ok {
			rest = append(rest, code)
		}
	}
	slices.Sort(rest)
	for _, code := range rest {
		answers.Set(code, e.Answers[code])
	}
	var prev any
	if e.Previous != nil {
		prev = e.Previous.Month
	}
	return jsonx.Obj(
		"month", e.Month,
		"jalali_year", jy,
		"jalali_month", jm,
		"total", e.Total,
		"max", sc.MaxTotal(),
		"band", bandOrNil(sc, e.Total, loc),
		"domains", domains,
		"answers", answers,
		"delta", ptr(e.Delta()),
		"previous_month", prev,
	)
}

func scoreOrNil(e *ScoreEntry, sc Scale, loc catalog.Localizer) any {
	if e == nil {
		return nil
	}
	return ScoreJSON(*e, sc, loc)
}

// TrendJSON is the chart: one point per Jalali month, oldest first (total null = not filled).
func TrendJSON(points []TrendPoint, sc Scale) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, len(points))
	for i, p := range points {
		jy, jm := jalali(p.Month)
		var total, band any
		if p.Score != nil {
			total = p.Score.Total
			if b := sc.Band(p.Score.Total); b != nil {
				band = b.Item.Code
			}
		}
		out[i] = jsonx.Obj("month", p.Month, "jalali_year", jy, "jalali_month", jm, "total", total, "band", band)
	}
	return out
}

func hrtJSON(h *HRTEffect) any {
	if h == nil {
		return nil
	}
	var baseMonth, baseTotal, latestMonth, latestTotal any
	if h.Baseline != nil {
		baseMonth, baseTotal = h.Baseline.Month, h.Baseline.Total
	}
	if h.Latest != nil {
		latestMonth, latestTotal = h.Latest.Month, h.Latest.Total
	}
	return jsonx.Obj(
		"started_on", h.StartedOn,
		"baseline_month", baseMonth,
		"baseline_total", baseTotal,
		"latest_month", latestMonth,
		"latest_total", latestTotal,
		"change", ptr(h.Change()),
	)
}

// HistoryJSON is GET /menopause/scores.
func HistoryJSON(h History, sc Scale, loc catalog.Localizer) *jsonx.OrderedMap {
	bands := make([]*jsonx.OrderedMap, len(sc.Bands))
	for i, b := range sc.Bands {
		bands[i] = BandJSON(b, loc)
	}
	items := make([]*jsonx.OrderedMap, len(h.Entries))
	for i, e := range h.Entries {
		items[i] = ScoreJSON(e, sc, loc)
	}
	return jsonx.Obj(
		"months", h.Months,
		"max", sc.MaxTotal(),
		"bands", bands,
		"latest", scoreOrNil(h.Latest(), sc, loc),
		"trend", TrendJSON(h.Trend, sc),
		"items", items,
		"hrt", hrtJSON(h.HRT),
	)
}

// TreatmentJSON is one active item of the home's treatment card.
func TreatmentJSON(t TreatmentToday) *jsonx.OrderedMap {
	it := t.Item
	var dose, schedule, goal, unit, review any
	if it.Dose.Valid {
		dose = it.Dose.String
	}
	if it.Schedule.Valid {
		schedule = it.Schedule.String
	}
	if it.WeeklyGoal.Valid {
		goal = int(it.WeeklyGoal.Int16)
	}
	if it.GoalUnit.Valid {
		unit = it.GoalUnit.String
	}
	if it.ReviewOn.Valid {
		review = it.ReviewOn.Date
	}
	return jsonx.Obj(
		"id", it.ID,
		"kind", it.Kind,
		"name", it.Name,
		"dose", dose,
		"schedule", schedule,
		"taken_today", t.TakenToday,
		"days_taken", t.DaysTaken,
		"days", t.Days,
		"adherence_pct", ptr(t.AdherencePct()),
		"weekly_goal", goal,
		"goal_unit", unit,
		"amount", t.Amount,
		"review_on", review,
	)
}

// TodayJSON is GET /menopause/today.
func TodayJSON(t Today, now time.Time, loc catalog.Localizer, cl checkups.Lang) *jsonx.OrderedMap {
	var sleep any
	if s := t.Sleep; s != nil {
		sleep = jsonx.Obj("date", s.Date, "code", s.Code, "hours", s.Hours)
	}
	ups := make([]*jsonx.OrderedMap, len(t.Checkups))
	for i, it := range t.Checkups {
		ups[i] = t.Plan.ItemJSON(it, cl)
	}
	treatment := make([]*jsonx.OrderedMap, len(t.Treatment))
	for i, it := range t.Treatment {
		treatment[i] = TreatmentJSON(it)
	}
	return jsonx.Obj(
		"date", t.Date,
		"profile", ProfileJSON(t.Stage, t.StageTip, loc),
		"hot_flashes", jsonx.Obj(
			"count", t.Flashes.Count(),
			"night_count", t.Flashes.NightCount(),
			"avg_duration_s", ptr(t.Flashes.AvgDuration()),
			"running", runningJSON(t.Flashes.Running, now),
		),
		"night_sweats", jsonx.Obj("count", t.NightSweats, "level", strOrNil(t.NightLevel)),
		"sleep", sleep,
		"score", jsonx.Obj(
			"max", t.Scale.MaxTotal(),
			"latest", scoreOrNil(t.Score, t.Scale, loc),
			"trend", TrendJSON(t.Trend, t.Scale),
		),
		"checkups", ups,
		"treatment", treatment,
		"bleeding", jsonx.Obj(
			"alert", t.Bleeding.Alert,
			"last_on", ptr(t.Bleeding.LastOn),
			"window_days", BleedingAlertDays,
			"alert_item", publicOrNil(t.Bleeding.Item, loc),
		),
	)
}

// PatternJSON is one pattern card; text is the localized sentence of a found pattern (needs review).
func PatternJSON(p Pattern, text string) *jsonx.OrderedMap {
	groups := make([]*jsonx.OrderedMap, len(p.Groups))
	for i, g := range p.Groups {
		pct := 0
		if g.Days > 0 {
			pct = int(phpround.Round(g.Rate()*100, 0))
		}
		obj := jsonx.Obj("key", g.Key, "days", g.Days, "hits", g.Hits, "pct", pct)
		if i < len(p.MeanFlashes) {
			obj.Set("mean_flashes", phpround.Round(p.MeanFlashes[i], 1))
		}
		groups[i] = obj
	}
	var value, strength any
	if p.Status != analysis.CorrNotEnoughData {
		value = p.Value
	}
	if p.Strength != "" {
		strength = p.Strength
	}
	return jsonx.Obj(
		"key", p.Key,
		"trigger", strOrNil(p.Trigger),
		"status", p.Status,
		"found", p.Found(),
		"direction", strOrNil(p.Direction()),
		"statistic", p.Statistic,
		"value", value,
		"strength", strength,
		"n", p.N,
		"groups", groups,
		"ratio", ptr(p.Ratio),
		"text", strOrNil(text),
		"needs_review", true,
		"not_causal", true,
		"not_a_diagnosis", true,
	)
}

// PatternsJSON is GET /menopause/patterns; text(p) is a found pattern's sentence.
func PatternsJSON(ps Patterns, text func(Pattern) string, loc catalog.Localizer) *jsonx.OrderedMap {
	items := make([]*jsonx.OrderedMap, len(ps.Items))
	found := 0
	for i, p := range ps.Items {
		t := ""
		if p.Found() {
			found++
			t = text(p)
		}
		items[i] = PatternJSON(p, t)
	}
	return jsonx.Obj(
		"from", ps.From,
		"to", ps.To,
		"window_days", PatternWindowDays,
		"days_logged", ps.DaysLogged,
		"min_days", analysis.CorrMinDays,
		"min_group_days", analysis.CorrMinGroupDays,
		"min_outcome", analysis.CorrMinOutcome,
		"found", found,
		"not_a_diagnosis", true,
		"not_causal", true,
		"disclaimer", publicOrNil(ps.Disclaimer, loc),
		"items", items,
	)
}
