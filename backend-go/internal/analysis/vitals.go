package analysis

import (
	"context"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/vitals"
)

// VitalsSource reads the merged vitals readings of a user (internal/vitals.Service.Merged: vital_readings + the log
// sheet's day values, a timed reading hiding that day's log value of the same type — D-63). bloom B-N6-03 follow-up
// of B-N6-01: the analysis hub / monthly vitals and the pregnancy BP / glucose sections read it instead of the log
// sheet alone.
type VitalsSource interface {
	Merged(ctx context.Context, userID uint64, typ string, from, to civildate.Date) ([]vitals.Reading, error)
}

// VitalReading is one merged reading as the analysis reads it.
type VitalReading struct {
	Date                civildate.Date
	Systolic, Diastolic float64 // BP (0 = not a BP reading)
	Glucose             float64 // mg/dL (0 = not a glucose reading)
	Context             string  // glucose context (vitals.Context*)
	Timed               bool    // a vital_readings row (false = a log-sheet day value)
}

// SummaryGlucose reports whether the reading counts in the hub's blood-sugar average: a fasting reading, or a log-sheet
// value (the sheet has no context; it is what the card averaged before the merge). Readings after a meal / at bedtime
// would skew a fasting-style average and stay in the vitals reports.
func (r VitalReading) SummaryGlucose() bool {
	return !r.Timed || r.Context == vitals.ContextFasting
}

// VitalReadingsOf maps merged readings (BP and glucose; heart rate is not read by the analysis).
func VitalReadingsOf(rs []vitals.Reading) []VitalReading {
	out := make([]VitalReading, 0, len(rs))
	for _, r := range rs {
		v := VitalReading{Date: r.Date, Context: r.Context, Timed: r.Timed()}
		switch r.Type {
		case vitals.TypeBP:
			v.Systolic, v.Diastolic = r.Systolic, r.Diastolic
		case vitals.TypeGlucose:
			v.Glucose = r.MgDl
		default:
			continue
		}
		out = append(out, v)
	}
	return out
}

// loadVitals fills in.VitalReadings for [from, to] when a source is wired.
func loadVitals(ctx context.Context, src VitalsSource, uid uint64, in *Input, from, to civildate.Date) error {
	if src == nil {
		return nil
	}
	rs, err := src.Merged(ctx, uid, "", from, to)
	if err != nil {
		return err
	}
	in.VitalReadings, in.VitalsLoaded = VitalReadingsOf(rs), true
	return nil
}

// mergeVitals puts the merged readings into the pregnancy input: per day the highest BP reading (a single high reading
// must not be averaged away in pregnancy), and the timed glucose readings by slot (fasting → fasting, after a meal →
// the 2-hour target, the vitals «after a meal» meaning 2 h) replacing a weekly-log value of the same day and slot.
// Log-sheet glucose values have no slot and stay out, as before.
func mergeVitals(in *PregnancyInput, rs []VitalReading) {
	timedGlucose := map[[2]string]bool{}
	bp := map[civildate.Date]BPReading{}
	var glucose []GlucoseReading
	for _, r := range rs {
		if r.Date.Before(in.Start.AddDays(-BaselineLookbackDays)) || r.Date.After(in.Today) {
			continue
		}
		if r.Systolic > 0 && r.Diastolic > 0 {
			cur, ok := bp[r.Date]
			if !ok || r.Systolic > cur.Systolic || (r.Systolic == cur.Systolic && r.Diastolic > cur.Diastolic) {
				bp[r.Date] = BPReading{Systolic: r.Systolic, Diastolic: r.Diastolic}
			}
		}
		if r.Glucose > 0 && r.Timed {
			slot := ""
			switch r.Context {
			case vitals.ContextFasting:
				slot = GlucoseFasting
			case vitals.ContextAfterMeal:
				slot = GlucoseTwoHour
			}
			if slot != "" {
				glucose = append(glucose, GlucoseReading{Date: r.Date, Slot: slot, Value: r.Glucose})
				timedGlucose[[2]string{r.Date.String(), slot}] = true
			}
		}
	}
	for d, r := range bp {
		in.BP[d] = r
	}
	kept := make([]GlucoseReading, 0, len(in.Glucose)+len(glucose))
	for _, g := range in.Glucose {
		if !timedGlucose[[2]string{g.Date.String(), g.Slot}] {
			kept = append(kept, g)
		}
	}
	kept = append(kept, glucose...)
	in.Glucose = kept
}
