package languages_test

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/languages"
	publiccontent "github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	langfs "github.com/ritme/backend-go/resources/lang"
	"github.com/ritme/backend-go/resources/translations"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// laravelKey is Laravel's registry cache key with the default prefixes (Redis DB 1).
const laravelKey = languages.DefaultLaravelKeyPrefix + "languages.registry"

func newEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	laravel, err := languages.NewLaravelCache(e.Cache)
	require.NoError(t, err)
	languages.New(languages.Deps{
		DB: e.DB, Registry: e.Registry, Laravel: laravel, Logger: admintest.Quiet,
		Bundles: languages.NewBundlesFromSeed(translations.FS, e.Storage, langfs.FS),
	}).Routes(e.Route(), e.Kit)
	// The public endpoints the frontend reads, wired as routes_content.go does.
	pub := publiccontent.New(publiccontent.Deps{
		Translations: i18n.NewTranslationStore(translations.FS, e.Storage), StoragePath: e.Storage,
	})
	e.App.Get("/api/v1/languages", i18n.Middleware(e.Registry), pub.Languages)
	e.App.Get("/api/v1/languages/:code/messages", i18n.Middleware(e.Registry), pub.Messages)
	return e
}

func publicGet(t *testing.T, e *admintest.Env, path string) map[string]any {
	t.Helper()
	res, err := e.App.Test(httptest.NewRequest(fiber.MethodGet, path, nil))
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	require.Equal(t, 200, res.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	return body["data"].(map[string]any)
}

func codes(data map[string]any) []string {
	var out []string
	for _, l := range data["languages"].([]any) {
		out = append(out, l.(map[string]any)["code"].(string))
	}
	return out
}

func TestSuperOnly(t *testing.T) {
	e := newEnv(t)
	ed := e.As(admintest.EditorID)
	for _, r := range []admintest.Resp{
		ed.Get("/languages"),
		ed.JSON(fiber.MethodPost, "/languages", map[string]any{"code": "ar"}),
		ed.Get("/languages/1/translations"),
		ed.JSON(fiber.MethodPut, "/languages/1/translations", map[string]any{"rows": []any{}}),
		ed.JSON(fiber.MethodPost, "/languages/1/toggle", nil),
		ed.JSON(fiber.MethodDelete, "/languages/2", nil),
	} {
		assert.Equal(t, 403, r.Status)
		assert.Equal(t, "forbidden", r.Code())
	}
	assert.Equal(t, 401, e.Anonymous().Get("/languages").Status)
	assert.Equal(t, 200, e.As(admintest.SuperID).Get("/languages").Status)
}

func TestProvisionNewLanguage(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.SuperID)
	e.Exec(`INSERT INTO message_contents (` + "`group`" + `, item_key, locale, label, payload, is_active, is_approved, sort_order, created_at, updated_at) VALUES
		('pattern', 'a', 'fa', 'A', '{"text":"سلام"}', 1, 1, 0, NOW(), NOW()),
		('pattern', 'b', 'fa', NULL, '{"tips":["x","y"]}', 0, 1, 1, NOW(), NOW()),
		('pattern', 'a', 'en', 'A', '{"text":"Hi"}', 1, 1, 0, NOW(), NOW())`)

	// Both stacks have the language list cached.
	assert.Equal(t, []string{"fa", "en"}, codes(publicGet(t, e, "/api/v1/languages")))
	require.True(t, e.Redis.Exists("ritme-go:languages.registry"))
	require.NoError(t, e.Redis.DB(1).Set(laravelKey, "a:0:{}"))

	// Validation.
	r := c.JSON(fiber.MethodPost, "/languages", map[string]any{"code": "../etc", "name": "x", "english_name": "x",
		"direction": "sideways"})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "code")
	assert.Contains(t, r.Errors(), "direction")
	r = c.JSON(fiber.MethodPost, "/languages", map[string]any{"code": "EN", "name": "x", "english_name": "x",
		"direction": "ltr"})
	require.Equal(t, 422, r.Status, "unique after normalising")
	assert.Contains(t, r.Errors(), "code")

	r = c.JSON(fiber.MethodPost, "/languages", map[string]any{
		"code": "AR", "name": "العربية", "english_name": "Arabic", "direction": "rtl", "is_active": true,
		"sort_order": 30, "copy_from": "fa",
	})
	require.Equal(t, 201, r.Status, r.Body)
	lang := r.Obj("language")
	assert.Equal(t, "ar", lang["code"])
	assert.Equal(t, "fa", r.Data()["source"])
	prov := r.Obj("provisioned")
	assert.Positive(t, prov["messages"])
	assert.EqualValues(t, 3, prov["lang_files"])
	assert.EqualValues(t, 2, prov["smart_messages"])

	// Caches flushed: Go's key is gone and so is Laravel's (DB 1).
	assert.False(t, e.Redis.Exists("ritme-go:languages.registry"))
	assert.False(t, e.Redis.DB(1).Exists(laravelKey), "Laravel's languages.registry forgotten")

	// The public API lists it at once and serves its bundle (copied from fa).
	assert.Equal(t, []string{"fa", "en", "ar"}, codes(publicGet(t, e, "/api/v1/languages")))
	msgs := publicGet(t, e, "/api/v1/languages/ar/messages")
	assert.Equal(t, "ar", msgs["locale"])
	assert.Equal(t, "rtl", msgs["direction"])
	fa := publicGet(t, e, "/api/v1/languages/fa/messages")
	assert.Equal(t, fa["messages"], msgs["messages"], "a copy of the source language")

	// Files on the volume.
	entries, err := os.ReadDir(filepath.Join(e.Storage, "app", "translations", "ar"))
	require.NoError(t, err)
	assert.Len(t, entries, int(prov["messages"].(float64)))
	for _, g := range []string{"validation", "profile", "cycle"} {
		assert.FileExists(t, filepath.Join(e.Storage, "app", "lang", "ar", g+".json"))
	}
	// Smart messages cloned, unapproved.
	assert.Equal(t, 2, e.Int(`SELECT COUNT(*) FROM message_contents WHERE locale = 'ar' AND is_approved = 0`))
	assert.Equal(t, `{"text":"سلام"}`, e.String(`SELECT payload FROM message_contents WHERE locale = 'ar' AND item_key = 'a'`))

	// Regenerating from en rewrites the bundles but never duplicates rows.
	aid := int(lang["id"].(float64))
	r = c.JSON(fiber.MethodPost, "/languages/"+itoa(aid)+"/regenerate", map[string]any{"copy_from": "en"})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, "en", r.Data()["source"])
	assert.EqualValues(t, 0, r.Obj("provisioned")["smart_messages"])
	assert.EqualValues(t, 0, r.Obj("provisioned")["lang_files"], "existing lang files are never clobbered")
	en := publicGet(t, e, "/api/v1/languages/en/messages")
	assert.Equal(t, en["messages"], publicGet(t, e, "/api/v1/languages/ar/messages")["messages"])

	// Toggle off: gone from the public list; delete removes files and rows.
	r = c.JSON(fiber.MethodPost, "/languages/"+itoa(aid)+"/toggle", nil)
	require.Equal(t, 200, r.Status)
	assert.Equal(t, false, r.Obj("language")["is_active"])
	assert.Equal(t, []string{"fa", "en"}, codes(publicGet(t, e, "/api/v1/languages")))

	r = c.JSON(fiber.MethodDelete, "/languages/"+itoa(aid), nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.NoDirExists(t, filepath.Join(e.Storage, "app", "translations", "ar"))
	assert.NoDirExists(t, filepath.Join(e.Storage, "app", "lang", "ar"))
	assert.Equal(t, 0, e.Int(`SELECT COUNT(*) FROM message_contents WHERE locale = 'ar'`))
	assert.Equal(t, 2, e.Int(`SELECT COUNT(*) FROM message_contents WHERE locale = 'fa'`))
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func TestDefaultLanguageRules(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.SuperID)
	faID := e.Int(`SELECT id FROM languages WHERE code = 'fa'`)
	enID := e.Int(`SELECT id FROM languages WHERE code = 'en'`)

	r := c.JSON(fiber.MethodDelete, "/languages/"+itoa(faID), nil)
	require.Equal(t, 422, r.Status)
	assert.Equal(t, languages.CodeDefaultLanguage, r.Code())
	r = c.JSON(fiber.MethodPost, "/languages/"+itoa(faID)+"/toggle", nil)
	require.Equal(t, 422, r.Status)
	assert.Equal(t, languages.CodeDefaultLanguage, r.Code())

	// Making an inactive en the default activates it and demotes fa.
	e.Exec(`UPDATE languages SET is_active = 0 WHERE id = ?`, enID)
	r = c.JSON(fiber.MethodPut, "/languages/"+itoa(enID), map[string]any{
		"code": "en", "name": "English", "english_name": "English", "direction": "ltr", "is_default": true,
	})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, true, r.Obj("language")["is_active"])
	assert.Equal(t, true, r.Obj("language")["is_default"])
	assert.Equal(t, 0, e.Int(`SELECT is_default FROM languages WHERE code = 'fa'`))
	assert.Equal(t, "en", publicGet(t, e, "/api/v1/languages")["default"])

	// Un-ticking the only default hands it to the first active language.
	r = c.JSON(fiber.MethodPut, "/languages/"+itoa(enID), map[string]any{
		"code": "en", "name": "English", "english_name": "English", "direction": "ltr", "is_active": true,
	})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, 1, e.Int(`SELECT COUNT(*) FROM languages WHERE is_default = 1`))
	assert.Equal(t, 1, e.Int(`SELECT is_default FROM languages WHERE code = 'fa'`))

	r = c.Get("/languages/options")
	require.Equal(t, 200, r.Status)
	assert.Equal(t, "fa", r.Data()["default_code"])
}

func TestTranslationEditor(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.SuperID)
	enID := itoa(e.Int(`SELECT id FROM languages WHERE code = 'en'`))

	r := c.Get("/languages/" + enID + "/translations?namespace=common")
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, "common", r.Data()["namespace"])
	rows := r.Data()["rows"].([]any)
	require.NotEmpty(t, rows)
	first := rows[0].(map[string]any)
	assert.Contains(t, first, "reference")
	key := first["key"].(string)

	r = c.Get("/languages/" + enID + "/translations?namespace=../../etc")
	require.Equal(t, 200, r.Status)
	assert.NotEqual(t, "../../etc", r.Data()["namespace"], "unknown namespaces fall back to the first")

	r = c.JSON(fiber.MethodPut, "/languages/"+enID+"/translations", map[string]any{
		"namespace": "common",
		"rows": []any{
			map[string]any{"key": key, "value": "Edited"},
			map[string]any{"key": "brandNew.nested", "value": "Nested"},
			map[string]any{"key": "blank", "value": ""},
		},
	})
	require.Equal(t, 200, r.Status, r.Body)
	raw, err := os.ReadFile(filepath.Join(e.Storage, "app", "translations", "en", "common.json"))
	require.NoError(t, err)
	var saved map[string]any
	require.NoError(t, json.Unmarshal(raw, &saved))
	assert.Equal(t, map[string]any{"nested": "Nested"}, saved["brandNew"])
	assert.NotContains(t, saved, "blank", "empty values fall back to the default language")

	msgs := publicGet(t, e, "/api/v1/languages/en/messages")["messages"].(map[string]any)["common"].(map[string]any)
	assert.Equal(t, "Nested", msgs["brandNew"].(map[string]any)["nested"])

	r = c.JSON(fiber.MethodPut, "/languages/"+enID+"/translations", map[string]any{
		"namespace": "common", "rows": []any{map[string]any{"value": "no key"}},
	})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "rows.0.key")
}
