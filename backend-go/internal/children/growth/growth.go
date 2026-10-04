// Package growth computes WHO Child Growth Standards z-scores and percentiles from the LMS tables (seeds/who).
//
// Method (WHO Anthro / "Computation of centiles and z-scores", WHO 2006 §7 and the igrowup macro):
//
//	z = ((y/M)^L − 1) / (L·S)            (L ≠ 0)      z = ln(y/M) / S   (L = 0)
//
// For weight-for-age (L ≠ 1, skewed) WHO restricts the LMS curve to ±3 SD and extends it linearly beyond, with the
// distance between the 2 and 3 SD curves as the unit:
//
//	z > 3:   z* = 3 + (y − SD3pos) / (SD3pos − SD2pos)
//	z < −3:  z* = −3 + (y − SD3neg) / (SD2neg − SD3neg)
//
// Length/height and head circumference have L = 1 (normal) and use z directly. Percentile = 100·Φ(z). Age is whole
// days since birth (the expanded tables are daily; no interpolation needed).
package growth

import (
	"math"

	"github.com/ritme/backend-go/seeds/who"
)

// Normal band of the screens: P3 to P97 («بازه طبیعی (صدک ۳ تا ۹۷)»).
const (
	BandLowPercentile  = 3.0
	BandHighPercentile = 97.0
)

// Z-scores of the reference curves the growth chart draws.
var (
	ZP3  = -1.880793608 // Φ⁻¹(0.03)
	ZP15 = -1.036433389 // Φ⁻¹(0.15)
	ZP85 = 1.036433389
	ZP97 = 1.880793608
)

// value is the measurement at z on the LMS curve (no restriction).
func value(p who.LMS, z float64) float64 {
	if p.L == 0 {
		return p.M * math.Exp(p.S*z)
	}
	return p.M * math.Pow(1+p.L*p.S*z, 1/p.L)
}

// rawZ is the unrestricted LMS z-score of y.
func rawZ(p who.LMS, y float64) float64 {
	if p.L == 0 {
		return math.Log(y/p.M) / p.S
	}
	return (math.Pow(y/p.M, p.L) - 1) / (p.L * p.S)
}

// Z is the WHO z-score of measurement y (kg or cm) for ind at p; restricted beyond ±3 SD for weight.
func Z(ind who.Indicator, p who.LMS, y float64) float64 {
	z := rawZ(p, y)
	if ind != who.Weight {
		return z
	}
	switch {
	case z > 3:
		sd3, sd2 := value(p, 3), value(p, 2)
		return 3 + (y-sd3)/(sd3-sd2)
	case z < -3:
		sd3, sd2 := value(p, -3), value(p, -2)
		return -3 + (y-sd3)/(sd2-sd3)
	}
	return z
}

// Percentile is 100·Φ(z).
func Percentile(z float64) float64 { return 50 * (1 + math.Erf(z/math.Sqrt2)) }

// ValueAt is the measurement on the curve of z-score z (the reference curves of the chart). Beyond ±3 SD weight uses
// the same linear extension as Z, so ValueAt and Z are inverse.
func ValueAt(ind who.Indicator, p who.LMS, z float64) float64 {
	if ind == who.Weight {
		switch {
		case z > 3:
			sd3, sd2 := value(p, 3), value(p, 2)
			return sd3 + (z-3)*(sd3-sd2)
		case z < -3:
			sd3, sd2 := value(p, -3), value(p, -2)
			return sd3 + (z+3)*(sd2-sd3)
		}
	}
	return value(p, z)
}

// Result is one measurement's position on the standard.
type Result struct {
	Z          float64
	Percentile float64
	InBand     bool // P3 ≤ percentile ≤ P97
}

// Assess places y (kg or cm, > 0) for a child of sex aged day days; ok false outside 0–1856 days, for an unknown sex or
// a non-positive value.
func Assess(ind who.Indicator, sex who.Sex, day int, y float64) (Result, bool) {
	if y <= 0 || math.IsNaN(y) || math.IsInf(y, 0) {
		return Result{}, false
	}
	p, ok := who.At(ind, sex, day)
	if !ok {
		return Result{}, false
	}
	z := Z(ind, p, y)
	pc := Percentile(z)
	return Result{Z: z, Percentile: pc, InBand: pc >= BandLowPercentile && pc <= BandHighPercentile}, true
}

// Round rounds v to n decimals (half away from zero).
func Round(v float64, n int) float64 {
	f := math.Pow(10, float64(n))
	return math.Round(v*f) / f
}
