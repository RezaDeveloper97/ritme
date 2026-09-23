package home

import (
	"math"
	"slices"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

// ---------------------------------------------------------------------------
// CycleHistoryDigest (backend/app/Services/HomePage/Support/CycleHistoryDigest.php).

const (
	digestValidCycleMin    = 21
	digestValidCycleMax    = 45
	digestValidDurationMin = 2
	digestValidDurationMax = 10
)

// DigestCycle is one recorded cycle of the digest.
type DigestCycle struct {
	ID           int64
	Start        civildate.Date
	End          civildate.Date // zero = no logged end
	CycleLength  *int
	PeriodLength *int
	IsCurrent    bool
	IsConfirmed  bool
	IsEstimated  bool
	Source       string
}

// JSON is the digest entry array, in its key order.
func (c DigestCycle) JSON() *jsonx.OrderedMap {
	var end any
	if !c.End.IsZero() {
		end = c.End.String()
	}
	return jsonx.Obj(
		"id", c.ID,
		"period_start_date", c.Start.String(),
		"period_end_date", end,
		"cycle_length", intOrNil(c.CycleLength),
		"period_length", intOrNil(c.PeriodLength),
		"is_ongoing", c.End.IsZero(),
		"is_current", c.IsCurrent,
		"is_confirmed", c.IsConfirmed,
		"is_estimated", c.IsEstimated,
		"source", c.Source,
	)
}

// Digest is the per-cycle facts of a user's period history, newest first.
type Digest struct{ cycles []DigestCycle }

// NewDigest is CycleHistoryDigest::fromHistories (histories in any order).
func NewDigest(histories []model.History) *Digest {
	ordered := slices.Clone(histories)
	slices.SortStableFunc(ordered, func(a, b model.History) int { return a.PeriodStart.Compare(b.PeriodStart) })

	cycles := make([]DigestCycle, 0, len(ordered))
	for i, h := range ordered {
		c := DigestCycle{
			ID: h.ID, Start: h.PeriodStart, End: h.PeriodEnd,
			IsConfirmed: h.IsConfirmed, IsEstimated: h.IsEstimated, Source: h.Source,
		}
		if i+1 < len(ordered) {
			n := h.PeriodStart.DiffDays(ordered[i+1].PeriodStart)
			c.CycleLength = &n
		} else {
			c.IsCurrent = true
		}
		if !h.PeriodEnd.IsZero() && !h.PeriodEnd.Before(h.PeriodStart) {
			n := h.PeriodStart.DiffDays(h.PeriodEnd) + 1
			c.PeriodLength = &n
		}
		cycles = append(cycles, c)
	}
	slices.Reverse(cycles)
	return &Digest{cycles: cycles}
}

// StartingOn is the cycle that started on d, if recorded.
func (d *Digest) StartingOn(day civildate.Date) (DigestCycle, bool) {
	for _, c := range d.cycles {
		if c.Start == day {
			return c, true
		}
	}
	return DigestCycle{}, false
}

// Previous are the cycles behind the current one, newest first (limit < 0 = all).
func (d *Digest) Previous(limit int) []DigestCycle {
	if len(d.cycles) <= 1 {
		return nil
	}
	prev := d.cycles[1:]
	if limit >= 0 && len(prev) > limit {
		prev = prev[:limit]
	}
	return prev
}

// LastCycleLength is the first known cycle length, newest first.
func (d *Digest) LastCycleLength() *int {
	for _, c := range d.cycles {
		if c.CycleLength != nil {
			return c.CycleLength
		}
	}
	return nil
}

// LastPeriodLength is the most recently finished bleed's duration.
func (d *Digest) LastPeriodLength() *int {
	for _, c := range d.cycles {
		if c.PeriodLength != nil {
			return c.PeriodLength
		}
	}
	return nil
}

func (d *Digest) collectValid(field func(DigestCycle) *int, lo, hi int) []int {
	out := []int{}
	for _, c := range d.cycles {
		if v := field(c); v != nil && *v >= lo && *v <= hi {
			out = append(out, *v)
		}
	}
	return out
}

// ValidCycleLengths are the cycle lengths within 21..45, newest first.
func (d *Digest) ValidCycleLengths() []int {
	return d.collectValid(func(c DigestCycle) *int { return c.CycleLength }, digestValidCycleMin, digestValidCycleMax)
}

// ValidPeriodLengths are the durations within 2..10, newest first.
func (d *Digest) ValidPeriodLengths() []int {
	return d.collectValid(func(c DigestCycle) *int { return c.PeriodLength }, digestValidDurationMin, digestValidDurationMax)
}

// AverageCycleLength is (int) round(mean) of the valid cycle lengths.
func (d *Digest) AverageCycleLength() *int { return averageOrNil(d.ValidCycleLengths()) }

// AveragePeriodLength is (int) round(mean) of the valid period lengths.
func (d *Digest) AveragePeriodLength() *int { return averageOrNil(d.ValidPeriodLengths()) }

// ShortestCycle is min(valid cycle lengths).
func (d *Digest) ShortestCycle() *int {
	v := d.ValidCycleLengths()
	if len(v) == 0 {
		return nil
	}
	m := slices.Min(v)
	return &m
}

// LongestCycle is max(valid cycle lengths).
func (d *Digest) LongestCycle() *int {
	v := d.ValidCycleLengths()
	if len(v) == 0 {
		return nil
	}
	m := slices.Max(v)
	return &m
}

// averageOrNil is `$values === [] ? null : (int) round(array_sum($values) / count($values))`.
func averageOrNil(values []int) *int {
	if len(values) == 0 {
		return nil
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	n := int(phpround.Round(float64(sum)/float64(len(values)), 0))
	return &n
}

// ---------------------------------------------------------------------------
// HealthMetricScorer (backend/app/Services/HomePage/Support/HealthMetricScorer.php).

var (
	moodScores = map[string]int{
		"happy": 100, "calm": 85, "sensitive": 55, "bored": 45,
		"frustrated": 35, "anxious": 30, "angry": 25, "sad": 20,
	}
	sleepQualityScores  = map[string]int{"good": 100, "medium": 60, "bad": 25}
	sleepDurationScores = map[string]int{"9_plus": 85, "6_9": 100, "3_6": 55, "0_3": 25}
)

// scoredLog is the part of a daily log the scorer and the challenge signal read.
type scoredLog struct {
	Moods         []string
	SleepQuality  *string
	SleepDuration *string
	EnergyLevel   *string
	Fatigue       *bool
}

func moodScore(moods []string) *int {
	var scores []int
	for _, m := range moods {
		if s, ok := moodScores[m]; ok {
			scores = append(scores, s)
		}
	}
	return averageOrNil(scores)
}

func sleepScore(quality, duration *string) *int {
	var parts []int
	if quality != nil {
		if s, ok := sleepQualityScores[*quality]; ok {
			parts = append(parts, s)
		}
	}
	if duration != nil {
		if s, ok := sleepDurationScores[*duration]; ok {
			parts = append(parts, s)
		}
	}
	return averageOrNil(parts)
}

func energyScore(level *string, fatigue *bool) *int {
	if level != nil {
		if e, ok := enums.EnergyLevelFrom(*level); ok {
			s := e.Score()
			return &s
		}
	}
	if fatigue != nil {
		s := 70
		if *fatigue {
			s = 30
		}
		return &s
	}
	return nil
}

// weeklyAverages is HealthMetricScorer::weeklyAverages: [mood, sleep, energy].
func weeklyAverages(logs []scoredLog) [3]*int {
	var buckets [3][]int
	for _, l := range logs {
		for i, v := range [3]*int{moodScore(l.Moods), sleepScore(l.SleepQuality, l.SleepDuration), energyScore(l.EnergyLevel, l.Fatigue)} {
			if v != nil {
				buckets[i] = append(buckets[i], *v)
			}
		}
	}
	return [3]*int{averageOrNil(buckets[0]), averageOrNil(buckets[1]), averageOrNil(buckets[2])}
}

// ---------------------------------------------------------------------------
// MoonPhase (backend/app/Services/HomePage/Support/MoonPhase.php).

const (
	newMoonEpoch  = 947182440 // 2000-01-06 18:14 UTC
	synodicMonth  = 29.530588853
	secondsPerDay = 86400
)

var moonPhases = []struct{ key, fa, en string }{
	{"new_moon", "ماه نو", "New moon"},
	{"waxing_crescent", "هلال افزاینده", "Waxing crescent"},
	{"first_quarter", "تربیع اول", "First quarter"},
	{"waxing_gibbous", "محدب افزاینده", "Waxing gibbous"},
	{"full_moon", "ماه کامل", "Full moon"},
	{"waning_gibbous", "محدب کاهنده", "Waning gibbous"},
	{"last_quarter", "تربیع آخر", "Last quarter"},
	{"waning_crescent", "هلال کاهنده", "Waning crescent"},
}

// MoonPhaseFor is MoonPhase::forDate for the Tehran midnight of day. The label table holds the
// fa/en pair only (`PHASES[$key][$locale] ?? PHASES[$key]['en']`).
func MoonPhaseFor(day civildate.Date, locale string) *jsonx.OrderedMap {
	days := float64(day.TehranMidnight().Unix()-newMoonEpoch) / secondsPerDay
	age := math.Mod(days, synodicMonth)
	if age < 0 {
		age += synodicMonth
	}
	fraction := age / synodicMonth
	index := int(math.Floor(fraction*8+0.5)) % 8
	illumination := (1 - math.Cos(2*math.Pi*fraction)) / 2

	p := moonPhases[index]
	label := p.en
	if locale == "fa" {
		label = p.fa
	}
	return jsonx.Obj(
		"key", p.key,
		"label", label,
		"illumination", jsonx.Float(phpround.Round(illumination, 2)),
		"age_days", jsonx.Float(phpround.Round(age, 1)),
	)
}

// ---------------------------------------------------------------------------
// CyclePhasePalette (backend/app/Services/HomePage/Support/CyclePhasePalette.php).

var phaseColors = map[string]string{
	"menstruation": "#F2567C",
	"follicular":   "#7BC99A",
	"ovulation":    "#F09BB5",
	"luteal":       "#9C8BD6",
}

// phaseColor is CyclePhasePalette::color ("" phase = null).
func phaseColor(phase string) any {
	if phase == "" {
		return nil
	}
	if c, ok := phaseColors[phase]; ok {
		return c
	}
	return "#CFCFE6"
}

// phaseLabel is CyclePhasePalette::label.
func phaseLabel(phase, locale string) any {
	if phase == "" {
		return nil
	}
	p, ok := enums.CyclePhaseFrom(phase)
	if !ok {
		return nil
	}
	return p.Label(locale)
}

func intOrNil(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}
