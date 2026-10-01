package taxonomy

import (
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Change is the new state of one param of a day: Entries replace every stored row of the param (empty =
// the param is cleared).
type Change struct {
	Category, Param string
	Entries         []Entry
	// Source is the health_log_entries.source of the new rows ("" = manual; "voice" for params the client
	// took from a voice-log suggestion, B-N3-05).
	Source string
}

// Key is "cat.param".
func (c Change) Key() string { return c.Category + "." + c.Param }

// dynamicItem is the shape of a free item code (custom items, care reminder ids).
var dynamicItem = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// CustomItems are a user's active custom items (B-N3-02) as "category.param.code" slots: the only
// non-taxonomy item codes a Custom param accepts for new input.
type CustomItems map[string]bool

// Has reports whether code is an active custom item of cat.param.
func (c CustomItems) Has(cat, param, code string) bool { return c[cat+"."+param+"."+code] }

// Add registers an active custom item.
func (c CustomItems) Add(cat, param, code string) { c[cat+"."+param+"."+code] = true }

// parser validates a PUT /logs/days/{date} body.
type parser struct {
	custom   CustomItems
	mode     string
	locale   string
	existing map[string]Entry // slot → stored entry (unchanged values always pass)
	errs     *httpx.ValidationError
	changes  []Change
}

// Parse validates body ({"categories": {cat: {param: value|null} | null}}) for a user in mode, against the
// day's stored entries, and returns the per-param changes in body order. A category set to null clears
// every storable param of it; a param set to null clears it; params not sent are left alone.
//
// Rules: categories, params, options, items and levels must exist; a new value must be available in the
// user's mode (legacy-only values never are), but a value equal to the stored one always passes, so a
// re-save of a legacy day (or of a day logged in another mode) never fails; numbers stay inside the
// param's range; texts inside their length; link params are read-only. Custom params (B-N3-02) take,
// besides their options, only the user's active custom items (custom); a deleted custom item stays valid
// where it is stored.
func Parse(body phpval.Map, mode, locale string, existing []Entry, custom CustomItems) ([]Change, error) {
	p := &parser{custom: custom, mode: mode, locale: locale, existing: map[string]Entry{}, errs: httpx.NewValidationError()}
	for _, e := range existing {
		p.existing[e.Slot()] = e
	}
	raw, ok := body.Get("categories")
	cats, isMap := raw.(phpval.Map)
	switch {
	case !ok || raw == nil:
		p.fail("categories", "validation.required", nil)
	case !isMap:
		p.fail("categories", "validation.array", nil)
	default:
		for _, code := range cats.Keys() {
			v, _ := cats.Get(code)
			p.category(code, v)
		}
	}
	if !p.errs.Empty() {
		return nil, p.errs
	}
	return p.changes, nil
}

func (p *parser) fail(field, key string, params map[string]string) {
	all := map[string]string{"attribute": strings.ReplaceAll(field, "_", " ")}
	for k, v := range params {
		all[k] = v
	}
	p.errs.Add(field, lang.Default().Trans(key, all, p.locale))
}

func (p *parser) category(code string, v any) {
	field := "categories." + code
	cat, ok := CategoryByCode(code)
	if !ok {
		p.fail(field, "validation.in", nil)
		return
	}
	if v == nil {
		for i := range cat.Params {
			if cat.Params[i].Storable() {
				p.changes = append(p.changes, Change{Category: code, Param: cat.Params[i].Code})
			}
		}
		return
	}
	params, isMap := v.(phpval.Map)
	if !isMap {
		p.fail(field, "validation.array", nil)
		return
	}
	for _, pc := range params.Keys() {
		pv, _ := params.Get(pc)
		p.param(cat, pc, pv)
	}
}

func (p *parser) param(cat *Category, code string, v any) {
	field := "categories." + cat.Code + "." + code
	par, ok := cat.Param(code)
	if !ok || !par.Storable() {
		p.fail(field, "validation.in", nil)
		return
	}
	ch := Change{Category: cat.Code, Param: code}
	if v == nil {
		p.changes = append(p.changes, ch)
		return
	}
	base := Entry{Category: cat.Code, Param: code}
	// a whole new value must be available in the mode unless it is the stored one
	paramOK := cat.Available(par, p.mode)
	switch par.Type {
	case Single:
		s, isStr := v.(string)
		if !isStr {
			p.fail(field, "validation.string", nil)
			break
		}
		if !p.codeOK(cat, par, paramOK, "", s) {
			p.fail(field, "validation.in", nil)
			break
		}
		e := base
		e.Code = str(s)
		ch.Entries = append(ch.Entries, e)
	case Multi:
		list, isList := v.([]any)
		if !isList {
			p.fail(field, "validation.array", nil)
			break
		}
		seen := map[string]bool{}
		for i, it := range list {
			f := field + "." + strconv.Itoa(i)
			s, isStr := it.(string)
			switch {
			case !isStr:
				p.fail(f, "validation.string", nil)
			case seen[s]:
				p.fail(f, "validation.distinct", nil)
			case !p.codeOK(cat, par, paramOK, s, s):
				p.fail(f, "validation.in", nil)
			default:
				seen[s] = true
				e := base
				e.Item, e.Code = s, str(Yes)
				ch.Entries = append(ch.Entries, e)
			}
		}
	case Items:
		m, isMap := v.(phpval.Map)
		if !isMap {
			p.fail(field, "validation.array", nil)
			break
		}
		for _, item := range m.Keys() {
			iv, _ := m.Get(item)
			if e, ok := p.itemLevel(cat, par, paramOK, field+"."+item, item, iv); ok {
				ch.Entries = append(ch.Entries, e)
			}
		}
	case Number, Integer:
		num, ok := p.number(field, par, v)
		if !ok {
			break
		}
		if !paramOK && !p.sameNum(base, num) {
			p.fail(field, "validation.in", nil)
			break
		}
		e := base
		e.Num = str(num)
		ch.Entries = append(ch.Entries, e)
	case Text:
		s, isStr := v.(string)
		if !isStr {
			p.fail(field, "validation.string", nil)
			break
		}
		if utf8.RuneCountInString(s) > par.MaxLen {
			p.fail(field, "validation.max.string", map[string]string{"max": strconv.Itoa(par.MaxLen)})
			break
		}
		e := base
		e.Text = str(s)
		ch.Entries = append(ch.Entries, e)
	case TextItems:
		m, isMap := v.(phpval.Map)
		if !isMap {
			p.fail(field, "validation.array", nil)
			break
		}
		for _, item := range m.Keys() {
			iv, _ := m.Get(item)
			f := field + "." + item
			if !p.itemOK(cat, par, paramOK, item) {
				p.fail(f, "validation.in", nil)
				continue
			}
			if iv == nil {
				continue
			}
			s, isStr := iv.(string)
			if !isStr {
				p.fail(f, "validation.string", nil)
				continue
			}
			if utf8.RuneCountInString(s) > par.MaxLen {
				p.fail(f, "validation.max.string", map[string]string{"max": strconv.Itoa(par.MaxLen)})
				continue
			}
			e := base
			e.Item, e.Text = item, str(s)
			ch.Entries = append(ch.Entries, e)
		}
	case Bool:
		b, isBool := v.(bool)
		if !isBool {
			p.fail(field, "validation.boolean", nil)
			break
		}
		code := No
		if b {
			code = Yes
		}
		if !paramOK && !p.sameCode("", base, code) {
			p.fail(field, "validation.in", nil)
			break
		}
		e := base
		e.Code = str(code)
		ch.Entries = append(ch.Entries, e)
	}
	p.changes = append(p.changes, ch) // discarded by Parse when anything failed
}

// itemLevel parses one item of an items param: {"level": code, "score": n} or the bare level code.
func (p *parser) itemLevel(cat *Category, par *Param, paramOK bool, field, item string, v any) (Entry, bool) {
	e := Entry{Category: cat.Code, Param: par.Code, Item: item}
	if !p.itemOK(cat, par, paramOK, item) {
		p.fail(field, "validation.in", nil)
		return e, false
	}
	var levelV, scoreV any
	switch x := v.(type) {
	case nil:
		return e, false // an item set to null is simply not logged
	case string:
		levelV = x
	case phpval.Map:
		levelV, _ = x.Get("level")
		scoreV, _ = x.Get("score")
	default:
		p.fail(field, "validation.array", nil)
		return e, false
	}
	level, isStr := levelV.(string)
	if !isStr {
		p.fail(field+".level", "validation.required", nil)
		return e, false
	}
	stored, had := p.existing[e.Slot()]
	if known := slices.Contains(par.Levels, level) || (had && stored.Code.String == level); !known {
		p.fail(field+".level", "validation.in", nil)
		return e, false
	}
	e.Code = str(level)
	if scoreV != nil {
		if par.Score == nil {
			p.fail(field+".score", "validation.prohibited", nil)
			return e, false
		}
		n, isInt := intValue(scoreV)
		if !isInt {
			p.fail(field+".score", "validation.integer", nil)
			return e, false
		}
		if float64(n) < par.Score.Min || float64(n) > par.Score.Max {
			p.fail(field+".score", "validation.between.numeric", rangeParams(par.Score))
			return e, false
		}
		e.Num = str(strconv.FormatInt(n, 10) + ".00")
	}
	return e, true
}

// itemOK: a known item available in the mode, one of the user's custom items for custom params, a free
// code for (other) dynamic params, or the stored item.
func (p *parser) itemOK(cat *Category, par *Param, paramOK bool, item string) bool {
	if _, had := p.existing[cat.Code+"."+par.Code+"."+item]; had {
		return true
	}
	if !paramOK {
		return false
	}
	if o, ok := par.Option(item); ok {
		return o.OptionAvailable(p.mode)
	}
	if par.Custom {
		return p.custom.Has(cat.Code, par.Code, item)
	}
	return par.Dynamic && dynamicItem.MatchString(item)
}

// codeOK checks an option code of a single (item "") or multi (item = code) param.
func (p *parser) codeOK(cat *Category, par *Param, paramOK bool, item, code string) bool {
	base := Entry{Category: cat.Code, Param: par.Code}
	if p.sameCode(item, base, code) {
		return true
	}
	if !paramOK {
		return false
	}
	o, ok := par.Option(code)
	if !ok {
		return par.Custom && item != "" && p.custom.Has(cat.Code, par.Code, code)
	}
	return o.OptionAvailable(p.mode)
}

func (p *parser) sameCode(item string, base Entry, code string) bool {
	base.Item = item
	stored, had := p.existing[base.Slot()]
	if !had {
		return false
	}
	if item != "" { // multi: the item itself is the code
		return true
	}
	return stored.Code.Valid && stored.Code.String == code
}

func (p *parser) sameNum(base Entry, num string) bool {
	stored, had := p.existing[base.Slot()]
	return had && stored.Num.Valid && stored.Num.String == num
}

// number validates a number / integer param and returns it as decimal(8,2) text.
func (p *parser) number(field string, par *Param, v any) (string, bool) {
	var f float64
	if par.Type == Integer {
		n, ok := intValue(v)
		if !ok {
			p.fail(field, "validation.integer", nil)
			return "", false
		}
		f = float64(n)
	} else {
		if !phpval.IsNumeric(v) {
			p.fail(field, "validation.numeric", nil)
			return "", false
		}
		f = phpval.ToFloat(v)
	}
	if par.Range != nil && (f < par.Range.Min || f > par.Range.Max) {
		p.fail(field, "validation.between.numeric", rangeParams(par.Range))
		return "", false
	}
	scale := math.Pow(10, float64(par.Scale))
	f = math.Round(f*scale) / scale
	return strconv.FormatFloat(f, 'f', 2, 64), true
}

func rangeParams(r *Range) map[string]string {
	return map[string]string{
		"min": strconv.FormatFloat(r.Min, 'f', -1, 64),
		"max": strconv.FormatFloat(r.Max, 'f', -1, 64),
	}
}

// intValue accepts an int, an integral float or an integer string.
func intValue(v any) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case float64:
		if x == math.Trunc(x) && math.Abs(x) < 1e15 {
			return int64(x), true
		}
	case string:
		if n, err := strconv.ParseInt(x, 10, 64); err == nil {
			return n, true
		}
	}
	return 0, false
}
