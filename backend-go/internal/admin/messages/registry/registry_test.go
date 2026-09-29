package registry_test

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/messages/registry"
	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func TestGroups(t *testing.T) {
	names := registry.GroupNames()
	assert.Equal(t, []string{registry.WeekTipGroup, registry.AlertGroup, registry.SetupGroup}, names[:3])
	for _, g := range content.Groups() {
		assert.Contains(t, names, g, "every code-fallback group is registered")
		assert.Equal(t, content.Items(g), registry.Keys(g))
	}

	keys := registry.Keys(registry.WeekTipGroup)
	require.Len(t, keys, registry.MaxWeek)
	assert.Equal(t, "1", keys[0])
	assert.Equal(t, strconv.Itoa(registry.MaxWeek), keys[len(keys)-1])

	alert := registry.Keys(registry.AlertGroup)
	assert.Equal(t, []string{"vomiting_streak", "severe_symptom_count", "critical_symptom", "weight_missing_week",
		"week_entered", "bp_high", "sugar_high", "fetal_movement", "legend"}, alert)
	assert.Equal(t, []string{"welcome", "dating", "source_lmp", "source_ultrasound", "source_manual", "history",
		"result", "due_disclaimer", "calendar_note"}, registry.Keys(registry.SetupGroup))

	it, ok := registry.Lookup(registry.AlertGroup, "bp_high")
	require.True(t, ok)
	assert.True(t, it.Typed)
	_, ok = registry.Lookup(registry.AlertGroup, "nope")
	assert.False(t, ok)
	_, ok = registry.Lookup("free_text_group", "x")
	assert.False(t, ok)
}

// weight_missing_week.from_weekday is editable (review #11, T-M2-34 → T-M7-20): 0..6, nullable so a
// row saved before it existed keeps the engine default.
func TestWeightMissingFromWeekdayParam(t *testing.T) {
	rule, ok := registry.FindAlertRule("weight_missing_week")
	require.True(t, ok)
	require.Len(t, rule.Params, 2)
	f := rule.Params[1]
	assert.Equal(t, registry.Field{Key: "from_weekday", Kind: registry.KindInt, Min: 0, Max: 6, Nullable: true}, f)

	in, err := phpval.Decode([]byte(`{"from_week":"3"}`))
	require.NoError(t, err)
	b, err := json.Marshal(registry.Build(rule.Params, in))
	require.NoError(t, err)
	assert.JSONEq(t, `{"from_week":3,"from_weekday":null}`, string(b))
}

func TestDerivedSchema(t *testing.T) {
	it, ok := registry.Lookup("cycle_base_non_ttc", "menstruation")
	require.True(t, ok)
	assert.False(t, it.Typed)
	kinds := map[string]registry.Kind{}
	for _, f := range it.Fields {
		kinds[f.Key] = f.Kind
	}
	assert.Equal(t, map[string]registry.Kind{
		"short": registry.KindText, "long": registry.KindText, "action": registry.KindText,
		"dos": registry.KindTextList, "donts": registry.KindTextList,
	}, kinds)
}

func TestBuild(t *testing.T) {
	it, _ := registry.Lookup(registry.AlertGroup, "week_entered")
	in, err := phpval.Decode([]byte(`{"contact":"","extra":1,"title":"T","enabled":"1","level":"info",
		"window_days":"7","params":{},"actions":[{"label":"L","key":"ack","x":1}],"what_we_saw":"W",
		"how_sure":"H","advice":"A"}`))
	require.NoError(t, err)
	b, err := json.Marshal(registry.Build(it.Fields, in))
	require.NoError(t, err)
	assert.Equal(t, `{"enabled":true,"level":"info","window_days":7,"params":{},"title":"T","what_we_saw":"W",`+
		`"how_sure":"H","advice":"A","actions":[{"key":"ack","label":"L"}],"contact":null}`, string(b),
		"schema order, typed values, unknown keys dropped, empty nullable text → null, empty params → {}")
}

func TestIsLink(t *testing.T) {
	for s, want := range map[string]bool{
		"https://ritme.app/a": true, "http://x.y": true, "/pregnancy/weeks/8": true,
		"//evil.com": false, "javascript:alert(1)": false, "ftp://x.y": false, "no": false, "/a b": false,
	} {
		assert.Equal(t, want, registry.IsLink(s), s)
	}
}
