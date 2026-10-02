package postpartum

import (
	"embed"
	"io/fs"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// The postpartum copy (controller messages, validation attribute names, phase labels and the EPDS questions) is data:
// lang/<code>/postpartum.json, one file per language, read through the platform translator. A language without the
// file (or a key) falls back to English (lang.FallbackLocale). The EPDS wording is marked `epds.needs_review`
// [needs clinical review]; the tips, alerts and safety texts are admin message_contents (package guide).
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

// T is the postpartum line for key ("messages.check_saved") in locale.
func T(key, locale string) string { return translator().Trans("postpartum."+key, nil, locale) }

// Tp is T with :placeholders.
func Tp(key string, params map[string]string, locale string) string {
	return translator().Trans("postpartum."+key, params, locale)
}

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	line, ok := translator().Get("postpartum.attributes", locale)
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

// line is a string line of the postpartum file ("" when missing).
func line(key, locale string) string {
	v, ok := translator().Get("postpartum."+key, locale)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// optionLabels are the four option labels of an EPDS item in locale.
func optionLabels(code, locale string) []string {
	v, _ := translator().Get("postpartum.epds.items."+code+".options", locale)
	_, vals := phpval.Entries(v)
	out := make([]string, 0, len(vals))
	for _, x := range vals {
		s, _ := x.(string)
		out = append(out, s)
	}
	return out
}
