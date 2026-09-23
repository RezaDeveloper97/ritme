package lang_test

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/i18n/lang"
	langfs "github.com/ritme/backend-go/resources/lang"
)

func storageTranslator(t *testing.T) (*lang.Translator, string) {
	t.Helper()
	t.Cleanup(lang.SetCheckInterval(0))
	base, err := lang.New(langfs.FS, lang.FallbackLocale)
	require.NoError(t, err)
	root := t.TempDir()
	return base.WithStorage(root), filepath.Join(root, "app", "lang")
}

func writeLang(t *testing.T, dir, locale, group, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, locale), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, locale, group+".json"), []byte(body), 0o644))
}

func TestStorage_NewLocaleServedWithoutRestart(t *testing.T) {
	tr, dir := storageTranslator(t)
	first := map[string]string{"first": "x"}

	// Before the admin creates "de": English fallback.
	assert.Equal(t, "Validation failed: x", tr.Trans("profile.validation_failed", first, "de"))

	writeLang(t, dir, "de", "profile", `{"validation_failed": "Validierung fehlgeschlagen: :first"}`)
	writeLang(t, dir, "de", "validation", `{"required": "Das Feld :attribute ist erforderlich.", "attributes": {"log_date": "Datum"}}`)
	assert.Equal(t, "Validierung fehlgeschlagen: x", tr.Trans("profile.validation_failed", first, "de"))
	assert.Equal(t, "Das Feld Datum ist erforderlich.",
		tr.Trans("validation.required", map[string]string{"attribute": "Datum"}, "de"))
	attr, ok := tr.Get("validation.attributes.log_date", "de")
	require.True(t, ok)
	assert.Equal(t, "Datum", attr)
	// Keys missing from the storage group still fall back to English.
	assert.Equal(t, "The :attribute field must be between :min and :max.", tr.Trans("validation.between.numeric", nil, "de"))

	// An edit is picked up (size/mtime change), and a deleted language disappears.
	writeLang(t, dir, "de", "profile", `{"validation_failed": "Neu: :first"}`)
	future := time.Now().Add(time.Minute)
	require.NoError(t, os.Chtimes(filepath.Join(dir, "de", "profile.json"), future, future))
	assert.Equal(t, "Neu: x", tr.Trans("profile.validation_failed", first, "de"))
	require.NoError(t, os.RemoveAll(filepath.Join(dir, "de")))
	assert.Equal(t, "Validation failed: x", tr.Trans("profile.validation_failed", first, "de"))
}

func TestStorage_EmbeddedLocalesUnchanged(t *testing.T) {
	tr, dir := storageTranslator(t)
	writeLang(t, dir, "de", "profile", `{"validation_failed": "DE"}`)
	first := map[string]string{"first": "x"}
	assert.Equal(t, "خطای اعتبارسنجی: x", tr.Trans("profile.validation_failed", first, "fa"))
	assert.Equal(t, "Validation failed: x", tr.Trans("profile.validation_failed", first, "en"))
	assert.Equal(t, ":attribute باید حداقل :min باشد.", tr.Trans("validation.min.numeric", nil, "fa"))
}

func TestStorage_GroupInStorageWinsOverEmbedded(t *testing.T) {
	tr, dir := storageTranslator(t)
	writeLang(t, dir, "fa", "profile", `{"validation_failed": "override :first"}`)
	assert.Equal(t, "override x", tr.Trans("profile.validation_failed", map[string]string{"first": "x"}, "fa"))
	// Other embedded fa groups are untouched.
	assert.Equal(t, ":attribute باید حداقل :min باشد.", tr.Trans("validation.min.numeric", nil, "fa"))
}

func TestStorage_MalformedFileIgnoredAndLogged(t *testing.T) {
	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	tr, dir := storageTranslator(t)
	writeLang(t, dir, "de", "profile", `{"validation_failed": `)
	writeLang(t, dir, "de", "validation", `{"required": "DE :attribute"}`)
	writeLang(t, dir, "fa", "profile", `not json`)

	first := map[string]string{"first": "x"}
	assert.Equal(t, "Validation failed: x", tr.Trans("profile.validation_failed", first, "de"))
	assert.Equal(t, "DE a", tr.Trans("validation.required", map[string]string{"attribute": "a"}, "de"), "valid siblings still load")
	assert.Equal(t, "خطای اعتبارسنجی: x", tr.Trans("profile.validation_failed", first, "fa"), "embedded group stays in effect")
	assert.Contains(t, logs.String(), "ignoring storage lang file")
	assert.Contains(t, logs.String(), filepath.Join("de", "profile.json"))
}

func TestStorage_UnsafeLocaleNeverTouchesDisk(t *testing.T) {
	tr, dir := storageTranslator(t)
	writeLang(t, dir, "de", "profile", `{"validation_failed": "DE"}`)
	for _, loc := range []string{"../app/lang/de", "de/..", "", "."} {
		assert.Equal(t, "profile.nope", tr.Trans("profile.nope", nil, loc))
		assert.Equal(t, "Validation failed: ", tr.Trans("profile.validation_failed", map[string]string{"first": ""}, loc), loc)
	}
}

func TestWithStorage_EmptyPathIsEmbeddedOnly(t *testing.T) {
	base, err := lang.New(langfs.FS, lang.FallbackLocale)
	require.NoError(t, err)
	assert.Same(t, base, base.WithStorage(""))
}
