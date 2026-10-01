package menopause

import (
	"embed"
	"io/fs"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// The menopause copy (controller messages, validation attribute names and the pattern sentences) is data:
// lang/<code>/menopause.json, one file per language, read through the platform translator. A language without the
// file (or a key) falls back to English (lang.FallbackLocale). The pattern sentences are marked
// `patterns.needs_review` [needs clinical review]; lists and clinical copy are catalog content, not here.
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

// T is the menopause line for key ("messages.score_saved") in locale.
func T(key, locale string) string { return translator().Trans("menopause."+key, nil, locale) }

// Tp is T with :placeholders.
func Tp(key string, params map[string]string, locale string) string {
	return translator().Trans("menopause."+key, params, locale)
}

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	line, ok := translator().Get("menopause.attributes", locale)
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
