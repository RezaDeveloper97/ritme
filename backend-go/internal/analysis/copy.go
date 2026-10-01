package analysis

import (
	"strconv"
	"strings"

	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Namespace is the translation namespace of the analysis sentences (resources/translations/<code>/
// analysis.json, a byte-identical copy of frontend/messages/<code>/analysis.json). Templates use ICU-style
// {name} placeholders, so the client can render the same keys itself from `key` + `params`.
const Namespace = "analysis"

// Arg kinds: how a placeholder value is written.
const (
	argNum     = iota // a number in the locale's digits (≤ 1 decimal)
	argSymptom        // a symptom key → its log-taxonomy label
	argPhase          // a phase code → analysis phases.<code>
)

// Arg is one placeholder of a Phrase.
type Arg struct {
	Name  string
	Value any // int | float64 | string (a code for symptom / phase args)
	kind  int
}

// Num, Symptom and PhaseArg build Args.
func Num(name string, v any) Arg      { return Arg{Name: name, Value: v, kind: argNum} }
func Symptom(name, key string) Arg    { return Arg{Name: name, Value: key, kind: argSymptom} }
func PhaseArg(name, phase string) Arg { return Arg{Name: name, Value: phase, kind: argPhase} }

// Phrase is one localized sentence: a key under the namespace and its placeholder values. Params carry
// codes (symptom keys, phase codes) and plain numbers, never rendered text.
type Phrase struct {
	Key  string
	Args []Arg
}

// Copy renders Phrases from the analysis namespace and labels symptoms from the log-taxonomy namespace,
// both already resolved for the request locale with the default language underneath
// (i18n.TranslationStore.NamespaceMessages). A missing template renders "" (the client falls back to
// key + params); a missing label falls back to the code.
type Copy struct {
	ns, taxonomy any
}

// NewCopy wraps the two namespaces.
func NewCopy(analysisNS, taxonomyNS any) *Copy { return &Copy{ns: analysisNS, taxonomy: taxonomyNS} }

func getString(ns any, path string) (string, bool) {
	if v, ok := phpval.Get(ns, path); ok {
		if s, isStr := v.(string); isStr && s != "" {
			return s, true
		}
	}
	return "", false
}

func (c *Copy) text(path, fallback string) string {
	if c == nil {
		return fallback
	}
	if s, ok := getString(c.ns, path); ok {
		return s
	}
	return fallback
}

// SymptomLabel is a symptom's name: the analysis override symptom_names.<c>.<p>.<i> (pain locations read
// as a symptom — «سر» → «سردرد»), else the taxonomy label categories.<c>.params.<p>.options.<i>, else the key.
func (c *Copy) SymptomLabel(key string) string {
	if c == nil {
		return key
	}
	if s, ok := getString(c.ns, "symptom_names."+key); ok {
		return s
	}
	parts := strings.SplitN(key, ".", 3)
	if len(parts) != 3 {
		return key
	}
	if s, ok := getString(c.taxonomy, "categories."+parts[0]+".params."+parts[1]+".options."+parts[2]); ok {
		return s
	}
	return key
}

// Number writes v (an int, or a float rounded to 1 decimal, trailing .0 dropped) in the locale's digits
// and decimal separator (format.digits / format.decimal).
func (c *Copy) Number(v any) string {
	var s string
	switch n := v.(type) {
	case int:
		s = strconv.Itoa(n)
	case float64:
		s = strconv.FormatFloat(round(n, 1), 'f', -1, 64)
	default:
		return ""
	}
	digits := []rune(c.text("format.digits", "0123456789"))
	dec := c.text("format.decimal", ".")
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9' && len(digits) == 10:
			b.WriteRune(digits[r-'0'])
		case r == '.':
			b.WriteString(dec)
		case r == '-':
			b.WriteRune('−')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (c *Copy) arg(a Arg) string {
	switch a.kind {
	case argSymptom:
		return c.SymptomLabel(a.Value.(string))
	case argPhase:
		p := a.Value.(string)
		return c.text("phases."+p, p)
	}
	return c.Number(a.Value)
}

// Render is the sentence of p ("" when the template is missing or there is no Copy).
func (c *Copy) Render(p Phrase) string {
	tpl := c.text(p.Key, "")
	if tpl == "" {
		return ""
	}
	for _, a := range p.Args {
		tpl = strings.ReplaceAll(tpl, "{"+a.Name+"}", c.arg(a))
	}
	return tpl
}

// List joins parts with format.list_separator and format.list_last («الف، ب و ج» / "a, b and c").
func (c *Copy) List(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	}
	sep, last := c.text("format.list_separator", ", "), c.text("format.list_last", " and ")
	return strings.Join(parts[:len(parts)-1], sep) + last + parts[len(parts)-1]
}

// JSON is {key, params, text}.
func (p Phrase) JSON(c *Copy) *jsonx.OrderedMap {
	params := jsonx.NewObject()
	for _, a := range p.Args {
		params.Set(a.Name, a.Value)
	}
	return jsonx.Obj("key", p.Key, "params", params, "text", c.Render(p))
}

// phraseOrNil is p.JSON or null for a nil *Phrase.
func phraseOrNil(p *Phrase, c *Copy) any {
	if p == nil {
		return nil
	}
	return p.JSON(c)
}
