package pregnancy

import (
	"database/sql"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// Saving a scan (T-M10-18, SPEC v6 §3.1, D-169): with an LMP on the profile, the scan replaces the LMP dating only when
// the discrepancy is more than the band's limit (calc.Redate); without one the scan dates the pregnancy. A basis the
// user chose (an explicit age_source in the request) stays — the rule only decides when nobody chose.

// ApplyScanRule decides the age source of a scan in attrs (the attributes about to be written onto existing; existing
// may be nil). It returns nil when attrs carry no scan. When explicit is false it writes the age source the rule picks
// with its confidence and uncertainty into attrs; when explicit is true attrs keep the user's age_source and the
// result only reports the comparison.
func ApplyScanRule(existing *store.PregnancyProfile, attrs *jsonx.OrderedMap, explicit bool) *calc.Redating {
	scanDate, gaAtScan, ok := scanOf(existing, attrs)
	if !ok {
		return nil
	}
	lmp, hasLMP := lmpOf(existing, attrs)
	r := calc.Redate(lmp, hasLMP, scanDate, gaAtScan)
	if explicit {
		if src, ok := attrs.Get("age_source"); ok && src != nil {
			r.Basis = phpval.ToString(src)
		}
		return &r
	}
	conf, unc, _ := confidenceFor(r.Basis)
	attrs.Set("age_source", r.Basis).
		Set("confidence_level", string(conf)).
		Set("uncertainty_days", int64(unc))
	return &r
}

// scanOf is the scan the profile will hold: attrs' ultrasound fields over existing's. ok is false when attrs carry no
// ultrasound field or the merged scan is incomplete.
func scanOf(p *store.PregnancyProfile, attrs *jsonx.OrderedMap) (civildate.Date, int, bool) {
	_, hasDate := attrs.Get("ultrasound_date")
	_, hasWeeks := attrs.Get("ultrasound_weeks")
	_, hasDays := attrs.Get("ultrasound_days")
	if !hasDate && !hasWeeks && !hasDays {
		return civildate.Date{}, 0, false
	}
	var date civildate.NullDate
	var weeks, days sql.NullInt32
	if p != nil {
		date, weeks, days = p.UltrasoundDate, p.UltrasoundWeeks, p.UltrasoundDays
	}
	if v, ok := attrs.Get("ultrasound_date"); ok {
		d, valid := toDate(v)
		date = civildate.NullDate{Date: d, Valid: valid && v != nil}
	}
	if v, ok := attrs.Get("ultrasound_weeks"); ok {
		weeks = wInt(v)
	}
	if v, ok := attrs.Get("ultrasound_days"); ok {
		days = wInt(v)
	}
	if !date.Valid || !weeks.Valid || !days.Valid {
		return civildate.Date{}, 0, false
	}
	return date.Date, int(weeks.Int32)*7 + int(days.Int32), true
}

// lmpOf is the LMP the profile will hold: attrs' lmp_date (null clears it) over existing's.
func lmpOf(p *store.PregnancyProfile, attrs *jsonx.OrderedMap) (civildate.Date, bool) {
	if v, ok := attrs.Get("lmp_date"); ok {
		if v == nil {
			return civildate.Date{}, false
		}
		return toDate(v)
	}
	if p == nil || !p.LmpDate.Valid {
		return civildate.Date{}, false
	}
	return p.LmpDate.Date, true
}
