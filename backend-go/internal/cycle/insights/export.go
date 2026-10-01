package insights

import (
	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Exported building blocks for the analysis tab (internal/analysis, B-N3-07), so the medians, the
// plausibility bounds and the confirmed-start selection stay one implementation.

// Median is the median of values (nil for none); an even count averages the middle pair with PHP round().
func Median(values []int) *int { return median(values) }

// ValidCycleLength reports whether n is a plausible cycle length (21–45 days, the metrics' bounds).
func ValidCycleLength(n int) bool { return validLength(n) }

// ValidPeriodLength reports whether n is a plausible bleed length (2–10 days, the metrics' bounds).
func ValidPeriodLength(n int) bool { return n >= validPeriodMin && n <= validPeriodMax }

// ConfirmedStarts are the user-confirmed, non-estimated periods that started on or before today, oldest first.
func ConfirmedStarts(histories []model.History, today civildate.Date) []model.History {
	return confirmedStarts(histories, today)
}

// ClosedPeriodDays is end−start+1 of a closed period (0 when open or inconsistent).
func ClosedPeriodDays(h model.History) int { return closedPeriodDays(h) }

// LutealDays is the fixed luteal length the pattern alignment and the engine use.
const LutealDays = lutealDays
