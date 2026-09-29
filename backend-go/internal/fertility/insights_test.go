package fertility

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/fertility/bbt"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

var insightsToday = civildate.MustParse("2026-09-23")

// fixture is a TTC user (profile 28/5, LMP = the newest start) with confirmed periods starting
// on starts (newest first); readings / tests are logged data.
func fixture(t *testing.T, starts []string, readings []bbt.Reading, tests []LHTest) Insights {
	t.Helper()
	histories := make([]model.History, 0, len(starts))
	for _, s := range starts {
		d := civildate.MustParse(s)
		histories = append(histories, model.History{PeriodStart: d, PeriodEnd: d.AddDays(4), IsConfirmed: true, Source: "user_logged"})
	}
	profile := &model.Profile{CycleDuration: model.Int(28), PeriodDuration: model.Int(5), LastPeriodStart: civildate.MustParse("2026-09-10")}
	in := cycleInputs{histories: histories, profile: profile, metrics: metrics.Calculate(histories, profile)}
	return in.insights(in.bbtCycles(insightsToday), readings, tests, insightsToday)
}

// biphasicReadings is a 28-day cycle from start whose readings rise on shiftDay.
func biphasicReadings(start string, shiftDay int) []bbt.Reading {
	d := civildate.MustParse(start)
	out := make([]bbt.Reading, 0, 28)
	for i := range 28 {
		v := 3620 + (i%3)*5
		if i+1 >= shiftDay {
			v = 3680
		}
		out = append(out, bbt.Reading{Date: d.AddDays(i), Value: v})
	}
	return out
}

func strengths(ins Insights) map[string]string {
	m := map[string]string{}
	for _, e := range ins.Evidence {
		m[e.Key] = e.Strength
	}
	return m
}

func TestInsights_NoCycles(t *testing.T) {
	ins := fixture(t, nil, nil, nil)

	assert.Equal(t, 0, ins.CyclesUsed)
	require.NotNil(t, ins.Window, "the profile's LMP anchors the cycle view")
	assert.Equal(t, "2026-09-19", ins.Window.Start.String())
	assert.Equal(t, "2026-09-24", ins.Window.End.String())
	assert.Equal(t, "2026-09-24", ins.Window.Ovulation.String())
	assert.Equal(t, 10, ins.Window.FirstCycleDay)
	assert.Equal(t, map[string]string{"cycles": "none", "bbt_shift": "none", "lh": "none"}, strengths(ins))
	assert.Equal(t, ConfidenceLow, ins.Confidence)
	assert.Empty(t, ins.History)
	assert.Equal(t, []string{TipBBTDaily, TipLHFromDay, TipLogPeriods}, ins.Tips)

	body := InsightsJSON(ins, "fa")
	raw, err := body.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"cycles_used": 0,
		"window": {"start": "2026-09-19", "end": "2026-09-24", "ovulation": "2026-09-24"},
		"confidence": "low",
		"evidence": [
			{"key": "cycles", "title": "هنوز سیکل کاملی ثبت نشده", "detail": "طول معمول سیکل بعد از اولین سیکل کامل مشخص می\u200cشود", "strength": "none"},
			{"key": "bbt_shift", "title": "جهش دما", "detail": "هنوز جهش تأییدشده\u200cای دیده نشده", "strength": "none"},
			{"key": "lh", "title": "تست LH", "detail": "هنوز در این سیکل ثبت نشده", "strength": "none"}
		],
		"history": [],
		"tips": [
			"دمای پایه را هر روز صبح، قبل از بلند شدن و در یک ساعت ثابت ثبت کن.",
			"تست LH را از روز ۱۰ سیکل شروع کن و هر روز تا مثبت شدن ادامه بده.",
			"شروع و پایان هر پریود را ثبت کن تا طول معمول سیکلت دقیق\u200cتر شود."
		]
	}`, string(raw))
}

func TestInsights_NoAnchor(t *testing.T) {
	in := cycleInputs{metrics: metrics.Calculate(nil, nil)}
	ins := in.insights(in.bbtCycles(insightsToday), nil, nil, insightsToday)

	assert.Nil(t, ins.Window)
	assert.Equal(t, ConfidenceLow, ins.Confidence)
	assert.Equal(t, []string{TipBBTDaily, TipLogPeriods}, ins.Tips, "no LH day without a window")
	raw, err := InsightsJSON(ins, "en").MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"window":null`)
}

// Two finished cycles with BBT shifts on day 15 and 16 and a positive LH test today−1.
func TestInsights_TwoCycles(t *testing.T) {
	readings := append(biphasicReadings("2026-07-16", 15), biphasicReadings("2026-08-13", 16)...)
	tests := []LHTest{
		{Date: civildate.MustParse("2026-09-20"), Value: "negative"},
		{Date: civildate.MustParse("2026-09-21"), Value: "faint"},
		{Date: civildate.MustParse("2026-09-22"), Value: "positive"},
	}
	ins := fixture(t, []string{"2026-09-10", "2026-08-13", "2026-07-16"}, readings, tests)

	assert.Equal(t, 2, ins.CyclesUsed)
	assert.Equal(t, map[string]string{"cycles": "medium", "bbt_shift": "strong", "lh": "strong"}, strengths(ins))
	assert.Equal(t, ConfidenceHigh, ins.Confidence)
	assert.Equal(t, []int{16, 15}, ins.Evidence[1].ShiftDays, "newest cycle first")
	assert.Equal(t, 13, ins.Evidence[2].LHDay)
	assert.Equal(t, []string{TipLogPeriods}, ins.Tips)

	require.Len(t, ins.History, 2)
	assert.Equal(t, "2026-08-27", ins.History[0].Ovulation.String(), "shift day 16 − 1")
	assert.Equal(t, SourceBBT, ins.History[0].Source)
	assert.Equal(t, "2026-07-29", ins.History[1].Ovulation.String())

	raw, err := InsightsJSON(ins, "fa").MarshalJSON()
	require.NoError(t, err)
	s := string(raw)
	assert.Contains(t, s, `"history":[{"month_label":"شهریور","ovulation_day":15,"date":"2026-08-27","source":"bbt","cycle_start":"2026-08-13","period_days":5},{"month_label":"مرداد","ovulation_day":14,`)
	// v19_TTC_Insights: the count/fact is the title, the detail carries the numbers (audit #18).
	assert.Contains(t, s, `{"key":"cycles","title":"۲ سیکل کامل ثبت شده","detail":"طول معمول ۲۸ روز","strength":"medium"}`, "no «(±۰)»")
	assert.Contains(t, s, `{"key":"bbt_shift","title":"جهش دما در ۲ سیکل اخیر","detail":"روز ۱۵ و ۱۶ سیکل","strength":"strong"}`, "days ascending")
	assert.Contains(t, s, `{"key":"lh","title":"تست LH","detail":"مثبت در روز ۱۳ سیکل","strength":"strong"}`)

	en, err := InsightsJSON(ins, "en").MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(en), `"month_label":"August","ovulation_day":15`)
	assert.Contains(t, string(en), `"title":"Temperature rise in 2 recent cycles","detail":"Cycle days 15 and 16"`)
	assert.Contains(t, string(en), `"title":"2 complete cycles logged","detail":"Usual length 28 days"`)
}

// Six regular finished cycles, no BBT; one past cycle has a positive LH test.
func TestInsights_SixCycles(t *testing.T) {
	starts := []string{"2026-09-10", "2026-08-13", "2026-07-16", "2026-06-18", "2026-05-21", "2026-04-23", "2026-03-26"}
	tests := []LHTest{{Date: civildate.MustParse("2026-06-30"), Value: "positive"}} // day 13 of 06-18
	ins := fixture(t, starts, nil, tests)

	assert.Equal(t, 6, ins.CyclesUsed)
	assert.Equal(t, map[string]string{"cycles": "strong", "bbt_shift": "none", "lh": "none"}, strengths(ins))
	assert.Equal(t, ConfidenceMedium, ins.Confidence)
	assert.Equal(t, []string{TipBBTDaily, TipLHFromDay}, ins.Tips)

	require.Len(t, ins.History, 5, "the last 5 finished cycles")
	var sources []string
	var days []int
	for _, h := range ins.History {
		sources = append(sources, h.Source)
		require.NotNil(t, h.OvulationDay)
		days = append(days, *h.OvulationDay)
	}
	assert.Equal(t, []string{SourceEstimate, SourceEstimate, SourceLH, SourceEstimate, SourceEstimate}, sources)
	assert.Equal(t, []int{15, 15, 14, 15, 15}, days, "engine: 28 − 14 → day 15; LH positive day 13 + 1")
	assert.Equal(t, "2026-08-13", ins.History[0].Start.String())
	assert.Equal(t, "2026-04-23", ins.History[4].Start.String())
}

func TestConfidenceRule(t *testing.T) {
	assert.Equal(t, ConfidenceLow, confidence(false, StrengthStrong, StrengthStrong, StrengthStrong))
	assert.Equal(t, ConfidenceHigh, confidence(true, StrengthStrong, StrengthStrong, StrengthNone))
	assert.Equal(t, ConfidenceMedium, confidence(true, StrengthNone, StrengthStrong, StrengthStrong), "no cycles → never high")
	assert.Equal(t, ConfidenceMedium, confidence(true, StrengthMedium, StrengthMedium, StrengthNone))
	assert.Equal(t, ConfidenceLow, confidence(true, StrengthMedium, StrengthNone, StrengthNone))
}

func TestEvidenceCopy(t *testing.T) {
	variability := func(v int) *int { return &v }
	cases := []struct {
		name          string
		e             Evidence
		title, detail string
	}{
		{"cycles with spread", Evidence{Key: EvidenceCycles, Cycles: 6, Length: 29, Variability: variability(3)},
			"۶ سیکل کامل ثبت شده", "طول معمول ۲۹ روز، نوسان ±۲"},
		{"cycles, zero spread", Evidence{Key: EvidenceCycles, Cycles: 3, Length: 28, Variability: variability(0)},
			"۳ سیکل کامل ثبت شده", "طول معمول ۲۸ روز"},
		{"cycles, one cycle (no spread yet)", Evidence{Key: EvidenceCycles, Cycles: 1, Length: 30},
			"۱ سیکل کامل ثبت شده", "طول معمول ۳۰ روز"},
		{"one shift", Evidence{Key: EvidenceBBTShift, ShiftDays: []int{15}},
			"جهش دما در ۱ سیکل اخیر", "روز ۱۵ سیکل"},
		{"two shifts, same day once", Evidence{Key: EvidenceBBTShift, ShiftDays: []int{16, 15, 16}},
			"جهش دما در ۳ سیکل اخیر", "روز ۱۵ و ۱۶ سیکل"},
		{"lh faint", Evidence{Key: EvidenceLH, Strength: StrengthMedium, LHDay: 12}, "تست LH", "کم\u200cرنگ در روز ۱۲ سیکل"},
		{"lh negatives", Evidence{Key: EvidenceLH, Strength: StrengthNone, LHTests: 2}, "تست LH", "۲ تست در این سیکل، هنوز مثبت نشده"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.title, evidenceTitle(c.e, "fa"))
			assert.Equal(t, c.detail, evidenceDetail(c.e, "fa"))
			assert.NotContains(t, evidenceDetail(c.e, "fa"), "(")
		})
	}
	assert.Equal(t, "1 complete cycle logged", evidenceTitle(Evidence{Key: EvidenceCycles, Cycles: 1}, "en"))
	assert.Equal(t, "Temperature rise in 1 recent cycle", evidenceTitle(Evidence{Key: EvidenceBBTShift, ShiftDays: []int{14}}, "en"))
	assert.Equal(t, "1 test this cycle, not positive yet", evidenceDetail(Evidence{Key: EvidenceLH, LHTests: 1}, "en"))
	assert.Equal(t, "Usual length 29 days, varies ±2", evidenceDetail(Evidence{Key: EvidenceCycles, Cycles: 6, Length: 29, Variability: variability(4)}, "en"))
}

// period_days is each finished cycle's own logged period (end − start + 1), not the effective one.
func TestInsights_HistoryPeriodDays(t *testing.T) {
	starts := []string{"2026-09-10", "2026-08-13", "2026-07-16", "2026-06-18"}
	ends := []int{4, 6, 2, -1} // days after the start; −1 = no logged end
	histories := make([]model.History, 0, len(starts))
	for i, s := range starts {
		d := civildate.MustParse(s)
		h := model.History{PeriodStart: d, IsConfirmed: true, Source: "user_logged"}
		if ends[i] >= 0 {
			h.PeriodEnd = d.AddDays(ends[i])
		}
		histories = append(histories, h)
	}
	bleeding := 4
	histories[3].BleedingLength = &bleeding
	histories = append(histories, model.History{PeriodStart: civildate.MustParse("2026-05-21"), IsConfirmed: true, Source: "user_logged"})
	profile := &model.Profile{CycleDuration: model.Int(28), PeriodDuration: model.Int(5), LastPeriodStart: civildate.MustParse("2026-09-10")}
	in := cycleInputs{histories: histories, profile: profile, metrics: metrics.Calculate(histories, profile)}
	ins := in.insights(in.bbtCycles(insightsToday), nil, nil, insightsToday)

	require.Len(t, ins.History, 4)
	var got []any
	for _, h := range ins.History {
		if h.PeriodDays == nil {
			got = append(got, nil)
			continue
		}
		got = append(got, *h.PeriodDays)
	}
	assert.Equal(t, []any{7, 3, 4, nil}, got, "08-13: 7 days, 07-16: 3 days, 06-18: bleeding_length, 05-21: unknown")

	raw, err := InsightsJSON(ins, "en").MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"cycle_start":"2026-05-21","period_days":null}`)
}

// The T-M5-11 staging case (last start 09-13, 5 days, 28-day cycle, today = cycle day 16): the BBT
// chart's fertile window is task.md §19's display window from the cycle view's anchors —
// max(O−5, period end + 1) … O — which is what the home schedule now derives too (days 10–15,
// never day 16).
func TestBBTWindow_CycleDay16MatchesAnchors(t *testing.T) {
	today := civildate.MustParse("2026-09-28")
	var histories []model.History
	for _, s := range []string{"2026-09-13", "2026-08-16", "2026-07-19", "2026-06-21"} {
		d := civildate.MustParse(s)
		histories = append(histories, model.History{PeriodStart: d, PeriodEnd: d.AddDays(4), IsConfirmed: true, Source: "user_logged"})
	}
	profile := &model.Profile{CycleDuration: model.Int(28), PeriodDuration: model.Int(5), LastPeriodStart: civildate.MustParse("2026-09-13")}
	in := cycleInputs{histories: histories, profile: profile, metrics: metrics.Calculate(histories, profile)}

	st := in.resolve(today, today)
	require.NotNil(t, st.CycleDay)
	require.Equal(t, 16, *st.CycleDay)
	ov := st.EstimatedOvulationDate
	start := ov.AddDays(-5)
	if next := st.CurrentPeriodEnd.AddDays(1); next.After(start) {
		start = next
	}
	cycles := in.bbtCycles(today)
	require.NotEmpty(t, cycles)
	w := cycles[0].FertileWindow
	require.NotNil(t, w)
	assert.Equal(t, bbt.DayRange{FromDay: st.CurrentPeriodStart.DiffDays(start) + 1, ToDay: st.CurrentPeriodStart.DiffDays(ov) + 1}, *w)
	assert.Equal(t, bbt.DayRange{FromDay: 10, ToDay: 15}, *w)
}
