package conditionnudges

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/messages/compiled"
	mfacts "github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// legacyDetect is the retired D-41 engine (CB-COND-06) verbatim, kept as the parity oracle of T-M10-09: its MinDays
// and the enrolment check are now conditions of the seeded cond.* rules.
func legacyDetect(f Facts) []Hit {
	const minDays = 2
	if !slices.Contains(nudgedModes, f.Mode) || f.From.IsZero() {
		return nil
	}
	type legacyRule struct {
		key, program, link string
		day                func([]taxonomy.Entry) bool
	}
	var out []Hit
	for _, r := range []legacyRule{
		{RuleHeavyPain, ProgramEndo, "/programs/pain", strongPainDay},
		{RuleHeavyBleeding, ProgramHeavyBleeding, "/programs/bleeding", heavyFlowDay},
	} {
		if f.Enrolled[r.program] {
			continue
		}
		var dates []civildate.Date
		for _, d := range f.Days {
			if d.Date.Before(f.From) || d.Date.After(f.To) {
				continue
			}
			if r.day(d.Entries) {
				dates = append(dates, d.Date)
			}
		}
		if len(dates) >= minDays {
			out = append(out, Hit{Rule: r.key, Program: r.program, Link: r.link, Dates: dates})
		}
	}
	return out
}

// parityFixtures are the rules_test fixtures with every mode and enrolment variant.
func parityFixtures() map[string]Facts {
	both := []healthlog.DayEntries{
		day("2026-09-01", flow("heavy"), pain("abdomen", "severe", "8")),
		day("2026-09-02", flow("heavy"), pain("abdomen", "severe", "10")),
	}
	out := map[string]Facts{
		"pain 7 and 8": facts("cycle", day("2026-09-02", pain("abdomen", "severe", "7")),
			day("2026-09-03", pain("pelvis", "moderate", "5"), pain("back", "severe", "8.0"))),
		"one day only": facts("cycle", day("2026-09-02", pain("abdomen", "severe", "9"))),
		"scores of 6": facts("cycle", day("2026-09-02", pain("abdomen", "moderate", "6")),
			day("2026-09-03", pain("abdomen", "moderate", "6"))),
		"score beats level": facts("cycle", day("2026-09-02", pain("abdomen", "severe", "6")),
			day("2026-09-03", pain("abdomen", "severe", "6"))),
		"no items": facts("cycle", day("2026-09-02", pain("abdomen", taxonomy.No, "9")),
			day("2026-09-03", pain("abdomen", taxonomy.No, "9"))),
		"outside the cycle": facts("cycle", day("2026-08-30", pain("abdomen", "severe", "9")),
			day("2026-09-02", pain("abdomen", "severe", "9"))),
		"severe level unscored": facts("ttc", day("2026-09-02", pain("abdomen", "severe", "")),
			day("2026-09-05", pain("head", "severe", ""))),
		"heavy flow teen": facts("teen", day("2026-09-01", flow("very_heavy")), day("2026-09-02", flow("medium")),
			day("2026-09-03", flow("heavy"))),
		"one heavy day": facts("cycle", day("2026-09-01", flow("heavy")), day("2026-09-02", flow("medium"))),
		"both":          facts("cycle", both...),
		"no window":     {Mode: "cycle", Enrolled: map[string]bool{}, Days: both},
	}
	for _, en := range [][]string{{ProgramEndo}, {ProgramHeavyBleeding}, {ProgramEndo, ProgramHeavyBleeding}, {"pmdd"}} {
		f := facts("cycle", both...)
		for _, p := range en {
			f.Enrolled[p] = true
		}
		out["enrolled "+en[0]+" "+string(rune('0'+len(en)))] = f
	}
	for _, mode := range []string{"pregnancy", "postpartum", "menopause", ""} {
		out["mode "+mode] = facts(mode, both...)
	}
	return out
}

// T-M10-09 parity table: the retired engine's nudges equal the cond.* rule matches over every fixture — through the
// code-held rules, the seeded rules compiled as the engine compiles them, and the alerts inbox selection.
func TestParity_LegacyEngineEqualsRules(t *testing.T) {
	var box []rules.Rule
	for _, d := range Defs {
		box = append(box, d.Rule(CodeRuleID(d.Key)))
	}
	comp := compiled.CompileRules(box)
	byKey := map[string]*rules.Rule{}
	for _, r := range comp.BySlot[rules.SlotAlertsInbox] {
		byKey[r.Src.RuleKey] = r.Src
	}
	require.Len(t, byKey, len(Defs))
	compiledHolds := func(d *Def, f rules.Facts) bool { return rules.MatchRule(byKey[d.Key], f) }
	now := d("2026-09-20").TehranMidnight().Add(10 * 3600e9)

	for name, f := range parityFixtures() {
		want := legacyDetect(f)
		assert.Equal(t, want, Detect(f), "%s: code-held", name)
		assert.Equal(t, want, DetectWith(f, compiledHolds), "%s: seeded", name)

		selected := []string{}
		for _, ch := range comp.Select(rules.SlotAlertsInbox, rules.Facts(mfacts.ConditionValues(Input(f))), nil, 1, now) {
			selected = append(selected, ch.Rule.RuleKey)
		}
		wantKeys := []string{}
		for _, h := range want {
			wantKeys = append(wantKeys, KeyPrefix+h.Rule)
		}
		slices.Sort(selected)
		slices.Sort(wantKeys)
		assert.Equal(t, wantKeys, selected, "%s: inbox selection", name)
	}
}

func TestSeedCopy_EveryRuleInEveryLanguage(t *testing.T) {
	for _, code := range CopyLanguages() {
		for _, d := range Defs {
			cp, ok := embedded()[code][d.Key]
			require.True(t, ok, "%s %s", code, d.Key)
			assert.NotEmpty(t, cp.Title, "%s %s", code, d.Key)
			assert.NotEmpty(t, cp.Body, "%s %s", code, d.Key)
			for _, v := range rules.Placeholders(cp.Title, cp.Body, cp.CTA) {
				def, ok := mfacts.Lookup(v)
				assert.True(t, ok && def.Variable, "%s %s: {%s} is a registered text variable", code, d.Key, v)
			}
		}
	}
	for _, d := range Defs {
		assert.True(t, rules.LevelAllowed(d.Level, d.Kind), d.Key)
		assert.True(t, rules.KindAllowedIn(Slot, d.Kind), d.Key)
	}
}
