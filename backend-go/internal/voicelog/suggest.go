package voicelog

import (
	"encoding/json"
	"math"
	"slices"
	"strconv"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Limits of what a parse may suggest.
const (
	// MinConfidence drops guesses the provider itself doubts.
	MinConfidence = 0.5
	// MaxSuggestions caps one answer (a provider gone wild cannot flood the review).
	MaxSuggestions = 20
)

// Suggestion is one reviewed-before-saving item: the PUT /logs/days/{date} slot it fills (target "log") or the
// canvas item it fills (target = its category, saved by POST /logs/voice/commit), and its value.
type Suggestion struct {
	Target                string
	Category, Param, Item string
	// Value: single → option code, multi → true, items → level code, number/integer → float64, bool → bool,
	// text → string, time → "HH:MM".
	Value      any
	Confidence float64
	Label      string
	// Options are the other slots the same words may mean (the review's chooser); empty when unambiguous.
	Options []Option
}

// Option is one alternative reading of a suggestion: a log slot with the value it would get.
type Option struct {
	Category, Param, Item string
	Value                 any
	Label                 string
}

func wireValue(v any) any {
	if f, ok := v.(float64); ok {
		return jsonx.Float(f)
	}
	return v
}

func wireItem(item string) any {
	if item == "" {
		return nil
	}
	return item
}

// JSON is the wire shape {target, category, param, item|null, value, confidence, label, options[]}.
func (s Suggestion) JSON() *jsonx.OrderedMap {
	opts := make([]*jsonx.OrderedMap, 0, len(s.Options))
	for _, o := range s.Options {
		opts = append(opts, jsonx.Obj("category", o.Category, "param", o.Param, "item", wireItem(o.Item),
			"value", wireValue(o.Value), "label", o.Label))
	}
	return jsonx.Obj("target", s.Target, "category", s.Category, "param", s.Param, "item", wireItem(s.Item),
		"value", wireValue(s.Value), "confidence", jsonx.Float(math.Round(s.Confidence*100)/100), "label", s.Label,
		"options", opts)
}

// JSON is the wire shape of a committed item {target, category, param, item: null, value, label}.
func (s Saved) JSON() *jsonx.OrderedMap {
	return jsonx.Obj("target", s.Target, "category", s.Target, "param", s.Param, "item", nil,
		"value", wireValue(s.Value), "label", s.Label)
}

// validate keeps the candidates that are real, available slots for the user with a valid value, in the
// provider's order, de-duplicated (first wins). Every kept value is re-checked by taxonomy.Parse, the same
// validator PUT /logs/days uses, so a suggestion can always be saved.
func (v *vocabulary) validate(cands []ai.Candidate, mode, locale string) []Suggestion {
	out := []Suggestion{}
	seen := map[string]bool{}
	for _, c := range cands {
		if len(out) >= MaxSuggestions {
			break
		}
		if seen[c.Key] || math.IsNaN(c.Confidence) || c.Confidence < MinConfidence {
			continue
		}
		conf := math.Min(c.Confidence, 1)
		if f, ok := v.canvas[c.Key]; ok {
			value, ok := normalizeCanvas(f, c.Value)
			if !ok {
				continue
			}
			seen[c.Key] = true
			out = append(out, Suggestion{Target: f.target, Category: f.target, Param: f.param, Value: value,
				Confidence: conf, Label: canvasLabel(f, value, locale), Options: []Option{}})
			continue
		}
		s, ok := v.slots[c.Key]
		if !ok {
			continue
		}
		value, body, ok := normalize(s, c.Value)
		if !ok || !v.parses(s, body, mode, locale) {
			continue
		}
		seen[c.Key] = true
		out = append(out, Suggestion{Target: TargetLog, Category: s.cat.Code, Param: s.param.Code, Item: s.item,
			Value: value, Confidence: conf, Label: label(s, value, locale),
			Options: v.options(c, value, mode, locale)})
	}
	return out
}

// MaxOptions caps the alternatives of one suggestion.
const MaxOptions = 4

// options are the valid alternatives of a log suggestion: other log slots of the multi / items / bool kind (their
// value follows from the suggestion's: true, the same level when the slot has it, else "yes" / its default).
func (v *vocabulary) options(c ai.Candidate, value any, mode, locale string) []Option {
	out := []Option{}
	done := map[string]bool{c.Key: true}
	for _, key := range c.Alternatives {
		if len(out) >= MaxOptions {
			break
		}
		s, ok := v.slots[key]
		if !ok || done[key] {
			continue
		}
		raw, ok := altValue(s, value)
		if !ok {
			continue
		}
		val, body, ok := normalize(s, raw)
		if !ok || !v.parses(s, body, mode, locale) {
			continue
		}
		done[key] = true
		out = append(out, Option{Category: s.cat.Code, Param: s.param.Code, Item: s.item, Value: val,
			Label: label(s, val, locale)})
	}
	return out
}

// altValue is the value an alternative slot takes for the suggestion's value.
func altValue(s slot, value any) (any, bool) {
	switch s.param.Type {
	case taxonomy.Multi, taxonomy.Bool:
		return true, true
	case taxonomy.Items:
		if level, ok := value.(string); ok && level != taxonomy.No && slices.Contains(s.param.Levels, level) {
			return level, true
		}
		for _, l := range []string{taxonomy.Yes, "moderate"} {
			if slices.Contains(s.param.Levels, l) {
				return l, true
			}
		}
	}
	return nil, false
}

// normalize checks the candidate value against the slot type and returns it plus its PUT body value.
func normalize(s slot, raw any) (any, any, bool) {
	p := s.param
	switch p.Type {
	case taxonomy.Single:
		code, ok := raw.(string)
		return code, code, ok && code != ""
	case taxonomy.Multi:
		if b, ok := raw.(bool); ok && !b {
			return nil, nil, false
		}
		return true, []any{s.item}, true
	case taxonomy.Items:
		level, ok := raw.(string)
		if !ok || level == "" || level == taxonomy.No {
			return nil, nil, false
		}
		return level, map[string]any{s.item: level}, true
	case taxonomy.Number, taxonomy.Integer:
		var n float64
		switch x := raw.(type) {
		case float64:
			n = x
		case int:
			n = float64(x)
		case int64:
			n = float64(x)
		case string:
			f, err := strconv.ParseFloat(x, 64)
			if err != nil {
				return nil, nil, false
			}
			n = f
		default:
			return nil, nil, false
		}
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return nil, nil, false
		}
		if p.Type == taxonomy.Integer {
			n = math.Round(n)
		} else {
			f := math.Pow(10, float64(p.Scale))
			n = math.Round(n*f) / f
		}
		return n, n, true
	case taxonomy.Bool:
		b, ok := raw.(bool)
		return b, b, ok
	}
	return nil, nil, false
}

// parses runs the PUT /logs/days validator on {"categories":{cat:{param:value}}} for the slot alone.
func (v *vocabulary) parses(s slot, value any, mode, locale string) bool {
	raw, err := json.Marshal(map[string]any{"categories": map[string]any{s.cat.Code: map[string]any{s.param.Code: value}}})
	if err != nil {
		return false
	}
	decoded, err := phpval.Decode(raw)
	if err != nil {
		return false
	}
	body, ok := decoded.(phpval.Map)
	if !ok {
		return false
	}
	_, err = taxonomy.Parse(body, mode, locale, nil, v.custom)
	return err == nil
}

// label is the chip text: «درد شکم · متوسط», «نفخ», «کیفیت خواب · خوب», «وزن · 58.5 کیلو».
func label(s slot, value any, locale string) string {
	p := s.param
	l := func(key string, params map[string]string) string { return T("labels."+key, locale, params) }
	switch p.Type {
	case taxonomy.Items:
		item := s.itemLabel
		if s.cat.Code == "pain" && p.Code == "location" {
			item = l("pain_location", map[string]string{"item": item})
		}
		level, _ := value.(string)
		if graded(p) && level != "" {
			return l("graded", map[string]string{"item": item, "level": or(s.levels[level], level)})
		}
		return item
	case taxonomy.Multi:
		return s.itemLabel
	case taxonomy.Single:
		code, _ := value.(string)
		return l("single", map[string]string{"category": s.catLabel, "param": s.paramLabel, "option": or(s.options[code], code)})
	case taxonomy.Number, taxonomy.Integer:
		n, _ := value.(float64)
		return l("number", map[string]string{"param": s.paramLabel, "value": strconv.FormatFloat(n, 'f', -1, 64), "unit": s.unit})
	default:
		return s.paramLabel
	}
}

// graded: an items param with intensity levels (pain) rather than yes/no.
func graded(p *taxonomy.Param) bool {
	hasYes, hasMild := false, false
	for _, l := range p.Levels {
		hasYes = hasYes || l == taxonomy.Yes
		hasMild = hasMild || l == "mild"
	}
	return hasMild && !hasYes
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
