// Package jsonx encodes JSON the way PHP's json_encode does for Laravel responses.
//
// Marshal(v, 0) is json_encode($v) with flags 0 — what response()->json() sends:
// "/" as "\/", non-ASCII as \uXXXX, but <, > and & raw (Go's HTML escaping is off).
// PrettyPrint|UnescapedSlashes is the framework error-page form (4-space indent).
// The contract harness normalises escaping; the types and values must still match,
// so pick the Laravel flags anyway.
package jsonx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// Flags mirror the PHP JSON_* constants that the Laravel code uses.
type Flags uint8

const (
	// UnescapedSlashes is JSON_UNESCAPED_SLASHES: "/" instead of "\/".
	UnescapedSlashes Flags = 1 << iota
	// UnescapedUnicode is JSON_UNESCAPED_UNICODE: raw UTF-8 instead of \uXXXX
	// (U+2028/U+2029 stay escaped, as in PHP without JSON_UNESCAPED_LINE_TERMINATORS).
	UnescapedUnicode
	// PrettyPrint is JSON_PRETTY_PRINT: 4-space indent, "key": value.
	PrettyPrint
)

// Framework is the flag set of Laravel's exception renderer (Handler::prepareJsonResponse).
const Framework = PrettyPrint | UnescapedSlashes

// Marshal encodes v like json_encode($v, flags).
func Marshal(v any, flags Flags) ([]byte, error) {
	b, err := marshalRaw(v)
	if err != nil {
		return nil, err
	}
	if flags&PrettyPrint != 0 {
		var buf bytes.Buffer
		if err := json.Indent(&buf, b, "", "    "); err != nil {
			return nil, fmt.Errorf("jsonx: indent: %w", err)
		}
		b = buf.Bytes()
	}
	return escape(b, flags&UnescapedSlashes == 0, flags&UnescapedUnicode == 0), nil
}

// marshalRaw is json.Marshal without HTML escaping and without the trailing newline.
// Custom MarshalJSON methods in this package use it for nested values, because the
// outer encoder cannot undo a < an inner json.Marshal already wrote.
func marshalRaw(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("jsonx: %w", err)
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

const hexDigits = "0123456789abcdef"

// escape rewrites "/" and non-ASCII runes inside the already valid JSON b. Both can
// only occur inside string literals, so no tokenizing is needed.
func escape(b []byte, slashes, unicode bool) []byte {
	if !slashes && !unicode {
		return b
	}
	out := make([]byte, 0, len(b)+len(b)/8)
	for i := 0; i < len(b); {
		c := b[i]
		switch {
		case c == '/' && slashes:
			out = append(out, '\\', '/')
			i++
		case c >= utf8.RuneSelf && unicode:
			r, size := utf8.DecodeRune(b[i:])
			i += size
			if r > 0xFFFF { // surrogate pair, as PHP writes it
				r -= 0x10000
				out = appendU(out, 0xD800+(r>>10))
				out = appendU(out, 0xDC00+(r&0x3FF))
			} else {
				out = appendU(out, r)
			}
		default:
			out = append(out, c)
			i++
		}
	}
	return out
}

func appendU(out []byte, r rune) []byte {
	return append(out, '\\', 'u',
		hexDigits[r>>12&0xF], hexDigits[r>>8&0xF], hexDigits[r>>4&0xF], hexDigits[r&0xF])
}

// quote is strconv.Quote for JSON strings (no HTML escaping).
func quote(s string) []byte {
	b, _ := marshalRaw(s) // a string always encodes
	return b
}

var null = []byte("null")
