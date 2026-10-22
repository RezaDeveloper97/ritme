package labs

import (
	"strings"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
)

// Fallback ranges, borderline and per-value confirmation (T-M10-20, SPEC v6 §3.2–§3.3, deviation D-171; DECISIONS.md
// H-07, H-08). Every number comes from the clinical registry (internal/clinicalconfig) in the version valid on the lab
// date (labDate: the sheet's date, else the upload day):
//
//   - lab.ocr min_confidence: a value read below it waits for the user's confirmation (lab_markers.confirmed_at) and
//     is excluded from the status rules, red flags, trends and the AI payload until then; editing a value confirms it;
//   - lab.borderline enabled / margin: 1 = the «مرزی» class within margin outside the range (today's behaviour);
//     0 = no class, only the near_edge hint;
//   - only when the sheet prints no range (the sheet always wins) and the value is in the catalog unit, the «محدوده
//     عمومی» of the registry replaces the catalog's typical range for hemoglobin (lab.hb, by pregnancy and trimester on
//     the lab date — calc.TrimesterOn), ferritin (lab.ferritin.low, lab.ferritin.inflamed when CRP is above its printed
//     range on the same sheet), vitamin D (lab.vitd bands), TSH in pregnancy (lab.tsh.pregnancy) and HbA1c (lab.hba1c
//     bands, never a diagnosis).

// Catalog codes with a registry fallback range.
const (
	codeHemoglobin = "hemoglobin"
	codeFerritin   = "ferritin"
	codeVitaminD   = "vitamin_d"
	codeTSH        = "tsh"
	codeHbA1c      = "hba1c"
)

// Bands of the banded markers (vitamin D, HbA1c): the category the registry gives a value, shown next to the state.
const (
	BandVitDDeficient    = "vitd_deficient"    // < deficient_below (12 ng/mL)
	BandVitDInsufficient = "vitd_insufficient" // < sufficient_from (20) «کمتر از حد کافی»
	BandVitDSufficient   = "vitd_sufficient"
	BandHbA1cTypical     = "hba1c_typical" // < prediabetes_from (5.7 %)
	BandHbA1cRaised      = "hba1c_raised"  // < diabetes_from (6.5 %): «معمولاً پزشک بررسی بیشتری می‌خواهد»
	BandHbA1cHigh        = "hba1c_high"
)

// Range bases: why a general range differs from the everyday adult one.
const (
	BasisPregnancy   = "pregnancy"
	BasisPregnancyT2 = "pregnancy_t2"
	BasisInflamed    = "inflammation"
)

// maxGestationDays: a lab dated more than this after the pregnancy's day 0 is not read as a pregnancy lab.
const maxGestationDays = 44 * 7

// sheetFacts is what the fallback ranges of one lab sheet depend on besides the catalog.
type sheetFacts struct {
	Date      civildate.Date // the lab date: the registry version and the gestational age are read on it
	Pregnant  bool
	Trimester int  // 1..3 on Date, 0 = unknown
	Inflamed  bool // CRP above its printed range on the same sheet (a confirmed value)
}

// dating says whether the user was pregnant on a lab date. Zero = not pregnant (no active pregnancy profile).
type dating struct{ calc *calc.Calculator }

// facts are the sheet facts of a lab dated d (Inflamed is filled by evaluateSheet).
func (p dating) facts(d civildate.Date) sheetFacts {
	f := sheetFacts{Date: d}
	if p.calc == nil {
		return f
	}
	ga := p.calc.GestationalAgeOn(d)
	if !ga.Valid {
		f.Pregnant = true // pregnant today, dating unknown: the pregnancy range without a trimester
		return f
	}
	if ga.TotalDays < 0 || ga.TotalDays > maxGestationDays {
		return f // taken before this pregnancy (or the dating is far off): the everyday range
	}
	f.Pregnant, f.Trimester = true, ga.Trimester
	return f
}

// borderline is lab.borderline on a day.
type borderline struct {
	Enabled bool
	Margin  float64
}

func borderlineOn(d civildate.Date) borderline {
	return borderline{
		Enabled: clinicalconfig.Int(clinicalconfig.KeyLabBorderline, "enabled", d) == 1,
		Margin:  clinicalconfig.Num(clinicalconfig.KeyLabBorderline, "margin", d),
	}
}

// MinConfidence is lab.ocr min_confidence on d: a value read below it needs the user's confirmation.
func MinConfidence(d civildate.Date) float64 {
	return clinicalconfig.Num(clinicalconfig.KeyLabOCR, "min_confidence", d)
}

// classifyRange is the state of v against r with the borderline rule b, and whether v is outside the range by at most
// b.Margin of the bound (near_edge; the state is then borderline_* only when b.Enabled).
func classifyRange(v *float64, r Range, b borderline) (string, bool) {
	if v == nil || r.Empty() {
		return StateUnknown, false
	}
	x := *v
	if r.Low != nil && x < *r.Low {
		if *r.Low > 0 && x >= *r.Low*(1-b.Margin) {
			if b.Enabled {
				return StateBorderlineLow, true
			}
			return StateLow, true
		}
		return StateLow, false
	}
	if r.High != nil && x > *r.High {
		if *r.High > 0 && x <= *r.High*(1+b.Margin) {
			if b.Enabled {
				return StateBorderlineHigh, true
			}
			return StateHigh, true
		}
		return StateHigh, false
	}
	return StateNormal, false
}

func ptr(f float64) *float64 { return &f }

// rangeTextOf writes a general range for display («12–15.5», «≥ 20», «< 5.7»).
func rangeTextOf(low, high *float64, highExclusive bool) string {
	switch {
	case low != nil && high != nil && !highExclusive:
		return fmtNum(low) + "–" + fmtNum(high)
	case high != nil && highExclusive:
		return "< " + fmtNum(high)
	case low != nil:
		return "≥ " + fmtNum(low)
	case high != nil:
		return "≤ " + fmtNum(high)
	}
	return ""
}

// generalRange is the «محدوده عمومی» of a catalogued marker on f.Date (the catalog's typical range unless the
// registry has one), with its basis ("" = everyday adult).
func generalRange(mk *Marker, f sheetFacts) (Range, string) {
	r := mk.TypicalRange()
	d := f.Date
	reg := func(key, field string) *float64 { return ptr(clinicalconfig.Num(key, field, d)) }
	basis := ""
	switch mk.Code {
	case codeHemoglobin:
		r.Low = reg(clinicalconfig.KeyLabHb, "nonpregnant_low")
		if f.Pregnant {
			r.Low, basis = reg(clinicalconfig.KeyLabHb, "pregnant_low"), BasisPregnancy
			if f.Trimester == 2 {
				r.Low, basis = reg(clinicalconfig.KeyLabHb, "pregnant_t2_low"), BasisPregnancyT2
			}
		}
	case codeFerritin:
		r.Low = reg(clinicalconfig.KeyFerritinLow, "ngml")
		if f.Inflamed {
			r.Low, basis = reg(clinicalconfig.KeyLabFerritinInflame, "low"), BasisInflamed
		}
	case codeVitaminD:
		r.Low = reg(clinicalconfig.KeyLabVitD, "sufficient_from")
	case codeTSH:
		if f.Pregnant {
			r.High, basis = reg(clinicalconfig.KeyLabTSHPregnancy, "upper"), BasisPregnancy
		}
	case codeHbA1c:
		r.High = reg(clinicalconfig.KeyLabHbA1c, "prediabetes_from")
		r.Text = rangeTextOf(nil, r.High, true)
		r.Source = RangeTypical
		return r, basis
	default:
		return r, basis
	}
	r.Source = RangeTypical
	if t := mk.TypicalRange(); !sameBound(t.Low, r.Low) || !sameBound(t.High, r.High) {
		r.Text = rangeTextOf(r.Low, r.High, false) // the catalog's printed text no longer matches the bounds
	}
	return r, basis
}

func sameBound(a, b *float64) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }

// band is the registry band of a banded marker (vitamin D, HbA1c) and the state it implies; ok=false otherwise.
func band(code string, v float64, r Range, d civildate.Date) (string, string, bool) {
	switch code {
	case codeVitaminD:
		state := StateNormal
		if r.High != nil && v > *r.High {
			state = StateHigh
		}
		switch {
		case v < clinicalconfig.Num(clinicalconfig.KeyLabVitD, "deficient_below", d):
			return BandVitDDeficient, StateLow, true
		case v < clinicalconfig.Num(clinicalconfig.KeyLabVitD, "sufficient_from", d):
			return BandVitDInsufficient, StateLow, true
		}
		return BandVitDSufficient, state, true
	case codeHbA1c:
		switch {
		case v >= clinicalconfig.Num(clinicalconfig.KeyLabHbA1c, "diabetes_from", d):
			return BandHbA1cHigh, StateHigh, true
		case v >= clinicalconfig.Num(clinicalconfig.KeyLabHbA1c, "prediabetes_from", d):
			return BandHbA1cRaised, StateHigh, true
		case r.Low != nil && v < *r.Low:
			return BandHbA1cTypical, StateLow, true
		}
		return BandHbA1cTypical, StateNormal, true
	}
	return "", "", false
}

// isCRP reports whether a printed marker name is C-reactive protein (CRP, hs-CRP, «سی‌آرپی»).
func isCRP(name string) bool {
	n := normalizeName(name)
	return n == "crp" || n == "hscrp" || strings.HasPrefix(n, "crp") || strings.Contains(n, "creactiveprotein") ||
		strings.Contains(n, "سیآرپی")
}
