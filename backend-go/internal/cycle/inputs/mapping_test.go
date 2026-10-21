package inputs

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/contraception"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func day(s string) engine.Day { return engine.DayOf(civildate.MustParse(s)) }

func nd(s string) civildate.NullDate {
	if s == "" {
		return civildate.NullDate{}
	}
	return civildate.NullDate{Date: civildate.MustParse(s), Valid: true}
}

func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }

func TestHormonal_EveryMethod(t *testing.T) {
	// I-03: the full method table.
	for method, want := range map[string]bool{
		contraception.MethodCombinedPill:  true,
		contraception.MethodProgestinPill: true,
		contraception.MethodHormonalIUD:   true,
		contraception.MethodInjection:     true,
		contraception.MethodImplant:       true,
		contraception.MethodCopperIUD:     false,
		contraception.MethodCondom:        false,
		contraception.MethodOther:         false,
		"":                                false,
		"unknown":                         false,
	} {
		assert.Equal(t, want, contraception.IsHormonal(method), method)
	}

	// Onboarding: contraceptive_pill → hormonal; iud (type unknown) and hormonal_medication → not.
	assert.True(t, contraception.HormonalFromOnboarding([]string{"contraceptive_pill"}))
	assert.True(t, contraception.HormonalFromOnboarding([]string{"iud", "contraceptive_pill"}))
	assert.False(t, contraception.HormonalFromOnboarding([]string{"iud"}))
	assert.False(t, contraception.HormonalFromOnboarding([]string{"hormonal_medication"}))
	assert.False(t, contraception.HormonalFromOnboarding(nil))

	// A saved method decides alone; without one the onboarding answer decides.
	assert.False(t, contraception.HormonalActive("copper_iud", true, []string{"contraceptive_pill"}))
	assert.True(t, contraception.HormonalActive("implant", true, []string{"iud"}))
	assert.True(t, contraception.HormonalActive("", false, []string{"contraceptive_pill"}))
	assert.False(t, contraception.HormonalActive("", false, []string{"iud"}))
}

func TestParseCenti(t *testing.T) {
	for in, want := range map[string]int32{
		"36.55": 3655, "36.5": 3650, "37": 3700, "37.": 3700, "36.554": 3655, "36.555": 3656, "36.999": 3700,
		"36.40": 3640, "0.00": 0,
	} {
		got, ok := parseCenti(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	for _, in := range []string{"", ".5", "-36.5", "36,5", "abc", "36.5x", "12345.0"} {
		_, ok := parseCenti(in)
		assert.False(t, ok, in)
	}
}

func marker(date, kind, code, num string) store.ListEngineInputMarkersRow {
	r := store.ListEngineInputMarkersRow{UserID: 1, LogDate: civildate.MustParse(date), Kind: kind}
	if code != "" {
		r.ValueCode = ns(code)
	}
	if num != "" {
		r.ValueNum = ns(num)
	}
	return r
}

func TestMapMarkers_BBT(t *testing.T) {
	rows := []store.ListEngineInputMarkersRow{
		marker("2026-10-01", kindLegacyBBT, "", "36.40"),
		marker("2026-10-01", kindBBT, "", "36.45"), // v2 wins over the legacy column
		marker("2026-10-02", kindLegacyBBT, "", "36.50"),
		marker("2026-10-03", kindBBT, "", "33.99"), // out of range
		marker("2026-10-04", kindBBT, "", "34.00"), // bounds are inclusive
		marker("2026-10-05", kindBBT, "", "43.00"),
		marker("2026-10-06", kindBBT, "", "43.01"),
		marker("2026-10-07", kindBBT, "", "98.60"), // Fahrenheit typo
		marker("2026-10-08", kindLegacyBBT, "", "36.70"),
		marker("2026-10-08", kindBBT, "", "99.00"), // the chosen (v2) reading is out of range: the day is dropped
		marker("2026-09-01", kindBBT, "", "36.60"), // before the window
		marker("2026-10-23", kindBBT, "", "36.60"), // after today + 1
	}
	_, bbt, _ := mapMarkers(rows, day("2026-09-02"), day("2026-10-22"))
	assert.Equal(t, []engine.BBTReading{
		{Date: day("2026-10-01"), Centi: 3645},
		{Date: day("2026-10-02"), Centi: 3650},
		{Date: day("2026-10-04"), Centi: 3400},
		{Date: day("2026-10-05"), Centi: 4300},
	}, bbt)
}

func TestMapMarkers_LHAndPregnancyTests(t *testing.T) {
	rows := []store.ListEngineInputMarkersRow{
		marker("2026-10-10", kindLH, "positive", ""),
		marker("2026-10-10", kindFertilityLH, "positive", ""), // the same day in both logs: one test
		marker("2026-10-11", kindFertilityLH, "faint", ""),    // faint is not positive
		marker("2026-10-12", kindLH, "negative", ""),
		marker("2026-10-09", kindFertilityLH, "positive", ""),
		marker("2026-10-22", kindLH, "positive", ""), // today + 1 is kept
		marker("2026-10-23", kindLH, "positive", ""), // later is dropped
		marker("2026-10-15", kindPregnancy, "negative", ""),
		marker("2026-10-20", kindPregnancy, "positive", ""),
		marker("2026-10-20", kindPregnancy, "positive", ""),
	}
	lh, bbt, preg := mapMarkers(rows, day("2026-01-01"), day("2026-10-22"))
	assert.Equal(t, []engine.LHTest{
		{Date: day("2026-10-09"), Positive: true},
		{Date: day("2026-10-10"), Positive: true},
		{Date: day("2026-10-22"), Positive: true},
	}, lh)
	assert.Nil(t, bbt)
	assert.Equal(t, []engine.PregnancyTest{{Date: day("2026-10-20"), Positive: true}}, preg)

	lh, bbt, preg = mapMarkers(nil, 0, 1)
	assert.Nil(t, lh)
	assert.Nil(t, bbt)
	assert.Nil(t, preg)
}

func TestMapMode_Precedence(t *testing.T) {
	cases := []struct {
		stored   string
		pregnant bool
		goal     string
		want     engine.Mode
	}{
		{"postpartum", true, "ttc", engine.ModePregnancy}, // an active pregnancy wins
		{"", true, "", engine.ModePregnancy},
		{"postpartum", false, "ttc", engine.ModePostpartum},
		{"menopause", false, "", engine.ModeMenopause},
		{"teen", false, "ttc", engine.ModeTeen},
		{"ttc", false, "ttc", engine.ModeTTC},
		{"", false, "ttc", engine.ModeTTC},
		{"cycle", false, "track_cycle", engine.ModeCycle},
		{"", false, "", engine.ModeCycle},
		{"garbage", false, "", engine.ModeCycle},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, mapMode(c.stored, c.pregnant, c.goal), "%+v", c)
	}
}

func TestMapProfile(t *testing.T) {
	t.Run("everything", func(t *testing.T) {
		in := mapProfile(store.ListEngineInputProfilesRow{
			UserID: 1, Birthday: nd("1995-03-10"), PeriodDuration: sql.NullInt16{Int16: 5, Valid: true},
			CycleDuration: sql.NullInt16{Int16: 29, Valid: true}, OnboardingLastPeriodStart: nd("2026-09-01"),
			UserGoal: ns("track_cycle"), LengthsAuto: false, LifeMode: ns("postpartum"),
			Medications:     db.NullRawJSON{V: json.RawMessage(`["contraceptive_pill"]`), Valid: true},
			DeliveryDate:    nd("2026-06-01"),
			Breastfeeding:   sql.NullBool{Bool: true, Valid: true},
			AgeBand:         ns("16_17"),
			PregnancyActive: 0,
		})
		assert.Equal(t, engine.Input{
			Declared: engine.Declared{
				LastPeriodStart: engine.Some(day("2026-09-01")), CycleLength: engine.Some(29),
				PeriodLength: engine.Some(5), Manual: true,
			},
			BirthDate:     engine.Some(day("1995-03-10")),
			TeenBandOnly:  false, // a birthday is known
			Mode:          engine.ModePostpartum,
			Hormonal:      true, // onboarding pill, no saved method
			DeliveryDate:  engine.Some(day("2026-06-01")),
			Breastfeeding: true,
		}, in)
	})
	t.Run("nothing", func(t *testing.T) {
		in := mapProfile(store.ListEngineInputProfilesRow{UserID: 1, LengthsAuto: true})
		assert.Equal(t, engine.Input{}, in)
	})
	t.Run("teen band without birthday", func(t *testing.T) {
		in := mapProfile(store.ListEngineInputProfilesRow{UserID: 1, LengthsAuto: true, AgeBand: ns("13_15")})
		assert.True(t, in.TeenBandOnly)
		assert.False(t, in.BirthDate.OK())
	})
	t.Run("breastfeeding not asked, malformed medications, zero lengths", func(t *testing.T) {
		in := mapProfile(store.ListEngineInputProfilesRow{
			UserID: 1, LengthsAuto: true, Breastfeeding: sql.NullBool{},
			Medications:   db.NullRawJSON{V: json.RawMessage(`{"x":1}`), Valid: true},
			CycleDuration: sql.NullInt16{Int16: 0, Valid: true},
		})
		assert.False(t, in.Breastfeeding)
		assert.False(t, in.Hormonal)
		assert.False(t, in.Declared.CycleLength.OK())
	})
	t.Run("saved copper IUD beats onboarding pill", func(t *testing.T) {
		in := mapProfile(store.ListEngineInputProfilesRow{
			UserID: 1, LengthsAuto: true, ContraceptionMethod: ns("copper_iud"),
			Medications: db.NullRawJSON{V: json.RawMessage(`["contraceptive_pill"]`), Valid: true},
		})
		assert.False(t, in.Hormonal)
	})
}

func period(start, end, source string, confirmed, estimated bool) store.ListEngineInputPeriodsRow {
	return store.ListEngineInputPeriodsRow{
		UserID: 1, PeriodStartDate: civildate.MustParse(start), PeriodEndDate: nd(end), Source: source,
		IsConfirmed: confirmed, IsEstimated: estimated,
	}
}

func TestMapPeriods(t *testing.T) {
	periods, seed := mapPeriods([]store.ListEngineInputPeriodsRow{
		period("2026-07-01", "2026-07-05", "user_logged", true, false),
		period("2026-05-01", "2026-05-04", "user_profile_confirmed", true, false), // older seed
		period("2026-06-02", "2026-06-06", "user_profile_confirmed", true, false), // newest seed → anchor
		period("2026-08-01", "2026-08-04", "bleeding_log", true, false),
		period("2026-08-29", "2026-09-02", "user_logged", false, false), // unconfirmed user_logged is actual (I-02)
		period("2026-09-15", "", "onboarding_estimate", false, true),    // estimated: ignored
		period("2026-09-27", "", "user_logged", true, false),            // open
	})
	assert.Equal(t, []engine.Period{
		{Start: day("2026-07-01"), End: day("2026-07-05"), Source: engine.PeriodUserLogged},
		{Start: day("2026-08-01"), End: day("2026-08-04"), Source: engine.PeriodBleedingLog},
		{Start: day("2026-08-29"), End: day("2026-09-02"), Source: engine.PeriodUserLogged},
		{Start: day("2026-09-27"), End: engine.NoDay, Source: engine.PeriodUserLogged},
	}, periods)
	assert.True(t, periods[3].Open())
	assert.Equal(t, engine.Some(day("2026-06-02")), seed)

	periods, seed = mapPeriods(nil)
	assert.Nil(t, periods)
	assert.False(t, seed.OK())
}

func TestMarkerWindowFrom(t *testing.T) {
	today := day("2026-10-21")
	assert.Equal(t, today-400, markerWindowFrom(nil, today))

	var ps []engine.Period
	for i := range 10 { // monthly periods, the newest 2026-10-12
		ps = append(ps, engine.Period{Start: today - engine.Day(9+28*(9-i)), End: engine.NoDay})
	}
	// 7th newest = ps[3]: today − 9 − 168 = 177 days back; 400 days back is earlier.
	assert.Equal(t, today-400, markerWindowFrom(ps, today))

	var long []engine.Period // 7th newest is 9 + 6·90 = 549 days back: earlier than 400
	for i := range 8 {
		long = append(long, engine.Period{Start: today - engine.Day(9+90*(7-i)), End: engine.NoDay})
	}
	assert.Equal(t, long[1].Start, markerWindowFrom(long, today))

	// Fewer than 7: the oldest period counts.
	short := []engine.Period{{Start: today - 500}, {Start: today - 30}}
	assert.Equal(t, today-500, markerWindowFrom(short, today))
}

// fakeQ counts calls and serves fixed rows for any id list.
type fakeQ struct {
	calls    int
	batches  [][]uint64
	profiles func(ids []uint64) []store.ListEngineInputProfilesRow
}

func (f *fakeQ) ListEngineInputProfiles(_ context.Context, ids []uint64) ([]store.ListEngineInputProfilesRow, error) {
	f.calls++
	f.batches = append(f.batches, ids)
	if f.profiles == nil {
		return nil, nil
	}
	return f.profiles(ids), nil
}

func (f *fakeQ) ListEngineInputPeriods(context.Context, []uint64) ([]store.ListEngineInputPeriodsRow, error) {
	f.calls++
	return nil, nil
}

func (f *fakeQ) ListEngineInputSkips(context.Context, []uint64) ([]store.ListEngineInputSkipsRow, error) {
	f.calls++
	return nil, nil
}

func (f *fakeQ) ListEngineInputMarkers(context.Context, store.ListEngineInputMarkersParams) ([]store.ListEngineInputMarkersRow, error) {
	f.calls++
	return nil, nil
}

func TestLoad_UnknownUserIsZeroInOneQuery(t *testing.T) {
	f := &fakeQ{}
	in, err := Load(context.Background(), f, 42, civildate.New(2026, 10, 21))
	require.NoError(t, err)
	assert.Equal(t, engine.Input{}, in)
	assert.Equal(t, 1, f.calls)
}

func TestLoadMany_BatchesOf500(t *testing.T) {
	f := &fakeQ{profiles: func(ids []uint64) []store.ListEngineInputProfilesRow {
		rows := make([]store.ListEngineInputProfilesRow, 0, len(ids))
		for _, id := range ids {
			rows = append(rows, store.ListEngineInputProfilesRow{UserID: id, LengthsAuto: true})
		}
		return rows
	}}
	ids := make([]uint64, 0, 1002)
	for i := range 1001 {
		ids = append(ids, uint64(i+1)) //nolint:gosec // small test ids
	}
	ids = append(ids, 7) // a duplicate collapses
	m, err := LoadMany(context.Background(), f, ids, civildate.New(2026, 10, 21))
	require.NoError(t, err)
	assert.Len(t, m, 1001)
	require.Len(t, f.batches, 3)
	assert.Len(t, f.batches[0], 500)
	assert.Len(t, f.batches[1], 500)
	assert.Len(t, f.batches[2], 1)
	assert.Equal(t, 12, f.calls, "4 queries per batch")
}
