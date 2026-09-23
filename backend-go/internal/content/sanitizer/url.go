package sanitizer

import (
	"strings"
)

// parseURLScheme is parse_url($url, PHP_URL_SCHEME) (php_url_parse_ex2, PHP 8.4):
// the scheme ("" when none), ok=false when parse_url returns false.
func parseURLScheme(s string) (string, bool) {
	ue := len(s)
	e := strings.IndexByte(s, ':')
	switch {
	case e > 0:
		for p := 0; p < e; p++ {
			c := s[p]
			if !isASCIILetter(c) && !isASCIIDigit(c) && c != '+' && c != '.' && c != '-' {
				q := strings.IndexByte(s, '?')
				switch {
				case e+1 < ue && q >= 0 && e < q:
					return parsePort(s, 0, e)
				case 1 < ue && s[0] == '/' && s[1] == '/':
					return parseHost(s, 2, false)
				default:
					return "", true
				}
			}
		}
		if e+1 == ue {
			return s[:e], true
		}
		if s[e+1] != '/' {
			p := e + 1
			for p < ue && isASCIIDigit(s[p]) {
				p++
			}
			if (p == ue || s[p] == '/') && p-e < 7 {
				return parsePort(s, 0, e)
			}
			return s[:e], true
		}
		scheme := s[:e]
		if e+2 < ue && s[e+2] == '/' {
			start := e + 3
			if strings.EqualFold(scheme, "file") && e+3 < ue && s[e+3] == '/' {
				return scheme, true
			}
			if _, ok := parseHost(s, start, false); !ok {
				return "", false
			}
		}
		return scheme, true
	case e == 0:
		return parsePort(s, 0, e)
	case 1 < ue && s[0] == '/' && s[1] == '/':
		return parseHost(s, 2, false)
	}
	return "", true
}

// parsePort is the parse_port label: s[0:] is the rest, e the ':' index. No scheme.
func parsePort(s string, start, e int) (string, bool) {
	ue := len(s)
	p := e + 1
	pp := p
	for pp < ue && pp-p < 6 && isASCIIDigit(s[pp]) {
		pp++
	}
	hasPort := false
	switch {
	case pp-p > 0 && pp-p < 6 && (pp == ue || s[pp] == '/'):
		port, ok := strtol(s[p:pp])
		if !ok || port < 0 || port > 65535 {
			return "", false
		}
		hasPort = true
		if start+1 < ue && s[start] == '/' && s[start+1] == '/' {
			start += 2
		}
	case p == pp && pp == ue:
		return "", false
	case start+1 < ue && s[start] == '/' && s[start+1] == '/':
		start += 2
	default:
		return "", true // just_path
	}
	return parseHost(s, start, hasPort)
}

// parseHost is the parse_host label (only its failure modes matter here).
func parseHost(s string, start int, hasPort bool) (string, bool) {
	rest := s[start:]
	e := strings.IndexAny(rest, "/?#")
	if e < 0 {
		e = len(rest)
	}
	host := rest[:e]
	if at := strings.LastIndexByte(host, '@'); at >= 0 {
		host = host[at+1:]
	}
	var colon int
	if len(host) > 0 && host[0] == '[' && host[len(host)-1] == ']' {
		colon = -1
	} else {
		colon = strings.LastIndexByte(host, ':')
	}
	hostLen := len(host)
	if colon >= 0 {
		if !hasPort {
			port := host[colon+1:]
			if len(port) > 5 {
				return "", false
			}
			if len(port) > 0 {
				n, ok := strtol(port)
				if !ok || n < 0 || n > 65535 {
					return "", false
				}
			}
		}
		hostLen = colon
	}
	if hostLen < 1 {
		return "", false
	}
	return "", true
}

// strtol is ZEND_STRTOL(…, 10): leading whitespace, sign, digits; ok=false when no
// digit was read.
func strtol(s string) (int64, bool) {
	i := 0
	for i < len(s) && (s[i] == ' ' || (s[i] >= '\t' && s[i] <= '\r')) {
		i++
	}
	neg := false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	start := i
	var n int64
	for i < len(s) && isASCIIDigit(s[i]) {
		if n < 1<<40 {
			n = n*10 + int64(s[i]-'0')
		}
		i++
	}
	if i == start {
		return 0, false
	}
	if neg {
		n = -n
	}
	return n, true
}
