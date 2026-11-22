package projections

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func tehranNoon(s string) time.Time {
	return civildate.MustParse(s).TehranMidnight().Add(12 * time.Hour)
}

func forecastBuilder() *Builder {
	return (&Builder{}).WithEngine(fakeStates{st: engine.Prepare(engine.Input{Periods: history29}, engine.DefaultParams())})
}

var cycleView = map[family.Domain]family.Level{family.DomainCyclePhase: family.LevelView}

func TestForecastWindow_ClampsToTodayAndHorizon(t *testing.T) {
	today := civildate.MustParse("2026-09-10")
	from, to, ok := ForecastWindow(civildate.MustParse("2026-08-01"), civildate.MustParse("2027-06-01"), today)
	require.True(t, ok)
	assert.Equal(t, today, from, "no past day: logged history never reaches a member")
	assert.Equal(t, today.AddDays(PeriodForecastMaxDays), to)
	_, _, ok = ForecastWindow(civildate.MustParse("2026-08-01"), civildate.MustParse("2026-08-31"), today)
	assert.False(t, ok, "a window wholly in the past is empty")
}

func TestPeriodForecast_PredictedDaysTodayOnwards(t *testing.T) {
	b := forecastBuilder()
	now := tehranNoon("2026-09-05")
	got, err := b.PeriodForecast(context.Background(), snap(family.RolePartner, enums.LifeModeCycle, cycleView),
		civildate.MustParse("2026-08-20"), civildate.MustParse("2026-10-10"), now)
	require.NoError(t, err)
	require.NotEmpty(t, got.Days)
	assert.Equal(t, "2026-09-25", got.Days[0], "the next predicted start (no August history days)")
	for _, d := range got.Days {
		assert.GreaterOrEqual(t, d, "2026-09-05")
	}
	keys := jsonKeys(t, got)
	assert.Equal(t, []string{"days"}, keys, "dates only — no cycle numbers or confidence")
}

func TestPeriodForecast_LateIsEmpty(t *testing.T) {
	got, err := forecastBuilder().PeriodForecast(context.Background(), snap(family.RolePartner, enums.LifeModeCycle, cycleView),
		civildate.MustParse("2026-10-01"), civildate.MustParse("2026-11-01"), tehranNoon("2026-10-01"))
	require.NoError(t, err)
	assert.Empty(t, got.Days)
}

func TestPeriodForecast_DeniedOutsidePartnerCycleGrant(t *testing.T) {
	b := forecastBuilder()
	now := tehranNoon("2026-09-05")
	from, to := civildate.MustParse("2026-09-05"), civildate.MustParse("2026-10-05")
	cases := map[string]family.Snapshot{
		"friend role":    snap(family.RoleFriend, enums.LifeModeCycle, cycleView),
		"no cycle grant": snap(family.RolePartner, enums.LifeModeCycle, nil),
		"pregnant owner": snap(family.RolePartner, enums.LifeModePregnancy, cycleView),
	}
	for name, s := range cases {
		got, err := b.PeriodForecast(context.Background(), s, from, to, now)
		require.ErrorIs(t, err, ErrDenied, name)
		assert.Empty(t, got.Days, name)
	}
}
