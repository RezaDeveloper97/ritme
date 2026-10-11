package challenges

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/challenges/store"
)

func TestEligibility_Allows(t *testing.T) {
	for _, tc := range []struct {
		name        string
		e           Eligibility
		mode, phase string
		want        bool
	}{
		{"no rules", Eligibility{}, "cycle", "", true},
		{"mode include hit", Eligibility{ModesInclude: []string{"cycle", "ttc"}}, "ttc", "", true},
		{"mode include miss", Eligibility{ModesInclude: []string{"cycle", "ttc"}}, "pregnancy", "", false},
		{"mode exclude wins over include", Eligibility{ModesInclude: []string{"cycle"}, ModesExclude: []string{"cycle"}}, "cycle", "", false},
		{"phase include hit", Eligibility{PhasesInclude: []string{"luteal", "menstrual"}}, "cycle", "luteal", true},
		{"phase include miss", Eligibility{PhasesInclude: []string{"luteal"}}, "cycle", "follicular", false},
		{"phase include without a phase", Eligibility{PhasesInclude: []string{"luteal"}}, "pregnancy", "", false},
		{"phase exclude", Eligibility{PhasesExclude: []string{"fertile"}}, "cycle", "fertile", false},
		{"phase exclude other", Eligibility{PhasesExclude: []string{"fertile"}}, "cycle", "luteal", true},
		{"artboard rule", Eligibility{ModesInclude: []string{"cycle", "ttc"}, PhasesInclude: []string{"luteal", "menstrual"}, ModesExclude: []string{"pregnancy", "postpartum"}}, "cycle", "luteal", true},
		{"artboard rule, postpartum", Eligibility{ModesInclude: []string{"cycle", "ttc"}, PhasesInclude: []string{"luteal", "menstrual"}, ModesExclude: []string{"pregnancy", "postpartum"}}, "postpartum", "luteal", false},
	} {
		assert.Equal(t, tc.want, tc.e.Allows(tc.mode, tc.phase), tc.name)
	}
}

func TestProgress_CompletionRule(t *testing.T) {
	tasks := []store.ChallengeTask{{ID: 1, DayFrom: 1, DayTo: 3}, {ID: 2, DayFrom: 2, DayTo: 3}}
	done := map[[2]uint64]bool{{1, 1}: true, {1, 2}: true, {2, 2}: true}
	p := Program{DurationDays: 3, CompletionMinDays: 2}
	d, req := Progress(p, tasks, done)
	assert.Equal(t, 2, d, "day 1 (task 1) and day 2 (tasks 1+2) are done")
	assert.Equal(t, 2, req)
	_, req = Progress(Program{DurationDays: 3}, tasks, done)
	assert.Equal(t, 3, req, "no minimum = every day")
	_, req = Progress(Program{DurationDays: 3, CompletionMinDays: 9}, tasks, done)
	assert.Equal(t, 3, req, "never more than the days that have tasks")
	d, _ = Progress(p, tasks, map[[2]uint64]bool{{2, 2}: true})
	assert.Equal(t, 0, d, "a day counts only when every task of it is done")
}
