package skips

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func day(s string) engine.Day { return engine.DayOf(civildate.MustParse(s)) }

func period(start, end string) engine.Period {
	return engine.Period{Start: day(start), End: day(end), Source: engine.PeriodUserLogged}
}

// Three 28-day cycles: the last start is 08-26, so the next one is predicted on 09-23.
func history() engine.Input {
	return engine.Input{Periods: []engine.Period{
		period("2026-07-01", "2026-07-05"), period("2026-07-29", "2026-08-02"), period("2026-08-26", "2026-08-30"),
	}}
}

func TestCheck(t *testing.T) {
	p := engine.DefaultParams()
	started := history()
	started.Periods = append(started.Periods, engine.Period{Start: day("2026-09-25"), End: engine.NoDay})
	pregnant := history()
	pregnant.Mode = engine.ModePregnancy

	cases := []struct {
		name            string
		in              engine.Input
		expected, today string
		want            string
	}{
		{"on the expected day", history(), "2026-09-23", "2026-09-23", ""},
		{"1 day late", history(), "2026-09-23", "2026-09-24", ""},
		{"15 days late (beyond max_late)", history(), "2026-09-23", "2026-10-08", ""},
		{"60 days late", history(), "2026-09-23", "2026-11-22", ""},
		{"before the expected day", history(), "2026-09-23", "2026-09-20", CodeNotExpectedYet},
		{"wrong date", history(), "2026-09-22", "2026-09-23", CodePredictionMismatch},
		{"future wrong date", history(), "2026-10-21", "2026-09-23", CodePredictionMismatch},
		{"a real period started since", started, "2026-09-23", "2026-09-26", CodePeriodStarted},
		{"no history", engine.Input{}, "2026-09-23", "2026-09-23", CodePredictionMismatch},
		{"pregnancy: nothing predicted", pregnant, "2026-09-23", "2026-09-23", CodePredictionMismatch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Check(tc.in, p, day(tc.expected), day(tc.today)))
		})
	}
}

// A recorded skip moves the prediction by one cycle; the next skip must be the chained one (skip + ECL).
func TestCheck_ConsecutiveSkips(t *testing.T) {
	p := engine.DefaultParams()
	in := history()
	in.Skips = []engine.Day{day("2026-09-23")}
	assert.Equal(t, CodePredictionMismatch, Check(in, p, day("2026-09-23"), day("2026-10-25")))
	assert.Equal(t, "", Check(in, p, day("2026-10-21"), day("2026-10-25")))
	assert.Equal(t, CodeNotExpectedYet, Check(in, p, day("2026-10-21"), day("2026-10-01")))

	st := engine.Prepare(in, p).Resolve(day("2026-10-08"), day("2026-10-08"))
	assert.True(t, st.Warnings.Has(engine.WarnSkippedCycleRecorded))
	assert.Equal(t, day("2026-10-21"), st.Anchors.PredictedNextPeriodStart)
}

// Every refusal code and the 404 have copy in every embedded language, and the copy differs from the key.
func TestCopy(t *testing.T) {
	for _, code := range []string{CodeNotExpectedYet, CodePeriodStarted, CodePredictionMismatch, "not_found"} {
		for _, loc := range []string{lang.FallbackLocale, "fa"} {
			msg := t2(code, loc)
			require.NotEqual(t, "skips."+code, msg, loc)
		}
	}
	assert.NotEqual(t, t2(CodePeriodStarted, "fa"), t2(CodePeriodStarted, "en"))
}

func t2(key, locale string) string { return t(key, locale) }
