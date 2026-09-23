package validation

import (
	"fmt"
	"regexp"
	"strings"
)

// Rule is one parsed Laravel rule: Name is Laravel's StudlyCase method name
// ("Required", "BeforeOrEqual"), Params its parameters.
type Rule struct {
	Name   string
	Params []string
	re     *regexp.Regexp // compiled pattern for Regex / NotRegex
}

// String renders the rule back in Laravel syntax ("before_or_equal:today").
func (r Rule) String() string {
	s := snake(r.Name)
	if len(r.Params) > 0 {
		s += ":" + strings.Join(r.Params, ",")
	}
	return s
}

// Field is one entry of a rules array: an attribute (dot path, `*` wildcards allowed)
// and its rules in order.
type Field struct {
	Attr  string
	Rules []Rule
}

// Rules is a Laravel rules array; the order of the fields is the order errors come out in.
type Rules []Field

// F builds a Field the way Laravel reads a rules-array entry. Each argument is either a
// Rule (In(…), Regex(…)) or a string in Laravel syntax; strings are split on "|"
// ("nullable|integer|min:1"), except a "regex:"/"not_regex:" rule, which is kept whole
// (Laravel requires the array form for those, and so do we).
//
//	validation.F("log_date", "required|date|before_or_equal:today")
//	validation.F("type", "required", validation.In(enums.ReminderTypeValues()...))
func F(attr string, rules ...any) Field {
	f := Field{Attr: attr}
	for _, r := range rules {
		switch x := r.(type) {
		case Rule:
			f.Rules = append(f.Rules, x)
		case []Rule:
			f.Rules = append(f.Rules, x...)
		case string:
			lower := strings.ToLower(strings.TrimSpace(x))
			if strings.HasPrefix(lower, "regex:") || strings.HasPrefix(lower, "not_regex:") {
				f.Rules = append(f.Rules, ParseRule(x))
				continue
			}
			f.Rules = append(f.Rules, Parse(x)...)
		case []string:
			for _, s := range x {
				f.Rules = append(f.Rules, ParseRule(s))
			}
		default:
			panic(fmt.Sprintf("validation.F(%q): unsupported rule %T", attr, r))
		}
	}
	return f
}

// Parse splits a pipe-delimited rule string ("nullable|integer|min:1").
func Parse(spec string) []Rule {
	var out []Rule
	for _, part := range strings.Split(spec, "|") {
		out = append(out, ParseRule(part))
	}
	return out
}

// ParseRule is ValidationRuleParser::parseStringRule for a single rule ("min:1",
// `in:"a","b"`, "regex:/^x$/"): the name is studly-cased, parameters are CSV
// (str_getcsv), except for regex rules whose parameter is taken whole.
func ParseRule(s string) Rule {
	name, param, hasParam := strings.Cut(s, ":")
	r := Rule{Name: normalizeName(studly(strings.TrimSpace(name)))}
	if hasParam {
		if r.Name == "Regex" || r.Name == "NotRegex" {
			r.Params = []string{param}
			r.re = mustCompilePHPRegex(param)
		} else {
			r.Params = strGetCSV(param)
		}
	}
	return r
}

// In is Rule::in($values): the value must equal one of values (compared as strings).
func In(values ...string) Rule { return Rule{Name: "In", Params: append([]string(nil), values...)} }

// NotIn is Rule::notIn($values).
func NotIn(values ...string) Rule {
	return Rule{Name: "NotIn", Params: append([]string(nil), values...)}
}

// Regex is 'regex:<php pattern>' with a PCRE-delimited pattern such as `/^09[0-9]{9}$/`.
func Regex(phpPattern string) Rule { return ParseRule("regex:" + phpPattern) }

func normalizeName(n string) string {
	switch n {
	case "Int":
		return "Integer"
	case "Bool":
		return "Boolean"
	}
	return n
}

// studly is Str::studly: words split on "-", "_" and spaces, each upper-cased first.
func studly(s string) string {
	s = strings.NewReplacer("-", " ", "_", " ").Replace(s)
	var b strings.Builder
	for _, w := range strings.Fields(s) {
		b.WriteString(strings.ToUpper(w[:1]) + w[1:])
	}
	return b.String()
}

// snake is Str::snake($value) with "_": ucwords, drop whitespace, "_" before every
// upper-case letter that follows another character, lower-case. A value that is
// already all lower-case letters is returned unchanged.
func snake(s string) string {
	if isCtypeLower(s) {
		return s
	}
	words := strings.Fields(ucwords(s))
	joined := []rune(strings.Join(words, ""))
	var b strings.Builder
	for i, r := range joined {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

func isCtypeLower(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 'a' || s[i] > 'z' {
			return false
		}
	}
	return true
}

// ucwords upper-cases the first byte of every whitespace-separated word.
func ucwords(s string) string {
	b := []byte(s)
	start := true
	for i, c := range b {
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f' || c == '\v' {
			start = true
			continue
		}
		if start && c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
		start = false
	}
	return string(b)
}

// strGetCSV is str_getcsv($s) with "," / `"` / `\`: quoted fields may contain commas,
// and "" inside quotes is a literal quote.
func strGetCSV(s string) []string {
	var out []string
	var cur strings.Builder
	inQuotes, quotedField := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inQuotes && c == '"' && i+1 < len(s) && s[i+1] == '"':
			cur.WriteByte('"')
			i++
		case inQuotes && c == '"':
			inQuotes = false
		case !inQuotes && c == '"' && cur.Len() == 0 && !quotedField:
			inQuotes, quotedField = true, true
		case !inQuotes && c == ',':
			out = append(out, cur.String())
			cur.Reset()
			quotedField = false
		default:
			cur.WriteByte(c)
		}
	}
	return append(out, cur.String())
}

// mustCompilePHPRegex converts a delimited PCRE pattern ("/^a$/i") to Go RE2.
// PCRE's "$" (without the m or D modifier) also matches before a final "\n"; that is kept.
// It panics on patterns RE2 cannot express: rule sets are static program data.
func mustCompilePHPRegex(p string) *regexp.Regexp {
	re, err := compilePHPRegex(p)
	if err != nil {
		panic(fmt.Sprintf("validation: regex %q: %v", p, err))
	}
	return re
}

func compilePHPRegex(p string) (*regexp.Regexp, error) {
	if len(p) < 2 {
		return nil, fmt.Errorf("pattern too short")
	}
	delim := p[0]
	closer := delim
	switch delim {
	case '(':
		closer = ')'
	case '{':
		closer = '}'
	case '[':
		closer = ']'
	case '<':
		closer = '>'
	}
	end := strings.LastIndexByte(p, closer)
	if end <= 0 {
		return nil, fmt.Errorf("no closing delimiter")
	}
	body, mods := p[1:end], p[end+1:]
	flags := ""
	multiline, dollarEndOnly := false, false
	for _, m := range mods {
		switch m {
		case 'i':
			flags += "i"
		case 'm':
			flags += "m"
			multiline = true
		case 's':
			flags += "s"
		case 'u':
		case 'D':
			dollarEndOnly = true
		default:
			return nil, fmt.Errorf("unsupported modifier %q", m)
		}
	}
	if !multiline && !dollarEndOnly {
		body = pcreDollar(body)
	}
	if flags != "" {
		body = "(?" + flags + ")" + body
	}
	return regexp.Compile(body)
}

// pcreDollar rewrites every unescaped "$" outside a character class to `(?:\n?\z)`.
func pcreDollar(body string) string {
	var b strings.Builder
	inClass := false
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch {
		case c == '\\' && i+1 < len(body):
			b.WriteByte(c)
			b.WriteByte(body[i+1])
			i++
		case c == '[' && !inClass:
			inClass = true
			b.WriteByte(c)
		case c == ']' && inClass:
			inClass = false
			b.WriteByte(c)
		case c == '$' && !inClass:
			b.WriteString(`(?:\n?\z)`)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
