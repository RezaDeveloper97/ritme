// Package content is the editorial part of the admin API (T-M2-21): CRUD (+ toggle where
// the Blade panel has it) for articles, affirmations, challenges (+ the completions
// report), recommendations, banners, task templates, info sections, pregnancy weeks and
// phase contents. Validation follows the Admin\*Controller rules; translatable columns
// accept one value per active language and only the default language is required.
package content

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Deps wires the handlers.
type Deps struct {
	DB        *sql.DB
	Disk      *media.Disk // public disk (STORAGE_PATH/app/public)
	Optimizer media.Optimizer
	AppURL    string // APP_URL: image URLs are APP_URL/storage/<path>
	Logger    *slog.Logger
}

// Handlers serve the content endpoints.
type Handlers struct {
	q      *store.Queries
	disk   *media.Disk
	opt    media.Optimizer
	appURL string
	logger *slog.Logger
}

// New builds the handlers.
func New(d Deps) *Handlers {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.Optimizer == (media.Optimizer{}) {
		d.Optimizer = media.DefaultOptimizer
	}
	if d.Disk == nil {
		d.Disk = media.NewDisk("")
	}
	return &Handlers{q: store.New(d.DB), disk: d.Disk, opt: d.Optimizer, appURL: d.AppURL, logger: d.Logger}
}

// Route registers one endpoint (httpadmin.Handle with the prefix applied).
type Route func(method, path string, chain httpadmin.Chain)

// Routes registers every content endpoint; static paths come before :id paths.
func (h *Handlers) Routes(route Route, kit *httpadmin.Kit) {
	get, post, put, del := fiber.MethodGet, fiber.MethodPost, fiber.MethodPut, fiber.MethodDelete
	a := kit.Admin

	route(get, "/articles", a(h.ListArticles))
	route(get, "/articles/options", a(h.ArticleOptions))
	route(post, "/articles", a(h.StoreArticle))
	route(get, "/articles/:id", a(h.ShowArticle))
	route(put, "/articles/:id", a(h.UpdateArticle))
	route(post, "/articles/:id", a(h.UpdateArticle)) // multipart forms may not support PUT everywhere
	route(del, "/articles/:id", a(h.DestroyArticle))
	route(post, "/articles/:id/toggle", a(h.ToggleArticle))

	route(get, "/affirmations", a(h.ListAffirmations))
	route(get, "/affirmations/options", a(h.AffirmationOptions))
	route(post, "/affirmations", a(h.StoreAffirmation))
	route(get, "/affirmations/:id", a(h.ShowAffirmation))
	route(put, "/affirmations/:id", a(h.UpdateAffirmation))
	route(del, "/affirmations/:id", a(h.DestroyAffirmation))
	route(post, "/affirmations/:id/toggle", a(h.ToggleAffirmation))

	route(get, "/challenge-completions", a(h.ChallengeCompletions))
	route(get, "/challenges", a(h.ListChallenges))
	route(get, "/challenges/options", a(h.ChallengeOptions))
	route(post, "/challenges", a(h.StoreChallenge))
	route(get, "/challenges/:id", a(h.ShowChallenge))
	route(put, "/challenges/:id", a(h.UpdateChallenge))
	route(del, "/challenges/:id", a(h.DestroyChallenge))
	route(post, "/challenges/:id/toggle", a(h.ToggleChallenge))

	route(get, "/recommendations", a(h.ListRecommendations))
	route(get, "/recommendations/options", a(h.RecommendationOptions))
	route(post, "/recommendations", a(h.StoreRecommendation))
	route(get, "/recommendations/:id", a(h.ShowRecommendation))
	route(put, "/recommendations/:id", a(h.UpdateRecommendation))
	route(del, "/recommendations/:id", a(h.DestroyRecommendation))
	route(post, "/recommendations/:id/toggle", a(h.ToggleRecommendation))

	route(get, "/banners", a(h.ListBanners))
	route(get, "/banners/options", a(h.BannerOptions))
	route(post, "/banners", a(h.StoreBanner))
	route(get, "/banners/:id", a(h.ShowBanner))
	route(put, "/banners/:id", a(h.UpdateBanner))
	route(post, "/banners/:id", a(h.UpdateBanner))
	route(del, "/banners/:id", a(h.DestroyBanner))
	route(post, "/banners/:id/toggle", a(h.ToggleBanner))

	route(get, "/task-templates", a(h.ListTaskTemplates))
	route(get, "/task-templates/options", a(h.TaskTemplateOptions))
	route(post, "/task-templates", a(h.StoreTaskTemplate))
	route(get, "/task-templates/:id", a(h.ShowTaskTemplate))
	route(put, "/task-templates/:id", a(h.UpdateTaskTemplate))
	route(del, "/task-templates/:id", a(h.DestroyTaskTemplate))
	route(post, "/task-templates/:id/toggle", a(h.ToggleTaskTemplate))

	route(get, "/info-sections", a(h.ListInfoSections))
	route(get, "/info-sections/options", a(h.InfoSectionOptions))
	route(post, "/info-sections", a(h.StoreInfoSection))
	route(get, "/info-sections/:id", a(h.ShowInfoSection))
	route(put, "/info-sections/:id", a(h.UpdateInfoSection))
	route(del, "/info-sections/:id", a(h.DestroyInfoSection))
	route(post, "/info-sections/:id/toggle", a(h.ToggleInfoSection))

	route(get, "/pregnancy-weeks", a(h.ListPregnancyWeeks))
	route(post, "/pregnancy-weeks", a(h.StorePregnancyWeek))
	route(get, "/pregnancy-weeks/:id", a(h.ShowPregnancyWeek))
	route(put, "/pregnancy-weeks/:id", a(h.UpdatePregnancyWeek))
	route(del, "/pregnancy-weeks/:id", a(h.DestroyPregnancyWeek))

	route(get, "/phase-contents", a(h.ListPhaseContents))
	route(post, "/phase-contents", a(h.StorePhaseContent))
	route(get, "/phase-contents/:id", a(h.ShowPhaseContent))
	route(put, "/phase-contents/:id", a(h.UpdatePhaseContent))
	route(del, "/phase-contents/:id", a(h.DestroyPhaseContent))
}

// ---------------------------------------------------------------------------
// Shared helpers

// find loads the row named by :id, answering 404 for a missing row or a non-numeric id.
func find[T any](c fiber.Ctx, what string, get func(context.Context, uint64) (T, error)) (T, error) {
	var zero T
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return zero, httpadmin.NotFound(what)
	}
	row, err := get(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return zero, httpadmin.NotFound(what)
	}
	return row, err
}

// insertedID is the auto-increment id of an INSERT.
func insertedID(res sql.Result) (uint64, error) {
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return uint64(id), nil //nolint:gosec // G115: auto-increment ids are positive
}

// deleted maps a DELETE result to 404 when nothing was removed (a concurrent delete).
func deleted(res sql.Result, err error, what string) error {
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return httpadmin.NotFound(what)
	}
	return nil
}

func (h *Handlers) now(c fiber.Ctx) sql.NullTime { return httpadmin.DBTime(httpadmin.Now(c)) }

// options are enum {value,label} pairs in the admin's (default) language.
func options(opts []enums.Option) []enums.Option { return jsonx.List(opts) }

func locale(c fiber.Ctx) string { return i18n.Locale(c) }

// deleteFile removes an uploaded file, logging (not failing) on error: the row change
// has already been committed.
func (h *Handlers) deleteFile(c fiber.Ctx, rel string) {
	if err := h.disk.Delete(rel); err != nil {
		h.logger.WarnContext(c.Context(), "admin: could not delete old upload",
			slog.String("path", rel), slog.String("error", err.Error()))
	}
}
