package menopause

import (
	"context"
	"slices"
	"sort"
	"time"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Patterns (nbl_Meno_Score «الگوهایی که دیدیم — بر اساس ثبت‌های خودت · تشخیص پزشکی نیست»): associations in her own
// logs, computed as φ cards by bloom's analysis engine (analysis.Binary) with its minimum-data rules — at least
// analysis.CorrMinDays paired days, analysis.CorrMinGroupDays days in each group and analysis.CorrMinOutcome
// outcome days — and Cohen strengths. Never causal, never a diagnosis; the sentences are flagged needs_review.
const (
	// PatternWindowDays is how far back the patterns read (today included).
	PatternWindowDays = 90

	PatternTriggerFlashes     = "trigger_flashes"      // a logged trigger × a day with more flashes than her usual
	PatternNightSweatsFatigue = "night_sweats_fatigue" // night sweats × fatigue logged the next day

	DirectionMore = "more"
	DirectionLess = "less"

	groupWithout = "without"
	groupWith    = "with"
)

// Pattern is one card: the engine's correlation plus what it compares.
type Pattern struct {
	analysis.Correlation
	Trigger string // trigger code (trigger_flashes only)
	// MeanFlashes per group (trigger_flashes only), in group order without / with.
	MeanFlashes []float64
}

// Found: enough data and an association (the card the score screen shows).
func (p Pattern) Found() bool { return p.Status == analysis.CorrReady }

// Direction is more | less for a found pattern ("" otherwise).
func (p Pattern) Direction() string {
	switch {
	case !p.Found():
		return ""
	case p.Value < 0:
		return DirectionLess
	}
	return DirectionMore
}

// Patterns is GET /menopause/patterns.
type Patterns struct {
	From, To   civildate.Date
	DaysLogged int
	Items      []Pattern
	Disclaimer *catalog.Item
}

// flashCounts are the flashes per day: timer flashes, or 1 for a day whose log has hot flashes above «no» but no
// timer flash.
func flashCounts(logs map[civildate.Date]LogDay, flashes []Flash) map[civildate.Date]int {
	out := map[civildate.Date]int{}
	for _, f := range flashes {
		out[f.Day()]++
	}
	for d, l := range logs {
		if out[d] == 0 && l.Has(SlotHotFlashes) {
			out[d] = 1
		}
	}
	return out
}

// usualFlashes is the median daily count over the logged days (lower median).
func usualFlashes(days []civildate.Date, counts map[civildate.Date]int) int {
	if len(days) == 0 {
		return 0
	}
	v := make([]int, len(days))
	for i, d := range days {
		v[i] = counts[d]
	}
	sort.Ints(v)
	return v[(len(v)-1)/2]
}

// sweatyNight: a sweaty night flash in the night that ends on day's morning ([day−1 22:00, day 06:00)).
func sweatyNight(flashes []Flash, day civildate.Date) bool {
	from := day.AddDays(-1).TehranMidnight().Add(NightFromHour * time.Hour)
	to := day.TehranMidnight().Add(NightToHour * time.Hour)
	for _, f := range flashes {
		if f.Sweat && !f.StartedAt.Before(from) && f.StartedAt.Before(to) {
			return true
		}
	}
	return false
}

// BuildPatterns computes the cards over the logged days of [from, to]: one trigger_flashes card per log trigger
// she logged at least once (taxonomy order), then night_sweats_fatigue.
//
//   - trigger_flashes: on a logged day, exposed = the trigger is in that day's menopause.triggers; outcome = more
//     flashes than her usual (the median of her logged days; with a median of 0, any flash).
//   - night_sweats_fatigue: on a logged day, exposed = night sweats logged that day or a sweaty flash in the night
//     before it; outcome = fatigue logged that day.
func BuildPatterns(logs map[civildate.Date]LogDay, flashes []Flash, triggers []string) []Pattern {
	days := make([]civildate.Date, 0, len(logs))
	for d := range logs {
		days = append(days, d)
	}
	slices.SortFunc(days, func(a, b civildate.Date) int { return a.Compare(b) })
	counts := flashCounts(logs, flashes)
	usual := usualFlashes(days, counts)

	out := []Pattern{}
	for _, t := range triggers {
		without, with := analysis.Group{Key: groupWithout}, analysis.Group{Key: groupWith}
		sums := [2]int{}
		seen := false
		for _, d := range days {
			g, i := &without, 0
			if slices.Contains(logs[d].Triggers(), t) {
				g, i, seen = &with, 1, true
			}
			g.Days++
			sums[i] += counts[d]
			if counts[d] > usual {
				g.Hits++
			}
		}
		if !seen {
			continue
		}
		p := Pattern{Correlation: analysis.Binary(PatternTriggerFlashes, without, with), Trigger: t}
		p.MeanFlashes = []float64{meanOf(sums[0], without.Days), meanOf(sums[1], with.Days)}
		out = append(out, p)
	}

	without, with := analysis.Group{Key: groupWithout}, analysis.Group{Key: groupWith}
	for _, d := range days {
		l := logs[d]
		g := &without
		if l.Has(SlotNightSweats) || sweatyNight(flashes, d) {
			g = &with
		}
		g.Days++
		if l.Has(SlotFatigue) {
			g.Hits++
		}
	}
	out = append(out, Pattern{Correlation: analysis.Binary(PatternNightSweatsFatigue, without, with)})
	return out
}

func meanOf(sum, n int) float64 {
	if n == 0 {
		return 0
	}
	return float64(sum) / float64(n)
}

// Patterns loads the window and builds the cards.
func (s *Service) Patterns(ctx context.Context, userID uint64, today civildate.Date) (Patterns, error) {
	from := today.AddDays(-(PatternWindowDays - 1))
	logs, err := s.logDays(ctx, userID, from, today)
	if err != nil {
		return Patterns{}, err
	}
	flashes, err := s.Flashes(ctx, userID, from.AddDays(-1), today) // the night before the first day
	if err != nil {
		return Patterns{}, err
	}
	p := Patterns{From: from, To: today, DaysLogged: len(logs), Items: BuildPatterns(logs, flashes, LogTriggers())}
	if p.Disclaimer, err = s.item(ctx, GroupTips, TipPatternsDisclaimer); err != nil {
		return Patterns{}, err
	}
	return p, nil
}
