package babylog

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"math"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/babylog/store"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// The controller messages and validation attribute names are data: lang/<code>/babylog.json (English fallback).
//
//go:embed lang/*/*.json
var langFS embed.FS

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err) // embedded files are checked by the package tests
	}
	return t
})

// T is the babylog line for key ("messages.saved") in locale.
func T(key, locale string) string { return translator().Trans("babylog."+key, nil, locale) }

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	line, ok := translator().Get("babylog.attributes", locale)
	m, isMap := line.(phpval.Map)
	if !ok || !isMap {
		return nil
	}
	var kv []string
	for _, k := range m.Keys() {
		if v, ok := m.Get(k); ok {
			if s, ok := v.(string); ok {
				kv = append(kv, k, s)
			}
		}
	}
	return kv
}

func iso(t time.Time) any { return jsonx.ISO8601(t.In(civildate.Tehran)) }

func isoNull(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return iso(t.Time)
}

func strNull(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	return s.String
}

func mlNull(n sql.NullInt16) any {
	if !n.Valid {
		return nil
	}
	return int(n.Int16)
}

// FeedJSON is one feed at now. left_seconds / right_seconds are the closed side segments; a running breast feed's
// open segment is active_side since side_started_at (duration_seconds includes it).
func FeedJSON(f store.BabyFeed, now time.Time) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", f.ID,
		"type", f.Type,
		"started_at", iso(f.StartedAt),
		"ended_at", isoNull(f.EndedAt),
		"is_active", f.ActiveLock.Valid,
		"active_side", strNull(f.ActiveSide),
		"side_started_at", isoNull(f.SideStartedAt),
		"last_side", strNull(f.LastSide),
		"left_seconds", int64(f.LeftSeconds),
		"right_seconds", int64(f.RightSeconds),
		"duration_seconds", TimesOf(f, now).Total,
		"amount_ml", mlNull(f.AmountMl),
		"note", strNull(f.Note),
	)
}

func feedNull(f *store.BabyFeed, now time.Time) any {
	if f == nil {
		return nil
	}
	return FeedJSON(*f, now)
}

// SleepJSON is one sleep at now.
func SleepJSON(s store.BabySleep, now time.Time) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", s.ID,
		"started_at", iso(s.StartedAt),
		"ended_at", isoNull(s.EndedAt),
		"is_active", s.ActiveLock.Valid,
		"duration_seconds", SleepSeconds(s, now),
		"note", strNull(s.Note),
	)
}

func sleepNull(s *store.BabySleep, now time.Time) any {
	if s == nil {
		return nil
	}
	return SleepJSON(*s, now)
}

// DiaperJSON is one diaper change.
func DiaperJSON(d store.BabyDiaper) *jsonx.OrderedMap {
	return jsonx.Obj("id", d.ID, "changed_at", iso(d.ChangedAt), "kind", d.Kind, "note", strNull(d.Note))
}

// FeedTotalsJSON are a day's feeds.
func FeedTotalsJSON(t FeedTotals) *jsonx.OrderedMap {
	return jsonx.Obj(
		"count", t.Count, "breast", t.Breast, "bottle", t.Bottle, "pump", t.Pump,
		"left_seconds", t.LeftSeconds, "right_seconds", t.RightSeconds, "total_seconds", t.TotalSeconds,
		"bottle_ml", t.BottleMl, "pump_ml", t.PumpMl,
	)
}

// SleepTotalsJSON are a day's sleep.
func SleepTotalsJSON(t SleepTotals) *jsonx.OrderedMap {
	return jsonx.Obj("count", t.Count, "seconds", t.Seconds, "longest_seconds", t.LongestSeconds)
}

// DiaperTotalsJSON are a day's diapers.
func DiaperTotalsJSON(t DiaperTotals) *jsonx.OrderedMap {
	return jsonx.Obj("count", t.Count, "wet", t.Wet, "dirty", t.Dirty, "both", t.Both)
}

// DayJSON is one day summary.
func DayJSON(d Day) *jsonx.OrderedMap {
	return jsonx.Obj(
		"date", d.Date.String(),
		"feeds", FeedTotalsJSON(d.Feeds),
		"sleep", SleepTotalsJSON(d.Sleep),
		"diapers", DiaperTotalsJSON(d.Diapers),
	)
}

func round1(f float64) jsonx.Float { return jsonx.Float(math.Round(f*10) / 10) }

// AveragesJSON are the per-day means over days and the breast side split (percent of breast time; null without
// breast time).
func AveragesJSON(days []Day) *jsonx.OrderedMap {
	var feeds, diapers int
	var sleep, left, right int64
	for _, d := range days {
		feeds += d.Feeds.Count
		diapers += d.Diapers.Count
		sleep += d.Sleep.Seconds
		left += d.Feeds.LeftSeconds
		right += d.Feeds.RightSeconds
	}
	n := float64(max(1, len(days)))
	var lp, rp any
	if left+right > 0 {
		l := int(math.Round(float64(left) * 100 / float64(left+right)))
		lp, rp = l, 100-l
	}
	return jsonx.Obj(
		"feeds_per_day", round1(float64(feeds)/n),
		"sleep_seconds_per_day", int64(math.Round(float64(sleep)/n)),
		"diapers_per_day", round1(float64(diapers)/n),
		"left_percent", lp,
		"right_percent", rp,
	)
}

// Today is the child home's «امروز» card (children.TodayProvider): today's feeds (with the last ended feed and
// whether one is running), sleep and diapers.
func (s *Service) Today(ctx context.Context, childID uint64, now time.Time) (any, error) {
	today := civildate.InTehran(now)
	days, err := s.Days(ctx, childID, today, 1, now)
	if err != nil {
		return nil, err
	}
	last, err := s.LastFeed(ctx, childID)
	if err != nil {
		return nil, err
	}
	feed, err := s.ActiveFeed(ctx, childID)
	if err != nil {
		return nil, err
	}
	sleep, err := s.ActiveSleep(ctx, childID)
	if err != nil {
		return nil, err
	}
	out := DayJSON(days[0])
	feeds, _ := out.Get("feeds")
	if m, ok := feeds.(*jsonx.OrderedMap); ok {
		m.Set("last", feedNull(last, now))
	}
	out.Set("feeding_now", feed != nil)
	out.Set("sleeping_now", sleep != nil)
	return out, nil
}
