package sanitizer

import (
	"strings"
)

// serialize is libxml2 2.9.14's htmlNodeDumpFormatOutput with format=0 (what
// DOMDocument::saveHTML($node) prints).
func serialize(b *strings.Builder, n *node) {
	switch n.typ {
	case textNode:
		if n.parent != nil && (strings.EqualFold(n.parent.name, "script") || strings.EqualFold(n.parent.name, "style")) {
			b.WriteString(n.data)
			return
		}
		b.WriteString(encodeEntities(n.data, false))
	case commentNode:
		b.WriteString("<!--" + n.data + "-->")
	case piNode:
		b.WriteString("<?" + n.name)
		if n.hasData {
			b.WriteString(" " + n.data)
		}
		b.WriteString(">")
	case elementNode:
		info, known := knownElements[n.name]
		b.WriteString("<" + n.name)
		for _, a := range n.attrs {
			writeAttr(b, n, a)
		}
		switch {
		case known && info.empty:
			b.WriteString(">")
		case len(n.children) == 0:
			if known && info.saveEndTag && n.name != "html" && n.name != "body" {
				b.WriteString(">")
			} else {
				b.WriteString("></" + n.name + ">")
			}
		default:
			b.WriteString(">")
			for _, c := range n.children {
				serialize(b, c)
			}
			b.WriteString("</" + n.name + ">")
		}
	}
}

// booleanAttrs is htmlBooleanAttrs: printed without a value.
var booleanAttrs = []string{"checked", "compact", "declare", "defer", "disabled", "ismap",
	"multiple", "nohref", "noresize", "noshade", "nowrap", "readonly", "selected"}

// writeAttr is htmlAttrDumpOutput.
func writeAttr(b *strings.Builder, el *node, a attr) {
	b.WriteString(" " + a.name)
	if !a.hasValue {
		return
	}
	for _, x := range booleanAttrs {
		if strings.EqualFold(x, a.name) {
			return
		}
	}
	value := encodeEntities(a.value, true)
	b.WriteString("=")
	lname := strings.ToLower(a.name)
	if lname == "href" || lname == "action" || lname == "src" || (lname == "name" && strings.EqualFold(el.name, "a")) {
		b.WriteString(quoted(uriEscape(strings.TrimLeft(value, " \t\n\r"), "@/:=?;#%&,+<>")))
		return
	}
	b.WriteString(quoted(value))
}

// encodeEntities is xmlEncodeEntitiesInternal for an HTML document: < > & escaped,
// UTF-8 copied as is, control characters other than TAB/LF/CR dropped, and in
// attributes the HTML 4 `&{…}` and `<!--…-->` constructs passed through.
func encodeEntities(s string, isAttr bool) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '<':
			if isAttr && strings.HasPrefix(s[i:], "<!--") {
				if end := strings.Index(s[i:], "-->"); end >= 0 {
					b.WriteString(s[i : i+end+3])
					i += end + 2
					continue
				}
			}
			b.WriteString("&lt;")
		case c == '>':
			b.WriteString("&gt;")
		case c == '&':
			if isAttr && i+1 < len(s) && s[i+1] == '{' {
				if end := strings.IndexByte(s[i:], '}'); end >= 0 {
					b.WriteString(s[i : i+end+1])
					i += end
					continue
				}
			}
			b.WriteString("&amp;")
		case (c >= 0x20 && c < 0x80) || c == '\n' || c == '\t' || c == '\r' || c >= 0x80:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// uriEscape is xmlURIEscapeStr: everything but unreserved characters, '@' and list is
// percent-encoded byte by byte (upper-case hex).
func uriEscape(s, list string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '@' && !isUnreserved(c) && strings.IndexByte(list, c) < 0 {
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&0xF])
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func isUnreserved(c byte) bool {
	return isASCIILetter(c) || isASCIIDigit(c) || strings.IndexByte("-_.!~*'()", c) >= 0
}

// quoted is xmlBufWriteQuotedString.
func quoted(s string) string {
	if strings.Contains(s, `"`) {
		if strings.Contains(s, "'") {
			return `"` + strings.ReplaceAll(s, `"`, "&quot;") + `"`
		}
		return "'" + s + "'"
	}
	return `"` + s + `"`
}
