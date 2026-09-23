package home

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/crc32"
	"slices"
	"strconv"
	"time"

	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/home/store"
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

// challengeSeed is crc32($user->id.':'.$date->toDateString()).
func challengeSeed(userID uint64, date civildate.Date) uint32 {
	return crc32.ChecksumIEEE([]byte(strconv.FormatUint(userID, 10) + ":" + date.String()))
}

// ChallengePayload is DailyChallengeService::payload(): nil when no challenge is available.
func ChallengePayload(ctx context.Context, q store.Querier, userID uint64, date civildate.Date, locale, defLocale string,
	cycleDay *int, recentLogs []cyclestore.DailyHealthLog,
) (*jsonx.OrderedMap, error) {
	c, ok, err := challengeFor(ctx, q, userID, date, cycleDay, recentLogs)
	if err != nil || !ok {
		return nil, err
	}
	done, err := q.ChallengeCompletedOn(ctx, store.ChallengeCompletedOnParams{UserID: userID, ChallengeID: c.ID, CompletionDate: date})
	if err != nil {
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

func challengeFor(ctx context.Context, q store.Querier, userID uint64, date civildate.Date, cycleDay *int,
	recentLogs []cyclestore.DailyHealthLog,
) (challengeRow, bool, error) {
	pool, err := q.ListActiveChallenges(ctx)
	if err != nil {
		return challengeRow{}, false, fmt.Errorf("home: challenges: %w", err)
	}
	if len(pool) == 0 {
		return challengeRow{}, false, nil
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
	recent, err := q.ListChallengeIDsCompletedBetween(ctx, store.ListChallengeIDsCompletedBetweenParams{
		UserID: userID, FromDate: date.AddDays(-challengeRepeatWindowDays), BeforeDate: date,
	})
	if err != nil {
		return challengeRow{}, false, fmt.Errorf("home: recent challenge completions: %w", err)
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

	return pool[int(challengeSeed(userID, date)%uint32(len(pool)))], true, nil //nolint:gosec // G115: len > 0
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
