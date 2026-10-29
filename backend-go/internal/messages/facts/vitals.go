package facts

import (
	"context"
	"slices"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/vitals"
)

// Vitals facts: the class of the newest blood pressure and glucose reading of the last VitalsWindowDays days (timed
// readings and log-sheet values merged, classified as the vitals screens classify them: the pregnancy table in
// pregnancy, the diabetic targets for a diabetic user). Absent without a reading in the window.

// VitalsWindowDays is the look-back of the last-reading facts (today included).
const VitalsWindowDays = 30

// LastVitals are the classes of the newest readings ("" = no reading in the window, or unclassified).
type LastVitals struct {
	BP      string
	Glucose string
}

// VitalsSource reads the newest classified readings of [from, to].
type VitalsSource interface {
	LastVitals(ctx context.Context, from, to civildate.Date) (LastVitals, error)
}

// GroupVitals is the vitals producer group.
const GroupVitals = "vitals"

func init() {
	g := GroupVitals
	Register(
		Def{Name: "bp_last_category", Type: TypeEnum, Group: g,
			Enum:        vitals.BPClasses,
			Description: desc("رده آخرین فشار خون در ۳۰ روز", "The class of the last blood pressure reading in 30 days.")},
		Def{Name: "glucose_last_category", Type: TypeEnum, Group: g,
			Enum:        append(append(slices.Clone(vitals.GlucoseClasses), vitals.ClassDiscuss), vitals.GlucoseTargetClasses...),
			Description: desc("رده آخرین قند خون در ۳۰ روز", "The class of the last glucose reading in 30 days.")},
	)
	RegisterGroup(g, func(ctx context.Context, f *Facts) (Values, error) {
		if f.src.Vitals == nil {
			return nil, nil
		}
		last, err := f.src.Vitals.LastVitals(ctx, f.today.AddDays(-(VitalsWindowDays - 1)), f.today)
		if err != nil {
			return nil, err
		}
		v := Values{}
		if last.BP != "" {
			v["bp_last_category"] = last.BP
		}
		if last.Glucose != "" {
			v["glucose_last_category"] = last.Glucose
		}
		return v, nil
	})
}
