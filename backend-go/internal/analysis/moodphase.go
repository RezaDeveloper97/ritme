package analysis

import (
	"slices"
	"sort"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Mood by phase (§4.1 «حال در فازها», D-175): the share of days with each mood in each cycle phase, only for a phase
// with at least P.MoodPhaseMinDays mood-logged days; the card is ready with P.MoodPhaseMinPhases such phases. It
// describes the user's own days, it never claims the phase causes a mood.

// MoodShare is one mood's share of a phase's mood-logged days.
type MoodShare struct {
	Mood string // mood.moods code
	Days int
}

// PhaseMoods is one phase of the card.
type PhaseMoods struct {
	Phase    string
	Days     int // mood-logged days in the phase
	GoodDays int
	Shares   []MoodShare // most frequent first (empty below the minimum)
}

// Ready: enough mood-logged days for shares.
func (p PhaseMoods) Ready() bool { return p.Days >= P.MoodPhaseMinDays }

func (p PhaseMoods) pct(n int) int { return int(round(float64(n)*100/float64(p.Days), 0)) }

// MoodByPhase is the card.
type MoodByPhase struct {
	Phases     []PhaseMoods // cycle order
	DaysNeeded int
	Last       civildate.Date // the latest mood-logged day in a phase
	Phrase     *Phrase
}

// Ready: at least P.MoodPhaseMinPhases phases meet the minimum.
func (m MoodByPhase) Ready() bool {
	n := 0
	for _, p := range m.Phases {
		if p.Ready() {
			n++
		}
	}
	return n >= P.MoodPhaseMinPhases
}

// moodByPhase computes the card over the range's phase days.
func (in *Input) moodByPhase(phases map[civildate.Date]string) MoodByPhase {
	out := MoodByPhase{Phases: make([]PhaseMoods, len(Phases))}
	counts := make([]map[string]int, len(Phases))
	for i, p := range Phases {
		out.Phases[i] = PhaseMoods{Phase: p, Shares: []MoodShare{}}
		counts[i] = map[string]int{}
	}
	for _, d := range sortedDates(in.Days) {
		phase, ok := phases[d]
		day := in.Days[d]
		if !ok || !in.Range.Contains(d) || len(day.Moods) == 0 {
			continue
		}
		i := slices.Index(Phases, phase)
		out.Phases[i].Days++
		if good, _ := day.GoodMood(); good {
			out.Phases[i].GoodDays++
		}
		seen := map[string]bool{}
		for _, m := range day.Moods {
			if !seen[m] {
				seen[m] = true
				counts[i][m]++
			}
		}
		out.Last = d
	}
	for i := range out.Phases {
		p := &out.Phases[i]
		if !p.Ready() {
			continue
		}
		for m, n := range counts[i] {
			p.Shares = append(p.Shares, MoodShare{Mood: m, Days: n})
		}
		sort.Slice(p.Shares, func(a, b int) bool {
			if p.Shares[a].Days != p.Shares[b].Days {
				return p.Shares[a].Days > p.Shares[b].Days
			}
			return p.Shares[a].Mood < p.Shares[b].Mood
		})
	}
	// Days still needed: the two best-logged phases brought up to the minimum.
	byDays := slices.Clone(out.Phases)
	sort.SliceStable(byDays, func(a, b int) bool { return byDays[a].Days > byDays[b].Days })
	for _, p := range byDays[:P.MoodPhaseMinPhases] {
		out.DaysNeeded += max(0, P.MoodPhaseMinDays-p.Days)
	}
	if out.Ready() {
		var best *PhaseMoods
		tie := false
		for i := range out.Phases {
			p := &out.Phases[i]
			if !p.Ready() {
				continue
			}
			switch {
			case best == nil || p.pct(p.GoodDays) > best.pct(best.GoodDays):
				best, tie = p, false
			case p.pct(p.GoodDays) == best.pct(best.GoodDays):
				tie = true
			}
		}
		if best != nil && !tie && best.GoodDays > 0 {
			out.Phrase = &Phrase{Key: "correlations.phase_mood.best", Args: []Arg{
				PhaseArg("phase", best.Phase), Num("pct", best.pct(best.GoodDays))}}
		}
	}
	return out
}

// JSON is the hub's «حال در فازهای چرخه» card body (and correlations' mood_by_phase).
func (m MoodByPhase) JSON(c *Copy) *jsonx.OrderedMap {
	phases := make([]*jsonx.OrderedMap, 0, len(m.Phases))
	for _, p := range m.Phases {
		var good any
		shares := make([]*jsonx.OrderedMap, 0, len(p.Shares))
		if p.Ready() {
			good = p.pct(p.GoodDays)
			for _, s := range p.Shares {
				key := symptomKey("mood", "moods", s.Mood)
				shares = append(shares, jsonx.Obj("mood", s.Mood, "label", c.SymptomLabel(key), "days", s.Days, "pct", p.pct(s.Days)))
			}
		}
		phases = append(phases, jsonx.Obj("phase", p.Phase, "ready", p.Ready(), "good_pct", good, "days", p.Days, "moods", shares))
	}
	return jsonx.Obj(
		"ready", m.Ready(),
		"min_days", P.MoodPhaseMinDays,
		"days_needed", m.DaysNeeded,
		"phases", phases,
		"finding", phraseOrNil(m.Phrase, c),
		"not_causal", true,
	)
}
