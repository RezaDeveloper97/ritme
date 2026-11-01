package inboxalerts

import (
	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/inbox"
	"github.com/ritme/backend-go/internal/messages/rules"
)

// FactsOf are the rule facts of one input (pure: the pregnancy alert producer plus pregnancy_week), what the preg.*
// rules read in production.
func FactsOf(in *facts.PregnancyAlertInput) rules.Facts {
	f := rules.Facts(facts.PregnancyAlertValues(in))
	f["pregnancy_week"] = float64(in.Facts.Week)
	return f
}

// Fired is one code-held rule that holds on an input, rendered from the embedded copy.
type Fired struct {
	Key, Level      string // the rule key and its stored level (rules.Level*)
	Title, Seen     string
	Confidence, Ask string // Ask is the contact line (urgent rules)
}

// Fire evaluates the code-held definitions (Defs, the seeded rules as shipped) on in and renders each match in locale
// from the embedded copy. It reads no database: the safety scenarios and the parity table run it, and it is the same
// definition the emergency fallback serves when a locked row is gone (SPEC §4.2). A follow_up / urgent match without
// renderable evidence is left out, as the inbox leaves it out (SPEC §5.4.1).
func Fire(in *facts.PregnancyAlertInput, locale string) []Fired {
	f := FactsOf(in)
	var out []Fired
	for i, d := range Defs {
		r := d.Rule(CodeRuleIDBase + uint64(i)) //nolint:gosec // small index
		if !rules.MatchRule(&r, f) {
			continue
		}
		if _, _, ok := inbox.EvidenceOf(&r, f); !ok {
			continue
		}
		cp, _ := CopyOf(d.Key, locale)
		out = append(out, Fired{Key: d.Key, Level: d.Level, Title: Render(cp.Title, f, locale),
			Seen: Render(cp.Seen, f, locale), Confidence: Render(cp.Confidence, f, locale), Ask: Render(cp.Contact, f, locale)})
	}
	return out
}

// Fires is the match of key on in (ok false when the rule does not hold).
func Fires(key string, in *facts.PregnancyAlertInput, locale string) (Fired, bool) {
	for _, m := range Fire(in, locale) {
		if m.Key == key {
			return m, true
		}
	}
	return Fired{}, false
}
