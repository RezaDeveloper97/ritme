package companion

import (
	"embed"
	"io/fs"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
)

// The companion copy (controller messages, validation lines, owner-inbox notices) is data: lang/<code>/companion.json,
// read through the platform translator; a language without the file (or key) falls back to English.
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

// T is the companion line for key ("messages.not_found") in locale, with Laravel-style :name replacements.
func T(key, locale string, params ...string) string {
	var p map[string]string
	if len(params) > 1 {
		p = make(map[string]string, len(params)/2)
		for i := 0; i+1 < len(params); i += 2 {
			p[params[i]] = params[i+1]
		}
	}
	return translator().Trans("companion."+key, p, locale)
}
