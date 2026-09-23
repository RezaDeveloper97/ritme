// Package lang is Laravel's translator (__() / trans()) over the lang files converted to
// JSON in backend-go/resources/lang: "group.item.sub" keys, the request locale first,
// then the fallback locale (config/app.php fallback_locale = "en"), then the key itself.
//
// Only the PHP group files exist (validation, profile, cycle); there are no lang/<locale>.json
// files in the Laravel app, so JSON-string keys are not supported.
//
// D-04: groups of admin-created languages live on the storage volume as
// STORAGE_PATH/app/lang/<code>/<group>.json (written by internal/admin/languages). They are
// overlaid on the embedded files per group (storage wins), loaded lazily per locale and
// re-read when the directory's files change (see storage.go), so no restart is needed.
package lang

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	langfs "github.com/ritme/backend-go/resources/lang"
)

// FallbackLocale is Laravel's config('app.fallback_locale'): the framework's own English
// lines are the last resort for every locale (fa, ar, …) that lacks a key.
const FallbackLocale = "en"

// Translator resolves translation keys. It is safe for concurrent use: the embedded lines
// are immutable and the optional storage overlay synchronises itself.
type Translator struct {
	lines    map[string]map[string]any // locale → group → decoded file
	fallback string
	storage  *storageOverlay // nil = embedded files only
}

// New loads every <locale>/<group>.json from fsys.
func New(fsys fs.FS, fallback string) (*Translator, error) {
	files, err := fs.Glob(fsys, "*/*.json")
	if err != nil {
		return nil, fmt.Errorf("lang: %w", err)
	}
	t := &Translator{lines: map[string]map[string]any{}, fallback: fallback}
	for _, f := range files {
		b, err := fs.ReadFile(fsys, f)
		if err != nil {
			return nil, fmt.Errorf("lang: read %s: %w", f, err)
		}
		v, err := phpval.Decode(b)
		if err != nil {
			return nil, fmt.Errorf("lang: %s: %w", f, err)
		}
		locale, group := path.Dir(f), strings.TrimSuffix(path.Base(f), ".json")
		if t.lines[locale] == nil {
			t.lines[locale] = map[string]any{}
		}
		t.lines[locale][group] = v
	}
	return t, nil
}

var defaultTranslator = sync.OnceValue(func() *Translator {
	t, err := New(langfs.FS, FallbackLocale)
	if err != nil {
		panic(err) // embedded files are checked by the package tests
	}
	// STORAGE_PATH is required by internal/platform/config, so it is set whenever the API
	// runs; reading it here keeps Default() a drop-in for every caller.
	return t.WithStorage(os.Getenv("STORAGE_PATH"))
})

// Default returns the translator over the embedded resources/lang files, overlaid with
// STORAGE_PATH/app/lang when STORAGE_PATH is set.
func Default() *Translator { return defaultTranslator() }

// Get returns the line for key ("validation.custom.log_date.before_or_equal") in locale,
// falling back to the fallback locale. A line is a string or a non-empty array
// (phpval.Map / []any); anything else counts as missing, as in Translator::getLine.
func (t *Translator) Get(key, locale string) (any, bool) {
	group, item, _ := strings.Cut(key, ".")
	for _, loc := range []string{locale, t.fallback} {
		file, ok := t.group(loc, group)
		if !ok {
			continue
		}
		line, ok := phpval.Get(file, item)
		if !ok {
			continue
		}
		switch v := line.(type) {
		case string:
			return v, true
		case []any, phpval.Map:
			if phpval.Count(v) > 0 {
				return v, true
			}
		}
	}
	return nil, false
}

// group is the decoded <locale>/<group> file: the storage copy when there is one,
// else the embedded one.
func (t *Translator) group(locale, group string) (any, bool) {
	if t.storage != nil {
		if file, ok := t.storage.groups(locale)[group]; ok {
			return file, true
		}
	}
	file, ok := t.lines[locale][group]
	return file, ok
}

// Trans is __($key, $params) in locale: the translated string with :placeholders
// replaced, or the key itself when there is no string line.
func (t *Translator) Trans(key string, params map[string]string, locale string) string {
	line, ok := t.Get(key, locale)
	s, isString := line.(string)
	if !ok || !isString {
		s = key
	}
	return MakeReplacements(s, params)
}

// MakeReplacements is Translator::makeReplacements: for each param "name" the
// placeholders :name, :Name (ucfirst) and :NAME (upper) are replaced in one strtr pass.
func MakeReplacements(line string, params map[string]string) string {
	if len(params) == 0 {
		return line
	}
	pairs := make(map[string]string, len(params)*3)
	for k, v := range params {
		pairs[":"+UcFirst(k)] = UcFirst(v)
		pairs[":"+strings.ToUpper(k)] = strings.ToUpper(v)
		pairs[":"+k] = v
	}
	return Strtr(line, pairs)
}

// Strtr is PHP's strtr($s, $pairs): at every position the longest matching key wins,
// and replaced text is never scanned again.
func Strtr(s string, pairs map[string]string) string {
	if len(pairs) == 0 {
		return s
	}
	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		if k != "" {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	var b strings.Builder
	for i := 0; i < len(s); {
		matched := false
		for _, k := range keys {
			if strings.HasPrefix(s[i:], k) {
				b.WriteString(pairs[k])
				i += len(k)
				matched = true
				break
			}
		}
		if !matched {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// UcFirst is Str::ucfirst (multibyte: upper-cases the first character).
func UcFirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if n == 0 {
		return s
	}
	return strings.ToUpper(string(r)) + s[n:]
}
