package ivf

import (
	"embed"
	"io/fs"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// The IVF copy (controller messages, validation messages and attribute names, the titles of the care appointments a
// cycle creates) is data: lang/<code>/ivf.json, one file per language, read through the platform translator. A
// language without the file (or a key) falls back to English (lang.FallbackLocale). Clinical copy and lists are
// catalog content (Group* constants), not here.
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

// T is the IVF line for key ("messages.med_added") in locale.
func T(key, locale string) string { return translator().Trans("ivf."+key, nil, locale) }

// attributeName is the validation attribute name of field in locale (the field itself when unnamed).
func attributeName(field, locale string) string {
	line, ok := translator().Get("ivf.attributes", locale)
	if m, isMap := line.(phpval.Map); ok && isMap {
		if v, ok := m.Get(field); ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return field
}

// fieldMessage is validation.<key> in locale with :attribute replaced by field's name.
func fieldMessage(field, key, locale string) string {
	return translator().Trans("ivf.validation."+key, map[string]string{"attribute": attributeName(field, locale)}, locale)
}

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	line, ok := translator().Get("ivf.attributes", locale)
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
