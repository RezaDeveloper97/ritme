package messages

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/postpartum"
	"github.com/ritme/backend-go/internal/postpartum/guide"
)

// fakeSignals returns the same recovery log for every day.
type fakeSignals struct{ sig postpartum.Signals }

func (f fakeSignals) SignalsOn(context.Context, uint64, civildate.Date) (postpartum.Signals, error) {
	return f.sig, nil
}

// T-M9-06c: the daily message reads the red-flag thresholds in force on the log date, so a later
// postpartum.redflags.fever_c version applies from its effective day on and not before.
func TestPostpartumSource_DatedFeverThreshold(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.Reset()
	clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: clinicalconfig.KeyPostpartumRedFlags, Version: 2,
		Value: clinicalconfig.Value{"fever_c": 38.5}, EffectiveFrom: civildate.MustParse("2026-10-01")}})

	birth := civildate.MustParse("2026-09-01")
	temp := 38.2
	src := WithPostpartum(NewStoreSource(nil, nil, 7, civildate.MustParse("2026-10-21")),
		fakeSignals{postpartum.Signals{BirthDate: &birth, Recovery: postpartum.Recovery{TemperatureC: &temp}}})
	pp, ok := src.(*postpartumSource)
	require.True(t, ok)

	before, err := pp.Postpartum(context.Background(), civildate.MustParse("2026-09-30"))
	require.NoError(t, err)
	assert.Equal(t, []string{guide.AlertFever}, before.Alerts, "version 1 (38.0) on a log before the new version")

	after, err := pp.Postpartum(context.Background(), civildate.MustParse("2026-10-21"))
	require.NoError(t, err)
	assert.Empty(t, after.Alerts, "version 2 (38.5) on a log from its effective day on")

	temp = 38.5
	after, err = pp.Postpartum(context.Background(), civildate.MustParse("2026-10-21"))
	require.NoError(t, err)
	assert.Equal(t, []string{guide.AlertFever}, after.Alerts)
}
