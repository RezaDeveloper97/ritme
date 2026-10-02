package ivf

import (
	"database/sql"
	"time"

	"github.com/ritme/backend-go/internal/ivf/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Cycle is one treatment cycle. Zero dates/times are unset.
type Cycle struct {
	ID              uint64
	Number          int
	Protocol        string
	Stage           string
	StartedOn       civildate.Date
	StimStartedOn   civildate.Date
	RetrievalAt     time.Time
	TransferAt      time.Time
	BetaOn          civildate.Date
	NextScanAt      time.Time
	NotifyCompanion bool
	Outcome         string
	OutcomeOn       civildate.Date
	Open            bool
}

func nullDate(d civildate.NullDate) civildate.Date {
	if !d.Valid {
		return civildate.Date{}
	}
	return d.Date
}

func nullTime(t sql.NullTime) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time.In(civildate.Tehran)
}

func cycleFromRow(r store.IvfCycle) Cycle {
	return Cycle{
		ID: r.ID, Number: int(r.Number), Protocol: r.Protocol.String, Stage: r.Stage, StartedOn: r.StartedOn,
		StimStartedOn: nullDate(r.StimStartedOn), RetrievalAt: nullTime(r.RetrievalAt), TransferAt: nullTime(r.TransferAt),
		BetaOn: nullDate(r.BetaOn), NextScanAt: nullTime(r.NextScanAt), NotifyCompanion: r.NotifyCompanion,
		Outcome: r.Outcome.String, OutcomeOn: nullDate(r.OutcomeOn), Open: r.ActiveUserID.Valid,
	}
}

// dayOf is t's Tehran calendar day (zero for a zero time).
func dayOf(t time.Time) civildate.Date {
	if t.IsZero() {
		return civildate.Date{}
	}
	return civildate.InTehran(t)
}

// TransferOn is the transfer day (zero when not set).
func (c Cycle) TransferOn() civildate.Date { return dayOf(c.TransferAt) }

// StageStart is the first day of stage in this cycle: the start (prep), stimulation start, retrieval day, transfer
// day (transfer and the two-week wait) or the beta day (test). Zero when not known.
func (c Cycle) StageStart(stage string) civildate.Date {
	switch stage {
	case StagePrep:
		return c.StartedOn
	case StageStim:
		return c.StimStartedOn
	case StageRetrieval:
		return dayOf(c.RetrievalAt)
	case StageTransfer, StageTWW:
		return c.TransferOn()
	case StageTest:
		return c.BetaOn
	}
	return civildate.Date{}
}

// StageDay is the day number within the current stage on today («روز ۷ تحریک»): today − stage start + 1, when the
// start is known and not after today.
func (c Cycle) StageDay(today civildate.Date) (int, bool) {
	start := c.StageStart(c.Stage)
	if start.IsZero() || start.After(today) {
		return 0, false
	}
	return start.DiffDays(today) + 1, true
}

// StimDay is the stimulation day number of d («روز ۷ تحریک» on a scan), when stimulation started on or before d.
func (c Cycle) StimDay(d civildate.Date) (int, bool) {
	if c.StimStartedOn.IsZero() || c.StimStartedOn.After(d) {
		return 0, false
	}
	return c.StimStartedOn.DiffDays(d) + 1, true
}

// DaysToBeta is the number of days from today to the beta test (negative once past), when beta_on is set.
func (c Cycle) DaysToBeta(today civildate.Date) (int, bool) {
	if c.BetaOn.IsZero() {
		return 0, false
	}
	return today.DiffDays(c.BetaOn), true
}

// DaysSinceTransfer is today − transfer day, when the transfer is set and not after today.
func (c Cycle) DaysSinceTransfer(today civildate.Date) (int, bool) {
	t := c.TransferOn()
	if t.IsZero() || t.After(today) {
		return 0, false
	}
	return t.DiffDays(today), true
}

// Step is one row of the StepTimeline.
type Step struct {
	Stage  string
	Status string // done | current | todo
	Date   civildate.Date
}

// Timeline statuses.
const (
	StepDone    = "done"
	StepCurrent = "current"
	StepTodo    = "todo"
)

// Timeline is the six stages with their status relative to the current stage (a closed cycle has every stage up
// to its last one done) and their start date when known.
func (c Cycle) Timeline() []Step {
	cur := stageIndex(c.Stage)
	out := make([]Step, len(Stages))
	for i, s := range Stages {
		status := StepTodo
		switch {
		case i < cur, i == cur && !c.Open:
			status = StepDone
		case i == cur:
			status = StepCurrent
		}
		out[i] = Step{Stage: s, Status: status, Date: c.StageStart(s)}
	}
	return out
}
