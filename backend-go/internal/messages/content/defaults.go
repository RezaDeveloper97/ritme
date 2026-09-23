package content

import (
	_ "embed"
	"fmt"

	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// defaultsJSON is every ProvidesMessageContent::contentDefaults() (MessageContentSeeder
// providers, seeder order): group → item_key → locale → payload. Regenerate with
//
//	php internal/messages/content/export_defaults.php ../backend > internal/messages/content/defaults.json
//
// TestDefaultsMatchPHPExport keeps it byte-identical to the PHP export.
//
//go:embed defaults.json
var defaultsJSON []byte

// seedLocale is the locale every class's slice() falls back to (`$val[$locale] ?? $val['fa']`,
// BmiService::messageFor the same): the language the code copy was written in. It is a
// property of the embedded export, not a list of supported languages.
const seedLocale = "fa"

// defaults is the decoded export.
var defaults = mustDecodeDefaults()

func mustDecodeDefaults() phpval.Map {
	v, err := phpval.Decode(defaultsJSON)
	if err != nil {
		panic(fmt.Sprintf("messages/content: defaults.json: %v", err))
	}
	m, ok := v.(phpval.Map)
	if !ok {
		panic("messages/content: defaults.json is not an object")
	}
	return m
}

// DefaultsJSON returns the embedded export (the Go MessageContentSeeder of T-M2-27 seeds it).
func DefaultsJSON() []byte { return append([]byte(nil), defaultsJSON...) }

// Groups returns the content groups in export order.
func Groups() []string { return defaults.Keys() }

// Items returns the item keys of a group in export order (the keys of the class's PHP source
// array, e.g. the phases a message exists for); nil for an unknown group.
func Items(group string) []string {
	g, ok := defaults.Get(group)
	if !ok {
		return nil
	}
	m, _ := g.(phpval.Map)
	if m == nil {
		return nil
	}
	return m.Keys()
}

// HasItem is `array_key_exists($itemKey, <the class's source array>)`.
func HasItem(group, itemKey string) bool {
	_, ok := item(group, itemKey)
	return ok
}

func item(group, itemKey string) (phpval.Map, bool) {
	g, ok := defaults.Get(group)
	if !ok {
		return nil, false
	}
	gm, _ := g.(phpval.Map)
	if gm == nil {
		return nil, false
	}
	it, ok := gm.Get(itemKey)
	if !ok {
		return nil, false
	}
	im, _ := it.(phpval.Map)
	return im, im != nil
}

// Default is the code fallback of one entry: `slice($entry, $locale)` — the locale's copy, else
// the seed locale's. The zero Payload (every field missing) when the entry does not exist.
func Default(group, itemKey, locale string) Payload {
	it, ok := item(group, itemKey)
	if !ok {
		return Payload{}
	}
	if v, ok := it.Get(locale); ok && v != nil {
		return Payload{v: v}
	}
	v, _ := it.Get(seedLocale)
	return Payload{v: v}
}
