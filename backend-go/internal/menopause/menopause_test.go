package menopause

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/menopause/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

var day = civildate.MustParse

func boolp(b bool) *bool { return &b }

func datep(s string) *civildate.Date {
	d := day(s)
	return &d
}

func TestWholeMonths(t *testing.T) {
	assert.Equal(t, 14, WholeMonths(day("2025-07-23"), day("2026-10-01")))
	assert.Equal(t, 11, WholeMonths(day("2025-10-02"), day("2026-10-01")))
	assert.Equal(t, 12, WholeMonths(day("2025-10-01"), day("2026-10-01")))
	assert.Equal(t, 0, WholeMonths(day("2026-10-02"), day("2026-10-01")))
	assert.Equal(t, 1, WholeMonths(day("2026-01-31"), day("2026-02-28"))) // clamped month end
}

func TestResolveStage(t *testing.T) {
	today := day("2026-10-01")
	cases := []struct {
		name      string
		in        Answers
		stage     string
		months    any
		suggested string
	}{
		{"nothing", Answers{}, "", nil, ""},
		{"unsure no period", Answers{Stage: StageUnsure}, "", nil, ""},
		{"unsure 14 months", Answers{Stage: StageUnsure, LastPeriod: datep("2025-08-01")}, StageMeno, 14, ""},
		{"unsure 5 months", Answers{Stage: StageUnsure, LastPeriod: datep("2026-05-01")}, StagePeri, 5, ""},
		{"no answer 12 months", Answers{LastPeriod: datep("2025-10-01")}, StageMeno, 12, ""},
		{"peri 13 months suggests meno", Answers{Stage: StagePeri, LastPeriod: datep("2025-09-01")}, StagePeri, 13, StageMeno},
		{"peri 3 months", Answers{Stage: StagePeri, LastPeriod: datep("2026-07-01")}, StagePeri, 3, ""},
		{"meno stays", Answers{Stage: StageMeno, LastPeriod: datep("2026-07-01")}, StageMeno, 3, ""},
		{"post stays", Answers{Stage: StagePost}, StagePost, nil, ""},
		{"surgical peri", Answers{Stage: StagePeri, Surgical: boolp(true), LastPeriod: datep("2026-07-01")}, StageMeno, 3, ""},
		{"surgical unsure", Answers{Stage: StageUnsure, Surgical: boolp(true)}, StageMeno, nil, ""},
		{"surgical post", Answers{Stage: StagePost, Surgical: boolp(true)}, StagePost, nil, ""},
		{"future last period ignored", Answers{Stage: StageUnsure, LastPeriod: datep("2026-11-01")}, "", nil, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st := ResolveStage(c.in, today)
			assert.Equal(t, c.stage, st.Stage)
			assert.Equal(t, c.suggested, st.Suggested)
			if c.months == nil {
				assert.Nil(t, st.MonthsWithoutPeriod)
			} else {
				require.NotNil(t, st.MonthsWithoutPeriod)
				assert.Equal(t, c.months, *st.MonthsWithoutPeriod)
			}
			assert.Equal(t, c.stage == StageMeno || c.stage == StagePost, st.PostMenopausal())
			assert.Equal(t, c.stage == "", st.NeedsStage())
		})
	}
}

func TestJalaliMonths(t *testing.T) {
	assert.Equal(t, day("2026-09-23"), MonthStart(day("2026-10-01"))) // 1 Mehr 1405
	assert.Equal(t, day("2026-09-23"), MonthStart(day("2026-09-23")))
	assert.Equal(t, day("2026-08-23"), MonthStart(day("2026-09-22"))) // Shahrivar
	assert.Equal(t, day("2026-08-23"), PrevMonth(day("2026-09-23")))
	assert.Equal(t, day("2026-02-20"), PrevMonth(day("2026-03-21"))) // Esfand ← Farvardin 1405
}

func meta(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func seededScale() Scale {
	q := func(code, domain string) catalog.Item {
		return catalog.Item{Code: code, Meta: meta(map[string]any{"domain": domain, "max": 4})}
	}
	items := []catalog.Item{
		q("hot_flashes", DomainSomatic), q("heart_discomfort", DomainSomatic), q("sleep_problems", DomainSomatic),
		q("joint_muscle", DomainSomatic), q("depressive_mood", DomainPsychological), q("irritability", DomainPsychological),
		q("anxiety", DomainPsychological), q("exhaustion", DomainPsychological), q("sexual_problems", DomainUrogenital),
		q("bladder_problems", DomainUrogenital), q("vaginal_dryness", DomainUrogenital),
	}
	b := func(code string, lo, hi int) catalog.Item {
		return catalog.Item{Code: code, Title: json.RawMessage(`{"en":"` + code + `"}`), Meta: meta(map[string]any{"min": lo, "max": hi})}
	}
	bands := []catalog.Item{b("none", 0, 4), b("mild", 5, 8), b("moderate", 9, 16), b("severe", 17, 44), {Code: "broken"}}
	return NewScale(items, bands)
}

func TestScale(t *testing.T) {
	sc := seededScale()
	assert.Equal(t, 44, sc.MaxTotal())
	assert.Equal(t, 16, sc.DomainMax(DomainSomatic))
	assert.Equal(t, 16, sc.DomainMax(DomainPsychological))
	assert.Equal(t, 12, sc.DomainMax(DomainUrogenital))
	assert.Len(t, sc.Bands, 4, "a band without min/max is skipped")

	// the board: 14 of 44 = 7 + 4 + 3, moderate
	answers := map[string]int{
		"hot_flashes": 3, "heart_discomfort": 1, "sleep_problems": 2, "joint_muscle": 1,
		"depressive_mood": 1, "irritability": 1, "anxiety": 1, "exhaustion": 1,
		"sexual_problems": 1, "bladder_problems": 0, "vaginal_dryness": 2,
	}
	s := sc.Compute(day("2026-09-23"), answers)
	assert.Equal(t, 14, s.Total)
	assert.Equal(t, map[string]int{DomainSomatic: 7, DomainPsychological: 4, DomainUrogenital: 3}, s.Subtotals)
	require.NotNil(t, sc.Band(14))
	assert.Equal(t, "moderate", sc.Band(14).Item.Code)
	assert.Equal(t, "none", sc.Band(0).Item.Code)
	assert.Equal(t, "severe", sc.Band(44).Item.Code)
	assert.Nil(t, sc.Band(45))

	e := ScoreEntry{Score: s, Previous: &Score{Total: 18}}
	require.NotNil(t, e.Delta())
	assert.Equal(t, -4, *e.Delta())
	assert.Nil(t, ScoreEntry{Score: s}.Delta())

	// a question without meta counts 0–4 towards the total only
	odd := NewScale([]catalog.Item{{Code: "x"}}, nil)
	assert.Equal(t, Question{Code: "x", Max: DefaultItemMax}, odd.Questions[0])
	assert.Equal(t, 3, odd.Compute(day("2026-09-23"), map[string]int{"x": 3}).Total)
}

func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, civildate.Tehran)
	if err != nil {
		panic(err)
	}
	return t
}

func TestFlashRules(t *testing.T) {
	assert.True(t, IsNightHour(at("2026-10-01 03:20")))
	assert.True(t, IsNightHour(at("2026-10-01 22:00")))
	assert.False(t, IsNightHour(at("2026-10-01 06:00")))
	assert.False(t, IsNightHour(at("2026-10-01 16:10")))

	d1, d2 := 240, 120
	day := FlashDay{Flashes: []Flash{
		{StartedAt: at("2026-10-01 16:10"), Duration: &d1},
		{StartedAt: at("2026-10-01 11:30"), Duration: &d2},
		{StartedAt: at("2026-10-01 03:20"), Night: true},
	}}
	assert.Equal(t, 3, day.Count())
	assert.Equal(t, 1, day.NightCount())
	require.NotNil(t, day.AvgDuration())
	assert.Equal(t, 180, *day.AvgDuration())
	assert.Nil(t, FlashDay{}.AvgDuration())

	running := Flash{StartedAt: at("2026-10-01 10:00")}
	assert.True(t, running.Running())
	assert.Equal(t, 102, running.Elapsed(at("2026-10-01 10:00").Add(102*time.Second)))
	assert.Equal(t, MaxFlashSeconds, running.Elapsed(at("2026-10-01 13:00")))

	assert.Equal(t, []string{"caffeine", "spicy_food", "stress", "exercise", "warm_room", "hot_drink"}, LogTriggers())
	assert.Equal(t, "unknown", Triggers()[len(Triggers())-1])
}

func entry(slot, code string) taxonomy.Entry {
	parts := strings.SplitN(slot, ".", 3)
	e := taxonomy.Entry{Category: parts[0], Param: parts[1], Item: parts[2]}
	if code != "" {
		e.Code = sql.NullString{String: code, Valid: true}
	}
	return e
}

func logDay(entries ...taxonomy.Entry) LogDay {
	views := analysis.BuildDays([]analysis.DayEntries{{Date: day("2026-10-01"), Entries: entries}})
	return LogDay{Entries: entries, View: views[day("2026-10-01")]}
}

func TestLogDay(t *testing.T) {
	assert.False(t, logDay(entry("bleeding.presence.", "none")).Bleeding())
	assert.True(t, logDay(entry("bleeding.presence.", "spotting")).Bleeding())
	assert.True(t, logDay(entry("bleeding.flow.", "light")).Bleeding())
	assert.True(t, logDay(entry("bleeding.spotting.", "yes")).Bleeding())
	assert.False(t, logDay(entry("bleeding.spotting.", "no")).Bleeding())

	d := logDay(entry("menopause.triggers.caffeine", ""), entry("menopause.triggers.stress", ""),
		entry(SlotNightSweats, "moderate"), entry(SlotFatigue, "no"))
	assert.Equal(t, []string{"caffeine", "stress"}, d.Triggers())
	assert.True(t, d.Has(SlotNightSweats))
	assert.Equal(t, "moderate", d.Level(SlotNightSweats))
	assert.False(t, d.Has(SlotFatigue))
	assert.False(t, LogDay{}.Has(SlotFatigue))
}

// patternLogs builds n logged days ending 2026-10-01: on every 3rd day caffeine; flashes 4 on caffeine days, 1
// otherwise; night sweats every other day with fatigue on exactly those days.
func patternLogs(n int) (map[civildate.Date]LogDay, []Flash) {
	logs := map[civildate.Date]LogDay{}
	var flashes []Flash
	for i := range n {
		d := day("2026-10-01").AddDays(-i)
		var es []taxonomy.Entry
		count := 1
		if i%3 == 0 {
			es = append(es, entry("menopause.triggers.caffeine", ""))
			count = 4
		}
		if i%2 == 0 {
			es = append(es, entry(SlotNightSweats, "mild"), entry(SlotFatigue, "moderate"))
		} else {
			es = append(es, entry(SlotFatigue, "no"))
		}
		views := analysis.BuildDays([]analysis.DayEntries{{Date: d, Entries: es}})
		logs[d] = LogDay{Entries: es, View: views[d]}
		for k := range count {
			flashes = append(flashes, Flash{StartedAt: d.TehranMidnight().Add(time.Duration(10+k) * time.Hour)})
		}
	}
	return logs, flashes
}

func TestBuildPatterns(t *testing.T) {
	logs, flashes := patternLogs(30)
	ps := BuildPatterns(logs, flashes, LogTriggers())
	require.Len(t, ps, 2, "only caffeine was logged, plus night sweats")

	caf := ps[0]
	assert.Equal(t, PatternTriggerFlashes, caf.Key)
	assert.Equal(t, "caffeine", caf.Trigger)
	assert.True(t, caf.Found())
	assert.Equal(t, DirectionMore, caf.Direction())
	assert.Equal(t, analysis.StrengthStrong, caf.Strength)
	assert.Equal(t, 30, caf.N)
	assert.Equal(t, []analysis.Group{{Key: "without", Days: 20, Hits: 0}, {Key: "with", Days: 10, Hits: 10}}, caf.Groups)
	assert.Equal(t, []float64{1, 4}, caf.MeanFlashes)

	ns := ps[1]
	assert.Equal(t, PatternNightSweatsFatigue, ns.Key)
	assert.True(t, ns.Found())
	assert.Equal(t, DirectionMore, ns.Direction())
	assert.InDelta(t, 1.0, ns.Value, 0.001)

	// too few days: the engine's minimum (analysis.CorrMinDays) holds the cards back
	logs, flashes = patternLogs(analysis.CorrMinDays - 1)
	for _, p := range BuildPatterns(logs, flashes, LogTriggers()) {
		assert.Equal(t, analysis.CorrNotEnoughData, p.Status, p.Key)
		assert.False(t, p.Found())
		assert.Empty(t, p.Direction())
	}

	// no logs at all: only the night-sweats card, not enough data
	ps = BuildPatterns(map[civildate.Date]LogDay{}, nil, LogTriggers())
	require.Len(t, ps, 1)
	assert.Equal(t, analysis.CorrNotEnoughData, ps[0].Status)
}

func TestSweatyNightWindow(t *testing.T) {
	d := day("2026-10-01")
	assert.True(t, sweatyNight([]Flash{{StartedAt: at("2026-09-30 23:30"), Sweat: true}}, d))
	assert.True(t, sweatyNight([]Flash{{StartedAt: at("2026-10-01 05:59"), Sweat: true}}, d))
	assert.False(t, sweatyNight([]Flash{{StartedAt: at("2026-10-01 06:00"), Sweat: true}}, d))
	assert.False(t, sweatyNight([]Flash{{StartedAt: at("2026-09-30 23:30")}}, d))
}

func TestTreatmentAdherence(t *testing.T) {
	daily := TreatmentToday{Item: store.TreatmentItem{Kind: KindHRT}, DaysTaken: 6, Days: 7}
	require.NotNil(t, daily.AdherencePct())
	assert.Equal(t, 86, *daily.AdherencePct())
	weekly := TreatmentToday{Item: store.TreatmentItem{Kind: "supplement", Schedule: sql.NullString{String: ScheduleWeekly, Valid: true}}, Days: 7}
	assert.Nil(t, weekly.AdherencePct())
	assert.Nil(t, TreatmentToday{Item: store.TreatmentItem{Kind: KindLifestyle}, Days: 7}.AdherencePct())

	today := day("2026-10-01")
	it := store.TreatmentItem{StartedOn: civildate.NullDate{Date: day("2026-10-02"), Valid: true}}
	assert.False(t, activeOn(it, today), "not started yet")
	it = store.TreatmentItem{StoppedOn: civildate.NullDate{Date: today, Valid: true}}
	assert.False(t, activeOn(it, today), "stopped today")
	assert.True(t, activeOn(store.TreatmentItem{}, today))
}

func TestLangFiles(t *testing.T) {
	for _, locale := range []string{"en", "fa"} {
		for _, key := range []string{"messages.validation_failed", "messages.profile_saved", "messages.flash_started",
			"messages.flash_running", "messages.flash_logged", "messages.flash_saved", "messages.flash_not_found",
			"messages.score_saved", "validation.date_future", "validation.started_too_old", "validation.answers_missing",
			"patterns.trigger_flashes.more", "patterns.trigger_flashes.less", "patterns.night_sweats_fatigue.more",
			"patterns.night_sweats_fatigue.less"} {
			assert.NotEqual(t, "menopause."+key, T(key, locale), locale+" "+key)
		}
		assert.NotEmpty(t, attributes(locale))
	}
	assert.Equal(t, "On days you logged Coffee, you had more hot flashes.",
		Tp("patterns.trigger_flashes.more", map[string]string{"trigger": "Coffee"}, "en"))
	assert.Equal(t, T("messages.score_saved", "en"), T("messages.score_saved", "de"), "fallback to English")
}
