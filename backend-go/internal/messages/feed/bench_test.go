package feed

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/events"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// fakeQ answers the two reads Build makes (history, inbox) from memory; every other method is nil (unused).
type fakeQ struct {
	store.Querier
	history []store.ListUserMessageEventStatsRow
}

func (f *fakeQ) ListUserMessageEventStats(context.Context, store.ListUserMessageEventStatsParams) ([]store.ListUserMessageEventStatsRow, error) {
	return f.history, nil
}

func (f *fakeQ) CountUserOpenInbox(context.Context, uint64) (store.CountUserOpenInboxRow, error) {
	return store.CountUserOpenInboxRow{}, nil
}

func (f *fakeQ) ListActiveMessageRules(context.Context, store.ListActiveMessageRulesParams) ([]store.MessageRule, error) {
	return nil, nil
}

type nopIns struct{}

func (nopIns) InsertMessageEventsBatch(context.Context, store.InsertMessageEventsBatchParams) error {
	return nil
}

// benchRules is 200 served rules over the six screen slots, each with an engine-fact condition.
func benchRules() []rules.Rule {
	kinds := [][3]string{{"urgent_modal", rules.KindSafety, rules.LevelFollowSoon}, {"smart_tip", rules.KindClinical, rules.LevelInfo},
		{"daily_advice", rules.KindContent, rules.LevelInfo}, {"daily_challenge", rules.KindProduct, rules.LevelSuggestion},
		{"modal", rules.KindMarketing, rules.LevelInfo}, {"status_banner", rules.KindEngine, rules.LevelInfo}}
	out := make([]rules.Rule, 0, 200)
	for i := range 200 {
		k := kinds[i%len(kinds)]
		lo, hi := rules.Band(k[1])
		out = append(out, rules.Rule{ID: uint64(i + 1), Kind: k[1], Level: k[2], Placement: k[0], Priority: lo + i%(hi-lo+1), //nolint:gosec // small
			Status: rules.StatusApproved, IsActive: true, Version: 1, AudiencePercent: 100, HoursFrom: -1, HoursTo: -1,
			Content: rules.Content{A: rules.Variant{Title: fmt.Sprintf("t%d {cycle_day}", i), Body: "b", CTALabel: "go", CTATarget: "ritme://x"}},
			Groups:  []rules.Group{{Op: "and", Items: []rules.Condition{{Fact: "cycle_day", Op: "gte", Value: float64(i % 30)}}}}})
	}
	return out
}

// BenchmarkFeedBuild_200Rules is the in-process part of GET /messages/feed (selection of six slots, the parsed
// eligible pass, rendering, the event marks without Redis) with 200 rules and a warm snapshot; the DB reads are
// in-memory fakes (the int test TestFeedInt_Budget measures the whole request).
func BenchmarkFeedBuild_200Rules(b *testing.B) {
	comp := compiled.CompileRules(benchRules())
	q := &fakeQ{}
	w := events.NewWriter(nopIns{}, nil)
	defer w.Close(context.Background())
	s := New(q, w, nil, nil).WithLoader(func(context.Context) (*compiled.Compiled, error) { return comp, nil })
	f := rules.Facts{"cycle_day": float64(17), "confidence": "high", "current_period_start": "2026-10-05"}
	now := time.Date(2026, 10, 21, 10, 0, 0, 0, civildate.Tehran)
	req := Request{UserID: 42, Slots: DefaultSlots, Locale: "en", Now: now, Alerts: true,
		Facts: func(context.Context, []string) (rules.Facts, error) { return f, nil }}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := s.Build(context.Background(), req); err != nil {
			b.Fatal(err)
		}
	}
}
