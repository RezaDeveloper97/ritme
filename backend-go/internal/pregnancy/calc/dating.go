package calc

import (
	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// The one dating rule set of the app (T-M10-18, SPEC v6 §3.1, deviation D-169): trimester by completed days, due
// date by Naegele (+ the optional cycle-length correction) and the ACOG Committee Opinion 700 re-dating rule. Every
// value comes from the clinical registry (internal/clinicalconfig, keys pregnancy.trimester, pregnancy.edd,
// pregnancy.edd.cycle_correction, pregnancy.redating) in the version valid on the day the rule is about; a zero day
// reads version 1. v1, v2, analysis, labs and the message facts call these functions — nothing else may compare
// weeks against trimester bounds (TestNoSecondTrimesterRule).

// referenceCycleDays is the cycle length Naegele's rule assumes (the correction adds the difference to it).
const referenceCycleDays = 28

// TrimesterOn is the trimester (1..3) of a gestational age of totalDays completed days, by the pregnancy.trimester
// version valid on day: T1 below t2_from_days (98 = 14+0), T2 below t3_from_days (196 = 28+0), else T3.
func TrimesterOn(totalDays int, day civildate.Date) int {
	switch {
	case totalDays < clinicalconfig.Int(clinicalconfig.KeyPregnancyTrimester, "t2_from_days", day):
		return 1
	case totalDays < clinicalconfig.Int(clinicalconfig.KeyPregnancyTrimester, "t3_from_days", day):
		return 2
	}
	return 3
}

// TrimesterOfWeek is the trimester of 1-based pregnancy week n (its first day, (n−1)×7 completed days) by the version
// valid on day.
func TrimesterOfWeek(n int, day civildate.Date) int { return TrimesterOn((n-1)*7, day) }

// TrimesterStartDays are the completed days at which trimesters 1, 2 and 3 begin (0, t2_from_days, t3_from_days).
func TrimesterStartDays(day civildate.Date) [3]int {
	return [3]int{
		0,
		clinicalconfig.Int(clinicalconfig.KeyPregnancyTrimester, "t2_from_days", day),
		clinicalconfig.Int(clinicalconfig.KeyPregnancyTrimester, "t3_from_days", day),
	}
}

// NaegeleDays is the LMP → due date span (pregnancy.edd naegele_days, 280) of the version valid on day.
func NaegeleDays(day civildate.Date) int {
	return clinicalconfig.Int(clinicalconfig.KeyPregnancyEDD, "naegele_days", day)
}

// EDDFromStart is the due date of a pregnancy whose day 0 (LMP, or the start a scan or a manual entry implies) is
// start: start + naegele_days, the version valid on start. Version 1 is the former `LmpDate.Date.AddDays(280)`.
func EDDFromStart(start civildate.Date) civildate.Date { return start.AddDays(NaegeleDays(start)) }

// EDDFromLMP is the due date by the LMP: EDDFromStart(lmp), plus (cycleDays − 28) when
// pregnancy.edd.cycle_correction is enabled (version valid on lmp; off by default) and the effective cycle length is
// known (cycleDays > 0, from the cycle engine).
func EDDFromLMP(lmp civildate.Date, cycleDays int) civildate.Date {
	edd := EDDFromStart(lmp)
	if cycleDays > 0 && clinicalconfig.Int(clinicalconfig.KeyPregnancyEDDCycleCorrect, "enabled", lmp) == 1 {
		edd = edd.AddDays(cycleDays - referenceCycleDays)
	}
	return edd
}

// Basis values of a Redating (the age_source the rule picks).
const (
	BasisLMP        = "lmp"
	BasisUltrasound = "ultrasound"
	BasisManual     = "manual"
)

// Redating is the ACOG re-dating decision for a scan (SPEC v6 §3.1). With an LMP: the discrepancy is |GA by scan −
// GA by LMP| on the scan day, the band is the first pregnancy.redating row (version valid on the scan day) whose
// ga_until_days is ≥ the GA by LMP on the scan day (null = open), and the scan replaces the LMP dating only when the
// discrepancy is more than the band's max_diff_days. Without an LMP the scan dates the pregnancy (Replaced false,
// the comparison fields nil).
type Redating struct {
	Basis           string
	Replaced        bool
	DiscrepancyDays *int
	BandUntilDays   *int // nil = the open last band (28+0 on) or no comparison
	LimitDays       *int
}

// Redate applies the rule to a scan (scanDate, gaAtScanDays = weeks × 7 + days) against an LMP (hasLMP false =
// none).
func Redate(lmp civildate.Date, hasLMP bool, scanDate civildate.Date, gaAtScanDays int) Redating {
	if !hasLMP {
		return Redating{Basis: BasisUltrasound}
	}
	byLMP := lmp.DiffDays(scanDate)
	diff := gaAtScanDays - byLMP
	if diff < 0 {
		diff = -diff
	}
	until, limit := redatingBand(byLMP, scanDate)
	r := Redating{Basis: BasisLMP, DiscrepancyDays: &diff, BandUntilDays: until, LimitDays: &limit}
	if diff > limit {
		r.Basis, r.Replaced = BasisUltrasound, true
	}
	return r
}

// redatingBand is the pregnancy.redating row of a GA by LMP (days) on the scan day: its ga_until_days (nil = open)
// and max_diff_days. Past the last bounded row without an open one, the last row applies.
func redatingBand(gaByLMP int, scanDate civildate.Date) (*int, int) {
	rows := clinicalconfig.Rows(clinicalconfig.KeyPregnancyRedating, scanDate)
	var until *int
	limit := 0
	for _, r := range rows {
		lim, _ := r.Num("max_diff_days")
		u, bounded := r.Num("ga_until_days")
		limit = int(lim)
		if !bounded {
			return nil, limit
		}
		n := int(u)
		until = &n
		if gaByLMP <= n {
			return until, limit
		}
	}
	return until, limit
}

// JSON is the `dating` block {basis, replaced, discrepancy_days, band_until_days, limit_days}.
func (r Redating) JSON() *jsonx.OrderedMap {
	return jsonx.Obj(
		"basis", nullableString(r.Basis),
		"replaced", r.Replaced,
		"discrepancy_days", intPtr(r.DiscrepancyDays),
		"band_until_days", intPtr(r.BandUntilDays),
		"limit_days", intPtr(r.LimitDays),
	)
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func intPtr(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

// Dating is the profile's dating block: the basis in effect (age_source) and, when the profile holds both an LMP and
// a scan, the re-dating comparison of the two (Replaced = the rule picks the scan). Basis "" without an age source.
func (c *Calculator) Dating() Redating {
	if c.p == nil {
		return Redating{}
	}
	basis := ""
	if c.p.AgeSource.Valid {
		basis = c.p.AgeSource.String
	}
	scan := c.p.UltrasoundDate.Valid && c.p.UltrasoundWeeks.Valid && c.p.UltrasoundDays.Valid
	if !scan || !c.p.LmpDate.Valid {
		return Redating{Basis: basis}
	}
	r := Redate(c.p.LmpDate.Date, true, c.p.UltrasoundDate.Date,
		int(c.p.UltrasoundWeeks.Int32)*7+int(c.p.UltrasoundDays.Int32))
	r.Basis = basis
	return r
}
