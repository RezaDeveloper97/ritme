package sanitizer

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// This file is a port of the parts of libxml2 2.9.14's HTML parser (HTMLparser.c +
// the SAX2 tree builder) that PHP's DOMDocument::loadHTML runs, so the tree HtmlSanitizer
// walks — including libxml2's quirks with malformed markup (implied end tags, ignored
// stray end tags, raw script/style content, HTML 4 entities only) — is the same one PHP
// builds. Options are PHP's defaults: no RECOVER, no NOBLANKS, no NOIMPLIED.

type nodeType uint8

const (
	elementNode nodeType = iota
	textNode
	commentNode
	piNode
)

type attr struct {
	name     string
	value    string
	hasValue bool // `<img alt>` has no value (serialised without `=""`)
}

type node struct {
	typ      nodeType
	name     string // element / PI target name
	data     string // text / comment / PI content
	hasData  bool   // PI: a content part was present (`<?x >` vs `<?x>`)
	attrs    []attr
	parent   *node
	children []*node
}

func (n *node) appendChild(c *node) {
	c.parent = n
	n.children = append(n.children, c)
}

// elemInfo is the subset of libxml2's html40ElementTable the parser and serialiser use.
type elemInfo struct {
	empty      bool // void element (<br>)
	saveEndTag bool // an empty element is written without its end tag (<li>)
}

var knownElements = func() map[string]elemInfo {
	m := map[string]elemInfo{}
	for _, n := range strings.Fields(`a abbr acronym address applet area b base basefont bdo big blockquote
		body br button caption center cite code col colgroup dd del dfn dir div dl dt em embed fieldset font
		form frame frameset h1 h2 h3 h4 h5 h6 head hr html i iframe img input ins isindex kbd label legend li
		link map menu meta noframes noscript object ol optgroup option p param pre q s samp script select
		small span strike strong style sub sup table tbody td textarea tfoot th thead title tr tt u ul var`) {
		m[n] = elemInfo{}
	}
	for _, n := range strings.Fields("area base basefont br col frame hr img input isindex link meta param") {
		m[n] = elemInfo{empty: true}
	}
	m["li"] = elemInfo{saveEndTag: true}
	return m
}()

// endPriority is htmlEndPriority: a stray end tag only closes elements of lower or
// equal priority.
func endPriority(name string) int {
	switch name {
	case "div":
		return 150
	case "td", "th":
		return 160
	case "tr":
		return 170
	case "thead", "tbody", "tfoot":
		return 180
	case "table":
		return 190
	case "head", "body":
		return 200
	case "html":
		return 220
	}
	return 100
}

type parser struct {
	in    []byte
	p     int
	names []string // ctxt->nameTab
	nodes []*node  // ctxt->nodeTab (top = ctxt->node)
	doc   *node
	html  int // ctxt->html: 3 once a head was pushed, 10 once a body was
	depth int // ctxt->depth: discarded misplaced html/head/body start tags
}

// parseDocument parses a complete HTML document into a node tree rooted at a
// document node.
func parseDocument(src string) *node {
	// libxml2 stops at a NUL byte (C strings). CR is kept as is (2.9.14 does not fold it).
	if i := strings.IndexByte(src, 0); i >= 0 {
		src = src[:i]
	}
	ps := &parser{in: []byte(src), doc: &node{typ: elementNode, name: "#document"}, html: 1}

	ps.skipBlanks()
	for (ps.at(0) == '<' && ps.at(1) == '!' && ps.at(2) == '-' && ps.at(3) == '-') ||
		(ps.at(0) == '<' && ps.at(1) == '?') {
		ps.parseComment()
		ps.parsePI()
		ps.skipBlanks()
	}
	if ps.isDoctype() {
		ps.parseDoctype()
	}
	ps.skipBlanks()
	for (ps.at(0) == '<' && ps.at(1) == '!' && ps.at(2) == '-' && ps.at(3) == '-') ||
		(ps.at(0) == '<' && ps.at(1) == '?') {
		ps.parseComment()
		ps.parsePI()
		ps.skipBlanks()
	}
	ps.parseContent()
	if ps.at(0) == 0 {
		ps.autoCloseOnEnd()
	}
	return ps.doc
}

// ---- input ----

func (ps *parser) at(k int) byte {
	if ps.p+k < len(ps.in) {
		return ps.in[ps.p+k]
	}
	return 0
}

func (ps *parser) cur() byte { return ps.at(0) }

func (ps *parser) next() {
	if ps.p < len(ps.in) {
		ps.p++
	}
}

func isBlank(b byte) bool { return b == ' ' || b == '\t' || b == '\n' || b == '\r' }

func isASCIILetter(b byte) bool { return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') }

func isASCIIDigit(b byte) bool { return b >= '0' && b <= '9' }

func (ps *parser) skipBlanks() {
	for isBlank(ps.cur()) {
		ps.next()
	}
}

// curChar is CUR_CHAR: the rune at the cursor and its byte length (invalid UTF-8 is
// taken byte by byte).
func (ps *parser) curChar() (rune, int) {
	if ps.p >= len(ps.in) {
		return 0, 0
	}
	if b := ps.in[ps.p]; b < 0x80 {
		return rune(b), 1
	}
	r, n := utf8.DecodeRune(ps.in[ps.p:])
	return r, n
}

// isXMLChar is IS_CHAR.
func isXMLChar(r rune) bool {
	return r == 0x9 || r == 0xA || r == 0xD || (r >= 0x20 && r <= 0xD7FF) ||
		(r >= 0xE000 && r <= 0xFFFD) || (r >= 0x10000 && r <= 0x10FFFF)
}

func (ps *parser) isDoctype() bool {
	return ps.at(0) == '<' && ps.at(1) == '!' && strings.EqualFold(string(ps.slice(2, 9)), "DOCTYPE")
}

func (ps *parser) slice(from, to int) []byte {
	a, b := ps.p+from, ps.p+to
	if a > len(ps.in) {
		a = len(ps.in)
	}
	if b > len(ps.in) {
		b = len(ps.in)
	}
	return ps.in[a:b]
}

// ---- stacks and SAX ----

func (ps *parser) name() string {
	if len(ps.names) == 0 {
		return ""
	}
	return ps.names[len(ps.names)-1]
}

func (ps *parser) curNode() *node {
	if len(ps.nodes) == 0 {
		return nil
	}
	return ps.nodes[len(ps.nodes)-1]
}

func (ps *parser) namePush(n string) {
	if ps.html < 3 && n == "head" {
		ps.html = 3
	}
	if ps.html < 10 && n == "body" {
		ps.html = 10
	}
	ps.names = append(ps.names, n)
}

func (ps *parser) namePop() {
	if len(ps.names) > 0 {
		ps.names = ps.names[:len(ps.names)-1]
	}
}

func (ps *parser) nodePop() {
	if len(ps.nodes) > 0 {
		ps.nodes = ps.nodes[:len(ps.nodes)-1]
	}
}

// startElement is xmlSAX2StartElement: the element becomes a child of the current node
// (or of the document) and the new current node.
func (ps *parser) startElement(name string, attrs []attr) {
	el := &node{typ: elementNode, name: name, attrs: attrs}
	if parent := ps.curNode(); parent != nil {
		parent.appendChild(el)
	} else {
		ps.doc.appendChild(el)
	}
	ps.nodes = append(ps.nodes, el)
}

// endElement is xmlSAX2EndElement.
func (ps *parser) endElement() { ps.nodePop() }

// characters is xmlSAX2Characters: text is appended to the current node, merged into
// a trailing text node; with no current node it is dropped.
func (ps *parser) characters(s string) {
	parent := ps.curNode()
	if parent == nil || s == "" {
		return
	}
	if n := len(parent.children); n > 0 && parent.children[n-1].typ == textNode {
		parent.children[n-1].data += s
		return
	}
	parent.appendChild(&node{typ: textNode, data: s})
}

func (ps *parser) addMisc(n *node) {
	if parent := ps.curNode(); parent != nil {
		parent.appendChild(n)
	} else {
		ps.doc.appendChild(n)
	}
}

// ---- auto-close / implied ----

func checkAutoClose(newTag, oldTag string) bool { return htmlStartClose[oldTag+" "+newTag] }

func (ps *parser) autoClose(newTag string) {
	for ps.name() != "" && checkAutoClose(newTag, ps.name()) {
		ps.endElement()
		ps.namePop()
	}
}

func (ps *parser) autoCloseOnClose(newTag string) {
	priority := endPriority(newTag)
	i := len(ps.names) - 1
	for ; i >= 0; i-- {
		if ps.names[i] == newTag {
			break
		}
		if endPriority(ps.names[i]) > priority {
			return
		}
	}
	if i < 0 {
		return
	}
	for ps.name() != newTag {
		ps.endElement()
		ps.namePop()
	}
}

func (ps *parser) autoCloseOnEnd() {
	for len(ps.names) > 0 {
		ps.endElement()
		ps.namePop()
	}
}

func (ps *parser) checkImplied(newTag string) {
	if newTag == "html" {
		return
	}
	if len(ps.names) == 0 {
		ps.namePush("html")
		ps.startElement("html", nil)
	}
	if newTag == "body" || newTag == "head" {
		return
	}
	if len(ps.names) <= 1 && (newTag == "script" || newTag == "style" || newTag == "meta" ||
		newTag == "link" || newTag == "title" || newTag == "base") {
		if ps.html >= 3 {
			return
		}
		ps.namePush("head")
		ps.startElement("head", nil)
	} else if newTag != "noframes" && newTag != "frame" && newTag != "frameset" {
		if ps.html >= 10 {
			return
		}
		for _, n := range ps.names {
			if n == "body" || n == "head" {
				return
			}
		}
		ps.namePush("body")
		ps.startElement("body", nil)
	}
}

func (ps *parser) checkParagraph() {
	tag := ps.name()
	if tag == "" || tag == "html" || tag == "head" {
		ps.autoClose("p")
		ps.checkImplied("p")
		ps.namePush("p")
		ps.startElement("p", nil)
	}
}

// ---- content ----

func (ps *parser) parseContent() {
	currentNode := ps.name()
	depth := len(ps.names)
	for {
		if ps.cur() == '<' && ps.at(1) == '/' {
			if ps.parseEndTag() && (currentNode != "" || len(ps.names) == 0) {
				currentNode = ps.name()
				depth = len(ps.names)
			}
			continue
		} else if ps.cur() == '<' && (isASCIILetter(ps.at(1)) || ps.at(1) == '_' || ps.at(1) == ':') {
			if name := ps.peekName(); ps.name() != "" && checkAutoClose(name, ps.name()) {
				ps.autoClose(name)
				continue
			}
		}

		if len(ps.names) > 0 && depth >= len(ps.names) && currentNode != ps.name() {
			currentNode = ps.name()
			depth = len(ps.names)
			continue
		}

		if ps.cur() != 0 && (currentNode == "script" || currentNode == "style") {
			ps.parseScript()
			continue
		}
		if ps.isDoctype() {
			ps.parseDoctype()
		}
		switch {
		case ps.cur() == '<' && ps.at(1) == '!' && ps.at(2) == '-' && ps.at(3) == '-':
			ps.parseComment()
		case ps.cur() == '<' && ps.at(1) == '?':
			ps.parsePI()
		case ps.cur() == '<' && isASCIILetter(ps.at(1)):
			ps.parseElement()
			currentNode = ps.name()
			depth = len(ps.names)
		case ps.cur() == '<':
			ps.characters("<")
			ps.next()
		case ps.cur() == '&':
			ps.parseReference()
		case ps.cur() == 0:
			ps.autoCloseOnEnd()
			return
		default:
			ps.parseCharData()
		}
	}
}

// peekName is htmlParseHTMLName_nonInvasive (lower-cased, no '.').
func (ps *parser) peekName() string {
	var b strings.Builder
	for i := 1; ; i++ {
		c := ps.at(i)
		if !isASCIILetter(c) && !isASCIIDigit(c) && c != ':' && c != '-' && c != '_' {
			break
		}
		b.WriteByte(lower(c))
	}
	return b.String()
}

// parseHTMLName is htmlParseHTMLName (lower-cased; "" = NULL).
func (ps *parser) parseHTMLName() string {
	c := ps.cur()
	if !isASCIILetter(c) && c != '_' && c != ':' && c != '.' {
		return ""
	}
	var b strings.Builder
	for {
		c = ps.cur()
		if !isASCIILetter(c) && !isASCIIDigit(c) && c != ':' && c != '-' && c != '_' && c != '.' {
			break
		}
		b.WriteByte(lower(c))
		ps.next()
	}
	return b.String()
}

// parseName is htmlParseName for the ASCII names entity references and PI targets
// use ("" = NULL). A non-ASCII continuation is consumed too (libxml2's complex path),
// which only ever makes a longer unknown name.
func (ps *parser) parseName() string {
	c := ps.cur()
	if !isASCIILetter(c) && c != '_' && c != ':' {
		if c < 0x80 {
			return ""
		}
		if r, _ := ps.curChar(); !unicode.IsLetter(r) {
			return ""
		}
	}
	start := ps.p
	for {
		c = ps.cur()
		if isASCIILetter(c) || isASCIIDigit(c) || c == '_' || c == '-' || c == ':' || c == '.' {
			ps.next()
			continue
		}
		if c >= 0x80 {
			r, n := ps.curChar()
			if isNameRune(r) {
				ps.p += n
				continue
			}
		}
		break
	}
	return string(ps.in[start:ps.p])
}

// isNameRune approximates libxml2's IS_LETTER / IS_DIGIT / IS_COMBINING / IS_EXTENDER
// for non-ASCII name characters.
func isNameRune(r rune) bool {
	return r != utf8.RuneError && (unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.In(r, unicode.Mn, unicode.Mc) ||
		r == 0xB7 || r == 0x2D0 || r == 0x2D1 || r == 0x387 || r == 0x640 || r == 0xE46 || r == 0xEC6 || r == 0x3005)
}

func (ps *parser) parseElement() {
	failed := ps.parseStartTag()
	name := ps.name()
	if failed == -1 || name == "" {
		if ps.cur() == '>' {
			ps.next()
		}
		return
	}
	info, known := knownElements[name]
	if ps.cur() == '/' && ps.at(1) == '>' {
		ps.p += 2
		ps.endElement()
		ps.namePop()
		return
	}
	if ps.cur() == '>' {
		ps.next()
	} else {
		// "Couldn't find end of Start Tag": end parsing of this node.
		if name == ps.name() {
			ps.nodePop()
			ps.namePop()
		}
		return
	}
	if known && info.empty {
		ps.endElement()
		ps.namePop()
	}
}

// parseStartTag is htmlParseStartTag: -1 failure, 1 discarded, 0 pushed.
func (ps *parser) parseStartTag() int {
	if ps.cur() != '<' {
		return -1
	}
	ps.next()
	name := ps.parseHTMLName()
	if name == "" {
		for ps.cur() != 0 && ps.cur() != '>' {
			ps.next()
		}
		return -1
	}
	ps.autoClose(name)
	ps.checkImplied(name)

	discard := false
	if len(ps.names) > 0 && name == "html" {
		discard = true
		ps.depth++
	}
	if len(ps.names) != 1 && name == "head" {
		discard = true
		ps.depth++
	}
	if name == "body" {
		for _, n := range ps.names {
			if n == "body" {
				discard = true
				ps.depth++
			}
		}
	}

	var attrs []attr
	ps.skipBlanks()
	for ps.cur() != 0 && ps.cur() != '>' && (ps.cur() != '/' || ps.at(1) != '>') {
		aname, value, hasValue := ps.parseAttribute()
		if aname != "" {
			dup := false
			for _, a := range attrs {
				if a.name == aname {
					dup = true
					break
				}
			}
			if !dup {
				attrs = append(attrs, attr{name: aname, value: value, hasValue: hasValue})
			}
		} else {
			for ps.cur() != 0 && !isBlank(ps.cur()) && ps.cur() != '>' && (ps.cur() != '/' || ps.at(1) != '>') {
				ps.next()
			}
		}
		ps.skipBlanks()
	}

	if !discard {
		ps.namePush(name)
		ps.startElement(name, attrs)
		return 0
	}
	return 1
}

func (ps *parser) parseAttribute() (name, value string, hasValue bool) {
	name = ps.parseHTMLName()
	if name == "" {
		return "", "", false
	}
	ps.skipBlanks()
	if ps.cur() == '=' {
		ps.next()
		ps.skipBlanks()
		switch ps.cur() {
		case '"', '\'':
			q := ps.cur()
			ps.next()
			value = ps.parseAttrValue(q)
			if ps.cur() == q {
				ps.next()
			}
		default:
			value = ps.parseAttrValue(0)
		}
		hasValue = true
	}
	return name, value, hasValue
}

// parseAttrValue is htmlParseHTMLAttribute: up to stop (or a blank / '>' when
// unquoted), with character and HTML 4 entity references resolved. A reference to
// U+0000 ends the C string, so the rest of the value is lost.
func (ps *parser) parseAttrValue(stop byte) string {
	var b strings.Builder
	truncated := false
	write := func(s string) {
		if !truncated {
			b.WriteString(s)
		}
	}
	for ps.cur() != 0 && ps.cur() != stop {
		if stop == 0 && (ps.cur() == '>' || isBlank(ps.cur())) {
			break
		}
		if ps.cur() == '&' {
			if ps.at(1) == '#' {
				c := ps.parseCharRef()
				if c == 0 {
					truncated = true
				} else {
					write(string(c))
				}
				continue
			}
			name, r, ok := ps.parseEntityRef()
			switch {
			case name == "":
				write("&")
			case !ok:
				write("&" + name)
			default:
				write(string(r))
			}
			continue
		}
		r, n := ps.curChar()
		write(string(ps.in[ps.p : ps.p+n]))
		_ = r
		ps.p += n
	}
	return b.String()
}

// parseEntityRef is htmlParseEntityRef: consumes '&' and the name, and the ';' only
// when the name is a known HTML 4 entity.
func (ps *parser) parseEntityRef() (name string, value rune, ok bool) {
	ps.next() // '&'
	name = ps.parseName()
	if name == "" {
		return "", 0, false
	}
	if ps.cur() == ';' {
		if r, known := html4Entities[name]; known {
			ps.next()
			return name, r, true
		}
	}
	return name, 0, false
}

// parseCharRef is htmlParseCharRef (0 = invalid, nothing is emitted).
func (ps *parser) parseCharRef() rune {
	val := 0
	if ps.at(0) == '&' && ps.at(1) == '#' && (ps.at(2) == 'x' || ps.at(2) == 'X') {
		ps.p += 3
		for ps.cur() != ';' {
			c := ps.cur()
			var d int
			switch {
			case c >= '0' && c <= '9':
				d = int(c - '0')
			case c >= 'a' && c <= 'f':
				d = int(c-'a') + 10
			case c >= 'A' && c <= 'F':
				d = int(c-'A') + 10
			default:
				d = -1
			}
			if d < 0 {
				break
			}
			if val < 0x110000 {
				val = val*16 + d
			}
			ps.next()
		}
		if ps.cur() == ';' {
			ps.next()
		}
	} else if ps.at(0) == '&' && ps.at(1) == '#' {
		ps.p += 2
		for ps.cur() != ';' {
			c := ps.cur()
			if c < '0' || c > '9' {
				break
			}
			if val < 0x110000 {
				val = val*10 + int(c-'0')
			}
			ps.next()
		}
		if ps.cur() == ';' {
			ps.next()
		}
	}
	if isXMLChar(rune(val)) {
		return rune(val)
	}
	return 0
}

func (ps *parser) parseReference() {
	if ps.at(1) == '#' {
		c := ps.parseCharRef()
		if c == 0 {
			return
		}
		ps.checkParagraph()
		ps.characters(string(c))
		return
	}
	name, r, ok := ps.parseEntityRef()
	ps.checkParagraph()
	switch {
	case name == "":
		ps.characters("&")
	case !ok:
		ps.characters("&" + name)
	default:
		ps.characters(string(r))
	}
}

// parseEndTag is htmlParseEndTag; true when the current element was closed.
func (ps *parser) parseEndTag() bool {
	ps.p += 2
	name := ps.parseHTMLName()
	if name == "" {
		return false
	}
	ps.skipBlanks()
	if ps.cur() != '>' {
		for ps.cur() != 0 && ps.cur() != '>' {
			ps.next()
		}
	}
	if ps.cur() == '>' {
		ps.next()
	}
	if ps.depth > 0 && (name == "html" || name == "body" || name == "head") {
		ps.depth--
		return false
	}
	found := false
	for i := len(ps.names) - 1; i >= 0; i-- {
		if ps.names[i] == name {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	ps.autoCloseOnClose(name)
	if ps.name() != "" && ps.name() == name {
		ps.endElement()
		ps.namePop()
		return true
	}
	return false
}

// parseCharData is htmlParseCharDataInternal (keepBlanks: blanks are kept but do not
// imply a paragraph).
func (ps *parser) parseCharData() {
	var b strings.Builder
	blank := true
	for ps.cur() != '<' && ps.cur() != '&' && ps.cur() != 0 {
		r, n := ps.curChar()
		if isXMLChar(r) && (r != utf8.RuneError || n != 1) {
			b.Write(ps.in[ps.p : ps.p+n])
			if r > 0x7f || !isBlank(byte(r)) { //nolint:gosec // G115: r ≤ 0x7f here
				blank = false
			}
		}
		ps.p += n
	}
	if b.Len() == 0 {
		return
	}
	// areBlanks() is false when the run is followed by a reference ('&').
	if !blank || ps.cur() == '&' {
		ps.checkParagraph()
	}
	ps.characters(b.String())
}

// parseScript is htmlParseScript without RECOVER: raw text up to "</" + a letter.
func (ps *parser) parseScript() {
	var b strings.Builder
	for ps.cur() != 0 {
		if ps.cur() == '<' && ps.at(1) == '/' && isASCIILetter(ps.at(2)) {
			break
		}
		r, n := ps.curChar()
		if isXMLChar(r) && (r != utf8.RuneError || n != 1) {
			b.Write(ps.in[ps.p : ps.p+n])
		}
		ps.p += n
	}
	if b.Len() > 0 {
		// cdataBlock: a CDATA/preserve node; the sanitiser drops script/style whole.
		ps.characters(b.String())
	}
}

func (ps *parser) parseComment() {
	if ps.at(0) != '<' || ps.at(1) != '!' || ps.at(2) != '-' || ps.at(3) != '-' {
		return
	}
	ps.p += 4
	var b strings.Builder
	q, ql := ps.curChar()
	if q == 0 {
		return
	}
	ps.p += ql
	r, rl := ps.curChar()
	if r == 0 {
		return
	}
	ps.p += rl
	cur, l := ps.curChar()
	for cur != 0 && (cur != '>' || r != '-' || q != '-') {
		ps.p += l
		next, nl := ps.curChar()
		if q == '-' && r == '-' && cur == '!' && next == '>' {
			cur = '>'
			break
		}
		if isXMLChar(q) {
			b.WriteRune(q)
		}
		q, ql = r, rl
		r, rl = cur, l
		cur, l = next, nl
		_ = ql
	}
	if cur == '>' {
		ps.next()
		ps.addMisc(&node{typ: commentNode, data: b.String()})
	}
}

func (ps *parser) parsePI() {
	if ps.at(0) != '<' || ps.at(1) != '?' {
		return
	}
	ps.p += 2
	target := ps.parseName()
	if target == "" {
		return
	}
	if ps.cur() == '>' {
		ps.next()
		ps.addMisc(&node{typ: piNode, name: target})
		return
	}
	ps.skipBlanks()
	var b strings.Builder
	cur, l := ps.curChar()
	for cur != 0 && cur != '>' {
		if isXMLChar(cur) {
			b.WriteString(string(ps.in[ps.p : ps.p+l]))
		}
		ps.p += l
		cur, l = ps.curChar()
	}
	if cur == '>' {
		ps.next()
		ps.addMisc(&node{typ: piNode, name: target, data: b.String(), hasData: true})
	}
}

// parseDoctype is htmlParseDocTypeDecl for a (misplaced) <!DOCTYPE …>: it only
// consumes input (the internal subset never reaches the body).
func (ps *parser) parseDoctype() {
	ps.p += 9
	ps.skipBlanks()
	ps.parseName()
	ps.skipBlanks()
	ps.parseExternalID()
	ps.skipBlanks()
	if ps.cur() != '>' {
		for ps.cur() != 0 && ps.cur() != '>' {
			ps.next()
		}
	}
	if ps.cur() == '>' {
		ps.next()
	}
}

// parseExternalID is htmlParseExternalID (SYSTEM "…" | PUBLIC "…" ["…"]).
func (ps *parser) parseExternalID() {
	switch {
	case strings.EqualFold(string(ps.slice(0, 6)), "SYSTEM"):
		ps.p += 6
		ps.skipBlanks()
		ps.parseLiteral()
	case strings.EqualFold(string(ps.slice(0, 6)), "PUBLIC"):
		ps.p += 6
		ps.skipBlanks()
		ps.parseLiteral()
		ps.skipBlanks()
		if ps.cur() == '"' || ps.cur() == '\'' {
			ps.parseLiteral()
		}
	}
}

// parseLiteral is htmlParseSystemLiteral / htmlParsePubidLiteral: a quoted literal,
// consumed up to its closing quote (or the end of input).
func (ps *parser) parseLiteral() {
	q := ps.cur()
	if q != '"' && q != '\'' {
		return
	}
	ps.next()
	for ps.cur() != 0 && ps.cur() != q {
		ps.next()
	}
	if ps.cur() == q {
		ps.next()
	}
}
