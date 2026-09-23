package home

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// fakeSection is a section whose Build is scripted.
type fakeSection struct {
	key   string
	order int
	build func() (*Rendered, error)
}

func (f fakeSection) Key() string                       { return f.key }
func (f fakeSection) Order() int                        { return f.order }
func (f fakeSection) Supports(*Context) bool            { return true }
func (f fakeSection) Build(*Context) (*Rendered, error) { return f.build() }
func okSection(key string, order int) fakeSection {
	return fakeSection{key, order, func() (*Rendered, error) {
		return &Rendered{Key: key, Type: key, Order: order, Data: jsonx.Obj("ok", true)}, nil
	}}
}

// HomePageService::safeBuild: a section that throws is logged and left out; the page still builds.
func TestFailingSectionIsOmitted(t *testing.T) {
	var logs bytes.Buffer
	page := NewPage(slog.New(slog.NewJSONHandler(&logs, nil)),
		okSection("late", 30),
		fakeSection{"broken", 20, func() (*Rendered, error) { return nil, errors.New("db down") }},
		fakeSection{"panics", 15, func() (*Rendered, error) { panic("nil map") }},
		fakeSection{"empty", 5, func() (*Rendered, error) { return nil, nil }},
		okSection("early", 10),
	)
	hc := newContext(context.Background(), &Deps{})
	hc.UserID = 42

	got := page.Build(hc)
	keys := make([]string, len(got))
	for i, s := range got {
		k, _ := s.Get("key")
		keys[i] = k.(string)
		meta, ok := s.Get("meta")
		assert.True(t, ok)
		assert.Nil(t, meta, "meta is null when empty")
	}
	assert.Equal(t, []string{"early", "late"}, keys, "failed / empty sections omitted, rest sorted by order")
	assert.Contains(t, logs.String(), `"section":"broken"`)
	assert.Contains(t, logs.String(), `"section":"panics"`)
	assert.Contains(t, logs.String(), "HomePage section failed to build")

	assert.Nil(t, page.BuildSection("broken", hc))
	assert.Nil(t, page.BuildSection("panics", hc))
	assert.NotNil(t, page.BuildSection("late", hc))
	assert.Equal(t, []string{"late", "broken", "panics", "empty", "early"}, page.Keys())
}

func TestDefaultSectionsOrderAndKeys(t *testing.T) {
	p := NewPage(nil)
	assert.Equal(t, []string{"header", "week_calendar", "next_period", "cycle_prediction", "recommendations",
		"tasks", "challenge", "doctor_reminder", "medication_reminder", "smart_tip", "affirmation",
		"weekly_summary", "vitals", "status_charts", "articles", "my_cycles", "cycle_summary"}, p.Keys())
	cycleOnlyKeys := map[string]bool{"week_calendar": true, "next_period": true, "cycle_prediction": true,
		"recommendations": true, "my_cycles": true, "cycle_summary": true}
	preg := &Context{Mode: "pregnancy"}
	prev := 0
	for _, s := range DefaultSections() {
		assert.Greater(t, s.Order(), prev)
		prev = s.Order()
		assert.Equal(t, !cycleOnlyKeys[s.Key()], s.Supports(preg), s.Key())
	}
}

func TestMoonPhase(t *testing.T) {
	m, err := MoonPhaseFor(civildate.MustParse("2026-09-23"), "fa").MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"key":"waxing_gibbous","label":"محدب افزاینده","illumination":0.85,"age_days":11}`, string(m))
	m, err = MoonPhaseFor(civildate.MustParse("2026-09-23"), "ar").MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(m), `"Waxing gibbous"`)
}

func TestNum(t *testing.T) {
	fa := &Context{Locale: "fa"}
	assert.Equal(t, "۱۲", fa.Num(12))
	assert.Equal(t, "۱۲.۵", fa.Num(12.5))
	assert.Equal(t, "۱۲", fa.Num(12.0))
	assert.Equal(t, "12.5", (&Context{Locale: "en"}).Num(12.5))
}

func TestDigest(t *testing.T) {
	d := civildate.MustParse
	dg := NewDigest([]model.History{
		{ID: 3, PeriodStart: d("2026-09-17"), PeriodEnd: d("2026-09-21"), IsConfirmed: true, Source: "user_logged"},
		{ID: 1, PeriodStart: d("2026-06-26"), PeriodEnd: d("2026-06-30"), IsConfirmed: true, Source: "user_logged"},
		{ID: 2, PeriodStart: d("2026-07-23"), IsConfirmed: true, Source: "user_logged"},
	})
	require.Len(t, dg.cycles, 3)
	assert.Equal(t, int64(3), dg.cycles[0].ID)
	assert.True(t, dg.cycles[0].IsCurrent)
	assert.Nil(t, dg.cycles[0].CycleLength)
	assert.Equal(t, 56, *dg.cycles[1].CycleLength)
	assert.Equal(t, 27, *dg.cycles[2].CycleLength)
	assert.Equal(t, 56, *dg.LastCycleLength())
	assert.Equal(t, 5, *dg.LastPeriodLength())
	assert.Equal(t, []int{27}, dg.ValidCycleLengths())
	assert.Len(t, dg.Previous(-1), 2)
	assert.Len(t, dg.Previous(1), 1)
	cur, ok := dg.StartingOn(d("2026-09-17"))
	assert.True(t, ok)
	assert.Equal(t, 5, *cur.PeriodLength)
	assert.Equal(t, 28, *averageOrNil([]int{27, 28, 28}))
	assert.Equal(t, 29, *averageOrNil([]int{28, 29})) // round half away from zero
}

func TestScorer(t *testing.T) {
	s := func(v string) *string { return &v }
	tr := true
	avg := weeklyAverages([]scoredLog{
		{Moods: []string{"happy", "sad", "unknown"}, SleepQuality: s("good"), SleepDuration: s("3_6"), EnergyLevel: s("high")},
		{Fatigue: &tr},
		{},
	})
	assert.Equal(t, 60, *avg[0]) // (100+20)/2
	assert.Equal(t, 78, *avg[1]) // round((100+55)/2 = 77.5)
	assert.Equal(t, 55, *avg[2]) // (80+30)/2
	assert.Nil(t, weeklyAverages(nil)[0])
}

func TestMysqlKeyAndIntCast(t *testing.T) {
	for raw, want := range map[string]uint64{"99": 99, "12abc": 12, "1.0": 1, "1e2": 100} {
		got, ok := mysqlKey(raw)
		assert.True(t, ok, raw)
		assert.Equal(t, want, got, raw)
	}
	for _, raw := range []string{"abc", "", "0", "-3", "1.5", "0x10"} {
		_, ok := mysqlKey(raw)
		assert.False(t, ok, raw)
	}
	assert.Equal(t, int64(0), phpIntCast("abc"))
	assert.Equal(t, int64(500), phpIntCast("500"))
	assert.Equal(t, int64(12), phpIntCast("12abc"))
	assert.Equal(t, int64(1), phpIntCast("1.9"))
}

func TestChallengeSeedIsPHPCrc32(t *testing.T) {
	// php -r 'echo crc32("1018:2026-09-23");'
	assert.Equal(t, uint32(2956981644), challengeSeed(1018, civildate.MustParse("2026-09-23")))
}
