package legacy

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/cycle/recommendation"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func marshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func TestEmptyCalculation_FullAndCalendar(t *testing.T) {
	e := New(Input{Profile: &model.Profile{}, Tips: noTips{}}) // profile without last_period_start
	date := civildate.New(2026, 9, 23)

	full, err := e.CalculateForDate(context.Background(), date, true)
	require.NoError(t, err)
	_, ok := full.CycleDay()
	assert.False(t, ok)
	assert.Equal(t, `{"calculation_date":"2026-09-23","cycle_day":null,"phase":null,"subphase":null,`+
		`"estimated_ovulation_day":null,"cycle_length_used":null,"is_fertile_window":false,"is_pms_window":false,`+
		`"is_period_tomorrow":false,"is_luteal_spotting":false,"cycle_score":null,"age_factor":null,`+
		`"base_probability":null,"symptom_score":null,"final_probability":null,"cycle_variability":null,`+
		`"uncertainty_range":null,"text_flags":{"incomplete_profile":{"en":"Please complete your profile to get cycle predictions.",`+
		"\"fa\":\"لطفاً پروفایل خود را کامل کنید تا پیش\u200cبینی سیکل را دریافت کنید.\"}},\"daily_tips\":[],"+
		`"source_profile_data":null,"source_daily_log_data":null}`, marshal(t, full))

	cal, err := e.CalculateForDate(context.Background(), date, false)
	require.NoError(t, err)
	assert.Equal(t, `{"calculation_date":"2026-09-23","cycle_day":null,"phase":null,"subphase":null,`+
		`"estimated_ovulation_day":null,"cycle_length_used":null,"is_fertile_window":false,"is_pms_window":false,`+
		`"is_period_tomorrow":false,"final_probability":null,"cycle_variability":null}`, marshal(t, cal))
}

// localizeCalculation: a locale key the {en, fa} blob lacks keeps the whole object; tips fall back
// to English text and titles.
func TestLocalize_UnknownLocaleKeepsBilingualFlags(t *testing.T) {
	e := New(Input{Profile: &model.Profile{}, Tips: noTips{}})
	c, err := e.CalculateForDate(context.Background(), civildate.New(2026, 9, 23), true)
	require.NoError(t, err)
	c.DailyTips = []recommendation.Tip{{Type: "rest", EN: "Rest.", FA: "استراحت."}}

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(marshal(t, c.Localize("ar"))), &got))
	assert.Equal(t, map[string]any{"incomplete_profile": map[string]any{
		"en": "Please complete your profile to get cycle predictions.",
		"fa": "لطفاً پروفایل خود را کامل کنید تا پیش\u200cبینی سیکل را دریافت کنید.",
	}}, got["text_flags"])
	assert.Equal(t, []any{map[string]any{"type": "rest", "title": "Rest", "icon": "bed", "text": "Rest."}}, got["daily_tips"])

	require.NoError(t, json.Unmarshal([]byte(marshal(t, c.Localize("fa"))), &got))
	assert.Equal(t, map[string]any{"incomplete_profile": "لطفاً پروفایل خود را کامل کنید تا پیش\u200cبینی سیکل را دریافت کنید."}, got["text_flags"])
}

func TestAnchorSearchUsesAllRowsAndRollsForward(t *testing.T) {
	// An unconfirmed (auto-detected) row still anchors the legacy engine; the roll-forward uses the
	// effective length (profile 30 here, no valid confirmed cycles).
	p := &model.Profile{LastPeriodStart: civildate.New(2026, 1, 1), CycleDuration: model.Int(30), PeriodDuration: model.Int(4)}
	h := []model.History{
		{PeriodStart: civildate.New(2026, 6, 1), BleedingLength: model.Int(7)},
		{PeriodStart: civildate.New(2026, 3, 1), IsConfirmed: true, BleedingLength: model.Int(3)},
	}
	e := New(Input{Profile: p, Histories: h, Tips: noTips{}})
	ctx := context.Background()

	c, err := e.CalculateForDate(ctx, civildate.New(2026, 6, 7), false)
	require.NoError(t, err)
	assert.Equal(t, 7, c.Day)
	assert.Equal(t, enums.CyclePhaseMenstruation, c.Phase, "logged bleeding 7 of the anchoring row")

	// 2026-07-01 is 30 days after the anchor: rolled forward to day 1, not a logged start → profile 4 days.
	c, err = e.CalculateForDate(ctx, civildate.New(2026, 7, 5), false)
	require.NoError(t, err)
	assert.Equal(t, 5, c.Day)
	assert.Equal(t, enums.CyclePhaseFollicular, c.Phase)

	// Between the two rows the March start anchors (bleeding 3).
	c, err = e.CalculateForDate(ctx, civildate.New(2026, 3, 4), false)
	require.NoError(t, err)
	assert.Equal(t, 4, c.Day)
	assert.Equal(t, enums.CyclePhaseFollicular, c.Phase)
}

func TestBleedingLengthGuard(t *testing.T) {
	e := New(Input{Profile: &model.Profile{}})
	assert.Equal(t, 5, e.bleedingLength(28, model.Int(30)), "longer than the cycle → min(5, L-1)")
	assert.Equal(t, 3, e.bleedingLength(4, model.Int(9)))
	assert.Equal(t, 1, e.bleedingLength(28, model.Int(0)))
	assert.Equal(t, 6, e.bleedingLength(28, model.Int(6)))
	assert.Equal(t, 5, e.bleedingLength(28, nil), "effective default")
}

func TestOvulationDayFloor(t *testing.T) {
	assert.Equal(t, 15, OvulationDay(28))
	assert.Equal(t, 7, OvulationDay(15))
	assert.Equal(t, 32, OvulationDay(45))
}

type failingTips struct{}

func (failingTips) HasContent(context.Context) (bool, error) { return false, errors.New("db down") }
func (failingTips) ForDay(context.Context, enums.CyclePhase, enums.CycleSubphase, enums.TriggerLog) ([]recommendation.Tip, error) {
	return nil, nil
}

func TestTipSourceErrorPropagates(t *testing.T) {
	e := New(Input{Profile: &model.Profile{LastPeriodStart: civildate.New(2026, 9, 1)}, Tips: failingTips{}})
	_, err := e.CalculateForDate(context.Background(), civildate.New(2026, 9, 3), true)
	require.ErrorContains(t, err, "db down")

	_, err = e.CalculateForDate(context.Background(), civildate.New(2026, 9, 3), false)
	require.NoError(t, err, "calendar mode never reads tips")
}
