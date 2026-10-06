package vitals

import (
	"sort"
	"strconv"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/vitals/store"
)

// Day-level vitals of the log sheet (taxonomy v2 measurements.bp_systolic / bp_diastolic / heart_rate / blood_sugar,
// one value per day, no time) are read merged into the vitals views instead of being migrated (D-63):
//
//   - a day that has a timed vital of a type hides that day's log value of the same type (the user re-entered it);
//   - BP needs both numbers; heart rate counts as «resting» only in the sense that it is classified with the resting
//     range (context ""); glucose has no context on the sheet → the random band;
//   - log readings have no time: they never count in morning-vs-night or towards the weekly plan, and they are not
//     editable through /vitals (id null; the log sheet owns them).

// LogReadings builds the log readings of rows, skipping the (day, type) pairs in timed.
func LogReadings(rows []store.ListLogMeasurementsRow, timed map[dayType]bool) []Reading {
	type day struct{ sys, dia, hr, glu *float64 }
	days := map[civildate.Date]*day{}
	for _, r := range rows {
		if !r.ValueNum.Valid {
			continue
		}
		v, err := strconv.ParseFloat(r.ValueNum.String, 64)
		if err != nil {
			continue
		}
		d := days[r.LogDate]
		if d == nil {
			d = &day{}
			days[r.LogDate] = d
		}
		switch r.Param {
		case "bp_systolic":
			d.sys = &v
		case "bp_diastolic":
			d.dia = &v
		case "heart_rate":
			d.hr = &v
		case "blood_sugar":
			d.glu = &v
		}
	}
	var out []Reading
	for date, d := range days {
		if d.sys != nil && d.dia != nil && !timed[dayType{date, TypeBP}] {
			out = append(out, Reading{Source: SourceLog, Type: TypeBP, Date: date, Systolic: *d.sys, Diastolic: *d.dia})
		}
		if d.hr != nil && !timed[dayType{date, TypeHR}] {
			out = append(out, Reading{Source: SourceLog, Type: TypeHR, Date: date, Pulse: *d.hr})
		}
		if d.glu != nil && !timed[dayType{date, TypeGlucose}] {
			out = append(out, Reading{Source: SourceLog, Type: TypeGlucose, Date: date, MgDl: *d.glu, Unit: UnitMgDl,
				Context: ContextRandom})
		}
	}
	return out
}

type dayType struct {
	date civildate.Date
	typ  string
}

// timedDays is the set of (day, type) that have a timed reading.
func timedDays(rs []Reading) map[dayType]bool {
	m := map[dayType]bool{}
	for _, r := range rs {
		m[dayType{r.Date, r.Type}] = true
	}
	return m
}

// Merge returns timed + log readings newest first (a log reading sorts after the timed readings of its day).
func Merge(timed []Reading, rows []store.ListLogMeasurementsRow) []Reading {
	out := append(append([]Reading{}, timed...), LogReadings(rows, timedDays(timed))...)
	SortNewestFirst(out)
	return out
}

// SortNewestFirst orders readings by day, then time (untimed last within a day), then id, newest first.
func SortNewestFirst(rs []Reading) {
	sort.SliceStable(rs, func(i, j int) bool {
		a, b := rs[i], rs[j]
		if c := a.Date.Compare(b.Date); c != 0 {
			return c > 0
		}
		if a.Timed() != b.Timed() {
			return a.Timed()
		}
		if !a.MeasuredAt.Equal(b.MeasuredAt) {
			return a.MeasuredAt.After(b.MeasuredAt)
		}
		if a.ID != b.ID {
			return a.ID > b.ID
		}
		return typeOrder(a.Type) < typeOrder(b.Type)
	})
}

func typeOrder(t string) int {
	for i, x := range Types {
		if x == t {
			return i
		}
	}
	return len(Types)
}

// OfType filters readings by type ("" = all).
func OfType(rs []Reading, typ string) []Reading {
	if typ == "" {
		return rs
	}
	var out []Reading
	for _, r := range rs {
		if r.Type == typ {
			out = append(out, r)
		}
	}
	return out
}
