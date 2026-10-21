package manager

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/postpartum"
	"github.com/ritme/backend-go/internal/postpartum/guide"
)

type postpartumSource struct {
	fakeSource
	state *PostpartumState
}

func (p *postpartumSource) Postpartum(context.Context, civildate.Date) (*PostpartumState, error) {
	return p.state, nil
}

// adminContent is defaultsContent plus live message_contents rows (guide.Payloader).
type adminContent struct {
	defaultsContent
	rows map[string]any // "group|key|locale"
}

func (a adminContent) Payload(_ context.Context, group, key, locale string) (any, bool, error) {
	v, ok := a.rows[group+"|"+key+"|"+locale]
	return v, ok, nil
}

func genJSON(t *testing.T, src Source, c Content, locale string) map[string]any {
	t.Helper()
	res, err := New(src, c, locale, day).Generate(context.Background(), day, enums.MessageModePostpartum)
	require.NoError(t, err)
	b, err := json.Marshal(res.JSON())
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

// B-N5-01: the postpartum mode no longer answers Laravel's empty result.
func TestPostpartum_WeekTipFromBirth(t *testing.T) {
	birth := day.AddDays(-17) // week 3 (2 weeks 3 days)
	m := genJSON(t, &postpartumSource{state: &PostpartumState{BirthDate: &birth}}, defaultsContent{}, "en")
	assert.Equal(t, "postpartum", m["mode"])
	assert.Equal(t, "non_ttc", m["user_goal"])
	ci := m["context_info"].(map[string]any)
	assert.EqualValues(t, 17, ci["days_since_birth"])
	assert.EqualValues(t, 3, ci["week"])
	assert.Equal(t, guide.PhasePuerperium, ci["phase"])
	pm := m["primary_message"].(map[string]any)
	assert.Equal(t, "Bleeding gets lighter", pm["short_message"])
	assert.Equal(t, false, pm["has_override"])
	assert.Equal(t, "3", pm["week_tip"].(map[string]any)["key"])
	assert.Equal(t, []any{}, m["correlations"])
}

func TestPostpartum_NoBirthOrNoSourceGetsTheLateTip(t *testing.T) {
	m := genJSON(t, &fakeSource{}, defaultsContent{}, "fa")
	assert.Equal(t, []any{}, m["context_info"])
	pm := m["primary_message"].(map[string]any)
	assert.Equal(t, "ادامه مسیر بهبودی", pm["short_message"])
	assert.Nil(t, pm["week"])
}

func TestPostpartum_OverrideOrder(t *testing.T) {
	birth := day.AddDays(-30)
	src := &postpartumSource{state: &PostpartumState{BirthDate: &birth, Alerts: []string{guide.AlertLargeClots, guide.AlertHeavyBleeding}, CheckinDue: "full"}}
	pm := genJSON(t, src, defaultsContent{}, "en")["primary_message"].(map[string]any)
	assert.Equal(t, true, pm["has_override"])
	assert.Equal(t, guide.AlertHeavyBleeding, pm["override_type"])
	assert.Equal(t, "115", pm["call_number"])

	src.state.Alerts = nil
	pm = genJSON(t, src, defaultsContent{}, "en")["primary_message"].(map[string]any)
	assert.Equal(t, guide.AlertCheckinFull, pm["override_type"])
	assert.Nil(t, pm["call_number"])
}

func TestPostpartum_AdminTipWins(t *testing.T) {
	birth := day.AddDays(-3)
	row := phpval.NewMap()
	row.Set("title", "Admin week 1")
	row.Set("body", "Body")
	c := adminContent{rows: map[string]any{guide.TipGroup + "|1|en": row}}
	pm := genJSON(t, &postpartumSource{state: &PostpartumState{BirthDate: &birth}}, c, "en")["primary_message"].(map[string]any)
	assert.Equal(t, "Admin week 1", pm["short_message"])
}

// T-M9-06b: the urgent red flags of the recovery log (fever ≥ postpartum.redflags.fever_c, breathlessness, …) win
// over the lochia warnings and the due check-in, with level urgent and the 115 call.
func TestPostpartum_RedFlagOverride(t *testing.T) {
	birth := day.AddDays(-10)
	fever, yes := 38.0, true
	rec := postpartum.Recovery{TemperatureC: &fever, LochiaAmount: "heavy"}
	src := &postpartumSource{state: &PostpartumState{BirthDate: &birth, Alerts: rec.Alerts(), CheckinDue: "short"}}
	pm := genJSON(t, src, defaultsContent{}, "en")["primary_message"].(map[string]any)
	assert.Equal(t, true, pm["has_override"])
	assert.Equal(t, guide.AlertFever, pm["override_type"])
	assert.Equal(t, "urgent", pm["alert_level"])
	assert.Equal(t, "115", pm["call_number"])
	assert.Equal(t, "Your temperature is high", pm["short_message"])

	below := 37.9
	rec = postpartum.Recovery{TemperatureC: &below, Breathless: &yes}
	src.state.Alerts = rec.Alerts()
	pm = genJSON(t, src, defaultsContent{}, "en")["primary_message"].(map[string]any)
	assert.Equal(t, guide.AlertBreathless, pm["override_type"])
	assert.Equal(t, "urgent", pm["alert_level"])
	assert.Equal(t, "115", pm["call_number"])
	assert.Equal(t, "You are short of breath", pm["short_message"])

	rec = postpartum.Recovery{TemperatureC: &below, LochiaAmount: "heavy"}
	src.state.Alerts = rec.Alerts()
	pm = genJSON(t, src, defaultsContent{}, "en")["primary_message"].(map[string]any)
	assert.Equal(t, guide.AlertHeavyBleeding, pm["override_type"], "37.9 °C is no fever")
	assert.Equal(t, "warning", pm["alert_level"])
}

// An admin row with an empty body never silences a red flag: the embedded copy is used instead.
func TestPostpartum_RedFlagEmptyAdminRowFallsBack(t *testing.T) {
	birth := day.AddDays(-5)
	row := phpval.NewMap()
	row.Set("title", "Admin title")
	row.Set("body", "")
	c := adminContent{rows: map[string]any{guide.AlertGroup + "|" + guide.AlertPadPerHour + "|en": row}}
	src := &postpartumSource{state: &PostpartumState{BirthDate: &birth, Alerts: []string{guide.AlertPadPerHour}}}
	pm := genJSON(t, src, c, "en")["primary_message"].(map[string]any)
	assert.Equal(t, guide.AlertPadPerHour, pm["override_type"])
	assert.NotEmpty(t, pm["long_message"])
	assert.NotEqual(t, "Admin title", pm["short_message"])
}
