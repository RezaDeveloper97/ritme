// Package validation is a port of the parts of Laravel's Illuminate\Validation\Validator
// that the Ritme API uses, message for message: the same rules, the same error order
// (fields in rules-array order, wildcard expansions appended after the explicit fields,
// rules in declaration order), the same fa/en lines (resources/lang), the same
// :attribute humanisation and the same quirks (a non-numeric value under a numeric
// "min" is measured as a string, "before_or_equal:today" compares timestamps, …).
//
// Typical handler use:
//
//	data, err := validation.Input(c)                 // $request->all() after TrimStrings etc.
//	v := validation.Make(lang.Default(), locale, data, rules,
//		validation.Now(clk.Now()), validation.Messages("log_date.required", "Date is required"))
//	if v.Fails() {
//		return v.Errors()                            // framework 422 {message, errors}
//	}
//	attrs := v.Validated()
//
// Supported rules: accepted, array, after, after_or_equal, before, before_or_equal, bail,
// between, boolean, date, date_format, email (RFC approximation), filled, in, integer,
// list, max, min, not_in, not_regex, nullable, numeric, present, regex, required,
// required_if, size, sometimes, string. An unknown rule panics (a programming error).
package validation

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

var (
	implicitRules = []string{"Accepted", "AcceptedIf", "Declined", "DeclinedIf", "Filled", "Missing",
		"MissingIf", "MissingUnless", "MissingWith", "MissingWithAll", "Present", "PresentIf",
		"PresentUnless", "PresentWith", "PresentWithAll", "Required", "RequiredIf",
		"RequiredIfAccepted", "RequiredIfDeclined", "RequiredUnless", "RequiredWith",
		"RequiredWithAll", "RequiredWithout", "RequiredWithoutAll"}
	dependentRules = []string{"After", "AfterOrEqual", "Before", "BeforeOrEqual", "Confirmed",
		"Different", "Gt", "Gte", "Lt", "Lte", "AcceptedIf", "DeclinedIf", "RequiredIf",
		"RequiredUnless", "RequiredWith", "RequiredWithAll", "RequiredWithout",
		"RequiredWithoutAll", "Same"}
	sizeRules    = []string{"Size", "Between", "Min", "Max", "Gt", "Lt", "Gte", "Lte"}
	numericRules = []string{"Numeric", "Integer", "Decimal"}
)

// KV is one custom message or attribute name (the Laravel arrays are ordered, and the
// order decides between two wildcard keys that both match).
type KV struct{ Key, Value string }

// Option configures a Validator.
type Option func(*Validator)

// Messages sets the custom messages ($messages / FormRequest::messages()) as
// key/value pairs: "log_date.required", "Date is required", "moods.*.in", "…".
func Messages(kv ...string) Option {
	return func(v *Validator) { v.customMessages = append(v.customMessages, pairs(kv)...) }
}

// Attributes sets custom attribute names ($attributes / FormRequest::attributes()).
func Attributes(kv ...string) Option {
	return func(v *Validator) { v.customAttributes = append(v.customAttributes, pairs(kv)...) }
}

// Now sets the clock "today"/"now" and date parameters are evaluated against
// (the request clock; default: the real time in Asia/Tehran).
func Now(t time.Time) Option { return func(v *Validator) { v.now = t } }

func pairs(kv []string) []KV {
	if len(kv)%2 != 0 {
		panic("validation: odd number of key/value arguments")
	}
	out := make([]KV, 0, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		out = append(out, KV{kv[i], kv[i+1]})
	}
	return out
}

type attrRules struct {
	attr  string
	rules []Rule
}

// implicitAttr is one wildcard attribute and the keys it expanded to ($implicitAttributes).
type implicitAttr struct {
	primary string
	keys    []string
}

// Validator validates one data set against one rules array (Validator::make).
type Validator struct {
	tr               *lang.Translator
	locale           string
	now              time.Time
	data             any
	rules            []attrRules
	implicit         []implicitAttr
	customMessages   []KV
	customAttributes []KV

	ran    bool
	fields []string
	msgs   map[string][]string
	failed map[string]map[string]bool
}

// Make builds a validator for data ($request->all(): a phpval.Map, see Input) in locale.
func Make(tr *lang.Translator, locale string, data any, rules Rules, opts ...Option) *Validator {
	if data == nil {
		data = phpval.NewMap()
	}
	v := &Validator{tr: tr, locale: locale, data: data, now: time.Now().In(civildate.Tehran)}
	for _, o := range opts {
		o(v)
	}
	v.explode(rules)
	return v
}

// ---------------------------------------------------------------------------
// Rule explosion (ValidationRuleParser::explode)

func (v *Validator) explode(rules Rules) {
	type entry struct {
		attr  string
		rules []Rule
	}
	// $rules: the rules array being rewritten; originals: the foreach copy PHP iterates.
	var results []*entry
	index := map[string]int{}
	var originals []entry
	for _, f := range rules {
		if i, ok := index[f.Attr]; ok { // a repeated key: the later entry wins, first position kept
			results[i].rules = f.Rules
			originals[i].rules = f.Rules
			continue
		}
		index[f.Attr] = len(results)
		results = append(results, &entry{attr: f.Attr, rules: f.Rules})
		originals = append(originals, entry{attr: f.Attr, rules: f.Rules})
	}
	for _, orig := range originals {
		if !strings.Contains(orig.attr, "*") {
			// $rules[$key] = explodeExplicitRule($rule): back to the original rules, which
			// drops anything an earlier wildcard merged into this key.
			results[index[orig.attr]].rules = orig.rules
			continue
		}
		pattern := wildcardRegexp(orig.attr, `[^.]*`, true)
		keys, _ := gatherData(orig.attr, v.data)
		var expanded []string
		for _, key := range keys {
			if !strings.HasPrefix(key, orig.attr) && !pattern.MatchString(key) {
				continue
			}
			expanded = append(expanded, key)
			if i, ok := index[key]; ok && results[i] != nil { // mergeRules: append to the key
				results[i].rules = append(append([]Rule(nil), results[i].rules...), orig.rules...)
				continue
			}
			index[key] = len(results)
			results = append(results, &entry{attr: key, rules: orig.rules})
		}
		v.implicit = append(v.implicit, implicitAttr{primary: orig.attr, keys: expanded})
		results[index[orig.attr]] = nil // unset($rules[$key])
		delete(index, orig.attr)
	}
	for _, e := range results {
		if e != nil {
			v.rules = append(v.rules, attrRules{attr: e.attr, rules: e.rules})
		}
	}
}

// primaryAttribute is getPrimaryAttribute: "moods.0" → "moods.*".
func (v *Validator) primaryAttribute(attr string) string {
	for _, ia := range v.implicit {
		if slices.Contains(ia.keys, attr) {
			return ia.primary
		}
	}
	return attr
}

func (v *Validator) isImplicitAttribute(primary string) bool {
	for _, ia := range v.implicit {
		if ia.primary == primary {
			return true
		}
	}
	return false
}

// wildcardRegexp quotes attr and turns every "*" into repl; anchored at the end when full.
func wildcardRegexp(attr, repl string, full bool) *regexp.Regexp {
	parts := strings.Split(attr, "*")
	for i, p := range parts {
		parts[i] = regexp.QuoteMeta(p)
	}
	expr := "^" + strings.Join(parts, repl)
	if full {
		expr += `\z`
	}
	return regexp.MustCompile(expr)
}

// gatherData is ValidationData::initializeAndGatherData: the dotted data under the
// attribute's explicit prefix, wildcard paths filled with null, then the real values of
// every wildcard match merged in (array_merge: existing keys keep their place).
func gatherData(attr string, master any) ([]string, []any) {
	explicit := strings.TrimRight(strings.SplitN(attr, "*", 2)[0], ".")
	var data any
	if explicit == "" {
		data = phpval.Clone(master)
	} else {
		data = phpval.NewMap()
		if val, ok := phpval.Get(master, explicit); ok {
			data = arrSet(data, strings.Split(explicit, "."), phpval.Clone(val))
		}
	}
	if strings.Contains(attr, "*") && !strings.HasSuffix(attr, "*") {
		data = dataSet(data, strings.Split(attr, "."), nil)
	}
	keys, vals := phpval.Dot(data)

	prefix := wildcardRegexp(attr, `[^.]+`, false)
	var wild []string
	for _, k := range keys {
		if m := prefix.FindString(k); m != "" && !slices.Contains(wild, m) {
			wild = append(wild, m)
		}
	}
	for _, w := range wild {
		val, _ := phpval.Get(master, w)
		if i := slices.Index(keys, w); i >= 0 {
			vals[i] = val
			continue
		}
		keys = append(keys, w)
		vals = append(vals, val)
	}
	return keys, vals
}

// arrSet is Arr::set($array, $path, $value): creates nested arrays along the path.
func arrSet(target any, segs []string, value any) any {
	if len(segs) == 0 {
		return value
	}
	cur, ok := childOf(target, segs[0])
	if !ok || !phpval.IsArray(cur) {
		cur = phpval.NewMap()
	}
	if len(segs) == 1 {
		return setChild(target, segs[0], value)
	}
	return setChild(target, segs[0], arrSet(cur, segs[1:], value))
}

// dataSet is data_set($target, $segments, $value, overwrite: true).
func dataSet(target any, segs []string, value any) any {
	seg, rest := segs[0], segs[1:]
	switch {
	case seg == "*":
		if !phpval.IsArray(target) {
			target = phpval.NewMap()
		}
		keys, vals := phpval.Entries(target)
		for i, k := range keys {
			if len(rest) > 0 {
				target = setChild(target, k, dataSet(vals[i], rest, value))
			} else {
				target = setChild(target, k, value)
			}
		}
		return target
	case phpval.IsArray(target):
		if len(rest) > 0 {
			cur, ok := childOf(target, seg)
			if !ok {
				cur = phpval.NewMap()
			}
			return setChild(target, seg, dataSet(cur, rest, value))
		}
		return setChild(target, seg, value)
	default:
		target = phpval.NewMap()
		if len(rest) > 0 {
			return setChild(target, seg, dataSet(nil, rest, value))
		}
		return setChild(target, seg, value)
	}
}

func childOf(target any, key string) (any, bool) { return phpval.Get(target, key) }

// setChild sets $target[$key] = $value on a list or map; a list gaining a non-index key
// becomes a map (PHP arrays are both).
func setChild(target any, key string, value any) any {
	switch a := target.(type) {
	case []any:
		if i, err := strconv.Atoi(key); err == nil && strconv.Itoa(i) == key && i >= 0 {
			if i < len(a) {
				a[i] = value
				return a
			}
			if i == len(a) {
				return append(a, value)
			}
		}
		m := phpval.NewMap()
		for i, x := range a {
			m.Set(strconv.Itoa(i), x)
		}
		m.Set(key, value)
		return m
	case phpval.Map:
		a.Set(key, value)
		return a
	}
	m := phpval.NewMap()
	m.Set(key, value)
	return m
}

// ---------------------------------------------------------------------------
// Running (Validator::passes)

// Fails runs the rules (once) and reports whether any failed.
func (v *Validator) Fails() bool { return !v.Passes() }

// Passes runs the rules (once) and reports whether all passed.
func (v *Validator) Passes() bool {
	if !v.ran {
		v.run()
	}
	return len(v.fields) == 0
}

func (v *Validator) run() {
	v.ran = true
	v.msgs = map[string][]string{}
	v.failed = map[string]map[string]bool{}
	for _, ar := range v.rules {
		for _, r := range ar.rules {
			v.validateAttribute(ar.attr, r)
			if v.shouldStopValidating(ar.attr) {
				break
			}
		}
	}
}

func (v *Validator) validateAttribute(attr string, r Rule) {
	if r.Name == "" {
		return
	}
	params := r.Params
	if slices.Contains(dependentRules, r.Name) {
		if keys := v.explicitKeys(attr); len(keys) > 0 {
			params = replaceAsterisks(params, keys)
		}
	}
	value, _ := phpval.Get(v.data, attr)
	if !v.isValidatable(r.Name, attr, value) {
		return
	}
	if !v.check(r, attr, value, params) {
		v.addFailure(attr, r.Name, params)
	}
}

// explicitKeys is getExplicitKeys: "foo.1.bar" for "foo.*.bar" → ["1"].
func (v *Validator) explicitKeys(attr string) []string {
	primary := v.primaryAttribute(attr)
	if primary == attr {
		return nil
	}
	m := wildcardRegexp(primary, `([^.]+)`, false).FindStringSubmatch(attr)
	if len(m) < 2 {
		return nil
	}
	return m[1:]
}

// replaceAsterisks is vsprintf(str_replace('*', '%s', $param), $keys).
func replaceAsterisks(params, keys []string) []string {
	out := make([]string, len(params))
	for i, p := range params {
		parts := strings.Split(p, "*")
		var b strings.Builder
		for j, part := range parts {
			if j > 0 && j-1 < len(keys) {
				b.WriteString(keys[j-1])
			}
			b.WriteString(part)
		}
		out[i] = b.String()
	}
	return out
}

func (v *Validator) isValidatable(rule, attr string, value any) bool {
	return v.presentOrRuleIsImplicit(rule, attr, value) &&
		v.passesOptionalCheck(attr) &&
		v.isNotNullIfMarkedAsNullable(rule, attr)
}

func (v *Validator) presentOrRuleIsImplicit(rule, attr string, value any) bool {
	if s, ok := value.(string); ok && phpTrim(s) == "" {
		return isImplicit(rule)
	}
	return phpval.Has(v.data, attr) || isImplicit(rule)
}

func isImplicit(rule string) bool { return slices.Contains(implicitRules, rule) }

func (v *Validator) passesOptionalCheck(attr string) bool {
	if !v.hasRule(attr, "Sometimes") {
		return true
	}
	return phpval.Has(v.data, attr)
}

func (v *Validator) isNotNullIfMarkedAsNullable(rule, attr string) bool {
	if isImplicit(rule) || !v.hasRule(attr, "Nullable") {
		return true
	}
	val, ok := phpval.Get(v.data, attr)
	return !ok || val != nil
}

func (v *Validator) shouldStopValidating(attr string) bool {
	if v.hasRule(attr, "Bail") {
		return len(v.msgs[attr]) > 0
	}
	if !v.hasRule(attr, implicitRules...) {
		return false
	}
	for r := range v.failed[attr] {
		if isImplicit(r) {
			return true
		}
	}
	return false
}

// hasRule reports whether attr carries any of names.
func (v *Validator) hasRule(attr string, names ...string) bool {
	_, ok := v.getRule(attr, names...)
	return ok
}

func (v *Validator) getRule(attr string, names ...string) (Rule, bool) {
	for _, ar := range v.rules {
		if ar.attr != attr {
			continue
		}
		for _, r := range ar.rules {
			if slices.Contains(names, r.Name) {
				return r, true
			}
		}
	}
	return Rule{}, false
}

func (v *Validator) addFailure(attr, rule string, params []string) {
	msg := v.makeReplacements(v.getMessage(attr, rule), attr, rule, params)
	if _, ok := v.msgs[attr]; !ok {
		v.fields = append(v.fields, attr)
	}
	if !slices.Contains(v.msgs[attr], msg) { // MessageBag::add keeps messages unique per key
		v.msgs[attr] = append(v.msgs[attr], msg)
	}
	if v.failed[attr] == nil {
		v.failed[attr] = map[string]bool{}
	}
	v.failed[attr][rule] = true
}

// ---------------------------------------------------------------------------
// Results

// Errors returns the failures as the framework 422 error (nil when validation passed).
func (v *Validator) Errors() *httpx.ValidationError {
	if v.Passes() {
		return nil
	}
	e := httpx.NewValidationError()
	for _, f := range v.fields {
		for _, m := range v.msgs[f] {
			e.Add(f, m)
		}
	}
	return e
}

// ErrorBag is $validator->errors() as JSON ({field: [messages…]}), for controllers that
// wrap it in their own envelope ({success:false, message, errors}).
func (v *Validator) ErrorBag() *jsonx.OrderedMap {
	v.Passes()
	bag := jsonx.NewArray()
	for _, f := range v.fields {
		bag.Set(f, v.msgs[f])
	}
	return bag
}

// First is $validator->errors()->first(): the first message, or "".
func (v *Validator) First() string {
	v.Passes()
	if len(v.fields) == 0 {
		return ""
	}
	return v.msgs[v.fields[0]][0]
}

// Fields returns the failed attributes in order.
func (v *Validator) Fields() []string {
	v.Passes()
	return append([]string(nil), v.fields...)
}

// Messages returns the messages for attr.
func (v *Validator) Messages(attr string) []string {
	v.Passes()
	return append([]string(nil), v.msgs[attr]...)
}

// Validated is $validator->validated() for a passing validator: the input restricted to
// the ruled attributes, in rule order (an array with ruled children is rebuilt from
// those children only). It returns nil when validation failed.
func (v *Validator) Validated() phpval.Map {
	if !v.Passes() {
		return nil
	}
	results := phpval.NewMap()
	var out any = results
	for _, ar := range v.rules {
		value, ok := phpval.Get(v.data, ar.attr)
		if hasArrayRule(ar.rules) && value != nil && v.hasChildRules(ar.attr) {
			continue
		}
		if ok {
			out = arrSet(out, strings.Split(ar.attr, "."), value)
		}
	}
	m, ok := out.(phpval.Map)
	if !ok {
		return results
	}
	packed := phpval.NewMap() // keep the top level a map; pack nested sequential arrays
	for _, k := range m.Keys() {
		x, _ := m.Get(k)
		packed.Set(k, phpval.Packed(x))
	}
	return packed
}

func hasArrayRule(rs []Rule) bool {
	for _, r := range rs {
		if (r.Name == "Array" || r.Name == "List") && len(r.Params) == 0 {
			return true
		}
	}
	return false
}

func (v *Validator) hasChildRules(attr string) bool {
	re := regexp.MustCompile("^" + regexp.QuoteMeta(attr) + `\.+`)
	for _, ar := range v.rules {
		if re.MatchString(ar.attr) {
			return true
		}
	}
	return false
}
