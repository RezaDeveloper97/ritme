package main

import (
	"fmt"
	"regexp"
	"strings"
)

// phpEnum is the part of a PHP string-backed enum that enumgen turns into Go.
type phpEnum struct {
	Name    string     // PHP enum name, reused as the Go type name
	Source  string     // path relative to the repository root, for the header comment
	Cases   []phpCase  // PHP case order
	Methods []phpTable // label / description / icon, in a fixed order
	// GenOptions is true when options($locale) maps over self::cases() (or the enum has a label but no
	// options()); false when the PHP options() iterates something else (CycleSubphase → hand-written).
	GenOptions bool
}

type phpCase struct {
	Name  string // PHP case name (UPPER_SNAKE)
	Value string // backed value, exact
}

// tableKind says how a string-returning method picks its text.
type tableKind int

const (
	kindSingle  tableKind = iota // one table; the locale (if any) is ignored
	kindLocale                   // one table per locale key plus a default ("" key) table
	kindIsValue                  // returns $this->value
)

// phpTable is one parsed string method (label, description, icon).
type phpTable struct {
	Method   string // PHP method name
	Localize bool   // the PHP signature takes $locale
	Kind     tableKind
	// Tables maps locale key → case name → text. kindSingle uses key "". kindLocale uses "" for the
	// default arm and e.g. "fa" for explicit arms.
	Tables map[string]map[string]string
}

var (
	reEnum   = regexp.MustCompile(`(?m)^\s*enum\s+(\w+)\s*:\s*string\b`)
	reCase   = regexp.MustCompile(`(?m)^\s*case\s+(\w+)\s*=\s*'((?:[^'\\]|\\.)*)'\s*;`)
	reMethod = regexp.MustCompile(`(?m)^\s*public\s+(static\s+)?function\s+(\w+)\s*\(([^)]*)\)\s*:\s*(\??[\w\\]+)\s*\{`)

	reReturnValue  = regexp.MustCompile(`^\s*return\s+\$this->value\s*;\s*$`)
	reMatchLocale  = regexp.MustCompile(`match\s*\(\s*\$locale\s*\)\s*\{`)
	reTernaryMatch = regexp.MustCompile(`\$locale\s*===\s*'(\w+)'\s*\?\s*match\s*\(\s*\$this\s*\)\s*\{`)
	reElseMatch    = regexp.MustCompile(`^\s*:\s*match\s*\(\s*\$this\s*\)\s*\{`)
	reMatchThis    = regexp.MustCompile(`match\s*\(\s*\$this\s*\)\s*\{`)
	reLocaleArmKey = regexp.MustCompile(`^[\s,]*(?:'(\w+)'|(default))\s*=>\s*match\s*\(\s*\$this\s*\)\s*\{`)

	phpStr       = `'((?:[^'\\]|\\.)*)'`
	armLHS       = `((?:self::\w+\s*,\s*)*self::\w+)\s*=>\s*`
	reArmTernary = regexp.MustCompile(armLHS + `\$locale\s*===\s*'(\w+)'\s*\?\s*` + phpStr + `\s*:\s*` + phpStr)
	reArmPlain   = regexp.MustCompile(armLHS + phpStr)
	reArrow      = regexp.MustCompile(`=>`)
	reSelfCase   = regexp.MustCompile(`self::(\w+)`)
)

// stringMethods are the instance methods enumgen generates, in output order.
var stringMethods = []string{"label", "description", "icon"}

func parseEnum(src, rel string) (*phpEnum, error) {
	m := reEnum.FindStringSubmatch(src)
	if m == nil {
		return nil, fmt.Errorf("%s: no string-backed enum found", rel)
	}
	e := &phpEnum{Name: m[1], Source: rel}

	// Cases live before the first method.
	head := src
	if loc := reMethod.FindStringIndex(src); loc != nil {
		head = src[:loc[0]]
	}
	for _, c := range reCase.FindAllStringSubmatch(head, -1) {
		e.Cases = append(e.Cases, phpCase{Name: c[1], Value: unquotePHP(c[2])})
	}
	if len(e.Cases) == 0 {
		return nil, fmt.Errorf("%s: enum %s has no cases", rel, e.Name)
	}

	type method struct{ static, params, ret, body string }
	methods := map[string]method{}
	for _, loc := range reMethod.FindAllStringSubmatchIndex(src, -1) {
		name := src[loc[4]:loc[5]]
		body, err := extractBlock(src, loc[1]-1)
		if err != nil {
			return nil, fmt.Errorf("%s: %s(): %w", rel, name, err)
		}
		methods[name] = method{
			static: strings.TrimSpace(src[max(loc[2], 0):max(loc[3], 0)]),
			params: src[loc[6]:loc[7]],
			ret:    src[loc[8]:loc[9]],
			body:   body,
		}
	}

	for _, name := range stringMethods {
		mm, ok := methods[name]
		if !ok || mm.static != "" || mm.ret != "string" {
			continue
		}
		t, err := parseStringMethod(name, mm.params, mm.body, e.Cases)
		if err != nil {
			return nil, fmt.Errorf("%s: %s(): %w", rel, name, err)
		}
		e.Methods = append(e.Methods, *t)
	}

	hasLabel := false
	for _, t := range e.Methods {
		if t.Method == "label" {
			hasLabel = true
		}
	}
	if opt, ok := methods["options"]; ok {
		e.GenOptions = hasLabel && strings.Contains(opt.body, "self::cases()")
	} else {
		e.GenOptions = hasLabel
	}
	return e, nil
}

func parseStringMethod(name, params, body string, cases []phpCase) (*phpTable, error) {
	t := &phpTable{
		Method:   name,
		Localize: strings.Contains(params, "$locale"),
		Tables:   map[string]map[string]string{},
	}

	switch {
	case reReturnValue.MatchString(body):
		t.Kind = kindIsValue
		return t, nil

	case reMatchLocale.MatchString(body):
		// match ($locale) { 'fa' => match ($this) {…}, default => match ($this) {…} }
		t.Kind = kindLocale
		loc := reMatchLocale.FindStringIndex(body)
		outer, err := extractBlock(body, loc[1]-1)
		if err != nil {
			return nil, err
		}
		rest := outer
		for strings.TrimSpace(strings.Trim(rest, ",\n\t ")) != "" {
			km := reLocaleArmKey.FindStringSubmatchIndex(rest)
			if km == nil {
				return nil, fmt.Errorf("unparseable match($locale) arm near %q", head(rest))
			}
			key := ""
			if km[2] >= 0 {
				key = rest[km[2]:km[3]]
			}
			inner, err := extractBlock(rest, km[1]-1)
			if err != nil {
				return nil, err
			}
			tbl, err := parseArms(inner, cases)
			if err != nil {
				return nil, err
			}
			t.Tables[key] = tbl
			rest = rest[km[1]+len(inner)+1:]
		}

	case reTernaryMatch.MatchString(body):
		// $locale === 'fa' ? match ($this) {…} : match ($this) {…}
		t.Kind = kindLocale
		loc := reTernaryMatch.FindStringSubmatchIndex(body)
		key := body[loc[2]:loc[3]]
		first, err := extractBlock(body, loc[1]-1)
		if err != nil {
			return nil, err
		}
		after := body[loc[1]+len(first)+1:]
		em := reElseMatch.FindStringIndex(after)
		if em == nil {
			return nil, fmt.Errorf("ternary without a ': match ($this)' else branch")
		}
		second, err := extractBlock(after, em[1]-1)
		if err != nil {
			return nil, err
		}
		if t.Tables[key], err = parseArms(first, cases); err != nil {
			return nil, err
		}
		if t.Tables[""], err = parseArms(second, cases); err != nil {
			return nil, err
		}

	case reMatchThis.MatchString(body):
		loc := reMatchThis.FindStringIndex(body)
		block, err := extractBlock(body, loc[1]-1)
		if err != nil {
			return nil, err
		}
		if reArmTernary.MatchString(block) {
			// self::X => $locale === 'fa' ? '…' : '…'  (MessageMode)
			t.Kind = kindLocale
			if err := parseTernaryArms(block, cases, t.Tables); err != nil {
				return nil, err
			}
		} else {
			t.Kind = kindSingle
			if t.Tables[""], err = parseArms(block, cases); err != nil {
				return nil, err
			}
		}

	default:
		return nil, fmt.Errorf("unsupported body shape: %q", head(body))
	}
	return t, nil
}

// parseArms reads `self::A, self::B => 'text',` arms and checks every case is covered exactly once.
func parseArms(block string, cases []phpCase) (map[string]string, error) {
	out := map[string]string{}
	arms := reArmPlain.FindAllStringSubmatch(block, -1)
	if n := len(reArrow.FindAllString(block, -1)); n != len(arms) {
		return nil, fmt.Errorf("parsed %d of %d match arms in %q", len(arms), n, head(block))
	}
	for _, a := range arms {
		for _, c := range reSelfCase.FindAllStringSubmatch(a[1], -1) {
			if _, dup := out[c[1]]; dup {
				return nil, fmt.Errorf("case %s listed twice", c[1])
			}
			out[c[1]] = unquotePHP(a[2])
		}
	}
	return out, checkCoverage(out, cases)
}

func parseTernaryArms(block string, cases []phpCase, tables map[string]map[string]string) error {
	arms := reArmTernary.FindAllStringSubmatch(block, -1)
	if n := len(reArrow.FindAllString(block, -1)); n != len(arms) {
		return fmt.Errorf("parsed %d of %d ternary arms", len(arms), n)
	}
	for _, a := range arms {
		key := a[2]
		if tables[key] == nil {
			tables[key] = map[string]string{}
		}
		if tables[""] == nil {
			tables[""] = map[string]string{}
		}
		for _, c := range reSelfCase.FindAllStringSubmatch(a[1], -1) {
			tables[key][c[1]] = unquotePHP(a[3])
			tables[""][c[1]] = unquotePHP(a[4])
		}
	}
	for _, tbl := range tables {
		if err := checkCoverage(tbl, cases); err != nil {
			return err
		}
	}
	return nil
}

func checkCoverage(tbl map[string]string, cases []phpCase) error {
	known := map[string]bool{}
	for _, c := range cases {
		known[c.Name] = true
		if _, ok := tbl[c.Name]; !ok {
			return fmt.Errorf("case %s has no arm", c.Name)
		}
	}
	for name := range tbl {
		if !known[name] {
			return fmt.Errorf("arm for unknown case %s", name)
		}
	}
	return nil
}

// extractBlock returns the text between the '{' at open and its matching '}', skipping quoted
// strings and comments.
func extractBlock(s string, open int) (string, error) {
	if open < 0 || open >= len(s) || s[open] != '{' {
		return "", fmt.Errorf("expected '{' at offset %d", open)
	}
	depth := 0
	for i := open; i < len(s); i++ {
		switch c := s[i]; c {
		case '\'', '"':
			for i++; i < len(s) && s[i] != c; i++ {
				if s[i] == '\\' {
					i++
				}
			}
		case '/':
			if i+1 < len(s) && s[i+1] == '/' {
				for i < len(s) && s[i] != '\n' {
					i++
				}
			} else if i+1 < len(s) && s[i+1] == '*' {
				end := strings.Index(s[i+2:], "*/")
				if end < 0 {
					return "", fmt.Errorf("unterminated comment")
				}
				i += end + 3
			}
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[open+1 : i], nil
			}
		}
	}
	return "", fmt.Errorf("unbalanced braces")
}

// unquotePHP decodes a single-quoted PHP string body: only \' and \\ are escapes.
func unquotePHP(s string) string {
	return strings.NewReplacer(`\'`, `'`, `\\`, `\`).Replace(s)
}

func head(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 80 {
		return s[:80] + "…"
	}
	return s
}
