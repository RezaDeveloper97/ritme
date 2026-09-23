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
			{"key": "cycles", "title": "سیکل\u200cهای کامل", "detail": "هنوز سیکل کاملی ثبت نشده است.", "strength": "none"},
			{"key": "bbt_shift", "title": "جهش دمای پایه", "detail": "هنوز جهش تأییدشده\u200cای دیده نشده است.", "strength": "none"},
			{"key": "lh", "title": "تست LH این سیکل", "detail": "این سیکل تستی ثبت نشده است.", "strength": "none"}
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
	assert.Contains(t, s, `"history":[{"month_label":"شهریور","ovulation_day":15,"date":"2026-08-27","source":"bbt","cycle_start":"2026-08-13"},{"month_label":"مرداد","ovulation_day":14,`)
	assert.Contains(t, s, `"detail":"جهش در ۲ سیکل، روز ۱۶ و ۱۵"`)
	assert.Contains(t, s, `"detail":"مثبت در روز ۱۳"`)
	assert.Contains(t, s, `"detail":"۲ سیکل کامل، معمولاً ۲۸ روز (±۰)"`)

	en, err := InsightsJSON(ins, "en").MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(en), `"month_label":"August","ovulation_day":15`)
	assert.Contains(t, string(en), `"detail":"A rise in 2 cycles, on days 16 and 15"`)
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
