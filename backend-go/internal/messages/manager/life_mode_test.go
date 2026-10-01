package manager

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/legacy"
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

// B-N2-11b (N2 stage smoke B-3): a teen / menopause user in the fertile window never reads fertility or
// conception copy in the daily message (the cycle_base_ttc fertility_info line, TTC tips), whatever
// user_goal says — the phase blurb the home card falls back to stays a plain cycle message.
func TestLifeMode_NoFertilityCopyInFertileWindow(t *testing.T) {
	fertile := legacy.Calculation{
		Complete: true, Day: 14, Phase: enums.CyclePhaseOvulation, CurrentSubphase: enums.CycleSubphaseOvulationLikely,
		CycleLength: 28, OvulationDay: 14, IsFertileWindow: true,
	}
	for _, mode := range []string{"teen", "menopause"} {
		for _, locale := range []string{"fa", "en"} {
			src := &lifeSource{
				fakeSource: fakeSource{profile: &Profile{UserGoal: "ttc", SubscriptionType: "premium", HasLastPeriodStart: true}, calc: fertile},
				mode:       mode,
			}
			res, err := New(src, defaultsContent{}, locale, day).Generate(context.Background(), day, "")
			require.NoError(t, err, mode)
			primary, err := json.Marshal(toJSON(t, res.JSON())["primary_message"])
			require.NoError(t, err)
			text := strings.ToLower(string(primary))
			for _, w := range []string{"باروری", "باردار", "fertil", "conceive", "ttc_tips", "fertility_info"} {
				assert.NotContains(t, text, w, "%s %s", mode, locale)
			}
		}
	}
}
