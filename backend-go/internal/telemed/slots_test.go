package telemed_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/telemed"
)

// tue is Tuesday 2026-10-06 (Tehran); mon is the Monday before it.
var (
	tue = civildate.MustParse("2026-10-06")
	mon = time.Date(2026, 10, 5, 9, 0, 0, 0, civildate.Tehran)
)

func wall(d civildate.Date, hhmm string) time.Time {
	m, ok := telemed.ParseClock(hhmm)
	if !ok {
		panic(hhmm)
	}
	return d.TehranMidnight().Add(time.Duration(m) * time.Minute)
}

func times(day telemed.Day) []string {
	out := []string{}
	for _, s := range day.Slots {
		out = append(out, s.Start.In(civildate.Tehran).Format("15:04"))
	}
	return out
}

// eveningRule is Tuesday 16:00–20:00 on a 30-minute grid (nbl_v17_DoctorProfile: ۱۶:۰۰ … ۱۹:۳۰).
var eveningRule = telemed.Rule{Weekday: time.Tuesday, StartMinute: 16 * 60, EndMinute: 20 * 60, SlotMinutes: 30}

func query(rules []telemed.Rule, mode string, duration int, now time.Time, blocked ...telemed.Interval) telemed.SlotQuery {
	return telemed.SlotQuery{Rules: rules, Mode: mode, Duration: duration, From: tue, Days: 1, Now: now, Blocked: blocked}
}

func TestGenerateDays_Grid(t *testing.T) {
	days := telemed.GenerateDays(query([]telemed.Rule{eveningRule}, telemed.ModeVideo, 20, mon))
	require.Len(t, days, 1)
	assert.Equal(t, tue, days[0].Date)
	assert.Equal(t, []string{"16:00", "16:30", "17:00", "17:30", "18:00", "18:30", "19:00", "19:30"}, times(days[0]))
	s := days[0].Slots[0]
	assert.Equal(t, 20*time.Minute, s.End.Sub(s.Start))
	assert.Equal(t, "2026-10-06T16:00:00+03:30", s.Start.Format(time.RFC3339))
}

func TestGenerateDays_VisitMustEndInsideTheWindow(t *testing.T) {
	r := telemed.Rule{Weekday: time.Tuesday, StartMinute: 16 * 60, EndMinute: 18 * 60, SlotMinutes: 30}
	days := telemed.GenerateDays(query([]telemed.Rule{r}, telemed.ModeInPerson, 45, mon))
	assert.Equal(t, []string{"16:00", "16:30", "17:00"}, times(days[0]), "17:30 + 45 min ends after 18:00")
}

func TestGenerateDays_LeadTimeHidesSoonAndPastSlots(t *testing.T) {
	now := wall(tue, "16:10") // earliest start 16:40
	days := telemed.GenerateDays(query([]telemed.Rule{eveningRule}, telemed.ModeVideo, 20, now))
	assert.Equal(t, []string{"17:00", "17:30", "18:00", "18:30", "19:00", "19:30"}, times(days[0]))

	days = telemed.GenerateDays(query([]telemed.Rule{eveningRule}, telemed.ModeVideo, 20, wall(tue, "21:00")))
	assert.Empty(t, days[0].Slots, "the whole evening is past")
}

func TestGenerateDays_TimeOffAndBusyRemoveOverlappingSlots(t *testing.T) {
	off := telemed.Interval{Start: wall(tue, "17:00"), End: wall(tue, "18:00")}
	busy := telemed.Interval{Start: wall(tue, "19:10"), End: wall(tue, "19:30")}
	days := telemed.GenerateDays(query([]telemed.Rule{eveningRule}, telemed.ModeVideo, 20, mon, off, busy))
	// 19:00–19:20 overlaps the busy 19:10–19:30; 19:30 starts when it ends (half-open) and stays.
	assert.Equal(t, []string{"16:00", "16:30", "18:00", "18:30", "19:30"}, times(days[0]))

	edge := telemed.Interval{Start: wall(tue, "16:20"), End: wall(tue, "16:30")}
	days = telemed.GenerateDays(query([]telemed.Rule{eveningRule}, telemed.ModeVideo, 20, mon, edge))
	assert.Equal(t, "16:00", times(days[0])[0], "16:00–16:20 ends when the block starts")
	assert.Equal(t, "16:30", times(days[0])[1])
}

func TestGenerateDays_RuleModes(t *testing.T) {
	inPerson := telemed.Rule{Weekday: time.Tuesday, StartMinute: 9 * 60, EndMinute: 10 * 60, SlotMinutes: 30,
		Modes: []string{telemed.ModeInPerson}}
	rules := []telemed.Rule{eveningRule, inPerson}
	video := telemed.GenerateDays(query(rules, telemed.ModeVideo, 20, mon.Add(-24*time.Hour)))
	assert.Equal(t, "16:00", times(video[0])[0], "the morning window takes in-person visits only")
	office := telemed.GenerateDays(query(rules, telemed.ModeInPerson, 30, mon.Add(-24*time.Hour)))
	assert.Equal(t, []string{"09:00", "09:30", "16:00"}, times(office[0])[:3])
}

func TestGenerateDays_OverlappingRulesNeverRepeatAStart(t *testing.T) {
	a := telemed.Rule{Weekday: time.Tuesday, StartMinute: 16 * 60, EndMinute: 17 * 60, SlotMinutes: 30}
	b := telemed.Rule{Weekday: time.Tuesday, StartMinute: 16*60 + 30, EndMinute: 18 * 60, SlotMinutes: 30}
	days := telemed.GenerateDays(query([]telemed.Rule{b, a}, telemed.ModeVideo, 30, mon))
	assert.Equal(t, []string{"16:00", "16:30", "17:00", "17:30"}, times(days[0]))
}

func TestGenerateDays_WeekdaysAndEmptyDays(t *testing.T) {
	sat := telemed.Rule{Weekday: time.Saturday, StartMinute: 10 * 60, EndMinute: 11 * 60, SlotMinutes: 60}
	q := query([]telemed.Rule{sat, eveningRule}, telemed.ModeVideo, 60, mon)
	q.Days = 7
	days := telemed.GenerateDays(q)
	require.Len(t, days, 7)
	counts := map[string]int{}
	for _, d := range days {
		counts[d.Date.String()] = len(d.Slots)
		assert.NotNil(t, d.Slots, "empty days still carry an empty list")
	}
	assert.Equal(t, 7, counts["2026-10-06"], "Tuesday 16:00 … 19:00 (an hour each, 30-minute grid)")
	assert.Equal(t, 1, counts["2026-10-10"], "Saturday 10:00")
	assert.Equal(t, 0, counts["2026-10-07"])
	assert.Equal(t, 0, counts["2026-10-12"], "the next Monday has no rule")
}

func TestGenerateDays_Horizon(t *testing.T) {
	q := query([]telemed.Rule{eveningRule}, telemed.ModeVideo, 20, mon)
	q.From = civildate.InTehran(mon).AddDays(telemed.HorizonDays) // first day past the horizon
	q.Days = 7
	for _, d := range telemed.GenerateDays(q) {
		assert.Empty(t, d.Slots, d.Date.String())
	}
}

func TestGenerateDays_NoDurationOrStep(t *testing.T) {
	days := telemed.GenerateDays(query([]telemed.Rule{eveningRule}, telemed.ModeVideo, 0, mon))
	require.Len(t, days, 1)
	assert.Empty(t, days[0].Slots)
	broken := telemed.Rule{Weekday: time.Tuesday, StartMinute: 0, EndMinute: 60}
	days = telemed.GenerateDays(query([]telemed.Rule{broken}, telemed.ModeVideo, 20, mon))
	assert.Empty(t, days[0].Slots, "a zero step yields nothing instead of looping")
}

func TestFirstSlotAny_EarliestOverModes(t *testing.T) {
	morning := telemed.Rule{Weekday: time.Wednesday, StartMinute: 9 * 60, EndMinute: 12 * 60, SlotMinutes: 30,
		Modes: []string{telemed.ModePhone}}
	rules := []telemed.Rule{eveningRule, morning}
	visits := []telemed.VisitLength{{Mode: telemed.ModeVideo, Duration: 20}, {Mode: telemed.ModePhone, Duration: 15}}
	now := wall(tue, "19:20") // video 19:30 is within the lead time; next video is a week later
	s, mode, ok := telemed.FirstSlotAny(rules, visits, tue, 14, now, nil)
	require.True(t, ok)
	assert.Equal(t, telemed.ModePhone, mode)
	assert.Equal(t, wall(tue.AddDays(1), "09:00"), s.Start)

	_, _, ok = telemed.FirstSlotAny(nil, visits, tue, 14, now, nil)
	assert.False(t, ok)
}

func TestParseClock(t *testing.T) {
	for in, want := range map[string]int{"00:00": 0, "09:30": 570, "23:59": 1439, "24:00": 1440} {
		got, ok := telemed.ParseClock(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	for _, in := range []string{"24:01", "9:30", "12:60", "ab:cd", ""} {
		_, ok := telemed.ParseClock(in)
		assert.False(t, ok, in)
	}
}

func TestIntervalOverlaps(t *testing.T) {
	a := telemed.Interval{Start: wall(tue, "10:00"), End: wall(tue, "11:00")}
	assert.True(t, a.Overlaps(telemed.Interval{Start: wall(tue, "10:59"), End: wall(tue, "12:00")}))
	assert.False(t, a.Overlaps(telemed.Interval{Start: wall(tue, "11:00"), End: wall(tue, "12:00")}))
	assert.False(t, a.Overlaps(telemed.Interval{Start: wall(tue, "09:00"), End: wall(tue, "10:00")}))
}
