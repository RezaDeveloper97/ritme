package analysis

import (
	"slices"

	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Correlation keys, in screen order.
const (
	CorrSleepMood      = "sleep_mood"      // nights under 6 h × «کلافه» / «زودرنج» logged that day (φ)
	CorrPhaseEnergy    = "phase_energy"    // high energy × cycle phase (Cramér's V)
	CorrExerciseCramps = "exercise_cramps" // activity × severe cramps on period days (φ)
	CorrPhaseMood      = "phase_mood"      // good mood × cycle phase (Cramér's V)
)

// Correlation statuses.
const (
	CorrReady         = "ready"
	CorrNotEnoughData = "not_enough_data"
	CorrNoAssociation = "no_association"
)

// irritableMoods are the moods of the sleep card («کلافه» frustrated, «زودرنج» irritable).
var irritableMoods = []string{"frustrated", "irritable"}

// Group is one compared group of an association: Hits of Days had the outcome.
type Group struct {
	Key        string
	Days, Hits int
}

// Rate is Hits / Days (0 for an empty group).
func (g Group) Rate() float64 {
	if g.Days == 0 {
		return 0
	}
	return float64(g.Hits) / float64(g.Days)
}

// Correlation is one card of the correlations screen. It is always `not_causal`.
type Correlation struct {
	Key, Statistic string
	Value          float64
	Strength       string
	Status         string
	N              int
	Groups         []Group
	// Ratio compares the groups (exposed ÷ unexposed rate, or highest ÷ lowest phase rate); nil when the
	// denominator is 0.
	Ratio  *float64
	Phrase *Phrase
}

// CorrelationsReport is GET /analysis/correlations (Plus).
type CorrelationsReport struct {
	DaysLogged int
	Items      []Correlation
}

func enough(groups []Group) bool {
	n, hits := 0, 0
	for _, g := range groups {
		if g.Days < CorrMinGroupDays {
			return false
		}
		n += g.Days
		hits += g.Hits
	}
	return n >= CorrMinDays && hits >= CorrMinOutcome
}

func (c *Correlation) finish(value float64, groups []Group) {
	c.Groups = groups
	for _, g := range groups {
		c.N += g.Days
	}
	switch {
	case !enough(groups):
		c.Status = CorrNotEnoughData
		return
	case Strength(value) == "":
		c.Status = CorrNoAssociation
	default:
		c.Status = CorrReady
	}
	c.Value = round(value, 2)
	c.Strength = Strength(value)
}

func ratio(num, den float64) *float64 {
	if den == 0 {
		return nil
	}
	v := round(num/den, 1)
	return &v
}

// binary builds a φ card from exposed / unexposed groups (groups[0] unexposed, groups[1] exposed, the
// order the cards draw them).
func binary(key string, unexposed, exposed Group) Correlation {
	c := Correlation{Key: key, Statistic: StatPhi}
	phi := Phi(exposed.Hits, exposed.Days-exposed.Hits, unexposed.Hits, unexposed.Days-unexposed.Hits)
	c.finish(phi, []Group{unexposed, exposed})
	if c.Status == CorrReady {
		c.Ratio = ratio(exposed.Rate(), unexposed.Rate())
	}
	return c
}

// byPhase builds a Cramér's V card from per-phase groups.
func byPhase(key string, groups []Group) Correlation {
	c := Correlation{Key: key, Statistic: StatCramersV}
	hits, totals := make([]int, len(groups)), make([]int, len(groups))
	for i, g := range groups {
		hits[i], totals[i] = g.Hits, g.Days
	}
	c.finish(CramersV(hits, totals), groups)
	return c
}

// extremes are the phases with the highest and lowest rate (first wins ties).
func extremes(groups []Group) (hi, lo Group) {
	hi, lo = groups[0], groups[0]
	for _, g := range groups[1:] {
		if g.Rate() > hi.Rate() {
			hi = g
		}
		if g.Rate() < lo.Rate() {
			lo = g
		}
	}
	return hi, lo
}

func phaseGroups() []Group {
	out := make([]Group, len(Phases))
	for i, p := range Phases {
		out[i] = Group{Key: p}
	}
	return out
}

// BuildCorrelations computes the association cards of the range (the caller checks the entitlement).
func BuildCorrelations(in *Input) CorrelationsReport {
	phases := in.phaseDays()
	sleepShort, sleepOK := Group{Key: "sleep_under_6"}, Group{Key: "sleep_6_plus"}
	exercise, rest := Group{Key: "exercise"}, Group{Key: "no_exercise"}
	energy, mood := phaseGroups(), phaseGroups()
	logged := 0
	for _, d := range sortedDates(in.Days) {
		if !in.Range.Contains(d) {
			continue
		}
		logged++
		day := in.Days[d]
		if day.Sleep != "" && len(day.Moods) > 0 {
			hit := slices.ContainsFunc(day.Moods, func(m string) bool { return slices.Contains(irritableMoods, m) })
			g := &sleepOK
			if day.ShortSleep() {
				g = &sleepShort
			}
			g.Days++
			if hit {
				g.Hits++
			}
		}
		phase, ok := phases[d]
		if !ok {
			continue
		}
		i := slices.Index(Phases, phase)
		if day.Energy != "" {
			energy[i].Days++
			if day.HighEnergy() {
				energy[i].Hits++
			}
		}
		if good, ok := day.GoodMood(); ok {
			mood[i].Days++
			if good {
				mood[i].Hits++
			}
		}
		if phase == PhasePeriod {
			g := &rest
			if day.Active {
				g = &exercise
			}
			g.Days++
			if day.SevereCramps() {
				g.Hits++
			}
		}
	}

	sleep := binary(CorrSleepMood, sleepOK, sleepShort)
	if sleep.Status == CorrReady {
		switch {
		case sleep.Value < 0:
			sleep.Phrase = &Phrase{Key: "correlations.sleep_mood.less"}
		case sleep.Ratio == nil:
			sleep.Phrase = &Phrase{Key: "correlations.sleep_mood.only", Args: []Arg{
				Num("hits", sleepShort.Hits), Num("days", sleepShort.Days)}}
		default:
			sleep.Phrase = &Phrase{Key: "correlations.sleep_mood.more", Args: []Arg{
				Num("ratio", *sleep.Ratio), Num("hits", sleepShort.Hits), Num("days", sleepShort.Days)}}
		}
	}

	en := byPhase(CorrPhaseEnergy, energy)
	if en.Status == CorrReady {
		hi, lo := extremes(energy)
		en.Ratio = ratio(hi.Rate(), lo.Rate())
		if en.Ratio == nil {
			en.Phrase = &Phrase{Key: "correlations.phase_energy.only", Args: []Arg{PhaseArg("high_phase", hi.Key)}}
		} else {
			en.Phrase = &Phrase{Key: "correlations.phase_energy.more", Args: []Arg{
				PhaseArg("high_phase", hi.Key), Num("ratio", *en.Ratio), PhaseArg("low_phase", lo.Key)}}
		}
	}

	ex := binary(CorrExerciseCramps, rest, exercise)
	if ex.Status == CorrReady {
		key := "correlations.exercise_cramps.more"
		if ex.Value < 0 {
			key = "correlations.exercise_cramps.less"
		}
		ex.Phrase = &Phrase{Key: key}
	}

	md := byPhase(CorrPhaseMood, mood)
	if md.Status == CorrReady {
		hi, lo := extremes(mood)
		md.Ratio = ratio(hi.Rate(), lo.Rate())
		md.Phrase = &Phrase{Key: "correlations.phase_mood.best", Args: []Arg{
			PhaseArg("phase", hi.Key), Num("pct", int(round(hi.Rate()*100, 0)))}}
	}

	return CorrelationsReport{DaysLogged: logged, Items: []Correlation{sleep, en, ex, md}}
}

// Item finds a card by key.
func (r CorrelationsReport) Item(key string) (Correlation, bool) {
	for _, c := range r.Items {
		if c.Key == key {
			return c, true
		}
	}
	return Correlation{}, false
}

// JSON is one card.
func (c Correlation) JSON(cp *Copy) *jsonx.OrderedMap {
	groups := make([]*jsonx.OrderedMap, 0, len(c.Groups))
	for _, g := range c.Groups {
		groups = append(groups, jsonx.Obj(
			"key", g.Key, "days", g.Days, "hits", g.Hits, "pct", int(round(g.Rate()*100, 0)),
		))
	}
	var value, strength any
	if c.Status != CorrNotEnoughData {
		value = c.Value
	}
	if c.Strength != "" {
		strength = c.Strength
	}
	return jsonx.Obj(
		"key", c.Key,
		"status", c.Status,
		"statistic", c.Statistic,
		"value", value,
		"strength", strength,
		"n", c.N,
		"min_days", CorrMinDays,
		"groups", groups,
		"ratio", ptrOrNil(c.Ratio),
		"finding", phraseOrNil(c.Phrase, cp),
		"not_causal", true,
	)
}

// MoodByPhaseJSON is the hub's «حال در فازهای سیکل» card body (good-mood share per phase).
func (c Correlation) MoodByPhaseJSON(cp *Copy) *jsonx.OrderedMap {
	phases := make([]*jsonx.OrderedMap, 0, len(c.Groups))
	for _, g := range c.Groups {
		var pct any
		if g.Days > 0 {
			pct = int(round(g.Rate()*100, 0))
		}
		phases = append(phases, jsonx.Obj("phase", g.Key, "good_pct", pct, "days", g.Days))
	}
	return jsonx.Obj("phases", phases, "finding", phraseOrNil(c.Phrase, cp), "not_causal", true)
}

// JSON is the `data` body of GET /analysis/correlations for an entitled user.
func (r CorrelationsReport) JSON(cp *Copy) *jsonx.OrderedMap {
	items := make([]*jsonx.OrderedMap, 0, len(r.Items))
	for _, c := range r.Items {
		items = append(items, c.JSON(cp))
	}
	return jsonx.Obj("days_logged", r.DaysLogged, "items", items)
}
