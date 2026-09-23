package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/ritme/backend-go/cmd/contract/internal/ojson"
)

// defaultHeaders are captured into every golden when the response carries them.
var defaultHeaders = []string{
	"content-type", "retry-after", "x-ratelimit-limit", "x-ratelimit-remaining", "x-ratelimit-reset", "location",
}

// headerPatterns are volatile headers checked by pattern instead of value.
var headerPatterns = map[string]*regexp.Regexp{
	"x-ratelimit-reset": regexp.MustCompile(`^\d+$`),
}

func capturedHeaders(s *Step) []string {
	out := slices.Clone(defaultHeaders)
	for _, h := range s.CaptureHeaders {
		if h = strings.ToLower(h); !slices.Contains(out, h) {
			out = append(out, h)
		}
	}
	return out
}

func goldenPath(root string, c *Case) string {
	return filepath.Join(root, "golden", c.Group, c.ID+".json")
}

// obj builds an ordered object.
type obj struct{ v *ojson.Value }

func newObj() *obj {
	return &obj{v: &ojson.Value{Kind: ojson.Object, Keys: []string{}, Vals: []*ojson.Value{}}}
}

func (o *obj) set(k string, v *ojson.Value) *obj {
	o.v.Keys = append(o.v.Keys, k)
	o.v.Vals = append(o.v.Vals, v)
	return o
}

func (o *obj) str(k, s string) *obj { return o.set(k, ojson.NewString(s)) }

func num(n int) *ojson.Value { return &ojson.Value{Kind: ojson.Number, Num: strconv.Itoa(n)} }

// BuildGolden turns Laravel results into the golden document. Volatile values are
// masked; pattern mismatches and unexpected statuses are returned as problems.
func BuildGolden(c *Case, results []Result) ([]byte, []string) {
	var problems []string
	steps := &ojson.Value{Kind: ojson.Array, Arr: []*ojson.Value{}}
	for i := range results {
		s, res := &c.Steps[i], &results[i]
		label := fmt.Sprintf("step %d %s %s", i+1, res.Method, res.URL)
		if s.Status != res.Status {
			problems = append(problems, fmt.Sprintf("%s: expected status %d, Laravel answered %d: %s",
				label, s.Status, res.Status, snippet(res.Raw)))
		}
		req := newObj().str("method", res.Method).str("url", res.URL).str("persona", s.Persona)
		if s.Locale != "" {
			req.str("accept_language", s.Locale)
		}
		if s.Now != "none" {
			req.str("now", s.Now)
		}
		reqBody := res.GoldenBody
		if reqBody == nil {
			reqBody = res.Body
		}
		if len(reqBody) > 0 {
			if bv, err := ojson.Parse(reqBody); err == nil {
				req.set("body", bv)
			} else {
				req.str("raw_body", string(reqBody))
			}
		}
		st := newObj().set("request", req.v).set("status", num(res.Status))

		hdr := newObj()
		for _, name := range capturedHeaders(s) {
			val := res.Headers.Get(name)
			if val == "" {
				continue
			}
			switch {
			case slices.Contains(s.IgnoreHeaders, name):
				val = ojson.IgnoredPlaceholder
			case headerPatterns[name] != nil:
				if !headerPatterns[name].MatchString(val) {
					problems = append(problems, fmt.Sprintf("%s: header %s %q does not match %s", label, name, val, headerPatterns[name]))
				}
				val = "<pattern:" + headerPatterns[name].String() + ">"
			}
			hdr.str(name, val)
		}
		st.set("headers", hdr.v)

		switch {
		case s.IgnoreBody:
			st.set("body_ignored", &ojson.Value{Kind: ojson.Bool, Bool: true})
		default:
			bv, err := ojson.Parse(res.Raw)
			if err != nil {
				st.str("body_raw", string(res.Raw))
				break
			}
			for _, d := range ojson.Mask(bv, s.Rules) {
				problems = append(problems, label+": "+d.String())
			}
			st.set("body", bv)
			if s.StrictBytes {
				st.str("body_raw", string(res.Raw))
			}
		}
		steps.Arr = append(steps.Arr, st.v)
	}
	doc := newObj().str("case", c.Group+"/"+c.ID).set("steps", steps)
	if c.CompactGolden {
		return compactGolden(doc.v), problems
	}
	return append(ojson.Encode(doc.v, "  "), '\n'), problems
}

// compactGolden writes one line per step so big sweep goldens stay small but diffable.
func compactGolden(doc *ojson.Value) []byte {
	var b bytes.Buffer
	b.WriteString("{\n  \"case\": ")
	b.Write(ojson.Encode(doc.Vals[0], ""))
	b.WriteString(",\n  \"steps\": [\n")
	steps := doc.Vals[1].Arr
	for i, s := range steps {
		b.WriteString("    ")
		b.Write(ojson.Encode(s, ""))
		if i < len(steps)-1 {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	b.WriteString("  ]\n}\n")
	return b.Bytes()
}

// GoldenStep is the parsed form of one golden step.
type GoldenStep struct {
	Status      int
	Headers     map[string]string
	Body        *ojson.Value
	BodyRaw     *string
	BodyIgnored bool
}

// LoadGolden reads a golden file.
func LoadGolden(path string) ([]GoldenStep, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: repo-local goldens
	if err != nil {
		return nil, err
	}
	doc, err := ojson.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	stepsV := doc.Get("steps")
	if stepsV == nil || stepsV.Kind != ojson.Array {
		return nil, fmt.Errorf("%s: no steps", path)
	}
	out := make([]GoldenStep, 0, len(stepsV.Arr))
	for _, sv := range stepsV.Arr {
		var gs GoldenStep
		if st := sv.Get("status"); st != nil {
			gs.Status, _ = strconv.Atoi(st.Num)
		}
		gs.Headers = map[string]string{}
		if h := sv.Get("headers"); h != nil {
			for i, k := range h.Keys {
				gs.Headers[k] = h.Vals[i].Str
			}
		}
		gs.Body = sv.Get("body")
		if br := sv.Get("body_raw"); br != nil {
			s := br.Str
			gs.BodyRaw = &s
		}
		if bi := sv.Get("body_ignored"); bi != nil {
			gs.BodyIgnored = bi.Bool
		}
		out = append(out, gs)
	}
	return out, nil
}

// Mismatch is one difference found by the differ.
type Mismatch struct {
	Step int    // 1-based
	Kind string // status | header | body
	// Where is "status", "header:<name>", "body" or a JSON path ($.data.x).
	Where string
	Path  ojson.Path
	Msg   string
}

func (m Mismatch) String() string { return fmt.Sprintf("step %d %s: %s", m.Step, m.Where, m.Msg) }

// CompareStep diffs one Go result against its golden step.
func CompareStep(n int, s *Step, g *GoldenStep, res *Result) []Mismatch {
	var out []Mismatch
	add := func(kind, where string, p ojson.Path, msg string) {
		out = append(out, Mismatch{Step: n, Kind: kind, Where: where, Path: p, Msg: msg})
	}
	if g.Status != res.Status {
		add("status", "status", nil, fmt.Sprintf("want %d, got %d", g.Status, res.Status))
	}
	for _, name := range capturedHeaders(s) {
		want, inGolden := g.Headers[name]
		got := res.Headers.Get(name)
		switch {
		case slices.Contains(s.IgnoreHeaders, name):
			if inGolden && got == "" {
				add("header", "header:"+name, nil, "missing")
			}
		case !inGolden && got != "":
			add("header", "header:"+name, nil, fmt.Sprintf("unexpected (got %q)", got))
		case inGolden && headerPatterns[name] != nil:
			if !headerPatterns[name].MatchString(got) {
				add("header", "header:"+name, nil, fmt.Sprintf("got %q, want pattern %s", got, headerPatterns[name]))
			}
		case inGolden && want != got:
			add("header", "header:"+name, nil, fmt.Sprintf("want %q, got %q", want, got))
		}
	}
	if g.BodyIgnored {
		return out
	}
	if g.Body != nil {
		gv, err := ojson.Parse(res.Raw)
		if err != nil {
			add("body", "body", nil, "not JSON: "+snippet(res.Raw))
			return out
		}
		for _, d := range ojson.Compare(g.Body, gv, s.Rules) {
			add("body", d.Path.String(), d.Path, d.Msg)
		}
	}
	if g.BodyRaw != nil && (g.Body == nil || s.StrictBytes) && *g.BodyRaw != string(res.Raw) {
		add("body", "body", nil, fmt.Sprintf("bytes differ:\n      want %q\n      got  %q", clip(*g.BodyRaw), clip(string(res.Raw))))
	}
	return out
}

func snippet(b []byte) string { return clip(string(b)) }

func clip(s string) string {
	if len(s) > 300 {
		return s[:297] + "..."
	}
	return s
}
