// Package guide is the postpartum mode's time model and its admin-editable copy (bloom B-N5-01): days and weeks since
// the birth, the recovery phase, and the texts — week tips, alerts, the EPDS safety messages — read from
// message_contents with an embedded fa/en fallback (lang/<code>/postpartum_copy.json, [needs clinical review]).
//
// It is a leaf package (no store, no HTTP): internal/postpartum (the API), messages/manager (the postpartum message
// engine) and the admin messages registry (the slots below) all use it, so the three always agree.
package guide

import (
	"context"
	"embed"
	"io/fs"
	"strconv"
	"strings"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// message_contents groups of the postpartum copy (registered in internal/admin/messages/registry).
const (
	TipGroup    = "postpartum_week_tip" // one tip per week since the birth: item keys "1".."12", then "late"
	AlertGroup  = "postpartum_alert"    // home / log alerts and the check-in prompts
	SafetyGroup = "postpartum_safety"   // the EPDS result messages (urgent with call actions, advice, …)
)

// Time model.
const (
	// PuerperiumDays is the «دوره نفاس» (6 weeks): the home's week ring counts towards it.
	PuerperiumDays = 42
	// RecoveryDays ends the recovery phase (26 weeks); later the mode is in its late phase.
	RecoveryDays = 182
	// MaxTipWeek is the last week with its own tip; LateTipKey serves every later week.
	MaxTipWeek = 12
	LateTipKey = "late"
)

// Recovery phases.
const (
	PhasePuerperium = "puerperium" // days 0–41
	PhaseRecovery   = "recovery"   // weeks 6–25
	PhaseLate       = "late"       // from week 26
)

// Alert item keys (AlertGroup).
const (
	AlertCallWhen      = "call_when"      // «کی فوراً تماس بگیرم؟»
	AlertHeavyBleeding = "heavy_bleeding" // lochia heavy logged today
	AlertLargeClots    = "large_clots"    // large clots logged today
	AlertCheckinShort  = "checkin_short"  // the weekly 3-question check is due
	AlertCheckinFull   = "checkin_full"   // the 10-question check is due
)

// Safety item keys (SafetyGroup).
const (
	SafetyUrgent        = "urgent"         // EPDS item 10 > 0 or total ≥ 13: call actions
	SafetyAdvice        = "advice"         // full EPDS 10–12: talk to a doctor or counsellor
	SafetyShortElevated = "short_elevated" // EPDS-3 ≥ cutoff: take the full check
	SafetyDisclaimer    = "disclaimer"     // non-diagnostic note above the questions
)

// Slot is one (group, item key) of the copy and its text fields.
type Slot struct {
	Group, Key string
	Fields     []string
}

// TipKeys are the item keys of TipGroup in order.
func TipKeys() []string {
	out := make([]string, 0, MaxTipWeek+1)
	for w := 1; w <= MaxTipWeek; w++ {
		out = append(out, strconv.Itoa(w))
	}
	return append(out, LateTipKey)
}

// Slots are every slot of the copy, in admin display order.
func Slots() []Slot {
	var out []Slot
	for _, k := range TipKeys() {
		out = append(out, Slot{TipGroup, k, []string{"title", "body"}})
	}
	out = append(out, Slot{AlertGroup, AlertCallWhen, []string{"title", "body"}})
	for _, k := range []string{AlertHeavyBleeding, AlertLargeClots, AlertCheckinShort, AlertCheckinFull} {
		out = append(out, Slot{AlertGroup, k, []string{"title", "body", "action"}})
	}
	return append(out,
		Slot{SafetyGroup, SafetyUrgent, []string{"title", "body", "emergency_label", "social_label", "counsel_label"}},
		Slot{SafetyGroup, SafetyAdvice, []string{"title", "body"}},
		Slot{SafetyGroup, SafetyShortElevated, []string{"title", "body", "action"}},
		Slot{SafetyGroup, SafetyDisclaimer, []string{"body"}},
	)
}

// SlotOf finds a slot.
func SlotOf(group, key string) (Slot, bool) {
	for _, s := range Slots() {
		if s.Group == group && s.Key == key {
			return s, true
		}
	}
	return Slot{}, false
}

// Status is where the mother is on day today of the birth on birth.
type Status struct {
	DaysSinceBirth int // 0 on the birth day
	Weeks, Days    int // completed weeks and the days after them («۲ هفته و ۳ روز»)
	Week           int // the 1-based current week (days 0–6 = week 1)
	Phase          string
	// PuerperiumDaysLeft counts down the 6 weeks (0 after them); Progress is DaysSinceBirth / PuerperiumDays ≤ 1.
	PuerperiumDaysLeft int
	Progress           float64
}

// StatusOn computes the status (a birth date after today counts as today).
func StatusOn(birth, today civildate.Date) Status {
	d := max(birth.DiffDays(today), 0)
	s := Status{DaysSinceBirth: d, Weeks: d / 7, Days: d % 7, Week: d/7 + 1}
	switch {
	case d < PuerperiumDays:
		s.Phase = PhasePuerperium
	case d < RecoveryDays:
		s.Phase = PhaseRecovery
	default:
		s.Phase = PhaseLate
	}
	s.PuerperiumDaysLeft = max(PuerperiumDays-d, 0)
	s.Progress = min(float64(d)/PuerperiumDays, 1)
	return s
}

// TipKey is the TipGroup item of a 1-based week.
func TipKey(week int) string {
	if week > MaxTipWeek {
		return LateTipKey
	}
	return strconv.Itoa(max(week, 1))
}

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

// Payloader reads one live message_contents payload (messages/content.Repository.Payload: decoded PHP array,
// ok=false without a live row).
type Payloader interface {
	Payload(ctx context.Context, group, itemKey, locale string) (any, bool, error)
}

// Copy resolves the postpartum texts for one request: the admin row of the request locale, else of the default
// language, else the embedded copy of the locale (English when the locale has none). A written row is used as is
// (an empty field stays empty). p may be nil (embedded copy only).
type Copy struct {
	p             Payloader
	locale, deflt string
}

// NewCopy returns the resolver.
func NewCopy(p Payloader, locale, deflt string) *Copy {
	return &Copy{p: p, locale: locale, deflt: deflt}
}

// Text is the slot's fields (every field of the slot present, "" when unset).
func (c *Copy) Text(ctx context.Context, group, key string) (map[string]string, error) {
	slot, ok := SlotOf(group, key)
	if !ok {
		return map[string]string{}, nil
	}
	if c.p != nil {
		locales := []string{c.locale}
		if c.deflt != "" && c.deflt != c.locale {
			locales = append(locales, c.deflt)
		}
		for _, l := range locales {
			v, ok, err := c.p.Payload(ctx, group, key, l)
			if err != nil {
				return nil, err
			}
			if ok {
				return fields(slot, v), nil
			}
		}
	}
	line, _ := translator().Get("postpartum_copy."+group+"."+key, c.locale)
	return fields(slot, line), nil
}

func fields(slot Slot, v any) map[string]string {
	out := make(map[string]string, len(slot.Fields))
	for _, f := range slot.Fields {
		s, _ := phpval.Get(v, f)
		str, _ := s.(string)
		out[f] = strings.TrimSpace(str)
	}
	return out
}
