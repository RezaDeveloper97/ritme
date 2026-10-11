package catalog

import (
	"embed"
	"io/fs"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
)

// The catalog's server copy (only the teen 403 today) is data: lang/<code>/shop_catalog.json, one file per language,
// read through the platform translator. A language without the file (or key) falls back to English.
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

// T is the catalog line for key ("errors.teen_commercial_blocked") in locale.
func T(key, locale string) string { return translator().Trans("shop_catalog."+key, nil, locale) }
