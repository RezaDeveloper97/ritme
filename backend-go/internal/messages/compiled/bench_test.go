package compiled_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
)

// T-M10-05 budget: compile 500 rule rows (JSON parse + compile + fingerprint) ≤ 20 ms.
func BenchmarkCompile_500Rows(b *testing.B) {
	rows := make([]store.MessageRule, 0, 500)
	for i := range 500 {
		r := row(uint64(i+1), uint16(100+i%600), fmt.Sprintf("t%d", i)) //nolint:gosec // small
		r.Placement = []string{"daily_advice", "smart_tip", "push", "alerts"}[i%4]
		r.Kind = []string{rules.KindContent, rules.KindClinical, rules.KindContent, rules.KindClinical}[i%4]
		r.Conditions = fmt.Sprintf(`[{"op":"and","items":[{"fact":"mode","op":"eq","value":"cycle"},`+
			`{"fact":"cycle_day","op":"between","value":[%d,%d]},{"fact":"symptoms","op":"nin","value":"s%d,s%d"}]}]`,
			1+i%10, 20+i%20, i%17, i%5)
		r.Exceptions = `[{"fact":"confidence","op":"eq","value":"low"}]`
		rows = append(rows, r)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := compiled.Compile(rows); err != nil {
			b.Fatal(err)
		}
	}
}

// T-M10-05 budget: Select for one slot over 200 compiled rules ≤ 50 µs and ≤ 2 allocs/op. Same mix as the rules
// package's BenchmarkSelectSlot_200Rules (a third do not match, a quarter shown recently, caps and cooldowns).
func BenchmarkSelect_200Compiled(b *testing.B) {
	const n = 200
	all := make([]rules.Rule, 0, n)
	h := &rules.UserHistory{Rules: map[uint64]rules.RuleHistory{}, Slots: map[string]rules.WindowCounts{}}
	for i := range n {
		id := uint64(i + 1) //nolint:gosec // small
		r := rules.Rule{ID: id, Level: rules.LevelInfo, Kind: rules.KindContent, Priority: 100 + i%500,
			Placement: "daily_advice", Status: rules.StatusApproved, IsActive: true, AudiencePercent: 100,
			HoursFrom: -1, HoursTo: -1, ABSplit: 50, Content: rules.Content{A: rules.Variant{Title: "t"}}}
		r.Caps = []rules.Cap{{Count: 1 + i%3, Period: []string{"day", "week", "cycle"}[i%3]}}
		r.CooldownDays = i % 5
		mode := "cycle"
		if i%3 == 0 {
			mode = "pregnancy"
		}
		r.Groups = []rules.Group{{Op: "and", Items: []rules.Condition{
			{Fact: "mode", Op: "eq", Value: mode},
			{Fact: "cycle_day", Op: "between", Value: []any{float64(1 + i%10), float64(20 + i%20)}},
			{Fact: "symptoms", Op: "nin", Value: fmt.Sprintf("s%d", i%17)},
		}}}
		if i%4 == 0 {
			at := tNow.Add(-time.Duration(i+1) * time.Hour)
			h.Rules[id] = rules.RuleHistory{LastShown: at, Shown: rules.WindowCounts{Day: 1, Week: 1, Cycle: 1}}
		}
		all = append(all, r)
	}
	facts := rules.Facts{"mode": "cycle", "cycle_day": 12.0, "cycle_length": 29.0, "symptoms": []string{"s1"},
		"current_period_start": "2026-10-10", "confidence": "high"}
	c := compiled.CompileRules(all)
	if want, _ := rules.SelectSlot("daily_advice", all, facts, h, 7, tNow); len(want) != 3 ||
		fmt.Sprint(wins(want)) != fmt.Sprint(wins(c.Select("daily_advice", facts, h, 7, tNow))) {
		b.Fatal("compiled and parsed winners differ")
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = c.Select("daily_advice", facts, h, 7, tNow)
	}
}

// The worst case for the early stop: every rule shares one priority, so all 200 are evaluated.
func BenchmarkSelect_200CompiledOnePriority(b *testing.B) {
	all := make([]rules.Rule, 0, 200)
	for i := range 200 {
		all = append(all, rules.Rule{ID: uint64(i + 1), Level: rules.LevelInfo, Kind: rules.KindContent, //nolint:gosec // small
			Priority: 300, Placement: "daily_advice", Status: rules.StatusApproved, IsActive: true,
			AudiencePercent: 50, HoursFrom: -1, HoursTo: -1, Variants: []rules.VariantWeight{{Key: "a", Weight: 1}, {Key: "b", Weight: 1}},
			Groups: []rules.Group{{Op: "or", Items: []rules.Condition{
				{Fact: "mode", Op: "in", Value: []any{"cycle", "ttc"}}, {Fact: "flag", Op: "eq", Value: true}}}}})
	}
	c := compiled.CompileRules(all)
	facts := rules.Facts{"mode": "cycle"}
	b.ReportAllocs()
	for b.Loop() {
		_ = c.Select("daily_advice", facts, nil, 7, tNow)
	}
}
