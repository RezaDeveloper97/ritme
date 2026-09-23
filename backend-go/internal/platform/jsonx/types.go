package jsonx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

// ---------------------------------------------------------------------------
// Dates

// laravelLayout is Carbon::toJSON() (the default model date serialisation).
const laravelLayout = "2006-01-02T15:04:05.000000Z"

// LaravelDateTime is a `datetime` cast / timestamp column: the instant in UTC with
// microseconds, "2026-09-23T09:30:00.000000Z". The zero time encodes as null.
type LaravelDateTime struct{ time.Time }

// DateTime wraps t.
func DateTime(t time.Time) LaravelDateTime { return LaravelDateTime{t} }

// MarshalJSON implements json.Marshaler.
func (t LaravelDateTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return null, nil
	}
	return quote(t.UTC().Format(laravelLayout)), nil
}

// LaravelDateCast is a plain `date` cast: Carbon keeps midnight in Asia/Tehran and
// toJSON() converts it to UTC, so 2026-09-23 serialises as the *previous* day,
// "2026-09-22T20:30:00.000000Z" (DST era: "…T19:30:00.000000Z"). The zero Date is null.
type LaravelDateCast struct{ civildate.Date }

// DateCast wraps d.
func DateCast(d civildate.Date) LaravelDateCast { return LaravelDateCast{d} }

// MarshalJSON implements json.Marshaler.
func (d LaravelDateCast) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return null, nil
	}
	return quote(d.TehranMidnight().UTC().Format(laravelLayout)), nil
}

// ISO8601Tehran is Carbon::toIso8601String() with the app timezone:
// "2026-09-23T13:00:00+03:30" ("+04:30" in the DST era). The zero time is null.
type ISO8601Tehran struct{ time.Time }

// ISO8601 wraps t.
func ISO8601(t time.Time) ISO8601Tehran { return ISO8601Tehran{t} }

// MarshalJSON implements json.Marshaler.
func (t ISO8601Tehran) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return null, nil
	}
	return quote(t.In(civildate.Tehran).Format("2006-01-02T15:04:05-07:00")), nil
}

// YMD is the `date:Y-m-d` cast / toDateString(): "2026-09-23" (zero → null).
// civildate.Date already encodes this way; the alias names the intent at call sites.
type YMD = civildate.Date

// ---------------------------------------------------------------------------
// Numbers

// Float encodes like json_encode(float): 10.0 → 10, 1e25 → 1.0e+25, 1e-5 → 1.0e-5.
type Float float64

// MarshalJSON implements json.Marshaler.
func (f Float) MarshalJSON() ([]byte, error) {
	s, err := phpround.JSONFloat(float64(f))
	if err != nil {
		return nil, fmt.Errorf("jsonx: %w", err)
	}
	return []byte(s), nil
}

// DecimalString is a `decimal:N` cast: Laravel returns
// (string) BigDecimal::of($value)->toScale(N, RoundingMode::HALF_UP), so JSON gets a
// string with exactly N decimals ("65.50", blood_sugar decimal:1 "98.5"). The zero
// value is SQL NULL and encodes as null.
type DecimalString struct {
	s     string
	valid bool
}

// NullDecimal is the NULL DecimalString.
var NullDecimal = DecimalString{}

// String returns the formatted decimal ("" for NULL).
func (d DecimalString) String() string { return d.s }

// Valid reports whether d is not NULL.
func (d DecimalString) Valid() bool { return d.valid }

// MarshalJSON implements json.Marshaler.
func (d DecimalString) MarshalJSON() ([]byte, error) {
	if !d.valid {
		return null, nil
	}
	return quote(d.s), nil
}

var reDecimal = regexp.MustCompile(`^([+-]?)(\d*)(?:\.(\d*))?(?:[eE]([+-]?\d+))?$`)

// Decimal casts a column value to `decimal:scale`. src is what the DB driver or a
// request hands over: nil (→ NULL), string / []byte ("65.5", "1e2"), float64
// (converted with PHP's 14-digit (string) cast first, as brick/math does), int or
// int64. Malformed input is an error (Laravel throws MathException).
func Decimal(src any, scale int) (DecimalString, error) {
	var s string
	switch v := src.(type) {
	case nil:
		return NullDecimal, nil
	case string:
		s = v
	case []byte:
		s = string(v)
	case float64:
		s = phpround.String(v)
	case float32:
		s = phpround.String(float64(v))
	case int:
		s = strconv.Itoa(v)
	case int64:
		s = strconv.FormatInt(v, 10)
	case bool: // PHP: (string) true = "1"
		s = map[bool]string{true: "1", false: "0"}[v]
	default:
		return NullDecimal, fmt.Errorf("jsonx: cannot cast %T to decimal", src)
	}
	out, err := formatDecimal(s, scale)
	if err != nil {
		return NullDecimal, err
	}
	return DecimalString{s: out, valid: true}, nil
}

// MustDecimal is Decimal for literals; it panics on malformed input.
func MustDecimal(src any, scale int) DecimalString {
	d, err := Decimal(src, scale)
	if err != nil {
		panic(err)
	}
	return d
}

func formatDecimal(s string, scale int) (string, error) {
	m := reDecimal.FindStringSubmatch(s)
	if m == nil || (m[2] == "" && m[3] == "") {
		return "", fmt.Errorf("jsonx: %q is not a decimal", s)
	}
	neg := m[1] == "-"
	digits := strings.TrimLeft(m[2]+m[3], "0")
	exp := -len(m[3]) // value = digits × 10^exp
	if m[4] != "" {
		e, err := strconv.Atoi(m[4])
		if err != nil {
			return "", fmt.Errorf("jsonx: %q: %w", s, err)
		}
		exp += e
	}
	unscaled := new(big.Int)
	if digits != "" {
		unscaled.SetString(digits, 10)
	}

	// Rescale to `scale` decimals: value × 10^scale = unscaled × 10^(exp+scale).
	shift := exp + scale
	ten := big.NewInt(10)
	if shift >= 0 {
		unscaled.Mul(unscaled, new(big.Int).Exp(ten, big.NewInt(int64(shift)), nil))
	} else {
		div := new(big.Int).Exp(ten, big.NewInt(int64(-shift)), nil)
		q, r := new(big.Int).QuoRem(unscaled, div, new(big.Int))
		// HALF_UP: away from zero when the dropped part is ≥ half.
		if new(big.Int).Mul(r, big.NewInt(2)).Cmp(div) >= 0 {
			q.Add(q, big.NewInt(1))
		}
		unscaled = q
	}

	str := unscaled.String()
	if scale > 0 {
		if len(str) <= scale {
			str = strings.Repeat("0", scale-len(str)+1) + str
		}
		str = str[:len(str)-scale] + "." + str[len(str)-scale:]
	}
	if neg && unscaled.Sign() != 0 {
		str = "-" + str
	}
	return str, nil
}

// ---------------------------------------------------------------------------
// Arrays and maps

// PHPArray is a string-keyed PHP array: an empty one encodes as [] (PHP cannot tell
// an empty map from an empty list), a non-empty one as an object with Go's sorted
// key order. Use OrderedMap when the literal key order matters.
type PHPArray[V any] map[string]V

// MarshalJSON implements json.Marshaler.
func (a PHPArray[V]) MarshalJSON() ([]byte, error) {
	if len(a) == 0 {
		return []byte("[]"), nil
	}
	return marshalRaw(map[string]V(a))
}

// Slice is a list that encodes nil as [] instead of null.
type Slice[T any] []T

// MarshalJSON implements json.Marshaler.
func (s Slice[T]) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return marshalRaw([]T(s))
}

// List returns s, or an empty non-nil slice when s is nil, so it encodes as [].
func List[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// OrderedMap is an object with insertion-ordered keys — a PHP array literal like
// ['success' => true, 'message' => …, 'data' => …]. Setting an existing key keeps
// its position (PHP semantics). NewObject maps encode empty as {}, NewArray maps
// (PHP associative arrays) as [].
type OrderedMap struct {
	keys        []string
	vals        map[string]any
	emptyAsList bool
}

// NewObject returns an empty ordered object (encodes as {} when empty).
func NewObject() *OrderedMap { return &OrderedMap{vals: map[string]any{}} }

// NewArray returns an empty ordered PHP array (encodes as [] when empty).
func NewArray() *OrderedMap { return &OrderedMap{vals: map[string]any{}, emptyAsList: true} }

// Obj builds an ordered object from key/value pairs: Obj("success", true, "data", d).
// It panics on an odd argument count or a non-string key (a programming error).
func Obj(kv ...any) *OrderedMap {
	m := NewObject()
	if len(kv)%2 != 0 {
		panic("jsonx.Obj: odd number of arguments")
	}
	for i := 0; i+1 < len(kv); i += 2 {
		k, ok := kv[i].(string)
		if !ok {
			panic(fmt.Sprintf("jsonx.Obj: key %v is %T, not string", kv[i], kv[i]))
		}
		m.Set(k, kv[i+1])
	}
	return m
}

// Set adds or replaces key and returns m for chaining.
func (m *OrderedMap) Set(key string, v any) *OrderedMap {
	if m.vals == nil {
		m.vals = map[string]any{}
	}
	if _, ok := m.vals[key]; !ok {
		m.keys = append(m.keys, key)
	}
	m.vals[key] = v
	return m
}

// Get returns the value at key.
func (m *OrderedMap) Get(key string) (any, bool) {
	v, ok := m.vals[key]
	return v, ok
}

// Delete removes key (unset($a[$key])).
func (m *OrderedMap) Delete(key string) {
	if _, ok := m.vals[key]; !ok {
		return
	}
	delete(m.vals, key)
	for i, k := range m.keys {
		if k == key {
			m.keys = append(m.keys[:i], m.keys[i+1:]...)
			break
		}
	}
}

// Keys returns the keys in order.
func (m *OrderedMap) Keys() []string { return append([]string(nil), m.keys...) }

// Len is the number of keys.
func (m *OrderedMap) Len() int { return len(m.keys) }

// MarshalJSON implements json.Marshaler.
func (m *OrderedMap) MarshalJSON() ([]byte, error) {
	if m.Len() == 0 {
		if m.emptyAsList {
			return []byte("[]"), nil
		}
		return []byte("{}"), nil
	}
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range m.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(quote(k))
		buf.WriteByte(':')
		b, err := marshalRaw(m.vals[k])
		if err != nil {
			return nil, err
		}
		buf.Write(b)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// Raw passes pre-encoded JSON through (e.g. a JSON column echoed verbatim).
type Raw = json.RawMessage
