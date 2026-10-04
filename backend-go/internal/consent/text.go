package consent

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"sync"
)

// The consent copy (texts per code and version, controller messages, validation attribute names) is data:
// lang/<code>/consent.json. A language without the file, or without a key, falls back to English.
//
//go:embed lang/*/consent.json
var langFS embed.FS

const fallbackLocale = "en"

// Text is one version of a consent's copy.
type Text struct {
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Points []string `json:"points"`
}

type langFile struct {
	Messages   map[string]string          `json:"messages"`
	Attributes map[string]string          `json:"attributes"`
	Texts      map[string]map[string]Text `json:"texts"`
}

var bundles = sync.OnceValues(func() (map[string]langFile, error) {
	out := map[string]langFile{}
	dirs, err := fs.ReadDir(langFS, "lang")
	if err != nil {
		return nil, err
	}
	for _, d := range dirs {
		raw, err := langFS.ReadFile(path.Join("lang", d.Name(), "consent.json"))
		if err != nil {
			return nil, err
		}
		var f langFile
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, fmt.Errorf("consent: lang/%s: %w", d.Name(), err)
		}
		out[d.Name()] = f
	}
	if _, ok := out[fallbackLocale]; !ok {
		return nil, fmt.Errorf("consent: lang/%s missing", fallbackLocale)
	}
	return out, nil
})

func bundle(locale string) (langFile, langFile) {
	b, err := bundles()
	if err != nil {
		panic(err) // embedded files are checked by the package tests
	}
	return b[locale], b[fallbackLocale]
}

// T is the message for key ("consent_required") in locale.
func T(key, locale string) string {
	l, en := bundle(locale)
	if s, ok := l.Messages[key]; ok {
		return s
	}
	if s, ok := en.Messages[key]; ok {
		return s
	}
	return key
}

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	l, en := bundle(locale)
	m := l.Attributes
	if len(m) == 0 {
		m = en.Attributes
	}
	kv := make([]string, 0, 2*len(m))
	for _, k := range []string{"granted", "version"} {
		if v, ok := m[k]; ok {
			kv = append(kv, k, v)
		}
	}
	return kv
}

// TextOf is the copy of code at version in locale (English when the language lacks it).
func TextOf(code string, version int, locale string) (Text, bool) {
	l, en := bundle(locale)
	v := strconv.Itoa(version)
	if t, ok := l.Texts[code][v]; ok {
		return t, true
	}
	t, ok := en.Texts[code][v]
	return t, ok
}
