package i18n_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/resources/translations"
)

func canonical(t *testing.T, v any) string {
	t.Helper()
	b, err := jsonx.Marshal(v, jsonx.UnescapedUnicode|jsonx.UnescapedSlashes)
	require.NoError(t, err)
	return string(b)
}

// testdata/messages_<code>.json hold data.{locale,direction,messages} of the Laravel
// goldens for GET /api/v1/languages/<code>/messages (backend-go/contract/golden/public),
// recorded with languages fa (default), en and ar (a row only: no translation files).
func TestBundle_MatchesLaravelGoldens(t *testing.T) {
	store := i18n.NewTranslationStore(translations.FS, "")
	for _, code := range []string{"fa", "en", "ar"} {
		t.Run(code, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join("testdata", "messages_"+code+".json"))
			require.NoError(t, err)
			golden, err := phpval.Decode(b)
			require.NoError(t, err)
			want, _ := golden.(phpval.Map).Get("messages")

			got := store.Bundle(code, "fa")
			assert.Equal(t, canonical(t, want), canonical(t, got))
		})
	}
}

func TestBundle_LiveLayerAndDefaultFill(t *testing.T) {
	storage := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(storage, "app", "translations", rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}
	// fa (default) edits one key, blanks another (ignored), adds a namespace.
	write("fa/nav.json", `{"home":"خانه من","calendar":"","extra":{"x":"y"}}`)
	write("fa/zzz.json", `{"only":"live"}`)
	// ar: a partial live translation; everything else inherits fa.
	write("ar/nav.json", `{"home":"الرئيسية"}`)
	write("ar/broken.json", `not json`)

	store := i18n.NewTranslationStore(translations.FS, storage)
	seed := i18n.NewTranslationStore(translations.FS, "")

	assert.Contains(t, store.Namespaces("fa"), "zzz")
	assert.NotContains(t, store.Namespaces("fa"), "broken", "namespaces come from the default language only")

	fa := store.Bundle("fa", "fa")
	nav, _ := fa.Get("nav")
	home, _ := nav.(phpval.Map).Get("home")
	assert.Equal(t, "خانه من", home)
	seedNav, _ := seed.Bundle("fa", "fa").Get("nav")
	seedCal, _ := seedNav.(phpval.Map).Get("calendar")
	cal, _ := nav.(phpval.Map).Get("calendar")
	assert.Equal(t, seedCal, cal, "an empty live value keeps the seed text")
	extra, _ := nav.(phpval.Map).Get("extra")
	assert.Equal(t, `{"x":"y"}`, canonical(t, extra))

	ar := store.Bundle("ar", "fa")
	arNav, _ := ar.Get("nav")
	arHome, _ := arNav.(phpval.Map).Get("home")
	assert.Equal(t, "الرئيسية", arHome)
	arCal, _ := arNav.(phpval.Map).Get("calendar")
	assert.Equal(t, seedCal, arCal, "missing keys are filled from the default language")
	zzz, _ := ar.Get("zzz")
	assert.Equal(t, `{"only":"live"}`, canonical(t, zzz))

	// Unknown code: everything from the default language.
	xx := store.Bundle("xx", "fa")
	assert.Equal(t, canonical(t, fa), canonical(t, xx))
}
