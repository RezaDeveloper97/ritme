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
