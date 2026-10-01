package contraception

import (
	"embed"
	"io/fs"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// The contraception copy (controller messages, validation attribute names, the titles of the care reminders a
// method creates) is data: lang/<code>/contraception.json, one file per language, read through the platform
// translator. A language without the file (or a key) falls back to English (lang.FallbackLocale). Missed-pill
// guidance is catalog content (`missed_pill_rules`), not here.
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

// T is the contraception line for key ("messages.method_saved") in locale.
func T(key, locale string) string { return translator().Trans("contraception."+key, nil, locale) }

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	line, ok := translator().Get("contraception.attributes", locale)
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
