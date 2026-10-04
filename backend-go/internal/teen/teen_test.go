package teen

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func TestWeekBucket(t *testing.T) {
	today := civildate.MustParse("2026-10-02") // Friday; the week started Saturday 09-26
	for next, want := range map[string]PeriodWeek{
		"2026-10-02": WeekThis,
		"2026-09-29": WeekThis, // 3 days overdue: within the grace
		"2026-10-03": WeekNext, // Saturday = next week
		"2026-10-09": WeekNext,
		"2026-10-10": WeekLater,
	} {
		assert.Equal(t, want, WeekBucket(civildate.MustParse(next), 28, today), next)
	}
	assert.Equal(t, WeekUnknown, WeekBucket(civildate.Date{}, 28, today))
	assert.Equal(t, WeekUnknown, WeekBucket(civildate.MustParse("2026-09-20"), 0, today), "overdue without a cycle length")
}

// Lateness must not leak: a period overdue by 10 or 20 days never shows «this week» in two consecutive weeks.
func TestWeekBucket_OverdueRollsForward(t *testing.T) {
	for _, overdue := range []int{4, 10, 20, 40} {
		for _, cycleLen := range []int{21, 28, 35} {
			week1 := civildate.MustParse("2026-10-02")
			next := week1.AddDays(-overdue)
			b1 := WeekBucket(next, cycleLen, week1)
			b2 := WeekBucket(next, cycleLen, week1.AddDays(7))
			assert.False(t, b1 == WeekThis && b2 == WeekThis, "overdue %d, cycle %d: %s then %s", overdue, cycleLen, b1, b2)
		}
	}
	// Concretely, on a 28-day cycle (prediction 09-22 = 10 days late, 09-12 = 20 days late on Friday 10-02):
	today := civildate.MustParse("2026-10-02")
	assert.Equal(t, WeekLater, WeekBucket(today.AddDays(-10), 28, today))            // rolled to 10-20
	assert.Equal(t, WeekLater, WeekBucket(today.AddDays(-20), 28, today))            // rolled to 10-10
	assert.Equal(t, WeekLater, WeekBucket(today.AddDays(-10), 28, today.AddDays(7))) // 10-20 seen from 10-09
	assert.Equal(t, WeekNext, WeekBucket(today.AddDays(-20), 28, today.AddDays(7)))  // 10-10 seen from 10-09

	// CB-TEEN-04: the grace never carries a prediction over a week boundary. Thursday 10-01 is «this week» on
	// 09-27; on Sunday 10-04 (3 days late, new week) it rolls forward instead of reading «this week» again.
	assert.Equal(t, WeekThis, WeekBucket(civildate.MustParse("2026-10-01"), 28, civildate.MustParse("2026-09-27")))
	assert.Equal(t, WeekLater, WeekBucket(civildate.MustParse("2026-10-01"), 28, civildate.MustParse("2026-10-04")))
	// Exhaustively: for any prediction and any day, «this week» never shows on that day and 7 days later.
	base := civildate.MustParse("2026-09-01")
	for p := 0; p < 60; p++ {
		next := base.AddDays(p)
		for d := 0; d < 90; d++ {
			for _, cycleLen := range []int{21, 28, 35, 45} {
				day := base.AddDays(d)
				b1, b2 := WeekBucket(next, cycleLen, day), WeekBucket(next, cycleLen, day.AddDays(7))
				assert.False(t, b1 == WeekThis && b2 == WeekThis, "prediction %s, day %s, cycle %d", next, day, cycleLen)
			}
		}
	}
}

func TestAllowsFor(t *testing.T) {
	assert.Equal(t, Allows{}, AllowsFor(enums.LifeModeTeen), "a teen gets no commercial surface")
	for _, m := range []enums.LifeMode{"", enums.LifeModeCycle, enums.LifeModeTTC, enums.LifeModeMenopause} {
		assert.Equal(t, Allows{true, true, true, true, true}, AllowsFor(m), m)
	}
}

func TestEnums(t *testing.T) {
	for _, a := range AgeBands {
		assert.True(t, a.Valid())
	}
	assert.False(t, AgeBand("18_99").Valid())
	for _, m := range Menarches {
		assert.True(t, m.Valid())
	}
	assert.False(t, Menarche("yes").Valid())
}

type fakeCatalog map[string][]catalog.Item

func (f fakeCatalog) Items(_ context.Context, group string) ([]catalog.Item, error) {
	return f[group], nil
}

func item(code, meta string, audiences ...string) catalog.Item {
	it := catalog.Item{Code: code, Title: json.RawMessage(`{"en":"` + code + `"}`), Audiences: audiences}
	if meta != "" {
		it.Meta = json.RawMessage(meta)
	}
	return it
}

func TestContent_PicksFirstMatchingEstimate(t *testing.T) {
	s := &Service{catalog: fakeCatalog{
		GroupSigns: {
			item("sign_all", `{"kind":"sign"}`, "teen"),
			item("sign_not_yet", `{"kind":"sign","menarche":["not_yet"]}`, "teen"),
			item("est_young", `{"kind":"estimate","menarche":["not_yet"],"age_bands":["10_12"]}`, "teen"),
			item("est_any_not_yet", `{"kind":"estimate","menarche":["not_yet"]}`, "teen"),
			item("est_adult", `{"kind":"estimate"}`, "menopause"), // another audience: never shown
			item("talk", `{"kind":"talk"}`),
			item("broken", `{not json`, "teen"),
		},
		GroupFAQ: {item("faq", "", "teen"), item("meno_faq", "", "menopause")},
	}}
	c, err := s.Content(context.Background(), &Profile{AgeBand: Age13to15, Menarche: MenarcheNotYet})
	require.NoError(t, err)
	require.NotNil(t, c.Readiness)
	assert.Equal(t, "est_any_not_yet", c.Readiness.Code)
	require.Len(t, c.Signs, 2)
	assert.Equal(t, "talk", c.Talk.Code)
	require.Len(t, c.FAQ, 1)

	c, err = s.Content(context.Background(), &Profile{AgeBand: Age10to12, Menarche: MenarcheNotYet})
	require.NoError(t, err)
	assert.Equal(t, "est_young", c.Readiness.Code)

	c, err = s.Content(context.Background(), &Profile{AgeBand: Age10to12, Menarche: MenarcheOver1y})
	require.NoError(t, err)
	assert.Nil(t, c.Readiness)
	require.Len(t, c.Signs, 1)
	assert.Equal(t, "sign_all", c.Signs[0].Code)

	c, err = s.Content(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, c.Readiness)
	require.Len(t, c.Signs, 1, "no answers: only unfiltered signs")
}

func TestKitReady(t *testing.T) {
	assert.False(t, Kit{}.Ready(), "an empty list is never ready")
	k := Kit{Items: []KitItem{{Checked: true}, {Checked: false}}, Checked: 1}
	assert.False(t, k.Ready())
	k.Checked = 2
	assert.True(t, k.Ready())
}

func TestCardJSON_OnlyGrantedParts(t *testing.T) {
	week := WeekNext
	raw, err := json.Marshal(CardJSON(ParentCard{TeenName: "Nila", View: ParentView{PeriodWeek: &week}}))
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	assert.Equal(t, "next_week", m["next_period_week"])
	assert.Nil(t, m["kit_ready"])
	assert.Nil(t, m["note"])
	assert.Equal(t, true, m["read_only"])
	assert.Equal(t, map[string]any{"teen_period_week": "none", "teen_kit": "none", "teen_notes": "none"}, m["grants"])
}

func TestAllowsJSON(t *testing.T) {
	raw, err := json.Marshal(AllowsJSON(AllowsFor(enums.LifeModeTeen)))
	require.NoError(t, err)
	assert.JSONEq(t, `{"shop":false,"banners":false,"ads":false,"plus_upsell":false,"commercial_recommendations":false}`, string(raw))
}

// Every language file has the same keys.
func TestLangFiles_SameKeys(t *testing.T) {
	keys := map[string][]string{}
	err := fs.WalkDir(langFS, "lang", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := langFS.ReadFile(path)
		if err != nil {
			return err
		}
		var m map[string]map[string]string
		if err := json.Unmarshal(raw, &m); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		var ks []string
		for g, inner := range m {
			for k := range inner {
				ks = append(ks, g+"."+k)
			}
		}
		sort.Strings(ks)
		keys[path] = ks
		return nil
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(keys), 2)
	var first []string
	for _, ks := range keys {
		if first == nil {
			first = ks
		}
		assert.Equal(t, first, ks)
	}
	assert.Equal(t, "Saved. Your answers stay yours.", T("messages.profile_saved", "en"))
	assert.NotEmpty(t, attributes("fa"))
}
