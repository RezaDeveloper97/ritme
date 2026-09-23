package validation

import (
	"fmt"
	"math/big"
	"net/mail"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// check runs one rule (the validate<Rule> methods of ValidatesAttributes).
func (v *Validator) check(r Rule, attr string, value any, params []string) bool {
	switch r.Name {
	case "Nullable", "Sometimes", "Bail":
		return true
	case "Required":
		return validateRequired(value)
	case "Present":
		return phpval.Has(v.data, attr)
	case "Filled":
		return !phpval.Has(v.data, attr) || validateRequired(value)
	case "RequiredIf":
		return v.validateRequiredIf(value, params)
	case "Accepted":
		return validateRequired(value) && isAccepted(value)
	case "String":
		_, ok := value.(string)
		return ok
	case "Integer":
		if len(params) > 0 && params[0] == "strict" {
			_, ok := value.(int64)
			return ok
		}
		return filterValidateInt(value)
	case "Numeric":
		if _, isStr := value.(string); isStr && len(params) > 0 && params[0] == "strict" {
			return false
		}
		return phpval.IsNumeric(value)
	case "Boolean":
		return isBooleanish(value, len(params) > 0 && params[0] == "strict")
	case "Array":
		return validateArray(value, params)
	case "List":
		return isList(value)
	case "In":
		return v.validateIn(attr, value, params)
	case "NotIn":
		return v.validateNotIn(attr, value, params)
	case "Regex", "NotRegex":
		if _, isStr := value.(string); !isStr && !phpval.IsNumeric(value) {
			return false
		}
		matched := r.re.MatchString(phpval.ToString(value))
		return matched == (r.Name == "Regex")
	case "Size", "Min", "Max", "Between":
		return v.validateSize(r.Name, attr, value, params)
	case "Date":
		return validateDate(value, v.now)
	case "DateFormat":
		return validateDateFormat(value, params)
	case "Before":
		return v.compareDates(attr, value, params, "<")
	case "BeforeOrEqual":
		return v.compareDates(attr, value, params, "<=")
	case "After":
		return v.compareDates(attr, value, params, ">")
	case "AfterOrEqual":
		return v.compareDates(attr, value, params, ">=")
	case "Email":
		return validateEmail(value)
	}
	panic(fmt.Sprintf("validation: rule %q is not implemented", r.Name))
}

func validateRequired(value any) bool {
	switch x := value.(type) {
	case nil:
		return false
	case string:
		return phpTrim(x) != ""
	case []any, phpval.Map:
		return phpval.Count(x) > 0
	}
	return true
}

// phpTrim is PHP's trim() with the default character list " \t\n\r\0\x0B".
func phpTrim(s string) string { return strings.Trim(s, " \t\n\r\x00\x0B") }

func isAccepted(value any) bool {
	switch x := value.(type) {
	case string:
		return x == "yes" || x == "on" || x == "1" || x == "true"
	case int64:
		return x == 1
	case bool:
		return x
	}
	return false
}

// filterValidateInt is filter_var($value, FILTER_VALIDATE_INT) !== false: the value's
// string form, trimmed, is an optional sign and a decimal integer without leading zeros
// that fits in int64. true ("1") and whole floats (5.0 → "5") pass.
func filterValidateInt(value any) bool {
	switch value.(type) {
	case []any, phpval.Map:
		return false
	case int64:
		return true
	}
	s := strings.Trim(phpval.ToString(value), " \t\n\r\x0B")
	if s == "" {
		return false
	}
	digits := s
	if digits[0] == '+' || digits[0] == '-' {
		digits = digits[1:]
	}
	if digits == "" || (len(digits) > 1 && digits[0] == '0') {
		return false
	}
	for i := 0; i < len(digits); i++ {
		if digits[i] < '0' || digits[i] > '9' {
			return false
		}
	}
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

func isBooleanish(value any, strict bool) bool {
	switch x := value.(type) {
	case bool:
		return true
	case int64:
		return !strict && (x == 0 || x == 1)
	case string:
		return !strict && (x == "0" || x == "1")
	}
	return false
}

func validateArray(value any, keys []string) bool {
	if !phpval.IsArray(value) {
		return false
	}
	if len(keys) == 0 {
		return true
	}
	ks, _ := phpval.Entries(value)
	for _, k := range ks {
		if !slices.Contains(keys, k) {
			return false
		}
	}
	return true
}

func isList(value any) bool {
	switch x := value.(type) {
	case []any:
		return true
	case phpval.Map:
		for i, k := range x.Keys() {
			if k != strconv.Itoa(i) {
				return false
			}
		}
		return true
	}
	return false
}

func (v *Validator) validateIn(attr string, value any, params []string) bool {
	if phpval.IsArray(value) && v.hasRule(attr, "Array") {
		_, elems := phpval.Entries(value)
		for _, e := range elems {
			if phpval.IsArray(e) {
				return false
			}
		}
		for _, e := range elems { // array_diff: string comparison
			if !slices.Contains(params, phpval.ToString(e)) {
				return false
			}
		}
		return true
	}
	return !phpval.IsArray(value) && inArrayLoose(phpval.ToString(value), params)
}

func (v *Validator) validateNotIn(attr string, value any, params []string) bool {
	if phpval.IsArray(value) && v.hasRule(attr, "Array") {
		_, elems := phpval.Entries(value)
		for _, e := range elems {
			if phpval.IsArray(e) || slices.Contains(params, phpval.ToString(e)) {
				return false
			}
		}
		return true
	}
	return !phpval.IsArray(value) && !inArrayLoose(phpval.ToString(value), params)
}

func inArrayLoose(s string, params []string) bool {
	for _, p := range params {
		if phpval.LooseEqual(s, p) {
			return true
		}
	}
	return false
}

func (v *Validator) validateRequiredIf(value any, params []string) bool {
	if len(params) < 2 {
		panic("validation: required_if needs at least 2 parameters")
	}
	if !phpval.Has(v.data, params[0]) {
		return true
	}
	other, _ := phpval.Get(v.data, params[0])
	values := make([]any, 0, len(params)-1)
	for _, p := range params[1:] {
		values = append(values, p)
	}
	_, otherIsBool := other.(bool)
	if otherIsBool || v.ruleIsLiteral(params[0], "Boolean") {
		for i, p := range values {
			switch p {
			case "true":
				values[i] = true
			case "false":
				values[i] = false
			}
		}
	}
	if other == nil {
		for i, p := range values {
			if s, ok := p.(string); ok && strings.ToLower(s) == "null" {
				values[i] = nil
			}
		}
	}
	strict := otherIsBool || other == nil
	for _, want := range values {
		if (strict && strictEqual(other, want)) || (!strict && phpval.LooseEqual(other, want)) {
			return validateRequired(value)
		}
	}
	return true
}

// ruleIsLiteral is in_array('boolean', $this->rules[$attr]): a bare rule without parameters.
func (v *Validator) ruleIsLiteral(attr, name string) bool {
	for _, ar := range v.rules {
		if ar.attr == attr {
			for _, r := range ar.rules {
				if r.Name == name && len(r.Params) == 0 {
					return true
				}
			}
		}
	}
	return false
}

func strictEqual(a, b any) bool {
	switch x := a.(type) {
	case nil:
		return b == nil
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case string:
		y, ok := b.(string)
		return ok && x == y
	}
	return false
}

// ---------------------------------------------------------------------------
// Sizes (getSize + Brick\Math comparisons)

var maxExponent = 1000

func (v *Validator) validateSize(rule, attr string, value any, params []string) bool {
	need := 1
	if rule == "Between" {
		need = 2
	}
	if len(params) < need {
		panic(fmt.Sprintf("validation: %s needs %d parameter(s)", rule, need))
	}
	size, ok := v.getSize(attr, value)
	if !ok {
		return false
	}
	bound := func(p string) (*big.Rat, bool) { return bigNumber(strings.TrimSpace(p)) }
	lo, ok := bound(params[0])
	if !ok {
		return false
	}
	switch rule {
	case "Size":
		return size.Cmp(lo) == 0
	case "Min":
		return size.Cmp(lo) >= 0
	case "Max":
		return size.Cmp(lo) <= 0
	}
	hi, ok := bound(params[1])
	return ok && size.Cmp(lo) >= 0 && size.Cmp(hi) <= 0
}

// getSize: the number itself when numeric and the attribute has a numeric rule, the
// element count for arrays, otherwise the string length in characters.
func (v *Validator) getSize(attr string, value any) (*big.Rat, bool) {
	if phpval.IsNumeric(value) && v.hasRule(attr, numericRules...) {
		s := phpval.ToString(value)
		if x, isStr := value.(string); isStr {
			s = phpTrim(x)
		}
		if i := strings.IndexAny(s, "eE"); i >= 0 {
			exp, _ := strconv.Atoi(strings.TrimLeft(s[i+1:], "+"))
			if exp > maxExponent || exp < -maxExponent {
				return nil, false // MathException: exponent outside the allowed range
			}
		}
		return bigNumber(s)
	}
	if phpval.IsArray(value) {
		return new(big.Rat).SetInt64(int64(phpval.Count(value))), true
	}
	return new(big.Rat).SetInt64(int64(utf8.RuneCountInString(phpval.ToString(value)))), true
}

var reBigNumber = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$`)

// bigNumber is BigNumber::of($s) for a numeric string.
func bigNumber(s string) (*big.Rat, bool) {
	if !reBigNumber.MatchString(s) {
		return nil, false
	}
	s = strings.TrimPrefix(s, "+")
	mant, exp, _ := strings.Cut(strings.ToLower(s), "e")
	if strings.HasSuffix(mant, ".") {
		mant += "0"
	}
	if strings.HasPrefix(mant, ".") {
		mant = "0" + mant
	} else if strings.HasPrefix(mant, "-.") {
		mant = "-0" + mant[1:]
	}
	if exp != "" {
		mant += "e" + exp
	}
	r, ok := new(big.Rat).SetString(mant)
	return r, ok
}

// ---------------------------------------------------------------------------
// Dates

var (
	reDateYMD     = regexp.MustCompile(`^(\d{4})-(\d{1,2})(?:-(\d{1,2}))?`)
	reDateYMDSl   = regexp.MustCompile(`^(\d{4})/(\d{1,2})/(\d{1,2})`)
	reDateCompact = regexp.MustCompile(`^(\d{4})(\d{2})(\d{2})$`)
	reDateDMY     = regexp.MustCompile(`^(\d{1,2})-(\d{1,2})-(\d{4})`)
	reDateMDY     = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})/(\d{4})`)
)

// validateDate is strtotime($v) !== false && checkdate(date_parse($v)) over the formats
// civildate.ParseLenient understands; relative words ("today") have no date part and fail.
func validateDate(value any, now time.Time) bool {
	if _, isStr := value.(string); !isStr && !phpval.IsNumeric(value) {
		return false
	}
	s := strings.TrimSpace(phpval.ToString(value))
	if _, err := civildate.ParseLenient(s, now, civildate.Tehran); err != nil {
		return false
	}
	y, m, d, ok := rawDateParts(s)
	return ok && checkdate(m, d, y)
}

func rawDateParts(s string) (y, m, d int, ok bool) {
	atoi := func(x string) int { n, _ := strconv.Atoi(x); return n }
	if p := reDateYMD.FindStringSubmatch(s); p != nil {
		d = 1
		if p[3] != "" {
			d = atoi(p[3])
		}
		return atoi(p[1]), atoi(p[2]), d, true
	}
	if p := reDateYMDSl.FindStringSubmatch(s); p != nil {
		return atoi(p[1]), atoi(p[2]), atoi(p[3]), true
	}
	if p := reDateCompact.FindStringSubmatch(s); p != nil {
		return atoi(p[1]), atoi(p[2]), atoi(p[3]), true
	}
	if p := reDateDMY.FindStringSubmatch(s); p != nil {
		return atoi(p[3]), atoi(p[2]), atoi(p[1]), true
	}
	if p := reDateMDY.FindStringSubmatch(s); p != nil {
		return atoi(p[3]), atoi(p[1]), atoi(p[2]), true
	}
	return 0, 0, 0, false
}

func checkdate(m, d, y int) bool {
	if m < 1 || m > 12 || d < 1 || y < 1 || y > 32767 {
		return false
	}
	return d <= daysIn(y, m)
}

func daysIn(y, m int) int {
	return time.Date(y, time.Month(m)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// validateDateFormat is DateTime::createFromFormat('!'.$format, $v) + the round trip
// $date->format($format) == $v, for formats built from Y m d H i s and literals.
func validateDateFormat(value any, formats []string) bool {
	if _, isStr := value.(string); !isStr && !phpval.IsNumeric(value) {
		return false
	}
	s := phpval.ToString(value)
	for _, f := range formats {
		if _, ok := parseFormat(f, s); ok {
			return true
		}
	}
	return false
}

// parseFormat parses s strictly in format f (the round trip rejects anything
// createFromFormat would normalise, e.g. "7:05" for H:i or month 13).
func parseFormat(f, s string) (time.Time, bool) {
	var expr strings.Builder
	var order []byte
	for i := 0; i < len(f); i++ {
		switch c := f[i]; c {
		case 'Y':
			expr.WriteString(`(\d{4})`)
			order = append(order, c)
		case 'm', 'd', 'H', 'i', 's':
			expr.WriteString(`(\d{2})`)
			order = append(order, c)
		case '\\':
			if i+1 < len(f) {
				i++
				expr.WriteString(regexp.QuoteMeta(string(f[i])))
			}
		default:
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				panic(fmt.Sprintf("validation: date_format %q: format character %q not supported", f, c))
			}
			expr.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	m := regexp.MustCompile(`^` + expr.String() + `$`).FindStringSubmatch(s)
	if m == nil {
		return time.Time{}, false
	}
	y, mo, d, h, mi, sec := 1970, 1, 1, 0, 0, 0
	for i, c := range order {
		n, _ := strconv.Atoi(m[i+1])
		switch c {
		case 'Y':
			y = n
		case 'm':
			mo = n
		case 'd':
			d = n
		case 'H':
			h = n
		case 'i':
			mi = n
		case 's':
			sec = n
		}
	}
	if mo < 1 || mo > 12 || d < 1 || d > daysIn(y, mo) || h > 23 || mi > 59 || sec > 59 {
		return time.Time{}, false
	}
	return time.Date(y, time.Month(mo), d, h, mi, sec, 0, civildate.Tehran), true
}

// compareDates: the value and the parameter (a date such as "today", or another field's
// value) are both parsed like Carbon::parse and compared as timestamps with PHP 8
// comparison semantics (an unparseable side is null, and null compares as false).
func (v *Validator) compareDates(attr string, value any, params []string, op string) bool {
	if len(params) < 1 {
		panic("validation: date comparison needs a parameter")
	}
	if _, isStr := value.(string); !isStr && !phpval.IsNumeric(value) {
		return false
	}
	if r, ok := v.getRule(attr, "DateFormat"); ok && len(r.Params) > 0 {
		return v.checkDateTimeOrder(r.Params[0], value, params[0], op)
	}
	other := v.timestamp(params[0])
	if other == nil {
		ov, _ := phpval.Get(v.data, params[0])
		other = v.timestamp(ov)
	}
	return compareNullable(v.timestamp(value), other, op)
}

func (v *Validator) checkDateTimeOrder(format string, first any, second string, op string) bool {
	firstT := v.dateWithOptionalFormat(format, first)
	f2 := format
	if r, ok := v.getRule(second, "DateFormat"); ok && len(r.Params) > 0 {
		f2 = r.Params[0]
	}
	secondT := v.dateWithOptionalFormat(f2, second)
	if secondT == nil {
		sv, _ := phpval.Get(v.data, second)
		if sv == nil {
			return true
		}
		secondT = v.dateWithOptionalFormat(f2, sv)
	}
	return firstT != nil && secondT != nil && compareNullable(firstT, secondT, op)
}

func (v *Validator) dateWithOptionalFormat(format string, value any) *int64 {
	if s, ok := value.(string); ok {
		if t, ok := parseFormat(format, s); ok {
			u := t.Unix()
			return &u
		}
	}
	return v.timestamp(value)
}

// timestamp is getDateTimestamp: Carbon::parse($value)->getTimestamp(), nil on failure.
func (v *Validator) timestamp(value any) *int64 {
	if value == nil || phpval.IsArray(value) {
		return nil
	}
	if _, isBool := value.(bool); isBool {
		return nil
	}
	t, err := civildate.ParseLenient(phpval.ToString(value), v.now, civildate.Tehran)
	if err != nil {
		return nil
	}
	u := t.Unix()
	return &u
}

func compareNullable(a, b *int64, op string) bool {
	var c int
	if a != nil && b != nil {
		c = cmpInt(*a, *b)
	} else { // PHP 8: null <=> int compares both as bool
		c = cmpBool(a != nil && *a != 0, b != nil && *b != 0)
	}
	switch op {
	case "<":
		return c < 0
	case "<=":
		return c <= 0
	case ">":
		return c > 0
	case ">=":
		return c >= 0
	}
	return c == 0
}

func cmpInt(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func cmpBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case !a:
		return -1
	}
	return 1
}

// validateEmail approximates egulias RFCValidation (Laravel's default "email"): a bare
// RFC 5322 addr-spec as accepted by net/mail (no display name, no surrounding spaces).
func validateEmail(value any) bool {
	s, ok := value.(string)
	if !ok || s == "" || strings.ContainsAny(s, " \t\r\n<>") {
		return false
	}
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Name == "" && addr.Address == s
}
