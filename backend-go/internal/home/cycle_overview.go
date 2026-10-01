package home

// GET /home/cycle-overview (B-N1-06, Go only): the numbers the Night & Bloom cycle home shows beside
// `cycle_view` — the logging streak, the predicted period range and PMS window, and the cycle-length
// summary (median, spread, regularity). No new engine math: the dates come from the §35 resolver and
// the lengths from the cycle metrics, exactly what `/cycle/today` and `/fertility/insights` read.

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/cycle/resolver"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

const (
	// streakWindowDays is how far back the logging streak looks; a longer run reports this many days.
	streakWindowDays = 90
	// pmsWindowDays mirrors the resolver's `pms_possible` subphase: the 1–3 days before the predicted period.
	pmsWindowDays = 3
)

// CycleOverview is the response body of GET /home/cycle-overview.
type CycleOverview struct {
	Date civildate.Date
	// StreakDays counts consecutive logged days ending today, or yesterday while today has no log yet.
	StreakDays  int
	LoggedToday bool
	// NextPeriodStart/End is the predicted period (start + effective period length − 1); zero = unknown.
	NextPeriodStart, NextPeriodEnd civildate.Date
	// PmsStart/End is the resolver's PMS-possible run before NextPeriodStart; zero = unknown.
	PmsStart, PmsEnd civildate.Date
	CycleLength      int
	CycleLengthSrc   enums.EffectiveSource
	BasedOnCycles    int
	// Spread is ±days around the median: ceil(range/2) of the last ≤3 valid lengths; nil with < 2 cycles.
	Spread     *int
	Regularity enums.RegularityStatus
}

// BuildCycleOverview computes the overview from the engine inputs. logDates are the days with a
// daily_health_logs row (any order, duplicates allowed).
func BuildCycleOverview(histories []model.History, profile *model.Profile, logDates []civildate.Date, today civildate.Date) CycleOverview {
	m := metrics.Calculate(histories, profile)
	st := resolver.Resolve(histories, profile, today, today, m)
	out := CycleOverview{
		Date:           today,
		CycleLength:    m.EffectiveCycleLength,
		CycleLengthSrc: m.CycleLengthSource,
		BasedOnCycles:  m.ValidCyclesCount,
		Regularity:     m.RegularityStatus,
	}
	if m.CycleVariabilityRange != nil {
		s := (*m.CycleVariabilityRange + 1) / 2
		out.Spread = &s
	}
	if next := st.PredictedNextPeriodStart; !next.IsZero() {
		out.NextPeriodStart = next
		out.NextPeriodEnd = next.AddDays(max(1, m.EffectivePeriodDuration) - 1)
		out.PmsStart, out.PmsEnd = next.AddDays(-pmsWindowDays), next.AddDays(-1)
	}
	out.StreakDays, out.LoggedToday = loggingStreak(logDates, today)
	return out
}

// loggingStreak counts consecutive logged days ending today (or yesterday when today is empty).
func loggingStreak(dates []civildate.Date, today civildate.Date) (int, bool) {
	logged := make(map[civildate.Date]bool, len(dates))
	for _, d := range dates {
		logged[d] = true
	}
	day := today
	if !logged[today] {
		day = today.AddDays(-1)
	}
	n := 0
	for n < streakWindowDays && logged[day] {
		n++
		day = day.AddDays(-1)
	}
	return n, logged[today]
}

func rangeOrNil(from, to civildate.Date) any {
	if from.IsZero() {
		return nil
	}
	return jsonx.Obj("start", from.String(), "end", to.String())
}

// JSON is the `data` object.
func (o CycleOverview) JSON() *jsonx.OrderedMap {
	var spread any
	if o.Spread != nil {
		spread = *o.Spread
	}
	var basedOn any
	if o.BasedOnCycles > 0 {
		basedOn = o.BasedOnCycles
	}
	return jsonx.Obj(
		"date", o.Date.String(),
		"logging", jsonx.Obj("streak_days", o.StreakDays, "logged_today", o.LoggedToday),
		"next_period", rangeOrNil(o.NextPeriodStart, o.NextPeriodEnd),
		"pms_window", rangeOrNil(o.PmsStart, o.PmsEnd),
		"cycle_length", jsonx.Obj(
			"median", o.CycleLength,
			"source", string(o.CycleLengthSrc),
			"based_on_cycles", basedOn,
			"spread_days", spread,
			"regularity", string(o.Regularity),
		),
	)
}

// CycleOverview is GET /home/cycle-overview: the current user's own rows only (no id parameter), plus the Ritme
// Plus trial banner (`plus_trial_offer`, null when no trial offer runs; B-N2-06).
func (h *Handlers) CycleOverview(c fiber.Ctx) error {
	user, err := currentUser(c)
	if err != nil {
		return err
	}
	today := civildate.InTehran(h.now(c))
	sn, err := h.cycle.Load(c.Context(), user.ID, today.AddDays(-streakWindowDays), today, today)
	if err != nil {
		return err
	}
	dates := make([]civildate.Date, 0, len(sn.LogRows))
	for _, r := range sn.LogRows {
		dates = append(dates, r.LogDate)
	}
	var banner any
	if h.deps.Plus != nil {
		if banner, err = h.deps.Plus.TrialBannerJSON(c.Context(), user.ID, h.now(c), i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode()); err != nil {
			return err
		}
	}
	return httpx.OK(c, BuildCycleOverview(sn.Histories, sn.EngineProfile(), dates, today).JSON().Set("plus_trial_offer", banner))
}
