package engine

import (
	"slices"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Registry-driven screening (T-M10-21, SPEC v6 §3.4, DECISIONS.md H-11, D-172). A catalog type whose key has rows in
// the clinical registry key screening.uspstf (cervical, mammography, BP, glucose, dental) takes its interval and age
// span from the rows valid on the evaluation day instead of the catalog columns:
//
//   - eligible rows: the BMI condition holds (min_bmi null, or the profile BMI is unknown, or BMI ≥ min_bmi; a boundary
//     lands in the higher band like bmi.who). No eligible row = the type does not apply (glucose at BMI 24.9).
//   - the age span is the union of the eligible bands (age_min = the lowest, age_max = the highest, open when any band
//     is open): younger = not_yet, older = left out, like the catalog columns.
//   - the band rows are the eligible rows whose ages contain today's age (every eligible row when the birthday is
//     unknown). A known age in no band (a gap between bands) leaves the type out.
//   - the interval is the band row whose last_test_kind is the latest record's test_kind, else the band row without
//     a kind, else the shortest band interval (no record, a record without a kind, or a kind the band has no row for:
//     Pap 21–29 «cytology» counts with no last record).
//
// Custom checkups and catalog keys without rows keep the catalog columns.

// screeningRow is one screening.uspstf row; ages and min_bmi 0 = null (open / no condition).
type screeningRow struct {
	ageMin, ageMax int
	months         int
	kind           string
	minBMI         float64
}

func screeningRows(key string, day civildate.Date) []screeningRow {
	if key == "" {
		return nil
	}
	var out []screeningRow
	for _, r := range clinicalconfig.Rows(clinicalconfig.KeyScreeningUSPSTF, day) {
		if r.Str("type_key") != key {
			continue
		}
		row := screeningRow{kind: r.Str("last_test_kind")}
		if v, ok := r.Num("age_min"); ok {
			row.ageMin = int(v)
		}
		if v, ok := r.Num("age_max"); ok {
			row.ageMax = int(v)
		}
		if v, ok := r.Num("interval_months"); ok {
			row.months = int(v)
		}
		if v, ok := r.Num("min_bmi"); ok {
			row.minBMI = v
		}
		out = append(out, row)
	}
	return out
}

func (r screeningRow) contains(age int) bool {
	return (r.ageMin == 0 || age >= r.ageMin) && (r.ageMax == 0 || age <= r.ageMax)
}

// Schedule is the effective schedule of a type for one user on one day.
type Schedule struct {
	// Type is the type with the effective interval and age span (the catalog row itself when not registry-driven).
	Type Type
	// Registry: the interval and ages come from screening.uspstf.
	Registry bool
	// Applies is false when the registry leaves the type out (BMI condition, past the last band, a band gap).
	Applies bool
	// TestKinds are the distinct non-null last_test_kind values of the bands that contain the age (all bands when
	// the age is unknown): the test kinds a record of this type may carry.
	TestKinds []string
	ageKnown  bool
}

// TestKindRequired: the user's band (her age known) offers more than one test kind, so the kind changes the
// interval (Pap from 30).
func (s Schedule) TestKindRequired() bool { return s.ageKnown && len(s.TestKinds) > 1 }

// ScheduleFor resolves t for a user of age (nil = unknown) with the profile bmi (0 = unknown) whose latest record of
// t has testKind ("" = none), on day.
func ScheduleFor(t Type, age *int, bmi float64, testKind string, day civildate.Date) Schedule {
	s := Schedule{Type: t, Applies: true, ageKnown: age != nil}
	if t.UserID != 0 {
		return s
	}
	rows := screeningRows(t.Key, day)
	if len(rows) == 0 {
		return s
	}
	s.Registry = true
	s.Type.IntervalMonthsMax = 0

	var eligible []screeningRow
	for _, r := range rows {
		if r.minBMI == 0 || bmi <= 0 || bmi >= r.minBMI {
			eligible = append(eligible, r)
		}
	}
	if len(eligible) == 0 {
		s.Applies = false
		s.Type.IntervalMonths = shortest(rows)
		return s
	}
	lo, hi := eligible[0].ageMin, eligible[0].ageMax
	for _, r := range eligible[1:] {
		lo = min(lo, r.ageMin)
		if hi != 0 && (r.ageMax == 0 || r.ageMax > hi) {
			hi = r.ageMax
		}
	}
	s.Type.AgeMin, s.Type.AgeMax = lo, hi

	band := eligible
	if age != nil {
		band = nil
		for _, r := range eligible {
			if r.contains(*age) {
				band = append(band, r)
			}
		}
		if len(band) == 0 {
			// Younger than the first band: not_yet on the first band's interval. Older, or in a gap: left out.
			s.Applies = lo > 0 && *age < lo
			s.Type.IntervalMonths = shortest(eligible)
			return s
		}
	}
	for _, r := range band {
		if r.kind != "" && !slices.Contains(s.TestKinds, r.kind) {
			s.TestKinds = append(s.TestKinds, r.kind)
		}
	}
	s.Type.IntervalMonths = pickInterval(band, testKind)
	return s
}

func pickInterval(band []screeningRow, kind string) int {
	if kind != "" {
		for _, r := range band {
			if r.kind == kind {
				return r.months
			}
		}
	}
	for _, r := range band {
		if r.kind == "" {
			return r.months
		}
	}
	return shortest(band)
}

func shortest(rows []screeningRow) int {
	m := 0
	for _, r := range rows {
		if m == 0 || (r.months > 0 && r.months < m) {
			m = r.months
		}
	}
	return m
}

// RegistryDriven reports whether a catalog key takes its interval and ages from screening.uspstf on day.
func RegistryDriven(key string, day civildate.Date) bool { return len(screeningRows(key, day)) > 0 }

// TestKindsOf are the distinct test kinds any screening.uspstf row of key names on day (empty for a type whose
// records carry no test kind): a record of the type may carry one of them.
func TestKindsOf(key string, day civildate.Date) []string {
	var out []string
	for _, r := range screeningRows(key, day) {
		if r.kind != "" && !slices.Contains(out, r.kind) {
			out = append(out, r.kind)
		}
	}
	return out
}
