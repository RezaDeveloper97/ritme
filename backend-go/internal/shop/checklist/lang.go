package checklist

import (
	"embed"
	"io/fs"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Controller messages, validation lines, attribute names, group titles and the (discreet) care-reminder title are
// data: lang/<code>/shop_checklist.json (English fallback for a language without the file or a key).
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

// T is the shop_checklist line for key ("messages.saved") in locale.
func T(key, locale string) string { return translator().Trans("shop_checklist."+key, nil, locale) }

// GroupTitle is the title of a group key (the key itself for an unknown group).
func GroupTitle(key, locale string) string {
	if _, ok := translator().Get("shop_checklist.groups."+key, locale); !ok {
		return key
	}
	return T("groups."+key, locale)
}

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	v, ok := translator().Get("shop_checklist.attributes", locale)
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
