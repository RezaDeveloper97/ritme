package analysis

import (
	"math"
	"slices"

	"github.com/ritme/backend-go/internal/babylog"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/postpartum"
	"github.com/ritme/backend-go/internal/postpartum/guide"
)

// The postpartum analysis hub (SPEC v6 §4.1 An_Hub_Post, T-M10-27, D-176): GET /api/v1/analysis/postpartum. A pure
// function of a PostpartumInput, loaded by PostpartumHandlers from what exists — the EPDS checks (totals and bands
// only, never the answers: D-54), the recovery log (taxonomy v2 slots + the red-flag rows, T-M9-06c), the baby log,
// the child's WHO placement (with the T-M10-22 crossing hint) and the day log's weights.
//
// Descriptive only, never a diagnosis. An urgent signal is never turned into a finding here: an urgent EPDS check
// went to the safety screen when it was taken (postpartum.ScoreOn), and a bleeding red flag of today is only marked
// (`alert_today`) so the client points to the existing urgent path (GET /postpartum alerts).
//
// Cards: the EPDS card is never Plus (SPEC §4.3: «هرگز روی … EPDS»); bleeding, breastfeeding and the child's growth
// are free (safety-adjacent or the child module's own free data); mother and baby sleep and the weight trend are the
// mode's Plus cards (§4.3 «کارت‌های پلاس هر حالت»; «روند وزن» is Plus in pregnancy and postpartum).

// Postpartum hub minimums. The EPDS ones are P.EPDSMinScores / P.EPDSTrendMinScores (§4.1); the others are this
// hub's own (the §4.1 table has no row for them), counted as logged days in the card's window.
const (
	PostpartumBleedingWindowDays = 21 // the lochia strip: the last 3 weeks, never before the birth
	PostpartumBleedingMinDays    = 4  // logged days for a direction (first third vs last third)
	PostpartumWeekDays           = 7  // the feeds and sleep window, today included
	PostpartumFeedsMinDays       = 3  // days with feeds in the window
	PostpartumSleepMinDays       = 3  // nights (mother) / days (baby) with sleep in the window
	PostpartumWeightMinLogs      = 2  // weights since the birth for a trend (as P.PregWeightMinLogs)
	// PostpartumPregnancyDays estimates the pregnancy start (birth − 280 days) when the birth was not recorded from a
	// dated pregnancy profile.
	PostpartumPregnancyDays = 280
)

// Directions.
const (
	TrendDecreasing = "decreasing"
	TrendSteady     = "steady"
	TrendIncreasing = "increasing"
	TrendUp         = "up"
	TrendDown       = "down"
)

// lochiaLevel orders bleeding.lochia_amount (none 0 … heavy 4).
var lochiaLevel = map[string]int{"none": 0, "spotting": 1, "light": 2, "medium": 3, "heavy": 4}

// bleedingAlerts are the recovery alerts (postpartum.Recovery.AlertsOn) that are about bleeding.
var bleedingAlerts = []string{guide.AlertPadPerHour, guide.AlertHeavyBleeding, guide.AlertLargeClots}

// PostpartumChild is the hub's baby, already placed on the WHO standard by the children package.
type PostpartumChild struct {
	ID        uint64
	Name      string
	Latest    any   // children.PointJSON of the newest point (nil without one)
	Verdict   any   // children.VerdictJSON
	Crossings []any // the T-M10-22 crossing hint of each indicator whose latest value crossed (children.SeriesJSON)
	Placed    bool  // the latest point has at least one percentile
}

// PostpartumInput is everything the postpartum hub reads.
type PostpartumInput struct {
	Today, Birth civildate.Date
	// PregnancyStart is day 0 of the pregnancy that ended in the birth (dated profile, else Birth − 280 days).
	PregnancyStart civildate.Date
	DeepAnalysis   bool
	// Checks are the EPDS checks since the birth, newest first (totals only).
	Checks []postpartum.Check
	// Recovery is the recovery log by day, Birth … Today (taxonomy slots + the red-flag row).
	Recovery map[civildate.Date]postpartum.Recovery
	// Breastfeeding is the postpartum profile's flag (T-M9-28, I-04; nil = not answered).
	Breastfeeding *bool
	// Weights (kg) by day, from BaselineLookbackDays before PregnancyStart to Today.
	Weights map[civildate.Date]float64
	// ChildrenEnabled is the children module switch; Child the hub's baby (nil without one).
	ChildrenEnabled bool
	Child           *PostpartumChild
	// BabyDays are the child's baby-log days of the last PostpartumWeekDays, oldest first (nil without a child).
	BabyDays []babylog.Day
}

// card is one hub card: the Section plus how many more days (or scores) it needs; locked cards carry null.
func card(s Section, daysNeeded int) *jsonx.OrderedMap {
	out := s.JSON()
	if s.Locked {
		out.Set("days_needed", nil)
	} else {
		out.Set("days_needed", max(0, daysNeeded))
	}
	return out
}

// BuildPostpartum is the `data` of GET /analysis/postpartum.
func BuildPostpartum(in *PostpartumInput) *jsonx.OrderedMap {
	st := guide.StatusOn(in.Birth, in.Today)
	sleepNeeded := 0
	sleep := plusSection(in.DeepAnalysis, func() (bool, any) {
		ready, data, needed := in.sleepCard()
		sleepNeeded = needed
		return ready, data
	})
	weightNeeded := 0
	weight := plusSection(in.DeepAnalysis, func() (bool, any) {
		ready, data, needed := in.weightCard()
		weightNeeded = needed
		return ready, data
	})
	epds, epdsNeeded := in.epdsCard()
	bleeding, bleedingNeeded := in.bleedingCard()
	feeds, feedsNeeded := in.feedsCard()
	growth := in.growthCard()
	return jsonx.Obj(
		"birth_date", in.Birth.String(),
		"days_since_birth", st.DaysSinceBirth,
		"week", st.Week,
		"cards", jsonx.Obj(
			"epds", card(epds, epdsNeeded),
			"bleeding", card(bleeding, bleedingNeeded),
			"breastfeeding", card(feeds, feedsNeeded),
			"sleep", card(sleep, sleepNeeded),
			"growth", card(growth, 0),
			"weight", card(weight, weightNeeded),
		),
	)
}

// ── EPDS ─────────────────────────────────────────────────────────

// epdsCard: the latest score with its band (≥ P.EPDSMinScores) and the trend of the plotted questionnaire
// (≥ P.EPDSTrendMinScores of it). The full EPDS is plotted when there is one, else the weekly EPDS-3. days_needed is
// always 0: without a score a check-in is due today (postpartum.ScheduleOn with no check). Totals and bands only — the answers and item 10 are never
// read here (D-54); `urgent` is the stored flag the history already shows.
func (in *PostpartumInput) epdsCard() (Section, int) {
	cut := postpartum.CutoffsOn(in.Today)
	thresholds := jsonx.Obj("short_elevated", cut.ShortElevated, "full_possible", cut.Possible, "full_likely", cut.Likely)
	kind := postpartum.KindShort
	for _, c := range in.Checks {
		if c.Kind == postpartum.KindFull {
			kind = postpartum.KindFull
			break
		}
	}
	var points []postpartum.Check // oldest first, the plotted kind
	for i := len(in.Checks) - 1; i >= 0; i-- {
		if in.Checks[i].Kind == kind {
			points = append(points, in.Checks[i])
		}
	}
	ready := len(in.Checks) >= P.EPDSMinScores
	var latest, plotted any
	list := make([]*jsonx.OrderedMap, 0, len(points))
	for _, c := range points {
		list = append(list, in.epdsPointJSON(c))
	}
	if len(points) > 0 {
		latest = list[len(list)-1]
		plotted = kind
	}
	trendReady := len(points) >= P.EPDSTrendMinScores
	var direction any
	if trendReady {
		a, b := points[len(points)-2].Total, points[len(points)-1].Total
		switch {
		case b < a:
			direction = TrendDown
		case b > a:
			direction = TrendUp
		default:
			direction = TrendSteady
		}
	}
	return freeSection(ready, jsonx.Obj(
		"scores", len(in.Checks),
		"min_scores", P.EPDSMinScores,
		"trend_min_scores", P.EPDSTrendMinScores,
		"kind", plotted,
		"latest", latest,
		"trend", jsonx.Obj("ready", trendReady, "direction", direction),
		"points", list,
		"thresholds", thresholds,
	)), 0
}

func (in *PostpartumInput) epdsPointJSON(c postpartum.Check) *jsonx.OrderedMap {
	return jsonx.Obj(
		"kind", c.Kind,
		"taken_on", c.TakenOn.String(),
		"week", guide.StatusOn(in.Birth, c.TakenOn).Week,
		"total", c.Total,
		"max", c.Max(),
		"band", c.Band(),
		"urgent", c.Urgent,
	)
}

// ── bleeding ─────────────────────────────────────────────────────

// bleedingCard: bleeding.lochia_amount over the last PostpartumBleedingWindowDays since the birth (oldest first), the
// direction from the first third to the last third of the logged days (≥ PostpartumBleedingMinDays), large clots in
// the window, and whether today raised a bleeding red flag with the thresholds valid today (Recovery.AlertsOn).
func (in *PostpartumInput) bleedingCard() (Section, int) {
	n := max(1, min(PostpartumBleedingWindowDays, in.Birth.DiffDays(in.Today)+1))
	cells := make([]*jsonx.OrderedMap, 0, n)
	var levels []float64
	clots := false
	for i := n - 1; i >= 0; i-- {
		d := in.Today.AddDays(-i)
		r := in.Recovery[d]
		var level, code any
		if l, ok := lochiaLevel[r.LochiaAmount]; ok {
			level, code = l, r.LochiaAmount
			levels = append(levels, float64(l))
		}
		clots = clots || r.LargeClots
		cells = append(cells, jsonx.Obj("date", d.String(), "day", in.Birth.DiffDays(d), "level", level, "code", code))
	}
	ready := len(levels) >= PostpartumBleedingMinDays
	var trend any
	if ready {
		third := max(1, len(levels)/3)
		delta := meanOf(levels[len(levels)-third:]) - meanOf(levels[:third])
		switch {
		case delta <= -0.5:
			trend = TrendDecreasing
		case delta >= 0.5:
			trend = TrendIncreasing
		default:
			trend = TrendSteady
		}
	}
	alertToday := false
	for _, k := range in.Recovery[in.Today].AlertsOn(in.Today) {
		alertToday = alertToday || slices.Contains(bleedingAlerts, k)
	}
	return freeSection(ready, jsonx.Obj(
		"logged_days", len(levels),
		"min_days", PostpartumBleedingMinDays,
		"window_days", n,
		"weeks", max(1, int(math.Round(float64(n)/7))),
		"trend", trend,
		"large_clots", clots,
		"alert_today", alertToday,
		"cells", cells,
	)), PostpartumBleedingMinDays - len(levels)
}

// ── breastfeeding ────────────────────────────────────────────────

// weekDays are the PostpartumWeekDays days ending today, oldest first (never before the birth).
func (in *PostpartumInput) weekDays() []civildate.Date {
	n := max(1, min(PostpartumWeekDays, in.Birth.DiffDays(in.Today)+1))
	out := make([]civildate.Date, 0, n)
	for i := n - 1; i >= 0; i-- {
		out = append(out, in.Today.AddDays(-i))
	}
	return out
}

func (in *PostpartumInput) babyDay(d civildate.Date) *babylog.Day {
	for i := range in.BabyDays {
		if in.BabyDays[i].Date == d {
			return &in.BabyDays[i]
		}
	}
	return nil
}

// feedsCard: feeds per day of the last week — the baby log's sessions when there is a child (with the breast side
// split), else the mother's baby.feeds_count slot of the recovery log — and the breastfeeding flag (T-M9-28).
func (in *PostpartumInput) feedsCard() (Section, int) {
	source := "recovery"
	if in.Child != nil {
		source = "baby_log"
	}
	days := in.weekDays()
	list := make([]*jsonx.OrderedMap, 0, len(days))
	logged, total := 0, 0
	var left, right int64
	for _, d := range days {
		var count any
		if in.Child != nil {
			if b := in.babyDay(d); b != nil && b.Feeds.Count > 0 {
				count = b.Feeds.Count
				logged++
				total += b.Feeds.Count
				left += b.Feeds.LeftSeconds
				right += b.Feeds.RightSeconds
			}
		} else if r := in.Recovery[d]; r.FeedsCount != nil {
			count = *r.FeedsCount
			logged++
			total += *r.FeedsCount
		}
		list = append(list, jsonx.Obj("date", d.String(), "count", count))
	}
	var perDay, lp, rp any
	if logged > 0 {
		perDay = round(float64(total)/float64(logged), 1)
	}
	if left+right > 0 {
		l := int(math.Round(float64(left) * 100 / float64(left+right)))
		lp, rp = l, 100-l
	}
	var childID any
	if in.Child != nil {
		childID = in.Child.ID
	}
	return freeSection(logged >= PostpartumFeedsMinDays, jsonx.Obj(
		"breastfeeding", ptrOrNil(in.Breastfeeding),
		"source", source,
		"child_id", childID,
		"logged_days", logged,
		"min_days", PostpartumFeedsMinDays,
		"feeds_per_day", perDay,
		"left_percent", lp,
		"right_percent", rp,
		"days", list,
	)), PostpartumFeedsMinDays - logged
}

// ── sleep (Plus) ─────────────────────────────────────────────────

// sleepCard: the mother's mean sleep.hours over the logged nights of the last week and the baby's mean seconds asleep
// over the days with baby-log sleep; ready when either side reaches PostpartumSleepMinDays.
func (in *PostpartumInput) sleepCard() (bool, any, int) {
	var hours []float64
	var babySeconds []float64
	for _, d := range in.weekDays() {
		if r := in.Recovery[d]; r.SleepHours != nil {
			hours = append(hours, *r.SleepHours)
		}
		if b := in.babyDay(d); b != nil && (b.Sleep.Count > 0 || b.Sleep.Seconds > 0) {
			babySeconds = append(babySeconds, float64(b.Sleep.Seconds))
		}
	}
	motherReady := len(hours) >= PostpartumSleepMinDays
	babyReady := len(babySeconds) >= PostpartumSleepMinDays
	var motherAvg, babyAvg any
	if len(hours) > 0 {
		motherAvg = round(meanOf(hours), 1)
	}
	if len(babySeconds) > 0 {
		babyAvg = int64(math.Round(meanOf(babySeconds)))
	}
	var babyName any
	if in.Child != nil {
		babyName = in.Child.Name
	}
	needed := PostpartumSleepMinDays - max(len(hours), len(babySeconds))
	return motherReady || babyReady, jsonx.Obj(
		"min_days", PostpartumSleepMinDays,
		"mother", jsonx.Obj("ready", motherReady, "nights", len(hours), "avg_hours", motherAvg),
		"baby", jsonx.Obj("ready", babyReady, "name", babyName, "days", len(babySeconds), "avg_seconds", babyAvg),
	), needed
}

// ── growth ───────────────────────────────────────────────────────

// growthCard: the hub baby's latest WHO percentiles, the growth verdict and the T-M10-22 crossing hints. Ready when
// the latest point is placed. Off with the children module.
func (in *PostpartumInput) growthCard() Section {
	if !in.ChildrenEnabled || in.Child == nil {
		return freeSection(false, jsonx.Obj("available", in.ChildrenEnabled, "child", nil, "latest", nil,
			"verdict", nil, "crossings", []any{}))
	}
	c := in.Child
	crossings := c.Crossings
	if crossings == nil {
		crossings = []any{}
	}
	return freeSection(c.Placed, jsonx.Obj(
		"available", true,
		"child", jsonx.Obj("id", c.ID, "name", c.Name),
		"latest", c.Latest,
		"verdict", c.Verdict,
		"crossings", crossings,
	))
}

// ── weight (Plus) ────────────────────────────────────────────────

// weightCard: the weights since the birth (oldest first), the change since the first of them, and the latest against
// the pre-pregnancy weight — the pregnancy hub's baseline rule (PregnancyInput.baseline without the profile weight,
// which is today's weight, not the one before the pregnancy).
func (in *PostpartumInput) weightCard() (bool, any, int) {
	var since []civildate.Date
	for _, d := range sortedDates(in.Weights) {
		if !d.Before(in.Birth) && !d.After(in.Today) {
			since = append(since, d)
		}
	}
	points := make([]*jsonx.OrderedMap, 0, len(since))
	for _, d := range since {
		points = append(points, jsonx.Obj("date", d.String(), "kg", round(in.Weights[d], 1)))
	}
	pre := &PregnancyInput{Start: in.PregnancyStart, Weights: in.Weights}
	var baseline, latest, deltaPre, deltaFirst, fromWeek any
	base, source, baseDate, hasBase := pre.baseline()
	if hasBase {
		var bd any
		if baseDate != nil {
			bd = baseDate.String()
		}
		baseline = jsonx.Obj("kg", round(base, 1), "source", source, "date", bd)
	}
	if n := len(since); n > 0 {
		last := in.Weights[since[n-1]]
		latest = jsonx.Obj("date", since[n-1].String(), "kg", round(last, 1))
		fromWeek = guide.StatusOn(in.Birth, since[0]).Week
		if n >= PostpartumWeightMinLogs {
			deltaFirst = round(last-in.Weights[since[0]], 1)
		}
		if hasBase {
			deltaPre = round(last-base, 1)
		}
	}
	return len(since) >= PostpartumWeightMinLogs, jsonx.Obj(
		"logs", len(since),
		"min_logs", PostpartumWeightMinLogs,
		"unit", "kg",
		"pre_pregnancy", baseline,
		"latest", latest,
		"delta_vs_pre_pregnancy", deltaPre,
		"delta_since_first", deltaFirst,
		"from_week", fromWeek,
		"points", points,
	), PostpartumWeightMinLogs - len(since)
}

// meanOf is the mean of a non-empty list (0 when empty).
func meanOf(xs []float64) float64 {
	m, _ := mean(xs)
	return m
}
