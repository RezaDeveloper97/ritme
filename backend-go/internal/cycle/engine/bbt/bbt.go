// Package bbt is the pure "three over six" basal-body-temperature shift detector of the cycle engine v6
// (docs/engine-v6/SPEC.md §2.17, DECISIONS.md E-16). It has no I/O, no globals and no clock; temperatures are
// integer hundredths of °C and dates are any int32-based day type (engine.Day).
//
// Rule (E-16): a cycle needs at least MinReadings readings. The baseline of reading i is the maximum of the 6
// readings before it. Three consecutive *readings* (a missing day does not break the run) above the baseline are a
// shift; in strict mode the third one must also be ≥ baseline + RiseCenti. Ovulation is the calendar day before
// the first high reading. The shift confirms only once its third reading exists (the detector never projects).
package bbt

// Baseline is the number of readings before the first high reading whose maximum is the baseline ("six").
const Baseline = 6

// Run is the number of consecutive high readings that confirm a shift ("three").
const Run = 3

// Reading is one basal body temperature on day Date, in hundredths of °C.
type Reading[D ~int32] struct {
	Date  D
	Centi int32
}

// Params is engine.bbt (clinical config, X-11).
type Params struct {
	Strict      bool  // the third high reading must be ≥ RiseCenti over the baseline
	MinReadings int   // readings needed in the cycle (all of the cycle's readings, not only up to the shift)
	RiseCenti   int32 // hundredths of °C
}

// Result is a detected shift. With OK false the other fields are zero.
type Result[D ~int32] struct {
	OK bool
	// Ovulation is the day before the first high reading.
	Ovulation D
	// First is the index of the first high reading, Confirm the index of the third (confirming) one.
	First   int
	Confirm int
	// Baseline is the maximum of the 6 readings before First (the coverline).
	Baseline int32
}

// Detect finds the first confirmed shift in one cycle's readings. rs must be sorted by date with at most one reading
// per day (the engine's Prepare guarantees it); it is not modified and nothing is allocated. Fewer than
// max(MinReadings, Baseline + Run) readings never confirm.
func Detect[D ~int32](rs []Reading[D], p Params) Result[D] {
	if len(rs) < p.MinReadings || len(rs) < Baseline+Run {
		return Result[D]{}
	}
	for i := Baseline; i+Run <= len(rs); i++ {
		base := rs[i-Baseline].Centi
		for _, r := range rs[i-Baseline+1 : i] {
			base = max(base, r.Centi)
		}
		if !high(rs[i:i+Run], base) {
			continue
		}
		if p.Strict && int64(rs[i+Run-1].Centi) < int64(base)+int64(p.RiseCenti) {
			continue
		}
		return Result[D]{OK: true, Ovulation: rs[i].Date - 1, First: i, Confirm: i + Run - 1, Baseline: base}
	}
	return Result[D]{}
}

// high reports whether every reading of run is above base.
func high[D ~int32](run []Reading[D], base int32) bool {
	for _, r := range run {
		if r.Centi <= base {
			return false
		}
	}
	return true
}
