package analysis

import (
	"slices"
	"strconv"

	"github.com/ritme/backend-go/internal/cycle/insights"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// DayEntries is one logged day of health_log_entries (healthlog.Service.Range).
type DayEntries struct {
	Date    civildate.Date
	Entries []taxonomy.Entry
}

// Day is what the reports read of one logged day.
type Day struct {
	// Flow is bleeding.flow (light | medium | heavy | very_heavy), "" when not logged.
	Flow     string
	Spotting bool
	// Symptoms maps a symptom key (category.param.item, see symptomKey) to its weight in (0, 1].
	Symptoms map[string]float64
	// Moods are mood.moods codes.
	Moods []string
	// Sleep is sleep.duration (0_3 | 3_6 | 6_9 | 9_plus), "" when not logged.
	Sleep string
	// Energy is appetite_energy.energy (low | medium | high, legacy very_low | very_high).
	Energy string
	// Active: any activity type or duration logged.
	Active bool
	// Weight (kg), blood pressure (mmHg) and blood sugar (mg/dL); nil when not logged.
	Weight, Systolic, Diastolic, Glucose *float64
}

// symptomParams are the items/multi params whose items are symptoms (category.param).
var symptomParams = []string{
	"pain.location", "symptoms.digestive", "symptoms.general", "urogenital.symptoms", "skin_hair.symptoms",
	"breasts.symptoms", "discharge.symptoms",
}

// positiveMoods / negativeMoods split mood.moods for the good-mood share; sensitive and bored are
// neutral. Negative and neutral moods also count as symptoms (the trend legend shows «زودرنج»).
var (
	positiveMoods = []string{"calm", "happy", "energetic"}
	negativeMoods = []string{"irritable", "sad", "anxious", "angry", "frustrated"}
)

// levelWeight maps an item level onto (0, 1]: yes / severe 1, moderate ⅔, mild ⅓; no → 0.
func levelWeight(level string) float64 {
	switch level {
	case taxonomy.Yes, "severe":
		return 1
	case "moderate":
		return 2.0 / 3
	case "mild":
		return 1.0 / 3
	}
	return 0
}

func parseNum(s string) (float64, bool) {
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

// symptomKey is the stable symptom code clients label from the log-taxonomy namespace
// (categories.<category>.params.<param>.options.<item>).
func symptomKey(category, param, item string) string { return category + "." + param + "." + item }

// BuildDays turns day entries into Days keyed by date.
func BuildDays(rows []DayEntries) map[civildate.Date]*Day {
	out := make(map[civildate.Date]*Day, len(rows))
	for _, r := range rows {
		d := &Day{Symptoms: map[string]float64{}}
		for _, e := range r.Entries {
			d.add(e)
		}
		out[r.Date] = d
	}
	return out
}

func numPtr(e taxonomy.Entry) *float64 {
	if !e.Num.Valid {
		return nil
	}
	if f, ok := parseNum(e.Num.String); ok {
		return &f
	}
	return nil
}

func (d *Day) add(e taxonomy.Entry) {
	pk := e.ParamKey()
	switch {
	case pk == "bleeding.flow":
		d.Flow = e.Code.String
	case pk == "bleeding.spotting":
		d.Spotting = e.Code.String == taxonomy.Yes
	case pk == "mood.moods":
		d.Moods = append(d.Moods, e.Item)
		if !slices.Contains(positiveMoods, e.Item) {
			d.Symptoms[symptomKey(e.Category, e.Param, e.Item)] = 1
		}
	case pk == "sleep.duration":
		d.Sleep = e.Code.String
	case pk == "appetite_energy.energy":
		d.Energy = e.Code.String
	case pk == "activity.types":
		d.Active = true
	case pk == "activity.duration":
		if n := numPtr(e); n != nil && *n > 0 {
			d.Active = true
		}
	case pk == "measurements.weight":
		d.Weight = numPtr(e)
	case pk == "measurements.bp_systolic":
		d.Systolic = numPtr(e)
	case pk == "measurements.bp_diastolic":
		d.Diastolic = numPtr(e)
	case pk == "measurements.blood_sugar":
		d.Glucose = numPtr(e)
	case slices.Contains(symptomParams, pk) && e.Item != "":
		w := levelWeight(e.Code.String)
		if n := numPtr(e); n != nil && *n/10 > w {
			w = min(1, *n/10) // a 1–10 pain score without (or above) a level
		}
		if w > 0 {
			d.Symptoms[symptomKey(e.Category, e.Param, e.Item)] = w
		}
	}
}

// GoodMood: at least one positive mood and no more negative than positive ones. ok is false when no
// mood was logged.
func (d *Day) GoodMood() (good, ok bool) {
	if d == nil || len(d.Moods) == 0 {
		return false, false
	}
	pos, neg := 0, 0
	for _, m := range d.Moods {
		switch {
		case slices.Contains(positiveMoods, m):
			pos++
		case slices.Contains(negativeMoods, m):
			neg++
		}
	}
	return pos > 0 && pos >= neg, true
}

// SleepHours maps a duration bucket onto hours (bucket midpoints; 9+ → 10). ok is false when none.
func (d *Day) SleepHours() (float64, bool) {
	if d == nil {
		return 0, false
	}
	switch d.Sleep {
	case "0_3":
		return 1.5, true
	case "3_6":
		return 4.5, true
	case "6_9":
		return 7.5, true
	case "9_plus":
		return 10, true
	}
	return 0, false
}

// ShortSleep: a night under 6 hours (0_3 or 3_6).
func (d *Day) ShortSleep() bool { return d != nil && (d.Sleep == "0_3" || d.Sleep == "3_6") }

// HighEnergy: energy high (or the legacy very_high).
func (d *Day) HighEnergy() bool { return d != nil && (d.Energy == "high" || d.Energy == "very_high") }

// SevereCramps: abdomen or pelvis pain logged severe (or a score ≥ 7).
func (d *Day) SevereCramps() bool {
	if d == nil {
		return false
	}
	return d.Symptoms["pain.location.abdomen"] >= 0.7 || d.Symptoms["pain.location.pelvis"] >= 0.7
}

// flowScore maps bleeding.flow onto 1–4 (0 = none).
func flowScore(flow string) int {
	switch flow {
	case "light":
		return 1
	case "medium":
		return 2
	case "heavy":
		return 3
	case "very_heavy":
		return 4
	}
	return 0
}

// flowCodes are the flow codes by score (index 0 unused).
var flowCodes = []string{"", "light", "medium", "heavy", "very_heavy"}

// symptomLogs projects Days onto the insights strip input (symptom key → weight per day).
func symptomLogs(days map[civildate.Date]*Day) map[civildate.Date]insights.DayLog {
	out := make(map[civildate.Date]insights.DayLog, len(days))
	for date, d := range days {
		if len(d.Symptoms) > 0 {
			out[date] = d.Symptoms
		}
	}
	return out
}
