package learning

import (
	"embed"
	"io/fs"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Controller messages, validation lines, attribute names and the «دوره برایت باز شد» notice are data:
// lang/<code>/learning.json (English fallback).
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

// T is the learning line for key ("messages.saved") in locale; kv are :param name / value pairs.
func T(key, locale string, kv ...string) string {
	var params map[string]string
	if len(kv) > 1 {
		params = make(map[string]string, len(kv)/2)
		for i := 0; i+1 < len(kv); i += 2 {
			params[kv[i]] = kv[i+1]
		}
	}
	return translator().Trans("learning."+key, params, locale)
}

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	v, ok := translator().Get("learning.attributes", locale)
	m, isMap := v.(phpval.Map)
	if !ok || !isMap {
		return nil
	}
	var kv []string
	for _, k := range m.Keys() {
		if s, ok := m.Get(k); ok {
			if str, ok := s.(string); ok {
				kv = append(kv, k, str)
			}
		}
	}
	return kv
}
