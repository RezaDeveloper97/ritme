// Package notifications owns the user's notification settings (B-N1-11): which reminder categories may be pushed,
// a quiet-hours window and «متن خنثی» (neutral lock-screen copy). It serves GET/PUT
// /api/v1/profile/notification-settings and is the policy every push / web-push sender must go through:
//
//	prefs, _ := notifications.Load(ctx, q, userID)
//	if d := notifications.Decide(prefs, notifications.BeforePeriod, now, lastSent); d.Send {
//	    push := notifications.Render(prefs, msg, locale) // never carries health data when NeutralCopy
//	    …
//	}
//
// Rows live in `notification_preferences` (goose 00011); queries are in db/queries/profile (profile store).
package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/ritme/backend-go/internal/profile/store"
)

// Category is one switch on the settings screen (nbl_Me_Notifications).
type Category string

// The categories, grouped as on the board: cycle · health · other.
const (
	BeforePeriod  Category = "before_period"
	PMS           Category = "pms"
	FertileWindow Category = "fertile_window"
	DailyLog      Category = "daily_log"
	Medications   Category = "medications"
	Appointments  Category = "appointments"
	Checkups      Category = "checkups"
	Vitals        Category = "vitals"
	Learning      Category = "learning"
	Companion     Category = "companion"
	Articles      Category = "articles"
	// Pill is «قرص پیشگیری» (B-N1-09): switched and timed on the cycle-settings screen only, so it is not in
	// Groups (the nbl_Me_Notifications list) but is a known, opt-in category every sender gates on.
	Pill Category = "pill"
)

// extraCategories are known categories that are not a switch of the notification-settings screen.
var extraCategories = []Category{Pill}

// Group is a titled block of categories.
type Group struct {
	Code       string
	Categories []Category
}

// Groups is the screen order.
var Groups = []Group{
	{Code: "cycle", Categories: []Category{BeforePeriod, PMS, FertileWindow, DailyLog}},
	{Code: "health", Categories: []Category{Medications, Appointments, Checkups, Vitals}},
	{Code: "other", Categories: []Category{Learning, Companion, Articles}},
}

// offByDefault are the categories a new user has switched off (the board's grey switches): opt-in only.
var offByDefault = []Category{FertileWindow, Vitals, Articles, Pill}

// Known reports whether c is a category of this build.
func Known(c Category) bool {
	if slices.Contains(extraCategories, c) {
		return true
	}
	for _, g := range Groups {
		if slices.Contains(g.Categories, c) {
			return true
		}
	}
	return false
}

// DefaultEnabled is a category's state before the user touched it.
func DefaultEnabled(c Category) bool { return Known(c) && !slices.Contains(offByDefault, c) }

// Default quiet hours: 23:00 → 08:00 (the board), in minutes since midnight.
const (
	DefaultQuietStart = 23 * 60
	DefaultQuietEnd   = 8 * 60
)

// Preferences are one user's settings. Categories holds explicit choices only (missing = default).
type Preferences struct {
	Categories   map[Category]bool
	QuietEnabled bool
	QuietStart   int // minutes since midnight, Tehran wall-clock
	QuietEnd     int
	NeutralCopy  bool
	// Schedule is when the timed reminders fire (B-N1-09); explicit choices only (missing = default).
	Schedule Schedule
}

// Defaults are the settings of a user without a row.
func Defaults() Preferences {
	return Preferences{
		Categories:   map[Category]bool{},
		QuietEnabled: true,
		QuietStart:   DefaultQuietStart,
		QuietEnd:     DefaultQuietEnd,
		NeutralCopy:  true,
		Schedule:     Schedule{Times: map[Category]int{}},
	}
}

// Enabled is whether category c may be pushed at all. Unknown categories never are.
func (p Preferences) Enabled(c Category) bool {
	if !Known(c) {
		return false
	}
	if v, ok := p.Categories[c]; ok {
		return v
	}
	return DefaultEnabled(c)
}

// Getter is the read the loader needs (profile store).
type Getter interface {
	GetNotificationPreferences(ctx context.Context, userID uint64) (store.NotificationPreference, error)
}

// Load reads userID's settings; no row = Defaults().
func Load(ctx context.Context, q Getter, userID uint64) (Preferences, error) {
	row, err := q.GetNotificationPreferences(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return Defaults(), nil
	}
	if err != nil {
		return Preferences{}, fmt.Errorf("notifications: load: %w", err)
	}
	return FromRow(row), nil
}

// FromRow decodes a stored row; unreadable values fall back to the defaults (a push must never fail on them).
func FromRow(r store.NotificationPreference) Preferences {
	p := Defaults()
	p.QuietEnabled = r.QuietHoursEnabled
	p.NeutralCopy = r.NeutralCopy
	if m, ok := ParseClock(r.QuietStart); ok {
		p.QuietStart = m
	}
	if m, ok := ParseClock(r.QuietEnd); ok {
		p.QuietEnd = m
	}
	if r.Schedule.Valid {
		p.Schedule = parseSchedule(r.Schedule.V)
	}
	if r.Categories.Valid {
		var raw map[string]bool
		if json.Unmarshal(r.Categories.V, &raw) == nil {
			for k, v := range raw {
				if Known(Category(k)) {
					p.Categories[Category(k)] = v
				}
			}
		}
	}
	return p
}

// CategoriesJSON is the stored form of the explicit choices (known categories only, sorted keys).
func (p Preferences) CategoriesJSON() json.RawMessage {
	m := map[string]bool{}
	for k, v := range p.Categories {
		if Known(k) {
			m[string(k)] = v
		}
	}
	b, _ := json.Marshal(m) // map[string]bool always marshals; keys come out sorted
	return b
}

// ParseClock reads "HH:MM" or "HH:MM:SS" into minutes since midnight.
func ParseClock(s string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// FormatClock is minutes since midnight as "HH:MM".
func FormatClock(min int) string { return fmt.Sprintf("%02d:%02d", min/60, min%60) }

// dbClock is minutes since midnight as a TIME literal.
func dbClock(min int) string { return FormatClock(min) + ":00" }

// ── Reminder schedule (B-N1-09) ─────────────────────────────────────────────────────────────────────────────

// Timed are the categories with a time of day, in cycle-settings screen order.
var Timed = []Category{BeforePeriod, PMS, FertileWindow, DailyLog, Pill}

// defaultTimes are the board's times (nbl_Cycle_Settings), minutes since midnight Tehran.
var defaultTimes = map[Category]int{
	BeforePeriod:  9 * 60,
	PMS:           9 * 60,
	FertileWindow: 9 * 60,
	DailyLog:      22 * 60,
	Pill:          21 * 60,
}

// Days-before-period bounds; the board's default is 2.
const (
	DefaultDaysBefore = 2
	MinDaysBefore     = 1
	MaxDaysBefore     = 7
)

// Schedule holds explicit choices only: Times per timed category, DaysBefore (0 = default) for BeforePeriod.
type Schedule struct {
	Times      map[Category]int
	DaysBefore int
}

// TimeOf is when category c fires (minutes since midnight); ok=false for a category without a time.
func (p Preferences) TimeOf(c Category) (int, bool) {
	def, ok := defaultTimes[c]
	if !ok {
		return 0, false
	}
	if m, set := p.Schedule.Times[c]; set {
		return m, true
	}
	return def, true
}

// DaysBeforePeriod is how many days before the predicted period the BeforePeriod reminder fires.
func (p Preferences) DaysBeforePeriod() int {
	if d := p.Schedule.DaysBefore; d >= MinDaysBefore && d <= MaxDaysBefore {
		return d
	}
	return DefaultDaysBefore
}

type slotJSON struct {
	DaysBefore int    `json:"days_before,omitempty"`
	Time       string `json:"time,omitempty"`
}

// parseSchedule reads the stored column; unknown keys and unreadable values are dropped (defaults apply).
func parseSchedule(raw []byte) Schedule {
	out := Schedule{Times: map[Category]int{}}
	var m map[string]slotJSON
	if json.Unmarshal(raw, &m) != nil {
		return out
	}
	for k, v := range m {
		c := Category(k)
		if _, timed := defaultTimes[c]; !timed {
			continue
		}
		if t, ok := ParseClock(v.Time); ok {
			out.Times[c] = t
		}
		if c == BeforePeriod && v.DaysBefore >= MinDaysBefore && v.DaysBefore <= MaxDaysBefore {
			out.DaysBefore = v.DaysBefore
		}
	}
	return out
}

// ScheduleJSON is the stored form of the explicit schedule choices (sorted keys).
func (p Preferences) ScheduleJSON() json.RawMessage {
	m := map[string]slotJSON{}
	for c, t := range p.Schedule.Times {
		if _, timed := defaultTimes[c]; timed {
			m[string(c)] = slotJSON{Time: FormatClock(t)}
		}
	}
	if d := p.Schedule.DaysBefore; d >= MinDaysBefore && d <= MaxDaysBefore {
		s := m[string(BeforePeriod)]
		s.DaysBefore = d
		m[string(BeforePeriod)] = s
	}
	b, _ := json.Marshal(m) // map of plain structs always marshals
	return b
}

// DBClock is minutes since midnight as a TIME literal (for writers outside this package).
func DBClock(min int) string { return dbClock(min) }
