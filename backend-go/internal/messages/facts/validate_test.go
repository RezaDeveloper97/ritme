package facts_test

// T-M10-03 (D-161): rule validation against the fact registry (rules.Input.Check / rules.CheckFact).

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/messages/rules"
)

func clinical(conds ...rules.Condition) rules.Input {
	lo, _ := rules.Band(rules.KindClinical)
	return rules.Input{Title: "x", Level: rules.LevelInfo, Kind: rules.KindClinical, Priority: lo, Placement: "smart_tip",
		Content:    rules.Content{A: rules.Variant{Title: "t", Body: "b"}},
		Conditions: []rules.Group{{Op: "and", Items: conds}}}
}

func TestCheck_FactRegistry(t *testing.T) {
	for _, tc := range []struct {
		name string
		c    rules.Condition
		code string
	}{
		{"valid enum in", rules.Condition{Fact: "mode", Op: "in", Value: []any{"cycle", "ttc"}}, ""},
		{"valid number between", rules.Condition{Fact: "days_late", Op: "between", Value: []any{8.0, 30.0}}, ""},
		{"valid set contains", rules.Condition{Fact: "warnings", Op: "contains", Value: "predicted_period_overdue"}, ""},
		{"valid bool", rules.Condition{Fact: "requires_user_input", Op: "eq", Value: true}, ""},
		{"valid symptom pattern", rules.Condition{Fact: "symptom_history.cramps", Op: "gte", Value: 3.0}, ""},
		{"valid banner", rules.Condition{Fact: "today_ui.banner", Op: "eq", Value: "overdue"}, ""},
		{"valid date eq", rules.Condition{Fact: "current_period_start", Op: "eq", Value: "2026-10-01"}, ""},
		{"unknown fact", rules.Condition{Fact: "bp", Op: "gt", Value: 180.0}, rules.CodeUnknownFact},
		{"variable is not a fact", rules.Condition{Fact: "name", Op: "eq", Value: "Sara"}, rules.CodeUnknownFact},
		{"between on an enum", rules.Condition{Fact: "main_phase", Op: "between", Value: []any{1.0, 2.0}}, rules.CodeBadValue},
		{"bad enum value", rules.Condition{Fact: "main_phase", Op: "eq", Value: "ovulation"}, rules.CodeBadValue},
		{"bad enum value in list", rules.Condition{Fact: "plan", Op: "in", Value: []any{"plus", "gold"}}, rules.CodeBadValue},
		{"in on a number", rules.Condition{Fact: "cycle_day", Op: "in", Value: []any{1.0}}, rules.CodeBadValue},
		{"contains on an enum", rules.Condition{Fact: "mode", Op: "contains", Value: "cycle"}, rules.CodeBadValue},
		{"unknown warning", rules.Condition{Fact: "warnings", Op: "contains", Value: "made_up"}, rules.CodeBadValue},
		{"eq on a set", rules.Condition{Fact: "warnings", Op: "eq", Value: "low_history"}, rules.CodeBadValue},
		{"number expected", rules.Condition{Fact: "days_late", Op: "gt", Value: "eight"}, rules.CodeBadValue},
		{"between low > high", rules.Condition{Fact: "days_late", Op: "between", Value: []any{9.0, 2.0}}, rules.CodeBadValue},
		{"bool expected", rules.Condition{Fact: "requires_user_input", Op: "eq", Value: 1.0}, rules.CodeBadValue},
		{"date range never matches", rules.Condition{Fact: "current_period_start", Op: "gt", Value: "2026-10-01"}, rules.CodeBadValue},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, code := rules.CheckFact(tc.c)
			assert.Equal(t, tc.code, code)
			in := clinical(tc.c)
			errs, code := in.Check()
			assert.Equal(t, tc.code, code)
			if tc.code == "" {
				assert.Empty(t, errs)
			} else {
				assert.Contains(t, errs, "conditions")
			}
		})
	}
	// An exception is checked the same way; unknown_fact wins over bad_value.
	in := clinical(rules.Condition{Fact: "main_phase", Op: "eq", Value: "bogus"})
	in.Exceptions = []rules.Condition{{Fact: "nope", Op: "eq", Value: 1.0}}
	errs, code := in.Check()
	assert.Equal(t, rules.CodeUnknownFact, code)
	assert.Contains(t, errs, "exceptions")
}
