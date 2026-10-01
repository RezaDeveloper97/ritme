package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/enums"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// lifeSource is a fakeSource that also knows the stored life-stage mode (B-N2-01).
type lifeSource struct {
	fakeSource
	mode string
}

func (s *lifeSource) LifeMode(context.Context) (string, error) { return s.mode, nil }

func TestLifeMode_DetectMode(t *testing.T) {
	ctx := context.Background()
	for stored, want := range map[string]enums.MessageMode{
		"": enums.MessageModeCycle, "cycle": enums.MessageModeCycle, "ttc": enums.MessageModeCycle,
		"menopause": enums.MessageModeCycle, "teen": enums.MessageModeCycle, "postpartum": enums.MessageModePostpartum,
	} {
		got, err := New(&lifeSource{mode: stored}, defaultsContent{}, "en", day).DetectMode(ctx)
		require.NoError(t, err)
		assert.Equal(t, want, got, stored)
	}
	// An active pregnancy profile wins over every stored mode.
	src := &lifeSource{fakeSource: fakeSource{preg: &pstore.PregnancyProfile{PregnancyMode: true}}, mode: "postpartum"}
	got, err := New(src, defaultsContent{}, "en", day).DetectMode(ctx)
	require.NoError(t, err)
	assert.Equal(t, enums.MessageModePregnancy, got)
}

// Menopause and teen run on the cycle engine with the safe defaults: never TTC content, whatever user_goal says.
func TestLifeMode_MenopauseAndTeenNeverTTC(t *testing.T) {
	for _, mode := range []string{"menopause", "teen"} {
		src := &lifeSource{
			fakeSource: fakeSource{profile: &Profile{UserGoal: "ttc", SubscriptionType: "premium", HasLastPeriodStart: true},
				recent: logs(16, "very_low"), calc: cycleCalc()},
			mode: mode,
		}
		res, err := New(src, defaultsContent{}, "en", day).Generate(context.Background(), day, "")
		require.NoError(t, err, mode)
		m := toJSON(t, res.JSON())
		assert.Equal(t, "cycle", m["mode"], mode)
		assert.Equal(t, "non_ttc", m["user_goal"], mode)
		assert.NotContains(t, m["primary_message"], "ttc_tips", mode)
		for _, p := range m["patterns"].([]any) {
			assert.NotEqual(t, "ttc_tracking", p.(map[string]any)["pattern_type"], mode)
		}
	}
	// A TTC user (stored ttc) still gets TTC content.
	src := &lifeSource{
		fakeSource: fakeSource{profile: &Profile{UserGoal: "ttc", SubscriptionType: "premium", HasLastPeriodStart: true},
			recent: logs(16, "very_low"), calc: cycleCalc()},
		mode: "ttc",
	}
	res, err := New(src, defaultsContent{}, "en", day).Generate(context.Background(), day, "")
	require.NoError(t, err)
	assert.Contains(t, toJSON(t, res.JSON())["primary_message"], "ttc_tips")
}
