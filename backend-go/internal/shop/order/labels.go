package order

import (
	"embed"
	"io/fs"
	"sync"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// TooManyDiscountTries is the localized 429 of PUT /shop/cart/discount's per-user limit (security audit: code guessing).
func TooManyDiscountTries(locale string, retryAfter int, headers map[string]string) *httpx.FailError {
	e := httpx.Fail(fiber.StatusTooManyRequests, T("errors.too_many_discount_tries", locale),
		"error_code", "too_many_requests", "retry_after", retryAfter)
	for k, v := range headers {
		e = e.WithHeader(k, v)
	}
	return e
}

// The order API's server copy (errors, messages, the order SMS, validation attribute names) is data:
// lang/<code>/shop_order.json, one file per language, read through the platform translator. A language without the
// file (or key) falls back to English.
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

// T is the order line for key ("errors.out_of_stock") in locale, with Laravel-style :param replacements.
func T(key, locale string, params ...string) string {
	var p map[string]string
	if len(params) > 1 {
		p = map[string]string{}
		for i := 0; i+1 < len(params); i += 2 {
			p[params[i]] = params[i+1]
		}
	}
	return translator().Trans("shop_order."+key, p, locale)
}

// attributeNames are the validation attribute pairs (field, display name) for locale.
func attributeNames(locale string, fields ...string) []string {
	out := make([]string, 0, 2*len(fields))
	for _, f := range fields {
		out = append(out, f, T("attributes."+f, locale))
	}
	return out
}
