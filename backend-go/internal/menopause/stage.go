package menopause

import (
	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// MenopauseMonths is the stage rule's threshold: 12 months or more without a period ⇒ menopause
// (docs/canvas-build/menopause.md §2) [needs clinical review].
const MenopauseMonths = 12

// Stage codes (enums.MenopauseStage, the Onb_Meno / Meno_Stage answers).
const (
	StagePeri   = string(enums.MenopauseStagePeri)
	StageMeno   = string(enums.MenopauseStageMeno)
	StagePost   = string(enums.MenopauseStagePost)
	StageUnsure = string(enums.MenopauseStageUnsure)
)

// Answers are the stored menopause columns of user_life_profiles ("" / nil = not answered).
type Answers struct {
	Stage      string
	LastPeriod *civildate.Date // approximate last period (first day of that month)
	Surgical   *bool
	HRT        *bool
}

// Stage is the stage rule applied to the answers on a day. It is computed, never stored.
type Stage struct {
	Answers
	// Stage is the effective stage: peri | meno | post, "" when it cannot be told (no answer, no last period).
	Stage string
	// MonthsWithoutPeriod are the whole months from the last period to the day (nil without a last period).
	MonthsWithoutPeriod *int
	// Suggested is "meno" when the stored answer is peri but the last period is 12+ months ago (the API suggests,
	// it never changes her answer); "" otherwise.
	Suggested string
}

// PostMenopausal reports whether any bleeding or spotting now raises the postmenopausal_bleeding alert.
func (s Stage) PostMenopausal() bool { return s.Stage == StageMeno || s.Stage == StagePost }

// NeedsStage: neither a usable answer nor a last period — the home sends her to the stage screen.
func (s Stage) NeedsStage() bool { return s.Stage == "" }

// WholeMonths are the whole calendar months from `from` to `to` (0 when to is before from).
func WholeMonths(from, to civildate.Date) int {
	if to.Before(from) {
		return 0
	}
	n := (to.Year-from.Year)*12 + int(to.Month) - int(from.Month)
	if engine.AddMonths(from, n).After(to) {
		n--
	}
	return max(n, 0)
}

// ResolveStage applies the stage rule (docs/canvas-build/menopause.md §2):
//   - peri → peri (meno suggested at 12+ months), meno → meno, post → post;
//   - unsure / no answer → meno at 12+ months without a period, else peri; no last period → "" (ask);
//   - surgical menopause counts as menopause regardless of the months (post stays post).
func ResolveStage(a Answers, today civildate.Date) Stage {
	st := Stage{Answers: a}
	if a.LastPeriod != nil && !a.LastPeriod.After(today) {
		m := WholeMonths(*a.LastPeriod, today)
		st.MonthsWithoutPeriod = &m
	}
	reached := st.MonthsWithoutPeriod != nil && *st.MonthsWithoutPeriod >= MenopauseMonths
	surgical := a.Surgical != nil && *a.Surgical
	switch a.Stage {
	case StageMeno, StagePost:
		st.Stage = a.Stage
	case StagePeri:
		st.Stage = StagePeri
		if reached && !surgical {
			st.Suggested = StageMeno
		}
	default: // unsure or not answered
		switch {
		case reached:
			st.Stage = StageMeno
		case st.MonthsWithoutPeriod != nil:
			st.Stage = StagePeri
		}
	}
	if surgical && st.Stage != StagePost {
		st.Stage = StageMeno
	}
	return st
}
