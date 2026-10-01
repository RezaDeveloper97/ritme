// Package contraception is the contraception API (CB-CONTRA-01, boards nbl_Contra_Setup / _Pill / _Missed /
// _Other): the user's method, the pill pack (day, placebo/break days, streak, next pack, refill) and the reminders
// of the long-acting methods, under /api/v1/contraception. Go only — no Laravel counterpart (deviations.md).
//
// Built on bloom, never beside it (docs/canvas-build/README.md C5): the «track contraception» switch is
// user_life_profiles.track_contraception (B-N2-01/03) and the daily pill reminder is the `pill` category and time of
// notification_preferences (B-N1-09, internal/notifications) — saving a method writes those, so there is exactly one
// switch and one pill schedule. Long-acting reminders (IUD string check / 6-week visit / replacement, injection every
// 12 weeks, implant date) and the pack refill reminder are care reminders: rows in `reminders` (type appointment or
// custom) linked through contraception_reminders. Missed-pill guidance is catalog content (`missed_pill_rules`).
package contraception

import (
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Methods (nbl_Contra_Setup order).
const (
	MethodCombinedPill  = "combined_pill"
	MethodProgestinPill = "progestin_pill"
	MethodCopperIUD     = "copper_iud"
	MethodHormonalIUD   = "hormonal_iud"
	MethodInjection     = "injection"
	MethodImplant       = "implant"
	MethodCondom        = "condom"
	MethodOther         = "other"
)

// Methods is every method in screen order.
var Methods = []string{
	MethodCombinedPill, MethodProgestinPill, MethodCopperIUD, MethodHormonalIUD,
	MethodInjection, MethodImplant, MethodCondom, MethodOther,
}

// Pack types of the combined pill (board: «۲۱ قرص + ۷ روز استراحت», «۲۸ قرص», «۲۴ + ۴»).
const (
	Pack21Plus7 = "21_7"
	Pack28      = "28"
	Pack24Plus4 = "24_4"
)

// PackTypes is every pack type in screen order.
var PackTypes = []string{Pack21Plus7, Pack28, Pack24Plus4}

// Pill statuses a user logs.
const (
	StatusTaken  = "taken"
	StatusMissed = "missed"
)

// PillStatuses are the loggable statuses.
var PillStatuses = []string{StatusTaken, StatusMissed}

// Day statuses of the pack grid besides the logged ones.
const (
	StatusPending   = "pending"   // today, nothing logged yet
	StatusUpcoming  = "upcoming"  // a later day of the pack
	StatusUntracked = "untracked" // an earlier day before tracking began (Logs.Since), nothing logged
)

// Day kinds of a pack.
const (
	KindActive  = "active"  // a hormone pill
	KindPlacebo = "placebo" // a reminder pill of a 28 / 24+4 pack (taken, no hormone)
	KindBreak   = "break"   // the 7 pill-free days of a 21+7 pack
)

// IUD lifetime bounds (years) and the 6-week check-up visit.
const (
	MinIUDLifetime = 1
	MaxIUDLifetime = 12
	FollowupDays   = 42
	InjectionDays  = 84 // «معمولاً هر ۱۲ هفته»
	RefillDaysLead = 5  // «یادآور خرید ۵ روز قبل از تمام شدن»
	MaxPacksLeft   = 24
)

// IsPill reports whether method is a pill method (pack, daily reminder, pill log).
func IsPill(method string) bool { return method == MethodCombinedPill || method == MethodProgestinPill }

// IsIUD reports whether method is an IUD.
func IsIUD(method string) bool { return method == MethodCopperIUD || method == MethodHormonalIUD }

// Pack is the shape of a pill pack: Length days, the first Active of them hormone pills; the rest are placebo
// pills (Placebo) or pill-free break days.
type Pack struct {
	Length  int
	Active  int
	Placebo bool
}

// PackFor is the pack of a pill method; ok=false for other methods. The progestogen-only pill is taken every day
// without a break (28 active), whatever pack type was sent.
func PackFor(method, packType string) (Pack, bool) {
	switch method {
	case MethodProgestinPill:
		return Pack{Length: 28, Active: 28}, true
	case MethodCombinedPill:
		switch packType {
		case Pack28:
			return Pack{Length: 28, Active: 21, Placebo: true}, true
		case Pack24Plus4:
			return Pack{Length: 28, Active: 24, Placebo: true}, true
		default:
			return Pack{Length: 28, Active: 21}, true
		}
	}
	return Pack{}, false
}

// KindOf is the kind of pack day n (1-based).
func (p Pack) KindOf(n int) string {
	switch {
	case n <= p.Active:
		return KindActive
	case p.Placebo:
		return KindPlacebo
	default:
		return KindBreak
	}
}

// floorDiv is a/b rounded toward minus infinity (b > 0).
func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && a < 0 {
		q--
	}
	return q
}

// Position is where d falls in the packs that start on start and follow each other: the 0-based pack index
// (negative before start) and the 1-based day in that pack.
func (p Pack) Position(start, d civildate.Date) (index, day int) {
	diff := start.DiffDays(d)
	index = floorDiv(diff, p.Length)
	return index, diff - index*p.Length + 1
}

// DayKind is the kind of day d.
func (p Pack) DayKind(start, d civildate.Date) string {
	_, n := p.Position(start, d)
	return p.KindOf(n)
}

// PackStart is the first day of the pack d falls in.
func (p Pack) PackStart(start, d civildate.Date) civildate.Date {
	idx, _ := p.Position(start, d)
	return start.AddDays(idx * p.Length)
}

// Logs are the logged pill days (date → taken | missed) and the day tracking began (Since, zero = unknown): an
// unlogged pill day before Since is untracked, not missed — a user who sets up on day 8 of a pack has not missed
// days 1–7.
type Logs struct {
	Days  map[civildate.Date]string
	Since civildate.Date
}

// NewLogs returns empty logs tracked since since.
func NewLogs(since civildate.Date) Logs { return Logs{Days: map[civildate.Date]string{}, Since: since} }

// StatusOn is the grid status of day d of a pack: the logged status, else upcoming (after today), pending (today),
// untracked (before tracking began) or missed (an earlier pill day). Break days have no status ("").
func (p Pack) StatusOn(start, d, today civildate.Date, logs Logs) string {
	if p.DayKind(start, d) == KindBreak {
		return ""
	}
	if s, ok := logs.Days[d]; ok {
		return s
	}
	switch {
	case d.After(today):
		return StatusUpcoming
	case d == today:
		return StatusPending
	case !logs.Since.IsZero() && d.Before(logs.Since):
		return StatusUntracked
	default:
		return StatusMissed
	}
}

// StreakWindow is how far back the streak looks (days).
const StreakWindow = 400

// Streak is «N روز پشت سر هم بدون جا انداختن»: the calendar days back from today without a missed pill, since the
// pack schedule began (start). Break days count; today counts once its pill is taken (a pending today neither
// counts nor breaks the run); a missed, unlogged or untracked earlier pill day ends it.
func (p Pack) Streak(start, today civildate.Date, logs Logs) int {
	count := 0
	switch s := p.StatusOn(start, today, today, logs); s {
	case StatusMissed:
		return 0
	case StatusTaken, "":
		count++
	}
	for d := today.AddDays(-1); !d.Before(start) && count < StreakWindow; d = d.AddDays(-1) {
		s := p.StatusOn(start, d, today, logs)
		if s != StatusTaken && s != "" {
			break
		}
		count++
	}
	return count
}

// MissedCount is how many hormone pills in a row were missed most recently (the Contra_Missed «۱ قرص / ۲ قرص یا
// بیشتر» choice): back from today (today only when logged missed) over active days, placebo/break days skipped,
// until a taken pill, an untracked day or the schedule start. 0 when the last active pill was taken.
func (p Pack) MissedCount(start, today civildate.Date, logs Logs) int {
	count := 0
	for d := today; !d.Before(start) && count < p.Length; d = d.AddDays(-1) {
		if p.DayKind(start, d) != KindActive {
			continue
		}
		switch p.StatusOn(start, d, today, logs) {
		case StatusMissed:
			count++
		case StatusPending:
			continue
		default:
			return count
		}
	}
	return count
}

// Refill is the pack stock as of today; Known=false when the user never counted her packs.
type Refill struct {
	Known     bool
	PacksLeft int            // unopened packs after the current one
	RunsOutOn civildate.Date // the first day without a pack
	RefillOn  civildate.Date // RefillDaysLead days before RunsOutOn
}

// RefillAt is the stock on today: packsLeft were counted when the pack starting countedOn was in use; every pack
// started since then used one.
func (p Pack) RefillAt(start, today civildate.Date, packsLeft int, countedOn civildate.Date) Refill {
	current := p.PackStart(start, today)
	used := 0
	if countedOn.Before(current) {
		used = floorDiv(countedOn.DiffDays(current)+p.Length-1, p.Length)
	}
	left := max(0, packsLeft-used)
	runsOut := current.AddDays(p.Length * (left + 1))
	return Refill{Known: true, PacksLeft: left, RunsOutOn: runsOut, RefillOn: runsOut.AddDays(-RefillDaysLead)}
}

// Day is one day of the pack grid.
type Day struct {
	N      int
	Date   civildate.Date
	Kind   string
	Status string // "" on break days
}

// Days is the grid of the pack today falls in.
func (p Pack) Days(start, today civildate.Date, logs Logs) []Day {
	first := p.PackStart(start, today)
	out := make([]Day, p.Length)
	for i := range out {
		d := first.AddDays(i)
		out[i] = Day{N: i + 1, Date: d, Kind: p.KindOf(i + 1), Status: p.StatusOn(start, d, today, logs)}
	}
	return out
}
