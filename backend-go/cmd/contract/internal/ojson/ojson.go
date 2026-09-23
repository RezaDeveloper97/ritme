// Package ojson is the contract harness's JSON model: an order-preserving parse of a
// response body, a deterministic encoder for goldens, path rules (ignore / pattern)
// and a type-strict semantic differ.
//
// Normalisation rules (docs/go-migration/README.md → Contract):
//   - object key order is ignored when comparing (but preserved when encoding goldens);
//   - strings compare by decoded value, so `\/` == `/` and `é` == `é`;
//   - types are strict: "65.50" (string) ≠ 65.5 (number), [] ≠ {} ≠ null, true ≠ 1;
//   - numbers must be the same kind of literal (integer vs. fraction/exponent) and
//     numerically equal: 10 ≠ 10.0 (a Kotlin Int field rejects 10.0), but 0.5 == 5e-1.
package ojson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Kind is a JSON value type.
type Kind int

// JSON value kinds.
const (
	Null Kind = iota
	Bool
	Number
	String
	Array
	Object
)

func (k Kind) String() string {
	switch k {
	case Null:
		return "null"
	case Bool:
		return "bool"
	case Number:
		return "number"
	case String:
		return "string"
	case Array:
		return "array"
	case Object:
		return "object"
	}
	return "?"
}

// Value is one JSON value. Objects keep their key order.
type Value struct {
	Kind Kind
	Bool bool
	// Num is the number literal exactly as it appeared in the input.
	Num string
	// Str is the decoded string value.
	Str string
	Arr []*Value
	// Keys and Vals are parallel slices (object members in input order).
	Keys []string
	Vals []*Value
}

// Get returns the member named key of an object, or nil.
func (v *Value) Get(key string) *Value {
	if v == nil || v.Kind != Object {
		return nil
	}
	for i, k := range v.Keys {
		if k == key {
			return v.Vals[i]
		}
	}
	return nil
}

// Scalar renders a scalar value as text (strings unquoted); used for captures and
// pattern checks. Arrays and objects render as compact JSON.
func (v *Value) Scalar() string {
	switch v.Kind {
	case Null:
		return "null"
	case Bool:
		if v.Bool {
			return "true"
		}
		return "false"
	case Number:
		return v.Num
	case String:
		return v.Str
	}
	return string(Encode(v, ""))
}

// NewString returns a string value.
func NewString(s string) *Value { return &Value{Kind: String, Str: s} }

// Parse decodes exactly one JSON value (surrounding whitespace allowed).
func Parse(data []byte) (*Value, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := parseValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("ojson: trailing data after JSON value")
	}
	return v, nil
}

func parseValue(dec *json.Decoder) (*Value, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("ojson: %w", err)
	}
	return fromToken(dec, tok)
}

func fromToken(dec *json.Decoder, tok json.Token) (*Value, error) {
	switch t := tok.(type) {
	case nil:
		return &Value{Kind: Null}, nil
	case bool:
		return &Value{Kind: Bool, Bool: t}, nil
	case json.Number:
		return &Value{Kind: Number, Num: t.String()}, nil
	case string:
		return &Value{Kind: String, Str: t}, nil
	case json.Delim:
		switch t {
		case '[':
			v := &Value{Kind: Array, Arr: []*Value{}}
			for dec.More() {
				el, err := parseValue(dec)
				if err != nil {
					return nil, err
				}
				v.Arr = append(v.Arr, el)
			}
			if _, err := dec.Token(); err != nil { // ']'
				return nil, fmt.Errorf("ojson: %w", err)
			}
			return v, nil
		case '{':
			v := &Value{Kind: Object, Keys: []string{}, Vals: []*Value{}}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, fmt.Errorf("ojson: %w", err)
				}
				key, ok := kt.(string)
				if !ok {
					return nil, fmt.Errorf("ojson: object key is %T", kt)
				}
				el, err := parseValue(dec)
				if err != nil {
					return nil, err
				}
				v.Keys = append(v.Keys, key)
				v.Vals = append(v.Vals, el)
			}
			if _, err := dec.Token(); err != nil { // '}'
				return nil, fmt.Errorf("ojson: %w", err)
			}
			return v, nil
		}
	}
	return nil, fmt.Errorf("ojson: unexpected token %v", tok)
}

// Encode renders v deterministically: input key order, strings escaped the Go way
// without HTML escaping, number literals verbatim. indent "" gives compact output.
func Encode(v *Value, indent string) []byte {
	var b bytes.Buffer
	encode(&b, v, indent, 0)
	return b.Bytes()
}

func encode(b *bytes.Buffer, v *Value, indent string, depth int) {
	nl := func(d int) {
		if indent != "" {
			b.WriteByte('\n')
			b.WriteString(strings.Repeat(indent, d))
		}
	}
	switch v.Kind {
	case Null:
		b.WriteString("null")
	case Bool:
		if v.Bool {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case Number:
		b.WriteString(v.Num)
	case String:
		writeString(b, v.Str)
	case Array:
		if len(v.Arr) == 0 {
			b.WriteString("[]")
			return
		}
		b.WriteByte('[')
		for i, el := range v.Arr {
			if i > 0 {
				b.WriteByte(',')
			}
			nl(depth + 1)
			encode(b, el, indent, depth+1)
		}
		nl(depth)
		b.WriteByte(']')
	case Object:
		if len(v.Keys) == 0 {
			b.WriteString("{}")
			return
		}
		b.WriteByte('{')
		for i, k := range v.Keys {
			if i > 0 {
				b.WriteByte(',')
			}
			nl(depth + 1)
			writeString(b, k)
			b.WriteByte(':')
			if indent != "" {
				b.WriteByte(' ')
			}
			encode(b, v.Vals[i], indent, depth+1)
		}
		nl(depth)
		b.WriteByte('}')
	}
}

func writeString(b *bytes.Buffer, s string) {
	var tmp bytes.Buffer
	enc := json.NewEncoder(&tmp)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s) // strings always encode
	b.Write(bytes.TrimRight(tmp.Bytes(), "\n"))
}

// FromGo converts a value produced by encoding/json-compatible Go data (maps,
// slices, strings, numbers, bools, nil) into a Value. Map keys are sorted.
func FromGo(x any) (*Value, error) {
	raw, err := json.Marshal(x)
	if err != nil {
		return nil, fmt.Errorf("ojson: %w", err)
	}
	return Parse(raw)
}
