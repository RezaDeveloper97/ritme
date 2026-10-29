package analysis

// Postpartum analysis hub (T-M10-27): unit tests of BuildPostpartum on a fixed day (2026-09-23, birth 6 weeks before).

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/babylog"
	childrenstore "github.com/ritme/backend-go/internal/children/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/postpartum"
)

var ppBirth = civildate.MustParse("2026-08-12")

func ppEmpty(deep bool) *PostpartumInput {
	return &PostpartumInput{
		Today: fixtureToday, Birth: ppBirth, PregnancyStart: ppBirth.AddDays(-PostpartumPregnancyDays), DeepAnalysis: deep,
		Recovery: map[civildate.Date]postpartum.Recovery{}, Weights: map[civildate.Date]float64{}, ChildrenEnabled: true,
	}
}

func ppIntp(n int) *int           { return &n }
func ppFloatp(f float64) *float64 { return &f }

// ppFull: two full EPDS checks (11 then 8), a decreasing lochia over 3 weeks, feeds and sleep every day of the last
// week, a placed child, a pre-pregnancy weight and three weights since the birth.
func ppFull(deep bool) *PostpartumInput {
	in := ppEmpty(deep)
	in.Checks = []postpartum.Check{ // newest first
		{ID: 2, Kind: postpartum.KindFull, TakenOn: fixtureToday.AddDays(-2), Total: 8},
		{ID: 1, Kind: postpartum.KindShort, TakenOn: fixtureToday.AddDays(-9), Total: 3},
		{ID: 0, Kind: postpartum.KindFull, TakenOn: fixtureToday.AddDays(-16), Total: 11},
	}
	amounts := []string{"heavy", "medium", "medium", "light", "light", "spotting", "spotting", "none"}
	for i, a := range amounts {
		d := fixtureToday.AddDays(-20 + i*2)
		in.Recovery[d] = postpartum.Recovery{LochiaAmount: a}
	}
	for i := range 7 {
		d := fixtureToday.AddDays(-i)
		r := in.Recovery[d]
		r.SleepHours = ppFloatp(5 + float64(i%2))
		r.FeedsCount = ppIntp(8)
		in.Recovery[d] = r
		in.BabyDays = append([]babylog.Day{{Date: d,
			Feeds: babylog.FeedTotals{Count: 9, Breast: 9, LeftSeconds: 600, RightSeconds: 400},
			Sleep: babylog.SleepTotals{Count: 5, Seconds: 14 * 3600}}}, in.BabyDays...)
	}
	in.Child = &PostpartumChild{ID: 7, Name: "Sara", Placed: true, Crossings: []any{}}
	in.Weights[in.PregnancyStart.AddDays(-30)] = 60
	in.Weights[ppBirth.AddDays(3)] = 70
	in.Weights[ppBirth.AddDays(20)] = 67.5
	in.Weights[fixtureToday.AddDays(-1)] = 66.2
	return in
}

func ppJSON(t *testing.T, in *PostpartumInput) map[string]any {
	t.Helper()
	raw, err := json.Marshal(BuildPostpartum(in))
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	return m
}

func ppCard(m map[string]any, key string) map[string]any {
	c, _ := m["cards"].(map[string]any)[key].(map[string]any)
	return c
}

func ppData(m map[string]any, key string) map[string]any {
	d, _ := ppCard(m, key)["data"].(map[string]any)
	return d
}

func TestPostpartum_EveryCardReady(t *testing.T) {
	m := ppJSON(t, ppFull(true))
	assert.Equal(t, "2026-08-12", m["birth_date"])
	assert.InDelta(t, 42, m["days_since_birth"], 0)
	assert.InDelta(t, 7, m["week"], 0)
	for _, k := range []string{"epds", "bleeding", "breastfeeding", "sleep", "growth", "weight"} {
		c := ppCard(m, k)
		assert.Equal(t, true, c["ready"], k)
		assert.Equal(t, false, c["locked"], k)
		assert.InDelta(t, 0, c["days_needed"], 0, k)
	}

	epds := ppData(m, "epds")
	assert.Equal(t, "full", epds["kind"], "the full EPDS is plotted when there is one")
	assert.Len(t, epds["points"], 2)
	assert.Equal(t, map[string]any{"ready": true, "direction": "down"}, epds["trend"])
	latest := epds["latest"].(map[string]any)
	assert.InDelta(t, 8, latest["total"], 0)
	assert.InDelta(t, 30, latest["max"], 0)
	assert.Equal(t, "low", latest["band"])
	assert.InDelta(t, 6, latest["week"], 0) // day 40 after the birth
	for _, p := range epds["points"].([]any) {
		for k := range p.(map[string]any) {
			assert.Contains(t, []string{"kind", "taken_on", "week", "total", "max", "band", "urgent"}, k,
				"only totals and bands leave the server (D-54)")
		}
	}

	b := ppData(m, "bleeding")
	assert.Equal(t, "decreasing", b["trend"])
	assert.InDelta(t, 8, b["logged_days"], 0)
	assert.Len(t, b["cells"], PostpartumBleedingWindowDays)
	assert.Equal(t, false, b["alert_today"])

	f := ppData(m, "breastfeeding")
	assert.Equal(t, "baby_log", f["source"])
	assert.InDelta(t, 9, f["feeds_per_day"], 0)
	assert.InDelta(t, 60, f["left_percent"], 0)
	assert.InDelta(t, 40, f["right_percent"], 0)
	assert.Nil(t, f["breastfeeding"])

	s := ppData(m, "sleep")
	assert.InDelta(t, 5.4, s["mother"].(map[string]any)["avg_hours"], 0.001)
	assert.InDelta(t, 14*3600, s["baby"].(map[string]any)["avg_seconds"], 0)
	assert.Equal(t, "Sara", s["baby"].(map[string]any)["name"])

	w := ppData(m, "weight")
	assert.InDelta(t, 3, w["logs"], 0)
	assert.InDelta(t, 60, w["pre_pregnancy"].(map[string]any)["kg"], 0)
	assert.Equal(t, "before_pregnancy", w["pre_pregnancy"].(map[string]any)["source"])
	assert.InDelta(t, 6.2, w["delta_vs_pre_pregnancy"], 0.001)
	assert.InDelta(t, -3.8, w["delta_since_first"], 0.001)
	assert.InDelta(t, 1, w["from_week"], 0)
}

func TestPostpartum_EPDSMinimums(t *testing.T) {
	in := ppEmpty(false)
	m := ppJSON(t, in)
	e := ppCard(m, "epds")
	assert.Equal(t, false, e["ready"])
	assert.Equal(t, false, e["plus"], "the EPDS card is never Plus (§4.3)")
	assert.Nil(t, ppData(m, "epds")["latest"])

	in.Checks = []postpartum.Check{{Kind: postpartum.KindFull, TakenOn: fixtureToday, Total: 12}}
	m = ppJSON(t, in)
	assert.Equal(t, true, ppCard(m, "epds")["ready"], "one score shows (P.EPDSMinScores)")
	assert.Equal(t, map[string]any{"ready": false, "direction": nil}, ppData(m, "epds")["trend"],
		"no trend with one score (P.EPDSTrendMinScores)")
	assert.Equal(t, "possible", ppData(m, "epds")["latest"].(map[string]any)["band"])

	// Short checks only: the EPDS-3 is plotted.
	in.Checks = []postpartum.Check{
		{Kind: postpartum.KindShort, TakenOn: fixtureToday, Total: 6},
		{Kind: postpartum.KindShort, TakenOn: fixtureToday.AddDays(-7), Total: 2},
	}
	d := ppData(ppJSON(t, in), "epds")
	assert.Equal(t, "short", d["kind"])
	assert.Equal(t, "up", d["trend"].(map[string]any)["direction"])
	assert.Equal(t, "elevated", d["latest"].(map[string]any)["band"])
}

func TestPostpartum_FreeUserPlusCardsLocked(t *testing.T) {
	m := ppJSON(t, ppFull(false))
	for _, k := range []string{"sleep", "weight"} {
		c := ppCard(m, k)
		assert.Equal(t, true, c["plus"], k)
		assert.Equal(t, true, c["locked"], k)
		assert.Nil(t, c["data"], k)
		assert.Nil(t, c["days_needed"], k)
	}
	for _, k := range []string{"epds", "bleeding", "breastfeeding", "growth"} {
		assert.Equal(t, false, ppCard(m, k)["plus"], k)
		assert.Equal(t, true, ppCard(m, k)["ready"], k)
	}
}

func TestPostpartum_NotEnoughData(t *testing.T) {
	in := ppEmpty(true)
	in.Recovery[fixtureToday] = postpartum.Recovery{LochiaAmount: "light"}
	in.Weights[fixtureToday] = 66
	m := ppJSON(t, in)
	assert.InDelta(t, 3, ppCard(m, "bleeding")["days_needed"], 0)
	assert.Nil(t, ppData(m, "bleeding")["trend"])
	assert.InDelta(t, PostpartumFeedsMinDays, ppCard(m, "breastfeeding")["days_needed"], 0)
	assert.Equal(t, "recovery", ppData(m, "breastfeeding")["source"], "no child: the mother's feeds slot")
	assert.InDelta(t, PostpartumSleepMinDays, ppCard(m, "sleep")["days_needed"], 0)
	assert.InDelta(t, 1, ppCard(m, "weight")["days_needed"], 0)
	assert.Nil(t, ppData(m, "weight")["delta_since_first"])
	assert.Nil(t, ppData(m, "weight")["pre_pregnancy"])
	assert.Equal(t, false, ppCard(m, "growth")["ready"])
	assert.Equal(t, true, ppData(m, "growth")["available"])

	in.ChildrenEnabled = false
	assert.Equal(t, false, ppData(ppJSON(t, in), "growth")["available"], "children module off")
}

func TestPostpartum_BleedingRedFlagTodayAndWindow(t *testing.T) {
	in := ppEmpty(false)
	yes := true
	in.Recovery[fixtureToday] = postpartum.Recovery{LochiaAmount: "medium", PadSoakedWithinHour: &yes}
	in.Recovery[fixtureToday.AddDays(-3)] = postpartum.Recovery{LochiaAmount: "light", LargeClots: true}
	d := ppData(ppJSON(t, in), "bleeding")
	assert.Equal(t, true, d["alert_today"], "a pad soaked within an hour today points to the urgent path")
	assert.Equal(t, true, d["large_clots"])

	// A few days after the birth the window starts at the birth.
	in = ppEmpty(false)
	in.Birth = fixtureToday.AddDays(-4)
	d = ppData(ppJSON(t, in), "bleeding")
	assert.Len(t, d["cells"], 5)
	assert.InDelta(t, 0, d["cells"].([]any)[0].(map[string]any)["day"], 0)
	assert.Len(t, ppData(ppJSON(t, in), "breastfeeding")["days"], 5)
}

func TestHubChild_ClosestToBirth(t *testing.T) {
	assert.Nil(t, hubChild(nil, ppBirth))
	kids := []childrenstore.Child{
		{ID: 1, BirthDate: civildate.MustParse("2024-02-01")},
		{ID: 2, BirthDate: ppBirth.AddDays(1)},
		{ID: 3, BirthDate: ppBirth.AddDays(1)}, // twin: the first listed wins
	}
	assert.Equal(t, uint64(2), hubChild(kids, ppBirth).ID)
}
