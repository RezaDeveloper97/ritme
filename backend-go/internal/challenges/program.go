// Package challenges is Challenges v2 (bloom B-N9-07, D-87): runs of several days with per-day tasks, eligibility and
// contraindication rules over the message fact registry (T-M10-10, D-167), a start rule, and a reward (badge + free
// Plus days) granted exactly once on completion.
// The legacy one-day daily challenge (internal/home) is untouched; a challenge is v2 when it has a program row.
package challenges

import (
	"encoding/json"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/challenges/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Start rules: when a user may join (SPEC §5.5, D-167).
const (
	StartOnJoin       = "on_join"        // any day while the challenge is active
	StartOnPhaseStart = "on_phase_start" // only on the first day of start_phase (the engine's main phase)
	StartFixedDate    = "fixed_date"     // from starts_on on

	// Legacy values (B-N9-07), still read: anytime = on_join; menstrual = any day while the main phase is menstrual.
	StartAnytime   = "anytime"
	StartMenstrual = "menstrual"
)

// StartRules are the start_rule values the admin API writes.
var StartRules = []string{StartOnJoin, StartOnPhaseStart, StartFixedDate}

// StartPhases are the main phases an on_phase_start challenge may start on (every v6 main phase but unknown).
var StartPhases = []string{"menstrual", "follicular", "fertile", "luteal", "period_expected"}

// Enrollment statuses.
const (
	StatusActive    = "active"
	StatusCompleted = "completed"
)

// NeedsPhase reports whether the start rule reads the main phase.
func (p Program) NeedsPhase() bool {
	return p.StartRule == StartMenstrual || p.StartRule == StartOnPhaseStart
}

// Facts are the fact names the program's rules read.
func (p Program) Facts() []string {
	return append(p.Eligibility.Facts(), p.Contraindications.Facts()...)
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
	Eligibility       Rule
	Contraindications Rule
	StartPhase        string // on_phase_start only
	RewardBadgeCode   string
	RewardBadgeTitle  db.NullRawJSON
	RewardPlusDays    int
	CompletionMinDays int // 0 = every day
}

func programOf(r store.ListActiveProgramsRow) Program {
	return Program{
		ID: r.ID, Title: r.Title, Description: r.Description, Category: r.Category.String,
		DurationDays: int(r.DurationDays), StartRule: r.StartRule, StartsOn: r.StartsOn, ReminderTime: r.ReminderTime.String,
		Eligibility: ParseRule(r.Eligibility), Contraindications: ParseRule(r.Contraindications), StartPhase: r.StartPhase.String,
		RewardBadgeCode:  r.RewardBadgeCode.String,
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

// DayRetention is one day of a challenge's drop-off curve (SPEC §5.5 «ریزش روز به روز», T-M10-10).
type DayRetention struct {
	Day int
	// Reached are the runs whose day Day has arrived (started_on + Day − 1 ≤ today): the denominator of the day.
	Reached int
	// Retained are the reached runs still going on Day: a completion on Day or later, or a completed run.
	Retained int
	// Dropped are the reached runs retained on Day − 1 (every run on day 0) but no longer on Day.
	Dropped int
	// RetainedPct / DroppedPct are Retained / Dropped over Reached, rounded half up (-1 = nothing reached).
	RetainedPct, DroppedPct int
}

// DropOff is the day-by-day retention of a challenge's runs over its duration (one row per day 1…duration).
func DropOff(runs []store.ProgramRunsRow, duration int, today civildate.Date) []DayRetention {
	out := make([]DayRetention, 0, max(duration, 0))
	retainedOn := func(r store.ProgramRunsRow, day int) bool {
		return day <= 0 || r.Status == StatusCompleted || int(r.LastDay) >= day
	}
	pct := func(n, d int) int {
		if d == 0 {
			return -1
		}
		return (n*100 + d/2) / d
	}
	for day := 1; day <= duration; day++ {
		d := DayRetention{Day: day}
		for _, r := range runs {
			if r.StartedOn.AddDays(day - 1).After(today) {
				continue
			}
			d.Reached++
			switch {
			case retainedOn(r, day):
				d.Retained++
			case retainedOn(r, day-1):
				d.Dropped++
			}
		}
		d.RetainedPct, d.DroppedPct = pct(d.Retained, d.Reached), pct(d.Dropped, d.Reached)
		out = append(out, d)
	}
	return out
}
