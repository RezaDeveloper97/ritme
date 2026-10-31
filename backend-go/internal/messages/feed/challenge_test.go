package feed

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func TestPickChallenge_OngoingUntilCompleted(t *testing.T) {
	cands := []ChallengeCandidate{{ID: 1, Eligible: true}, {ID: 2, Ongoing: true}, {ID: 3, Eligible: true}}
	p, ok := PickChallenge(ChallengeInput{Candidates: cands})
	assert.True(t, ok)
	assert.Equal(t, ChallengePick{ID: 2, Reason: ChallengeOngoing}, p, "the joined run wins over the sort order")

	// completed or abandoned (the caller no longer marks it ongoing; not eligible again once joined)
	cands[1].Ongoing = false
	p, _ = PickChallenge(ChallengeInput{Candidates: cands})
	assert.Equal(t, ChallengePick{ID: 1, Reason: ChallengeProduct}, p, "then the first eligible challenge in sort order")

	// an ongoing run that became contraindicated is never pushed
	p, _ = PickChallenge(ChallengeInput{Candidates: []ChallengeCandidate{{ID: 2, Ongoing: true, Contraindicated: true}, {ID: 3, Eligible: true}}})
	assert.Equal(t, uint64(3), p.ID)
	_, ok = PickChallenge(ChallengeInput{Candidates: []ChallengeCandidate{{ID: 2, Ongoing: true, Contraindicated: true}, {ID: 4, Contraindicated: true, Eligible: true}}})
	assert.False(t, ok, "nothing left: the legacy pick fills the card")

	_, ok = PickChallenge(ChallengeInput{Candidates: cands, Blocked: true})
	assert.False(t, ok, "blocked after a safety message")
	_, ok = PickChallenge(ChallengeInput{})
	assert.False(t, ok)
}

func TestPickChallenge_RulePromotes(t *testing.T) {
	promote := &rules.Rule{ID: 7, Priority: 300, Groups: []rules.Group{{Op: "and", Items: []rules.Condition{{Fact: FactChallengeID, Op: "eq", Value: 3.0}}}}}
	generic := &rules.Rule{ID: 8, Priority: 900}
	sel := func(f rules.Facts) []rules.Choice {
		var out []rules.Choice
		if rules.MatchRule(promote, f) {
			out = append(out, rules.Choice{Rule: promote, Variant: "a"})
		}
		return append(out, rules.Choice{Rule: generic, Variant: "a"})
	}
	cands := []ChallengeCandidate{{ID: 1, Eligible: true}, {ID: 3, Eligible: true}}
	p, ok := PickChallenge(ChallengeInput{Candidates: cands, Facts: rules.Facts{"mode": "cycle"}, Select: sel})
	assert.True(t, ok)
	assert.Equal(t, uint64(3), p.ID)
	assert.Equal(t, ChallengePromoted, p.Reason)
	assert.Equal(t, uint64(7), p.Choice.Rule.ID)

	// a rule that does not read challenge_id never promotes; an ongoing run still wins over a promotion
	p, _ = PickChallenge(ChallengeInput{Candidates: []ChallengeCandidate{{ID: 1, Eligible: true}}, Select: sel})
	assert.Equal(t, ChallengePick{ID: 1, Reason: ChallengeProduct}, p)
	p, _ = PickChallenge(ChallengeInput{Candidates: append(cands, ChallengeCandidate{ID: 9, Ongoing: true}), Select: sel})
	assert.Equal(t, ChallengeOngoing, p.Reason)
}

func TestChallengeBlocked(t *testing.T) {
	now := time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)
	assert.False(t, ChallengeBlocked(nil, rules.Facts{}, now))
	assert.True(t, ChallengeBlocked(&rules.UserHistory{LastSafetyShown: now.Add(-23 * time.Hour)}, nil, now))
	assert.False(t, ChallengeBlocked(&rules.UserHistory{LastSafetyShown: now.Add(-24 * time.Hour)}, nil, now))
	assert.True(t, ChallengeBlocked(nil, rules.Facts{"requires_user_input": true}, now))
}

type fakeSource struct {
	cands []ChallengeCandidate
	runs  []ChallengeRun
}

func (f *fakeSource) NeedFacts(context.Context) ([]string, error) { return []string{"life_mode"}, nil }
func (f *fakeSource) FeedCandidates(_ context.Context, _ civildate.Date, _ rules.Facts, runs []ChallengeRun) ([]ChallengeCandidate, error) {
	f.runs = runs
	return f.cands, nil
}
func (f *fakeSource) FeedCard(_ context.Context, id uint64, _ civildate.Date, _ []ChallengeRun) (ChallengeCard, bool, error) {
	return ChallengeCard{ID: id, Title: json.RawMessage(`{"fa":"کشش","en":"Stretch"}`), DurationDays: 7, Day: 2}, true, nil
}

func TestFeed_ChallengePickWiring(t *testing.T) {
	src := &fakeSource{cands: []ChallengeCandidate{{ID: 4, Eligible: true}, {ID: 5, Ongoing: true}}}
	InstallChallenges(src)
	t.Cleanup(func() { InstallChallenges(nil) })
	runs := []ChallengeRun{{ChallengeID: 5, StartedOn: "2026-10-05", Status: "active"}}
	req := Request{UserID: 1, Slots: []string{SlotDailyChallenge}, Locale: "en", Now: time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC),
		ChallengeRuns: func(context.Context) ([]ChallengeRun, bool, error) { return runs, true, nil }}
	comp := compiled.CompileRules(nil)

	assert.Equal(t, []string{"life_mode"}, challengeNeed(context.Background(), req))
	res, promoted, err := (&Service{}).pickChallenge(context.Background(), req, comp, rules.Facts{}, &rules.UserHistory{})
	require.NoError(t, err)
	assert.Nil(t, promoted)
	assert.Equal(t, runs, src.runs, "the runs come from the request (the profile row), not a query")
	b, err := (&Result{Challenge: res}).JSON("en").MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"daily_challenge":{"id":5,"title":"Stretch","description":null,"category":null,"duration_days":7,"day":2,"reason":"ongoing"}`)

	// blocked after a safety message: the key is present and null
	res, _, err = (&Service{}).pickChallenge(context.Background(), req, comp, rules.Facts{}, &rules.UserHistory{LastSafetyShown: req.Now.Add(-time.Hour)})
	require.NoError(t, err)
	b, _ = (&Result{Challenge: res}).JSON("en").MarshalJSON()
	assert.Contains(t, string(b), `"daily_challenge":null`)

	// off: slot not requested, or no runs hook (v1), or nothing installed → no key at all
	for _, r := range []Request{{Slots: []string{"smart_tip"}, ChallengeRuns: req.ChallengeRuns}, {Slots: req.Slots}} {
		res, _, err = (&Service{}).pickChallenge(context.Background(), r, comp, rules.Facts{}, nil)
		require.NoError(t, err)
		assert.Nil(t, res)
		b, _ = (&Result{Challenge: res}).JSON("en").MarshalJSON()
		assert.NotContains(t, string(b), "daily_challenge\":{")
		assert.NotContains(t, string(b), `"daily_challenge":null`)
	}
}

func TestParseChallengeRuns(t *testing.T) {
	want := []ChallengeRun{{ChallengeID: 3, StartedOn: "2026-10-01", Status: "completed"}}
	assert.Equal(t, want, ParseChallengeRuns([]byte(`[{"id": 3, "started_on": "2026-10-01", "status": "completed"}]`)))
	assert.Equal(t, want, ParseChallengeRuns(`[{"id": 3, "started_on": "2026-10-01", "status": "completed"}]`))
	assert.Nil(t, ParseChallengeRuns(nil))
	assert.Nil(t, ParseChallengeRuns([]byte("oops")))
}
