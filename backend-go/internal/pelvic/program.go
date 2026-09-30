// Package pelvic is the pelvic floor program (CB-PELV-01, boards nbl_Pelvic_Plan / nbl_Pelvic_Kegel): the user's
// 8-week Kegel program (week, level, streak, this week's trained days), the trained days themselves and the
// bladder diary (leak, night voids, UTI symptoms). Go only (deviations.md D-32).
//
//	GET    /api/v1/pelvic                the plan screen: program overview + today's diary
//	POST   /api/v1/pelvic/program        start (or restart) the program
//	DELETE /api/v1/pelvic/program        stop it (trained days and diary are kept)
//	POST   /api/v1/pelvic/sessions       save a Kegel session (adds to the day)
//	GET    /api/v1/pelvic/diary/{date}   one bladder-diary day
//	PUT    /api/v1/pelvic/diary/{date}   partial update of it
//
// Levels are admin-editable catalog content (group `pelvic_levels`, meta {week_from, hold_sec, rest_sec, reps,
// sets}); the program week picks the level. Health data: every query is scoped by user_id, nothing is sent to
// analytics.
package pelvic

import (
	"encoding/json"
	"sort"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Program constants and the catalog groups the package reads.
const (
	WeeksTotal  = 8
	LevelsGroup = "pelvic_levels"
	// streakWindow is how far back the streak looks (days); a longer streak is reported as this many days.
	streakWindow = 400
)

// Leak answers of the bladder diary («نشت ادرار داشتی؟»).
var Leaks = []string{"none", "cough", "urgency", "unexplained"}

// UTISymptoms of the bladder diary, in display order.
var UTISymptoms = []string{"burning", "frequency", "cloudy_odor"}

// Night voids bounds (per night).
const maxNightVoids = 20

// Level is one pelvic_levels catalog item with its parsed meta.
type Level struct {
	Item     catalog.Item
	Number   int // 1-based position among the valid levels
	WeekFrom int
	HoldSec  int
	RestSec  int
	Reps     int
	Sets     int
}

// SessionSec is the guided session's length: sets × reps × (hold + rest).
func (l Level) SessionSec() int { return l.Sets * l.Reps * (l.HoldSec + l.RestSec) }

type levelMeta struct {
	WeekFrom int `json:"week_from"`
	HoldSec  int `json:"hold_sec"`
	RestSec  int `json:"rest_sec"`
	Reps     int `json:"reps"`
	Sets     int `json:"sets"`
}

// ParseLevels keeps the items whose meta has positive integer week_from / hold_sec / rest_sec / reps / sets,
// ordered by week_from (catalog order breaks ties), numbered from 1. Items with other meta are skipped, so an
// admin typo hides one level instead of breaking the screen.
func ParseLevels(items []catalog.Item) []Level {
	out := make([]Level, 0, len(items))
	for _, it := range items {
		var m levelMeta
		if len(it.Meta) == 0 || json.Unmarshal(it.Meta, &m) != nil {
			continue
		}
		if m.WeekFrom < 1 || m.HoldSec < 1 || m.RestSec < 1 || m.Reps < 1 || m.Sets < 1 {
			continue
		}
		out = append(out, Level{Item: it, WeekFrom: m.WeekFrom, HoldSec: m.HoldSec, RestSec: m.RestSec, Reps: m.Reps, Sets: m.Sets})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].WeekFrom < out[j].WeekFrom })
	for i := range out {
		out[i].Number = i + 1
	}
	return out
}

// LevelFor is the level of a program week: the last level whose week_from ≤ week, else the first one; nil when
// there is none.
func LevelFor(levels []Level, week int) *Level {
	if len(levels) == 0 {
		return nil
	}
	pick := 0
	for i, l := range levels {
		if l.WeekFrom <= week {
			pick = i
		}
	}
	return &levels[pick]
}

// Week is the program week of today (1-based, capped at WeeksTotal) and whether the 8 weeks are over.
func Week(startedOn, today civildate.Date) (week int, completed bool) {
	days := startedOn.DiffDays(today)
	if days < 0 {
		days = 0
	}
	week = days/7 + 1
	if week > WeeksTotal {
		return WeeksTotal, true
	}
	return week, false
}

// Streak counts consecutive trained days ending today, or yesterday when today has no session yet.
func Streak(done map[civildate.Date]bool, today civildate.Date) int {
	d := today
	if !done[d] {
		d = d.AddDays(-1)
	}
	n := 0
	for done[d] && n < streakWindow {
		n++
		d = d.AddDays(-1)
	}
	return n
}

// WeekDay is one dot of the Saturday-to-Friday strip.
type WeekDay struct {
	Date civildate.Date
	Done bool
}

// WeekDays is today's Saturday-start week with the trained days marked.
func WeekDays(done map[civildate.Date]bool, today civildate.Date) []WeekDay {
	start := today.StartOfWeek()
	out := make([]WeekDay, 7)
	for i := range out {
		d := start.AddDays(i)
		out[i] = WeekDay{Date: d, Done: done[d]}
	}
	return out
}
