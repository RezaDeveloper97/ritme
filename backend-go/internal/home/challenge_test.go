package home

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/challenges"
	"github.com/ritme/backend-go/internal/messages/feed"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// T-M10-10: the feed's daily_challenge pick wins; the legacy pick (which reads the store) is not touched then.
func TestDailyChallengeCard_FeedPickWins(t *testing.T) {
	day := 3
	pick := &FeedChallenge{ID: 9, Title: json.RawMessage(`{"fa":"کشش","en":"Stretch"}`), Category: sql.NullString{String: "exercise", Valid: true},
		Reason: "ongoing", DoneToday: true, Run: jsonx.Obj("id", 9)}
	// q is nil: a legacy read would panic.
	got, err := DailyChallengeCard(context.Background(), nil, pick, 1, civildate.MustParse("2026-10-06"), "en", "fa", &day, nil, true)
	require.NoError(t, err)
	b, err := got.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":9,"title":"Stretch","description":null,"category":"exercise","cycle_day":3,"cycle_day_from":null,
		"cycle_day_to":null,"is_completed":true,"source":"feed","reason":"ongoing","challenge":{"id":9}}`, string(b))
}

type fakeCards struct {
	cands []feed.ChallengeCandidate
	items map[uint64]challenges.Item
}

func (f fakeCards) HomeCandidates(context.Context, uint64, civildate.Date) ([]feed.ChallengeCandidate, map[uint64]challenges.Item, error) {
	return f.cands, f.items, nil
}

func TestDailyChallengePick(t *testing.T) {
	ctx, today := context.Background(), civildate.MustParse("2026-10-06")
	p, err := DailyChallengePick(ctx, nil, 1, today, "en", "fa")
	require.NoError(t, err)
	assert.Nil(t, p, "no port: legacy")
	p, _ = DailyChallengePick(ctx, fakeCards{}, 1, today, "en", "fa")
	assert.Nil(t, p, "no v2 challenge: legacy")

	title := json.RawMessage(`{"fa":"کشش","en":"Stretch"}`)
	items := map[uint64]challenges.Item{
		1: {Program: challenges.Program{ID: 1, Title: title, DurationDays: 3}, CanJoin: true},
		2: {Program: challenges.Program{ID: 2, Title: title, DurationDays: 3, Category: "exercise"},
			Enrollment: &challenges.EnrollmentView{Status: "active", Day: 2, DoneToday: map[uint64]bool{5: true, 6: true}}},
	}
	cards := fakeCards{cands: []feed.ChallengeCandidate{{ID: 1, Eligible: true}, {ID: 2, Ongoing: true}}, items: items}
	p, err = DailyChallengePick(ctx, cards, 1, today, "en", "fa")
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, uint64(2), p.ID, "the ongoing run wins")
	assert.Equal(t, "ongoing", p.Reason)
	assert.True(t, p.DoneToday)
	assert.Equal(t, "exercise", p.Category.String)

	cards.cands[1].Contraindicated = true
	p, _ = DailyChallengePick(ctx, cards, 1, today, "en", "fa")
	assert.Equal(t, uint64(1), p.ID, "a contraindicated run is never the card")
	assert.False(t, p.DoneToday)
}
