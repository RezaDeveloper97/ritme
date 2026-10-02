// Package fertility is the TTC day log and home tiles (docs/fertility-ttc/README.md, T-M5-01):
// GET /fertility/today and GET|PUT /fertility/days/{date}, Go only. A day merges the new
// fertility_logs row (LH test, cervical mucus, BBT time) with the fertility columns of
// daily_health_logs (BBT, intercourse, symptoms, note), which it writes through the healthlog
// service so POST /health-logs' side effects run unchanged.
package fertility

import (
	"database/sql"

	"github.com/ritme/backend-go/internal/cycle/resolver"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/fertility/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Enum values (README → Data / API).
var (
	LHResults        = []string{"negative", "faint", "positive"}
	CervicalMucus    = []string{"dry", "sticky", "creamy", "egg_white"}
	IntercourseTypes = []string{"unprotected", "protected"}
	// Symptoms are the log's symptom chips, each backed by a daily_health_logs column.
	Symptoms = []string{SymptomOvarianPain, SymptomBloating, SymptomBreastSensitivity, SymptomSpotting}
)

// Symptom chips.
const (
	SymptomOvarianPain       = "ovarian_pain"
	SymptomBloating          = "bloating"
	SymptomBreastSensitivity = "breast_sensitivity"
	SymptomSpotting          = "spotting"
)

// symptomColumns maps the intensity-backed chips to their daily_health_logs column. spotting is
// the boolean `spotting` column.
var symptomColumns = map[string]string{
	SymptomOvarianPain:       "ovarian_pain_intensity",
	SymptomBloating:          "bloating_intensity",
	SymptomBreastSensitivity: "breast_sensitivity_intensity",
}

// selectedIntensity is what a chip writes into an intensity column that has no value yet
// (enums.PainIntensity has no "mild"; its lowest case is "low"). A value already logged
// through the full health log (medium, high) is kept.
const selectedIntensity = string(enums.PainIntensityLow)

// Day is the merged day.
type Day struct {
	Date        civildate.Date
	LH          sql.NullString
	Mucus       sql.NullString
	BBT         sql.NullString // decimal(4,2) text, "36.50"
	BBTTime     sql.NullString // "HH:MM:SS" from the TIME column
	Intercourse sql.NullString
	Symptoms    []string // in Symptoms order
	Note        sql.NullString
}

// DayFromRow builds the day from the merged-day query.
func DayFromRow(date civildate.Date, r store.GetMergedDayRow) Day {
	d := Day{
		Date: date, LH: strongerLH(r.LhTest, r.EntryLhTest), Mucus: moreFertileMucus(r.CervicalMucus, entryMucus(r.EntryMucus)),
		BBT:     r.BasalBodyTemperature,
		BBTTime: r.BbtTime, Intercourse: r.IntercourseType, Note: r.Notes, Symptoms: []string{},
	}
	intensity := map[string]sql.NullString{
		SymptomOvarianPain:       r.OvarianPainIntensity,
		SymptomBloating:          r.BloatingIntensity,
		SymptomBreastSensitivity: r.BreastSensitivityIntensity,
	}
	for _, s := range Symptoms {
		if s == SymptomSpotting {
			if r.Spotting.Valid && r.Spotting.Bool {
				d.Symptoms = append(d.Symptoms, s)
			}
			continue
		}
		if v := intensity[s]; v.Valid && v.String != "" {
			d.Symptoms = append(d.Symptoms, s)
		}
	}
	return d
}

// The log sheet (taxonomy v2: measurements.lh_test, discharge.consistency) and the /fertility log
// (fertility_logs) are not synced (QUESTIONS #80); a day merges both like the TTC analysis does
// (internal/analysis TTCSignals): the stronger LH result and the more fertile mucus win (B-N3-14b).
var (
	lhRank    = map[string]int{"negative": 0, "faint": 1, "positive": 2}
	mucusRank = map[string]int{"dry": 0, "sticky": 1, "creamy": 2, "egg_white": 3}
)

// strongerLH is the stronger of two LH results (an unknown value never wins).
func strongerLH(a, b sql.NullString) sql.NullString { return stronger(lhRank, a, b) }

// moreFertileMucus is the more fertile of two mucus values.
func moreFertileMucus(a, b sql.NullString) sql.NullString { return stronger(mucusRank, a, b) }

func stronger(rank map[string]int, a, b sql.NullString) sql.NullString {
	ra, okA := rank[a.String]
	rb, okB := rank[b.String]
	okA, okB = okA && a.Valid, okB && b.Valid
	switch {
	case okA && okB && rb > ra, !okA && okB:
		return b
	case okA:
		return a
	}
	return sql.NullString{}
}

// entryMucus maps the log sheet's discharge consistency onto the /fertility mucus values: none → dry;
// watery has no /fertility value (the day's PUT accepts only CervicalMucus) and is left out.
func entryMucus(v sql.NullString) sql.NullString {
	if v.Valid && v.String == "none" {
		return sql.NullString{String: "dry", Valid: true}
	}
	if _, ok := mucusRank[v.String]; v.Valid && ok {
		return v
	}
	return sql.NullString{}
}

// Chance is the chance card: the cycle view's fertility_level as a level, label and 0–5 bars.
type Chance struct {
	Level *string // nil = unknown (no resolvable cycle)
	Bars  int
}

// chanceLevels maps the cycle engine's fertility_level onto the card's levels and bars.
// The v1.1 resolver emits low|medium|high|peak|unknown; the legacy very_* cases fold in.
var chanceLevels = map[enums.FertilityLevel]struct {
	level string
	bars  int
}{
	enums.FertilityLevelNone:     {"none", 0},
	enums.FertilityLevelVeryLow:  {"low", 1},
	enums.FertilityLevelLow:      {"low", 1},
	enums.FertilityLevelMedium:   {"medium", 3},
	enums.FertilityLevelHigh:     {"high", 4},
	enums.FertilityLevelVeryHigh: {"high", 4},
	enums.FertilityLevelPeak:     {"peak", 5},
}

// ChanceOf is the chance card of a resolved cycle status.
func ChanceOf(level enums.FertilityLevel) Chance {
	if m, ok := chanceLevels[level]; ok {
		l := m.level
		return Chance{Level: &l, Bars: m.bars}
	}
	return Chance{}
}

// JSON is {level, label, bars}; an unknown level is null with the "unknown" label.
func (c Chance) JSON(locale string) *jsonx.OrderedMap {
	if c.Level == nil {
		return jsonx.Obj("level", nil, "label", Label("chance", "unknown", locale), "bars", 0)
	}
	return jsonx.Obj("level", *c.Level, "label", Label("chance", *c.Level, locale), "bars", c.Bars)
}

// Context is what the cycle engine says about a date: its cycle day and chance.
type Context struct {
	CycleDay *int
	Chance   Chance
}

// ContextOf is the cycle day and chance of a resolved status.
func ContextOf(st resolver.Status) Context {
	return Context{CycleDay: st.CycleDay, Chance: ChanceOf(st.FertilityLevel)}
}

func nullable(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	return s.String
}

// hhmm is a TIME column value as "HH:MM".
func hhmm(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	if len(s.String) >= 5 {
		return s.String[:5]
	}
	return s.String
}

func cycleDay(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

// JSON is the merged day of GET|PUT /fertility/days/{date}.
func (d Day) JSON(cx Context, locale string) *jsonx.OrderedMap {
	return jsonx.Obj(
		"date", d.Date.String(),
		"cycle_day", cycleDay(cx.CycleDay),
		"lh", nullable(d.LH),
		"mucus", nullable(d.Mucus),
		"bbt", nullable(d.BBT),
		"bbt_time", hhmm(d.BBTTime),
		"intercourse", nullable(d.Intercourse),
		"symptoms", d.Symptoms,
		"note", nullable(d.Note),
		"chance", cx.Chance.JSON(locale),
	)
}

// labeled is {value, label} (both null when unset).
func labeled(group string, v sql.NullString, locale string) *jsonx.OrderedMap {
	if !v.Valid {
		return jsonx.Obj("value", nil, "label", nil)
	}
	return jsonx.Obj("value", v.String, "label", Label(group, v.String, locale))
}

// TodayJSON is GET /fertility/today: the three home tiles, the chance and the cycle day.
func (d Day) TodayJSON(cx Context, locale string) *jsonx.OrderedMap {
	return jsonx.Obj(
		"date", d.Date.String(),
		"cycle_day", cycleDay(cx.CycleDay),
		"chance", cx.Chance.JSON(locale),
		"lh", labeled("lh", d.LH, locale),
		"bbt", jsonx.Obj("value", nullable(d.BBT)),
		"intercourse", labeled("intercourse", d.Intercourse, locale),
	)
}
