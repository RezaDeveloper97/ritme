package rules_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// validInput is a payload for (level, kind, slot) that satisfies every rule that does not depend on the combination:
// the level's mandatory fields, the kind's band and caps.
func validInput(level, kind, slot string) rules.Input {
	lo, _ := rules.Band(kind)
	in := rules.Input{Title: "x", Level: level, Kind: kind, Priority: lo, Placement: slot,
		Content: rules.Content{A: rules.Variant{Title: "t", Body: "b"}}}
	switch level {
	case rules.LevelSuggestion:
		in.Content.A.CTALabel, in.Content.A.CTATarget = "Go", "ritme://log"
	case rules.LevelWorthFollowUp:
		in.Content.Seen, in.Content.Confidence = "s", "c"
	case rules.LevelFollowSoon:
		in.Content.Seen, in.Content.Confidence, in.Content.Contact = "s", "c", "۱۱۵"
	}
	if kind == rules.KindProduct || kind == rules.KindMarketing {
		in.Caps = []rules.Cap{{Count: 1, Period: "day"}}
	}
	return in
}

// SPEC §5.3 level × kind and §5.4 slot × kind, spelled out as the spec tables (not derived from the code maps).
var (
	specLevelKinds = map[string][]string{
		"info":           {"engine", "clinical", "content", "product", "marketing"},
		"suggestion":     {"engine", "content", "product"},
		"worth_followup": {"engine", "clinical"},
		"follow_soon":    {"safety", "clinical"},
	}
	specSlotKinds = map[string][]string{
		"urgent_modal":    {"safety"},
		"status_banner":   {"engine"},
		"smart_tip":       {"clinical"},
		"daily_advice":    {"content"},
		"daily_challenge": {"product"},
		"push":            {"safety", "engine", "clinical", "content", "product", "marketing"},
		"modal":           {"marketing"},
		"alerts_inbox":    {"safety", "engine", "clinical", "content", "product"},
	}
)

func TestValidate_LevelKindSlotMatrix(t *testing.T) {
	for _, level := range rules.Levels {
		for _, kind := range rules.Kinds {
			for slot, slotKinds := range specSlotKinds {
				in := validInput(level, kind, slot)
				errs := in.Validate()
				want := slices.Contains(specLevelKinds[level], kind) && slices.Contains(slotKinds, kind)
				name := fmt.Sprintf("%s/%s/%s", level, kind, slot)
				if want {
					assert.Empty(t, errs, name)
					continue
				}
				assert.NotEmpty(t, errs, name)
				if !slices.Contains(specLevelKinds[level], kind) {
					assert.Contains(t, errs, "level", name)
				}
				if !slices.Contains(slotKinds, kind) {
					assert.Contains(t, errs, "placement", name)
				}
			}
		}
	}
}

func TestValidate_LevelRules(t *testing.T) {
	// suggestion: exactly one CTA.
	in := validInput(rules.LevelSuggestion, rules.KindProduct, "daily_challenge")
	in.Content.A.CTALabel, in.Content.A.CTATarget = "", ""
	assert.Contains(t, in.Validate(), "content.a")
	// follow_up: evidence required.
	in = validInput(rules.LevelWorthFollowUp, rules.KindClinical, "smart_tip")
	in.Content.Seen = ""
	assert.Contains(t, in.Validate(), "content.seen")
	// urgent: the call button is required, and never a Plus link.
	in = validInput(rules.LevelFollowSoon, rules.KindSafety, "urgent_modal")
	in.Content.Contact = ""
	assert.Contains(t, in.Validate(), "content.contact")
	in = validInput(rules.LevelFollowSoon, rules.KindClinical, "smart_tip")
	in.Content.A.CTALabel, in.Content.A.CTATarget = "Plus", "/plus"
	assert.Contains(t, in.Validate(), "content.a")
	in = validInput(rules.LevelInfo, rules.KindClinical, "smart_tip")
	in.Content.A.CTALabel, in.Content.A.CTATarget = "Plus", "/plus"
	assert.Empty(t, in.Validate(), "a non-urgent clinical CTA may link to Plus")
}

func TestValidate_SafetyExemptions(t *testing.T) {
	base := validInput(rules.LevelFollowSoon, rules.KindSafety, "urgent_modal")
	require.Empty(t, base.Validate())
	half := 50
	in := base
	in.AudiencePercent = &half
	assert.Contains(t, in.Validate(), "audience_percent")
	in = base
	in.Caps = []rules.Cap{{Count: 1, Period: "day"}}
	assert.Contains(t, in.Validate(), "caps")
	in = base
	in.Variants = []rules.VariantWeight{{Key: "a", Weight: 50}, {Key: "b", Weight: 50}}
	in.Content.B = &rules.Variant{Title: "t", Body: "b"}
	assert.Contains(t, in.Validate(), "variants")
}

func TestValidate_CapsVariantsTopicsGoal(t *testing.T) {
	base := validInput(rules.LevelInfo, rules.KindMarketing, "modal")
	require.Empty(t, base.Validate())

	in := base
	in.Caps = []rules.Cap{{Count: 1, Period: "cycle"}}
	assert.Contains(t, in.Validate(), "caps", "marketing needs a day or week cap")
	in.Caps = []rules.Cap{{Count: 1, Period: "cycle"}, {Count: 2, Period: "week"}}
	assert.Empty(t, in.Validate())
	in.Caps = []rules.Cap{{Count: 1, Period: "week"}, {Count: 2, Period: "week"}}
	assert.Contains(t, in.Validate(), "caps", "one cap per period")

	in = base
	in.Content.B = &rules.Variant{Title: "t", Body: "b"}
	in.Variants = []rules.VariantWeight{{Key: "a", Weight: 1}, {Key: "a", Weight: 1}}
	assert.Contains(t, in.Validate(), "variants", "unique keys")
	in.Variants = []rules.VariantWeight{{Key: "a", Weight: 1}, {Key: "b", Weight: 0}}
	assert.Contains(t, in.Validate(), "variants", "weights > 0")
	in.Variants = []rules.VariantWeight{{Key: "a", Weight: 1}, {Key: "z", Weight: 1}}
	assert.Contains(t, in.Validate(), "content.variants", "a text per key")
	in.Variants = nil
	for i := range 7 {
		in.Variants = append(in.Variants, rules.VariantWeight{Key: fmt.Sprintf("v%d", i), Weight: 1})
	}
	assert.Contains(t, in.Validate(), "variants", "at most 6")

	// M-06: a fertility fact (condition, exception, phase value) needs the fertile_window topic.
	clin := validInput(rules.LevelInfo, rules.KindClinical, "smart_tip")
	for _, c := range []rules.Condition{
		{Fact: "is_fertile_window", Op: "eq", Value: true},
		{Fact: "phase", Op: "eq", Value: "ovulation"},
		{Fact: "subphase", Op: "in", Value: []any{"follicular", "fertile_peak"}},
	} {
		in = clin
		in.Conditions = []rules.Group{{Op: "and", Items: []rules.Condition{c}}}
		assert.Contains(t, in.Validate(), "topics", c.Fact)
		in.Topics = []string{rules.TopicFertileWindow}
		assert.Empty(t, in.Validate(), c.Fact)
	}
	in = clin
	in.Exceptions = []rules.Condition{{Fact: "days_to_ovulation", Op: "lt", Value: 2}}
	assert.Contains(t, in.Validate(), "topics")
	in = clin
	in.Conditions = []rules.Group{{Op: "and", Items: []rules.Condition{{Fact: "phase", Op: "eq", Value: "luteal"}}}}
	assert.Empty(t, in.Validate())
	in.Topics = []string{"bogus"}
	assert.Contains(t, in.Validate(), "topics")

	in = clin
	in.GoalEvent = "period_logged"
	assert.Empty(t, in.Validate())
	in.GoalEvent = "opened_app"
	assert.Contains(t, in.Validate(), "goal_event")

	in = clin
	in.HourWindows = []rules.HourWindow{{8, 12}, {22, 6}}
	in.StartsAt, in.EndsAt = "2026-11-01 08:00:00", "2026-11-01 08:00:00"
	e := in.Validate()
	assert.NotContains(t, e, "hour_windows")
	assert.Contains(t, e, "ends_at")
	in.HourWindows = []rules.HourWindow{{25, 2}}
	assert.Contains(t, in.Validate(), "hour_windows")
	in = clin
	in.RuleKey = "Bad Key"
	assert.Contains(t, in.Validate(), "rule_key")
}

func TestNormalize_SpecAliases(t *testing.T) {
	in := rules.Input{Type: "clinical", Slot: "smart_tip", Name: "n", Level: "follow_up"}
	in.Normalize()
	assert.Equal(t, []string{"clinical", "smart_tip", "n", rules.LevelWorthFollowUp}, []string{in.Kind, in.Placement, in.Title, in.Level})
	in = rules.Input{Kind: "content", Type: "clinical", Level: "urgent"}
	in.Normalize()
	assert.Equal(t, "content", in.Kind, "the canonical key wins")
	assert.Equal(t, rules.LevelFollowSoon, in.Level)
	assert.Equal(t, "follow_up", rules.SpecLevel(rules.LevelWorthFollowUp))
}

func TestBandsAndApproval_V6Kinds(t *testing.T) {
	lo, hi := rules.Band(rules.KindContent)
	assert.Equal(t, []int{100, 799}, []int{lo, hi})
	lo, hi = rules.Band(rules.KindMarketing)
	assert.Equal(t, []int{1, 99}, []int{lo, hi})
	assert.True(t, rules.NeedsMedicalApproval(rules.LevelInfo, rules.KindContent))
	assert.False(t, rules.NeedsMedicalApproval(rules.LevelInfo, rules.KindMarketing))
	assert.Equal(t, rules.KindClinical, rules.KindOfPriority(300))
	assert.Equal(t, rules.KindProduct, rules.KindOfPriority(30))
}

// A row with only the pre-v6 columns loads with the same caps, variants, schedule and hours as before.
func TestFromRow_LegacyFallback(t *testing.T) {
	row := store.MessageRule{ID: 9, Level: "info", Kind: "product", Priority: 30, Placement: "push",
		Content:         `{"a":{"title":"t","body":"b"},"b":{"title":"t2","body":"b2"}}`,
		VariantsEnabled: true, AbSplit: 30, Conditions: "[]", Exceptions: "[]",
		CapCount: sql.NullInt16{Int16: 2, Valid: true}, CapPeriod: sql.NullString{String: "week", Valid: true},
		HoursFrom: sql.NullInt16{Int16: 22, Valid: true}, HoursTo: sql.NullInt16{Int16: 6, Valid: true},
		StartsOn:        civildate.NullDate{Date: civildate.MustParse("2026-11-01"), Valid: true},
		EndsOn:          civildate.NullDate{Date: civildate.MustParse("2026-11-30"), Valid: true},
		AudiencePercent: 100, Status: "draft", IsActive: true, Version: 1}
	r := rules.FromRow(&row)
	assert.Equal(t, []rules.Cap{{Count: 2, Period: "week"}}, r.EffectiveCaps())
	assert.Equal(t, []rules.VariantWeight{{Key: "a", Weight: 30}, {Key: "b", Weight: 70}}, r.EffectiveVariants())
	assert.Equal(t, []rules.HourWindow{{22, 6}}, r.EffectiveHourWindows())
	start, end := r.EffectiveSchedule()
	assert.Equal(t, time.Date(2026, 11, 1, 0, 0, 0, 0, civildate.Tehran), start)
	assert.Equal(t, time.Date(2026, 12, 1, 0, 0, 0, 0, civildate.Tehran), end)

	// The engine gates read the same as before: inside the dates and the overnight window, outside them.
	in := time.Date(2026, 11, 30, 23, 0, 0, 0, civildate.Tehran)
	assert.True(t, rules.InWindow(&r, in))
	assert.False(t, rules.InWindow(&r, time.Date(2026, 11, 30, 12, 0, 0, 0, civildate.Tehran)), "outside the hours")
	assert.False(t, rules.InWindow(&r, time.Date(2026, 12, 1, 1, 0, 0, 0, civildate.Tehran)), "after ends_on")
	assert.False(t, rules.InWindow(&r, time.Date(2026, 10, 31, 23, 0, 0, 0, civildate.Tehran)), "before starts_on")

	// Assignment is unchanged: the legacy split and an equivalent variants column agree for every user.
	v6 := r
	v6.Variants = r.EffectiveVariants()
	for uid := uint64(1); uid < 500; uid++ {
		require.Equal(t, r.Assign(uid), v6.Assign(uid), uid)
	}

	// Without variants / caps / hours / dates: a only, no caps, open.
	plain := rules.FromRow(&store.MessageRule{Content: `{"a":{"title":"t","body":"b"}}`, Conditions: "[]", Exceptions: "[]"})
	assert.Equal(t, []rules.VariantWeight{{Key: "a", Weight: 100}}, plain.EffectiveVariants())
	assert.Nil(t, plain.EffectiveCaps())
	assert.Nil(t, plain.EffectiveHourWindows())
}

// The v6 columns win over the legacy ones.
func TestFromRow_V6Columns(t *testing.T) {
	raw := func(s string) db.NullRawJSON { return db.NullRawJSON{V: json.RawMessage(s), Valid: true} }
	row := store.MessageRule{Content: `{"a":{"title":"t","body":"b"},"variants":{"c":{"title":"t3","body":"b3"}}}`,
		Conditions: "[]", Exceptions: "[]", CapCount: sql.NullInt16{Int16: 9, Valid: true},
		CapPeriod: sql.NullString{String: "cycle", Valid: true}, Caps: raw(`[{"count":1,"period":"day"},{"count":3,"period":"week"}]`),
		Variants: raw(`[{"key":"a","weight":1},{"key":"c","weight":3}]`), HourWindows: raw(`[[8,12],[18,22]]`),
		Topics: raw(`["fertile_window"]`), StartsAt: sql.NullTime{Time: time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC), Valid: true},
		ApprovedVersion: sql.NullInt32{Int32: 2, Valid: true}, Version: 3, Locked: true, IsActive: true}
	r := rules.FromRow(&row)
	assert.Equal(t, []rules.Cap{{Count: 1, Period: "day"}, {Count: 3, Period: "week"}}, r.EffectiveCaps())
	assert.Equal(t, []rules.HourWindow{{8, 12}, {18, 22}}, r.EffectiveHourWindows())
	assert.Equal(t, []string{"fertile_window"}, r.Topics)
	assert.Equal(t, 2, r.ApprovedVersion)
	assert.True(t, r.Locked)
	assert.Equal(t, time.Date(2026, 11, 1, 8, 0, 0, 0, civildate.Tehran), r.StartsAt, "DATETIME read as Tehran wall clock")
	assert.True(t, rules.InWindow(&r, time.Date(2026, 11, 2, 19, 0, 0, 0, civildate.Tehran)))
	assert.False(t, rules.InWindow(&r, time.Date(2026, 11, 2, 14, 0, 0, 0, civildate.Tehran)))
	seen := map[string]int{}
	for uid := uint64(1); uid <= 4000; uid++ {
		seen[r.Assign(uid)]++
	}
	assert.InDelta(t, 3000, seen["c"], 200, "weights 1:3")
	assert.Equal(t, "b3", r.VariantContent("c").Body)
}

func TestServed_ApprovedVersionKeepsServing(t *testing.T) {
	r := rules.Rule{Level: rules.LevelInfo, Kind: rules.KindClinical, IsActive: true, Status: rules.StatusPending}
	assert.False(t, r.Served())
	r.ApprovedVersion = 1
	assert.True(t, r.Served())
	r.IsActive = false
	assert.False(t, r.Served())
}

func TestVoidsApproval(t *testing.T) {
	base := rules.Rule{Level: rules.LevelInfo, Kind: rules.KindClinical, Placement: "smart_tip", Priority: 200,
		Content: rules.Content{A: rules.Variant{Title: "t", Body: "b"}},
		Groups:  []rules.Group{{Op: "and", Items: []rules.Condition{{Fact: "mode", Op: "eq", Value: "cycle"}}}}}
	ops := base
	ops.Priority, ops.AudiencePercent, ops.CooldownDays = 300, 50, 3
	ops.Caps, ops.HourWindows = []rules.Cap{{Count: 1, Period: "cycle"}}, []rules.HourWindow{{8, 20}}
	ops.StartsAt = time.Now()
	assert.False(t, rules.VoidsApproval(&base, &ops), "ops-only edit keeps approval")
	for name, mut := range map[string]func(r *rules.Rule){
		"body":       func(r *rules.Rule) { r.Content.A.Body = "x" },
		"level":      func(r *rules.Rule) { r.Level = rules.LevelWorthFollowUp },
		"kind":       func(r *rules.Rule) { r.Kind = rules.KindContent },
		"slot":       func(r *rules.Rule) { r.Placement = "push" },
		"conditions": func(r *rules.Rule) { r.Groups = nil },
		"exceptions": func(r *rules.Rule) { r.Exceptions = []rules.Condition{{Fact: "x", Op: "eq", Value: 1}} },
		"topics":     func(r *rules.Rule) { r.Topics = []string{"fertile_window"} },
		"variants": func(r *rules.Rule) {
			r.Variants = []rules.VariantWeight{{Key: "a", Weight: 1}, {Key: "c", Weight: 1}}
		},
		"variant text": func(r *rules.Rule) { r.Content.Variants = map[string]rules.Variant{"c": {Title: "t", Body: "b"}} },
	} {
		ed := base
		ed.Content.Variants = nil
		mut(&ed)
		assert.True(t, rules.VoidsApproval(&base, &ed), name)
	}
}

func TestApplySnapshot(t *testing.T) {
	head := store.MessageRule{ID: 1, Level: "info", Kind: "clinical", Priority: 400, Placement: "smart_tip",
		Content: `{"a":{"title":"t","body":"v2"}}`, Conditions: "[]", Exceptions: "[]", Status: "pending_approval",
		Version: 2, ApprovedVersion: sql.NullInt32{Int32: 1, Valid: true}, IsActive: true}
	out, ok := rules.ApplySnapshot(&head, `{"id":1,"level":"info","kind":"clinical","priority":100,"placement":"smart_tip",
		"content":{"a":{"title":"t","body":"v1"}},"conditions":[{"op":"and","items":[]}],"exceptions":[],"topics":null}`)
	require.True(t, ok)
	r := rules.FromRow(&out)
	assert.Equal(t, "v1", r.Content.A.Body)
	assert.Equal(t, 400, r.Priority, "ops fields stay the head's")
	assert.Equal(t, rules.StatusApproved, r.Status)
	_, ok = rules.ApplySnapshot(&head, `not json`)
	assert.False(t, ok)
}
