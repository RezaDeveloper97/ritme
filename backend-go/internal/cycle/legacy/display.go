package legacy

import "github.com/ritme/backend-go/internal/enums"

// DisplayWindow reads a legacy day by the v1.1 display fertile window (task.md §19:
// max(O − 5, period end + 1) … O).
//
// The legacy engine runs the biological window O − 5 … O + 1 and calls both O and O + 1 phase
// `ovulation`, so on O + 1 — sub-phase post_ovulation, fertility level low everywhere else (cycle
// view, calendar, /fertility/*) — Laravel still says «ovulation» and is_fertile_window = true,
// handing an avoiding user "peak fertility" copy. Bleeding days are already cleared by the legacy
// engine (menstruation wins), so dropping the days after ovulation is the whole difference: after
// the ovulation day the phase `ovulation` becomes `luteal` and the day is not fertile. The
// sub-phase is not touched.
//
// Shared by /messages/daily (D-28, messages/manager) and the /cycle/today|date calculation
// (D-30, Engine.CalculateForDisplay) so both read the same window.
func DisplayWindow(phase enums.CyclePhase, fertile bool, day, ovulation int) (enums.CyclePhase, bool) {
	if day <= ovulation {
		return phase, fertile
	}
	if phase == enums.CyclePhaseOvulation {
		phase = enums.CyclePhaseLuteal
	}
	return phase, false
}
