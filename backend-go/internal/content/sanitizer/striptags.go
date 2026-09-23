package sanitizer

import (
	"html"
	"strconv"
	"strings"
)

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\v' || b == '\f' || b == '\r'
}

func lower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}

// StripTags is PHP's strip_tags($s) without allowed tags (php_strip_tags_ex state
// machine, PHP 8.4): state 0 text, 1 HTML tag, 2 PHP tag, 3 <!…>, 4 <!-- comment -->.
func StripTags(s string) string {
	buf := []byte(s)
	end := len(buf)
	out := make([]byte, 0, len(buf))
	var state, depth, br int
	isXML := false
	var inQ, lc byte
	at := func(i int) byte {
		if i < 0 || i >= end {
			return 0
		}
		return buf[i]
	}
	for p := 0; p < end; p++ {
		c := buf[p]
		switch state {
		case 0:
			switch c {
			case 0:
			case '<':
				if inQ != 0 {
					break
				}
				if isSpace(at(p + 1)) {
					out = append(out, c)
					break
				}
				lc = '<'
				state = 1
			case '>':
				if depth > 0 {
					depth--
					break
				}
				if inQ != 0 {
					break
				}
				out = append(out, c)
			default:
				out = append(out, c)
			}
		case 1:
			switch c {
			case '<':
				if inQ != 0 || isSpace(at(p+1)) {
					break
				}
				depth++
			case '>':
				if depth > 0 {
					depth--
					break
				}
				if inQ != 0 {
					break
				}
				lc = '>'
				if isXML && at(p-1) == '-' {
					break
				}
				inQ, state = 0, 0
				isXML = false
			case '"', '\'':
				if p != 0 && (inQ == 0 || c == inQ) {
					if inQ != 0 {
						inQ = 0
					} else {
						inQ = c
					}
				}
			case '!':
				if at(p-1) == '<' {
					state = 3
					lc = c
				}
			case '?':
				if at(p-1) == '<' {
					br = 0
					state = 2
				}
			}
		case 2:
			switch c {
			case '(':
				if lc != '"' && lc != '\'' {
					lc = '('
					br++
				}
			case ')':
				if lc != '"' && lc != '\'' {
					lc = ')'
					br--
				}
			case '>':
				if depth > 0 {
					depth--
					break
				}
				if inQ != 0 {
					break
				}
				if br == 0 && p >= 1 && lc != '"' && at(p-1) == '?' {
					inQ, state = 0, 0
				}
			case '"', '\'':
				if p >= 1 && at(p-1) != '\\' {
					if lc == c {
						lc = 0
					} else if lc != '\\' {
						lc = c
					}
					if p != 0 && (inQ == 0 || c == inQ) {
						if inQ != 0 {
							inQ = 0
						} else {
							inQ = c
						}
					}
				}
			case 'l', 'L':
				if p > 4 && lower(at(p-1)) == 'm' && lower(at(p-2)) == 'x' && at(p-3) == '?' && at(p-4) == '<' {
					state = 1
					isXML = true
				}
			}
		case 3:
			switch c {
			case '>':
				if depth > 0 {
					depth--
					break
				}
				if inQ != 0 {
					break
				}
				inQ, state = 0, 0
			case '"', '\'':
				if p != 0 && at(p-1) != '\\' && (inQ == 0 || c == inQ) {
					if inQ != 0 {
						inQ = 0
					} else {
						inQ = c
					}
				}
			case '-':
				if p >= 2 && at(p-1) == '-' && at(p-2) == '!' {
					state = 4
				}
			case 'E', 'e':
				if p > 6 && strings.EqualFold(string(buf[p-6:p]), "doctyp") {
					state = 1
				}
			}
		case 4:
			if c == '>' && inQ == 0 && p >= 2 && at(p-1) == '-' && at(p-2) == '-' {
				inQ, state = 0, 0
			}
		}
	}
	return string(out)
}

// DecodeEntities is html_entity_decode($s, ENT_QUOTES | ENT_HTML5, 'UTF-8'): named
// HTML5 entities and numeric references, each only with its terminating ';', numeric
// ones only for code points HTML5 allows (anything else stays literal).
func DecodeEntities(s string) string {
	if !strings.Contains(s, "&") {
		return s
	}
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] != '&' {
			b.WriteByte(s[i])
			i++
			continue
		}
		if rep, n, ok := decodeAt(s[i:]); ok {
			b.WriteString(rep)
			i += n
			continue
		}
		b.WriteByte('&')
		i++
	}
	return b.String()
}

func decodeAt(s string) (string, int, bool) {
	if len(s) < 3 {
		return "", 0, false
	}
	if s[1] == '#' {
		j := 2
		base := 10
		if j < len(s) && (s[j] == 'x' || s[j] == 'X') {
			base = 16
			j++
		}
		start := j
		for j < len(s) && isDigit(s[j], base) {
			j++
		}
		if j == start || j >= len(s) || s[j] != ';' {
			return "", 0, false
		}
		code, err := strconv.ParseUint(s[start:j], base, 32)
		if err != nil || code > 0x10FFFF || !html5Allowed(rune(code)) || code == 0x0D {
			return "", 0, false
		}
		return string(rune(code)), j + 1, true
	}
	j := 1
	for j < len(s) && isAlnum(s[j]) {
		j++
	}
	if j == 1 || j >= len(s) || s[j] != ';' {
		return "", 0, false
	}
	ref := s[:j+1]
	dec := html.UnescapeString(ref)
	// Go also matches legacy prefixes ("&notit;" → "¬it;"); only a whole-name match counts.
	if dec == ref || (strings.HasSuffix(dec, ";") && ref != "&semi;") {
		return "", 0, false
	}
	return dec, j + 1, true
}

func isDigit(b byte, base int) bool {
	if b >= '0' && b <= '9' {
		return true
	}
	if base == 16 {
		l := lower(b)
		return l >= 'a' && l <= 'f'
	}
	return false
}

func isAlnum(b byte) bool {
	l := lower(b)
	return (l >= 'a' && l <= 'z') || (b >= '0' && b <= '9')
}

// html5Allowed is unicode_cp_is_allowed(cp, ENT_HTML_DOC_HTML5).
func html5Allowed(cp rune) bool {
	return (cp >= 0x20 && cp <= 0x7E) ||
		(cp >= 0x09 && cp <= 0x0D && cp != 0x0B) ||
		(cp >= 0xA0 && cp <= 0xD7FF) ||
		(cp >= 0xE000 && cp <= 0x10FFFF && (cp&0xFFFF) < 0xFFFE && (cp < 0xFDD0 || cp > 0xFDEF))
}
