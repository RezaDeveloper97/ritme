package feed

import (
	"context"
	"errors"
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
