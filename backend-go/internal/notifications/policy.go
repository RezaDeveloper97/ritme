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

// minuteOfDay is t's wall-clock minute in loc.
func minuteOfDay(t time.Time, loc *time.Location) int {
	t = t.In(loc)
	return t.Hour()*60 + t.Minute()
}

// InQuietHours reports whether now falls in the window [start, end) on the user's clock (p.Location()). start > end
// wraps midnight; start == end is an empty window (nothing is silenced).
func (p Preferences) InQuietHours(now time.Time) bool {
	if !p.QuietEnabled || p.QuietStart == p.QuietEnd {
		return false
	}
	m := minuteOfDay(now, p.Location())
	if p.QuietStart < p.QuietEnd {
		return m >= p.QuietStart && m < p.QuietEnd
	}
	return m >= p.QuietStart || m < p.QuietEnd
}

// QuietEndsAt is the first moment after now at which the quiet window is over (only meaningful inside it), on the
// user's clock: the end time today, else tomorrow (WallTime: an end inside a skipped DST hour is the transition).
func (p Preferences) QuietEndsAt(now time.Time) time.Time {
	loc := p.Location()
	day := civildate.FromTime(now.In(loc))
	end := WallTime(day, p.QuietEnd, loc)
	if !end.After(now) {
		end = WallTime(day.AddDays(1), p.QuietEnd, loc)
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

// Discreet is the CB-PRIV-01 «اعلان‌های محرمانه» flag (discreet_notifications). It is the same stored preference as
// B-N1-11's «متن خنثی» (NeutralCopy) — one switch, shown on both the notification and the privacy screens — and it
// covers every channel that can reach a lock screen: push and web push (Render) and SMS (RenderSMS, SMSTemplate).
func (p Preferences) Discreet() bool { return p.NeutralCopy }

// RenderSMS is the text of a free-text SMS about the user (e.g. a reminder by SMS): Render's title and body on two
// lines, so with the discreet flag on it is the neutral «یادآور امروز» copy and carries no health data. An SMS has
// no deep link, so the URL is dropped.
func RenderSMS(p Preferences, m Message, locale string) string {
	push := Render(p, m, locale)
	if push.Body == "" {
		return push.Title
	}
	return push.Title + "\n" + push.Body
}

// SMSTemplate picks the gateway (Kavenegar lookup) template of a templated SMS whose wording lives at the gateway:
// with the discreet flag on (Preferences.Discreet / Discreet), the deployment's neutral variant; off, the regular
// template. ok is false when the flag is on and no neutral variant is configured: the sender must then NOT send
// (no silent fallback to the regular wording).
func SMSTemplate(discreet bool, template, neutral string) (string, bool) {
	if !discreet {
		return template, true
	}
	return neutral, neutral != ""
}

// noticeCategories maps an inbox row's kind (user_notifications.type) to the switch that silences its push. Kinds
// that are not listed have no switch of their own: only quiet hours hold them back.
var noticeCategories = map[string]Category{
	"companion": Companion,
	"family":    Companion,
	"learning":  Learning,
}

// DecideNotice is Decide for the push of an inbox row of kind typ: its category must be on (when it has one) and
// now must be outside quiet hours (deferred to the window's end). The inbox row itself is written either way.
func DecideNotice(p Preferences, typ string, now time.Time) Decision {
	if c, ok := noticeCategories[typ]; ok {
		return Decide(p, c, now, time.Time{})
	}
	if p.InQuietHours(now) {
		return Decision{Reason: ReasonQuietHours, DeferUntil: p.QuietEndsAt(now)}
	}
	return Decision{Send: true}
}

// RenderNotice is Render for an inbox row: with «متن خنثی» on — or when the row has no title in any language — the
// push only says that there is something new in the app; the inbox shows the real text after the tap.
func RenderNotice(p Preferences, m Message, locale string) Push {
	if !p.NeutralCopy && m.Title != "" {
		return Push{Title: m.Title, Body: m.Body, URL: m.URL}
	}
	return Push{Title: T("push.notice_title", locale), Body: T("push.notice_body", locale), URL: m.URL}
}
