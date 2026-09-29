package service_test

// F-1 / D-30 (T-M2-36): /cycle/today|date's calculation on the legacy O + 1 day reads the §19
// display window — luteal, not fertile, no fertility_status flag, the seeded early-luteal tips —
// for an avoiding (non_ttc) and a trying (ttc) user, on the contract fixture with the seeded
// recommendations. The engine path of the month view and home is unchanged.

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/cache"
	"github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestDayJSON_OPlusOneUsesDisplayWindow(t *testing.T) {
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "..", "contract", "fixtures", "dump.sql"))
	svc := service.New(db, cache.New(nil, false, nil))
	ctx := context.Background()
	const uid = 1004 // regular

	for _, goal := range []enums.UserGoal{enums.UserGoalNonTtc, enums.UserGoalTtc} {
		t.Run(string(goal), func(t *testing.T) {
			_, err := db.ExecContext(ctx, "UPDATE user_profiles SET user_goal = ? WHERE user_id = ?", string(goal), uid)
			require.NoError(t, err)

			// Find the legacy O + 1 day (phase ovulation after the ovulation day) of the current cycle.
			start := civildate.MustParse("2026-09-01")
			var oPlus1 civildate.Date
			for d := start; d.Compare(start.AddDays(45)) < 0 && oPlus1.IsZero(); d = d.AddDays(1) {
				sn, err := svc.Load(ctx, uid, d, d, d)
				require.NoError(t, err)
				c, err := sn.Engine().CalculateForDate(ctx, d, true)
				require.NoError(t, err)
				if c.Phase == enums.CyclePhaseOvulation && c.Day > c.OvulationDay {
					require.True(t, c.IsFertileWindow, "legacy engine: O + 1 is fertile")
					oPlus1 = d
				}
			}
			require.False(t, oPlus1.IsZero(), "fixture has an O + 1 day")

			for _, locale := range []string{"fa", "en"} {
				p, err := svc.DayJSON(ctx, uid, oPlus1, oPlus1, locale) // /cycle/today on O + 1
				require.NoError(t, err)
				var body struct {
					Calculation struct {
						CycleDay  int               `json:"cycle_day"`
						Ovulation int               `json:"estimated_ovulation_day"`
						Phase     string            `json:"phase"`
						Subphase  string            `json:"subphase"`
						Fertile   bool              `json:"is_fertile_window"`
						TextFlags map[string]string `json:"text_flags"`
						DailyTips []struct {
							Type string `json:"type"`
							Text string `json:"text"`
						} `json:"daily_tips"`
					} `json:"calculation"`
				}
				require.NoError(t, json.Unmarshal(p.JSON, &body), string(p.JSON))
				c := body.Calculation
				assert.Equal(t, c.Ovulation+1, c.CycleDay)
				assert.Equal(t, "luteal", c.Phase, locale)
				assert.Equal(t, "post_ovulation", c.Subphase, locale)
				assert.False(t, c.Fertile, locale)
				assert.NotContains(t, c.TextFlags, "fertility_status", locale)
				assert.Contains(t, c.TextFlags["phase_info"], enums.CyclePhaseLuteal.Label(locale))
				require.NotEmpty(t, c.DailyTips, "seeded early-luteal tips, not an empty card")
				for _, tip := range c.DailyTips {
					assert.NotEqual(t, "fertility", tip.Type, locale)
					assert.False(t, strings.Contains(tip.Text, "اوج باروری") || strings.Contains(strings.ToLower(tip.Text), "peak fertility"), tip.Text)
				}
			}

			// The ovulation day itself is still fertile.
			o := oPlus1.AddDays(-1)
			p, err := svc.DayJSON(ctx, uid, o, o, "fa")
			require.NoError(t, err)
			var day struct {
				Calculation struct {
					Phase   string `json:"phase"`
					Fertile bool   `json:"is_fertile_window"`
				} `json:"calculation"`
			}
			require.NoError(t, json.Unmarshal(p.JSON, &day))
			assert.Equal(t, "ovulation", day.Calculation.Phase)
			assert.True(t, day.Calculation.Fertile)
		})
	}
}
