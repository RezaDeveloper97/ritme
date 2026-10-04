package legacy

import (
	"github.com/ritme/backend-go/internal/cycle/recommendation"
	"github.com/ritme/backend-go/internal/enums"
)

// Text flag keys and the tip type that carry fertility / conception copy.
const (
	flagFertilityStatus    = "fertility_status"
	flagProbabilityMessage = "probability_message"
	flagPhaseInfo          = "phase_info"
	tipTypeFertility       = "fertility"
)

// NeutralPhase is the phase a life mode without fertility content (teen, menopause — enums.LifeMode
// AllowsFertilityContent) reads: the ovulation phase becomes follicular up to the estimated ovulation day
// and luteal from it, the same split the web calendar uses for its phase line. Other phases are unchanged.
func NeutralPhase(phase enums.CyclePhase, day, ovulationDay int) enums.CyclePhase {
	if phase != enums.CyclePhaseOvulation {
		return phase
	}
	if day < ovulationDay {
		return enums.CyclePhaseFollicular
	}
	return enums.CyclePhaseLuteal
}

// WithoutFertilityCopy (CB-TEEN-04b, extends B-N2-11b's NoFertilityCopy) drops the fertility and
// pregnancy-chance copy from a calculation for a mode without fertility content: the `fertility_status`
// and `probability_message` text flags and the `fertility` daily tips go, and `phase_info` names the
// neutral phase without the (fertility-worded) sub-phase. The numeric fields stay as they are.
func (c Calculation) WithoutFertilityCopy() Calculation {
	if c.TextFlags != nil {
		flags := make([]TextFlag, 0, len(c.TextFlags))
		for _, f := range c.TextFlags {
			switch f.Key {
			case flagFertilityStatus, flagProbabilityMessage:
				continue
			case flagPhaseInfo:
				p := NeutralPhase(c.Phase, c.Day, c.OvulationDay)
				f.EN = "Current phase: " + p.Label("en")
				f.FA = "فاز فعلی: " + p.Label("fa")
			}
			flags = append(flags, f)
		}
		c.TextFlags = flags
	}
	if c.DailyTips != nil {
		tips := make([]recommendation.Tip, 0, len(c.DailyTips))
		for _, t := range c.DailyTips {
			if t.Type != tipTypeFertility {
				tips = append(tips, t)
			}
		}
		c.DailyTips = tips
	}
	return c
}
