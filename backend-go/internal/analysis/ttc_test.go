package analysis

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// TTC fixtures (B-N3-11). Fixed request day 2026-09-23 (fixtureToday). Goldens: testdata/golden/ttc_*.json
// (UPDATE_GOLDEN=1 go test ./internal/analysis/ -run TTC, review by hand).

// legacyLog is one seeded fertility_logs day.
type legacyLog struct {
	d         civildate.Date
	lh, mucus string
}

// ttcCycle seeds a cycle of `length` days (0 = the current cycle up to today) with ovulation on day ov:
// low temperatures (36.20–36.35) up to ov, high ones (36.65–36.80) after it, an LH surge on ov−1, egg-white
// mucus on ov−2..ov−1 and intercourse on ov−3, ov−1, ov+1. skipBBT leaves the temperatures out.
func ttcCycle(start civildate.Date, length, ov int, today civildate.Date, skipBBT bool, rows *[]DayEntries, legacy *[]legacyLog) {
	if length == 0 {
		length = start.DiffDays(today) + 1
	}
	for n := 1; n <= length; n++ {
		d := start.AddDays(n - 1)
		if d.After(today) {
			break
		}
		var es []taxonomy.Entry
		if !skipBBT && n%9 != 0 { // a missed morning now and then
			t := 36.20 + float64((n*7)%4)*0.05
			if n > ov {
				t = 36.65 + float64((n*5)%4)*0.05
			}
			es = append(es, number("measurements", "bbt", t))
		}
		switch n {
		case ov - 1:
			*legacy = append(*legacy, legacyLog{d, "positive", "egg_white"})
		case ov - 2:
			*legacy = append(*legacy, legacyLog{d, "faint", "creamy"})
			es = append(es, single("discharge", "consistency", "egg_white")) // v2 wins: more fertile
		case ov - 3:
			es = append(es, single("measurements", "lh_test", "negative"))
		}
		if n == ov-3 || n == ov-1 || n == ov+1 {
			es = append(es, single("sex", "intercourse", "unprotected"))
		}
		if n == ov+4 {
			es = append(es, single("sex", "intercourse", "protected")) // never counted
		}
		if n <= 5 {
			es = append(es, single("bleeding", "flow", "medium"))
		}
		if len(es) > 0 {
			*rows = append(*rows, DayEntries{Date: d, Entries: es})
		}
	}
}

// ttcFixture: 6 completed cycles (28 each) and the current one from 2026-09-09 (day 15 today). TTC logs
// started in the 4th-newest completed cycle (2026-06-17 cycle → 4 trying cycles + current = 4 completed … ),
// ovulation day 14 (luteal 14), BBT skipped in one cycle (LH-only), age 31, Plus.
func ttcFixture(plus bool, birthday string) *TTCInput {
	starts := []civildate.Date{}
	s := civildate.MustParse("2026-09-09")
	for range 7 {
		starts = append([]civildate.Date{s}, starts...)
		s = s.AddDays(-28)
	}
	var hist []model.History
	for i, st := range starts {
		hist = append(hist, model.History{ID: int64(i + 1), PeriodStart: st, PeriodEnd: st.AddDays(4), IsConfirmed: true, Source: "user_logged"})
	}
	var rows []DayEntries
	var legacy []legacyLog
	// trying since the 4th cycle from the end (starts[3]); BBT missing in starts[4].
	for i := 3; i < len(starts); i++ {
		length := 28
		if i == len(starts)-1 {
			length = 0
		}
		ttcCycle(starts[i], length, 14, fixtureToday, i == 4, &rows, &legacy)
	}
	in := &Input{
		Today: fixtureToday, Histories: hist, Profile: &model.Profile{}, DeepAnalysis: plus,
		Birthday: civildate.MustParse(birthday), Days: map[civildate.Date]*Day{},
	}
	sig := NewTTCSignals()
	sig.AddEntries(rows)
	for _, l := range legacy {
		sig.AddFertilityLog(l.d, l.lh, l.mucus)
	}
	return &TTCInput{Input: in, Signals: sig, FirstSignal: starts[3].AddDays(2)}
}

// ttcNoLogsFixture: cycles but no TTC log yet (recent-cycles fallback).
func ttcNoLogsFixture() *TTCInput {
	hist, _ := history(civildate.MustParse("2026-09-09"), []int{30, 27, 29}, func(int) int { return 5 })
	return &TTCInput{
		Input:   &Input{Today: fixtureToday, Histories: hist, Profile: &model.Profile{}, Days: map[civildate.Date]*Day{}},
		Signals: NewTTCSignals(),
	}
}

// ttcEmptyFixture: a new user — no cycle, no log.
func ttcEmptyFixture() *TTCInput {
	return &TTCInput{Input: &Input{Today: fixtureToday, Days: map[civildate.Date]*Day{}}, Signals: NewTTCSignals()}
}

// ttcLongFixture: 37 years old, trying for 8 months (referral due at 6 months from 35).
func ttcLongFixture() *TTCInput {
	in := ttcFixture(true, "1989-03-01")
	in.FirstSignal = civildate.MustParse("2026-01-20")
	starts := []civildate.Date{}
	s := in.Histories[0].PeriodStart
	for range 3 {
		s = s.AddDays(-28)
		starts = append(starts, s)
	}
	for i, st := range starts {
		in.Histories = append(in.Histories, model.History{ID: int64(100 + i), PeriodStart: st, PeriodEnd: st.AddDays(4), IsConfirmed: true})
	}
	return in
}

var ttcFixtures = map[string]func() *TTCInput{
	"plus":    func() *TTCInput { return ttcFixture(true, "1995-02-10") },
	"free":    func() *TTCInput { return ttcFixture(false, "1995-02-10") },
	"nologs":  ttcNoLogsFixture,
	"empty":   ttcEmptyFixture,
	"long_37": ttcLongFixture,
}

func TestTTC_Golden(t *testing.T) {
	fa := copyFor(t, "fa")
	for name, fx := range ttcFixtures {
		t.Run(name, func(t *testing.T) {
			in := fx()
			in.Copy = fa
			r := BuildTTC(in)
			golden(t, "ttc_"+name+"_hub", render(t, r.JSON()))
			if i, ok := r.FindCycle(civildate.Date{}); ok {
				golden(t, "ttc_"+name+"_fertility", render(t, r.FertilityJSON(i)))
			}
		})
	}
	in := ttcFixture(true, "1995-02-10")
	in.Copy = copyFor(t, "en")
	r := BuildTTC(in)
	golden(t, "ttc_plus_hub.en", render(t, r.JSON()))
	i, ok := r.FindCycle(r.Cycles[0].Start)
	require.True(t, ok)
	golden(t, "ttc_plus_fertility_first.en", render(t, r.FertilityJSON(i)))
}

func get(t *testing.T, m *jsonx.OrderedMap, path ...string) any {
	t.Helper()
	var cur any = m
	for _, k := range path {
		om, ok := cur.(*jsonx.OrderedMap)
		require.True(t, ok, "not an object at %s", k)
		cur, ok = om.Get(k)
		require.True(t, ok, "missing %s", k)
	}
	return cur
}

func TestTTC_Plus(t *testing.T) {
	in := ttcFixture(true, "1995-02-10")
	in.Copy = copyFor(t, "fa")
	r := BuildTTC(in)
	require.Len(t, r.Cycles, 4)
	assert.True(t, r.Trying)
	assert.Equal(t, 4, r.TryingCycles)
	assert.Equal(t, 3, r.Months) // 2026-06-17 → 2026-09-23
	// 3 completed cycles: two BBT-confirmed (day 14), one LH-only (day 13 + 1)
	for i, want := range []string{OvulationBBT, OvulationLH, OvulationBBT} {
		assert.Equal(t, want, r.Cycles[i].OvulationSource, i)
		assert.Equal(t, 14, r.Cycles[i].OvulationDay, i)
	}
	assert.Equal(t, []int{15, 16, 17}, r.Cycles[0].HighDays)
	assert.Equal(t, 14, r.Cycles[0].Luteal)
	assert.Equal(t, 0, r.Cycles[1].Luteal) // no BBT → no luteal
	cur := r.Cycles[3]
	assert.True(t, cur.Current)
	// day 15 today: one high reading so far — not confirmed yet, ovulation from the LH surge (day 13 + 1)
	assert.Equal(t, OvulationLH, cur.OvulationSource)
	assert.Equal(t, 14, cur.OvulationDay)
	assert.Equal(t, 13, cur.LHDay)
	assert.Equal(t, 2, r.Confirmed)
	assert.Equal(t, r.Confirmed, countSource(r.Cycles, OvulationBBT))

	out := r.JSON()
	assert.Equal(t, false, get(t, out, "timing", "locked"))
	// timing = the current cycle (window days 9–14 started): intercourse on 11 and 13 (15 is after it)
	assert.Equal(t, 2, get(t, out, "timing", "data", "in_window"))
	assert.Equal(t, 14, get(t, out, "luteal", "data", "days"))
	assert.Equal(t, LutealNormal, get(t, out, "luteal", "data", "status"))
	assert.Equal(t, 1, get(t, out, "mucus", "data", "lead_min"))
	assert.Equal(t, false, get(t, out, "trying", "referral", "due"))
	assert.Equal(t, 12, get(t, out, "trying", "referral", "threshold_months"))
}

func countSource(cs []TTCCycle, src string) int {
	n := 0
	for _, c := range cs {
		if c.OvulationSource == src {
			n++
		}
	}
	return n
}

func TestTTC_FreeLocksPlusCards(t *testing.T) {
	r := BuildTTC(ttcFixture(false, "1995-02-10")).JSON()
	for _, k := range []string{"timing", "mucus", "luteal"} {
		assert.Equal(t, true, get(t, r, k, "locked"), k)
		assert.Nil(t, get(t, r, k, "data"), k)
	}
	assert.Equal(t, false, get(t, r, "bbt", "locked"))
	assert.Equal(t, false, get(t, r, "regularity", "plus"))
}

func TestTTC_Referral(t *testing.T) {
	r := BuildTTC(ttcLongFixture()).JSON()
	assert.Equal(t, true, get(t, r, "trying", "referral", "due"))
	assert.Equal(t, "from_35", get(t, r, "trying", "referral", "age_band"))
	assert.Equal(t, 6, get(t, r, "trying", "referral", "threshold_months"))

	months, band := referral(0)
	assert.Equal(t, ReferralMonthsUnder35, months)
	assert.Equal(t, "unknown", band)
	months, band = referral(34)
	assert.Equal(t, 12, months)
	assert.Equal(t, "under_35", band)
	months, _ = referral(35)
	assert.Equal(t, 6, months)
}

func TestTTC_NoLogsAndEmpty(t *testing.T) {
	r := BuildTTC(ttcNoLogsFixture())
	assert.False(t, r.Trying)
	assert.Len(t, r.Cycles, 4) // 3 completed + current (fallback ≤ 6)
	assert.Nil(t, get(t, r.JSON(), "trying", "since"))

	e := BuildTTC(ttcEmptyFixture())
	assert.Empty(t, e.Cycles)
	_, ok := e.FindCycle(civildate.Date{})
	assert.False(t, ok)
	assert.Equal(t, false, get(t, e.JSON(), "bbt", "ready"))
}

func TestTTC_SignalMerge(t *testing.T) {
	s := NewTTCSignals()
	d := civildate.MustParse("2026-09-01")
	s.AddFertilityLog(d, "positive", "creamy")
	s.AddEntries([]DayEntries{{Date: d, Entries: []taxonomy.Entry{
		single("measurements", "lh_test", "negative"), single("discharge", "consistency", "egg_white"),
	}}})
	assert.Equal(t, "positive", s.LH[d])
	assert.Equal(t, "egg_white", s.Mucus[d])
	s.AddEntries([]DayEntries{{Date: d.AddDays(1), Entries: []taxonomy.Entry{single("discharge", "consistency", "none")}}})
	assert.Equal(t, "dry", s.Mucus[d.AddDays(1)])
	s.AddFertilityLog(d.AddDays(2), "bogus", "")
	_, ok := s.LH[d.AddDays(2)]
	assert.False(t, ok)
}

func TestTTC_LutealStatus(t *testing.T) {
	assert.Equal(t, LutealShort, LutealStatus(10))
	assert.Equal(t, LutealNormal, LutealStatus(11))
	assert.Equal(t, LutealNormal, LutealStatus(17))
	assert.Equal(t, LutealLong, LutealStatus(18))
}

func TestTTC_MonthsBetween(t *testing.T) {
	assert.Equal(t, 0, monthsBetween(civildate.MustParse("2026-09-10"), civildate.MustParse("2026-10-09")))
	assert.Equal(t, 1, monthsBetween(civildate.MustParse("2026-09-10"), civildate.MustParse("2026-10-10")))
	assert.Equal(t, 12, monthsBetween(civildate.MustParse("2025-09-10"), civildate.MustParse("2026-09-23")))
}
