package todo

import (
	"embed"
	"io/fs"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Controller messages, validation lines, attribute names, reminder copy and the code fallback of the suggestion copy
// are data: lang/<code>/todo.json (English fallback). The live suggestion copy is message_contents todo_suggestion.
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

// T is the todo line for key ("messages.saved") in locale.
func T(key, locale string) string { return translator().Trans("todo."+key, nil, locale) }

// Tp is T with :param replacements.
func Tp(key string, params map[string]string, locale string) string {
	return translator().Trans("todo."+key, params, locale)
}

// line is a raw (possibly nested) entry of the todo group.
func line(key, locale string) (any, bool) { return translator().Get("todo."+key, locale) }

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	v, ok := line("attributes", locale)
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
