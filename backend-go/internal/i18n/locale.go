package i18n

import (
	"slices"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/validation"
)

type localsKey int

const (
	localeKey localsKey = iota
	languagesKey
)

// Middleware is SetLocale on the api group: it resolves the request locale from
// `?locale=` (after TrimStrings/ConvertEmptyStringsToNull, so an empty value falls
// through), then Accept-Language, then the default language, and stores it together with
// the request's language list on the context. Read them with Locale / LanguagesOf.
func Middleware(reg *Registry) fiber.Handler {
	return func(c fiber.Ctx) error {
		langs := reg.All(c.Context())
		c.Locals(languagesKey, langs)
		requested, _ := requestedLocale(c)
		c.Locals(localeKey, langs.Resolve(requested))
		return c.Next()
	}
}

// DefaultMiddleware is `setlocale:default`: pins the default language (admin API).
func DefaultMiddleware(reg *Registry) fiber.Handler {
	return func(c fiber.Ctx) error {
		langs := reg.All(c.Context())
		c.Locals(languagesKey, langs)
		c.Locals(localeKey, langs.DefaultCode())
		return c.Next()
	}
}

// Locale is app()->getLocale() for the request (set by Middleware).
func Locale(c fiber.Ctx) string {
	if l, ok := c.Locals(localeKey).(string); ok {
		return l
	}
	return LanguagesOf(c).DefaultCode()
}

// LanguagesOf returns the language list the middleware loaded for this request
// (Bootstrap when the middleware did not run, e.g. in unit tests of a handler).
func LanguagesOf(c fiber.Ctx) Languages {
	if l, ok := c.Locals(languagesKey).(Languages); ok {
		return l
	}
	return Bootstrap
}

// requestedLocale is `$request->query('locale') ?? $request->header('Accept-Language')`,
// with ok=false when the result is not a string (missing, or ?locale[]=…).
func requestedLocale(c fiber.Ctx) (string, bool) {
	if q, present := validation.Query(c).Get("locale"); present && q != nil {
		s, isStr := q.(string)
		return s, isStr
	}
	if h := c.Request().Header.Peek(fiber.HeaderAcceptLanguage); h != nil {
		return string(h), true
	}
	return "", false
}

// ResolveLocale is the ResolvesLocale trait: like the middleware, except that when the
// request names no locale at all, def is used if it is an active language.
func ResolveLocale(c fiber.Ctx, def string) string {
	langs := LanguagesOf(c)
	requested, ok := requestedLocale(c)
	if !ok || requested == "" {
		if def != "" && langs.IsSupported(def) {
			return langs.Resolve(def)
		}
		return langs.DefaultCode()
	}
	return langs.Resolve(requested)
}

// LegacyPair is the fa/en pair some Laravel controllers still hard-code
// (`in_array($locale, ['fa', 'en'], true) ? $locale : 'fa'`). It exists only to port
// those call sites byte-for-byte; new code resolves locales through the registry.
var LegacyPair = []string{"fa", "en"}

// Clamp returns locale when it is one of allowed, else fallback:
//
//	PhaseContentController::show      Clamp(ResolveLocale(c, ""), "fa", LegacyPair...)
//	PregnancyWeeklyController::content Clamp(ResolveLocale(c, ""), "en", LegacyPair...)
func Clamp(locale, fallback string, allowed ...string) string {
	if slices.Contains(allowed, locale) {
		return locale
	}
	return fallback
}

// QueryLocaleIn returns ?locale= when it is exactly one of allowed. InfoController::show:
//
//	locale, ok := QueryLocaleIn(c, LegacyPair...); if !ok { locale = ResolveLocale(c, "") }
func QueryLocaleIn(c fiber.Ctx, allowed ...string) (string, bool) {
	q, _ := validation.Query(c).Get("locale")
	s, isStr := q.(string)
	if isStr && slices.Contains(allowed, s) {
		return s, true
	}
	return "", false
}
