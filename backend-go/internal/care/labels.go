package care

import (
	"embed"
	"io/fs"
	"strings"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// The care copy (enum labels, controller messages, validation attribute names) is data:
// lang/<code>/care.json, one file per language, read through the platform translator.
// A language without the file (or without a key) falls back to English, like every other
// server-side line (lang.FallbackLocale).
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

// T is the care line for key ("messages.validation_failed") in locale.
func T(key, locale string) string { return translator().Trans("care."+key, nil, locale) }

// labeled is a [{value, label}] list for the enums endpoint.
func labeled(group, locale string, values []string) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(values))
	for _, v := range values {
		out = append(out, jsonx.Obj("value", v, "label", T(group+"."+v, locale)))
	}
	return out
}

// Label is the localized label of an enum value (group "forms", "units", …); ok=false when
// the value has no label (a free-text unit, say).
func Label(group, value, locale string) (string, bool) {
	line, ok := translator().Get("care."+group, locale)
	if !ok {
		return "", false
	}
	s, ok := phpval.Get(line, value)
	if !ok {
		return "", false
	}
	str, isStr := s.(string)
	return str, isStr
}

// attributes are the validation attribute names for locale (flat "key", "name" pairs).
func attributes(locale string) []string {
	line, ok := translator().Get("care.attributes", locale)
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

// LocalizeDigits writes the ASCII digits of s in the locale's digit set ("digits" line).
func LocalizeDigits(s, locale string) string {
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
