package feed

import (
	"context"
	"encoding/json"
	"slices"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

var markErrors = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "messages_feed_mark_errors_total",
	Help: "Feed builds whose shown/eligible marks fell back to the database check because Redis failed.",
})

func init() { metrics.Registry().MustRegister(markErrors) }

// Translations are stored in the content JSON as content.translations.<code> = {a, b?, variants?, seen?,
// confidence?, contact?}. The base content is the default language (the language list is data: the codes come from
// the stored JSON, never from code).

// extras are what the feed derives once per compiled snapshot: the parsed rules of each slot (the parsed selection
// pass), the {variables} each slot's texts read, and the rules' translations.
type extras struct {
	comp         *compiled.Compiled
	parsed       map[string][]rules.Rule
	vars         map[string][]string
	translations map[uint64]map[string]rules.Content
}

var current atomic.Pointer[extras]

// extrasOf returns the extras of comp (built on the first request after a recompile).
func extrasOf(ctx context.Context, q store.Querier, comp *compiled.Compiled) *extras {
	if x := current.Load(); x != nil && x.comp == comp {
		return x
	}
	x := &extras{comp: comp, parsed: map[string][]rules.Rule{}, vars: map[string][]string{},
		translations: map[uint64]map[string]rules.Content{}}
	for slot, list := range comp.BySlot {
		ps := make([]rules.Rule, 0, len(list))
		var vars []string
		for _, r := range list {
			ps = append(ps, *r.Src)
			vars = append(vars, rules.Placeholders(contentTexts(&r.Src.Content)...)...)
		}
		slices.Sort(vars)
		x.parsed[slot], x.vars[slot] = ps, slices.Compact(vars)
	}
	// The translations live in the raw content JSON (rules.Content does not carry them); the rows come from the same
	// process snapshot compiled.Load read, so this costs no query once warm.
	rows, err := comp.Rows, error(nil)
	if rows == nil { // T-M9-49b: a snapshot built outside a Cache
		rows, err = rules.ActiveRows(ctx, q, "")
	}
	if err == nil {
		for i := range rows {
			if t := parseTranslations(rows[i].Content); len(t) > 0 {
				x.translations[rows[i].ID] = t
			}
		}
	}
	current.Store(x)
	return x
}

func parseTranslations(raw string) map[string]rules.Content {
	var c struct {
		Translations map[string]rules.Content `json:"translations"`
	}
	if json.Unmarshal([]byte(raw), &c) != nil {
		return nil
	}
	return c.Translations
}

// contentTexts are every text of a content (base variants; translations use the same variables).
func contentTexts(c *rules.Content) []string {
	out := []string{c.A.Title, c.A.Body, c.A.CTALabel}
	if c.B != nil {
		out = append(out, c.B.Title, c.B.Body, c.B.CTALabel)
	}
	for _, v := range c.Variants {
		out = append(out, v.Title, v.Body, v.CTALabel)
	}
	return out
}

// placeholders is the union of the slots' text variables.
func (x *extras) placeholders(slots []string) []string {
	var out []string
	for _, s := range slots {
		out = append(out, x.vars[s]...)
	}
	return out
}

// Text is the variant v of r in locale: the translation's text when the rule has one for locale, else the default
// language's.
func (x *extras) text(r *rules.Rule, v, locale string) rules.Variant {
	base := r.VariantContent(v)
	t, ok := x.translations[r.ID][locale]
	if !ok {
		return base
	}
	tr := (&rules.Rule{Content: t}).VariantContent(v)
	if tr.Title == "" && tr.Body == "" {
		return base
	}
	if tr.CTATarget == "" {
		tr.CTATarget = base.CTATarget
	}
	return tr
}

// Evidence is the «what we saw / how sure / contact» of r in locale.
func (x *extras) evidence(r *rules.Rule, locale string) rules.Content {
	c := r.Content
	if t, ok := x.translations[r.ID][locale]; ok {
		if t.Seen != "" {
			c.Seen = t.Seen
		}
		if t.Confidence != "" {
			c.Confidence = t.Confidence
		}
		if t.Contact != "" {
			c.Contact = t.Contact
		}
	}
	return c
}

// render renders every winner; ok is false when one of them lacks a variable value.
func (x *extras) render(chs []rules.Choice, f rules.Facts, locale string) ([]rules.Variant, bool) {
	out := make([]rules.Variant, 0, len(chs))
	all := true
	for _, ch := range chs {
		v, ok := rules.RenderVariant(x.text(ch.Rule, ch.Variant, locale), f)
		all = all && ok
		out = append(out, v)
	}
	return out, all
}

// renderable are the slot's parsed rules whose texts (the user's variant, in locale) render with f.
func (x *extras) renderable(slot string, f rules.Facts, locale string, userID uint64) []rules.Rule {
	all := x.parsed[slot]
	out := make([]rules.Rule, 0, len(all))
	for i := range all {
		if _, ok := rules.RenderVariant(x.text(&all[i], all[i].Assign(userID), locale), f); ok {
			out = append(out, all[i])
		}
	}
	return out
}
