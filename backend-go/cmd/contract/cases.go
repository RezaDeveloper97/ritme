package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/ritme/backend-go/cmd/contract/internal/ojson"
)

// DefaultNow is the X-Test-Now every request carries unless a case overrides it
// (a time on ContractFixtureSeeder::CONTRACT_TODAY, see docs/go-migration/contract.md).
const DefaultNow = "2026-09-23T10:00:00+03:30"

// Special persona names (everything else is a ContractFixtureSeeder persona key).
const (
	PersonaAnon = "anon" // no Authorization header
)

// CaseFile is contract/cases/<group>.yaml.
type CaseFile struct {
	Group string `yaml:"group"`
	// CompactGolden writes each step body on one line (big sweep groups).
	CompactGolden bool       `yaml:"compact_golden"`
	Defaults      CaseSpec   `yaml:"defaults"`
	Cases         []CaseSpec `yaml:"cases"`
}

// StrList accepts a scalar or a sequence.
type StrList []string

// UnmarshalYAML implements yaml.Unmarshaler.
func (s *StrList) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		*s = StrList{n.Value}
		return nil
	}
	var list []string
	if err := n.Decode(&list); err != nil {
		return err
	}
	*s = list
	return nil
}

// KV is an ordered string map (query strings, request headers).
type KV []KVPair

// KVPair is one KV entry.
type KVPair struct{ Key, Value string }

// UnmarshalYAML implements yaml.Unmarshaler.
func (kv *KV) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: expected a mapping", n.Line)
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		*kv = append(*kv, KVPair{Key: n.Content[i].Value, Value: n.Content[i+1].Value})
	}
	return nil
}

// JSONBody is a request body written as YAML; scalars keep their YAML type
// (quoted → string, 65.5 → number, 2026-09-23 → string, ~ → null).
type JSONBody struct{ V *ojson.Value }

// UnmarshalYAML implements yaml.Unmarshaler.
func (b *JSONBody) UnmarshalYAML(n *yaml.Node) error {
	v, err := yamlToJSON(n)
	b.V = v
	return err
}

func yamlToJSON(n *yaml.Node) (*ojson.Value, error) {
	switch n.Kind {
	case yaml.DocumentNode:
		return yamlToJSON(n.Content[0])
	case yaml.AliasNode:
		return yamlToJSON(n.Alias)
	case yaml.SequenceNode:
		v := &ojson.Value{Kind: ojson.Array, Arr: []*ojson.Value{}}
		for _, c := range n.Content {
			el, err := yamlToJSON(c)
			if err != nil {
				return nil, err
			}
			v.Arr = append(v.Arr, el)
		}
		return v, nil
	case yaml.MappingNode:
		v := &ojson.Value{Kind: ojson.Object, Keys: []string{}, Vals: []*ojson.Value{}}
		for i := 0; i+1 < len(n.Content); i += 2 {
			el, err := yamlToJSON(n.Content[i+1])
			if err != nil {
				return nil, err
			}
			v.Keys = append(v.Keys, n.Content[i].Value)
			v.Vals = append(v.Vals, el)
		}
		return v, nil
	case yaml.ScalarNode:
		switch n.ShortTag() {
		case "!!null":
			return &ojson.Value{Kind: ojson.Null}, nil
		case "!!bool":
			var b bool
			if err := n.Decode(&b); err != nil {
				return nil, err
			}
			return &ojson.Value{Kind: ojson.Bool, Bool: b}, nil
		case "!!int", "!!float":
			if _, err := ojson.Parse([]byte(n.Value)); err != nil {
				return nil, fmt.Errorf("line %d: %q is not a JSON number (quote it)", n.Line, n.Value)
			}
			return &ojson.Value{Kind: ojson.Number, Num: n.Value}, nil
		default: // !!str, !!timestamp, …
			return ojson.NewString(n.Value), nil
		}
	}
	return nil, fmt.Errorf("line %d: unsupported YAML node", n.Line)
}

// StepSpec is one HTTP exchange as written in YAML.
type StepSpec struct {
	Method  string    `yaml:"method"`
	Path    string    `yaml:"path"`
	Query   KV        `yaml:"query"`
	Body    *JSONBody `yaml:"body"`
	RawBody *string   `yaml:"raw_body"`
	// ContentType of RawBody (Body is always application/json).
	ContentType string `yaml:"content_type"`
	// Headers are extra request headers; an empty value removes a default header
	// (Accept, Accept-Language, Authorization, X-Test-Now).
	Headers KV `yaml:"headers"`
	// As overrides the case persona for this step.
	As string `yaml:"as"`
	// Now overrides X-Test-Now; "none" omits the header.
	Now string `yaml:"now"`
	// Status is the expected Laravel status (recording fails on mismatch).
	Status int `yaml:"status"`
	// Capture binds {{name}} to a body value (selector) for later steps.
	Capture map[string]string `yaml:"capture"`
	// OTPFor reads the latest OTP code of that mobile from the target DB into {{otp}}
	// before the request is sent.
	OTPFor string `yaml:"otp_for"`
	// StrictBytes compares the raw body byte-for-byte (framework error bodies).
	StrictBytes bool `yaml:"strict_bytes"`
	// IgnoreBody skips the body entirely (non-API responses such as /up).
	IgnoreBody bool              `yaml:"ignore_body"`
	Ignore     []string          `yaml:"ignore"`
	Patterns   map[string]string `yaml:"patterns"`
	// CaptureHeaders adds response headers to the golden (Content-Type, Retry-After,
	// X-RateLimit-* and Location are always captured when present).
	CaptureHeaders []string `yaml:"capture_headers"`
}

// CaseSpec is one entry of a case file (defaults use the same shape).
type CaseSpec struct {
	Name string `yaml:"name"`
	Doc  string `yaml:"doc"`
	// Persona and Locales expand the case: one golden per persona × locale.
	// Locale "none" sends no Accept-Language.
	Persona StrList `yaml:"persona"`
	Locales StrList `yaml:"locales"`
	// Reset forces a DB reset before the case even if nothing was written before.
	Reset bool `yaml:"reset"`
	// Sweep turns a single-step case into one step per value ({{<var>}} in path/query).
	Sweep *Sweep `yaml:"sweep"`

	StepSpec `yaml:",inline"`
	Steps    []StepSpec `yaml:"steps"`
}

// Sweep generates values for a variable: explicit Values, or Days consecutive dates
// (Y-m-d) starting at From.
type Sweep struct {
	Var    string   `yaml:"var"`
	Values []string `yaml:"values"`
	From   string   `yaml:"from"`
	Days   int      `yaml:"days"`
}

func (s *Sweep) values() ([]string, error) {
	if len(s.Values) > 0 {
		return s.Values, nil
	}
	start, err := time.Parse(time.DateOnly, s.From)
	if err != nil || s.Days <= 0 {
		return nil, fmt.Errorf("sweep %q: need values or from (Y-m-d) + days", s.Var)
	}
	out := make([]string, s.Days)
	for i := range out {
		out[i] = start.AddDate(0, 0, i).Format(time.DateOnly)
	}
	return out, nil
}

// Case is one expanded case: a golden file.
type Case struct {
	Group         string
	ID            string // file name without .json
	Persona       string
	Locale        string // "" = no Accept-Language
	Reset         bool
	CompactGolden bool
	Steps         []Step
}

// Step is a fully resolved step (templates in path/query/body are still {{…}}).
type Step struct {
	StepSpec
	Persona string
	Locale  string
	Rules   *ojson.Rules
	// HeaderRules applies to header names (lower case) under the "header:" prefix.
	IgnoreHeaders []string
}

// Writes reports whether any step can change the database.
func (c *Case) Writes() bool {
	for _, s := range c.Steps {
		if s.Method != http.MethodGet && s.Method != http.MethodHead {
			return true
		}
	}
	return false
}

var (
	idPart    = regexp.MustCompile(`^[a-z0-9_-]+$`)
	groupName = regexp.MustCompile(`^[a-z0-9-]+$`)
)

// LoadGroups reads every contract/cases/*.yaml and returns the groups (sorted).
func LoadGroups(dir string) (map[string]*CaseFile, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	out := map[string]*CaseFile{}
	for _, p := range paths {
		f, err := LoadCaseFile(p)
		if err != nil {
			return nil, err
		}
		want := strings.TrimSuffix(filepath.Base(p), ".yaml")
		if f.Group != want {
			return nil, fmt.Errorf("%s: group %q must match the file name", p, f.Group)
		}
		out[f.Group] = f
	}
	return out, nil
}

// LoadCaseFile parses one case file.
func LoadCaseFile(path string) (*CaseFile, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: repo-local case files
	if err != nil {
		return nil, err
	}
	var f CaseFile
	dec := yaml.NewDecoder(strings.NewReader(string(raw)))
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if !groupName.MatchString(f.Group) {
		return nil, fmt.Errorf("%s: bad group name %q", path, f.Group)
	}
	return &f, nil
}

// SelectGroups resolves a --routes value: "all", or comma-separated group names /
// globs ("cycle*").
func SelectGroups(all map[string]*CaseFile, routes string) ([]string, error) {
	names := make([]string, 0, len(all))
	for n := range all {
		names = append(names, n)
	}
	sort.Strings(names)
	if routes == "" || routes == "all" {
		return names, nil
	}
	var out []string
	for _, pat := range strings.Split(routes, ",") {
		pat = strings.TrimSpace(pat)
		if pat == "" {
			continue
		}
		matched := false
		for _, n := range names {
			if ok, _ := filepath.Match(pat, n); ok && !slices.Contains(out, n) {
				out = append(out, n)
				matched = true
			}
		}
		if !matched {
			return nil, fmt.Errorf("no case group matches %q (have: %s)", pat, strings.Join(names, ", "))
		}
	}
	return out, nil
}

// Expand turns a case file into golden-level cases.
func (f *CaseFile) Expand() ([]*Case, error) {
	var out []*Case
	seen := map[string]bool{}
	for i := range f.Cases {
		spec := f.Cases[i]
		if !idPart.MatchString(spec.Name) {
			return nil, fmt.Errorf("group %s: case #%d: bad name %q", f.Group, i+1, spec.Name)
		}
		personas := firstNonEmpty(spec.Persona, f.Defaults.Persona, StrList{PersonaAnon})
		locales := firstNonEmpty(spec.Locales, f.Defaults.Locales, StrList{"none"})
		for _, persona := range personas {
			for _, locale := range locales {
				c, err := f.expandOne(&spec, persona, locale, len(personas) > 1, len(locales) > 1)
				if err != nil {
					return nil, fmt.Errorf("group %s: case %s: %w", f.Group, spec.Name, err)
				}
				if seen[c.ID] {
					return nil, fmt.Errorf("group %s: duplicate case id %s", f.Group, c.ID)
				}
				seen[c.ID] = true
				out = append(out, c)
			}
		}
	}
	return out, nil
}

func (f *CaseFile) expandOne(spec *CaseSpec, persona, locale string, pInID, lInID bool) (*Case, error) {
	id := spec.Name
	if pInID {
		id += "." + persona
	}
	if lInID {
		id += "." + locale
	}
	if locale == "none" {
		locale = ""
	}
	c := &Case{Group: f.Group, ID: id, Persona: persona, Locale: locale, Reset: spec.Reset,
		CompactGolden: f.CompactGolden}

	specs := spec.Steps
	caseLevel := spec.StepSpec // settings every step inherits
	if len(specs) == 0 {
		caseLevel = StepSpec{}
		if spec.Method == "" || spec.Path == "" {
			return nil, errors.New("needs method+path or steps")
		}
		specs = []StepSpec{spec.StepSpec}
	} else if spec.Method != "" || spec.Path != "" {
		return nil, errors.New("use either method/path or steps, not both")
	}
	if spec.Sweep != nil {
		if len(specs) != 1 {
			return nil, errors.New("sweep needs a single-step case")
		}
		vals, err := spec.Sweep.values()
		if err != nil {
			return nil, err
		}
		tmpl := specs[0]
		specs = nil
		for _, v := range vals {
			s := tmpl
			s.Path = strings.ReplaceAll(s.Path, "{{"+spec.Sweep.Var+"}}", v)
			s.Query = nil
			for _, kv := range tmpl.Query {
				s.Query = append(s.Query, KVPair{kv.Key, strings.ReplaceAll(kv.Value, "{{"+spec.Sweep.Var+"}}", v)})
			}
			specs = append(specs, s)
		}
	}

	for _, s := range specs {
		s.Method = strings.ToUpper(s.Method)
		if s.Method == "" || !strings.HasPrefix(s.Path, "/") {
			return nil, fmt.Errorf("step %s %q: method and absolute path required", s.Method, s.Path)
		}
		if s.Status == 0 {
			return nil, fmt.Errorf("step %s %s: expected status required", s.Method, s.Path)
		}
		if s.Body != nil && s.RawBody != nil {
			return nil, fmt.Errorf("step %s %s: body and raw_body are exclusive", s.Method, s.Path)
		}
		// Case-level settings are inherited by every step; defaults by every case.
		s.StrictBytes = s.StrictBytes || caseLevel.StrictBytes || f.Defaults.StrictBytes
		s.IgnoreBody = s.IgnoreBody || caseLevel.IgnoreBody
		if s.Now == "" {
			s.Now = firstString(caseLevel.Now, f.Defaults.Now, DefaultNow)
		}
		s.Headers = append(append(KV{}, f.Defaults.Headers...), append(caseLevel.Headers, s.Headers...)...)
		s.CaptureHeaders = concat(f.Defaults.CaptureHeaders, caseLevel.CaptureHeaders, s.CaptureHeaders)

		var bodyIgnore, headerIgnore []string
		for _, sel := range concat(f.Defaults.Ignore, caseLevel.Ignore, s.Ignore) {
			if h, ok := strings.CutPrefix(sel, "header:"); ok {
				headerIgnore = append(headerIgnore, strings.ToLower(h))
				continue
			}
			bodyIgnore = append(bodyIgnore, sel)
		}
		patterns := map[string]string{}
		for _, m := range []map[string]string{f.Defaults.Patterns, caseLevel.Patterns, s.Patterns} {
			for k, v := range m {
				patterns[k] = v
			}
		}
		rules, err := ojson.NewRules(bodyIgnore, patterns)
		if err != nil {
			return nil, err
		}
		st := Step{StepSpec: s, Persona: firstString(s.As, persona), Locale: locale,
			Rules: rules, IgnoreHeaders: headerIgnore}
		c.Steps = append(c.Steps, st)
	}
	return c, nil
}

// URL renders path + query with {{var}} substitution.
func (s *Step) URL(vars map[string]string) string {
	p := subst(s.Path, vars)
	if len(s.Query) == 0 {
		return p
	}
	parts := make([]string, 0, len(s.Query))
	for _, kv := range s.Query {
		parts = append(parts, url.QueryEscape(kv.Key)+"="+url.QueryEscape(subst(kv.Value, vars)))
	}
	return p + "?" + strings.Join(parts, "&")
}

var tmplVar = regexp.MustCompile(`\{\{([a-z0-9_]+)\}\}`)

func subst(s string, vars map[string]string) string {
	return tmplVar.ReplaceAllStringFunc(s, func(m string) string {
		if v, ok := vars[m[2:len(m)-2]]; ok {
			return v
		}
		return m
	})
}

// substValue applies {{var}} substitution to every string in a JSON body. A string
// that is exactly "{{var}}" and whose value looks numeric stays a string: request
// bodies are typed by the YAML author.
func substValue(v *ojson.Value, vars map[string]string) *ojson.Value {
	switch v.Kind {
	case ojson.String:
		return ojson.NewString(subst(v.Str, vars))
	case ojson.Array:
		out := &ojson.Value{Kind: ojson.Array, Arr: make([]*ojson.Value, len(v.Arr))}
		for i, el := range v.Arr {
			out.Arr[i] = substValue(el, vars)
		}
		return out
	case ojson.Object:
		out := &ojson.Value{Kind: ojson.Object, Keys: v.Keys, Vals: make([]*ojson.Value, len(v.Vals))}
		for i, el := range v.Vals {
			out.Vals[i] = substValue(el, vars)
		}
		return out
	}
	return v
}

func firstNonEmpty(lists ...StrList) StrList {
	for _, l := range lists {
		if len(l) > 0 {
			return l
		}
	}
	return nil
}

func firstString(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func concat(lists ...[]string) []string {
	var out []string
	for _, l := range lists {
		out = append(out, l...)
	}
	return out
}
