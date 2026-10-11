// Package challenges is Challenges v2 (bloom B-N9-07, D-87): runs of several days with per-day tasks, eligibility
// rules by life mode and cycle phase, and a reward (badge + free Plus days) granted exactly once on completion.
// The legacy one-day daily challenge (internal/home) is untouched; a challenge is v2 when it has a program row.
package challenges

import (
	"encoding/json"
	"slices"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/challenges/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Start rules: when a user may join.
const (
	StartAnytime   = "anytime"    // any day while the challenge is active
	StartMenstrual = "menstrual"  // only while the main phase is menstrual
	StartFixedDate = "fixed_date" // from starts_on on
)

// StartRules are the valid start_rule values.
var StartRules = []string{StartAnytime, StartMenstrual, StartFixedDate}

// Enrollment statuses.
const (
	StatusActive    = "active"
	StatusCompleted = "completed"
)

// Eligibility is the include / exclude rule set of a challenge. An empty include list allows everything; an exclude
// match always refuses. Modes are life modes (cycle, ttc, pregnancy, postpartum, menopause, teen), phases are cycle
// main phases (menstrual, follicular, fertile, luteal, period_expected, unknown).
type Eligibility struct {
	ModesInclude  []string `json:"modes_include"`
	ModesExclude  []string `json:"modes_exclude"`
	PhasesInclude []string `json:"phases_include"`
	PhasesExclude []string `json:"phases_exclude"`
}

// ParseEligibility reads the stored JSON (NULL / invalid = no rule).
func ParseEligibility(raw db.NullRawJSON) Eligibility {
	var e Eligibility
	if raw.Valid {
		_ = json.Unmarshal(raw.V, &e)
	}
	return e
}

// NeedsPhase reports whether any phase rule exists (the cycle engine runs only then).
func (e Eligibility) NeedsPhase() bool { return len(e.PhasesInclude) > 0 || len(e.PhasesExclude) > 0 }

// Allows applies the rules to a user's life mode and main phase ("" = not available, e.g. no cycle data).
func (e Eligibility) Allows(mode, phase string) bool {
	if slices.Contains(e.ModesExclude, mode) {
		return false
	}
	if len(e.ModesInclude) > 0 && !slices.Contains(e.ModesInclude, mode) {
		return false
	}
	if phase != "" && slices.Contains(e.PhasesExclude, phase) {
		return false
	}
	if len(e.PhasesInclude) > 0 && !slices.Contains(e.PhasesInclude, phase) {
		return false
	}
	return true
}

// Program is a v2 challenge with its settings.
type Program struct {
	ID                uint64
	Title             json.RawMessage
	Description       db.NullRawJSON
	Category          string
	DurationDays      int
	StartRule         string
	StartsOn          civildate.NullDate
	ReminderTime      string
	Eligibility       Eligibility
	RewardBadgeCode   string
	RewardBadgeTitle  db.NullRawJSON
	RewardPlusDays    int
	CompletionMinDays int // 0 = every day
}

func programOf(r store.ListActiveProgramsRow) Program {
	return Program{
		ID: r.ID, Title: r.Title, Description: r.Description, Category: r.Category.String,
		DurationDays: int(r.DurationDays), StartRule: r.StartRule, StartsOn: r.StartsOn, ReminderTime: r.ReminderTime.String,
		Eligibility: ParseEligibility(r.Eligibility), RewardBadgeCode: r.RewardBadgeCode.String,
		RewardBadgeTitle: r.RewardBadgeTitle, RewardPlusDays: int(r.RewardPlusDays), CompletionMinDays: int(r.CompletionMinDays.Int16),
	}
}

// DayTasks are the tasks that apply to a day (1-based) of the run.
func DayTasks(tasks []store.ChallengeTask, day int) []store.ChallengeTask {
	var out []store.ChallengeTask
	for _, t := range tasks {
		if int(t.DayFrom) <= day && day <= int(t.DayTo) {
			out = append(out, t)
		}
	}
	return out
}

// Progress counts the finished days of a run (every task of the day done) and the days a completion needs: the
// program's minimum, or every day with tasks, never more than the days that have tasks. done is the set of
// "taskID:day" completions.
func Progress(p Program, tasks []store.ChallengeTask, done map[[2]uint64]bool) (doneDays, required int) {
	withTasks := 0
	for d := 1; d <= p.DurationDays; d++ {
		dt := DayTasks(tasks, d)
		if len(dt) == 0 {
			continue
		}
		withTasks++
		all := true
		for _, t := range dt {
			if !done[[2]uint64{t.ID, uint64(d)}] { //nolint:gosec // G115: d ≥ 1
				all = false
				break
			}
		}
		if all {
			doneDays++
		}
	}
	required = p.CompletionMinDays
	if required <= 0 || required > withTasks {
		required = withTasks
	}
	return doneDays, required
}
