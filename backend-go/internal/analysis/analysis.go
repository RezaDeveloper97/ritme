// Package analysis is the «تحلیل» tab read model (B-N3-07, Night & Bloom An_* artboards, Go only):
// GET /api/v1/analysis/{summary,cycle,period,symptoms,correlations,body,monthly/:ym}.
//
// Everything here is a pure function of an Input (the user's confirmed cycle history, the taxonomy v2
// day logs of health_log_entries, a few profile fields, the Plus entitlement and the request day), so
// the reports are golden-tested on seeded histories with a fixed clock. The handlers only load the
// Input and serialise.
//
// The reports are descriptive, never diagnostic: ranges are the FIGO 2018 reference ranges for
// non-pregnant adults, every correlation carries `not_causal: true`, and nothing is logged (no health
// value ever reaches a log line or an analytics event).
//
// Reuse: medians, plausibility bounds, confirmed starts and the per-symptom typical-cycle strips come
// from internal/cycle/insights (B-N1-08); the day logs from internal/healthlog (B-N3-01).
package analysis

import (
	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// FIGO 2018 normal-uterine-bleeding reference ranges for non-pregnant adults of reproductive age
// (Munro MG, Critchley HOD, Fraser IS; FIGO Menstrual Disorders Committee. «The two FIGO systems for
// normal and abnormal uterine bleeding symptoms and classification of causes of abnormal uterine
// bleeding in the reproductive years: 2018 revisions.» Int J Gynaecol Obstet 2018;143:393–408, Table 1).
const (
	// FIGOCycleMin / FIGOCycleMax: normal frequency is a cycle of 24–38 days (inclusive). Shorter is
	// «frequent», longer «infrequent».
	FIGOCycleMin = 24
	FIGOCycleMax = 38
	// FIGOPeriodMax: normal duration is ≤ 8 days; longer is «prolonged».
	FIGOPeriodMax = 8
	// FIGOVariationYoung / FIGOVariationAdult / FIGOVariationLate: normal regularity is a shortest-to-
	// longest cycle variation of ≤ 7–9 days — ≤ 9 days at ages 18–25, ≤ 7 days at 26–41, ≤ 9 days at
	// 42–45 (the 2018 revision's age bands).
	FIGOVariationYoung = 9
	FIGOVariationAdult = 7
	FIGOVariationLate  = 9
	// FIGOAgeMin / FIGOAgeMax bound the ages the ranges describe; outside them (teens, perimenopause)
	// the payload says `applies: false` and the client words the ranges as a rough guide.
	FIGOAgeMin = 18
	FIGOAgeMax = 45
)

// Minimum-data rules.
const (
	// MinPatternCycles: a pattern (regularity, symptom pattern, co-symptoms of the period) needs at
	// least 3 completed cycles («الگو فقط وقتی نمایش داده می‌شود که علامت در حداقل ۳ سیکل ثبت شده باشد»).
	MinPatternCycles = 3
	// MinTrendPoints: a trend (bars, a line, a delta) needs at least 2 points.
	MinTrendPoints = 2
)

// Cycle phases (the analysis split of a cycle: period, follicular, fertile window, luteal — the same
// ovulation = next start − 14 and 6-day fertile window the cycle resolver uses).
const (
	PhasePeriod     = "period"
	PhaseFollicular = "follicular"
	PhaseFertile    = "fertile"
	PhaseLuteal     = "luteal"
)

// Phases are the phases in cycle order.
var Phases = []string{PhasePeriod, PhaseFollicular, PhaseFertile, PhaseLuteal}

// fertileDaysBeforeOvulation: the fertile window is the ovulation day and the 5 days before it.
const fertileDaysBeforeOvulation = 5

// FIGO are the reference ranges valid on one day: versioned clinical config since B-N9-11 (D-88:
// cycle.figo.frequency / duration / regularity); the constants above are version 1, the store's fallback.
type FIGO struct {
	CycleMin, CycleMax, PeriodMax                 int
	VariationYoung, VariationAdult, VariationLate int
}

// FIGOOn reads the ranges valid on d (a zero d = version 1).
func FIGOOn(d civildate.Date) FIGO {
	n := func(key, field string) int { return clinicalconfig.Int(key, field, d) }
	return FIGO{
		CycleMin: n(clinicalconfig.KeyFIGOFrequency, "min"), CycleMax: n(clinicalconfig.KeyFIGOFrequency, "max"),
		PeriodMax:      n(clinicalconfig.KeyFIGODuration, "max"),
		VariationYoung: n(clinicalconfig.KeyFIGORegularity, "young"),
		VariationAdult: n(clinicalconfig.KeyFIGORegularity, "adult"),
		VariationLate:  n(clinicalconfig.KeyFIGORegularity, "late"),
	}
}

// VariationLimit is the normal shortest-to-longest variation for an age (unknown age → adult).
func (f FIGO) VariationLimit(age int) int {
	switch {
	case age <= 0:
		return f.VariationAdult
	case age <= 25:
		return f.VariationYoung
	case age <= 41:
		return f.VariationAdult
	}
	return f.VariationLate
}

// FIGOVariationLimit is FIGO.VariationLimit of version 1.
func FIGOVariationLimit(age int) int { return FIGOOn(civildate.Date{}).VariationLimit(age) }

// FIGOApplies reports whether the adult ranges describe this age (unknown age → yes).
func FIGOApplies(age int) bool { return age <= 0 || (age >= FIGOAgeMin && age <= FIGOAgeMax) }
