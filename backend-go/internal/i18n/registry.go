// Package i18n is the Go side of Laravel's App\Services\Language and App\Support\Translatable:
// the language registry (languages are data, never code), request locale resolution,
// translatable JSON columns and the UI-string bundles served by /languages/{code}/messages.
package i18n

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"strings"

	"github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/cache"
)

// CacheKey is LanguageRegistry::CACHE_KEY, stored under the Go prefix (ritme-go:languages.registry).
const CacheKey = "languages.registry"

// Language is one active row of the languages table (LanguageRegistry::all()).
type Language struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	EnglishName string `json:"english_name"`
	Direction   string `json:"direction"`
	IsDefault   bool   `json:"is_default"`
}

// Bootstrap is LanguageRegistry::BOOTSTRAP (mirrors LanguageSeeder): used only when the
// languages table is missing, unreadable or empty.
var Bootstrap = Languages{
	{Code: "fa", Name: "فارسی", EnglishName: "Persian", Direction: "rtl", IsDefault: true},
	{Code: "en", Name: "English", EnglishName: "English", Direction: "ltr", IsDefault: false},
}

// Languages is the active language list in display order (sort_order, id).
type Languages []Language

// Codes returns the language codes in order.
func (l Languages) Codes() []string {
	out := make([]string, len(l))
	for i, x := range l {
		out[i] = x.Code
	}
	return out
}

// DefaultCode is the default language: the first is_default row, else the first row,
// else "fa" (LanguageRegistry::defaultCode).
func (l Languages) DefaultCode() string {
	for _, x := range l {
		if x.IsDefault {
			return x.Code
		}
	}
	if len(l) > 0 {
		return l[0].Code
	}
	return Bootstrap[0].Code
}

// IsSupported reports whether code (normalised) is an active language.
func (l Languages) IsSupported(code string) bool {
	return slices.Contains(l.Codes(), NormalizeCode(code))
}

// Resolve maps a client-supplied locale (?locale= value or an Accept-Language header
// such as "fa-IR,fa;q=0.9,en;q=0.8") to an active code: each comma-separated part,
// with any ";q=" dropped, normalised, matched exactly or by its base language
// ("en-US" → "en", and "en" → "en-GB"); unmatched → the default language.
func (l Languages) Resolve(requested string) string {
	if requested == "" {
		return l.DefaultCode()
	}
	codes := l.Codes()
	for _, part := range strings.Split(requested, ",") {
		candidate := NormalizeCode(strings.SplitN(part, ";", 2)[0])
		if candidate == "" || candidate == "*" {
			continue
		}
		if slices.Contains(codes, candidate) {
			return candidate
		}
		base := strings.SplitN(candidate, "-", 2)[0]
		for _, code := range codes {
			if code == base || strings.SplitN(code, "-", 2)[0] == base {
				return code
			}
		}
	}
	return l.DefaultCode()
}

// Direction is "rtl" or "ltr" for code ("ltr" for unknown codes).
func (l Languages) Direction(code string) string {
	for _, x := range l {
		if x.Code == code {
			if x.Direction == "rtl" {
				return "rtl"
			}
			return "ltr"
		}
	}
	return "ltr"
}

// Name is the native name of code (the code itself when unknown).
func (l Languages) Name(code string) string {
	for _, x := range l {
		if x.Code == code {
			return x.Name
		}
	}
	return code
}

// NormalizeCode is Language::normalizeCode: trimmed, lower-case, "_" → "-".
func NormalizeCode(code string) string {
	return strings.ToLower(strings.ReplaceAll(strings.Trim(code, " \t\n\r\x00\x0B"), "_", "-"))
}

// LanguageLister reads the active languages (store.Queries implements it).
type LanguageLister interface {
	ListActiveLanguages(ctx context.Context) ([]store.ListActiveLanguagesRow, error)
}

// Registry is LanguageRegistry: the active languages, cached in Redis forever until
// Flush (every write to the languages table must call it).
type Registry struct {
	db     LanguageLister
	cache  *cache.Client // nil: no caching
	logger *slog.Logger
}

// NewRegistry builds a registry. cache and logger may be nil.
func NewRegistry(db LanguageLister, c *cache.Client, logger *slog.Logger) *Registry {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Registry{db: db, cache: c, logger: logger}
}

// All returns the active languages. It never fails: an unreadable table yields Bootstrap
// (not cached, so the next request retries); an empty table yields Bootstrap (cached,
// like Laravel's rememberForever).
func (r *Registry) All(ctx context.Context) Languages {
	if r.cache != nil {
		if raw, err := r.cache.Get(ctx, CacheKey); err == nil {
			var langs Languages
			if json.Unmarshal([]byte(raw), &langs) == nil && len(langs) > 0 {
				return langs
			}
		} else if !errors.Is(err, cache.ErrMiss) {
			r.logger.WarnContext(ctx, "language registry: cache read failed", slog.String("error", err.Error()))
		}
	}
	langs, err := r.load(ctx)
	if err != nil {
		r.logger.ErrorContext(ctx, "language registry: load failed, using bootstrap languages",
			slog.String("error", err.Error()))
		return Bootstrap
	}
	if r.cache != nil {
		if b, err := json.Marshal(langs); err == nil {
			if err := r.cache.Set(ctx, CacheKey, string(b), 0); err != nil {
				r.logger.WarnContext(ctx, "language registry: cache write failed", slog.String("error", err.Error()))
			}
		}
	}
	return langs
}

func (r *Registry) load(ctx context.Context) (Languages, error) {
	rows, err := r.db.ListActiveLanguages(ctx)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return Bootstrap, nil
	}
	langs := make(Languages, len(rows))
	for i, row := range rows {
		langs[i] = Language{Code: row.Code, Name: row.Name, EnglishName: row.EnglishName,
			Direction: row.Direction, IsDefault: row.IsDefault}
	}
	return langs, nil
}

// Flush drops the cached list (LanguageRegistry::flush) after any write to languages.
func (r *Registry) Flush(ctx context.Context) error {
	if r.cache == nil {
		return nil
	}
	return r.cache.Delete(ctx, CacheKey)
}

// DefaultCode is All(ctx).DefaultCode().
func (r *Registry) DefaultCode(ctx context.Context) string { return r.All(ctx).DefaultCode() }

// Resolve is All(ctx).Resolve(requested).
func (r *Registry) Resolve(ctx context.Context, requested string) string {
	return r.All(ctx).Resolve(requested)
}
