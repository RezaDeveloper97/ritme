package feed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

type fakeInbox struct {
	open []OpenAlert
	err  error
	got  int64
}

func (f *fakeInbox) Need(*compiled.Compiled) []string { return []string{"bp_sys"} }

func (f *fakeInbox) Sync(_ context.Context, _ *compiled.Compiled, _ uint64, _ rules.Facts, _ time.Time, open int64) ([]OpenAlert, error) {
	f.got = open
	return f.open, f.err
}

func inboxRule(id uint64, kind, level string) *rules.Rule {
	return &rules.Rule{ID: id, Kind: kind, Level: level, Placement: rules.SlotAlertsInbox, Status: rules.StatusApproved,
		IsActive: true, Version: 1, ApprovedVersion: 1, AudiencePercent: 100, HoursFrom: -1, HoursTo: -1,
		Content: rules.Content{A: rules.Variant{Title: "t", Body: "b"}}}
}

func buildWith(t *testing.T, in Inbox) *Result {
	t.Helper()
	comp := compiled.CompileRules(nil)
	s := New(&fakeQ{}, nil, nil, nil).WithLoader(func(context.Context) (*compiled.Compiled, error) { return comp, nil }).
		WithInbox(in)
	var need []string
	res, err := s.Build(context.Background(), Request{UserID: 42, Slots: DefaultSlots, Locale: "en", Alerts: true,
		Now:   time.Date(2026, 10, 31, 10, 0, 0, 0, civildate.Tehran),
		Facts: func(_ context.Context, n []string) (rules.Facts, error) { need = n; return rules.Facts{}, nil }})
	require.NoError(t, err)
	assert.Contains(t, need, "bp_sys")
	return res
}

func slotOf(res *Result, name string) Slot {
	for _, s := range res.Slots {
		if s.Name == name {
			return s
		}
	}
	return Slot{}
}

// SPEC §5.4.1 fan-out: an open safety row is the urgent_modal until acknowledged; an open engine row fills the empty
// status_banner; the badge counts come from the open rows after the diff.
func TestFeed_InboxFanOut(t *testing.T) {
	safety := inboxRule(1, rules.KindSafety, rules.LevelFollowSoon)
	engine := inboxRule(2, rules.KindEngine, rules.LevelSuggestion)
	clinical := inboxRule(3, rules.KindClinical, rules.LevelWorthFollowUp)
	in := &fakeInbox{open: []OpenAlert{
		{ID: 10, Rule: safety, Kind: safety.Kind, Level: safety.Level, State: StateActive},
		{ID: 11, Rule: engine, Kind: engine.Kind, Level: engine.Level, State: StateAcknowledged},
		{ID: 12, Rule: clinical, Kind: clinical.Kind, Level: clinical.Level, State: StateActive},
	}}
	res := buildWith(t, in)
	um := slotOf(res, "urgent_modal")
	require.Len(t, um.Choices, 1)
	assert.EqualValues(t, 1, um.Choices[0].Rule.ID)
	sb := slotOf(res, "status_banner")
	require.Len(t, sb.Choices, 1)
	assert.EqualValues(t, 2, sb.Choices[0].Rule.ID)
	// alert_id (the inbox row id) rides on fanned-out items only.
	body, err := json.Marshal(res.JSON("en"))
	require.NoError(t, err)
	var out struct {
		Slots map[string][]map[string]any `json:"slots"`
	}
	require.NoError(t, json.Unmarshal(body, &out))
	assert.EqualValues(t, 10, out.Slots["urgent_modal"][0]["alert_id"])
	assert.EqualValues(t, 11, out.Slots["status_banner"][0]["alert_id"])
	assert.Empty(t, slotOf(res, "smart_tip").Choices) // clinical stays in the inbox only
	assert.Equal(t, Alerts{Open: 3, Urgent: 1}, res.Alerts)

	in.open[0].State = StateAcknowledged
	res = buildWith(t, in)
	assert.Empty(t, slotOf(res, "urgent_modal").Choices)
	assert.Equal(t, Alerts{Open: 3, Urgent: 1}, res.Alerts)
}

// A failing evaluation never fails the feed: the counts read before it stay.
func TestFeed_InboxErrorKeepsCounts(t *testing.T) {
	res := buildWith(t, &fakeInbox{err: errors.New("boom")})
	assert.Equal(t, Alerts{}, res.Alerts)
	assert.Empty(t, slotOf(res, "urgent_modal").Choices)
}

// T-M10-08b: an open row of a rule with an evidence renderer (the preg.* alerts) renders with its stored evidence over
// the facts, in the renderer's terms; without one (or with a variable missing) it falls back to the fact render.
func TestRenderOpen_StoredEvidence(t *testing.T) {
	InstallEvidenceRenderer("preg.", func(text string, vars map[string]any, _ string) string {
		for _, name := range rules.Placeholders(text) {
			if v, ok := vars[name]; ok {
				text = strings.ReplaceAll(text, "{"+name+"}", "<"+fmt.Sprint(v)+">")
			}
		}
		return text
	})
	t.Cleanup(func() { InstallEvidenceRenderer("preg.", nil) })
	v := rules.Variant{Title: "Logged {preg_symptoms}", Body: "Week {preg_week}"}
	f := rules.Facts{"preg_week": 20.0}
	preg := &OpenAlert{ID: 7, Rule: &rules.Rule{ID: 1, RuleKey: "preg.critical_symptom"}}
	ev := openEvidence{7: {"preg_symptoms": "bleeding"}}

	out, ok := renderOpen(v, preg, ev, f, "en")
	require.True(t, ok)
	assert.Equal(t, "Logged <bleeding>", out.Title, "the stored evidence")
	assert.Equal(t, "Week <20>", out.Body, "the facts fill what the evidence lacks")

	// No stored evidence and no fact: not renderable.
	_, ok = renderOpen(v, preg, nil, f, "en")
	assert.False(t, ok)

	// Another domain's rule: the plain fact render.
	other := &OpenAlert{ID: 8, Rule: &rules.Rule{ID: 2, RuleKey: "cycle.late"}}
	out, ok = renderOpen(rules.Variant{Title: "Week {preg_week}"}, other, ev, f, "en")
	require.True(t, ok)
	assert.Equal(t, "Week 20", out.Title)
}

func TestInLifeMode_ScopedPrefix(t *testing.T) {
	InstallModeScope("preg.", "pregnancy")
	t.Cleanup(func() { InstallModeScope("preg.", "") })
	preg := &rules.Rule{RuleKey: "preg.critical_symptom"}
	other := &rules.Rule{RuleKey: "bp_crisis"}

	assert.True(t, inLifeMode(preg, rules.Facts{FactLifeMode: "pregnancy"}))
	assert.False(t, inLifeMode(preg, rules.Facts{FactLifeMode: "cycle"}), "a pregnancy alert never pops up in cycle mode")
	assert.True(t, inLifeMode(preg, rules.Facts{}), "unknown mode fails open")
	assert.True(t, inLifeMode(other, rules.Facts{FactLifeMode: "cycle"}), "unscoped rules are untouched")
	assert.Equal(t, []string{FactLifeMode}, ModeScopeFacts())
}
