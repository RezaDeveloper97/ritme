package content

import (
	"database/sql"
	"log/slog"
	"slices"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/admin/store"
	publiccontent "github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/content/sanitizer"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// ArticleImageDir is where optimised covers land on the public disk.
const ArticleImageDir = "articles"

// articleImageMaxKB is `max:8192` on the cover upload.
const articleImageMaxKB = 8192

func (h *Handlers) articleJSON(a *store.Article) *jsonx.OrderedMap {
	// cover_url is the Article::image_url accessor the app shows: an uploaded image wins
	// over the stored external URL.
	var cover any
	switch {
	case a.ImagePath.Valid && a.ImagePath.String != "":
		cover = publiccontent.PublicURL(h.appURL, a.ImagePath.String)
	case a.ImageUrl.Valid && a.ImageUrl.String != "":
		cover = a.ImageUrl.String
	}
	return jsonx.Obj(
		"id", a.ID,
		"slug", a.Slug,
		"title", form.Raw(a.Title),
		"excerpt", form.NullRaw(a.Excerpt),
		"body", form.NullRaw(a.Body),
		"cycle_phases", form.NullRaw(a.CyclePhases),
		"category", httpadmin.NullString(a.Category),
		"read_time_minutes", form.NullInt(a.ReadTimeMinutes),
		"image_url", httpadmin.NullString(a.ImageUrl),
		"image_path", httpadmin.NullString(a.ImagePath),
		"cover_url", cover,
		"is_published", a.IsPublished,
		"published_at", httpadmin.Time(a.PublishedAt),
		"sort_order", a.SortOrder,
		"created_at", httpadmin.Time(a.CreatedAt),
		"updated_at", httpadmin.Time(a.UpdatedAt),
	)
}

// phaseOption is a picker entry; legacy marks a stored main-phase key that the current
// sub-phase list no longer offers (kept so editing the row does not drop it).
type phaseOption struct {
	Value  string `json:"value"`
	Label  string `json:"label"`
	Legacy bool   `json:"legacy"`
}

// articlePhaseOptions is ArticleController::phaseOptions: the content-backed sub-phases,
// plus any value of current the sub-phase enum does not know.
func articlePhaseOptions(c fiber.Ctx, current []string) []phaseOption {
	loc := locale(c)
	out := []phaseOption{}
	for _, o := range enums.CycleSubphaseOptions(loc) {
		out = append(out, phaseOption{Value: o.Value, Label: o.Label})
	}
	known := enums.CycleSubphaseValues()
	seen := map[string]bool{}
	for _, p := range current {
		if slices.Contains(known, p) || seen[p] {
			continue
		}
		seen[p] = true
		label, ok := enums.CyclePhaseLabelFor(p, loc)
		if !ok {
			label = p
		}
		out = append(out, phaseOption{Value: p, Label: label, Legacy: true})
	}
	return out
}

// allowedArticlePhases: the sub-phases plus the legacy main-phase keys.
func allowedArticlePhases() []string {
	out := enums.CycleSubphaseValues()
	for _, p := range enums.CyclePhaseValues() {
		if !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}

func storedPhases(col db.NullRawJSON) []string {
	var out []string
	v := form.NullRaw(col)
	_, vals := phpval.Entries(v)
	for _, x := range vals {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// ListArticles is GET /articles (sort_order, newest first).
func (h *Handlers) ListArticles(c fiber.Ctx) error {
	total, err := h.q.CountAdminArticles(c.Context())
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListAdminArticles(c.Context(), store.ListAdminArticlesParams{
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, h.articleJSON(&rows[i]))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, int(total)))
}

// ArticleOptions is GET /articles/options (the create form's pickers).
func (h *Handlers) ArticleOptions(c fiber.Ctx) error {
	return httpadmin.OK(c, jsonx.Obj("phases", articlePhaseOptions(c, nil),
		"max_image_width", media.DefaultOptimizer.MaxWidth, "max_image_kb", articleImageMaxKB))
}

// ShowArticle is GET /articles/:id → {article, options}.
func (h *Handlers) ShowArticle(c fiber.Ctx) error {
	a, err := find(c, "Article", h.q.GetArticle)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("article", h.articleJSON(&a),
		"options", jsonx.Obj("phases", articlePhaseOptions(c, storedPhases(a.CyclePhases)))))
}

func (h *Handlers) articleRules(c fiber.Ctx) validation.Rules {
	rules := validation.Rules{validation.F("slug", "required|string|max:255")}
	rules = append(rules, form.Translatable(c, "title", true)...)
	rules = append(rules, form.Translatable(c, "excerpt", false)...)
	rules = append(rules, form.Translatable(c, "body", false)...)
	return append(rules,
		validation.F("cycle_phases", "nullable|array"),
		validation.F("cycle_phases.*", validation.In(allowedArticlePhases()...)),
		validation.F("category", "nullable|string|max:255"),
		validation.F("read_time_minutes", "nullable|integer|min:1|max:120"),
		validation.F("image_url", "nullable|string|max:1000"),
		validation.F("sort_order", "nullable|integer"),
		validation.F("is_published", "nullable"),
		validation.F("remove_image", "nullable"),
	)
}

// validateArticle runs the rules, the slug uniqueness and the cover upload checks.
func (h *Handlers) validateArticle(c fiber.Ctx, exceptID uint64) (phpval.Map, *media.Image, error) {
	var img *media.Image
	data, err := form.Validate(c, h.articleRules(c),
		func(in phpval.Map, add form.Add) error {
			slug, ok := in.Get("slug")
			s, isStr := slug.(string)
			if !ok || !isStr {
				return nil
			}
			taken, err := h.q.ArticleSlugTaken(c.Context(), store.ArticleSlugTakenParams{Slug: s, ExceptID: exceptID})
			if err != nil {
				return err
			}
			if taken {
				add("slug", form.Msg(c, "validation.unique", "slug"))
			}
			return nil
		},
		form.ImageRule{Field: "image", MaxKB: articleImageMaxKB}.Check(c, &img),
	)
	return data, img, err
}

// sanitizeBody runs every language's HTML through the T-M2-10 sanitizer (the app renders
// it with dangerouslySetInnerHTML); a value with nothing renderable left becomes null.
func sanitizeBody(data phpval.Map) {
	v, ok := data.Get("body")
	m, isMap := v.(phpval.Map)
	if !ok || !isMap {
		return
	}
	for _, k := range m.Keys() {
		x, _ := m.Get(k)
		s, isStr := x.(string)
		if !isStr {
			continue
		}
		if clean, ok := sanitizer.Clean(s); ok {
			m.Set(k, clean)
		} else {
			m.Set(k, nil)
		}
	}
}

// articlePhases is `array_values(array_unique($data['cycle_phases'] ?? [])) ?: null`.
func articlePhases(data phpval.Map) db.NullRawJSON {
	v, _ := data.Get("cycle_phases")
	_, vals := phpval.Entries(v)
	var out []any
	for _, x := range vals {
		if !slices.ContainsFunc(out, func(y any) bool { return phpval.LooseEqual(x, y) }) {
			out = append(out, x)
		}
	}
	if len(out) == 0 {
		return db.NullRawJSON{}
	}
	return db.NullRawJSON{V: form.JSON(out), Valid: true}
}

// StoreArticle is POST /articles (JSON or multipart with an `image` file).
func (h *Handlers) StoreArticle(c fiber.Ctx) error {
	data, img, err := h.validateArticle(c, 0)
	if err != nil {
		return err
	}
	sanitizeBody(data)
	now := h.now(c)
	published := httpadmin.Bool(data, "is_published")
	var publishedAt sql.NullTime
	if published {
		publishedAt = now
	}
	var imagePath sql.NullString
	if img != nil {
		rel, err := h.disk.Store(h.opt, ArticleImageDir, *img)
		if err != nil {
			return err
		}
		imagePath = sql.NullString{String: rel, Valid: true}
	}
	res, err := h.q.CreateArticle(c.Context(), store.CreateArticleParams{
		Slug: httpadmin.String(data, "slug"), Title: form.ReqJSON(data, "title"),
		Excerpt: form.NullJSON(data, "excerpt"), Body: form.NullJSON(data, "body"), CyclePhases: articlePhases(data),
		Category: form.Str(data, "category"), ReadTimeMinutes: form.NullInt16(data, "read_time_minutes"),
		ImageUrl: form.Str(data, "image_url"), ImagePath: imagePath, IsPublished: published,
		PublishedAt: publishedAt, SortOrder: form.Int32(data, "sort_order", 0), Now: now,
	})
	if err != nil {
		h.deleteFile(c, imagePath.String)
		return err
	}
	id, err := insertedID(res)
	if err != nil {
		return err
	}
	a, err := h.q.GetArticle(c.Context(), id)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "article.create", "article", id, slog.Bool("image", img != nil))
	return httpadmin.Created(c, jsonx.Obj("article", h.articleJSON(&a)), "Article created.")
}

// UpdateArticle is PUT (or multipart POST) /articles/:id. A new `image` replaces the
// cover (the old file is deleted after the row is saved); `remove_image` clears it.
func (h *Handlers) UpdateArticle(c fiber.Ctx) error {
	cur, err := find(c, "Article", h.q.GetArticle)
	if err != nil {
		return err
	}
	data, img, err := h.validateArticle(c, cur.ID)
	if err != nil {
		return err
	}
	sanitizeBody(data)
	now := h.now(c)
	published := httpadmin.Bool(data, "is_published")
	publishedAt := cur.PublishedAt
	if published && !cur.PublishedAt.Valid {
		publishedAt = now
	}
	imagePath := cur.ImagePath
	switch {
	case img != nil:
		rel, err := h.disk.Store(h.opt, ArticleImageDir, *img)
		if err != nil {
			return err
		}
		imagePath = sql.NullString{String: rel, Valid: true}
	case httpadmin.Bool(data, "remove_image"):
		imagePath = sql.NullString{}
	}
	err = h.q.UpdateArticle(c.Context(), store.UpdateArticleParams{
		Slug: httpadmin.String(data, "slug"), Title: form.ReqJSON(data, "title"),
		Excerpt: form.KeepJSON(data, "excerpt", cur.Excerpt), Body: form.KeepJSON(data, "body", cur.Body),
		CyclePhases: articlePhases(data), Category: form.KeepStr(data, "category", cur.Category),
		ReadTimeMinutes: form.KeepInt16(data, "read_time_minutes", cur.ReadTimeMinutes),
		ImageUrl:        form.KeepStr(data, "image_url", cur.ImageUrl), ImagePath: imagePath,
		IsPublished: published, PublishedAt: publishedAt, SortOrder: form.Int32(data, "sort_order", 0),
		Now: now, ID: cur.ID,
	})
	if err != nil {
		if img != nil {
			h.deleteFile(c, imagePath.String)
		}
		return err
	}
	if cur.ImagePath.Valid && cur.ImagePath != imagePath {
		h.deleteFile(c, cur.ImagePath.String)
	}
	a, err := h.q.GetArticle(c.Context(), cur.ID)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "article.update", "article", cur.ID, slog.Bool("image", img != nil))
	return httpadmin.OK(c, jsonx.Obj("article", h.articleJSON(&a)), "Article updated.")
}

// DestroyArticle is DELETE /articles/:id (the cover file goes too).
func (h *Handlers) DestroyArticle(c fiber.Ctx) error {
	cur, err := find(c, "Article", h.q.GetArticle)
	if err != nil {
		return err
	}
	res, err := h.q.DeleteArticle(c.Context(), cur.ID)
	if err := deleted(res, err, "Article"); err != nil {
		return err
	}
	if cur.ImagePath.Valid {
		h.deleteFile(c, cur.ImagePath.String)
	}
	httpadmin.Audit(c, h.logger, "article.delete", "article", cur.ID)
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID), "Article deleted.")
}

// ToggleArticle is POST /articles/:id/toggle: flips is_published; publishing sets
// published_at to now, unpublishing keeps it (ArticleController::toggle).
func (h *Handlers) ToggleArticle(c fiber.Ctx) error {
	cur, err := find(c, "Article", h.q.GetArticle)
	if err != nil {
		return err
	}
	now := h.now(c)
	publishedAt := cur.PublishedAt
	if !cur.IsPublished {
		publishedAt = now
	}
	if err := h.q.SetArticlePublished(c.Context(), store.SetArticlePublishedParams{
		IsPublished: !cur.IsPublished, PublishedAt: publishedAt, Now: now, ID: cur.ID,
	}); err != nil {
		return err
	}
	a, err := h.q.GetArticle(c.Context(), cur.ID)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "article.toggle", "article", cur.ID, slog.Bool("is_published", a.IsPublished))
	return httpadmin.OK(c, jsonx.Obj("article", h.articleJSON(&a)), "Article status changed.")
}
