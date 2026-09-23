// Package languages is language management in the admin API, super admins only
// (Admin\LanguageController, TranslationController and App\Services\Language\LanguageProvisioner):
// CRUD, toggle, default switching, regenerating a language's files and the per-namespace
// UI-string editor.
//
// Every write to the languages table flushes both registry caches: Go's
// (ritme-go:languages.registry) and — while the Laravel API still runs — Laravel's
// (see LaravelCache), so GET /api/v1/languages on either stack lists the change at once.
package languages

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// CodeDefaultLanguage is the 422 error_code for deleting or deactivating the default language.
const CodeDefaultLanguage = "default_language_protected"

// Deps wires the handlers.
type Deps struct {
	DB       *sql.DB
	Registry *i18n.Registry
	Bundles  *Bundles
	Laravel  *LaravelCache // nil: no Laravel API to notify
	Logger   *slog.Logger
}

// Handlers serve /languages.
type Handlers struct {
	db      *sql.DB
	q       *store.Queries
	reg     *i18n.Registry
	bundles *Bundles
	laravel *LaravelCache
	logger  *slog.Logger
}

// New builds the handlers.
func New(d Deps) *Handlers {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	return &Handlers{db: d.DB, q: store.New(d.DB), reg: d.Registry, bundles: d.Bundles, laravel: d.Laravel, logger: d.Logger}
}

// NewBundlesFromSeed is NewBundles over the embedded seeds (resources/translations, resources/lang).
func NewBundlesFromSeed(seed fs.FS, storagePath string, langFS fs.FS) *Bundles {
	return NewBundles(i18n.NewTranslationStore(seed, storagePath), storagePath, langFS)
}

// Routes registers the endpoints (all super-only).
func (h *Handlers) Routes(route func(method, path string, chain httpadmin.Chain), kit *httpadmin.Kit) {
	s := kit.Super
	get, post, put, del := fiber.MethodGet, fiber.MethodPost, fiber.MethodPut, fiber.MethodDelete
	route(get, "/languages", s(h.List))
	route(get, "/languages/options", s(h.Options))
	route(post, "/languages", s(h.Store))
	route(get, "/languages/:id", s(h.Show))
	route(put, "/languages/:id", s(h.Update))
	route(del, "/languages/:id", s(h.Destroy))
	route(post, "/languages/:id/toggle", s(h.Toggle))
	route(post, "/languages/:id/regenerate", s(h.Regenerate))
	route(get, "/languages/:id/translations", s(h.Translations))
	route(put, "/languages/:id/translations", s(h.UpdateTranslations))
}

func languageJSON(l *store.Language) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", l.ID,
		"code", l.Code,
		"name", l.Name,
		"english_name", l.EnglishName,
		"direction", l.Direction,
		"is_active", l.IsActive,
		"is_default", l.IsDefault,
		"sort_order", l.SortOrder,
		"created_at", httpadmin.Time(l.CreatedAt),
		"updated_at", httpadmin.Time(l.UpdatedAt),
	)
}

func (h *Handlers) find(c fiber.Ctx) (*store.Language, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return nil, httpadmin.NotFound("Language")
	}
	l, err := h.q.GetLanguage(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, httpadmin.NotFound("Language")
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// flush drops the Go and the Laravel registry caches (LanguageRegistry::flush).
func (h *Handlers) flush(ctx context.Context) error {
	if err := h.reg.Flush(ctx); err != nil {
		return fmt.Errorf("languages: flush registry: %w", err)
	}
	return h.laravel.Forget(ctx)
}

// List is GET /languages: every language (active or not) in display order.
func (h *Handlers) List(c fiber.Ctx) error {
	rows, err := h.q.ListAllLanguages(c.Context())
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, languageJSON(&rows[i]))
	}
	return httpadmin.OK(c, jsonx.Obj("items", items, "default_code", h.reg.DefaultCode(c.Context())))
}

// Options is GET /languages/options: directions, the languages a new one can copy its
// text from (active ones), the default code and the next sort_order (max + 10).
func (h *Handlers) Options(c fiber.Ctx) error {
	langs := h.reg.All(c.Context())
	maxSort, err := h.q.MaxLanguageSortOrder(c.Context())
	if err != nil {
		return err
	}
	sources := make([]*jsonx.OrderedMap, 0, len(langs))
	for _, l := range langs {
		sources = append(sources, jsonx.Obj("code", l.Code, "name", l.Name))
	}
	return httpadmin.OK(c, jsonx.Obj(
		"directions", jsonx.List(enums.TextDirectionValues()),
		"sources", sources,
		"default_code", langs.DefaultCode(),
		"next_sort_order", maxSort+10,
	))
}

// Show is GET /languages/:id.
func (h *Handlers) Show(c fiber.Ctx) error {
	l, err := h.find(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("language", languageJSON(l)))
}

// validate is LanguageController::validated: the code keeps the BCP-47 shape (it ends up
// in URLs, JSON keys, Accept-Language and directory names) and is unique.
func (h *Handlers) validate(c fiber.Ctx, exceptID uint64) (phpval.Map, error) {
	return form.Validate(c, validation.Rules{
		validation.F("code", "required|string|max:12", validation.Regex(`/^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})?$/`)),
		validation.F("name", "required|string|max:60"),
		validation.F("english_name", "required|string|max:60"),
		validation.F("direction", "required", validation.In(enums.TextDirectionValues()...)),
		validation.F("sort_order", "nullable|integer"),
		validation.F("is_active", "nullable"),
		validation.F("is_default", "nullable"),
	}, func(in phpval.Map, add form.Add) error {
		v, _ := in.Get("code")
		s, ok := v.(string)
		if !ok {
			return nil
		}
		taken, err := h.q.LanguageCodeTaken(c.Context(), store.LanguageCodeTakenParams{
			Code: i18n.NormalizeCode(s), ExceptID: exceptID,
		})
		if err != nil {
			return err
		}
		if taken {
			add("code", form.Msg(c, "validation.unique", "code"))
		}
		return nil
	})
}

type languageInput struct {
	code, name, englishName, direction string
	active, isDefault                  bool
	sortOrder                          int32
}

func inputOf(data phpval.Map) languageInput {
	return languageInput{
		code: i18n.NormalizeCode(httpadmin.String(data, "code")), name: httpadmin.String(data, "name"),
		englishName: httpadmin.String(data, "english_name"), direction: httpadmin.String(data, "direction"),
		active: httpadmin.Bool(data, "is_active"), isDefault: httpadmin.Bool(data, "is_default"),
		sortOrder: form.Int32(data, "sort_order", 0),
	}
}

// applyDefault keeps exactly one default: a default language demotes the others and is
// forced active; if the table is left without a default, the first active language
// (display order) becomes it (LanguageController::applyDefault).
func applyDefault(ctx context.Context, q *store.Queries, id uint64, in languageInput, now sql.NullTime) error {
	if !in.isDefault {
		found, err := q.AnyDefaultLanguage(ctx)
		if err != nil || found {
			return err
		}
		first, err := q.FirstActiveLanguageID(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		return q.SetLanguageDefault(ctx, store.SetLanguageDefaultParams{Now: now, ID: first})
	}
	if err := q.ClearOtherDefaultLanguages(ctx, store.ClearOtherDefaultLanguagesParams{Now: now, ID: id}); err != nil {
		return err
	}
	if !in.active {
		return q.SetLanguageActive(ctx, store.SetLanguageActiveParams{IsActive: true, Now: now, ID: id})
	}
	return nil
}

// inTx runs fn in a transaction.
func (h *Handlers) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(h.q.WithTx(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// copySource is the language a new/regenerated one copies from: copy_from when it is an
// active language, else the default language.
func (h *Handlers) copySource(ctx context.Context, c fiber.Ctx) (source string, langs i18n.Languages) {
	langs = h.reg.All(ctx)
	v, _ := validation.Input(c).Get("copy_from")
	if s, ok := v.(string); ok && langs.IsSupported(s) {
		return i18n.NormalizeCode(s), langs
	}
	return langs.DefaultCode(), langs
}

// Store is POST /languages {code, name, english_name, direction, sort_order?, is_active?,
// is_default?, copy_from?}: creates the row, keeps one default, flushes the registry
// caches and provisions the language (UI bundles copied from copy_from, lang files, and
// unapproved copies of copy_from's smart messages).
func (h *Handlers) Store(c fiber.Ctx) error {
	data, err := h.validate(c, 0)
	if err != nil {
		return err
	}
	in := inputOf(data)
	now := httpadmin.DBTime(httpadmin.Now(c))
	// The source and the default are read before the insert, so the new language can
	// neither copy from itself nor (when created as the default) from an empty default.
	source, before := h.copySource(c.Context(), c)
	var id uint64
	err = h.inTx(c.Context(), func(q *store.Queries) error {
		res, err := q.CreateLanguage(c.Context(), store.CreateLanguageParams{
			Code: in.code, Name: in.name, EnglishName: in.englishName, Direction: in.direction,
			IsActive: in.active, IsDefault: in.isDefault, SortOrder: in.sortOrder, Now: now,
		})
		if err != nil {
			return err
		}
		n, err := res.LastInsertId()
		if err != nil {
			return err
		}
		id = uint64(n) //nolint:gosec // G115: auto-increment ids are positive
		return applyDefault(c.Context(), q, id, in, now)
	})
	if err != nil {
		return err
	}
	if err := h.flush(c.Context()); err != nil {
		return err
	}
	result, err := h.provision(c.Context(), in.code, source, before.DefaultCode(), now)
	if err != nil {
		return err
	}
	l, err := h.q.GetLanguage(c.Context(), id)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "language.create", "language", id, slog.String("code", l.Code),
		slog.String("copy_from", source))
	return httpadmin.Created(c, jsonx.Obj("language", languageJSON(&l), "source", source, "provisioned", result),
		"Language created.")
}

// Update is PUT /languages/:id (same fields as Store; files are not renamed when the
// code changes, as in Laravel).
func (h *Handlers) Update(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	data, err := h.validate(c, cur.ID)
	if err != nil {
		return err
	}
	in := inputOf(data)
	now := httpadmin.DBTime(httpadmin.Now(c))
	if err := h.inTx(c.Context(), func(q *store.Queries) error {
		if err := q.UpdateLanguage(c.Context(), store.UpdateLanguageParams{
			Code: in.code, Name: in.name, EnglishName: in.englishName, Direction: in.direction,
			IsActive: in.active, IsDefault: in.isDefault, SortOrder: in.sortOrder, Now: now, ID: cur.ID,
		}); err != nil {
			return err
		}
		return applyDefault(c.Context(), q, cur.ID, in, now)
	}); err != nil {
		return err
	}
	if err := h.flush(c.Context()); err != nil {
		return err
	}
	l, err := h.q.GetLanguage(c.Context(), cur.ID)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "language.update", "language", cur.ID, slog.String("code", l.Code),
		slog.Bool("is_active", l.IsActive), slog.Bool("is_default", l.IsDefault))
	return httpadmin.OK(c, jsonx.Obj("language", languageJSON(&l)), "Language updated.")
}

// Destroy is DELETE /languages/:id: never the default language (422
// default_language_protected); removes the language's files and smart-message rows.
func (h *Handlers) Destroy(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	if cur.IsDefault {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeDefaultLanguage,
			"The default language cannot be deleted. Make another language the default first.")
	}
	res, err := h.q.DeleteLanguage(c.Context(), cur.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return httpadmin.NotFound("Language")
	}
	if err := h.bundles.DeleteFor(cur.Code); err != nil {
		h.logger.WarnContext(c.Context(), "languages: could not remove language files",
			slog.String("code", cur.Code), slog.String("error", err.Error()))
	}
	if err := h.q.DeleteMessageContentsForLocale(c.Context(), cur.Code); err != nil {
		return err
	}
	if err := h.flush(c.Context()); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "language.delete", "language", cur.ID, slog.String("code", cur.Code))
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID), "Language and its translation files deleted.")
}

// Toggle is POST /languages/:id/toggle (is_active); the active default language cannot be
// deactivated (422 default_language_protected).
func (h *Handlers) Toggle(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	if cur.IsDefault && cur.IsActive {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeDefaultLanguage,
			"The default language cannot be deactivated.")
	}
	if err := h.q.SetLanguageActive(c.Context(), store.SetLanguageActiveParams{
		IsActive: !cur.IsActive, Now: httpadmin.DBTime(httpadmin.Now(c)), ID: cur.ID,
	}); err != nil {
		return err
	}
	if err := h.flush(c.Context()); err != nil {
		return err
	}
	l, err := h.q.GetLanguage(c.Context(), cur.ID)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "language.toggle", "language", cur.ID, slog.Bool("is_active", l.IsActive))
	return httpadmin.OK(c, jsonx.Obj("language", languageJSON(&l)), "Language status changed.")
}

// Regenerate is POST /languages/:id/regenerate {copy_from?}: rewrites the language's UI
// bundles from copy_from (default: the default language) and fills in missing lang files
// and smart-message rows. Bundles are overwritten, as in Laravel.
func (h *Handlers) Regenerate(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	source, langs := h.copySource(c.Context(), c)
	result, err := h.provision(c.Context(), cur.Code, source, langs.DefaultCode(), httpadmin.DBTime(httpadmin.Now(c)))
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "language.regenerate", "language", cur.ID, slog.String("copy_from", source))
	return httpadmin.OK(c, jsonx.Obj("language", languageJSON(cur), "source", source, "provisioned", result),
		"Translation files regenerated.")
}

// provision is LanguageProvisioner::provision.
func (h *Handlers) provision(ctx context.Context, code, source, defaultCode string, now sql.NullTime) (*jsonx.OrderedMap, error) {
	msgs, err := h.bundles.GenerateFor(code, source, defaultCode)
	if err != nil {
		return nil, err
	}
	langFiles, err := h.bundles.CopyLangFiles(code, source)
	if err != nil {
		return nil, err
	}
	smart, err := h.cloneMessages(ctx, code, source, now)
	if err != nil {
		return nil, err
	}
	return jsonx.Obj("messages", msgs, "lang_files", langFiles, "smart_messages", smart), nil
}

// cloneMessages is LanguageProvisioner::cloneMessageContents: only for a language with no
// rows yet; the copies start unapproved (they still hold the source language's words).
func (h *Handlers) cloneMessages(ctx context.Context, code, source string, now sql.NullTime) (int64, error) {
	if code == source {
		return 0, nil
	}
	exists, err := h.q.MessageLocaleExists(ctx, code)
	if err != nil || exists {
		return 0, err
	}
	res, err := h.q.CloneMessageContents(ctx, store.CloneMessageContentsParams{Code: code, Now: now, Source: source})
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
