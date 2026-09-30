package notifications

import (
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Reasons a push is held back.
const (
	ReasonCategoryOff = "category_off"
	ReasonQuietHours  = "quiet_hours"
	ReasonWeeklyCap   = "weekly_cap"
)

// weeklyCapped are categories sent at most once per 7 days (board: «حداکثر هفته‌ای ۱ بار»).
var weeklyCapped = map[Category]bool{Articles: true}

// Decision is a sender's verdict for one push.
type Decision struct {
	Send   bool
	Reason string // empty when Send
	// DeferUntil is when a held push may go instead (quiet hours / weekly cap); zero = drop it.
	DeferUntil time.Time
}

// minuteOfDay is t's Tehran wall-clock minute.
func minuteOfDay(t time.Time) int {
	t = t.In(civildate.Tehran)
	return t.Hour()*60 + t.Minute()
}

// InQuietHours reports whether now falls in the window [start, end). start > end wraps midnight; start == end
// is an empty window (nothing is silenced).
func (p Preferences) InQuietHours(now time.Time) bool {
	if !p.QuietEnabled || p.QuietStart == p.QuietEnd {
		return false
	}
	m := minuteOfDay(now)
	if p.QuietStart < p.QuietEnd {
		return m >= p.QuietStart && m < p.QuietEnd
	}
	return m >= p.QuietStart || m < p.QuietEnd
}

// QuietEndsAt is the first moment after now at which the quiet window is over (only meaningful inside it).
func (p Preferences) QuietEndsAt(now time.Time) time.Time {
	t := now.In(civildate.Tehran)
	end := time.Date(t.Year(), t.Month(), t.Day(), p.QuietEnd/60, p.QuietEnd%60, 0, 0, civildate.Tehran)
	if !end.After(t) {
		end = end.AddDate(0, 0, 1)
	}
	return end
}

// Decide is the gate every push / web-push sender calls: the category must be on, now must be outside quiet
// hours (the push is deferred to the window's end, not dropped) and capped categories go out at most weekly.
// lastSent is the previous push of the same category (zero = never).
func Decide(p Preferences, c Category, now, lastSent time.Time) Decision {
	if !p.Enabled(c) {
		return Decision{Reason: ReasonCategoryOff}
	}
	if weeklyCapped[c] && !lastSent.IsZero() {
		if next := lastSent.Add(7 * 24 * time.Hour); now.Before(next) {
			return Decision{Reason: ReasonWeeklyCap, DeferUntil: next}
		}
	}
	if p.InQuietHours(now) {
		return Decision{Reason: ReasonQuietHours, DeferUntil: p.QuietEndsAt(now)}
	}
	return Decision{Send: true}
}

// Message is what a sender wants to say; Title/Body may carry health data («پریودت ۲ روز دیگر است»).
type Message struct {
	Category Category
	Title    string
	Body     string
	URL      string // in-app path opened on tap; not shown on the lock screen
}

// Push is what actually leaves the server.
type Push struct {
	Title string
	Body  string
	URL   string
}

// Render applies «متن خنثی»: with NeutralCopy on, title and body are replaced by a generic line that says nothing
// about the user's health (lock-screen privacy). The deep link stays so a tap still lands on the right screen.
func Render(p Preferences, m Message, locale string) Push {
	if !p.NeutralCopy {
		return Push{Title: m.Title, Body: m.Body, URL: m.URL}
	}
	return Push{Title: T("push.neutral_title", locale), Body: T("push.neutral_body", locale), URL: m.URL}
}
