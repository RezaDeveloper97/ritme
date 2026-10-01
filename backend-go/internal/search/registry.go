package search

import (
	"embed"
	"io/fs"
	"strings"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// The search copy is data: lang/<code>/search.json (validation attribute names, and the title, subtitle and
// keywords of every registry destination), read through the platform translator. A language without the
// file (or a key) falls back to English (lang.FallbackLocale), like internal/pelvic.
//
//go:embed lang/*/*.json
var langFS embed.FS

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err) // embedded files are checked by the package tests
	}
	return t
})

// T is the search line for key in locale ("" when missing).
func T(key, locale string) string {
	line, ok := translator().Get("search."+key, locale)
	if s, isStr := line.(string); ok && isStr {
		return s
	}
	return ""
}

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	line, ok := translator().Get("search.attributes", locale)
	m, isMap := line.(phpval.Map)
	if !ok || !isMap {
		return nil
	}
	var kv []string
	for _, k := range m.Keys() {
		if v, ok := m.Get(k); ok {
			if s, ok := v.(string); ok {
				kv = append(kv, k, s)
			}
		}
	}
	return kv
}

// destination is a fixed in-app place the search can open (a program or a service screen).
type destination struct {
	code  string
	route string
}

// programRegistry are the care programs that exist. Condition programs (endometriosis pain diary, PMDD,
// heavy bleeding — CB-COND-01) are appended here when they ship, with their copy in lang/*/search.json.
var programRegistry = []destination{
	{code: "contraception", route: "/contraception"},
	{code: "pelvic", route: "/programs/pelvic"},
}

// serviceRegistry are the services that exist today (Services › «همین حالا»). Doctors, labs, record,
// vitals, insurance and city services / the directory join when their B-N7 / CB tasks ship.
var serviceRegistry = []destination{
	{code: "checkups", route: "/checkups"},
	{code: "reminders", route: "/reminders"},
}

// keywords splits a comma list (Latin or Arabic comma).
func keywords(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '،' })
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// registryHits matches the destinations of one registry section ("programs" | "services").
func registryHits(section, hitType string, list []destination, q Query, m Matcher, startOrder int) []Hit {
	var hits []Hit
	for i, d := range list {
		base := section + "." + d.code + "."
		title := T(base+"title", q.Locale)
		if title == "" {
			continue
		}
		sub := T(base+"subtitle", q.Locale)
		secondary := append([]string{sub}, keywords(T(base+"keywords", q.Locale))...)
		rank := m.Best(title, secondary...)
		if rank == 0 {
			continue
		}
		hits = append(hits, Hit{
			Type: hitType, ID: d.code, Title: title, Subtitle: sub, Route: d.route,
			rank: rank, order: startOrder + i,
		})
	}
	return hits
}

// programs is the programs group.
func programs(q Query, m Matcher) []Hit {
	return registryHits("programs", "program", programRegistry, q, m, 0)
}
