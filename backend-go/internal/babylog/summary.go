package babylog

import (
	"time"

	"github.com/ritme/backend-go/internal/babylog/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Codes.
var (
	FeedTypes   = []string{"breast", "bottle", "pump"}
	Sides       = []string{"left", "right"}
	DiaperKinds = []string{"wet", "dirty", "both"}
)

const typeBreast = "breast"

func span(from, to time.Time) int64 {
	if !to.After(from) {
		return 0
	}
	return int64(to.Sub(from) / time.Second)
}

// FeedTimes are a feed's per-side and total seconds at now (the running side segment included).
type FeedTimes struct {
	Left, Right, Total int64
}

// TimesOf works out a feed's seconds at now: breast = left + right (+ the running segment); bottle / pump = the
// stored duration once ended, else now − started.
func TimesOf(f store.BabyFeed, now time.Time) FeedTimes {
	t := FeedTimes{Left: int64(f.LeftSeconds), Right: int64(f.RightSeconds)}
	if f.ActiveLock.Valid && f.ActiveSide.Valid && f.SideStartedAt.Valid {
		run := span(f.SideStartedAt.Time, now)
		if f.ActiveSide.String == "left" {
			t.Left += run
		} else {
			t.Right += run
		}
	}
	switch {
	case f.Type == typeBreast:
		t.Total = t.Left + t.Right
	case f.DurationSeconds.Valid:
		t.Total = int64(f.DurationSeconds.Int32)
	case f.EndedAt.Valid:
		t.Total = span(f.StartedAt, f.EndedAt.Time)
	default:
		t.Total = span(f.StartedAt, now)
	}
	return t
}

// SleepSeconds is a sleep's length at now (running sleeps count up to now).
func SleepSeconds(s store.BabySleep, now time.Time) int64 {
	end := now
	if s.EndedAt.Valid {
		end = s.EndedAt.Time
	}
	return span(s.StartedAt, end)
}

// FeedTotals are a day's feeds (by start day).
type FeedTotals struct {
	Count, Breast, Bottle, Pump int
	LeftSeconds, RightSeconds   int64
	TotalSeconds                int64
	BottleMl, PumpMl            int
}

// SleepTotals are a day's baby sleep: Seconds is the time asleep inside the day (sleeps crossing midnight are split),
// Count / Longest the sleeps started that day.
type SleepTotals struct {
	Count          int
	Seconds        int64
	LongestSeconds int64
}

// DiaperTotals are a day's diapers; a «both» counts as wet and as dirty.
type DiaperTotals struct {
	Count, Wet, Dirty, Both int
}

// Day is one day of baby logs.
type Day struct {
	Date    civildate.Date
	Feeds   FeedTotals
	Sleep   SleepTotals
	Diapers DiaperTotals
}

// Summarize buckets the rows into days [from, from+days) (Tehran days) at now.
func Summarize(from civildate.Date, days int, feeds []store.BabyFeed, sleeps []store.BabySleep, diapers []store.BabyDiaper, now time.Time) []Day {
	out := make([]Day, days)
	index := map[civildate.Date]int{}
	for i := range out {
		out[i].Date = from.AddDays(i)
		index[out[i].Date] = i
	}
	for _, f := range feeds {
		i, ok := index[civildate.InTehran(f.StartedAt)]
		if !ok {
			continue
		}
		d := &out[i].Feeds
		t := TimesOf(f, now)
		d.Count++
		d.TotalSeconds += t.Total
		switch f.Type {
		case typeBreast:
			d.Breast++
			d.LeftSeconds += t.Left
			d.RightSeconds += t.Right
		case "bottle":
			d.Bottle++
			d.BottleMl += int(f.AmountMl.Int16)
		case "pump":
			d.Pump++
			d.PumpMl += int(f.AmountMl.Int16)
		}
	}
	for _, s := range sleeps {
		end := now
		if s.EndedAt.Valid {
			end = s.EndedAt.Time
		}
		if i, ok := index[civildate.InTehran(s.StartedAt)]; ok {
			out[i].Sleep.Count++
			out[i].Sleep.LongestSeconds = max(out[i].Sleep.LongestSeconds, span(s.StartedAt, end))
		}
		for i := range out {
			lo, hi := out[i].Date.TehranMidnight(), out[i].Date.AddDays(1).TehranMidnight()
			a, b := s.StartedAt, end
			if a.Before(lo) {
				a = lo
			}
			if b.After(hi) {
				b = hi
			}
			out[i].Sleep.Seconds += span(a, b)
		}
	}
	for _, p := range diapers {
		i, ok := index[civildate.InTehran(p.ChangedAt)]
		if !ok {
			continue
		}
		d := &out[i].Diapers
		d.Count++
		switch p.Kind {
		case "wet":
			d.Wet++
		case "dirty":
			d.Dirty++
		case "both":
			d.Both++
			d.Wet++
			d.Dirty++
		}
	}
	return out
}

// NextSide is the side to offer first: the other one than lastSide (the side the last breast feed ended on; ""
// when unknown).
func NextSide(lastSide string) string {
	switch lastSide {
	case "left":
		return "right"
	case "right":
		return "left"
	}
	return ""
}
