package i18n_test

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/store"
)

type fakeLister struct {
	rows  []store.ListActiveLanguagesRow
	err   error
	calls int
}

func (f *fakeLister) ListActiveLanguages(context.Context) ([]store.ListActiveLanguagesRow, error) {
	f.calls++
	return f.rows, f.err
}

// withArabic is the contract fixture: fa (default), en, and an extra active ar.
func withArabic() *fakeLister {
	return &fakeLister{rows: []store.ListActiveLanguagesRow{
		{Code: "fa", Name: "فارسی", EnglishName: "Persian", Direction: "rtl", IsDefault: true},
		{Code: "en", Name: "English", EnglishName: "English", Direction: "ltr"},
		{Code: "ar", Name: "العربية", EnglishName: "Arabic", Direction: "rtl"},
	}}
}

func TestLanguages_Resolve(t *testing.T) {
	langs := i18n.NewRegistry(withArabic(), nil, nil).All(context.Background())
	require.Equal(t, []string{"fa", "en", "ar"}, langs.Codes())

	for in, want := range map[string]string{
		"":                             "fa",
		"en":                           "en",
		"ar":                           "ar",
		"AR":                           "ar",
		" ar ":                         "ar",
		"ar-SA":                        "ar",
		"ar_SA":                        "ar",
		"en-US,en;q=0.9":               "en",
		"fa-IR,fa;q=0.9,en;q=0.8":      "fa",
		"de-DE,de;q=0.9,ar;q=0.8":      "ar",
		"*":                            "fa",
		"*, en":                        "en",
		";q=1,en":                      "en",
		"de":                           "fa", // unknown → default
		"xx-YY,zz":                     "fa",
		"en-GB":                        "en",
		"ar-EG;q=0.5, en-US;q=0.9":     "ar", // parts are taken in order, q is ignored
		"de;q=1.0, fr;q=0.9, ar;q=0.1": "ar",
	} {
		assert.Equal(t, want, langs.Resolve(in), "Resolve(%q)", in)
	}

	// A registry language with a region matches a bare request ("en" → "en-gb").
	regional := i18n.Languages{{Code: "fa", IsDefault: true}, {Code: "en-gb"}}
	assert.Equal(t, "en-gb", regional.Resolve("en"))
	assert.Equal(t, "en-gb", regional.Resolve("en-US"))

	assert.Equal(t, "rtl", langs.Direction("ar"))
	assert.Equal(t, "ltr", langs.Direction("en"))
	assert.Equal(t, "ltr", langs.Direction("xx"))
	assert.Equal(t, "العربية", langs.Name("ar"))
	assert.Equal(t, "xx", langs.Name("xx"))
	assert.True(t, langs.IsSupported("AR"))
	assert.False(t, langs.IsSupported("de"))
}

func TestLanguages_DefaultCode(t *testing.T) {
	assert.Equal(t, "en", i18n.Languages{{Code: "fa"}, {Code: "en", IsDefault: true}}.DefaultCode())
	assert.Equal(t, "ar", i18n.Languages{{Code: "ar"}, {Code: "en"}}.DefaultCode(), "no default row → first")
	assert.Equal(t, "fa", i18n.Languages{}.DefaultCode())
}

func TestRegistry_BootstrapFallback(t *testing.T) {
	ctx := context.Background()
	empty := i18n.NewRegistry(&fakeLister{}, nil, nil)
	assert.Equal(t, i18n.Bootstrap, empty.All(ctx), "empty table → bootstrap fa/en")

	broken := i18n.NewRegistry(&fakeLister{err: errors.New("no such table")}, nil, nil)
	assert.Equal(t, i18n.Bootstrap, broken.All(ctx), "unreadable table → bootstrap fa/en")
	assert.Equal(t, "fa", broken.DefaultCode(ctx))
	assert.Equal(t, "en", broken.Resolve(ctx, "en-US"))
	assert.Equal(t, "fa", broken.Resolve(ctx, "ar"), "ar is not a bootstrap language")
}

func newLocaleApp(reg *i18n.Registry) *fiber.App {
	app := fiber.New()
	app.Use(i18n.Middleware(reg))
	app.Get("/", func(c fiber.Ctx) error {
		info, ok := i18n.QueryLocaleIn(c, i18n.LegacyPair...)
		if !ok {
			info = i18n.ResolveLocale(c, "")
		}
		return c.JSON(fiber.Map{
			"locale":   i18n.Locale(c),
			"withEn":   i18n.ResolveLocale(c, "en"),
			"withDe":   i18n.ResolveLocale(c, "de"),
			"phase":    i18n.Clamp(i18n.ResolveLocale(c, ""), "fa", i18n.LegacyPair...),
			"pregnant": i18n.Clamp(i18n.ResolveLocale(c, ""), "en", i18n.LegacyPair...),
			"info":     info,
		})
	})
	return app
}

func TestMiddleware_LocaleResolution(t *testing.T) {
	app := newLocaleApp(i18n.NewRegistry(withArabic(), nil, nil))

	type want struct{ locale, withEn, withDe, phase, pregnant, info string }
	cases := []struct {
		name   string
		query  string
		header *string
		want   want
	}{
		{"android: no header, no query", "", nil, want{"fa", "en", "fa", "fa", "fa", "fa"}},
		{"header en", "", ptr("en-US,en;q=0.9"), want{"en", "en", "en", "en", "en", "en"}},
		{"header ar (3rd language)", "", ptr("ar"), want{"ar", "ar", "ar", "fa", "en", "ar"}},
		{"query wins over header", "?locale=ar", ptr("en"), want{"ar", "ar", "ar", "fa", "en", "ar"}},
		{"empty query falls through to header", "?locale=", ptr("en"), want{"en", "en", "en", "en", "en", "en"}},
		{"blank query is trimmed to null", "?locale=%20%20", ptr("ar"), want{"ar", "ar", "ar", "fa", "en", "ar"}},
		{"query value trimmed", "?locale=%20en%20", nil, want{"en", "en", "en", "en", "en", "en"}},
		{"unknown query → default", "?locale=de", ptr("en"), want{"fa", "fa", "fa", "fa", "fa", "fa"}},
		{"array query → default", "?locale[]=en", ptr("en"), want{"fa", "en", "fa", "fa", "fa", "fa"}},
		{"empty header → default", "", ptr(""), want{"fa", "en", "fa", "fa", "fa", "fa"}},
		{"info query must be exact", "?locale=EN", nil, want{"en", "en", "en", "en", "en", "en"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(fiber.MethodGet, "/"+tc.query, nil)
			if tc.header != nil {
				req.Header.Set("Accept-Language", *tc.header)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			body, _ := io.ReadAll(resp.Body)
			w := tc.want
			assert.JSONEq(t, `{"locale":"`+w.locale+`","withEn":"`+w.withEn+`","withDe":"`+w.withDe+
				`","phase":"`+w.phase+`","pregnant":"`+w.pregnant+`","info":"`+w.info+`"}`, string(body))
		})
	}
}

func TestDefaultMiddleware_PinsDefault(t *testing.T) {
	app := fiber.New()
	app.Use(i18n.DefaultMiddleware(i18n.NewRegistry(withArabic(), nil, nil)))
	app.Get("/", func(c fiber.Ctx) error { return c.SendString(i18n.Locale(c)) })
	req := httptest.NewRequest(fiber.MethodGet, "/?locale=en", nil)
	req.Header.Set("Accept-Language", "ar")
	resp, err := app.Test(req)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "fa", string(body))
}

func ptr(s string) *string { return &s }
