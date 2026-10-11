package checklist

import (
	"encoding/json"
	"io/fs"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

func TestAssemble_GroupOrderAndCounts(t *testing.T) {
	items := []Item{
		{Group: "toys", State: StateHave},
		{Group: GroupSleep, State: StateTodo},
		{Group: GroupClothing, State: StateHave},
		{Group: GroupClothing, State: StateNotNeeded},
		{Group: GroupClothing, State: StateInCart},
	}
	groups, sum := Assemble(items)
	keys := make([]string, 0, len(groups))
	for _, g := range groups {
		keys = append(keys, g.Key)
	}
	assert.Equal(t, []string{GroupClothing, GroupSleep, "toys"}, keys)
	assert.Equal(t, 1, groups[0].Ready)
	assert.Equal(t, 2, groups[0].Total, "not needed leaves the total")
	assert.Equal(t, Summary{Ready: 2, Total: 4, NotNeeded: 1, Percent: 50}, sum)

	groups, sum = Assemble(nil)
	assert.Empty(t, groups)
	assert.Equal(t, Summary{}, sum)
}

func TestWeeksLeft(t *testing.T) {
	today := civildate.MustParse("2026-10-06")
	assert.Equal(t, -1, WeeksLeft(civildate.Date{}, today))
	assert.Equal(t, 9, WeeksLeft(civildate.MustParse("2026-12-11"), today))
	assert.Equal(t, 0, WeeksLeft(civildate.MustParse("2026-10-01"), today))
}

func TestRemindOn(t *testing.T) {
	today := civildate.MustParse("2026-10-06")
	on := Supply{Enabled: true, LeadDays: 3}
	_, ok := RemindOn(Supply{LeadDays: 3}, civildate.MustParse("2026-10-20"), today)
	assert.False(t, ok, "off never plans")
	d, ok := RemindOn(on, civildate.MustParse("2026-10-20"), today)
	assert.True(t, ok)
	assert.Equal(t, "2026-10-17", d.String())
	d, _ = RemindOn(on, civildate.MustParse("2026-10-07"), today)
	assert.Equal(t, today, d, "not before today")
	_, ok = RemindOn(on, today, today)
	assert.False(t, ok, "due today: no reminder")
	_, ok = RemindOn(on, civildate.Date{}, today)
	assert.False(t, ok)
	d, _ = RemindOn(Supply{Enabled: true, LeadDays: 0}, civildate.MustParse("2026-10-20"), today)
	assert.Equal(t, "2026-10-17", d.String(), "invalid lead → default")
	assert.Equal(t, time.Date(2026, 10, 17, 9, 0, 0, 0, civildate.Tehran), ReminderAt(civildate.MustParse("2026-10-17")))
}

func flatKeys(prefix string, v any, out map[string]bool) {
	if m, ok := v.(map[string]any); ok {
		for k, child := range m {
			flatKeys(prefix+"."+k, child, out)
		}
		return
	}
	out[prefix] = true
}

// Every embedded locale has the English keys, every group has a title, and the reminder title is discreet.
func TestLangFiles(t *testing.T) {
	read := func(locale string) map[string]bool {
		b, err := fs.ReadFile(langFS, "lang/"+locale+"/shop_checklist.json")
		require.NoError(t, err)
		var m map[string]any
		require.NoError(t, json.Unmarshal(b, &m))
		out := map[string]bool{}
		flatKeys("", m, out)
		return out
	}
	en := read("en")
	entries, err := fs.ReadDir(langFS, "lang")
	require.NoError(t, err)
	for _, e := range entries {
		got := read(e.Name())
		assert.Equal(t, slices.Sorted(maps.Keys(en)), slices.Sorted(maps.Keys(got)), e.Name())
		for _, g := range Groups {
			assert.NotEqual(t, g, GroupTitle(g, e.Name()))
		}
	}
	assert.Equal(t, "toys", GroupTitle("toys", "en"))
	assert.Equal(t, "Check your personal supplies", T("reminder.title", "en"))
}
