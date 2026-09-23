// Package sanitizer ports App\Services\Content\HtmlSanitizer: the allow-list cleaner
// for admin-authored article HTML (clients render it with dangerouslySetInnerHTML) and
// its plain-text helper.
//
// PHP builds the DOM with libxml2 (DOMDocument::loadHTML) and prints it with
// DOMDocument::saveHTML, so the output depends on libxml2's HTML 4 parser and
// serialiser quirks. An HTML5 parser (x/net/html, bluemonday) builds a different tree
// for malformed input (<b><p>, stray </p>, tables without tbody) and prints it
// differently (<br/>, attribute escaping), so this package carries a port of the
// libxml2 2.9.14 code paths instead (parser.go, serialize.go) — the version in the
// production PHP image. Golden tests compare against PHP's own output.
package sanitizer

import (
	"slices"
	"strings"
)

// allowed maps the tags whose markup is kept to the attributes they may carry.
var allowed = map[string][]string{
	"p": nil, "br": nil, "strong": nil, "b": nil, "em": nil, "i": nil,
	"u": nil, "s": nil, "mark": nil, "sub": nil, "sup": nil, "span": nil,
	"h1": nil, "h2": nil, "h3": nil, "h4": nil, "h5": nil, "h6": nil,
	"ul": nil, "ol": nil, "li": nil, "blockquote": nil, "pre": nil, "code": nil,
	"hr": nil, "figure": nil, "figcaption": nil,
	"table": nil, "thead": nil, "tbody": nil, "tfoot": nil, "tr": nil,
	"th": {"colspan", "rowspan"}, "td": {"colspan", "rowspan"},
	"a":   {"href", "title", "target", "rel"},
	"img": {"src", "alt", "width", "height"},
}

// dropped tags are removed with everything inside them.
var dropped = []string{"script", "style", "iframe", "object", "embed", "form", "input", "button", "link", "meta", "svg"}

// allowedSchemes are the URL schemes a link or image may use.
var allowedSchemes = []string{"http", "https", "mailto", "tel"}

// Clean is HtmlSanitizer::clean: the sanitized HTML, ok=false (PHP null) when the input
// is blank or nothing renderable is left.
func Clean(html string) (string, bool) {
	if phpTrim(html) == "" {
		return "", false
	}
	doc := parseDocument(`<html><head><meta http-equiv="Content-Type" content="text/html; charset=utf-8"></head><body>` +
		html + `</body></html>`)
	body := findFirst(doc, "body")
	if body == nil {
		return "", false
	}
	cleanChildren(body)
	var out strings.Builder
	for _, c := range body.children {
		serialize(&out, c)
	}
	res := phpTrim(out.String())
	return res, res != ""
}

func phpTrim(s string) string { return strings.Trim(s, " \t\n\r\x00\x0B") }

// findFirst is getElementsByTagName($name)->item(0): document order.
func findFirst(n *node, name string) *node {
	for _, c := range n.children {
		if c.typ != elementNode {
			continue
		}
		if c.name == name {
			return c
		}
		if f := findFirst(c, name); f != nil {
			return f
		}
	}
	return nil
}

func removeChild(parent, child *node) {
	if i := slices.Index(parent.children, child); i >= 0 {
		parent.children = slices.Delete(parent.children, i, i+1)
	}
	child.parent = nil
}

func cleanChildren(n *node) {
	for _, child := range slices.Clone(n.children) {
		if child.typ == commentNode {
			removeChild(n, child)
			continue
		}
		if child.typ != elementNode {
			continue
		}
		tag := strings.ToLower(child.name)
		if slices.Contains(dropped, tag) {
			removeChild(n, child)
			continue
		}
		attrs, ok := allowed[tag]
		if !ok {
			unwrap(child)
			continue
		}
		cleanAttributes(child, tag, attrs)
		cleanChildren(child)
	}
}

// unwrap replaces an element with its (sanitized) children.
func unwrap(el *node) {
	cleanChildren(el)
	parent := el.parent
	if parent == nil {
		return
	}
	i := slices.Index(parent.children, el)
	kids := el.children
	for _, k := range kids {
		k.parent = parent
	}
	parent.children = slices.Concat(parent.children[:i], kids, parent.children[i+1:])
	el.children, el.parent = nil, nil
}

func cleanAttributes(el *node, tag string, allowedAttrs []string) {
	kept := el.attrs[:0:0]
	for _, a := range el.attrs {
		name := strings.ToLower(a.name)
		if !slices.Contains(allowedAttrs, name) {
			continue
		}
		if (name == "href" || name == "src") && !isSafeURL(a.value) {
			continue
		}
		kept = append(kept, a)
	}
	el.attrs = kept
	// A link that opens a new tab must not hand the opener to the target.
	if tag == "a" && getAttribute(el, "target") != "" {
		setAttribute(el, "rel", "noopener noreferrer")
	}
}

func getAttribute(el *node, name string) string {
	for _, a := range el.attrs {
		if a.name == name {
			return a.value
		}
	}
	return ""
}

func setAttribute(el *node, name, value string) {
	for i, a := range el.attrs {
		if a.name == name {
			el.attrs[i].value, el.attrs[i].hasValue = value, true
			return
		}
	}
	el.attrs = append(el.attrs, attr{name: name, value: value, hasValue: true})
}

// isSafeURL: relative and anchor URLs pass; absolute ones need an allowed scheme
// (parse_url's idea of a scheme, quirks included: "tel:0912" parses as a port).
func isSafeURL(raw string) bool {
	u := phpTrim(raw)
	if u == "" {
		return false
	}
	if strings.HasPrefix(u, "/") || strings.HasPrefix(u, "#") || strings.HasPrefix(u, "?") {
		return true
	}
	scheme, ok := parseURLScheme(u)
	if !ok || scheme == "" {
		return !strings.Contains(u, ":")
	}
	return slices.Contains(allowedSchemes, strings.ToLower(scheme))
}
