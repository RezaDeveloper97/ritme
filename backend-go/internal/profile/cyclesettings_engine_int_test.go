package profile_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cyclemetrics "github.com/ritme/backend-go/internal/cycle/metrics"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// B-N1-09: switching «خودکار از داده‌ها» off makes the cycle engine read the manual lengths.
func TestCycleSettings_EngineHonoursManualLengths(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000105")
	for _, start := range []string{"2026-06-01", "2026-06-30", "2026-07-29", "2026-08-27"} {
		_, err := e.db.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, is_confirmed, is_estimated, source, created_at, updated_at)
			VALUES (?, ?, DATE_ADD(?, INTERVAL 4 DAY), 1, 0, 'user', NOW(), NOW())`, id, start, start)
		require.NoError(t, err)
	}
	svc := cycleservice.New(e.db, nil)
	today := civildate.MustParse("2026-09-23")
	effective := func() (int, int) {
		sn, err := svc.Load(context.Background(), id, today, today, today)
		require.NoError(t, err)
		m := cyclemetrics.Calculate(sn.Histories, sn.EngineProfile())
		return m.EffectiveCycleLength, m.EffectivePeriodDuration
	}

	r := e.do(t, "PUT", "/api/v1/profile/cycle-settings", tok, `{"cycle_length": 33, "period_length": 6}`)
	require.Equal(t, 200, r.status, r.raw)
	c, p := effective()
	assert.Equal(t, [2]int{29, 5}, [2]int{c, p}, "automatic: the medians win over the stored profile values")

	r = e.do(t, "PUT", "/api/v1/profile/cycle-settings", tok, `{"lengths_auto": false}`)
	require.Equal(t, 200, r.status, r.raw)
	c, p = effective()
	assert.Equal(t, [2]int{33, 6}, [2]int{c, p}, "manual: the profile values win")
	l := csData(t, r)["lengths"].(map[string]any)
	assert.EqualValues(t, 33, l["cycle_length"])
	assert.EqualValues(t, 31, csReminder(t, r, "pms")["cycle_day"])
}
