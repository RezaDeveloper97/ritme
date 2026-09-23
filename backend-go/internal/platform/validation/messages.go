package validation

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// getMessage is FormatsMessages::getMessage: the inline (custom) message, then
// validation.custom.<attr>.<rule>, then the size line for the attribute type, then
// validation.<rule> — each looked up in the request locale with the "en" fallback.
func (v *Validator) getMessage(attr, rule string) string {
	lowerRule := snake(rule)
	isSize := slices.Contains(sizeRules, rule)
	if msg, ok := v.fromLocalArray(attr, lowerRule, v.customMessages); ok {
		return msg
	}
	customKey := "validation.custom." + attr + "." + lowerRule
	keys := []string{customKey}
	if isSize {
		keys = []string{customKey + "." + v.attributeType(attr), customKey}
	}
	if msg, ok := v.customMessageFromTranslator(keys); ok {
		return msg
	}
	if isSize {
		return v.transString("validation." + lowerRule + "." + v.attributeType(attr))
	}
	return v.transString("validation." + lowerRule)
}

// transString is $translator->get($key) for a string line (the key when missing).
func (v *Validator) transString(key string) string {
	if line, ok := v.tr.Get(key, v.locale); ok {
		if s, isStr := line.(string); isStr {
			return s
		}
	}
	return key
}

// fromLocalArray is getFromLocalArray: keys "<attr>.<rule>", "<rule>", "<attr>" are
// tried in turn against every source key (source keys with "*" match one segment each).
func (v *Validator) fromLocalArray(attr, lowerRule string, source []KV) (string, bool) {
	for _, key := range []string{attr + "." + lowerRule, lowerRule, attr} {
		for _, kv := range source {
			if strings.Contains(kv.Key, "*") {
				if wildcardRegexp(kv.Key, `([^.]*)`, true).MatchString(key) {
					return kv.Value, true
				}
				continue
			}
			if kv.Key == key {
				return kv.Value, true
			}
		}
	}
	return "", false
}

// customMessageFromTranslator is getCustomMessageFromTranslator: the exact
// validation.custom key, else a wildcard key of validation.custom (Str::is).
func (v *Validator) customMessageFromTranslator(keys []string) (string, bool) {
	for _, key := range keys {
		if line, ok := v.tr.Get(key, v.locale); ok {
			if s, isStr := line.(string); isStr {
				return s, true
			}
		}
		short := strings.TrimPrefix(key, "validation.custom.")
		custom, _ := v.tr.Get("validation.custom", v.locale)
		ks, vals := phpval.Dot(custom)
		for i, k := range ks {
			if short == k || (strings.Contains(k, "*") && strIs(k, short)) {
				if s, isStr := vals[i].(string); isStr {
					return s, true
				}
			}
		}
	}
	return "", false
}

// strIs is Str::is($pattern, $value) with "*" matching anything.
func strIs(pattern, value string) bool {
	if pattern == value {
		return true
	}
	parts := strings.Split(pattern, "*")
	for i, p := range parts {
		parts[i] = regexp.QuoteMeta(p)
	}
	return regexp.MustCompile(`^` + strings.Join(parts, ".*") + `\z`).MatchString(value)
}

// attributeType is getAttributeType: numeric / array / string (no uploads in the API).
func (v *Validator) attributeType(attr string) string {
	switch {
	case v.hasRule(attr, numericRules...):
		return "numeric"
	case v.hasRule(attr, "Array", "List"):
		return "array"
	}
	return "string"
}

// ---------------------------------------------------------------------------
// Placeholders (FormatsMessages::makeReplacements + ReplacesAttributes)

func (v *Validator) makeReplacements(msg, attr, rule string, params []string) string {
	msg = replaceCase(msg, "attribute", v.displayableAttribute(attr))
	msg = v.replaceInput(msg, attr)
	msg = replaceIndexOrPosition(msg, attr, "index", 0)
	msg = replaceIndexOrPosition(msg, attr, "position", 1)

	switch rule {
	case "Between":
		msg = strings.ReplaceAll(msg, ":min", param(params, 0))
		msg = strings.ReplaceAll(msg, ":max", param(params, 1))
	case "DateFormat":
		msg = strings.ReplaceAll(msg, ":format", param(params, 0))
	case "In", "NotIn":
		shown := make([]string, len(params))
		ucShown := make([]string, len(params))
		for i, p := range params {
			shown[i] = v.displayableValue(attr, p)
			ucShown[i] = lang.UcFirst(shown[i])
		}
		joined := strings.Join(shown, ", ")
		msg = strings.ReplaceAll(msg, ":values", joined)
		msg = strings.ReplaceAll(msg, ":VALUES", strings.ToUpper(joined))
		msg = strings.ReplaceAll(msg, ":Values", strings.Join(ucShown, ", "))
	case "Min":
		msg = strings.ReplaceAll(msg, ":min", param(params, 0))
	case "Max":
		msg = strings.ReplaceAll(msg, ":max", param(params, 0))
	case "Size":
		msg = strings.ReplaceAll(msg, ":size", param(params, 0))
	case "RequiredIf", "AcceptedIf":
		otherValue, _ := phpval.Get(v.data, param(params, 0))
		value := v.displayableValue(param(params, 0), otherValue)
		other := v.displayableAttribute(param(params, 0))
		msg = replaceCase(msg, "other", other)
		msg = replaceCase(msg, "value", value)
	case "Before", "BeforeOrEqual", "After", "AfterOrEqual":
		p := param(params, 0)
		if v.strtotime(p) {
			msg = strings.ReplaceAll(msg, ":date", v.displayableValue(attr, p))
		} else {
			msg = strings.ReplaceAll(msg, ":date", v.displayableAttribute(p))
		}
	}
	return msg
}

func param(params []string, i int) string {
	if i < len(params) {
		return params[i]
	}
	return ""
}

// replaceCase replaces :name, :NAME and :Name (in that order, like str_replace with arrays).
func replaceCase(msg, name, value string) string {
	msg = strings.ReplaceAll(msg, ":"+name, value)
	msg = strings.ReplaceAll(msg, ":"+strings.ToUpper(name), strings.ToUpper(value))
	return strings.ReplaceAll(msg, ":"+lang.UcFirst(name), lang.UcFirst(value))
}

// strtotime reports whether strtotime($s) is truthy (a parseable, non-epoch date).
func (v *Validator) strtotime(s string) bool {
	t, err := civildate.ParseLenient(s, v.now, civildate.Tehran)
	return err == nil && t.Unix() != 0
}

func (v *Validator) replaceInput(msg, attr string) string {
	if !strings.Contains(msg, ":input") {
		return msg
	}
	value, _ := phpval.Get(v.data, attr)
	if phpval.IsArray(value) {
		return msg
	}
	return strings.ReplaceAll(msg, ":input", v.displayableValue(attr, value))
}

var positionWords = []string{"", "first", "second", "third", "fourth", "fifth", "sixth",
	"seventh", "eighth", "ninth", "tenth"}

// replaceIndexOrPosition handles :index / :position (and :first-index, :second-position …)
// for numeric segments of a wildcard key such as "moods.2"; offset is 1 for positions.
func replaceIndexOrPosition(msg, attr, placeholder string, offset int) string {
	if !strings.Contains(strings.ToLower(msg), ":") {
		return msg
	}
	n := 1
	for _, seg := range strings.Split(attr, ".") {
		if !phpval.IsNumericString(seg) {
			continue
		}
		val := strconv.Itoa(int(phpval.ToFloat(seg)) + offset)
		if n == 1 {
			msg = replaceInsensitive(msg, ":"+placeholder, val)
		}
		word := "other"
		if n < len(positionWords) {
			word = positionWords[n]
		}
		msg = replaceInsensitive(msg, ":"+word+"-"+placeholder, val)
		n++
	}
	return msg
}

// replaceInsensitive is str_ireplace for an ASCII search string.
func replaceInsensitive(s, search, repl string) string {
	re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(search))
	return re.ReplaceAllLiteralString(s, repl)
}

// displayableAttribute is getDisplayableAttribute: custom attribute names, then
// validation.attributes (both with "*" keys), then the raw key for wildcard expansions,
// else snake_case with spaces ("log_date" → "log date").
func (v *Validator) displayableAttribute(attr string) string {
	primary := v.primaryAttribute(attr)
	names := []string{attr}
	if primary != attr {
		names = append(names, primary)
	}
	for _, name := range names {
		if s, ok := attributeFromList(name, v.customAttributes); ok && s != "" {
			return s
		}
		if line, ok := v.tr.Get("validation.attributes", v.locale); ok && phpval.IsArray(line) {
			ks, vals := phpval.Dot(line)
			list := make([]KV, 0, len(ks))
			for i, k := range ks {
				list = append(list, KV{k, phpval.ToString(vals[i])})
			}
			if len(list) == 0 {
				list = v.customAttributes
			}
			if s, ok := attributeFromList(name, list); ok && s != "" {
				return s
			}
		}
	}
	if v.isImplicitAttribute(primary) {
		return attr
	}
	return strings.ReplaceAll(snake(attr), "_", " ")
}

func attributeFromList(attr string, source []KV) (string, bool) {
	for _, kv := range source {
		if kv.Key == attr {
			return kv.Value, true
		}
	}
	for _, kv := range source {
		if strings.Contains(kv.Key, "*") && wildcardRegexp(kv.Key, `([^.]*)`, true).MatchString(attr) {
			return kv.Value, true
		}
	}
	return "", false
}

// displayableValue is getDisplayableValue.
func (v *Validator) displayableValue(attr string, value any) string {
	if phpval.IsArray(value) {
		return "array"
	}
	key := "validation.values." + attr + "." + phpval.ToString(value)
	if line, ok := v.tr.Get(key, v.locale); ok {
		if s, isStr := line.(string); isStr {
			return s
		}
	}
	switch x := value.(type) {
	case bool:
		if x {
			return "true"
		}
		return "false"
	case nil:
		return "empty"
	}
	return phpval.ToString(value)
}
