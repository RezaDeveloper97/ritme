package home

// T-M9-34 (D-122): the v6 home sections over engine personas built in memory (engine.Prepare), no database.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/cycle/model"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/view"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

var v6Today = civildate.MustParse("2026-09-23")

func day(s string) engine.Day { return engine.DayOf(civildate.MustParse(s)) }

// closedPeriods are n closed periods of length days every gap days, the last starting on last.
func closedPeriods(last string, n, gap, length int) []engine.Period {
	out := make([]engine.Period, n)
	l := day(last)
	for i := range n {
		s := l - engine.Day((n-1-i)*gap)
		out[i] = engine.Period{Start: s, End: s + engine.Day(length-1)}
	}
	return out
}

func v6Personas() map[string]engine.Input {
	regular := closedPeriods("2026-09-10", 6, 28, 5)
	open := append(closedPeriods("2026-08-24", 5, 28, 5), engine.Period{Start: day("2026-09-21"), End: engine.NoDay})
	return map[string]engine.Input{
		"regular":    {Periods: regular},
		"late_8":     {Periods: closedPeriods("2026-08-18", 6, 28, 5)},
		"pill":       {Periods: regular, Hormonal: true},
		"onboarding": {Declared: engine.Declared{LastPeriodStart: engine.Some(day("2026-09-10")), CycleLength: engine.Some(28), PeriodLength: engine.Some(5)}},
		"no_data":    {},
		"short_18":   {Periods: closedPeriods("2026-09-15", 6, 18, 4)},
		"open":       {Periods: open},
		"teen":       {Periods: regular, Mode: engine.ModeTeen},
	}
}

func v6Context(t *testing.T, in engine.Input, date civildate.Date) *Context {
	t.Helper()
	hc := newContext(context.Background(), &Deps{})
	hc.Date, hc.Today, hc.Locale, hc.DefaultLocale, hc.Mode = date, v6Today, "en", "fa", enums.MessageModeCycle
	hs := make([]model.History, len(in.Periods))
	for i, p := range in.Periods {
		hs[i] = model.History{ID: int64(i + 1), PeriodStart: p.Start.Civil(), IsConfirmed: true, Source: "user_logged"}
		if p.End.IsSet() {
			hs[i].PeriodEnd = p.End.Civil()
		}
	}
	hc.Snapshot = &cycleservice.Snapshot{Today: v6Today, Histories: hs}
	hc.v6 = newEngineView(engine.Prepare(in, engineparams.For(v6Today)), v6Today)
	return hc
}

// rendered is a section's JSON as a generic map (nil when the section is omitted).
func rendered(t *testing.T, s Section, hc *Context) map[string]any {
	t.Helper()
	r, err := s.Build(hc)
	require.NoError(t, err)
	if r == nil {
		return nil
	}
	raw, err := jsonx.Marshal(r.JSON(), jsonx.UnescapedUnicode)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	return m
}

func dataOf(m map[string]any) map[string]any { return m["data"].(map[string]any) }

func TestV6_HasCycleDataIsTheEngineAnchor(t *testing.T) {
	ps := v6Personas()
	assert.True(t, v6Context(t, ps["onboarding"], v6Today).HasCycleData(), "onboarding-only has an anchor")
	assert.False(t, v6Context(t, ps["no_data"], v6Today).HasCycleData())
	assert.True(t, v6Context(t, ps["regular"], v6Today).HasCycleData())

	hc := v6Context(t, ps["no_data"], v6Today)
	for _, s := range []Section{nextPeriodSection{}, cyclePredictionSection{}, myCyclesSection{}, cycleSummarySection{}} {
		r, err := s.Build(hc)
		require.NoError(t, err)
		assert.Nil(t, r, s.Key())
	}
	week := dataOf(rendered(t, weekCalendarSection{}, hc))["days"].([]any)
	require.Len(t, week, 7)
	assert.Nil(t, week[0].(map[string]any)["cycle_day"])
}

// Week calendar == cycle_view per day: each week day equals the single-day resolve /cycle/date answers (7 days × 8
// personas, the week of today and the week after).
func TestV6_WeekCalendarEqualsCycleViewPerDay(t *testing.T) {
	for name, in := range v6Personas() {
		for _, date := range []civildate.Date{v6Today, v6Today.AddDays(7)} {
			hc := v6Context(t, in, date)
			if !hc.HasCycleData() {
				continue
			}
			week := dataOf(rendered(t, weekCalendarSection{}, hc))["days"].([]any)
			require.Len(t, week, 7, name)
			for _, raw := range week {
				got := raw.(map[string]any)
				d := civildate.MustParse(got["date"].(string))
				st := hc.v6.st
				s := st.Resolve(engine.DayOf(d), engine.DayOf(v6Today))
				ui := st.TodayUI(s)
				legacy := view.LegacyFromV6(view.V6Input{State: st, Status: &s, UI: &ui, Today: engine.DayOf(v6Today), Locale: "en"})

				msg := name + " " + d.String()
				if cd, ok := s.CycleDay.Get(); ok {
					assert.EqualValues(t, cd, got["cycle_day"], msg)
				} else {
					assert.Nil(t, got["cycle_day"], msg)
				}
				if legacy.Phase != nil && s.CycleDay.OK() {
					assert.Equal(t, string(*legacy.Phase), got["phase"], msg)
				} else {
					assert.Nil(t, got["phase"], msg)
				}
				assert.Equal(t, s.MainPhase.String(), got["main_phase"], msg)
				assert.Equal(t, cycleservice.PeriodKindOf(&s) != "", got["is_period"], msg)
				assert.Equal(t, s.CycleDay.OK() && s.MainPhase == engine.MainPhaseFertile && !s.Limits.FertilitySuppressed, got["is_fertile"], msg)
				assert.Equal(t, s.IsPredicted(), got["is_predicted"], msg)
				if late, ok := s.DaysLate.Get(); ok {
					assert.EqualValues(t, late, got["days_late"], msg)
				} else {
					assert.Nil(t, got["days_late"], msg)
				}
			}
		}
	}
}

func TestV6_LateCycleIsLateNotRolled(t *testing.T) {
	hc := v6Context(t, v6Personas()["late_8"], v6Today)
	np := rendered(t, nextPeriodSection{}, hc)
	require.NotNil(t, np)
	data := dataOf(np)
	assert.EqualValues(t, 8, data["days_late"])
	assert.EqualValues(t, 0, data["days_until_period"])
	assert.Equal(t, "2026-09-15", data["next_period_date"])
	assert.EqualValues(t, 37, data["cycle_day"], "day 37 of the same cycle")
	assert.Contains(t, np["title"], "8 days late")

	// Through today the late cycle keeps counting (period_expected); from tomorrow the rebased chain predicts (E-04).
	week := dataOf(rendered(t, weekCalendarSection{}, hc))["days"].([]any)
	prev := 0
	for _, raw := range week {
		d := raw.(map[string]any)
		if civildate.MustParse(d["date"].(string)).After(v6Today) {
			break
		}
		cd := int(d["cycle_day"].(float64))
		if prev > 0 {
			assert.Equal(t, prev+1, cd, "the cycle day never wraps")
		}
		prev = cd
		assert.Equal(t, "period_expected", d["main_phase"], d["date"])
	}
	assert.Equal(t, 37, prev)
}

func TestV6_PillHasNoFertileDots(t *testing.T) {
	hc := v6Context(t, v6Personas()["pill"], v6Today)
	for _, date := range []civildate.Date{v6Today, v6Today.AddDays(-7), v6Today.AddDays(7), v6Today.AddDays(14)} {
		hc.Date = date
		for _, raw := range dataOf(rendered(t, weekCalendarSection{}, hc))["days"].([]any) {
			d := raw.(map[string]any)
			assert.Equal(t, false, d["is_fertile"], d["date"])
			assert.Equal(t, false, d["is_ovulation"], d["date"])
		}
	}
	hc.Date = v6Today
	if cp := rendered(t, cyclePredictionSection{}, hc); cp != nil {
		cards := dataOf(cp)["cards"].([]any)
		assert.Nil(t, cards[0].(map[string]any)["start_date"], "no fertile window under suppression")
		assert.Nil(t, cards[1].(map[string]any)["date"], "no ovulation under suppression")
	}
	np := dataOf(rendered(t, nextPeriodSection{}, hc))
	assert.Equal(t, false, np["is_fertile_window"])
	assert.Equal(t, "unknown", np["pregnancy_probability"].(map[string]any)["level"])
}

func TestV6_CyclePredictionFromTodayUI(t *testing.T) {
	hc := v6Context(t, v6Personas()["regular"], v6Today)
	cp := rendered(t, cyclePredictionSection{}, hc)
	require.NotNil(t, cp)
	card, ok := hc.v6.st.TodayUI(hc.v6.status(v6Today)).PredictionCard.Get()
	require.True(t, ok)
	cards := dataOf(cp)["cards"].([]any)
	f, _ := card.Fertile.Get()
	ov, _ := card.Ovulation.Get()
	assert.Equal(t, f.Start.String(), cards[0].(map[string]any)["start_date"])
	assert.Equal(t, ov.Date.String(), cards[0].(map[string]any)["end_date"], "fertile window ends on ovulation (O−5..O)")
	assert.Equal(t, ov.Date.String(), cards[1].(map[string]any)["date"])
	assert.Equal(t, "calendar", cards[1].(map[string]any)["source"])
	assert.Equal(t, card.Period.Start.String(), cards[2].(map[string]any)["date"])
	pms := dataOf(cp)["pms_window"].(map[string]any)
	assert.Equal(t, (card.Period.Start - 3).String(), pms["start"])
	assert.Equal(t, (card.Period.Start - 1).String(), pms["end"])
}

// 18-day real cycles are "frequent" in every health-check item that reads the cycle length, never "normal".
func TestV6_HealthCheckOnClinicalFlags(t *testing.T) {
	hc := v6Context(t, v6Personas()["short_18"], v6Today)
	cs := rendered(t, cycleSummarySection{}, hc)
	require.NotNil(t, cs)
	data := dataOf(cs)
	assert.Nil(t, data["normal_ranges"])
	assert.Equal(t, "frequent", data["clinical_flags"].(map[string]any)["frequency"])
	for _, raw := range data["items"].([]any) {
		it := raw.(map[string]any)
		assert.Nil(t, it["normal_range"], it["key"])
		if it["key"] == "last_cycle_length" || it["key"] == "average_cycle_length" {
			assert.NotEqual(t, "normal", it["status"], it["key"])
		}
		if it["key"] == "last_cycle_length" {
			assert.EqualValues(t, 18, it["value"])
			assert.Equal(t, "frequent", it["flag"])
			assert.Equal(t, "outside_range", it["status"])
		}
	}

	reg := dataOf(rendered(t, cycleSummarySection{}, v6Context(t, v6Personas()["regular"], v6Today)))
	for _, raw := range reg["items"].([]any) {
		it := raw.(map[string]any)
		if it["key"] == "last_cycle_length" {
			assert.Equal(t, "normal", it["flag"])
			assert.Equal(t, "normal", it["status"])
		}
	}
}

func TestV6_MyCyclesAndCycleDay(t *testing.T) {
	hc := v6Context(t, v6Personas()["regular"], v6Today)
	mc := dataOf(rendered(t, myCyclesSection{}, hc))
	cur := mc["current"].(map[string]any)
	assert.EqualValues(t, 14, cur["cycle_day"])
	assert.Equal(t, "2026-09-10", cur["started_at"])
	assert.Equal(t, "recent_valid_cycles", cur["cycle_length_source"])
	d, err := hc.CycleDay()
	require.NoError(t, err)
	assert.Equal(t, 14, *d)
}

func TestV6_ActualHistoriesDropEstimatesAndSeeds(t *testing.T) {
	hs := []model.History{
		{ID: 1, Source: "user_logged"},
		{ID: 2, Source: "user_logged", IsEstimated: true},
		{ID: 3, Source: string(enums.DataSourceUserProfileConfirmed)},
		{ID: 4, Source: string(enums.DataSourceBleedingLog)},
	}
	var ids []int64
	for _, h := range actualHistories(hs) {
		ids = append(ids, h.ID)
	}
	assert.Equal(t, []int64{1, 4}, ids)
}

func TestBeyondChallengeContent(t *testing.T) {
	assert.False(t, beyondChallengeContent(nil, 40))
	pool := []challengeRow{{}}
	assert.False(t, beyondChallengeContent(pool, 40), "untargeted only: no content range")
	pool[0].CycleDayFrom.Int16, pool[0].CycleDayFrom.Valid = 1, true
	pool[0].CycleDayTo.Int16, pool[0].CycleDayTo.Valid = 35, true
	assert.False(t, beyondChallengeContent(pool, 35))
	assert.True(t, beyondChallengeContent(pool, 36))
}

func TestV6_CycleOverview(t *testing.T) {
	st := engine.Prepare(v6Personas()["regular"], engineparams.For(v6Today))
	raw, err := jsonx.Marshal(BuildCycleOverviewV6(st, []civildate.Date{v6Today, v6Today.AddDays(-1)}, v6Today), 0)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	assert.Equal(t, map[string]any{"start": "2026-10-08", "end": "2026-10-12"}, m["next_period"])
	assert.Equal(t, map[string]any{"start": "2026-10-05", "end": "2026-10-07"}, m["pms_window"])
	cl := m["cycle_length"].(map[string]any)
	assert.EqualValues(t, 28, cl["median"])
	assert.Contains(t, []any{"regular", "irregular", "insufficient_data"}, cl["regularity"])
	assert.EqualValues(t, 2, m["logging"].(map[string]any)["streak_days"])

	none := BuildCycleOverviewV6(engine.Prepare(engine.Input{}, engineparams.For(v6Today)), nil, v6Today)
	v, _ := none.Get("next_period")
	assert.Nil(t, v)
	cl2, _ := none.Get("cycle_length")
	r, _ := cl2.(*jsonx.OrderedMap).Get("regularity")
	assert.Equal(t, "insufficient_data", r)
}
