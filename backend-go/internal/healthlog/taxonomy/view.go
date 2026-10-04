package taxonomy

import (
	"sort"
	"strconv"

	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// DayJSON renders a day's entries in the PUT body shape ({cat: {param: value}}), categories and params in
// taxonomy order (anything the registry does not know — a param a later release dropped — after them,
// sorted). single → code, multi → [codes], items → {item: {level, score}}, number/integer → number,
// text → string, text_items → {item: text}, bool → true/false.
func DayJSON(entries []Entry) *jsonx.OrderedMap {
	byParam := map[string][]Entry{}
	var unknown []string
	for _, e := range entries {
		k := e.ParamKey()
		if _, seen := byParam[k]; !seen {
			if !known(e.Category, e.Param) {
				unknown = append(unknown, k)
			}
		}
		byParam[k] = append(byParam[k], e)
	}
	out := jsonx.NewObject()
	put := func(cat, param string, v any) {
		cv, ok := out.Get(cat)
		if !ok {
			cv = jsonx.NewObject()
			out.Set(cat, cv)
		}
		cv.(*jsonx.OrderedMap).Set(param, v)
	}
	for i := range registry {
		c := &registry[i]
		for j := range c.Params {
			p := &c.Params[j]
			if es, ok := byParam[c.Code+"."+p.Code]; ok {
				put(c.Code, p.Code, value(p.Type, es))
			}
		}
	}
	sort.Strings(unknown)
	for _, k := range unknown {
		es := byParam[k]
		put(es[0].Category, es[0].Param, value(guessType(es), es))
	}
	return out
}

func known(cat, param string) bool {
	c, ok := CategoryByCode(cat)
	if !ok {
		return false
	}
	_, ok = c.Param(param)
	return ok
}

func guessType(es []Entry) Type {
	e := es[0]
	switch {
	case e.Item != "" && e.Text.Valid:
		return TextItems
	case e.Item != "":
		return Items
	case e.Num.Valid:
		return Number
	case e.Text.Valid:
		return Text
	}
	return Single
}

func num(s string) any {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return jsonx.Float(f)
}

func value(t Type, es []Entry) any {
	first := es[0]
	switch t {
	case Single:
		return first.Code.String
	case Bool:
		return first.Code.String != No
	case Multi:
		out := make([]string, 0, len(es))
		for _, e := range es {
			out = append(out, e.Item)
		}
		return out
	case Items:
		out := jsonx.NewObject()
		for _, e := range es {
			var score any
			if e.Num.Valid {
				if f, err := strconv.ParseFloat(e.Num.String, 64); err == nil {
					score = int64(f)
				}
			}
			out.Set(e.Item, jsonx.Obj("level", e.Code.String, "score", score))
		}
		return out
	case Number:
		return num(first.Num.String)
	case Integer:
		f, err := strconv.ParseFloat(first.Num.String, 64)
		if err != nil {
			return nil
		}
		return int64(f)
	case Text:
		return first.Text.String
	case TextItems:
		out := jsonx.NewObject()
		for _, e := range es {
			out.Set(e.Item, e.Text.String)
		}
		return out
	}
	return nil
}

// Labels reads the `log-taxonomy` translation namespace (already resolved for the request locale with the
// default language underneath). A missing label falls back to the code.
type Labels struct{ ns any }

// NewLabels wraps a namespace bundle (TranslationStore.NamespaceMessages).
func NewLabels(ns any) Labels { return Labels{ns: ns} }

func (l Labels) get(path, fallback string) string {
	if v, ok := phpval.Get(l.ns, path); ok {
		if s, isStr := v.(string); isStr && s != "" {
			return s
		}
	}
	return fallback
}

func labeled(values []string, label func(string) string) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(values))
	for _, v := range values {
		out = append(out, jsonx.Obj("value", v, "label", label(v)))
	}
	return out
}

func rangeJSON(r *Range) any {
	if r == nil {
		return nil
	}
	return jsonx.Obj("min", jsonx.Float(r.Min), "max", jsonx.Float(r.Max))
}

// CategoriesJSON is GET /logs/taxonomy: the registry with labels. mode "" lists everything (legacy-only
// values flagged); a mode keeps only what shows in it and drops legacy-only values.
func CategoriesJSON(mode string, l Labels) []*jsonx.OrderedMap {
	out := []*jsonx.OrderedMap{}
	for i := range registry {
		c := &registry[i]
		if mode != "" && !contains(c.Modes, mode) {
			continue
		}
		params := []*jsonx.OrderedMap{}
		for j := range c.Params {
			p := &c.Params[j]
			if mode != "" && !c.Available(p, mode) {
				continue
			}
			params = append(params, paramJSON(c, p, mode, l))
		}
		conds := jsonx.NewObject()
		for _, m := range AllModes {
			if v, ok := c.Conditions[m]; ok && (mode == "" || m == mode) {
				conds.Set(m, jsonx.Obj("value", v, "label", l.get("conditions."+v, v)))
			}
		}
		out = append(out, jsonx.Obj(
			"code", c.Code,
			"label", categoryLabel(c.Code, mode, l),
			"group", jsonx.Obj("value", c.Group, "label", l.get("groups."+c.Group, c.Group)),
			"modes", c.Modes,
			"conditions", conds,
			"params", params,
		))
	}
	return out
}

// categoryLabel is the category title; a mode may override it with `categories.<code>.mode_titles.<mode>`
// (CB-TEEN-04b: teen's «وزن و دمای پایه» is just «وزن» — BBT is a fertile-modes-only param).
func categoryLabel(code, mode string, l Labels) string {
	title := l.get("categories."+code+".title", code)
	if mode == "" {
		return title
	}
	return l.get("categories."+code+".mode_titles."+mode, title)
}

func paramJSON(c *Category, p *Param, mode string, l Labels) *jsonx.OrderedMap {
	base := "categories." + c.Code + ".params." + p.Code
	options := []*jsonx.OrderedMap{}
	for k := range p.Options {
		o := &p.Options[k]
		if mode != "" && !o.OptionAvailable(mode) {
			continue
		}
		options = append(options, jsonx.Obj(
			"value", o.Code,
			"label", l.get(base+".options."+o.Code, o.Code),
			"modes", o.Modes,
			"legacy_only", o.LegacyOnly,
		))
	}
	var unit any
	if p.Unit != "" {
		unit = jsonx.Obj("value", p.Unit, "label", l.get("units."+p.Unit, p.Unit))
	}
	var maxLen, scale, source any
	if p.MaxLen > 0 {
		maxLen = p.MaxLen
	}
	if p.Type == Number {
		scale = p.Scale
	}
	if p.Source != "" {
		source = p.Source
	}
	return jsonx.Obj(
		"code", p.Code,
		"type", string(p.Type),
		"label", l.get(base+".title", p.Code),
		"modes", c.ParamModes(p),
		"detail", p.Detail,
		"alert", p.Alert,
		"options", options,
		"levels", labeled(p.Levels, func(v string) string { return l.get("levels."+v, v) }),
		"score", rangeJSON(p.Score),
		"range", rangeJSON(p.Range),
		"scale", scale,
		"unit", unit,
		"max_length", maxLen,
		"dynamic", p.Dynamic,
		"source", source,
	)
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
