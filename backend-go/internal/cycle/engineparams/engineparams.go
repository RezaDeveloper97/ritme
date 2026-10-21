// Package engineparams builds the cycle engine v6 parameters (engine.Params) from the clinical-config snapshot
// valid on a date (DECISIONS.md X-11). It is the only reader of the engine.*, cycle.figo.* and cycle.teen.* keys
// for v6; the engine package itself never imports clinicalconfig (X-01).
package engineparams

import (
	"strconv"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// SchemaVersion is bumped whenever the meaning or the set of engine.Params fields changes, so cached prepared
// states (README §3, keyed by Fingerprint) are rebuilt after a deploy that changes how params are derived.
const SchemaVersion = 1

// Keys is the one list of clinical-config keys the v6 engine reads. engine.fertile_window is listed for its
// `before` field; its `after` field is ignored by v6 (O−5..O, §2.18) and retired in T-M9-51, so a version that only
// changes `after` still changes the fingerprint (harmless: one re-prepare).
func Keys() []string {
	return []string{
		clinicalconfig.KeyPeriodHardCap,
		clinicalconfig.KeyWarningCap,
		clinicalconfig.KeyMaxLateDays,
		clinicalconfig.KeyFertileWindow,
		clinicalconfig.KeyLuteal,
		clinicalconfig.KeyBBT,
		clinicalconfig.KeyLHBBTConflict,
		clinicalconfig.KeyPregnancyTest,
		clinicalconfig.KeyStaleAnchor,
		clinicalconfig.KeyFIGOFrequency,
		clinicalconfig.KeyFIGODuration,
		clinicalconfig.KeyFIGORegularity,
		clinicalconfig.KeyTeenFrequency,
	}
}

// For is engine.Params with every clinical value taken from the version valid on d (a zero d reads version 1).
func For(d civildate.Date) engine.Params {
	p := engine.DefaultParams()
	num := func(key, field string) int { return clinicalconfig.Int(key, field, d) }

	p.PeriodHardCap = num(clinicalconfig.KeyPeriodHardCap, "days")
	p.WarningCapExtra = num(clinicalconfig.KeyWarningCap, "extra_days")
	p.WarningCapMin = num(clinicalconfig.KeyWarningCap, "min_days")
	p.MaxLateDays = num(clinicalconfig.KeyMaxLateDays, "days")
	p.FertileBefore = num(clinicalconfig.KeyFertileWindow, "before")

	p.Luteal = engine.LutealParams{
		Default: num(clinicalconfig.KeyLuteal, "default"),
		Min:     num(clinicalconfig.KeyLuteal, "min"),
		Max:     num(clinicalconfig.KeyLuteal, "max"),
	}
	p.BBT = engine.BBTParams{
		Strict:      clinicalconfig.Num(clinicalconfig.KeyBBT, "strict", d) != 0,
		MinReadings: num(clinicalconfig.KeyBBT, "min_readings"),
		RiseCenti:   int32(num(clinicalconfig.KeyBBT, "rise_centi")), //nolint:gosec // registry bounds 5..50
	}
	p.LHBBTConflictDays = num(clinicalconfig.KeyLHBBTConflict, "days")
	p.PregnancyTestCycleDaysLate = num(clinicalconfig.KeyPregnancyTest, "cycle_days_late")
	p.PregnancyTestTTCDaysLate = num(clinicalconfig.KeyPregnancyTest, "ttc_days_late")
	p.StaleAnchorExtraDays = num(clinicalconfig.KeyStaleAnchor, "extra_days")

	p.FIGO = engine.FIGOParams{
		Frequency: engine.RangeParams{
			Min: num(clinicalconfig.KeyFIGOFrequency, "min"),
			Max: num(clinicalconfig.KeyFIGOFrequency, "max"),
		},
		DurationMax:     num(clinicalconfig.KeyFIGODuration, "max"),
		RegularityYoung: num(clinicalconfig.KeyFIGORegularity, "young"),
		RegularityAdult: num(clinicalconfig.KeyFIGORegularity, "adult"),
		RegularityLate:  num(clinicalconfig.KeyFIGORegularity, "late"),
	}
	p.TeenFrequency = engine.RangeParams{
		Min: num(clinicalconfig.KeyTeenFrequency, "min"),
		Max: num(clinicalconfig.KeyTeenFrequency, "max"),
	}
	return p
}

// Fingerprint identifies the params valid on d: "p<schema>" plus ":" and clinicalconfig.Fingerprint of Keys when
// any key is past version 1 ("p1" with every key at its code default).
func Fingerprint(d civildate.Date) string {
	fp := "p" + strconv.Itoa(SchemaVersion)
	if cfg := clinicalconfig.Fingerprint(d, Keys()...); cfg != "" {
		fp += ":" + cfg
	}
	return fp
}
