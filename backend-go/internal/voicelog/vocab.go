package voicelog

import (
	"slices"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// slot is one loggable target of a suggestion: a param, or one item / option of an items / multi param.
type slot struct {
	cat   *taxonomy.Category
	param *taxonomy.Param
	item  string // "" for single / number / integer / bool
	// labels in the request language
	catLabel, paramLabel, itemLabel string
	levels                          map[string]string // items: level code → label
	options                         map[string]string // single: option code → label
	unit                            string            // number / integer: unit label
}

// labelsOf reads the `log-taxonomy` namespace (request language over the default one).
type labelsOf struct{ ns any }

func (l labelsOf) get(path, fallback string) string {
	if v, ok := phpval.Get(l.ns, path); ok {
		if s, isStr := v.(string); isStr && s != "" {
			return s
		}
	}
	return fallback
}

func (l labelsOf) level(code string) string { return l.get("levels."+code, code) }

// voiceTypes are the param types voice suggestions may fill. Text (the note), text_items (other meds),
// link tiles and meds.taken (care reminder ids) stay manual.
var voiceTypes = []taxonomy.Type{taxonomy.Single, taxonomy.Multi, taxonomy.Items, taxonomy.Number, taxonomy.Integer, taxonomy.Bool}

// vocabulary is what the parser may answer with for a user: every voice-fillable slot available in her
// mode, plus her active custom items, with labels in the request language.
type vocabulary struct {
	entries []ai.VocabEntry
	slots   map[string]slot
	custom  taxonomy.CustomItems
	canvas  map[string]canvasField // CB-VOICE-01 items the user may log (addCanvas)
}

// buildVocabulary lists the slots of mode; custom are the user's active custom items.
func buildVocabulary(mode string, ns any, custom []store.HealthLogCustomItem) *vocabulary {
	l := labelsOf{ns: ns}
	v := &vocabulary{slots: map[string]slot{}, custom: taxonomy.CustomItems{}}
	customBy := map[string][]store.HealthLogCustomItem{}
	for _, ci := range custom {
		customBy[ci.Category+"."+ci.Param] = append(customBy[ci.Category+"."+ci.Param], ci)
		v.custom.Add(ci.Category, ci.Param, taxonomy.CustomItemCode(ci.ID))
	}
	for _, cat := range taxonomy.Categories() {
		if !slices.Contains(cat.Modes, mode) {
			continue
		}
		c := cat
		catLabel := l.get("categories."+c.Code+".title", c.Code)
		for i := range c.Params {
			p := &c.Params[i]
			if !c.Available(p, mode) || !slices.Contains(voiceTypes, p.Type) || p.Dynamic && !p.Custom {
				continue
			}
			base := "categories." + c.Code + ".params." + p.Code
			paramLabel := l.get(base+".title", p.Code)
			key := c.Code + "." + p.Code
			s := slot{cat: catByCode(c.Code), param: p, catLabel: catLabel, paramLabel: paramLabel}
			switch p.Type {
			case taxonomy.Multi, taxonomy.Items:
				var levels []ai.VocabValue
				s.levels = map[string]string{}
				for _, lv := range p.Levels {
					levels = append(levels, ai.VocabValue{Code: lv, Label: l.level(lv)})
					s.levels[lv] = l.level(lv)
				}
				add := func(code, label string) {
					it := s
					it.item, it.itemLabel = code, label
					k := key + "." + code
					v.slots[k] = it
					v.entries = append(v.entries, ai.VocabEntry{Key: k, Type: string(p.Type),
						Label: catLabel + " › " + paramLabel + " › " + label, Values: levels})
				}
				for j := range p.Options {
					o := &p.Options[j]
					if o.OptionAvailable(mode) {
						add(o.Code, l.get(base+".options."+o.Code, o.Code))
					}
				}
				if p.Custom {
					for _, ci := range customBy[key] {
						add(taxonomy.CustomItemCode(ci.ID), ci.Label)
					}
				}
			default:
				e := ai.VocabEntry{Key: key, Type: string(p.Type), Label: catLabel + " › " + paramLabel}
				if p.Type == taxonomy.Single {
					s.options = map[string]string{}
					for j := range p.Options {
						o := &p.Options[j]
						if o.OptionAvailable(mode) {
							label := l.get(base+".options."+o.Code, o.Code)
							e.Values = append(e.Values, ai.VocabValue{Code: o.Code, Label: label})
							s.options[o.Code] = label
						}
					}
				}
				if p.Range != nil {
					e.Min, e.Max = p.Range.Min, p.Range.Max
					if p.Unit != "" {
						s.unit = l.get("units."+p.Unit, p.Unit)
					}
					e.Unit = s.unit
				}
				v.slots[key] = s
				v.entries = append(v.entries, e)
			}
		}
	}
	return v
}

func catByCode(code string) *taxonomy.Category {
	c, _ := taxonomy.CategoryByCode(code)
	return c
}
