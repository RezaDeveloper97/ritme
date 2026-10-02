package postpartum

import (
	"slices"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Edinburgh Postnatal Depression Scale (Cox, Holden & Sagovsky 1987). Ten items, each answered with one of four
// options scored 0–3; items 1, 2 and 4 score their options 0→3 in order, the others 3→0. The client sends the score
// of the chosen option (GET /postpartum/epds/questions lists every option with its score), the server recomputes
// the total. The weekly short check is the EPDS-3 anxiety subscale (items 3, 4, 5; Kabir, Sheeder & Kelly 2008); the
// full 10-item check runs every two weeks. Thresholds are named constants [needs clinical review]; the admin config
// table of B-N9-11 can take them over (epds.cutoff.likely in nbl_Admin_Config).
const (
	KindShort = "short"
	KindFull  = "full"

	// ItemMax is the top score of an item.
	ItemMax = 3
	// SelfHarmItem is item 10 («The thought of harming myself has occurred to me»): any score above 0 is urgent.
	SelfHarmItem = "q10"
	// FullLikelyMin: a full total from here is «likely depression» and urgent (Cox 1987 cut-off 12/13).
	FullLikelyMin = 13
	// FullPossibleMin: a full total from here (below FullLikelyMin) is «possible» — advice to talk to a doctor.
	FullPossibleMin = 10
	// ShortElevatedMin: an EPDS-3 total from here suggests taking the full check now.
	ShortElevatedMin = 6

	// ShortIntervalDays / FullIntervalDays are the check-in cadence; FullFromDay is the first day after the birth
	// the full check is offered (EPDS is validated from about two weeks postpartum; earlier days get the short one).
	ShortIntervalDays = 7
	FullIntervalDays  = 14
	FullFromDay       = 14
)

// Bands.
const (
	BandLow      = "low"
	BandPossible = "possible" // full 10–12
	BandLikely   = "likely"   // full ≥ 13
	BandElevated = "elevated" // short ≥ ShortElevatedMin
)

// Urgent reasons.
const (
	ReasonSelfHarm = "self_harm"
	ReasonScore    = "score"
)

// Item is one EPDS item: its code and the score of each option in display order.
type Item struct {
	Code   string
	Scores [4]int
}

var (
	ascending  = [4]int{0, 1, 2, 3}
	descending = [4]int{3, 2, 1, 0}
)

// Items are the ten items in questionnaire order.
var Items = []Item{
	{"q1", ascending}, {"q2", ascending}, {"q3", descending}, {"q4", ascending}, {"q5", descending},
	{"q6", descending}, {"q7", descending}, {"q8", descending}, {"q9", descending}, {"q10", descending},
}

var shortCodes = []string{"q3", "q4", "q5"}

// Kinds are the check kinds.
var Kinds = []string{KindShort, KindFull}

// ItemsOf are the items of a kind, in questionnaire order.
func ItemsOf(kind string) []Item {
	if kind != KindShort {
		return Items
	}
	out := make([]Item, 0, len(shortCodes))
	for _, it := range Items {
		if slices.Contains(shortCodes, it.Code) {
			out = append(out, it)
		}
	}
	return out
}

// Result is a scored check.
type Result struct {
	Kind     string
	Answers  map[string]int // item code → score 0–3
	Total    int
	Max      int
	SelfHarm *int // item 10's score (full only)
	Band     string
	Urgent   bool
	Reasons  []string // why it is urgent (ReasonSelfHarm, ReasonScore)
	FollowUp string   // KindFull when a short check suggests the full one now
}

// Score scores a check. answers must hold a 0–ItemMax score for every item of the kind (validated by the caller);
// anything else in it is ignored.
func Score(kind string, answers map[string]int) Result {
	items := ItemsOf(kind)
	r := Result{Kind: kind, Answers: map[string]int{}, Max: len(items) * ItemMax, Band: BandLow}
	for _, it := range items {
		v := min(max(answers[it.Code], 0), ItemMax)
		r.Answers[it.Code] = v
		r.Total += v
	}
	if kind == KindShort {
		if r.Total >= ShortElevatedMin {
			r.Band, r.FollowUp = BandElevated, KindFull
		}
		return r
	}
	sh := r.Answers[SelfHarmItem]
	r.SelfHarm = &sh
	switch {
	case r.Total >= FullLikelyMin:
		r.Band = BandLikely
	case r.Total >= FullPossibleMin:
		r.Band = BandPossible
	}
	if sh > 0 {
		r.Reasons = append(r.Reasons, ReasonSelfHarm)
	}
	if r.Total >= FullLikelyMin {
		r.Reasons = append(r.Reasons, ReasonScore)
	}
	r.Urgent = len(r.Reasons) > 0
	return r
}

// Check is a stored check as the schedule and the history read it.
type Check struct {
	ID       uint64
	Kind     string
	TakenOn  civildate.Date
	Total    int
	SelfHarm *int
	Urgent   bool
}

// Band is the stored check's band.
func (c Check) Band() string {
	switch {
	case c.Kind == KindShort && c.Total >= ShortElevatedMin:
		return BandElevated
	case c.Kind == KindShort:
		return BandLow
	case c.Total >= FullLikelyMin:
		return BandLikely
	case c.Total >= FullPossibleMin:
		return BandPossible
	}
	return BandLow
}

// Max is the stored check's top total.
func (c Check) Max() int { return len(ItemsOf(c.Kind)) * ItemMax }

// Schedule is which check is due.
type Schedule struct {
	Due       string // KindShort | KindFull | "" (nothing due)
	NextDueOn civildate.Date
}

// ScheduleOn is the check-in schedule on today for a birth on birth: the full check from FullFromDay, then every
// FullIntervalDays (or right after an elevated short check); otherwise a check every ShortIntervalDays (a full check
// counts as that week's check).
func ScheduleOn(today, birth civildate.Date, last, lastFull *Check) Schedule {
	fullFrom := birth.AddDays(FullFromDay)
	fullNext := fullFrom
	if lastFull != nil {
		fullNext = laterOf(fullFrom, lastFull.TakenOn.AddDays(FullIntervalDays))
	}
	elevated := last != nil && last.Kind == KindShort && last.Total >= ShortElevatedMin &&
		(lastFull == nil || last.TakenOn.After(lastFull.TakenOn))
	if elevated || !today.Before(fullNext) {
		return Schedule{Due: KindFull, NextDueOn: today}
	}
	if last == nil || !today.Before(last.TakenOn.AddDays(ShortIntervalDays)) {
		return Schedule{Due: KindShort, NextDueOn: today}
	}
	next := last.TakenOn.AddDays(ShortIntervalDays)
	if fullNext.Before(next) {
		next = fullNext
	}
	return Schedule{NextDueOn: next}
}

func laterOf(a, b civildate.Date) civildate.Date {
	if a.After(b) {
		return a
	}
	return b
}
