package fertility

import (
	"embed"
	"io/fs"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// The fertility copy (enum and chance labels, controller messages, validation attribute
// names) is data: lang/<code>/fertility.json, one file per language, read through the platform
// translator. A language without the file (or a key) falls back to English (lang.FallbackLocale).
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

// T is the fertility line for key ("messages.day_saved") in locale.
func T(key, locale string) string { return translator().Trans("fertility."+key, nil, locale) }

// Label is the localized label of an enum value (group "lh", "intercourse", "chance", …);
// "" when the value has no label.
func Label(group, value, locale string) string {
	line, ok := translator().Get("fertility."+group, locale)
	if !ok {
		return ""
	}
	s, ok := phpval.Get(line, value)
	if !ok {
		return ""
	}
	str, _ := s.(string)
	return str
}

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	line, ok := translator().Get("fertility.attributes", locale)
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
