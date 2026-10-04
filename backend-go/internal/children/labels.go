package children

import (
	"embed"
	"io/fs"
	"strconv"
	"strings"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// The children copy (controller messages, validation attribute names, age / visit / status labels, notes) is data:
// lang/<code>/children.json, one file per language, read through the platform translator. A language without the
// file (or a key) falls back to English (lang.FallbackLocale). The clinical catalogs (vaccines, milestones, learn tips)
// are admin catalog_items rows, not this file.
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

// T is the children line for key ("messages.created") in locale.
func T(key, locale string) string { return translator().Trans("children."+key, nil, locale) }

// Tp is T with :placeholders.
func Tp(key string, params map[string]string, locale string) string {
	return translator().Trans("children."+key, params, locale)
}

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	line, ok := translator().Get("children.attributes", locale)
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

// Digits writes the ASCII digits of s in the locale's digit set.
func Digits(s, locale string) string {
	digits := []rune(T("digits", locale))
	if len(digits) != 10 {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(digits[r-'0'])
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func num(n int, locale string) string { return Digits(strconv.Itoa(n), locale) }
