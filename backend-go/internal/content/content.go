// Package content ports the read-mostly content group of the Laravel API:
// LanguageController, InfoController, BannerController, ArticleController,
// PhaseContentController and the public-disk /storage files (api-inventory §1.3, §1.4,
// §1.7 phase-content, §1.11). Admin CRUD for these tables lives in T-M2-21.
package content

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/content/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Querier is the subset of store.Queries the handlers use (fakes in tests).
type Querier interface {
	ListActiveInfoSections(ctx context.Context, group string) ([]store.ListActiveInfoSectionsRow, error)
	ListActiveInfoPageSections(ctx context.Context, group string) ([]store.ListActiveInfoPageSectionsRow, error)
	ListActiveBanners(ctx context.Context, arg store.ListActiveBannersParams) ([]store.ListActiveBannersRow, error)
	GetPhaseContent(ctx context.Context, phase string) (store.GetPhaseContentRow, error)
	CountPublishedArticles(ctx context.Context, arg store.CountPublishedArticlesParams) (int64, error)
	ListPublishedArticles(ctx context.Context, arg store.ListPublishedArticlesParams) ([]store.ListPublishedArticlesRow, error)
	ListPublishedArticleCategories(ctx context.Context) ([]sql.NullString, error)
	GetPublishedArticleBySlug(ctx context.Context, slug string) (store.GetPublishedArticleBySlugRow, error)
	ListRelatedArticles(ctx context.Context, arg store.ListRelatedArticlesParams) ([]store.ListRelatedArticlesRow, error)
}

// CommercialPolicy decides whether an account may see commercial content (internal/teen.Policy: no banners for a
// teen-mode account, CB-TEEN-01). nil = everyone may.
type CommercialPolicy interface {
	AllowsCommercial(ctx context.Context, userID uint64) (bool, error)
}

// Deps wires the handlers.
type Deps struct {
	Queries      Querier
	Translations *i18n.TranslationStore
	AppURL       string // APP_URL (image URLs are APP_URL/storage/<path>)
	StoragePath  string // STORAGE_PATH; files are served from STORAGE_PATH/app/public
	Clock        clock.Clock
	Logger       *slog.Logger
	Commercial   CommercialPolicy
}

// Handlers are the content controllers.
type Handlers struct {
	q        Querier
	tr       *i18n.TranslationStore
	appURL   string
	public   string
	clock    clock.Clock
	logger   *slog.Logger
	fileRoot fileRoot
	policy   CommercialPolicy
}

// New builds the handlers.
func New(d Deps) *Handlers {
	if d.Clock == nil {
		d.Clock = clock.Real{}
	}
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	public := ""
	if d.StoragePath != "" {
		public = strings.TrimRight(d.StoragePath, "/") + "/app/public"
	}
	return &Handlers{q: d.Queries, tr: d.Translations, appURL: d.AppURL, public: public,
		clock: d.Clock, logger: d.Logger, fileRoot: fileRoot{dir: public}, policy: d.Commercial}
}

func (h *Handlers) now(c fiber.Ctx) time.Time { return clock.FromContext(c, h.clock).Now() }

// PublicURL is Storage::disk('public')->url($path): rtrim(APP_URL.'/storage', '/').'/'.ltrim($path, '/').
func PublicURL(appURL, path string) string {
	return strings.TrimRight(appURL+"/storage", "/") + "/" + strings.TrimLeft(path, "/")
}

// phpTruthy is PHP truthiness of a string ("" and "0" are false).
func phpTruthy(s string) bool { return s != "" && s != "0" }

// phpValue decodes raw JSON the way json_decode($raw, true) + json_encode would echo it.
func phpValue(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	v, err := phpval.Decode(raw)
	if err != nil {
		return nil
	}
	return phpval.Packed(v)
}

// pickString is Translatable::pick decoded: (value, true) only when the pick is a string.
func pickString(raw json.RawMessage, locale, def string) (string, bool) {
	var s string
	if err := json.Unmarshal(i18n.Pick(raw, locale, def), &s); err != nil {
		return "", false
	}
	return s, true
}

// RegistryTTL bounds the age of the shared language-registry cache entry
// (ritme-go:languages.registry, written without TTL by internal/i18n): during the
// strangler period Laravel's admin edits languages, and Go must pick them up. It sets
// an expiry only when the key has none (EXPIRE … NX), so it never extends one.
func RegistryTTL(c *cache.Client, ttl time.Duration) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if c != nil {
			_ = c.Redis().ExpireNX(ctx, c.Key(i18n.CacheKey), ttl).Err()
		}
		return ctx.Next()
	}
}
