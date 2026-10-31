package home

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"slices"
	"strconv"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/challenges"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/home/store"
	"github.com/ritme/backend-go/internal/messages/feed"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// DailyChallengeService (backend/app/Services/Challenges/DailyChallengeService.php): one
// admin-authored challenge per user and day — cycle-day filter (targeted ones first), no repeats
// of the last 14 days, the recent-log signal category, then crc32("{uid}:{Y-m-d}") % count.
// Every filter is soft: it is skipped when it would empty the pool.

const (
	challengeRepeatWindowDays = 14
	challengeSignalWindowDays = 3
)

type challengeRow = store.ListActiveChallengesRow

// isDayTargeted is Challenge::isDayTargeted().
func isDayTargeted(c challengeRow) bool { return c.CycleDayFrom.Valid || c.CycleDayTo.Valid }

// appliesToCycleDay is Challenge::appliesToCycleDay().
func appliesToCycleDay(c challengeRow, day *int) bool {
	if !isDayTargeted(c) {
		return true
	}
	if day == nil {
		return false
	}
	return (!c.CycleDayFrom.Valid || *day >= int(c.CycleDayFrom.Int16)) &&
		(!c.CycleDayTo.Valid || *day <= int(c.CycleDayTo.Int16))
}

// beyondChallengeContent reports whether day lies after the last cycle day any day-targeted challenge covers (false
// without day-targeted challenges or when one is open-ended).
func beyondChallengeContent(pool []challengeRow, day int) bool {
	last, targeted := 0, false
	for _, c := range pool {
		if !isDayTargeted(c) {
			continue
		}
		if !c.CycleDayTo.Valid {
			return false
		}
		targeted = true
		last = max(last, int(c.CycleDayTo.Int16))
	}
	return targeted && day > last
}

// challengeSeed is crc32($user->id.':'.$date->toDateString()).
func challengeSeed(userID uint64, date civildate.Date) uint32 {
	return crc32.ChecksumIEEE([]byte(strconv.FormatUint(userID, 10) + ":" + date.String()))
}

// ChallengePayload is DailyChallengeService::payload(): nil when no challenge is available.
func ChallengePayload(ctx context.Context, q store.Querier, userID uint64, date civildate.Date, locale, defLocale string,
	cycleDay *int, recentLogs []cyclestore.DailyHealthLog, strictDay bool,
) (*jsonx.OrderedMap, error) {
	c, ok, today, err := challengeFor(ctx, q, userID, date, cycleDay, recentLogs, strictDay)
	if err != nil || !ok {
		return nil, err
	}
	var done bool
	if strictDay {
		done = slices.Contains(today, c.ID) // v6: the window read already holds the day's completions (T-M9-34b)
	} else if done, err = q.ChallengeCompletedOn(ctx, store.ChallengeCompletedOnParams{UserID: userID, ChallengeID: c.ID, CompletionDate: date}); err != nil {
		return nil, fmt.Errorf("home: challenge completion: %w", err)
	}
	return jsonx.Obj(
		"id", c.ID,
		"title", localized(c.Title, locale, defLocale),
		"description", localizedNull(c.Description, locale, defLocale),
		"category", nullString(c.Category),
		"cycle_day", intOrNil(cycleDay),
		"cycle_day_from", nullInt16(c.CycleDayFrom),
		"cycle_day_to", nullInt16(c.CycleDayTo),
		"is_completed", done,
	), nil
}

// FeedChallenge is the daily_challenge slot's v2 pick (T-M10-10, D-167: messages/feed.PickChallenge over the
// challenges catalogue), rendered by the caller from the challenge and the user's run.
type FeedChallenge struct {
	ID          uint64
	Title       json.RawMessage
	Description db.NullRawJSON
	Category    sql.NullString
	// Reason is the pick reason (ongoing | promoted | product).
	Reason string
	// DoneToday: every task of today's day of the run is done (false before joining).
	DoneToday bool
	// Run is the v2 challenge object (GET /challenges item) the card links to.
	Run *jsonx.OrderedMap
}

// ChallengeCards is the challenges v2 port of the home (challenges.Service).
type ChallengeCards interface {
	HomeCandidates(ctx context.Context, userID uint64, today civildate.Date) ([]feed.ChallengeCandidate, map[uint64]challenges.Item, error)
}

// DailyChallengePick is the daily_challenge pick of the home until it reads the feed (T-M10-10): feed.PickChallenge
// over the user's challenge list (an ongoing run first, else the first joinable challenge in admin sort order; never
// a contraindicated one). nil = no v2 candidate (the legacy pick fills the card) or no port. The post-safety block
// and the challenge_id promotions need the message history and rules, which only the feed has.
func DailyChallengePick(ctx context.Context, svc ChallengeCards, userID uint64, today civildate.Date, locale, defLocale string,
) (*FeedChallenge, error) {
	if svc == nil {
		return nil, nil
	}
	cands, items, err := svc.HomeCandidates(ctx, userID, today)
	if err != nil || len(cands) == 0 {
		return nil, err
	}
	pick, ok := feed.PickChallenge(feed.ChallengeInput{Candidates: cands})
	if !ok {
		return nil, nil
	}
	it := items[pick.ID]
	done := false
	if e := it.Enrollment; e != nil && e.Status == challenges.StatusActive && len(e.DoneToday) > 0 {
		done = true
		for _, d := range e.DoneToday {
			done = done && d
		}
	}
	cat := sql.NullString{String: it.Program.Category, Valid: it.Program.Category != ""}
	return &FeedChallenge{ID: pick.ID, Title: it.Program.Title, Description: it.Program.Description, Category: cat,
		Reason: pick.Reason, DoneToday: done, Run: challenges.ItemJSON(it, locale, defLocale)}, nil
}

// DailyChallengeCard is the home challenge card (T-M10-10): the feed's daily_challenge pick wins; the legacy
// DailyChallengeService pick (ChallengePayload) is read only when the feed has no candidate (pick = nil). A v2 card
// keeps the legacy keys (cycle_day_from / _to null: a v2 challenge is not day-targeted) and adds source, reason and
// challenge.
func DailyChallengeCard(ctx context.Context, q store.Querier, pick *FeedChallenge, userID uint64, date civildate.Date,
	locale, defLocale string, cycleDay *int, recentLogs []cyclestore.DailyHealthLog, strictDay bool,
) (*jsonx.OrderedMap, error) {
	if pick == nil {
		return ChallengePayload(ctx, q, userID, date, locale, defLocale, cycleDay, recentLogs, strictDay)
	}
	return jsonx.Obj(
		"id", pick.ID,
		"title", localized(pick.Title, locale, defLocale),
		"description", localizedNull(pick.Description, locale, defLocale),
		"category", nullString(pick.Category),
		"cycle_day", intOrNil(cycleDay),
		"cycle_day_from", nil,
		"cycle_day_to", nil,
		"is_completed", pick.DoneToday,
		"source", "feed",
		"reason", pick.Reason,
		"challenge", pick.Run,
	), nil
}

// challengeFor is DailyChallengeService::challengeFor(). strictDay is the v6 path: besides the day rule it reads the repeat window and the day's completions in one query
// and returns the challenge ids completed on date (today); v1 leaves today nil and keeps its two reads.
func challengeFor(ctx context.Context, q store.Querier, userID uint64, date civildate.Date, cycleDay *int,
	recentLogs []cyclestore.DailyHealthLog, strictDay bool,
) (picked challengeRow, ok bool, today []uint64, err error) {
	pool, err := activeChallenges(ctx, q)
	if err != nil {
		return challengeRow{}, false, nil, err
	}
	if len(pool) == 0 {
		return challengeRow{}, false, nil, nil
	}
	// v6 (T-M9-34): the cycle day never wraps, so a late cycle can run past the authored content. A day beyond every
	// day-targeted challenge gets no daily challenge (the admin validator should warn about uncovered days).
	if strictDay && cycleDay != nil && beyondChallengeContent(pool, *cycleDay) {
		return challengeRow{}, false, nil, nil
	}

	// narrowToCycleDay
	eligible := filter(pool, func(c challengeRow) bool { return appliesToCycleDay(c, cycleDay) })
	if len(eligible) > 0 {
		if targeted := filter(eligible, isDayTargeted); len(targeted) > 0 {
			pool = targeted
		} else {
			pool = eligible
		}
	}

	// withoutRecentlyCompleted (today excluded)
	recent, today, err := challengeCompletions(ctx, q, userID, date, strictDay)
	if err != nil {
		return challengeRow{}, false, nil, err
	}
	if kept := filter(pool, func(c challengeRow) bool { return !slices.Contains(recent, c.ID) }); len(kept) > 0 {
		pool = kept
	}

	// preferSignalCategory
	if cat := signalCategory(date, recentLogs); cat != "" {
		if kept := filter(pool, func(c challengeRow) bool { return c.Category.Valid && c.Category.String == cat }); len(kept) > 0 {
			pool = kept
		}
	}

	return pool[int(challengeSeed(userID, date)%uint32(len(pool)))], true, today, nil //nolint:gosec // G115: len > 0
}

// challengeCompletions returns the challenge ids completed in [date − 14, date) (recent) and, with merged (v6), the
// ids completed on date from the same read; v1 reads only the window (ListChallengeIDsCompletedBetween).
func challengeCompletions(ctx context.Context, q store.Querier, userID uint64, date civildate.Date, merged bool,
) (recent, today []uint64, err error) {
	from := date.AddDays(-challengeRepeatWindowDays)
	if !merged {
		recent, err = q.ListChallengeIDsCompletedBetween(ctx, store.ListChallengeIDsCompletedBetweenParams{
			UserID: userID, FromDate: from, BeforeDate: date,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("home: recent challenge completions: %w", err)
		}
		return recent, nil, nil
	}
	rows, err := q.ListChallengeCompletionsSince(ctx, store.ListChallengeCompletionsSinceParams{UserID: userID, FromDate: from, Day: date})
	if err != nil {
		return nil, nil, fmt.Errorf("home: challenge completions: %w", err)
	}
	recent = []uint64{}
	for _, r := range rows {
		if r.CompletionDate == date {
			today = append(today, r.ChallengeID)
		} else {
			recent = append(recent, r.ChallengeID)
		}
	}
	return recent, today, nil
}

// signalCategory names the category the last three days of logs point at ("" = none).
func signalCategory(date civildate.Date, logs []cyclestore.DailyHealthLog) string {
	since := date.AddDays(-(challengeSignalWindowDays - 1))
	var recent []cyclestore.DailyHealthLog
	for _, l := range logs {
		if !l.LogDate.Before(since) {
			recent = append(recent, l)
		}
	}
	if len(recent) == 0 {
		return "tracking"
	}
	is := func(v sql.NullString, vals ...string) bool { return v.Valid && slices.Contains(vals, v.String) }
	for _, l := range recent {
		if is(l.SleepQuality, "bad") || is(l.SleepDuration, "0_3", "3_6") {
			return "sleep"
		}
	}
	for _, l := range recent {
		for _, p := range []sql.NullString{l.HeadacheIntensity, l.StomachAcheIntensity, l.PelvicPainIntensity, l.BackPainIntensity} {
			if is(p, "high", "medium") {
				return "mindfulness"
			}
		}
		for _, m := range scoreView(l).Moods {
			if slices.Contains([]string{"anxious", "sad", "angry", "frustrated"}, m) {
				return "mindfulness"
			}
		}
	}
	for _, l := range recent {
		if is(l.EnergyLevel, "very_low", "low") {
			return "nutrition"
		}
	}
	for _, l := range recent {
		if is(l.EnergyLevel, "high", "very_high") {
			return "exercise"
		}
	}
	return ""
}

// ToggleChallenge is DailyChallengeService::toggle(): flip the day's completion.
func ToggleChallenge(ctx context.Context, q store.Querier, userID, challengeID uint64, date civildate.Date, now time.Time) (bool, error) {
	id, err := q.GetChallengeCompletionOn(ctx, store.GetChallengeCompletionOnParams{UserID: userID, ChallengeID: challengeID, CompletionDate: date})
	switch {
	case err == nil:
		if err := q.DeleteChallengeCompletion(ctx, id); err != nil {
			return false, fmt.Errorf("home: delete challenge completion: %w", err)
		}
		return false, nil
	case !errors.Is(err, sql.ErrNoRows):
		return false, fmt.Errorf("home: challenge completion: %w", err)
	}
	ts := dbNow(now)
	err = q.InsertChallengeCompletion(ctx, store.InsertChallengeCompletionParams{
		UserID: userID, ChallengeID: challengeID, CompletionDate: date,
		CompletedAt: sql.NullTime{Time: ts, Valid: true},
		CreatedAt:   sql.NullTime{Time: ts, Valid: true},
		UpdatedAt:   sql.NullTime{Time: ts, Valid: true},
	})
	if err != nil {
		return false, fmt.Errorf("home: insert challenge completion: %w", err)
	}
	return true, nil
}

func filter[T any](in []T, keep func(T) bool) []T {
	out := make([]T, 0, len(in))
	for _, x := range in {
		if keep(x) {
			out = append(out, x)
		}
	}
	return out
}

// dbNow is now() as Eloquent writes it: Tehran wall clock, whole seconds.
func dbNow(now time.Time) time.Time { return now.In(civildate.Tehran).Truncate(time.Second) }
