// Package phpval models the PHP values Laravel works with after json_decode($body, true)
// or parse_str(): nil, bool, int64, float64, string, []any (a list) and *jsonx.OrderedMap
// (an associative PHP array; it encodes as [] when empty, like PHP).
//
// It holds the few PHP semantics the validation engine and the translation files need:
// decoding with PHP's number rules, dot-path access (Arr::get / Arr::has / Arr::dot),
// string conversion, is_numeric and loose (==) comparison.
package phpval

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

// Map is an associative PHP array with insertion-ordered keys.
type Map = *jsonx.OrderedMap

// NewMap returns an empty associative array (encodes as []).
func NewMap() Map { return jsonx.NewArray() }

// Decode is json_decode($b, true): objects become ordered Maps, arrays []any,
// integers int64 (floats when they overflow), other numbers float64.
func Decode(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	v, err := decodeValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("phpval: trailing data after JSON value")
	}
	return v, nil
}

func decodeValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("phpval: decode: %w", err)
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			m := NewMap()
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, fmt.Errorf("phpval: decode: %w", err)
				}
				key, _ := kt.(string)
				val, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				m.Set(key, val)
			}
			if _, err := dec.Token(); err != nil {
				return nil, fmt.Errorf("phpval: decode: %w", err)
			}
			return m, nil
		case '[':
			list := []any{}
			for dec.More() {
				val, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				list = append(list, val)
			}
			if _, err := dec.Token(); err != nil {
				return nil, fmt.Errorf("phpval: decode: %w", err)
			}
			return list, nil
		}
		return nil, fmt.Errorf("phpval: unexpected delimiter %v", t)
	case json.Number:
		return number(string(t)), nil
	default:
		return t, nil // string, bool, nil
	}
}

// number converts a JSON number like PHP: integer literals that fit int64 stay ints.
func number(s string) any {
	if !strings.ContainsAny(s, ".eE") {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// IsArray reports whether v is a PHP array (list or map).
func IsArray(v any) bool {
	switch v.(type) {
	case []any, Map:
		return true
	}
	return false
}

// Count is count($v) for arrays (0 for anything else).
func Count(v any) int {
	switch a := v.(type) {
	case []any:
		return len(a)
	case Map:
		return a.Len()
	}
	return 0
}

// Entries returns the key/value pairs of an array in order (list keys are "0", "1", …).
func Entries(v any) (keys []string, vals []any) {
	switch a := v.(type) {
	case []any:
		for i, x := range a {
			keys = append(keys, strconv.Itoa(i))
			vals = append(vals, x)
		}
	case Map:
		for _, k := range a.Keys() {
			x, _ := a.Get(k)
			keys = append(keys, k)
			vals = append(vals, x)
		}
	}
	return keys, vals
}

// child returns $arr[$key] for a list or map.
func child(v any, key string) (any, bool) {
	switch a := v.(type) {
	case Map:
		return a.Get(key)
	case []any:
		i, err := strconv.Atoi(key)
		if err != nil || i < 0 || i >= len(a) || strconv.Itoa(i) != key {
			return nil, false
		}
		return a[i], true
	}
	return nil, false
}

// Get is Arr::get($data, $path) for a dot path (no wildcards); ok=false when missing.
// An empty path returns data itself.
func Get(data any, path string) (any, bool) {
	if path == "" {
		return data, true
	}
	cur := data
	for _, seg := range strings.Split(path, ".") {
		next, ok := child(cur, seg)
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

// Has is Arr::has($data, $path).
func Has(data any, path string) bool {
	_, ok := Get(data, path)
	return ok
}

// Dot is Arr::dot(): leaves (and empty arrays) keyed by their dot path, in order.
func Dot(data any) (keys []string, vals []any) {
	var walk func(v any, prefix string)
	walk = func(v any, prefix string) {
		ks, vs := Entries(v)
		for i, k := range ks {
			if IsArray(vs[i]) && Count(vs[i]) > 0 {
				walk(vs[i], prefix+k+".")
				continue
			}
			keys = append(keys, prefix+k)
			vals = append(vals, vs[i])
		}
	}
	walk(data, "")
	return keys, vals
}

// Clone deep-copies arrays (scalars are immutable).
func Clone(v any) any {
	switch a := v.(type) {
	case []any:
		out := make([]any, len(a))
		for i, x := range a {
			out[i] = Clone(x)
		}
		return out
	case Map:
		out := NewMap()
		for _, k := range a.Keys() {
			x, _ := a.Get(k)
			out.Set(k, Clone(x))
		}
		return out
	}
	return v
}

// ToString is PHP's (string) cast for scalars: true "1", false/null "", floats in
// PHP's shortest form ("1.5", "1.0E+25"). Arrays return "Array".
func ToString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case bool:
		if x {
			return "1"
		}
		return ""
	case int64:
		return strconv.FormatInt(x, 10)
	case int:
		return strconv.Itoa(x)
	case float64:
		return phpround.String(x)
	case string:
		return x
	case []any, Map:
		return "Array"
	}
	return fmt.Sprint(v)
}

// IsNumeric is PHP 8's is_numeric().
func IsNumeric(v any) bool {
	switch x := v.(type) {
	case int64, int, float64:
		return true
	case string:
		return IsNumericString(x)
	}
	return false
}

// IsNumericString is is_numeric() for a string: optional surrounding whitespace,
// sign, digits with an optional fraction, optional exponent. No hex, no "_".
func IsNumericString(s string) bool {
	s = strings.TrimLeft(s, " \t\n\r\v\f")
	s = strings.TrimRight(s, " \t\n\r\v\f")
	if s == "" {
		return false
	}
	i := 0
	if s[i] == '+' || s[i] == '-' {
		i++
	}
	digits := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
		digits++
	}
	if i < len(s) && s[i] == '.' {
		i++
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
			digits++
		}
	}
	if digits == 0 {
		return false
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		exp := 0
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
			exp++
		}
		if exp == 0 {
			return false
		}
	}
	return i == len(s)
}

// ToFloat converts a numeric value (IsNumeric true) to float64.
func ToFloat(v any) float64 {
	switch x := v.(type) {
	case int64:
		return float64(x)
	case int:
		return float64(x)
	case float64:
		return x
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f
	case bool:
		if x {
			return 1
		}
	}
	return 0
}

// LooseEqual is PHP 8's == for scalars: numeric strings compare as numbers, bool/null
// compare by truthiness, a number against a non-numeric string compares as strings.
func LooseEqual(a, b any) bool {
	switch {
	case a == nil && b == nil:
		return true
	case isBool(a) || isBool(b) || a == nil || b == nil:
		return Truthy(a) == Truthy(b)
	}
	as, aStr := a.(string)
	bs, bStr := b.(string)
	switch {
	case aStr && bStr:
		if IsNumericString(as) && IsNumericString(bs) {
			return ToFloat(as) == ToFloat(bs)
		}
		return as == bs
	case aStr || bStr:
		s, n := as, b
		if bStr {
			s, n = bs, a
		}
		if IsNumericString(s) {
			return ToFloat(s) == ToFloat(n)
		}
		return s == ToString(n)
	}
	if IsNumeric(a) && IsNumeric(b) {
		return ToFloat(a) == ToFloat(b)
	}
	return false
}

func isBool(v any) bool { _, ok := v.(bool); return ok }

// Truthy is PHP's (bool) cast.
func Truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case int64:
		return x != 0
	case int:
		return x != 0
	case float64:
		return x != 0
	case string:
		return x != "" && x != "0"
	case []any, Map:
		return Count(x) > 0
	}
	return true
}

// Packed turns every Map whose keys are exactly "0".."n-1" (in order) into a list,
// recursively — what json_encode does with such a PHP array.
func Packed(v any) any {
	switch a := v.(type) {
	case []any:
		out := make([]any, len(a))
		for i, x := range a {
			out[i] = Packed(x)
		}
		return out
	case Map:
		keys := a.Keys()
		sequential := len(keys) > 0
		for i, k := range keys {
			if k != strconv.Itoa(i) {
				sequential = false
				break
			}
		}
		if sequential {
			out := make([]any, len(keys))
			for i, k := range keys {
				x, _ := a.Get(k)
				out[i] = Packed(x)
			}
			return out
		}
		out := NewMap()
		for _, k := range keys {
			x, _ := a.Get(k)
			out.Set(k, Packed(x))
		}
		return out
	}
	return v
}
