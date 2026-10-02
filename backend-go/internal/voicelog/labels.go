package voicelog

import (
	"embed"
	"io/fs"
	"strconv"
	"strings"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

// The voice-log copy (controller messages, validation attribute names, suggestion label patterns) is data:
// lang/<code>/voicelog.json, one file per language, read through the platform translator. A language
// without the file (or a key) falls back to English (lang.FallbackLocale).
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

// T is the voice-log line for key ("messages.ai_failed") in locale, with :placeholders replaced.
func T(key, locale string, params map[string]string) string {
	return strings.TrimSpace(translator().Trans("voicelog."+key, params, locale))
}

// num is a chip number in the language's digits (B-N3-14b, N3 stage smoke B-8): fa → «۵۸٫۵», else «58.5».
func num(n float64, locale string) string {
	return digits(strconv.FormatFloat(n, 'f', -1, 64), locale)
}

// digits writes a value's ASCII digits (and decimal point) in the language's digits: fa → Persian, else as is.
func digits(s, locale string) string {
	if locale != "fa" {
		return s
	}
	return phpround.PersianDigits(strings.ReplaceAll(s, ".", "٫"))
}
