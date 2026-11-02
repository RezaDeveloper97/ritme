package inputs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
)

func fullInput() engine.Input {
	return engine.Input{
		Periods: []engine.Period{
			{Start: day("2026-08-01"), End: day("2026-08-05"), Source: engine.PeriodUserLogged},
			{Start: day("2026-08-29"), End: day("2026-09-02"), Source: engine.PeriodBleedingLog},
			{Start: day("2026-09-27"), End: engine.NoDay, Source: engine.PeriodUserLogged},
		},
		Skips: []engine.Day{day("2026-07-01")},
		Declared: engine.Declared{
			LastPeriodStart: engine.Some(day("2026-06-01")), CycleLength: engine.Some(29),
			PeriodLength: engine.Some(5), Manual: true,
		},
		BirthDate:      engine.Some(day("1995-03-10")),
		TeenBandOnly:   true,
		Mode:           engine.ModePostpartum,
		Hormonal:       true,
		HormonalFrom:   engine.Some(day("2026-09-27")),
		DeliveryDate:   engine.Some(day("2026-05-01")),
		Breastfeeding:  true,
		LH:             []engine.LHTest{{Date: day("2026-10-10"), Positive: true}},
		BBT:            []engine.BBTReading{{Date: day("2026-10-01"), Centi: 3645}, {Date: day("2026-10-02"), Centi: 3700}},
		PregnancyTests: []engine.PregnancyTest{{Date: day("2026-10-20"), Positive: true}},
	}
}

func TestJSON_RoundTrip(t *testing.T) {
	for name, in := range map[string]engine.Input{"full": fullInput(), "zero": {}} {
		t.Run(name, func(t *testing.T) {
			b, err := Encode(in)
			require.NoError(t, err)
			got, err := Decode(b)
			require.NoError(t, err)
			assert.Equal(t, in, got)
		})
	}
}

func TestJSON_StableShape(t *testing.T) {
	b, err := Encode(engine.Input{Periods: []engine.Period{{Start: day("2026-09-27"), End: engine.NoDay}}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"v":2,"periods":[{"start":"2026-09-27","end":null,"source":"user_logged"}],
		"declared":{"last_period_start":null,"cycle_length":null,"period_length":null,"manual":false},
		"birth_date":null,"teen_band_only":false,"mode":"cycle","hormonal":false,"hormonal_from":null,"delivery_date":null,
		"breastfeeding":false}`, string(b))
}

func TestJSON_UnknownVersionForcesReload(t *testing.T) {
	for _, s := range []string{`{"v":1,"mode":"cycle"}`, `{"v":3,"mode":"cycle"}`, `{"mode":"cycle"}`, `{"v":0}`} {
		_, err := Decode([]byte(s))
		require.ErrorIs(t, err, ErrVersion, s)
	}
}

func TestJSON_MalformedFails(t *testing.T) {
	for _, s := range []string{
		`not json`,
		`{"v":2,"mode":"sleeping"}`,
		`{"v":2,"mode":"cycle","periods":[{"start":"2026-02-30","end":null,"source":"user_logged"}]}`,
		`{"v":2,"mode":"cycle","periods":[{"start":"2026-02-01","end":null,"source":"guess"}]}`,
		`{"v":2,"mode":"cycle","birth_date":"yesterday"}`,
		`{"v":2,"mode":"cycle","hormonal":true,"hormonal_from":"2026-13-01"}`,
	} {
		_, err := Decode([]byte(s))
		require.Error(t, err, s)
		assert.NotErrorIs(t, err, ErrVersion, s)
	}
}
